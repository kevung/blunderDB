package gui

import (
	"math"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// racePosition puts every checker of each colour on one point — enough
// structure for a legal 0-ply search without needing the full opening board.
func racePosition(whitePoint, blackPoint int, dice [2]int, onRoll int) domain.Position {
	var p domain.Position
	p.Board.Points[whitePoint] = domain.Point{Checkers: 15, Color: domain.White}
	p.Board.Points[blackPoint] = domain.Point{Checkers: 15, Color: domain.Black}
	p.PlayerOnRoll = onRoll
	p.Dice = dice
	p.Score = [2]int{-1, -1} // money
	p.Cube = domain.Cube{Owner: domain.None, Value: 0}
	return p
}

func TestEvaluateGammonNetMovesCarriesNotationAndDepthLabel(t *testing.T) {
	pos := racePosition(24, 1, [2]int{6, 5}, domain.White)

	result, err := evaluateGammonNet(pos, 0, 0, 0)
	if err != nil {
		t.Fatalf("evaluateGammonNet: %v", err)
	}
	if len(result.Moves) == 0 {
		t.Fatal("expected candidate moves for a position with dice set")
	}
	if result.Cube != nil {
		t.Error("dice were set: no cube decision should be returned")
	}
	for _, m := range result.Moves {
		if m.AnalysisDepth != "0-ply" {
			t.Errorf("move %q: AnalysisDepth = %q, want %q (the depth that actually ran)", m.Move, m.AnalysisDepth, "0-ply")
		}
		if m.AnalysisEngine != gammonNetEngineVersion {
			t.Errorf("move %q: AnalysisEngine = %q, want %q", m.Move, m.AnalysisEngine, gammonNetEngineVersion)
		}
		if strings.TrimSpace(m.Move) == "" {
			t.Errorf("candidate %d has no notation — did it fail to match any domain.LegalMoves play?", m.Index)
		}
	}
}

func TestEvaluateGammonNetCubeDecisionMoneyNoDice(t *testing.T) {
	pos := racePosition(6, 19, [2]int{0, 0}, domain.White)

	result, err := evaluateGammonNet(pos, 0, 0, 0)
	if err != nil {
		t.Fatalf("evaluateGammonNet: %v", err)
	}
	if result.Moves != nil {
		t.Error("no dice were set: no candidate moves should be returned")
	}
	if result.Cube == nil {
		t.Fatal("expected a cube decision for a position with no dice set")
	}
	if result.Cube.AnalysisDepth != "0-ply" {
		t.Errorf("AnalysisDepth = %q, want %q", result.Cube.AnalysisDepth, "0-ply")
	}
	if result.Cube.AnalysisEngine != gammonNetEngineVersion {
		t.Errorf("AnalysisEngine = %q, want %q", result.Cube.AnalysisEngine, gammonNetEngineVersion)
	}
	switch result.Cube.BestCubeAction {
	case "No Double", "Double, Take", "Double, Pass":
	default:
		t.Errorf("BestCubeAction = %q, not one of the three the frontend understands", result.Cube.BestCubeAction)
	}
}

func TestEvaluateGammonNetDepthLabelReflectsWhatRan(t *testing.T) {
	pos := racePosition(24, 1, [2]int{6, 5}, domain.White)

	// A ply far past MaxPly must clamp — the label must say what actually
	// ran (DefaultConfig's own clamp), never the requested value: #125's
	// non-negotiable rule.
	// pruneK=1, candidates=1: the label rule is about the clamp, not the
	// search — a full 2-ply on a race position costs five minutes under the
	// race detector and proves nothing more.
	result, err := evaluateGammonNet(pos, 99, 1, 1)
	if err != nil {
		t.Fatalf("evaluateGammonNet: %v", err)
	}
	for _, m := range result.Moves {
		if m.AnalysisDepth == "99-ply" {
			t.Fatalf("AnalysisDepth reports the requested depth (99-ply), not the depth that ran")
		}
	}
}

// TestEvaluateGammonNetRaceMatchesCubeAtMatchScore is #190/C.3 point 1's own
// regression. Before the fix, evaluateRaceRegime built its own plain
// gammonnet.DefaultConfig (money, cubeless) and only fed the match state and
// cube owner to Decide at the very end — a verdict tariffed at the score,
// read off a distribution that was never searched with the score, or the
// cube, in view (ADR-0023 already has a name for that mismatch: "Open").
// evaluateGammonNet's main Cube branch (evaluateMoves/evaluateCube in
// package gammonnet) already went through gammonnet.ConfigForPosition; now
// evaluateRaceRegime does too, on the SAME position, so the two must land on
// bit-identical equities (the parallel search is bit-identical by
// construction — CLAUDE.md's parallelism invariant — and at 0-ply here there
// is no ply-order sum to even reorder).
func TestEvaluateGammonNetRaceMatchesCubeAtMatchScore(t *testing.T) {
	pos := racePosition(24, 1, [2]int{0, 0}, domain.White)
	pos.Score = [2]int{5, 7} // White 5-away, Black 7-away — not Crawford, well within the MET horizon

	result, err := evaluateGammonNet(pos, 0, 0, 0)
	if err != nil {
		t.Fatalf("evaluateGammonNet: %v", err)
	}
	if result.Cube == nil {
		t.Fatal("expected a cube decision — no dice are set on the board")
	}
	if result.Race == nil || result.Race.Money == nil {
		t.Fatal("expected an evaluated-regime race verdict at this score (ADR-0012)")
	}

	const tol = 1e-9
	cube, money := result.Cube, result.Race.Money
	if diff := math.Abs(cube.CubefulNoDoubleEquity - money.NoDouble); diff > tol {
		t.Errorf("NoDouble disagrees between Cube (%.9f) and Race.Money (%.9f), diff %.2e — evaluateRaceRegime is not reading pos's own match/cube configuration", cube.CubefulNoDoubleEquity, money.NoDouble, diff)
	}
	if diff := math.Abs(cube.CubefulDoubleTakeEquity - money.DoubleTake); diff > tol {
		t.Errorf("DoubleTake disagrees between Cube (%.9f) and Race.Money (%.9f), diff %.2e", cube.CubefulDoubleTakeEquity, money.DoubleTake, diff)
	}
	if diff := math.Abs(cube.CubefulDoublePassEquity - money.DoublePass); diff > tol {
		t.Errorf("DoublePass disagrees between Cube (%.9f) and Race.Money (%.9f), diff %.2e", cube.CubefulDoublePassEquity, money.DoublePass, diff)
	}
}
