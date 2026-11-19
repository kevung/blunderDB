//go:build postgres

package postgres_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	pg "github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/storagetest"
)

// TestReadAcross_PostgresRLS runs the read-across checks with row-level
// security in force: the store connects as a non-superuser role (superusers
// bypass RLS) with EnableRLS, so each call sees only the tenant its context
// carries. A read across tenants that did not scope each call's context would
// read nothing for the tenants after the first.
func TestReadAcross_PostgresRLS(t *testing.T) {
	dsn := startPostgres(t)
	storagetest.RunReadAcrossTests(t, func() storage.Storage {
		return restrictedRLSStore(t, dsn, "rls_across")
	}, []string{"201", "202"}, "203")
}

// restrictedRLSStore resets dsn's schema, migrates it and applies RLS as the
// superuser, then opens it as role, a non-superuser with the grants a
// deployment gives the daemon's role: superusers bypass RLS, so only such a
// store holds the policies.
func restrictedRLSStore(t *testing.T, dsn, role string) storage.Storage {
	t.Helper()
	ctx := context.Background()
	resetPublicSchema(t, dsn)
	admin, err := pg.Open(ctx, dsn, &storage.Options{EnableRLS: true})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := admin.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := admin.ApplyRLS(ctx); err != nil {
		t.Fatalf("ApplyRLS: %v", err)
	}
	admin.Close()

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("admin connect: %v", err)
	}
	defer conn.Close(ctx)
	for _, stmt := range []string{
		`DROP ROLE IF EXISTS ` + role,
		`CREATE ROLE ` + role + ` LOGIN PASSWORD 'app'`,
		`GRANT USAGE, CREATE ON SCHEMA public TO ` + role, // Open ensures schema_migrations exists
		`GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO ` + role,
		`GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO ` + role,
	} {
		if _, err := conn.Exec(ctx, stmt); err != nil {
			t.Fatalf("setup role (%s): %v", stmt, err)
		}
	}
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parse dsn: %v", err)
	}
	app, err := pg.Open(ctx, roleDSN(cfg, role, "app"), &storage.Options{EnableRLS: true})
	if err != nil {
		t.Fatalf("Open as %s: %v", role, err)
	}
	return app
}

// roleDSN is cfg's database reached as user/password.
func roleDSN(cfg *pgx.ConnConfig, user, password string) string {
	return "postgres://" + user + ":" + password + "@" + cfg.Host + ":" + strconv.Itoa(int(cfg.Port)) + "/" + cfg.Database + "?sslmode=disable"
}
