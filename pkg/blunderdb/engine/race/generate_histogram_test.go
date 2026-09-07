package race

import (
	"math/rand/v2"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// Is the generator REALISTIC? — measured, not asserted (ADR-0041 rule 5).
//
// The claim ADR-0041 rests on is that a played-out position carries the gaps,
// low stacks and asymmetries of a real bear-off, and that a placement does not.
// A claim of that shape is exactly the one nobody can check by eye, so this
// file checks it by number: the wastage histogram of what the generator
// produces, against the wastage histogram of every bear-off in the ten real
// matches of testdata/, against — as a control — a uniform placement of the
// same chequers.
//
// Wastage (EPC minus pip count) is the right axis because it IS what the
// exercise trains: it is the whole difference between counting pips and
// counting a bear-off, and it is the number a placement gets wrong. The pip
// count alone would be nearly uninformative — any arrangement of n chequers on
// six points has a pip count in a narrow band.
//
// The control is what gives the threshold teeth. A test that only said
// "TVD ≤ 0.20" would pass for any distribution close enough by accident; this
// one also requires the rejected alternative — the "weighted random placement"
// of ADR-0041's considered options — to sit measurably further away. If the
// generator ever degenerates into a placement, the second assertion falls
// before the first.

// wastageBinLow/High bound the histogram, in whole pips. A bear-off's wastage
// runs from about seven (a tight bear-in) to the mid-twenties (chequers buried
// on the ace point); anything outside falls into the end bins.
const (
	wastageBinLow  = 6
	wastageBinHigh = 26
)

// maxHistogramDistance is the stated threshold: half the total-variation
// distance between the two histograms, 0 for identical and 1 for disjoint.
//
// It is not tighter for a measured reason. The reference sample is finite (a
// few hundred sides), and splitting it in half against itself already gives a
// distance of about 0.15 — sampling noise alone, with no generator involved.
// Measured on 2026-09-07: the generator sits at 0.154, a uniform placement of
// the same chequers at 0.297. A threshold under the reference's own noise
// floor would be a test of the fixtures, not of the generator.
const maxHistogramDistance = 0.20

// minControlRatio is how much further the rejected alternative must sit. At
// 1.5 the assertion holds with the measured 0.297 / 0.154 = 1.93 and fails the
// moment the generator's advantage over a placement stops being a fact.
const minControlRatio = 1.5

// bearoffSample is one side of one bear-off position.
type bearoffSample struct {
	checkers int
	wastage  float64
}

func TestGeneratedBearoffsMatchTheHistogramOfRealRaces(t *testing.T) {
	if testing.Short() {
		t.Skip("reads and parses ten match files")
	}
	real := realBearoffSamples(t)
	// A fixture that moved would otherwise turn this test green by emptying
	// it: an empty reference matches everything.
	if len(real) < 200 {
		t.Fatalf("only %d real bear-off sides in testdata/: the reference is too small to measure against", len(real))
	}

	rng := rand.New(rand.NewPCG(0x9e3779b9, 0x7f4a7c15))
	var generated []bearoffSample
	for i := 0; i < 4000; i++ {
		q := generateBearoff(BearoffRequest{Source: SourcePool}, rng)
		if !q.Generated {
			t.Fatalf("draw %d refused: %q", i, q.Refusal)
		}
		generated = append(generated, samplesOf(&q.Position.Board)...)
	}
	placed := uniformPlacements(rng, generated)

	realHist := wastageHistogram(real)
	genHist := wastageHistogram(generated)
	placedHist := wastageHistogram(placed)

	generatedDistance := totalVariation(realHist, genHist)
	placedDistance := totalVariation(realHist, placedHist)
	t.Logf("wastage histogram distance: generated %.3f, uniform placement %.3f (threshold %.2f, over %d real sides and %d generated)",
		generatedDistance, placedDistance, maxHistogramDistance, len(real), len(generated))

	if generatedDistance > maxHistogramDistance {
		t.Errorf("generated bear-offs sit %.3f from real ones, over the stated %.2f", generatedDistance, maxHistogramDistance)
	}
	if placedDistance < generatedDistance*minControlRatio {
		t.Errorf("a uniform placement sits %.3f away and the generator %.3f: the generator is no longer measurably more realistic than the placement ADR-0041 rejected",
			placedDistance, generatedDistance)
	}
}

// realBearoffSamples reads every bear-off this repository's match fixtures
// contain — ten real matches, races included, exactly the "imported library"
// ADR-0041 rule 5 compares against. Positions are kept only when they are in
// the exercise's own domain, so both samples are filtered identically.
func realBearoffSamples(t *testing.T) []bearoffSample {
	t.Helper()
	var out []bearoffSample
	for _, path := range matchFixtures(t) {
		graph, err := ingest.MapXG(path)
		if err != nil {
			t.Fatalf("reading %s: %v", filepath.Base(path), err)
		}
		for _, game := range graph.Games {
			for _, move := range game.Moves {
				if move.Position == nil {
					continue
				}
				out = append(out, samplesOf(&move.Position.Board)...)
			}
		}
	}
	return out
}

// matchFixtures locates testdata/ from this file's own path: a test binary
// runs in its own package's directory, so a relative path would depend on
// which package is asking (the same reason bearofftest resolves its fixtures
// this way).
func matchFixtures(t *testing.T) []string {
	t.Helper()
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate this test file")
	}
	root := filepath.Join(filepath.Dir(self), "..", "..", "..", "..")
	var files []string
	for _, pattern := range []string{"testdata/*.xg", "testdata/*/*/*.xg"} {
		found, err := filepath.Glob(filepath.Join(root, pattern))
		if err != nil {
			t.Fatalf("globbing %s: %v", pattern, err)
		}
		files = append(files, found...)
	}
	if len(files) == 0 {
		t.Fatalf("no match fixture found under %s", root)
	}
	return files
}

// samplesOf yields one sample per side, and nothing at all unless the position
// is in the Bearoff domain.
func samplesOf(b *domain.Board) []bearoffSample {
	if _, refusal := seedFromBoard(b); refusal != "" {
		return nil
	}
	epc := ComputeEPC(b)
	out := make([]bearoffSample, 0, 2)
	for _, side := range []Side{epc.Bottom, epc.Top} {
		if side.EPC == nil {
			return nil
		}
		out = append(out, bearoffSample{checkers: side.CheckerCount, wastage: side.EPC.Wastage})
	}
	return out
}

// uniformPlacements is the alternative ADR-0041 rejected: the same chequer
// counts, dropped uniformly on the six home points. Keeping the counts
// identical is what makes it a control of the SHAPE and not of the ply budget.
func uniformPlacements(rng *rand.Rand, like []bearoffSample) []bearoffSample {
	out := make([]bearoffSample, 0, len(like))
	for _, sample := range like {
		var side sideBoard
		for i := 0; i < sample.checkers; i++ {
			side[rng.IntN(6)]++
		}
		board := boardOf(bearoffSeed{black: side, white: side})
		epc := ComputeEPC(&board)
		if epc.Bottom.EPC == nil {
			continue
		}
		out = append(out, bearoffSample{checkers: sample.checkers, wastage: epc.Bottom.EPC.Wastage})
	}
	return out
}

// wastageHistogram bins the samples into whole pips of wastage and normalises.
func wastageHistogram(samples []bearoffSample) []float64 {
	hist := make([]float64, wastageBinHigh-wastageBinLow+1)
	for _, sample := range samples {
		bin := int(sample.wastage)
		if bin < wastageBinLow {
			bin = wastageBinLow
		}
		if bin > wastageBinHigh {
			bin = wastageBinHigh
		}
		hist[bin-wastageBinLow]++
	}
	for i := range hist {
		hist[i] /= float64(len(samples))
	}
	return hist
}

// totalVariation is half the L1 distance between two normalised histograms:
// the largest difference in probability the two can assign to any one event.
func totalVariation(a, b []float64) float64 {
	sum := 0.0
	for i := range a {
		if d := a[i] - b[i]; d > 0 {
			sum += d
		} else {
			sum -= d
		}
	}
	return sum / 2
}
