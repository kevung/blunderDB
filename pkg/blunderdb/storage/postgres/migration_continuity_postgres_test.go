//go:build postgres

package postgres_test

import (
	"context"
	"os"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	pg "github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
)

// TestMigrationChain_HistoricalReplayMatchesFreshBootstrap (#235): SQLite has
// TestMigrationSteps_ContinuousChain guarding an unbroken chain from 1.0.0 to
// DatabaseVersion; PostgreSQL had no equivalent — postgres_test.go's other
// tests all start from bootstrap() (001 plus a Migrate that finds every
// forward migration already recorded as a no-op), so 002..013 had never
// actually been exercised as real, executed SQL against a database that
// lacked their changes.
//
// This test rebuilds that "historical" starting point directly — apply ONLY
// the raw 001 baseline SQL, bypassing bootstrap()/Migrate() — then calls the
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
