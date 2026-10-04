package ingest

import (
	"encoding/json"
	"math"
	"os"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/ogxmparser"
	"github.com/kevung/ogxmparser/ogid"
)

// A match exported by HedgeHog with its analysis (3 points, rchoice v
// Jezebel), the file ogxmparser's corpus holds to the reference
// implementation.
const ogxmRealFile = "../../../testdata/hedgehog-3pt-analysed.ogxm"

func TestMapOGXMRealMatch(t *testing.T) {
	g, err := MapOGXM(ogxmRealFile)
	if err != nil {
		t.Fatal(err)
	}
	m := g.Match
	// blunderDB's player 1 is Black.
	if m.Player1Name != "Jezebel" || m.Player2Name != "rchoice" || m.MatchLength != 3 || m.GameCount != 4 {
		t.Fatalf("match %+v", m)
	}
	if m.MatchHash == "" || m.CanonicalHash == "" {
		t.Error("missing hashes")
	}
	if len(g.Games) != 4 {
		t.Fatalf("%d games", len(g.Games))
	}
	// Game 1: Black wins 1 point on a dropped double; game 4 is Crawford.
	if g.Games[0].Game.Winner != domain.WinnerFromSide(domain.Black) || g.Games[0].Game.PointsWon != 1 {
		t.Errorf("game 1: %+v", g.Games[0].Game)
	}

	first := g.Games[0].Moves[0]
	pos := first.Position
	if first.Move.MoveType != "checker" || pos.PlayerOnRoll != domain.White || pos.Dice != [2]int{4, 6} {
		t.Fatalf("first move %+v on %+v", first.Move, pos)
	}
	opening := domain.Position{}
	opening.Board.Points[1] = domain.Point{Checkers: 2, Color: domain.White}
	if pos.Board.Points[1] != opening.Board.Points[1] || pos.Board.Points[24] != (domain.Point{Checkers: 2, Color: domain.Black}) {
		t.Errorf("opening board %v", pos.Board.Points)
	}
	if pos.Score != [2]int{3, 3} {
		t.Errorf("away score %v at 0-0 of 3", pos.Score)
	}
	// White played 13/9 24/18 (reference notation); the played move is
	// among the alternatives under the same spelling.
	a := first.Analyses[0]
	if a.CheckerAnalysis == nil || len(a.PlayedMoves) != 1 {
		t.Fatalf("analysis %+v", a)
	}
	found := false
	for _, cm := range a.CheckerAnalysis.Moves {
		if cm.Move == a.PlayedMoves[0] {
			found = true
		}
	}
	if !found || a.PlayedMoves[0] != "24/18 13/9" {
		t.Errorf("played %q not among %+v", a.PlayedMoves, a.CheckerAnalysis.Moves)
	}
	// HedgeHog rates no opening roll; the reply's luck is -9.185e-3.
	if first.Move.LuckMP != nil {
		t.Errorf("luck %d on the opening roll", *first.Move.LuckMP)
	}
	if l := g.Games[0].Moves[1].Move.LuckMP; l == nil || *l != -9 {
		t.Errorf("luck of Black's reply: %v, want -9 millipoints", l)
	}

	// Black's cube decision before rolling 4-6, against the normalised
	// equities of the reference's projection; every cube decision of the
	// match is held to it in TestOGXMCubeEquitiesMatchReference.
	second := g.Games[0].Moves[1]
	var cube *domain.DoublingCubeAnalysis
	for _, an := range second.Analyses {
		if an.DoublingCubeAnalysis != nil {
			cube = an.DoublingCubeAnalysis
		}
	}
	if cube == nil {
		t.Fatal("no cube analysis on Black's first roll")
	}
	if math.Abs(cube.CubefulNoDoubleEquity+0.011112) > ogxmCubeTol || cube.CubefulDoublePassEquity != 1 {
		t.Errorf("cube %+v, reference no double -0.011112", cube)
	}

	// Each game's cube decisions on a double are cube moves.
	cubes := 0
	for _, gg := range g.Games {
		for _, mv := range gg.Moves {
			if mv.Move.MoveType == "cube" {
				cubes++
				if mv.Move.CubeAction != "Double/Pass" || mv.Position.DecisionType != domain.CubeAction {
					t.Errorf("cube move %+v", mv.Move)
				}
			}
		}
	}
	if cubes != 2 {
		t.Errorf("%d doubles, the match has 2", cubes)
	}
}

// A block that is not in cubeful money currency would be stored as cubeful
// equities it is not: it is left out whole, luck included.
func TestOGXMIgnoresNonCubefulBlocks(t *testing.T) {
	for _, c := range []ogxmparser.Currency{ogxmparser.Cubeless, ogxmparser.CurrencyUnset, ogxmparser.CubefulMatch} {
		f := &ogxmparser.File{
			Match:    &ogxmparser.Match{Length: 3},
			Analyses: []ogxmparser.Analysis{{Currency: c, Decisions: []ogxmparser.Decision{{Kind: ogxmparser.KindRoll, Roll: &ogxmparser.RollDecision{Luck: 0.1}}}}},
		}
		if an := ogxmDecisions(f); an != nil {
			t.Errorf("currency %d: block imported", c)
		}
	}
}

// ogxmCubeTol is what separates the reference's projection from equities
// normalised on the file's own anchors: HedgeHog rounds MWC to 1e-4, and the
// projection normalises on its MET where blunderDB uses the file's
// double/pass, a few 1e-4 after scaling.
const ogxmCubeTol = 3e-4

func parseOGXMReal(t *testing.T) (*ogxmparser.File, []ogxmparser.GameReplay, *ogxmAnalysis) {
	t.Helper()
	f, err := ogxmparser.ParseFile(ogxmRealFile)
	if err != nil {
		t.Fatal(err)
	}
	an := ogxmDecisions(f)
	replays := f.Replay()
	an.indexCubeAnchors(f, replays)
	return f, replays, an
}

// The losing anchor is the opponent's own double/pass at the same score and
// cube, which the file holds at every score of this match.
func TestOGXMCubeAnchorsFromFile(t *testing.T) {
	_, _, an := parseOGXMReal(t)
	if an.noCube {
		t.Fatal("cube analysis left out")
	}
	lose := func(player, black, white int) float64 {
		k := ogxmAnchorKey{player: 1 - player, scoreBlack: black, scoreWhite: white, cube: 1}
		w, ok := an.winAnchor[k]
		if !ok {
			t.Fatalf("no anchor %+v", k)
		}
		return 1 - w
	}
	for _, c := range []struct {
		name                 string
		player, black, white int
		want                 float64
	}{
		{"0-0 Black", domain.Black, 0, 0, 0.4010},
		{"0-0 White", domain.White, 0, 0, 0.4010},
		{"1-0 Black", domain.Black, 1, 0, 0.5000},
		{"1-0 White", domain.White, 1, 0, 0.2488},
	} {
		if got := lose(c.player, c.black, c.white); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("%s: losing anchor %.4f, want %.4f", c.name, got, c.want)
		}
	}
}

// Every cube decision a doubler faces in the match, against the normalised
// equities of HedgeHog's reference projection.
func TestOGXMCubeEquitiesMatchReference(t *testing.T) {
	raw, err := os.ReadFile("../../../testdata/hedgehog-3pt-analysed.cube-norm.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows struct {
		CubeDecisions []struct {
			Game       int     `json:"game"`
			Ply        int     `json:"ply"`
			OGID       string  `json:"ogid"`
			NoDouble   float64 `json:"no_double"`
			DoubleTake float64 `json:"double_take"`
			DoublePass float64 `json:"double_pass"`
		} `json:"cube_decisions"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		t.Fatal(err)
	}
	f, replays, an := parseOGXMReal(t)
	worst := 0.0
	for _, r := range rows.CubeDecisions {
		p := &f.Games[r.Game].Plies[r.Ply]
		rp := replays[r.Game]
		before := rp.Plies[r.Ply].Before
		want, err := ogid.Parse(r.OGID)
		if err != nil {
			t.Fatal(err)
		}
		if before.Board != want.Board || before.Score != want.Score || before.CubeLog2 != want.CubeLog2 {
			t.Fatalf("game %d ply %d: replay is not at %s", r.Game, r.Ply, r.OGID)
		}
		pd := an.byRef[p.Ref]
		if pd == nil || pd.cube == nil {
			t.Fatalf("game %d ply %d: no cube decision", r.Game, r.Ply)
		}
		got := ogxmCubeAnalysis(an, pd.cube, &before, ogxmColour(p.Seat), f.Match.Length, rp.Crawford)
		if got == nil {
			t.Fatalf("game %d ply %d: cube analysis left out", r.Game, r.Ply)
		}
		for _, c := range []struct {
			name      string
			got, want float64
		}{
			{"no double", got.CubefulNoDoubleEquity, r.NoDouble},
			{"double/take", got.CubefulDoubleTakeEquity, r.DoubleTake},
			{"double/pass", got.CubefulDoublePassEquity, r.DoublePass},
		} {
			d := math.Abs(c.got - c.want)
			worst = math.Max(worst, d)
			if d > ogxmCubeTol {
				t.Errorf("game %d ply %d %s: %.6f, reference %.6f", r.Game, r.Ply, c.name, c.got, c.want)
			}
		}
	}
	if len(rows.CubeDecisions) < 70 {
		t.Fatalf("%d cube decisions, the match has 77", len(rows.CubeDecisions))
	}
	t.Logf("worst deviation %.2e over %d decisions", worst, len(rows.CubeDecisions))
}

// A double/pass far from the MET's winning anchor is not the MWC it is read
// as: the block's cube analysis is left out, the rest of it kept.
func TestOGXMImplausibleDoublePassDropsCubeAnalysis(t *testing.T) {
	f, replays, _ := parseOGXMReal(t)
	an := ogxmDecisions(f)
	// The first decision indexCubeAnchors checks, found in ply order: a
	// decision it skips (a ply past a failed replay, an action that is neither
	// a roll nor a double) would leave the cube analysis in place.
	perturbed := false
plies:
	for gi := range f.Games {
		for pi, p := range f.Games[gi].Plies {
			if rp := replays[gi]; rp.FailedAt >= 0 && pi >= rp.FailedAt {
				break
			}
			pd := an.byRef[p.Ref]
			if pd == nil || pd.cube == nil || !ogxmCubeful(pd.cube) || pd.cube.DoublePassEquity == nil {
				continue
			}
			if !p.IsDiceAction() && p.Action != ogxmparser.ActionDouble {
				continue
			}
			off := *pd.cube.DoublePassEquity + 0.05
			pd.cube.DoublePassEquity = &off
			perturbed = true
			break plies
		}
	}
	if !perturbed {
		t.Fatal("no double/pass equity to perturb in the fixture")
	}
	an.indexCubeAnchors(f, replays)
	if !an.noCube {
		t.Fatal("an implausible double/pass kept the cube analysis")
	}
	for _, pd := range an.byRef {
		if pd.cube != nil {
			if c := ogxmCubeAnalysis(an, pd.cube, &replays[0].Plies[0].Before, domain.Black, f.Match.Length, false); c != nil {
				t.Fatalf("cube analysis %+v imported", c)
			}
		}
	}
}

// At money play a cube decision stated in match winning chances has no
// meaning: it is left out.
func TestOGXMMoneyCubeInMatchCurrency(t *testing.T) {
	eq := func(v float64) *float64 { return &v }
	cur := ogxmparser.CubefulMatch
	d := &ogxmparser.CubeDecision{NoDoubleEquity: eq(0.1), DoubleTakeEquity: eq(0.2), DoublePassEquity: eq(1), Currency: &cur}
	an := &ogxmAnalysis{block: &ogxmparser.Analysis{Currency: ogxmparser.CubefulMoney}}
	before := ogid.New()
	if c := ogxmCubeAnalysis(an, d, &before, domain.Black, 0, false); c != nil {
		t.Errorf("money cube in MWC imported: %+v", c)
	}
	d.Currency = nil
	if c := ogxmCubeAnalysis(an, d, &before, domain.Black, 0, false); c == nil || c.CubefulDoubleTakeEquity != 0.2 {
		t.Errorf("money cube in the block's currency: %+v", c)
	}
}

// The positions an OGXM import stores are those the OGID codec decodes from
// the same plies: same Position, same Zobrist hash, so a match imported from
// OGXM and an OGID typed in meet on one position.
func TestOGXMPositionsMatchOGIDCodec(t *testing.T) {
	raw, err := os.ReadFile("../../../testdata/ogid_hedgehog.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Plies []struct {
			OGID string `json:"ogid"`
		} `json:"plies"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	f, err := ogxmparser.ParseFile(ogxmRealFile)
	if err != nil {
		t.Fatal(err)
	}
	g, err := MapOGXM(ogxmRealFile)
	if err != nil {
		t.Fatal(err)
	}
	replays := f.Replay()
	k, checked := 0, 0
	for gi := range f.Games {
		moves := g.Games[gi].Moves
		mi := 0
		for pi := range f.Games[gi].Plies {
			p := &f.Games[gi].Plies[pi]
			if k >= len(fixture.Plies) {
				t.Fatalf("the fixture stops at ply %d", k)
			}
			in := fixture.Plies[k].OGID
			k++
			emitted := (p.IsDiceAction() && !replays[gi].Plies[pi].UnplayedRoll) || p.Action == ogxmparser.ActionDouble
			if !emitted {
				continue
			}
			want, err := domain.DecodeOGID(in)
			if err != nil {
				t.Fatalf("DecodeOGID(%s): %v", in, err)
			}
			got := moves[mi].Position
			mi++
			if *got != want {
				t.Errorf("game %d ply %d: OGXM %+v, OGID %s decodes %+v", gi+1, pi, *got, in, want)
			}
			if engine.ZobristHash(got) != engine.ZobristHash(&want) {
				t.Errorf("game %d ply %d: Zobrist hashes differ from %s", gi+1, pi, in)
			}
			checked++
		}
		if mi != len(moves) {
			t.Errorf("game %d: %d moves mapped, %d aligned", gi+1, len(moves), mi)
		}
	}
	if k != len(fixture.Plies) || checked < 100 {
		t.Fatalf("aligned %d of %d plies, %d positions checked", k, len(fixture.Plies), checked)
	}
}

// The losing anchor is looked up under the opponent, the score, the cube's
// value and the Crawford rule; any of them differing falls back to the MET,
// and each fallback is counted.
func TestOGXMCubeAnchorKeys(t *testing.T) {
	const length = 5
	at := func(cubeLog2 int) *ogid.OGID {
		o := ogid.New()
		o.Score[ogid.Black], o.Score[ogid.White] = 4, 2
		o.CubeLog2 = cubeLog2
		return &o
	}
	an := &ogxmAnalysis{
		block: &ogxmparser.Analysis{Currency: ogxmparser.CubefulMoney, METID: "other"},
		winAnchor: map[ogxmAnchorKey]float64{
			ogxmAnchorKeyAt(at(1), domain.Black, false): 0.9,
			ogxmAnchorKeyAt(at(1), domain.White, false): 0.4,
			ogxmAnchorKeyAt(at(1), domain.White, true):  0.3,
		},
	}
	eq := func(v float64) *float64 { return &v }
	nd := func(player int, o *ogid.OGID, crawford bool, win float64) float64 {
		d := &ogxmparser.CubeDecision{NoDoubleEquity: eq(0.75), DoubleTakeEquity: eq(0.75), DoublePassEquity: eq(win)}
		c := ogxmCubeAnalysis(an, d, o, player, length, crawford)
		if c == nil || c.CubefulDoublePassEquity != 1 {
			t.Fatalf("player %d cube %d crawford %v: %+v", player, o.CubeValue(), crawford, c)
		}
		return c.CubefulNoDoubleEquity
	}
	lose := func(player int, o *ogid.OGID, crawford bool) float64 {
		return engine.GnuBGGetME(4, 2, length, player, o.CubeValue(), 1-player, crawford)
	}
	norm := func(mwc, win, lose float64) float64 { return (2*mwc - win - lose) / (win - lose) }
	for _, c := range []struct {
		name      string
		player    int
		cubeLog2  int
		crawford  bool
		win, want float64
		fallback  bool
	}{
		{"Black, cube 2, White's anchor", domain.Black, 1, false, 0.9, norm(0.75, 0.9, 0.6), false},
		{"Black, cube 2, Crawford anchor", domain.Black, 1, true, 0.9, norm(0.75, 0.9, 0.7), false},
		{"White, cube 2, Black's anchor", domain.White, 1, false, 0.4, norm(0.75, 0.4, 0.1), false},
		{"White, cube 2, no Crawford anchor", domain.White, 1, true, 0.95, norm(0.75, 0.95, lose(domain.White, at(1), true)), true},
		{"Black, cube 1, no anchor", domain.Black, 0, false, 0.95, norm(0.75, 0.95, lose(domain.Black, at(0), false)), true},
	} {
		before := an.metFallbacks
		if got := nd(c.player, at(c.cubeLog2), c.crawford, c.win); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("%s: no double %.6f, want %.6f", c.name, got, c.want)
		}
		if fell := an.metFallbacks > before; fell != c.fallback {
			t.Errorf("%s: MET fallback %v, want %v", c.name, fell, c.fallback)
		}
	}
	if an.metFallbacks != 2 {
		t.Errorf("%d fallbacks counted, want 2", an.metFallbacks)
	}
}
