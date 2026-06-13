package database

import (
	"os"
	"path/filepath"
	"testing"
)

// TestImportBGFPosition characterizes the single-position BGF import now that it
// delegates to the ingest pipeline: importing a BGBlitz text position file
// yields a stored position, and importing the same content again (via the
// clipboard/text path) deduplicates to the same position id.
func TestImportBGFPosition(t *testing.T) {
	db := newTestDB(t)
	fixture := filepath.Join("testdata", "bgf_positions", "01_checkerPosition_FR.txt")

	id, err := db.ImportBGFPosition(fixture)
	if err != nil {
		t.Fatalf("ImportBGFPosition: %v", err)
	}
	if id <= 0 {
		t.Fatalf("position id = %d, want > 0", id)
	}

	content, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	id2, err := db.ImportBGFPositionFromText(string(content))
	if err != nil {
		t.Fatalf("ImportBGFPositionFromText: %v", err)
	}
	if id2 != id {
		t.Errorf("re-import of same position via text: id = %d, want %d (should dedup)", id2, id)
	}
}
