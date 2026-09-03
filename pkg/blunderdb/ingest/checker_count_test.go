package ingest

import (
	"errors"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/gnubgparser"
	"github.com/kevung/xgparser/xgparser"
)

// The three match importers used to derive the borne-off count as 15 − (on
// board) with no guard: a corrupt file giving a side 16 checkers produced a
// bearoff of −1 that went through the Zobrist hash and into the database.
// Each createPositionFrom* now refuses it through Board.RecomputeBearoff, and
// the game loops name the game and the move so the user can find the record.

func TestCreatePositionFromXGRefusesSixteenCheckers(t *testing.T) {
	game := &xgparser.Game{InitialScore: [2]int32{0, 0}}
	var xgPos xgparser.Position
	xgPos.Checkers[6] = 5
	xgPos.Checkers[8] = 3
	xgPos.Checkers[13] = 5
	xgPos.Checkers[24] = 3 // 16 for the active player
	xgPos.Checkers[19] = -15
	xgPos.Cube = 1

	_, err := createPositionFromXG(xgPos, game, 7, 1)
	var tooMany *domain.TooManyCheckersError
	if !errors.As(err, &tooMany) {
		t.Fatalf("got %v, want *domain.TooManyCheckersError", err)
	}
	if tooMany.OnBoard != 16 {
		t.Errorf("OnBoard = %d, want 16", tooMany.OnBoard)
	}

	// The same board with 15 is accepted, and the bearoff is derived from it.
	xgPos.Checkers[24] = 2
	pos, err := createPositionFromXG(xgPos, game, 7, 1)
	if err != nil {
		t.Fatalf("15 checkers refused: %v", err)
	}
	if pos.Board.Bearoff != [2]int{0, 0} {
		t.Errorf("Bearoff = %v, want [0 0]", pos.Board.Bearoff)
	}
	if pos.Score != [2]int{7, 7} || pos.Cube.Value != 0 {
		t.Errorf("score/cube: got %v / %d, want [7 7] / 0", pos.Score, pos.Cube.Value)
	}
}

// sixteenCheckerSGFGame is a game whose set-up position gives player 1 one
// checker too many, followed by one checker move.
func sixteenCheckerSGFGame(number int) gnubgparser.Game {
	setup := initStandardGnuBGPosition()
	setup.Board[0][12]++ // 16 checkers for player 0
	return gnubgparser.Game{
		GameNumber: number,
		Moves: []gnubgparser.MoveRecord{
			{Type: gnubgparser.MoveTypeSetBoard, Position: &setup},
			{Type: gnubgparser.MoveTypeNormal, Player: 0, Dice: [2]int{3, 1}, Move: [8]int{7, 4, 5, 4, -1, -1, -1, -1}},
		},
	}
}

func TestGnuBGImportNamesTheGameAndMoveWithSixteenCheckers(t *testing.T) {
	game := sixteenCheckerSGFGame(3)
	_, err := mapGnuBGGameMoves(&game, 7, true)
	if err == nil {
		t.Fatal("a 16-checker position was mapped without error")
	}
	var tooMany *domain.TooManyCheckersError
	if !errors.As(err, &tooMany) || tooMany.Color != domain.Black {
		t.Fatalf("got %v, want *domain.TooManyCheckersError for player 1", err)
	}
	if !strings.HasPrefix(err.Error(), "game 3, move 1: ") {
		t.Errorf("error does not name the game and the move: %q", err)
	}

	// Through the match mapper the file name leads the message, so a batch
	// import says which file to look at.
	match := &gnubgparser.Match{Games: []gnubgparser.Game{game}}
	match.Metadata.MatchLength = 7
	_, err = mapGnuBGMatch(match, true, "/somewhere/corrupt.sgf")
	if err == nil || !strings.HasPrefix(err.Error(), "ingest: corrupt.sgf: game 1, move 1: ") {
		t.Errorf("match-level error = %v, want it to lead with the file, game and move", err)
	}
}

func TestBGFImportNamesTheMoveWithSixteenCheckers(t *testing.T) {
	gameData := map[string]interface{}{"scoreGreen": float64(0), "scoreRed": float64(0)}
	var board [28]int
	board[0] = 16   // Green: 16 checkers on one point
	board[23] = -15 // Red
	moves := []interface{}{
		map[string]interface{}{"type": "amove", "player": float64(1), "from": []interface{}{float64(6), float64(-1), float64(-1), float64(-1)}, "to": []interface{}{float64(3), float64(-1), float64(-1), float64(-1)}},
	}
	// bgfInitBoardFromGame reads the board from the game record; feed the
	// mapper directly with the corrupt state instead.
	_, err := mapBGFCheckerMove(moves[0].(map[string]interface{}), gameData, 7, board, 1, -1, bgfRules{})
	var tooMany *domain.TooManyCheckersError
	if !errors.As(err, &tooMany) || tooMany.OnBoard != 16 {
		t.Fatalf("got %v, want *domain.TooManyCheckersError with 16 on board", err)
	}
}
