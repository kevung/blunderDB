// SPDX-License-Identifier: MIT

package gammonnet

// maxLevel bounds the number of DISTINCT positions reachable after k sub-moves.
// Deduplicating level by level rather than at the end is what keeps this small:
// a double can be played in tens of thousands of orders, but they land on a few
// hundred distinct positions at most.
const maxLevel = 1024

type levelEntry struct {
	pos   Position
	moves [MaxMovesPerPlay]Move
	n     int
}

// Generator holds the scratch legal-play generation needs. Allocate one per
// goroutine and reuse it; generating then allocates nothing.
//
// The two working levels sit in one array and generation ALTERNATES between
// them. It used to hold them in two fields and swap them by value — and a
// levelEntry is 48 octets, maxLevel is 1024, so that assignment moved 147 Ko
// per die played, four times for a double, at every node of the search. That
// single copy was the whole of the generator's cost: generating the sixteen
// legal plays of the opening 3-1 took 17 µs before this and 1,5 µs after.
type Generator struct {
	levels [2][maxLevel]levelEntry
	side   int // which half of levels is the current one
	sub    [NumPoints + 1]Move
}

// cur is the level being played from, next the one being written. advance
// makes the written level current — what the swap used to do, without moving
// anything. side survives from one call to the next, which is harmless:
// every entry point rewrites cur[0] before reading it.
func (g *Generator) cur() *[maxLevel]levelEntry  { return &g.levels[g.side] }
func (g *Generator) next() *[maxLevel]levelEntry { return &g.levels[g.side^1] }
func (g *Generator) advance()                    { g.side ^= 1 }

// LegalPlays fills out with every distinct legal play for p.Turn given the
// dice, and returns how many there are.
//
// It returns 0 when the player has no legal play — a real and legal outcome, a
// dance, not an error. It returns -1 when the position is invalid, the dice are
// out of range, or generation would exceed its capacity.
//
// A truncated list is NEVER returned. A silently short candidate list is
// indistinguishable from a position that genuinely has fewer options, and would
// make the search quietly blind to moves it never saw.
func (g *Generator) LegalPlays(p *Position, d1, d2 int, out []Play) int {
	if !p.Valid() || d1 < 1 || d1 > 6 || d2 < 1 || d2 > 6 {
		return -1
	}
	player := p.Turn

	if d1 == d2 {
		return g.generate(p, player, []int{d1, d1, d1, d1}, out, 0)
	}
	// Both orders are explored; the "as many dice as possible" rule is applied
	// over their union, and the larger-die rule only when a single die is all
	// that can be played.
	hi, lo := d1, d2
	if lo > hi {
		hi, lo = lo, hi
	}
	return g.generateOrders(p, player, hi, lo, out)
}

// expand plays one die from every entry of cur into next, deduplicating by
// resulting position. It returns the number of distinct entries written, or -1
// on capacity overflow.
func (g *Generator) expand(count, die int, player uint8) int {
	cur, next := g.cur(), g.next()
	n := 0
	for i := 0; i < count; i++ {
		e := &cur[i]
		k := subMoves(&e.pos, player, die, &g.sub)
		for s := 0; s < k; s++ {
			res := apply(&e.pos, player, g.sub[s])
			dup := false
			for j := 0; j < n; j++ {
				if next[j].pos == res {
					dup = true
					break
				}
			}
			if dup {
				continue
			}
			if n == maxLevel {
				return -1
			}
			next[n].pos = res
			next[n].moves = e.moves
			next[n].moves[e.n] = g.sub[s]
			next[n].n = e.n + 1
			n++
		}
	}
	return n
}

// generate plays a fixed sequence of dice as deeply as it can, and returns the
// plays that used the most dice. `usedFloor` is the number of dice a play must
// use to be admissible at all (0 for doubles, set by the caller otherwise).
func (g *Generator) generate(p *Position, player uint8, dice []int, out []Play, usedFloor int) int {
	g.cur()[0] = levelEntry{pos: *p}
	count := 1
	deepest := 0
	deepestCount := 1

	for _, die := range dice {
		n := g.expand(count, die, player)
		if n < 0 {
			return -1
		}
		if n == 0 {
			break
		}
		g.advance()
		count = n
		deepest++
		deepestCount = n
	}
	if deepest < usedFloor {
		return 0
	}
	if deepest == 0 {
		return 0 // a dance
	}
	return g.emit(deepestCount, player, out)
}

// emit writes the current level into out, switching the turn on each result.
func (g *Generator) emit(count int, player uint8, out []Play) int {
	if count > len(out) || count > MaxPlays {
		return -1
	}
	opp := uint8(White)
	if player == White {
		opp = Black
	}
	cur := g.cur()
	for i := 0; i < count; i++ {
		e := &cur[i]
		out[i].Result = e.pos
		out[i].Result.Turn = opp
		out[i].Moves = e.moves
		out[i].NumMoves = e.n
	}
	return count
}

// generateOrders handles a non-double. Both orders are tried; the union is
// filtered by the two rules that make a play compulsory:
//
//   - as many dice as possible must be played, so a two-dice play beats any
//     one-dice play;
//   - when only one die can be played and either would do alone, it must be the
//     larger one.
func (g *Generator) generateOrders(p *Position, player uint8, hi, lo int, out []Play) int {
	// Depth 2 first: whichever order reaches it, those plays are compulsory.
	two := g.twoDice(p, player, hi, lo, out)
	if two != 0 {
		return two
	}

	// Only one die can be played. The larger one wins if it can be played at
	// all; the smaller is only a fallback.
	for _, die := range [2]int{hi, lo} {
		g.cur()[0] = levelEntry{pos: *p}
		n := g.expand(1, die, player)
		if n < 0 {
			return -1
		}
		if n > 0 {
			g.advance()
			return g.emit(n, player, out)
		}
	}
	return 0 // a dance
}

// twoDice collects the plays that use BOTH dice, in either order, deduplicated
// across the two orders. It returns 0 when neither order reaches depth two.
func (g *Generator) twoDice(p *Position, player uint8, hi, lo int, out []Play) int {
	total := 0
	for _, order := range [2][2]int{{hi, lo}, {lo, hi}} {
		g.cur()[0] = levelEntry{pos: *p}
		n := g.expand(1, order[0], player)
		if n < 0 {
			return -1
		}
		if n == 0 {
			continue
		}
		g.advance()
		n = g.expand(n, order[1], player)
		if n < 0 {
			return -1
		}
		if n == 0 {
			continue
		}
		g.advance()

		// Merge into out, deduplicating against what the other order produced.
		opp := uint8(White)
		if player == White {
			opp = Black
		}
		cur := g.cur()
		for i := 0; i < n; i++ {
			res := cur[i].pos
			res.Turn = opp
			dup := false
			for j := 0; j < total; j++ {
				if out[j].Result == res {
					dup = true
					break
				}
			}
			if dup {
				continue
			}
			if total == len(out) || total == MaxPlays {
				return -1
			}
			out[total].Result = res
			out[total].Moves = cur[i].moves
			out[total].NumMoves = cur[i].n
			total++
		}
	}
	return total
}
