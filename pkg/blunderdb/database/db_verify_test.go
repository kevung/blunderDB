package database

import (
	"database/sql"
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
