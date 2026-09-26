package gui

import (
	"math"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/bearoffgen/bearofftest"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/race"
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

	result, err := (&App{}).evaluateGammonNet(pos, 0, 0, 0)
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

	result, err := (&App{}).evaluateGammonNet(pos, 0, 0, 0)
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

	// A ply past MaxPly clamps and the label names what ran. pruneK=1,
	// candidates=1 keep it cheap under the race detector.
	result, err := (&App{}).evaluateGammonNet(pos, 99, 1, 1)
	if err != nil {
		t.Fatalf("evaluateGammonNet: %v", err)
	}
	for _, m := range result.Moves {
		if m.AnalysisDepth == "99-ply" {
			t.Fatalf("AnalysisDepth reports the requested depth (99-ply), not the depth that ran")
		}
	}
}

// isolateRaceSources points the race resolver at the generated TS-06-06 only,
// not the developer's wider tables (an empty directory would mean no table,
// ADR-0027), and restores the package's shared directory afterwards.
func isolateRaceSources(t *testing.T) {
	t.Helper()
	shared := bearofftest.DataDir(t)
	race.SetDataDir(shared)
	race.SetExternalPath("")
	race.Invalidate()
	t.Cleanup(func() {
		race.SetDataDir(shared)
		race.SetExternalPath("")
		race.Invalidate()
	})
}

// The three race regimes as the panel receives them (ADR-0012, ADR-0017
// decision 4): exact at money leaves Race nil; an estimate, or exact at a
// match score, is replaced by the evaluated regime.
func TestEvaluateGammonNetRaceRegimes(t *testing.T) {
	isolateRaceSources(t)

	inside := func(score [2]int) domain.Position {
		// 3 checkers a side on the ace points: inside TS-06-06.
		pos := racePosition(24, 1, [2]int{0, 0}, domain.White)
		pos.Board.Points[24] = domain.Point{Checkers: 3, Color: domain.White}
		pos.Board.Points[1] = domain.Point{Checkers: 3, Color: domain.Black}
		pos.Board.Bearoff[domain.White] = 12
		pos.Board.Bearoff[domain.Black] = 12
		pos.Score = score
		return pos
	}
	outside := func(score [2]int) domain.Position {
		pos := racePosition(24, 1, [2]int{0, 0}, domain.White) // 15 a side: far outside
		pos.Score = score
		return pos
	}

	cases := []struct {
		name string
		pos  domain.Position
		fast race.Regime // what race.Evaluate says on its own
		gui  race.Regime // what the panel gets; "" = Race nil, keep the fast row
	}{
		{"exact, money", inside([2]int{-1, -1}), race.RegimeExact, ""},
		{"exact, at a score", inside([2]int{5, 7}), race.RegimeExact, race.RegimeEvaluated},
		{"estimated, money", outside([2]int{-1, -1}), race.RegimeEstimated, race.RegimeEvaluated},
		{"estimated, at a score", outside([2]int{5, 7}), race.RegimeEstimated, race.RegimeEvaluated},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fast := race.Evaluate(&c.pos)
			if fast.Race == nil || fast.Race.Regime != c.fast {
				t.Fatalf("race.Evaluate regime = %+v, want %q", fast.Race, c.fast)
			}
			if c.fast == race.RegimeEstimated && (fast.Race.Sigma <= 0 || fast.Race.P99 <= 0) {
				t.Errorf("an estimate without its error bounds: %+v", fast.Race)
			}
			if c.fast == race.RegimeExact && (fast.Race.Money == nil || fast.Race.SourceCheckers == 0) {
				t.Errorf("an exact lookup without its table or verdict: %+v", fast.Race)
			}

			result, err := (&App{}).evaluateGammonNet(c.pos, 0, 0, 0)
			if err != nil {
				t.Fatalf("evaluateGammonNet: %v", err)
			}
			if result.Refused {
				t.Fatal("refused a position the engine evaluates")
			}
			if c.gui == "" {
				if result.Race != nil {
					t.Fatalf("Race = %+v, want nil: the exact money row is not displaced", result.Race)
				}
				return
			}
			if result.Race == nil || result.Race.Regime != c.gui {
				t.Fatalf("Race = %+v, want regime %q", result.Race, c.gui)
			}
			if result.Race.OnRoll != fast.Race.OnRoll {
				t.Errorf("OnRoll %d, fast path says %d", result.Race.OnRoll, fast.Race.OnRoll)
			}
			if result.Race.Money == nil || result.Race.Money.Verdict == "" {
				t.Errorf("evaluated regime without a verdict: %+v", result.Race)
			}
			if result.Race.Depth != "0-ply" {
				t.Errorf("Depth %q, want the depth that ran", result.Race.Depth)
			}
			// In a symmetric bear-off both agree the roller is favourite.
			if (fast.Race.WinProb > 0.5) != (result.Race.WinProb > 0.5) {
				t.Errorf("WinProb %.3f (fast) and %.3f (evaluated) disagree on the favourite", fast.Race.WinProb, result.Race.WinProb)
			}
		})
	}
}

// A score past the MET horizon comes back as Refused alone (ADR-0019 rule 4).
func TestEvaluateGammonNetRefusesBeyondTheHorizon(t *testing.T) {
	isolateRaceSources(t)
	for _, dice := range [][2]int{{0, 0}, {6, 5}} {
		pos := racePosition(24, 1, dice, domain.White)
		pos.Score = [2]int{65, 65}

		result, err := (&App{}).evaluateGammonNet(pos, 0, 0, 0)
		if err != nil {
			t.Fatalf("dice %v: a refusal must be data, got an error: %v", dice, err)
		}
		if !result.Refused {
			t.Fatalf("dice %v: 65-away/65-away evaluated: %+v", dice, result)
		}
		if result.Moves != nil || result.Cube != nil || result.Race != nil || result.PreRoll != nil || result.CubeVerdict != "" {
			t.Errorf("dice %v: a refused result carries values: %+v", dice, result)
		}
	}
}

// TestEvaluateGammonNetRaceMatchesCubeAtMatchScore: the race regime and the
// Cube branch both use gammonnet.ConfigForPosition (ADR-0023), so their
// equities at a match score are bit-identical.
func TestEvaluateGammonNetRaceMatchesCubeAtMatchScore(t *testing.T) {
	pos := racePosition(24, 1, [2]int{0, 0}, domain.White)
	pos.Score = [2]int{5, 7} // White 5-away, Black 7-away — not Crawford, well within the MET horizon

	result, err := (&App{}).evaluateGammonNet(pos, 0, 0, 0)
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
