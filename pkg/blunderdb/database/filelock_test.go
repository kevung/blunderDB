package database

import (
	"path/filepath"
	"testing"
)

// TestFileLock_SecondInstanceReadOnly verifies the single-writer guard: a second
// open of the same file while the first is still open falls back to read-only,
// its writes are rejected, and once the first closes a fresh open is writable
// again. Two in-process *Database values stand in for two app instances — flock
// treats their separate lock-file descriptors as distinct holders.
func TestFileLock_SecondInstanceReadOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shared.db")

	d1 := NewDatabase()
	if err := d1.SetupDatabase(path); err != nil {
		t.Fatalf("SetupDatabase (instance 1): %v", err)
	}
	defer d1.Close()
	if d1.IsReadOnly() {
		t.Fatal("instance 1 should hold the write lock, not be read-only")
	}

	// Second instance opens the same file → read-only fallback.
	d2 := NewDatabase()
	if err := d2.OpenDatabase(path); err != nil {
		t.Fatalf("OpenDatabase (instance 2): %v", err)
	}
	if !d2.IsReadOnly() {
		t.Fatal("instance 2 should be read-only while instance 1 holds the lock")
	}
	if _, err := d2.Conn().Exec(`INSERT INTO position (state) VALUES ('x')`); err == nil {
		t.Error("write on the read-only instance should have been rejected")
	}

	// Creating/replacing the file while it is held must be refused outright.
	d2b := NewDatabase()
	if err := d2b.SetupDatabase(path); err == nil {
		t.Error("SetupDatabase on a file held by another instance should fail")
		d2b.Close()
	}

	// Reads still work on the read-only instance.
	if _, err := d2.Conn().Query(`SELECT count(*) FROM position`); err != nil {
		t.Errorf("read on the read-only instance should succeed: %v", err)
	}
	d2.Close()

	// Once instance 1 releases the lock, a fresh open is writable again.
	if err := d1.Close(); err != nil {
		t.Fatalf("closing instance 1: %v", err)
	}
	d3 := NewDatabase()
	if err := d3.OpenDatabase(path); err != nil {
		t.Fatalf("OpenDatabase (instance 3): %v", err)
	}
	defer d3.Close()
	if d3.IsReadOnly() {
		t.Fatal("after the holder closed, the database should open writable")
	}
	if _, err := d3.Conn().Exec(`INSERT INTO position (state) VALUES ('y')`); err != nil {
		t.Errorf("write after reacquiring the lock should succeed: %v", err)
	}
}

// TestFileLock_MemorySkipsLock ensures in-memory databases (tests) are never
// locked or forced read-only.
func TestFileLock_MemorySkipsLock(t *testing.T) {
	d := NewDatabase()
	if err := d.SetupDatabase(":memory:"); err != nil {
		t.Fatalf("SetupDatabase(:memory:): %v", err)
	}
	defer d.Close()
	if d.IsReadOnly() {
		t.Error(":memory: must never be read-only")
	}
	if d.lock != nil {
		t.Error(":memory: must not hold a file lock")
	}
}
