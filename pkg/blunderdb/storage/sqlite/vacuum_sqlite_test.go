package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestVacuum_ReclaimsSpaceAfterDeletes: a file inflated by deletions shrinks
// back down after Vacuum, the reported sizes agree with the file on disk,
// and the rows that were not deleted survive.
func TestVacuum_ReclaimsSpaceAfterDeletes(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "vacuum.db")
	st, err := Open(ctx, dbPath, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer st.Close()

	res, err := st.sqlDB.Exec(`INSERT INTO position (state) VALUES (?)`, "kept-position")
	if err != nil {
		t.Fatalf("insert survivor: %v", err)
	}
	survivorID, _ := res.LastInsertId()

	blob := strings.Repeat("x", 2000)
	tx, err := st.sqlDB.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	for i := 0; i < 3000; i++ {
		if _, err := tx.Exec(`INSERT INTO position (state) VALUES (?)`, blob); err != nil {
			t.Fatalf("insert padding %d: %v", i, err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit padding: %v", err)
	}
	if _, err := st.sqlDB.Exec(`DELETE FROM position WHERE id != ?`, survivorID); err != nil {
		t.Fatalf("delete padding: %v", err)
	}

	result, err := st.Vacuum(ctx)
	if err != nil {
		t.Fatalf("Vacuum: %v", err)
	}
	if result.SizeBefore == 0 {
		t.Fatal("SizeBefore = 0 for a file-backed database")
	}
	if result.SizeAfter >= result.SizeBefore {
		t.Fatalf("file did not shrink: before=%d after=%d", result.SizeBefore, result.SizeAfter)
	}
	info, err := os.Stat(dbPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Size() != result.SizeAfter {
		t.Errorf("SizeAfter=%d, file is %d bytes", result.SizeAfter, info.Size())
	}

	var gotState string
	if err := st.sqlDB.QueryRow(`SELECT state FROM position WHERE id = ?`, survivorID).Scan(&gotState); err != nil {
		t.Fatalf("select survivor: %v", err)
	}
	if gotState != "kept-position" {
		t.Errorf("survivor state = %q", gotState)
	}
}

// TestVacuum_InMemoryDatabase: no file to size, but VACUUM/ANALYZE still run.
func TestVacuum_InMemoryDatabase(t *testing.T) {
	ctx := context.Background()
	st, err := Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer st.Close()

	result, err := st.Vacuum(ctx)
	if err != nil {
		t.Fatalf("Vacuum: %v", err)
	}
	if result.SizeBefore != 0 || result.SizeAfter != 0 {
		t.Errorf("Vacuum on :memory: reported %+v, want zeros", result)
	}
}

// TestFreeSpaceBytes is a light sanity check on the platform-specific helper:
// the working directory's filesystem must report a nonzero amount of free
// space (Vacuum would read zero as "no room").
func TestFreeSpaceBytes(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	free, err := freeSpaceBytes(wd)
	if err != nil {
		t.Fatalf("freeSpaceBytes(%q): %v", wd, err)
	}
	if free == 0 {
		t.Errorf("freeSpaceBytes(%q) = 0, want > 0", wd)
	}
}
