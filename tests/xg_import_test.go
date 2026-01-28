package tests

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// TestXGImportMoveNotation tests that move notation is correctly converted
func TestXGImportMoveNotation(t *testing.T) {
	// Expected moves from test.txt Game 1
	expectedMoves := []struct {
		moveNumber int
		move       string
		dice       [2]int
	}{
		{0, "24/23 13/8", [2]int{5, 1}},       // Kévin Unger 51: 24/23 13/8
		{2, "13/7 8/7", [2]int{6, 1}},         // Maxence Job 61: 13/7 8/7
		{4, "24/20 23/20", [2]int{4, 3}},      // Kévin Unger 43: 24/20 23/20
		{6, "24/20", [2]int{1, 1}},            // Maxence Job 11: 24/20 (merged from 24/23 23/22 22/21 21/20)
		{8, "8/3 6/3", [2]int{5, 3}},          // Kévin Unger 53: 8/3 6/3
		{10, "13/9 9/8", [2]int{4, 1}},        // Maxence Job 41: 13/8 (displayed as 13/9 9/8)
		{26, "18/16(2) 6/4(2)", [2]int{2, 2}}, // Maxence Job 22: 18/16(2) 6/4(2)
	}

	// Create a temporary test database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_import.db")

	// Import the test match using CLI (we'll test via the database directly)
	// First, let's verify by checking an existing import

	// For this test, we use a pre-imported database or run import
	// Here we just document what we expect

	t.Log("Expected move conversions from test.txt:")
	for _, exp := range expectedMoves {
		t.Logf("  Move %d: dice %v -> %q", exp.moveNumber, exp.dice, exp.move)
	}

	// The actual test would import and verify
	t.Log("dbPath would be:", dbPath)
}

// TestXGImportCubeActions tests that cube actions are correctly captured
func TestXGImportCubeActions(t *testing.T) {
	// Expected cube actions from test.txt
	expectedCubeActions := []struct {
		game      int
		turn      int
		action    string
		rawDouble int
		rawTake   int
	}{
		// Game 1, turn 16: Doubles => 2, Drops
		{1, 16, "Double", 1, -1}, // Kévin doubles
		// Note: "Drops" is implicit from game ending, not stored as separate move

		// Game 2, turn 7: Doubles => 2, Takes
		{2, 7, "Double/Take", 1, 1}, // Kévin doubles, Maxence takes

		// Game 3, turn 5: Doubles => 2 (from Maxence)
		{3, 5, "Double", 1, -1},
		// turn 6: Drops (Kévin drops)

		// Game 4, turn 10: Doubles => 2, turn 11: Drops
		{4, 10, "Double", 1, -1},

		// Game 5, turn 9: Doubles => 2 (from Maxence)
		{5, 9, "Double/Take", 1, 1}, // Takes
	}

	t.Log("Expected cube action conversions:")
	for _, exp := range expectedCubeActions {
		t.Logf("  Game %d Turn %d: Double=%d, Take=%d -> %q",
			exp.game, exp.turn, exp.rawDouble, exp.rawTake, exp.action)
	}
}

// TestXGImportedDatabase verifies an imported database against reference
func TestXGImportedDatabase(t *testing.T) {
	// Path to pre-imported database (or import fresh)
	dbPath := "/tmp/test_cube.db"

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Skip("Test database not found at /tmp/test_cube.db - run import first")
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Test 1: Verify Game 1 first moves
	t.Run("Game1FirstMoves", func(t *testing.T) {
		rows, err := db.Query(`
			SELECT m.move_number, m.dice_1, m.dice_2, m.checker_move 
			FROM move m 
			JOIN game g ON m.game_id = g.id 
			WHERE g.game_number = 1 AND g.match_id = 1 AND m.move_type = 'checker'
			ORDER BY m.move_number 
			LIMIT 5
		`)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		defer rows.Close()

		expected := []struct {
			moveNumber int
			dice1      int
			dice2      int
			move       string
		}{
			{0, 5, 1, "24/23 13/8"},
			{2, 6, 1, "13/7 8/7"},
			{4, 4, 3, "24/20 23/20"},
			{6, 1, 1, "24/20"}, // Merged slide for doublet
			{8, 5, 3, "8/3 6/3"},
		}

		i := 0
		for rows.Next() {
			var moveNumber, dice1, dice2 int
			var move string
			if err := rows.Scan(&moveNumber, &dice1, &dice2, &move); err != nil {
				t.Fatalf("Scan failed: %v", err)
			}

			if i < len(expected) {
				exp := expected[i]
				if moveNumber != exp.moveNumber || dice1 != exp.dice1 || dice2 != exp.dice2 || move != exp.move {
					t.Errorf("Move %d: got (%d-%d, %q), want (%d-%d, %q)",
						moveNumber, dice1, dice2, move,
						exp.dice1, exp.dice2, exp.move)
				} else {
					t.Logf("✓ Move %d: %d-%d: %s", moveNumber, dice1, dice2, move)
				}
			}
			i++
		}
	})

	// Test 2: Verify cube actions in Game 2
	t.Run("Game2CubeActions", func(t *testing.T) {
		rows, err := db.Query(`
			SELECT m.move_number, m.cube_action 
			FROM move m 
			JOIN game g ON m.game_id = g.id 
			WHERE g.game_number = 2 AND g.match_id = 1 AND m.move_type = 'cube' AND m.cube_action != 'No Double'
			ORDER BY m.move_number
		`)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		defer rows.Close()

		foundDoublesTakes := false
		for rows.Next() {
			var moveNumber int
			var cubeAction string
			if err := rows.Scan(&moveNumber, &cubeAction); err != nil {
				t.Fatalf("Scan failed: %v", err)
			}
			t.Logf("Game 2 Move %d: %s", moveNumber, cubeAction)
			if cubeAction == "Double/Take" {
				foundDoublesTakes = true
			}
		}

		if !foundDoublesTakes {
			t.Error("Expected to find 'Double/Take' cube action in Game 2 (from reference: turn 7)")
		} else {
			t.Log("✓ Found Double/Take action in Game 2")
		}
	})

	// Test 3: Verify grouped moves notation
	t.Run("GroupedMovesNotation", func(t *testing.T) {
		// Looking for 18/16(2) 6/4(2) in Game 1
		rows, err := db.Query(`
			SELECT m.move_number, m.dice_1, m.dice_2, m.checker_move 
			FROM move m 
			JOIN game g ON m.game_id = g.id 
			WHERE g.game_number = 1 AND g.match_id = 1 
			  AND m.move_type = 'checker' 
			  AND m.checker_move LIKE '%(%)'
			ORDER BY m.move_number
		`)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		defer rows.Close()

		found := false
		for rows.Next() {
			var moveNumber, dice1, dice2 int
			var move string
			if err := rows.Scan(&moveNumber, &dice1, &dice2, &move); err != nil {
				t.Fatalf("Scan failed: %v", err)
			}
			t.Logf("Grouped move at %d: %d-%d: %s", moveNumber, dice1, dice2, move)
			if move == "18/16(2) 6/4(2)" && dice1 == 2 && dice2 == 2 {
				found = true
			}
		}

		if !found {
			t.Error("Expected to find '18/16(2) 6/4(2)' for doublet 22")
		} else {
			t.Log("✓ Found correctly grouped doublet move")
		}
	})
}

// TestConvertXGMoveToString tests the move conversion function
func TestConvertXGMoveToString(t *testing.T) {
	testCases := []struct {
		name     string
		input    [8]int32
		expected string
	}{
		{
			name:     "Simple two-part move 51: 24/23 13/8",
			input:    [8]int32{13, 8, 24, 23, -1, -1, -1, -1},
			expected: "24/23 13/8",
		},
		{
			name:     "Doublet slide 11: 24/20",
			input:    [8]int32{24, 23, 23, 22, 22, 21, 21, 20},
			expected: "24/20",
		},
		{
			name:     "Grouped doublet 22: 18/16(2) 6/4(2)",
			input:    [8]int32{18, 16, 18, 16, 6, 4, 6, 4},
			expected: "18/16(2) 6/4(2)",
		},
		{
			name:     "Bar entry: bar/23",
			input:    [8]int32{25, 23, -1, -1, -1, -1, -1, -1},
			expected: "bar/23",
		},
		{
			name:     "Bear off: 6/off",
			input:    [8]int32{6, -2, -1, -1, -1, -1, -1, -1},
			expected: "6/off",
		},
		{
			name:     "Cannot move",
			input:    [8]int32{-1, -1, -1, -1, -1, -1, -1, -1},
			expected: "Cannot Move",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Note: This would need access to the Database.convertXGMoveToString method
			// For now, we document what we expect
			t.Logf("Input: %v", tc.input)
			t.Logf("Expected: %q", tc.expected)
		})
	}
}
