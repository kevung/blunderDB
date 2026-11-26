package document

import (
	"encoding/binary"
	"errors"
	"math"
)

// A board font is known by what its glyphs draw, not by its name nor by the
// code a glyph sits at: a checker is a ring of ink at a constant radius
// around the glyph's centre, black when the inside is inked, white when it
// is not. This file reads the embedded TrueType program far enough to tell
// that — the outlines, never hinted nor rendered — and is bounded: the
// program comes from the document and is not trusted.

// ttFont is the part of a TrueType program the shape reader uses.
type ttFont struct {
	data       []byte
	glyf, loca []byte
	longLoca   bool
	numGlyphs  int
	cmaps      []ttCmap
}

type ttCmap struct {
	platform, encoding uint16
	sub                []byte
}

const (
	maxTTPoints     = 4096
	maxTTComponents = 32
	maxTTDepth      = 4
)

var errTT = errors.New("document: malformed TrueType program")

func parseTrueType(data []byte) (*ttFont, error) {
	if len(data) < 12 {
		return nil, errTT
	}
	n := int(binary.BigEndian.Uint16(data[4:]))
	if 12+16*n > len(data) {
		return nil, errTT
	}
	t := &ttFont{data: data}
	tables := map[string][]byte{}
	for i := range n {
		rec := data[12+16*i:]
		off, ln := binary.BigEndian.Uint32(rec[8:]), binary.BigEndian.Uint32(rec[12:])
		if uint64(off)+uint64(ln) > uint64(len(data)) {
			return nil, errTT
		}
		tables[string(rec[:4])] = data[off : off+ln]
	}
	head, maxp, cmap := tables["head"], tables["maxp"], tables["cmap"]
	t.glyf, t.loca = tables["glyf"], tables["loca"]
	if len(head) < 54 || len(maxp) < 6 || len(cmap) < 4 || t.glyf == nil || t.loca == nil {
		return nil, errTT
	}
	t.longLoca = binary.BigEndian.Uint16(head[50:]) != 0
	t.numGlyphs = int(binary.BigEndian.Uint16(maxp[4:]))
	nsub := int(binary.BigEndian.Uint16(cmap[2:]))
	for i := range nsub {
		if 4+8*i+8 > len(cmap) {
			return nil, errTT
		}
		rec := cmap[4+8*i:]
		off := binary.BigEndian.Uint32(rec[4:])
		if uint64(off) >= uint64(len(cmap)) {
			continue
		}
		t.cmaps = append(t.cmaps, ttCmap{binary.BigEndian.Uint16(rec), binary.BigEndian.Uint16(rec[2:]), cmap[off:]})
	}
	return t, nil
}

// glyphOf maps a simple font's character code to a glyph, the way a PDF
// consumer does for a symbolic TrueType font: the (3,0) table at the code or
// at 0xF000 plus the code, then the (1,0) table, then (3,1).
func (t *ttFont) glyphOf(code uint32) (int, bool) {
	try := func(p, e uint16, c uint32) (int, bool) {
		for _, cm := range t.cmaps {
			if cm.platform == p && cm.encoding == e {
				if g := cmapLookup(cm.sub, c); g > 0 && g < t.numGlyphs {
					return g, true
				}
			}
		}
		return 0, false
	}
	if g, ok := try(3, 0, code); ok {
		return g, true
	}
	if g, ok := try(3, 0, 0xF000|code); ok {
		return g, true
	}
	if g, ok := try(1, 0, code); ok {
		return g, true
	}
	return try(3, 1, code)
}

func cmapLookup(s []byte, c uint32) int {
	if len(s) < 6 {
		return 0
	}
	u16 := func(i int) int {
		if i < 0 || i+2 > len(s) {
			return 0
		}
		return int(binary.BigEndian.Uint16(s[i:]))
	}
	switch u16(0) {
	case 0:
		if c < 256 && 6+int(c) < len(s) {
			return int(s[6+c])
		}
	case 6:
		first, count := u16(6), u16(8)
		if int(c) >= first && int(c) < first+count {
			return u16(10 + 2*(int(c)-first))
		}
	case 4:
		segs := u16(6) / 2
		for i := range segs {
			end := u16(14 + 2*i)
			if int(c) > end {
				continue
			}
			startAt := 16 + 2*segs + 2*i
			start := u16(startAt)
			if int(c) < start {
				return 0
			}
			delta := u16(startAt + 2*segs)
			roAt := startAt + 4*segs
			ro := u16(roAt)
			if ro == 0 {
				return (int(c) + delta) & 0xFFFF
			}
			g := u16(roAt + ro + 2*(int(c)-start))
			if g == 0 {
				return 0
			}
			return (g + delta) & 0xFFFF
		}
	}
	return 0
}

type ttPt struct{ x, y float64 }

// outline returns the glyph's contours flattened to polygons, and its box.
func (t *ttFont) outline(g, depth int, budget *int) ([][]ttPt, error) {
	if g < 0 || g >= t.numGlyphs || depth > maxTTDepth {
		return nil, errTT
	}
	off := func(i int) (int, bool) {
		if t.longLoca {
			if 4*i+4 > len(t.loca) {
				return 0, false
			}
			return int(binary.BigEndian.Uint32(t.loca[4*i:])), true
		}
		if 2*i+2 > len(t.loca) {
			return 0, false
		}
		return 2 * int(binary.BigEndian.Uint16(t.loca[2*i:])), true
	}
	a, ok1 := off(g)
	b, ok2 := off(g + 1)
	if !ok1 || !ok2 || a > b || b > len(t.glyf) {
		return nil, errTT
	}
	if a == b {
		return nil, nil
	}
	d := t.glyf[a:b]
	if len(d) < 10 {
		return nil, errTT
	}
	nc := int(int16(binary.BigEndian.Uint16(d)))
	if nc < 0 {
		return t.composite(d[10:], depth, budget)
	}
	return simpleOutline(d[10:], nc, budget)
}

func simpleOutline(d []byte, nc int, budget *int) ([][]ttPt, error) {
	if 2*nc+2 > len(d) {
		return nil, errTT
	}
	ends := make([]int, nc)
	last := -1
	for i := range nc {
		ends[i] = int(binary.BigEndian.Uint16(d[2*i:]))
		if ends[i] <= last {
			return nil, errTT
		}
		last = ends[i]
	}
	np := last + 1
	*budget -= np
	if np > maxTTPoints || *budget < 0 {
		return nil, errTT
	}
	p := 2 * nc
	il := int(binary.BigEndian.Uint16(d[p:]))
	p += 2 + il
	flags := make([]byte, 0, np)
	for len(flags) < np {
		if p >= len(d) {
			return nil, errTT
		}
		f := d[p]
		p++
		flags = append(flags, f)
		if f&8 != 0 {
			if p >= len(d) {
				return nil, errTT
			}
			for r := int(d[p]); r > 0 && len(flags) < np; r-- {
				flags = append(flags, f)
			}
			p++
		}
	}
	coords := func(short, same byte) ([]float64, error) {
		out := make([]float64, np)
		v := 0
		for i, f := range flags {
			switch {
			case f&short != 0:
				if p >= len(d) {
					return nil, errTT
				}
				dv := int(d[p])
				p++
				if f&same == 0 {
					dv = -dv
				}
				v += dv
			case f&same == 0:
				if p+2 > len(d) {
					return nil, errTT
				}
				v += int(int16(binary.BigEndian.Uint16(d[p:])))
				p += 2
			}
			out[i] = float64(v)
		}
		return out, nil
	}
	xs, err := coords(2, 16)
	if err != nil {
		return nil, err
	}
	ys, err := coords(4, 32)
	if err != nil {
		return nil, err
	}
	var cs [][]ttPt
	start := 0
	for _, e := range ends {
		cs = append(cs, flatten(xs[start:e+1], ys[start:e+1], flags[start:e+1]))
		start = e + 1
	}
	return cs, nil
}

// flatten turns a quadratic contour into a polygon, each curve cut in four.
func flatten(xs, ys []float64, fl []byte) []ttPt {
	n := len(xs)
	if n == 0 {
		return nil
	}
	on := func(i int) bool { return fl[i]&1 != 0 }
	pt := func(i int) ttPt { return ttPt{xs[i%n], ys[i%n]} }
	mid := func(a, b ttPt) ttPt { return ttPt{(a.x + b.x) / 2, (a.y + b.y) / 2} }
	s := 0
	for s < n && !on(s) {
		s++
	}
	var start ttPt
	if s == n {
		start = mid(pt(0), pt(1))
		s = 0
	} else {
		start = pt(s)
		s++
	}
	out := []ttPt{start}
	cur := start
	for k := 0; k < n; k++ {
		i := (s + k) % n
		p := pt(i)
		if on(i) {
			out = append(out, p)
			cur = p
			continue
		}
		j := (i + 1) % n
		var end ttPt
		if on(j) {
			end = pt(j)
		} else {
			end = mid(p, pt(j))
		}
		for q := 1; q <= 4; q++ {
			u := float64(q) / 4
			a, b, c := (1-u)*(1-u), 2*u*(1-u), u*u
			out = append(out, ttPt{a*cur.x + b*p.x + c*end.x, a*cur.y + b*p.y + c*end.y})
		}
		cur = end
		if on(j) {
			k++
		}
	}
	return out
}

func (t *ttFont) composite(d []byte, depth int, budget *int) ([][]ttPt, error) {
	var out [][]ttPt
	p := 0
	for n := 0; ; n++ {
		if n >= maxTTComponents || p+4 > len(d) {
			return nil, errTT
		}
		flags := binary.BigEndian.Uint16(d[p:])
		g := int(binary.BigEndian.Uint16(d[p+2:]))
		p += 4
		var dx, dy float64
		if flags&1 != 0 {
			if p+4 > len(d) {
				return nil, errTT
			}
			dx, dy = float64(int16(binary.BigEndian.Uint16(d[p:]))), float64(int16(binary.BigEndian.Uint16(d[p+2:])))
			p += 4
		} else {
			if p+2 > len(d) {
				return nil, errTT
			}
			dx, dy = float64(int8(d[p])), float64(int8(d[p+1]))
			p += 2
		}
		if flags&2 == 0 {
			// Point-matched components are not placed by an offset.
			return nil, errTT
		}
		m := [4]float64{1, 0, 0, 1}
		f2 := func(i int) float64 { return float64(int16(binary.BigEndian.Uint16(d[i:]))) / 16384 }
		switch {
		case flags&8 != 0:
			if p+2 > len(d) {
				return nil, errTT
			}
			m[0], m[3] = f2(p), f2(p)
			p += 2
		case flags&0x40 != 0:
			if p+4 > len(d) {
				return nil, errTT
			}
			m[0], m[3] = f2(p), f2(p+2)
			p += 4
		case flags&0x80 != 0:
			if p+8 > len(d) {
				return nil, errTT
			}
			m = [4]float64{f2(p), f2(p + 2), f2(p + 4), f2(p + 6)}
			p += 8
		}
		cs, err := t.outline(g, depth+1, budget)
		if err != nil {
			return nil, err
		}
		for _, c := range cs {
			tc := make([]ttPt, len(c))
			for i, q := range c {
				tc[i] = ttPt{m[0]*q.x + m[2]*q.y + dx, m[1]*q.x + m[3]*q.y + dy}
			}
			out = append(out, tc)
		}
		if flags&0x20 == 0 {
			return out, nil
		}
	}
}

// inked reports whether a point is inside the outline (non-zero winding).
func inked(cs [][]ttPt, x, y float64) bool {
	w := 0
	for _, c := range cs {
		for i := range c {
			a, b := c[i], c[(i+1)%len(c)]
			if a.y <= y {
				if b.y > y && (b.x-a.x)*(y-a.y)-(x-a.x)*(b.y-a.y) > 0 {
					w++
				}
			} else if b.y <= y && (b.x-a.x)*(y-a.y)-(x-a.x)*(b.y-a.y) < 0 {
				w--
			}
		}
	}
	return w != 0
}

// discKind is what a glyph shows of a checker.
type discKind uint8

const (
	noDisc discKind = iota
	whiteDisc
	blackDisc
	unclearDisc // a ring whose inside is neither empty nor inked
)

// glyphShape is what the shape reader tells of one glyph.
type glyphShape struct {
	disc discKind
	// bars are the tray's checker bars a vertical line through the glyph's
	// centre crosses: filled (black) or outlined (white); mixed is both.
	bars      int
	barsBlack bool
	barsMixed bool
}

// Shape tests, in fractions of the glyph's box (its smaller side S, centred
// on the box): a ring is ink at one radius between 0.25 S and 0.52 S on at
// least ringCover of the rays; its inside, the disc of radius 0.25 S, is
// black from fillBlack of ink and white below fillWhite. A hatched or
// outlined point drawn behind a checker crosses a circle at scattered
// angles only, and the digits of a numbered checker cover a minority of its
// inside.
const (
	ringRays  = 96
	ringCover = 0.85
	// ringOutside bounds the rays inked just beyond the ring.
	ringOutside = 0.35
	fillBlack   = 0.6
	fillWhite   = 0.35
)

func shapeOf(cs [][]ttPt) glyphShape {
	var sh glyphShape
	if len(cs) == 0 {
		return sh
	}
	x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, c := range cs {
		for _, p := range c {
			x0, y0, x1, y1 = math.Min(x0, p.x), math.Min(y0, p.y), math.Max(x1, p.x), math.Max(y1, p.y)
		}
	}
	S := math.Min(x1-x0, y1-y0)
	if S <= 0 {
		return sh
	}
	cx, cy := (x0+x1)/2, (y0+y1)/2
	const steps = 56
	band := 0.012 * S
	// hit[r][a]: ink at radius step r on ray a.
	hit := make([][ringRays]bool, steps)
	for a := range ringRays {
		th := 2 * math.Pi * float64(a) / ringRays
		ct, st := math.Cos(th), math.Sin(th)
		for r := range steps {
			rad := (0.25 + 0.27*float64(r)/(steps-1)) * S
			hit[r][a] = inked(cs, cx+rad*ct, cy+rad*st)
		}
	}
	cover := func(r int) int {
		n := 0
		for a := range ringRays {
			for q := r; q < steps && (0.27*S*float64(q-r)/(steps-1)) <= band; q++ {
				if hit[q][a] {
					n++
					break
				}
			}
		}
		return n
	}
	outer := -1
	for r := steps - 1; r >= 0; r-- {
		if float64(cover(r)) >= ringCover*ringRays {
			outer = r
			break
		}
	}
	// A ring is a ring only with little ink just beyond it: a filled
	// square or a stack of bars is inked at every radius.
	ring := false
	if outer >= 0 {
		rad := (0.25+0.27*float64(outer)/(steps-1))*S + 0.04*S
		out := 0
		for a := range ringRays {
			th := 2 * math.Pi * float64(a) / ringRays
			if inked(cs, cx+rad*math.Cos(th), cy+rad*math.Sin(th)) {
				out++
			}
		}
		ring = float64(out) < ringOutside*ringRays
	}
	if ring {
		in, all := 0, 0
		for i := -6; i <= 6; i++ {
			for j := -6; j <= 6; j++ {
				dx, dy := float64(i)/6*0.25*S, float64(j)/6*0.25*S
				if dx*dx+dy*dy > 0.0625*S*S {
					continue
				}
				all++
				if inked(cs, cx+dx, cy+dy) {
					in++
				}
			}
		}
		switch f := float64(in) / float64(all); {
		case f >= fillBlack:
			sh.disc = blackDisc
		case f < fillWhite:
			sh.disc = whiteDisc
		default:
			sh.disc = unclearDisc
		}
		return sh
	}
	// The tray's bars, along the vertical through the centre: a run of ink
	// a twentieth of the box or thicker is a filled bar, a thinner one the
	// edge of an outlined bar.
	const vsteps = 400
	run, thick, thin := 0, 0, 0
	end := func() {
		if run == 0 {
			return
		}
		if float64(run)/vsteps*(y1-y0) >= 0.05*S {
			thick++
		} else {
			thin++
		}
		run = 0
	}
	for k := range vsteps {
		if inked(cs, cx, y0+(float64(k)+0.5)/vsteps*(y1-y0)) {
			run++
		} else {
			end()
		}
	}
	end()
	switch {
	case thick > 0 && thin == 0:
		sh.bars, sh.barsBlack = thick, true
	case thin > 0 && thick == 0 && thin%2 == 0:
		sh.bars = thin / 2
	case thick > 0 || thin > 0:
		sh.barsMixed = true
	}
	return sh
}

// fontShapes reads and caches the shapes of one embedded font's glyphs.
type fontShapes struct {
	tt    *ttFont
	cache map[uint32]glyphShape
	bad   map[uint32]bool
}

func newFontShapes(program []byte) *fontShapes {
	tt, err := parseTrueType(program)
	if err != nil {
		return nil
	}
	return &fontShapes{tt: tt, cache: map[uint32]glyphShape{}, bad: map[uint32]bool{}}
}

// shape returns the shape of the glyph a code draws; false when the program
// does not draw it or cannot be read.
func (f *fontShapes) shape(code uint32) (glyphShape, bool) {
	if sh, ok := f.cache[code]; ok {
		return sh, true
	}
	if f.bad[code] {
		return glyphShape{}, false
	}
	g, ok := f.tt.glyphOf(code)
	if !ok {
		f.bad[code] = true
		return glyphShape{}, false
	}
	budget := maxTTPoints
	cs, err := f.tt.outline(g, 0, &budget)
	if err != nil {
		f.bad[code] = true
		return glyphShape{}, false
	}
	sh := shapeOf(cs)
	f.cache[code] = sh
	return sh, true
}
