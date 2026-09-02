package sqlite

import (
	"context"
	"database/sql"
	"reflect"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func openMemory(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// TestSchemaStatements_CreateOnly (issue #177): the fresh DDL is made of
// CREATE ... IF NOT EXISTS statements only, so Bootstrap is idempotent and
// referenceSchema, which reads the statements back, sees every column where
// it is declared. The six ALTER TABLE statements that used to trail the match
// and tournament tables are folded into them.
func TestSchemaStatements_CreateOnly(t *testing.T) {
	for _, stmt := range schemaStatements {
		if !strings.HasPrefix(stmt, "CREATE ") {
			t.Errorf("schema statement is not a CREATE: %.60q", stmt)
		}
		if !strings.Contains(stmt, " IF NOT EXISTS ") {
			t.Errorf("schema statement is not idempotent: %.60q", stmt)
		}
	}

	db := openMemory(t)
	ctx := context.Background()
	for i := range 2 {
		if err := Bootstrap(ctx, db); err != nil {
			t.Fatalf("Bootstrap run %d: %v", i+1, err)
		}
	}
	// The folded columns are there, once.
	for _, want := range []string{"tournament_id", "last_visited_position", "canonical_hash", "comment", "tournament_sort_order"} {
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('match') WHERE name = ?`, want).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Errorf("match.%s declared %d times, want 1", want, n)
		}
	}
	var onDelete string
	if err := db.QueryRow(`SELECT on_delete FROM pragma_foreign_key_list('match') WHERE "from" = 'tournament_id'`).Scan(&onDelete); err != nil {
		t.Fatalf("match.tournament_id foreign key: %v", err)
	}
	if onDelete != "SET NULL" {
		t.Errorf("match.tournament_id ON DELETE = %s, want SET NULL", onDelete)
	}
}

// TestCheckSchema (issue #177): CheckSchema names every table, column and
// index the database lacks against the reference, and nothing on a database
// that has them all. It reads only — the drift is still there afterwards.
func TestCheckSchema(t *testing.T) {
	db := openMemory(t)
	ctx := context.Background()
	if err := Bootstrap(ctx, db); err != nil {
		t.Fatal(err)
	}

	drift, err := CheckSchema(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if drift.Count() != 0 {
		t.Fatalf("fresh database reports drift: %+v", drift)
	}

	for _, stmt := range []string{
		`DROP INDEX idx_match_canonical`,
		`ALTER TABLE match DROP COLUMN comment`,
		`ALTER TABLE tournament DROP COLUMN comment`,
		`DROP TABLE anki_review_log`,
		// Not drift: the reference does not name these.
		`CREATE INDEX idx_extra ON match(event)`,
		`ALTER TABLE match ADD COLUMN extra TEXT`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}

	drift, err = CheckSchema(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	// A dropped table takes its indexes with it; they are listed too.
	want := SchemaDrift{
		MissingTables:  []string{"anki_review_log"},
		MissingColumns: []string{"match.comment", "tournament.comment"},
		MissingIndexes: []string{"idx_anki_review_log_card", "idx_anki_review_log_deck", "idx_match_canonical"},
	}
	if !reflect.DeepEqual(drift, want) {
		t.Errorf("CheckSchema = %+v, want %+v", drift, want)
	}
	if drift.Count() != 6 {
		t.Errorf("Count = %d, want 6", drift.Count())
	}

	// The check changed nothing: what was dropped is still gone.
	again, err := CheckSchema(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(again, drift) {
		t.Errorf("second CheckSchema = %+v, want the same drift", again)
	}
}
