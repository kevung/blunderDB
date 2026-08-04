package database

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adrg/xdg"
)

// The single-writer marker used to be written as `<database>.lock`, next to the user's own
// files. It looked like debris, and it could not be deleted on release without the classic
// unlink race — another instance may already hold a descriptor to that inode. It now lives
// in the cache directory instead.
func TestTheLockFileDoesNotLandBesideTheDatabase(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	xdg.Reload()
	t.Cleanup(xdg.Reload)

	dir := t.TempDir()
	path := filepath.Join(dir, "cours.db")

	db := NewDatabase()
	if err := db.SetupDatabase(path); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".lock") {
			t.Fatalf("a lock file was left in the user's folder: %s", entry.Name())
		}
	}
}

// Wherever it lives, the marker must still do its job: the same database opened twice gives
// the second caller a read-only handle.
func TestTheLockStillExcludesASecondWriter(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	xdg.Reload()
	t.Cleanup(xdg.Reload)

	path := filepath.Join(t.TempDir(), "cours.db")
	first := NewDatabase()
	if err := first.SetupDatabase(path); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	t.Cleanup(func() { _ = first.Close() })

	second := NewDatabase()
	if err := second.OpenDatabase(path); err != nil {
		t.Fatalf("OpenDatabase: %v", err)
	}
	t.Cleanup(func() { _ = second.Close() })

	if !second.readOnly {
		t.Fatal("the second handle on the same database must be read-only")
	}

	// And two different databases must not exclude each other.
	other := filepath.Join(t.TempDir(), "autre.db")
	third := NewDatabase()
	if err := third.SetupDatabase(other); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	t.Cleanup(func() { _ = third.Close() })
	if third.readOnly {
		t.Fatal("a different database must get its own lock")
	}
}

// Closing must leave the folder as it found it: SQLite removes its -wal and -shm companions
// when the last connection closes cleanly, which is exactly what the application failed to
// do at shutdown.
func TestClosingLeavesNoCompanionFiles(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	xdg.Reload()
	t.Cleanup(xdg.Reload)

	dir := t.TempDir()
	path := filepath.Join(dir, "cours.db")
	db := NewDatabase()
	if err := db.SetupDatabase(path); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	if _, err := db.GetAllMatches(); err != nil {
		t.Fatalf("GetAllMatches: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	var leftovers []string
	for _, entry := range entries {
		if entry.Name() != "cours.db" {
			leftovers = append(leftovers, entry.Name())
		}
	}
	if len(leftovers) != 0 {
		t.Fatalf("closing left files behind: %v", leftovers)
	}
}
