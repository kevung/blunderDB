package ingest

import (
	"path/filepath"
	"testing"
)

// gnubgLuckFixture is the gnuBG .sgf of the very match xgLuckFixture holds as
// an .xg — the pair that makes the cross-format check below possible.
func gnubgLuckFixture() string {
	return filepath.Join("..", "..", "..", "testdata", "charlot1-charlot2_7p_2025-11-08-2305.sgf")
}

// luckBySeat sums the luck of a mapped match per seat, and counts the rolls it
// was measured over.
func luckBySeat(t *testing.T, g *MatchGraph) (sum map[int32]int64, rolls map[int32]int) {
	t.Helper()
	sum, rolls = map[int32]int64{}, map[int32]int{}
	for gi := range g.Games {
		for mi := range g.Games[gi].Moves {
			mv := g.Games[gi].Moves[mi].Move
			if mv.LuckMP == nil {
				continue
			}
			sum[mv.Player] += int64(*mv.LuckMP)
			rolls[mv.Player]++
		}
	}
	return sum, rolls
}

// TestMapGnuBGCarriesLuck pins that a gnuBG import carries the luck of its
// rolls. It did not until gnubgparser v1.4.0: gnuBG writes the property as a
// single value, LU[-0.00537], while the parser required a rating word in front
// of it and returned nothing on every real file. Nothing here failed at the
// time — an absent LU is a legitimate state — which is exactly why this test
// exists.
func TestMapGnuBGCarriesLuck(t *testing.T) {
	g, err := MapGnuBG(gnubgLuckFixture())
	if err != nil {
		t.Fatalf("MapGnuBG: %v", err)
	}

	sum, rolls := luckBySeat(t, g)
	total := rolls[1] + rolls[-1]
	if total != 189 {
		t.Errorf("rolls carrying luck: got %d, want 189", total)
	}
	// Both signs must be present: a match where nobody was ever unlucky would
	// mean the sign was lost on the way in.
	if sum[1] <= 0 || sum[-1] >= 0 {
		t.Errorf("expected one lucky and one unlucky side, got %+d and %+d", sum[1], sum[-1])
	}
}

// TestLuckAgreesAcrossFormats is the check that gives the unit its meaning. The
// same match exists here as an .xg and as a gnuBG .sgf, analysed by two
// different engines, so the numbers cannot match exactly — but the seat each
// value lands on, the number of rolls, and the sign of each player's total all
// have to. An inverted seat or a flipped sign in either importer would still
// look perfectly plausible read on its own; it cannot survive this comparison.
func TestLuckAgreesAcrossFormats(t *testing.T) {
	xg, err := MapXG(xgLuckFixture())
	if err != nil {
		t.Fatalf("MapXG: %v", err)
	}
	sgf, err := MapGnuBG(gnubgLuckFixture())
	if err != nil {
		t.Fatalf("MapGnuBG: %v", err)
	}
	if xg.Match.Player1Name != sgf.Match.Player1Name {
		t.Fatalf("the two fixtures are not the same match seen the same way round: %q vs %q",
			xg.Match.Player1Name, sgf.Match.Player1Name)
	}

	xgSum, xgRolls := luckBySeat(t, xg)
	sgfSum, sgfRolls := luckBySeat(t, sgf)

	for _, seat := range []int32{1, -1} {
		if xgRolls[seat] != sgfRolls[seat] {
			t.Errorf("seat %+d: %d measured rolls from XG, %d from gnuBG", seat, xgRolls[seat], sgfRolls[seat])
		}
		if (xgSum[seat] > 0) != (sgfSum[seat] > 0) {
			t.Errorf("seat %+d: XG says %+d millipoints, gnuBG says %+d — the two disagree on who was lucky",
				seat, xgSum[seat], sgfSum[seat])
		}
		// Different engines, so only the order of magnitude is comparable.
		lo, hi := xgSum[seat], sgfSum[seat]
		if lo < 0 {
			lo, hi = -lo, -hi
		}
		if hi > 3*lo || lo > 3*hi {
			t.Errorf("seat %+d: %+d millipoints from XG against %+d from gnuBG is not the same measurement",
				seat, xgSum[seat], sgfSum[seat])
		}
	}
}
