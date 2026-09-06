// SPDX-License-Identifier: MIT

package gammonnet

import (
	"fmt"
	"math"
	"os"
	"runtime"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// TestThreePlyMeasure is #272 (fiche I.16): MaxPly has been 4 since the port
// landed and DefaultConfig has always carried a filter for depth 3, but no
// measurement said what a 3-ply search buys — so nothing could decide whether
// to offer it as a setting.
//
// This is a measurement, not an assertion. It replays the gate's own corpus of
// real analysed decisions at 2-ply and at 3-ply and reports two things:
//
//  1. how often the deeper search picks a DIFFERENT move;
//  2. what that difference is worth, priced by the deeper search itself — the
//     equity 3-ply assigns to its own choice minus the equity it assigns to
//     2-ply's. Pricing by the deeper judge is the only direction that means
//     anything: asking 2-ply to grade 3-ply's choice would grade the judge.
//
// # Cost, measured on this build (TestProbeDecisionCost, AVX2, 16 logical cores)
//
//	2-ply canonical, one core      263 ms   13 438 evaluations
//	3-ply canonical, one core     25.0 s  1 166 992 evaluations
//	2-ply canonical, NumCPU         69 ms
//	3-ply canonical, NumCPU        6.8 s
//
// Ninety-five times the cost per decision, single core; ninety-nine on
// sixteen. That is one side of the scale; this test measures the other.
//
// # Result, measured on 2026-09-06 (40 checker decisions of the gate corpus, 16 workers)
//
//	move changed:      2/40 = 5.0%
//	time per decision: 2-ply 99 ms   3-ply 8.425 s   (85x)
//	equity gained where it changed: mean 0.0003  median 0.0005  max 0.0005
//	equity gained per decision:     0.0000
//
// **The setting is not offered, and this is why.** Eighty-five times the cost
// buys 0.0000 normalised equity per decision. On the two decisions where the
// deeper search did change its mind, the gain IT claimed for its own choice
// was at most 0.0005 — two orders of magnitude below 0.020, the threshold at
// which XG calls a decision an error at all. Forty decisions is a small
// sample and a larger one could turn up a rarer, bigger case; but the ratio is
// lopsided by four orders of magnitude, and no plausible tail moves a
// conclusion that far.
//
// What this does NOT say is that 3-ply is worthless in general — only that on
// THIS network, at the canonical filter, it does not pay for a user waiting on
// a panel. If the network is ever distilled or the filter widened (J.7), the
// measurement is one command away and the conclusion is re-decidable.
//
// Behind BLUNDERDB_THREE_PLY, like the other probes: it asserts nothing and
// costs minutes. BLUNDERDB_THREE_PLY_N caps the sample (default below).
const defaultThreePlyN = 60

func TestThreePlyMeasure(t *testing.T) {
	if os.Getenv("BLUNDERDB_THREE_PLY") == "" {
		t.Skip("set BLUNDERDB_THREE_PLY to measure; this test asserts nothing")
	}
	n := defaultThreePlyN
	if v := os.Getenv("BLUNDERDB_THREE_PLY_N"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil || parsed <= 0 {
			t.Fatalf("BLUNDERDB_THREE_PLY_N: invalid value %q", v)
		}
		n = parsed
	}

	net, err := embeddedNetwork()
	if err != nil {
		t.Fatal(err)
	}
	prune, err := embeddedPruneNetwork()
	if err != nil {
		t.Fatal(err)
	}

	// The gate's own corpus: real decisions at real scores with real cubes.
	var decisions []gateDecision
	for _, path := range []string{"../../../../testdata/test.sgf", "../../../../testdata/charlot1-charlot2_7p_2025-11-08-2305.sgf"} {
		mg, err := ingest.MapGnuBG(path)
		if err != nil {
			t.Fatalf("MapGnuBG(%s): %v", path, err)
		}
		decisions = append(decisions, extractDecisions(mg, "gnubg")...)
	}
	var checkers []gateDecision
	for _, d := range decisions {
		if d.kind == "checker" {
			checkers = append(checkers, d)
		}
	}
	if len(checkers) > n {
		checkers = checkers[:n]
	}
	if len(checkers) == 0 {
		t.Fatal("no checker decision in the corpus")
	}

	workers := runtime.NumCPU()
	var (
		changed  int
		compared int
		gains    []float64
		t2, t3   time.Duration
	)
	for _, d := range checkers {
		start := time.Now()
		two, ok2 := threePlyChoice(t, net, prune, DefaultConfig(2), d, workers)
		t2 += time.Since(start)

		start = time.Now()
		three, ok3 := threePlyChoice(t, net, prune, DefaultConfig(3), d, workers)
		t3 += time.Since(start)

		if !ok2 || !ok3 {
			continue
		}
		compared++
		if two.play == three.play {
			continue
		}
		changed++
		// Priced by the deeper search: what IT thinks its own choice is worth,
		// minus what it thinks 2-ply's choice is worth.
		gain := three.equityOf(two.play)
		if math.IsNaN(gain) {
			continue
		}
		gains = append(gains, math.Max(0, three.bestEquity-gain))
	}

	fmt.Printf("\n3-ply vs 2-ply, %d checker decisions of the gate corpus (%d workers)\n",
		compared, workers)
	fmt.Printf("  move changed:     %d/%d = %.1f%%\n", changed, compared,
		100*float64(changed)/float64(max(compared, 1)))
	fmt.Printf("  time per decision: 2-ply %v   3-ply %v   (%.0fx)\n",
		(t2 / time.Duration(max(compared, 1))).Round(time.Millisecond),
		(t3 / time.Duration(max(compared, 1))).Round(time.Millisecond),
		float64(t3)/float64(max(int(t2), 1)))
	if len(gains) > 0 {
		sort.Float64s(gains)
		var sum float64
		for _, g := range gains {
			sum += g
		}
		fmt.Printf("  equity gained where it changed (3-ply's own scale):\n")
		fmt.Printf("    mean %.4f  median %.4f  max %.4f\n",
			sum/float64(len(gains)), gains[len(gains)/2], gains[len(gains)-1])
		fmt.Printf("  equity gained per decision (changed or not): %.4f\n",
			sum/float64(compared))
	} else {
		fmt.Println("  the two depths never chose differently on this sample")
	}
}

// threePlyResult is one search's answer: the chosen play, and the equity the
// same search assigns to every candidate, so the deeper one can price the
// shallower one's choice.
type threePlyResult struct {
	play       Position
	bestEquity float64
	byPlay     map[Position]float64
}

func (r threePlyResult) equityOf(p Position) float64 {
	if v, ok := r.byPlay[p]; ok {
		return v
	}
	return math.NaN()
}

// threePlyChoice runs one search and returns its ranking of every legal play.
func threePlyChoice(t *testing.T, net, prune *Network, cfg SearchConfig, d gateDecision, workers int) (threePlyResult, bool) {
	t.Helper()
	pos := *d.pos
	pos.Dice = [2]int{d.dice[0], d.dice[1]}
	if len(domain.LegalMoves(&pos)) < 2 {
		return threePlyResult{}, false
	}
	var state *MatchState
	if pos.Score[0] >= 0 && pos.Score[1] >= 0 {
		m, _, ok := matchStateFor(&pos, d.crawford)
		if !ok {
			return threePlyResult{}, false
		}
		state = &m
	}
	s, ok := searcherFor(t, net, prune, cfg, state, CubeOwnerOf(&pos))
	if !ok {
		return threePlyResult{}, false
	}
	if workers > 1 {
		s = s.WithWorkers(workers)
	}
	gpos, err := FromDomain(&pos)
	if err != nil {
		return threePlyResult{}, false
	}
	out := make([]Candidate, MaxPlays)
	n, err := s.Plays(&gpos, d.dice[0], d.dice[1], out)
	if err != nil || n == 0 {
		return threePlyResult{}, false
	}
	res := threePlyResult{
		play:       out[0].Play.Result,
		bestEquity: out[0].Equity,
		byPlay:     make(map[Position]float64, n),
	}
	for _, p := range out[:n] {
		res.byPlay[p.Play.Result] = p.Equity
	}
	return res, true
}
