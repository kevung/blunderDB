package arkqr

import "github.com/kevung/blunderdb/pkg/blunderdb/domain"

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
	from, to int
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
	var srcs []int
	if r.my[25] > 0 {
		srcs = []int{25}
	} else {
		for p := 24; p >= 1; p-- {
			if r.my[p] > 0 {
				srcs = append(srcs, p)
			}
		}
	}
	hi := 0
	if len(srcs) > 0 {
		hi = srcs[0]
	}
	var out []step
	for _, p := range srcs {
		dst := p - d
		if dst >= 1 {
			o := r.opp[25-dst]
			if o >= 2 {
				continue
			}
			out = append(out, step{p, dst, o == 1})
			continue
		}
		if hi > 6 {
			continue
		}
		if dst == 0 || p == hi {
			out = append(out, step{p, 0, false})
		}
	}
	return out
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
	type term struct {
		used, first int
		steps       []step
	}
	found := map[rel][]term{}
	var order []rel
	best := 0
	var rec func(b rel, dice []int, used, first int, steps []step)
	rec = func(b rel, dice []int, used, first int, steps []step) {
		moved := false
		if len(dice) > 0 {
			for _, s := range b.single(dice[0]) {
				nb, _ := b.applyStep(s)
				f := first
				if f == 0 {
					f = dice[0]
				}
				moved = true
				rec(nb, dice[1:], used+1, f, append(append([]step(nil), steps...), s))
			}
		}
		if !moved {
			best = max(best, used)
			if _, ok := found[b]; !ok {
				order = append(order, b)
			}
			found[b] = append(found[b], term{used, first, steps})
		}
	}
	if d1 == d2 {
		rec(r, []int{d1, d1, d1, d1}, 0, 0, nil)
	} else {
		rec(r, []int{d1, d2}, 0, 0, nil)
		rec(r, []int{d2, d1}, 0, 0, nil)
	}
	larger := false
	if d1 != d2 && best == 1 {
		for _, b := range order {
			for _, t := range found[b] {
				if t.used == 1 && t.first == d1 {
					larger = true
				}
			}
		}
	}
	var out []play
	for _, b := range order {
		for _, t := range found[b] {
			if t.used == best && (!larger || t.first == d1) {
				out = append(out, play{b, t.steps})
				break
			}
		}
	}
	sortPlays(out)
	return out
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
