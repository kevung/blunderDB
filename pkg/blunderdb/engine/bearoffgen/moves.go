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

// homeBoard is the six points a chequer must be on before ANY chequer may come
// off. It is a rule of the game and has nothing to do with how wide the table
// is: a table over seven points describes positions whose farthest chequer
// stands on the 7-point, and in those positions bearing off is simply illegal
// until that chequer comes home.
const homeBoard = 6

// canBearOff reports whether the chequer on point `pt` (1-based) may come off
// with a die of `die`.
//
// Three conditions, and the first one is the one a six-point table can never
// notice: every chequer must be inside the home board, then the die must match
// the point exactly, or exceed it with no chequer further back.
//
// Leaving the first condition out is correct for every table over six points —
// there is nothing outside the home board to check — and wrong for every wider
// one, where it lets a position bear a chequer off while another still sits on
// the 7-point. That produces a table that is better than the game allows, and
// it diverges from gnubg only from OS-07 on. gnubg gets this for free: it calls
// its ordinary GenerateMoves on a full board.
func canBearOff(board []int, pt, die int) bool {
	for outside := homeBoard + 1; outside <= len(board); outside++ {
		if board[outside-1] > 0 {
			return false
		}
	}
	if die == pt {
		return true
	}
	if die < pt {
		return false
	}
	// A table narrower than the home board — the small domains the tests use —
	// has no points above its own width to look at.
	top := homeBoard
	if len(board) < top {
		top = len(board)
	}
	for higher := pt + 1; higher <= top; higher++ {
		if board[higher-1] > 0 {
			return false
		}
	}
	return true
}

// gen holds the scratch space the move search reuses. successors is called
// nPos × 21 times — 1.1 million times for OS-06, far more for a wider domain —
// and allocating a slice per die per call cost 7.3 GiB of garbage and most of
// the wall clock. Everything below writes into buffers owned by gen.
type gen struct {
	points   int
	checkers int
	// boards[d] is the scratch board at depth d of the search.
	boards [5][]int
	seen   map[int]struct{}
	out    []int
	best   int
	dice   [4]int
	nDice  int
	used   [4]bool
}

func newGen(points, checkers int) *gen {
	g := &gen{points: points, checkers: checkers, seen: make(map[int]struct{}, 64)}
	for i := range g.boards {
		g.boards[i] = make([]int, points)
	}
	return g
}

// playDie writes into dst the arrangement reached from src by playing `die`
// from point `pt`, and reports whether that move is legal at all.
func (g *gen) playDie(dst, src []int, pt, die int) bool {
	if src[pt-1] == 0 {
		return false
	}
	if pt > die {
		copy(dst, src)
		dst[pt-1]--
		dst[pt-die-1]++
		return true
	}
	if !canBearOff(src, pt, die) {
		return false
	}
	copy(dst, src)
	dst[pt-1]--
	return true
}

// successors returns the indices of every arrangement reachable from `board`
// with the roll (d1, d2), playing as many dice as the rules require. The slice
// is owned by g and valid until the next call.
//
// The search tracks which dice are still in hand rather than alternating by
// depth: with two distinct dice, "played the 5, the 2 remains" and "played the
// 2, the 5 remains" are different states, and collapsing them lets the same die
// be played twice.
func (g *gen) successors(board []int, d1, d2 int) []int {
	g.nDice = 2
	g.dice[0], g.dice[1] = d1, d2
	if d1 == d2 {
		g.nDice = 4
		g.dice[2], g.dice[3] = d1, d1
	}
	for i := range g.used {
		g.used[i] = false
	}
	for k := range g.seen {
		delete(g.seen, k)
	}
	g.out = g.out[:0]
	g.best = -1
	g.walk(board, 0)
	return g.out
}

// walk explores one branch of the search. depth counts the dice played so far,
// and doubles as the index of the scratch board the next ply writes into.
func (g *gen) walk(b []int, depth int) {
	moved := false
	for i := 0; i < g.nDice; i++ {
		if g.used[i] {
			continue
		}
		// With doubles every die is the same: trying the first unused one is
		// enough, and keeps the branching factor at one per point.
		if g.dice[0] == g.dice[1] && i > 0 && !g.used[i-1] {
			continue
		}
		die := g.dice[i]
		next := g.boards[depth+1]
		// The die is taken out of hand for the whole descent, whether or not an
		// earlier die already moved: marking it only for the first mover let
		// the second die be played again at the next ply, and the search ran
		// past the four plies a double allows.
		g.used[i] = true
		for pt := g.points; pt >= 1; pt-- {
			if !g.playDie(next, b, pt, die) {
				continue
			}
			moved = true
			g.walk(next, depth+1)
		}
		g.used[i] = false
	}
	if moved {
		return
	}
	// A leaf: no die in hand can be played. Only the deepest leaves are legal
	// plays — "use as many dice as you can".
	switch {
	case depth > g.best:
		g.best = depth
		g.out = g.out[:0]
		for k := range g.seen {
			delete(g.seen, k)
		}
	case depth < g.best:
		return
	}
	idx := PositionIndex(b, g.points, g.checkers)
	if _, dup := g.seen[idx]; dup {
		return
	}
	g.seen[idx] = struct{}{}
	g.out = append(g.out, idx)
}
