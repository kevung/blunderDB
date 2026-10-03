package ingest

import (
	"math"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
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

	// Black's cube decision before rolling 4-6, against the reference's
	// normalised equities (-0.0111, -0.4485, 1.0000). Its MET is not
	// blunderDB's, hence the tolerance.
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
	for _, c := range []struct {
		name      string
		got, want float64
	}{
		{"no double", cube.CubefulNoDoubleEquity, -0.0111},
		{"double/take", cube.CubefulDoubleTakeEquity, -0.4485},
		{"double/pass", cube.CubefulDoublePassEquity, 1.0},
	} {
		if math.Abs(c.got-c.want) > 0.03 {
			t.Errorf("%s: %.4f, reference %.4f", c.name, c.got, c.want)
		}
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
