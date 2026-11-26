package jbig2

// refineParams are the parameters of the generic refinement region decoding
// procedure (6.3).
type refineParams struct {
	w, h     int
	template int
	ref      *bitmap
	dx, dy   int
	tpgron   bool
	at       [2][2]int // template 0 only
}

const refineCX = 1 << 13

// The typical-prediction bit is coded in the context where only the
// reference pixel at the centre is set.
var refineSLTP = [2]int{0x100, 0x080}

func refineContext(m *bitmap, x, y int, p *refineParams) int {
	g := func(dx, dy int) int { return m.get(x+dx, y+dy) }
	r := func(dx, dy int) int { return p.ref.get(x-p.dx+dx, y-p.dy+dy) }
	if p.template == 0 {
		a := p.at
		return g(-1, 0) | g(1, -1)<<1 | g(0, -1)<<2 | g(a[0][0], a[0][1])<<3 |
			r(1, 1)<<4 | r(0, 1)<<5 | r(-1, 1)<<6 | r(1, 0)<<7 | r(0, 0)<<8 |
			r(-1, 0)<<9 | r(1, -1)<<10 | r(0, -1)<<11 | r(a[1][0], a[1][1])<<12
	}
	return g(-1, 0) | g(1, -1)<<1 | g(0, -1)<<2 | g(-1, -1)<<3 |
		r(1, 1)<<4 | r(0, 1)<<5 | r(1, 0)<<6 | r(0, 0)<<7 | r(-1, 0)<<8 | r(0, -1)<<9
}

// typical tells whether the 3×3 reference pixels around (x, y) of the region
// are all of one colour, and which.
func typical(p *refineParams, x, y int) (int, bool) {
	cx, cy := x-p.dx, y-p.dy
	v := p.ref.get(cx, cy)
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if p.ref.get(cx+dx, cy+dy) != v {
				return 0, false
			}
		}
	}
	return v, true
}

func decodeRefine(d *mq, cx contexts, p refineParams, check func() error) (*bitmap, error) {
	m := newBitmap(p.w, p.h)
	ltp := 0
	for y := 0; y < p.h; y++ {
		if y&255 == 255 && check != nil {
			if err := check(); err != nil {
				return nil, err
			}
		}
		if p.tpgron {
			ltp ^= d.bit(cx, refineSLTP[p.template])
		}
		for x := 0; x < p.w; x++ {
			if ltp == 1 {
				if v, ok := typical(&p, x, y); ok {
					if v == 1 {
						m.set(x, y, 1)
					}
					continue
				}
			}
			if d.bit(cx, refineContext(m, x, y, &p)) == 1 {
				m.set(x, y, 1)
			}
		}
	}
	return m, nil
}
