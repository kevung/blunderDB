//go:build postgres

package postgres_test

import (
	"context"
	"net/url"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	pg "github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/storagetest"
)

// TestTenantWriteIsolation runs storagetest's by-id write cases — tenant b
// writing to tenant a's rows — with the application filter as the only guard
// (row-level security off, the default).
func TestTenantWriteIsolation(t *testing.T) {
	dsn := startPostgres(t)
	storagetest.RunTenantWriteIsolationTests(t, func() storage.Storage {
		resetPublicSchema(t, dsn)
		s, err := pg.Open(context.Background(), dsn, nil)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		return s
	}, nil, "101", "102")
}

// TestTenantWriteIsolation_RLS runs the same cases with row-level security
// enforced, connected as a non-superuser role (a superuser bypasses RLS even
// when FORCEd), each scope's calls carrying its tenant in the context.
func TestTenantWriteIsolation_RLS(t *testing.T) {
	ctx := context.Background()
	dsn := startPostgres(t)
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse dsn: %v", err)
	}
	u.User = url.UserPassword("rls_writer", "writer")
	appDSN := u.String()

	storagetest.RunTenantWriteIsolationTests(t, func() storage.Storage {
		resetPublicSchema(t, dsn)
		owner, err := pg.Open(ctx, dsn, nil)
		if err != nil {
			t.Fatalf("Open as owner: %v", err)
		}
		if err := owner.ApplyRLS(ctx); err != nil {
			t.Fatalf("ApplyRLS: %v", err)
		}
		owner.Close()

		admin, err := pgx.Connect(ctx, dsn)
		if err != nil {
			t.Fatalf("admin connect: %v", err)
		}
		defer admin.Close(ctx)
		for _, stmt := range []string{
			`DO $$ BEGIN CREATE ROLE rls_writer LOGIN PASSWORD 'writer' NOSUPERUSER NOBYPASSRLS;
			 EXCEPTION WHEN duplicate_object THEN NULL; END $$`,
			// CREATE only because Open runs the migration runner, whose
			// CREATE TABLE IF NOT EXISTS checks the right before the table.
			`GRANT USAGE, CREATE ON SCHEMA public TO rls_writer`,
			`GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO rls_writer`,
			`GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO rls_writer`,
		} {
			if _, err := admin.Exec(ctx, stmt); err != nil {
				t.Fatalf("setup role (%s): %v", stmt, err)
			}
		}

		s, err := pg.Open(ctx, appDSN, &storage.Options{EnableRLS: true})
		if err != nil {
			t.Fatalf("Open as rls_writer: %v", err)
		}
		var user string
		var super bool
		if err := admin.QueryRow(ctx, `SELECT rolname, rolsuper FROM pg_roles WHERE rolname = 'rls_writer'`).Scan(&user, &super); err != nil || super {
			t.Fatalf("rls_writer role: %q super=%v, %v", user, super, err)
		}
		return s
	}, func(scope string) context.Context {
		n, err := storage.ParseTenant(scope)
		if err != nil {
			t.Fatalf("ParseTenant(%q): %v", scope, err)
		}
		return storage.WithTenant(ctx, n)
	}, "101", "102")
}
