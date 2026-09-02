package ingest

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/gnubgparser"
)

// crawfordGameOf returns the index of the one Crawford game of a mapped match
// — the first game in which exactly one player is 1-away — and fails when the
// match has none. Both importers derive the away score from the score before
// the game, so this reads the same for an .xg and an .sgf.
func crawfordGameOf(t *testing.T, g *MatchGraph) int {
	t.Helper()
	L := int(g.Match.MatchLength)
	for gi := range g.Games {
		s := g.Games[gi].Game.InitialScore
		a, b := L-int(s[0]), L-int(s[1])
		if (a == 1) != (b == 1) {
			return gi
		}
	}
	t.Fatalf("no Crawford game in %s", g.Match.FilePath)
	return -1
}

// TestGnuBGCrawfordGameIsFlaggedAndConverted pins issue #170's fix at the
// seam: the SGF parser reports the Crawford game (RU[Crawford:CrawfordGame])
// and the conversion receives that flag rather than a hard-wired false. The
// fixture is a 7-point match whose fourth game is the Crawford game (6-2).
func TestGnuBGCrawfordGameIsFlaggedAndConverted(t *testing.T) {
	match, err := gnubgparser.ParseSGFFile(gnubgLuckFixture())
	if err != nil {
		t.Fatalf("ParseSGFFile: %v", err)
	}
	var flagged []int
	for i, g := range match.Games {
		if g.CrawfordGame {
			flagged = append(flagged, i)
		}
	}
	if len(flagged) != 1 || flagged[0] != 3 {
		t.Fatalf("Crawford game index: got %v, want [3] (the 6-2 game of the 7-point fixture)", flagged)
	}
	g := match.Games[3]
	if g.Score != [2]int{6, 2} {
		t.Fatalf("Crawford game score: got %v, want [6 2]", g.Score)
	}

	// gnuBG writes no cube analysis in the Crawford game: the cube is dead, so
	// there is nothing to double and nothing to convert. This is why the
	// hard-wired flag never reached a stored equity.
	for i, mv := range g.Moves {
		if mv.CubeAnalysis != nil {
			t.Errorf("move %d of the Crawford game carries a cube analysis; gnuBG does not evaluate a dead cube", i)
		}
	}

	// The conversion at that score, with the flag the parser reports and with
	// the value that used to be hard-wired, gives the same equities — and both
	// are the normalised scale: a double/pass cashes exactly one cube.
	mk := func() *gnubgparser.CubeAnalysis {
		return &gnubgparser.CubeAnalysis{CubefulNoDouble: 0.62, CubefulDoubleTake: 0.60, CubefulDoublePass: 0.7}
	}
	withFlag, without := mk(), mk()
	convertGnuBGCubeMWCToEMG(withFlag, g.Score[0], g.Score[1], 0, 1, match.Metadata.MatchLength, g.CrawfordGame)
	convertGnuBGCubeMWCToEMG(without, g.Score[0], g.Score[1], 0, 1, match.Metadata.MatchLength, false)
	if *withFlag != *without {
		t.Errorf("Crawford flag changed the conversion at a Crawford score:\n with %+v\n without %+v", *withFlag, *without)
	}
	if withFlag.CubefulDoublePass != 1.0 {
		t.Errorf("double/pass after conversion = %v, want exactly 1 (normalised scale, ADR-0019)", withFlag.CubefulDoublePass)
	}
}

// TestCrawfordFlagIsRedundantWithTheScoreInTheMET is the reason issue #170
// produced no wrong equity. gnuBG's getME (engine.GnuBGGetME) switches to the
// post-Crawford table when fCrawford is set OR when either player is 1-away
// before the game; in a Crawford game one player is 1-away by definition, so
// the flag is implied by the score for every match length, both movers and
// every live cube. The control at the end shows the flag is not inert in
// general: at a score where nobody is 1-away it does change the lookup —
// which is exactly why it must be propagated, never guessed.
func TestCrawfordFlagIsRedundantWithTheScoreInTheMET(t *testing.T) {
	compared := 0
	for L := 2; L <= 25; L++ {
		for other := 0; other < L-1; other++ { // the other player, strictly more than 1-away
			for _, scores := range [][2]int{{L - 1, other}, {other, L - 1}} {
				for fMove := 0; fMove < 2; fMove++ {
					for cube := 1; cube <= 32; cube *= 2 {
						for who := 0; who < 2; who++ {
							a := engine.GnuBGGetME(scores[0], scores[1], L, fMove, cube, who, true)
							b := engine.GnuBGGetME(scores[0], scores[1], L, fMove, cube, who, false)
							if a != b {
								t.Fatalf("GnuBGGetME(%v, L=%d, fMove=%d, cube=%d, who=%d): Crawford %v vs not %v",
									scores, L, fMove, cube, who, a, b)
							}
							compared++
						}
					}
				}
			}
		}
	}
	if compared == 0 {
		t.Fatal("no Crawford score compared")
	}

	// Control: 5-away/5-away is not a Crawford score, and there the flag moves
	// the result (it would wrongly price the next game as post-Crawford).
	if a, b := engine.GnuBGGetME(2, 2, 7, 0, 1, 0, true), engine.GnuBGGetME(2, 2, 7, 0, 1, 0, false); a == b {
		t.Fatalf("control failed: fCrawford has no effect at 5-away/5-away (%v); the sweep above proves nothing", a)
	}
}

// TestCrawfordGameAgreesAcrossFormats extends TestLuckAgreesAcrossFormats to
// the Crawford game of the same match seen through XG and gnuBG. What must
// agree: which game is the Crawford game and the away score every position of
// that game carries. On the gnuBG side no cube analysis may exist there at
// all — the cube is dead and gnuBG does not evaluate it — which is the whole
// reach of issue #170's flag: it had nothing to convert. (XG is different: it
// records one degenerate cube line at the end of the game, so its side is not
// asserted beyond the score.)
func TestCrawfordGameAgreesAcrossFormats(t *testing.T) {
	xg, err := MapXG(xgLuckFixture())
	if err != nil {
		t.Fatalf("MapXG: %v", err)
	}
	sgf, err := MapGnuBG(gnubgLuckFixture())
	if err != nil {
		t.Fatalf("MapGnuBG: %v", err)
	}
	xgIdx, sgfIdx := crawfordGameOf(t, xg), crawfordGameOf(t, sgf)
	if xgIdx != sgfIdx {
		t.Fatalf("Crawford game index: XG %d, gnuBG %d", xgIdx, sgfIdx)
	}

	for name, g := range map[string]*GameGraph{"XG": &xg.Games[xgIdx], "gnuBG": &sgf.Games[sgfIdx]} {
		if len(g.Moves) == 0 {
			t.Fatalf("%s: empty Crawford game", name)
		}
		for i, mv := range g.Moves {
			if mv.Position == nil {
				continue
			}
			if got := mv.Position.Score; got != [2]int{domain.Crawford, 5} {
				t.Errorf("%s move %d: away score %v, want [1 5] (6-2 in a 7-point match)", name, i, got)
			}
		}
	}
	for i, mv := range sgf.Games[sgfIdx].Moves {
		for _, a := range mv.Analyses {
			if a.DoublingCubeAnalysis != nil {
				t.Errorf("gnuBG move %d: cube analysis in the Crawford game (%+v); gnuBG does not evaluate a dead cube", i, *a.DoublingCubeAnalysis)
			}
		}
	}
}
