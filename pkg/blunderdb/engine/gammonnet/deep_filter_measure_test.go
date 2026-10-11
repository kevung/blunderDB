// SPDX-License-Identifier: MIT

package gammonnet

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// TestDeepFilterMeasure prices the 3- and 4-ply move filters: for each named
// configuration it reports the time and big-network evaluations of one
// single-threaded checker decision, and against a reference configuration of
// the same depth how often the chosen play differs and what the reference
// says that difference costs.
//
// Positions come from BLUNDERDB_DEEP_POS (a TSV whose first two columns are an
// XGID and the dice, as "31"), else from the gate corpus.
// BLUNDERDB_DEEP_N caps the sample, BLUNDERDB_DEEP_STEP takes every n-th
// position, BLUNDERDB_DEEP_CFG lists "ply:name" pairs (the first of a ply is
// its reference), BLUNDERDB_DEEP_PAR runs that many decisions at once (each
// one serial). Asserts nothing.
//
// Measured on 19 decisions of one real 9-point match (every 12th), 4 to 8
// decisions at once, each serial; "legacy" deepened five replies at every
// depth past 2, "default" deepens one per interior node:
//
//	config        s/decision  big-net evals  agree   loss mean / max
//	2:default        0.22          6 268
//	3:legacy        34-38        460 677     (ref)
//	3:default         5.0         98 911     16/19   0.00006 / 0.00095
//	4:r0.8.16       314        6 274 014     (ref, 9 decisions)
//	4:default        93        1 970 309      8/9    0.00020 / 0.00181
//
// The 4-ply legacy filter did not finish a decision in 900 s. Pre-roll (cube)
// distributions at 3 ply: 10.4 s -> 4.9 s, mean |dWin| 0.00037.
func TestDeepFilterMeasure(t *testing.T) {
	if os.Getenv("BLUNDERDB_DEEP") == "" {
		t.Skip("set BLUNDERDB_DEEP to measure; this test asserts nothing")
	}
	decisions := deepMeasureDecisions(t)
	specs := strings.Split(envOr("BLUNDERDB_DEEP_CFG", "3:legacy,3:default"), ",")
	par := envInt("BLUNDERDB_DEEP_PAR", 1)

	net, err := embeddedNetwork()
	if err != nil {
		t.Fatal(err)
	}
	prune, err := embeddedPruneNetwork()
	if err != nil {
		t.Fatal(err)
	}

	type outcome struct {
		res   threePlyResult
		ok    bool
		dt    time.Duration
		evals uint64
	}
	results := make([][]outcome, len(specs))
	for ci, spec := range specs {
		ply, name := parseDeepSpec(t, spec)
		results[ci] = make([]outcome, len(decisions))
		var wg sync.WaitGroup
		sem := make(chan struct{}, par)
		for di := range decisions {
			wg.Add(1)
			sem <- struct{}{}
			go func(di int) {
				defer wg.Done()
				defer func() { <-sem }()
				start := time.Now()
				var res threePlyResult
				var evals uint64
				var ok bool
				if name[0] == 's' || name[0] == 'g' {
					res, evals, ok = deepStaged(t, net, prune, ply, name, decisions[di])
				} else {
					res, evals, ok = deepChoice(net, prune, deepNamedConfig(t, ply, name), decisions[di])
				}
				results[ci][di] = outcome{res, ok, time.Since(start), evals}
			}(di)
		}
		wg.Wait()
	}

	fmt.Printf("\n%d checker decisions, %d at once, each serial\n", len(decisions), par)
	fmt.Printf("%-18s %10s %10s %12s %8s %10s %10s %10s\n",
		"config", "mean s", "max s", "mean evals", "agree", "loss mean", "loss max", "|dEq| mean")
	refOf := map[int]int{}
	for ci, spec := range specs {
		ply, _ := parseDeepSpec(t, spec)
		ref, has := refOf[ply]
		if !has {
			refOf[ply] = ci
			ref = ci
		}
		var tot, worst time.Duration
		var evals uint64
		var n, agree int
		var losses, dEq []float64
		for di := range decisions {
			o, r := results[ci][di], results[ref][di]
			if !o.ok || !r.ok {
				continue
			}
			n++
			tot += o.dt
			worst = max(worst, o.dt)
			evals += o.evals
			if o.res.play == r.res.play {
				agree++
			}
			if v := r.res.equityOf(o.res.play); !math.IsNaN(v) {
				losses = append(losses, math.Max(0, r.res.bestEquity-v))
			}
			dEq = append(dEq, math.Abs(o.res.bestEquity-r.res.bestEquity))
		}
		if n == 0 {
			continue
		}
		fmt.Printf("%-18s %10.3f %10.3f %12d %4d/%-3d %10.5f %10.5f %10.5f\n", spec,
			tot.Seconds()/float64(n), worst.Seconds(), evals/uint64(n), agree, n,
			mean(losses), maxOf(losses), mean(dEq))
	}
}

func deepMeasureDecisions(t *testing.T) []gateDecision {
	t.Helper()
	var out []gateDecision
	if path := os.Getenv("BLUNDERDB_DEEP_POS"); path != "" {
		f, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			cols := strings.Split(sc.Text(), "\t")
			if len(cols) < 2 || len(cols[1]) != 2 {
				continue
			}
			pos, err := domain.DecodeXGID(cols[0])
			if err != nil {
				t.Fatalf("XGID %q: %v", cols[0], err)
			}
			crawford, _ := domain.XGIDCrawfordGame(cols[0])
			d := gateDecision{kind: "checker", pos: &pos, crawford: crawford}
			d.dice = [2]int{int(cols[1][0] - '0'), int(cols[1][1] - '0')}
			out = append(out, d)
		}
	} else {
		for _, path := range []string{"../../../../testdata/test.sgf", "../../../../testdata/charlot1-charlot2_7p_2025-11-08-2305.sgf"} {
			mg, err := ingest.MapGnuBG(path)
			if err != nil {
				t.Fatalf("MapGnuBG(%s): %v", path, err)
			}
			for _, d := range extractDecisions(mg, "gnubg") {
				if d.kind == "checker" {
					out = append(out, d)
				}
			}
		}
	}
	step := envInt("BLUNDERDB_DEEP_STEP", 1)
	var kept []gateDecision
	for i := 0; i < len(out); i += step {
		pos := *out[i].pos
		pos.Dice = out[i].dice
		if len(domain.LegalMoves(&pos)) >= 2 {
			kept = append(kept, out[i])
		}
	}
	if n := envInt("BLUNDERDB_DEEP_N", 20); len(kept) > n {
		kept = kept[:n]
	}
	return kept
}

func parseDeepSpec(t *testing.T, spec string) (int, string) {
	t.Helper()
	ply, name, ok := strings.Cut(spec, ":")
	k, err := strconv.Atoi(ply)
	if !ok || err != nil {
		t.Fatalf("BLUNDERDB_DEEP_CFG: %q is not ply:name", spec)
	}
	return k, name
}

// deepNamedConfig builds the named filter at ply. "legacy" is the plain count
// of five at every depth past the published filter; "default" is
// DefaultConfig (staged root at 3 and 4 ply); "rA.E.T" deepens A+E candidates within T at the root, and
// one (the shallow best) at every interior depth past the published filter;
// "pA.E.T" is the same with the published depth-2 filter kept inside. A
// "/kN" suffix sets PruneK to N.
func deepNamedConfig(t *testing.T, ply int, name string) SearchConfig {
	if base, k, ok := strings.Cut(name, "/k"); ok {
		cfg := deepNamedConfig(t, ply, base)
		n, err := strconv.Atoi(k)
		if err != nil {
			t.Errorf("unknown config %q", name)
		}
		cfg.PruneK = n
		return cfg
	}
	if name == "default" {
		return DefaultConfig(ply)
	}
	// Every other shape names an unstaged root: "r1.2.04" is the 3- and
	// 4-ply default before the root was staged.
	cfg := DefaultConfig(ply)
	_ = cfg.SetRootStage(0, 0, 0, 0)
	switch {
	case name == "legacy":
		for d := len(defaultLevel.Filter); d <= MaxPly; d++ {
			_ = cfg.SetFilter(d, 5, 0, 0)
		}
		_ = cfg.SetFilter(2, defaultLevel.Filter[2], defaultLevel.FilterExtra[2], defaultLevel.FilterThreshold[2])
		return cfg
	case len(name) > 1 && (name[0] == 'r' || name[0] == 'p'):
		parts := strings.SplitN(name[1:], ".", 3)
		if len(parts) != 3 {
			break
		}
		a, err1 := strconv.Atoi(parts[0])
		e, err2 := strconv.Atoi(parts[1])
		th, err3 := strconv.ParseFloat("0."+parts[2], 64)
		if err1 != nil || err2 != nil || err3 != nil {
			break
		}
		for d := 2; d < ply; d++ {
			_ = cfg.SetFilter(d, 1, 0, 0)
		}
		if name[0] == 'p' && ply > 2 {
			_ = cfg.SetFilter(2, defaultLevel.Filter[2], defaultLevel.FilterExtra[2], defaultLevel.FilterThreshold[2])
		}
		_ = cfg.SetFilter(ply, a, e, th)
		return cfg
	}
	t.Errorf("unknown config %q", name)
	return cfg
}

// deepStaged emulates a staged root, gnubg's way: the canonical 2-ply ranks
// every play, the plays its sN.E.T filter keeps are searched at ply with one
// reply per interior node, the rest keep their 2-ply equity. "sA.E.T/kN"
// sets the deep pass's PruneK; "g" instead of "s" widens the 2-ply pass to
// gnubg's (0, 8, 0.16).
func deepStaged(t *testing.T, net, prune *Network, ply int, name string, d gateDecision) (threePlyResult, uint64, bool) {
	base, k, hasK := strings.Cut(name, "/k")
	parts := strings.SplitN(base[1:], ".", 3)
	a, _ := strconv.Atoi(parts[0])
	e, _ := strconv.Atoi(parts[1])
	th, _ := strconv.ParseFloat("0."+parts[2], 64)
	first := DefaultConfig(2)
	if name[0] == 'g' {
		_ = first.SetFilter(2, 0, 8, 0.16)
	}
	two, evals2, ok := deepChoiceRanked(net, prune, first, d)
	if !ok {
		return threePlyResult{}, 0, false
	}
	deepName := "r1.0.0"
	if hasK {
		deepName += "/k" + k
	}
	cfg := deepNamedConfig(t, ply, deepName)
	var filt SearchConfig
	_ = filt.SetFilter(ply, a, e, th)
	keep := filt.filterSurvivors(ply, two.ranked, len(two.ranked))
	s, owner, ok := deepSearcher(net, prune, cfg, d)
	if !ok {
		return threePlyResult{}, 0, false
	}
	s.seedMatchState(s.matchState())
	cands := append([]Candidate(nil), two.ranked...)
	for i := 0; i < keep; i++ {
		if cands[i].Play.Result.isOver() {
			continue
		}
		v, ok := s.positionEquity(&cands[i].Play.Result, ply, 1, s.childMatchState(0), owner.Mirror())
		if !ok {
			return threePlyResult{}, 0, false
		}
		cands[i].Equity = -v
	}
	sortByEquity(cands[:keep])
	res := threePlyResult{play: cands[0].Play.Result, bestEquity: cands[0].Equity, byPlay: map[Position]float64{}}
	for _, c := range cands {
		res.byPlay[c.Play.Result] = c.Equity
	}
	evals, _, _ := s.Counters()
	return res, evals + evals2, true
}

type rankedResult struct {
	ranked []Candidate
}

func deepChoiceRanked(net, prune *Network, cfg SearchConfig, d gateDecision) (rankedResult, uint64, bool) {
	s, _, ok := deepSearcher(net, prune, cfg, d)
	if !ok {
		return rankedResult{}, 0, false
	}
	pos := *d.pos
	gpos, err := FromDomain(&pos)
	if err != nil {
		return rankedResult{}, 0, false
	}
	out := make([]Candidate, MaxPlays)
	n, err := s.Plays(&gpos, d.dice[0], d.dice[1], out)
	if err != nil || n == 0 {
		return rankedResult{}, 0, false
	}
	evals, _, _ := s.Counters()
	return rankedResult{ranked: out[:n]}, evals, true
}

// deepSearcher builds a serial searcher valued as the production batch values d.
func deepSearcher(net, prune *Network, cfg SearchConfig, d gateDecision) (*Searcher, CubeOwner, bool) {
	pos := *d.pos
	pos.Dice = d.dice
	if pos.Score[0] >= 0 && pos.Score[1] >= 0 {
		m, _, ok := matchStateFor(&pos, d.crawford)
		if !ok {
			return nil, 0, false
		}
		cfg.UseMatch = true
		cfg.Match = m
	}
	owner := CubeOwnerOf(&pos)
	cfg.UseCube = true
	cfg.CubeOwner = owner
	cfg.CubeX = DefaultEfficiency(owner)
	return newSearcherWith(cfg, net, prune), owner, true
}

// deepChoice is threePlyChoice, serial, with the evaluations it cost.
func deepChoice(net, prune *Network, cfg SearchConfig, d gateDecision) (threePlyResult, uint64, bool) {
	s, _, ok := deepSearcher(net, prune, cfg, d)
	if !ok {
		return threePlyResult{}, 0, false
	}
	pos := *d.pos
	gpos, err := FromDomain(&pos)
	if err != nil {
		return threePlyResult{}, 0, false
	}
	out := make([]Candidate, MaxPlays)
	n, err := s.Plays(&gpos, d.dice[0], d.dice[1], out)
	if err != nil || n == 0 {
		return threePlyResult{}, 0, false
	}
	res := threePlyResult{play: out[0].Play.Result, bestEquity: out[0].Equity, byPlay: make(map[Position]float64, n)}
	for _, p := range out[:n] {
		res.byPlay[p.Play.Result] = p.Equity
	}
	evals, _, _ := s.Counters()
	return res, evals, true
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func maxOf(xs []float64) float64 {
	if len(xs) == 0 {
		return math.NaN()
	}
	c := append([]float64(nil), xs...)
	sort.Float64s(c)
	return c[len(c)-1]
}

// TestDeepCubeMeasure times the pre-roll distribution the cube decision reads
// (Probs) at each named configuration, single-threaded, on the positions of
// TestDeepFilterMeasure taken before their roll, and reports the mean gap in
// winning and gammon chances against the first configuration of the same ply.
// Behind BLUNDERDB_DEEP_CUBE; asserts nothing.
func TestDeepCubeMeasure(t *testing.T) {
	if os.Getenv("BLUNDERDB_DEEP_CUBE") == "" {
		t.Skip("set BLUNDERDB_DEEP_CUBE to measure; this test asserts nothing")
	}
	decisions := deepMeasureDecisions(t)
	specs := strings.Split(envOr("BLUNDERDB_DEEP_CFG", "3:legacy,3:default"), ",")
	net, _ := embeddedNetwork()
	prune, _ := embeddedPruneNetwork()
	type outcome struct {
		probs [NumOutputs]float32
		dt    time.Duration
		ok    bool
	}
	res := make([][]outcome, len(specs))
	par := envInt("BLUNDERDB_DEEP_PAR", 1)
	for ci, spec := range specs {
		ply, name := parseDeepSpec(t, spec)
		res[ci] = make([]outcome, len(decisions))
		var wg sync.WaitGroup
		sem := make(chan struct{}, par)
		for di, d := range decisions {
			wg.Add(1)
			sem <- struct{}{}
			go func(di int, d gateDecision) {
				defer wg.Done()
				defer func() { <-sem }()
				s, _, ok := deepSearcher(net, prune, deepNamedConfig(t, ply, name), d)
				if !ok {
					return
				}
				gpos, err := FromDomain(d.pos)
				if err != nil {
					return
				}
				start := time.Now()
				p, ok := s.Probs(&gpos)
				res[ci][di] = outcome{p, time.Since(start), ok}
			}(di, d)
		}
		wg.Wait()
	}
	fmt.Printf("\n%d pre-roll positions, %d at once, each serial\n", len(decisions), par)
	fmt.Printf("%-18s %10s %10s %12s %12s\n", "config", "mean s", "max s", "|dWin| mean", "|dGam| mean")
	refOf := map[int]int{}
	for ci, spec := range specs {
		ply, _ := parseDeepSpec(t, spec)
		ref, has := refOf[ply]
		if !has {
			refOf[ply], ref = ci, ci
		}
		var tot, worst time.Duration
		var dw, dg []float64
		for di := range decisions {
			o, r := res[ci][di], res[ref][di]
			if !o.ok || !r.ok {
				continue
			}
			tot += o.dt
			worst = max(worst, o.dt)
			dw = append(dw, math.Abs(float64(o.probs[PWin]-r.probs[PWin])))
			dg = append(dg, math.Abs(float64(o.probs[PWinGammon]-r.probs[PWinGammon])))
		}
		if len(dw) == 0 {
			continue
		}
		fmt.Printf("%-18s %10.3f %10.3f %12.5f %12.5f\n", spec, tot.Seconds()/float64(len(dw)), worst.Seconds(), mean(dw), mean(dg))
	}
}
