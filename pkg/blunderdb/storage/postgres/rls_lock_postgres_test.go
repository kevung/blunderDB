//go:build postgres

package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	pg "github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
)

// A start must not queue behind a long query: every later request would queue behind it.
func TestApplyRLSNeverWaitsOnAPolicedTable(t *testing.T) {
	ctx := context.Background()
	dsn := startPostgres(t)
	s, err := pg.Open(ctx, dsn, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := s.ApplyRLS(ctx); err != nil {
		t.Fatalf("ApplyRLS: %v", err)
	}

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)
	longQuery, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer longQuery.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op
	if _, err := longQuery.Exec(ctx, `LOCK TABLE position IN ACCESS SHARE MODE`); err != nil {
		t.Fatal(err)
	}

	bounded, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := s.ApplyRLS(bounded); err != nil {
		t.Fatalf("ApplyRLS on policed tables behind a long query: %v", err)
	}

	// A table that does need it gives way rather than freeze the service.
	other, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close(ctx)
	if err := longQuery.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := other.Exec(ctx, `ALTER TABLE position NO FORCE ROW LEVEL SECURITY`); err != nil {
		t.Fatal(err)
	}
	if longQuery, err = conn.Begin(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := longQuery.Exec(ctx, `LOCK TABLE position IN ACCESS SHARE MODE`); err != nil {
		t.Fatal(err)
	}
	var pgErr *pgconn.PgError
	if err := s.ApplyRLS(ctx); !errors.As(err, &pgErr) || pgErr.Code != "55P03" {
		t.Fatalf("ApplyRLS on a busy table to police: %v, want lock_not_available", err)
	}
	if err := longQuery.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	if err := s.ApplyRLS(ctx); err != nil {
		t.Fatalf("ApplyRLS once free: %v", err)
	}
	var forced bool
	if err := other.QueryRow(ctx, `SELECT relforcerowsecurity FROM pg_class WHERE oid = 'position'::regclass`).Scan(&forced); err != nil || !forced {
		t.Fatalf("position forced %v, err %v", forced, err)
	}
}
