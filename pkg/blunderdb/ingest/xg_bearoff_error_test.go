package ingest

import (
	"math"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// TestXGBearOffPlayedMoveErrors pins two bear-offs of the #595 match whose
// played move used to be mislabelled (a -1 inside the move read as its end):
// the play must name its candidate and carry XG's error.
func TestXGBearOffPlayedMoveErrors(t *testing.T) {
	t.Parallel()
	g, err := MapXG(filepath.Join("..", "..", "..", "testdata", "Kev_GammonNetNormal_2026-10-06_7p.xg"))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]float64{engine.NormalizeMove("5/3 1/off(2)"): 0.0205, engine.NormalizeMove("3/off 2/off"): 0.0013}
	found := map[string]bool{}
	for _, game := range g.Games {
		for _, mv := range game.Moves {
			for _, a := range mv.Analyses {
				if a == nil || a.CheckerAnalysis == nil || len(a.PlayedMoves) == 0 {
					continue
				}
				err, ok := want[engine.NormalizeMove(a.PlayedMoves[0])]
				if !ok {
					continue
				}
				m := engine.PlayedCandidate(a.CheckerAnalysis.Moves, a.PlayedMoves[0])
				if m == nil {
					t.Errorf("%q names no candidate", a.PlayedMoves[0])
					continue
				}
				// The same play recurs elsewhere at no cost; the pinned one is
				// the occurrence that carries XG's error.
				if m.EquityError != nil && math.Abs(*m.EquityError-err) <= 0.0005 {
					found[engine.NormalizeMove(a.PlayedMoves[0])] = true
				}
			}
		}
	}
	if len(found) != len(want) {
		t.Errorf("plays found with XG's error: %v, want every one of %v", found, want)
	}
}
