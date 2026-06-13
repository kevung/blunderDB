//go:build postgres

// Validates the PostgreSQL player filter (ILIKE clause in search_postgres.go) on
// a real database: import an XG match through the legacy Database, migrate it
// into a fresh tenant, then search by player name via the storage backend.
// Needs Docker; gated behind the `postgres` build tag.
package postgres_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/migrate"
	pg "github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

func findPlayer(t *testing.T, dst *pg.Storage, scope string, f domain.SearchFilters) int {
	t.Helper()
	n := 0
	for _, err := range dst.Search().Find(context.Background(), scope, f) {
		if err != nil {
			t.Fatalf("Search.Find: %v", err)
		}
		n++
	}
	return n
}

func TestSearchPlayerFilterPostgres(t *testing.T) {
	ctx := context.Background()
	dsn := startPostgres(t)
	const scope = ""
	resetPublicSchema(t, dsn)

	// Seed: import an XG match via the legacy Database, migrate into Postgres.
	dbPath := filepath.Join(t.TempDir(), "src.db")
	d := database.NewDatabase()
	if err := d.SetupDatabase(dbPath); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	xg := filepath.Join("..", "..", "..", "..", "testdata", "charlot1-charlot2_7p_2025-11-08-2305.xg")
	if _, err := d.ImportXGMatch(xg); err != nil {
		t.Fatalf("ImportXGMatch: %v", err)
	}
	d.Close()

	src, err := sqlite.Open(ctx, dbPath, nil)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	defer src.Close()
	dst, err := pg.Open(ctx, dsn, nil)
	if err != nil {
		t.Fatalf("pg.Open: %v", err)
	}
	defer dst.Close()
	if _, err := migrate.Run(ctx, src, dst, scope, migrate.Options{}); err != nil {
		t.Fatalf("migrate.Run: %v", err)
	}

	total := findPlayer(t, dst, scope, domain.SearchFilters{})
	if total == 0 {
		t.Fatal("no positions migrated")
	}

	// Exact name (the fixture's players are charlot1 / charlot2).
	if n := findPlayer(t, dst, scope, domain.SearchFilters{PlayerFilter: "charlot1"}); n == 0 {
		t.Errorf("PlayerFilter=charlot1 returned 0, expected the match's positions")
	}
	// ILIKE is case-insensitive.
	if n := findPlayer(t, dst, scope, domain.SearchFilters{PlayerFilter: "CHARLOT1"}); n == 0 {
		t.Errorf("PlayerFilter=CHARLOT1 (uppercase) returned 0; ILIKE should be case-insensitive")
	}
	// A name in no match returns nothing.
	if n := findPlayer(t, dst, scope, domain.SearchFilters{PlayerFilter: "ZZ_NoSuchPlayer_ZZ"}); n != 0 {
		t.Errorf("unknown player returned %d, want 0", n)
	}
}
