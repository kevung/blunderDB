//go:build postgres

package postgres_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"

	pg "github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
)

// A fence holds while any transaction running when it was taken is still
// open — one that wrote (it holds a transaction id) as well as one that has
// not yet (only its start names it) — and clears once both have ended,
// however long they lasted.
func TestSyncFenceWaitsForEveryOpenTransaction(t *testing.T) {
	ctx := context.Background()
	dsn := startPostgres(t)
	s, err := pg.Open(ctx, dsn, nil)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	begin := func(withID bool) pgx.Tx {
		conn, err := pgx.Connect(ctx, dsn)
		if err != nil {
			t.Fatalf("connect: %v", err)
		}
		t.Cleanup(func() { _ = conn.Close(ctx) })
		tx, err := conn.Begin(ctx)
		if err != nil {
			t.Fatalf("begin: %v", err)
		}
		q := "SELECT 1"
		if withID {
			q = "SELECT pg_current_xact_id()::text"
		}
		if _, err := tx.Exec(ctx, q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
		return tx
	}
	cleared := func(token string) bool {
		t.Helper()
		ok, err := s.SyncFenceCleared(ctx, token)
		if err != nil {
			t.Fatalf("cleared: %v", err)
		}
		return ok
	}

	writer, reader := begin(true), begin(false)
	token, err := s.SyncFence(ctx)
	if err != nil {
		t.Fatalf("fence: %v", err)
	}
	// A transaction begun after the fence does not hold it.
	later := begin(true)
	defer func() { _ = later.Rollback(ctx) }()

	if cleared(token) {
		t.Fatal("fence cleared while both transactions run")
	}
	if err := writer.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if cleared(token) {
		t.Fatal("fence cleared while the transaction without an id still runs")
	}
	if err := reader.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	if !cleared(token) {
		t.Fatal("fence still holds once every transaction it named has ended")
	}
	if cleared("not a token") {
		t.Fatal("an unreadable token must not clear")
	}
}
