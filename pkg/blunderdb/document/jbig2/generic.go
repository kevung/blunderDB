package jbig2

import (
	"bytes"
	"io"

	"golang.org/x/image/ccitt"
)

// genericParams are the parameters of the generic region decoding procedure
// (6.2) this decoder takes: no EXTTEMPLATE, no USESKIP.
type genericParams struct {
	w, h     int
	mmr      bool
	template int
	tpgdon   bool
	at       [4][2]int // dx, dy of the adaptive pixels; template 0 uses four
}

// The adaptive pixels at their nominal place, and the context of the
// typical-prediction bit, by template.
var (
	nominalAT = [4][4][2]int{{{3, -1}, {-3, -1}, {2, -2}, {-2, -2}}, {{3, -1}}, {{2, -1}}, {{2, -1}}}
	sltpCX    = [4]int{0x9B25, 0x0795, 0x00E5, 0x0195}
	genericCX = [4]int{1 << 16, 1 << 13, 1 << 10, 1 << 10}
	// With nominal adaptive pixels, the context of a template is three
	// runs of neighbours: on row y-2 the pixels lo..hi around x, shifted
	// by s2; on row y-1 the same, shifted by s1; on row y the n0 pixels
	// left of x.
	windows = [4]struct{ lo2, hi2, s2, lo1, hi1, s1, n0 int }{
		{-2, 2, 11, -3, 3, 4, 4},
		{-1, 2, 9, -2, 3, 3, 3},
		{-1, 1, 7, -2, 2, 2, 2},
		{0, -1, 0, -3, 2, 4, 4},
	}
)

func (p *genericParams) nominal() bool {
	n := 1
	if p.template == 0 {
		n = 4
	}
	for i := 0; i < n; i++ {
		if p.at[i] != nominalAT[p.template][i] {
			return false
		}
	}
	return true
}

// decodeGeneric decodes an arithmetic generic region with the decoder d and
// the contexts cx. check is called every few rows.
func decodeGeneric(d *mq, cx contexts, p genericParams, check func() error) (*bitmap, error) {
	m := newBitmap(p.w, p.h)
	ltp := 0
	nominal := p.nominal()
	wn := windows[p.template]
	for y := 0; y < p.h; y++ {
		if y&255 == 255 && check != nil {
			if err := check(); err != nil {
				return nil, err
			}
		}
		if p.tpgdon {
			ltp ^= d.bit(cx, sltpCX[p.template])
			if ltp == 1 {
				if y > 0 {
					copy(m.row(y), m.row(y-1))
				}
				continue
			}
		}
		row, r1, r2 := m.row(y), m.row(y-1), m.row(y-2)
		if !nominal {
			for x := 0; x < p.w; x++ {
				if d.bit(cx, contextSlow(m, x, y, &p)) == 1 {
					row[x>>3] |= 0x80 >> uint(x&7)
				}
			}
			continue
		}
		var w2, w1, w0 uint32
		if p.template != 3 {
			for k := wn.lo2; k <= wn.hi2; k++ {
				w2 = w2<<1 | px(r2, k, p.w)
			}
		}
		for k := wn.lo1; k <= wn.hi1; k++ {
			w1 = w1<<1 | px(r1, k, p.w)
		}
		m2 := uint32(1)<<uint(wn.hi2-wn.lo2+1) - 1
		m1 := uint32(1)<<uint(wn.hi1-wn.lo1+1) - 1
		m0 := uint32(1)<<uint(wn.n0) - 1
		for x := 0; x < p.w; x++ {
			v := d.bit(cx, int(w2<<uint(wn.s2)|w1<<uint(wn.s1)|w0))
			if v == 1 {
				row[x>>3] |= 0x80 >> uint(x&7)
			}
			w0 = (w0<<1 | uint32(v)) & m0
			w1 = (w1<<1 | px(r1, x+1+wn.hi1, p.w)) & m1
			if p.template != 3 {
				w2 = (w2<<1 | px(r2, x+1+wn.hi2, p.w)) & m2
			}
		}
	}
	return m, nil
}

// contextSlow is the context of pixel (x, y) in the order of 6.2.5.3, for
// adaptive pixels anywhere.
func contextSlow(m *bitmap, x, y int, p *genericParams) int {
	g := func(dx, dy int) int { return m.get(x+dx, y+dy) }
	a := p.at
	switch p.template {
	case 0:
		return g(-1, 0) | g(-2, 0)<<1 | g(-3, 0)<<2 | g(-4, 0)<<3 | g(a[0][0], a[0][1])<<4 |
			g(2, -1)<<5 | g(1, -1)<<6 | g(0, -1)<<7 | g(-1, -1)<<8 | g(-2, -1)<<9 |
			g(a[1][0], a[1][1])<<10 | g(a[2][0], a[2][1])<<11 |
			g(1, -2)<<12 | g(0, -2)<<13 | g(-1, -2)<<14 | g(a[3][0], a[3][1])<<15
	case 1:
		return g(-1, 0) | g(-2, 0)<<1 | g(-3, 0)<<2 | g(a[0][0], a[0][1])<<3 |
			g(2, -1)<<4 | g(1, -1)<<5 | g(0, -1)<<6 | g(-1, -1)<<7 | g(-2, -1)<<8 |
			g(2, -2)<<9 | g(1, -2)<<10 | g(0, -2)<<11 | g(-1, -2)<<12
	case 2:
		return g(-1, 0) | g(-2, 0)<<1 | g(a[0][0], a[0][1])<<2 |
			g(1, -1)<<3 | g(0, -1)<<4 | g(-1, -1)<<5 | g(-2, -1)<<6 |
			g(1, -2)<<7 | g(0, -2)<<8 | g(-1, -2)<<9
	default:
		return g(-1, 0) | g(-2, 0)<<1 | g(-3, 0)<<2 | g(-4, 0)<<3 | g(a[0][0], a[0][1])<<4 |
			g(1, -1)<<5 | g(0, -1)<<6 | g(-1, -1)<<7 | g(-2, -1)<<8 | g(-3, -1)<<9
	}
}

// decodeMMR decodes a generic region coded as T.6 (Group 4 fax), whose 1
// bits are black. A stream broken midway keeps the rows decoded before the
// fault, the others white, as viewers draw it.
func decodeMMR(data []byte, w, h int) *bitmap {
	m := newBitmap(w, h)
	if w == 0 || h == 0 {
		return m
	}
	r := ccitt.NewReader(bytes.NewReader(data), ccitt.MSB, ccitt.Group4, w, h, &ccitt.Options{Invert: true})
	_, _ = io.ReadFull(r, m.b)
	return m
}
