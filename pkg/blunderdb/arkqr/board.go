package arkqr

import (
	"sync"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// half is one side of the board in its OWN numbering: index 1..24 is the
// side's points counted from its home (1 is the deepest home point), 25 its
// bar; index 0 is unused. A board as the codec sees it is the pair (my, opp):
// the side to play, then the other one, each in its own numbering.
type half [26]int8

type rel struct{ my, opp half }

// start is the opening position, identical for both sides.
var start = func() rel {
	var h half
	h[6], h[8], h[13], h[24] = 5, 3, 5, 2
	return rel{h, h}
}()

func (r rel) swap() rel { return rel{r.opp, r.my} }

// less orders boards lexicographically, my then opp, point 0 to 25: the order
// of a candidate list, fixed by the format and independent of any generator.
func (r rel) less(o rel) bool {
	for i := range 26 {
		if r.my[i] != o.my[i] {
			return r.my[i] < o.my[i]
		}
	}
	for i := range 26 {
		if r.opp[i] != o.opp[i] {
			return r.opp[i] < o.opp[i]
		}
	}
	return false
}

func (h half) count() int {
	n := 0
	for _, c := range h {
		n += int(c)
	}
	return n
}

// relOf reads a domain board from mover's side.
func relOf(b domain.Board, mover int) rel {
	var r rel
	for i := 1; i <= 24; i++ {
		p := b.Points[i]
		if p.Checkers <= 0 {
			continue
		}
		switch p.Color {
		case domain.Black:
			r.blackHalf(mover)[i] = int8(p.Checkers)
		case domain.White:
			r.whiteHalf(mover)[25-i] = int8(p.Checkers)
		}
	}
	r.blackHalf(mover)[25] = int8(b.Points[domain.BlackBar].Checkers)
	r.whiteHalf(mover)[25] = int8(b.Points[domain.WhiteBar].Checkers)
	return r
}

func (r *rel) blackHalf(mover int) *half {
	if mover == domain.Black {
		return &r.my
	}
	return &r.opp
}

func (r *rel) whiteHalf(mover int) *half {
	if mover == domain.Black {
		return &r.opp
	}
	return &r.my
}

// absPoint maps a point of mover's numbering (25 = bar, 0 = off) to a board
// index of the domain.
func absPoint(mover, p int) int {
	switch {
	case p == 0:
		return domain.Off
	case p == 25 && mover == domain.Black:
		return domain.BlackBar
	case p == 25:
		return domain.WhiteBar
	case mover == domain.Black:
		return p
	default:
		return 25 - p
	}
}

// relPoint is absPoint's inverse; ok is false for an index mover cannot use.
func relPoint(mover, idx int) (int, bool) {
	switch {
	case idx == domain.Off:
		return 0, true
	case idx == domain.BlackBar && mover == domain.Black, idx == domain.WhiteBar && mover == domain.White:
		return 25, true
	case idx < 1 || idx > 24:
		return 0, false
	case mover == domain.Black:
		return idx, true
	default:
		return 25 - idx, true
	}
}

// step is one die's move in the mover's numbering; to 0 is borne off.
type step struct {
	from, to int8
	hit      bool
}

// applyStep moves one checker; ok is false when the step cannot be made on r.
func (r rel) applyStep(s step) (rel, bool) {
	if s.from < 1 || s.from > 25 || s.to < 0 || s.to > 24 || r.my[s.from] <= 0 {
		return r, false
	}
	r.my[s.from]--
	if s.to == 0 {
		return r, true
	}
	o := &r.opp[25-s.to]
	switch {
	case *o >= 2:
		return r, false
	case *o == 1 && s.hit:
		*o = 0
		r.opp[25]++
	case *o == 1, s.hit:
		return r, false
	}
	r.my[s.to]++
	return r, true
}

// single lists the moves one die allows.
func (r rel) single(d int) []step {
	var buf [16]step
	return append([]step(nil), r.singleInto(d, &buf)...)
}

// singleInto is single, written into buf: a side has at most fifteen sources.
func (r rel) singleInto(d int, buf *[16]step) []step {
	out := buf[:0]
	if r.my[25] > 0 {
		dst := 25 - d
		if r.opp[25-dst] < 2 {
			out = append(out, step{25, int8(dst), r.opp[25-dst] == 1})
		}
		return out
	}
	hi := 0
	for p := 24; p >= 1; p-- {
		if r.my[p] == 0 {
			continue
		}
		if hi == 0 {
			hi = p
		}
		dst := p - d
		if dst >= 1 {
			o := r.opp[25-dst]
			if o < 2 {
				out = append(out, step{int8(p), int8(dst), o == 1})
			}
			continue
		}
		if hi > 6 {
			continue
		}
		if dst == 0 || p == hi {
			out = append(out, step{int8(p), 0, false})
		}
	}
	return out
}

// canMove reports whether one die of d moves a checker: single(d) is not empty.
func (r rel) canMove(d int) bool {
	if r.my[25] > 0 {
		return r.opp[25-(25-d)] < 2
	}
	hi := 0
	for p := 24; p >= 1; p-- {
		if r.my[p] == 0 {
			continue
		}
		if hi == 0 {
			hi = p
		}
		if dst := p - d; dst >= 1 {
			if r.opp[25-dst] < 2 {
				return true
			}
		} else if hi <= 6 && (dst == 0 || p == hi) {
			return true
		}
	}
	return false
}

// play is one legal play: the board it leaves and the steps that reach it.
type play struct {
	after rel
	steps []step
}

// plays lists the legal plays of the roll (d1, d2), deduplicated by the board
// they leave, in the codec's order (rel.less). The steps of a play are the
// first sequence the exploration finds — larger die first —, which is how a
// decoded play is written.
func (r rel) plays(d1, d2 int) []play {
	if d1 < d2 {
		d1, d2 = d2, d1
	}
	g := explorers.Get().(*explorer)
	defer g.release()
	if d1 == d2 {
		g.memo = true
		g.rec(r, [4]int8{int8(d1), int8(d1), int8(d1), int8(d1)}, 4, 0, 0, [4]step{})
	} else {
		g.rec(r, [4]int8{int8(d1), int8(d2)}, 2, 0, 0, [4]step{})
		g.rec(r, [4]int8{int8(d2), int8(d1)}, 2, 0, 0, [4]step{})
	}
	larger := false
	if d1 != d2 && g.best == 1 {
		for _, t := range g.terms {
			if t.used == 1 && int(t.first) == d1 {
				larger = true
				break
			}
		}
	}
	// The first sequence of each board that uses the most dice — the larger
	// die when only one can be played.
	pick := g.pick[:0]
	for range g.boards {
		pick = append(pick, -1)
	}
	g.pick = pick
	n := 0
	for i, t := range g.terms {
		if pick[t.board] < 0 && int(t.used) == g.best && (!larger || int(t.first) == d1) {
			pick[t.board] = int32(i)
			n++
		}
	}
	out := make([]play, 0, n)
	backing := make([]step, 0, 4*n)
	for b, i := range pick {
		if i < 0 {
			continue
		}
		t := &g.terms[i]
		start := len(backing)
		backing = append(backing, t.steps[:t.used]...)
		var steps []step
		if t.used > 0 {
			steps = backing[start:len(backing):len(backing)]
		}
		out = append(out, play{g.boards[b], steps})
	}
	sortPlays(out)
	return out
}

// explorers keeps the tables of an exploration from one roll to the next.
var explorers = sync.Pool{New: func() any {
	return &explorer{index: make(map[rel]int32, 64), seen: make(map[node]struct{}, 256)}
}}

func (g *explorer) release() {
	clear(g.index)
	clear(g.seen)
	g.boards, g.terms, g.best, g.memo = g.boards[:0], g.terms[:0], 0, false
	explorers.Put(g)
}

// explorer walks every sequence of moves a roll allows, recording where each
// one ends in the order it finds them.
type explorer struct {
	index  map[rel]int32
	boards []rel
	terms  []term
	pick   []int32
	best   int
	// memo skips a board already explored at the same depth, which a double
	// reaches by every order of its moves: the sequences below it end on
	// boards already recorded, by a sequence found earlier, so the plays and
	// their steps are the ones the full exploration gives.
	memo bool
	seen map[node]struct{}
}

type node struct {
	b    rel
	used int8
}

// term is one sequence that cannot go further: the board it leaves (an index
// into boards), the dice it used and the first of them.
type term struct {
	board       int32
	used, first int8
	steps       [4]step
}

func (g *explorer) rec(b rel, dice [4]int8, nd, used, first int, steps [4]step) {
	if g.memo && used > 0 {
		k := node{b, int8(used)}
		if _, ok := g.seen[k]; ok {
			return
		}
		g.seen[k] = struct{}{}
	}
	moved := false
	if used < nd {
		d := int(dice[used])
		var buf [16]step
		for _, s := range b.singleInto(d, &buf) {
			nb, _ := b.applyStep(s)
			f := first
			if f == 0 {
				f = d
			}
			moved = true
			steps[used] = s
			g.rec(nb, dice, nd, used+1, f, steps)
		}
	}
	if !moved {
		g.best = max(g.best, used)
		k, ok := g.index[b]
		if !ok {
			k = int32(len(g.boards))
			g.index[b] = k
			g.boards = append(g.boards, b)
		}
		g.terms = append(g.terms, term{k, int8(used), int8(first), steps})
	}
}

func sortPlays(ps []play) {
	for i := 1; i < len(ps); i++ {
		for j := i; j > 0 && ps[j].after.less(ps[j-1].after); j-- {
			ps[j], ps[j-1] = ps[j-1], ps[j]
		}
	}
}

// contact reports whether the two sides can still meet: the regime of the
// play model.
func (r rel) contact() int {
	a, b := 0, 0
	for p := 25; p >= 1; p-- {
		if r.my[p] > 0 {
			a = p
			break
		}
	}
	for p := 25; p >= 1; p-- {
		if r.opp[p] > 0 {
			b = p
			break
		}
	}
	if a+b >= 25 {
		return 1
	}
	return 0
}
