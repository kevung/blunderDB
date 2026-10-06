// SPDX-License-Identifier: MIT

package gammonnet

import (
	"math/rand"
	"testing"
)

// The order of LegalPlays is part of the port, not a convenience: the search
// sorts candidates stably, so plays of equal value (two bear-offs both worth
// −0, say) come out in generation order and the gold files record that order.
// These tests pin it without the network.

// dfsOracle is a line-for-line port of upstream's play generation:
// gn_legal_plays sorts the dice ascending, then bg_engine.c's generate_plays
// walks the dice depth first, emitting every leaf — dead ends included — and
// deduplicating at emission. A later leaf reaching a known position with more
// dice takes over that slot's moves, not its place. Then the plays using fewer
// than the most dice are dropped, and, when only one die can be played, those
// that cannot be read as the larger die.
func dfsOracle(p *Position, d1, d2 int) []Play {
	if d1 > d2 {
		d1, d2 = d2, d1
	}
	dice := []int{d1, d2}
	if d1 == d2 {
		dice = []int{d1, d1, d1, d1}
	}
	player := p.Turn
	var plays []Play
	maxUsed := 0
	var cur [MaxMovesPerPlay]Move

	var walk func(s Position, remaining []int, depth int)
	walk = func(s Position, remaining []int, depth int) {
		found := false
		var tried [7]bool
		for i, die := range remaining {
			if tried[die] {
				continue
			}
			tried[die] = true
			rest := make([]int, 0, len(remaining)-1)
			rest = append(rest, remaining[:i]...)
			rest = append(rest, remaining[i+1:]...)
			var sub [NumPoints + 1]Move
			k := subMoves(&s, player, die, &sub)
			for m := 0; m < k; m++ {
				found = true
				cur[depth] = sub[m]
				walk(apply(&s, player, sub[m]), rest, depth+1)
			}
		}
		if found {
			return
		}
		if depth > maxUsed {
			maxUsed = depth
		}
		for j := range plays {
			if plays[j].Result == s {
				if depth > plays[j].NumMoves {
					plays[j].Moves, plays[j].NumMoves = cur, depth
				}
				return
			}
		}
		plays = append(plays, Play{Moves: cur, NumMoves: depth, Result: s})
	}
	walk(*p, dice, 0)
	if maxUsed == 0 {
		return nil
	}

	kept := plays[:0]
	for _, pl := range plays {
		if pl.NumMoves == maxUsed {
			kept = append(kept, pl)
		}
	}
	if d1 != d2 && maxUsed == 1 {
		var big []Play
		for _, pl := range kept {
			if usesDie(p, pl.Moves[0], d2) {
				big = append(big, pl)
			}
		}
		if len(big) > 0 {
			kept = big
		}
	}
	opp := uint8(White)
	if player == White {
		opp = Black
	}
	out := make([]Play, len(kept))
	for i, pl := range kept {
		for m := pl.NumMoves; m < MaxMovesPerPlay; m++ {
			pl.Moves[m] = Move{}
		}
		pl.Result.Turn = opp
		out[i] = pl
	}
	return out
}

// usesDie is bg_engine.c's play_uses_die: whether the single move m can be
// read as a move of `die` from p.
func usesDie(p *Position, m Move, die int) bool {
	player := p.Turn
	switch {
	case m.From == Bar:
		return int(m.To) == entry(player, die)
	case m.To == Off:
		need := distanceOff(player, int(m.From))
		return need == die || (need < die && int(m.From) == highest(p, player))
	default:
		return int(m.To) == step(player, int(m.From), die)
	}
}

// homeBoards puts every remaining checker of both sides in its home board.
func homeBoards(rng *rand.Rand, turn uint8) Position {
	var p Position
	p.Turn = turn
	for _, pl := range []uint8{White, Black} {
		left := 1 + rng.Intn(15)
		p.Off[pl] = uint8(15 - left)
		for ; left > 0; left-- {
			k := rng.Intn(6)
			if pl == White {
				p.Points[k]++
			} else {
				p.Points[23-k]--
			}
		}
	}
	return p
}

func samePlays(a []Play, n int, b []Play) bool {
	if n != len(b) {
		return false
	}
	for i := 0; i < n; i++ {
		x, y := a[i], b[i]
		for m := x.NumMoves; m < MaxMovesPerPlay; m++ {
			x.Moves[m] = Move{}
		}
		if x != y {
			return false
		}
	}
	return true
}

// TestLegalPlaysOrderMatchesUpstream: over contact, bar and bear-off boards and
// all 36 ordered rolls, the generator returns upstream's plays, moves and order
// included.
func TestLegalPlaysOrderMatchesUpstream(t *testing.T) {
	rng := rand.New(rand.NewSource(20261006))
	var g Generator
	plays := make([]Play, MaxPlays)

	var boards []Position
	for b := 0; b < 600; b++ {
		dp := randomBoard(rng, b%2)
		gp, err := FromDomain(&dp)
		if err != nil {
			t.Fatal(err)
		}
		boards = append(boards, gp, homeBoards(rng, uint8(b%2)))
	}
	for b := range boards {
		p := &boards[b]
		for d1 := 1; d1 <= 6; d1++ {
			for d2 := 1; d2 <= 6; d2++ {
				n := g.LegalPlays(p, d1, d2, plays)
				want := dfsOracle(p, d1, d2)
				if !samePlays(plays, n, want) {
					t.Fatalf("board %d %+v roll %d-%d: generator order differs from upstream's (%d vs %d plays)",
						b, *p, d1, d2, n, len(want))
				}
			}
		}
	}
}

// TestLegalPlaysOrderOnABearoffTie: White has a checker on its 1 and 5 points
// and rolls 6-2. Upstream plays the 2 first, so 5/3 3/off precedes 5/off 1/off
// although both are worth the same once the game is over — in either dice order.
func TestLegalPlaysOrderOnABearoffTie(t *testing.T) {
	var p Position
	p.Points[0], p.Points[4] = 1, 1
	p.Off[White] = 13
	p.Points[23] = -15
	p.Turn = White

	var g Generator
	plays := make([]Play, MaxPlays)
	for _, dice := range [][2]int{{6, 2}, {2, 6}} {
		n := g.LegalPlays(&p, dice[0], dice[1], plays)
		if n != 2 {
			t.Fatalf("%v: %d plays, want 2", dice, n)
		}
		first := plays[0].Moves[:plays[0].NumMoves]
		if len(first) != 2 || first[0] != (Move{4, 2}) || first[1] != (Move{2, Off}) {
			t.Errorf("%v: first play %v, want 5/3 3/off", dice, first)
		}
		second := plays[1].Moves[:plays[1].NumMoves]
		if len(second) != 2 || second[0] != (Move{4, Off}) || second[1] != (Move{0, Off}) {
			t.Errorf("%v: second play %v, want 5/off 1/off", dice, second)
		}
	}
}
