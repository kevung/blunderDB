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

// TestSchemaStatements_CreateOnly: the fresh DDL is made of CREATE ... IF NOT
// EXISTS statements only, so Bootstrap is idempotent and referenceSchema sees
// every column where it is declared.
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

// TestCheckSchema: CheckSchema names every table, column and
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

// TestAnalysisMETFallsBackOnDelete: deleting a match equity table nulls the
// met_id of the analyses computed with it, on a fresh library and on one whose
// column EnsureSchema adds — the PostgreSQL key does the same.
func TestAnalysisMETFallsBackOnDelete(t *testing.T) {
	ctx := context.Background()
	db := openMemory(t)
	if err := Bootstrap(ctx, db); err != nil {
		t.Fatal(err)
	}
	var onDelete string
	if err := db.QueryRowContext(ctx, `SELECT on_delete FROM pragma_foreign_key_list('analysis')
		WHERE "from" = 'met_id'`).Scan(&onDelete); err != nil {
		t.Fatalf("analysis.met_id foreign key: %v", err)
	}
	if onDelete != "SET NULL" {
		t.Errorf("analysis.met_id ON DELETE %s, want SET NULL", onDelete)
	}

	ref, err := referenceSchema(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, tbl := range ref.tables {
		if tbl.name != "analysis" {
			continue
		}
		for _, c := range tbl.columns {
			if c.name == "met_id" && !strings.Contains(c.definition, "ON DELETE SET NULL") {
				t.Errorf("met_id is added as %q, without ON DELETE SET NULL", c.definition)
			}
		}
	}
}

// TestEnsureSchemaReclustersDerivedTables: a breakdown table created with a
// rowid is rebuilt clustered, and match_stats emptied with it so the next
// fill recomputes the cells it dropped; a clustered one is left alone.
func TestEnsureSchemaReclustersDerivedTables(t *testing.T) {
	ctx := context.Background()
	db := openMemory(t)
	if err := Bootstrap(ctx, db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO match_stats (match_id, seat) VALUES (1, 1)`); err != nil {
		t.Fatal(err)
	}
	countStats := func() int {
		var n int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM match_stats`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if err := EnsureSchema(ctx, db); err != nil {
		t.Fatal(err)
	}
	if countStats() != 1 {
		t.Fatal("EnsureSchema emptied match_stats over clustered breakdowns")
	}
	if _, err := db.ExecContext(ctx, `DROP TABLE match_stats_cell`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `CREATE TABLE match_stats_cell (match_id INTEGER NOT NULL, seat INTEGER NOT NULL, kind INTEGER NOT NULL, PRIMARY KEY (match_id, seat, kind))`); err != nil {
		t.Fatal(err)
	}
	if err := EnsureSchema(ctx, db); err != nil {
		t.Fatal(err)
	}
	if countStats() != 0 {
		t.Error("match_stats kept rows whose cells were dropped")
	}
	for _, name := range derivedClusteredTables {
		var ddl string
		if err := db.QueryRowContext(ctx, `SELECT sql FROM sqlite_master WHERE name = ?`, name).Scan(&ddl); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(ddl, "WITHOUT ROWID") {
			t.Errorf("%s not clustered after EnsureSchema: %s", name, ddl)
		}
	}
	if drift, err := CheckSchema(ctx, db); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(drift, SchemaDrift{}) {
		t.Errorf("schema drift after reclustering: %+v", drift)
	}
}

// TestEnsureSchemaLeavesANewerLibrarysLayout: a library a newer build wrote
// keeps its breakdown tables and match_stats as they are.
func TestEnsureSchemaLeavesANewerLibrarysLayout(t *testing.T) {
	ctx := context.Background()
	db := openMemory(t)
	if err := Bootstrap(ctx, db); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`INSERT OR REPLACE INTO metadata (key, value) VALUES ('database_version', '99.0.0')`,
		`INSERT INTO match_stats (match_id, seat) VALUES (1, 1)`,
		`DROP TABLE match_stats_cell`,
		`CREATE TABLE match_stats_cell (match_id INTEGER NOT NULL, seat INTEGER NOT NULL, kind INTEGER NOT NULL, PRIMARY KEY (match_id, seat, kind))`,
	} {
		if _, err := db.ExecContext(ctx, q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	if err := EnsureSchema(ctx, db); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM match_stats`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	var ddl string
	if err := db.QueryRowContext(ctx, `SELECT sql FROM sqlite_master WHERE name = 'match_stats_cell'`).Scan(&ddl); err != nil {
		t.Fatal(err)
	}
	if n != 1 || strings.Contains(ddl, "WITHOUT ROWID") {
		t.Errorf("a newer library was reclustered: %d match_stats rows, %s", n, ddl)
	}
}
