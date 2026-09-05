package race

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/bearoffgen"
)

// Does the correction still hold beyond the home board?
//
// The polynomial in correction_coeffs.go was fitted against the TS-06-11
// oracle: races where every chequer of both sides is on one of six points. A
// seven-point one-sided table lets the convolution answer for a side whose
// farthest chequer is on the 7-point, and nothing says a correction fitted on
// one domain describes the other. ADR-0027 §9 is explicit that this must be
// MEASURED and not extrapolated, which is what this test does.
//
// It needs two oracles gnubg produces and this repository does not carry:
//
//	makebearoff -t 7x6 -f $TS/ts7x6.bd     # 20 s, 23 MB
//	makebearoff -o 7   -f $OS/os7.bd       #  5 s, 4.9 MB
//	BLUNDERDB_TS_ORACLE_DIR=$TS BLUNDERDB_OS_ORACLE_DIR=$OS go test ./pkg/blunderdb/engine/race/
//
// It reads the two-sided file itself rather than through OpenTwoSided, which
// deliberately refuses anything but a six-point domain: the exact verdict the
// panel shows is defined over home boards (ADR-0009), and this measurement
// must not be the reason that restriction is loosened.

// wideTwoSided is a minimal reader for a TS-<p>x<c> file, enough to take the
// cubeless win probability out of it.
type wideTwoSided struct {
	raw       []byte
	nPoints   int
	nCheckers int
	nPos      int
}

func openWideTwoSided(path string) (*wideTwoSided, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(raw) < 40 || string(raw[:9]) != "gnubg-TS-" {
		return nil, fmt.Errorf("%s: not a two-sided database", path)
	}
	var p, c int
	if _, err := fmt.Sscanf(string(raw[9:14]), "%02d-%02d", &p, &c); err != nil {
		return nil, fmt.Errorf("%s: unreadable domain %q", path, raw[9:14])
	}
	n := combination(p+c, p)
	if want := 40 + n*n*8; len(raw) != want {
		return nil, fmt.Errorf("%s: %d bytes, want %d for TS-%02d-%02d", path, len(raw), want, p, c)
	}
	return &wideTwoSided{raw: raw, nPoints: p, nCheckers: c, nPos: n}, nil
}

// winProb reads plane 0 of (us, them) as a win probability, exactly as
// TwoSided.Lookup does for the six-point file.
func (w *wideTwoSided) winProb(us, them []int) float64 {
	iu := positionIndexOf(us, w.nPoints, w.nCheckers)
	it := positionIndexOf(them, w.nPoints, w.nCheckers)
	off := 40 + (iu*w.nPos+it)*8
	return float64(binary.LittleEndian.Uint16(w.raw[off:])) / 65535.0
}

// positionIndexOf is the combinatorial ranking for a board of any width. It is
// bearoffgen's, which the OS-07 and TS-07-06 identity tests prove against
// gnubg — hand-rolling a second copy here got the measurement wrong once
// already, and a wrong oracle reading looks exactly like a broken estimator.
func positionIndexOf(board []int, points, checkers int) int {
	return bearoffgen.PositionIndex(board, points, checkers)
}

// randWide draws a race position of `n` chequers whose farthest is on point
// `points` or below.
func randWide(rng *rand.Rand, n, points int) []int {
	b := make([]int, points)
	for i := 0; i < n; i++ {
		b[rng.Intn(points)]++
	}
	return b
}

func TestCorrection_BeyondTheHomeBoard(t *testing.T) {
	tsDir, osDir := os.Getenv("BLUNDERDB_TS_ORACLE_DIR"), os.Getenv("BLUNDERDB_OS_ORACLE_DIR")
	if tsDir == "" || osDir == "" {
		t.Skip("set BLUNDERDB_TS_ORACLE_DIR and BLUNDERDB_OS_ORACLE_DIR; see this test's comment")
	}
	oracle, err := openWideTwoSided(filepath.Join(tsDir, "ts7x6.bd"))
	if err != nil {
		t.Skipf("no seven-point oracle: %v", err)
	}
	// The estimate must read a seven-point one-sided table, not the six-point
	// one the rest of the suite loads.
	if err := engine.LoadOneSided(filepath.Join(osDir, "os7.bd")); err != nil {
		t.Skipf("no seven-point one-sided table: %v", err)
	}
	t.Cleanup(func() { _ = engine.LoadOneSided("") })
	if engine.OneSidedPoints() != 7 {
		t.Fatalf("loaded a %d-point table, want 7", engine.OneSidedPoints())
	}

	// Three samples, each answering a different question: inside the old
	// domain (does widening the table change what we already had?), with one
	// side beyond it, with both.
	for _, sample := range []struct {
		name         string
		usPts, thPts int
	}{
		{"six against six", 6, 6},
		{"seven against six", 7, 6},
		{"seven against seven", 7, 7},
	} {
		rng := rand.New(rand.NewSource(4242))
		var raw, corrected []float64
		for len(corrected) < 20000 {
			us := randWide(rng, 1+rng.Intn(oracle.nCheckers), sample.usPts)
			them := randWide(rng, 1+rng.Intn(oracle.nCheckers), sample.thPts)
			// Only positions that actually reach the sample's width.
			if sample.usPts == 7 && us[6] == 0 {
				continue
			}
			if sample.thPts == 7 && them[6] == 0 {
				continue
			}
			usW, thW := make([]int, 7), make([]int, 7)
			copy(usW, us)
			copy(thW, them)

			exact := oracle.winProb(usW, thW)
			p, _, _, err := winProbRaw(usW, thW)
			if err != nil {
				t.Fatal(err)
			}
			e, err := EstimatedWinProbPoints(usW, thW)
			if err != nil {
				t.Fatal(err)
			}
			raw = append(raw, math.Abs(p-exact))
			corrected = append(corrected, math.Abs(e-exact))
		}
		rs, rp, rm := summarise(raw)
		cs, cp, cm := summarise(corrected)
		t.Logf("%-22s raw        sigma %.5f  p99 %.5f  max %.5f", sample.name, rs, rp, rm)
		t.Logf("%-22s corrected  sigma %.5f  p99 %.5f  max %.5f", sample.name, cs, cp, cm)

		// What the measurement concluded, turned into a guard. The correction
		// is applied beyond six points because it TRANSFERS: it must not make
		// the tail worse anywhere, and it must stay inside the bound recorded
		// for this domain. A correction that stopped transferring would show
		// up here rather than in a user's panel.
		if cp > rp {
			t.Errorf("%s: the correction worsens p99 (%.5f against %.5f raw)", sample.name, cp, rp)
		}
		if cm > rm {
			t.Errorf("%s: the correction worsens the maximum (%.5f against %.5f raw)", sample.name, cm, rm)
		}
		if cs > BeyondHomeSigma || cp > BeyondHomeP99 || cm > BeyondHomeMax {
			t.Errorf("%s: beyond the recorded bound — sigma %.5f/%.5f, p99 %.5f/%.5f, max %.5f/%.5f",
				sample.name, cs, BeyondHomeSigma, cp, BeyondHomeP99, cm, BeyondHomeMax)
		}
	}
}

func summarise(res []float64) (sigma, p99, max float64) {
	sort.Float64s(res)
	var sq float64
	for _, r := range res {
		sq += r * r
	}
	return math.Sqrt(sq / float64(len(res))), res[int(0.99*float64(len(res)))], res[len(res)-1]
}
