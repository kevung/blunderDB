// SPDX-License-Identifier: MIT

package gammonnet

import (
	"bytes"
	"encoding/binary"
	"math"
	"math/rand"
	"os"
	"testing"
)

// The cube gold corpus: a fixed list of (distribution, owner, efficiency,
// jacoby, money-or-match-state) cube decisions that both this port and the C
// reference (gn_cube_decide) must answer identically.
//
// Match-state cases are held to away scores <= cubeGoldMaxAway: gammonNet's
// own gn_match_state_is_valid refuses beyond GN_MET_MAX_AWAY (25). Its
// post-Crawford table now carries the full 25 entries on both sides of the
// port (#24: blunderDB's own MET reads the same gammonNet export instead of
// a hand transcription, so the trailer-at-25-away boundary this file used to
// avoid is answered identically by both engines) — cubeGoldMaxAway is
// therefore 25, gammonNet's own horizon, not one short of it.
const (
	cubeCorpusMagic = "GNCB"
	cubeGoldMagic   = "GNCG"
	cubeCorpusPath  = "testdata/cube_corpus.bin"
	cubeGoldPath    = "testdata/cube_gold.bin"
	cubeGoldMaxAway = 25

	// cubeGoldTolerance matches the search gold's 1e-6 (#24): blunderDB's own
	// MET used to store Kazaross-XG2 in float32, transcribed by hand, where
	// gammonNet's table is double — that gap (measured max|Δ| = 2.463e-06
	// over 2320 decisions, see README.md) forced a tolerance ten times
	// looser than the rest of this harness. blunderDB's MET now reads
	// gammonNet's own float64 export for every index this file exercises
	// (metPre/metPost in engine/met.go), closing the gap this tolerance
	// existed to paper over.
	cubeGoldTolerance = 1e-6

	// cubeActionTieTolerance is the decision tie zone: an action disagreement
	// is tolerated only when the reported equity_no_double and equity_double
	// already agree to within this — i.e. the two engines are disagreeing
	// about which side of an exact tie residual floating-point noise falls
	// on, not about the decision itself. See TestCubeDecideMatchesTheGoldFile.
	cubeActionTieTolerance = 1e-8
)

type cubeGoldCase struct {
	probs        [NumOutputs]float32
	owner        CubeOwner
	efficiency   float64
	jacoby       bool
	hasState     bool
	awayOnRoll   int32
	awayOpponent int32
	cube         int32
	crawford     bool
}

// buildCubeCorpus is the fixed set of cube decisions checked against the C
// reference. Coverage: every owner, both Jacoby settings, efficiency at the
// measured defaults plus the two extremes (fully dead / fully live), a
// spread of distributions from near-certain loss to near-certain
// backgammon-heavy win, and match states spanning even and lopsided scores,
// every cube power up to 16, and both Crawford settings.
func buildCubeCorpus() []cubeGoldCase {
	rng := rand.New(rand.NewSource(20260828))
	var cases []cubeGoldCase

	dists := [][5]float32{
		{0.02, 0.003, 0.0002, 0.15, 0.02}, // near-certain loss
		{0.5, 0.1, 0.01, 0.1, 0.01},       // even game, some gammons
		{0.65, 0.3, 0.05, 0.05, 0.005},    // clear favourite, gammonish
		{0.5, 0.0, 0.0, 0.0, 0.0},         // exactly even, gammonless
		{0.97, 0.4, 0.1, 0.0, 0.0},        // near-certain win
		{0.8, 0.5, 0.2, 0.02, 0.0},        // backgammon-heavy favourite
		{1.0, 0.6, 0.1, 0.0, 0.0},         // p(win) = 1: the win-side degenerate case
		{0.0, 0.0, 0.0, 0.3, 0.05},        // p(win) = 0: the lose-side degenerate case
	}
	owners := []CubeOwner{CubeCentred, CubeOwned, CubeOpponent}
	effs := []float64{0.0, 0.566, 0.687, 0.688, 1.0}

	// Money.
	for _, d := range dists {
		for _, owner := range owners {
			for _, eff := range effs {
				for _, jacoby := range []bool{false, true} {
					cases = append(cases, cubeGoldCase{
						probs: d, owner: owner, efficiency: eff, jacoby: jacoby,
					})
				}
			}
		}
	}

	// Match: a spread of scores, deliberately including 1-away on either
	// side (Crawford/post-Crawford territory) and near-even long matches.
	scores := [][2]int32{
		{1, 1}, {1, 5}, {5, 1}, {2, 2}, {2, 4}, {4, 2}, {3, 7}, {7, 3},
		{cubeGoldMaxAway, cubeGoldMaxAway}, {cubeGoldMaxAway, 3}, {3, cubeGoldMaxAway},
		{9, 9}, {1, cubeGoldMaxAway},
	}
	cubes := []int32{1, 2, 4, 8, 16}
	for _, d := range dists {
		for _, sc := range scores {
			// The Crawford flag is only coherent when one side is already at
			// match point entering the game (MatchState.IsValid refuses any
			// other combination) — so it is only exercised at scores that
			// actually have a 1-away side.
			crawfordChoices := []bool{false}
			if sc[0] == 1 || sc[1] == 1 {
				crawfordChoices = []bool{false, true}
			}
			for _, cube := range cubes {
				for _, owner := range owners {
					for _, crawford := range crawfordChoices {
						cases = append(cases, cubeGoldCase{
							probs: d, owner: owner, efficiency: DefaultEfficiency(owner),
							hasState: true, awayOnRoll: sc[0], awayOpponent: sc[1],
							cube: cube, crawford: crawford,
						})
					}
				}
			}
		}
	}

	// A handful of randomised distributions at randomised (small) scores, to
	// broaden coverage beyond the hand-picked corners above.
	for i := 0; i < 40; i++ {
		win := rng.Float32()
		winG := win * rng.Float32() * 0.6
		winBG := winG * rng.Float32() * 0.3
		loseG := (1 - win) * rng.Float32() * 0.6
		loseBG := loseG * rng.Float32() * 0.3
		d := [5]float32{win, winG, winBG, loseG, loseBG}
		owner := owners[i%len(owners)]
		if i%3 == 0 {
			cases = append(cases, cubeGoldCase{
				probs: d, owner: owner, efficiency: DefaultEfficiency(owner), jacoby: i%2 == 0,
			})
			continue
		}
		awayOnRoll := int32(1 + rng.Intn(cubeGoldMaxAway))
		awayOpponent := int32(1 + rng.Intn(cubeGoldMaxAway))
		crawford := i%7 == 0 && (awayOnRoll == 1 || awayOpponent == 1)
		cases = append(cases, cubeGoldCase{
			probs: d, owner: owner, efficiency: DefaultEfficiency(owner), hasState: true,
			awayOnRoll: awayOnRoll, awayOpponent: awayOpponent,
			cube: int32(1 << uint(rng.Intn(5))), crawford: crawford,
		})
	}

	return cases
}

func encodeCubeCorpus(cases []cubeGoldCase) []byte {
	var buf bytes.Buffer
	buf.WriteString(cubeCorpusMagic)
	_ = binary.Write(&buf, binary.LittleEndian, int32(len(cases)))
	for _, c := range cases {
		_ = binary.Write(&buf, binary.LittleEndian, c.probs)
		_ = binary.Write(&buf, binary.LittleEndian, int32(c.owner))
		_ = binary.Write(&buf, binary.LittleEndian, c.efficiency)
		_ = binary.Write(&buf, binary.LittleEndian, boolToInt32(c.jacoby))
		_ = binary.Write(&buf, binary.LittleEndian, boolToInt32(c.hasState))
		_ = binary.Write(&buf, binary.LittleEndian, c.awayOnRoll)
		_ = binary.Write(&buf, binary.LittleEndian, c.awayOpponent)
		_ = binary.Write(&buf, binary.LittleEndian, c.cube)
		_ = binary.Write(&buf, binary.LittleEndian, boolToInt32(c.crawford))
	}
	return buf.Bytes()
}

func boolToInt32(b bool) int32 {
	if b {
		return 1
	}
	return 0
}

const cubeCorpusEntry = 4*5 + 4 + 8 + 4 + 4 + 4 + 4 + 4 + 4 // 56 bytes

func decodeCubeCorpus(t *testing.T, raw []byte) []cubeGoldCase {
	t.Helper()
	if len(raw) < 8 || string(raw[:4]) != cubeCorpusMagic {
		t.Fatalf("%s: unexpected magic", cubeCorpusPath)
	}
	n := int(int32(binary.LittleEndian.Uint32(raw[4:])))
	if len(raw) != 8+n*cubeCorpusEntry {
		t.Fatalf("%s: %d bytes for %d cases", cubeCorpusPath, len(raw), n)
	}
	out := make([]cubeGoldCase, n)
	for i := range out {
		b := raw[8+i*cubeCorpusEntry:]
		r := bytes.NewReader(b)
		var owner, jacoby, hasState, crawford int32
		_ = binary.Read(r, binary.LittleEndian, &out[i].probs)
		_ = binary.Read(r, binary.LittleEndian, &owner)
		_ = binary.Read(r, binary.LittleEndian, &out[i].efficiency)
		_ = binary.Read(r, binary.LittleEndian, &jacoby)
		_ = binary.Read(r, binary.LittleEndian, &hasState)
		_ = binary.Read(r, binary.LittleEndian, &out[i].awayOnRoll)
		_ = binary.Read(r, binary.LittleEndian, &out[i].awayOpponent)
		_ = binary.Read(r, binary.LittleEndian, &out[i].cube)
		_ = binary.Read(r, binary.LittleEndian, &crawford)
		out[i].owner = CubeOwner(owner)
		out[i].jacoby = jacoby != 0
		out[i].hasState = hasState != 0
		out[i].crawford = crawford != 0
	}
	return out
}

// TestWriteCubeGoldCorpus regenerates the corpus. Deliberate, never automatic.
func TestWriteCubeGoldCorpus(t *testing.T) {
	if os.Getenv("BLUNDERDB_WRITE_CORPUS") == "" {
		t.Skip("set BLUNDERDB_WRITE_CORPUS to regenerate")
	}
	cases := buildCubeCorpus()
	if err := os.WriteFile(cubeCorpusPath, encodeCubeCorpus(cases), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %d cases to %s", len(cases), cubeCorpusPath)
}

// The corpus on disk must be exactly what buildCubeCorpus produces.
func TestCubeGoldCorpusIsInSync(t *testing.T) {
	raw, err := os.ReadFile(cubeCorpusPath)
	if err != nil {
		t.Skipf("no corpus yet: %v", err)
	}
	if !bytes.Equal(raw, encodeCubeCorpus(buildCubeCorpus())) {
		t.Fatal("testdata/cube_corpus.bin does not match buildCubeCorpus(); the gold file answers different questions")
	}
	cases := decodeCubeCorpus(t, raw)
	money, match := 0, 0
	for _, c := range cases {
		if c.hasState {
			match++
		} else {
			money++
		}
	}
	t.Logf("%d cases: %d money, %d match", len(cases), money, match)
}

type cubeGoldEntry struct {
	ok             bool
	action         int32
	equityNoDouble float64
	equityDouble   float64
	takePoint      float64
}

func loadCubeGold(t *testing.T) []cubeGoldEntry {
	t.Helper()
	raw, err := os.ReadFile(cubeGoldPath)
	if err != nil {
		t.Skipf("no gold file: %v", err)
	}
	if len(raw) < 8 || string(raw[:4]) != cubeGoldMagic {
		t.Fatalf("%s: unexpected magic", cubeGoldPath)
	}
	n := int(int32(binary.LittleEndian.Uint32(raw[4:])))
	const entry = 4 + 4 + 8 + 8 + 8
	if len(raw) != 8+n*entry {
		t.Fatalf("%s: %d bytes for %d cases", cubeGoldPath, len(raw), n)
	}
	out := make([]cubeGoldEntry, n)
	for i := range out {
		b := raw[8+i*entry:]
		out[i].ok = int32(binary.LittleEndian.Uint32(b[0:])) != 0
		out[i].action = int32(binary.LittleEndian.Uint32(b[4:]))
		out[i].equityNoDouble = math.Float64frombits(binary.LittleEndian.Uint64(b[8:]))
		out[i].equityDouble = math.Float64frombits(binary.LittleEndian.Uint64(b[16:]))
		out[i].takePoint = math.Float64frombits(binary.LittleEndian.Uint64(b[24:]))
	}
	return out
}

// TestCubeDecideMatchesTheGoldFile is the parity gate: for every corpus case,
// this port's Decide must agree with gammonNet's C gn_cube_decide on
// refusal, the action taken, and the reported equities/take point to
// cubeGoldTolerance.
func TestCubeDecideMatchesTheGoldFile(t *testing.T) {
	corpusRaw, err := os.ReadFile(cubeCorpusPath)
	if err != nil {
		t.Skipf("no corpus: %v", err)
	}
	cases := decodeCubeCorpus(t, corpusRaw)
	gold := loadCubeGold(t)
	if len(gold) != len(cases) {
		t.Fatalf("gold file has %d entries, corpus has %d", len(gold), len(cases))
	}

	var maxDelta float64
	var worst string
	var ties int
	for i, c := range cases {
		var state *MatchState
		if c.hasState {
			state = &MatchState{
				AwayOnRoll:   int(c.awayOnRoll),
				AwayOpponent: int(c.awayOpponent),
				Cube:         int(c.cube),
				Crawford:     c.crawford,
			}
		}

		dec, ok := Decide(&c.probs, c.owner, state, c.efficiency, c.jacoby)
		want := gold[i]

		if ok != want.ok {
			t.Errorf("case %d: ok = %v, want %v (owner=%v state=%+v)", i, ok, want.ok, c.owner, state)
			continue
		}
		if !ok {
			continue
		}
		if int32(dec.Action) != want.action {
			// At an exact decision boundary (eND == eDouble in real
			// arithmetic — e.g. a perfectly symmetric score with p = 0.5),
			// residual floating-point noise can push a strict eDouble > eND
			// either side of zero and flip NoDouble/DoubleTake. Rarer since
			// #24 (blunderDB's MET now reads gammonNet's own float64 export
			// instead of a float32 hand transcription), but still possible.
			// Verdict's own comparisons are strict for exactly this reason
			// (an exact tie must read NoDouble), so this is a tie-zone
			// disagreement, not a computed-value one — tolerated only within
			// cubeActionTieTolerance, and counted so a growing count is
			// itself a signal, exactly as the search gold file treats its
			// own equity ties.
			if delta := math.Abs(dec.EquityNoDouble - dec.EquityDouble); delta <= cubeActionTieTolerance {
				ties++
			} else {
				t.Errorf("case %d: action = %v, want %v (owner=%v eff=%v state=%+v probs=%v, |eND-eD|=%v)",
					i, dec.Action, want.action, c.owner, c.efficiency, state, c.probs, delta)
			}
		}
		for name, d := range map[string][2]float64{
			"equity_no_double": {dec.EquityNoDouble, want.equityNoDouble},
			"equity_double":    {dec.EquityDouble, want.equityDouble},
			"take_point":       {dec.TakePoint, want.takePoint},
		} {
			delta := math.Abs(d[0] - d[1])
			if delta > cubeGoldTolerance {
				t.Errorf("case %d: %s = %v, want %v (Δ=%v > %v)", i, name, d[0], d[1], delta, cubeGoldTolerance)
			}
			if delta > maxDelta {
				maxDelta = delta
				worst = name
			}
		}
	}
	t.Logf("%d cases, max|Δ| = %.3e (%s), %d action ties tolerated", len(cases), maxDelta, worst, ties)
}
