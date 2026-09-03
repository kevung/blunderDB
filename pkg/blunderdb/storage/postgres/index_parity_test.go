// TestIndexParityWithSQLite never touches a database: it reads the SQLite
// backend's schema source and this package's embedded migrations, and compares
// the indexes the two declare — by name, by uniqueness and by the columns each
// one covers.
//
// Why: the eight search-range indexes fiche-05 added to schema_sqlite.go
// (back_checkers_1/2, pip_1, no_contact, the four remaining rate columns) went
// missing from the PostgreSQL chain for whole releases, because nothing tied
// the two lists together. Names alone caught that; they did not catch an index
// that exists on both sides over different columns, which is the same bug one
// step further in (fiche-05's own covering index gained a trailing position_id
// on one side only, and only a comment said so). So the columns are compared
// too — as a SET, deliberately:
//
//   - `tenant_id` leads every composite index here and exists nowhere in
//     SQLite, so it is dropped before comparing;
//   - a boolean column sits in the KEY on the SQLite side (`ON position
//     (individually_imported) WHERE individually_imported = 1`) and in the
//     PREDICATE here (`ON position (tenant_id) WHERE individually_imported`),
//     so predicate columns are folded in with the key columns;
//   - what is left is which columns the index covers, which is the fact that
//     must not drift. Column ORDER legitimately differs once tenant_id leads
//     one side, so it is not compared.
package postgres

import (
	"io/fs"
	"os"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"
)

// sqliteSchemaSource is the SQLite backend's bootstrap DDL, read from disk
// relative to this package directory (go test runs with cwd = package dir).
const sqliteSchemaSource = "../sqlite/schema_sqlite.go"

// sqliteOnlyIndexes are declared by SQLite and deliberately absent here.
// The *_scope indexes serve the SQLite `scope` column that partitions
// history/filter rows per database file; PostgreSQL has no such column —
// tenant_id plays that role, and those tables are tiny.
var sqliteOnlyIndexes = []string{
	"idx_command_history_scope",
	"idx_filter_library_scope_name",
	"idx_search_history_scope",
}

// postgresOnlyIndexes are declared here and deliberately absent from SQLite.
// None today; add an entry (with a reason) rather than loosening the test.
var postgresOnlyIndexes = []string{}

// indexDecl is one CREATE INDEX statement, normalised for comparison.
type indexDecl struct {
	name    string
	unique  bool
	table   string
	columns []string // key columns and predicate columns, tenant_id dropped, sorted
}

var createIndexRE = regexp.MustCompile(
	`CREATE\s+(UNIQUE\s+)?INDEX\s+IF\s+NOT\s+EXISTS\s+(idx_\w+)\s+ON\s+(\w+)\s*\(([^)]*)\)([^;` + "`" + `]*)`)

// wordRE picks the identifiers out of a partial index's WHERE clause, so that
// `WHERE individually_imported = 1` and `WHERE individually_imported` fold to
// the same column.
var wordRE = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*`)

var predicateKeywords = map[string]bool{
	"where": true, "and": true, "or": true, "not": true,
	"true": true, "false": true, "null": true, "is": true,
}

// parseIndexes returns the indexes declared by src, keyed by name. A name
// declared more than once (a migration that drops and rebuilds an index) keeps
// the LAST declaration, which is the one a database ends up with. Comment
// lines (prefix `--` for SQL, `//` for Go) are skipped so that prose mentioning
// a dropped index does not count as a declaration.
func parseIndexes(src string, commentPrefix string) map[string]indexDecl {
	out := map[string]indexDecl{}
	for _, line := range strings.Split(src, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), commentPrefix) {
			continue
		}
		for _, m := range createIndexRE.FindAllStringSubmatch(line, -1) {
			d := indexDecl{
				name:   m[2],
				unique: strings.TrimSpace(m[1]) != "",
				table:  strings.ToLower(m[3]),
			}
			seen := map[string]bool{}
			add := func(col string) {
				col = strings.ToLower(strings.TrimSpace(col))
				if col == "" || col == "tenant_id" || seen[col] {
					return
				}
				seen[col] = true
				d.columns = append(d.columns, col)
			}
			for _, col := range strings.Split(m[4], ",") {
				add(col)
			}
			for _, w := range wordRE.FindAllString(strings.ToLower(m[5]), -1) {
				if !predicateKeywords[w] {
					add(w)
				}
			}
			sort.Strings(d.columns)
			out[d.name] = d
		}
	}
	return out
}

func names(decls map[string]indexDecl) []string {
	out := make([]string, 0, len(decls))
	for name := range decls {
		out = append(out, name)
	}
	slices.Sort(out)
	return out
}

func TestIndexParityWithSQLite(t *testing.T) {
	src, err := os.ReadFile(sqliteSchemaSource)
	if err != nil {
		t.Fatalf("read %s: %v", sqliteSchemaSource, err)
	}
	sqliteIdx := parseIndexes(string(src), "//")
	if len(sqliteIdx) == 0 {
		t.Fatalf("no CREATE INDEX found in %s — did the DDL move?", sqliteSchemaSource)
	}

	pgIdx := map[string]indexDecl{}
	var files []string
	err = fs.WalkDir(migrationsFS, "migrations", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".sql") {
			return err
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		t.Fatalf("walk migrations: %v", err)
	}
	// Numeric order: a later migration's declaration wins over the baseline's.
	slices.Sort(files)
	for _, path := range files {
		b, err := migrationsFS.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for name, decl := range parseIndexes(string(b), "--") {
			pgIdx[name] = decl
		}
	}
	if len(pgIdx) == 0 {
		t.Fatal("no CREATE INDEX found in the embedded migrations")
	}

	// Each direction: names declared on one side, absent on the other, minus
	// the justified exceptions.
	for _, name := range names(sqliteIdx) {
		if _, ok := pgIdx[name]; !ok && !slices.Contains(sqliteOnlyIndexes, name) {
			t.Errorf("%s is declared in %s but in no PostgreSQL migration — add a migration (see migrations/README.md) or list it in sqliteOnlyIndexes with a reason", name, sqliteSchemaSource)
		}
	}
	for _, name := range names(pgIdx) {
		if _, ok := sqliteIdx[name]; !ok && !slices.Contains(postgresOnlyIndexes, name) {
			t.Errorf("%s is declared in a PostgreSQL migration but not in %s — add it there or list it in postgresOnlyIndexes with a reason", name, sqliteSchemaSource)
		}
	}

	// Same name on both sides: same table, same uniqueness, same columns.
	for _, name := range names(sqliteIdx) {
		want, ok := pgIdx[name]
		if !ok {
			continue
		}
		got := sqliteIdx[name]
		if got.table != want.table {
			t.Errorf("%s: SQLite indexes %s, PostgreSQL indexes %s", name, got.table, want.table)
		}
		if got.unique != want.unique {
			t.Errorf("%s: UNIQUE is %v in %s and %v in the PostgreSQL migrations",
				name, got.unique, sqliteSchemaSource, want.unique)
		}
		if !slices.Equal(got.columns, want.columns) {
			t.Errorf("%s covers %v in %s and %v in the PostgreSQL migrations (tenant_id dropped, partial-index predicate columns folded in, order not compared)",
				name, got.columns, sqliteSchemaSource, want.columns)
		}
	}

	// The allow-lists must stay honest: an entry that no longer describes a
	// real asymmetry is stale and must go.
	for _, name := range sqliteOnlyIndexes {
		_, inSQLite := sqliteIdx[name]
		_, inPG := pgIdx[name]
		if !inSQLite || inPG {
			t.Errorf("sqliteOnlyIndexes lists %s, which is no longer SQLite-only — remove it", name)
		}
	}
	for _, name := range postgresOnlyIndexes {
		_, inSQLite := sqliteIdx[name]
		_, inPG := pgIdx[name]
		if !inPG || inSQLite {
			t.Errorf("postgresOnlyIndexes lists %s, which is no longer PostgreSQL-only — remove it", name)
		}
	}
}
