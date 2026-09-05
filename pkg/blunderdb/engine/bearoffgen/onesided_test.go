package bearoffgen

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// gnubg_os6.bd is the one-sided table blunderDB embedded until ADR-0027,
// produced by `makebearoff -o 6`. Same rule as the two-sided one: identical or
// it is a failure.
func TestOneSided_6_IdenticalToGnubg(t *testing.T) {
	want, err := os.ReadFile("testdata/gnubg_os6.bd")
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
	want, err := os.ReadFile("testdata/gnubg_os6.bd")
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

// The six-point identity above is necessary and not sufficient: bearing off is
// unconditional inside a six-point table, so a generator that forgets the home
// board rule passes it and diverges from OS-07 on. That is exactly what
// happened — see canBearOff's comment.
//
// The wider oracles are 5 MB, 15 MB, 45 MB and 130 MB and are not in the
// repository. Produce them once with gnubg's own tool and point the test at
// them:
//
//	makebearoff -o 7 -f $DIR/os7.bd     # 5 s
//	makebearoff -o 8 -f $DIR/os8.bd     # 20 s
//	BLUNDERDB_OS_ORACLE_DIR=$DIR go test ./pkg/blunderdb/engine/bearoffgen/
//
// Their fingerprints ARE recorded (KnownFingerprints), so a table generated on
// any machine can still be verified without the oracle; this test is what
// proves those fingerprints describe gnubg's file and not merely our own.
func TestOneSided_WiderDomainsIdenticalToGnubg(t *testing.T) {
	dir := os.Getenv("BLUNDERDB_OS_ORACLE_DIR")
	if dir == "" {
		t.Skip("set BLUNDERDB_OS_ORACLE_DIR to a directory holding makebearoff's os7.bd, os8.bd, …")
	}
	for _, points := range []int{7, 8, 9, 10} {
		path := filepath.Join(dir, fmt.Sprintf("os%d.bd", points))
		want, err := os.ReadFile(path)
		if err != nil {
			t.Logf("OS-%02d: no oracle at %s, skipped", points, path)
			continue
		}
		var got bytes.Buffer
		if err := OneSided(context.Background(), &got, points, nil); err != nil {
			t.Fatalf("OS-%02d: %v", points, err)
		}
		if !bytes.Equal(want, got.Bytes()) {
			t.Errorf("OS-%02d differs from gnubg: %d bytes against %d", points, got.Len(), len(want))
			for i := 0; i < len(want) && i < got.Len(); i++ {
				if want[i] != got.Bytes()[i] {
					t.Errorf("  first difference at byte %d: %02x against %02x", i, want[i], got.Bytes()[i])
					break
				}
			}
			continue
		}
		// And the fingerprint recorded for it must be that file's.
		d := Domain{Kind: OneSidedKind, Points: points, Checkers: osCheckers}
		if want, ok := KnownFingerprints[d]; ok {
			if sum := fmt.Sprintf("%x", sha256.Sum256(got.Bytes())); sum != want {
				t.Errorf("OS-%02d: recorded fingerprint %s, file hashes to %s", points, want, sum)
			}
		} else {
			t.Errorf("OS-%02d: no fingerprint recorded, %x", points, sha256.Sum256(got.Bytes()))
		}
	}
}
