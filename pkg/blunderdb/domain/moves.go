package domain

import (
	"sort"
	"strconv"
	"strings"
)

// Legal checker-move generation. Given a position (board + dice + player on roll)
// it returns every legal complete play, each with its checker steps, the
// resulting position, and a notation. Pure (no deps) so the GUI/Desktop and the
// headless server share it. It implements the standard rules: enter from the bar
// first, play as many dice as possible, when only one die can be played use the
// larger if either works alone, bear off (exact or overage from the highest
// occupied home point), and hit blots.
//
// Geometry (board indices): WhiteBar=0, BlackBar=25, points 1..24. Black moves
// high→low (home 1..6, bears off past 1); White moves low→high (home 19..24,
// bears off past 24). The two dice of a non-double may be played in either order;
// doubles give four moves of the same value.

// Off is the destination sentinel for a checker borne off.
const Off = -1

// CheckerStep is one die's worth of movement. From is a point 1..24 or a bar
// (BlackBar/WhiteBar); To is a point 1..24 or Off (bear off). Hit marks a blot hit.
type CheckerStep struct {
	From int  `json:"from"`
	To   int  `json:"to"`
	Hit  bool `json:"hit"`
}

// LegalPlay is one complete legal play for the rolled dice.
type LegalPlay struct {
	Steps    []CheckerStep `json:"steps"`
	Result   Position      `json:"result"`
	Notation string        `json:"notation"`
}

// LegalMoves returns all distinct legal plays for p (deduplicated by resulting
// position). Returns nil when no dice are set; an empty (non-nil) slice means the
// player has no legal move (a dance).
func LegalMoves(p *Position) []LegalPlay {
	d1, d2 := p.Dice[0], p.Dice[1]
	if d1 < 1 || d1 > 6 || d2 < 1 || d2 > 6 {
		return nil
	}
	var dice []int
	if d1 == d2 {
		dice = []int{d1, d1, d1, d1}
	} else {
		dice = []int{d1, d2}
	}
	mover := p.PlayerOnRoll

	type seq struct {
		pos   Position
		steps []CheckerStep
	}
	var terminals []seq
	maxUsed := 0

	var explore func(pos Position, remaining []int, steps []CheckerStep)
	explore = func(pos Position, remaining []int, steps []CheckerStep) {
		moved := false
		tried := map[int]bool{}
		for i, die := range remaining {
			if tried[die] {
				continue // equal dice: trying one index is enough at this level
			}
			tried[die] = true
			for _, mv := range singleMoves(&pos, mover, die) {
				np := applyStep(&pos, mover, mv)
				rem := make([]int, 0, len(remaining)-1)
				rem = append(rem, remaining[:i]...)
				rem = append(rem, remaining[i+1:]...)
				explore(np, rem, append(append([]CheckerStep{}, steps...), mv))
				moved = true
			}
		}
		if !moved {
			if len(steps) > maxUsed {
				maxUsed = len(steps)
			}
			terminals = append(terminals, seq{pos, steps})
		}
	}
	explore(*p, dice, nil)

	// Keep only plays using the maximum number of dice (must play as many as
	// possible). For non-doubles where only one die is playable, the larger-die
	// rule is applied below.
	if maxUsed == 0 {
		return []LegalPlay{} // no legal move (dance)
	}
	larger := d1
	if d2 > larger {
		larger = d2
	}
	// Larger-die rule: if only one die can be played and the larger die is
	// playable alone, plays using only the smaller die are illegal.
	mustUseLarger := false
	if maxUsed == 1 && d1 != d2 {
		for _, t := range terminals {
			if len(t.steps) == 1 && stepUsesDie(t.steps[0], mover, larger) {
				mustUseLarger = true
				break
			}
		}
	}

	seen := map[string]bool{}
	var plays []LegalPlay
	for _, t := range terminals {
		if len(t.steps) != maxUsed {
			continue
		}
		if mustUseLarger && !stepUsesDie(t.steps[0], mover, larger) {
			continue
		}
		key := boardKey(&t.pos)
		if seen[key] {
			continue
		}
		seen[key] = true
		plays = append(plays, LegalPlay{Steps: t.steps, Result: t.pos, Notation: notation(t.steps, mover)})
	}
	return plays
}

// singleMoves lists the legal single-die moves in pos for mover. If mover has
// checkers on the bar, only bar entries are returned.
func singleMoves(pos *Position, mover, die int) []CheckerStep {
	bar := barIndex(mover)
	if pos.Board.Points[bar].Checkers > 0 {
		entry := entryPoint(mover, die)
		if canLand(pos, mover, entry) {
			return []CheckerStep{{From: bar, To: entry, Hit: isBlot(pos, mover, entry)}}
		}
		return nil
	}

	var moves []CheckerStep
	home := allInHome(pos, mover)
	for src := 1; src <= 24; src++ {
		pt := pos.Board.Points[src]
		if pt.Checkers <= 0 || pt.Color != mover {
			continue
		}
		dest := forward(mover, src, die)
		if dest >= 1 && dest <= 24 {
			if canLand(pos, mover, dest) {
				moves = append(moves, CheckerStep{From: src, To: dest, Hit: isBlot(pos, mover, dest)})
			}
			continue
		}
		// Off the board → bear off (only if all checkers are home).
		if home && canBearOff(pos, mover, src, die) {
			moves = append(moves, CheckerStep{From: src, To: Off})
		}
	}
	return moves
}

func barIndex(mover int) int {
	if mover == Black {
		return BlackBar
	}
	return WhiteBar
}

func entryPoint(mover, die int) int {
	if mover == Black {
		return 25 - die
	}
	return die
}

func forward(mover, src, die int) int {
	if mover == Black {
		return src - die
	}
	return src + die
}

func opponent(mover int) int {
	if mover == Black {
		return White
	}
	return Black
}

// canLand reports whether mover may land on point dest (1..24): not blocked by
// two or more opponent checkers.
func canLand(pos *Position, mover, dest int) bool {
	if dest < 1 || dest > 24 {
		return false
	}
	pt := pos.Board.Points[dest]
	if pt.Color == opponent(mover) && pt.Checkers >= 2 {
		return false
	}
	return true
}

func isBlot(pos *Position, mover, dest int) bool {
	if dest < 1 || dest > 24 {
		return false
	}
	pt := pos.Board.Points[dest]
	return pt.Color == opponent(mover) && pt.Checkers == 1
}

// allInHome reports whether every mover checker is in the home board (and none on
// the bar) — the precondition for bearing off.
func allInHome(pos *Position, mover int) bool {
	if pos.Board.Points[barIndex(mover)].Checkers > 0 {
		return false
	}
	for i := 1; i <= 24; i++ {
		pt := pos.Board.Points[i]
		if pt.Color != mover || pt.Checkers == 0 {
			continue
		}
		if mover == Black && i > 6 {
			return false
		}
		if mover == White && i < 19 {
			return false
		}
	}
	return true
}

// canBearOff reports whether mover may bear a checker off from src with die,
// assuming all checkers are home. Exact bear-off always; overage only from the
// highest occupied home point.
func canBearOff(pos *Position, mover, src, die int) bool {
	var need int
	if mover == Black {
		need = src // pips to bear off from src
	} else {
		need = 25 - src
	}
	if die == need {
		return true
	}
	if die < need {
		return false
	}
	// Overage: legal only if no checker sits further from bearing off than src.
	if mover == Black {
		for i := src + 1; i <= 6; i++ {
			if pos.Board.Points[i].Color == Black && pos.Board.Points[i].Checkers > 0 {
				return false
			}
		}
	} else {
		for i := 19; i < src; i++ {
			if pos.Board.Points[i].Color == White && pos.Board.Points[i].Checkers > 0 {
				return false
			}
		}
	}
	return true
}

// applyStep returns a copy of pos with the step applied (mover's checker moved,
// blot hit sent to the bar, bear-off counted).
func applyStep(pos *Position, mover int, s CheckerStep) Position {
	np := *pos // Board is a value (arrays) → copied

	from := &np.Board.Points[s.From]
	from.Checkers--
	if from.Checkers == 0 {
		from.Color = None
	}

	if s.To == Off {
		np.Board.Bearoff[mover]++
		return np
	}
	dst := &np.Board.Points[s.To]
	if s.Hit {
		ob := barIndex(opponent(mover))
		np.Board.Points[ob].Checkers++
		np.Board.Points[ob].Color = opponent(mover)
		dst.Checkers = 0
		dst.Color = None
	}
	dst.Checkers++
	dst.Color = mover
	return np
}

// boardKey is a stable dedup key for a resulting board (no engine dependency, so
// it stays in the dependency-free domain package).
func boardKey(p *Position) string {
	var b strings.Builder
	for i := 0; i <= 25; i++ {
		pt := p.Board.Points[i]
		b.WriteByte(byte('a' + pt.Checkers))
		b.WriteByte(byte('0' + pt.Color + 1)) // None(-1)→'0', Black→'1', White→'2'
	}
	b.WriteByte(byte('a' + p.Board.Bearoff[Black]))
	b.WriteByte(byte('a' + p.Board.Bearoff[White]))
	return b.String()
}

// stepUsesDie reports whether step s corresponds to moving exactly `die` pips.
func stepUsesDie(s CheckerStep, mover, die int) bool {
	switch {
	case s.From == BlackBar:
		return 25-s.To == die
	case s.From == WhiteBar:
		return s.To == die
	case s.To == Off:
		// Bear-off may be exact or overage; treat as using `die` if the pip need
		// is <= die (overage). For the larger-die rule a bear-off with the larger
		// die qualifies.
		var need int
		if mover == Black {
			need = s.From
		} else {
			need = 25 - s.From
		}
		return die >= need
	default:
		if mover == Black {
			return s.From-s.To == die
		}
		return s.To-s.From == die
	}
}

// notation renders the play in a compact "src/dst" form (Bar / off / * hit),
// collapsing identical steps as "(n)" and sorting tokens (matching the engine's
// NormalizeMove convention for comparison against analysed candidate moves).
func notation(steps []CheckerStep, mover int) string {
	counts := map[string]int{}
	var order []string
	for _, s := range steps {
		tok := pointLabel(mover, s.From) + "/" + pointLabel(mover, s.To)
		if s.Hit {
			tok += "*"
		}
		if _, ok := counts[tok]; !ok {
			order = append(order, tok)
		}
		counts[tok]++
	}
	tokens := make([]string, 0, len(order))
	for _, tok := range order {
		if counts[tok] > 1 {
			tok += "(" + strconv.Itoa(counts[tok]) + ")"
		}
		tokens = append(tokens, tok)
	}
	sort.Strings(tokens)
	return strings.Join(tokens, " ")
}

func pointLabel(mover, idx int) string {
	switch idx {
	case Off:
		return "off"
	case BlackBar, WhiteBar:
		return "Bar"
	default:
		return strconv.Itoa(idx)
	}
}
