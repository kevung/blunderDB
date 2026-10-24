package storagetest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testStatsDegenerateCubeLabelNotCounted pins that a cube decision counts by
// what its played action means, not by which spellings it is not: a label the
// XG importer writes for a code it cannot map ("Unknown(D=…,T=…)") is neither a
// double, a take nor a pass, so it stays out of the PR denominator as a quiet
// no-double does — on the live statistics and on the per-player table alike.
func testStatsDegenerateCubeLabelNotCounted(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	if _, err := s.Stats().DateRange(ctx, ""); errors.Is(err, storage.ErrInternal) {
		t.Skip("Stats not implemented on this backend")
	}
	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7,
		MatchDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)}
	matchID, err := s.Matches().Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}
	gameID, err := s.Matches().CreateGame(ctx, "", &domain.Game{MatchID: matchID, GameNumber: 1})
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	// One real double, then the two degenerate spellings of the XG importer,
	// all on a position where no double is clearly right: not a close decision.
	statsCubeDecision(t, s, gameID, 0, 1, "Double", "No Double", 0.90, 0.30, 1.00)
	statsCubeDecision(t, s, gameID, 1, 1, "Unknown(D=2,T=0)", "No Double", 0.90, 0.30, 1.00)
	statsCubeDecision(t, s, gameID, 2, 1, "Unknown(7)", "No Double", 0.90, 0.30, 1.00)

	res, err := s.Stats().Compute(ctx, "", storage.StatsFilter{DecisionType: -1})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if res.Totals.NumDecisions != 1 {
		t.Errorf("Compute counts %d decisions, want 1: a degenerate cube label entered the denominator",
			res.Totals.NumDecisions)
	}
	rows, err := s.Stats().PlayerTable(ctx, "", storage.StatsFilter{DecisionType: -1})
	if err != nil {
		t.Fatalf("PlayerTable: %v", err)
	}
	for _, r := range rows {
		if r.Name == "Alice" && r.CubeDecisions != 1 {
			t.Errorf("PlayerTable gives Alice %d cube decisions, want 1", r.CubeDecisions)
		}
	}
}
