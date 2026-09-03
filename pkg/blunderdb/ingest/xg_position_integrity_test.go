package ingest

import (
	"testing"

	"github.com/kevung/xgparser/xgparser"
)

// Moved from the legacy tests/debug_xg_test.go (root "tests" package,
// invisible to coverage): that file was a debugging script — fmt.Printf
// dumps of parsed XG data with no assertions at all, so it could never fail.
// The one durable invariant it was exploring — neither side's checkers on
// board+bar ever exceeds the 15 it starts with, since Position.Checkers does
// not track checkers already borne off — is kept here as a real assertion;
// the rest (raw JSON dumps, ad-hoc notation conversion duplicating
// convertXGMoveToString, prints keyed to the real players of the source
// match) added nothing once this test's real assertion was extracted.
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
