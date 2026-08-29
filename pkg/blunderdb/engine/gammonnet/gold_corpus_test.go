// SPDX-License-Identifier: MIT

package gammonnet

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"os"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// The gold corpus: a fixed list of (position, dice, ply) decisions that both
// this port and the C reference must answer identically.
//
// The file is versioned so the two sides provably start from the same
// positions. Regenerating it is a deliberate act — run this test with
// -run TestWriteGoldCorpus and BLUNDERDB_WRITE_CORPUS set — because a corpus
// that drifts silently would turn the gold file into a comparison of two
// different questions.
const (
	corpusMagic = "GNCP"
	corpusPath  = "testdata/search_corpus.bin"
)

type goldCase struct {
	pos    Position
	d1, d2 int8
	ply    int8
}

// The mix is chosen by cost. A 2-ply decision costs seconds; 0-ply costs
// milliseconds. Breadth comes from the cheap plies, depth from a handful.
func buildCorpus() []goldCase {
	rng := rand.New(rand.NewSource(20260828))
	var cases []goldCase

	// The opening, every distinct roll, at 0-ply and the four canonical ones at 2-ply.
	dp, _ := domain.DecodeXGID(openingXGID)
	dp.PlayerOnRoll = domain.White
	opening, _ := FromDomain(&dp)
	for d1 := int8(1); d1 <= 6; d1++ {
		for d2 := d1; d2 <= 6; d2++ {
			cases = append(cases, goldCase{pos: opening, d1: d1, d2: d2, ply: 0})
		}
	}
	for _, r := range [][2]int8{{3, 1}, {6, 1}, {4, 2}, {6, 5}} {
		cases = append(cases, goldCase{pos: opening, d1: r[0], d2: r[1], ply: 2})
	}

	// Random boards: broad coverage of bar, bear-off and dance branches.
	added := 0
	for added < 60 {
		onRoll := domain.White
		if added%2 == 1 {
			onRoll = domain.Black
		}
		b := randomBoard(rng, onRoll)
		p, err := FromDomain(&b)
		if err != nil || p.isOver() {
			continue
		}
		d1 := int8(1 + rng.Intn(6))
		d2 := int8(1 + rng.Intn(6))
		if d2 < d1 {
			d1, d2 = d2, d1
		}
		ply := int8(0)
		switch {
		case added%10 == 0:
			ply = 2
		case added%3 == 0:
			ply = 1
		}
		cases = append(cases, goldCase{pos: p, d1: d1, d2: d2, ply: ply})
		added++
	}
	return cases
}

func encodeCorpus(cases []goldCase) []byte {
	var buf bytes.Buffer
	buf.WriteString(corpusMagic)
	_ = binary.Write(&buf, binary.LittleEndian, int32(len(cases)))
	for _, c := range cases {
		for _, v := range c.pos.Points {
			buf.WriteByte(byte(v))
		}
		buf.Write([]byte{c.pos.Bar[0], c.pos.Bar[1], c.pos.Off[0], c.pos.Off[1], c.pos.Turn})
		buf.Write([]byte{byte(c.d1), byte(c.d2), byte(c.ply)})
	}
	return buf.Bytes()
}

func decodeCorpus(t *testing.T, raw []byte) []goldCase {
	t.Helper()
	if len(raw) < 8 || string(raw[:4]) != corpusMagic {
		t.Fatalf("%s: unexpected magic", corpusPath)
	}
	n := int(int32(binary.LittleEndian.Uint32(raw[4:])))
	const entry = 29 + 3
	if len(raw) != 8+n*entry {
		t.Fatalf("%s: %d bytes for %d cases", corpusPath, len(raw), n)
	}
	out := make([]goldCase, n)
	for i := range out {
		b := raw[8+i*entry:]
		for j := 0; j < NumPoints; j++ {
			out[i].pos.Points[j] = int8(b[j])
		}
		out[i].pos.Bar = [2]uint8{b[24], b[25]}
		out[i].pos.Off = [2]uint8{b[26], b[27]}
		out[i].pos.Turn = b[28]
		out[i].d1, out[i].d2, out[i].ply = int8(b[29]), int8(b[30]), int8(b[31])
	}
	return out
}

// TestWriteGoldCorpus regenerates the corpus. Deliberate, never automatic.
func TestWriteGoldCorpus(t *testing.T) {
	if os.Getenv("BLUNDERDB_WRITE_CORPUS") == "" {
		t.Skip("set BLUNDERDB_WRITE_CORPUS to regenerate")
	}
	cases := buildCorpus()
	if err := os.WriteFile(corpusPath, encodeCorpus(cases), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %d cases to %s", len(cases), corpusPath)
}

// The corpus on disk must be exactly what buildCorpus produces. If it is not,
// the gold file was computed for different questions than the ones being asked.
func TestGoldCorpusIsInSync(t *testing.T) {
	raw, err := os.ReadFile(corpusPath)
	if err != nil {
		t.Skipf("no corpus yet: %v", err)
	}
	if !bytes.Equal(raw, encodeCorpus(buildCorpus())) {
		t.Fatal("testdata/search_corpus.bin does not match buildCorpus(); the gold file answers different questions")
	}
	cases := decodeCorpus(t, raw)
	var byPly [3]int
	for _, c := range cases {
		if c.ply < 3 {
			byPly[c.ply]++
		}
		if !c.pos.Valid() {
			t.Fatal("corpus holds an invalid position")
		}
	}
	t.Logf("%d cases: %d at 0-ply, %d at 1-ply, %d at 2-ply", len(cases), byPly[0], byPly[1], byPly[2])
}
