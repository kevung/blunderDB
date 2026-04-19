package main

import (
	"strings"
	"testing"
)

// TestSchemaV200_PositionColumns verifies that a fresh v2.0.0 database has all
// expected scalar columns on the position table and all v2.0.0 indexes.
func TestSchemaV200_PositionColumns(t *testing.T) {
	d := NewDatabase()
	if err := d.SetupDatabase(":memory:"); err != nil {
		t.Fatalf("SetupDatabase failed: %v", err)
	}
	defer d.db.Close()

	wantPositionCols := []string{
		"id", "zobrist_hash", "decision_type", "player_on_roll",
		"dice_1", "dice_2", "cube_value", "cube_owner",
		"score_1", "score_2", "match_length", "has_jacoby", "has_beaver",
		"pip_1", "pip_2", "pip_diff", "off_1", "off_2",
		"back_checkers_1", "back_checkers_2", "no_contact",
		"occupancy_1", "occupancy_2", "point_mask_1", "point_mask_2",
		"state",
	}

	var sql_ string
	if err := d.db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name='position'`).Scan(&sql_); err != nil {
		t.Fatalf("could not read position schema: %v", err)
	}
	for _, col := range wantPositionCols {
		if !strings.Contains(sql_, col) {
			t.Errorf("position table missing column: %s", col)
		}
	}

	wantAnalysisCols := []string{
		"id", "position_id", "data",
		"best_cube_action", "cube_error", "best_move_equity_error",
		"player1_win_rate", "player1_gammon_rate", "player1_backgammon_rate",
		"player2_win_rate", "player2_gammon_rate", "player2_backgammon_rate",
	}
	if err := d.db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name='analysis'`).Scan(&sql_); err != nil {
		t.Fatalf("could not read analysis schema: %v", err)
	}
	for _, col := range wantAnalysisCols {
		if !strings.Contains(sql_, col) {
			t.Errorf("analysis table missing column: %s", col)
		}
	}
}

// TestSchemaV200_Indexes verifies that all expected v2.0.0 indexes are created.
func TestSchemaV200_Indexes(t *testing.T) {
	d := NewDatabase()
	if err := d.SetupDatabase(":memory:"); err != nil {
		t.Fatalf("SetupDatabase failed: %v", err)
	}
	defer d.db.Close()

	wantIndexes := []string{
		"idx_position_zobrist",
		"idx_position_decision_pip",
		"idx_position_decision_dice",
		"idx_position_pip_diff",
		"idx_position_dice",
		"idx_position_off",
		"idx_position_score",
		"idx_position_score_cube",
		"idx_analysis_position",
		"idx_analysis_win_gammon",
		"idx_analysis_win1",
		"idx_analysis_cube_error",
		"idx_analysis_move_error",
		"idx_move_position",
		"idx_move_game",
		"idx_game_match",
	}
	rows, err := d.db.Query(`SELECT name FROM sqlite_master WHERE type='index'`)
	if err != nil {
		t.Fatalf("could not query indexes: %v", err)
	}
	defer rows.Close()
	gotSet := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		gotSet[name] = true
	}
	for _, idx := range wantIndexes {
		if !gotSet[idx] {
			t.Errorf("index missing: %s", idx)
		}
	}
}

// TestSchemaV200_SavePositionColumns verifies that SavePosition writes scalar columns.
func TestSchemaV200_SavePositionColumns(t *testing.T) {
	d := NewDatabase()
	if err := d.SetupDatabase(":memory:"); err != nil {
		t.Fatalf("SetupDatabase failed: %v", err)
	}
	defer d.db.Close()

	pos := &Position{
		Board:        initialBoard(),
		Cube:         Cube{Owner: -1, Value: 0},
		Dice:         [2]int{3, 1},
		Score:        [2]int{5, 5},
		PlayerOnRoll: 0,
		DecisionType: CheckerAction,
	}
	id, err := d.SavePosition(pos)
	if err != nil {
		t.Fatalf("SavePosition failed: %v", err)
	}

	var zobrist int64
	var pip1, pip2 int
	if err := d.db.QueryRow(`SELECT zobrist_hash, pip_1, pip_2 FROM position WHERE id=?`, id).
		Scan(&zobrist, &pip1, &pip2); err != nil {
		t.Fatalf("could not read back position columns: %v", err)
	}
	if zobrist == 0 {
		t.Error("zobrist_hash should not be 0 for initial position")
	}
	if pip1 != 167 || pip2 != 167 {
		t.Errorf("expected pip1=167 pip2=167 for initial position, got pip1=%d pip2=%d", pip1, pip2)
	}
}

// TestSchemaV200_DatabaseVersion verifies that SetupDatabase writes the current DatabaseVersion.
func TestSchemaV200_DatabaseVersion(t *testing.T) {
	d := NewDatabase()
	if err := d.SetupDatabase(":memory:"); err != nil {
		t.Fatalf("SetupDatabase failed: %v", err)
	}
	defer d.db.Close()

	var version string
	if err := d.db.QueryRow(`SELECT value FROM metadata WHERE key='database_version'`).Scan(&version); err != nil {
		t.Fatalf("could not read version: %v", err)
	}
	if version != DatabaseVersion {
		t.Errorf("expected DatabaseVersion %s, got %s", DatabaseVersion, version)
	}
}
