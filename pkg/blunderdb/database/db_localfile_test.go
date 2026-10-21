package database

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// contractOnly hides every method of a backend outside storage.Storage: it
// stands for a library that is no file of this machine.
type contractOnly struct{ storage.Storage }

// The file operations reach the backend through storage.LocalFile only: a
// store without it has no path to resolve sources against and nothing to
// compact, and Vacuum says so instead of touching the file.
func TestLocalFile_StoreWithoutAFile(t *testing.T) {
	t.Parallel()
	path := filepath.Join(tempDir(t), "local.db")
	d := NewDatabase()
	if err := d.SetupDatabase(path); err != nil {
		t.Fatal(err)
	}
	closeOnCleanup(t, d)

	if _, ok := d.store.(storage.LocalFile); !ok {
		t.Fatal("the SQLite store does not implement storage.LocalFile")
	}
	if got, want := d.libraryDir(context.Background()), filepath.Dir(path); got != want {
		t.Fatalf("libraryDir = %q, want %q", got, want)
	}

	d.mu.Lock()
	d.store = contractOnly{d.store}
	d.mu.Unlock()

	if got := d.libraryDir(context.Background()); got != "" {
		t.Errorf("libraryDir without a local file = %q, want \"\"", got)
	}
	before := fileSizeOf(t, path)
	if _, err := d.Vacuum(); !errors.Is(err, errVacuumNotLocal) {
		t.Errorf("Vacuum without a local file: err = %v, want errVacuumNotLocal", err)
	}
	if after := fileSizeOf(t, path); after != before {
		t.Errorf("Vacuum without a local file changed the file: %d -> %d bytes", before, after)
	}
}
