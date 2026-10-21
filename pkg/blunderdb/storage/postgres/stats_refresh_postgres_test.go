//go:build postgres

package postgres_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	pg "github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
)

// TestRefreshStats_RefusesARoleThatMayNotAnalyze checks the owner refreshes
// and a role without the right to ANALYZE is refused with the reason, where
// ANALYZE alone would only warn and leave the quadratic plan in place.
func TestRefreshStats_RefusesARoleThatMayNotAnalyze(t *testing.T) {
	ctx := context.Background()
	dsn := startPostgres(t)
	owner, err := pg.Open(ctx, dsn, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer owner.Close()
	if err := owner.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	tx, err := owner.BeginTx(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	rs, ok := tx.(storage.StatsRefresher)
	if !ok {
		t.Fatal("the postgres transaction does not refresh its statistics")
	}
	if err := rs.RefreshStats(ctx); err != nil {
		t.Fatalf("RefreshStats: %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	for _, stmt := range []string{
		`DROP ROLE IF EXISTS stats_guest`,
		`CREATE ROLE stats_guest LOGIN PASSWORD 'guest'`,
		`GRANT USAGE, CREATE ON SCHEMA public TO stats_guest`,
		`GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO stats_guest`,
		`GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO stats_guest`,
	} {
		if _, err := conn.Exec(ctx, stmt); err != nil {
			t.Fatalf("setup role (%s): %v", stmt, err)
		}
	}
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	guest, err := pg.Open(ctx, roleDSN(cfg, "stats_guest", "guest"), nil)
	if err != nil {
		t.Fatalf("Open as stats_guest: %v", err)
	}
	defer guest.Close()
	gtx, err := guest.BeginTx(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = gtx.Rollback() }()
	err = gtx.(storage.StatsRefresher).RefreshStats(ctx)
	if err == nil || !strings.Contains(err.Error(), "may not ANALYZE") {
		t.Fatalf("RefreshStats as a non-owner: %v, want a refusal naming ANALYZE", err)
	}
}
