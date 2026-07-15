package ingest

import (
	"context"
	"strings"
	"testing"
	"time"

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

// TestSuggestMATFilename covers the default .mat filename helper: CamelCased
// player names (accents kept, no spaces), the Np/unlimited length segment, date
// fallback, forbidden-character stripping, and empty-name defaults.
func TestSuggestMATFilename(t *testing.T) {
	d := func(s string) time.Time {
		tm, err := time.Parse("2006-01-02", s)
		if err != nil {
			t.Fatalf("parse date %q: %v", s, err)
		}
		return tm
	}
	tests := []struct {
		name string
		m    *domain.Match
		want string
	}{
		{
			name: "normal match with accents and spaces",
			m:    &domain.Match{Player1Name: "Kévin Unger", Player2Name: "Joe Smith", MatchDate: d("2024-01-15"), MatchLength: 7},
			want: "KévinUnger_JoeSmith_2024-01-15_7p.mat",
		},
		{
			name: "money game (length 0) renders unlimited",
			m:    &domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchDate: d("2024-03-02"), MatchLength: 0},
			want: "Alice_Bob_2024-03-02_unlimited.mat",
		},
		{
			name: "money game (length -1) renders unlimited",
			m:    &domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchDate: d("2024-03-02"), MatchLength: -1},
			want: "Alice_Bob_2024-03-02_unlimited.mat",
		},
		{
			name: "empty names fall back to Player1/Player2",
			m:    &domain.Match{MatchDate: d("2024-05-09"), MatchLength: 5},
			want: "Player1_Player2_2024-05-09_5p.mat",
		},
		{
			name: "forbidden characters are stripped",
			m:    &domain.Match{Player1Name: "a/b:c", Player2Name: "x?y*z", MatchDate: d("2024-06-01"), MatchLength: 3},
			want: "Abc_Xyz_2024-06-01_3p.mat",
		},
		{
			name: "no date at all omits the date segment",
			m:    &domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7},
			want: "Alice_Bob_7p.mat",
		},
		{
			name: "falls back to import date when match date is zero",
			m:    &domain.Match{Player1Name: "Alice", Player2Name: "Bob", ImportDate: d("2024-07-04"), MatchLength: 7},
			want: "Alice_Bob_2024-07-04_7p.mat",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SuggestMATFilename(tt.m); got != tt.want {
				t.Errorf("SuggestMATFilename() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestRenderMATMoneyGame: a money session (MatchLength 0 or the Unlimited
// sentinel) renders a "0 point match" header — the gnubg/Jellyfish money
// convention — never "-1 point match", and round-trips through gnubgparser as a
// length-0 match. Guards the money-game header fix (both RenderMAT's clamp and
// gnubgparser accepting a 0-length header).
func TestRenderMATMoneyGame(t *testing.T) {
	for _, ml := range []int32{0, domain.Unlimited} {
		m := &domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: ml}
		games := []*domain.Game{
			{ID: 1, GameNumber: 1, InitialScore: [2]int32{0, 0}, Winner: 0, PointsWon: 1},
		}
		moves := map[int64][]*domain.Move{
			1: {
				{Player: 1, MoveType: "checker", Dice: [2]int32{3, 1}, CheckerMove: "8/5 6/5"},
				{Player: -1, MoveType: "checker", Dice: [2]int32{6, 4}, CheckerMove: "24/18 13/9"},
			},
		}
		out := RenderMAT(m, games, moves)
		if !strings.Contains(out, "0 point match") {
			t.Fatalf("MatchLength %d: expected '0 point match' header, got:\n%s", ml, out)
		}
		if strings.Contains(out, "-1 point match") {
			t.Fatalf("MatchLength %d: header must not be '-1 point match'", ml)
		}
		parsed, err := gnubgparser.ParseMAT(strings.NewReader(out))
		if err != nil {
			t.Fatalf("MatchLength %d: money game must round-trip, got: %v\n%s", ml, err, out)
		}
		if parsed.Metadata.MatchLength != 0 {
			t.Errorf("MatchLength %d: parsed length = %d, want 0", ml, parsed.Metadata.MatchLength)
		}
	}
}

// TestRenderMATBlanksCannotMove: a dance stored as the display marker
// "Cannot Move" must render as an empty cell (dice only), never the literal
// "Cannot Move" — which gnubg rejects as move notation.
func TestRenderMATBlanksCannotMove(t *testing.T) {
	m := &domain.Match{Player1Name: "A", Player2Name: "B", MatchLength: 7}
	games := []*domain.Game{{ID: 1, GameNumber: 1, InitialScore: [2]int32{0, 0}}}
	moves := map[int64][]*domain.Move{
		1: {
			{Player: 1, MoveType: "checker", Dice: [2]int32{3, 1}, CheckerMove: "8/5 6/5"},
			{Player: -1, MoveType: "checker", Dice: [2]int32{6, 4}, CheckerMove: "Cannot Move"},
		},
	}
	out := RenderMAT(m, games, moves)
	if strings.Contains(out, "Cannot Move") {
		t.Fatalf("output must not contain the literal \"Cannot Move\":\n%s", out)
	}
	if !strings.Contains(out, "64:") {
		t.Fatalf("the danced roll should still show its dice \"64:\":\n%s", out)
	}
}
