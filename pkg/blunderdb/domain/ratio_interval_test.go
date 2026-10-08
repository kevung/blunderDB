package domain

import (
	"math"
	"testing"
)

// Units that all share one ratio leave no spread: the interval collapses on
// the ratio. A single unit brings no interval at all.
func TestRatioPoolInterval(t *testing.T) {
	var one RatioPool
	one.Add(10, 5)
	if iv := one.Interval(1); iv.Available || iv.Units != 1 {
		t.Fatalf("one unit: %+v, want no interval", iv)
	}

	var flat RatioPool
	flat.Add(10, 5)
	flat.Add(20, 10)
	iv := flat.Interval(0.5)
	if !iv.Available || math.Abs(iv.Low-1) > 1e-12 || math.Abs(iv.High-1) > 1e-12 {
		t.Fatalf("same ratio: %+v, want [1, 1]", iv)
	}

	// Ratios 0 and 2 over equal weights: R = 1, residuals ∓1·10, so
	// SE = √(2·200)/20 = 1 and the bounds are 1 ∓ 1.96, the low one floored.
	var spread RatioPool
	spread.Add(0, 10)
	spread.Add(20, 10)
	spread.Add(0, 0) // no decision: not a unit
	iv = spread.Interval(1)
	if iv.Units != 2 || iv.Low != 0 || math.Abs(iv.High-2.96) > 1e-12 {
		t.Fatalf("spread: %+v, want [0, 2.96] over 2 units", iv)
	}
}
