package bearoffgen

// Move generation for a pure bearoff position: every chequer is home, none is
// on the bar, and the opponent is not in the way — the two sides of a two-sided
// table never interact, which is what makes the table a product of two
// independent races.
//
// What the sweep needs is not the moves but the *positions they reach*: it
// takes a maximum over them, so two orders of the same dice reaching the same
// arrangement count once. successors returns those arrangements, as indices.
//
// The backgammon rule that has to be right here is "play as many dice as you
// can": with 5-2, a position that can play both must; one that can only play
// the 5 plays the 5 alone, not the 2. The search therefore goes to the deepest
// reachable ply and keeps only what is found there.

// canBearOff reports whether the chequer on point `pt` (1-based) may come off
// with a die of `die`: exactly (die == pt), or with a larger die when no
// chequer sits further back.
func canBearOff(board []int, pt, die int) bool {
	if die == pt {
		return true
	}
	if die < pt {
		return false
	}
	for higher := pt + 1; higher <= len(board); higher++ {
		if board[higher-1] > 0 {
			return false
		}
	}
	return true
}

// applyDie returns the arrangements reachable from board by playing one die,
// appending them to dst. A die that can be played at all must be.
func applyDie(dst [][]int, board []int, die int, points int) [][]int {
	for pt := points; pt >= 1; pt-- {
		if board[pt-1] == 0 {
			continue
		}
		next := make([]int, points)
		copy(next, board)
		if pt > die {
			// An ordinary move inside the home board.
			next[pt-1]--
			next[pt-die-1]++
			dst = append(dst, next)
			continue
		}
		if canBearOff(board, pt, die) {
			next[pt-1]--
			dst = append(dst, next)
		}
	}
	return dst
}

// successors returns the indices of every arrangement reachable from `board`
// with the roll (d1, d2), playing as many dice as the rules require. The result
// is deduplicated; order is irrelevant since the caller takes a maximum.
//
// The search tracks which dice are still in hand rather than alternating by
// depth: with two distinct dice, "played the 5, now the 2 remains" and "played
// the 2, now the 5 remains" are different states, and collapsing them lets the
// same die be played twice.
func successors(board []int, d1, d2, points, checkers int, seen map[int]struct{}) []int {
	dice := []int{d1, d2}
	if d1 == d2 {
		dice = []int{d1, d1, d1, d1}
	}

	for k := range seen {
		delete(seen, k)
	}
	var out []int
	best := -1 // the deepest ply reached so far, in dice played

	// walk explores one branch: `remaining` is the multiset of dice still in
	// hand, as a slice whose used entries are marked in `used`.
	var used [4]bool
	var walk func(b []int, depth int)
	walk = func(b []int, depth int) {
		moved := false
		for i, die := range dice {
			if used[i] {
				continue
			}
			// With doubles every die is the same; trying the first unused one
			// is enough and keeps the branching factor down.
			if d1 == d2 && i > 0 && !used[i-1] {
				continue
			}
			nexts := applyDie(nil, b, die, points)
			if len(nexts) == 0 {
				continue
			}
			moved = true
			used[i] = true
			for _, n := range nexts {
				walk(n, depth+1)
			}
			used[i] = false
		}
		if moved {
			return
		}
		// A leaf: no die in hand can be played. Only the deepest leaves are
		// legal plays — "use as many dice as you can".
		switch {
		case depth > best:
			best = depth
			out = out[:0]
			for k := range seen {
				delete(seen, k)
			}
		case depth < best:
			return
		}
		idx := PositionIndex(b, points, checkers)
		if _, dup := seen[idx]; dup {
			return
		}
		seen[idx] = struct{}{}
		out = append(out, idx)
	}
	walk(board, 0)
	return out
}
