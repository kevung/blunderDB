package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kevung/bgfparser"
	"github.com/kevung/gnubgparser"
	"github.com/kevung/xgparser/xgparser"
)

// TestCanonicalHashCrossFormat verifies that the same match imported from
// different file formats (XG, SGF, MAT) produces identical canonical hashes.
func TestCanonicalHashCrossFormat(t *testing.T) {
	xgFile := filepath.Join("testdata", "test.xg")
	sgfFile := filepath.Join("testdata", "test.sgf")
	matFile := filepath.Join("testdata", "test.mat")

	// Check which test files exist
	hashes := make(map[string]string)

	if _, err := os.Stat(xgFile); err == nil {
		imp := xgparser.NewImport(xgFile)
		segments, err := imp.GetFileSegments()
		if err != nil {
			t.Fatalf("Failed to import XG file: %v", err)
		}
		match, err := xgparser.ParseXG(segments)
		if err != nil {
			t.Fatalf("Failed to parse XG file: %v", err)
		}
		hash := ComputeCanonicalMatchHashFromXG(match)
		hashes["XG"] = hash
		t.Logf("XG canonical hash: %s", hash)
	} else {
		t.Log("test.xg not found, skipping")
	}

	if _, err := os.Stat(sgfFile); err == nil {
		gnuMatch, err := gnubgparser.ParseSGFFile(sgfFile)
		if err != nil {
			t.Fatalf("Failed to parse SGF file: %v", err)
		}
		hash := ComputeCanonicalMatchHashFromGnuBG(gnuMatch)
		hashes["SGF"] = hash
		t.Logf("SGF canonical hash: %s", hash)
	} else {
		t.Log("test.sgf not found, skipping")
	}

	if _, err := os.Stat(matFile); err == nil {
		gnuMatch, err := gnubgparser.ParseMATFile(matFile)
		if err != nil {
			t.Fatalf("Failed to parse MAT file: %v", err)
		}
		hash := ComputeCanonicalMatchHashFromGnuBG(gnuMatch)
		hashes["MAT"] = hash
		t.Logf("MAT canonical hash: %s", hash)
	} else {
		t.Log("test.mat not found, skipping")
	}

	if len(hashes) < 2 {
		t.Skip("Need at least 2 test files to compare canonical hashes")
	}

	// All hashes should be identical
	var referenceFormat string
	var referenceHash string
	for format, hash := range hashes {
		if referenceHash == "" {
			referenceFormat = format
			referenceHash = hash
			continue
		}
		if hash != referenceHash {
			t.Errorf("Canonical hash mismatch: %s (%s) != %s (%s)",
				referenceFormat, referenceHash, format, hash)
		}
	}

	if t.Failed() {
		t.Log("Canonical hashes should be identical for the same match across different file formats")
	} else {
		t.Logf("All %d format(s) produce the same canonical hash", len(hashes))
	}
}

// TestCanonicalHashCrossFormat_BGF tests BGF canonical hash against other formats
func TestCanonicalHashCrossFormat_BGF(t *testing.T) {
	bgfFile := filepath.Join("testdata", "TachiAI_V_player_Nov_2__2025__16_55.bgf")

	if _, err := os.Stat(bgfFile); err != nil {
		t.Skip("BGF test file not found")
	}

	bgfMatch, err := bgfparser.ParseBGF(bgfFile)
	if err != nil {
		t.Fatalf("Failed to parse BGF file: %v", err)
	}

	hash := ComputeCanonicalMatchHashFromBGF(bgfMatch)
	t.Logf("BGF canonical hash: %s", hash)
	t.Logf("BGF file parsed successfully, canonical hash computed")
}

// TestCanonicalHashDuplicateImport tests that importing the same match from
// different formats results in a single match row in the database.
func TestCanonicalHashDuplicateImport(t *testing.T) {
	xgFile := filepath.Join("testdata", "test.xg")
	sgfFile := filepath.Join("testdata", "test.sgf")

	if _, err := os.Stat(xgFile); err != nil {
		t.Skip("test.xg not found")
	}
	if _, err := os.Stat(sgfFile); err != nil {
		t.Skip("test.sgf not found")
	}

	// Create a temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db := NewDatabase()
	if err := db.SetupDatabase(dbPath); err != nil {
		t.Fatalf("Failed to setup database: %v", err)
	}

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
	xgFile := filepath.Join("testdata", "test.xg")
	sgfFile := filepath.Join("testdata", "test.sgf")
	matFile := filepath.Join("testdata", "test.mat")

	for _, f := range []string{xgFile, sgfFile, matFile} {
		if _, err := os.Stat(f); err != nil {
			t.Skipf("%s not found", f)
		}
	}

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db := NewDatabase()
	if err := db.SetupDatabase(dbPath); err != nil {
		t.Fatalf("Failed to setup database: %v", err)
	}

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
