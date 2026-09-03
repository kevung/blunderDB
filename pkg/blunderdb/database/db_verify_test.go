package database

import (
	"database/sql"
	"path/filepath"
	"reflect"
	"testing"
)

// plantOrphans inserts one orphan of each kind through db with foreign keys
// switched off — the only way to create rows the schema's ON DELETE CASCADE
// would otherwise forbid, and exactly how the pre-#157 pool produced them.
// db must be pinned to a single connection so the PRAGMA toggles apply to
// the connection the inserts run on.
func plantOrphans(t *testing.T, db *sql.DB) {
	t.Helper()
	stmts := []string{
		`PRAGMA foreign_keys = OFF`,
		`INSERT INTO game (id, match_id, game_number) VALUES (9001, 424242, 1)`,
		`INSERT INTO move (id, game_id, move_number, move_type) VALUES (9002, 424242, 1, 'checker')`,
		`INSERT INTO move_analysis (id, move_id, analysis_type) VALUES (9003, 424242, 'checker')`,
		`INSERT INTO analysis (position_id, data) VALUES (424242, X'00')`,
		`PRAGMA foreign_keys = ON`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("%s: %v", s, err)
		}
	}
}

func TestCountOrphans(t *testing.T) {
	d := NewDatabase()
	if err := d.SetupDatabase(":memory:"); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })

	clean, err := d.CountOrphans()
	if err != nil {
		t.Fatalf("CountOrphans on a fresh database: %v", err)
	}
	if clean != (OrphanCounts{}) {
		t.Fatalf("fresh database reports orphans: %+v", clean)
	}

	plantOrphans(t, d.Conn())

	got, err := d.CountOrphans()
	if err != nil {
		t.Fatalf("CountOrphans: %v", err)
	}
	want := OrphanCounts{GamesWithoutMatch: 1, MovesWithoutGame: 1, MoveAnalysesWithoutMove: 1, AnalysesWithoutPosition: 1}
	if got != want {
		t.Errorf("CountOrphans = %+v, want %+v", got, want)
	}
	if got.Total() != 4 {
		t.Errorf("Total = %d, want 4", got.Total())
	}
}

// TestCheckSchema_ReportsWhatEnsureSchemaCouldNotAdd (issue #177): EnsureSchema
// degrades to a warning when an index cannot be built; the drift it leaves
// is what CheckSchema reports after the open. Two matches sharing a
// canonical hash keep idx_match_canonical (UNIQUE) from being rebuilt.
func TestCheckSchema_ReportsWhatEnsureSchemaCouldNotAdd(t *testing.T) {
	path := filepath.Join(t.TempDir(), "drift.db")
	d := NewDatabase()
	if err := d.SetupDatabase(path); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	drift, err := d.CheckSchema()
	if err != nil {
		t.Fatalf("CheckSchema on a fresh database: %v", err)
	}
	if drift.Count() != 0 {
		t.Fatalf("fresh database reports drift: %+v", drift)
	}
	for _, stmt := range []string{
		`DROP INDEX idx_match_canonical`,
		`INSERT INTO match (player1_name, player2_name, canonical_hash) VALUES ('a', 'b', 'same')`,
		`INSERT INTO match (player1_name, player2_name, canonical_hash) VALUES ('c', 'd', 'same')`,
	} {
		if _, err := d.Conn().Exec(stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	d.Close()

	// Reopening runs EnsureSchema, which cannot rebuild the index and says so
	// in the log only; the database opens all the same.
	d = NewDatabase()
	if err := d.OpenDatabase(path); err != nil {
		t.Fatalf("OpenDatabase: %v", err)
	}
	defer d.Close()
	drift, err = d.CheckSchema()
	if err != nil {
		t.Fatalf("CheckSchema: %v", err)
	}
	if want := []string{"idx_match_canonical"}; !reflect.DeepEqual(drift.MissingIndexes, want) {
		t.Errorf("MissingIndexes = %v, want %v", drift.MissingIndexes, want)
	}
	if len(drift.MissingTables)+len(drift.MissingColumns) != 0 {
		t.Errorf("unexpected drift beyond the index: %+v", drift)
	}
}

// TestCheckConstraints_ReportsWhatSQLiteCannotEnforce (issue #173): the fresh
// DDL states range constraints SQLite cannot add to a table that already
// exists, so an upgraded database can hold rows a new one would refuse. That
// gap is not a silence — `blunderdb verify` reads it here.
func TestCheckConstraints_ReportsWhatSQLiteCannotEnforce(t *testing.T) {
	d := newTestDB(t)

	violations, err := d.CheckConstraints()
	if err != nil {
		t.Fatalf("CheckConstraints on a fresh database: %v", err)
	}
	if len(violations) == 0 {
		t.Fatal("CheckConstraints returned no rule at all")
	}
	if n := TotalConstraintViolations(violations); n != 0 {
		t.Fatalf("fresh database: %d violation(s), want 0: %+v", n, violations)
	}

	// A fresh database refuses the offending rows outright — which is the point
	// of the CHECK — so the violation has to be created the way an upgraded
	// file carries it: on a table built before the rule existed.
	for _, stmt := range []string{
		`ALTER TABLE position RENAME TO position_checked`,
		`CREATE TABLE position (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			zobrist_hash INTEGER, decision_type INTEGER, player_on_roll INTEGER,
			dice_1 INTEGER, dice_2 INTEGER, cube_value INTEGER, cube_owner INTEGER,
			score_1 INTEGER, score_2 INTEGER, match_length INTEGER,
			has_jacoby INTEGER, has_beaver INTEGER,
			pip_1 INTEGER, pip_2 INTEGER, pip_diff INTEGER, off_1 INTEGER, off_2 INTEGER,
			back_checkers_1 INTEGER, back_checkers_2 INTEGER, no_contact INTEGER,
			occupancy_1 INTEGER, occupancy_2 INTEGER, point_mask_1 INTEGER, point_mask_2 INTEGER,
			state TEXT NOT NULL, is_cube_response INTEGER NOT NULL DEFAULT 0,
			individually_imported INTEGER NOT NULL DEFAULT 0, flagged INTEGER NOT NULL DEFAULT 0)`,
		// Three rows no current schema would take: no hash, an impossible die,
		// more than fifteen checkers borne off.
		`INSERT INTO position (state) VALUES ('{}')`,
		`INSERT INTO position (zobrist_hash, dice_1, state) VALUES (11, 9, '{}')`,
		`INSERT INTO position (zobrist_hash, off_1, state) VALUES (12, 42, '{}')`,
	} {
		if _, err := d.Conn().Exec(stmt); err != nil {
			t.Fatalf("build a pre-rule position table (%s): %v", stmt, err)
		}
	}

	violations, err = d.CheckConstraints()
	if err != nil {
		t.Fatalf("CheckConstraints: %v", err)
	}
	got := map[string]int64{}
	for _, v := range violations {
		got[v.Name] = v.Count
	}
	for name, want := range map[string]int64{
		"position.zobrist_hash NOT NULL":  1,
		"position.dice_1 BETWEEN 0 AND 6": 1,
		"position.off_1 BETWEEN 0 AND 15": 1,
		"position.dice_2 BETWEEN 0 AND 6": 0,
	} {
		if got[name] != want {
			t.Errorf("%s: got %d, want %d", name, got[name], want)
		}
	}
	if n := TotalConstraintViolations(violations); n != 3 {
		t.Errorf("total violations: got %d, want 3", n)
	}
}

// TestCheckCounters_RecomputesTheDenormalisedFigures (issue #185):
// match.game_count and game.move_count are written once, at import, from what
// the source file held, and are what the match list displays. Nothing else in
// the database says the figure and the rows disagree.
func TestCheckCounters_RecomputesTheDenormalisedFigures(t *testing.T) {
	d := newTestDB(t)

	drift, err := d.CheckCounters()
	if err != nil {
		t.Fatalf("CheckCounters on an empty database: %v", err)
	}
	if drift.Total() != 0 {
		t.Fatalf("empty database: %+v, want no drift", drift)
	}

	// One match with an honest count, one that claims three games and holds one.
	honest, err := d.Conn().Exec(`INSERT INTO match (player1_name, game_count) VALUES ('a', 1)`)
	if err != nil {
		t.Fatal(err)
	}
	honestID, _ := honest.LastInsertId()
	liar, err := d.Conn().Exec(`INSERT INTO match (player1_name, game_count) VALUES ('b', 3)`)
	if err != nil {
		t.Fatal(err)
	}
	liarID, _ := liar.LastInsertId()
	for _, matchID := range []int64{honestID, liarID} {
		if _, err := d.Conn().Exec(
			`INSERT INTO game (match_id, game_number, move_count) VALUES (?, 1, 0)`, matchID); err != nil {
			t.Fatal(err)
		}
	}

	drift, err = d.CheckCounters()
	if err != nil {
		t.Fatalf("CheckCounters: %v", err)
	}
	if drift.MatchesWithWrongGameCount != 1 {
		t.Errorf("matches with a wrong game_count: got %d, want 1", drift.MatchesWithWrongGameCount)
	}
	if drift.WorstGameCountGap != 2 {
		t.Errorf("worst game_count gap: got %d, want 2", drift.WorstGameCountGap)
	}
	if drift.GamesWithWrongMoveCount != 0 {
		t.Errorf("games with a wrong move_count: got %d, want 0", drift.GamesWithWrongMoveCount)
	}

	// A game that claims no move and holds one.
	if _, err := d.Conn().Exec(
		`INSERT INTO move (game_id, move_number, move_type) VALUES (1, 1, 'checker')`); err != nil {
		t.Fatal(err)
	}
	drift, err = d.CheckCounters()
	if err != nil {
		t.Fatalf("CheckCounters: %v", err)
	}
	if drift.GamesWithWrongMoveCount != 1 || drift.WorstMoveCountGap != 1 {
		t.Errorf("move_count drift: got %d game(s), worst gap %d, want 1 and 1",
			drift.GamesWithWrongMoveCount, drift.WorstMoveCountGap)
	}
	if drift.Total() != 2 {
		t.Errorf("total drift: got %d, want 2", drift.Total())
	}
}
