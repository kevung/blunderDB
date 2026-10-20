package database

import (
	"database/sql"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func insertMatchRow(t *testing.T, db *Database, hash string, date any, imported string) int64 {
	t.Helper()
	res, err := db.db.Exec(`INSERT INTO match (player1_name, player2_name, match_length, match_date, import_date, game_count, match_hash, event, location, round, file_path)
		VALUES ('a', 'b', 5, ?, ?, 1, ?, '', '', '', '')`, date, imported, hash)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	return id
}

// Every legacy text form of the zero time sorts as undated, by import date.
func TestUndatedMatch_EveryZeroTimeFormSortsByImportDate(t *testing.T) {
	db := newTestDB(t)
	dated := insertMatchRow(t, db, "h1", "2020-06-01 00:00:00 +0000 UTC", "2020-07-01T00:00:00Z")
	goString := insertMatchRow(t, db, "h2", "0001-01-01 00:00:00 +0000 UTC", "2023-01-01T00:00:00Z")
	driver := insertMatchRow(t, db, "h3", "0001-01-01 00:00:00+00:00", "2022-01-01T00:00:00Z")
	listed, err := db.ListMatches(storage.MatchListOpts{})
	if err != nil {
		t.Fatal(err)
	}
	var got []int64
	for _, m := range listed {
		got = append(got, m.ID)
	}
	want := []int64{goString, driver, dated}
	if len(got) != 3 || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("order = %v, want %v", got, want)
	}
}

// An unreadable date reads as absent instead of failing the whole list.
func TestUndatedMatch_UnreadableDateDoesNotFailTheList(t *testing.T) {
	db := newTestDB(t)
	insertMatchRow(t, db, "h1", "2020-06-01T00:00:00Z", "2020-07-01T00:00:00Z")
	bad := insertMatchRow(t, db, "h2", "17/05/2025", "2023-01-01T00:00:00Z")
	listed, err := db.ListMatches(storage.MatchListOpts{})
	if err != nil {
		t.Fatalf("list with an unreadable date: %v", err)
	}
	if len(listed) != 2 {
		t.Fatalf("listed %d matches, want 2", len(listed))
	}
	for _, m := range listed {
		if m.ID == bad && !m.MatchDate.IsZero() {
			t.Errorf("unreadable date read as %v, want absent", m.MatchDate)
		}
	}
	if _, err := db.GetAllMatches(); err != nil {
		t.Fatalf("GetAllMatches: %v", err)
	}
	if _, err := db.GetLastVisitedMatch(); err != nil {
		t.Fatalf("GetLastVisitedMatch: %v", err)
	}
}

// Saving an undated match, whether the client sends ” or echoes the zero
// time back, keeps it undated: NULL, sorted by its import date.
func TestUndatedMatch_EditKeepsItUndated(t *testing.T) {
	db := newTestDB(t)
	older := insertMatchRow(t, db, "h1", "2020-06-01T00:00:00Z", "2020-07-01T00:00:00Z")
	echoed := insertMatchRow(t, db, "h2", nil, "2023-01-01T00:00:00Z")
	blank := insertMatchRow(t, db, "h3", nil, "2022-01-01T00:00:00Z")
	if err := db.UpdateMatch(echoed, "Renamed", "b", "0001-01-01"); err != nil {
		t.Fatal(err)
	}
	if err := db.UpdateMatch(blank, "Renamed", "b", ""); err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{echoed, blank} {
		var date sql.NullString
		if err := db.db.QueryRow(`SELECT match_date FROM match WHERE id = ?`, id).Scan(&date); err != nil {
			t.Fatal(err)
		}
		if date.Valid {
			t.Errorf("match %d stored %q after an edit, want NULL", id, date.String)
		}
	}
	listed, err := db.ListMatches(storage.MatchListOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 3 || listed[0].ID != echoed || listed[1].ID != blank || listed[2].ID != older {
		t.Fatalf("order after edit = %+v", listed)
	}
}
