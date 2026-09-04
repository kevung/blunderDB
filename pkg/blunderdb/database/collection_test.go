package database

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateCollection(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)

	id, err := db.CreateCollection("Test Collection", "A description")
	if err != nil {
		t.Fatalf("CreateCollection: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected positive ID, got %d", id)
	}

	col, err := db.GetCollectionByID(id)
	if err != nil {
		t.Fatalf("GetCollectionByID: %v", err)
	}
	if col.Name != "Test Collection" {
		t.Errorf("name = %q, want %q", col.Name, "Test Collection")
	}
	if col.Description != "A description" {
		t.Errorf("description = %q, want %q", col.Description, "A description")
	}
}

func TestGetAllCollections(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)

	for i := 0; i < 3; i++ {
		if _, err := db.CreateCollection("Col", ""); err != nil {
			t.Fatalf("CreateCollection: %v", err)
		}
	}

	cols, err := db.GetAllCollections()
	if err != nil {
		t.Fatalf("GetAllCollections: %v", err)
	}
	if len(cols) != 3 {
		t.Fatalf("got %d collections, want 3", len(cols))
	}
	for _, c := range cols {
		if c.PositionCount != 0 {
			t.Errorf("collection %d: positionCount = %d, want 0", c.ID, c.PositionCount)
		}
	}
}

func TestUpdateCollection(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)

	id, _ := db.CreateCollection("Old", "OldDesc")
	if err := db.UpdateCollection(id, "New", "NewDesc"); err != nil {
		t.Fatalf("UpdateCollection: %v", err)
	}
	col, _ := db.GetCollectionByID(id)
	if col.Name != "New" || col.Description != "NewDesc" {
		t.Errorf("got name=%q desc=%q", col.Name, col.Description)
	}
}

func TestDeleteCollection(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)

	importTestMatch(t, db)
	ids := getPositionIDs(t, db, 2)

	colID, _ := db.CreateCollection("ToDelete", "")
	if err := db.AddPositionsToCollection(colID, ids); err != nil {
		t.Fatalf("AddPositionsToCollection: %v", err)
	}

	if err := db.DeleteCollection(colID); err != nil {
		t.Fatalf("DeleteCollection: %v", err)
	}

	// Positions should still exist
	for _, pid := range ids {
		pos, err := db.loadPositionByIDUnlocked(pid)
		if err != nil {
			t.Errorf("position %d deleted unexpectedly: %v", pid, err)
		}
		_ = pos
	}
}

func TestAddPositionToCollection(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)

	importTestMatch(t, db)
	ids := getPositionIDs(t, db, 1)
	colID, _ := db.CreateCollection("C", "")

	if err := db.AddPositionToCollection(colID, ids[0]); err != nil {
		t.Fatalf("AddPositionToCollection: %v", err)
	}

	positions, err := db.GetCollectionPositions(colID)
	if err != nil {
		t.Fatalf("GetCollectionPositions: %v", err)
	}
	if len(positions) != 1 {
		t.Fatalf("got %d positions, want 1", len(positions))
	}
}

func TestAddPositionsToCollection(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)

	importTestMatch(t, db)
	ids := getPositionIDs(t, db, 5)
	colID, _ := db.CreateCollection("C", "")

	if err := db.AddPositionsToCollection(colID, ids); err != nil {
		t.Fatalf("AddPositionsToCollection: %v", err)
	}

	positions, err := db.GetCollectionPositions(colID)
	if err != nil {
		t.Fatalf("GetCollectionPositions: %v", err)
	}
	if len(positions) != len(ids) {
		t.Fatalf("got %d positions, want %d", len(positions), len(ids))
	}
}

func TestAddDuplicatePosition(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)

	importTestMatch(t, db)
	ids := getPositionIDs(t, db, 1)
	colID, _ := db.CreateCollection("C", "")

	_ = db.AddPositionToCollection(colID, ids[0])
	// Second add should not error (idempotent or graceful)
	err := db.AddPositionToCollection(colID, ids[0])
	if err != nil {
		t.Fatalf("duplicate add should not error: %v", err)
	}

	positions, _ := db.GetCollectionPositions(colID)
	if len(positions) != 1 {
		t.Errorf("got %d positions after duplicate add, want 1", len(positions))
	}
}

func TestRemovePositionFromCollection(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)

	importTestMatch(t, db)
	ids := getPositionIDs(t, db, 1)
	colID, _ := db.CreateCollection("C", "")
	_ = db.AddPositionToCollection(colID, ids[0])

	if err := db.RemovePositionFromCollection(colID, ids[0]); err != nil {
		t.Fatalf("RemovePositionFromCollection: %v", err)
	}

	positions, _ := db.GetCollectionPositions(colID)
	if len(positions) != 0 {
		t.Errorf("got %d positions, want 0", len(positions))
	}
}

func TestRemovePositionsFromCollection(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)

	importTestMatch(t, db)
	ids := getPositionIDs(t, db, 3)
	colID, _ := db.CreateCollection("C", "")
	_ = db.AddPositionsToCollection(colID, ids)

	if err := db.RemovePositionsFromCollection(colID, ids[:2]); err != nil {
		t.Fatalf("RemovePositionsFromCollection: %v", err)
	}

	positions, _ := db.GetCollectionPositions(colID)
	if len(positions) != 1 {
		t.Errorf("got %d positions, want 1", len(positions))
	}
}

func TestReorderCollections(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)

	id1, _ := db.CreateCollection("A", "")
	id2, _ := db.CreateCollection("B", "")
	id3, _ := db.CreateCollection("C", "")

	// Reorder to C, A, B
	if err := db.ReorderCollections([]int64{id3, id1, id2}); err != nil {
		t.Fatalf("ReorderCollections: %v", err)
	}

	cols, _ := db.GetAllCollections()
	if len(cols) != 3 {
		t.Fatalf("got %d collections", len(cols))
	}
	if cols[0].ID != id3 || cols[1].ID != id1 || cols[2].ID != id2 {
		t.Errorf("order = [%d,%d,%d], want [%d,%d,%d]",
			cols[0].ID, cols[1].ID, cols[2].ID, id3, id1, id2)
	}
}

func TestReorderCollectionPositions(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)

	importTestMatch(t, db)
	ids := getPositionIDs(t, db, 3)
	colID, _ := db.CreateCollection("C", "")
	_ = db.AddPositionsToCollection(colID, ids)

	// Reverse order
	reversed := []int64{ids[2], ids[1], ids[0]}
	if err := db.ReorderCollectionPositions(colID, reversed); err != nil {
		t.Fatalf("ReorderCollectionPositions: %v", err)
	}

	positions, _ := db.GetCollectionPositions(colID)
	if len(positions) < 3 {
		t.Fatalf("got %d positions", len(positions))
	}
	if positions[0].ID != ids[2] {
		t.Errorf("first position ID = %d, want %d", positions[0].ID, ids[2])
	}
}

func TestMovePositionBetweenCollections(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)

	importTestMatch(t, db)
	ids := getPositionIDs(t, db, 1)
	colA, _ := db.CreateCollection("A", "")
	colB, _ := db.CreateCollection("B", "")
	_ = db.AddPositionToCollection(colA, ids[0])

	if err := db.MovePositionBetweenCollections(colA, colB, ids[0]); err != nil {
		t.Fatalf("MovePositionBetweenCollections: %v", err)
	}

	posA, _ := db.GetCollectionPositions(colA)
	posB, _ := db.GetCollectionPositions(colB)
	if len(posA) != 0 {
		t.Errorf("collection A has %d positions, want 0", len(posA))
	}
	if len(posB) != 1 {
		t.Errorf("collection B has %d positions, want 1", len(posB))
	}
}

func TestCopyPositionToCollection(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)

	importTestMatch(t, db)
	ids := getPositionIDs(t, db, 1)
	colA, _ := db.CreateCollection("A", "")
	colB, _ := db.CreateCollection("B", "")
	_ = db.AddPositionToCollection(colA, ids[0])

	if err := db.CopyPositionToCollection(colB, ids[0]); err != nil {
		t.Fatalf("CopyPositionToCollection: %v", err)
	}

	posA, _ := db.GetCollectionPositions(colA)
	posB, _ := db.GetCollectionPositions(colB)
	if len(posA) != 1 || len(posB) != 1 {
		t.Errorf("A=%d B=%d, want both 1", len(posA), len(posB))
	}
}

func TestGetPositionCollections(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)

	importTestMatch(t, db)
	ids := getPositionIDs(t, db, 1)
	col1, _ := db.CreateCollection("C1", "")
	col2, _ := db.CreateCollection("C2", "")
	_ = db.AddPositionToCollection(col1, ids[0])
	_ = db.AddPositionToCollection(col2, ids[0])

	cols, err := db.GetPositionCollections(ids[0])
	if err != nil {
		t.Fatalf("GetPositionCollections: %v", err)
	}
	if len(cols) != 2 {
		t.Errorf("got %d collections, want 2", len(cols))
	}
}

func TestExportCollections(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)

	importTestMatch(t, db)
	ids := getPositionIDs(t, db, 3)
	colID, _ := db.CreateCollection("Export", "")
	_ = db.AddPositionsToCollection(colID, ids)

	exportPath := filepath.Join(t.TempDir(), "export.db")
	err := db.ExportCollections(exportPath, []int64{colID}, map[string]string{}, true, true, "", "")
	if err != nil {
		t.Fatalf("ExportCollections: %v", err)
	}

	// Verify the export file exists and is non-empty
	info, err := os.Stat(exportPath)
	if err != nil {
		t.Fatalf("export file missing: %v", err)
	}
	if info.Size() == 0 {
		t.Error("export file is empty")
	}
}

// TestExportCollections_RoundTrip_ScalarColumnsAndDedup is fiche-04's core
// regression for the collection export path — same defect and same fix as
// ExportDatabase (see the comment block above
// TestExportDatabase_RoundTrip_ScalarColumnsAndDedup in export_test.go):
// before the fix, ExportCollections hand-rolled its own two-column position
// table and never wrote zobrist_hash or any scalar column.
func TestExportCollections_RoundTrip_ScalarColumnsAndDedup(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)

	importTestMatch(t, db)
	ids := getPositionIDs(t, db, 3)
	colID, _ := db.CreateCollection("Export", "")
	if err := db.AddPositionsToCollection(colID, ids); err != nil {
		t.Fatalf("AddPositionsToCollection: %v", err)
	}

	exportPath := filepath.Join(t.TempDir(), "export.db")
	if err := db.ExportCollections(exportPath, []int64{colID}, map[string]string{}, true, true, "", ""); err != nil {
		t.Fatalf("ExportCollections: %v", err)
	}

	reopened := NewDatabase()
	if err := reopened.OpenDatabase(exportPath); err != nil {
		t.Fatalf("OpenDatabase(export): %v", err)
	}
	defer reopened.Close()

	positions, err := reopened.LoadAllPositions()
	if err != nil {
		t.Fatalf("LoadAllPositions: %v", err)
	}
	if len(positions) != len(ids) {
		t.Fatalf("expected %d positions, got %d", len(ids), len(positions))
	}

	var nullHashes int
	if err := reopened.db.QueryRow(`SELECT COUNT(*) FROM position WHERE zobrist_hash IS NULL`).Scan(&nullHashes); err != nil {
		t.Fatalf("query zobrist_hash: %v", err)
	}
	if nullHashes != 0 {
		t.Errorf("expected 0 NULL zobrist_hash after export+reopen, got %d", nullHashes)
	}

	var nullScalars int
	if err := reopened.db.QueryRow(`SELECT COUNT(*) FROM position WHERE dice_1 IS NULL OR score_1 IS NULL OR score_2 IS NULL OR pip_diff IS NULL`).Scan(&nullScalars); err != nil {
		t.Fatalf("query scalar columns: %v", err)
	}
	if nullScalars != 0 {
		t.Errorf("expected 0 rows with a NULL scalar column, got %d", nullScalars)
	}

	// A pure-SQL score search must find the first position by its own score.
	first := positions[0]
	var found int
	if err := reopened.db.QueryRow(`SELECT COUNT(*) FROM position WHERE score_1 = ? AND score_2 = ?`,
		first.Score[0], first.Score[1]).Scan(&found); err != nil {
		t.Fatalf("query score search: %v", err)
	}
	if found == 0 {
		t.Error("expected at least one position findable by score via SQL columns")
	}

	// Re-importing the export into a database that already holds these positions
	// must create no duplicates.
	before, err := db.LoadAllPositions()
	if err != nil {
		t.Fatalf("LoadAllPositions on the source db: %v", err)
	}
	result, err := db.CommitImportDatabase(exportPath)
	if err != nil {
		t.Fatalf("CommitImportDatabase(export) into the source db: %v", err)
	}
	if added, _ := result["added"].(int); added != 0 {
		t.Errorf("re-importing the export added %d new positions, want 0", added)
	}
	after, err := db.LoadAllPositions()
	if err != nil {
		t.Fatalf("LoadAllPositions after re-import: %v", err)
	}
	if len(after) != len(before) {
		t.Errorf("re-importing the export duplicated positions: before=%d after=%d", len(before), len(after))
	}
}

// TestExportCollections_MetadataAllowListAndWatermark covers fiche-04's other
// collection-export defect: metadata used to be copied by raw inclusion
// (`for key, value := range metadata { INSERT ... }`), and no watermark was
// ever written. See ADR-0007 and issuance.CarriedMetadataKeys.
func TestExportCollections_MetadataAllowListAndWatermark(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)

	importTestMatch(t, db)
	ids := getPositionIDs(t, db, 1)
	colID, _ := db.CreateCollection("Export", "")
	if err := db.AddPositionsToCollection(colID, ids); err != nil {
		t.Fatalf("AddPositionsToCollection: %v", err)
	}

	exportPath := filepath.Join(t.TempDir(), "export.db")
	metadata := map[string]string{
		"user":           "Jean",
		"description":    "A course",
		"issue_register": "list of every recipient and their password", // NOT on the allow-list
	}
	if err := db.ExportCollections(exportPath, []int64{colID}, metadata, false, false, "Cours de Jean", "Ne pas redistribuer"); err != nil {
		t.Fatalf("ExportCollections: %v", err)
	}

	edb := openExportDB(t, exportPath)
	defer edb.Close()

	var user string
	if err := edb.QueryRow(`SELECT value FROM metadata WHERE key = 'user'`).Scan(&user); err != nil || user != "Jean" {
		t.Errorf("expected allow-listed metadata 'user'='Jean', got %q (err=%v)", user, err)
	}

	var offAllowList int
	if err := edb.QueryRow(`SELECT COUNT(*) FROM metadata WHERE key = 'issue_register'`).Scan(&offAllowList); err != nil {
		t.Fatalf("query metadata: %v", err)
	}
	if offAllowList != 0 {
		t.Error("a metadata key outside issuance.CarriedMetadataKeys travelled into the collection export")
	}

	var watermark string
	if err := edb.QueryRow(`SELECT value FROM metadata WHERE key = 'watermark'`).Scan(&watermark); err != nil {
		t.Fatalf("expected a watermark row: %v", err)
	}
	if watermark == "" {
		t.Error("expected a non-empty watermark document")
	}
}
