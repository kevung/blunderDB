package database

import (
	"context"
	"reflect"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// A match without a date (NULL, ” or the zero time) sorts by its import
// date, on the wrapper's GetAllMatches exactly as on the contract's List.
func TestMatchReadParity_UndatedSortsByImportDate(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	ins := func(hash string, date any, imported string) int64 {
		res, err := db.db.Exec(`INSERT INTO match (player1_name, player2_name, match_length, match_date, import_date, game_count, match_hash)
			VALUES ('a', 'b', 5, ?, ?, 1, ?)`, date, imported, hash)
		if err != nil {
			t.Fatal(err)
		}
		id, _ := res.LastInsertId()
		return id
	}
	old := ins("h1", "2020-06-01T00:00:00Z", "2023-01-01T00:00:00Z")
	zero := ins("h2", "0001-01-01T00:00:00Z", "2021-01-01T00:00:00Z")
	empty := ins("h3", nil, "2019-01-01T00:00:00Z")
	blank := ins("h5", "", "2018-01-01T00:00:00Z")
	recent := ins("h4", "2022-03-01T00:00:00Z", "2023-01-01T00:00:00Z")
	want := []int64{recent, zero, old, empty, blank}

	all, err := db.GetAllMatches()
	if err != nil {
		t.Fatal(err)
	}
	listed, err := db.ListMatches(storage.MatchListOpts{})
	if err != nil {
		t.Fatal(err)
	}
	ids := func(ms []Match) []int64 {
		var out []int64
		for _, m := range ms {
			out = append(out, m.ID)
		}
		return out
	}
	if !reflect.DeepEqual(ids(all), want) {
		t.Errorf("GetAllMatches order = %v, want %v", ids(all), want)
	}
	if !reflect.DeepEqual(ids(listed), ids(all)) {
		t.Errorf("ListMatches order = %v, GetAllMatches = %v", ids(listed), ids(all))
	}
}

// GetMatchMovePositions reads position.max_cube like the contract's
// MovePositions.
func TestMatchReadParity_MovePositionsMaxCube(t *testing.T) {
	t.Parallel()
	db := newTestDBWithXG(t)
	var matchID int64
	if err := db.db.QueryRow(`SELECT id FROM match LIMIT 1`).Scan(&matchID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.db.Exec(`UPDATE position SET max_cube = 7 WHERE id IN (SELECT position_id FROM move)`); err != nil {
		t.Fatal(err)
	}
	got, err := db.GetMatchMovePositions(matchID)
	if err != nil || len(got) == 0 {
		t.Fatalf("GetMatchMovePositions: %d rows, %v", len(got), err)
	}
	var want []MatchMovePosition
	for mp, err := range db.store.Matches().MovePositions(context.Background(), "", matchID) {
		if err != nil {
			t.Fatal(err)
		}
		want = append(want, *mp)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wrapper and contract differ")
	}
	if got[0].Position.MaxCube != 7 {
		t.Errorf("MaxCube = %d, want 7", got[0].Position.MaxCube)
	}
}

// With no match visited, GetLastVisitedMatch falls back to the match the list
// shows first: an undated match (legacy zero time) by its import date, as the
// contract's LastVisited now sorts it on both backends.
func TestMatchReadParity_LastVisitedFallbackSortsUndatedByImportDate(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	insertMatchRow(t, db, "h1", "2020-06-01T00:00:00Z", "2020-07-01T00:00:00Z")
	zero := insertMatchRow(t, db, "h2", "0001-01-01T00:00:00Z", "2023-01-01T00:00:00Z")
	insertMatchRow(t, db, "h3", "", "2019-01-01T00:00:00Z")

	m, err := db.GetLastVisitedMatch()
	if err != nil {
		t.Fatal(err)
	}
	if m.ID != zero {
		t.Errorf("GetLastVisitedMatch = %d, want the undated match imported last (%d)", m.ID, zero)
	}
	listed, err := db.ListMatches(storage.MatchListOpts{Limit: 1})
	if err != nil || len(listed) != 1 || listed[0].ID != m.ID {
		t.Errorf("ListMatches first = %v (%v), want %d", listed, err, m.ID)
	}
}

// An empty library still answers with the text the GUI reads.
func TestMatchReadParity_LastVisitedEmptyLibrary(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	if _, err := db.GetLastVisitedMatch(); err == nil || err.Error() != "no matches in database" {
		t.Errorf("err = %v, want \"no matches in database\"", err)
	}
}

// A position's provenance is ordered as the match list orders: an undated
// match sorts by its import date, not after every dated one.
func TestMatchReadParity_ProvenanceInListOrder(t *testing.T) {
	t.Parallel()
	db := newTestDBWithXG(t)
	var posID, xgMatch int64
	if err := db.db.QueryRow(`SELECT mv.position_id, g.match_id FROM move mv JOIN game g ON mv.game_id = g.id
		WHERE mv.position_id IS NOT NULL LIMIT 1`).Scan(&posID, &xgMatch); err != nil {
		t.Fatal(err)
	}
	undated := insertMatchRow(t, db, "undated", "0001-01-01T00:00:00Z", "2099-01-01T00:00:00Z")
	res, err := db.db.Exec(`INSERT INTO game (match_id, game_number, initial_score_1, initial_score_2, winner, points_won, move_count)
		VALUES (?, 1, 0, 0, 0, 0, 1)`, undated)
	if err != nil {
		t.Fatal(err)
	}
	gameID, _ := res.LastInsertId()
	if _, err := db.db.Exec(`INSERT INTO move (game_id, move_number, move_type, position_id, player, dice_1, dice_2)
		VALUES (?, 1, 'checker', ?, 0, 3, 1)`, gameID, posID); err != nil {
		t.Fatal(err)
	}

	prov, err := db.GetPositionProvenance(posID)
	if err != nil {
		t.Fatal(err)
	}
	if len(prov) != 2 || prov[0].ID != undated || prov[1].ID != xgMatch {
		var ids []int64
		for _, m := range prov {
			ids = append(ids, m.ID)
		}
		t.Errorf("provenance = %v, want [%d %d]", ids, undated, xgMatch)
	}
}

// GetMovesByGame reads the moves as recorded: no cost of the played move,
// even on an analysed position, so the export reads no analysis.
func TestMatchReadParity_MovesCarryNoPlayedError(t *testing.T) {
	t.Parallel()
	db := newTestDBWithXG(t)
	matches, err := db.GetAllMatches()
	if err != nil {
		t.Fatal(err)
	}
	var games []int64
	for _, m := range matches {
		gs, err := db.GetGamesByMatch(m.ID)
		if err != nil {
			t.Fatal(err)
		}
		for _, g := range gs {
			games = append(games, g.ID)
		}
	}
	moves := 0
	for _, g := range games {
		got, err := db.GetMovesByGame(g)
		if err != nil {
			t.Fatal(err)
		}
		for _, mv := range got {
			moves++
			if mv.ErrorMP != nil {
				t.Fatalf("move %d carries ErrorMP = %d; GetMovesByGame must not score", mv.ID, *mv.ErrorMP)
			}
		}
	}
	if moves == 0 {
		t.Fatal("the fixture has no move")
	}
}
