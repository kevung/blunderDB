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

// widest is a position of the most bits the codec spends: Away scores of 99,
// dice, a cube held at the highest exponent the ArkID spells, and fifteen
// checkers a side, none on a bar or borne off.
func widest(t *testing.T) *domain.Position {
	t.Helper()
	p, err := domain.DecodeArkID("[99-99]65:x10")
	if err != nil {
		t.Fatal(err)
	}
	p.Board = domain.Board{}
	for k := range 8 {
		n := 2
		if k == 7 {
			n = 1
		}
		p.Board.Points[1+k] = domain.Point{Checkers: n, Color: domain.Black}
		p.Board.Points[24-k] = domain.Point{Checkers: n, Color: domain.White}
	}
	if _, err := domain.EncodeArkID(&p); err != nil {
		t.Fatal(err)
	}
	return &p
}

// The bound Tranche rests on: a full tranche of the widest positions, from
// the highest first number a two-byte header holds, is two data codes and
// their parity.
func TestPositionsWorstTrancheTwoCodes(t *testing.T) {
	w := widest(t)
	ps := make([]*domain.Position, Tranche)
	for i := range ps {
		ps[i] = w
	}
	data, err := EncodePositions(1<<14-1, ps)
	if err != nil {
		t.Fatal(err)
	}
	if want := 5 + (Tranche*109+7)/8; len(data) != want || len(data) > 2*maxPart {
		t.Fatalf("%d bytes, want %d and at most %d", len(data), want, 2*maxPart)
	}
	frames, err := PositionFrames(data)
	if err != nil {
		t.Fatal(err)
	}
	codes, parity := 0, 0
	for _, f := range frames {
		if f.Parity {
			parity++
		} else {
			codes++
		}
	}
	if codes != 2 || parity != 1 {
		t.Fatalf("%d data codes and %d parity codes, want 2 and 1", codes, parity)
	}
}

// Two tranches of two different lists read together: each is rebuilt from
// its own codes only, a lost code of each repaired by its own parity.
func TestPositionFramesTwoListsApart(t *testing.T) {
	ps := randomPositions(t, 2*Tranche)
	var read []Frame
	want := map[int][]string{}
	for k, first := range []int{1, 146} {
		part := ps[k*Tranche : (k+1)*Tranche]
		data, err := EncodePositions(first, part)
		if err != nil {
			t.Fatal(err)
		}
		frames, err := PositionFrames(data)
		if err != nil {
			t.Fatal(err)
		}
		// Each list loses a different data code.
		for i, f := range frames {
			if i != k {
				read = append(read, f)
			}
		}
		for _, p := range part {
			a, _ := domain.EncodeArkID(p)
			want[first] = append(want[first], a)
		}
	}
	asm := Assemble(read)
	if asm.Repaired != 2 || len(asm.Missing) != 0 || len(asm.Tranches) != 2 {
		t.Fatalf("repaired %d, missing %v, %d tranches", asm.Repaired, asm.Missing, len(asm.Tranches))
	}
	for _, data := range asm.Tranches {
		tr, err := DecodePositions(data)
		if err != nil {
			t.Fatal(err)
		}
		w := want[tr.First]
		if len(tr.Positions) != len(w) {
			t.Fatalf("tranche %d: %d positions, want %d", tr.First, len(tr.Positions), len(w))
		}
		for i := range tr.Positions {
			if a, _ := domain.EncodeArkID(&tr.Positions[i]); a != w[i] {
				t.Fatalf("tranche %d, position %d: %s, want %s", tr.First, i, a, w[i])
			}
		}
		delete(want, tr.First)
	}
}
