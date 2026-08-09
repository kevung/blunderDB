package race

import (
	"math"
	"math/rand"
	"os"
	"sort"
	"testing"
)

func randHome(rng *rand.Rand, n int) [6]int {
	var b [6]int
	for i := 0; i < n; i++ {
		b[rng.Intn(6)]++
	}
	return b
}

// The raw convolution must track the exact two-sided probability on the
// embedded TS-06-06 domain. Bound from the design measurements (ADR-0009):
// raw error peaks around 1.3 % of win probability at 5–6 checkers.
func TestConvolution_RawAgainstEmbeddedExact(t *testing.T) {
	ts := EmbeddedTwoSided()
	rng := rand.New(rand.NewSource(7))
	worst := 0.0
	for i := 0; i < 3000; i++ {
		us := randHome(rng, 1+rng.Intn(6))
		them := randHome(rng, 1+rng.Intn(6))
		e, err := ts.Lookup(us, them)
		if err != nil {
			t.Fatal(err)
		}
		p, _, err := RawWinProbFeatures(us, them)
		if err != nil {
			t.Fatal(err)
		}
		if d := math.Abs(p - e.WinProb); d > worst {
			worst = d
		}
	}
	if worst > 0.02 {
		t.Fatalf("raw convolution error %.4f exceeds 2%% on the ≤6 domain", worst)
	}
}

func TestEstimatedWinProb_Sanity(t *testing.T) {
	// Symmetric position, on-roll advantage: p must be > 0.5 and ≤ 1.
	b := [6]int{2, 2, 2, 2, 0, 0}
	p, err := EstimatedWinProb(b, b)
	if err != nil {
		t.Fatal(err)
	}
	if p <= 0.5 || p > 1 {
		t.Fatalf("symmetric position, on roll: p=%v, want (0.5, 1]", p)
	}
	// Overwhelming material edge → p close to 1.
	p, err = EstimatedWinProb([6]int{1, 1, 0, 0, 0, 0}, [6]int{2, 2, 2, 3, 3, 3})
	if err != nil {
		t.Fatal(err)
	}
	if p < 0.95 {
		t.Fatalf("huge lead: p=%v, want ≥ 0.95", p)
	}
	// Correction constants must have been calibrated (non-zero bounds).
	if CorrectionSigma <= 0 || CorrectionP99 <= 0 {
		t.Fatal("correction bounds are zero — correction_coeffs.go was not generated")
	}
}

// Env-gated oracle test: with the TS-06-11 file available, the corrected
// estimator must hold its calibrated bounds (with margin) on a fresh sample.
func TestEstimatedWinProb_OracleBounds(t *testing.T) {
	path := os.Getenv("BLUNDERDB_TS11_PATH")
	if path == "" {
		t.Skip("BLUNDERDB_TS11_PATH not set; oracle test skipped")
	}
	oracle, err := OpenTwoSided(path)
	if err != nil {
		t.Fatal(err)
	}
	rng := rand.New(rand.NewSource(99)) // deliberately not the calibration seed
	res := make([]float64, 0, 20000)
	for len(res) < 20000 {
		nu, nt := 1+rng.Intn(oracle.Checkers()), 1+rng.Intn(oracle.Checkers())
		if nu <= 6 && nt <= 6 {
			continue
		}
		us, them := randHome(rng, nu), randHome(rng, nt)
		e, err := oracle.Lookup(us, them)
		if err != nil {
			t.Fatal(err)
		}
		p, err := EstimatedWinProb(us, them)
		if err != nil {
			t.Fatal(err)
		}
		res = append(res, math.Abs(p-e.WinProb))
	}
	sort.Float64s(res)
	var sq float64
	for _, r := range res {
		sq += r * r
	}
	sigma := math.Sqrt(sq / float64(len(res)))
	p99 := res[int(0.99*float64(len(res)))]
	maxAbs := res[len(res)-1]
	t.Logf("oracle residuals: sigma %.6f p99 %.6f max %.6f", sigma, p99, maxAbs)
	if sigma > 1.5*CorrectionSigma || p99 > 1.5*CorrectionP99 || maxAbs > 2*CorrectionMax {
		t.Fatalf("estimator degraded beyond calibrated bounds: sigma %.6f (cal %.6f), p99 %.6f (cal %.6f), max %.6f (cal %.6f)",
			sigma, CorrectionSigma, p99, CorrectionP99, maxAbs, CorrectionMax)
	}
}
