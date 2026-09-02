// SPDX-License-Identifier: MIT

package gammonnet

import (
	"bytes"
	"encoding/binary"
	"math"
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

	// The second corpus (ADR-0023): the same kind of decisions, each with a
	// full search configuration — match state and cube — so the gold covers
	// use_match and use_cube, which the first corpus (money, cubeless, 32
	// bytes a case) cannot express. Record = the 32 bytes above, then
	// use_match, away_on_roll, away_opponent, cube, crawford, use_cube,
	// owner, one pad byte, then x as a little-endian float64: 48 bytes.
	searchCubeCorpusMagic = "GNC2"
	searchCubeCorpusPath  = "testdata/search_cube_corpus.bin"
	searchCubeGoldPath    = "testdata/search_cube_gold.bin"
)

type goldCase struct {
	pos    Position
	d1, d2 int8
	ply    int8

	// The configuration, GNC2 only (zero values in the GNCP corpus).
	useMatch     bool
	awayOnRoll   int8
	awayOpponent int8
	cube         int8
	crawford     bool
	useCube      bool
	owner        CubeOwner
	x            float64
}

// config is the SearchConfig a case asks for — the canonical filter and
// pruning, plus whatever referential and cube the case carries.
func (c goldCase) config() SearchConfig {
	cfg := DefaultConfig(int(c.ply))
	if c.useMatch {
		cfg.UseMatch = true
		cfg.Match = MatchState{AwayOnRoll: int(c.awayOnRoll), AwayOpponent: int(c.awayOpponent), Cube: int(c.cube), Crawford: c.crawford}
	}
	if c.useCube {
		cfg.UseCube = true
		cfg.CubeOwner = c.owner
		cfg.CubeX = c.x
	}
	return cfg
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

// buildSearchCubeCorpus is the ADR-0023 corpus: use_match and use_cube in every
// combination the port has to get right — the opening at the two scores
// where the cube decides the whole play (gammon-go, gammon-save), money with
// each owner, a Crawford game, post-Crawford, an owned cube at 2, and random
// boards through a cycle of states — at 0 and 1 ply mostly, a few at 2.
func buildSearchCubeCorpus() []goldCase {
	rng := rand.New(rand.NewSource(20260902))
	var cases []goldCase

	dp, _ := domain.DecodeXGID(openingXGID)
	dp.PlayerOnRoll = domain.White
	opening, _ := FromDomain(&dp)

	type ctx struct {
		useMatch        bool
		away, opp, cube int8
		crawford        bool
		owner           CubeOwner
	}
	x := func(o CubeOwner) float64 { return DefaultEfficiency(o) }
	mk := func(pos Position, d1, d2, ply int8, c ctx, useCube bool) goldCase {
		g := goldCase{pos: pos, d1: d1, d2: d2, ply: ply, useMatch: c.useMatch,
			awayOnRoll: c.away, awayOpponent: c.opp, cube: c.cube, crawford: c.crawford}
		if useCube {
			g.useCube, g.owner, g.x = true, c.owner, x(c.owner)
		}
		return g
	}

	gammonGo := ctx{useMatch: true, away: 4, opp: 2, cube: 1, owner: CubeCentred}
	gammonSave := ctx{useMatch: true, away: 2, opp: 4, cube: 1, owner: CubeCentred}
	// The opening at 2 ply where the cube changes the play (6-4, 5-4, 2-1)
	// and one where it does not (3-1), both scores, cubeful; 6-4 cubeless too,
	// so the gold pins the contrast and not just one side of it.
	for _, r := range [][2]int8{{6, 4}, {5, 4}, {2, 1}, {3, 1}} {
		cases = append(cases, mk(opening, r[0], r[1], 2, gammonGo, true), mk(opening, r[0], r[1], 2, gammonSave, true))
	}
	cases = append(cases, mk(opening, 6, 4, 2, gammonGo, false))
	// Money with the cube, every owner, at 2 ply on the 3-1 and 0 ply on all rolls.
	for _, o := range []CubeOwner{CubeCentred, CubeOwned, CubeOpponent} {
		c := ctx{cube: 1, owner: o}
		cases = append(cases, mk(opening, 3, 1, 2, c, true))
		for d1 := int8(1); d1 <= 6; d1++ {
			for d2 := d1; d2 <= 6; d2++ {
				cases = append(cases, mk(opening, d1, d2, 0, c, true))
			}
		}
	}

	// Random boards through a cycle of states: 0-ply broad, 1-ply regular, 2-ply rare.
	states := []ctx{
		{useMatch: true, away: 3, opp: 5, cube: 1, owner: CubeCentred},
		{useMatch: true, away: 2, opp: 2, cube: 1, owner: CubeCentred},
		{useMatch: true, away: 7, opp: 3, cube: 2, owner: CubeOwned},
		{useMatch: true, away: 5, opp: 4, cube: 2, owner: CubeOpponent},
		{useMatch: true, away: 1, opp: 4, cube: 1, crawford: true, owner: CubeCentred},
		{useMatch: true, away: 4, opp: 1, cube: 1, crawford: true, owner: CubeCentred},
		{useMatch: true, away: 2, opp: 1, cube: 1, owner: CubeCentred},
		{useMatch: true, away: 1, opp: 3, cube: 1, owner: CubeCentred},
		{useMatch: true, away: 11, opp: 9, cube: 4, owner: CubeOwned},
		{cube: 1, owner: CubeCentred},
		{cube: 1, owner: CubeOwned},
		{cube: 1, owner: CubeOpponent},
	}
	added := 0
	for added < 48 {
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
		case added%12 == 0:
			ply = 2
		case added%3 == 0:
			ply = 1
		}
		cases = append(cases, mk(p, d1, d2, ply, states[added%len(states)], true))
		added++
	}
	return cases
}

func encodeSearchCubeCorpus(cases []goldCase) []byte {
	var buf bytes.Buffer
	buf.WriteString(searchCubeCorpusMagic)
	_ = binary.Write(&buf, binary.LittleEndian, int32(len(cases)))
	b2i := func(b bool) byte {
		if b {
			return 1
		}
		return 0
	}
	for _, c := range cases {
		for _, v := range c.pos.Points {
			buf.WriteByte(byte(v))
		}
		buf.Write([]byte{c.pos.Bar[0], c.pos.Bar[1], c.pos.Off[0], c.pos.Off[1], c.pos.Turn})
		buf.Write([]byte{byte(c.d1), byte(c.d2), byte(c.ply)})
		buf.Write([]byte{b2i(c.useMatch), byte(c.awayOnRoll), byte(c.awayOpponent), byte(c.cube), b2i(c.crawford), b2i(c.useCube), byte(c.owner), 0})
		_ = binary.Write(&buf, binary.LittleEndian, c.x)
	}
	return buf.Bytes()
}

func decodeSearchCubeCorpus(t *testing.T, raw []byte) []goldCase {
	t.Helper()
	if len(raw) < 8 || string(raw[:4]) != searchCubeCorpusMagic {
		t.Fatalf("%s: unexpected magic", searchCubeCorpusPath)
	}
	n := int(int32(binary.LittleEndian.Uint32(raw[4:])))
	const entry = 48
	if len(raw) != 8+n*entry {
		t.Fatalf("%s: %d bytes for %d cases", searchCubeCorpusPath, len(raw), n)
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
		out[i].useMatch = b[32] != 0
		out[i].awayOnRoll, out[i].awayOpponent, out[i].cube = int8(b[33]), int8(b[34]), int8(b[35])
		out[i].crawford = b[36] != 0
		out[i].useCube = b[37] != 0
		out[i].owner = CubeOwner(b[38])
		out[i].x = math.Float64frombits(binary.LittleEndian.Uint64(b[40:]))
	}
	return out
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
	cube := buildSearchCubeCorpus()
	if err := os.WriteFile(searchCubeCorpusPath, encodeSearchCubeCorpus(cube), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %d cases to %s", len(cube), searchCubeCorpusPath)
}

// The cube corpus on disk must be exactly what buildSearchCubeCorpus produces —
// same guard, same reason, as TestGoldCorpusIsInSync.
func TestSearchCubeCorpusIsInSync(t *testing.T) {
	raw, err := os.ReadFile(searchCubeCorpusPath)
	if err != nil {
		t.Skipf("no cube corpus yet: %v", err)
	}
	if !bytes.Equal(raw, encodeSearchCubeCorpus(buildSearchCubeCorpus())) {
		t.Fatalf("%s is out of sync with buildSearchCubeCorpus(); regenerate BOTH corpus and gold", searchCubeCorpusPath)
	}
	if got := decodeSearchCubeCorpus(t, raw); len(got) != len(buildSearchCubeCorpus()) {
		t.Fatalf("decoded %d cases, built %d", len(got), len(buildSearchCubeCorpus()))
	}
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
