package database

// search_displayed_board_test.go
//
// The front sends a board with every search. Outside EDIT mode it is the
// position on screen, all thirty checkers, which must not be read as an
// "at least" structure — that would answer with that position alone, or
// nothing.
//
// These tests send the exact payload loadPositionsByFilters builds
// (positionService.js): the displayed library position as Wails serialises it,
// mirrored when player 2 is on roll, wrapped in the SearchFilters JSON, through
// the real database path.

import (
	"encoding/json"
	"testing"
)

// frontMirror is positionService.js's mirrorPositionForSearch, line for line.
func frontMirror(t *testing.T, raw map[string]any) {
	t.Helper()
	board := raw["board"].(map[string]any)
	points := board["points"].([]any)
	mirrored := make([]any, len(points))
	for i, p := range points {
		pt := p.(map[string]any)
		c := pt["color"].(float64)
		if c != -1 && c != 2 {
			c = 1 - c
		}
		mirrored[len(points)-1-i] = map[string]any{"color": c, "checkers": pt["checkers"]}
	}
	board["points"] = mirrored
	bo := board["bearoff"].([]any)
	board["bearoff"] = []any{bo[1], bo[0]}
	raw["player_on_roll"] = 1 - raw["player_on_roll"].(float64)
	sc := raw["score"].([]any)
	raw["score"] = []any{sc[1], sc[0]}
	cube := raw["cube"].(map[string]any)
	if cube["owner"].(float64) != -1 {
		cube["owner"] = 1 - cube["owner"].(float64)
	}
}

// frontPayload builds the SearchFilters the front sends for `s <moveError>`
// while `displayed` is on screen. clearCheckers reproduces the front's
// out-of-EDIT board: the checkers go, the rest of the position stays.
func frontPayload(t *testing.T, displayed Position, moveError string, clearCheckers bool) SearchFilters {
	t.Helper()
	js, err := json.Marshal(displayed) // what Wails hands the front
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(js, &raw); err != nil {
		t.Fatal(err)
	}
	if clearCheckers {
		points := raw["board"].(map[string]any)["points"].([]any)
		for i := range points {
			points[i] = map[string]any{"checkers": 0.0, "color": -1.0}
		}
	}
	if raw["player_on_roll"].(float64) == 1 {
		frontMirror(t, raw)
	}
	empty := map[string]any{
		"board":          map[string]any{"points": make([]map[string]any, 0), "bearoff": []int{15, 15}},
		"cube":           map[string]any{"owner": -1, "value": 0},
		"dice":           []int{0, 0},
		"score":          []int{0, 0},
		"player_on_roll": 0, "decision_type": 0, "has_jacoby": 0, "has_beaver": 0,
	}
	pts := empty["board"].(map[string]any)
	pl := make([]map[string]any, 26)
	for i := range pl {
		pl[i] = map[string]any{"checkers": 0, "color": -1}
	}
	pts["points"] = pl
	body, err := json.Marshal(map[string]any{"filter": raw, "excludeFilter": empty, "moveErrorFilter": moveError})
	if err != nil {
		t.Fatal(err)
	}
	var f SearchFilters
	if err := json.Unmarshal(body, &f); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestSearch_DisplayedBoardOutsideEdit(t *testing.T) {
	db := NewDatabase()
	if err := db.SetupDatabase(":memory:"); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	defer db.Close()
	for _, f := range []string{"testdata/HsbtMarseille_main_ronde4_LamourDeCaslouGildas_UngerKevin_7p.xg", "testdata/test.xg"} {
		if _, err := db.ImportXGMatch(f); err != nil {
			t.Fatalf("import %s: %v", f, err)
		}
	}

	const query = "E>80"
	var noBoard SearchFilters
	if err := json.Unmarshal([]byte(`{"moveErrorFilter":"`+query+`"}`), &noBoard); err != nil {
		t.Fatal(err)
	}
	want, err := db.LoadPositionIDsByFilters(noBoard)
	if err != nil {
		t.Fatal(err)
	}
	if len(want) < 5 {
		t.Fatalf("fixture too thin: %d positions match %s with no board", len(want), query)
	}
	t.Logf("%s with no board: %d positions", query, len(want))

	// A displayed library position among the matches, and the same position as
	// the board shows it with player 2 on roll: the front mirrors that one back
	// before sending it, which is the path the mirror must not break.
	ps, err := db.LoadPositionsByIDs([]int64{want[0]})
	if err != nil || len(ps) != 1 {
		t.Fatalf("LoadPositionsByIDs(%d): %v", want[0], err)
	}
	stored := ps[0]
	if stored.PlayerOnRoll != 0 {
		t.Fatalf("expected a stored position with player 1 on roll, got %d", stored.PlayerOnRoll)
	}
	js, _ := json.Marshal(stored)
	var raw map[string]any
	_ = json.Unmarshal(js, &raw)
	frontMirror(t, raw)
	js, _ = json.Marshal(raw)
	var shownForPlayer2 Position
	if err := json.Unmarshal(js, &shownForPlayer2); err != nil {
		t.Fatal(err)
	}
	displayed := map[int]*Position{0: &stored, 1: &shownForPlayer2}
	for onRoll, pos := range displayed {
		before, err := db.LoadPositionIDsByFilters(frontPayload(t, *pos, query, false))
		if err != nil {
			t.Fatal(err)
		}
		after, err := db.LoadPositionIDsByFilters(frontPayload(t, *pos, query, true))
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("displayed position %d (player_on_roll=%d): full board → %d, board without checkers → %d", pos.ID, onRoll, len(before), len(after))
		// The backend reads a board with checkers as an "at least" structure: a
		// full position matches itself and little else. That is right for a
		// drawn query, and why the front must not send the position on screen.
		if len(before) >= len(want) {
			t.Errorf("player_on_roll=%d: a full board should restrict %s (%d), got %d", onRoll, query, len(want), len(before))
		}
		if len(after) != len(want) {
			t.Errorf("player_on_roll=%d: the out-of-EDIT board must not restrict %s: got %d, want %d", onRoll, query, len(after), len(want))
		}
	}
}
