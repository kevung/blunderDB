package sqlite_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/storagetest"
)

func openLib(t *testing.T) (*sqlite.Storage, string) {
	path := filepath.Join(t.TempDir(), "lib.db")
	s, err := sqlite.Open(context.Background(), path, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s, path
}

// Rows that predate the journal have no entry: here a library whose journal
// is dropped and made again, as an upgrade leaves one.
func TestChangeSetPreJournal_SQLite(t *testing.T) {
	ctx := context.Background()
	a, path := openLib(t)
	b, _ := openLib(t)
	storagetest.CheckChangeSetPreJournal(t, a, ctx, "", b, ctx, "", func() {
		db, err := sql.Open("sqlite", path)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		if err := sqlite.ForgetChangeLog(ctx, db); err != nil {
			t.Fatal(err)
		}
		if err := sqlite.EnsureChangeLog(ctx, db); err != nil {
			t.Fatal(err)
		}
	})
}

func TestChangeSet_SQLite(t *testing.T) {
	open := func() *sqlite.Storage { s, _ := openLib(t); return s }
	ctx := context.Background()
	storagetest.CheckChangeSet(t, open(), ctx, "", open(), ctx, "")
}

// A first sync with no journal, whatever the size of the pages.
func TestChangeSetFirstSync_SQLite(t *testing.T) {
	ctx := context.Background()
	a, path := openLib(t)
	b, _ := openLib(t)
	storagetest.CheckChangeSetFirstSync(t, a, ctx, "", b, ctx, "", func() {
		db, err := sql.Open("sqlite", path)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		if err := sqlite.ForgetChangeLog(ctx, db); err != nil {
			t.Fatal(err)
		}
		if err := sqlite.EnsureChangeLog(ctx, db); err != nil {
			t.Fatal(err)
		}
	})
}
