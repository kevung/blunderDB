package reach

import (
	"fmt"
)

// MaxOrder is the largest distance a search is served for. Beyond it the
// exact relation costs more than a search over a library of 100 000
// positions can wait, and a search asking for more is refused rather than
// answered approximately.
const MaxOrder = 2

// DefaultOrder is the distance a search reaches when it names none.
const DefaultOrder = 2

// ErrOrder is returned for an order the search does not serve.
type ErrOrder struct{ Asked int }

func (e ErrOrder) Error() string {
	return fmt.Sprintf("a search within %d plays is not served: at most %d", e.Asked, MaxOrder)
}

// CheckOrder validates the order a search asks for.
func CheckOrder(n int) error {
	if n < 0 || n > MaxOrder {
		return ErrOrder{Asked: n}
	}
	return nil
}

// Target answers, for one structure, how many plays separate it from others,
// up to a bound. It keeps the plays from the target itself, which every
// candidate of a search would otherwise expand again.
type Target struct {
	g     goal
	max   int
	first [2][]state // successors of the target, each side playing first
}

// goal is a structure some search must end on, its pip counts read once.
type goal struct {
	s      Structure
	pa, pb int
}

func newGoal(s Structure) goal { return goal{s: s, pa: s.A.pips(), pb: s.B.pips()} }

// NewTarget prepares the distances to s up to max plays.
func NewTarget(s Structure, max int) *Target {
	t := &Target{g: newGoal(s), max: max}
	if max > 1 {
		t.first[0] = successors(state{s.A, s.B}, nil)
		t.first[1] = successors(state{s.B, s.A}, nil)
	}
	return t
}

// Distance returns the fewest plays separating the target from c, and false
// when more than the bound.
func (t *Target) Distance(c Structure) (int, bool) {
	if c == t.g.s {
		return 0, true
	}
	cg := newGoal(c)
	for k := 1; k <= t.max; k++ {
		if t.reachesFromTarget(&cg, k) || reaches(c, &t.g, k) {
			return k, true
		}
	}
	return 0, false
}

// reachesFromTarget is reaches(target, c, k), its first play read from the
// cache.
func (t *Target) reachesFromTarget(c *goal, k int) bool {
	s := t.g.s
	for i, st := range [2]state{{s.A, s.B}, {s.B, s.A}} {
		if k == 1 {
			if dfs(st, c, 1) {
				return true
			}
			continue
		}
		if canReach(&st, c, k) == 0 {
			continue
		}
		for _, n := range t.first[i] {
			if dfs(n, c, k-1) {
				return true
			}
		}
	}
	return false
}

// reaches reports whether exactly k plays, starting with either side of
// from, lead to g.
func reaches(from Structure, g *goal, k int) bool {
	return dfs(state{from.A, from.B}, g, k) || dfs(state{from.B, from.A}, g, k)
}

func dfs(s state, g *goal, r int) bool {
	if r == 0 {
		return (s.P == g.s.A && s.Q == g.s.B) || (s.P == g.s.B && s.Q == g.s.A)
	}
	fits := canReach(&s, g, r)
	if fits == 0 {
		return false
	}
	for _, n := range successors(s, rollsToward(&s, g, r, fits)) {
		if dfs(n, g, r-1) {
			return true
		}
	}
	return false
}

// Which side of the goal P can still become: A, B, or both.
const (
	fitsA = 1 << iota
	fitsB
)

// canReach is a necessary condition for s to reach g in exactly r plays, P
// playing first: it prunes, it never decides. It reports which sides of g P
// may end as; zero means neither.
func canReach(s *state, g *goal, r int) int {
	mP, mQ := (r+1)/2, r/2
	// P's single play comes before every hit it suffers when r is 2: those
	// hits are still on the bar at the end.
	pAfter := r == 2
	fits := 0
	// The side with fewer plays is checked first: with none, it is an exact
	// test, and the cheapest.
	if sideCanReach(&s.Q, &g.s.B, g.pb, mQ, mP, false) && sideCanReach(&s.P, &g.s.A, g.pa, mP, mQ, pAfter) {
		fits |= fitsA
	}
	if sideCanReach(&s.Q, &g.s.A, g.pa, mQ, mP, false) && sideCanReach(&s.P, &g.s.B, g.pb, mP, mQ, pAfter) {
		fits |= fitsB
	}
	return fits
}

// rollsToward admits the rolls P's next play can be made of. Within the last
// two plays P is never hit before it plays again, so the pips its play moves
// are bounded by its pips now and at the end: exactly their difference when
// nothing hits it afterwards, and up to 24 more per checker on the bar in g
// otherwise — a checker there may also never have entered. A play bearing a
// checker off from below the die moves fewer pips than the die, so then only
// the lower bound holds.
func rollsToward(s *state, g *goal, r int, fits int) func(d1, d2 int) bool {
	if r > 2 {
		return nil
	}
	type window struct {
		lo, hi int
		open   bool
	}
	var ws [2]window
	n := 0
	pp := s.P.pips()
	for i, side := range [2]*Side{&g.s.A, &g.s.B} {
		if fits&(1<<i) == 0 {
			continue
		}
		pg := g.pa
		if i == 1 {
			pg = g.pb
		}
		hits := 0
		if r == 2 {
			hits = int(side[25])
		}
		ws[n] = window{lo: pp - pg, hi: pp - pg + 24*hits, open: side[0] > s.P[0]}
		n++
	}
	return func(d1, d2 int) bool {
		uses := [5]int{0, d1, d2, d1 + d2, 0}
		k := 4
		if d1 == d2 {
			uses = [5]int{0, d1, 2 * d1, 3 * d1, 4 * d1}
			k = 5
		}
		for _, w := range ws[:n] {
			for _, u := range uses[:k] {
				if u >= w.lo && (w.open || u <= w.hi) {
					return true
				}
			}
		}
		return false
	}
}

// sideCanReach says whether one side can turn from s into t, whose pips are
// pt, when it plays m times and its opponent h times. A side's checkers only
// move forward, except those the opponent hits, at most four per play, each
// sent at most 24 pips back to the bar. hitsLast says every hit comes after
// the side's last play, so each hit checker is still on the bar in t.
func sideCanReach(s, t *Side, pt, m, h int, hitsLast bool) bool {
	hits := 4 * h
	if hits > 15 {
		hits = 15
	}
	if hitsLast && int(t[25]) < hits {
		hits = int(t[25])
	}
	if t[0] < s[0] || int(t[0]-s[0]) > 4*m {
		return false
	}
	if m == 0 {
		moved := 0
		for p := 1; p <= 24; p++ {
			if t[p] > s[p] {
				return false
			}
			moved += int(s[p] - t[p])
		}
		return int(t[25]-s[25]) == moved && moved <= hits
	}
	ps := s.pips()
	if pt < ps-24*m || pt > ps+24*hits {
		return false
	}
	// A play moves at most four checkers and a hit one more: every other
	// checker stands where it stood.
	left := 0
	for p := 1; p <= 25; p++ {
		if s[p] > t[p] {
			left += int(s[p] - t[p])
		}
	}
	if left > 4*m+hits {
		return false
	}
	// Moving forward only adds to the count of checkers at or below a point;
	// a hit takes one away.
	cs, ct := 0, 0
	for k := 0; k <= 24; k++ {
		cs += int(s[k])
		ct += int(t[k])
		if ct < cs-hits {
			return false
		}
	}
	return true
}
