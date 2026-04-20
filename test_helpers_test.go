package main

import (
	"path/filepath"
	"testing"
)

// newTestDB creates a file-backed database in t.TempDir with the current schema.
// Cleanup is registered automatically via t.Cleanup.
func newTestDB(t *testing.T) *Database {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db := NewDatabase()
	if err := db.SetupDatabase(dbPath); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	t.Cleanup(func() {
		if db.db != nil {
			db.db.Close()
		}
	})
	return db
}

// newTestDBWithXG creates a file-backed database and imports testdata/test.xg.
func newTestDBWithXG(t *testing.T) *Database {
	t.Helper()
	db := newTestDB(t)
	if _, err := db.ImportXGMatch(filepath.Join("testdata", "test.xg")); err != nil {
		t.Fatalf("ImportXGMatch: %v", err)
	}
	return db
}

// newTestDBWithSGF creates a file-backed database and imports testdata/test.sgf.
func newTestDBWithSGF(t *testing.T) *Database {
	t.Helper()
	db := newTestDB(t)
	if _, err := db.ImportGnuBGMatch(filepath.Join("testdata", "test.sgf")); err != nil {
		t.Fatalf("ImportGnuBGMatch: %v", err)
	}
	return db
}

// importTestMatch imports testdata/test.sgf and returns the match ID.
func importTestMatch(t *testing.T, db *Database) int64 {
	t.Helper()
	matchID, err := db.ImportGnuBGMatch(filepath.Join("testdata", "test.sgf"))
	if err != nil {
		t.Fatalf("ImportGnuBGMatch: %v", err)
	}
	return matchID
}

// getPositionIDs returns position IDs from the database (up to limit).
func getPositionIDs(t *testing.T, db *Database, limit int) []int64 {
	t.Helper()
	rows, err := db.db.Query(`SELECT id FROM position ORDER BY id LIMIT ?`, limit)
	if err != nil {
		t.Fatalf("query positions: %v", err)
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan: %v", err)
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		t.Fatal("no positions in database")
	}
	return ids
}

// positionCount returns the number of positions in the database.
func positionCount(t *testing.T, db *Database) int {
	t.Helper()
	var count int
	if err := db.db.QueryRow("SELECT COUNT(*) FROM position").Scan(&count); err != nil {
		t.Fatalf("positionCount: %v", err)
	}
	return count
}

// matchCount returns the number of matches in the database.
func matchCount(t *testing.T, db *Database) int {
	t.Helper()
	var count int
	if err := db.db.QueryRow("SELECT COUNT(*) FROM match").Scan(&count); err != nil {
		t.Fatalf("matchCount: %v", err)
	}
	return count
}
