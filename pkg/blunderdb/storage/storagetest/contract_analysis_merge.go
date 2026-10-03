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
		wrote, err := s.Analyses().Merge(ctx, "", posID, fn)
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
