//go:build postgres

package postgres_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	pg "github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
)

// seedBestMovePending stores perTenant analyses in tenants 1 and 2 under
// FORCEd RLS, then leaves them as 077 finds them: best_move NULL, both
// tenants listed in the cursor.
func seedBestMovePending(t *testing.T, perTenant int) (*pg.Storage, *pgx.Conn) {
	t.Helper()
	ctx := context.Background()
	s, conn, _ := openAsRLSOwner(t)
	analysis := &domain.PositionAnalysis{AnalysisType: "CheckerMove", CheckerAnalysis: &domain.CheckerAnalysis{
		Moves: []domain.CheckerMove{{Index: 0, Move: "24/20 13/11", Equity: 0.1}}}}
	for i, scope := range []string{"1", "2"} {
		for j := range perTenant {
			p := domain.InitializePosition()
			p.Board.Points[1+i].Checkers = 1 + j%5
			p.Board.Points[7+j/5].Checkers = 1 + j%3
			pid, err := s.Positions().Save(ctx, scope, &p)
			if err != nil {
				t.Fatal(err)
			}
			if err := s.Analyses().Save(ctx, scope, pid, analysis); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := s.ApplyRLS(ctx); err != nil {
		t.Fatalf("ApplyRLS: %v", err)
	}
	execUnforced(t, conn, []string{"analysis"}, []string{
		`UPDATE analysis SET best_move = NULL`,
		`INSERT INTO metadata (key, value) VALUES ('best_move_backfill_pending', '0 1 2')
		 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`,
	})
	return s, conn
}

// bestMoveState reads, per tenant, how many rows are still NULL and the
// xmin of every row, by id.
func bestMoveState(t *testing.T, conn *pgx.Conn) (nulls int, xmins []string) {
	t.Helper()
	ctx := context.Background()
	for _, scope := range []string{"1", "2"} {
		asTenant(t, conn, scope, func() {
			nulls += count(t, conn, `SELECT count(*) FROM analysis WHERE best_move IS NULL`)
			rows, err := conn.Query(ctx, `SELECT xmin::text FROM analysis ORDER BY id`)
			if err != nil {
				t.Fatal(err)
			}
			x, err := pgx.CollectRows(rows, pgx.RowTo[string])
			if err != nil {
				t.Fatal(err)
			}
			xmins = append(xmins, x...)
		})
	}
	return nulls, xmins
}

// TestBestMoveBackfillResumesWhereItStopped: a pass cut after a few batches
// keeps what it wrote and the next one finishes from there; once ended, a
// further pass rewrites nothing.
func TestBestMoveBackfillResumesWhereItStopped(t *testing.T) {
	ctx := context.Background()
	s, conn := seedBestMovePending(t, 7)
	done, err := pg.BackfillAnalysisBestMove(ctx, s, 3, 3)
	if err != nil || done {
		t.Fatalf("three batches of three: done %v, %v; want unfinished", done, err)
	}
	cut, _ := bestMoveState(t, conn)
	// Tenant 1's seven rows take batches of 3, 3 and 1; tenant 2 is untouched.
	if cut != 7 {
		t.Fatalf("after three batches of three, %d rows NULL, want 7", cut)
	}
	if done, err := pg.BackfillAnalysisBestMove(ctx, s, 3, 0); err != nil || !done {
		t.Fatalf("resumed pass: done %v, %v", done, err)
	}
	nulls, xmins := bestMoveState(t, conn)
	if nulls != 0 {
		t.Fatalf("%d rows NULL after the resumed pass, want 0", nulls)
	}
	var cursor *string
	if err := conn.QueryRow(ctx, `SELECT (SELECT value FROM metadata WHERE key = 'best_move_backfill_pending')`).Scan(&cursor); err != nil {
		t.Fatal(err)
	}
	if cursor != nil {
		t.Errorf("cursor %q left after the pass ended", *cursor)
	}
	if done, err := pg.BackfillAnalysisBestMove(ctx, s, 3, 0); err != nil || !done {
		t.Fatalf("pass after the end: done %v, %v", done, err)
	}
	if _, again := bestMoveState(t, conn); !slices.Equal(xmins, again) {
		t.Errorf("a pass after the end rewrote rows: xmin %v, then %v", xmins, again)
	}
}

// TestBestMoveBackfillRunsBesideAReader: a transaction holding a read of
// analysis open does not hold the pass back, nor does the pass hold it: no
// batch asks for a lock that conflicts with a read.
func TestBestMoveBackfillRunsBesideAReader(t *testing.T) {
	ctx := context.Background()
	s, conn := seedBestMovePending(t, 4)
	reader, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reader.Rollback(ctx) }()
	if _, err := reader.Exec(ctx, `SELECT count(*) FROM analysis`); err != nil {
		t.Fatal(err)
	}
	bounded, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	start := time.Now()
	if done, err := pg.BackfillAnalysisBestMove(bounded, s, 2, 0); err != nil || !done {
		t.Fatalf("pass beside an open reader: done %v, %v", done, err)
	}
	if took := time.Since(start); took > 3*time.Second {
		t.Errorf("pass beside an open reader took %v: it waited on a lock", took)
	}
	if _, err := reader.Exec(ctx, `SELECT count(*) FROM analysis`); err != nil {
		t.Fatalf("reader after the pass: %v", err)
	}
	if err := reader.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if nulls, _ := bestMoveState(t, conn); nulls != 0 {
		t.Errorf("%d rows NULL after the pass, want 0", nulls)
	}
}

// TestMigrateLeavesTheBestMoveBackfillToTheService: Migrate returns with
// every row still NULL and the cursor pending, so the daemon serves before
// the pass ends; BackfillDeferred then ends it.
func TestMigrateLeavesTheBestMoveBackfillToTheService(t *testing.T) {
	ctx := context.Background()
	s, conn := seedBestMovePending(t, 4)
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if nulls, _ := bestMoveState(t, conn); nulls != 8 {
		t.Fatalf("Migrate derived the best move: %d rows NULL, want 8", nulls)
	}
	if err := s.BackfillDeferred(ctx); err != nil {
		t.Fatalf("BackfillDeferred: %v", err)
	}
	if nulls, _ := bestMoveState(t, conn); nulls != 0 {
		t.Errorf("%d rows NULL after BackfillDeferred, want 0", nulls)
	}
}
