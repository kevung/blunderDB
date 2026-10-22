package gammonnet

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"testing"
)

// evaluationDigest pins, bit for bit, what the engine answers on the two gold
// corpora: every candidate's equity and network distribution at up to 1 ply
// (money and match, cubeless and cubeful), and every field of every cube
// decision. The golds compare against the C reference within a tolerance;
// this digest is the exact answer of amd64 at the default GOAMD64 level, and
// every other platform must reproduce it (engine/gammonnet/CLAUDE.md, bit
// parity). An intentional change of the engine's arithmetic re-pins it, from
// amd64 at the default level, with the digest the failure prints.
const evaluationDigest = "3186a4ee377dec0e8702850f0e6064079acbe592dbdf1730a67d3ef4151459e8"

func TestEvaluationBitsMatchThePinnedDigest(t *testing.T) {
	raw, err := os.ReadFile(corpusPath)
	if err != nil {
		t.Skipf("no corpus: %v", err)
	}
	cubeRaw, err := os.ReadFile(cubeCorpusPath)
	if err != nil {
		t.Skipf("no cube corpus: %v", err)
	}

	h := sha256.New()
	var buf [8]byte
	put64 := func(v uint64) {
		binary.LittleEndian.PutUint64(buf[:], v)
		h.Write(buf[:])
	}

	searchers := map[SearchConfig]*Searcher{}
	out := make([]Candidate, MaxPlays)
	for i, c := range decodeCorpus(t, raw) {
		if c.ply > 1 {
			c.ply = 1
		}
		cfg := c.config()
		s, ok := searchers[cfg]
		if !ok {
			if s, err = NewSearcher(cfg); err != nil {
				t.Fatal(err)
			}
			searchers[cfg] = s
		}
		pos := c.pos
		n, err := s.Plays(&pos, int(c.d1), int(c.d2), out)
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		put64(uint64(n))
		for _, cand := range out[:n] {
			put64(math.Float64bits(cand.Equity))
			for _, p := range cand.Probs {
				put64(uint64(math.Float32bits(p)))
			}
		}
	}

	for _, c := range decodeCubeCorpus(t, cubeRaw) {
		var state *MatchState
		if c.hasState {
			state = &MatchState{
				AwayOnRoll:   int(c.awayOnRoll),
				AwayOpponent: int(c.awayOpponent),
				Cube:         int(c.cube),
				Crawford:     c.crawford,
			}
		}
		// %v prints each float as the shortest decimal that reads back to
		// the same bits, so the text is as exact as the bits.
		dec, ok := DecideEx(&c.probs, c.owner, state, c.efficiency, c.jacoby, true)
		fmt.Fprintf(h, "%v %+v\n", ok, dec)
	}

	got := hex.EncodeToString(h.Sum(nil))
	if got != evaluationDigest {
		t.Errorf("evaluation digest %s, pinned %s: the engine's answer changed in its bits", got, evaluationDigest)
	}
}
