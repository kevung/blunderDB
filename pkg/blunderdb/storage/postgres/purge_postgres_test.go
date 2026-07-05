//go:build postgres

// TestPurgeTenant and friends provision a real PostgreSQL via
// testcontainers-go and therefore need Docker, exactly like every other
// postgres-tagged test in this package (see postgres_test.go):
//
//	go test -tags postgres ./pkg/blunderdb/storage/postgres/... -run TestPurgeTenant -v
//
// This file is `package postgres` (white-box), not `postgres_test`, because
// TestPurgeOrderMatchesRLSTables needs direct access to the two unexported
// package-level variables it guards (purgeOrder in purge_postgres.go,
// rlsTables in rls_postgres.go). That means the usual startPostgres/
// resetPublicSchema helpers from postgres_test.go (a different package,
// even though compiled into the same test binary) are not reachable from
// here, so purgeTestDB/purgeResetSchema below are self-contained local
// equivalents rather than reuses.
package postgres

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpg "github.com/testcontainers/testcontainers-go/modules/postgres"
)

// purgeTestDB boots a throwaway PostgreSQL 16 container and returns its DSN.
// The test is skipped (not failed) when Docker is unavailable.
func purgeTestDB(t *testing.T) string {
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

// purgeResetSchema drops every object in the public schema, giving the next
// Open a fresh database to bootstrap.
func purgeResetSchema(t *testing.T, dsn string) {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("reset connect: %v", err)
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public`); err != nil {
		t.Fatalf("reset schema: %v", err)
	}
}

// purgeSeedRows inserts exactly one row per rlsTables-covered table for
// tenantID, respecting every FK constraint — parents before children, the
// mirror image of PurgeTenant's own child-before-parent delete order.
func purgeSeedRows(t *testing.T, pool *pgxpool.Pool, tenantID int64) {
	t.Helper()
	ctx := context.Background()

	scalar := func(sql string, args ...any) int64 {
		t.Helper()
		var id int64
		if err := pool.QueryRow(ctx, sql, args...).Scan(&id); err != nil {
			t.Fatalf("seed insert %q: %v", sql, err)
		}
		return id
	}
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("seed insert %q: %v", sql, err)
		}
	}

	positionID := scalar(`INSERT INTO position (tenant_id, state) VALUES ($1, 'x') RETURNING id`, tenantID)
	scalar(`INSERT INTO analysis (tenant_id, position_id) VALUES ($1, $2) RETURNING id`, tenantID, positionID)
	scalar(`INSERT INTO comment (tenant_id, position_id, text) VALUES ($1, $2, 'c') RETURNING id`, tenantID, positionID)
	exec(`INSERT INTO filter_library (tenant_id, name, command) VALUES ($1, 'f', 'cmd')`, tenantID)
	exec(`INSERT INTO command_history (tenant_id, command) VALUES ($1, 'cmd')`, tenantID)
	exec(`INSERT INTO search_history (tenant_id, command, position, timestamp) VALUES ($1, 'cmd', 'pos', 0)`, tenantID)

	tournamentID := scalar(`INSERT INTO tournament (tenant_id, name) VALUES ($1, 't') RETURNING id`, tenantID)
	matchID := scalar(`INSERT INTO match (tenant_id, player1_name, tournament_id) VALUES ($1, 'p1', $2) RETURNING id`, tenantID, tournamentID)
	gameID := scalar(`INSERT INTO game (tenant_id, match_id, game_number) VALUES ($1, $2, 1) RETURNING id`, tenantID, matchID)
	moveID := scalar(`INSERT INTO move (tenant_id, game_id, position_id, move_number) VALUES ($1, $2, $3, 1) RETURNING id`, tenantID, gameID, positionID)
	scalar(`INSERT INTO move_analysis (tenant_id, move_id, analysis_type) VALUES ($1, $2, 'a') RETURNING id`, tenantID, moveID)

	collectionID := scalar(`INSERT INTO collection (tenant_id, name) VALUES ($1, 'coll') RETURNING id`, tenantID)
	exec(`INSERT INTO collection_position (tenant_id, collection_id, position_id) VALUES ($1, $2, $3)`, tenantID, collectionID, positionID)

	deckID := scalar(`INSERT INTO anki_deck (tenant_id, name) VALUES ($1, 'deck') RETURNING id`, tenantID)
	cardID := scalar(`INSERT INTO anki_card (tenant_id, deck_id, position_id) VALUES ($1, $2, $3) RETURNING id`, tenantID, deckID, positionID)
	exec(`INSERT INTO anki_review_log (tenant_id, card_id, deck_id, position_id, rating) VALUES ($1, $2, $3, $4, 1)`, tenantID, cardID, deckID, positionID)
}

// purgeCountRows returns the number of rows in table belonging to tenantID.
func purgeCountRows(t *testing.T, pool *pgxpool.Pool, table string, tenantID int64) int {
	t.Helper()
	var n int
	sql := fmt.Sprintf(`SELECT count(*) FROM %s WHERE tenant_id = $1`, table)
	if err := pool.QueryRow(context.Background(), sql, tenantID).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

// TestPurgeTenant seeds one row per rlsTables-covered table for two tenants,
// purges tenant A, and asserts every one of tenant A's rows is gone while
// tenant B's rows of the same tables are untouched. It also purges tenant A
// a second time (idempotency: no error, zero rows affected either time).
func TestPurgeTenant(t *testing.T) {
	ctx := context.Background()
	dsn := purgeTestDB(t)
	purgeResetSchema(t, dsn)
	s, err := Open(ctx, dsn, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	const tenantA, tenantB int64 = 101, 202
	purgeSeedRows(t, s.pool, tenantA)
	purgeSeedRows(t, s.pool, tenantB)

	for _, tbl := range rlsTables {
		if got := purgeCountRows(t, s.pool, tbl, tenantA); got != 1 {
			t.Fatalf("seed sanity: %s tenant A: got %d rows, want 1", tbl, got)
		}
		if got := purgeCountRows(t, s.pool, tbl, tenantB); got != 1 {
			t.Fatalf("seed sanity: %s tenant B: got %d rows, want 1", tbl, got)
		}
	}

	if err := s.PurgeTenant(ctx, tenantA); err != nil {
		t.Fatalf("PurgeTenant: %v", err)
	}

	for _, tbl := range rlsTables {
		if got := purgeCountRows(t, s.pool, tbl, tenantA); got != 0 {
			t.Errorf("after purge: %s tenant A: got %d rows, want 0", tbl, got)
		}
		if got := purgeCountRows(t, s.pool, tbl, tenantB); got != 1 {
			t.Errorf("after purge: %s tenant B: got %d rows, want 1 (untouched)", tbl, got)
		}
	}

	// Idempotency: purging an already-purged tenant is a harmless no-op.
	if err := s.PurgeTenant(ctx, tenantA); err != nil {
		t.Fatalf("second PurgeTenant (idempotency): %v", err)
	}
	for _, tbl := range rlsTables {
		if got := purgeCountRows(t, s.pool, tbl, tenantA); got != 0 {
			t.Errorf("after second purge: %s tenant A: got %d rows, want 0", tbl, got)
		}
	}
}

// TestPurgeTenantEmpty confirms purging a tenant with no data at all (never
// provisioned) succeeds without error.
func TestPurgeTenantEmpty(t *testing.T) {
	ctx := context.Background()
	dsn := purgeTestDB(t)
	purgeResetSchema(t, dsn)
	s, err := Open(ctx, dsn, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	if err := s.PurgeTenant(ctx, 999); err != nil {
		t.Fatalf("PurgeTenant on a tenant with no data: %v", err)
	}
}

// TestPurgeOrderMatchesRLSTables guards against purgeOrder (purge_postgres.go)
// and rlsTables (rls_postgres.go) silently drifting apart: sorting both and
// comparing catches a table added to one list but not the other, in either
// direction. This test needs no database.
func TestPurgeOrderMatchesRLSTables(t *testing.T) {
	got := slices.Clone(purgeOrder)
	want := slices.Clone(rlsTables)
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("purgeOrder and rlsTables have drifted apart:\n purgeOrder (sorted) = %v\n rlsTables  (sorted) = %v", got, want)
	}
}
