package main

import (
	"fmt"
	"testing"

	"github.com/kevung/xgparser/xgparser"
)

func TestDebugCubePositions(t *testing.T) {
	imp := xgparser.NewImport("testdata/test.xg")
	segments, err := imp.GetFileSegments()
	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	match, err := xgparser.ParseXG(segments)
	if err != nil {
		t.Fatalf("Error parsing: %v", err)
	}

	// Look at Game 2 moves around the cube action
	fmt.Println("=== Game 2 positions around cube action ===")
	if len(match.Games) >= 2 {
		game := match.Games[1] // Game 2 (0-indexed)
		for i, move := range game.Moves {
			if i >= 18 && i <= 30 {
				if move.MoveType == "checker" && move.CheckerMove != nil {
					cm := move.CheckerMove
					fmt.Printf("Move %d (checker): ActivePlayer=%d, Dice=%v, CubePos=%d, Cube=%d\n",
						i, cm.ActivePlayer, cm.Dice, cm.Position.CubePos, cm.Position.Cube)
				} else if move.MoveType == "cube" && move.CubeMove != nil {
					cm := move.CubeMove
					fmt.Printf("Move %d (cube): ActivePlayer=%d, CubeAction=%d, CubePos=%d, Cube=%d\n",
						i, cm.ActivePlayer, cm.CubeAction, cm.Position.CubePos, cm.Position.Cube)
				}
			}
		}
	}
}
