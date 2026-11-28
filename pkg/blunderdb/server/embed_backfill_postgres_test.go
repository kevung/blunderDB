//go:build postgres

// Needs Docker (testcontainers):
//
//	go test -tags postgres ./pkg/blunderdb/server/ -run TestBootstrapDeferred -v
package server

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
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

// waitActivity waits until a backend of pg_stat_activity matches where.
func waitActivity(t *testing.T, conn *pgx.Conn, where string) {
	t.Helper()
	for deadline := time.Now().Add(30 * time.Second); ; {
		var seen bool
		if err := conn.QueryRow(context.Background(),
			`SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE pid <> pg_backend_pid() AND `+where+`)`).Scan(&seen); err != nil {
			t.Fatal(err)
		}
		if seen {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("no backend where %s", where)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func waitDeferred(t *testing.T, c io.Closer) {
	t.Helper()
	select {
	case <-c.(*closer).deferred:
	case <-time.After(60 * time.Second):
		t.Fatal("the deferred passes did not end")
	}
}

// TestBootstrapDeferredBackfills: Bootstrap returns before the passes Migrate
// defers, which then build the best move index and fill the column, past the
// end of the setup ctx.
func TestBootstrapDeferredBackfills(t *testing.T) {
	dsn, conn := seedDeferredPending(t)
	// A host may bound the setup ctx to the start: the passes outlive it.
	setup, cancel := context.WithCancel(context.Background())
	_, c, err := Bootstrap(setup, Config{Backend: "postgres", DSN: dsn})
	cancel()
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	defer func() { _ = c.Close() }()
	waitDeferred(t, c)
	if valid, nulls := deferredState(t, conn); valid == nil || !*valid || nulls != 0 {
		t.Errorf("after the deferred passes: index valid %v, %d rows NULL; want valid, 0", valid, nulls)
	}
}

// TestBootstrapCloseStopsDeferredBackfills: the engine serves while a pass
// cannot end (an open snapshot holds CREATE INDEX CONCURRENTLY), Close
// interrupts that pass without an error, and the next start resumes it.
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

	h, c, err := Bootstrap(ctx, Config{Backend: "postgres", DSN: dsn})
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	// Close must meet the build itself, not a pass that has not reached it.
	waitActivity(t, conn, `query LIKE 'CREATE INDEX CONCURRENTLY%idx_analysis_best_move%' AND wait_event_type IS NOT NULL`)
	ts := httptest.NewServer(h)
	defer ts.Close()
	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("healthz during the build: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("healthz during the build: %d, want 200", resp.StatusCode)
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

// TestBootstrapDeferredWaitsForAnotherBuilder: an engine that finds another
// process holding the index build waits for it, then builds what that
// process left unbuilt.
func TestBootstrapDeferredWaitsForAnotherBuilder(t *testing.T) {
	ctx := context.Background()
	dsn, conn := seedDeferredPending(t)
	other, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = other.Close(ctx) }()
	if _, err := other.Exec(ctx, `SELECT pg_advisory_lock($1)`, int64(0x5f62657374696478)); err != nil {
		t.Fatal(err)
	}
	_, c, err := Bootstrap(ctx, Config{Backend: "postgres", DSN: dsn})
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	defer func() { _ = c.Close() }()
	waitActivity(t, conn, `wait_event = 'advisory' AND query LIKE '%pg_advisory_lock%'`)
	if err := other.Close(ctx); err != nil {
		t.Fatal(err)
	}
	waitDeferred(t, c)
	if valid, nulls := deferredState(t, conn); valid == nil || !*valid || nulls != 0 {
		t.Errorf("after the other builder left: index valid %v, %d rows NULL; want valid, 0", valid, nulls)
	}
}
