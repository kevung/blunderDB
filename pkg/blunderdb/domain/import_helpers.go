package domain

import "fmt"

// CheckersPerPlayer is the number of checkers each side owns in standard
// backgammon; what is not on the board (points and bar) has been borne off.
const CheckersPerPlayer = 15

// AwayScores turns the score before a game into the away scores blunderDB
// stores in Position.Score, indexed like the score itself. A match length of
// zero (or less) is money play, which the engine spells [Unlimited, Unlimited].
// Shared by the XG, gnuBG and BGF importers so that one convention exists.
func AwayScores(matchLength, score0, score1 int) [2]int {
	if matchLength <= 0 {
		return [2]int{Unlimited, Unlimited}
	}
	return [2]int{matchLength - score0, matchLength - score1}
}

// AwayScoresWithCrawford is AwayScores with the Crawford sentinel written in,
// and it is what an importer that knows WHICH game it is reading must call.
//
// The away score carries the Crawford rule INSIDE the number (CONTEXT.md,
// « Away score »): `1` means "one point to go, and this IS the Crawford game",
// `0` means "one point to go, and Crawford is behind us". AwayScores says `1`
// for both, which every sentinel reader would take as a cube-dead Crawford
// game. So outside the Crawford game a raw away of 1 becomes 0; money play and
// aways of 2 or more are left alone. A 1-point match is its own Crawford game,
// and [1, 1] is the cube-dead reading it wants.
func AwayScoresWithCrawford(matchLength, score0, score1 int, isCrawfordGame bool) [2]int {
	away := AwayScores(matchLength, score0, score1)
	if isCrawfordGame {
		return away
	}
	for i, a := range away {
		if a == Crawford {
			away[i] = PostCrawford
		}
	}
	return away
}

// PointsAway decodes an away score into the DISTANCE to victory it means, and
// is the reading every consumer of a stored score owes CONTEXT.md's « Away
// score » entry: the sentinels 0 and 1 are the SAME distance, one point. Code
// that subtracts a stored 0 from the match length reads "has already won", and
// a match-equity lookup then silently drops the row.
//
// Money play (Unlimited) has no distance and is returned unchanged, for the
// caller to recognise as it already must.
func PointsAway(away int) int {
	if away == PostCrawford {
		return Crawford
	}
	return away
}

// CubeExponent converts a cube value as the match files write it (1, 2, 4, …)
// into the exponent Cube.Value stores (0, 1, 2, …). A value of 0 or 1 — an
// unset or centred cube — is exponent 0. Values that are not a power of two
// round down: 3 is treated as 2.
func CubeExponent(cubeValue int) int {
	exp := 0
	for v := cubeValue; v > 1; v >>= 1 {
		exp++
	}
	return exp
}

// TooManyCheckersError reports a colour with more than CheckersPerPlayer
// checkers on the board, which no legal position can hold. Colour is Black or
// White; OnBoard counts points and bar.
type TooManyCheckersError struct {
	Color   int
	OnBoard int
}

func (e *TooManyCheckersError) Error() string {
	return fmt.Sprintf("player %d has %d checkers on the board (%d expected)", e.Color+1, e.OnBoard, CheckersPerPlayer)
}

// RecomputeBearoff sets Bearoff from what is on the board: each colour has
// CheckersPerPlayer checkers, and those not on a point or the bar are off. It
// is the one place the match importers derive the borne-off count; a corrupt
// file with 16 checkers gets an error naming the colour, Bearoff untouched,
// rather than a −1 travelling into the Zobrist hash.
//
// Only Black and White are counted; ExcludeEmpty markers of a search
// structure are not checkers.
func (b *Board) RecomputeBearoff() error {
	var onBoard [2]int
	for _, pt := range b.Points {
		if pt.Color == Black || pt.Color == White {
			onBoard[pt.Color] += pt.Checkers
		}
	}
	for color, n := range onBoard {
		if n > CheckersPerPlayer {
			return &TooManyCheckersError{Color: color, OnBoard: n}
		}
	}
	b.Bearoff = [2]int{CheckersPerPlayer - onBoard[Black], CheckersPerPlayer - onBoard[White]}
	return nil
}
