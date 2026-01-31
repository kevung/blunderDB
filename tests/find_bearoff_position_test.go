package tests

import (
	"fmt"
	"testing"

	"github.com/kevung/xgparser/xgparser"
)

func TestFindBearoffPosition(t *testing.T) {
	match, err := xgparser.ParseXGFromFile("../testdata/test.xg")
	if err != nil {
		t.Fatalf("Failed to parse XG file: %v", err)
	}

	fmt.Println("Looking for bear-off positions with dice 4-2...")

	for gi, game := range match.Games {
		for mi, move := range game.Moves {
			if move.MoveType == "checker" && move.CheckerMove != nil {
				cm := move.CheckerMove

				// Check if dice are 4-2 or 2-4
				if !((cm.Dice[0] == 4 && cm.Dice[1] == 2) || (cm.Dice[0] == 2 && cm.Dice[1] == 4)) {
					continue
				}

				// Check if this is a bear-off position (all checkers on points 1-6)
				pos := cm.Position.Checkers
				isBearoff := true
				for i := 6; i < 24; i++ {
					if pos[i] > 0 {
						isBearoff = false
						break
					}
				}

				if isBearoff && pos[24] == 0 { // No checkers on bar either
					fmt.Printf("\nGame %d, Move %d - Bear-off position with dice 4-2:\n", gi+1, mi+1)
					fmt.Printf("  Position (points 1-6): [%d %d %d %d %d %d]\n", pos[0], pos[1], pos[2], pos[3], pos[4], pos[5])
					fmt.Printf("  Dice: %v\n", cm.Dice)
					fmt.Printf("  PlayedMove: %v\n", cm.PlayedMove)

					fmt.Printf("  Analysis:\n")
					for ai, analysis := range cm.Analysis {
						if ai >= 5 {
							break
						}
						fmt.Printf("    %d. %v\n", ai+1, analysis.Move)

						// Check if there's a move from point 6 with destination 2 (should be off)
						for j := 0; j < 8; j += 2 {
							from := analysis.Move[j]
							to := analysis.Move[j+1]
							if from == -1 {
								break
							}
							if from == 6 && to == 2 {
								fmt.Printf("       ^^ 6->2 found - this should probably be 6/off!\n")
							}
						}
					}
				}
			}
		}
	}
}
