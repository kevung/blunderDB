package database

import (
	"errors"
	"path/filepath"
	"testing"
)

// TestImportBGFMatch is the database-level characterization test for the BGF
// match import now that it delegates to the ingest pipeline: a fixture imports
// into positions, and re-importing the same file is reported as a duplicate
// (the GUI/CLI contract). The detailed position/analysis correctness of the
// ingest BGF path is covered by the server's TestImportBGFEndToEnd.
func TestImportBGFMatch(t *testing.T) {
	db := newTestDB(t)
	fixture := filepath.Join("testdata", "TachiAI_V_player_Nov_2__2025__16_55.bgf")

	matchID, err := db.ImportBGFMatch(fixture)
	if err != nil {
		t.Fatalf("ImportBGFMatch: %v", err)
	}
	if matchID <= 0 {
		t.Fatalf("matchID = %d, want > 0", matchID)
	}

	var positions int
	if err := db.db.QueryRow("SELECT COUNT(*) FROM position").Scan(&positions); err != nil {
		t.Fatalf("count positions: %v", err)
	}
	if positions == 0 {
		t.Fatal("no positions imported from BGF fixture")
	}

	// Re-importing the same file must be reported as a duplicate, not silently
	// added again.
	if _, err := db.ImportBGFMatch(fixture); !errors.Is(err, ErrDuplicateMatch) {
		t.Fatalf("re-import error = %v, want ErrDuplicateMatch", err)
	}
}
