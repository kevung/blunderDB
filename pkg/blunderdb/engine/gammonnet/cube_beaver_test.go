// SPDX-License-Identifier: MIT

package gammonnet

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

func gammonless(p float32) [NumOutputs]float32 {
	var probs [NumOutputs]float32
	probs[PWin] = p
	return probs
}

// The anchors of spec §4bis, gammonless: at x = 0, e = 2p − 1 in every cube
// state, so beaver ⇔ p < 1/2 and raccoon ⇔ p > 1/2; at x = 1, raccoon ⇔
// p > 0.2 and beaver ⇔ p < 1/3.
func TestBeaverAnchors(t *testing.T) {
	cases := []struct {
		x               float64
		p               float32
		beaver, raccoon bool
	}{
		{0, 0.40, true, false},
		{0, 0.60, false, true},
		{0, 0.80, false, true},
		{1, 0.15, true, false},
		{1, 0.25, true, true},
		{1, 0.30, true, true},
		{1, 0.40, false, true},
	}
	for _, c := range cases {
		probs := gammonless(c.p)
		dec, ok := DecideEx(&probs, CubeCentred, nil, c.x, false, true)
		if !ok || !dec.Beaver.Enabled {
			t.Fatalf("x=%v p=%v: no beaver decision", c.x, c.p)
		}
		if dec.Beaver.Beaver != c.beaver || dec.Beaver.Raccoon != c.raccoon {
			t.Errorf("x=%v p=%v: beaver=%v raccoon=%v, want %v %v",
				c.x, c.p, dec.Beaver.Beaver, dec.Beaver.Raccoon, c.beaver, c.raccoon)
		}
		if c.x == 0 && c.p > 0.75 && dec.Beaver.Action != DoublePass {
			t.Errorf("x=0 p=%v: action %v, want DoublePass", c.p, dec.Beaver.Action)
		}
	}
}

// The flag off, a match, or a cube the opponent owns: the plain fields never
// move, and the beaver answer is empty or says nothing can be answered.
func TestBeaverLeavesThePlainFieldsAlone(t *testing.T) {
	probs := gammonless(0.40)
	plain, _ := Decide(&probs, CubeCentred, nil, 0.688, false)
	with, _ := DecideEx(&probs, CubeCentred, nil, 0.688, false, true)
	if plain.Beaver.Enabled {
		t.Error("Decide filled a beaver answer")
	}
	stripped := with
	stripped.Beaver = BeaverDecision{}
	if stripped != plain {
		t.Errorf("the beaver flag moved a plain field: %+v, want %+v", stripped, plain)
	}
	state := MatchState{AwayOnRoll: 5, AwayOpponent: 5, Cube: 1}
	if m, ok := DecideEx(&probs, CubeCentred, &state, 0.688, false, true); !ok || m.Beaver.Enabled {
		t.Errorf("a match filled a beaver answer (ok=%v)", ok)
	}
	opp, _ := DecideEx(&probs, CubeOpponent, nil, 0.687, false, true)
	if opp.Beaver.Action != NoDouble || opp.Beaver.Beaver || opp.Beaver.Raccoon {
		t.Errorf("opponent's cube: %+v, want no double and no answer", opp.Beaver)
	}
}

// DecideForSession reads HasBeaver at money only and folds the opponent's
// best non-pass answer into the take branch.
func TestDecideForSessionFoldsTheBeaver(t *testing.T) {
	probs := gammonless(0.30)
	pos := &domain.Position{HasBeaver: 1}
	got, ok := DecideForSession(&probs, CubeCentred, nil, 0.688, pos)
	if !ok {
		t.Fatal("refused")
	}
	raw, _ := DecideEx(&probs, CubeCentred, nil, 0.688, false, true)
	if !raw.Beaver.Beaver {
		t.Fatal("expected a beaver at p = 0.30")
	}
	if got.EquityDoubleTake != raw.Beaver.EquityBeaver || got.EquityDouble != raw.Beaver.EquityDouble || got.Action != raw.Beaver.Action {
		t.Errorf("not folded: %+v from %+v", got, raw.Beaver)
	}
	if got.EquityDoubleTake >= raw.EquityDoubleTake {
		t.Errorf("a beaver must cost the doubler: take branch %v, plain %v", got.EquityDoubleTake, raw.EquityDoubleTake)
	}

	pos.HasBeaver = 0
	plain, _ := DecideForSession(&probs, CubeCentred, nil, 0.688, pos)
	if plain.Beaver.Enabled || plain.EquityDoubleTake != raw.EquityDoubleTake {
		t.Errorf("HasBeaver = 0 still read a beaver: %+v", plain)
	}

	pos.HasBeaver = 1
	state := MatchState{AwayOnRoll: 5, AwayOpponent: 5, Cube: 1}
	if m, _ := DecideForSession(&probs, CubeCentred, &state, 0.688, pos); m.Beaver.Enabled {
		t.Error("a match read the beaver rule")
	}
}
