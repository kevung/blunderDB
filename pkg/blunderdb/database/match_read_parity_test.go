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
