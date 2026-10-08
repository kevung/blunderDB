package domain

import (
	"math"
	"testing"
)

// Units that all share one ratio leave no spread, and fewer than
// IntervalMinUnits units too little: neither has an interval.
func TestRatioPoolInterval(t *testing.T) {
	var one RatioPool
	one.Add(10, 5)
	if iv := one.Interval(1); iv.Available || iv.Units != 1 {
		t.Fatalf("one unit: %+v, want no interval", iv)
	}

	// Two one-decision matches of the same error: a cell that once read
	// "0.00 [0.00–0.00]" at full strength.
	var two RatioPool
	two.Add(0, 1)
	two.Add(0, 1)
	if iv := two.Interval(1); iv.Available || iv.Units != 2 {
		t.Fatalf("two units: %+v, want no interval", iv)
	}

	var flat RatioPool
	flat.Add(10, 5)
	flat.Add(20, 10)
	flat.Add(2, 1)
	if iv := flat.Interval(0.5); iv.Available || iv.Units != 3 {
		t.Fatalf("same ratio: %+v, want no interval", iv)
	}

	// Ratios 0, 1 and 2 over equal weights: R = 1, residuals −10, 0, 10, so
	// SE = √(3/2·200)/30 = √300/30 and the bounds are 1 ∓ 1.96·SE, the low
	// one floored at 0.
	var spread RatioPool
	spread.Add(0, 10)
	spread.Add(10, 10)
	spread.Add(20, 10)
	spread.Add(0, 0) // no decision: not a unit
	iv := spread.Interval(1)
	se := math.Sqrt(300) / 30
	if !iv.Available || iv.Units != 3 || iv.Low != 0 || math.Abs(iv.High-(1+1.96*se)) > 1e-12 {
		t.Fatalf("spread: %+v, want 1 ∓ %v over 3 units", iv, 1.96*se)
	}
}
