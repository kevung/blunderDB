//go:build postgres

// These tests provision a real PostgreSQL via testcontainers-go and therefore
// need Docker. Run with:
//
//	go test -tags postgres ./pkg/blunderdb/migrate/...
package migrate_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	tcpg "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/migrate"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

func startPostgres(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	container, err := tcpg.Run(ctx, "postgres:16-alpine",
		tcpg.WithDatabase("blunderdb"),
		tcpg.WithUsername("test"),
		tcpg.WithPassword("test"),
		tcpg.BasicWaitStrategies(),
	)
	if err != nil {
		t.Skipf("postgres container unavailable (Docker required): %v", err)
	}
	t.Cleanup(func() { _ = testcontainers.TerminateContainer(container) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	return dsn
}

// seedSQLite builds a populated SQLite source: a real XG match (positions,
// analyses, comments, match/games/moves, auto-linked tournament) plus a
// collection, and returns the open store and its counts.
func seedSQLite(t *testing.T) (storage.Storage, storage.Counts, int) {
	t.Helper()
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "src.db")

	db := database.NewDatabase()
	if err := db.SetupDatabase(dbPath); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	if _, err := db.ImportXGMatch(filepath.Join("..", "..", "..", "testdata", "charlot1-charlot2_7p_2025-11-08-2305.xg")); err != nil {
		db.Close()
		t.Fatalf("seed ImportXGMatch: %v", err)
	}
	db.Close()

	src, err := sqlite.Open(ctx, dbPath, nil)
	if err != nil {
		t.Fatalf("open source: %v", err)
	}
	t.Cleanup(func() { src.Close() })

	// Add a collection referencing the first two positions.
	cid, err := src.Collections().Create(ctx, "", "Favs", "two positions")
	if err != nil {
		t.Fatalf("create collection: %v", err)
	}
	var pids []int64
	for p, err := range src.Positions().List(ctx, "", storage.ListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		pids = append(pids, p.ID)
		if len(pids) == 2 {
			break
		}
	}
	if err := src.Collections().AddPositions(ctx, "", cid, pids); err != nil {
		t.Fatalf("add positions to collection: %v", err)
	}

	counts, err := src.Metadata().Counts(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	return src, counts, 1 // one collection
}

func collectionCount(t *testing.T, s storage.Storage, scope string) int {
	t.Helper()
	ctx := context.Background()
	n := 0
	for _, err := range s.Collections().List(ctx, scope) {
		if err != nil {
			t.Fatal(err)
		}
		n++
	}
	return n
}

// scopeCounts tallies a scope's positions/matches/games/moves by iterating the
// stores (the Postgres MetadataStore.Counts is not implemented yet).
type scopeCounts struct{ Positions, Matches, Games, Moves int }

func countByIteration(t *testing.T, s storage.Storage, scope string) scopeCounts {
	t.Helper()
	ctx := context.Background()
	var c scopeCounts
	for _, err := range s.Positions().List(ctx, scope, storage.ListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		c.Positions++
	}
	var matchIDs []int64
	for m, err := range s.Matches().List(ctx, scope) {
		if err != nil {
			t.Fatal(err)
		}
		matchIDs = append(matchIDs, m.ID)
		c.Matches++
	}
	for _, mid := range matchIDs {
		var gameIDs []int64
		for g, err := range s.Matches().Games(ctx, scope, mid) {
			if err != nil {
				t.Fatal(err)
			}
			gameIDs = append(gameIDs, g.ID)
			c.Games++
		}
		for _, gid := range gameIDs {
			for _, err := range s.Matches().Moves(ctx, scope, gid) {
				if err != nil {
					t.Fatal(err)
				}
				c.Moves++
			}
		}
	}
	return c
}

func TestMigrateSQLiteToPostgres(t *testing.T) {
	ctx := context.Background()
	src, srcCounts, srcColls := seedSQLite(t)

	dsn := startPostgres(t)
	dst, err := postgres.Open(ctx, dsn, nil)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	defer dst.Close()
	if err := dst.Migrate(ctx); err != nil {
		t.Fatalf("migrate dst schema: %v", err)
	}

	rep, err := migrate.Run(ctx, src, dst, "1", migrate.Options{})
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// The report matches the source.
	if rep.Positions != srcCounts.Positions || rep.Matches != srcCounts.Matches ||
		rep.Games != srcCounts.Games || rep.Moves != srcCounts.Moves {
		t.Fatalf("report mismatch: rep=%+v srcCounts=%+v", rep, srcCounts)
	}
	if rep.Collections != srcColls {
		t.Fatalf("collections copied = %d, want %d", rep.Collections, srcColls)
	}

	// The destination tenant holds the same data (counted by iteration, since
	// the Postgres MetadataStore.Counts is not implemented yet).
	want := scopeCounts{srcCounts.Positions, srcCounts.Matches, srcCounts.Games, srcCounts.Moves}
	if got := countByIteration(t, dst, "1"); got != want {
		t.Fatalf("dst tenant 1 counts %+v != src %+v", got, want)
	}
	if got := collectionCount(t, dst, "1"); got != srcColls {
		t.Fatalf("dst collections = %d, want %d", got, srcColls)
	}

	// Re-running into the same tenant aborts (destination not empty).
	if _, err := migrate.Run(ctx, src, dst, "1", migrate.Options{}); err == nil {
		t.Fatal("expected conflict error re-migrating into a non-empty tenant")
	}

	// A second tenant is isolated and gets its own full copy.
	if _, err := migrate.Run(ctx, src, dst, "2", migrate.Options{}); err != nil {
		t.Fatalf("migrate tenant 2: %v", err)
	}
	if got := countByIteration(t, dst, "2"); got != want {
		t.Fatalf("tenant 2 counts %+v != src %+v", got, want)
	}
	// Tenant 1 is unchanged by the tenant-2 migration.
	if got := countByIteration(t, dst, "1"); got != want {
		t.Fatalf("tenant 1 counts changed: %+v", got)
	}
}

func TestMigrateDryRun(t *testing.T) {
	src, srcCounts, _ := seedSQLite(t)
	rep, err := migrate.Run(context.Background(), src, nil, "1", migrate.Options{DryRun: true})
	if err != nil {
		t.Fatalf("dry run: %v", err)
	}
	if rep.Positions != srcCounts.Positions || rep.Matches != srcCounts.Matches {
		t.Fatalf("dry-run report %+v != source %+v", rep, srcCounts)
	}
}
