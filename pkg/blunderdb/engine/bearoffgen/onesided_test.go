package bearoffgen

import (
	"bytes"
	"context"
	"os"
	"testing"
)

// gnubg_os6.bd is the one-sided table blunderDB embedded until ADR-0027,
// produced by `makebearoff -o 6`. Same rule as the two-sided one: identical or
// it is a failure.
func TestOneSided_6_IdenticalToGnubg(t *testing.T) {
	want, err := os.ReadFile("../gnubg_os6.bd")
	if err != nil {
		t.Skipf("reference table not available: %v", err)
	}

	var got bytes.Buffer
	if err := OneSided(context.Background(), &got, 6, nil); err != nil {
		t.Fatalf("OneSided: %v", err)
	}

	if got.Len() != len(want) {
		t.Fatalf("generated %d bytes, gnubg's file is %d", got.Len(), len(want))
	}
	if bytes.Equal(got.Bytes(), want) {
		return
	}

	g := got.Bytes()
	n := NumPositions(6, 15)
	for i := range want {
		if g[i] == want[i] {
			continue
		}
		switch {
		case i < 40:
			t.Fatalf("header differs at byte %d:\n got %q\nwant %q", i, g[:40], want[:40])
		case i < 40+n*8:
			entry := (i - 40) / 8
			t.Fatalf("index differs at byte %d (entry %d, field %d):\n got %v\nwant %v",
				i, entry, (i-40)%8, g[40+entry*8:40+entry*8+8], want[40+entry*8:40+entry*8+8])
		default:
			t.Fatalf("body differs at byte %d (offset %d into the runs): got %#02x, want %#02x",
				i, i-40-n*8, g[i], want[i])
		}
	}
}

func TestOneSided_HeaderMatchesTheShippedFile(t *testing.T) {
	want, err := os.ReadFile("../gnubg_os6.bd")
	if err != nil {
		t.Skipf("reference table not available: %v", err)
	}
	if got := headerOneSided(6); !bytes.Equal(got, want[:40]) {
		t.Errorf("header:\n got %q\nwant %q", got, want[:40])
	}
}

func TestCalcIndex_FindsTheRunOfNonZeroes(t *testing.T) {
	dist := make([]uint16, 2*osRolls)
	dist[7], dist[8], dist[11] = 100, 200, 5
	idx, nz := calcIndex(dist, 0)
	if idx != 7 || nz != 5 {
		t.Errorf("run = (%d, %d), want start 7 length 5 (7..11 inclusive)", idx, nz)
	}

	// An all-zero distribution: the C leaves j at 32 and scans one entry past
	// the end, into the first gammon slot of the same array. Reproduced here
	// (see calcIndex) rather than bounded, so a case that does not arise today
	// cannot diverge silently if it ever does.
	zero := make([]uint16, 2*osRolls)
	if idx, nz := calcIndex(zero, 0); idx != 0 || nz != 33 {
		t.Errorf("all-zero run = (%d, %d), want (0, 33) as the C produces", idx, nz)
	}
	// And when that neighbouring entry is set, the scan finds it.
	neighbour := make([]uint16, 2*osRolls)
	neighbour[osRolls] = 7
	if idx, nz := calcIndex(neighbour, 0); idx != osRolls || nz != 1 {
		t.Errorf("run = (%d, %d), want the C's read past the end to land on entry 32", idx, nz)
	}
}

func TestNormalise_TheModeAbsorbsTheRounding(t *testing.T) {
	sums := make([]int64, osRolls)
	// Three equal thirds: 36 outcomes each rounding to the same value, whose
	// sum cannot be 0xFFFF exactly.
	sums[3], sums[4], sums[5] = 0x5555*36, 0x5555*36, 0x5555*36
	out := make([]uint16, osRolls)
	normalise(sums, out)

	var total int
	for _, v := range out {
		total += int(v)
	}
	if total != 0xFFFF {
		t.Errorf("the distribution sums to %#x, want exactly 0xFFFF", total)
	}
}

func TestOneSided_CancellationStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var buf bytes.Buffer
	if err := OneSided(ctx, &buf, 6, nil); err == nil {
		t.Error("a cancelled context must stop the sweep")
	}
}
