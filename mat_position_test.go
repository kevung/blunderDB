package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestCompareTXTvsXGPositions compares positions imported from TXT and XG formats
// to verify that MAT/TXT player-relative moves produce correct absolute positions.
func TestCompareTXTvsXGPositions(t *testing.T) {
	xgFile := filepath.Join("testdata", "test.xg")
	txtFile := filepath.Join("testdata", "test.txt")

	if _, err := os.Stat(xgFile); os.IsNotExist(err) {
		t.Skip("test.xg not found")
	}
	if _, err := os.Stat(txtFile); os.IsNotExist(err) {
		t.Skip("test.txt not found")
	}

	// Import XG
	dbXG, cleanupXG := setupTestDB(t)
	defer cleanupXG()
	xgMatchID, err := dbXG.ImportXGMatch(xgFile)
	if err != nil {
		t.Fatalf("XG import failed: %v", err)
	}

	// Import TXT
	dbTXT, cleanupTXT := setupTestDB(t)
	defer cleanupTXT()
	txtMatchID, err := dbTXT.ImportGnuBGMatch(txtFile)
	if err != nil {
		t.Fatalf("TXT import failed: %v", err)
	}

	// Query positions for each game
	queryPositions := func(db *Database, matchID int64, gameNum int) []struct {
		Pos     Position
		MoveNum int
		Player  int
		Dice1   int
		Dice2   int
		MoveStr string
	} {
		rows, err := db.db.Query(`
			SELECT p.state, m.move_number, m.player, m.dice_1, m.dice_2, COALESCE(m.checker_move, '')
			FROM move m
			JOIN game g ON m.game_id = g.id
			JOIN position p ON m.position_id = p.id
			WHERE g.match_id = ? AND g.game_number = ? AND m.move_type = 'checker'
			ORDER BY m.move_number
		`, matchID, gameNum)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		defer rows.Close()

		var results []struct {
			Pos     Position
			MoveNum int
			Player  int
			Dice1   int
			Dice2   int
			MoveStr string
		}
		for rows.Next() {
			var stateJSON string
			var r struct {
				Pos     Position
				MoveNum int
				Player  int
				Dice1   int
				Dice2   int
				MoveStr string
			}
			rows.Scan(&stateJSON, &r.MoveNum, &r.Player, &r.Dice1, &r.Dice2, &r.MoveStr)
			json.Unmarshal([]byte(stateJSON), &r.Pos)
			results = append(results, r)
		}
		return results
	}

	// Compare all 7 games
	for gameNum := 1; gameNum <= 7; gameNum++ {
		t.Run("Game"+string(rune('0'+gameNum)), func(t *testing.T) {
			xgData := queryPositions(dbXG, xgMatchID, gameNum)
			txtData := queryPositions(dbTXT, txtMatchID, gameNum)

			t.Logf("Game %d: XG has %d positions, TXT has %d positions", gameNum, len(xgData), len(txtData))

			minPos := len(xgData)
			if len(txtData) < minPos {
				minPos = len(txtData)
			}

			mismatchCount := 0
			for i := 0; i < minPos; i++ {
				// Always log the first 5 moves
				if i < 5 {
					t.Logf("  Move %d: XG player=%d dice=%d/%d move=%q | TXT player=%d dice=%d/%d move=%q",
						i, xgData[i].Player, xgData[i].Dice1, xgData[i].Dice2, xgData[i].MoveStr,
						txtData[i].Player, txtData[i].Dice1, txtData[i].Dice2, txtData[i].MoveStr)
				}
				xp := xgData[i].Pos
				tp := txtData[i].Pos

				boardMatch := true
				for pt := 0; pt < 26; pt++ {
					if xp.Board.Points[pt].Checkers != tp.Board.Points[pt].Checkers ||
						xp.Board.Points[pt].Color != tp.Board.Points[pt].Color {
						boardMatch = false
						if mismatchCount < 3 {
							t.Errorf("Game %d Move %d, point %d: XG={%d, color=%d}, TXT={%d, color=%d}",
								gameNum, i, pt, xp.Board.Points[pt].Checkers, xp.Board.Points[pt].Color,
								tp.Board.Points[pt].Checkers, tp.Board.Points[pt].Color)
						}
					}
				}

				if xp.Board.Bearoff != tp.Board.Bearoff {
					boardMatch = false
					if mismatchCount < 3 {
						t.Errorf("Game %d Move %d bearoff: XG=%v, TXT=%v", gameNum, i, xp.Board.Bearoff, tp.Board.Bearoff)
					}
				}

				if !boardMatch {
					mismatchCount++
					if mismatchCount <= 3 {
						t.Logf("  XG  Move %d: player=%d dice=%d/%d move=%q",
							xgData[i].MoveNum, xgData[i].Player, xgData[i].Dice1, xgData[i].Dice2, xgData[i].MoveStr)
						t.Logf("  TXT Move %d: player=%d dice=%d/%d move=%q",
							txtData[i].MoveNum, txtData[i].Player, txtData[i].Dice1, txtData[i].Dice2, txtData[i].MoveStr)
						// Also show the previous move that led to this position
						if i > 0 {
							t.Logf("  (prev XG  Move %d: player=%d dice=%d/%d move=%q)",
								xgData[i-1].MoveNum, xgData[i-1].Player, xgData[i-1].Dice1, xgData[i-1].Dice2, xgData[i-1].MoveStr)
							t.Logf("  (prev TXT Move %d: player=%d dice=%d/%d move=%q)",
								txtData[i-1].MoveNum, txtData[i-1].Player, txtData[i-1].Dice1, txtData[i-1].Dice2, txtData[i-1].MoveStr)
						}
					}
				}
			}

			if mismatchCount > 0 {
				t.Errorf("Game %d: %d/%d positions have board mismatches", gameNum, mismatchCount, minPos)
			} else if minPos > 0 {
				t.Logf("Game %d: all %d compared positions match ✓", gameNum, minPos)
			}
		})
	}
}

// TestCompareMATvsXGPositions compares positions imported from MAT and XG formats
// to verify that MAT numeric bar (25) and bearoff (0) conventions are handled correctly.
func TestCompareMATvsXGPositions(t *testing.T) {
	xgFile := filepath.Join("testdata", "test.xg")
	matFile := filepath.Join("testdata", "test.mat")

	if _, err := os.Stat(xgFile); os.IsNotExist(err) {
		t.Skip("test.xg not found")
	}
	if _, err := os.Stat(matFile); os.IsNotExist(err) {
		t.Skip("test.mat not found")
	}

	// Import XG
	dbXG, cleanupXG := setupTestDB(t)
	defer cleanupXG()
	xgMatchID, err := dbXG.ImportXGMatch(xgFile)
	if err != nil {
		t.Fatalf("XG import failed: %v", err)
	}

	// Import MAT
	dbMAT, cleanupMAT := setupTestDB(t)
	defer cleanupMAT()
	matMatchID, err := dbMAT.ImportGnuBGMatch(matFile)
	if err != nil {
		t.Fatalf("MAT import failed: %v", err)
	}

	// Query positions for each game
	queryPositions := func(db *Database, matchID int64, gameNum int) []struct {
		Pos     Position
		MoveNum int
		Player  int
		Dice1   int
		Dice2   int
		MoveStr string
	} {
		rows, err := db.db.Query(`
			SELECT p.state, m.move_number, m.player, m.dice_1, m.dice_2, COALESCE(m.checker_move, '')
			FROM move m
			JOIN game g ON m.game_id = g.id
			JOIN position p ON m.position_id = p.id
			WHERE g.match_id = ? AND g.game_number = ? AND m.move_type = 'checker'
			ORDER BY m.move_number
		`, matchID, gameNum)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		defer rows.Close()

		var results []struct {
			Pos     Position
			MoveNum int
			Player  int
			Dice1   int
			Dice2   int
			MoveStr string
		}
		for rows.Next() {
			var stateJSON string
			var r struct {
				Pos     Position
				MoveNum int
				Player  int
				Dice1   int
				Dice2   int
				MoveStr string
			}
			rows.Scan(&stateJSON, &r.MoveNum, &r.Player, &r.Dice1, &r.Dice2, &r.MoveStr)
			json.Unmarshal([]byte(stateJSON), &r.Pos)
			results = append(results, r)
		}
		return results
	}

	// Compare all 7 games
	for gameNum := 1; gameNum <= 7; gameNum++ {
		t.Run(fmt.Sprintf("Game%d", gameNum), func(t *testing.T) {
			xgData := queryPositions(dbXG, xgMatchID, gameNum)
			matData := queryPositions(dbMAT, matMatchID, gameNum)

			t.Logf("Game %d: XG has %d positions, MAT has %d positions", gameNum, len(xgData), len(matData))

			minPos := len(xgData)
			if len(matData) < minPos {
				minPos = len(matData)
			}

			mismatchCount := 0
			for i := 0; i < minPos; i++ {
				xp := xgData[i].Pos
				mp := matData[i].Pos

				boardMatch := true
				for pt := 0; pt < 26; pt++ {
					if xp.Board.Points[pt].Checkers != mp.Board.Points[pt].Checkers ||
						xp.Board.Points[pt].Color != mp.Board.Points[pt].Color {
						boardMatch = false
						if mismatchCount < 5 {
							t.Errorf("Game %d Move %d, point %d: XG={%d, color=%d}, MAT={%d, color=%d}",
								gameNum, i, pt, xp.Board.Points[pt].Checkers, xp.Board.Points[pt].Color,
								mp.Board.Points[pt].Checkers, mp.Board.Points[pt].Color)
						}
					}
				}

				if xp.Board.Bearoff != mp.Board.Bearoff {
					boardMatch = false
					if mismatchCount < 5 {
						t.Errorf("Game %d Move %d bearoff: XG=%v, MAT=%v", gameNum, i, xp.Board.Bearoff, mp.Board.Bearoff)
					}
				}

				if !boardMatch {
					mismatchCount++
					if mismatchCount <= 3 {
						t.Logf("  XG  Move %d: player=%d dice=%d/%d move=%q",
							xgData[i].MoveNum, xgData[i].Player, xgData[i].Dice1, xgData[i].Dice2, xgData[i].MoveStr)
						t.Logf("  MAT Move %d: player=%d dice=%d/%d move=%q",
							matData[i].MoveNum, matData[i].Player, matData[i].Dice1, matData[i].Dice2, matData[i].MoveStr)
					}
				}
			}

			if mismatchCount > 0 {
				t.Errorf("Game %d: %d/%d positions have board mismatches", gameNum, mismatchCount, minPos)
			} else if minPos > 0 {
				t.Logf("Game %d: all %d compared positions match ✓", gameNum, minPos)
			}
		})
	}
}

// TestImportMatchFromText simulates clipboard import by importing both MAT and TXT
// content as strings and validating checker counts are correct (15 per player).
func TestImportMatchFromText(t *testing.T) {
	matFile := filepath.Join("testdata", "test.mat")
	txtFile := filepath.Join("testdata", "test.txt")

	for _, tc := range []struct {
		name string
		file string
	}{
		{"MAT", matFile},
		{"TXT", txtFile},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := os.Stat(tc.file); os.IsNotExist(err) {
				t.Skipf("%s not found", tc.file)
			}

			content, err := os.ReadFile(tc.file)
			if err != nil {
				t.Fatalf("Failed to read %s: %v", tc.file, err)
			}

			db, cleanup := setupTestDB(t)
			defer cleanup()

			// Import via text (clipboard simulation)
			matchID, err := db.ImportGnuBGMatchFromText(string(content))
			if err != nil {
				t.Fatalf("ImportGnuBGMatchFromText failed for %s: %v", tc.name, err)
			}

			t.Logf("Imported %s match with ID %d", tc.name, matchID)

			// Verify all positions have correct checker counts
			rows, err := db.db.Query(`
				SELECT p.state, m.move_number, g.game_number
				FROM move m
				JOIN game g ON m.game_id = g.id
				JOIN position p ON m.position_id = p.id
				WHERE g.match_id = ? AND m.move_type = 'checker'
				ORDER BY g.game_number, m.move_number
			`, matchID)
			if err != nil {
				t.Fatalf("Query failed: %v", err)
			}
			defer rows.Close()

			totalPositions := 0
			badPositions := 0
			for rows.Next() {
				var stateJSON string
				var moveNum, gameNum int
				rows.Scan(&stateJSON, &moveNum, &gameNum)

				var pos Position
				json.Unmarshal([]byte(stateJSON), &pos)

				totalPositions++

				// Count checkers per player
				player0 := 0
				player1 := 0
				for i := 0; i < 26; i++ {
					if pos.Board.Points[i].Color == 0 {
						player0 += pos.Board.Points[i].Checkers
					} else if pos.Board.Points[i].Color == 1 {
						player1 += pos.Board.Points[i].Checkers
					}
				}
				player0 += pos.Board.Bearoff[0]
				player1 += pos.Board.Bearoff[1]

				if player0 != 15 || player1 != 15 {
					badPositions++
					if badPositions <= 5 {
						t.Errorf("Game %d Move %d: player0=%d, player1=%d (expected 15 each)",
							gameNum, moveNum, player0, player1)
					}
				}
			}

			if badPositions > 0 {
				t.Errorf("%d/%d positions have wrong checker count", badPositions, totalPositions)
			} else {
				t.Logf("All %d positions have correct checker counts (15 per player) ✓", totalPositions)
			}
		})
	}
}
