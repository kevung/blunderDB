//go:build postgres

package postgres_test

import (
	"context"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	pg "github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
)

// TestMigrationChain_HistoricalReplayMatchesFreshBootstrap is PostgreSQL's
// counterpart of SQLite's TestMigrationSteps_ContinuousChain: every other test
// starts from bootstrap(), which records the forward migrations without
// executing them against a database that lacks their changes.
//
// It applies ONLY the raw 001 baseline SQL, bypassing bootstrap()/Migrate(),
// then calls the
// ordinary Migrate on it, so migrateForward has to apply 002 through the
// current last migration for real. The resulting schema must match, table
// for table, column for column, index for index, a database that went
// through the normal path (Open on an empty database: bootstrap + a
// migrateForward that finds everything already a no-op).
func TestMigrationChain_HistoricalReplayMatchesFreshBootstrap(t *testing.T) {
	ctx := context.Background()
	dsn := startPostgres(t)

	// --- historical replay: bare 001, then the real forward chain ---
	resetPublicSchema(t, dsn)
	applyBaseline001(t, ctx, dsn)
	replayed, err := pg.Open(ctx, dsn, nil)
	if err != nil {
		t.Fatalf("Open (post-001, forward chain applies for real): %v", err)
	}
	if err := replayed.Migrate(ctx); err != nil {
		t.Fatalf("Migrate (replay): %v", err)
	}
	if v, err := replayed.Version(ctx); err != nil || v != domain.DatabaseVersion {
		t.Fatalf("replayed version = %q, %v; want %q, nil", v, err, domain.DatabaseVersion)
	}
	replayedSchema := snapshotSchema(t, dsn)
	replayed.Close()

	// --- fresh bootstrap: 001 already contains every change, 002+ are all
	// no-ops recorded without altering anything ---
	resetPublicSchema(t, dsn)
	fresh, err := pg.Open(ctx, dsn, nil)
	if err != nil {
		t.Fatalf("Open (fresh bootstrap): %v", err)
	}
	freshSchema := snapshotSchema(t, dsn)
	fresh.Close()

	if !slices.Equal(replayedSchema.tables, freshSchema.tables) {
		t.Errorf("tables differ:\n replayed %v\n fresh    %v", replayedSchema.tables, freshSchema.tables)
	}
	if !slices.Equal(replayedSchema.columns, freshSchema.columns) {
		t.Errorf("columns differ:\n replayed %v\n fresh    %v", diffLines(replayedSchema.columns, freshSchema.columns), diffLines(freshSchema.columns, replayedSchema.columns))
	}
	if !slices.Equal(replayedSchema.indexes, freshSchema.indexes) {
		t.Errorf("indexes differ:\n replayed-only %v\n fresh-only    %v", diffLines(replayedSchema.indexes, freshSchema.indexes), diffLines(freshSchema.indexes, replayedSchema.indexes))
	}
	if !slices.Equal(replayedSchema.constraints, freshSchema.constraints) {
		t.Errorf("constraints differ:\n replayed-only %v\n fresh-only    %v", diffLines(replayedSchema.constraints, freshSchema.constraints), diffLines(freshSchema.constraints, replayedSchema.constraints))
	}
}

// applyBaseline001 connects directly (bypassing bootstrap()/Migrate()) and
// runs exactly the v2.7.0 baseline SQL — the historical starting point every
// PostgreSQL database this backend has ever bootstrapped actually had before
// any forward migration applied to it.
func applyBaseline001(t *testing.T, ctx context.Context, dsn string) {
	t.Helper()
	sql, err := os.ReadFile("migrations/001_initial_v2_7_0.sql")
	if err != nil {
		t.Fatalf("read 001 baseline: %v", err)
	}
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx, string(sql)); err != nil {
		t.Fatalf("apply 001 baseline: %v", err)
	}
}

// schemaShape is a comparable snapshot of a database's structure: every
// table, every column (with its type and nullability), every named index,
// every named constraint (PK/UNIQUE/FK, with its full definition — catches a
// composite foreign key or an ON DELETE action differing between the two
// paths, not just its name) — not comment/rls_postgres_test.go content, just
// DDL shape.
type schemaShape struct {
	tables      []string
	columns     []string
	indexes     []string
	constraints []string
}

// snapshotSchema reads dsn's public schema shape via a fresh connection.
func snapshotSchema(t *testing.T, dsn string) schemaShape {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("snapshot connect: %v", err)
	}
	defer conn.Close(ctx)

	var shape schemaShape
	shape.tables = queryStrings(t, ctx, conn,
		`SELECT tablename FROM pg_tables WHERE schemaname='public' ORDER BY tablename`)
	shape.columns = queryStrings(t, ctx, conn,
		`SELECT table_name || '.' || column_name || ' ' || data_type || ' nullable=' || is_nullable
		 FROM information_schema.columns WHERE table_schema='public'
		 ORDER BY table_name, column_name`)
	shape.indexes = queryStrings(t, ctx, conn,
		`SELECT indexname || ': ' || indexdef FROM pg_indexes
		 WHERE schemaname='public' ORDER BY indexname`)
	shape.constraints = queryStrings(t, ctx, conn,
		`SELECT conrelid::regclass::text || '.' || conname || ': ' || pg_get_constraintdef(oid)
		 FROM pg_constraint WHERE connamespace = 'public'::regnamespace
		 ORDER BY conrelid::regclass::text, conname`)
	return shape
}

func queryStrings(t *testing.T, ctx context.Context, conn *pgx.Conn, sql string) []string {
	t.Helper()
	rows, err := conn.Query(ctx, sql)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatalf("scan: %v", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	return out
}

// diffLines returns the elements of a not present in b, for a readable
// failure message instead of two full multi-hundred-line slices.
func diffLines(a, b []string) []string {
	inB := make(map[string]bool, len(b))
	for _, s := range b {
		inB[s] = true
	}
	var out []string
	for _, s := range a {
		if !inB[s] {
			out = append(out, s)
		}
	}
	if out == nil {
		out = []string{}
	}
	return out
}

// TestMigrationChain_DatabaseCreatedBefore016Upgrades starts from the shape a
// production database actually has when it was bootstrapped before 016 shipped
// composite: the 001 baseline as it stood at 011 (testdata, frozen — the live
// 001 has since been rewritten to already carry every UNIQUE (tenant_id, id)),
// with 002..011 recorded. The real migrator must then carry it to the current
// version and to the same constraints as a fresh bootstrap. 016's composite
// foreign keys need anki_deck/position UNIQUE (tenant_id, id), which only 017
// used to create: without 016 creating them first, this fails with 42830.
func TestMigrationChain_DatabaseCreatedBefore016Upgrades(t *testing.T) {
	ctx := context.Background()
	dsn := startPostgres(t)

	resetPublicSchema(t, dsn)
	baseline, err := os.ReadFile("testdata/001_baseline_at_011.sql")
	if err != nil {
		t.Fatalf("read frozen baseline: %v", err)
	}
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if _, err := conn.Exec(ctx, string(baseline)); err != nil {
		t.Fatalf("apply frozen baseline: %v", err)
	}
	if _, err := conn.Exec(ctx, `CREATE TABLE schema_migrations (
			version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now());
		INSERT INTO schema_migrations (version) VALUES
			('002_is_cube_response'), ('003_anki_review_log'), ('004_anki_card_suspend'),
			('005_individually_imported'), ('006_comment_position_index'), ('007_flagged'),
			('008_win_gammon_covering_index'), ('009_luck_mp'), ('010_search_range_indexes'),
			('011_exclude_position')`); err != nil {
		t.Fatalf("record 002..011: %v", err)
	}
	conn.Close(ctx)

	upgraded, err := pg.Open(ctx, dsn, nil)
	if err != nil {
		t.Fatalf("Open (pre-016 database): %v", err)
	}
	if err := upgraded.Migrate(ctx); err != nil {
		t.Fatalf("Migrate (pre-016 database): %v", err)
	}
	if v, err := upgraded.Version(ctx); err != nil || v != domain.DatabaseVersion {
		t.Fatalf("upgraded version = %q, %v; want %q, nil", v, err, domain.DatabaseVersion)
	}
	upgradedSchema := snapshotSchema(t, dsn)
	upgraded.Close()

	resetPublicSchema(t, dsn)
	fresh, err := pg.Open(ctx, dsn, nil)
	if err != nil {
		t.Fatalf("Open (fresh bootstrap): %v", err)
	}
	freshSchema := snapshotSchema(t, dsn)
	fresh.Close()

	// An upgraded database adds its constraints NOT VALID on purpose (existing
	// rows are not rescanned); only the definition itself must match.
	upgradedConstraints := make([]string, len(upgradedSchema.constraints))
	for i, c := range upgradedSchema.constraints {
		upgradedConstraints[i] = strings.TrimSuffix(c, " NOT VALID")
	}
	if !slices.Equal(upgradedConstraints, freshSchema.constraints) {
		t.Errorf("constraints differ:\n upgraded-only %v\n fresh-only    %v", diffLines(upgradedConstraints, freshSchema.constraints), diffLines(freshSchema.constraints, upgradedConstraints))
	}
}
