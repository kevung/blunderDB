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
// compact, and Vacuum refuses before it purges the trash or touches the file.
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
	// SQLite reports the resolved path; a temp dir may sit behind a symlink
	// (/var -> /private/var on macOS), so both sides are resolved.
	resolve := func(p string) string {
		r, err := filepath.EvalSymlinks(p)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	if got, want := resolve(d.libraryDir(context.Background())), resolve(filepath.Dir(path)); got != want {
		t.Fatalf("libraryDir = %q, want %q", got, want)
	}

	// A trash entry past its retention, which a Vacuum would purge.
	if _, err := d.db.Exec(`INSERT INTO trash (kind, payload, deleted_at) VALUES ('comment', '{}', '2000-01-01 00:00:00')`); err != nil {
		t.Fatal(err)
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
	var trashed int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM trash`).Scan(&trashed); err != nil {
		t.Fatal(err)
	}
	if trashed != 1 {
		t.Errorf("Vacuum without a local file purged the trash: %d entries left, want 1", trashed)
	}
}
