// SPDX-License-Identifier: MIT

package gammonnet

import (
	"math"
	"testing"
)

// FuzzDecide drives Decide and Value over arbitrary distributions and match
// states: any five float32 in [0, 1] (nested or not), any owner, any away
// scores (0, negative, past the MET's horizon), any cube value, both Crawford
// flags, Jacoby, any efficiency in [0, 1].
//
// It asserts the contract: no panic; a valid state is never refused and an
// invalid one always is; every equity is finite; the verdict is one of the
// four, and where doubling is off the table (cube against, Crawford game) it
// is NoDouble with the double branch priced at the no-double value. For a
// NESTED distribution the MWC lies in [0, 1] and Value in [-1, 1]. A
// non-nested one is only finite: probsExclusive floors negative masses but
// renormalises nothing, as gn_probs_exclusive does (seed below).
//
// NaN, infinities and values outside [0, 1] are skipped: the sigmoid never
// emits them, and like gn_cube.c the model clamps nothing (seed below).
func FuzzDecide(f *testing.F) {
	// The textbook shapes, then the edges the T10 floor exists for.
	f.Add(float32(0.5), float32(0.1), float32(0.01), float32(0.1), float32(0.01), uint8(0), int16(5), int16(5), int32(1), false, 0.688, false, false)
	f.Add(float32(0.75), float32(0.3), float32(0.05), float32(0.05), float32(0), uint8(0), int16(4), int16(2), int32(1), false, 0.688, false, false)
	f.Add(float32(0.75), float32(0.3), float32(0.05), float32(0.05), float32(0), uint8(1), int16(0), int16(0), int32(2), false, 0.566, true, true)
	f.Add(float32(0.9), float32(0.4), float32(0.1), float32(0), float32(0), uint8(2), int16(1), int16(7), int32(1), true, 0.687, false, false)
	f.Add(float32(1), float32(1), float32(1), float32(0), float32(0), uint8(0), int16(2), int16(2), int32(1), false, 1.0, false, false)
	f.Add(float32(0), float32(0), float32(0), float32(1), float32(1), uint8(0), int16(3), int16(3), int32(4), false, 0.0, false, false)
	f.Add(float32(0.5), float32(0.9), float32(0.95), float32(0.9), float32(0.95), uint8(0), int16(7), int16(3), int32(1), false, 0.5, false, false) // not nested
	f.Add(float32(0.5), float32(0.1), float32(0.01), float32(0.1), float32(0.01), uint8(0), int16(64), int16(64), int32(1), false, 0.688, false, false)
	f.Add(float32(0.5), float32(0.1), float32(0.01), float32(0.1), float32(0.01), uint8(0), int16(65), int16(5), int32(1), false, 0.688, false, false)
	f.Add(float32(0.5), float32(0.1), float32(0.01), float32(0.1), float32(0.01), uint8(0), int16(5), int16(5), int32(3), false, 0.688, false, false)
	f.Add(float32(0.5), float32(0.1), float32(0.01), float32(0.1), float32(0.01), uint8(0), int16(5), int16(5), int32(128), false, 0.688, false, false)
	f.Add(float32(0.5), float32(0.1), float32(0.01), float32(0.1), float32(0.01), uint8(0), int16(2), int16(4), int32(1), true, 0.688, false, false) // crawford with nobody at 1
	f.Add(float32(0.5), float32(0.1), float32(0.01), float32(0.1), float32(0.01), uint8(0), int16(-1), int16(4), int32(1), false, 0.688, false, false)
	f.Add(float32(0.5), float32(0.1), float32(0.01), float32(0.1), float32(0.01), uint8(0), int16(1), int16(1), int32(1), true, 0.688, false, false)
	// Found by the fuzzer: a win "probability" of 92 at 3-away/3-away, cube 4,
	// is answered with an MWC of 92. Outside the domain, hence skipped.
	f.Add(float32(92), float32(0), float32(0), float32(1), float32(1), uint8(0), int16(3), int16(3), int32(4), false, 0.0, false, false)
	// Found by the fuzzer: not nested, 1.9 of winning mass, MWC 1.32.
	f.Add(float32(1), float32(0.1), float32(1), float32(0), float32(0), uint8(1), int16(2), int16(2), int32(1), false, 41.0, false, false)

	f.Fuzz(func(t *testing.T, p0, p1, p2, p3, p4 float32, ownerRaw uint8, away1, away2 int16, cube int32, crawford bool, efficiency float64, jacoby, money bool) {
		probs := [NumOutputs]float32{p0, p1, p2, p3, p4}
		for _, p := range probs {
			if math.IsNaN(float64(p)) || p < 0 || p > 1 {
				t.Skip("outside the model's domain")
			}
		}
		if math.IsNaN(efficiency) || math.IsInf(efficiency, 0) {
			t.Skip("outside the model's domain")
		}
		efficiency = math.Abs(efficiency)
		if efficiency > 1 {
			efficiency = math.Mod(efficiency, 1)
		}
		owner := CubeOwner(ownerRaw % 3)
		nested := p0 >= p1 && p1 >= p2 && 1-p0 >= p3 && p3 >= p4

		var state *MatchState
		if !money {
			state = &MatchState{AwayOnRoll: int(away1), AwayOpponent: int(away2), Cube: int(cube), Crawford: crawford}
		}

		dec, ok := Decide(&probs, owner, state, efficiency, jacoby)
		if state != nil && ok != state.IsValid() {
			t.Fatalf("Decide ok=%v for state %+v whose IsValid()=%v", ok, *state, state.IsValid())
		}
		if !ok {
			if dec != (Decision{}) {
				t.Fatalf("a refused decision carries values: %+v", dec)
			}
			return
		}

		if dec.Action < NoDouble || dec.Action > TooGood {
			t.Fatalf("Action %d is not one of the four", dec.Action)
		}
		for name, v := range map[string]float64{
			"EquityNoDouble": dec.EquityNoDouble, "EquityDouble": dec.EquityDouble,
			"EquityDoubleTake": dec.EquityDoubleTake, "EquityDoublePass": dec.EquityDoublePass,
			"TakePoint": dec.TakePoint,
		} {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				t.Fatalf("%s = %v for probs %v owner %v state %+v x %v", name, v, probs, owner, state, efficiency)
			}
		}
		if dec.EquityDouble != math.Min(dec.EquityDoubleTake, dec.EquityDoublePass) && !(state != nil && state.Crawford) {
			t.Fatalf("EquityDouble %v is not min(take %v, pass %v)", dec.EquityDouble, dec.EquityDoubleTake, dec.EquityDoublePass)
		}
		if owner == CubeOpponent && dec.Action != NoDouble {
			t.Fatalf("a cube the opponent owns was turned: %v", dec.Action)
		}

		if state == nil {
			if dec.EquityDoublePass != 1 {
				t.Fatalf("money double/pass = %v, want exactly 1", dec.EquityDoublePass)
			}
			return
		}

		// At a score every equity is a match winning chance — of a nested
		// distribution (see above).
		for name, v := range map[string]float64{
			"EquityNoDouble": dec.EquityNoDouble, "EquityDouble": dec.EquityDouble,
			"EquityDoubleTake": dec.EquityDoubleTake, "EquityDoublePass": dec.EquityDoublePass,
		} {
			if nested && (v < 0 || v > 1) {
				t.Fatalf("%s = %v is not a match winning chance (probs %v, state %+v)", name, v, probs, *state)
			}
		}
		if state.Crawford {
			if dec.Action != NoDouble {
				t.Fatalf("Crawford game doubled: %v", dec.Action)
			}
			if dec.EquityDouble != dec.EquityNoDouble || dec.EquityDoubleTake != dec.EquityNoDouble {
				t.Fatalf("Crawford double branch (%v/%v) priced away from no-double (%v)", dec.EquityDouble, dec.EquityDoubleTake, dec.EquityNoDouble)
			}
		}

		v, ok := Value(&probs, owner, state, efficiency)
		if !ok {
			t.Fatalf("Value refused a state Decide accepted: %+v", *state)
		}
		if math.IsNaN(v) || math.IsInf(v, 0) {
			t.Fatalf("Value = %v (probs %v, owner %v, state %+v, x %v)", v, probs, owner, *state, efficiency)
		}
		if nested && (v < -1 || v > 1) {
			t.Fatalf("Value = %v at a score, want [-1, 1] (probs %v, owner %v, state %+v, x %v)", v, probs, owner, *state, efficiency)
		}
	})
}
