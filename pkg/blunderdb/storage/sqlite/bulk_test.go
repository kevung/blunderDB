package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func openBulkTestDB(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()
	dsn := filepath.Join(t.TempDir(), "bulk.db")
	db, err := sql.Open("sqlite", DSN(dsn))
	if err != nil {
		t.Fatal(err)
	}
	ConfigurePool(db, dsn)
	t.Cleanup(func() { _ = db.Close() })
	if err := EnsureSchema(ctx, db); err != nil {
		t.Fatal(err)
	}
	return db
}

func hasIndex(t *testing.T, db *sql.DB, name string) bool {
	t.Helper()
	have, err := indexNames(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	return have[name]
}

// A bulk session drops the secondary indexes and keeps those deduplication
// reads; Close rebuilds every one of them.
func TestBulkSession_DropsAndRebuildsIndexes(t *testing.T) {
	ctx := context.Background()
	db := openBulkTestDB(t)
	s, err := BeginBulk(ctx, db, BulkOptions{Unsafe: true, DropIndexes: true})
	if err != nil {
		t.Fatal(err)
	}
	if hasIndex(t, db, "idx_position_pip_diff") || hasIndex(t, db, "idx_analysis_cube_error") {
		t.Error("secondary indexes still present during the bulk session")
	}
	if !hasIndex(t, db, "idx_position_zobrist") || !hasIndex(t, db, "idx_analysis_position") {
		t.Error("an index deduplication reads was dropped")
	}
	tx, err := s.BeginTx(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(ctx); err != nil {
		t.Fatal(err)
	}
	for name := range bulkDroppableIndexes() {
		if !hasIndex(t, db, name) {
			t.Errorf("index %s not rebuilt on Close", name)
		}
	}
	var sync int
	if err := db.QueryRow("PRAGMA synchronous").Scan(&sync); err != nil || sync != 1 {
		t.Errorf("pool connection synchronous = %d (%v), want NORMAL (1)", sync, err)
	}
}

// A bulk import cut before Close leaves indexes missing: the next open's
// EnsureSchema rebuilds them by name.
func TestBulkSession_CutRepairedByEnsureSchema(t *testing.T) {
	ctx := context.Background()
	db := openBulkTestDB(t)
	s, err := BeginBulk(ctx, db, BulkOptions{DropIndexes: true})
	if err != nil {
		t.Fatal(err)
	}
	_ = s.conn.Close() // the process dies: nothing is restored
	if hasIndex(t, db, "idx_position_pip_diff") {
		t.Fatal("index present after the drop")
	}
	if err := EnsureSchema(ctx, db); err != nil {
		t.Fatal(err)
	}
	for name := range bulkDroppableIndexes() {
		if !hasIndex(t, db, name) {
			t.Errorf("index %s not rebuilt at open", name)
		}
	}
}
