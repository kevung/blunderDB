//go:build postgres

package postgres

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// TestImportReportReadsItsBatchOnly: with no statistics to go on, the report
// reads the batch's rows, not the tenant's.
// Counted by Postgres itself, in the report's own transaction.
func TestImportReportReadsItsBatchOnly(t *testing.T) {
	ctx := context.Background()
	dsn := purgeTestDB(t)
	s, err := Open(ctx, dsn, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	if err := s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join("..", "..", "..", "..", "testdata")
	files, err := filepath.Glob(filepath.Join(dir, "2024-08-10-Aachen-*", "*", "*.xg"))
	if err != nil || len(files) < 3 {
		t.Fatalf("corpus: %d files, %v", len(files), err)
	}
	batches := s.ImportBatches()
	var last int64
	for _, f := range files {
		if last, err = batches.Begin(ctx, "", "bench", "xg"); err != nil {
			t.Fatal(err)
		}
		if _, err := (ingest.XGImporter{S: s}).Import(ctx, "", ingest.Source{Path: f, BatchID: last}, nil); err != nil {
			t.Fatal(err)
		}
	}
	// What a library looks like to the planner before any ANALYZE.
	for _, table := range []string{"position", "analysis", "move", "game", "match"} {
		if _, err := s.pool.Exec(ctx, `UPDATE pg_class SET reltuples = -1, relpages = 0 WHERE oid = $1::regclass`, table); err != nil {
			t.Fatal(err)
		}
		if _, err := s.pool.Exec(ctx, `DELETE FROM pg_statistic WHERE starelid = $1::regclass`, table); err != nil {
			t.Fatal(err)
		}
	}

	var library, batchMoves int64
	if err := s.pool.QueryRow(ctx, `SELECT (SELECT COUNT(*) FROM position),
		(SELECT COUNT(mv.position_id) FROM move mv JOIN game g ON g.id = mv.game_id JOIN match m ON m.id = g.match_id WHERE m.import_batch_id = $1)`, last).
		Scan(&library, &batchMoves); err != nil {
		t.Fatal(err)
	}
	if batchMoves == 0 || library < 2*batchMoves {
		t.Fatalf("library %d, batch %d: the test needs a library larger than its batch", library, batchMoves)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op
	// The counters also hold what the connection read before, not yet flushed.
	positionReads := func() (n int64) {
		t.Helper()
		if err := tx.QueryRow(ctx, `SELECT COALESCE(seq_tup_read,0) + COALESCE(idx_tup_fetch,0)
			FROM pg_stat_xact_user_tables WHERE relname = 'position'`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	before := positionReads()
	store := &sqlshared.ImportBatchStore{DB: binder{db: tx}.shared()}
	if _, err := store.Report(ctx, "", last, nil); err != nil {
		t.Fatal(err)
	}
	read := positionReads() - before
	// Three queries, each reading a move's position once.
	if read > 3*batchMoves {
		t.Errorf("the report read %d position rows, for a batch of %d moves in a library of %d", read, batchMoves, library)
	}
}
