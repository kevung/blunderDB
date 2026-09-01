// TestIndexParityWithSQLite never touches a database: it reads the SQLite
// backend's schema source and this package's embedded migrations, and compares
// the two sets of idx_* names. It lives untagged (no `postgres` build tag, no
// Docker) so that `go test ./...` — the only CI job — runs it, and in
// `package postgres` (white-box) because it needs migrationsFS.
//
// Why: the eight search-range indexes fiche-05 added to schema_sqlite.go
// (back_checkers_1/2, pip_1, no_contact, the four remaining rate columns) went
// missing from the PostgreSQL chain for whole releases, because nothing tied
// the two lists together. Only the NAMES are compared — columns legitimately
// differ (tenant_id leads every composite index here, partial-index
// predicates are boolean rather than `= 1`).
package postgres

import (
	"io/fs"
	"os"
	"regexp"
	"slices"
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

var createIndexRE = regexp.MustCompile(`CREATE\s+(?:UNIQUE\s+)?INDEX\s+IF\s+NOT\s+EXISTS\s+(idx_\w+)`)

// indexNames returns the sorted, deduplicated idx_* names created by src,
// ignoring comment lines (prefix `--` for SQL, `//` for Go) so that prose
// mentioning a dropped index does not count as a declaration.
func indexNames(src string, commentPrefix string) []string {
	var names []string
	for _, line := range strings.Split(src, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), commentPrefix) {
			continue
		}
		for _, m := range createIndexRE.FindAllStringSubmatch(line, -1) {
			names = append(names, m[1])
		}
	}
	slices.Sort(names)
	return slices.Compact(names)
}

func TestIndexParityWithSQLite(t *testing.T) {
	src, err := os.ReadFile(sqliteSchemaSource)
	if err != nil {
		t.Fatalf("read %s: %v", sqliteSchemaSource, err)
	}
	sqliteIdx := indexNames(string(src), "//")
	if len(sqliteIdx) == 0 {
		t.Fatalf("no CREATE INDEX found in %s — did the DDL move?", sqliteSchemaSource)
	}

	var pgIdx []string
	err = fs.WalkDir(migrationsFS, "migrations", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".sql") {
			return err
		}
		b, err := migrationsFS.ReadFile(path)
		if err != nil {
			return err
		}
		pgIdx = append(pgIdx, indexNames(string(b), "--")...)
		return nil
	})
	if err != nil {
		t.Fatalf("walk migrations: %v", err)
	}
	slices.Sort(pgIdx)
	pgIdx = slices.Compact(pgIdx)
	if len(pgIdx) == 0 {
		t.Fatal("no CREATE INDEX found in the embedded migrations")
	}

	// Each direction: names declared on one side, absent on the other, minus
	// the justified exceptions.
	for _, name := range sqliteIdx {
		if !slices.Contains(pgIdx, name) && !slices.Contains(sqliteOnlyIndexes, name) {
			t.Errorf("%s is declared in %s but in no PostgreSQL migration — add a migration (see migrations/README.md) or list it in sqliteOnlyIndexes with a reason", name, sqliteSchemaSource)
		}
	}
	for _, name := range pgIdx {
		if !slices.Contains(sqliteIdx, name) && !slices.Contains(postgresOnlyIndexes, name) {
			t.Errorf("%s is declared in a PostgreSQL migration but not in %s — add it there or list it in postgresOnlyIndexes with a reason", name, sqliteSchemaSource)
		}
	}

	// The allow-lists must stay honest: an entry that no longer describes a
	// real asymmetry is stale and must go.
	for _, name := range sqliteOnlyIndexes {
		if !slices.Contains(sqliteIdx, name) || slices.Contains(pgIdx, name) {
			t.Errorf("sqliteOnlyIndexes lists %s, which is no longer SQLite-only — remove it", name)
		}
	}
	for _, name := range postgresOnlyIndexes {
		if !slices.Contains(pgIdx, name) || slices.Contains(sqliteIdx, name) {
			t.Errorf("postgresOnlyIndexes lists %s, which is no longer PostgreSQL-only — remove it", name)
		}
	}
}
