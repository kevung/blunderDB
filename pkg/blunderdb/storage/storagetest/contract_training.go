package storagetest

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The Training journal's contract (ADR-0040 rule 6, issue #320).
//
// Three things are worth holding both backends to, and they are the three
// this file checks: a session and its items are written as one thing, the
// per-number aggregate counts by TYPE and not by session, and a number that
// carries no deviation stays out of the mean instead of entering it as a zero
// error — which is what separates "no answer was given" from "the answer was
// exactly right".

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
