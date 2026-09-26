package ingest

import (
	"testing"

	"github.com/kevung/xgparser/xgparser"
)

// TestXGPositionCheckerCountInvariant: neither side's checkers on board+bar
// ever exceed 15 in a parsed XG match.
func TestXGPositionCheckerCountInvariant(t *testing.T) {
	match, err := xgparser.ParseXGFromFile(xgFixture())
	if err != nil {
		t.Fatalf("ParseXGFromFile: %v", err)
	}
	if len(match.Games) == 0 {
		t.Fatal("fixture carries no games")
	}

	checked := 0
	for _, game := range match.Games {
		for _, move := range game.Moves {
			if move.MoveType != "checker" || move.CheckerMove == nil {
				continue
			}
			pos := move.CheckerMove.Position.Checkers

			var activeTotal, opponentTotal int
			for i := 0; i < 26; i++ {
				switch {
				case pos[i] > 0:
					activeTotal += int(pos[i])
				case pos[i] < 0:
					opponentTotal += int(-pos[i])
				}
			}
			if activeTotal > 15 {
				t.Errorf("game %d move: active player has %d checkers on board+bar, want <= 15", game.GameNumber, activeTotal)
			}
			if opponentTotal > 15 {
				t.Errorf("game %d move: opponent has %d checkers on board+bar, want <= 15", game.GameNumber, opponentTotal)
			}
			checked++
		}
	}
	if checked == 0 {
		t.Fatal("fixture carries no checker moves to check")
	}
}
