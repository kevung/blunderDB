package ingest

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// A NaN in one decision's analysis and an infinity in another's leave those
// two decisions unanalysed and counted; the match, every other analysis and
// every stored blob still go in.
func TestNonFiniteAnalysisIsDroppedNotTheMatch(t *testing.T) {
	ctx := context.Background()
	g, err := MapXG("../../../testdata/charlot1-charlot2_7p_2025-11-08-2305.xg")
	if err != nil {
		t.Fatalf("MapXG: %v", err)
	}
	var checkerHit, cubeHit bool
	for gi := range g.Games {
		for mi := range g.Games[gi].Moves {
			for _, a := range g.Games[gi].Moves[mi].Analyses {
				if a == nil {
					continue
				}
				switch {
				case !checkerHit && a.CheckerAnalysis != nil && len(a.CheckerAnalysis.Moves) > 0:
					a.CheckerAnalysis.Moves[0].Equity = math.NaN()
					checkerHit = true
				case !cubeHit && a.DoublingCubeAnalysis != nil:
					a.DoublingCubeAnalysis.PlayerWinChances = math.Inf(1)
					cubeHit = true
				}
			}
		}
	}
	if !checkerHit || !cubeHit {
		t.Fatalf("fixture lacks a checker or a cube analysis (checker %v, cube %v)", checkerHit, cubeHit)
	}

	s, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	defer s.Close()
	res := writeGraph(t, s, g)
	if res.DroppedAnalyses != 2 {
		t.Errorf("DroppedAnalyses = %d, want 2", res.DroppedAnalyses)
	}
	if res.MatchID == 0 || res.SavedPositions == 0 {
		t.Fatalf("match not imported: %+v", res)
	}

	// Load, not LoadMany: LoadMany leaves an undecodable blob out silently.
	stored := map[int64]*domain.PositionAnalysis{}
	// The list is drained first: ":memory:" has a single connection, which a
	// Load nested in the iterator would wait on forever.
	var ids []int64
	for p, err := range s.Positions().List(ctx, "", storage.ListOpts{}) {
		if err != nil {
			t.Fatalf("list positions: %v", err)
		}
		ids = append(ids, p.ID)
	}
	for _, id := range ids {
		a, err := s.Analyses().Load(ctx, "", id)
		switch {
		case errors.Is(err, storage.ErrNotFound):
			continue
		case err != nil:
			t.Fatalf("Load analysis of position %d: %v", id, err)
		}
		stored[id] = a
	}
	if len(stored) == 0 {
		t.Fatal("no analysis stored")
	}
	for id, a := range stored {
		if a.CheckerAnalysis != nil {
			for _, m := range a.CheckerAnalysis.Moves {
				if math.IsNaN(m.Equity) || math.IsInf(m.Equity, 0) {
					t.Errorf("position %d stored a non-finite equity", id)
				}
			}
		}
	}
}
