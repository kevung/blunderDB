package arkqr

import (
	"math/rand/v2"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// randomPositions draws positions every ArkID spells: random scores, dice,
// cubes and checkers, the refused ones left out.
func randomPositions(t *testing.T, n int) []*domain.Position {
	t.Helper()
	rng := rand.New(rand.NewPCG(7, 11))
	heads := []string{"[$]", "[11]", "[12c]", "[34]", "[11-9]", "[99-1]", "[1-25c]"}
	dice := []string{"", "61", "33", "21", "66"}
	cubes := []string{"", "x", "X", "x0", "X3", "x10", "X?", "X2?"}
	var out []*domain.Position
	for len(out) < n {
		head := heads[rng.IntN(len(heads))] + dice[rng.IntN(len(dice))] + ":" + cubes[rng.IntN(len(cubes))]
		p, err := domain.DecodeArkID(head)
		if err != nil {
			continue
		}
		p.Board = domain.Board{}
		for side, color := range []int{domain.Black, domain.White} {
			left := 15 - rng.IntN(16)
			for left > 0 {
				i := 1 + rng.IntN(24)
				if rng.IntN(12) == 0 {
					i = []int{25, 0}[side]
				}
				pt := &p.Board.Points[i]
				if pt.Checkers > 0 && pt.Color != color {
					continue
				}
				k := 1 + rng.IntN(min(left, 5))
				pt.Checkers += k
				pt.Color = color
				left -= k
			}
		}
		if _, err := domain.EncodeArkID(&p); err != nil {
			continue
		}
		out = append(out, &p)
	}
	return out
}

func TestPositionsRoundTrip(t *testing.T) {
	ps := randomPositions(t, 2000)
	data, err := EncodePositions(321, ps)
	if err != nil {
		t.Fatal(err)
	}
	if per := float64(len(data)) / float64(len(ps)); per > 14 {
		t.Errorf("%.1f bytes a position, the format promises about 12", per)
	}
	tr, err := DecodePositions(data)
	if err != nil {
		t.Fatal(err)
	}
	if tr.First != 321 || len(tr.Positions) != len(ps) {
		t.Fatalf("first %d, %d positions; want 321, %d", tr.First, len(tr.Positions), len(ps))
	}
	for i, p := range ps {
		want, _ := domain.EncodeArkID(p)
		got, err := domain.EncodeArkID(&tr.Positions[i])
		if err != nil || got != want {
			t.Fatalf("position %d: %q (%v), want %q", i, got, err, want)
		}
	}
}

func TestDecodePositionsDamaged(t *testing.T) {
	data, err := EncodePositions(1, randomPositions(t, 20))
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range [][]byte{nil, {9}, data[:len(data)/2], append([]byte{PositionsVersion, 1, 200}, data[3:]...)} {
		if _, err := DecodePositions(bad); err == nil {
			t.Errorf("% x: decoded", bad[:min(len(bad), 4)])
		}
	}
}

// A tranche is an archive of its own: two data codes and their parity for
// Tranche positions, a lost data code rebuilt, match codes read beside it
// left to the matches.
func TestPositionFramesTrancheAlone(t *testing.T) {
	ps := randomPositions(t, Tranche)
	data, err := EncodePositions(161, ps)
	if err != nil {
		t.Fatal(err)
	}
	frames, err := PositionFrames(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 3 {
		t.Errorf("%d codes for %d positions (%d bytes)", len(frames), Tranche, len(data))
	}
	var read []Frame
	for i, f := range frames {
		if f.Kind != KindPositions {
			t.Fatalf("frame %d: kind %d", i, f.Kind)
		}
		g, err := ParseText(f.Text())
		if err != nil {
			t.Fatal(err)
		}
		// The first data code is lost: its parity brings it back.
		if i > 0 {
			read = append(read, g)
		}
	}
	m, _, err := Frames([][]byte{{1, 2, 3}})
	if err != nil {
		t.Fatal(err)
	}
	if m[0].Bytes()[0] == frames[0].Bytes()[0] {
		t.Error("a tranche code opens like a match code")
	}
	asm := Assemble(append(read, m...))
	if asm.Repaired != 1 || len(asm.Missing) != 0 || len(asm.Tranches) != 1 || len(asm.Matches) != 1 {
		t.Fatalf("repaired %d, missing %v, %d tranches, %d matches", asm.Repaired, asm.Missing, len(asm.Tranches), len(asm.Matches))
	}
	tr, err := DecodePositions(asm.Tranches[0])
	if err != nil || tr.First != 161 || len(tr.Positions) != Tranche {
		t.Fatalf("tranche: first %d, %d positions, %v", tr.First, len(tr.Positions), err)
	}
}
