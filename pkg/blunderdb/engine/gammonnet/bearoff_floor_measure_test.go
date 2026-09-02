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
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine/race"
)

// TestBearoffFloorMeasure answers one question: if the embedded TS-06-06
// two-sided database were NOT shipped, and the "evaluated" regime (gammonNet)
// became the floor on its domain, what would the user lose?
//
// It differs from TestEvalMeasure (ADR-0012, 2026-08-29) on three points:
//
//   - the searcher is configured exactly as the live panel's
//     evaluateRaceRegime does (ConfigForPosition: UseCube, CubeOwner, CubeX),
//     so the leaves are valued with the cube (ADR-0023) — the 08-29 run used
//     DefaultConfig, cubeless leaves;
//   - it measures the COST of following gammonNet's verdict in EXACT equity
//     (the table's own ND / min(DT, DP)), not only the agreement rate — a
//     disagreement by 0.001 and one by 0.3 are not the same disagreement;
//   - it reports 0, 1 and 2 ply side by side, so the reader sees what each
//     ply buys on this domain.
//
// Sampling is the same as TestEvalMeasure (uniform over home-board
// configurations with 1..6 checkers a side, seed 1), so the two are
// comparable. Recipe step, not part of go test ./...:
//
//	BLUNDERDB_BEAROFF_FLOOR=1 go test -run TestBearoffFloorMeasure -v ./pkg/blunderdb/engine/gammonnet/
//
// # Measured on 2026-09-03 (embedded TS-06-06, N=4000 positions, seed 1, gammonNet v1.2.1 weights)
//
//	                         0-ply          1-ply          2-ply
//	|dP win| mean / p95 / max   1.71 / 5.6 / 20.3 %   1.41 / 5.1 / 17.2 %   0.85 / 3.1 / 8.4 %
//	D/ND agreement, centred     94.0 %          95.8 %          94.8 %
//	D/ND agreement, owned       95.5 %          97.1 %          96.3 %
//	mean exact cost, centred    0.0095          0.0051          0.0065
//	mean exact cost, owned      0.0067          0.0034          0.0042
//	cost >= 0.08, centred       3.65 %          2.50 %          3.00 %
//	cost >= 0.08, owned         2.35 %          1.55 %          1.82 %
//	max cost                    0.500           0.336           0.446
//	take/pass agreement         95-96 %         96 %            96 %
//
// The verdict is dominated by one structural error, not by the network:
// gammonNet's cube model prices a LIVE cube (efficiency, recube vig), and
// the TS-06-06 domain is exactly where the cube is dead — one or two rolls
// from the end. The confusion is almost entirely DT->ND (2-ply centred:
// 189 of 206 disagreements): the exact table says double/take and gammonNet
// declines, undervaluing DT by 0.3-0.45 on last-roll positions where the
// exact answer is p > 0.5 => double. The reader's ply does not fix it: the
// distribution converges (2-ply pWin is 4x closer than 0-ply) but the cube
// model is applied on top of it identically. For comparison, the
// convolution ESTIMATOR of race.EstimatedWinProb is at sigma 0.05 % on the
// win probability — thirty times closer than 2-ply on this domain.
//
// Conclusion recorded in tasks/taille-binaire-2026-09.md: gammonNet 2-ply
// is NOT an acceptable floor for the money cube verdict on the TS-06-06
// domain; the exact table stays (ADR-0009/ADR-0012 hold). The 6.8 MB can
// still leave the binary another way: the table is derivable at runtime by
// backward induction (byte-identical to gnubg's makebearoff, ~2 s on 4
// cores), which keeps the "never estimated" invariant intact.
func TestBearoffFloorMeasure(t *testing.T) {
	if os.Getenv("BLUNDERDB_BEAROFF_FLOOR") == "" {
		t.Skip("BLUNDERDB_BEAROFF_FLOOR not set")
	}
	n := 4000
	if v := os.Getenv("BLUNDERDB_BEAROFF_FLOOR_N"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil || p <= 0 {
			t.Fatalf("BLUNDERDB_BEAROFF_FLOOR_N: %q", v)
		}
		n = p
	}
	maxPly := 2
	if v := os.Getenv("BLUNDERDB_BEAROFF_FLOOR_PLY"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil || p < 0 || p > MaxPly {
			t.Fatalf("BLUNDERDB_BEAROFF_FLOOR_PLY: %q", v)
		}
		maxPly = p
	}

	net, err := Embedded()
	if err != nil {
		t.Fatal(err)
	}
	prune, err := EmbeddedPruneNetwork()
	if err != nil {
		t.Fatal(err)
	}
	ts := race.EmbeddedTwoSided()

	// Draw the sample once; every ply and cube state sees the same positions.
	rng := rand.New(rand.NewSource(1))
	type sample struct {
		us, them [6]int
		entry    race.Entry
	}
	var samples []sample
	for len(samples) < n {
		us := randHome(rng, 1+rng.Intn(6))
		them := randHome(rng, 1+rng.Intn(6))
		e, err := ts.Lookup(us, them)
		if err != nil {
			t.Fatal(err)
		}
		samples = append(samples, sample{us, them, e})
	}

	states := []struct {
		name string
		rc   race.CubeState
		gn   CubeOwner
	}{
		{"centered", race.CubeCentered, CubeCentred},
		{"owned", race.CubeOwned, CubeOwned},
	}

	for ply := 0; ply <= maxPly; ply++ {
		for _, st := range states {
			cfg := DefaultConfig(ply)
			cfg.UseCube = true
			cfg.CubeOwner = st.gn
			cfg.CubeX = DefaultEfficiency(st.gn)
			searcher := NewSearcherWith(cfg, net, prune)
			eff := DefaultEfficiency(st.gn)

			var rows []floorRow
			t0 := time.Now()
			for _, s := range samples {
				pos := buildPosition(s.us, s.them)
				probs, ok := searcher.Probs(&pos)
				if !ok {
					t.Fatal("Probs failed")
				}
				dec, ok := Decide(&probs, st.gn, nil, eff, false)
				if !ok {
					t.Fatal("Decide failed")
				}
				money := race.MoneyFromEntry(s.entry, st.rc)
				rows = append(rows, floorRow{
					pExact:   s.entry.WinProb,
					pGN:      float64(probs[PWin]),
					cubeless: s.entry.Cubeless,
					cubelGN:  float64(MoneyEquity(&probs)),
					ndExact:  money.NoDouble,
					dtExact:  money.DoubleTake,
					dpExact:  money.DoublePass,
					ndGN:     dec.EquityNoDouble,
					dtGN:     dec.EquityDoubleTake,
					dpGN:     dec.EquityDoublePass,
					vExact:   money.Verdict,
					vGN:      dec.Action,
					checkers: sum6(s.us) + sum6(s.them),
					us:       s.us,
					them:     s.them,
				})
			}
			t.Logf("(%s)", time.Since(t0).Round(time.Millisecond))
			t.Logf("\n%s", formatFloorReport(fmt.Sprintf("%d-ply, cube %s", ply, st.name), rows))
		}
	}
}

type floorRow struct {
	pExact, pGN               float64
	cubeless, cubelGN         float64
	ndExact, dtExact, dpExact float64
	ndGN, dtGN, dpGN          float64
	vExact                    race.Verdict
	vGN                       CubeAction
	checkers                  int
	us, them                  [6]int
}

func sum6(h [6]int) int {
	s := 0
	for _, x := range h {
		s += x
	}
	return s
}

// exactValue is the exact equity of an action, opponent replying optimally
// to a double (takes iff DT < DP). TooGood is "play on" = ND.
func (r floorRow) exactValue(a CubeAction) float64 {
	switch a {
	case DoubleTake, DoublePass:
		return math.Min(r.dtExact, r.dpExact)
	default:
		return r.ndExact
	}
}

func (r floorRow) exactAction() CubeAction {
	switch r.vExact {
	case race.VerdictDoubleTake:
		return DoubleTake
	case race.VerdictDoublePass:
		return DoublePass
	default:
		return NoDouble
	}
}

func actionName(a CubeAction) string {
	switch a {
	case NoDouble:
		return "ND"
	case DoubleTake:
		return "DT"
	case DoublePass:
		return "DP"
	case TooGood:
		return "TG"
	}
	return "?"
}

func formatFloorReport(label string, rows []floorRow) string {
	var dp, dcl, dnd, cost, takeCost []float64
	agreeD, agreeTP := 0, 0
	nDoubles := 0
	confusion := map[string]int{}
	type bucket struct {
		name   string
		lo, hi float64
		n      int
		agree  int
		cost   float64
	}
	buckets := []bucket{
		{"<0.01", 0, 0.01, 0, 0, 0},
		{"0.01-0.05", 0.01, 0.05, 0, 0, 0},
		{"0.05-0.10", 0.05, 0.10, 0, 0, 0},
		{">=0.10", 0.10, math.Inf(1), 0, 0, 0},
	}
	blunder2, blunder5, blunder8 := 0, 0, 0
	type worstCase struct {
		cost float64
		r    floorRow
		gn   CubeAction
	}
	var worst []worstCase
	for _, r := range rows {
		dp = append(dp, math.Abs(r.pExact-r.pGN))
		dcl = append(dcl, math.Abs(r.cubeless-r.cubelGN))
		dnd = append(dnd, math.Abs(r.ndExact-r.ndGN))

		ex := r.exactAction()
		gnRaw := r.vGN
		gn := gnRaw
		if gn == TooGood {
			gn = NoDouble
		}
		// Double / no double: the mover's decision.
		exD := ex != NoDouble
		gnD := gn != NoDouble
		c := r.exactValue(ex) - r.exactValue(gn)
		if c < 0 {
			c = 0 // quantisation ties
		}
		cost = append(cost, c)
		if exD == gnD {
			agreeD++
		}
		if c >= 0.02 {
			blunder2++
		}
		if c >= 0.05 {
			blunder5++
		}
		if c >= 0.08 {
			blunder8++
		}
		confusion[actionName(ex)+"->"+actionName(gnRaw)]++
		worst = append(worst, worstCase{c, r, gnRaw})

		// Take / pass: the opponent's decision, when the exact table says
		// double. Cost to the taker of following gammonNet's take/pass.
		if exD {
			nDoubles++
			exTake := r.dtExact < r.dpExact
			gnTake := r.dtGN < r.dpGN
			tc := 0.0
			if exTake != gnTake {
				tc = math.Abs(r.dtExact - r.dpExact)
			} else {
				agreeTP++
			}
			takeCost = append(takeCost, tc)
		}

		// Bucket by the exact margin between doubling and not doubling.
		margin := math.Abs(r.ndExact - math.Min(r.dtExact, r.dpExact))
		for i := range buckets {
			b := &buckets[i]
			if margin >= b.lo && margin < b.hi {
				b.n++
				if exD == gnD {
					b.agree++
				}
				b.cost += c
				break
			}
		}
	}
	sort.Float64s(dp)
	sort.Float64s(dcl)
	sort.Float64s(dnd)
	sorted := append([]float64(nil), cost...)
	sort.Float64s(sorted)
	sortedTake := append([]float64(nil), takeCost...)
	sort.Float64s(sortedTake)

	N := float64(len(rows))
	out := fmt.Sprintf("=== bearoff floor: %s (%d positions, money, UseCube leaves) ===\n", label, len(rows))
	out += fmt.Sprintf("|dP win|           mean=%.5f p50=%.5f p95=%.5f p99=%.5f max=%.5f\n", mean(dp), pct(dp, .5), pct(dp, .95), pct(dp, .99), dp[len(dp)-1])
	out += fmt.Sprintf("|d cubeless eq|    mean=%.5f p50=%.5f p95=%.5f p99=%.5f max=%.5f\n", mean(dcl), pct(dcl, .5), pct(dcl, .95), pct(dcl, .99), dcl[len(dcl)-1])
	out += fmt.Sprintf("|d ND cubeful eq|  mean=%.5f p50=%.5f p95=%.5f p99=%.5f max=%.5f\n", mean(dnd), pct(dnd, .5), pct(dnd, .95), pct(dnd, .99), dnd[len(dnd)-1])
	out += fmt.Sprintf("double/no-double agreement: %d/%d = %.1f%%\n", agreeD, len(rows), 100*float64(agreeD)/N)
	out += fmt.Sprintf("exact-equity cost of following gammonNet's D/ND: mean=%.5f p95=%.5f p99=%.5f max=%.5f\n", mean(cost), pct(sorted, .95), pct(sorted, .99), sorted[len(sorted)-1])
	out += fmt.Sprintf("  cost >= 0.02: %d (%.2f%%)   >= 0.05: %d (%.2f%%)   >= 0.08: %d (%.2f%%)\n", blunder2, 100*float64(blunder2)/N, blunder5, 100*float64(blunder5)/N, blunder8, 100*float64(blunder8)/N)
	if nDoubles > 0 {
		out += fmt.Sprintf("take/pass agreement when exact says double: %d/%d = %.1f%%; taker's exact cost mean=%.5f p95=%.5f max=%.5f\n", agreeTP, nDoubles, 100*float64(agreeTP)/float64(nDoubles), mean(takeCost), pct(sortedTake, .95), sortedTake[len(sortedTake)-1])
	}
	out += "by exact margin |ND - D| (closeness of the true decision): agreement, mean cost\n"
	for _, b := range buckets {
		if b.n == 0 {
			continue
		}
		out += fmt.Sprintf("  %-10s n=%-5d agree=%5.1f%%  mean cost=%.5f\n", b.name, b.n, 100*float64(b.agree)/float64(b.n), b.cost/float64(b.n))
	}
	out += "confusion exact->gammonNet:"
	keys := make([]string, 0, len(confusion))
	for k := range confusion {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		out += fmt.Sprintf(" %s=%d", k, confusion[k])
	}
	out += "\n"
	sort.Slice(worst, func(i, j int) bool { return worst[i].cost > worst[j].cost })
	out += "worst 6 (us | them | pExact pGN | exact ND DT DP | gammonNet action ND DT DP):\n"
	for i := 0; i < 6 && i < len(worst); i++ {
		w := worst[i]
		out += fmt.Sprintf("  cost=%.3f %v | %v | %.3f %.3f | %s %.3f %.3f %.3f | %s %.3f %.3f %.3f\n", w.cost, w.r.us, w.r.them, w.r.pExact, w.r.pGN, actionName(w.r.exactAction()), w.r.ndExact, w.r.dtExact, w.r.dpExact, actionName(w.gn), w.r.ndGN, w.r.dtGN, w.r.dpGN)
	}
	return out
}
