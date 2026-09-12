//go:build postgres

// TestPurgeTenant and friends provision a real PostgreSQL via
// testcontainers-go and therefore need Docker, exactly like every other
// postgres-tagged test in this package (see postgres_test.go):
//
//	go test -tags postgres ./pkg/blunderdb/storage/postgres/... -run TestPurgeTenant -v
//
// This file is `package postgres` (white-box) because purgeSeedRows below
// needs direct access to s.pool. The one test that instead needed direct
// access to the unexported purgeOrder/rlsTables variables and no database,
// TestPurgeOrderMatchesRLSTables, lives in purge_order_test.go (untagged, so
// it runs on the default CI path) rather than here. That means the usual
// startPostgres/resetPublicSchema helpers from postgres_test.go (a different
// package, even though compiled into the same test binary) are not reachable
// from here, so purgeTestDB/purgeResetSchema below are self-contained local
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

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
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

// purgeSeedRows inserts exactly one row per tenant-scoped table for tenantID,
// respecting every FK constraint — parents before children, the mirror image
// of PurgeTenant's own child-before-parent delete order. TestPurgeTenant reads
// the table set from the live schema and fails its seed sanity check on any
// table this function leaves empty, so a new tenant-scoped table must be
// seeded here before the purge can be said to cover it.
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
	exec(`INSERT INTO session_state (tenant_id, key, value) VALUES ($1, 'seed', 'v')`, tenantID)
	exec(`INSERT INTO library_settings (tenant_id, key, value) VALUES ($1, 'seed', 'v')`, tenantID)

	exec(`INSERT INTO trash (tenant_id, kind, payload) VALUES ($1, 'match', '{}')`, tenantID)

	tournamentID := scalar(`INSERT INTO tournament (tenant_id, name) VALUES ($1, 't') RETURNING id`, tenantID)
	exec(`INSERT INTO direction (tournament_id, tenant_id) VALUES ($1, $2)`, tournamentID, tenantID)
	exec(`INSERT INTO direction_event (tournament_id, tenant_id, seq, kind, time, payload) VALUES ($1, $2, 1, 'created', now(), '{}')`, tournamentID, tenantID)
	batchID := scalar(`INSERT INTO import_batch (tenant_id) VALUES ($1) RETURNING id`, tenantID)
	matchID := scalar(`INSERT INTO match (tenant_id, player1_name, tournament_id, import_batch_id) VALUES ($1, 'p1', $2, $3) RETURNING id`, tenantID, tournamentID, batchID)
	gameID := scalar(`INSERT INTO game (tenant_id, match_id, game_number) VALUES ($1, $2, 1) RETURNING id`, tenantID, matchID)
	moveID := scalar(`INSERT INTO move (tenant_id, game_id, position_id, move_number) VALUES ($1, $2, $3, 1) RETURNING id`, tenantID, gameID, positionID)
	scalar(`INSERT INTO move_analysis (tenant_id, move_id, analysis_type) VALUES ($1, $2, 'a') RETURNING id`, tenantID, moveID)
	exec(`INSERT INTO transcription (tenant_id, format_version, match_id, label, document) VALUES ($1, '1', $2, 'draft', '{}')`, tenantID, matchID)

	collectionID := scalar(`INSERT INTO collection (tenant_id, name) VALUES ($1, 'coll') RETURNING id`, tenantID)
	exec(`INSERT INTO collection_position (tenant_id, collection_id, position_id) VALUES ($1, $2, $3)`, tenantID, collectionID, positionID)

	sessionID := scalar(`INSERT INTO training_session (tenant_id, exercise) VALUES ($1, 'scores') RETURNING id`, tenantID)
	exec(`INSERT INTO training_item (tenant_id, session_id, number_type) VALUES ($1, $2, 'tp4.last')`, tenantID, sessionID)

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

// purgeQueryNames runs a query returning one text column and collects it.
func purgeQueryNames(t *testing.T, pool *pgxpool.Pool, sql string) []string {
	t.Helper()
	rows, err := pool.Query(context.Background(), sql)
	if err != nil {
		t.Fatalf("query %q: %v", sql, err)
	}
	names, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatalf("collect %q: %v", sql, err)
	}
	return names
}

// purgeTenantTables returns, sorted, every base table of the live schema that
// carries a tenant_id column — read from information_schema rather than from
// rlsTables or purgeOrder, so that a table both lists forget is still counted
// (#363: trash and import_batch were missing from both).
func purgeTenantTables(t *testing.T, pool *pgxpool.Pool) []string {
	t.Helper()
	tables := purgeQueryNames(t, pool, `
		SELECT c.table_name::text
		FROM information_schema.columns c
		JOIN information_schema.tables tb
		  ON tb.table_schema = c.table_schema AND tb.table_name = c.table_name
		WHERE c.table_schema = current_schema()
		  AND c.column_name = 'tenant_id'
		  AND tb.table_type = 'BASE TABLE'
		ORDER BY 1`)
	if len(tables) == 0 {
		t.Fatal("no table with a tenant_id column in the live schema")
	}
	return tables
}

// TestPurgeTenant seeds one row per tenant-scoped table plus a session
// (six session_state rows since schema 2.17.0, #156) for two tenants, purges
// tenant A, and asserts every one of tenant A's rows — domain tables and
// session alike — is gone, while tenant B's rows of the same tables and the
// global schema-version metadata row are untouched. It also purges tenant A
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

	const scopeA, scopeB = "101", "202"
	tenantA, tenantB := tenantID(scopeA), tenantID(scopeB)
	purgeSeedRows(t, s.pool, tenantA)
	purgeSeedRows(t, s.pool, tenantB)

	// The table set is the schema's, not a list kept beside PurgeTenant: a
	// tenant-scoped table nobody seeds fails the sanity check below, and one
	// nobody purges fails the after-purge count.
	tables := purgeTenantTables(t, s.pool)
	for _, tbl := range tables {
		if got := purgeCountRows(t, s.pool, tbl, tenantA); got != 1 {
			t.Fatalf("seed sanity: %s tenant A: got %d rows, want 1", tbl, got)
		}
		if got := purgeCountRows(t, s.pool, tbl, tenantB); got != 1 {
			t.Fatalf("seed sanity: %s tenant B: got %d rows, want 1", tbl, got)
		}
	}

	// The session is written through the store — six session_state rows per
	// tenant on top of the seed row — so tenant B's counts are snapshotted
	// rather than assumed.
	sessionA := storage.SessionState{LastSearchCommand: "search-A", LastSearchPosition: "pos-A"}
	sessionB := storage.SessionState{LastSearchCommand: "search-B", LastSearchPosition: "pos-B"}
	if err := s.Session().Save(ctx, scopeA, sessionA); err != nil {
		t.Fatalf("seed session A: %v", err)
	}
	if err := s.Session().Save(ctx, scopeB, sessionB); err != nil {
		t.Fatalf("seed session B: %v", err)
	}
	wantB := make(map[string]int, len(tables))
	for _, tbl := range tables {
		wantB[tbl] = purgeCountRows(t, s.pool, tbl, tenantB)
	}
	if got := purgeCountRows(t, s.pool, "session_state", tenantA); got != 7 {
		t.Fatalf("seed sanity: session_state tenant A: got %d rows, want 7 (seed + six session keys)", got)
	}

	if err := s.PurgeTenant(ctx, scopeA); err != nil {
		t.Fatalf("PurgeTenant: %v", err)
	}

	for _, tbl := range tables {
		if got := purgeCountRows(t, s.pool, tbl, tenantA); got != 0 {
			t.Errorf("after purge: %s tenant A: got %d rows, want 0", tbl, got)
		}
		if got := purgeCountRows(t, s.pool, tbl, tenantB); got != wantB[tbl] {
			t.Errorf("after purge: %s tenant B: got %d rows, want %d (untouched)", tbl, got, wantB[tbl])
		}
	}

	loadedA, err := s.Session().Load(ctx, scopeA)
	if err != nil {
		t.Fatalf("load session A after purge: %v", err)
	}
	if loadedA.LastSearchCommand != "" || loadedA.LastSearchPosition != "" {
		t.Errorf("after purge: session A = %+v, want zero value (session_state rows purged)", *loadedA)
	}
	loadedB, err := s.Session().Load(ctx, scopeB)
	if err != nil {
		t.Fatalf("load session B after purge: %v", err)
	}
	if loadedB.LastSearchCommand != sessionB.LastSearchCommand || loadedB.LastSearchPosition != sessionB.LastSearchPosition {
		t.Errorf("after purge: session B = %+v, want untouched %+v", *loadedB, sessionB)
	}
	if _, err := s.Version(ctx); err != nil {
		t.Errorf("after purge: global schema-version metadata row missing/unreadable: %v", err)
	}

	// Idempotency: purging an already-purged tenant is a harmless no-op.
	if err := s.PurgeTenant(ctx, scopeA); err != nil {
		t.Fatalf("second PurgeTenant (idempotency): %v", err)
	}
	for _, tbl := range tables {
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

	if err := s.PurgeTenant(ctx, "999"); err != nil {
		t.Fatalf("PurgeTenant on a tenant with no data: %v", err)
	}
}

// TestTenantTablesSchemaGuards checks purgeOrder and rlsTables against what
// the live schema says rather than against each other, on one container:
//
//   - every foreign key's child table comes before its parent in purgeOrder,
//     so the purge never leans on an ON DELETE action to finish its job;
//   - ApplyRLS leaves every table carrying a tenant_id column with RLS
//     enabled, forced and under the tenant_isolation policy — a table the list
//     forgets stays readable across tenants on a database bootstrapped with
//     RLS, whatever its migration installed where RLS already existed.
func TestTenantTablesSchemaGuards(t *testing.T) {
	ctx := context.Background()
	dsn := purgeTestDB(t)
	purgeResetSchema(t, dsn)
	s, err := Open(ctx, dsn, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()
	tables := purgeTenantTables(t, s.pool)

	t.Run("purge order respects foreign keys", func(t *testing.T) {
		rank := make(map[string]int, len(purgeOrder))
		for i, tbl := range purgeOrder {
			rank[tbl] = i
		}
		rows, err := s.pool.Query(ctx, `
			SELECT child.relname::text, parent.relname::text
			FROM pg_constraint k
			JOIN pg_class child  ON child.oid  = k.conrelid
			JOIN pg_class parent ON parent.oid = k.confrelid
			WHERE k.contype = 'f' AND k.connamespace = current_schema()::regnamespace
			ORDER BY 1, 2`)
		if err != nil {
			t.Fatalf("foreign keys: %v", err)
		}
		type fk struct{ Child, Parent string }
		fks, err := pgx.CollectRows(rows, pgx.RowToStructByPos[fk])
		if err != nil {
			t.Fatalf("collect foreign keys: %v", err)
		}
		if len(fks) == 0 {
			t.Fatal("no foreign key in the live schema")
		}
		for _, k := range fks {
			if k.Child == k.Parent {
				continue
			}
			ci, okC := rank[k.Child]
			pi, okP := rank[k.Parent]
			switch {
			case !okC || !okP:
				t.Errorf("foreign key %s → %s: a table is missing from purgeOrder", k.Child, k.Parent)
			case ci > pi:
				t.Errorf("foreign key %s → %s: purgeOrder deletes the parent (#%d) before the child (#%d)", k.Child, k.Parent, pi, ci)
			}
		}
	})

	t.Run("ApplyRLS covers every tenant table", func(t *testing.T) {
		if err := s.ApplyRLS(ctx); err != nil {
			t.Fatalf("ApplyRLS: %v", err)
		}
		policed := purgeQueryNames(t, s.pool, `
			SELECT tablename::text FROM pg_policies
			WHERE schemaname = current_schema() AND policyname = 'tenant_isolation'
			ORDER BY 1`)
		forced := purgeQueryNames(t, s.pool, `
			SELECT c.relname::text FROM pg_class c
			WHERE c.relnamespace = current_schema()::regnamespace
			  AND c.relkind = 'r' AND c.relrowsecurity AND c.relforcerowsecurity
			ORDER BY 1`)
		if !slices.Equal(policed, tables) {
			t.Errorf("tenant_isolation policy after ApplyRLS:\n policed = %v\n tenant  = %v", policed, tables)
		}
		if !slices.Equal(forced, tables) {
			t.Errorf("forced row-level security after ApplyRLS:\n forced = %v\n tenant = %v", forced, tables)
		}
	})
}
