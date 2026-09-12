// TestPurgeOrderMatchesRLSTables never touches a database — it compares two
// package-level slices with the tenant-scoped tables the embedded migrations
// actually create — unlike TestPurgeTenant and friends
// (purge_postgres_test.go, tagged `//go:build postgres`, which provision a
// real PostgreSQL via testcontainers-go). It used to live in that tagged
// file, so `go test ./...` (no `-tags postgres`) never ran it. It lives in its
// own untagged file, `package postgres` (white-box) like its old home, because
// it needs direct access to the unexported package-level variables it guards
// (purgeOrder in purge_postgres.go, rlsTables in rls_postgres.go,
// migrationsFS in migrate_postgres.go).
package postgres

import (
	"io/fs"
	"regexp"
	"slices"
	"strings"
	"testing"
)

var (
	// createTableRE captures a CREATE TABLE statement's name and its column
	// list, up to the `);` that closes it at the start of a line — the layout
	// every migration file follows.
	createTableRE = regexp.MustCompile(`(?ms)^CREATE TABLE IF NOT EXISTS (\w+) \((.*?)^\);`)
	// tenantColumnRE matches a tenant_id column definition inside that list.
	tenantColumnRE = regexp.MustCompile(`(?m)^\s*tenant_id\s`)
	// dropTableRE catches a table a later migration removes again.
	dropTableRE = regexp.MustCompile(`(?m)^DROP TABLE (?:IF EXISTS )?(\w+)`)
)

// migratedTenantTables returns, sorted, every table the embedded migrations
// create with a tenant_id column and do not drop again. It is read from the
// SQL itself so that a table added by a future migration joins the set without
// anyone having to remember a third list.
func migratedTenantTables(t *testing.T) []string {
	t.Helper()
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		t.Fatalf("read embedded migrations: %v", err)
	}
	set := map[string]bool{}
	for _, e := range entries { // ReadDir sorts by name: migrations in order
		if !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		sql, err := migrationsFS.ReadFile("migrations/" + e.Name())
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		for _, m := range createTableRE.FindAllStringSubmatch(string(sql), -1) {
			if tenantColumnRE.MatchString(m[2]) {
				set[m[1]] = true
			}
		}
		for _, m := range dropTableRE.FindAllStringSubmatch(string(sql), -1) {
			delete(set, m[1])
		}
	}
	tables := make([]string, 0, len(set))
	for name := range set {
		tables = append(tables, name)
	}
	slices.Sort(tables)
	return tables
}

// TestPurgeOrderMatchesRLSTables guards purgeOrder (purge_postgres.go) and
// rlsTables (rls_postgres.go) against the schema itself, not merely against
// each other: comparing the two hand-written lists let trash and import_batch
// (2.19.0) and direction and direction_event (2.24.0) go missing from BOTH at
// once, and PurgeTenant left their rows behind while the test stayed green
// (#363). This test needs no database.
func TestPurgeOrderMatchesRLSTables(t *testing.T) {
	want := migratedTenantTables(t)
	if len(want) == 0 {
		t.Fatal("no tenant-scoped table found in the embedded migrations: the parser no longer matches their layout")
	}
	for _, list := range []struct {
		name   string
		tables []string
	}{
		{"purgeOrder", purgeOrder},
		{"rlsTables", rlsTables},
	} {
		got := slices.Clone(list.tables)
		slices.Sort(got)
		if !slices.Equal(got, want) {
			t.Errorf("%s does not list the tenant-scoped tables the migrations create:\n %s (sorted) = %v\n migrations (sorted) = %v",
				list.name, list.name, got, want)
		}
	}
}
