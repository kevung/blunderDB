package database

import (
	"path/filepath"
	"testing"
	"time"
)

// Visiting B then A makes A the last visited match, whatever the order the
// matches were imported in.
func TestGetLastVisitedMatchFollowsVisitOrder(t *testing.T) {
	t.Parallel()
	d := NewDatabase()
	if err := d.SetupDatabase(filepath.Join(tempDir(t), "visit.db")); err != nil {
		t.Fatalf("open: %v", err)
	}
	closeOnCleanup(t, d)
	var ids [2]int64 // A, B
	for i, name := range []string{"A", "B"} {
		res, err := d.db.Exec(`INSERT INTO match (player1_name, player2_name, event, location, round, match_length, match_date, import_date, file_path, game_count, match_hash)
		 VALUES (?, 'Adversaire', '', '', '', 7, ?, ?, '', 0, ?)`, name, time.Now(), time.Now(), "hash-"+name)
		if err != nil {
			t.Fatalf("insert match %s: %v", name, err)
		}
		if ids[i], err = res.LastInsertId(); err != nil {
			t.Fatalf("match id: %v", err)
		}
	}
	a, b := ids[0], ids[1]
	visit := func(id int64) {
		t.Helper()
		if err := d.SaveLastVisitedPosition(id, 2); err != nil {
			t.Fatalf("SaveLastVisitedPosition(%d): %v", id, err)
		}
		time.Sleep(3 * time.Millisecond) // the instant has millisecond resolution
	}
	visit(b)
	visit(a)
	if m, err := d.GetLastVisitedMatch(); err != nil || m.ID != a {
		t.Fatalf("after visiting B then A: %+v, %v; want A (%d)", m, err, a)
	}
	visit(b)
	if m, err := d.GetLastVisitedMatch(); err != nil || m.ID != b {
		t.Errorf("after visiting A then B: %+v, %v; want B (%d)", m, err, b)
	}
}
