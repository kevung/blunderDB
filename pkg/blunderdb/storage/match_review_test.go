package storage

import (
	"math"
	"testing"
)

func f(v float64) *float64 { return &v }
func i64(v int64) *int64   { return &v }

// A finished match: player 1 lost it, but was dealt the worse dice, so the
// adjusted result is positive; the pace splits errors at the player's median
// time; the review ranks by loss less difficulty.
func TestBuildMatchReview(t *testing.T) {
	d := []DecisionLoss{
		{MoveID: 1, Rolled: true, GameNumber: 1, MoveNumber: 1, Player: 0, DecisionType: "checker", ErrorMP: i64(0), MWCLoss: f(0), Luck: f(-0.10), DurationMS: i64(10000)},
		{MoveID: 2, Rolled: true, GameNumber: 1, MoveNumber: 2, Player: 1, DecisionType: "checker", ErrorMP: i64(80), Error: true, MWCLoss: f(0.02), Luck: f(0.20), DurationMS: i64(3000)},
		{MoveID: 3, Rolled: true, GameNumber: 2, MoveNumber: 1, Player: 0, DecisionType: "checker", ErrorMP: i64(100), Error: true, MWCLoss: f(0.03), Difficulty: f(0.025), DurationMS: i64(2000)},
		{MoveID: 4, Rolled: true, GameNumber: 2, MoveNumber: 2, Player: 0, DecisionType: "checker", ErrorMP: i64(60), Error: true, MWCLoss: f(0.01), DurationMS: i64(20000)},
		{MoveID: 5, GameNumber: 2, MoveNumber: 3, Player: 1, DecisionType: "cube", ErrorMP: i64(0), MWCLoss: f(0)},
		// A checker row without its position can never carry luck: it is
		// not a roll the coverage misses.
		{MoveID: 6, GameNumber: 2, MoveNumber: 4, Player: 1, DecisionType: "checker"},
	}
	r := BuildMatchReview(9, d, 5, 0.5, -1)
	p := r.Players[0]
	l := p.Luck
	if !l.Available || l.Result != -0.5 || math.Abs(l.Luck+0.30) > 1e-12 || math.Abs(l.Adjusted+0.20) > 1e-12 {
		t.Errorf("luck: %+v, want result -0.5, luck -0.30, adjusted -0.20", l)
	}
	if l.Rolls != 4 || l.RollsMeasured != 2 || math.Abs(l.ErrorBalance-(0.02-0.04)) > 1e-12 {
		t.Errorf("luck coverage or balance: %+v", l)
	}
	if o := r.Players[1].Luck; math.Abs(o.Adjusted+l.Adjusted) > 1e-12 {
		t.Errorf("the two adjusted results should be opposite: %+v vs %+v", o, l)
	}
	// Move 4 keeps 0.01 avoidable, move 3 only 0.005.
	if len(p.ToReview) != 2 || p.ToReview[0].MoveID != 4 || p.ToReview[1].MoveID != 3 {
		t.Errorf("review order: %+v", p.ToReview)
	}
	// Durations 10 s, 2 s, 20 s: median 10 s, move 3 hasty, move 4 deliberate.
	if p.Pace.Hasty != 1 || p.Pace.Deliberate != 1 || p.Pace.HastyLoss != 0.03 {
		t.Errorf("pace: %+v", p.Pace)
	}
	// Two games are too few units for a band (domain.IntervalMinUnits).
	if p.Decisions != 3 || p.PRInterval.Available || p.PRInterval.Units != 2 || p.MWC7.HasInterval {
		t.Errorf("PR %v over %d, interval %+v, L7 %+v", p.PR, p.Decisions, p.PRInterval, p.MWC7)
	}
	for seat, want := range SummariseDifficulty(d) {
		if got := r.Players[seat].Difficulty; got.Decisions != want.Decisions || got.Excess != want.Excess || got.Avoidable != want.Avoidable {
			t.Errorf("seat %d difficulty %+v, want %+v", seat, got, want)
		}
	}
	if r.Players[0].Difficulty.Decisions != 1 {
		t.Errorf("player 1's difficulty covers move 3 only: %+v", r.Players[0].Difficulty)
	}
	if m := BuildMatchReview(9, d, 0, 0.5, 1); m.Players[0].Luck.Available {
		t.Error("money play has no luck-adjusted result")
	}
}
