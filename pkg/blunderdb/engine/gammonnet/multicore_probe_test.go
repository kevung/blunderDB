// SPDX-License-Identifier: MIT

package gammonnet

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"testing"
	"time"
)

// TestProbeMulticoreLevels mesure une décision de coup du corpus de parité,
// niveau par niveau, en série puis sur tous les cœurs, caches froids — le
// pendant de bench_decision --threads de gammonNet.
//
//	BLUNDERDB_PROBE=1 BLUNDERDB_PROBE_N=12 go test -run TestProbeMulticoreLevels -v ./pkg/blunderdb/engine/gammonnet/
func TestProbeMulticoreLevels(t *testing.T) {
	if os.Getenv("BLUNDERDB_PROBE") == "" {
		t.Skip("set BLUNDERDB_PROBE to measure; this test asserts nothing")
	}
	raw, err := os.ReadFile(corpusPath)
	if err != nil {
		t.Skipf("no corpus: %v", err)
	}
	cases := decodeCorpus(t, raw)
	n := 12
	if v, err := strconv.Atoi(os.Getenv("BLUNDERDB_PROBE_N")); err == nil && v > 0 {
		n = min(v, len(cases))
	}
	cases = cases[:n]
	fmt.Printf("noyau %s, GOMAXPROCS=%d, %d décisions\n", KernelName(), runtime.GOMAXPROCS(0), n)
	plies := []int{2, 3}
	if v, err := strconv.Atoi(os.Getenv("BLUNDERDB_PROBE_PLY")); err == nil {
		plies = []int{v}
	}
	for _, ply := range plies {
		var serial time.Duration
		for _, nw := range []int{1, runtime.GOMAXPROCS(0)} {
			s, err := NewSearcher(DefaultConfig(ply))
			if err != nil {
				t.Fatal(err)
			}
			s = s.WithWorkers(nw)
			start := time.Now()
			for _, c := range cases {
				coldCaches(s)
				pos := c.pos
				if _, _, err := s.BestPlay(&pos, int(c.d1), int(c.d2)); err != nil {
					t.Fatal(err)
				}
			}
			d := time.Since(start)
			if nw == 1 {
				serial = d
			}
			evals, _, hits := s.Counters()
			fmt.Printf("ply %d  %2d fils  %8s/décision  ×%.2f  %d évals, %d hits\n", ply, nw,
				(d / time.Duration(n)).Round(time.Millisecond), float64(serial)/float64(d), evals, hits)
		}
	}
}
