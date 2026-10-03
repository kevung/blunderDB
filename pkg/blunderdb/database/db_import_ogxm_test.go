package database

import (
	"errors"
	"path/filepath"
	"testing"
)

// TestImportOGXMMatch: a HedgeHog export imports into positions with their
// analyses, and importing it again is a duplicate (the GUI/CLI contract).
// What each position holds is pinned by ingest's TestMapOGXMRealMatch.
func TestImportOGXMMatch(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	fixture := filepath.Join("testdata", "hedgehog-3pt-analysed.ogxm")

	matchID, err := db.ImportOGXMMatch(fixture)
	if err != nil {
		t.Fatalf("ImportOGXMMatch: %v", err)
	}
	if matchID <= 0 {
		t.Fatalf("matchID = %d, want > 0", matchID)
	}
	var positions, analysed int
	if err := db.db.QueryRow("SELECT COUNT(*) FROM position").Scan(&positions); err != nil {
		t.Fatal(err)
	}
	if err := db.db.QueryRow("SELECT COUNT(*) FROM analysis").Scan(&analysed); err != nil {
		t.Fatal(err)
	}
	if positions == 0 || analysed == 0 {
		t.Fatalf("%d positions, %d analyses", positions, analysed)
	}
	if _, err := db.ImportOGXMMatch(fixture); !errors.Is(err, ErrDuplicateMatch) {
		t.Fatalf("re-import error = %v, want ErrDuplicateMatch", err)
	}
}
