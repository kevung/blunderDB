package sqlite_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// A prepared insert carries the triggers compiled into it: it must follow the
// schema, or a position saved after the journal came back would go unjournaled.
func TestPreparedInsertFollowsTheTriggers(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "t.db")
	s, err := sqlite.Open(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	save := func(i int) {
		t.Helper()
		p := benchPos(i)
		if _, err := s.Positions().Save(ctx, "", &p); err != nil {
			t.Fatalf("Save %d: %v", i, err)
		}
	}
	entries := func() (n int) {
		t.Helper()
		if err := raw.QueryRow(`SELECT count(*) FROM change_log`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	save(1)
	if got := entries(); got != 1 {
		t.Fatalf("after the first save: %d entries, want 1", got)
	}
	if err := sqlite.ForgetChangeLog(ctx, raw); err != nil {
		t.Fatal(err)
	}
	save(2) // the statement's journal is gone with the table: no "no such table"
	if err := sqlite.EnsureChangeLog(ctx, raw); err != nil {
		t.Fatal(err)
	}
	save(3)
	if got := entries(); got != 1 {
		t.Fatalf("after the journal came back: %d entries, want 1 (position 3)", got)
	}
}

// Closing a Storage over a borrowed handle leaves the handle in use.
func TestPreparedInsertSurvivesABorrowedClose(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "t.db")
	owner, err := sqlite.Open(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	owner.Close()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := sqlite.New(db)
	for i := 1; i <= 2; i++ {
		p := benchPos(i)
		if _, err := s.Positions().Save(ctx, "", &p); err != nil {
			t.Fatalf("Save %d: %v", i, err)
		}
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
	}
}
