//go:build postgres

// Needs Docker (testcontainers):
//
//	go test -tags postgres ./pkg/blunderdb/server/ -run TestBootstrapDeferred -v
package server

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go"
	tcpg "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	pg "github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
)

// seedDeferredPending migrates a fresh database, stores analyses in tenants
// 1 and 2, and leaves them as an upgrade to 0.63 finds them: no best move
// index, best_move NULL, both tenants pending.
func seedDeferredPending(t *testing.T) (string, *pgx.Conn) {
	t.Helper()
	ctx := context.Background()
	container, err := tcpg.Run(ctx, "postgres:16-alpine",
		tcpg.WithDatabase("blunderdb"), tcpg.WithUsername("test"), tcpg.WithPassword("test"),
		tcpg.BasicWaitStrategies())
	if err != nil {
		t.Skipf("postgres container unavailable (Docker required): %v", err)
	}
	t.Cleanup(func() { _ = testcontainers.TerminateContainer(container) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	s, err := pg.Open(ctx, dsn, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	analysis := &domain.PositionAnalysis{AnalysisType: "CheckerMove", CheckerAnalysis: &domain.CheckerAnalysis{
		Moves: []domain.CheckerMove{{Index: 0, Move: "24/20 13/11", Equity: 0.1}}}}
	for i, scope := range []string{"1", "2"} {
		for j := range 4 {
			p := domain.InitializePosition()
			p.Board.Points[1+i].Checkers = 1 + j
			pid, err := s.Positions().Save(ctx, scope, &p)
			if err != nil {
				t.Fatal(err)
			}
			if err := s.Analyses().Save(ctx, scope, pid, analysis); err != nil {
				t.Fatal(err)
			}
		}
	}
	s.Close()

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	for _, q := range []string{
		`DROP INDEX IF EXISTS idx_analysis_best_move`,
		`UPDATE analysis SET best_move = NULL`,
		`INSERT INTO metadata (key, value) VALUES ('best_move_backfill_pending', '0 1 2')
		 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`,
	} {
		if _, err := conn.Exec(ctx, q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	return dsn, conn
}

// deferredState is nil without idx_analysis_best_move, else its indisvalid,
// and the count of analyses whose best_move is still NULL.
func deferredState(t *testing.T, conn *pgx.Conn) (valid *bool, nulls int) {
	t.Helper()
	if err := conn.QueryRow(context.Background(), `SELECT
		(SELECT indisvalid FROM pg_index WHERE indexrelid = to_regclass('idx_analysis_best_move')),
		(SELECT count(*) FROM analysis WHERE best_move IS NULL)`).Scan(&valid, &nulls); err != nil {
		t.Fatal(err)
	}
	return valid, nulls
}

func waitDeferred(t *testing.T, c any) {
	t.Helper()
	select {
	case <-c.(*closer).deferred:
	case <-time.After(60 * time.Second):
		t.Fatal("the deferred passes did not end")
	}
}

// TestBootstrapDeferredBackfills: the embedded engine serves before the
// passes Migrate defers, then builds the best move index and fills the column.
func TestBootstrapDeferredBackfills(t *testing.T) {
	dsn, conn := seedDeferredPending(t)
	_, c, err := Bootstrap(context.Background(), Config{Backend: "postgres", DSN: dsn})
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	defer func() { _ = c.Close() }()
	waitDeferred(t, c)
	if valid, nulls := deferredState(t, conn); valid == nil || !*valid || nulls != 0 {
		t.Errorf("after the deferred passes: index valid %v, %d rows NULL; want valid, 0", valid, nulls)
	}
}

// TestBootstrapCloseStopsDeferredBackfills: Close interrupts a pass that
// cannot end (an open snapshot holds CREATE INDEX CONCURRENTLY) without an
// error, and the next start resumes it.
func TestBootstrapCloseStopsDeferredBackfills(t *testing.T) {
	ctx := context.Background()
	dsn, conn := seedDeferredPending(t)
	holder, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = holder.Close(ctx) }()
	snapshot, err := holder.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := snapshot.Exec(ctx, `SELECT count(*) FROM analysis`); err != nil {
		t.Fatal(err)
	}

	_, c, err := Bootstrap(ctx, Config{Backend: "postgres", DSN: dsn})
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	closed := make(chan error, 1)
	go func() { closed <- c.Close() }()
	select {
	case err := <-closed:
		if err != nil {
			t.Fatalf("Close during the pass: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("Close did not stop the pass")
	}
	if valid, _ := deferredState(t, conn); valid != nil && *valid {
		t.Fatal("the index was built behind an open snapshot")
	}

	if err := snapshot.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	_, c, err = Bootstrap(ctx, Config{Backend: "postgres", DSN: dsn})
	if err != nil {
		t.Fatalf("second Bootstrap: %v", err)
	}
	defer func() { _ = c.Close() }()
	waitDeferred(t, c)
	if valid, nulls := deferredState(t, conn); valid == nil || !*valid || nulls != 0 {
		t.Errorf("after the resumed passes: index valid %v, %d rows NULL; want valid, 0", valid, nulls)
	}
}
