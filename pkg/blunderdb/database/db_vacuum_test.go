package database

import (
	"database/sql"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// TestVacuum_ReclaimsSpaceAfterDeletes: a database inflated by deletions shrinks back down after Vacuum, and the
// rows that were not deleted survive intact.
func TestVacuum_ReclaimsSpaceAfterDeletes(t *testing.T) {
	t.Parallel()
	dbPath := filepath.Join(t.TempDir(), "vacuum.db")

	d := NewDatabase()
	if err := d.SetupDatabase(dbPath); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	defer d.Close()

	// A row that must survive the whole exercise, so Vacuum's "the file
	// shrinks" is checked alongside "and nothing legitimate was lost".
	survivorState := "kept-position"
	res, err := d.conn().Exec(`INSERT INTO position (state) VALUES (?)`, survivorState)
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
	tx, err := d.conn().Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	stmt, err := tx.Prepare(`INSERT INTO position (state) VALUES (?)`)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	defer stmt.Close()
	for i := 0; i < numPadding; i++ {
		if _, err := stmt.Exec(blob); err != nil {
			t.Fatalf("insert padding %d: %v", i, err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit padding: %v", err)
	}

	if _, err := d.conn().Exec(`DELETE FROM position WHERE id != ?`, survivorID); err != nil {
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
	if err := d.conn().QueryRow(`SELECT state FROM position WHERE id = ?`, survivorID).Scan(&gotState); err != nil {
		t.Fatalf("select survivor after vacuum: %v", err)
	}
	if gotState != survivorState {
		t.Errorf("survivor state = %q, want %q", gotState, survivorState)
	}

	var remaining int
	if err := d.conn().QueryRow(`SELECT COUNT(*) FROM position`).Scan(&remaining); err != nil {
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
	t.Parallel()
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
	t.Parallel()
	d := NewDatabase()
	if _, err := d.Vacuum(); err == nil {
		t.Fatal("Vacuum on an unopened Database: want error, got nil")
	}
}

// vacuumFixture is a library with free pages to give back: two imports of
// the same match under different names would deduplicate, so one match is
// imported and its analyses' blobs inflated, then deflated again.
func vacuumFixture(t *testing.T) (*Database, string) {
	t.Helper()
	path := filepath.Join(tempDir(t), "vacuum.db")
	d := NewDatabase()
	if err := d.SetupDatabase(path); err != nil {
		t.Fatal(err)
	}
	closeOnCleanup(t, d)
	if _, err := d.ImportXGMatch(filepath.Join("testdata", "test.xg")); err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []string{
		`CREATE TABLE filler (b BLOB)`,
		`INSERT INTO filler SELECT randomblob(4000) FROM (SELECT 1 FROM position LIMIT 500)`,
		`DROP TABLE filler`,
	} {
		if _, err := d.db.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}
	return d, path
}

func withoutStats(m map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range m {
		if !strings.Contains(k, "sqlite_stat") {
			out[k] = v
		}
	}
	return out
}

// TestVacuum_ReplacesTheFile: the library is compacted into a copy that
// replaces the file, content unchanged, nothing left beside it, and the
// handle works on the new file.
func TestVacuum_ReplacesTheFile(t *testing.T) {
	t.Parallel()
	d, path := vacuumFixture(t)
	want := withoutStats(libraryContent(t, path))
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	res, err := d.Vacuum()
	if err != nil {
		t.Fatalf("Vacuum: %v", err)
	}
	if after, err := os.Stat(path); err != nil || os.SameFile(before, after) {
		t.Fatalf("the file was not replaced (%v)", err)
	}
	if res.SizeAfter >= res.SizeBefore || res.SizeAfter == 0 {
		t.Fatalf("size %d → %d; want it to shrink", res.SizeBefore, res.SizeAfter)
	}
	if _, err := os.Stat(path + ".vacuum"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the compacted copy is still beside the file: %v", err)
	}
	if got := withoutStats(libraryContent(t, path)); !maps.Equal(got, want) {
		t.Fatalf("content changed by the vacuum (%d cells, want %d)", len(got), len(want))
	}
	var n int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM position`).Scan(&n); err != nil || n == 0 {
		t.Fatalf("the handle after the swap: %d positions, %v", n, err)
	}
	if _, err := d.db.Exec(`INSERT INTO metadata (key, value) VALUES ('vacuum_probe', '1')`); err != nil {
		t.Fatalf("writing after the swap: %v", err)
	}
}

// TestVacuum_KeepsTheFileMode: the copy that replaces the library is created
// under the umask; it must come out with the library's own mode.
func TestVacuum_KeepsTheFileMode(t *testing.T) {
	t.Parallel()
	d, path := vacuumFixture(t)
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Vacuum(); err != nil {
		t.Fatalf("Vacuum: %v", err)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if os.SameFile(before, after) {
		t.Fatal("the file was not replaced; the test proves nothing")
	}
	if got := after.Mode().Perm(); got != 0o600 {
		t.Fatalf("mode after the vacuum %v, want -rw-------", after.Mode().Perm())
	}
}

// TestVacuum_InPlaceWhileAnotherConnectionHoldsTheFile: a connection of
// another process would keep reading the replaced inode, so the file is not
// replaced while one exists; the vacuum runs in place instead.
func TestVacuum_InPlaceWhileAnotherConnectionHoldsTheFile(t *testing.T) {
	t.Parallel()
	d, path := vacuumFixture(t)
	other, err := sql.Open("sqlite", sqlite.DSN(path))
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	var n int
	if err := other.QueryRow(`SELECT COUNT(*) FROM position`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	res, err := d.Vacuum()
	if err != nil {
		t.Fatalf("Vacuum: %v", err)
	}
	if res.SizeAfter >= res.SizeBefore {
		t.Fatalf("size %d → %d; want it to shrink", res.SizeBefore, res.SizeAfter)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) {
		t.Fatal("the file was replaced under another connection")
	}
	if _, err := os.Stat(path + ".vacuum"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the compacted copy is still beside the file: %v", err)
	}
	if err := other.QueryRow(`SELECT COUNT(*) FROM position`).Scan(&n); err != nil || n == 0 {
		t.Fatalf("the other connection after the vacuum: %d, %v", n, err)
	}
}
