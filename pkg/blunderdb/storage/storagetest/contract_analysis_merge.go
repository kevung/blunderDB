// Contract case for AnalysisStore.Merge. The table that runs it lives in
// contract.go.
package storagetest

import (
	"context"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testAnalysisMergeSkipsAnIdenticalResult: Merge hands the stored analysis
// to its callback, writes what comes back, and writes nothing when the result
// differs from the stored one only by LastModifiedDate — the case of every
// re-import of an already-stored match. The proof that nothing was written is
// the stored LastModifiedDate, which an upsert would have replaced.
func testAnalysisMergeSkipsAnIdenticalResult(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	p := checkerPos()
	posID, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}
	first := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	analysis := func(move string, modified time.Time) *domain.PositionAnalysis {
		return &domain.PositionAnalysis{
			AnalysisType:     "CheckerMove",
			LastModifiedDate: modified,
			CheckerAnalysis: &domain.CheckerAnalysis{
				Moves: []domain.CheckerMove{{Index: 0, Move: move, Equity: 0.1234}},
			},
			PlayedMoves: []string{move},
		}
	}
	merge := func(fn func(*domain.PositionAnalysis) *domain.PositionAnalysis) bool {
		t.Helper()
		wrote, err := s.Analyses().Merge(ctx, "", posID, nil, fn)
		if err != nil {
			t.Fatalf("Merge: %v", err)
		}
		return wrote
	}

	if !merge(func(existing *domain.PositionAnalysis) *domain.PositionAnalysis {
		if existing != nil {
			t.Errorf("first Merge got an existing analysis: %+v", existing)
		}
		return analysis("13/11 24/23", first)
	}) {
		t.Fatal("first Merge wrote nothing")
	}

	if merge(func(existing *domain.PositionAnalysis) *domain.PositionAnalysis {
		if existing == nil || existing.CheckerAnalysis == nil {
			t.Fatalf("second Merge did not get the stored analysis: %+v", existing)
		}
		return analysis("13/11 24/23", first.Add(time.Hour))
	}) {
		t.Error("an identical analysis (LastModifiedDate aside) was rewritten")
	}
	got, err := s.Analyses().Load(ctx, "", posID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !got.LastModifiedDate.Equal(first) {
		t.Errorf("LastModifiedDate = %v, want %v: the identical merge wrote", got.LastModifiedDate, first)
	}

	if merge(func(*domain.PositionAnalysis) *domain.PositionAnalysis { return nil }) {
		t.Error("a nil merge result reported a write")
	}

	if !merge(func(*domain.PositionAnalysis) *domain.PositionAnalysis {
		return analysis("8/5 6/5", first.Add(2*time.Hour))
	}) {
		t.Fatal("a changed analysis was not written")
	}
	got, err = s.Analyses().Load(ctx, "", posID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.CheckerAnalysis == nil || got.CheckerAnalysis.Moves[0].Move != "8/5 6/5" {
		t.Errorf("changed analysis not stored: %+v", got.CheckerAnalysis)
	}
}

// testAnalysisMergeTakesPlayedFromTheCaller: the played actions Merge is
// given fill the columns where the analysis names none, in place of the
// move table; nil reads the move table. The move table here says the
// second-best move was played, the caller says the best: RepairDenormalised-
// Columns, which always reads the move table, tells which source the columns
// came from.
func testAnalysisMergeTakesPlayedFromTheCaller(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	matchID, err := s.Matches().Save(ctx, "", &domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7})
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}
	gameID, err := s.Matches().CreateGame(ctx, "", &domain.Game{MatchID: matchID, GameNumber: 1})
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	const best, second = "8/6 6/5", "13/11 24/23"
	p := checkerPos()
	posID, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}
	if _, err := s.Matches().CreateMove(ctx, "", &domain.Move{GameID: gameID, MoveNumber: 1, MoveType: "checker",
		PositionID: posID, Player: 1, CheckerMove: second}); err != nil {
		t.Fatalf("CreateMove: %v", err)
	}
	loss := 0.080
	analysis := func(*domain.PositionAnalysis) *domain.PositionAnalysis {
		return &domain.PositionAnalysis{
			AnalysisType: "CheckerMove",
			CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
				{Index: 0, Move: best, Equity: 0.400},
				{Index: 1, Move: second, Equity: 0.320, EquityError: &loss},
			}},
		}
	}

	if _, err := s.Analyses().Merge(ctx, "", posID, &storage.PlayedActions{CheckerMove: best}, analysis); err != nil {
		t.Fatalf("Merge with played: %v", err)
	}
	if n, err := s.Analyses().RepairDenormalisedColumns(ctx, ""); err != nil || n != 1 {
		t.Errorf("after a Merge given the best move, repair changed %d rows (err %v), want 1: the columns read the move table", n, err)
	}

	if err := s.Analyses().Delete(ctx, "", posID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Analyses().Merge(ctx, "", posID, nil, analysis); err != nil {
		t.Fatalf("Merge without played: %v", err)
	}
	if n, err := s.Analyses().RepairDenormalisedColumns(ctx, ""); err != nil || n != 0 {
		t.Errorf("after a Merge without played, repair changed %d rows (err %v), want 0: the columns ignored the move table", n, err)
	}
}
