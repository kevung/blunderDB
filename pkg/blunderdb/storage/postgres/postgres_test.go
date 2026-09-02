//go:build postgres

// These tests provision a real PostgreSQL via testcontainers-go and therefore
// need Docker. They are gated behind the `postgres` build tag so the default
// `go test ./...` (and CI runners without Docker) skip them:
//
//	go test -tags postgres ./pkg/blunderdb/storage/postgres/...
package postgres_test

import (
	"context"
	"os"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go"
	tcpg "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	pg "github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/storagetest"
)

// startPostgres boots a throwaway PostgreSQL 16 container and returns its DSN.
// The test is skipped when Docker is unavailable — unless BLUNDERDB_REQUIRE_PG=1
// is set, in which case a missing container is a hard test failure. CI's
// test-postgres job sets that variable so a broken/absent Docker daemon fails
// the job loudly instead of silently skipping the whole PostgreSQL contract.
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
		if os.Getenv("BLUNDERDB_REQUIRE_PG") == "1" {
			t.Fatalf("postgres container unavailable (BLUNDERDB_REQUIRE_PG=1 requires Docker): %v", err)
		}
		t.Skipf("postgres container unavailable (Docker required): %v", err)
	}
	t.Cleanup(func() { _ = testcontainers.TerminateContainer(container) })

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	return dsn
}

// wantTables is the full table set, sorted (the v2.7.0 baseline plus the
// schema_migrations bookkeeping table created by the forward-migration runner).
var wantTables = []string{
	"analysis", "anki_card", "anki_deck", "anki_review_log",
	"collection", "collection_position",
	"command_history", "comment", "filter_library", "game", "match",
	"metadata", "move", "move_analysis", "position", "schema_migrations",
	"search_history", "session_state", "tournament",
}

// wantIndexes is the full set of named idx_* indexes, sorted.
// idx_analysis_win1 and idx_position_score (E3, index redundancy pass) are
// not listed: both were strict column prefixes of an index still here
// (idx_analysis_win_gammon_covering, idx_position_score_cube respectively).
var wantIndexes = []string{
	"idx_analysis_backgammon1", "idx_analysis_backgammon2",
	"idx_analysis_cube_error", "idx_analysis_gammon2",
	"idx_analysis_is_close_cube",
	"idx_analysis_is_forced", "idx_analysis_move_error",
	"idx_analysis_position", "idx_analysis_win2",
	"idx_analysis_win_gammon_covering",
	"idx_anki_card_deck", "idx_anki_card_due",
	"idx_anki_review_log_card", "idx_anki_review_log_deck",
	"idx_collection_position_collection", "idx_comment_position",
	"idx_game_match", "idx_match_canonical",
	"idx_match_hash", "idx_move_game", "idx_move_position",
	"idx_position_back_checkers_1", "idx_position_back_checkers_2",
	"idx_position_cube_response",
	"idx_position_decision_dice", "idx_position_decision_pip",
	"idx_position_dice", "idx_position_flagged", "idx_position_individual",
	"idx_position_no_contact", "idx_position_off",
	"idx_position_pip_1", "idx_position_pip_diff",
	"idx_position_score_cube", "idx_position_zobrist",
}

// TestMigratePostgres opens a fresh database, runs Migrate, and confirms the
// schema landed: all 18 tables, every named index, the database_version row,
// and a tenant_id column on every domain table.
func TestMigratePostgres(t *testing.T) {
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

	v, err := s.Version(ctx)
	if err != nil {
		t.Fatalf("Version: %v", err)
	}
	if v != domain.DatabaseVersion {
		t.Errorf("Version: got %q, want %q", v, domain.DatabaseVersion)
	}

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("inspect connect: %v", err)
	}
	defer conn.Close(ctx)

	tables := queryNames(t, conn,
		`SELECT tablename FROM pg_tables WHERE schemaname='public' ORDER BY tablename`)
	if !slices.Equal(tables, wantTables) {
		t.Errorf("tables:\n got  %v\n want %v", tables, wantTables)
	}

	indexes := queryNames(t, conn,
		`SELECT indexname FROM pg_indexes
		 WHERE schemaname='public' AND indexname LIKE 'idx_%' ORDER BY indexname`)
	if !slices.Equal(indexes, wantIndexes) {
		t.Errorf("indexes:\n got  %v\n want %v", indexes, wantIndexes)
	}

	// Every domain table is multi-tenant; metadata and schema_migrations are
	// database-level infrastructure.
	for _, tbl := range wantTables {
		if tbl == "metadata" || tbl == "schema_migrations" {
			continue
		}
		var n int
		if err := conn.QueryRow(ctx,
			`SELECT count(*) FROM information_schema.columns
			 WHERE table_schema='public' AND table_name=$1 AND column_name='tenant_id'`,
			tbl).Scan(&n); err != nil {
			t.Fatalf("tenant_id probe for %s: %v", tbl, err)
		}
		if n != 1 {
			t.Errorf("table %s: expected a tenant_id column, found %d", tbl, n)
		}
	}
}

// TestMigrateIdempotent confirms a second Migrate on an already-migrated
// database is a harmless no-op.
func TestMigrateIdempotent(t *testing.T) {
	ctx := context.Background()
	dsn := startPostgres(t)

	s, err := pg.Open(ctx, dsn, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("first Migrate: %v", err)
	}
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
}

// TestContract_Postgres runs the backend-agnostic storage contract suite
// against PostgreSQL. The factory drops and recreates the public schema before
// each case so every case starts from a freshly bootstrapped database, the
// same isolation an in-memory SQLite database gives for free.
func TestContract_Postgres(t *testing.T) {
	ctx := context.Background()
	dsn := startPostgres(t)
	storagetest.RunContractTests(t, func() storage.Storage {
		resetPublicSchema(t, dsn)
		s, err := pg.Open(ctx, dsn, nil)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		return s
	})
}

// resetPublicSchema drops every object in the public schema, giving the next
// pg.Open a fresh database to bootstrap.
func resetPublicSchema(t *testing.T, dsn string) {
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

func queryNames(t *testing.T, conn *pgx.Conn, sql string) []string {
	t.Helper()
	rows, err := conn.Query(context.Background(), sql)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan: %v", err)
		}
		out = append(out, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	return out
}

// TestMigrate_013_SessionOutOfMetadata rebuilds what a 2.16.0 database held —
// no session_state table, the session as '<scope>:session_*' rows of the
// global metadata table (bare 'session_*' for the empty scope) — on a
// database that already enforces RLS, and checks that 013 moves every
// integer-named tenant's rows into session_state under its tenant_id, keeps
// the empty scope on tenant 0, drops the rows of a named tenant the daemon no
// longer accepts (ADR-0005), leaves metadata with no session row, installs
// the tenant_isolation policy on the new table, and is idempotent (#156).
func TestMigrate_013_SessionOutOfMetadata(t *testing.T) {
	ctx := context.Background()
	dsn := startPostgres(t)

	s, err := pg.Open(ctx, dsn, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()
	if err := s.ApplyRLS(ctx); err != nil {
		t.Fatalf("ApplyRLS: %v", err)
	}

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)
	for _, stmt := range []string{
		`DELETE FROM schema_migrations WHERE version = '013_session_state'`,
		`DROP TABLE session_state`,
		`UPDATE metadata SET value = '2.16.0' WHERE key = 'database_version'`,
		`INSERT INTO metadata (key, value) VALUES
			('session_last_search_command', 'desktop'),
			('session_views', '{"tabs":["zero"]}'),
			('7:session_last_search_command', 'cube'),
			('7:session_last_position_ids', '[9]'),
			('7:session_views', '{"tabs":["seven"]}'),
			('alice:session_views', '{"tabs":["alice"]}')`,
	} {
		if _, err := conn.Exec(ctx, stmt); err != nil {
			t.Fatalf("rebuild 2.16.0 shape (%s): %v", stmt, err)
		}
	}

	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if v, _ := s.Version(ctx); v != domain.DatabaseVersion {
		t.Errorf("Version after 013: got %q, want %q", v, domain.DatabaseVersion)
	}

	seven, err := s.Session().Load(ctx, "7")
	if err != nil {
		t.Fatalf("load scope 7: %v", err)
	}
	if seven.LastSearchCommand != "cube" || seven.ViewsJSON != `{"tabs":["seven"]}` || len(seven.LastPositionIDs) != 1 {
		t.Errorf("scope 7 after 013: %+v", *seven)
	}
	zero, err := s.Session().Load(ctx, "")
	if err != nil {
		t.Fatalf("load empty scope: %v", err)
	}
	if zero.LastSearchCommand != "desktop" || zero.ViewsJSON != `{"tabs":["zero"]}` {
		t.Errorf("empty scope after 013: %+v", *zero)
	}
	if other, _ := s.Session().Load(ctx, "8"); other.LastSearchCommand != "" || other.ViewsJSON != "" {
		t.Errorf("scope 8 sees another scope's session: %+v", *other)
	}

	var leftovers int
	if err := conn.QueryRow(ctx,
		`SELECT count(*) FROM metadata WHERE key LIKE '%session\_%'`).Scan(&leftovers); err != nil {
		t.Fatalf("count metadata leftovers: %v", err)
	}
	if leftovers != 0 {
		t.Errorf("metadata still holds %d session row(s) after 013", leftovers)
	}
	var strays int
	if err := conn.QueryRow(ctx,
		`SELECT count(*) FROM session_state WHERE value LIKE '%alice%'`).Scan(&strays); err != nil {
		t.Fatalf("count strays: %v", err)
	}
	if strays != 0 {
		t.Errorf("a named tenant's session survived 013 in session_state (%d row(s))", strays)
	}

	var rls bool
	var policies int
	if err := conn.QueryRow(ctx,
		`SELECT c.relrowsecurity, (SELECT count(*) FROM pg_policies WHERE tablename = 'session_state')
		 FROM pg_class c WHERE c.relname = 'session_state'`).Scan(&rls, &policies); err != nil {
		t.Fatalf("inspect RLS on session_state: %v", err)
	}
	if !rls || policies != 1 {
		t.Errorf("session_state after 013 on an RLS database: rowsecurity=%v policies=%d, want true/1", rls, policies)
	}

	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
	if again, _ := s.Session().Load(ctx, "7"); again.LastSearchCommand != "cube" {
		t.Errorf("second Migrate altered scope 7: %+v", *again)
	}
}
