package sqlite_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/storagetest"
)

func TestReaderFloor_SQLite(t *testing.T) {
	s := openTempDB(t)
	storagetest.RunReaderFloor(t, s, sqlite.SharedExecer(s))
}

// TestOpenReadOnly_RefusesAboveFloor: a file read as it stands — an import
// source, a file the CLI reads beside the open one — is refused when a newer
// blunderDB raised its floor above this binary's schema.
func TestOpenReadOnly_RefusesAboveFloor(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "newer.db")
	rw, err := sqlite.Open(ctx, path, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	rw.Close()
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`UPDATE metadata SET value = '99.0.0' WHERE key = ?`, storage.MinReaderVersionKey); err != nil {
		t.Fatal(err)
	}
	raw.Close()

	ro, err := sqlite.OpenReadOnly(ctx, path)
	if !errors.Is(err, storage.ErrDatabaseTooNew) {
		if ro != nil {
			ro.Close()
		}
		t.Fatalf("OpenReadOnly under floor 99.0.0: err = %v, want ErrDatabaseTooNew", err)
	}
}
