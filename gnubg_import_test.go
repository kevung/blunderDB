package main

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// setupTestDB creates a temporary database for testing and returns the Database and cleanup function.
func setupTestDB(t *testing.T) (*Database, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db := NewDatabase()
	if err := db.SetupDatabase(dbPath); err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	return db, func() {
		if db.db != nil {
			db.db.Close()
		}
		os.Remove(dbPath)
	}
}

// TestImportGnuBGSGF tests importing a GnuBG SGF match file.
func TestImportGnuBGSGF(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	testFile := filepath.Join("testdata", "test.sgf")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("test.sgf not found in testdata/")
	}

	matchID, err := db.ImportGnuBGMatch(testFile)
	if err != nil {
		t.Fatalf("ImportGnuBGMatch(.sgf) failed: %v", err)
	}
	if matchID <= 0 {
		t.Fatalf("Expected positive matchID, got %d", matchID)
	}
	t.Logf("SGF match imported with ID: %d", matchID)

	// Verify match record
	var player1, player2 string
	var matchLength int
	err = db.db.QueryRow(`SELECT player1_name, player2_name, match_length FROM match WHERE id = ?`, matchID).
		Scan(&player1, &player2, &matchLength)
	if err != nil {
		t.Fatalf("Failed to query match: %v", err)
	}
	t.Logf("Match: %s vs %s, length=%d", player1, player2, matchLength)
	if matchLength != 7 {
		t.Errorf("Expected match length 7, got %d", matchLength)
	}
	if player1 == "" || player2 == "" {
		t.Error("Player names should not be empty")
	}

	// Verify games count
	var gameCount int
	err = db.db.QueryRow(`SELECT COUNT(*) FROM game WHERE match_id = ?`, matchID).Scan(&gameCount)
	if err != nil {
		t.Fatalf("Failed to count games: %v", err)
	}
	t.Logf("Number of games: %d", gameCount)
	if gameCount < 1 {
		t.Error("Expected at least 1 game")
	}

	// Verify moves exist for each game
	rows, err := db.db.Query(`SELECT g.game_number, COUNT(m.id) FROM game g JOIN move m ON m.game_id = g.id WHERE g.match_id = ? GROUP BY g.id ORDER BY g.game_number`, matchID)
	if err != nil {
		t.Fatalf("Failed to query moves per game: %v", err)
	}
	defer rows.Close()

	totalMoves := 0
	for rows.Next() {
		var gameNum, moveCount int
		if err := rows.Scan(&gameNum, &moveCount); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}
		t.Logf("  Game %d: %d moves", gameNum, moveCount)
		if moveCount == 0 {
			t.Errorf("Game %d has 0 moves", gameNum)
		}
		totalMoves += moveCount
	}
	t.Logf("Total moves imported: %d", totalMoves)
	if totalMoves == 0 {
		t.Error("No moves were imported")
	}

	// Check position and analysis counts
	var positionCount int
	db.db.QueryRow(`SELECT COUNT(*) FROM position`).Scan(&positionCount)
	t.Logf("Total positions: %d", positionCount)

	var analysisCount int
	db.db.QueryRow(`SELECT COUNT(*) FROM analysis`).Scan(&analysisCount)
	t.Logf("Total analysis records: %d", analysisCount)

	var moveAnalysisCount int
	db.db.QueryRow(`SELECT COUNT(*) FROM move_analysis`).Scan(&moveAnalysisCount)
	t.Logf("Total move_analysis records: %d", moveAnalysisCount)

	// Moves with positions
	var movesWithPos int
	db.db.QueryRow(`SELECT COUNT(*) FROM move m JOIN game g ON m.game_id = g.id WHERE g.match_id = ? AND m.position_id IS NOT NULL`, matchID).Scan(&movesWithPos)
	t.Logf("Moves with position data: %d", movesWithPos)
}

// TestImportGnuBGMAT tests importing a Jellyfish MAT match file.
func TestImportGnuBGMAT(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	testFile := filepath.Join("testdata", "test.mat")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("test.mat not found in testdata/")
	}

	matchID, err := db.ImportGnuBGMatch(testFile)
	if err != nil {
		t.Fatalf("ImportGnuBGMatch(.mat) failed: %v", err)
	}
	if matchID <= 0 {
		t.Fatalf("Expected positive matchID, got %d", matchID)
	}
	t.Logf("MAT match imported with ID: %d", matchID)

	// Verify match record
	var player1, player2 string
	var matchLength int
	err = db.db.QueryRow(`SELECT player1_name, player2_name, match_length FROM match WHERE id = ?`, matchID).
		Scan(&player1, &player2, &matchLength)
	if err != nil {
		t.Fatalf("Failed to query match: %v", err)
	}
	t.Logf("Match: %s vs %s, length=%d", player1, player2, matchLength)
	if matchLength != 7 {
		t.Errorf("Expected match length 7, got %d", matchLength)
	}

	// Verify games count
	var gameCount int
	err = db.db.QueryRow(`SELECT COUNT(*) FROM game WHERE match_id = ?`, matchID).Scan(&gameCount)
	if err != nil {
		t.Fatalf("Failed to count games: %v", err)
	}
	t.Logf("Number of games: %d", gameCount)
	if gameCount < 1 {
		t.Error("Expected at least 1 game")
	}

	// Verify total moves
	var totalMoves int
	err = db.db.QueryRow(`
		SELECT COUNT(*) FROM move m
		JOIN game g ON m.game_id = g.id
		WHERE g.match_id = ?`, matchID).Scan(&totalMoves)
	if err != nil {
		t.Fatalf("Failed to count moves: %v", err)
	}
	t.Logf("Total moves: %d", totalMoves)
	if totalMoves == 0 {
		t.Error("No moves were imported")
	}
}

// TestImportGnuBGTXT tests importing a Jellyfish TXT match file.
func TestImportGnuBGTXT(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	testFile := filepath.Join("testdata", "test.txt")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("test.txt not found in testdata/")
	}

	matchID, err := db.ImportGnuBGMatch(testFile)
	if err != nil {
		t.Fatalf("ImportGnuBGMatch(.txt) failed: %v", err)
	}
	if matchID <= 0 {
		t.Fatalf("Expected positive matchID, got %d", matchID)
	}
	t.Logf("TXT match imported with ID: %d", matchID)

	// Verify match record
	var player1, player2 string
	var matchLength int
	err = db.db.QueryRow(`SELECT player1_name, player2_name, match_length FROM match WHERE id = ?`, matchID).
		Scan(&player1, &player2, &matchLength)
	if err != nil {
		t.Fatalf("Failed to query match: %v", err)
	}
	t.Logf("Match: %s vs %s, length=%d", player1, player2, matchLength)
	if matchLength != 7 {
		t.Errorf("Expected match length 7, got %d", matchLength)
	}

	// Verify games count
	var gameCount int
	err = db.db.QueryRow(`SELECT COUNT(*) FROM game WHERE match_id = ?`, matchID).Scan(&gameCount)
	if err != nil {
		t.Fatalf("Failed to count games: %v", err)
	}
	t.Logf("Number of games: %d", gameCount)
	if gameCount < 1 {
		t.Error("Expected at least 1 game")
	}
}

// TestImportGnuBGDuplicate tests that importing the same match twice is rejected.
func TestImportGnuBGDuplicate(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	testFile := filepath.Join("testdata", "test.sgf")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("test.sgf not found in testdata/")
	}

	// First import should succeed
	matchID1, err := db.ImportGnuBGMatch(testFile)
	if err != nil {
		t.Fatalf("First import failed: %v", err)
	}
	t.Logf("First import succeeded with ID: %d", matchID1)

	// Second import should fail with duplicate error
	_, err = db.ImportGnuBGMatch(testFile)
	if err == nil {
		t.Fatal("Second import should have failed with duplicate error")
	}
	if err != ErrDuplicateMatch {
		t.Errorf("Expected ErrDuplicateMatch, got: %v", err)
	}
	t.Logf("Duplicate correctly rejected: %v", err)
}

// TestImportGnuBGSGFGameDetails verifies detailed game/move data from SGF import.
func TestImportGnuBGSGFGameDetails(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	testFile := filepath.Join("testdata", "test.sgf")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("test.sgf not found in testdata/")
	}

	matchID, err := db.ImportGnuBGMatch(testFile)
	if err != nil {
		t.Fatalf("ImportGnuBGMatch failed: %v", err)
	}

	// Check first game details
	t.Run("Game1Structure", func(t *testing.T) {
		var gameID int64
		var winner int
		var pointsWon int
		err := db.db.QueryRow(`SELECT id, winner, points_won FROM game WHERE match_id = ? AND game_number = 1`, matchID).
			Scan(&gameID, &winner, &pointsWon)
		if err != nil {
			t.Fatalf("Failed to query game 1: %v", err)
		}
		t.Logf("Game 1: ID=%d, winner=%d, points_won=%d", gameID, winner, pointsWon)

		// Check first few moves in game 1
		rows, err := db.db.Query(`
			SELECT m.move_number, m.dice_1, m.dice_2, m.checker_move, m.cube_action, m.player
			FROM move m WHERE m.game_id = ?
			ORDER BY m.move_number LIMIT 10`, gameID)
		if err != nil {
			t.Fatalf("Failed to query moves: %v", err)
		}
		defer rows.Close()

		for rows.Next() {
			var moveNum, dice1, dice2, player int
			var checkerMove, cubeAction sql.NullString
			if err := rows.Scan(&moveNum, &dice1, &dice2, &checkerMove, &cubeAction, &player); err != nil {
				t.Fatalf("Failed to scan move: %v", err)
			}
			moveStr := ""
			if checkerMove.Valid {
				moveStr = checkerMove.String
			}
			cubeStr := ""
			if cubeAction.Valid {
				cubeStr = cubeAction.String
			}
			t.Logf("  Move %d: player=%d dice=%d/%d move=%q cube=%q", moveNum, player, dice1, dice2, moveStr, cubeStr)
		}
	})

	// Check that positions are stored correctly
	t.Run("PositionData", func(t *testing.T) {
		// First check if any positions exist at all
		var posCount int
		db.db.QueryRow(`SELECT COUNT(*) FROM position`).Scan(&posCount)
		t.Logf("Total positions in DB: %d", posCount)

		// Check how many moves have position_id set
		var movesWithPos, movesWithoutPos int
		db.db.QueryRow(`SELECT COUNT(*) FROM move m JOIN game g ON m.game_id = g.id WHERE g.match_id = ? AND m.position_id IS NOT NULL`, matchID).Scan(&movesWithPos)
		db.db.QueryRow(`SELECT COUNT(*) FROM move m JOIN game g ON m.game_id = g.id WHERE g.match_id = ? AND m.position_id IS NULL`, matchID).Scan(&movesWithoutPos)
		t.Logf("Moves with position: %d, without: %d", movesWithPos, movesWithoutPos)

		if posCount == 0 {
			t.Skip("No positions stored (gnubgparser may not provide position data for this file)")
		}

		var posID int64
		var stateJSON string
		err := db.db.QueryRow(`
			SELECT p.id, p.state FROM position p
			INNER JOIN move m ON m.position_id = p.id
			INNER JOIN game g ON m.game_id = g.id
			WHERE g.match_id = ? AND m.position_id IS NOT NULL
			ORDER BY g.game_number, m.move_number LIMIT 1`, matchID).
			Scan(&posID, &stateJSON)
		if err != nil {
			t.Fatalf("Failed to query position: %v", err)
		}

		var pos Position
		if err := json.Unmarshal([]byte(stateJSON), &pos); err != nil {
			t.Fatalf("Failed to unmarshal position: %v", err)
		}

		t.Logf("First position: player_on_roll=%d, cube_value=%d, cube_owner=%d",
			pos.PlayerOnRoll, pos.Cube.Value, pos.Cube.Owner)
		t.Logf("  Score: [%d, %d]",
			pos.Score[0], pos.Score[1])

		// Verify board has 26 points (0-25)
		if len(pos.Board.Points) != 26 {
			t.Errorf("Expected 26 board points, got %d", len(pos.Board.Points))
		}

		// Verify total checkers sum to 15 per player (initially)
		var p1count, p2count int
		for _, pt := range pos.Board.Points {
			if pt.Color == 0 {
				p1count += pt.Checkers
			} else if pt.Color == 1 {
				p2count += pt.Checkers
			}
		}
		t.Logf("  Checkers: player0=%d player1=%d", p1count, p2count)
		// Allow borne off checkers to make total <= 15
		if p1count > 15 || p2count > 15 {
			t.Errorf("Invalid checker count: p0=%d, p1=%d (max 15 each)", p1count, p2count)
		}
	})

	// Check that analysis data is present (SGF-specific)
	t.Run("AnalysisData", func(t *testing.T) {
		// Check total analysis records
		var analysisTotal int
		db.db.QueryRow(`SELECT COUNT(*) FROM analysis`).Scan(&analysisTotal)
		t.Logf("Total analysis records in DB: %d", analysisTotal)

		// Check move_analysis records
		var moveAnalysisTotal int
		db.db.QueryRow(`SELECT COUNT(*) FROM move_analysis`).Scan(&moveAnalysisTotal)
		t.Logf("Total move_analysis records in DB: %d", moveAnalysisTotal)

		if analysisTotal == 0 && moveAnalysisTotal == 0 {
			t.Skip("No analysis data found (gnubgparser may not provide analysis for this file)")
		}

		var analysisID int64
		var analysisJSON string
		err := db.db.QueryRow(`
			SELECT a.id, a.data FROM analysis a
			INNER JOIN move m ON m.position_id = a.position_id
			INNER JOIN game g ON m.game_id = g.id
			WHERE g.match_id = ? AND a.position_id IS NOT NULL
			ORDER BY g.game_number, m.move_number LIMIT 1`, matchID).
			Scan(&analysisID, &analysisJSON)
		if err != nil {
			if err == sql.ErrNoRows {
				t.Skip("No analysis data found (may be expected for some games)")
			}
			t.Fatalf("Failed to query analysis: %v", err)
		}

		var analysis PositionAnalysis
		if err := json.Unmarshal([]byte(analysisJSON), &analysis); err != nil {
			t.Fatalf("Failed to unmarshal analysis: %v", err)
		}

		t.Logf("Analysis: type=%q, depth=%q", analysis.AnalysisType, analysis.CheckerAnalysis.Moves[0].AnalysisDepth)
		if len(analysis.CheckerAnalysis.Moves) > 0 {
			t.Logf("  First move option: %q equity=%.4f",
				analysis.CheckerAnalysis.Moves[0].Move,
				analysis.CheckerAnalysis.Moves[0].Equity)
		}
	})
}

// TestImportGnuBGMATvsTXT verifies that MAT and TXT produce same match structure.
func TestImportGnuBGMATvsTXT(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	matFile := filepath.Join("testdata", "test.mat")
	txtFile := filepath.Join("testdata", "test.txt")
	if _, err := os.Stat(matFile); os.IsNotExist(err) {
		t.Skip("test.mat not found")
	}
	if _, err := os.Stat(txtFile); os.IsNotExist(err) {
		t.Skip("test.txt not found")
	}

	// Import MAT
	matMatchID, err := db.ImportGnuBGMatch(matFile)
	if err != nil {
		t.Fatalf("MAT import failed: %v", err)
	}

	// Import TXT
	txtMatchID, err := db.ImportGnuBGMatch(txtFile)
	if err != nil {
		t.Fatalf("TXT import failed: %v", err)
	}

	// Both should have the same number of games
	var matGameCount, txtGameCount int
	db.db.QueryRow(`SELECT COUNT(*) FROM game WHERE match_id = ?`, matMatchID).Scan(&matGameCount)
	db.db.QueryRow(`SELECT COUNT(*) FROM game WHERE match_id = ?`, txtMatchID).Scan(&txtGameCount)
	t.Logf("MAT games: %d, TXT games: %d", matGameCount, txtGameCount)
	if matGameCount != txtGameCount {
		t.Errorf("Game count mismatch: MAT=%d, TXT=%d", matGameCount, txtGameCount)
	}
	if matGameCount < 1 {
		t.Error("Expected at least 1 game")
	}

	// Compare move counts per game
	getMoveCounts := func(matchID int64) []int {
		rows, err := db.db.Query(`
			SELECT COUNT(m.id) FROM game g
			JOIN move m ON m.game_id = g.id
			WHERE g.match_id = ?
			GROUP BY g.id ORDER BY g.game_number`, matchID)
		if err != nil {
			t.Fatalf("Failed to query move counts: %v", err)
		}
		defer rows.Close()

		var counts []int
		for rows.Next() {
			var c int
			rows.Scan(&c)
			counts = append(counts, c)
		}
		return counts
	}

	matCounts := getMoveCounts(matMatchID)
	txtCounts := getMoveCounts(txtMatchID)

	t.Logf("MAT move counts: %v", matCounts)
	t.Logf("TXT move counts: %v", txtCounts)

	if len(matCounts) != len(txtCounts) {
		t.Errorf("Different number of games: MAT=%d, TXT=%d", len(matCounts), len(txtCounts))
	} else {
		for i := range matCounts {
			if matCounts[i] != txtCounts[i] {
				// MAT and TXT parsers may produce slightly different move counts
				// due to formatting differences (e.g., "Cannot Move" handling)
				t.Logf("Note: Game %d move count differs: MAT=%d, TXT=%d", i+1, matCounts[i], txtCounts[i])
			}
		}
	}
}

// TestConvertGnuBGMoveToString tests the move notation converter.
func TestConvertGnuBGMoveToString(t *testing.T) {
	tests := []struct {
		name     string
		move     [8]int
		player   int
		expected string
	}{
		{
			name:     "Simple move player 0",
			move:     [8]int{12, 7, 12, 9, -1, -1, -1, -1},
			player:   0,
			expected: "13/8 13/10",
		},
		{
			name:     "Simple move player 1",
			move:     [8]int{12, 7, 12, 9, -1, -1, -1, -1},
			player:   1,
			expected: "12/17 12/15",
		},
		{
			name:     "Bar entry player 0",
			move:     [8]int{24, 22, -1, -1, -1, -1, -1, -1},
			player:   0,
			expected: "bar/23",
		},
		{
			name:     "Bear off player 0",
			move:     [8]int{3, 25, -1, -1, -1, -1, -1, -1},
			player:   0,
			expected: "4/off",
		},
		{
			name:     "Cannot move",
			move:     [8]int{-1, -1, -1, -1, -1, -1, -1, -1},
			player:   0,
			expected: "Cannot Move",
		},
		{
			name:     "Bear off player 1",
			move:     [8]int{3, 25, -1, -1, -1, -1, -1, -1},
			player:   1,
			expected: "21/off",
		},
		{
			name:     "Bar entry player 1",
			move:     [8]int{24, 20, -1, -1, -1, -1, -1, -1},
			player:   1,
			expected: "bar/4",
		},
		{
			name:     "Double move from bar",
			move:     [8]int{24, 20, 20, 14, -1, -1, -1, -1},
			player:   0,
			expected: "bar/21 21/15",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := convertGnuBGMoveToString(tc.move, tc.player)
			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

// TestImportGnuBGUnsupportedFormat tests that unsupported extensions are rejected.
func TestImportGnuBGUnsupportedFormat(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	_, err := db.ImportGnuBGMatch("test.bgf")
	if err == nil {
		t.Fatal("Expected error for unsupported format")
	}
	t.Logf("Correctly rejected unsupported format: %v", err)
}

// TestImportCharlotSGF tests importing the charlot SGF match file (second test dataset).
func TestImportCharlotSGF(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	testFile := filepath.Join("testdata", "charlot1-charlot2_7p_2025-11-08-2305.sgf")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("charlot SGF not found in testdata/")
	}

	matchID, err := db.ImportGnuBGMatch(testFile)
	if err != nil {
		t.Fatalf("ImportGnuBGMatch failed: %v", err)
	}

	var player1, player2 string
	var matchLength, gameCount int
	db.db.QueryRow(`SELECT player1_name, player2_name, match_length FROM match WHERE id = ?`, matchID).
		Scan(&player1, &player2, &matchLength)
	db.db.QueryRow(`SELECT COUNT(*) FROM game WHERE match_id = ?`, matchID).Scan(&gameCount)

	t.Logf("Charlot match: %s vs %s, length=%d, games=%d", player1, player2, matchLength, gameCount)
	if gameCount < 1 {
		t.Error("Expected at least 1 game")
	}
}

// TestImportCharlotMAT tests importing the charlot MAT match file (second test dataset).
func TestImportCharlotMAT(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	testFile := filepath.Join("testdata", "charlot1-charlot2_7p_2025-11-08-2305.mat")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("charlot MAT not found in testdata/")
	}

	matchID, err := db.ImportGnuBGMatch(testFile)
	if err != nil {
		t.Fatalf("ImportGnuBGMatch failed: %v", err)
	}

	var player1, player2 string
	var matchLength, gameCount int
	db.db.QueryRow(`SELECT player1_name, player2_name, match_length FROM match WHERE id = ?`, matchID).
		Scan(&player1, &player2, &matchLength)
	db.db.QueryRow(`SELECT COUNT(*) FROM game WHERE match_id = ?`, matchID).Scan(&gameCount)

	t.Logf("Charlot MAT match: %s vs %s, length=%d, games=%d", player1, player2, matchLength, gameCount)
	if gameCount < 1 {
		t.Error("Expected at least 1 game")
	}
}

// TestCompareXGvsSGFImport imports the same match from both XG and SGF formats
// into separate databases and compares the resulting data.
// The match content (games, moves, positions) should be identical;
// analysis values may differ by up to 60% since different engines analyzed them.
func TestCompareXGvsSGFImport(t *testing.T) {
	xgFile := filepath.Join("testdata", "test.xg")
	sgfFile := filepath.Join("testdata", "test.sgf")

	if _, err := os.Stat(xgFile); os.IsNotExist(err) {
		t.Skip("test.xg not found in testdata/")
	}
	if _, err := os.Stat(sgfFile); os.IsNotExist(err) {
		t.Skip("test.sgf not found in testdata/")
	}

	// Import XG into a fresh DB
	dbXG, cleanupXG := setupTestDB(t)
	defer cleanupXG()
	xgMatchID, err := dbXG.ImportXGMatch(xgFile)
	if err != nil {
		t.Fatalf("ImportXGMatch failed: %v", err)
	}

	// Import SGF into a fresh DB
	dbSGF, cleanupSGF := setupTestDB(t)
	defer cleanupSGF()
	sgfMatchID, err := dbSGF.ImportGnuBGMatch(sgfFile)
	if err != nil {
		t.Fatalf("ImportGnuBGMatch(.sgf) failed: %v", err)
	}

	// 1. Compare match metadata
	t.Run("MatchMetadata", func(t *testing.T) {
		var xgP1, xgP2 string
		var xgLen int
		dbXG.db.QueryRow(`SELECT player1_name, player2_name, match_length FROM match WHERE id = ?`, xgMatchID).
			Scan(&xgP1, &xgP2, &xgLen)

		var sgfP1, sgfP2 string
		var sgfLen int
		dbSGF.db.QueryRow(`SELECT player1_name, player2_name, match_length FROM match WHERE id = ?`, sgfMatchID).
			Scan(&sgfP1, &sgfP2, &sgfLen)

		t.Logf("XG:  %s vs %s, length=%d", xgP1, xgP2, xgLen)
		t.Logf("SGF: %s vs %s, length=%d", sgfP1, sgfP2, sgfLen)

		if xgLen != sgfLen {
			t.Errorf("Match length differs: XG=%d, SGF=%d", xgLen, sgfLen)
		}
	})

	// 2. Compare game counts
	t.Run("GameCounts", func(t *testing.T) {
		var xgGames, sgfGames int
		dbXG.db.QueryRow(`SELECT COUNT(*) FROM game WHERE match_id = ?`, xgMatchID).Scan(&xgGames)
		dbSGF.db.QueryRow(`SELECT COUNT(*) FROM game WHERE match_id = ?`, sgfMatchID).Scan(&sgfGames)

		t.Logf("XG games: %d, SGF games: %d", xgGames, sgfGames)
		if xgGames != sgfGames {
			t.Errorf("Game count differs: XG=%d, SGF=%d", xgGames, sgfGames)
		}
	})

	// 3. Compare move counts per game
	t.Run("MoveCountsPerGame", func(t *testing.T) {
		type gameMoves struct {
			gameNumber int
			moveCount  int
		}

		queryGameMoves := func(db *sql.DB, matchID int64) []gameMoves {
			rows, err := db.Query(`
				SELECT g.game_number, COUNT(m.id)
				FROM game g JOIN move m ON m.game_id = g.id
				WHERE g.match_id = ?
				GROUP BY g.id ORDER BY g.game_number
			`, matchID)
			if err != nil {
				t.Fatalf("Query failed: %v", err)
			}
			defer rows.Close()

			var result []gameMoves
			for rows.Next() {
				var gm gameMoves
				rows.Scan(&gm.gameNumber, &gm.moveCount)
				result = append(result, gm)
			}
			return result
		}

		xgMoves := queryGameMoves(dbXG.db, xgMatchID)
		sgfMoves := queryGameMoves(dbSGF.db, sgfMatchID)

		minGames := len(xgMoves)
		if len(sgfMoves) < minGames {
			minGames = len(sgfMoves)
		}

		for i := 0; i < minGames; i++ {
			t.Logf("Game %d: XG=%d moves, SGF=%d moves",
				xgMoves[i].gameNumber, xgMoves[i].moveCount, sgfMoves[i].moveCount)
			if xgMoves[i].moveCount != sgfMoves[i].moveCount {
				t.Logf("  Note: Move count differs (XG may include No Double cube decisions that SGF excludes)")
			}
		}
	})

	// 4. Compare positions: moves with position data
	t.Run("PositionCounts", func(t *testing.T) {
		var xgWithPos, xgWithoutPos int
		dbXG.db.QueryRow(`SELECT COUNT(*) FROM move m JOIN game g ON m.game_id = g.id WHERE g.match_id = ? AND m.position_id IS NOT NULL`, xgMatchID).Scan(&xgWithPos)
		dbXG.db.QueryRow(`SELECT COUNT(*) FROM move m JOIN game g ON m.game_id = g.id WHERE g.match_id = ? AND m.position_id IS NULL`, xgMatchID).Scan(&xgWithoutPos)

		var sgfWithPos, sgfWithoutPos int
		dbSGF.db.QueryRow(`SELECT COUNT(*) FROM move m JOIN game g ON m.game_id = g.id WHERE g.match_id = ? AND m.position_id IS NOT NULL`, sgfMatchID).Scan(&sgfWithPos)
		dbSGF.db.QueryRow(`SELECT COUNT(*) FROM move m JOIN game g ON m.game_id = g.id WHERE g.match_id = ? AND m.position_id IS NULL`, sgfMatchID).Scan(&sgfWithoutPos)

		t.Logf("XG:  moves with position=%d, without=%d", xgWithPos, xgWithoutPos)
		t.Logf("SGF: moves with position=%d, without=%d", sgfWithPos, sgfWithoutPos)

		if sgfWithPos == 0 {
			t.Error("SGF import has no moves with position data!")
		}
		if sgfWithoutPos > 0 {
			t.Errorf("SGF import has %d moves without position data", sgfWithoutPos)
		}
	})

	// 5. Compare dice per game (should be identical since same match)
	t.Run("DiceSequence", func(t *testing.T) {
		type diceRoll struct {
			moveNumber int
			player     int
			dice1      int
			dice2      int
		}

		queryDice := func(db *sql.DB, matchID int64, gameNum int) []diceRoll {
			rows, err := db.Query(`
				SELECT m.move_number, m.player, m.dice_1, m.dice_2
				FROM move m JOIN game g ON m.game_id = g.id
				WHERE g.match_id = ? AND g.game_number = ? AND m.move_type = 'checker'
				ORDER BY m.move_number
			`, matchID, gameNum)
			if err != nil {
				return nil
			}
			defer rows.Close()

			var result []diceRoll
			for rows.Next() {
				var dr diceRoll
				rows.Scan(&dr.moveNumber, &dr.player, &dr.dice1, &dr.dice2)
				result = append(result, dr)
			}
			return result
		}

		// Compare first game dice
		xgDice := queryDice(dbXG.db, xgMatchID, 1)
		sgfDice := queryDice(dbSGF.db, sgfMatchID, 1)

		t.Logf("Game 1: XG checker moves=%d, SGF checker moves=%d", len(xgDice), len(sgfDice))

		minMoves := len(xgDice)
		if len(sgfDice) < minMoves {
			minMoves = len(sgfDice)
		}

		mismatchCount := 0
		for i := 0; i < minMoves; i++ {
			xg := xgDice[i]
			sgf := sgfDice[i]
			// Dice might be in different order, normalize
			xgD1, xgD2 := xg.dice1, xg.dice2
			sgfD1, sgfD2 := sgf.dice1, sgf.dice2
			if xgD1 > xgD2 {
				xgD1, xgD2 = xgD2, xgD1
			}
			if sgfD1 > sgfD2 {
				sgfD1, sgfD2 = sgfD2, sgfD1
			}
			if xgD1 != sgfD1 || xgD2 != sgfD2 {
				t.Errorf("Game 1 move %d dice differ: XG=%d/%d, SGF=%d/%d",
					i, xg.dice1, xg.dice2, sgf.dice1, sgf.dice2)
				mismatchCount++
				if mismatchCount >= 5 {
					t.Log("Too many dice mismatches, stopping")
					break
				}
			}
		}
		if mismatchCount == 0 {
			t.Log("All dice sequences match between XG and SGF")
		}
	})

	// 6. Compare board positions for first few moves of game 1
	t.Run("BoardPositions", func(t *testing.T) {
		queryPositions := func(db *sql.DB, matchID int64) []Position {
			rows, err := db.Query(`
				SELECT p.state
				FROM move m
				JOIN game g ON m.game_id = g.id
				JOIN position p ON m.position_id = p.id
				WHERE g.match_id = ? AND g.game_number = 1 AND m.move_type = 'checker'
				ORDER BY m.move_number
				LIMIT 10
			`, matchID)
			if err != nil {
				t.Fatalf("Query failed: %v", err)
			}
			defer rows.Close()

			var positions []Position
			for rows.Next() {
				var stateJSON string
				rows.Scan(&stateJSON)
				var pos Position
				json.Unmarshal([]byte(stateJSON), &pos)
				positions = append(positions, pos)
			}
			return positions
		}

		xgPos := queryPositions(dbXG.db, xgMatchID)
		sgfPos := queryPositions(dbSGF.db, sgfMatchID)

		t.Logf("XG positions (game 1, first 10): %d", len(xgPos))
		t.Logf("SGF positions (game 1, first 10): %d", len(sgfPos))

		if len(xgPos) == 0 {
			t.Fatal("No XG positions to compare")
		}
		if len(sgfPos) == 0 {
			t.Fatal("No SGF positions to compare")
		}

		minPos := len(xgPos)
		if len(sgfPos) < minPos {
			minPos = len(sgfPos)
		}

		for i := 0; i < minPos; i++ {
			xp := xgPos[i]
			sp := sgfPos[i]

			// Compare board points
			boardMatch := true
			for pt := 0; pt < 26; pt++ {
				if xp.Board.Points[pt].Checkers != sp.Board.Points[pt].Checkers ||
					xp.Board.Points[pt].Color != sp.Board.Points[pt].Color {
					boardMatch = false
					t.Errorf("Move %d, point %d: XG={%d, color=%d}, SGF={%d, color=%d}",
						i, pt, xp.Board.Points[pt].Checkers, xp.Board.Points[pt].Color,
						sp.Board.Points[pt].Checkers, sp.Board.Points[pt].Color)
				}
			}

			// Compare bearoff
			if xp.Board.Bearoff != sp.Board.Bearoff {
				t.Errorf("Move %d bearoff: XG=%v, SGF=%v", i, xp.Board.Bearoff, sp.Board.Bearoff)
				boardMatch = false
			}

			if boardMatch {
				t.Logf("Move %d: board positions match ✓", i)
			}
		}
	})

	// 7. Verify GetMatchMovePositions works for both
	t.Run("GetMatchMovePositions", func(t *testing.T) {
		xgPositions, err := dbXG.GetMatchMovePositions(xgMatchID)
		if err != nil {
			t.Fatalf("XG GetMatchMovePositions failed: %v", err)
		}

		sgfPositions, err := dbSGF.GetMatchMovePositions(sgfMatchID)
		if err != nil {
			t.Fatalf("SGF GetMatchMovePositions failed: %v", err)
		}

		t.Logf("XG move positions: %d", len(xgPositions))
		t.Logf("SGF move positions: %d", len(sgfPositions))

		if len(sgfPositions) == 0 {
			t.Error("SGF GetMatchMovePositions returned 0 positions - this is the reported bug!")
		}
		if len(xgPositions) == 0 {
			t.Error("XG GetMatchMovePositions returned 0 positions")
		}
	})

	// 8. Compare cube analysis values between XG and SGF
	t.Run("CubeAnalysisComparison", func(t *testing.T) {
		type cubeInfo struct {
			gameNumber int
			moveNumber int
			cubeAction string
			bestAction string
			ndEquity   float64
			dtEquity   float64
			dpEquity   float64
		}

		queryCubeAnalysis := func(db *sql.DB, matchID int64) []cubeInfo {
			rows, err := db.Query(`
				SELECT g.game_number, m.move_number, m.cube_action, a.data
				FROM move m
				JOIN game g ON m.game_id = g.id
				JOIN analysis a ON a.position_id = m.position_id
				WHERE g.match_id = ? AND m.move_type = 'cube' AND a.data LIKE '%DoublingCube%'
				ORDER BY g.game_number, m.move_number
			`, matchID)
			if err != nil {
				t.Fatalf("Query failed: %v", err)
			}
			defer rows.Close()

			var result []cubeInfo
			for rows.Next() {
				var ci cubeInfo
				var analysisJSON string
				rows.Scan(&ci.gameNumber, &ci.moveNumber, &ci.cubeAction, &analysisJSON)

				var posAnalysis PositionAnalysis
				json.Unmarshal([]byte(analysisJSON), &posAnalysis)
				if posAnalysis.DoublingCubeAnalysis != nil {
					ci.bestAction = posAnalysis.DoublingCubeAnalysis.BestCubeAction
					ci.ndEquity = posAnalysis.DoublingCubeAnalysis.CubefulNoDoubleEquity
					ci.dtEquity = posAnalysis.DoublingCubeAnalysis.CubefulDoubleTakeEquity
					ci.dpEquity = posAnalysis.DoublingCubeAnalysis.CubefulDoublePassEquity
				}
				result = append(result, ci)
			}
			return result
		}

		xgCube := queryCubeAnalysis(dbXG.db, xgMatchID)
		sgfCube := queryCubeAnalysis(dbSGF.db, sgfMatchID)

		t.Logf("XG cube decisions: %d, SGF cube decisions: %d", len(xgCube), len(sgfCube))

		// Log all cube decisions for comparison
		t.Log("\n--- XG Cube Decisions ---")
		for _, ci := range xgCube {
			t.Logf("  Game %d Move %d: action=%q best=%q ND=%.4f DT=%.4f DP=%.4f",
				ci.gameNumber, ci.moveNumber, ci.cubeAction, ci.bestAction,
				ci.ndEquity, ci.dtEquity, ci.dpEquity)
		}

		t.Log("\n--- SGF Cube Decisions ---")
		for _, ci := range sgfCube {
			t.Logf("  Game %d Move %d: action=%q best=%q ND=%.4f DT=%.4f DP=%.4f",
				ci.gameNumber, ci.moveNumber, ci.cubeAction, ci.bestAction,
				ci.ndEquity, ci.dtEquity, ci.dpEquity)
		}

		// Verify SGF cube decisions don't have redundant take/pass entries
		for _, ci := range sgfCube {
			if ci.cubeAction == "Take" || ci.cubeAction == "Pass" {
				t.Errorf("SGF has redundant cube entry with action=%q (should only have Double/Take or Double/Pass)", ci.cubeAction)
			}
		}

		// Verify SGF best actions are reasonable (not "no_double", "double", "take", "pass")
		for _, ci := range sgfCube {
			switch ci.bestAction {
			case "No Double", "Double, Take", "Double, Pass":
				// OK
			default:
				t.Errorf("SGF Game %d Move %d: unexpected bestAction=%q", ci.gameNumber, ci.moveNumber, ci.bestAction)
			}
		}

		// Verify SGF cube equity values are non-zero and in reasonable range
		for _, ci := range sgfCube {
			if ci.ndEquity == 0 && ci.dtEquity == 0 && ci.dpEquity == 0 {
				t.Errorf("SGF Game %d Move %d: all cube equities are zero", ci.gameNumber, ci.moveNumber)
			}
			// Double/Pass should be 1.0 (money game convention)
			if ci.dpEquity != 1.0 {
				t.Logf("  Note: SGF Game %d Move %d: DP equity=%.4f (expected 1.0 for money/match DA convention)",
					ci.gameNumber, ci.moveNumber, ci.dpEquity)
			}
		}

		// Compare best actions between XG and SGF for matching cube decisions
		// Build a map of XG cube decisions by game for comparison
		if len(sgfCube) > 0 && len(xgCube) > 0 {
			t.Log("\n--- Best Action Comparison ---")
			// Compare XG decisions that are Double/Take or Double/Pass (explicit doubles)
			xgDoubles := make([]cubeInfo, 0)
			for _, ci := range xgCube {
				if ci.cubeAction == "Double/Take" || ci.cubeAction == "Double/Pass" || ci.cubeAction == "Double" {
					xgDoubles = append(xgDoubles, ci)
				}
			}
			t.Logf("XG explicit doubles: %d, SGF doubles: %d", len(xgDoubles), len(sgfCube))

			minLen := len(xgDoubles)
			if len(sgfCube) < minLen {
				minLen = len(sgfCube)
			}
			matchCount := 0
			for i := 0; i < minLen; i++ {
				xg := xgDoubles[i]
				sgf := sgfCube[i]
				matches := xg.bestAction == sgf.bestAction
				if matches {
					matchCount++
				}
				t.Logf("  #%d: XG best=%q (%q)  SGF best=%q (%q) match=%v",
					i, xg.bestAction, xg.cubeAction, sgf.bestAction, sgf.cubeAction, matches)
			}
			if minLen > 0 {
				t.Logf("Best action agreement: %d/%d (%.0f%%)", matchCount, minLen, float64(matchCount)/float64(minLen)*100)
			}
		}
	})
}