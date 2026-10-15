package domain

import (
	"strconv"
	"strings"
)

// Reading a play's notation as the board it leaves.
//
// Two engines may spell one play differently ("13/7" or "13/11/7", "6/4(2)"
// or "6/4 6/4", with or without hit markers), and two plays of the same shape
// may leave different boards: with 4-1, "6/5 5/1" and "6/2* 2/1" move the same
// checker to the same point, the second hitting on 2. A notation is therefore
// never compared with another as text: each one is replayed on the position,
// following the points it writes, and two notations are the same play when
// they leave the same board.
//
// A notation that skips a point ("6/1" for a checker playing both dice) may go
// either way; hit markers then tell the ways apart for a dialect that writes
// them. A dialect that never writes them (gnubg's SGF candidates) cannot: its
// missing "*" means "maybe a hit", and a notation left with two boards is
// ambiguous — refused by the callers, never guessed.

// PlayBoards replays notation on p and returns every distinct position it may
// leave, following the points it writes and the dice of p. marksHits says
// whether the notation's dialect writes its hits: then the points it stars are
// exactly the points the play hits on; otherwise a starred point must be hit
// and an unstarred one may be. Nil when the notation cannot be read or names
// no play the dice allow.
func PlayBoards(p *Position, notation string, marksHits bool) []Position {
	d1, d2 := p.Dice[0], p.Dice[1]
	if d1 < 1 || d1 > 6 || d2 < 1 || d2 > 6 {
		return nil
	}
	mover := p.PlayerOnRoll
	segs, starred, ok := parsePlay(notation, mover)
	if !ok {
		return nil
	}
	dice := []int{d1, d2}
	if d1 == d2 {
		dice = []int{d1, d1, d1, d1}
	}

	var out []Position
	seen := map[string]bool{}
	record := func(pos Position, hits uint32) {
		if starred&^hits != 0 || (marksHits && hits != starred) {
			return
		}
		if key := boardKey(&pos); !seen[key] {
			seen[key] = true
			out = append(out, pos)
		}
	}

	var solve func(pos Position, dice []int, segs []playSegment, hits uint32)
	var advance func(pos Position, dice []int, at, target int, rest []playSegment, hits uint32)
	solve = func(pos Position, dice []int, segs []playSegment, hits uint32) {
		if len(segs) == 0 {
			record(pos, hits)
			return
		}
		// Segments are tried in every order: "2/1 6/2" is the same play as
		// "6/2 2/1" when the checker on 2 is the one that just arrived.
		for i, s := range segs {
			if i > 0 && segs[i-1] == s {
				continue
			}
			rest := append(append([]playSegment{}, segs[:i]...), segs[i+1:]...)
			advance(pos, dice, s.from, s.to, rest, hits)
		}
	}
	advance = func(pos Position, dice []int, at, target int, rest []playSegment, hits uint32) {
		tried := map[int]bool{}
		for i, die := range dice {
			if tried[die] {
				continue
			}
			tried[die] = true
			left := append(append([]int{}, dice[:i]...), dice[i+1:]...)
			for _, st := range singleMoves(&pos, mover, die) {
				if st.From != at {
					continue
				}
				np := applyStep(&pos, mover, st)
				h := hits
				if st.Hit {
					h |= 1 << uint(st.To)
				}
				switch {
				case st.To == target:
					solve(np, left, rest, h)
				case st.To != Off && progress(mover, st.To) > progress(mover, target):
					// A point the notation skips: the checker goes on.
					advance(np, left, st.To, target, rest, h)
				}
			}
		}
	}
	start := *p
	start.Board = emptyNone(start.Board)
	solve(start, dice, segs, 0)
	return out
}

// SameBoard reports whether two boards hold the same checkers: an empty
// point's colour, which stored positions do not keep alike, is not compared.
func SameBoard(a, b Board) bool { return emptyNone(a) == emptyNone(b) }

// emptyNone gives every empty point the colour None.
func emptyNone(b Board) Board {
	for i := range b.Points {
		if b.Points[i].Checkers == 0 {
			b.Points[i].Color = None
		}
	}
	return b
}

// playSegment is one written hop, as board indices: From a point or the bar,
// To a point or Off.
type playSegment struct{ from, to int }

// progress is how far a board index is from bearing off, for mover: 25 on the
// bar, 0 off.
func progress(mover, idx int) int {
	switch {
	case idx == Off:
		return 0
	case idx == barIndex(mover):
		return 25
	case mover == White:
		return 25 - idx
	}
	return idx
}

// parsePlay reads a notation in mover-relative numbering ("bar/22* 13/11",
// "8/5*/4", "6/4(2)", "6/off") into its hops and the set of points it stars,
// as a bit per board index.
func parsePlay(notation string, mover int) (segs []playSegment, starred uint32, ok bool) {
	tokens := strings.Fields(notation)
	if len(tokens) == 0 {
		return nil, 0, false
	}
	for _, tok := range tokens {
		count := 1
		if i := strings.IndexByte(tok, '('); i >= 0 && strings.HasSuffix(tok, ")") {
			n, err := strconv.Atoi(tok[i+1 : len(tok)-1])
			if err != nil || n < 1 || n > 4 {
				return nil, 0, false
			}
			count, tok = n, tok[:i]
		}
		labels := strings.Split(tok, "/")
		if len(labels) < 2 {
			return nil, 0, false
		}
		points := make([]int, len(labels))
		for j, l := range labels {
			star := strings.HasSuffix(l, "*")
			idx, ok := pointIndex(mover, strings.TrimRight(l, "*"), j == 0)
			if !ok || (star && (j == 0 || idx < 1 || idx > 24)) {
				return nil, 0, false
			}
			if star {
				starred |= 1 << uint(idx)
			}
			points[j] = idx
		}
		for range count {
			for j := 0; j+1 < len(points); j++ {
				segs = append(segs, playSegment{points[j], points[j+1]})
			}
		}
	}
	return segs, starred, true
}

// pointIndex reads one mover-relative label as a board index: "bar" (or 25)
// only where a hop starts, "off" only where it ends.
func pointIndex(mover int, label string, from bool) (int, bool) {
	switch strings.ToLower(label) {
	case "bar":
		return barIndex(mover), from
	case "off":
		return Off, !from
	}
	n, err := strconv.Atoi(label)
	switch {
	case err != nil || n < 1 || n > 25:
		return 0, false
	case n == 25:
		return barIndex(mover), from
	case mover == White:
		return 25 - n, true
	}
	return n, true
}
