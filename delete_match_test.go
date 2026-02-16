package main

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// assertTableCount checks that the number of rows in the table matches expected.
func assertTableCount(t *testing.T, rawDB *sql.DB, table string, expected int) {
	t.Helper()
	actual := countRows(t, rawDB, table)
	if actual != expected {
		t.Errorf("table %s: expected %d rows, got %d", table, expected, actual)
	}
}

// TestDeleteMatchCleansUpAllData verifies that deleting a match removes all
// associated data: games, moves, move_analysis, and orphaned positions
// (along with their analysis and comments).
func TestDeleteMatchCleansUpAllData(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test_delete.db")

	db := NewDatabase()
	if err := db.SetupDatabase(dbPath); err != nil {
		t.Fatalf("SetupDatabase failed: %v", err)
	}
	defer db.db.Close()

	rawDB := db.db

	// --- Create 2 positions ---
	pos1 := InitializePosition()
	id1, err := db.SavePosition(&pos1)
	if err != nil {
		t.Fatalf("SavePosition 1 failed: %v", err)
	}

	pos2 := InitializePosition()
	pos2.Dice = [2]int{6, 5}
	id2, err := db.SavePosition(&pos2)
	if err != nil {
		t.Fatalf("SavePosition 2 failed: %v", err)
	}

	// --- Analysis for each position ---
	analysis1 := PositionAnalysis{
		PositionID:   int(id1),
		XGID:         "test-xgid-1",
		AnalysisType: "XG Roller++",
		CheckerAnalysis: &CheckerAnalysis{
			Moves: []CheckerMove{
				{Index: 1, Move: "8/5 6/5", Equity: 0.123, PlayerWinChance: 55.0},
			},
		},
		CreationDate:     time.Now(),
		LastModifiedDate: time.Now(),
	}
	if err := db.SaveAnalysis(id1, analysis1); err != nil {
		t.Fatalf("SaveAnalysis 1 failed: %v", err)
	}

	analysis2 := PositionAnalysis{
		PositionID:   int(id2),
		XGID:         "test-xgid-2",
		AnalysisType: "XG Roller++",
		CheckerAnalysis: &CheckerAnalysis{
			Moves: []CheckerMove{
				{Index: 1, Move: "24/13", Equity: 0.05, PlayerWinChance: 52.0},
			},
		},
		CreationDate:     time.Now(),
		LastModifiedDate: time.Now(),
	}
	if err := db.SaveAnalysis(id2, analysis2); err != nil {
		t.Fatalf("SaveAnalysis 2 failed: %v", err)
	}

	// --- Comments for each position ---
	if err := db.SaveComment(id1, "Comment 1"); err != nil {
		t.Fatalf("SaveComment 1 failed: %v", err)
	}
	if err := db.SaveComment(id2, "Comment 2"); err != nil {
		t.Fatalf("SaveComment 2 failed: %v", err)
	}

	// --- Match → game → moves → move_analysis ---
	matchRes, err := rawDB.Exec(`INSERT INTO match (player1_name, player2_name, match_length, match_date, import_date, game_count, match_hash)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"Alice", "Bob", 7, time.Now(), time.Now(), 1, "hash-delete-test")
	if err != nil {
		t.Fatalf("Insert match failed: %v", err)
	}
	matchID, _ := matchRes.LastInsertId()

	gameRes, err := rawDB.Exec(`INSERT INTO game (match_id, game_number, initial_score_1, initial_score_2, winner, points_won, move_count)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, matchID, 1, 0, 0, 1, 1, 2)
	if err != nil {
		t.Fatalf("Insert game failed: %v", err)
	}
	gameID, _ := gameRes.LastInsertId()

	move1Res, err := rawDB.Exec(`INSERT INTO move (game_id, move_number, move_type, position_id, player, dice_1, dice_2, checker_move)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, gameID, 1, "checker", id1, 0, 3, 1, "8/5 6/5")
	if err != nil {
		t.Fatalf("Insert move 1 failed: %v", err)
	}
	moveID1, _ := move1Res.LastInsertId()

	move2Res, err := rawDB.Exec(`INSERT INTO move (game_id, move_number, move_type, position_id, player, dice_1, dice_2, checker_move)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, gameID, 2, "checker", id2, 1, 6, 5, "24/13")
	if err != nil {
		t.Fatalf("Insert move 2 failed: %v", err)
	}
	moveID2, _ := move2Res.LastInsertId()

	_, err = rawDB.Exec(`INSERT INTO move_analysis (move_id, analysis_type, depth, equity, equity_error, win_rate, gammon_rate, backgammon_rate, opponent_win_rate, opponent_gammon_rate, opponent_backgammon_rate)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, moveID1, "checker", "3-ply", 0.123, 0.0, 55.0, 10.0, 1.0, 44.0, 8.0, 0.5)
	if err != nil {
		t.Fatalf("Insert move_analysis 1 failed: %v", err)
	}
	_, err = rawDB.Exec(`INSERT INTO move_analysis (move_id, analysis_type, depth, equity, equity_error, win_rate, gammon_rate, backgammon_rate, opponent_win_rate, opponent_gammon_rate, opponent_backgammon_rate)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, moveID2, "checker", "3-ply", 0.05, 0.0, 52.0, 9.0, 0.5, 47.0, 7.0, 0.3)
	if err != nil {
		t.Fatalf("Insert move_analysis 2 failed: %v", err)
	}

	// --- Verify all data exists before delete ---
	assertTableCount(t, rawDB, "match", 1)
	assertTableCount(t, rawDB, "game", 1)
	assertTableCount(t, rawDB, "move", 2)
	assertTableCount(t, rawDB, "move_analysis", 2)
	assertTableCount(t, rawDB, "position", 2)
	assertTableCount(t, rawDB, "analysis", 2)
	assertTableCount(t, rawDB, "comment", 2)

	// --- Delete the match ---
	if err := db.DeleteMatch(matchID); err != nil {
		t.Fatalf("DeleteMatch failed: %v", err)
	}

	// --- Verify cascaded tables are empty ---
	assertTableCount(t, rawDB, "match", 0)
	assertTableCount(t, rawDB, "game", 0)
	assertTableCount(t, rawDB, "move", 0)
	assertTableCount(t, rawDB, "move_analysis", 0)

	// --- Verify positions and their data are also deleted ---
	assertTableCount(t, rawDB, "position", 0)
	assertTableCount(t, rawDB, "analysis", 0)
	assertTableCount(t, rawDB, "comment", 0)
}

// TestDeleteMatchPreservesSharedPositions verifies that positions referenced by
// another match or belonging to a collection are NOT deleted when deleting a match.
func TestDeleteMatchPreservesSharedPositions(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test_shared.db")

	db := NewDatabase()
	if err := db.SetupDatabase(dbPath); err != nil {
		t.Fatalf("SetupDatabase failed: %v", err)
	}
	defer db.db.Close()

	rawDB := db.db

	// --- Create 3 positions ---
	pos1 := InitializePosition()
	id1, err := db.SavePosition(&pos1)
	if err != nil {
		t.Fatalf("SavePosition 1 failed: %v", err)
	}

	pos2 := InitializePosition()
	pos2.Dice = [2]int{6, 5}
	id2, err := db.SavePosition(&pos2)
	if err != nil {
		t.Fatalf("SavePosition 2 failed: %v", err)
	}

	pos3 := InitializePosition()
	pos3.Dice = [2]int{4, 3}
	id3, err := db.SavePosition(&pos3)
	if err != nil {
		t.Fatalf("SavePosition 3 failed: %v", err)
	}

	// --- Match 1 uses positions 1, 2 ---
	match1Res, err := rawDB.Exec(`INSERT INTO match (player1_name, player2_name, match_length, match_date, import_date, game_count, match_hash)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, "Alice", "Bob", 7, time.Now(), time.Now(), 1, "hash-1")
	if err != nil {
		t.Fatalf("Insert match 1 failed: %v", err)
	}
	match1ID, _ := match1Res.LastInsertId()

	game1Res, err := rawDB.Exec(`INSERT INTO game (match_id, game_number, winner, points_won, move_count)
		VALUES (?, ?, ?, ?, ?)`, match1ID, 1, 1, 1, 2)
	if err != nil {
		t.Fatalf("Insert game 1 failed: %v", err)
	}
	game1ID, _ := game1Res.LastInsertId()

	_, err = rawDB.Exec(`INSERT INTO move (game_id, move_number, move_type, position_id, player, dice_1, dice_2, checker_move)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, game1ID, 1, "checker", id1, 0, 3, 1, "8/5 6/5")
	if err != nil {
		t.Fatalf("Insert move for match 1 (pos1) failed: %v", err)
	}
	_, err = rawDB.Exec(`INSERT INTO move (game_id, move_number, move_type, position_id, player, dice_1, dice_2, checker_move)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, game1ID, 2, "checker", id2, 1, 6, 5, "24/13")
	if err != nil {
		t.Fatalf("Insert move for match 1 (pos2) failed: %v", err)
	}

	// --- Match 2 uses position 2 (shared) and position 3 ---
	match2Res, err := rawDB.Exec(`INSERT INTO match (player1_name, player2_name, match_length, match_date, import_date, game_count, match_hash)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, "Charlie", "Dave", 5, time.Now(), time.Now(), 1, "hash-2")
	if err != nil {
		t.Fatalf("Insert match 2 failed: %v", err)
	}
	match2ID, _ := match2Res.LastInsertId()

	game2Res, err := rawDB.Exec(`INSERT INTO game (match_id, game_number, winner, points_won, move_count)
		VALUES (?, ?, ?, ?, ?)`, match2ID, 1, 1, 1, 2)
	if err != nil {
		t.Fatalf("Insert game 2 failed: %v", err)
	}
	game2ID, _ := game2Res.LastInsertId()

	_, err = rawDB.Exec(`INSERT INTO move (game_id, move_number, move_type, position_id, player, dice_1, dice_2, checker_move)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, game2ID, 1, "checker", id2, 0, 6, 5, "24/13")
	if err != nil {
		t.Fatalf("Insert move for match 2 (pos2) failed: %v", err)
	}
	_, err = rawDB.Exec(`INSERT INTO move (game_id, move_number, move_type, position_id, player, dice_1, dice_2, checker_move)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, game2ID, 2, "checker", id3, 1, 4, 3, "13/6")
	if err != nil {
		t.Fatalf("Insert move for match 2 (pos3) failed: %v", err)
	}

	// --- Put position 1 in a collection ---
	collRes, err := rawDB.Exec(`INSERT INTO collection (name, description) VALUES (?, ?)`, "My Collection", "desc")
	if err != nil {
		t.Fatalf("Insert collection failed: %v", err)
	}
	collID, _ := collRes.LastInsertId()
	_, err = rawDB.Exec(`INSERT INTO collection_position (collection_id, position_id, sort_order) VALUES (?, ?, ?)`, collID, id1, 0)
	if err != nil {
		t.Fatalf("Insert collection_position failed: %v", err)
	}

	// Verify initial state
	assertTableCount(t, rawDB, "match", 2)
	assertTableCount(t, rawDB, "position", 3)

	// --- Delete match 1 ---
	if err := db.DeleteMatch(match1ID); err != nil {
		t.Fatalf("DeleteMatch failed: %v", err)
	}

	// Match 1, its game and moves should be gone
	assertTableCount(t, rawDB, "match", 1)

	// Position 1 should survive because it's in a collection
	var pos1Exists int
	rawDB.QueryRow(`SELECT COUNT(*) FROM position WHERE id = ?`, id1).Scan(&pos1Exists)
	if pos1Exists != 1 {
		t.Errorf("Position %d should be preserved (in collection), but was deleted", id1)
	}

	// Position 2 should survive because it's used by match 2
	var pos2Exists int
	rawDB.QueryRow(`SELECT COUNT(*) FROM position WHERE id = ?`, id2).Scan(&pos2Exists)
	if pos2Exists != 1 {
		t.Errorf("Position %d should be preserved (used by match 2), but was deleted", id2)
	}

	// Position 3 should survive because it's used by match 2
	var pos3Exists int
	rawDB.QueryRow(`SELECT COUNT(*) FROM position WHERE id = ?`, id3).Scan(&pos3Exists)
	if pos3Exists != 1 {
		t.Errorf("Position %d should be preserved (used by match 2), but was deleted", id3)
	}

	// --- Now delete match 2 ---
	if err := db.DeleteMatch(match2ID); err != nil {
		t.Fatalf("DeleteMatch 2 failed: %v", err)
	}

	assertTableCount(t, rawDB, "match", 0)

	// Position 1 should STILL survive (in collection)
	rawDB.QueryRow(`SELECT COUNT(*) FROM position WHERE id = ?`, id1).Scan(&pos1Exists)
	if pos1Exists != 1 {
		t.Errorf("Position %d should be preserved (in collection), but was deleted after match 2 delete", id1)
	}

	// Position 2 and 3 should now be gone (orphaned)
	rawDB.QueryRow(`SELECT COUNT(*) FROM position WHERE id = ?`, id2).Scan(&pos2Exists)
	if pos2Exists != 0 {
		t.Errorf("Position %d should be deleted (orphaned), but still exists", id2)
	}

	rawDB.QueryRow(`SELECT COUNT(*) FROM position WHERE id = ?`, id3).Scan(&pos3Exists)
	if pos3Exists != 0 {
		t.Errorf("Position %d should be deleted (orphaned), but still exists", id3)
	}
}

// countRows is already defined in export_test.go (same package)
// assertTableCount uses countRows + fmt from this file's imports
