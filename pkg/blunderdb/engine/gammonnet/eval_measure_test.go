// SPDX-License-Identifier: MIT

package gammonnet

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine/race"
)

// TestEvalMeasure is #127, ADR-0012: before the "evaluated" regime can be
// offered on a race outside the exact table's domain, gammonNet 2-ply must be
// measured against the exact table where BOTH answer. This does not decide
// whether the exact table stays (it does, ADR-0012) — it decides what
// blunderDB is allowed to write in the documentation next to the word
// "evaluated".
//
// # Why this is money-only
//
// race.Evaluate/race.MoneyFromEntry only ever produce a MONEY cube verdict —
// the two-sided database's four planes carry no gammon-rate breakdown, so
// there is no exact oracle for a match-score verdict to compare against.
// "Evaluated at a match score" (ADR-0012's stated advantage over the old
// dead-cube chain) is therefore not something this measurement can check; it
// falls out of the codec/search/cube proof (#120-#123), not from an oracle
// that does not exist at a score.
//
// # Sampling, not exhaustion
//
// The embedded TS-06-06 domain (checkers 0..6 a side in the home board) holds
// 924 home-board configurations a side, ~852,000 (us, them) pairs. A 2-ply
// k=12 decision costs on the order of 13,400 big-network evaluations
// (ADR-0011); exhausting that domain is hours, not the "few minutes" #123's
// gate already set as the budget for a pre-merge recipe step. This test
// therefore draws a FIXED-SEED sample (BLUNDERDB_EVAL_MEASURE_N positions,
// default below) uniformly over home-board configurations with 1..6 checkers
// a side, deliberately excluding the closing move (0 checkers — no decision
// left to take).
//
// # Distance to the take point
//
// The two-sided table gives no gammon-rate breakdown either, so a true
// (ground-truth) take point cannot be computed from it. The bucketing instead
// uses gammonNet's OWN estimated take point (gammonnet.TakePoint, fed by its
// own CubeInputs) against the EXACT win probability: distance = |p_exact -
// tp_gammonNet|. This is deliberately the model-under-test's own take point,
// not an oracle one — it answers "how does gammonNet do near where gammonNet
// itself thinks the decision turns", which is the question a user reading a
// verdict near that line actually has.
//
// # Measured on 2026-08-29 (embedded TS-06-06, N=4000, seed 1, single core, 196s)
//
//	cube verdict agreement (overall): 3735/4000 = 93.4%
//	agreement by distance to gammonNet's own take point:
//	  <1%     61.1% (33/54)
//	  1-5%    88.3% (174/197)
//	  5-10%   91.5% (215/235)
//	  10-20%  94.0% (754/802)
//	  >=20%   94.4% (2559/2712)
//	|delta win prob|:        mean=0.00852 p50=0.00438 p95=0.03212 max=0.08295
//	|delta cubeful equity|:  mean=0.03881 p50=0.01794 p95=0.15068 max=0.40647
//
// The shape is exactly what ADR-0012 predicted rather than assumed: agreement
// is worst (61%) in the <1% band, right where a coin-flip decision is most
// sensitive to a small model disagreement, and climbs to 93-94% everywhere
// else. TS-06-11 (BLUNDERDB_TS11_PATH) was not set on this run; the extension
// exists and is exercised whenever the variable is provided, but was not run
// here. These are the numbers ADR-0012 requires next to the word "evaluated"
// in doc/source/manuel.rst (#126, not this ticket).
const defaultEvalMeasureN = 4000

func TestEvalMeasure(t *testing.T) {
	if os.Getenv("BLUNDERDB_EVAL_MEASURE") == "" {
		t.Skip("BLUNDERDB_EVAL_MEASURE not set; measurement skipped (recipe step, not part of go test ./...)")
	}
	n := defaultEvalMeasureN
	if v := os.Getenv("BLUNDERDB_EVAL_MEASURE_N"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil || parsed <= 0 {
			t.Fatalf("BLUNDERDB_EVAL_MEASURE_N: invalid value %q", v)
		}
		n = parsed
	}

	net, err := embeddedNetwork()
	if err != nil {
		t.Fatalf("Embedded network: %v", err)
	}
	prune, err := embeddedPruneNetwork()
	if err != nil {
		t.Fatalf("Embedded prune network: %v", err)
	}
	searcher := newSearcherWith(DefaultConfig(2), net, prune)

	ts := raceTestTwoSided(t)
	report := runEvalMeasure(t, searcher, ts, "embedded TS-06-06", n, 1)

	t.Logf("\n%s", report)

	if path := os.Getenv("BLUNDERDB_TS11_PATH"); path != "" {
		oracle, err := race.OpenTwoSided(path)
		if err != nil {
			t.Fatalf("BLUNDERDB_TS11_PATH: open: %v", err)
		}
		t.Cleanup(func() { oracle.Close() })
		searcher2 := newSearcherWith(DefaultConfig(2), net, prune)
		report2 := runEvalMeasure(t, searcher2, oracle, "TS-06-11 ("+path+")", n, 2)
		t.Logf("\n%s", report2)
	} else {
		t.Log("BLUNDERDB_TS11_PATH not set; TS-06-11 extension skipped")
	}
}

// evalSample is one measured (position, cube state) decision.
type evalSample struct {
	pExact    float64
	pGN       float64
	eqExact   float64 // exact table's continuation ("no double") equity
	eqGN      float64 // gammonNet's own EquityNoDouble at the same cube state
	distToTP  float64 // |p_exact - gammonNet's own take point|, see doc above
	verdictEx race.Verdict
	verdictGN CubeAction
	agree     bool
}

func randHome(rng *rand.Rand, n int) [6]int {
	var h [6]int
	for i := 0; i < n; i++ {
		h[rng.Intn(6)]++
	}
	return h
}

// buildPosition places us (on-roll player's home, points 1..6 own numbering)
// and them (opponent's home) as a pure-race position with White on roll.
// Choosing White on roll is WLOG: the encoding is proven symmetric by
// construction (ADR-0011's opening-position mirror test), and the cube model
// and race table are both stated from the on-roll player's point of view.
func buildPosition(us, them [6]int) Position {
	var p Position
	sumUs, sumThem := 0, 0
	for i := 0; i < 6; i++ {
		p.Points[i] = int8(us[i]) // White's home: index i = White's (i+1)-point
		sumUs += us[i]
		p.Points[23-i] = -int8(them[i]) // Black's home: index 23-i = Black's (i+1)-point
		sumThem += them[i]
	}
	p.Off[White] = uint8(NumCheckers - sumUs)
	p.Off[Black] = uint8(NumCheckers - sumThem)
	p.Turn = White
	return p
}

// raceCubeOwner and gnCubeOwner are the three cube dispositions tested per
// position, paired between race's and gammonNet's (differently named, same
// meaning) enums, both from the on-roll player's point of view.
var cubeStatesTested = []struct {
	name string
	rc   race.CubeState
	gn   CubeOwner
}{
	{"centered", race.CubeCentered, CubeCentred},
	{"owned", race.CubeOwned, CubeOwned},
	{"against", race.CubeAgainst, CubeOpponent},
}

func runEvalMeasure(t *testing.T, searcher *Searcher, ts *race.TwoSided, label string, n int, seed int64) string {
	t.Helper()
	rng := rand.New(rand.NewSource(seed))
	checkers := ts.Checkers()
	if checkers > 6 {
		checkers = 6 // gammonNet's search has no exact leaf oracle beyond this either way; the point of this test is the domain both answer.
	}

	var samples []evalSample
	attempts := 0
	for len(samples) < n && attempts < n*20 {
		attempts++
		nu := 1 + rng.Intn(checkers)
		nt := 1 + rng.Intn(checkers)
		us := randHome(rng, nu)
		them := randHome(rng, nt)
		if !ts.Covers(us, them) {
			continue
		}
		entry, err := ts.Lookup(us, them)
		if err != nil {
			t.Fatalf("%s: lookup: %v", label, err)
		}

		pos := buildPosition(us, them)
		probs, ok := searcher.Probs(&pos)
		if !ok {
			t.Fatalf("%s: Probs failed on a structurally built position", label)
		}

		for _, cs := range cubeStatesTested {
			money := race.MoneyFromEntry(entry, cs.rc)
			if cs.rc == race.CubeAgainst {
				// No decision on our side; race.MoneyFromEntry itself
				// documents there is nothing to verdict here. Skip.
				continue
			}
			eff := DefaultEfficiency(cs.gn)
			decision, ok := Decide(&probs, cs.gn, nil, eff, false)
			if !ok {
				t.Fatalf("%s: Decide failed on a structurally built position", label)
			}
			in := CubeInputsFromProbs(&probs)
			tp, tpOK := TakePoint(in, cs.gn, eff)
			dist := math.NaN()
			if tpOK {
				dist = math.Abs(entry.WinProb - tp)
			}

			gnVerdict := decision.Action
			if gnVerdict == TooGood {
				// The exact table's Verdict has no "too good" label — it
				// folds that case into NoDouble (money.go: the double
				// condition fails whenever DoublePass < NoDouble, which is
				// exactly the too-good case). Compare like with like.
				gnVerdict = NoDouble
			}
			agree := verdictsAgree(money.Verdict, gnVerdict)

			samples = append(samples, evalSample{
				pExact:    entry.WinProb,
				pGN:       float64(probs[PWin]),
				eqExact:   money.NoDouble,
				eqGN:      decision.EquityNoDouble,
				distToTP:  dist,
				verdictEx: money.Verdict,
				verdictGN: gnVerdict,
				agree:     agree,
			})
		}
	}
	if len(samples) < n {
		t.Fatalf("%s: only drew %d/%d valid samples in %d attempts", label, len(samples), n, attempts)
	}
	return formatEvalReport(label, samples)
}

func verdictsAgree(rc race.Verdict, gn CubeAction) bool {
	switch rc {
	case race.VerdictNoDouble:
		return gn == NoDouble
	case race.VerdictDoubleTake:
		return gn == DoubleTake
	case race.VerdictDoublePass:
		return gn == DoublePass
	default:
		return false
	}
}

func formatEvalReport(label string, samples []evalSample) string {
	buckets := []struct {
		name     string
		lo, hi   float64
		n, agree int
	}{
		{"<1%", 0, 0.01, 0, 0},
		{"1-5%", 0.01, 0.05, 0, 0},
		{"5-10%", 0.05, 0.10, 0, 0},
		{"10-20%", 0.10, 0.20, 0, 0},
		{">=20%", 0.20, math.Inf(1), 0, 0},
	}
	dp := make([]float64, 0, len(samples))
	deq := make([]float64, 0, len(samples))
	totalAgree := 0
	for _, s := range samples {
		dp = append(dp, math.Abs(s.pExact-s.pGN))
		deq = append(deq, math.Abs(s.eqExact-s.eqGN))
		if s.agree {
			totalAgree++
		}
		if math.IsNaN(s.distToTP) {
			continue
		}
		for i := range buckets {
			b := &buckets[i]
			if s.distToTP >= b.lo && s.distToTP < b.hi {
				b.n++
				if s.agree {
					b.agree++
				}
				break
			}
		}
	}
	sort.Float64s(dp)
	sort.Float64s(deq)

	out := fmt.Sprintf("=== eval measure: %s (%d decisions, money, 2-ply k=12) ===\n", label, len(samples))
	out += fmt.Sprintf("cube verdict agreement (overall): %d/%d = %.1f%%\n", totalAgree, len(samples), 100*float64(totalAgree)/float64(len(samples)))
	out += "agreement by distance to gammonNet's own take point:\n"
	for _, b := range buckets {
		rate := "n/a"
		if b.n > 0 {
			rate = fmt.Sprintf("%.1f%% (%d/%d)", 100*float64(b.agree)/float64(b.n), b.agree, b.n)
		}
		out += fmt.Sprintf("  %-7s %s\n", b.name, rate)
	}
	out += fmt.Sprintf("|delta win prob|: mean=%.5f p50=%.5f p95=%.5f max=%.5f\n", mean(dp), pct(dp, 0.50), pct(dp, 0.95), dp[len(dp)-1])
	out += fmt.Sprintf("|delta cubeful equity|: mean=%.5f p50=%.5f p95=%.5f max=%.5f\n", mean(deq), pct(deq, 0.50), pct(deq, 0.95), deq[len(deq)-1])
	return out
}

func mean(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := 0.0
	for _, x := range xs {
		s += x
	}
	return s / float64(len(xs))
}

// pct returns the value at quantile q (0..1) of a SORTED slice.
func pct(xs []float64, q float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	i := int(q * float64(len(xs)-1))
	return xs[i]
}
