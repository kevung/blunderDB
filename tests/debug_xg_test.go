package tests

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/kevung/xgparser/xgparser"
)

func TestDebugXGParsing(t *testing.T) {
	// Parse the test XG file
	match, err := xgparser.ParseXGFromFile("../testdata/test.xg")
	if err != nil {
		t.Fatalf("Failed to parse XG file: %v", err)
	}

	fmt.Printf("Match: %s vs %s\n", match.Metadata.Player1Name, match.Metadata.Player2Name)
	fmt.Printf("Match Length: %d\n", match.Metadata.MatchLength)
	fmt.Printf("Games: %d\n\n", len(match.Games))

	// Look at the first game
	if len(match.Games) > 0 {
		game := match.Games[0]
		fmt.Printf("Game 1:\n")
		fmt.Printf("  Initial Score: %v\n", game.InitialScore)
		fmt.Printf("  Winner: %d\n", game.Winner)
		fmt.Printf("  Points Won: %d\n", game.PointsWon)
		fmt.Printf("  Moves: %d\n\n", len(game.Moves))

		// Print first 10 moves
		for i, move := range game.Moves {
			if i >= 20 {
				break
			}

			fmt.Printf("Move %d: %s\n", i, move.MoveType)

			if move.MoveType == "checker" && move.CheckerMove != nil {
				cm := move.CheckerMove
				fmt.Printf("  ActivePlayer: %d\n", cm.ActivePlayer)
				fmt.Printf("  Dice: %v\n", cm.Dice)
				fmt.Printf("  PlayedMove (raw): %v\n", cm.PlayedMove)
				fmt.Printf("  Position.Score: %v\n", cm.Position.Score)
				fmt.Printf("  Position.Cube: %d, CubePos: %d\n", cm.Position.Cube, cm.Position.CubePos)

				// Print checker analysis if available
				if len(cm.Analysis) > 0 {
					fmt.Printf("  Analysis (first move):\n")
					a := cm.Analysis[0]
					fmt.Printf("    Move (raw int8): %v\n", a.Move)
					fmt.Printf("    Equity: %.4f\n", a.Equity)
					fmt.Printf("    Win Rate: %.4f\n", a.Player1WinRate)
				}
			} else if move.MoveType == "cube" && move.CubeMove != nil {
				cm := move.CubeMove
				fmt.Printf("  ActivePlayer: %d\n", cm.ActivePlayer)
				fmt.Printf("  CubeAction: %d\n", cm.CubeAction)
			}
			fmt.Println()
		}
	}
}

func TestConvertMoveNotation(t *testing.T) {
	// Test conversion of XG move format to standard notation
	// Reference from test.txt Game 1:
	// 1) 51: 24/23 13/8   (Kévin Unger, Player 1 / X)
	// 1) 61: 13/7 8/7     (Maxence Job, Player 2 / O)

	match, err := xgparser.ParseXGFromFile("../testdata/test.xg")
	if err != nil {
		t.Fatalf("Failed to parse XG file: %v", err)
	}

	fmt.Println("\n=== Testing Move Notation Conversion ===")
	fmt.Println("Reference from test.txt:")
	fmt.Println("  Game 1, Move 1: Kévin Unger 51: 24/23 13/8")
	fmt.Println("  Game 1, Move 1: Maxence Job 61: 13/7 8/7")
	fmt.Println()

	if len(match.Games) > 0 {
		game := match.Games[0]

		// Find the first checker move
		checkerMoveCount := 0
		for i, move := range game.Moves {
			if move.MoveType == "checker" && move.CheckerMove != nil {
				checkerMoveCount++
				cm := move.CheckerMove

				fmt.Printf("Checker Move %d (Index %d):\n", checkerMoveCount, i)
				fmt.Printf("  ActivePlayer: %d ", cm.ActivePlayer)
				if cm.ActivePlayer == -1 {
					fmt.Printf("(Player 1 / X = %s)\n", match.Metadata.Player1Name)
				} else {
					fmt.Printf("(Player 2 / O = %s)\n", match.Metadata.Player2Name)
				}
				fmt.Printf("  Dice: %v\n", cm.Dice)
				fmt.Printf("  PlayedMove (raw [8]int32): %v\n", cm.PlayedMove)

				// Convert to notation
				moveStr := convertMoveToNotation(cm.PlayedMove)
				fmt.Printf("  Converted notation: %s\n", moveStr)

				// If this is ActivePlayer -1, the position is stored from their perspective
				// But we need to show it from Player 1's standard perspective (point 24 is furthest)
				if cm.ActivePlayer == -1 {
					// No swap needed - Player 1 counts from their home (1-6) to opponent's home (19-24)
					fmt.Printf("  Standard notation for Player 1: %s\n", moveStr)
				} else {
					// Player 2's perspective - need to swap to get standard view
					swappedMove := swapMovePoints(cm.PlayedMove)
					swappedStr := convertMoveToNotation(swappedMove)
					fmt.Printf("  Standard notation for Player 2 (swapped): %s\n", swappedStr)
				}

				fmt.Println()

				if checkerMoveCount >= 4 {
					break
				}
			}
		}
	}
}

// convertMoveToNotation converts raw XG move format to standard notation
// XG format: [from, to, from, to, ...] where:
//   - 1-24 are board points
//   - 25 is the bar
//   - -2 is bear off
//   - -1 is unused/end of move
func convertMoveToNotation(playedMove [8]int32) string {
	var moves []string
	for i := 0; i < 8; i += 2 {
		from := playedMove[i]
		to := playedMove[i+1]
		if from == -1 || to == -1 {
			break
		}

		// Convert to notation
		var fromStr, toStr string

		if from == 25 {
			fromStr = "bar"
		} else {
			fromStr = fmt.Sprintf("%d", from)
		}

		if to == -2 {
			toStr = "off"
		} else {
			toStr = fmt.Sprintf("%d", to)
		}

		moves = append(moves, fmt.Sprintf("%s/%s", fromStr, toStr))
	}

	if len(moves) == 0 {
		return "Cannot Move"
	}
	return join(moves, " ")
}

// swapMovePoints converts move from one player's perspective to the other
func swapMovePoints(move [8]int32) [8]int32 {
	var swapped [8]int32
	for i := range swapped {
		swapped[i] = -1
	}

	for i := 0; i < 8; i += 2 {
		from := move[i]
		to := move[i+1]

		if from == -1 {
			break
		}

		// Swap: point N becomes point (25-N)
		if from == 25 {
			swapped[i] = 25 // Bar stays bar
		} else if from >= 1 && from <= 24 {
			swapped[i] = 25 - from
		} else {
			swapped[i] = from
		}

		if to == -2 {
			swapped[i+1] = -2 // Bear off stays bear off
		} else if to >= 1 && to <= 24 {
			swapped[i+1] = 25 - to
		} else {
			swapped[i+1] = to
		}
	}

	return swapped
}

func join(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}

func TestPrintMatchJSON(t *testing.T) {
	match, err := xgparser.ParseXGFromFile("../testdata/test.xg")
	if err != nil {
		t.Fatalf("Failed to parse XG file: %v", err)
	}

	// Print first game as JSON
	if len(match.Games) > 0 {
		game := match.Games[0]

		// Limit to first 5 moves for readability
		limitedGame := xgparser.Game{
			GameNumber:   game.GameNumber,
			InitialScore: game.InitialScore,
			Winner:       game.Winner,
			PointsWon:    game.PointsWon,
		}

		if len(game.Moves) > 10 {
			limitedGame.Moves = game.Moves[:10]
		} else {
			limitedGame.Moves = game.Moves
		}

		jsonData, err := json.MarshalIndent(limitedGame, "", "  ")
		if err != nil {
			t.Fatalf("Failed to marshal game: %v", err)
		}

		fmt.Println("\n=== Game 1 JSON (first 10 moves) ===")
		fmt.Println(string(jsonData))
	}
}

func TestAnalyzePositionFormat(t *testing.T) {
	// Analyze the position format in XG to understand board representation
	match, err := xgparser.ParseXGFromFile("../testdata/test.xg")
	if err != nil {
		t.Fatalf("Failed to parse XG file: %v", err)
	}

	fmt.Println("\n=== Analyzing XG Position Format ===")
	fmt.Println("XG stores positions from active player's perspective:")
	fmt.Println("  Index 0 = opponent's bar")
	fmt.Println("  Index 1-24 = board points (1=active player's 1-point, 24=opponent's 1-point)")
	fmt.Println("  Index 25 = active player's bar")
	fmt.Println("  Positive values = active player's checkers")
	fmt.Println("  Negative values = opponent's checkers")
	fmt.Println()

	if len(match.Games) > 0 && len(match.Games[0].Moves) > 0 {
		// Get first checker move
		for _, move := range match.Games[0].Moves {
			if move.MoveType == "checker" && move.CheckerMove != nil {
				cm := move.CheckerMove

				fmt.Printf("First checker move (ActivePlayer=%d):\n", cm.ActivePlayer)
				fmt.Printf("  Position.Checkers: %v\n", cm.Position.Checkers)
				fmt.Printf("  Position.Cube: %d, CubePos: %d\n", cm.Position.Cube, cm.Position.CubePos)
				fmt.Printf("  Position.Score: %v\n", cm.Position.Score)
				fmt.Println()

				// Count checkers to verify position integrity
				var activeTotal, opponentTotal int
				for i := 0; i < 26; i++ {
					if cm.Position.Checkers[i] > 0 {
						activeTotal += int(cm.Position.Checkers[i])
					} else if cm.Position.Checkers[i] < 0 {
						opponentTotal += int(-cm.Position.Checkers[i])
					}
				}
				fmt.Printf("  Active player checkers on board: %d\n", activeTotal)
				fmt.Printf("  Opponent checkers on board: %d\n", opponentTotal)

				// Show board visually
				fmt.Println("\n  Board (from active player's view):")
				fmt.Println("  Points 13-24 (opponent's home):")
				fmt.Print("  ")
				for i := 13; i <= 24; i++ {
					fmt.Printf("%3d", cm.Position.Checkers[i])
				}
				fmt.Println()
				fmt.Println("  Points 12-1 (active player's home):")
				fmt.Print("  ")
				for i := 12; i >= 1; i-- {
					fmt.Printf("%3d", cm.Position.Checkers[i])
				}
				fmt.Println()
				fmt.Printf("  Bar: opponent=%d, active=%d\n", cm.Position.Checkers[0], cm.Position.Checkers[25])

				break
			}
		}
	}
}

func TestCompareWithReference(t *testing.T) {
	// Compare imported moves with test.txt reference
	match, err := xgparser.ParseXGFromFile("../testdata/test.xg")
	if err != nil {
		t.Fatalf("Failed to parse XG file: %v", err)
	}

	fmt.Println("\n=== Comparing with test.txt Reference ===")
	fmt.Println("Reference Game 1 first moves:")
	fmt.Println("  1) Kévin Unger  51: 24/23 13/8")
	fmt.Println("  1) Maxence Job  61: 13/7 8/7")
	fmt.Println("  2) Kévin Unger  43: 24/20 23/20")
	fmt.Println("  2) Maxence Job  11: 24/20")
	fmt.Println()

	if len(match.Games) > 0 {
		game := match.Games[0]

		moveNum := 0
		for i, move := range game.Moves {
			if move.MoveType != "checker" || move.CheckerMove == nil {
				continue
			}

			moveNum++
			cm := move.CheckerMove

			// Determine player name
			var playerName string
			if cm.ActivePlayer == 1 {
				playerName = match.Metadata.Player1Name
			} else {
				playerName = match.Metadata.Player2Name
			}

			// Convert move
			moveStr := convertMoveToNotation(cm.PlayedMove)

			fmt.Printf("XG Move %d (index %d): %s %d%d: %s\n",
				moveNum, i, playerName, cm.Dice[0], cm.Dice[1], moveStr)

			if moveNum >= 4 {
				break
			}
		}
	}

	fmt.Println()
	fmt.Println("Analysis:")
	fmt.Println("  - XG uses ActivePlayer=1 for Player1 (Kévin), ActivePlayer=-1 for Player2 (Maxence)")
	fmt.Println("  - The first XG move is 51: 13/8 24/23 for Kévin (should be 24/23 13/8)")
	fmt.Println("  - Sub-moves order may differ but represent the same play")
}
