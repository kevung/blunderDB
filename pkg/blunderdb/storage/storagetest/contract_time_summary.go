package storagetest

import (
	"context"
	"slices"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcription"
)

// timedMatch writes, through the path a Duel's Match takes (a transcript
// document, its graph, ingest.WriteMatch), a match whose plays alternate
// between the two players, each with the durations given: the stored player
// codes are the ones the writer chooses, not ones typed in a fixture.
func timedMatch(t *testing.T, s storage.Storage, durations [][2]*int64) int64 {
	t.Helper()
	ctx := context.Background()
	doc := transcript.New(7)
	doc.Header.Player1, doc.Header.Player2 = "me", "them"
	rolls := [][2]int{{3, 1}, {6, 5}, {4, 2}, {2, 1}}
	for i, d := range durations {
		side := domain.Black
		if i%2 == 1 {
			side = domain.White
		}
		pos := transcript.Replay(doc, 0).Next.Position
		pos.Dice = rolls[i]
		pos.PlayerOnRoll = side
		plays := domain.LegalMoves(&pos)
		if len(plays) == 0 {
			t.Fatalf("no legal play for roll %d", i)
		}
		doc.Actions = append(doc.Actions, transcript.Action{Side: side, Kind: transcript.KindChecker,
			Dice: rolls[i], Steps: plays[0].Steps, DecisionMS: d[0], CubeDecisionMS: d[1]})
		doc.Cursor = len(doc.Actions)
	}
	parts := transcript.Build(doc)
	tx, err := s.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	res, err := ingest.WriteMatch(ctx, tx, "", transcription.MatchGraph(parts), nil)
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("WriteMatch: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	return res.MatchID
}

// testStatsMatchTimeSummary pins the per-player sum of decision times for both
// players of a Match written the way a Duel writes it: an unknown duration is
// counted apart and never as zero, and the overrun is the fact the origin
// records.
func testStatsMatchTimeSummary(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ms := func(v int64) *int64 { return &v }
	matchID := timedMatch(t, s, [][2]*int64{
		{ms(5000), ms(1000)}, // player 1
		{ms(4000), nil},      // player 2
		{nil, nil},           // player 1, unknown
		{ms(2000), nil},      // player 2
	})

	got, err := s.Stats().MatchTimeSummary(ctx, "", matchID)
	if err != nil {
		t.Fatalf("MatchTimeSummary: %v", err)
	}
	if got.HasCadence {
		t.Errorf("a Match not played here has no Cadence")
	}
	p1, p2 := got.Players[0], got.Players[1]
	if p1.TotalMS != 6000 || p1.CheckerCount != 1 || p1.CheckerTotalMS != 5000 || p1.CubeCount != 1 || p1.CubeTotalMS != 1000 || p1.Unknown != 1 {
		t.Errorf("player 1: %+v", p1)
	}
	if p2.TotalMS != 6000 || p2.CheckerCount != 2 || p2.CubeCount != 0 || p2.Unknown != 0 {
		t.Errorf("player 2: %+v", p2)
	}
	// Each known decision, checker or cube, enters one band: 1 s and 5 s for
	// player 1, 4 s and 2 s for player 2 — a bound opens the band above it.
	bands := func(at map[int]int) []int {
		h := make([]int, len(storage.DecisionTimeHistogramBounds)+1)
		for i, n := range at {
			h[i] = n
		}
		return h
	}
	if want := bands(map[int]int{0: 1, 2: 1}); !slices.Equal(p1.Histogram, want) {
		t.Errorf("player 1 histogram %v, want %v", p1.Histogram, want)
	}
	if want := bands(map[int]int{1: 2}); !slices.Equal(p2.Histogram, want) {
		t.Errorf("player 2 histogram %v, want %v", p2.Histogram, want)
	}
	if !slices.Equal(got.HistogramBoundsMS, storage.DecisionTimeHistogramBounds) {
		t.Errorf("histogram bounds %v", got.HistogramBoundsMS)
	}

	origin := storage.MatchOrigin{MatchID: matchID, DiceSeed: "s", Cadence: `{"reserve":10,"delay":2}`, OverTime: 2}
	if err := s.Duels().SetOrigin(ctx, "", &origin); err != nil {
		t.Fatalf("SetOrigin: %v", err)
	}
	got, err = s.Stats().MatchTimeSummary(ctx, "", matchID)
	if err != nil {
		t.Fatalf("MatchTimeSummary under a Cadence: %v", err)
	}
	if !got.HasCadence || got.Players[0].OverTime || !got.Players[1].OverTime {
		t.Errorf("over time: %+v", got)
	}
}
