package storagetest

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The Training journal's contract (ADR-0040 rule 6): a session and its items
// are written as one thing, the per-number aggregate counts by TYPE and not by
// session, and a number with no deviation stays out of the mean instead of
// entering it as a zero error.

func testTrainingSaveAndRead(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	tr := s.Training()

	id, err := tr.Save(ctx, "", storage.TrainingSession{
		Exercise:     "scores",
		SeedSource:   "pool",
		NumbersAsked: 3,
		Faults:       1,
		MedianMs:     4200,
		Items: []storage.TrainingItem{
			{NumberType: "tp2.live", Wrong: false},
			{NumberType: "tp2.last", Wrong: true},
			{NumberType: "gv1", Wrong: false},
		},
	})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if id == 0 {
		t.Fatal("Save returned id 0")
	}

	sessions, err := tr.Sessions(ctx, "", "scores", 0)
	if err != nil {
		t.Fatalf("Sessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("Sessions: %d rows, want 1", len(sessions))
	}
	got := sessions[0]
	if got.ID != id || got.Exercise != "scores" || got.SeedSource != "pool" {
		t.Errorf("Sessions[0] = %+v, want the row just saved", got)
	}
	if got.NumbersAsked != 3 || got.Faults != 1 || got.MedianMs != 4200 {
		t.Errorf("Sessions[0] aggregates = %+v, want 3 asked / 1 fault / 4200 ms", got)
	}
	if got.CreatedAt == "" {
		t.Error("Sessions[0].CreatedAt is empty: a journal entry with no date cannot be read as a trend")
	}

	// An exercise the journal has never seen answers with nothing, not with
	// everything: the summary of Pips must not show the Scores sessions.
	other, err := tr.Sessions(ctx, "", "pips", 0)
	if err != nil {
		t.Fatalf("Sessions(pips): %v", err)
	}
	if len(other) != 0 {
		t.Errorf("Sessions(pips) = %d rows, want 0", len(other))
	}
}

func testTrainingNumberStatsCountByType(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	tr := s.Training()

	// Two sessions, so the aggregate has to span them: the whole point of the
	// per-number detail is "tp4 last roll: 6 faults in 9", across sessions.
	for _, faults := range []bool{true, false} {
		if _, err := tr.Save(ctx, "", storage.TrainingSession{
			Exercise:     "scores",
			NumbersAsked: 2,
			Items: []storage.TrainingItem{
				{NumberType: "tp4.last", Wrong: faults},
				{NumberType: "gv2", Wrong: false},
			},
		}); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}
	// A session of another exercise must not leak into the Scores detail.
	if _, err := tr.Save(ctx, "", storage.TrainingSession{
		Exercise: "pips",
		Items:    []storage.TrainingItem{{NumberType: "pips.bottom", Wrong: true}},
	}); err != nil {
		t.Fatalf("Save(pips): %v", err)
	}

	stats, err := tr.NumberStats(ctx, "", "scores")
	if err != nil {
		t.Fatalf("NumberStats: %v", err)
	}
	byType := map[string]storage.TrainingNumberStat{}
	for _, st := range stats {
		byType[st.NumberType] = st
	}
	if len(byType) != 2 {
		t.Fatalf("NumberStats returned %d types (%v), want tp4.last and gv2", len(byType), byType)
	}
	if got := byType["tp4.last"]; got.Asked != 2 || got.Faults != 1 {
		t.Errorf("tp4.last = %+v, want 2 asked / 1 fault", got)
	}
	if got := byType["gv2"]; got.Asked != 2 || got.Faults != 0 {
		t.Errorf("gv2 = %+v, want 2 asked / 0 fault", got)
	}
}

func testTrainingDeviationStaysOutOfTheMeanWhenAbsent(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	tr := s.Training()

	// Three numbers of the same type: one deviated by 2, one by 4, and one
	// was never answered (out of time). The mean must be 3 — the mean of the
	// two that exist — and NOT 2, the mean a missing answer counted as zero
	// would produce.
	if _, err := tr.Save(ctx, "", storage.TrainingSession{
		Exercise: "bearoff",
		Items: []storage.TrainingItem{
			{NumberType: "epc.bottom", Wrong: true, HasDeviation: true, Deviation: 2},
			{NumberType: "epc.bottom", Wrong: true, HasDeviation: true, Deviation: -4},
			{NumberType: "epc.bottom", Wrong: true, HasDeviation: false},
		},
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	stats, err := tr.NumberStats(ctx, "", "bearoff")
	if err != nil {
		t.Fatalf("NumberStats: %v", err)
	}
	if len(stats) != 1 {
		t.Fatalf("NumberStats: %d types, want 1", len(stats))
	}
	got := stats[0]
	if got.Asked != 3 || got.Faults != 3 {
		t.Errorf("epc.bottom = %+v, want 3 asked / 3 faults", got)
	}
	if got.Deviations != 2 {
		t.Errorf("epc.bottom Deviations = %d, want 2: the unanswered number carries none", got.Deviations)
	}
	if got.MeanDeviation < 2.999 || got.MeanDeviation > 3.001 {
		t.Errorf("epc.bottom MeanDeviation = %v, want 3 (the mean of |2| and |-4|, the unanswered number excluded)", got.MeanDeviation)
	}
}

// A decision question keeps its position, its answer and its cost; Missed
// returns each position answered wrong once, the latest miss first, and
// forgets a position deleted since.
func testTrainingMissedPositions(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	tr := s.Training()
	var pos []int64
	for n := 1; n <= 4; n++ {
		p := provenancePos(n)
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save position %d: %v", n, err)
		}
		pos = append(pos, id)
	}
	mp := func(v int) *int { return &v }
	item := func(id int64, wrong bool, answer string, cost *int) storage.TrainingItem {
		return storage.TrainingItem{NumberType: "decision.checker", Wrong: wrong, PositionID: &id, Answer: answer, ErrorMp: cost}
	}
	first, err := tr.Save(ctx, "", storage.TrainingSession{Exercise: "decision", NumbersAsked: 3, Faults: 2,
		Items: []storage.TrainingItem{
			item(pos[0], true, "13/7 8/7", mp(120)),
			item(pos[1], false, "24/18", mp(0)),
			item(pos[2], true, "", nil), // out of time: wrong, nothing judged
		}})
	if err != nil {
		t.Fatalf("Save first: %v", err)
	}
	second, err := tr.Save(ctx, "", storage.TrainingSession{Exercise: "decision", NumbersAsked: 2, Faults: 2,
		Items: []storage.TrainingItem{
			item(pos[3], true, "Double/Take", mp(40)),
			item(pos[0], true, "13/7 6/5", mp(80)),
		}})
	if err != nil {
		t.Fatalf("Save second: %v", err)
	}
	// A number exercise has no position: it never enters the list.
	if _, err := tr.Save(ctx, "", storage.TrainingSession{Exercise: "pips", NumbersAsked: 1, Faults: 1,
		Items: []storage.TrainingItem{{NumberType: "pips.bottom", Wrong: true}}}); err != nil {
		t.Fatalf("Save pips: %v", err)
	}

	equal := func(name string, got, want []int64) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("%s = %v, want %v", name, got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("%s = %v, want %v", name, got, want)
			}
		}
	}
	all, err := tr.Missed(ctx, "", storage.TrainingMissedFilter{})
	if err != nil {
		t.Fatalf("Missed: %v", err)
	}
	equal("Missed(all)", all, []int64{pos[0], pos[3], pos[2]})
	one, err := tr.Missed(ctx, "", storage.TrainingMissedFilter{SessionID: first})
	if err != nil {
		t.Fatalf("Missed(first): %v", err)
	}
	equal("Missed(first)", one, []int64{pos[2], pos[0]})
	limited, err := tr.Missed(ctx, "", storage.TrainingMissedFilter{Exercise: "decision", Limit: 1})
	if err != nil {
		t.Fatalf("Missed(limit): %v", err)
	}
	equal("Missed(limit 1)", limited, []int64{pos[0]})
	none, err := tr.Missed(ctx, "", storage.TrainingMissedFilter{Exercise: "pips"})
	if err != nil {
		t.Fatalf("Missed(pips): %v", err)
	}
	equal("Missed(pips)", none, nil)

	if err := s.Positions().Delete(ctx, "", pos[3]); err != nil {
		t.Fatalf("Delete position: %v", err)
	}
	after, err := tr.Missed(ctx, "", storage.TrainingMissedFilter{SessionID: second})
	if err != nil {
		t.Fatalf("Missed(after delete): %v", err)
	}
	equal("Missed(after delete)", after, []int64{pos[0]})
}

// DecisionErrors lists the judged questions of the Decision exercise that still
// reach a position: out-of-time questions, number exercises and positions
// deleted since are left out.
func testTrainingDecisionErrors(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	tr := s.Training()
	p1, p2 := provenancePos(1), provenancePos(2)
	id1, err := s.Positions().Save(ctx, "", &p1)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}
	id2, err := s.Positions().Save(ctx, "", &p2)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}
	mp := func(v int) *int { return &v }
	item := func(id int64, cost *int) storage.TrainingItem {
		return storage.TrainingItem{NumberType: "decision.checker", PositionID: &id, ErrorMp: cost}
	}
	if _, err := tr.Save(ctx, "", storage.TrainingSession{Exercise: "decision", NumbersAsked: 3,
		Items: []storage.TrainingItem{item(id1, mp(120)), item(id1, nil), item(id2, mp(30))}}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := tr.Save(ctx, "", storage.TrainingSession{Exercise: "pips", NumbersAsked: 1,
		Items: []storage.TrainingItem{{NumberType: "pips.bottom"}}}); err != nil {
		t.Fatalf("Save pips: %v", err)
	}
	got, err := tr.DecisionErrors(ctx, "")
	if err != nil {
		t.Fatalf("DecisionErrors: %v", err)
	}
	if len(got) != 2 || got[0].ErrorMp != 120 || got[1].ErrorMp != 30 || got[0].CreatedAt == "" {
		t.Fatalf("DecisionErrors = %+v, want the two judged questions, oldest first", got)
	}
}
