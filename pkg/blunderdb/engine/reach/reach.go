// Package reach measures how far apart two checker structures stand in PLAYS:
// the fewest legal rolls, played one after the other by alternating sides,
// that turn one into the other (ADR-0112).
//
// Only the checkers count. Cube, score, dice shown and kind of decision are
// not part of a structure, and neither is the side on roll: a structure is the
// unordered pair of the two sides' checker layouts, which is also what a
// stored position keeps once it is normalised to the side on roll. A sequence
// of plays may start with either side, and runs in either direction (A to B or
// B to A), so the distance is symmetric.
//
// Legality is the one domain.LegalMoves implements — entering from the bar
// first, playing as many dice as possible, the larger die when only one can be
// played, bearing off exactly or from the highest point, hitting blots — and a
// roll that cannot be played still counts as a play. This package carries its
// own generator because domain.LegalMoves builds notations and steps the
// distance never reads; the tests hold the two to the same answers.
package reach

import "github.com/kevung/blunderdb/pkg/blunderdb/domain"

// Side is one player's fifteen checkers in that player's own numbering: index
// 0 holds those borne off, 1..24 the points (1 is the player's ace point),
// 25 the bar.
type Side [26]int8

// state is a structure with a side to move: P plays next, Q waits.
type state struct{ P, Q Side }

// Structure is the unordered pair of sides, stored in a canonical order so
// that equal structures compare equal whichever side was on roll.
type Structure struct{ A, B Side }

// NewStructure is the structure of two sides, whichever is given first.
func NewStructure(x, y Side) Structure {
	if less(y, x) {
		x, y = y, x
	}
	return Structure{A: x, B: y}
}

func less(a, b Side) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

// Of reads the checker structure of a position. A checker missing from the
// board counts as borne off, so a drawn board with fewer than fifteen
// checkers on a side is read as that side having borne the rest off.
func Of(p *domain.Position) Structure {
	var black, white Side
	for i := 1; i <= domain.NumPoints; i++ {
		pt := p.Board.Points[i]
		if pt.Checkers <= 0 {
			continue
		}
		switch pt.Color {
		case domain.Black:
			black[i] += int8(pt.Checkers)
		case domain.White:
			white[domain.NumPoints+1-i] += int8(pt.Checkers)
		}
	}
	black[25] = int8(p.Board.Points[domain.BlackBar].Checkers)
	white[25] = int8(p.Board.Points[domain.WhiteBar].Checkers)
	fillOff(&black)
	fillOff(&white)
	return NewStructure(black, white)
}

func fillOff(s *Side) {
	n := 0
	for i := 1; i < len(s); i++ {
		n += int(s[i])
	}
	if n < 15 {
		s[0] = int8(15 - n)
	}
}

func (s *Side) pips() int {
	n := 0
	for i := 1; i < len(s); i++ {
		n += i * int(s[i])
	}
	return n
}

// singles calls emit with every (mover, opponent) pair reached by moving one
// of the mover's checkers d pips, and reports whether there was any.
func singles(me, opp *Side, d int, emit func(Side, Side)) bool {
	any := false
	allHome := me[25] == 0
	highest := 0
	for p := 24; p >= 1; p-- {
		if me[p] > 0 {
			if highest == 0 {
				highest = p
			}
			if p > 6 {
				allHome = false
			}
		}
	}
	from, to := 1, 24
	if me[25] > 0 {
		from, to = 25, 25
	}
	for p := from; p <= to; p++ {
		if me[p] == 0 {
			continue
		}
		dst := p - d
		if dst >= 1 {
			o := 25 - dst
			if opp[o] >= 2 {
				continue
			}
			m, op := *me, *opp
			m[p]--
			m[dst]++
			if op[o] == 1 {
				op[o] = 0
				op[25]++
			}
			emit(m, op)
			any = true
			continue
		}
		if !allHome || (dst < 0 && p != highest) {
			continue
		}
		m := *me
		m[p]--
		m[0]++
		emit(m, *opp)
		any = true
	}
	return any
}

// level plays die d from every pair of from, deduplicated.
func level(from []state, d int) []state {
	var out []state
	for i := range from {
		singles(&from[i].P, &from[i].Q, d, func(m, o Side) {
			s := state{m, o}
			// A level holds a few dozen pairs: a scan beats hashing them.
			for j := range out {
				if out[j] == s {
					return
				}
			}
			out = append(out, s)
		})
	}
	return out
}

// plays returns the (mover, opponent) pairs a legal play of d1-d2 can leave,
// the mover still first. An unplayable roll leaves the position unchanged.
func plays(s state, d1, d2 int) []state {
	start := []state{s}
	if d1 == d2 {
		cur := start
		for k := 0; k < 4; k++ {
			next := level(cur, d1)
			if len(next) == 0 {
				break
			}
			cur = next
		}
		return cur
	}
	hi, lo := d1, d2
	if lo > hi {
		hi, lo = lo, hi
	}
	hi1, lo1 := level(start, hi), level(start, lo)
	both := append(level(hi1, lo), level(lo1, hi)...)
	switch {
	case len(both) > 0:
		return dedup(both)
	case len(hi1) > 0:
		return hi1
	case len(lo1) > 0:
		return lo1
	}
	return start
}

func dedup(in []state) []state {
	out := in[:0]
next:
	for _, s := range in {
		for j := range out {
			if out[j] == s {
				continue next
			}
		}
		out = append(out, s)
	}
	return out
}

// successors returns every state one play of P can lead to, over the rolls
// admit lets through (all 21 when nil), with the turn passed: the returned
// states have the old Q to move.
func successors(s state, admit func(d1, d2 int) bool) []state {
	seen := make(map[state]struct{}, 64)
	var out []state
	for d1 := 1; d1 <= 6; d1++ {
		for d2 := d1; d2 <= 6; d2++ {
			if admit != nil && !admit(d1, d2) {
				continue
			}
			for _, r := range plays(s, d1, d2) {
				n := state{P: r.Q, Q: r.P}
				if _, ok := seen[n]; !ok {
					seen[n] = struct{}{}
					out = append(out, n)
				}
			}
		}
	}
	return out
}

// Successors lists the structures one legal play, by either side, can lead
// to from s. Exposed for the tests and for measurement.
func Successors(s Structure) []Structure {
	seen := map[Structure]struct{}{}
	var out []Structure
	for _, st := range []state{{s.A, s.B}, {s.B, s.A}} {
		for _, n := range successors(st, nil) {
			k := NewStructure(n.P, n.Q)
			if _, ok := seen[k]; !ok {
				seen[k] = struct{}{}
				out = append(out, k)
			}
		}
	}
	return out
}
