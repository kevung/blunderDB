// SPDX-License-Identifier: MIT

package gammonnet

// Legal play generation over the evaluator's own representation.
//
// # Why this exists next to domain.LegalMoves
//
// This repository already has a legal-move generator, and it stays the
// canonical one for everything outside the search (it is what
// /v1/positions.legalMoves serves). It is not usable here: it returns a
// []LegalPlay whose every entry carries a Steps slice, a full domain.Position
// and a Notation STRING, and it deduplicates through a string board key. Two
// string allocations per candidate, in a search that generates on the order of
// five thousand plays per decision, is a factor of a hundred.
//
// So the search gets its own, writing into a caller-supplied buffer over the
// narrow representation. Two implementations are acceptable; two answers are
// not, which is why a differential test requires both to produce the SAME SET
// of resulting positions over a corpus × all 21 rolls.
//
// The rules enforced here are the full ones: enter from the bar before
// anything else, play as many dice as possible, play the larger die when only
// one can be played, bear off exactly or from the highest occupied point, and
// hit blots.

const (
	// MaxMovesPerPlay is four, for doubles.
	MaxMovesPerPlay = 4

	// Bar and Off are the sentinels used in Move.From and Move.To.
	Bar = -1
	Off = -2

	// MaxPlays bounds a legal-play buffer. The largest count observed upstream
	// over its corpus is far below this; reaching it means something is wrong,
	// and generation refuses rather than truncating.
	MaxPlays = 2048
)

// Move is one die's worth of movement.
type Move struct {
	From int8 // point index 0..23, or Bar
	To   int8 // point index 0..23, or Off
}

// Play is one complete legal play. Result already has the turn switched to the
// opponent — which is why the value of a play, to the player who made it, is
// the NEGATION of the network's answer on Result.
type Play struct {
	Moves    [MaxMovesPerPlay]Move
	NumMoves int
	Result   Position
}

// home returns the index range of a player's home board. White bears off
// towards index 0, so its home is 0..5; Black bears off towards 23, so its home
// is 18..23.
func homeLow(player uint8) int {
	if player == White {
		return 0
	}
	return 18
}

// step is where a checker of `player` lands from `from` using `die`. It returns
// Off when the checker bears off exactly or past the edge.
func step(player uint8, from, die int) int {
	if player == White {
		return from - die // White travels towards index 0
	}
	return from + die // Black travels towards index 23
}

// entry is the point a checker enters on from the bar.
func entry(player uint8, die int) int {
	if player == White {
		return NumPoints - die // die 1 enters on index 23
	}
	return die - 1 // die 1 enters on index 0
}

// occupancy returns the signed count on a point from `player`'s point of view:
// positive for its own checkers, negative for the opponent's.
func mine(p *Position, player uint8, idx int) int {
	n := int(p.Points[idx])
	if player == Black {
		n = -n
	}
	return n
}

// canLand reports whether `player` may land on idx: empty, its own, or a single
// opposing blot.
func canLand(p *Position, player uint8, idx int) bool {
	return mine(p, player, idx) >= -1
}

// allHome reports whether every checker of `player` is in its home board, which
// is what permits bearing off. A checker on the bar forbids it.
func allHome(p *Position, player uint8) bool {
	if p.Bar[player] > 0 {
		return false
	}
	low := homeLow(player)
	for i := 0; i < NumPoints; i++ {
		if i >= low && i < low+6 {
			continue
		}
		if mine(p, player, i) > 0 {
			return false
		}
	}
	return true
}

// highest returns the index of the player's checker furthest from bearing off,
// or -1 when it has none on the board.
func highest(p *Position, player uint8) int {
	if player == White {
		for i := NumPoints - 1; i >= 0; i-- {
			if mine(p, player, i) > 0 {
				return i
			}
		}
		return -1
	}
	for i := 0; i < NumPoints; i++ {
		if mine(p, player, i) > 0 {
			return i
		}
	}
	return -1
}

// distanceOff is how many pips a checker on idx needs to bear off exactly.
func distanceOff(player uint8, idx int) int {
	if player == White {
		return idx + 1
	}
	return NumPoints - idx
}

// apply plays one sub-move on a copy of p and returns it. The caller has
// already established that the move is legal.
func apply(p *Position, player uint8, m Move) Position {
	out := *p
	sign := int8(1)
	if player == Black {
		sign = -1
	}

	if m.From == Bar {
		out.Bar[player]--
	} else {
		out.Points[m.From] -= sign
	}

	if m.To == Off {
		out.Off[player]++
		return out
	}

	dest := int(m.To)
	if mine(p, player, dest) == -1 { // hit a blot
		out.Points[dest] = 0
		opp := White
		if player == White {
			opp = Black
		}
		out.Bar[opp]++
	}
	out.Points[dest] += sign
	return out
}

// subMoves writes every legal single move for `player` with `die` into out and
// returns how many. Entering from the bar is compulsory and exclusive.
func subMoves(p *Position, player uint8, die int, out *[NumPoints + 1]Move) int {
	n := 0
	if p.Bar[player] > 0 {
		dest := entry(player, die)
		if canLand(p, player, dest) {
			out[0] = Move{From: Bar, To: int8(dest)}
			n = 1
		}
		return n
	}

	bearing := allHome(p, player)
	far := -1
	if bearing {
		far = highest(p, player)
	}

	for i := 0; i < NumPoints; i++ {
		if mine(p, player, i) <= 0 {
			continue
		}
		dest := step(player, i, die)
		if dest >= 0 && dest < NumPoints {
			if canLand(p, player, dest) {
				out[n] = Move{From: int8(i), To: int8(dest)}
				n++
			}
			continue
		}
		// The checker runs off the edge: only legal while bearing off.
		if !bearing {
			continue
		}
		need := distanceOff(player, i)
		if need == die || (need < die && i == far) {
			// Exact, or over-bearing off from the highest occupied point.
			out[n] = Move{From: int8(i), To: Off}
			n++
		}
	}
	return n
}
