//go:build postgres

package postgres_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/searchquery"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	pg "github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// TestMatchRestReadUnderRLS searches, as a role row-level security binds, a
// match-level filter joined to other filters, read from those filters first
// or not: each tenant finds the same positions both ways, and only its own.
func TestMatchRestReadUnderRLS(t *testing.T) {
	ctx := context.Background()
	dsn := startPostgres(t)
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
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	for _, stmt := range []string{
		`DROP ROLE IF EXISTS rls_matchrest`,
		`CREATE ROLE rls_matchrest LOGIN PASSWORD 'app'`,
		`GRANT USAGE, CREATE ON SCHEMA public TO rls_matchrest`,
		`GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO rls_matchrest`,
		`GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO rls_matchrest`,
	} {
		if _, err := conn.Exec(ctx, stmt); err != nil {
			t.Fatalf("setup role (%s): %v", stmt, err)
		}
	}
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	app, err := pg.Open(ctx, roleDSN(cfg, "rls_matchrest", "app"), &storage.Options{EnableRLS: true})
	if err != nil {
		t.Fatalf("Open as rls_matchrest: %v", err)
	}
	defer app.Close()

	file := filepath.Join("..", "..", "..", "..", "testdata", "test.xg")
	for _, tenant := range []int64{1, 2} {
		c := storage.WithTenant(ctx, tenant)
		if _, err := (ingest.XGImporter{S: app}).Import(c, fmt.Sprint(tenant), ingest.Source{Path: file}, nil); err != nil {
			t.Fatalf("import (tenant %d): %v", tenant, err)
		}
	}
	var player string
	if err := conn.QueryRow(ctx, `SELECT player1_name FROM match WHERE tenant_id = 1 ORDER BY id LIMIT 1`).Scan(&player); err != nil {
		t.Fatal(err)
	}

	defer sqlshared.SetIDWalkProbe(8)()
	defer sqlshared.SetMatchListLimit(10)()
	seen := map[int64]int64{}
	for _, command := range []string{`s pl"%s" D`, `s pl"%s" cube`, `s pl"%s" nc`} {
		f, err := searchquery.Parse(fmt.Sprintf(command, player))
		if err != nil {
			t.Fatal(err)
		}
		for _, tenant := range []int64{1, 2} {
			c, scope := storage.WithTenant(ctx, tenant), fmt.Sprint(tenant)
			restore := sqlshared.SetMatchRestLimit(0)
			want, err := app.Search().FindIDs(c, scope, f, storage.ListOpts{Limit: 1000})
			restore()
			if err != nil {
				t.Fatal(err)
			}
			got, err := app.Search().FindIDs(c, scope, f, storage.ListOpts{Limit: 1000})
			if err != nil {
				t.Fatal(err)
			}
			if fmt.Sprint(got) != fmt.Sprint(want) {
				t.Errorf("tenant %d, %s: read first %v, without %v", tenant, command, got, want)
			}
			for _, id := range got {
				if owner, ok := seen[id]; ok && owner != tenant {
					t.Errorf("position %d found by tenants %d and %d", id, owner, tenant)
				}
				seen[id] = tenant
			}
		}
	}
	if len(seen) == 0 {
		t.Fatal("no search found a position: the fixture tells nothing")
	}
}
