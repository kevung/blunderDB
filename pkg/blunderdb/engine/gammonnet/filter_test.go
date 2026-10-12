// SPDX-License-Identifier: MIT

package gammonnet

import (
	"math"
	"testing"
)

func rankedEquities(eqs ...float64) []Candidate {
	out := make([]Candidate, len(eqs))
	for i, e := range eqs {
		out[i].Equity = e
	}
	return out
}

// The triplet's three regimes, as gn_search.c's filter_survivors: off, a
// plain count (threshold never read), and the band walk that stops at the
// first candidate outside the threshold.
func TestFilterSurvivors(t *testing.T) {
	ranked := rankedEquities(0.50, 0.48, 0.47, 0.40, 0.39)
	cases := []struct {
		name          string
		accept, extra int
		threshold     float64
		n, want       int
	}{
		{"off", 0, 0, 0, 5, 5},
		{"plain count", 3, 0, 0, 5, 3},
		{"plain count ignores a NaN threshold", 2, 0, math.NaN(), 5, 2},
		{"count above n", 9, 0, 0, 5, 5},
		{"band takes both within", 1, 2, 0.04, 5, 3},
		{"band stops at the first outside", 1, 4, 0.04, 5, 3},
		{"band bounded by extra", 1, 1, 0.5, 5, 2},
		{"band bounded by n", 1, 9, 1, 5, 5},
		{"band edge is inclusive", 1, 2, 0.02, 5, 2},
		{"extra alone", 0, 2, 0.025, 5, 2},
	}
	for _, c := range cases {
		var cfg SearchConfig
		cfg.Filter[2], cfg.FilterExtra[2], cfg.FilterThreshold[2] = c.accept, c.extra, c.threshold
		if got := cfg.filterSurvivors(2, ranked, c.n); got != c.want {
			t.Errorf("%s: filterSurvivors = %d, want %d", c.name, got, c.want)
		}
	}
}

// Pruning is never narrower than the widest the filter can deepen.
func TestFilterWidestRaisesPruneKeep(t *testing.T) {
	cfg := DefaultConfig(2)
	cfg.PruneK = 2
	s := &Searcher{cfg: cfg, prune: &Network{}}
	want := cfg.Filter[2] + cfg.FilterExtra[2]
	if got := s.pruneKeep(2); got != max(2, want) {
		t.Errorf("pruneKeep(2) = %d, want %d", got, max(2, want))
	}
}

func TestSetFilterRefusesAnIncoherentTriplet(t *testing.T) {
	bad := []struct {
		depth, accept, extra int
		threshold            float64
	}{
		{-1, 1, 0, 0}, {MaxPly + 1, 1, 0, 0}, {1, -1, 0, 0}, {1, 1, -1, 0},
		{1, 1, 1, -0.1}, {1, 1, 1, math.NaN()}, {1, 1, 1, math.Inf(1)},
	}
	for _, b := range bad {
		cfg := DefaultConfig(2)
		before := cfg
		if err := cfg.SetFilter(b.depth, b.accept, b.extra, b.threshold); err == nil {
			t.Errorf("SetFilter(%d, %d, %d, %v) accepted", b.depth, b.accept, b.extra, b.threshold)
		}
		if cfg.Filter != before.Filter || cfg.FilterExtra != before.FilterExtra {
			t.Errorf("SetFilter(%d, %d, %d, %v) touched the config on refusal", b.depth, b.accept, b.extra, b.threshold)
		}
	}
	cfg := DefaultConfig(2)
	if err := cfg.SetFilter(2, 1, 2, 0.04); err != nil {
		t.Fatal(err)
	}
	if cfg.Filter[2] != 1 || cfg.FilterExtra[2] != 2 || cfg.FilterThreshold[2] != 0.04 {
		t.Errorf("SetFilter did not set the triplet: %d %d %v", cfg.Filter[2], cfg.FilterExtra[2], cfg.FilterThreshold[2])
	}
}

// TestDefaultConfigDeepFilter pins the 3- and 4-ply vector to the published
// "ply3"/"ply4" levels: one reply at every interior depth, the root
// (0, 2, 0.04) over a stage (2, 0, 8, 0.16). 0 to 2 ply keep the "normal"
// filter and no stage.
func TestDefaultConfigDeepFilter(t *testing.T) {
	triplet := func(c SearchConfig, d int) [3]float64 {
		return [3]float64{float64(c.Filter[d]), float64(c.FilterExtra[d]), c.FilterThreshold[d]}
	}
	pub := func(d int) [3]float64 {
		return [3]float64{float64(defaultLevel.Filter[d]), float64(defaultLevel.FilterExtra[d]), defaultLevel.FilterThreshold[d]}
	}
	one := [3]float64{1, 0, 0}
	for ply := 0; ply <= MaxPly; ply++ {
		cfg := DefaultConfig(ply)
		if got := triplet(cfg, 1); got != one {
			t.Errorf("ply %d: depth 1 = %v, want %v", ply, got, one)
		}
		if ply <= 2 {
			if got := triplet(cfg, 2); got != pub(2) {
				t.Errorf("ply %d: depth 2 = %v, want the published %v", ply, got, pub(2))
			}
			if cfg.RootStagePly != 0 || cfg.PruneK != DefaultPruneK {
				t.Errorf("ply %d: stage %d, PruneK %d", ply, cfg.RootStagePly, cfg.PruneK)
			}
			continue
		}
		if got := triplet(cfg, ply); got != [3]float64{0, 2, 0.04} {
			t.Errorf("ply %d: root = %v, want (0, 2, 0.04)", ply, got)
		}
		for d := 2; d < ply; d++ {
			if got := triplet(cfg, d); got != one {
				t.Errorf("ply %d: interior depth %d = %v, want %v", ply, d, got, one)
			}
		}
		stage := [4]float64{float64(cfg.RootStagePly), float64(cfg.RootStageFilter), float64(cfg.RootStageFilterExtra), cfg.RootStageFilterThreshold}
		if stage != [4]float64{2, 0, 8, 0.16} || cfg.PruneK != 12 {
			t.Errorf("ply %d: stage %v, PruneK %d, want (2, 0, 8, 0.16) and 12", ply, stage, cfg.PruneK)
		}
	}
}
