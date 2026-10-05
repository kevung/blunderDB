package database

import (
	"database/sql"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"testing"
)

// TestMigrate_2_31_ForeignKeysMatchFreshSchema: a library migrated from
// 2.30.0 gets met_id by ALTER TABLE ADD COLUMN and its action columns
// retyped in place, so its foreign keys come from a different path than a
// fresh CREATE TABLE. They must still be the same rules — a migrated
// analysis.met_id without ON DELETE SET NULL would make deleting a MET fail
// (or dangle) on old libraries only.
func TestMigrate_2_31_ForeignKeysMatchFreshSchema(t *testing.T) {
	t.Parallel()
	dir := tempDir(t)
	fresh := filepath.Join(dir, "fresh.db")
	d := NewDatabase()
	if err := d.SetupDatabase(fresh); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	want := foreignKeys(t, d.db)
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	if want["analysis.met_id"] != "match_equity_table.id on_delete=SET NULL on_update=NO ACTION" {
		t.Fatalf("fresh analysis.met_id = %q; the reference itself lost its rule", want["analysis.met_id"])
	}
	for _, swap := range []bool{false, true} {
		t.Run(fmt.Sprintf("swap=%v", swap), func(t *testing.T) {
			path := filepath.Join(dir, fmt.Sprintf("v2300-%v.db", swap))
			write2_30Library(t, path, swap)
			d := NewDatabase()
			if err := d.OpenDatabase(path); err != nil {
				t.Fatalf("open v2.30.0 library: %v", err)
			}
			closeOnCleanup(t, d)
			got := foreignKeys(t, d.db)
			for _, k := range slices.Sorted(maps.Keys(want)) {
				if got[k] != want[k] {
					t.Errorf("migrated %s = %q, fresh %q", k, got[k], want[k])
				}
			}
			for k := range got {
				if _, ok := want[k]; !ok {
					t.Errorf("migrated %s = %q, absent from the fresh schema", k, got[k])
				}
			}
		})
	}
}

// foreignKeys maps table.column to the rule PRAGMA foreign_key_list states.
func foreignKeys(t *testing.T, db *sql.DB) map[string]string {
	t.Helper()
	out := map[string]string{}
	var tables []string
	eachRow(t, db, `SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`, func(r *sql.Rows) {
		var n string
		if err := r.Scan(&n); err != nil {
			t.Fatal(err)
		}
		tables = append(tables, n)
	})
	for _, tbl := range tables {
		eachRow(t, db, `SELECT "table", "from", COALESCE("to", ''), on_update, on_delete FROM pragma_foreign_key_list('`+tbl+`')`, func(r *sql.Rows) {
			var parent, from, to, upd, del string
			if err := r.Scan(&parent, &from, &to, &upd, &del); err != nil {
				t.Fatal(err)
			}
			out[tbl+"."+from] = fmt.Sprintf("%s.%s on_delete=%s on_update=%s", parent, to, del, upd)
		})
	}
	return out
}
