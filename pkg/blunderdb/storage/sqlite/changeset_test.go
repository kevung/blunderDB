package sqlite_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/changeset"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
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

// A card whose position row is gone leaves the export of the rest standing.
func TestChangeSetOrphanCard_SQLite(t *testing.T) {
	ctx := context.Background()
	a, path := openLib(t)
	q := domain.InitializePosition()
	pid, err := a.Positions().Save(ctx, "", &q)
	if err != nil {
		t.Fatal(err)
	}
	deck, err := a.Anki().CreateDeck(ctx, "", "Orphans", "", domain.AnkiSourceSearch, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Anki().SyncWithPositions(ctx, "", deck, []int64{pid}); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	for _, stmt := range []string{`PRAGMA foreign_keys = OFF`, `DELETE FROM position WHERE id = ` + strconv.FormatInt(pid, 10)} {
		if _, err := conn.ExecContext(ctx, stmt); err != nil {
			t.Fatal(err)
		}
	}
	set, err := changeset.Export(ctx, a, "", storage.Cursor{}, 1000)
	if err != nil {
		t.Fatalf("an orphan card failed the whole export: %v", err)
	}
	for _, r := range set.Records {
		if r.Family == "anki_card" && r.Op != storage.ChangeDelete {
			t.Errorf("the orphan card was exported: %+v", r)
		}
	}
}

// journalOnly is a backend that keeps a journal and uuids but no Anki state.
type journalOnly struct {
	storage.Stores
	s *sqlite.Storage
}

func (j journalOnly) ChangeJournal() storage.ChangeJournalStore { return j.s.ChangeJournal() }
func (j journalOnly) ObjectUUIDs() storage.ObjectUUIDStore      { return j.s.ObjectUUIDs() }

// A deck record meeting a backend with no Anki state is an error, not a panic.
func TestChangeSetWithoutAnkiState_SQLite(t *testing.T) {
	ctx := context.Background()
	a, _ := openLib(t)
	deck := changeset.Record{
		Change:  storage.Change{UUID: "0198a000-0000-7000-8000-000000000001", Family: "anki_deck", Op: storage.ChangeCreate, ChangedAt: "2026-01-01T00:00:00.000Z"},
		Payload: []byte(`{"name":"D","sourceType":"search"}`),
	}
	_, err := changeset.Import(ctx, journalOnly{Stores: a, s: a}, "", changeset.Set{Records: []changeset.Record{deck}})
	if err == nil {
		t.Fatal("a deck was imported by a backend that keeps none")
	}
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
