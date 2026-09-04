package database

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// The SQLite schema has one declaration, storage/sqlite's schemaStatements,
// but several paths lead a database to it: sqlite.Bootstrap (fresh databases,
// and since SetupDatabase delegates to it, the wrapper's fresh path too),
// sqlite.EnsureSchema (every open of an existing database, through
// ensureAllTablesExist in db_schema.go, deriving the missing columns from a
// reference database rather than a second list), and the version-by-version
// chain in db_migration.go, which keeps its own historical DDL. These tests
// take a normalised snapshot of sqlite_master (tables, columns with type /
// NOT NULL / default / pk, foreign keys, indexes with their columns and WHERE
// clause, triggers, views) on databases produced by each path and diff them.
//
// A difference fails the test unless it is listed in an allow-list, and every
// allow-list entry must explain itself and must actually match something —
// an entry that no longer matches fails too, so the list cannot rot.

// schemaFact is one normalised line describing a schema element.
type schemaFact = string

// allowedDiff is a schema fact known to differ between two paths, with the
// reason it is tolerated. pattern is a regexp matched against the fact line.
type allowedDiff struct {
	pattern string
	reason  string
}

var wsRe = regexp.MustCompile(`\s+`)

// snapshotSchema returns the sorted set of facts describing the schema of db.
// sqlite_stat* tables (ANALYZE output) are skipped: they are planner
// statistics, not schema, and only some paths run ANALYZE.
func snapshotSchema(t *testing.T, db *sql.DB) []schemaFact {
	t.Helper()
	var facts []schemaFact

	rows, err := db.Query(`SELECT type, name, tbl_name, COALESCE(sql, '')
		FROM sqlite_master
		WHERE name NOT LIKE 'sqlite_stat%'
		ORDER BY type, name`)
	if err != nil {
		t.Fatalf("sqlite_master: %v", err)
	}
	defer rows.Close()

	type object struct{ typ, name, tbl, sql string }
	var objects []object
	for rows.Next() {
		var o object
		if err := rows.Scan(&o.typ, &o.name, &o.tbl, &o.sql); err != nil {
			t.Fatalf("scan sqlite_master: %v", err)
		}
		objects = append(objects, o)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate sqlite_master: %v", err)
	}

	for _, o := range objects {
		switch o.typ {
		case "table":
			facts = append(facts, "table "+o.name)
			facts = append(facts, tableFacts(t, db, o.name)...)
		case "index":
			facts = append(facts, indexFact(t, db, o))
		case "trigger", "view":
			facts = append(facts, fmt.Sprintf("%s %s on %s sql=%s", o.typ, o.name, o.tbl, normaliseSQL(o.sql)))
		default:
			facts = append(facts, o.typ+" "+o.name)
		}
	}
	sort.Strings(facts)
	return facts
}

func tableFacts(t *testing.T, db *sql.DB, table string) []schemaFact {
	t.Helper()
	var facts []schemaFact

	cols, err := db.Query(`SELECT name, type, "notnull", COALESCE(dflt_value, '<none>'), pk FROM pragma_table_info(?)`, table)
	if err != nil {
		t.Fatalf("pragma_table_info(%s): %v", table, err)
	}
	defer cols.Close()
	for cols.Next() {
		var name, typ, dflt string
		var notnull, pk int
		if err := cols.Scan(&name, &typ, &notnull, &dflt, &pk); err != nil {
			t.Fatalf("scan table_info(%s): %v", table, err)
		}
		facts = append(facts, fmt.Sprintf("column %s.%s type=%s notnull=%d default=%s pk=%d",
			table, name, strings.ToUpper(typ), notnull, normaliseSQL(dflt), pk))
	}
	if err := cols.Err(); err != nil {
		t.Fatalf("iterate table_info(%s): %v", table, err)
	}

	fks, err := db.Query(`SELECT "table", "from", COALESCE("to", ''), on_update, on_delete FROM pragma_foreign_key_list(?)`, table)
	if err != nil {
		t.Fatalf("pragma_foreign_key_list(%s): %v", table, err)
	}
	defer fks.Close()
	for fks.Next() {
		var ref, from, to, onUpdate, onDelete string
		if err := fks.Scan(&ref, &from, &to, &onUpdate, &onDelete); err != nil {
			t.Fatalf("scan foreign_key_list(%s): %v", table, err)
		}
		facts = append(facts, fmt.Sprintf("fk %s.%s -> %s.%s on_update=%s on_delete=%s",
			table, from, ref, to, onUpdate, onDelete))
	}
	if err := fks.Err(); err != nil {
		t.Fatalf("iterate foreign_key_list(%s): %v", table, err)
	}
	return facts
}

// indexFact describes one index: table, uniqueness, ordered columns and, for a
// partial index, its WHERE clause (taken from the DDL, whitespace-normalised;
// pragma_index_list only says whether the index is partial).
func indexFact(t *testing.T, db *sql.DB, o struct{ typ, name, tbl, sql string }) schemaFact {
	t.Helper()
	var unique int
	if err := db.QueryRow(`SELECT "unique" FROM pragma_index_list(?) WHERE name = ?`, o.tbl, o.name).Scan(&unique); err != nil {
		t.Fatalf("pragma_index_list(%s) for %s: %v", o.tbl, o.name, err)
	}
	rows, err := db.Query(`SELECT COALESCE(name, '<expr>') FROM pragma_index_info(?) ORDER BY seqno`, o.name)
	if err != nil {
		t.Fatalf("pragma_index_info(%s): %v", o.name, err)
	}
	defer rows.Close()
	var cols []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			t.Fatalf("scan index_info(%s): %v", o.name, err)
		}
		cols = append(cols, c)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate index_info(%s): %v", o.name, err)
	}
	where := ""
	if i := strings.Index(strings.ToUpper(o.sql), " WHERE "); i >= 0 {
		where = normaliseSQL(o.sql[i+len(" WHERE "):])
	}
	return fmt.Sprintf("index %s on %s unique=%d cols=(%s) where=%q",
		o.name, o.tbl, unique, strings.Join(cols, ","), where)
}

func normaliseSQL(s string) string {
	return strings.TrimSpace(wsRe.ReplaceAllString(s, " "))
}

// diffSchemas reports the facts present in one snapshot and not the other,
// minus the allowed differences. It fails the test for every unexplained
// difference and for every allow-list entry that matched nothing.
func diffSchemas(t *testing.T, leftName string, left []schemaFact, rightName string, right []schemaFact, allowed []allowedDiff) {
	t.Helper()
	used := make([]bool, len(allowed))
	report := func(side string, only []schemaFact) {
		for _, f := range only {
			tolerated := false
			for i, a := range allowed {
				if regexp.MustCompile(a.pattern).MatchString(f) {
					used[i] = true
					tolerated = true
				}
			}
			if !tolerated {
				t.Errorf("only in %s: %s", side, f)
			}
		}
	}
	report(leftName, setMinus(left, right))
	report(rightName, setMinus(right, left))
	for i, a := range allowed {
		if !used[i] {
			t.Errorf("allow-list entry %q matched no difference between %s and %s — remove it (%s)",
				a.pattern, leftName, rightName, a.reason)
		}
	}
}

func setMinus(a, b []schemaFact) []schemaFact {
	in := make(map[schemaFact]bool, len(b))
	for _, f := range b {
		in[f] = true
	}
	var out []schemaFact
	for _, f := range a {
		if !in[f] {
			out = append(out, f)
		}
	}
	return out
}

// freshWrapperSchema snapshots a database created by Database.SetupDatabase.
func freshWrapperSchema(t *testing.T) []schemaFact {
	t.Helper()
	d := NewDatabase()
	if err := d.SetupDatabase(filepath.Join(t.TempDir(), "fresh_wrapper.db")); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	defer d.Close()
	return snapshotSchema(t, d.db)
}

// freshBackendSchema snapshots a database created by storage/sqlite.Open on a
// fresh file (Bootstrap).
func freshBackendSchema(t *testing.T) []schemaFact {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fresh_backend.db")
	s, err := sqlite.Open(context.Background(), path, nil)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate fresh backend: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	return snapshotFile(t, path)
}

// migratedWrapperSchema builds a database at oldVersion with createOldDatabase
// (the same fixture builder migration_test.go uses), opens it through
// Database.OpenDatabase — which runs the migration chain and
// ensureAllTablesExist — and snapshots the result.
func migratedWrapperSchema(t *testing.T, oldVersion string) []schemaFact {
	t.Helper()
	path := filepath.Join(t.TempDir(), "old_wrapper.db")
	createOldDatabase(t, path, oldVersion)
	d := NewDatabase()
	if err := d.OpenDatabase(path); err != nil {
		t.Fatalf("OpenDatabase v%s: %v", oldVersion, err)
	}
	defer d.Close()
	if v, err := d.CheckDatabaseVersion(); err != nil || v != DatabaseVersion {
		t.Fatalf("version after migration from %s: %q, %v", oldVersion, v, err)
	}
	return snapshotSchema(t, d.db)
}

// migratedBackendSchema is the headless twin of migratedWrapperSchema: the old
// database is upgraded through storage/sqlite's Migrate, i.e. the migrator
// registered by this package's init (migrate_hook.go).
func migratedBackendSchema(t *testing.T, oldVersion string) []schemaFact {
	t.Helper()
	path := filepath.Join(t.TempDir(), "old_backend.db")
	createOldDatabase(t, path, oldVersion)
	s, err := sqlite.Open(context.Background(), path, nil)
	if err != nil {
		t.Fatalf("sqlite.Open v%s: %v", oldVersion, err)
	}
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate v%s: %v", oldVersion, err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	return snapshotFile(t, path)
}

func snapshotFile(t *testing.T, path string) []schemaFact {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer db.Close()
	return snapshotSchema(t, db)
}

// fixtureVersions are the starting points createOldDatabase models
// faithfully: it builds the tables each 1.x version had, so the chain from
// there is the exact path a real 1.x database takes. It cannot build a
// faithful 2.x database (it only knows the 1.x tables and stamps the version;
// the 2.x steps then either trip on the missing columns — "2.0.0" fails on
// best_cube_action — or lean on ensureAllTablesExist to fill them, which is
// not what a real 2.x file looks like), so 2.x starting points are not
// diffed here. A real 2.x database was produced either by this same chain or
// by Bootstrap, both of which are covered.
var fixtureVersions = []string{
	"1.0.0", "1.1.0", "1.2.0", "1.3.0", "1.4.0", "1.5.0", "1.6.0", "1.7.0", "1.8.0", "1.9.0",
}

// TestSchemaParity_FreshWrapperVsBackend: a database created by the wrapper
// (GUI "new database", CLI create, :memory:) must be the one storage/sqlite
// bootstraps for the daemon.
func TestSchemaParity_FreshWrapperVsBackend(t *testing.T) {
	t.Parallel()
	diffSchemas(t,
		"fresh wrapper (SetupDatabase)", freshWrapperSchema(t),
		"fresh backend (sqlite.Open)", freshBackendSchema(t),
		nil)
}

// TestSchemaParity_MigratedVsFresh: a database migrated from any past version
// must end up with the same tables, columns and indexes as a fresh one.
func TestSchemaParity_MigratedVsFresh(t *testing.T) {
	t.Parallel()
	fresh := freshWrapperSchema(t)
	for _, v := range fixtureVersions {
		t.Run("wrapper_from_"+v, func(t *testing.T) {
			diffSchemas(t,
				"migrated from "+v+" (OpenDatabase)", migratedWrapperSchema(t, v),
				"fresh (SetupDatabase)", fresh,
				migratedFromV1Allowed)
		})
	}
}

// TestSchemaParity_MigratedBackendVsWrapper: the headless upgrade path
// (storage/sqlite.Migrate through the registered migrator) must produce the
// same schema as the GUI/CLI upgrade path for the oldest fixture.
func TestSchemaParity_MigratedBackendVsWrapper(t *testing.T) {
	t.Parallel()
	for _, v := range []string{"1.0.0", "2.14.0"} {
		t.Run("from_"+v, func(t *testing.T) {
			diffSchemas(t,
				"migrated from "+v+" (sqlite.Migrate)", migratedBackendSchema(t, v),
				"migrated from "+v+" (OpenDatabase)", migratedWrapperSchema(t, v),
				nil)
		})
	}
}

// migratedFromV1Allowed lists the differences a database migrated from 1.x is
// allowed to keep against a fresh one. Every entry is something SQLite cannot
// change in place (ALTER TABLE can neither add NOT NULL to an existing column
// nor change a declared type) and would need a full table rebuild of the two
// largest tables to remove — not worth it, since none of them changes what
// any query reads or writes.
var migratedFromV1Allowed = []allowedDiff{
	{
		pattern: `^column position\.state type=TEXT notnull=[01] default=<none> pk=0$`,
		reason: "position.state is NOT NULL on a fresh database; pre-2.0.0 tables " +
			"created it nullable and every writer has always supplied a state.",
	},
	{
		pattern: `^column analysis\.(cube_error|best_move_equity_error|player[12]_(win|gammon|backgammon)_rate) type=(REAL|INTEGER) notnull=0 default=<none> pk=0$`,
		reason: "the 1.9.0→2.0.0 migration added these as REAL where the fresh DDL says " +
			"INTEGER; affinity only — a fractional value is stored REAL under either, " +
			"an integral one round-trips through database/sql either way.",
	},
	{
		pattern: `^column move_analysis\.(equity|equity_error|(opponent_)?(win|gammon|backgammon)_rate) type=(REAL|INTEGER) notnull=0 default=<none> pk=0$`,
		reason: "the 1.3.0→1.4.0 migration created move_analysis with REAL rates where " +
			"the fresh DDL says INTEGER; same affinity-only argument as analysis.",
	},
	{
		pattern: `^fk anki_review_log\.(deck_id|position_id) -> (anki_deck|position)\.id on_update=NO ACTION on_delete=CASCADE$`,
		reason: "a fresh database constrains the review journal's deck_id and position_id " +
			"(issue #185); the 2.10.0→2.11.0 migration created the table without them and " +
			"SQLite adds no foreign key to a table that already exists. `blunderdb verify` " +
			"counts the rows that dangle instead.",
	},
}
