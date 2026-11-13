// Package reachtest is the oracle the distance in plays is tested against.
//
// It shares nothing with package reach: it plays through domain.LegalMoves,
// the move generator the interface uses, on whole domain.Position values, and
// compares boards by their own text up to colour mirroring. It is slow — a
// breadth-first enumeration of every play of every roll — and meant for a
// handful of pairs.
package reachtest

import (
	"strconv"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// Key names a checker structure: the board, both bars included, compared up
// to swapping the colours, since a stored position is mirrored to its side on
// roll and the side on roll is not part of a structure.
func Key(p *domain.Position) string {
	m := p.Mirror()
	a, b := boardText(p), boardText(&m)
	if b < a {
		return b
	}
	return a
}

func boardText(p *domain.Position) string {
	var sb strings.Builder
	for i := 0; i <= 25; i++ {
		pt := p.Board.Points[i]
		n := pt.Checkers
		switch {
		case n == 0:
		case pt.Color == domain.Black:
			n = -n
		case pt.Color != domain.White:
			n = 0
		}
		sb.WriteString(strconv.Itoa(n))
		sb.WriteByte(',')
	}
	return sb.String()
}

// Distance is the fewest plays, at most max, turning a into b or b into a;
// false when none does.
func Distance(a, b *domain.Position, max int) (int, bool) {
	if Key(a) == Key(b) {
		return 0, true
	}
	for k := 1; k <= max; k++ {
		if Reaches(a, b, k) || Reaches(b, a, k) {
			return k, true
		}
	}
	return 0, false
}

// Reaches reports whether exactly k plays, the first by either side, the
// sides alternating, turn from into to.
func Reaches(from, to *domain.Position, k int) bool {
	want := Key(to)
	for _, mover := range []int{domain.Black, domain.White} {
		q := bare(from)
		q.PlayerOnRoll = mover
		if reaches(&q, to, want, k) {
			return true
		}
	}
	return false
}

func reaches(p, to *domain.Position, want string, k int) bool {
	if k == 0 {
		return Key(p) == want
	}
	// Before the last play, the side that will not play it can only lose
	// checkers to the bar: its checkers on the points of to must already
	// stand on p. Enumerating every play of every roll is the oracle's
	// whole cost, and this skips the hopeless ones.
	if k == 1 && !waitingSideFits(p, to) {
		return false
	}
	for _, r := range playsOf(p) {
		if reaches(&r, to, want, k-1) {
			return true
		}
	}
	return false
}

// waitingSideFits says whether the side not on roll in p could stand as in
// to, in either colouring of to, after being hit and nothing else.
func waitingSideFits(p, to *domain.Position) bool {
	waiting := 1 - p.PlayerOnRoll
	m := to.Mirror()
	for _, t := range []*domain.Position{to, &m} {
		ok := count(t, waiting) == count(p, waiting)
		for i := 1; ok && i <= 24; i++ {
			if checkersOf(t, i, waiting) > checkersOf(p, i, waiting) {
				ok = false
			}
		}
		if ok {
			return true
		}
	}
	return false
}

func checkersOf(p *domain.Position, i, color int) int {
	if pt := p.Board.Points[i]; pt.Color == color {
		return pt.Checkers
	}
	return 0
}

// count is how many of color's checkers are still in play, bar included.
func count(p *domain.Position, color int) int {
	n := 0
	for i := 0; i <= 25; i++ {
		n += checkersOf(p, i, color)
	}
	return n
}

// playsOf is every position one roll of p's side on roll can leave, the turn
// passed to the other side; a roll that cannot be played leaves the board.
func playsOf(p *domain.Position) []domain.Position {
	var out []domain.Position
	for d1 := 1; d1 <= 6; d1++ {
		for d2 := d1; d2 <= 6; d2++ {
			q := *p
			q.Dice = [2]int{d1, d2}
			plays := domain.LegalMoves(&q)
			if len(plays) == 0 {
				r := *p
				r.PlayerOnRoll = 1 - p.PlayerOnRoll
				out = append(out, r)
				continue
			}
			for _, pl := range plays {
				r := bare(&pl.Result)
				r.PlayerOnRoll = 1 - p.PlayerOnRoll
				out = append(out, r)
			}
		}
	}
	return out
}

// bare keeps only the checkers of p.
func bare(p *domain.Position) domain.Position {
	var q domain.Position
	q.Board = p.Board
	q.PlayerOnRoll = p.PlayerOnRoll
	return q
}
