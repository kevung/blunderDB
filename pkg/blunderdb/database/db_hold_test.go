package database

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// A write naming rows read in one library is refused once the handle carries
// another, and while it holds its library no other one comes under it.
func TestHoldLibrary_RefusesRowsOfAnotherLibrary(t *testing.T) {
	t.Parallel()
	bound, pathA := openTemp(t, "a.db")
	parked, pathB := openTemp(t, "b.db")

	ticket, err := bound.HoldLibrary(pathA)
	if err != nil {
		t.Fatalf("hold of the open library: %v", err)
	}
	var busy *BusyError
	if err := Exchange(bound, parked); !errors.As(err, &busy) || busy.Job != JobWrite {
		t.Fatalf("exchange during a held write: %v, want a %s BusyError", err, JobWrite)
	}
	if err := bound.OpenDatabase(pathB); !errors.As(err, &busy) || busy.Job != JobWrite {
		t.Fatalf("open during a held write: %v, want a %s BusyError", err, JobWrite)
	}
	if got := OpenPath(bound); got != pathA {
		t.Fatalf("a refused open moved the handle to %q", got)
	}
	bound.ReleaseLibrary(ticket)

	if err := Exchange(bound, parked); err != nil {
		t.Fatalf("exchange after release: %v", err)
	}
	if _, err := bound.HoldLibrary(pathA); !errors.Is(err, ErrLibraryReplaced) {
		t.Fatalf("hold of the library left: %v, want ErrLibraryReplaced", err)
	}
	if _, err := bound.HoldLibrary(filepath.Join(filepath.Dir(pathB), ".", "b.db")); err != nil {
		t.Fatalf("hold of the library on the handle, spelt otherwise: %v", err)
	}
}

// A hold its holder never released lapses: the window is never stuck.
func TestHoldLibrary_LeaseLapses(t *testing.T) {
	t.Parallel()
	bound, pathA := openTemp(t, "a.db")
	parked, _ := openTemp(t, "b.db")
	ticket, err := bound.HoldLibrary(pathA)
	if err != nil {
		t.Fatal(err)
	}
	bound.cancelMu.Lock()
	bound.holds[ticket] = time.Now().Add(-holdLease - time.Second)
	bound.cancelMu.Unlock()
	if err := Exchange(bound, parked); err != nil {
		t.Fatalf("exchange after the lease: %v", err)
	}
}

// With nothing open, no write holds anything.
func TestHoldLibrary_NothingOpen(t *testing.T) {
	t.Parallel()
	if _, err := NewDatabase().HoldLibrary(""); !errors.Is(err, ErrLibraryReplaced) {
		t.Fatalf("hold with nothing open: %v", err)
	}
}

// A library recreated under its own path while a write holds it is neither
// erased nor replaced: the write still lands in the file it read.
func TestRecreateDatabase_RefusedWhileHeld(t *testing.T) {
	t.Parallel()
	d, path := openTemp(t, "a.db")
	if _, err := d.CreateCollection("kept", ""); err != nil {
		t.Fatal(err)
	}
	ticket, err := d.HoldLibrary(path)
	if err != nil {
		t.Fatal(err)
	}
	var busy *BusyError
	if err := d.RecreateDatabase(path); !errors.As(err, &busy) || busy.Job != JobWrite {
		t.Fatalf("recreate during a held write: %v, want a %s BusyError", err, JobWrite)
	}
	if got, err := d.GetAllCollections(); err != nil || len(got) != 1 {
		t.Fatalf("collections after a refused recreate: %v, %v", got, err)
	}
	d.ReleaseLibrary(ticket)
	if err := d.RecreateDatabase(path); err != nil {
		t.Fatalf("recreate after release: %v", err)
	}
	if got, err := d.GetAllCollections(); err != nil || len(got) != 0 {
		t.Fatalf("collections after recreate: %v, %v", got, err)
	}
}

// A recreation erases the journal files left beside the old one: the new library replays none.
func TestRecreateDatabase_ErasesOrphanJournal(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "a.db")
	for _, f := range []string{path, path + "-wal", path + "-shm"} {
		if err := os.WriteFile(f, []byte("orphan"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	d := NewDatabase()
	t.Cleanup(func() { _ = d.Close() })
	if err := d.RecreateDatabase(path); err != nil {
		t.Fatalf("recreate over an orphan journal: %v", err)
	}
	for _, f := range []string{path + "-wal", path + "-shm"} {
		if b, err := os.ReadFile(f); err == nil && string(b) == "orphan" {
			t.Fatalf("%s still holds the orphan journal", f)
		}
	}
}
