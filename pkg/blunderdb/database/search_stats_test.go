package database

import (
	"path/filepath"
	"testing"
)

// TestEnsureSearchStatsOnOpen verifies OpenDatabase populates query-planner
// statistics (sqlite_stat1) for a database that has none yet, so non-selective
// searches get a good plan instead of a temp-B-tree sort. ANALYZE is
// results-neutral, so search correctness is covered by the existing search
// tests; this only asserts the stats are created.
func TestEnsureSearchStatsOnOpen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stats.db")

	db := NewDatabase()
	if err := db.SetupDatabase(path); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	if _, err := db.ImportXGMatch(filepath.Join("testdata", "test.xg")); err != nil {
		db.Close()
		t.Fatalf("ImportXGMatch: %v", err)
	}
	db.Close()

	// Reopen: ensureSearchStats should run ANALYZE because the freshly-created
	// file has no sqlite_stat1 rows yet.
	db2 := NewDatabase()
	if err := db2.OpenDatabase(path); err != nil {
		t.Fatalf("OpenDatabase: %v", err)
	}
	defer db2.Close()

	var n int
	if err := db2.db.QueryRow(`SELECT count(*) FROM sqlite_stat1`).Scan(&n); err != nil {
		t.Fatalf("query sqlite_stat1: %v", err)
	}
	if n == 0 {
		t.Fatal("OpenDatabase did not populate sqlite_stat1 (ANALYZE missing)")
	}
}
