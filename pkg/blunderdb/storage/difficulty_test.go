package storage

import (
	"math"
	"testing"
)

func TestReferenceExpectedLoss(t *testing.T) {
	const tau = DifficultyTemperature
	cases := []struct {
		name  string
		costs []float64
		want  float64
	}{
		{"a forced decision costs nothing", []float64{0}, 0},
		{"no option costs nothing", nil, 0},
		{"two equal options cost nothing", []float64{0, 0}, 0},
		{"binary: Δ/(1+e^{Δ/τ})", []float64{0, 0.032}, 0.032 / (1 + math.Exp(0.032/tau))},
		{"an obvious decision weighs almost nothing", []float64{0, 0.1}, 0.1 / (1 + math.Exp(4))},
		{"the best need not be listed first", []float64{0.05, 0.018}, 0.032 / (1 + math.Exp(0.032/tau))},
		{"five candidates spaced 0.01", []float64{0, 0.01, 0.02, 0.03, 0.04}, func() float64 {
			var w, e float64
			for k := 0; k < 5; k++ {
				x := math.Exp(-0.4 * float64(k))
				w += x
				e += x * 0.01 * float64(k)
			}
			return e / w
		}()},
	}
	for _, c := range cases {
		if got := ReferenceExpectedLoss(c.costs, tau); math.Abs(got-c.want) > 1e-12 {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
	// ADR-0075's anchors: the hardest binary gap is near 1.28τ, five
	// candidates spaced 0.01 sit near PR 6.
	if got := ReferenceExpectedLoss([]float64{0, 0.01, 0.02, 0.03, 0.04}, tau); math.Abs(got-0.0125) > 0.0002 {
		t.Errorf("five candidates: got %v, want ≈ 0.0125", got)
	}
	best, bestGap := 0.0, 0.0
	for gap := 0.001; gap < 0.1; gap += 0.0005 {
		if d := ReferenceExpectedLoss([]float64{0, gap}, tau); d > best {
			best, bestGap = d, gap
		}
	}
	if math.Abs(bestGap-1.278*tau) > 0.001 {
		t.Errorf("hardest binary gap %v, want ≈ %v", bestGap, 1.278*tau)
	}
}

func TestIsAvoidable(t *testing.T) {
	if !IsAvoidable(0.02, 0.002, true) {
		t.Error("a difficulty of a tenth of the loss is avoidable")
	}
	if IsAvoidable(0.02, 0.0021, true) {
		t.Error("above a tenth of the loss it is not")
	}
	if IsAvoidable(0.02, 0, false) {
		t.Error("below the error threshold nothing is avoidable")
	}
	if IsAvoidable(0, 0, true) {
		t.Error("a play that cost nothing is no error")
	}
}

func TestSummariseDifficulty(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	losses := []DecisionLoss{
		{Player: 0, MWCLoss: f(0.03), Difficulty: f(0.002), Avoidable: true},
		{Player: 0, MWCLoss: f(0.01), Difficulty: f(0.008)},
		{Player: 0, MWCLoss: f(0.5)},               // no difficulty: left out of both sums
		{Player: 1, MWCLoss: nil, Difficulty: nil}, // unscored
		{Player: 1, MWCLoss: f(0.001), Difficulty: f(0.002)},
	}
	got := SummariseDifficulty(losses)
	p1 := got[0]
	if p1.Decisions != 2 || p1.Avoidable != 1 || math.Abs(p1.Loss-0.04) > 1e-12 || math.Abs(p1.Difficulty-0.01) > 1e-12 {
		t.Fatalf("player 1: %+v", p1)
	}
	if math.Abs(p1.Excess-0.03) > 1e-12 || p1.Ratio == nil || math.Abs(*p1.Ratio-4) > 1e-12 {
		t.Errorf("player 1 excess/ratio: %+v", p1)
	}
	p2 := got[1]
	if p2.Decisions != 1 || math.Abs(p2.Excess+0.001) > 1e-12 {
		t.Errorf("player 2: %+v", p2)
	}
	if p2.Ratio != nil {
		t.Errorf("a total difficulty under the floor gives no ratio, got %v", *p2.Ratio)
	}
}
