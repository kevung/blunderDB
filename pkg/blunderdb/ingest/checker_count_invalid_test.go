package ingest

import (
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/gnubgparser"
	"github.com/kevung/xgparser/xgparser"
)

// A side with 16 checkers is the file's fault: every mapper must classify it
// as storage.ErrInvalid (a named 4xx for a client), not as an engine fault,
// while keeping the typed cause reachable.
func TestSixteenCheckersIsClassifiedInvalid(t *testing.T) {
	check := func(t *testing.T, err error) {
		t.Helper()
		if !errors.Is(err, storage.ErrInvalid) {
			t.Errorf("got %v, want it to wrap storage.ErrInvalid", err)
		}
		var tooMany *domain.TooManyCheckersError
		if !errors.As(err, &tooMany) {
			t.Errorf("got %v, want the *domain.TooManyCheckersError cause kept", err)
		}
	}

	t.Run("xg", func(t *testing.T) {
		var xgPos xgparser.Position
		xgPos.Checkers[6], xgPos.Checkers[8], xgPos.Checkers[13], xgPos.Checkers[24] = 5, 3, 5, 3
		xgPos.Checkers[19] = -15
		xgPos.Cube = 1
		_, err := createPositionFromXG(xgPos, &xgparser.Game{}, 7, 1, false)
		check(t, err)
	})
	t.Run("gnubg", func(t *testing.T) {
		pos := initStandardGnuBGPosition()
		pos.Board[0][12]++
		_, err := createPositionFromGnuBG(&pos, &gnubgparser.Game{}, 7)
		check(t, err)
	})
	t.Run("bgf", func(t *testing.T) {
		var board [28]int
		board[0], board[23] = 16, -15
		_, err := createPositionFromBGF(board, map[string]interface{}{}, 7, 1, -1, bgfRules{})
		check(t, err)
	})
}
