package gui

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/race"
)

// ADR-0012: the race panel's third regime. racePosition(24, 1, ...) puts 15
// checkers a side on each colour's ace point: all home, far outside TS-06-06,
// so always "evaluated".

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

// Race ignores the dice, so it is populated alongside Moves too.
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

// ADR-0012: a cube verdict at a match score, which the "estimated" regime
// cannot give (ADR-0009).
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

// Inside the exact table's domain, "exact" is never displaced (ADR-0012).
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
