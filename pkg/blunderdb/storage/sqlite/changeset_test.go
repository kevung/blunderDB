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

// Each trigger dates a write past what the object holds. A row inserted with
// its uuid is the one way to reach the insert trigger over an entry: nothing in
// the stores writes it yet, a restore goes through the update trigger.
func TestJournalTriggersDatePastTheEntry_SQLite(t *testing.T) {
	ctx := context.Background()
	_, path := openLib(t)
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	const u = "01a1a86f-0000-7000-8000-000000000001"
	const ahead = "2999-01-01T00:00:00.000Z"
	entry := func() (op, at string) {
		t.Helper()
		if err := db.QueryRowContext(ctx, `SELECT op, changed_at FROM change_log WHERE uuid = ?`, u).Scan(&op, &at); err != nil {
			t.Fatal(err)
		}
		return op, at
	}
	for _, step := range []struct{ name, stmt, op, at string }{
		{"insert", `INSERT INTO collection (uuid, name) VALUES ('` + u + `', 'one')`, "create", "2999-01-01T00:00:00.001Z"},
		{"update", `UPDATE collection SET name = 'two' WHERE uuid = '` + u + `'`, "update", "2999-01-01T00:00:00.002Z"},
		{"delete", `DELETE FROM collection WHERE uuid = '` + u + `'`, "delete", "2999-01-01T00:00:00.003Z"},
		{"insert over a delete", `INSERT INTO collection (uuid, name) VALUES ('` + u + `', 'three')`, "create", "2999-01-01T00:00:00.004Z"},
	} {
		if step.name == "insert" {
			if _, err := db.ExecContext(ctx, `INSERT INTO change_log (uuid, family, op, changed_at) VALUES (?, 'collection', 'update', ?)`, u, ahead); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := db.ExecContext(ctx, step.stmt); err != nil {
			t.Fatalf("%s: %v", step.name, err)
		}
		if op, at := entry(); op != step.op || at != step.at {
			t.Errorf("%s: entry %s %s, want %s %s", step.name, op, at, step.op, step.at)
		}
	}
}
