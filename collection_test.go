package main

import (
	"os"
	"path/filepath"
	"testing"
)

func setupCollectionTestDB(t *testing.T) (*Database, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db := NewDatabase()
	if err := db.SetupDatabase(dbPath); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}

	return db, func() {
		if db.db != nil {
			db.db.Close()
		}
		os.Remove(dbPath)
	}
}

// importTestMatch imports the test.sgf fixture and returns the match ID.
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

func TestCreateCollection(t *testing.T) {
	db, cleanup := setupCollectionTestDB(t)
	defer cleanup()

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
	db, cleanup := setupCollectionTestDB(t)
	defer cleanup()

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
	db, cleanup := setupCollectionTestDB(t)
	defer cleanup()

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
	db, cleanup := setupCollectionTestDB(t)
	defer cleanup()

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
	db, cleanup := setupCollectionTestDB(t)
	defer cleanup()

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
	db, cleanup := setupCollectionTestDB(t)
	defer cleanup()

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
	db, cleanup := setupCollectionTestDB(t)
	defer cleanup()

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
	db, cleanup := setupCollectionTestDB(t)
	defer cleanup()

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
	db, cleanup := setupCollectionTestDB(t)
	defer cleanup()

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
	db, cleanup := setupCollectionTestDB(t)
	defer cleanup()

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
	db, cleanup := setupCollectionTestDB(t)
	defer cleanup()

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
	db, cleanup := setupCollectionTestDB(t)
	defer cleanup()

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
	db, cleanup := setupCollectionTestDB(t)
	defer cleanup()

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
	db, cleanup := setupCollectionTestDB(t)
	defer cleanup()

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
	db, cleanup := setupCollectionTestDB(t)
	defer cleanup()

	importTestMatch(t, db)
	ids := getPositionIDs(t, db, 3)
	colID, _ := db.CreateCollection("Export", "")
	_ = db.AddPositionsToCollection(colID, ids)

	exportPath := filepath.Join(t.TempDir(), "export.db")
	err := db.ExportCollections(exportPath, []int64{colID}, map[string]string{}, true, true)
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
