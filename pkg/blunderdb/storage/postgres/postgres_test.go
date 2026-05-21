//go:build postgres

// These tests provision a real PostgreSQL via testcontainers-go and therefore
// need Docker. They are gated behind the `postgres` build tag so the default
// `go test ./...` (and CI runners without Docker) skip them:
//
//	go test -tags postgres ./pkg/blunderdb/storage/postgres/...
package postgres_test

import (
	"context"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go"
	tcpg "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	pg "github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
)

// startPostgres boots a throwaway PostgreSQL 16 container and returns its DSN.
// The test is skipped (not failed) when Docker is unavailable.
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

// wantTables is the full v2.7.0 table set, sorted.
var wantTables = []string{
	"analysis", "anki_card", "anki_deck", "collection", "collection_position",
	"command_history", "comment", "filter_library", "game", "match",
	"metadata", "move", "move_analysis", "position", "search_history",
	"tournament",
}

// wantIndexes is the full set of named idx_* indexes, sorted.
var wantIndexes = []string{
	"idx_analysis_cube_error", "idx_analysis_is_close_cube",
	"idx_analysis_is_forced", "idx_analysis_move_error",
	"idx_analysis_position", "idx_analysis_win1", "idx_analysis_win_gammon",
	"idx_anki_card_deck", "idx_anki_card_due",
	"idx_collection_position_collection", "idx_game_match", "idx_match_canonical",
	"idx_match_hash", "idx_move_game", "idx_move_position",
	"idx_position_decision_dice", "idx_position_decision_pip",
	"idx_position_dice", "idx_position_off", "idx_position_pip_diff",
	"idx_position_score", "idx_position_score_cube", "idx_position_zobrist",
}

// TestMigratePostgres opens a fresh database, runs Migrate, and confirms the
// v2.7.0 schema landed: all 16 tables, every named index, the database_version
// row, and a tenant_id column on every domain table.
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

	// Every domain table is multi-tenant; metadata is database-level.
	for _, tbl := range wantTables {
		if tbl == "metadata" {
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
