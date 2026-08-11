package tests

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// positionIsHeldSQL is stated three times on purpose (CLAUDE.md "Invariants"):
// storage/sqlite/matches_sqlite.go, storage/postgres/matches_postgres.go and
// database/db_match.go (the copy the GUI and CLI actually run). None of the
// three exports the constant, and this package cannot import all three
// anyway without pulling in unrelated backend wiring, so this test reads the
// constant's raw SQL text straight out of the source files instead — the same
// "read from the repo root" trick database/cli tests use via their TestMain
// chdir (see pkg/blunderdb/database/main_test.go), just computed per-call
// instead of via a package-wide chdir.

// positionIsHeldSourceFiles maps a short label to the file holding one copy
// of the predicate, relative to the repository root.
var positionIsHeldSourceFiles = map[string]string{
	"sqlite":   filepath.Join("pkg", "blunderdb", "storage", "sqlite", "matches_sqlite.go"),
	"postgres": filepath.Join("pkg", "blunderdb", "storage", "postgres", "matches_postgres.go"),
	"database": filepath.Join("pkg", "blunderdb", "database", "db_match.go"),
}

// positionIsHeldSQLPattern captures the backtick-quoted body of
// `const positionIsHeldSQL = ...` across the multiple lines it spans in every
// copy.
var positionIsHeldSQLPattern = regexp.MustCompile("(?s)const positionIsHeldSQL = `(.*?)`")

// repoRootFromThisFile locates the repository root from this test file's own
// path, so the test does not depend on `go test`'s working directory (the
// package directory, not the repo root).
func repoRootFromThisFile(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location")
	}
	// This file lives directly under <repoRoot>/tests/.
	return filepath.Join(filepath.Dir(thisFile), "..")
}

// extractPositionIsHeldSQL reads path and returns the raw SQL text of its
// positionIsHeldSQL constant.
func extractPositionIsHeldSQL(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	m := positionIsHeldSQLPattern.FindSubmatch(data)
	if m == nil {
		t.Fatalf("positionIsHeldSQL constant not found in %s (did it get renamed?)", path)
	}
	return string(m[1])
}

// sqlKeywords are excluded from the identifier set compared below: they are
// dialect plumbing (SELECT/EXISTS/...), not the tables/columns the predicate
// actually reasons about.
var sqlKeywords = map[string]bool{
	"SELECT": true, "EXISTS": true, "FROM": true, "WHERE": true,
	"AND": true, "OR": true, "NOT": true,
}

// tenantScopingIdentifiers are excluded too: PostgreSQL is multi-tenant (every
// table the predicate touches carries tenant_id) while SQLite is not (one
// implicit tenant per file, no such column exists at all) — see
// storage.go's "Design notes". The Postgres copy's four EXISTS clauses filter
// on `tenant_id = position.tenant_id` for consistency with the rest of that
// file (position_id is a globally unique key, so this is not a correctness
// requirement); that is infrastructure a single-tenant backend has nothing to
// mirror, not a change to what counts as "held". Excluding it here keeps the
// test doing its real job: catching a *held-by* clause silently gained or
// lost, not this expected, deliberate per-backend asymmetry.
var tenantScopingIdentifiers = map[string]bool{
	"tenant_id": true,
}

var identifierPattern = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*`)

// identifiersIn tokenizes a SQL fragment into the set of table/column names
// it references, dropping keywords, numeric literals (so `= 1` vs a bare
// boolean column doesn't matter) and placeholders (`?1`, `$1` start with a
// non-letter and are never matched by identifierPattern in the first place).
// This is deliberately coarse per the fiche's guard-rail: it must not trip on
// legitimate per-dialect spelling differences, only on a clause — a whole
// table or column — being added or removed.
func identifiersIn(sql string) map[string]bool {
	set := map[string]bool{}
	for _, tok := range identifierPattern.FindAllString(sql, -1) {
		upper := strings.ToUpper(tok)
		lower := strings.ToLower(tok)
		if sqlKeywords[upper] || tenantScopingIdentifiers[lower] {
			continue
		}
		set[lower] = true
	}
	return set
}

func sortedKeys(set map[string]bool) []string {
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// TestPositionIsHeldPredicateParity guards the CLAUDE.md invariant that the
// retention predicate is stated identically in its three deliberate copies.
// A position is held by: another match's move, a collection, an Anki card,
// individually_imported, or flagged (ADR-0006) — see the doc comment on any
// of the three positionIsHeldSQL constants. Nothing but this test enforces
// that the three lists of tables/columns stay in lockstep; the three cannot
// be a single shared Go constant (different SQL dialects: `?1` vs `$1`,
// `= 1` vs a bare boolean, plus different execer types).
//
// The comparison is over the *set* of tables and columns each copy
// references, not the SQL text, so placeholder syntax and boolean spelling
// are not significant differences — only gaining or losing a whole clause is.
func TestPositionIsHeldPredicateParity(t *testing.T) {
	root := repoRootFromThisFile(t)

	sets := make(map[string]map[string]bool, len(positionIsHeldSourceFiles))
	for label, rel := range positionIsHeldSourceFiles {
		sql := extractPositionIsHeldSQL(t, filepath.Join(root, rel))
		sets[label] = identifiersIn(sql)
		if len(sets[label]) == 0 {
			t.Fatalf("%s: no identifiers extracted from positionIsHeldSQL (extraction is broken)", label)
		}
	}

	const reference = "sqlite"
	want := sortedKeys(sets[reference])
	for label, set := range sets {
		if label == reference {
			continue
		}
		got := sortedKeys(set)
		if !equalStringSlices(want, got) {
			t.Errorf("positionIsHeldSQL drift: %s references %v, %s references %v",
				reference, want, label, got)
		}
	}
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
