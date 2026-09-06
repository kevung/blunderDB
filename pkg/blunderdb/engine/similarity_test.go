package engine

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

func simPos(black, white map[int]int, onRoll int) domain.Position {
	var p domain.Position
	for i := range p.Board.Points {
		p.Board.Points[i] = domain.Point{Checkers: 0, Color: domain.None}
	}
	for pt, n := range black {
		p.Board.Points[pt] = domain.Point{Checkers: n, Color: domain.Black}
	}
	for pt, n := range white {
		p.Board.Points[pt] = domain.Point{Checkers: n, Color: domain.White}
	}
	p.PlayerOnRoll = onRoll
	p.Score = [2]int{-1, -1}
	return p
}

// TestSimilarityDistance_IsMeasuredInCheckerPips is the property that makes the
// number readable: it is how much checker movement separates the two
// positions, so a checker advanced by one costs 1 and by six costs 6. L1 would
// have counted both as 2, which is the reason P7 recommends transport.
func TestSimilarityDistance_IsMeasuredInCheckerPips(t *testing.T) {
	// White stands on 17, 19 and 21 so the points Black moves through (12, 7)
	// are free: a fixture where the two sides share a point would silently
	// lose one side's checkers and measure something else.
	base := simPos(map[int]int{13: 5, 8: 5, 6: 5}, map[int]int{17: 5, 19: 5, 21: 5}, domain.Black)
	oneStep := simPos(map[int]int{13: 4, 12: 1, 8: 5, 6: 5}, map[int]int{17: 5, 19: 5, 21: 5}, domain.Black)
	sixSteps := simPos(map[int]int{13: 4, 7: 1, 8: 5, 6: 5}, map[int]int{17: 5, 19: 5, 21: 5}, domain.Black)

	near := SimilarityDistance(BuildSimilarityVector(&base), BuildSimilarityVector(&oneStep))
	far := SimilarityDistance(BuildSimilarityVector(&base), BuildSimilarityVector(&sixSteps))
	if near != 1 {
		t.Errorf("one checker moved one pip: got %d, want 1", near)
	}
	if far != 6 {
		t.Errorf("one checker moved six pips: got %d, want 6", far)
	}
}

// TestSimilarityDistance_IsZeroOnItself and symmetric — the two properties any
// distance must have, and the ones a mirroring bug breaks first.
func TestSimilarityDistance_IsZeroOnItselfAndSymmetric(t *testing.T) {
	a := simPos(map[int]int{13: 5, 8: 5, 6: 5}, map[int]int{17: 5, 19: 5, 21: 5}, domain.Black)
	b := simPos(map[int]int{13: 5, 8: 4, 7: 1, 6: 5}, map[int]int{17: 5, 19: 5, 21: 5}, domain.Black)
	va, vb := BuildSimilarityVector(&a), BuildSimilarityVector(&b)
	if got := SimilarityDistance(va, va); got != 0 {
		t.Errorf("a position is at distance 0 from itself: got %d", got)
	}
	if SimilarityDistance(va, vb) != SimilarityDistance(vb, va) {
		t.Error("the distance must be symmetric")
	}
}

// TestSimilarityVector_IsSeenFromTheSideOnRoll: the same physical position,
// with the other side on roll, is a DIFFERENT position for this purpose — and
// the mirror of one is the other. That is the invariance P7 asks for.
func TestSimilarityVector_IsSeenFromTheSideOnRoll(t *testing.T) {
	black := simPos(map[int]int{13: 5, 8: 5, 6: 5}, map[int]int{17: 5, 19: 5, 21: 5}, domain.Black)
	white := black
	white.PlayerOnRoll = domain.White

	vb, vw := BuildSimilarityVector(&black), BuildSimilarityVector(&white)
	if vb.Mover != vw.Opponent || vb.Opponent != vw.Mover {
		t.Error("swapping the side on roll must swap the two distributions")
	}
	// Fifteen checkers on each side, always: that is what makes the transport
	// distance defined rather than a difference of masses.
	for _, v := range []([26]int){vb.Mover, vb.Opponent, vw.Mover, vw.Opponent} {
		if got := sumFrom(&v, 0); got != 15 {
			t.Errorf("each distribution carries fifteen checkers: got %d", got)
		}
	}
}
