// SPDX-License-Identifier: MIT

package gammonnet

import (
	"fmt"
	"os"
	"testing"
	"time"
)

// TestProbeDecisionCost is a measurement, not an assertion. It reports what a
// decision costs at each depth and filter so the numbers that size #125 and
// #129 come from this build rather than from a citation. Run it on an idle
// machine: under load it measures the machine.
func TestProbeDecisionCost(t *testing.T) {
	// A measurement that cannot fail has no business lengthening the recipe.
	// testing.Short() is not enough: the full suite runs without it, and this
	// probe costs minutes there while asserting nothing. Behind an environment
	// variable, like the corpus regeneration, it runs when it is wanted.
	if os.Getenv("BLUNDERDB_PROBE") == "" {
		t.Skip("set BLUNDERDB_PROBE to measure; this test asserts nothing")
	}
	p := openingPosition(t)

	type row struct {
		label string
		cfg   SearchConfig
	}
	rows := []row{
		{"0-ply, k=12", SearchConfig{Ply: 0, PruneK: 12}},
		{"1-ply, k=12, no filter", SearchConfig{Ply: 1, PruneK: 12}},
		{"1-ply, k=12, filter[1]=3", SearchConfig{Ply: 1, PruneK: 12, Filter: [MaxPly + 1]int{0, 3}}},
		{"2-ply, k=12, filter (0,1,3)", SearchConfig{Ply: 2, PruneK: 12, Filter: [MaxPly + 1]int{0, 1, 3}}},
		{"2-ply, k=12, filter (0,2,8)", SearchConfig{Ply: 2, PruneK: 12, Filter: [MaxPly + 1]int{0, 2, 8}}},
	}

	fmt.Printf("%-30s %10s %12s %12s %12s\n", "configuration", "durée", "évals", "élagage", "cache")
	for _, r := range rows {
		s, err := NewSearcher(r.cfg)
		if err != nil {
			t.Fatal(err)
		}
		pos := p
		start := time.Now()
		_, ok, err := s.BestPlay(&pos, 3, 1)
		d := time.Since(start)
		if err != nil || !ok {
			t.Fatalf("%s: %v", r.label, err)
		}
		e, pe, ch := s.Counters()
		fmt.Printf("%-30s %10s %12d %12d %12d\n", r.label, d.Round(time.Millisecond), e, pe, ch)
	}
}
