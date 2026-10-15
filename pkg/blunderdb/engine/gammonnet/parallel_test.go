// SPDX-License-Identifier: MIT

package gammonnet

import (
	"fmt"
	"math"
	"os"
	"runtime"
	"sync"
	"testing"
)

// fingerprint is every ranking a searcher returns over cases, in the order
// returned: each candidate's result and its equity's bits. Two searches that
// differ anywhere, by one bit of one equity, differ here — the proof of
// gammonNet's spec §5 (bench_decision's « empreinte des classements »).
type fingerprint []string

func rankingFingerprint(t *testing.T, s *Searcher, cases []goldCase) fingerprint {
	t.Helper()
	out := make([]Candidate, MaxPlays)
	var fp fingerprint
	for i, c := range cases {
		pos := c.pos
		n, err := s.Plays(&pos, int(c.d1), int(c.d2), out)
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		for j := 0; j < n; j++ {
			fp = append(fp, fmt.Sprintf("%d/%d %v %x", i, j, out[j].Play.Result, math.Float64bits(out[j].Equity)))
		}
	}
	return fp
}

func probsFingerprint(t *testing.T, s *Searcher, cases []goldCase) fingerprint {
	t.Helper()
	var fp fingerprint
	for i, c := range cases {
		pos := c.pos
		probs, ok := s.Probs(&pos)
		if !ok {
			t.Fatalf("case %d: Probs refused", i)
		}
		for k, p := range probs {
			fp = append(fp, fmt.Sprintf("%d/%d %x", i, k, math.Float32bits(p)))
		}
	}
	return fp
}

func sameFingerprint(t *testing.T, serial, par fingerprint) {
	t.Helper()
	if len(serial) != len(par) {
		t.Fatalf("%d entries serial, %d parallel", len(serial), len(par))
	}
	for i := range serial {
		if serial[i] != par[i] {
			t.Fatalf("entry %d: serial %s, parallel %s — not bit-identical", i, serial[i], par[i])
		}
	}
}

// levelConfig is base searched as level: base keeps its match, cube and
// owner, the level sets depth, pruning, filter and root stage.
func levelConfig(level SearchLevel, base SearchConfig) SearchConfig {
	cfg := base
	cfg.Ply, cfg.PruneK = level.Ply, level.PruneK
	cfg.Filter, cfg.FilterExtra, cfg.FilterThreshold = [MaxPly + 1]int{}, [MaxPly + 1]int{}, [MaxPly + 1]float64{}
	copy(cfg.Filter[:], level.Filter)
	copy(cfg.FilterExtra[:], level.FilterExtra)
	copy(cfg.FilterThreshold[:], level.FilterThreshold)
	cfg.RootStagePly = level.RootStagePly
	cfg.RootStageFilter = level.RootStageFilter
	cfg.RootStageFilterExtra = level.RootStageFilterExtra
	cfg.RootStageFilterThreshold = level.RootStageFilterThreshold
	return cfg
}

// The cube corpus's cases that the race detector searches at 3 ply: late
// positions whose search still lends helpers work at every depth, for about
// 18 000 and 20 000 evaluations against 100 000 to 160 000 for an opening.
const (
	raceDecisionCase = 100
	raceProbsCase    = 92
)

// TestParallelSearchFingerprint holds the number of cores to changing
// nothing a search returns, on every path that forks: the root queue
// (normal, thorough), the nested deep passes and chance nodes (ply3, staged
// root), the distribution backups (Probs at 3 ply), and match play with the
// cube. A private budget of 6 forces a mix of helpers and caller-run tasks;
// NumCPU the widest spread. BLUNDERDB_PARALLEL widens every corpus.
//
// Under the race detector, which slows a forking search several times more
// than a serial one, each path searches one case at the budget of 6 only,
// and the 3-ply paths a late position of the cube corpus that still forks at
// every depth at a tenth of an opening's evaluations: the race job looks for
// unsynchronised writes, the full proof runs without the detector.
func TestParallelSearchFingerprint(t *testing.T) {
	if testing.Short() {
		t.Skip("3-ply searches, serial and parallel")
	}
	raw, err := os.ReadFile(corpusPath)
	if err != nil {
		t.Skipf("no corpus: %v", err)
	}
	cases := decodeCorpus(t, raw)
	raw2, err := os.ReadFile(searchCubeCorpusPath)
	if err != nil {
		t.Skipf("no cube corpus: %v", err)
	}
	cubeCases := decodeSearchCubeCorpus(t, raw2)
	wide := os.Getenv("BLUNDERDB_PARALLEL") != ""
	take := func(c []goldCase, n, all int) []goldCase {
		switch {
		case wide:
			n = all
		case raceEnabled:
			n = 1
		}
		return c[:min(n, len(c))]
	}
	// light is c, or under the race detector the cube corpus's case k.
	light := func(c []goldCase, k int) []goldCase {
		if raceEnabled && !wide && k < len(cubeCases) {
			return cubeCases[k : k+1]
		}
		return c
	}

	type path struct {
		name  string
		cases []goldCase
		cfg   func(goldCase) SearchConfig
		probs bool
		// nested: the path must fork from inside a forked task.
		nested bool
	}
	paths := []path{
		{"normal", take(cases, 6, 40), func(goldCase) SearchConfig { return DefaultConfig(2) }, false, false},
		{"thorough", take(cases, 2, 10), func(goldCase) SearchConfig { return levelConfig(mustLevel("thorough"), SearchConfig{}) }, false, false},
		{"ply3", light(take(cases, 2, 10), raceDecisionCase), func(goldCase) SearchConfig { return DefaultConfig(3) }, false, true},
		{"probs-ply3", light(take(cases, 1, 4), raceProbsCase), func(goldCase) SearchConfig { return DefaultConfig(3) }, true, true},
		{"match-cube", take(cubeCases, 6, len(cubeCases)), func(c goldCase) SearchConfig { return c.config() }, false, false},
		{"match-cube-ply3", light(take(cubeCases, 1, 6), raceDecisionCase), func(c goldCase) SearchConfig {
			return levelConfig(mustLevel("ply3"), c.config())
		}, false, true},
	}

	for _, p := range paths {
		t.Run(p.name, func(t *testing.T) {
			run := func(workers int) (fingerprint, uint64) {
				var fp fingerprint
				var nested uint64
				for _, c := range p.cases {
					s, err := NewSearcher(p.cfg(c))
					if err != nil {
						t.Fatal(err)
					}
					s = s.WithWorkers(workers)
					one := []goldCase{c}
					if p.probs {
						fp = append(fp, probsFingerprint(t, s, one)...)
					} else {
						fp = append(fp, rankingFingerprint(t, s, one)...)
					}
					nested += s.nestedForks
				}
				return fp, nested
			}
			serial, _ := run(1)
			widths := []int{6, runtime.NumCPU()}
			if raceEnabled {
				widths = widths[:1]
			}
			var firstNested uint64
			for i, nw := range widths {
				par, nested := run(nw)
				sameFingerprint(t, serial, par)
				if i == 0 {
					firstNested = nested
				} else if nested != firstNested {
					t.Errorf("%d workers: %d nested forks, %d at %d workers", nw, nested, firstNested, widths[0])
				}
				// Without a nested fork the case proves nothing about the
				// forks inside forks; a lighter case must not drop them.
				if p.nested && nested == 0 {
					t.Errorf("%d workers: no fork opened inside a forked task", nw)
				}
			}
		})
	}
}

// TestSharedBudgetSearchesAgree runs several batch searchers at once on the
// process budget, as a short batch does — each holding a token, forking into
// the rest — and holds every ranking to the serial one.
func TestSharedBudgetSearchesAgree(t *testing.T) {
	if testing.Short() {
		t.Skip("2-ply searches, serial and concurrent")
	}
	raw, err := os.ReadFile(corpusPath)
	if err != nil {
		t.Skipf("no corpus: %v", err)
	}
	cases := decodeCorpus(t, raw)[:4]

	want := make([]fingerprint, len(cases))
	for i, c := range cases {
		s, err := NewBatchSearcher(2, DefaultPruneK)
		if err != nil {
			t.Fatal(err)
		}
		want[i] = rankingFingerprint(t, s.WithCores(nil), []goldCase{c})
	}

	got := make([]fingerprint, len(cases))
	var wg sync.WaitGroup
	for i, c := range cases {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s, err := NewBatchSearcher(2, DefaultPruneK)
			if err != nil {
				t.Error(err)
				return
			}
			release := HoldCore()
			defer release()
			got[i] = rankingFingerprint(t, s, []goldCase{c})
		}()
	}
	wg.Wait()
	for i := range cases {
		sameFingerprint(t, want[i], got[i])
	}
}
