package ingest

import (
	"context"
	"strings"
	"testing"

	"github.com/kevung/gnubgparser"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// TestRenderMATRoundTrip builds a match graph, renders it to .mat, and re-parses
// it with gnubgparser — the format gate. It exercises checker moves for both
// players, a separate Double/Take, and a game-winning result.
func TestRenderMATRoundTrip(t *testing.T) {
	m := &domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7}
	games := []*domain.Game{
		{ID: 1, GameNumber: 1, InitialScore: [2]int32{0, 0}, Winner: 0, PointsWon: 2},
	}
	moves := map[int64][]*domain.Move{
		1: {
			{Player: 1, MoveType: "checker", Dice: [2]int32{3, 1}, CheckerMove: "8/5 6/5"},
			{Player: -1, MoveType: "checker", Dice: [2]int32{6, 4}, CheckerMove: "24/18 13/9"},
			{Player: 1, MoveType: "cube", CubeAction: "Double"},
			{Player: -1, MoveType: "cube", CubeAction: "Take"},
			{Player: 1, MoveType: "checker", Dice: [2]int32{5, 2}, CheckerMove: "13/8 13/11"},
			{Player: -1, MoveType: "checker", Dice: [2]int32{2, 1}, CheckerMove: "24/23 6/4"},
		},
	}

	out := RenderMAT(m, games, moves)
	if !strings.Contains(out, "7 point match") {
		t.Fatalf("missing match header:\n%s", out)
	}

	parsed, err := gnubgparser.ParseMAT(strings.NewReader(out))
	if err != nil {
		t.Fatalf("re-parse failed: %v\n---\n%s", err, out)
	}
	if len(parsed.Games) != 1 {
		t.Fatalf("got %d games, want 1\n%s", len(parsed.Games), out)
	}
	g := parsed.Games[0]
	if g.GameNumber != 1 || g.Score != [2]int{0, 0} {
		t.Errorf("game header wrong: number=%d score=%v", g.GameNumber, g.Score)
	}
	if g.Points != 2 {
		t.Errorf("points = %d, want 2", g.Points)
	}

	var checker, doubles, takes int
	var doubleVal int
	for _, mv := range g.Moves {
		switch mv.Type {
		case gnubgparser.MoveTypeNormal:
			checker++
		case gnubgparser.MoveTypeDouble:
			doubles++
			doubleVal = mv.CubeValue
		case gnubgparser.MoveTypeTake:
			takes++
		}
	}
	if checker != 4 {
		t.Errorf("checker moves = %d, want 4\n%s", checker, out)
	}
	if doubles != 1 || takes != 1 {
		t.Errorf("cube moves: doubles=%d takes=%d, want 1/1\n%s", doubles, takes, out)
	}
	if doubleVal != 2 {
		t.Errorf("doubled cube value = %d, want 2", doubleVal)
	}
}

// TestRenderMATCombinedCubeAndMatchWin: a gnubg-style combined "Double/Take" is
// split into two cells, and a match-ending win gets " and the match".
func TestRenderMATCombinedCubeAndMatchWin(t *testing.T) {
	m := &domain.Match{Player1Name: "A", Player2Name: "B", MatchLength: 3}
	games := []*domain.Game{
		{ID: 1, GameNumber: 1, InitialScore: [2]int32{1, 0}, Winner: 0, PointsWon: 2},
	}
	moves := map[int64][]*domain.Move{
		1: {
			{Player: -1, MoveType: "checker", Dice: [2]int32{3, 1}, CheckerMove: "8/5 6/5"},
			{Player: 1, MoveType: "cube", CubeAction: "Double/Take"},
			{Player: 1, MoveType: "checker", Dice: [2]int32{6, 6}, CheckerMove: "13/7 13/7 24/18 24/18"},
		},
	}
	out := RenderMAT(m, games, moves)
	if !strings.Contains(out, "and the match") {
		t.Errorf("winner reaching match length should say 'and the match':\n%s", out)
	}

	parsed, err := gnubgparser.ParseMAT(strings.NewReader(out))
	if err != nil {
		t.Fatalf("re-parse: %v\n%s", err, out)
	}
	var doubles, takes int
	for _, mv := range parsed.Games[0].Moves {
		if mv.Type == gnubgparser.MoveTypeDouble {
			doubles++
		}
		if mv.Type == gnubgparser.MoveTypeTake {
			takes++
		}
	}
	if doubles != 1 || takes != 1 {
		t.Errorf("combined cube split wrong: doubles=%d takes=%d\n%s", doubles, takes, out)
	}
}

// TestRenderMATFromStoredMatch is the strongest round-trip: import a real .mat
// into storage, read it back, render, and re-parse. The re-rendered transcript
// must reproduce the same game structure (count, scores, points) as parsing the
// original file directly.
func TestRenderMATFromStoredMatch(t *testing.T) {
	ctx := context.Background()
	s, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	defer s.Close()

	graph, err := MapGnuBG("../../../testdata/test.mat")
	if err != nil {
		t.Fatalf("MapGnuBG: %v", err)
	}
	res := writeGraph(t, s, graph)

	m, games, moves, err := ReadMatchForMAT(ctx, s, "", res.MatchID)
	if err != nil {
		t.Fatalf("ReadMatchForMAT: %v", err)
	}
	out := RenderMAT(m, games, moves)

	rt, err := gnubgparser.ParseMAT(strings.NewReader(out))
	if err != nil {
		t.Fatalf("re-parse rendered .mat: %v\n%s", err, out)
	}
	orig, err := gnubgparser.ParseMATFile("../../../testdata/test.mat")
	if err != nil {
		t.Fatalf("parse original: %v", err)
	}

	if len(rt.Games) != len(orig.Games) {
		t.Fatalf("game count: rendered %d vs original %d", len(rt.Games), len(orig.Games))
	}
	for i := range orig.Games {
		og, rg := orig.Games[i], rt.Games[i]
		if rg.Score != og.Score {
			t.Errorf("game %d score: rendered %v vs original %v", i+1, rg.Score, og.Score)
		}
		if rg.Points != og.Points {
			t.Errorf("game %d points: rendered %d vs original %d", i+1, rg.Points, og.Points)
		}
		// Checker-move count must be preserved (notation may differ: bar/off vs 25/0).
		oc, rc := countChecker(og.Moves), countChecker(rg.Moves)
		if rc != oc {
			t.Errorf("game %d checker moves: rendered %d vs original %d", i+1, rc, oc)
		}
	}
}

func countChecker(moves []gnubgparser.MoveRecord) int {
	n := 0
	for _, m := range moves {
		if m.Type == gnubgparser.MoveTypeNormal {
			n++
		}
	}
	return n
}
