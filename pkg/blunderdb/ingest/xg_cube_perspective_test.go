package ingest

import (
	"math"
	"testing"
)

// TestMapXGCubeResponseChancesAreFlipped pins the fix for a real bug found
// while building gammonNet's integration gate (#123): mapDoubleTakeMove and
// mapSingleCubeMove each synthesize a companion position for the
// responder's Take/Pass, correctly flipping PlayerOnRoll to the opponent —
// but were reattaching the doubler's own DoublingCubeAnalysis unchanged, so
// the companion node's "Player" win/gammon/backgammon chances stayed the
// doubler's, now mislabelled as the responder's.
//
// Every doubler/responder pair in the fixture must have complementary win
// chances (they describe the same game from opposite sides), and gammon
// chances must have swapped roles entirely (a responder's OWN gammon chance
// is the doubler's OPPONENT gammon chance, not its own).
func TestMapXGCubeResponseChancesAreFlipped(t *testing.T) {
	g, err := MapXG(xgLuckFixture())
	if err != nil {
		t.Fatalf("MapXG: %v", err)
	}

	pairs := 0
	for _, game := range g.Games {
		var doubler *MoveGraph
		for i := range game.Moves {
			mv := &game.Moves[i]
			if mv.Move.MoveType != "cube" {
				doubler = nil
				continue
			}
			switch mv.Move.CubeAction {
			case "Double":
				doubler = mv
			case "Take", "Pass":
				if doubler == nil || len(doubler.Analyses) == 0 || len(mv.Analyses) == 0 {
					continue
				}
				dDCA := doubler.Analyses[0].DoublingCubeAnalysis
				rDCA := mv.Analyses[0].DoublingCubeAnalysis
				if dDCA == nil || rDCA == nil {
					continue
				}
				pairs++

				if math.Abs((dDCA.PlayerWinChances+rDCA.PlayerWinChances)-100) > 0.01 {
					t.Errorf("%s: doubler PlayerWinChances=%.4f, responder PlayerWinChances=%.4f, want complementary (sum 100)",
						mv.Move.CubeAction, dDCA.PlayerWinChances, rDCA.PlayerWinChances)
				}
				if math.Abs(dDCA.PlayerGammonChances-rDCA.OpponentGammonChances) > 0.01 {
					t.Errorf("%s: doubler PlayerGammonChances=%.4f != responder OpponentGammonChances=%.4f",
						mv.Move.CubeAction, dDCA.PlayerGammonChances, rDCA.OpponentGammonChances)
				}
				doubler = nil
			default:
				doubler = nil
			}
		}
	}
	if pairs == 0 {
		t.Fatal("no doubler/responder pair found in the fixture — test is not exercising anything")
	}
	t.Logf("%d doubler/responder pairs checked", pairs)
}
