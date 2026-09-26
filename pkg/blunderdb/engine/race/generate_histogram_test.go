package race

import (
	"math/rand/v2"
	"path/filepath"
	"runtime"
	"sort"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// Is the generator REALISTIC? — measured, not asserted (ADR-0041 rule 5).
//
// The wastage histogram (EPC minus pip count — what the exercise trains, and
// what a placement gets wrong) of generated positions is compared with that of
// every bear-off in the ten real matches of testdata/, and with a control: the
// uniform placement ADR-0041 rejected. The control gives the threshold teeth —
// a generator degenerating into a placement fails it first.

// wastageBinLow/High bound the histogram, in whole pips. A bear-off's wastage
// runs from about seven (a tight bear-in) to the mid-twenties (chequers buried
// on the ace point); anything outside falls into the end bins.
const (
	wastageBinLow  = 6
	wastageBinHigh = 26
)

// Two bars, and neither is chosen for comfort.
//
// maxHistogramDistance is a RATCHET over deterministic inputs (fixtures, pinned
// dice), measured at 0.135: lower it when the generator improves, never raise
// it. The finite reference (157 positions) puts the noise ceiling at 0.118, so
// about 0.017 is a real difference, left standing: tuning the pool would fit
// this fixture set's noise, and widening the walk makes it worse (k ≤ 16:
// 0.134, k ≤ 24: 0.165), so ADR-0041's 0..10 stands. Closing it wants a wider
// reference.
//
// The rejected placement sits at 0.282, twice as far and past the noise.
const (
	maxHistogramDistance = 0.140
	minControlRatio      = 1.5
)

// noiseQuantile is which resample the ceiling is read at — the 95th, so a
// genuine sample crosses it one time in twenty. resamples is how many are
// drawn.
const (
	noiseQuantile = 0.95
	resamples     = 400
)

// bearoffSample is one side of one bear-off position.
type bearoffSample struct {
	checkers int
	wastage  float64
}

func TestGeneratedBearoffsMatchTheHistogramOfRealRaces(t *testing.T) {
	if testing.Short() {
		t.Skip("reads and parses ten match files")
	}
	positions := realBearoffPositions(t)
	// A fixture that moved would otherwise turn this test green by emptying
	// it: an empty reference matches everything.
	if len(positions) < 100 {
		t.Fatalf("only %d real bear-off positions in testdata/: the reference is too small to measure against", len(positions))
	}
	real := flatten(positions)

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
	noise := noiseCeiling(rng, positions, realHist)
	generatedDistance := totalVariation(realHist, wastageHistogram(generated))
	placedDistance := totalVariation(realHist, wastageHistogram(placed))

	t.Logf("wastage histogram distance: generated %.3f (ratchet %.3f), uniform placement %.3f, measured noise ceiling %.3f — %d real positions / %d sides against %d generated sides",
		generatedDistance, maxHistogramDistance, placedDistance, noise, len(positions), len(real), len(generated))

	if generatedDistance > maxHistogramDistance {
		t.Errorf("generated bear-offs sit %.3f from real ones, past the recorded %.3f: the ratchet only goes down", generatedDistance, maxHistogramDistance)
	}
	if placedDistance < generatedDistance*minControlRatio {
		t.Errorf("a uniform placement sits %.3f away and the generator %.3f: the generator is no longer measurably more realistic than the placement ADR-0041 rejected",
			placedDistance, generatedDistance)
	}
	// The control has to be outside the reference's own noise, or "further
	// away" would mean nothing.
	if placedDistance <= noise {
		t.Errorf("a uniform placement sits %.3f away, within the %.3f of sampling noise: this reference can no longer tell the two apart at all", placedDistance, noise)
	}
}

// noiseCeiling is how far a GENUINE sample of the reference's own size lands
// from the reference, by resampling positions with replacement.
//
// Positions and not sides: the two sides of one bear-off are drawn together in
// life, and resampling them apart would pretend the sample holds twice the
// information it does.
func noiseCeiling(rng *rand.Rand, positions [][]bearoffSample, reference []float64) float64 {
	distances := make([]float64, 0, resamples)
	draw := make([]bearoffSample, 0, len(positions)*2)
	for i := 0; i < resamples; i++ {
		draw = draw[:0]
		for j := 0; j < len(positions); j++ {
			draw = append(draw, positions[rng.IntN(len(positions))]...)
		}
		distances = append(distances, totalVariation(reference, wastageHistogram(draw)))
	}
	sort.Float64s(distances)
	at := int(float64(len(distances)-1) * noiseQuantile)
	return distances[at]
}

// realBearoffPositions reads every bear-off this repository's match fixtures
// contain — ten real matches, races included, exactly the "imported library"
// ADR-0041 rule 5 compares against. Positions are kept only when they are in
// the exercise's own domain, so both samples are filtered identically.
//
// A position is counted ONCE however many decisions it was recorded under
// (about three per position), or the bootstrap would believe it had three
// times the sample it has.
func realBearoffPositions(t *testing.T) [][]bearoffSample {
	t.Helper()
	seen := make(map[domain.Board]bool)
	var out [][]bearoffSample
	for _, path := range matchFixtures(t) {
		graph, err := ingest.MapXG(path)
		if err != nil {
			t.Fatalf("reading %s: %v", filepath.Base(path), err)
		}
		for _, game := range graph.Games {
			for _, move := range game.Moves {
				if move.Position == nil || seen[move.Position.Board] {
					continue
				}
				seen[move.Position.Board] = true
				if sides := samplesOf(&move.Position.Board); sides != nil {
					out = append(out, sides)
				}
			}
		}
	}
	return out
}

func flatten(positions [][]bearoffSample) []bearoffSample {
	out := make([]bearoffSample, 0, len(positions)*2)
	for _, sides := range positions {
		out = append(out, sides...)
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
