// SPDX-License-Identifier: MIT

package gammonnet

import (
	"os"
	"testing"
)

// BenchmarkDecision2PlyCorpus prices one canonical 2-ply "normal" decision
// over the first 2-ply cases of the gold corpus, serially, each on a cold
// searcher: a warm cache would hide what the network costs. It is the
// measure a change of network is judged by (its width is paid at every leaf).
func BenchmarkDecision2PlyCorpus(b *testing.B) {
	benchDecision2PlyCorpus(b, nil)
}

// BenchmarkDecision2PlyCorpusAlternate is the same over AlternateNetworkName,
// the price of the four-layer network beside the default one.
func BenchmarkDecision2PlyCorpusAlternate(b *testing.B) {
	net, err := AlternateNetwork()
	if err != nil {
		b.Fatal(err)
	}
	benchDecision2PlyCorpus(b, net)
}

// benchDecision2PlyCorpus runs the corpus over net, the default network when nil.
func benchDecision2PlyCorpus(b *testing.B, net *Network) {
	if testing.Short() {
		b.Skip("a 2-ply decision costs a fraction of a second")
	}
	raw, err := os.ReadFile(corpusPath)
	if err != nil {
		b.Fatal(err)
	}
	var cases []goldCase
	for _, c := range decodeCorpus(b, raw) {
		if c.ply == 2 && len(cases) < 6 {
			cases = append(cases, c)
		}
	}
	if len(cases) == 0 {
		b.Fatal("no 2-ply case in the corpus")
	}
	out := make([]Candidate, MaxPlays)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, c := range cases {
			s, err := NewSearcher(c.config())
			if err != nil {
				b.Fatal(err)
			}
			if net != nil {
				s = newSearcherWith(c.config(), net, s.prune)
			}
			pos := c.pos
			if _, err := s.Plays(&pos, int(c.d1), int(c.d2), out); err != nil {
				b.Fatal(err)
			}
		}
	}
	b.ReportMetric(float64(b.Elapsed().Seconds())/float64(b.N*len(cases)), "s/decision")
}
