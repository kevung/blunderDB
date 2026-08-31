// SPDX-License-Identifier: MIT

package gammonnet

import (
	"math"
	"testing"
)

// The scale's four defining properties (ADR-0019). Together they pin the map
// down completely: money is untouched, the current cube's two dry outcomes
// are ±1, the map negates with the point of view, and it degenerates to the
// search's own 2×MWC−1 exactly at double match point — the one score where
// the two scales that used to be printed interchangeably really do coincide.

func TestEquityScaleLeavesMoneyAlone(t *testing.T) {
	scale, ok := NewEquityScale(nil)
	if !ok {
		t.Fatal("money play must always have a referential")
	}
	if scale.IsMatch() {
		t.Fatal("money play reported as a match referential")
	}
	for _, v := range []float64{-3, -1.25, 0, 0.0555, 1, 2.4} {
		if got := scale.FromDecision(v); got != v {
			t.Errorf("FromDecision(%v) = %v, money must be the identity", v, got)
		}
		if got := scale.FromSearch(v); got != v {
			t.Errorf("FromSearch(%v) = %v, money must be the identity", v, got)
		}
	}
}

func TestEquityScaleAnchorsTheCubeAtPlusMinusOne(t *testing.T) {
	// Winning the cube's own value outright is +1 and losing it outright is
	// −1 at EVERY score — that is what makes a match equity comparable to a
	// money one, and what makes "double, pass" read +1.000 in the panel at a
	// score just as it does in money play.
	for _, state := range []MatchState{
		{AwayOnRoll: 5, AwayOpponent: 5, Cube: 1},
		{AwayOnRoll: 3, AwayOpponent: 7, Cube: 1},
		{AwayOnRoll: 2, AwayOpponent: 4, Cube: 2},
		{AwayOnRoll: 1, AwayOpponent: 1, Cube: 1},
		{AwayOnRoll: 1, AwayOpponent: 6, Cube: 4, Crawford: true},
	} {
		scale, ok := NewEquityScale(&state)
		if !ok {
			t.Fatalf("%+v: refused", state)
		}
		cash, _ := metAfter(state, state.Cube, true)
		pass, _ := metAfter(state, state.Cube, false)
		if got := scale.FromDecision(cash); math.Abs(got-1) > 1e-12 {
			t.Errorf("%+v: cashing the cube = %v, want +1", state, got)
		}
		if got := scale.FromDecision(pass); math.Abs(got+1) > 1e-12 {
			t.Errorf("%+v: conceding the cube = %v, want -1", state, got)
		}
	}
}

func TestEquityScaleNegatesWithThePointOfView(t *testing.T) {
	// The search negates values at every ply; a referential that did not
	// negate with it would put the two sides of one node on different
	// scales. mwc is the on-roll player's, so the opponent's is 1-mwc.
	for _, state := range []MatchState{
		{AwayOnRoll: 5, AwayOpponent: 5, Cube: 1},
		{AwayOnRoll: 3, AwayOpponent: 7, Cube: 2},
		{AwayOnRoll: 6, AwayOpponent: 2, Cube: 1},
	} {
		mine, ok := NewEquityScale(&state)
		if !ok {
			t.Fatalf("%+v: refused", state)
		}
		swapped := state.Swap()
		theirs, ok := NewEquityScale(&swapped)
		if !ok {
			t.Fatalf("%+v swapped: refused", state)
		}
		for _, mwc := range []float64{0.1, 0.35, 0.5, 0.72, 0.99} {
			got := theirs.FromDecision(1 - mwc)
			want := -mine.FromDecision(mwc)
			// Same 1e-5 margin as cubeGoldTolerance, and for the same
			// reason: blunderDB stores its MET in float32, so the two sides'
			// anchors are only symmetric to a float32 rounding — amplified
			// here by the 1/(cash-pass) the narrow scores carry.
			if math.Abs(got-want) > 1e-5 {
				t.Errorf("%+v at mwc=%v: opponent's equity %v, want %v", state, mwc, got, want)
			}
		}
	}
}

func TestEquityScaleCoincidesWithTheSearchScaleAtDMP(t *testing.T) {
	// Double match point is the whole reason the confusion survived: there,
	// and only there, cash = 1 and pass = 0, so 2×MWC−1 IS the normalised
	// equity. Anywhere else it is compressed by (cash − pass).
	dmp := MatchState{AwayOnRoll: 1, AwayOpponent: 1, Cube: 1}
	scale, ok := NewEquityScale(&dmp)
	if !ok {
		t.Fatal("DMP refused")
	}
	for _, mwc := range []float64{0.2, 0.5, 0.8} {
		searchValue := 2*mwc - 1
		if got := scale.FromSearch(searchValue); math.Abs(got-searchValue) > 1e-12 {
			t.Errorf("at DMP FromSearch(%v) = %v, want the identity", searchValue, got)
		}
	}

	// And the contrast that motivates the whole file: at 5-away/5-away the
	// same search value is worth several times more once stated in the
	// referential the panel prints.
	even := MatchState{AwayOnRoll: 5, AwayOpponent: 5, Cube: 1}
	evenScale, ok := NewEquityScale(&even)
	if !ok {
		t.Fatal("5-away/5-away refused")
	}
	searchValue := 2*0.6 - 1 // 60% MWC
	got := evenScale.FromSearch(searchValue)
	if math.Abs(got) <= math.Abs(searchValue) {
		t.Errorf("5-away/5-away: normalised %v is not larger in magnitude than the search's %v — the compression is still there", got, searchValue)
	}
}

func TestEquityScaleRefusesAnUnevaluableState(t *testing.T) {
	bad := MatchState{AwayOnRoll: 0, AwayOpponent: 3, Cube: 1} // away 0 is not a state, it is a sentinel
	if _, ok := NewEquityScale(&bad); ok {
		t.Fatal("an invalid match state must not yield a referential")
	}
}
