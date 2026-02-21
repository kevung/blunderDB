package main

import (
	"encoding/json"
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
