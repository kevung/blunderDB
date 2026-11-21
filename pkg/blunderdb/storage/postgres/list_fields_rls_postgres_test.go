//go:build postgres

package postgres_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	pg "github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
)

// TestListFieldsUnderRLS reads list fields as a role row-level security
// binds: ids of another tenant name nothing, and a tenant's collections and
// study decks are never listed against another's position.
func TestListFieldsUnderRLS(t *testing.T) {
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
		`DROP ROLE IF EXISTS rls_listfields`,
		`CREATE ROLE rls_listfields LOGIN PASSWORD 'app'`,
		`GRANT USAGE, CREATE ON SCHEMA public TO rls_listfields`,
		`GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO rls_listfields`,
		`GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO rls_listfields`,
	} {
		if _, err := conn.Exec(ctx, stmt); err != nil {
			t.Fatalf("setup role (%s): %v", stmt, err)
		}
	}
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	app, err := pg.Open(ctx, roleDSN(cfg, "rls_listfields", "app"), &storage.Options{EnableRLS: true})
	if err != nil {
		t.Fatalf("Open as rls_listfields: %v", err)
	}
	defer app.Close()

	seed := func(tenant int64, scope string, i int) int64 {
		c := storage.WithTenant(ctx, tenant)
		p := rlsPos(i)
		id, err := app.Positions().Save(c, scope, &p)
		if err != nil {
			t.Fatalf("Save (tenant %d): %v", tenant, err)
		}
		col, err := app.Collections().Create(c, scope, "of "+scope, "")
		if err != nil {
			t.Fatal(err)
		}
		if err := app.Collections().AddPosition(c, scope, col, id); err != nil {
			t.Fatal(err)
		}
		deck, err := app.Anki().CreateDeck(c, scope, "deck of "+scope, "", domain.AnkiSourceSearch, 0, "")
		if err != nil {
			t.Fatal(err)
		}
		if err := app.Anki().SyncWithPositions(c, scope, deck, []int64{id}); err != nil {
			t.Fatal(err)
		}
		return id
	}
	mine, theirs := seed(1, "1", 1), seed(2, "2", 2)

	lf := app.PositionListFields()
	for _, c := range []struct {
		tenant int64
		scope  string
		own    int64
	}{{1, "1", mine}, {2, "2", theirs}} {
		got, err := lf.ListFields(storage.WithTenant(ctx, c.tenant), c.scope, []int64{mine, theirs})
		if err != nil {
			t.Fatalf("tenant %s: %v", c.scope, err)
		}
		if len(got) != 1 || got[0].PositionID != c.own {
			t.Fatalf("tenant %s read %+v, want only its own position %d", c.scope, got, c.own)
		}
		if len(got[0].Collections) != 1 || got[0].Collections[0].Name != "of "+c.scope {
			t.Errorf("tenant %s collections = %+v, want only its own", c.scope, got[0].Collections)
		}
		if len(got[0].Decks) != 1 || got[0].Decks[0].Name != "deck of "+c.scope {
			t.Errorf("tenant %s decks = %+v, want only its own", c.scope, got[0].Decks)
		}
	}
}
