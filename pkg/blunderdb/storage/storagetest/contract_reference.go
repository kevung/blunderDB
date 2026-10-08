package storagetest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testStatsSuggestReferences pins the reference proposal (ADR-0078) on both
// backends: the six gammon errors share one board and the two missed doubles
// another, so each cluster yields one reference standing for all its errors;
// a position marked studied covers its cluster; the match narrowing holds.
func testStatsSuggestReferences(t *testing.T, s storage.Storage) {
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
	gammon := map[int64]bool{}
	for slot := 0; slot < 6; slot++ {
		gammon[statsCheckerDecision(t, s, gameID, slot, 1, "24/18 13/11", 0.100+0.020*float64(slot), 8)] = true
	}
	statsCubeDecision(t, s, gameID, 6, 1, "No Double", "Double, Take", 0.40, 0.55, 1.00)
	statsCubeDecision(t, s, gameID, 7, 1, "No Double", "Double, Take", 0.40, 0.60, 1.00)

	alice := storage.StatsFilter{DecisionType: -1, PlayerName: "Alice"}
	got, err := s.Stats().SuggestReferences(ctx, "", storage.ReferenceRequest{Filter: alice})
	if err != nil {
		t.Fatalf("SuggestReferences: %v", err)
	}
	if got.Size != storage.ReferenceDefaultSize || got.Radius != storage.ReferenceRadius || got.NumDecisions != 8 {
		t.Errorf("Size %d, Radius %d, NumDecisions %d", got.Size, got.Radius, got.NumDecisions)
	}
	if len(got.References) != 2 {
		t.Fatalf("References = %+v, want one per cluster", got.References)
	}
	var checker, cube storage.ReferenceSuggestion
	for _, r := range got.References {
		if r.Kind == "cube" {
			cube = r
		} else {
			checker = r
		}
	}
	if !gammon[checker.PositionID] || checker.Theme != "gammon" || checker.Errors != 6 || checker.Matches != 1 ||
		checker.FamilyErrors != 6 || checker.MatchID != matchID || checker.Label == "" || checker.Gain <= 0 {
		t.Errorf("checker reference = %+v, want a gammon position standing for the six errors", checker)
	}
	if cube.Theme != storage.CubeCellOfferMissed || cube.Errors != 2 || cube.AwayOnRoll == 0 {
		t.Errorf("cube reference = %+v, want the missed double at its score, standing for two errors", cube)
	}

	if err := s.ImportBatches().SetStudied(ctx, "", checker.PositionID, true); err != nil {
		t.Fatalf("SetStudied: %v", err)
	}
	got, err = s.Stats().SuggestReferences(ctx, "", storage.ReferenceRequest{Filter: alice, Size: 10})
	if err != nil {
		t.Fatalf("SuggestReferences after study: %v", err)
	}
	if len(got.References) != 1 || got.References[0].Kind != "cube" || got.Handled != 1 {
		t.Errorf("after study: Handled %d, References %+v; want the cube reference alone", got.Handled, got.References)
	}

	got, err = s.Stats().SuggestReferences(ctx, "", storage.ReferenceRequest{Filter: alice, MatchIDs: []int64{matchID + 1000}})
	if err != nil {
		t.Fatalf("SuggestReferences(other match): %v", err)
	}
	if got.Candidates+got.Handled != 0 || len(got.References) != 0 {
		t.Errorf("another match proposes %+v", got)
	}
}
