package training

import (
	"math/rand/v2"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// Are the pool's races REALISTIC? — measured, as race/generate_histogram_test.go
// measures the bear-offs (ADR-0041 rule 5).
//
// The axis is each side's pip count: what a race question is about, and the
// number a hand-made race gets wrong first — a placement of fifteen chequers
// sits in a narrow band around its mean, a race that was played into spreads
// from the midpoint still to clear down to nearly home. The reference is every
// race of the ten real matches of testdata/ that the Evaluation pool is meant
// to stand for (realRaces): contact just broken, then up to ten plies, and not
// yet a pure bear-off — that is the Bearoff exercise's domain.
//
// Only the WALK is measured: the truth attached afterwards does not move a
// chequer.
//
// # The ratchet
//
// maxPipHistogramDistance is a ratchet over deterministic inputs, measured at
// 0.218: it may go down, never up. The reference's sampling noise is 0.099;
// the remaining difference (sides walked down to a handful of pips, back-game
// races of 150+ pips no pool shape stands for) is left standing, since
// tuning bin by bin on 181 positions would fit this fixture set.
//
// The uniform placement ADR-0041 rejected sits at 0.715: that assertion falls
// first if the pool ever degenerates into a placement.

const (
	pipBinWidth = 10
	pipBins     = 17 // 0..169, the last bin catching everything above
)

func TestGeneratedRacesMatchThePipHistogramOfRealRaces(t *testing.T) {
	if testing.Short() {
		t.Skip("reads and parses ten match files")
	}
	positions := realRaces(t)
	if len(positions) < 100 {
		t.Fatalf("only %d real races in testdata/: the reference is too small to measure against", len(positions))
	}
	realHist := pipHistogram(flattenPips(positions))

	rng := rand.New(rand.NewPCG(0x9e3779b9, 0x7f4a7c15))
	w := walk{walker: generator.walker, rng: rng, now: time.Now, deadline: time.Now().Add(time.Hour)}
	var generated, placed []int
	for i := 0; i < 2000; i++ {
		q := w.playOut(poolSeed(rng), rng.IntN(maxPoolPlies+1), SourcePool, "")
		generated = append(generated, pipsOf(&q.Position.Board)...)
		placed = append(placed, uniformRacePips(rng)...)
	}

	noise := pipNoiseCeiling(rng, positions, realHist)
	generatedDistance := totalVariation(realHist, pipHistogram(generated))
	placedDistance := totalVariation(realHist, pipHistogram(placed))
	t.Logf("pip histogram distance: generated %.3f (ratchet %.3f), uniform placement %.3f, noise ceiling %.3f — %d real races against %d generated sides",
		generatedDistance, maxPipHistogramDistance, placedDistance, noise, len(positions), len(generated))

	if generatedDistance > maxPipHistogramDistance {
		t.Errorf("generated races sit %.3f from real ones, past the recorded %.3f: the ratchet only goes down", generatedDistance, maxPipHistogramDistance)
	}
	if placedDistance < generatedDistance*minPipControlRatio {
		t.Errorf("a uniform placement sits %.3f away and the generator %.3f: the pool is no longer measurably more realistic than a placement", placedDistance, generatedDistance)
	}
	if placedDistance <= noise {
		t.Errorf("a uniform placement sits %.3f away, within the %.3f of sampling noise: the reference can no longer tell them apart", placedDistance, noise)
	}
}

// realRaces reads the races of the match fixtures that the pool stands for:
// a position with no contact, not yet a pure bear-off, reached at most
// maxPoolPlies plies after the game's last contact position — « contact just
// broken, then 0 to 10 plies », read off real games in the order they were
// played. Each distinct board counts once, and only the first time it is met.
func realRaces(t *testing.T) [][]int {
	t.Helper()
	seen := make(map[domain.Board]bool)
	var out [][]int
	for _, path := range matchFixtures(t) {
		graph, err := ingest.MapXG(path)
		if err != nil {
			t.Fatalf("reading %s: %v", filepath.Base(path), err)
		}
		for _, game := range graph.Games {
			// sinceContact counts the distinct boards played since the last
			// contact one; -1 until contact has been seen at all.
			sinceContact := -1
			var previous domain.Board
			for _, move := range game.Moves {
				if move.Position == nil || move.Position.Board == previous {
					continue // the same position recorded under another decision
				}
				b := move.Position.Board
				previous = b
				if hasContact(&b) {
					sinceContact = 0
					continue
				}
				if sinceContact < 0 || emptyBoard(&b) {
					continue
				}
				sinceContact++
				if sinceContact > maxPoolPlies+1 || pureBearoff(&b) || seen[b] {
					continue
				}
				seen[b] = true
				out = append(out, pipsOf(&b))
			}
		}
	}
	return out
}

func pureBearoff(b *domain.Board) bool {
	for i := 1; i <= domain.NumPoints; i++ {
		pt := b.Points[i]
		if pt.Checkers == 0 {
			continue
		}
		if (pt.Color == domain.Black && i > 6) || (pt.Color == domain.White && i < 19) {
			return false
		}
	}
	return true
}

// pipsOf is both sides' pip counts, Black's then White's.
func pipsOf(b *domain.Board) []int {
	black, white := 0, 0
	for i := 1; i <= domain.NumPoints; i++ {
		pt := b.Points[i]
		switch {
		case pt.Checkers == 0:
		case pt.Color == domain.Black:
			black += i * pt.Checkers
		case pt.Color == domain.White:
			white += (25 - i) * pt.Checkers
		}
	}
	return []int{black, white}
}

// uniformRacePips is the control: fifteen chequers a side dropped uniformly on
// the 1- to 12-point, which never makes contact — the « weighted random
// placement » ADR-0041 rejected, at its simplest.
func uniformRacePips(rng *rand.Rand) []int {
	out := make([]int, 2)
	for s := range out {
		for c := 0; c < 15; c++ {
			out[s] += 1 + rng.IntN(12)
		}
	}
	return out
}

func flattenPips(positions [][]int) []int {
	var out []int
	for _, p := range positions {
		out = append(out, p...)
	}
	return out
}

func pipHistogram(pips []int) []float64 {
	h := make([]float64, pipBins)
	for _, p := range pips {
		bin := p / pipBinWidth
		if bin >= pipBins {
			bin = pipBins - 1
		}
		h[bin]++
	}
	for i := range h {
		h[i] /= float64(len(pips))
	}
	return h
}

func totalVariation(a, b []float64) float64 {
	d := 0.0
	for i := range a {
		if a[i] > b[i] {
			d += a[i] - b[i]
		} else {
			d += b[i] - a[i]
		}
	}
	return d / 2
}

// pipNoiseCeiling is how far a genuine sample of the reference's own size
// lands from it — positions resampled with replacement, the 95th resample.
func pipNoiseCeiling(rng *rand.Rand, positions [][]int, reference []float64) float64 {
	const resamples = 400
	distances := make([]float64, 0, resamples)
	for i := 0; i < resamples; i++ {
		draw := make([]int, 0, len(positions)*2)
		for j := 0; j < len(positions); j++ {
			draw = append(draw, positions[rng.IntN(len(positions))]...)
		}
		distances = append(distances, totalVariation(reference, pipHistogram(draw)))
	}
	sort.Float64s(distances)
	return distances[int(float64(len(distances)-1)*0.95)]
}

// matchFixtures locates testdata/ from this file's own path.
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
		t.Fatal("no match fixtures found under testdata/")
	}
	return files
}

const (
	maxPipHistogramDistance = 0.220
	minPipControlRatio      = 1.5
)
