package database

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCanonicalHashDuplicateImport tests that importing the same match from
// different formats results in a single match row in the database.
func TestCanonicalHashDuplicateImport(t *testing.T) {
	t.Parallel()
	xgFile := filepath.Join("testdata", "test.xg")
	sgfFile := filepath.Join("testdata", "test.sgf")

	if _, err := os.Stat(xgFile); err != nil {
		t.Fatalf("test.xg not found: %v", err)
	}
	if _, err := os.Stat(sgfFile); err != nil {
		t.Fatalf("test.sgf not found: %v", err)
	}

	// Create a temporary database
	tmpDir := tempDir(t)
	dbPath := filepath.Join(tmpDir, "test.db")

	db := NewDatabase()
	if err := db.SetupDatabase(dbPath); err != nil {
		t.Fatalf("Failed to setup database: %v", err)
	}
	closeOnCleanup(t, db)

	// Import XG match first
	matchID1, err := db.ImportXGMatch(xgFile)
	if err != nil {
		t.Fatalf("Failed to import XG match: %v", err)
	}
	t.Logf("XG match imported with ID: %d", matchID1)

	// Import SGF match - should detect canonical duplicate and reuse same match ID
	matchID2, err := db.ImportGnuBGMatch(sgfFile)
	if err != nil {
		t.Fatalf("Failed to import SGF match (expected canonical duplicate merge): %v", err)
	}
	t.Logf("SGF match imported with ID: %d", matchID2)

	if matchID1 != matchID2 {
		t.Errorf("Expected same match ID for canonical duplicates: XG=%d, SGF=%d", matchID1, matchID2)
	}

	// Count matches in database - should be exactly 1
	var matchCount int
	err = db.db.QueryRow(`SELECT COUNT(*) FROM match`).Scan(&matchCount)
	if err != nil {
		t.Fatalf("Failed to count matches: %v", err)
	}

	if matchCount != 1 {
		t.Errorf("Expected 1 match in database, got %d", matchCount)
	}
}

// TestCanonicalHashTripleImport tests XG + SGF + MAT import produces one match row
func TestCanonicalHashTripleImport(t *testing.T) {
	t.Parallel()
	xgFile := filepath.Join("testdata", "test.xg")
	sgfFile := filepath.Join("testdata", "test.sgf")
	matFile := filepath.Join("testdata", "test.mat")

	for _, f := range []string{xgFile, sgfFile, matFile} {
		if _, err := os.Stat(f); err != nil {
			t.Fatalf("%s not found: %v", f, err)
		}
	}

	tmpDir := tempDir(t)
	dbPath := filepath.Join(tmpDir, "test.db")

	db := NewDatabase()
	if err := db.SetupDatabase(dbPath); err != nil {
		t.Fatalf("Failed to setup database: %v", err)
	}
	closeOnCleanup(t, db)

	// Import XG first
	matchID, err := db.ImportXGMatch(xgFile)
	if err != nil {
		t.Fatalf("Failed to import XG match: %v", err)
	}
	t.Logf("XG match imported with ID: %d", matchID)

	// Import SGF - canonical duplicate
	matchID2, err := db.ImportGnuBGMatch(sgfFile)
	if err != nil {
		t.Fatalf("Failed to import SGF match: %v", err)
	}
	if matchID2 != matchID {
		t.Errorf("SGF should reuse match ID %d, got %d", matchID, matchID2)
	}

	// Import MAT - canonical duplicate
	matchID3, err := db.ImportGnuBGMatch(matFile)
	if err != nil {
		t.Fatalf("Failed to import MAT match: %v", err)
	}
	if matchID3 != matchID {
		t.Errorf("MAT should reuse match ID %d, got %d", matchID, matchID3)
	}

	// Verify only 1 match row exists
	var matchCount int
	err = db.db.QueryRow(`SELECT COUNT(*) FROM match`).Scan(&matchCount)
	if err != nil {
		t.Fatalf("Failed to count matches: %v", err)
	}
	if matchCount != 1 {
		t.Errorf("Expected 1 match in database after importing 3 formats, got %d", matchCount)
	}

	t.Logf("Successfully: 3 formats -> 1 match row (ID=%d)", matchID)
}
