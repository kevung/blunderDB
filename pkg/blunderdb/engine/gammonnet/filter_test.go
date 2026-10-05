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
