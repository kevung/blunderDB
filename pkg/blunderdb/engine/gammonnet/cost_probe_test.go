// SPDX-License-Identifier: MIT

package gammonnet

import (
	"fmt"
	"os"
	"runtime"
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
		label   string
		cfg     SearchConfig
		workers int
	}
	rows := []row{
		{"0-ply, k=12", SearchConfig{Ply: 0, PruneK: 12}, 1},
		{"1-ply, k=12, no filter", SearchConfig{Ply: 1, PruneK: 12}, 1},
		{"1-ply, k=12, filter[1]=3", SearchConfig{Ply: 1, PruneK: 12, Filter: [MaxPly + 1]int{0, 3}}, 1},
		{"2-ply, k=12, filter (0,1,3)", SearchConfig{Ply: 2, PruneK: 12, Filter: [MaxPly + 1]int{0, 1, 3}}, 1},
		{"2-ply, k=12, filter (0,2,8)", SearchConfig{Ply: 2, PruneK: 12, Filter: [MaxPly + 1]int{0, 2, 8}}, 1},
		{"2-ply, k=12, (0,1,3), NumCPU", SearchConfig{Ply: 2, PruneK: 12, Filter: [MaxPly + 1]int{0, 1, 3}}, runtime.NumCPU()},
		// 3-ply (#272/I.16). MaxPly has been 4 since the port landed and
		// DefaultConfig has always had a filter for it; what was missing was
		// the number that says whether it is offerable.
		{"3-ply, canonique", DefaultConfig(3), 1},
		{"3-ply, canonique, NumCPU", DefaultConfig(3), runtime.NumCPU()},
	}

	fmt.Printf("noyau %s, largeur de lot %d, %d cœurs logiques\n\n",
		KernelName(), EvalBatchWidth, runtime.NumCPU())
	fmt.Printf("%-32s %10s %12s %12s %12s %10s\n",
		"configuration", "durée", "évals", "élagage", "cache", "remplis.")
	for _, r := range rows {
		s, err := NewSearcher(r.cfg)
		if err != nil {
			t.Fatal(err)
		}
		if r.workers > 1 {
			s = s.WithWorkers(r.workers)
		}
		pos := p
		start := time.Now()
		_, ok, err := s.BestPlay(&pos, 3, 1)
		d := time.Since(start)
		if err != nil || !ok {
			t.Fatalf("%s: %v", r.label, err)
		}
		e, pe, ch := s.Counters()
		filled, slotted := s.BatchFill()
		fill := "—"
		if slotted > 0 {
			fill = fmt.Sprintf("%.1f%%", 100*float64(filled)/float64(slotted))
		}
		fmt.Printf("%-32s %10s %12d %12d %12d %10s\n",
			r.label, d.Round(time.Millisecond), e, pe, ch, fill)
	}
}
