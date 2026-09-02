// SPDX-License-Identifier: MIT

package gammonnet

import (
	"encoding/binary"
	"math"
	"os"
	"testing"
)

// BLUNDERDB_GOLD gates TestSearchMatchesTheGoldFile out of the default `go
// test ./...` run. `testing.Short()` alone does not do it: CI's test job
// (.github/workflows/build.yml) never passes `-short`, and this test pays the
// race detector's cost on a search that is already parallel — measured
// 2026-08-29 on a 16-core machine: 10.3s plain, still running past 300s under
// `-race` (the CI job's own flag), a >30x blow-up that alone exceeds the whole
// suite's 1200s CI budget. Same treatment as TestIntegrationGate
// (integration_gate_test.go, BLUNDERDB_GATE) and TestEvalMeasure
// (eval_measure_test.go, BLUNDERDB_EVAL_MEASURE): a pre-merge recipe step, not
// a CI test.

// The gold file: what gammonNet's C reference answers for every decision in the
// corpus, at the canonical configuration — prune k=12, filter (0,1,3).
//
// It is produced by the C, read here verbatim, and regenerated only on a
// deliberate upstream bump. Recomputing both sides from a seed would establish
// nothing if the two implementations had drifted apart.
const (
	goldMagic = "GNGD"
	goldPath  = "testdata/search_gold.bin"

	// goldTolerance is the equity agreement required. It is also the tie zone:
	// two plays whose equities agree to within it are indistinguishable AT THIS
	// TOLERANCE, so demanding that both engines pick the same one would be
	// demanding more than the tolerance states.
	//
	// That is not a convenience. The reference orders its candidates with
	// qsort, which is not stable and whose implementation has changed across
	// libc versions: on an exact tie, the play it returns depends on the C
	// library that produced this file. A gold file must not encode that.
	goldTolerance = 1e-6
)

type goldEntry struct {
	total int
	best  []Position
	eq    []float64
}

func loadGold(t *testing.T, goldPath string) []goldEntry {
	t.Helper()
	raw, err := os.ReadFile(goldPath)
	if err != nil {
		t.Skipf("no gold file: %v", err)
	}
	if len(raw) < 8 || string(raw[:4]) != goldMagic {
		t.Fatalf("%s: unexpected magic", goldPath)
	}
	n := int(int32(binary.LittleEndian.Uint32(raw[4:])))
	out := make([]goldEntry, n)
	off := 8
	for i := range out {
		out[i].total = int(int32(binary.LittleEndian.Uint32(raw[off:])))
		k := int(int32(binary.LittleEndian.Uint32(raw[off+4:])))
		off += 8
		for c := 0; c < k; c++ {
			var p Position
			for j := 0; j < NumPoints; j++ {
				p.Points[j] = int8(raw[off+j])
			}
			p.Bar = [2]uint8{raw[off+24], raw[off+25]}
			p.Off = [2]uint8{raw[off+26], raw[off+27]}
			p.Turn = raw[off+28]
			off += 29
			e := math.Float64frombits(binary.LittleEndian.Uint64(raw[off:]))
			off += 8
			out[i].best = append(out[i].best, p)
			out[i].eq = append(out[i].eq, e)
		}
	}
	if off != len(raw) {
		t.Fatalf("%s: %d trailing bytes", goldPath, len(raw)-off)
	}
	return out
}

// The search parity gate of #121: the same chosen move as the C reference, with
// equities agreeing to 1e-6.
func TestSearchMatchesTheGoldFile(t *testing.T) {
	if os.Getenv("BLUNDERDB_GOLD") == "" {
		t.Skip("set BLUNDERDB_GOLD to run the search-parity gold file (85 2-ply decisions; slow, and very slow under -race)")
	}
	raw, err := os.ReadFile(corpusPath)
	if err != nil {
		t.Skipf("no corpus: %v", err)
	}
	t.Run("money-cubeless", func(t *testing.T) { replayGold(t, decodeCorpus(t, raw), loadGold(t, goldPath)) })

	// The ADR-0023 corpus: match states and cube states, one searcher per
	// case since the configuration is part of the question.
	raw2, err := os.ReadFile(searchCubeCorpusPath)
	if err != nil {
		t.Skipf("no cube corpus: %v", err)
	}
	t.Run("match-and-cube", func(t *testing.T) { replayGold(t, decodeSearchCubeCorpus(t, raw2), loadGold(t, searchCubeGoldPath)) })
}

func replayGold(t *testing.T, cases []goldCase, gold []goldEntry) {
	t.Helper()
	if len(cases) != len(gold) {
		t.Fatalf("%d cases, %d gold entries — they are not the same question", len(cases), len(gold))
	}

	// One searcher per distinct configuration, reused: building one
	// allocates megabytes. Money cubeless collapses to one per ply.
	searchers := map[SearchConfig]*Searcher{}
	searcherFor := func(c goldCase) *Searcher {
		cfg := c.config()
		if s, ok := searchers[cfg]; ok {
			return s
		}
		s, err := NewSearcher(cfg)
		if err != nil {
			t.Fatal(err)
		}
		s = s.WithWorkers(8)
		searchers[cfg] = s
		return s
	}

	out := make([]Candidate, MaxPlays)
	var worst float64
	var ties int

	for i, c := range cases {
		g := gold[i]
		pos := c.pos
		n, err := searcherFor(c).Plays(&pos, int(c.d1), int(c.d2), out)
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		if n != g.total {
			t.Errorf("case %d (%d-ply, roll %d-%d): %d candidates, gold has %d",
				i, c.ply, c.d1, c.d2, n, g.total)
			continue
		}
		if n == 0 {
			continue
		}

		// The chosen move.
		if out[0].Play.Result != g.best[0] {
			d := math.Abs(out[0].Equity - g.eq[0])
			if d <= goldTolerance {
				ties++ // indistinguishable at the stated tolerance
			} else {
				t.Errorf("case %d (%d-ply, roll %d-%d): different best move, and not a tie — ours %.9f, gold %.9f (Δ %.2e)",
					i, c.ply, c.d1, c.d2, out[0].Equity, g.eq[0], d)
				continue
			}
		}

		// Every gold candidate we can find must carry the same equity.
		for c2 := range g.best {
			for j := 0; j < n; j++ {
				if out[j].Play.Result != g.best[c2] {
					continue
				}
				d := math.Abs(out[j].Equity - g.eq[c2])
				if d > worst {
					worst = d
				}
				if d > goldTolerance {
					t.Errorf("case %d (%d-ply): candidate %d equity %.9f, gold %.9f (Δ %.2e)",
						i, c.ply, c2, out[j].Equity, g.eq[c2], d)
				}
				break
			}
		}
	}
	t.Logf("%d decisions replayed against the C reference — max|Δ equity| = %.3e, %d ties within tolerance",
		len(cases), worst, ties)
}
