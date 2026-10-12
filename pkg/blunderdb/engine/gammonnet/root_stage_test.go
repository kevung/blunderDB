// SPDX-License-Identifier: MIT

package gammonnet

import (
	"fmt"
	"math"
	"os"
	"testing"
)

// The staged root (filtre-de-coups-spec §8), held to the same properties as
// gammonNet's tests/test_search_root_stage.py.

func TestSetRootStageRefusesAnIncoherentStage(t *testing.T) {
	for _, c := range []struct {
		ply, accept, extra int
		threshold          float64
	}{
		{-1, 0, 8, 0.16}, {MaxPly, 0, 8, 0.16}, {2, -1, 8, 0.16}, {2, 0, -1, 0.16},
		{2, 0, 8, -0.01}, {2, 0, 8, math.NaN()}, {2, 0, 8, math.Inf(1)},
	} {
		var cfg SearchConfig
		if err := cfg.SetRootStage(2, 0, 8, 0.16); err != nil {
			t.Fatal(err)
		}
		before := cfg
		if err := cfg.SetRootStage(c.ply, c.accept, c.extra, c.threshold); err == nil {
			t.Errorf("SetRootStage(%d, %d, %d, %v) accepted", c.ply, c.accept, c.extra, c.threshold)
		}
		if cfg != before {
			t.Errorf("SetRootStage(%d, %d, %d, %v) touched the config", c.ply, c.accept, c.extra, c.threshold)
		}
	}

	var cfg SearchConfig
	if err := cfg.SetRootStage(2, 0, 8, 0.16); err != nil {
		t.Fatal(err)
	}
	if err := cfg.SetRootStage(0, 3, 3, 0.5); err != nil {
		t.Fatal(err)
	}
	if cfg.RootStagePly != 0 || cfg.RootStageFilter != 0 || cfg.RootStageFilterExtra != 0 || cfg.RootStageFilterThreshold != 0 {
		t.Errorf("turning the stage off left %+v", cfg)
	}
}

// rootStageCorpus is a handful of decisions from the gold corpus.
func rootStageCorpus(t *testing.T, count int) []goldCase {
	t.Helper()
	raw, err := os.ReadFile(corpusPath)
	if err != nil {
		t.Skipf("no corpus: %v", err)
	}
	cases := decodeCorpus(t, raw)
	step := max(len(cases)/count, 1)
	var out []goldCase
	for i := 0; i < len(cases) && len(out) < count; i += step {
		out = append(out, cases[i])
	}
	return out
}

func rankingOf(t *testing.T, s *Searcher, c goldCase) []Candidate {
	t.Helper()
	out := make([]Candidate, MaxPlays)
	pos := c.pos
	n, err := s.Plays(&pos, int(c.d1), int(c.d2), out)
	if err != nil {
		t.Fatal(err)
	}
	return out[:n]
}

func sameRanking(a, b []Candidate) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Play.Result != b[i].Play.Result || math.Float64bits(a[i].Equity) != math.Float64bits(b[i].Equity) {
			return false
		}
	}
	return true
}

// A stage that is off, or not strictly shallower than the decision, changes
// nothing, bit for bit.
func TestRootStageOffOrTooDeepChangesNothing(t *testing.T) {
	plain := DefaultConfig(1)
	sp, err := NewSearcher(plain)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range rootStageCorpus(t, 6) {
		want := rankingOf(t, sp, c)
		for _, stage := range []int{0, 1, 2, 3} {
			cfg := plain
			if err := cfg.SetRootStage(stage, 0, 8, 0.16); err != nil {
				t.Fatal(err)
			}
			s, err := NewSearcher(cfg)
			if err != nil {
				t.Fatal(err)
			}
			if got := rankingOf(t, s, c); !sameRanking(got, want) {
				t.Errorf("stage %d at 1 ply moved the ranking", stage)
			}
		}
	}
}

// The staged root is the composition the spec writes, rebuilt by hand:
// 0-ply ranking, stage triplet, survivors scored at the stage's depth and
// sorted, Filter[Ply] over that ranking, survivors scored at Ply and sorted.
// Checked at 2 plies with a 1-ply stage, bit for bit.
func TestRootStageIsTheCompositionWrittenInTheSpec(t *testing.T) {
	if testing.Short() {
		t.Skip("2-ply searches")
	}
	const stageAccept, stageExtra, stageThreshold = 0, 4, 0.2
	const finalAccept, finalExtra, finalThreshold = 0, 2, 0.04

	at := func(ply int) SearchConfig {
		cfg := SearchConfig{Ply: ply, PruneK: DefaultPruneK}
		cfg.Filter[1] = 1
		return cfg
	}
	staged := at(2)
	if err := staged.SetFilter(2, finalAccept, finalExtra, finalThreshold); err != nil {
		t.Fatal(err)
	}
	if err := staged.SetRootStage(1, stageAccept, stageExtra, stageThreshold); err != nil {
		t.Fatal(err)
	}
	ss, err := NewSearcher(staged)
	if err != nil {
		t.Fatal(err)
	}
	ss = ss.WithWorkers(8)
	shallow, err := NewSearcher(at(0))
	if err != nil {
		t.Fatal(err)
	}
	deep := map[int]*Searcher{}
	for _, d := range []int{1, 2} {
		if deep[d], err = NewSearcher(at(d)); err != nil {
			t.Fatal(err)
		}
	}
	value := func(c Candidate, depth int) float64 {
		if c.Play.Result.isOver() {
			return c.Equity // exact already: never deepened
		}
		v, ok := deep[depth].positionEquity(&c.Play.Result, depth, 1, nil, CubeCentred)
		if !ok {
			t.Fatal("positionEquity refused")
		}
		return -v
	}

	differed := 0
	for _, c := range rootStageCorpus(t, 8) {
		want := rankingOf(t, shallow, c)
		n := bandSurvivors(stageAccept, stageExtra, stageThreshold, want, len(want))
		for i := 0; i < n; i++ {
			want[i].Equity = value(want[i], 1)
		}
		sortByEquity(want[:n])
		m := bandSurvivors(finalAccept, finalExtra, finalThreshold, want, n)
		for i := 0; i < m; i++ {
			want[i].Equity = value(want[i], 2)
		}
		sortByEquity(want[:m])

		if got := rankingOf(t, ss, c); !sameRanking(got, want) {
			t.Errorf("roll %d-%d: the staged root is not the written composition", c.d1, c.d2)
		}
		if n > m {
			differed++
		}
	}
	if differed == 0 {
		t.Fatal("the stage never cut: the test proves nothing")
	}
}

// The searches before the roll never read the stage.
func TestPreRollEquityIgnoresTheStage(t *testing.T) {
	if testing.Short() {
		t.Skip("2-ply searches")
	}
	plain := DefaultConfig(2)
	plain.RootStagePly = 0
	staged := plain
	if err := staged.SetRootStage(1, 0, 8, 0.16); err != nil {
		t.Fatal(err)
	}
	for _, c := range rootStageCorpus(t, 1) {
		var got [2][NumOutputs]float32
		for i, cfg := range []SearchConfig{plain, staged} {
			s, err := NewSearcher(cfg)
			if err != nil {
				t.Fatal(err)
			}
			pos := c.pos
			p, ok := s.Probs(&pos)
			if !ok {
				t.Fatal("Probs refused")
			}
			got[i] = p
		}
		if got[0] != got[1] {
			t.Errorf("the stage moved the pre-roll probabilities: %v vs %v", got[0], got[1])
		}
	}
}

// The policy's "ply3" and "ply4" levels search exactly DefaultConfig's tree,
// staged root included, and so pick the same play.
func TestPolicyDeepLevelsCarryTheStage(t *testing.T) {
	p, err := NewPolicy()
	if err != nil {
		t.Fatal(err)
	}
	c := rootStageCorpus(t, 4)[1]
	for _, ply := range []int{3, 4} {
		level := mustLevel(fmt.Sprintf("ply%d", ply))
		d := PolicyDecision{Position: c.pos, Pending: PendingMove, D1: int(c.d1), D2: int(c.d2), Cube: 1}
		if !p.configure(&level, &d) {
			t.Fatalf("ply%d refused", ply)
		}
		if p.s.cfg != DefaultConfig(ply) {
			t.Errorf("ply%d: policy config %+v, DefaultConfig %+v", ply, p.s.cfg, DefaultConfig(ply))
		}
	}
	if testing.Short() {
		return
	}
	d := PolicyDecision{Position: c.pos, Pending: PendingMove, D1: int(c.d1), D2: int(c.d2), Cube: 1}
	got, err := p.Decide("ply3", &d)
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewSearcher(DefaultConfig(3))
	if err != nil {
		t.Fatal(err)
	}
	want := rankingOf(t, s, c)[0]
	if !samePolicyPlay(&got.Play, &want.Play) {
		t.Errorf("Decide(ply3) played %v, DefaultConfig(3) %v", got.Play.Result, want.Play.Result)
	}
}
