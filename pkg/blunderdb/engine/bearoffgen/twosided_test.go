package bearoffgen

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"
)

// The whole point of ADR-0027: what this generates must be what gnubg
// generates, byte for byte. gnubg_ts0.bd is the TS-06-06 table shipped with
// blunderDB until now, produced by `makebearoff -t 6x6`.
func TestTwoSided_6x6_IdenticalToGnubg(t *testing.T) {
	want, err := os.ReadFile("testdata/gnubg_ts0.bd")
	if err != nil {
		t.Skipf("reference table not available: %v", err)
	}

	var got bytes.Buffer
	if err := TwoSided(context.Background(), &got, 6, 6, nil); err != nil {
		t.Fatalf("TwoSided: %v", err)
	}

	if got.Len() != len(want) {
		t.Fatalf("generated %d bytes, gnubg's file is %d", got.Len(), len(want))
	}
	if bytes.Equal(got.Bytes(), want) {
		return
	}

	// Locate the first divergence and say what it means, rather than dumping
	// six megabytes of hex: the byte offset maps back to a (us, them) pair.
	g := got.Bytes()
	for i := range want {
		if g[i] == want[i] {
			continue
		}
		if i < 40 {
			t.Fatalf("header differs at byte %d: got %q, want %q", i, g[:40], want[:40])
		}
		entry := (i - 40) / 2
		pair := entry / planeCount
		plane := entry % planeCount
		n := NumPositions(6, 6)
		t.Fatalf("first difference at byte %d: pair (us=%d, them=%d), plane %d — got %d, want %d",
			i, pair/n, pair%n, plane,
			int32(int16(uint16(g[i&^1])|uint16(g[i|1])<<8))-0x8000,
			int32(int16(uint16(want[i&^1])|uint16(want[i|1])<<8))-0x8000)
	}
}

func TestTwoSided_HeaderIsFortyBytes(t *testing.T) {
	h := headerTwoSided(6, 6)
	if len(h) != 40 {
		t.Fatalf("header length %d", len(h))
	}
	if string(h[:12]) != "gnubg-TS-06-" {
		t.Errorf("header = %q", h)
	}
}

func TestCubeEquity_MatchesTheCDecisionTable(t *testing.T) {
	// A certain win stays a certain win whatever the cube.
	if got := cubeEquity(equityWin, equityWin, equityWin); got != equityWin {
		t.Errorf("cubeEquity(win, win, win) = %d", got)
	}
	// A no-double: the take is not worth half the no-double.
	if got := cubeEquity(1000, 100, equityWin); got != 1000 {
		t.Errorf("cubeEquity(1000, 100, win) = %d, want the no-double 1000", got)
	}
	// A double/take: 2 × the take equity.
	if got := cubeEquity(1000, 600, equityWin); got != 1200 {
		t.Errorf("cubeEquity(1000, 600, win) = %d, want 1200", got)
	}
}

func TestTwoSided_CancellationStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var buf bytes.Buffer
	if err := TwoSided(ctx, &buf, 6, 6, nil); err == nil {
		t.Error("a cancelled context must stop the sweep")
	}
}

// A smaller domain exercises the same code with a different shape, and runs in
// milliseconds: the sweep order, the terminal cases and the roll weighting all
// have to hold whatever the size.
func TestTwoSided_SmallDomainIsSelfConsistent(t *testing.T) {
	const points, checkers = 3, 3
	var buf bytes.Buffer
	if err := TwoSided(context.Background(), &buf, points, checkers, nil); err != nil {
		t.Fatal(err)
	}
	n := NumPositions(points, checkers)
	wantSize := 40 + n*n*planeCount*2
	if buf.Len() != wantSize {
		t.Fatalf("size = %d, want 40 + %d²×8 = %d", buf.Len(), n, wantSize)
	}

	equity := func(us, them, plane int) int32 {
		off := 40 + ((us*n+them)*planeCount+plane)*2
		u := uint16(buf.Bytes()[off]) | uint16(buf.Bytes()[off+1])<<8
		return int32(u) - 0x8000
	}

	// Having borne off is a certain win; facing someone who has is a certain
	// loss. These are the two terminal cases the sweep is anchored on.
	for them := 1; them < n; them++ {
		if got := equity(0, them, 0); got != equityWin {
			t.Fatalf("(0,%d) cubeless = %d, want a certain win %d", them, got, equityWin)
		}
	}
	for us := 1; us < n; us++ {
		if got := equity(us, 0, 0); got != equityLoss {
			t.Fatalf("(%d,0) cubeless = %d, want a certain loss %d", us, got, equityLoss)
		}
	}

	// Owning the cube is never worse than the opponent owning it, and the
	// centred cube sits between the two — the ordering the whole table exists
	// to express.
	for us := 1; us < n; us++ {
		for them := 1; them < n; them++ {
			mine, centred, theirs := equity(us, them, 1), equity(us, them, 2), equity(us, them, 3)
			if mine < centred || centred < theirs {
				t.Fatalf("(%d,%d): owning %d, centred %d, opponent %d — the order must be non-increasing",
					us, them, mine, centred, theirs)
			}
		}
	}
}

// A progress callback must see the counter reach the total, exactly once per
// pair: the Bearoff tab's estimate is built on it.
func TestTwoSided_ProgressReachesTheTotal(t *testing.T) {
	var lastDone, lastTotal int64
	var calls int
	if err := TwoSided(context.Background(), io.Discard, 3, 3, func(done, total int64) {
		calls++
		if done < lastDone {
			t.Fatalf("progress went backwards: %d after %d", done, lastDone)
		}
		lastDone, lastTotal = done, total
	}); err != nil {
		t.Fatal(err)
	}
	n := int64(NumPositions(3, 3))
	if lastTotal != n*n || lastDone != n*n {
		t.Errorf("progress ended at %d/%d, want %d/%d", lastDone, lastTotal, n*n, n*n)
	}
	if calls == 0 {
		t.Error("progress was never called")
	}
}
