package sqlite_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/changeset"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
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

// The cursor stays at the last entry of a carried family, so a family a later
// version carries is not skipped by a peer that exported before it did.
func TestChangeSetCursorStopsAtUncarriedFamilies_SQLite(t *testing.T) {
	ctx := context.Background()
	s, path := openLib(t)
	id, err := s.Collections().Create(ctx, "", "Before the unknown", "")
	if err != nil {
		t.Fatal(err)
	}
	u, err := s.ObjectUUIDs().ObjectUUID(ctx, "", "collection", id)
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, `INSERT INTO change_log (uuid, family, op) VALUES ('0190a000-0000-7000-8000-00000000f00d', 'not-carried-yet', 'update')`); err != nil {
		t.Fatal(err)
	}
	set, err := changeset.Export(ctx, s, "", storage.Cursor{}, 1000)
	if err != nil {
		t.Fatal(err)
	}
	last, err := s.ChangeJournal().Entry(ctx, "", u)
	if err != nil {
		t.Fatal(err)
	}
	if set.Cursor.Seq != last.Seq {
		t.Errorf("cursor %d passed the carried entries (%d) over an uncarried family", set.Cursor.Seq, last.Seq)
	}
	for _, r := range set.Records {
		if r.Family == "not-carried-yet" {
			t.Errorf("a record of family %s", r.Family)
		}
	}
}
