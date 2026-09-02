package database

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestVacuum_ReclaimsSpaceAfterDeletes is the fiche's mandated scenario: a
// database inflated by deletions shrinks back down after Vacuum, and the
// rows that were not deleted survive intact.
func TestVacuum_ReclaimsSpaceAfterDeletes(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "vacuum.db")

	d := NewDatabase()
	if err := d.SetupDatabase(dbPath); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	defer d.Close()

	// A row that must survive the whole exercise, so Vacuum's "the file
	// shrinks" is checked alongside "and nothing legitimate was lost".
	survivorState := "kept-position"
	res, err := d.Conn().Exec(`INSERT INTO position (state) VALUES (?)`, survivorState)
	if err != nil {
		t.Fatalf("insert survivor: %v", err)
	}
	survivorID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("survivor LastInsertId: %v", err)
	}

	// Bulk-insert padding rows, each carrying a sizeable TEXT blob, then
	// delete almost all of them. SQLite never shrinks the file on DELETE by
	// itself — that's the whole premise this feature addresses — so before
	// Vacuum runs, the file must still reflect the padding having existed.
	const numPadding = 3000
	blob := strings.Repeat("x", 2000)
	tx, err := d.Conn().Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	stmt, err := tx.Prepare(`INSERT INTO position (state) VALUES (?)`)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	for i := 0; i < numPadding; i++ {
		if _, err := stmt.Exec(blob); err != nil {
			stmt.Close()
			t.Fatalf("insert padding %d: %v", i, err)
		}
	}
	stmt.Close()
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit padding: %v", err)
	}

	if _, err := d.Conn().Exec(`DELETE FROM position WHERE id != ?`, survivorID); err != nil {
		t.Fatalf("delete padding: %v", err)
	}

	sizeBeforeOnDisk := fileSizeOf(t, dbPath)

	result, err := d.Vacuum()
	if err != nil {
		t.Fatalf("Vacuum: %v", err)
	}
	sizeBefore, sizeAfter := result.SizeBefore, result.SizeAfter

	if sizeBefore == 0 {
		t.Fatalf("Vacuum reported sizeBefore=0 for a file-backed database")
	}
	if sizeAfter >= sizeBefore {
		t.Fatalf("Vacuum did not shrink the file: before=%d after=%d", sizeBefore, sizeAfter)
	}

	// The size Vacuum reported for "before" should agree with what was on
	// disk right before the call (the WAL checkpoint folds in cleanly).
	if sizeBefore != sizeBeforeOnDisk {
		t.Errorf("Vacuum sizeBefore=%d does not match on-disk size just before the call=%d", sizeBefore, sizeBeforeOnDisk)
	}

	// The reported "after" size must match reality.
	sizeAfterOnDisk := fileSizeOf(t, dbPath)
	if sizeAfter != sizeAfterOnDisk {
		t.Errorf("Vacuum sizeAfter=%d does not match on-disk size=%d", sizeAfter, sizeAfterOnDisk)
	}

	// The survivor row must still be there, untouched.
	var gotState string
	if err := d.Conn().QueryRow(`SELECT state FROM position WHERE id = ?`, survivorID).Scan(&gotState); err != nil {
		t.Fatalf("select survivor after vacuum: %v", err)
	}
	if gotState != survivorState {
		t.Errorf("survivor state = %q, want %q", gotState, survivorState)
	}

	var remaining int
	if err := d.Conn().QueryRow(`SELECT COUNT(*) FROM position`).Scan(&remaining); err != nil {
		t.Fatalf("count after vacuum: %v", err)
	}
	if remaining != 1 {
		t.Errorf("position count after vacuum = %d, want 1", remaining)
	}

	t.Logf("vacuum reclaimed %d bytes (%d -> %d)", sizeBefore-sizeAfter, sizeBefore, sizeAfter)
}

// fileSizeOf is os.Stat().Size() with the failure folded into the test.
func fileSizeOf(t *testing.T, path string) int64 {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	return info.Size()
}

// TestVacuum_InMemoryDatabase makes sure Vacuum degrades gracefully on
// :memory: (used throughout the test suite and by some CLI invocations):
// there is no file to size or free-space-check, but VACUUM/ANALYZE must
// still run without error.
func TestVacuum_InMemoryDatabase(t *testing.T) {
	d := NewDatabase()
	if err := d.SetupDatabase(":memory:"); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	defer d.Close()

	result, err := d.Vacuum()
	if err != nil {
		t.Fatalf("Vacuum: %v", err)
	}
	if result.SizeBefore != 0 || result.SizeAfter != 0 {
		t.Errorf("Vacuum on :memory: reported sizeBefore=%d sizeAfter=%d, want 0, 0", result.SizeBefore, result.SizeAfter)
	}
}

// TestVacuum_NoDatabaseOpen guards the nil-db error path (e.g. a fresh
// Database that was never Setup/Open'd).
func TestVacuum_NoDatabaseOpen(t *testing.T) {
	d := NewDatabase()
	if _, err := d.Vacuum(); err == nil {
		t.Fatal("Vacuum on an unopened Database: want error, got nil")
	}
}
