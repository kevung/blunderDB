package gui

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/race"
)

// #126 (ADR-0012): the race panel's third regime. racePosition (defined in
// gammonnet_eval_test.go) puts every checker of each colour on one point —
// White bears off toward point 24 (home 19..24), Black toward point 1 (home
// 1..6, race/epc.go's computeSide). racePosition(24, 1, ...) puts 15 a side
// on each colour's own ace-equivalent point: both players AllInHome, and far
// outside the embedded TS-06-06's 1..6-checker exact domain — guaranteed
// "evaluated" (or, pre-#126, "estimated" with no verdict at all), never
// "exact".

func TestEvaluateGammonNetRaceEvaluatedRegimeNoDice(t *testing.T) {
	pos := racePosition(24, 1, [2]int{0, 0}, domain.White)

	result, err := (&App{}).evaluateGammonNet(pos, 0, 0, 0)
	if err != nil {
		t.Fatalf("evaluateGammonNet: %v", err)
	}
	if result.Race == nil {
		t.Fatal("expected a race evaluation for a pure-bearoff position outside the exact domain")
	}
	if result.Race.Regime != race.RegimeEvaluated {
		t.Errorf("Regime = %q, want %q", result.Race.Regime, race.RegimeEvaluated)
	}
	if result.Race.Depth != "0-ply" {
		t.Errorf("Depth = %q, want the depth that actually ran (\"0-ply\")", result.Race.Depth)
	}
	if result.Race.Money == nil {
		t.Fatal("evaluated regime must carry a money cube verdict — that is the whole point of #126")
	}
	switch result.Race.Money.Verdict {
	case race.VerdictNoDouble, race.VerdictDoubleTake, race.VerdictDoublePass, race.VerdictTooGood:
	default:
		t.Errorf("Verdict = %q, not one of the four race.Verdict values", result.Race.Money.Verdict)
	}
}

// The race section ignores dice on the board (race.Evaluate's own contract),
// so it must still populate Race even when the primary result is candidate
// moves (dice set) rather than a cube decision.
func TestEvaluateGammonNetRacePopulatedAlongsideMoves(t *testing.T) {
	pos := racePosition(24, 1, [2]int{6, 5}, domain.White)

	result, err := (&App{}).evaluateGammonNet(pos, 0, 0, 0)
	if err != nil {
		t.Fatalf("evaluateGammonNet: %v", err)
	}
	if len(result.Moves) == 0 {
		t.Fatal("expected candidate moves for a position with dice set")
	}
	if result.Race == nil {
		t.Fatal("expected the race evaluation alongside the candidate moves — dice on the board are ignored for the race question")
	}
	if result.Race.Regime != race.RegimeEvaluated {
		t.Errorf("Regime = %q, want %q", result.Race.Regime, race.RegimeEvaluated)
	}
}

// The capability ADR-0009 refused and ADR-0012 unlocks: a cube verdict at a
// match score, which the convolution-only "estimated" regime could never
// offer (its chain, p -> MET, was the thing ADR-0009 measured and rejected).
func TestEvaluateGammonNetRaceEvaluatedAtMatchScore(t *testing.T) {
	pos := racePosition(24, 1, [2]int{0, 0}, domain.White)
	pos.Score = [2]int{5, 7} // White 5-away, Black 7-away — not Crawford

	result, err := (&App{}).evaluateGammonNet(pos, 0, 0, 0)
	if err != nil {
		t.Fatalf("evaluateGammonNet: %v", err)
	}
	if result.Race == nil || result.Race.Regime != race.RegimeEvaluated {
		t.Fatalf("expected an evaluated-regime race verdict at a match score, got %+v", result.Race)
	}
	if result.Race.Money == nil || result.Race.Money.Verdict == "" {
		t.Error("expected a cube verdict at the score — this is exactly what ADR-0009 could not offer and ADR-0012 unlocks")
	}
}

// A position inside the exact table's domain must keep showing "exact" — the
// evaluated regime never displaces it (ADR-0012: "it wins wherever it is
// available, and nothing displaces it").
func TestEvaluateGammonNetRaceExactDomainStaysExact(t *testing.T) {
	// 3 checkers a side on their ace point: well inside TS-06-06 (1..6).
	pos := racePosition(24, 1, [2]int{0, 0}, domain.White)
	pos.Board.Points[24] = domain.Point{Checkers: 3, Color: domain.White}
	pos.Board.Points[1] = domain.Point{Checkers: 3, Color: domain.Black}
	pos.Board.Bearoff[domain.White] = 12 // the other 12 already off — 15 total, structurally valid
	pos.Board.Bearoff[domain.Black] = 12

	result, err := (&App{}).evaluateGammonNet(pos, 0, 0, 0)
	if err != nil {
		t.Fatalf("evaluateGammonNet: %v", err)
	}
	if result.Race != nil {
		t.Errorf("exact-domain position must not get an evaluated-regime Race field, got %+v", result.Race)
	}
}
