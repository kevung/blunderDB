package gammonnet

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// The opening position is symmetric under exchanging the two players. Encoding
// it with White on roll and with Black on roll must therefore produce the SAME
// 196 features — the encoding is written from the on-roll player's point of
// view, so a symmetric board looks identical to both.
//
// This single assertion catches, together, the two errors this file is most
// exposed to: a mirroring done the wrong way round (index i instead of 23-i),
// and the colour identifiers being swapped between domain and gammonNet. Either
// one breaks it; neither crashes anything otherwise.
func TestOpeningEncodesIdenticallyForBothPlayers(t *testing.T) {
	const opening = "XGID=-b----E-C---eE---c-e----B-:0:0:1:00:0:0:0:0:10"

	white, err := domain.DecodeXGID(opening)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	black := white
	black.PlayerOnRoll = domain.Black
	white.PlayerOnRoll = domain.White

	var fw, fb [NumFeatures]float32
	for _, tc := range []struct {
		name string
		pos  domain.Position
		out  *[NumFeatures]float32
	}{{"white", white, &fw}, {"black", black, &fb}} {
		p, err := FromDomain(&tc.pos)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if !Encode(&p, tc.out) {
			t.Fatalf("%s: encoding refused a valid opening position", tc.name)
		}
	}

	for i := range fw {
		if fw[i] != fb[i] {
			t.Fatalf("feature %d differs between the two perspectives: %v vs %v\n"+
				"the opening is symmetric, so this is a mirroring or a colour swap",
				i, fw[i], fb[i])
		}
	}

	// And it is not trivially all zeros.
	var nonZero int
	for _, v := range fw {
		if v != 0 {
			nonZero++
		}
	}
	if nonZero == 0 {
		t.Fatal("the opening encoded to an all-zero vector")
	}
}

// The on-roll player's own checkers land in the MY block (features 0..97) and
// the opponent's in the OPP block (98..195) — but BOTH use the slot of the
// physical point as seen from the on-roll player's frame. The opponent's half
// is not re-indexed from its own ace point.
//
// So whoever is on roll, their own six point is slot 5 of the MY block, and the
// opponent's six point is slot 18 of the OPP block. Swap the mirroring or the
// colours and this stops holding.
func TestPerspectiveBlocksFollowTheOnRollPlayer(t *testing.T) {
	// Five White checkers on White's six point (gammonNet index 5), five Black
	// on Black's six point (index 18).
	var p Position
	p.Points[5] = 5
	p.Points[18] = -5
	p.Off[White] = 10
	p.Off[Black] = 10

	const (
		mySix  = myBlockOffset + 5*featuresPerPoint   // 20
		oppSix = oppBlockOffset + 18*featuresPerPoint // 170
	)
	want := [4]float32{1, 1, 1, 1} // five checkers: 1, 1, 1, (5-3)/2 = 1

	for _, turn := range []struct {
		name string
		who  uint8
	}{{"white on roll", White}, {"black on roll", Black}} {
		t.Run(turn.name, func(t *testing.T) {
			p.Turn = turn.who
			var f [NumFeatures]float32
			if !Encode(&p, &f) {
				t.Fatal("encoding refused a valid position")
			}
			for _, c := range []struct {
				label string
				idx   int
			}{{"my six point", mySix}, {"opponent six point", oppSix}} {
				got := [4]float32{f[c.idx], f[c.idx+1], f[c.idx+2], f[c.idx+3]}
				if got != want {
					t.Errorf("%s (feature %d) = %v, want %v", c.label, c.idx, got, want)
				}
			}
			// Nothing else may be set in either point block.
			for i := 0; i < oppBarIndex; i++ {
				inMy := i >= mySix && i < mySix+4
				inOpp := i >= oppSix && i < oppSix+4
				if !inMy && !inOpp && i != myBarIndex && i != myOffIndex && f[i] != 0 {
					t.Errorf("feature %d = %v, expected 0", i, f[i])
				}
			}
		})
	}
}

func TestThermometer(t *testing.T) {
	for _, tc := range []struct {
		count int
		want  [4]float32
	}{
		{0, [4]float32{0, 0, 0, 0}},
		{1, [4]float32{1, 0, 0, 0}},
		{2, [4]float32{1, 1, 0, 0}},
		{3, [4]float32{1, 1, 1, 0}},
		{4, [4]float32{1, 1, 1, 0.5}},
		{5, [4]float32{1, 1, 1, 1}},
		{15, [4]float32{1, 1, 1, 6}},
	} {
		var x [NumFeatures]float32
		encodeCheckers(&x, 0, tc.count)
		got := [4]float32{x[0], x[1], x[2], x[3]}
		if got != tc.want {
			t.Errorf("%d checkers → %v, want %v", tc.count, got, tc.want)
		}
	}
}

func TestBarAndOffScaling(t *testing.T) {
	var p Position
	p.Points[0] = 13
	p.Points[23] = -14
	p.Bar[White] = 2
	p.Off[White] = 0
	p.Bar[Black] = 1
	p.Off[Black] = 0
	p.Turn = White

	var f [NumFeatures]float32
	if !Encode(&p, &f) {
		t.Fatal("encoding refused a valid position")
	}
	if f[myBarIndex] != 1.0 { // 2 × 0.5
		t.Errorf("my bar = %v, want 1", f[myBarIndex])
	}
	if f[oppBarIndex] != 0.5 { // 1 × 0.5
		t.Errorf("opponent bar = %v, want 0.5", f[oppBarIndex])
	}

	// Off is scaled by 1/15, computed in float64 and rounded once — the same
	// double multiply the reference does. Rounding twice would move the last bit.
	p.Points[0] = 8
	p.Off[White] = 5
	if !Encode(&p, &f) {
		t.Fatal("encoding refused a valid position")
	}
	if want := float32(float64(5) * (1.0 / 15.0)); f[myOffIndex] != want {
		t.Errorf("my off = %v, want %v", f[myOffIndex], want)
	}
}

// Structurally impossible positions are refused, never approximated.
func TestEncodeRefusesInvalidPositions(t *testing.T) {
	for _, tc := range []struct {
		name string
		mut  func(*Position)
	}{
		{"too few checkers", func(p *Position) { p.Points[0] = 14 }},
		{"unknown player", func(p *Position) { p.Turn = 7 }},
		{"point over capacity", func(p *Position) { p.Points[0] = 16; p.Points[1] = -1 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := Position{Turn: White}
			p.Points[0] = 15
			p.Points[23] = -15
			tc.mut(&p)
			var f [NumFeatures]float32
			if Encode(&p, &f) {
				t.Fatal("encoded a position it should have refused")
			}
		})
	}
}
