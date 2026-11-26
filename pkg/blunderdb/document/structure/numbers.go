package structure

import (
	_ "embed"
	"math"
	"sort"
)

// Numbers printed around a diagram, read from the image itself: the pip
// counts of a page without a text layer. The reader abstains rather than
// guess: a number is read only when it stands alone and every one of its
// figures is recognised with a clear margin over every other figure, and no
// figure is ever chosen after the pip counts a diagram's checkers give.

// digitFont holds the figures 0 to 9 of eight common typefaces (sans serif,
// serif, their italics and bold), each normalised like a figure read from a
// page — its box centred in a square of glyphSide pixels — and quantised to
// levels 0 to 9, two pixels a byte (high nibble first): 8 × 10 × 400 levels.
// They start the reading on a document whose own figures are not yet known.
//
//go:embed digits.bin
var digitFont []byte

const glyphSide = 20

const (
	figureScore  = 0.75 // the least correlation for a figure to be read
	figureMargin = 0.05 // the least lead over the second best figure
	figureMinPx  = 15   // a figure lower than this, in pixels, is not read
)

// holesOf is how many holes each figure has, on its normalised square.
var holesOf = [10][]int{{1}, {0}, {0}, {0}, {0, 1}, {0}, {1}, {0}, {2}, {1}}

// printedNumber is a group of one to three figures beside a diagram.
type printedNumber struct {
	x0, y0, x1, y1 int        // in the page's pixels
	rel            [2]float64 // centre, relative to the diagram's box
	h              int
	alone          bool // no letter nor other ink beside it on its line
	cut            bool // near the edge of the zone searched
	glyphs         [][glyphSide * glyphSide]uint8
	val            int // the number read, or -1
}

func (n printedNumber) cx() float64 { return float64(n.x0+n.x1) / 2 }
func (n printedNumber) cy() float64 { return float64(n.y0+n.y1) / 2 }

// label8 numbers the 8-connected components of m from 1, in the order a
// raster scan meets them, and returns each one's box.
func label8(m *mask) ([]int32, []box) {
	lab := make([]int32, len(m.p))
	var boxes []box
	stack := []int{}
	for i, v := range m.p {
		if !v || lab[i] != 0 {
			continue
		}
		n := int32(len(boxes) + 1)
		b := box{m.w, m.h, 0, 0}
		lab[i] = n
		stack = append(stack[:0], i)
		for len(stack) > 0 {
			j := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			x, y := j%m.w, j/m.w
			b.x0, b.y0 = min(b.x0, x), min(b.y0, y)
			b.x1, b.y1 = max(b.x1, x+1), max(b.y1, y+1)
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					xx, yy := x+dx, y+dy
					if xx < 0 || yy < 0 || xx >= m.w || yy >= m.h {
						continue
					}
					k := yy*m.w + xx
					if m.p[k] && lab[k] == 0 {
						lab[k] = n
						stack = append(stack, k)
					}
				}
			}
		}
		boxes = append(boxes, b)
	}
	return lab, boxes
}

// normSquare centres a figure's box in a square, keeping its proportions,
// and resizes it to glyphSide × glyphSide as an 8-bit image (0 paper, 255
// ink).
func normSquare(w, h int, ink func(x, y int) bool) []float64 {
	d := max(w, h)
	sq := make([]float64, d*d)
	oy, ox := (d-h)/2, (d-w)/2
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if ink(x, y) {
				sq[(oy+y)*d+ox+x] = 255
			}
		}
	}
	return resample(sq, d, d, glyphSide, glyphSide, triangle, 1, true)
}

// squareHoles counts the holes of a figure on its normalised square: the
// 4-connected parts of the paper that do not reach the border, of two pixels
// or more (a speck of a halftone is not a hole), and gives the height of the
// largest one's centre (0 top, 1 bottom; -1 without a hole).
func squareHoles(v []float64) (int, float64) {
	const s = glyphSide + 2
	paper := newMask(s, s)
	for y := 0; y < s; y++ {
		for x := 0; x < s; x++ {
			paper.p[y*s+x] = x == 0 || y == 0 || x == s-1 || y == s-1 || v[(y-1)*glyphSide+x-1] <= 127.5
		}
	}
	lab, boxes, areas := label(paper)
	border := map[int32]bool{}
	for i := 0; i < s; i++ {
		border[lab[i]], border[lab[(s-1)*s+i]], border[lab[i*s]], border[lab[i*s+s-1]] = true, true, true, true
	}
	n, best, bestY := 0, 0, -1.0
	for k := range boxes {
		l := int32(k + 1)
		if border[l] {
			continue
		}
		if areas[k] >= 2 {
			n++
		}
		if areas[k] > best {
			sum := 0
			for i, x := range lab {
				if x == l {
					sum += i / s
				}
			}
			best, bestY = areas[k], (float64(sum)/float64(areas[k])-1)/float64(glyphSide-1)
		}
	}
	return n, bestY
}

// extractNumbers finds the numbers printed around the diagram whose box is b
// on the normalised page g: in the box widened by 5 point widths across and
// 1.8 along, components of ink of a figure's size, grouped into numbers of
// one to three figures of one height on one baseline.
func extractNumbers(g *raster, b box) []printedNumber {
	X0, Y0, X1, Y1 := float64(b.x0), float64(b.y0), float64(b.x1), float64(b.y1)
	W := X1 - X0
	p := W / 13
	rx0, rx1 := max(0, trunc(X0-5*p)), min(g.w, trunc(X1+5*p))
	ry0, ry1 := max(0, trunc(Y0-1.8*p)), min(g.h, trunc(Y1+1.8*p))
	if rx1-rx0 < 4 || ry1-ry0 < 4 {
		return nil
	}
	sub := g.crop(rx0, ry0, rx1, ry1)
	ink := hysteresis(sub)
	lab, comps := label8(ink)
	hmin, hmax := max(6, 0.10*p), 0.9*p
	var ok []int
	for i, c := range comps {
		h, w := float64(c.h()), float64(c.w())
		if !(hmin <= h && h <= hmax && w <= 1.3*h) {
			continue
		}
		if c.y0 == 0 || c.x0 == 0 || c.y1 == sub.h || c.x1 == sub.w {
			continue
		}
		ok = append(ok, i)
	}
	var out []printedNumber
	for _, grp := range groupFigures(comps, ok) {
		if len(grp.figs) > 3 {
			continue
		}
		alone := grp.alone
		n := printedNumber{val: -1}
		gy0, gy1, gx0, gx1 := sub.h, 0, sub.w, 0
		for _, i := range grp.figs {
			c := comps[i]
			gy0, gy1, gx0, gx1 = min(gy0, c.y0), max(gy1, c.y1), min(gx0, c.x0), max(gx1, c.x1)
			l := int32(i + 1)
			v := normSquare(c.w(), c.h(), func(x, y int) bool { return lab[(c.y0+y)*sub.w+c.x0+x] == l })
			var q [glyphSide * glyphSide]uint8
			for k, x := range v {
				q[k] = uint8(x)
			}
			n.glyphs = append(n.glyphs, q)
		}
		hh := float64(gy1 - gy0)
		if float64(min(gy0, gx0, sub.h-gy1, sub.w-gx1)) < 0.6*hh {
			alone = false
		}
		// Ink beside the number on its line, whatever its component (a
		// figure joined to the frame belongs to a large one): only a long
		// vertical stroke passing by is allowed (frame, bar); a dot or a
		// colon is ignored.
		ya, yb := trunc(float64(gy0)+0.2*hh), trunc(float64(gy1)-0.2*hh)+1
		in := map[int32]bool{}
		for _, i := range grp.figs {
			in[int32(i+1)] = true
		}
		for _, band := range [2][2]int{{max(0, trunc(float64(gx0)-0.6*hh)), gx0}, {gx1, min(sub.w, trunc(float64(gx1)+0.6*hh)+1)}} {
			xa, xb := band[0], band[1]
			seen := map[int32]bool{}
			for y := max(ya, 0); y < min(yb, sub.h); y++ {
				for x := xa; x < xb; x++ {
					l := lab[y*sub.w+x]
					if l == 0 || in[l] || seen[l] {
						continue
					}
					seen[l] = true
					c := comps[l-1]
					if float64(c.h()) < 0.3*hh && float64(c.w()) < 0.3*hh {
						continue
					}
					if !passingLine(lab, sub.w, sub.h, l, xa, xb, ya, yb, gy0, gy1, hh) {
						alone = false
					}
				}
			}
		}
		n.x0, n.y0, n.x1, n.y1 = gx0+rx0, gy0+ry0, gx1+rx0, gy1+ry0
		n.h, n.alone = n.y1-n.y0, alone
		n.rel = [2]float64{round4((n.cx() - X0) / W), round4((n.cy() - Y0) / (Y1 - Y0))}
		m := 0.6 * float64(n.h)
		n.cut = float64(n.x0)-(X0-5*p) < m || (X1+5*p)-float64(n.x1) < m || float64(n.y0)-(Y0-1.8*p) < m || (Y1+1.8*p)-float64(n.y1) < m
		out = append(out, n)
	}
	return out
}

func round4(x float64) float64 { return math.RoundToEven(x*1e4) / 1e4 }

type figureGroup struct {
	figs  []int
	alone bool
}

// groupFigures chains figures left to right: about one height, one baseline,
// a gap from -0.15 to 0.45 height. A group is not alone when a letter-sized
// component lies on its line within 0.6 height.
func groupFigures(comps []box, ok []int) []figureGroup {
	G := append([]int(nil), ok...)
	sort.SliceStable(G, func(a, b int) bool { return comps[G[a]].x0 < comps[G[b]].x0 })
	used := map[int]bool{}
	var groups [][]int
	for _, i := range G {
		if used[i] {
			continue
		}
		grp := []int{i}
		used[i] = true
		for {
			a := comps[grp[len(grp)-1]]
			ha := float64(a.h())
			nxt := -1
			for _, j := range G {
				if used[j] {
					continue
				}
				b := comps[j]
				hb := float64(b.h())
				h := max(ha, hb)
				if h > 1.35*min(ha, hb) {
					continue
				}
				if math.Abs(float64(a.y1-b.y1)) > 0.2*h || math.Abs(float64(a.y0-b.y0)) > 0.3*h {
					continue
				}
				gap := float64(b.x0 - a.x1)
				if -0.15*h <= gap && gap <= 0.45*h && (nxt < 0 || b.x0 < comps[nxt].x0) {
					nxt = j
				}
			}
			if nxt < 0 {
				break
			}
			grp = append(grp, nxt)
			used[nxt] = true
		}
		groups = append(groups, grp)
	}
	out := make([]figureGroup, 0, len(groups))
	for _, grp := range groups {
		y0, y1, x0, x1 := comps[grp[0]].y0, comps[grp[0]].y1, comps[grp[0]].x0, comps[grp[0]].x1
		in := map[int]bool{}
		for _, i := range grp {
			c := comps[i]
			y0, y1, x0, x1 = min(y0, c.y0), max(y1, c.y1), min(x0, c.x0), max(x1, c.x1)
			in[i] = true
		}
		h := float64(y1 - y0)
		alone := true
		for k, c := range comps {
			if in[k] {
				continue
			}
			ov := float64(min(c.y1, y1) - max(c.y0, y0))
			ch := float64(c.h())
			if ov < 0.5*h || !(0.25*h <= ch && ch <= 1.6*h) {
				continue // a frame's stroke is no letter
			}
			if float64(max(c.x0-x1, x0-c.x1)) < 0.6*h {
				alone = false
				break
			}
		}
		out = append(out, figureGroup{grp, alone})
	}
	return out
}

// passingLine tells whether the ink of component l in the band beside a
// number is a vertical stroke passing by: one narrow run a row at a constant
// abscissa, the stroke going on half a height above and below the number.
func passingLine(lab []int32, w, h int, l int32, xa, xb, ya, yb, gy0, gy1 int, hh float64) bool {
	var xs []float64
	for y := max(ya, 0); y < min(yb, h); y++ {
		lo, hi, n := -1, -1, 0
		for x := xa; x < xb; x++ {
			if lab[y*w+x] == l {
				if lo < 0 {
					lo = x - xa
				}
				hi = x - xa
				n++
			}
		}
		if n == 0 || float64(hi-lo+1) > max(3, 0.35*hh) || n != hi-lo+1 {
			return false
		}
		xs = append(xs, float64(lo+hi)/2)
	}
	if len(xs) == 0 {
		return false
	}
	lo, hi, sum := xs[0], xs[0], 0.0
	for _, x := range xs {
		lo, hi, sum = min(lo, x), max(hi, x), sum+x
	}
	if hi-lo > max(1.5, 0.1*hh) {
		return false
	}
	x := roundHalfEven(float64(xa) + sum/float64(len(xs)))
	half := max(2, trunc(0.2*hh))
	var rows []int
	for y := max(0, trunc(float64(gy0)-0.5*hh)); y < gy0; y++ {
		rows = append(rows, y)
	}
	for y := gy1; y < min(h, trunc(float64(gy1)+0.5*hh)); y++ {
		rows = append(rows, y)
	}
	for _, y := range rows {
		found := false
		for xx := max(0, x-half); xx < min(w, x+half+1); xx++ {
			if lab[y*w+xx] == l {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// shears are the slants tried to straighten an italic figure.
var shears = func() [17]float64 {
	var s [17]float64
	for i := range s {
		s[i] = -0.4 + float64(i)*0.8/16
	}
	s[16] = 0.4
	return s
}()

// deslant straightens an italic figure: the horizontal shear that makes the
// vertical projection of its ink the most peaked, then the box re-squared.
func deslant(m []float64) []float64 {
	var xs, ys []float64
	for i, v := range m {
		if v > 0.5 {
			ys = append(ys, float64(i/glyphSide))
			xs = append(xs, float64(i%glyphSide))
		}
	}
	if len(xs) < 4 {
		return m
	}
	yc := 0.0
	for _, y := range ys {
		yc += y
	}
	yc /= float64(len(ys))
	best, bestScore := 0.0, -1.0
	for _, s := range shears {
		hist := map[int]float64{}
		for i := range xs {
			hist[roundHalfEven(xs[i]+s*(ys[i]-yc)+40)]++
		}
		sc := 0.0
		for _, c := range hist {
			sc += c * c
		}
		if sc > bestScore {
			best, bestScore = s, sc
		}
	}
	if math.Abs(best) < 0.05 {
		return m
	}
	const bw = glyphSide + 40
	big := make([]bool, glyphSide*bw)
	for i := range xs {
		x := min(max(roundHalfEven(xs[i]+best*(ys[i]-yc)+20), 0), bw-1)
		big[int(ys[i])*bw+x] = true
	}
	x0, y0, x1, y1 := bw, glyphSide, 0, 0
	for i, v := range big {
		if v {
			x, y := i%bw, i/bw
			x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x+1), max(y1, y+1)
		}
	}
	v := normSquare(x1-x0, y1-y0, func(x, y int) bool { return big[(y0+y)*bw+x0+x] })
	for i := range v {
		v[i] /= 255
	}
	return v
}

// unitVector prepares a figure for correlation: straightened, blurred by one
// 3 × 3 mean (thickness and a small shift tolerated), centred and scaled to
// unit length.
func unitVector(m []float64) []float64 {
	v := deslant(m)
	b := make([]float64, len(v))
	for y := 0; y < glyphSide; y++ {
		for x := 0; x < glyphSide; x++ {
			s := 0.0
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					yy, xx := min(max(y+dy, 0), glyphSide-1), min(max(x+dx, 0), glyphSide-1)
					s += v[yy*glyphSide+xx]
				}
			}
			b[y*glyphSide+x] = s / 9
		}
	}
	return centred(b)
}

func centred(b []float64) []float64 {
	mean := 0.0
	for _, x := range b {
		mean += x
	}
	mean /= float64(len(b))
	norm := 0.0
	out := make([]float64, len(b))
	for i, x := range b {
		out[i] = x - mean
		norm += out[i] * out[i]
	}
	norm = math.Sqrt(norm) + 1e-6
	for i := range out {
		out[i] /= norm
	}
	return out
}

type template struct {
	digit int
	v     []float64
}

func fontTemplates() []template {
	var out []template
	for k := 0; k+200 <= len(digitFont); k += 200 {
		m := make([]float64, glyphSide*glyphSide)
		for i := range m {
			b := digitFont[k+i/2]
			if i%2 == 0 {
				b >>= 4
			}
			m[i] = float64(b & 15)
		}
		out = append(out, template{(k / 200) % 10, unitVector(m)})
	}
	return out
}

// readFigure gives the figure read, or -1: the best correlation reaches
// figureScore, leads every other figure by figureMargin, and the figure's
// holes agree (a 6 holed low, a 9 high).
func readFigure(ts []template, u []float64, holes int, holeY float64) int {
	var best [10]float64
	for i := range best {
		best[i] = math.Inf(-1)
	}
	for _, t := range ts {
		s := 0.0
		for i, x := range t.v {
			s += x * u[i]
		}
		if s > best[t.digit] {
			best[t.digit] = s
		}
	}
	d1, s1, s2 := -1, math.Inf(-1), -1.0
	for d := 9; d >= 0; d-- {
		if best[d] > s1 {
			d1, s1 = d, best[d]
		}
	}
	for d := 9; d >= 0; d-- {
		if d != d1 && !math.IsInf(best[d], -1) && best[d] > s2 {
			s2 = best[d]
		}
	}
	if d1 < 0 || s1 < figureScore || s1-s2 < figureMargin {
		return -1
	}
	okHoles := false
	for _, h := range holesOf[d1] {
		if h == holes {
			okHoles = true
		}
	}
	if !okHoles {
		return -1
	}
	if (d1 == 6 || d1 == 9) && holeY >= 0 && (holeY > 0.5) != (d1 == 6) {
		return -1
	}
	return d1
}

// numberingRow is the candidate numbering above (top) or below the board:
// numbers outside the field, across the board's width, of one height on one
// line, left to right.
func numberingRow(nums []*printedNumber, top bool) []*printedNumber {
	var c []*printedNumber
	for _, x := range nums {
		if -0.05 <= x.rel[0] && x.rel[0] <= 1.05 && ((top && x.rel[1] < 0.12) || (!top && x.rel[1] > 0.88)) {
			c = append(c, x)
		}
	}
	if len(c) < 12 {
		return nil
	}
	var best []*printedNumber
	for _, x := range c {
		var r []*printedNumber
		h := float64(x.h)
		for _, y := range c {
			if math.Abs(y.cy()-x.cy()) <= 0.35*h && 0.75*h <= float64(y.h) && float64(y.h) <= 1.33*h {
				r = append(r, y)
			}
		}
		if best == nil || len(r) > len(best) {
			best = r
		}
	}
	sort.SliceStable(best, func(a, b int) bool { return best[a].x0 < best[b].x0 })
	return best
}

type labelled struct {
	n   *printedNumber
	val int
}

// labelNumbering names the figures of a diagram's numbering by its shape
// alone, reading nothing: the row of point 1 holds nine one-figure numbers
// at one end and three of two figures at the other (1-9, 10-12), the other
// row twelve of two figures (25 less the number at the same column). A
// thirteenth number in a row, the one nearest the bar, is dropped.
func labelNumbering(nums []*printedNumber) []labelled {
	rows := [2][]*printedNumber{numberingRow(nums, true), numberingRow(nums, false)}
	if rows[0] == nil || rows[1] == nil {
		return nil
	}
	var pats [2][]int
	for k, r := range rows {
		if len(r) == 13 {
			mid := 0
			for i := range r {
				if math.Abs(r[i].rel[0]-0.5) < math.Abs(r[mid].rel[0]-0.5) {
					mid = i
				}
			}
			r = append(append([]*printedNumber(nil), r[:mid]...), r[mid+1:]...)
		}
		if len(r) != 12 {
			return nil
		}
		rows[k] = r
		for _, x := range r {
			pats[k] = append(pats[k], len(x.glyphs))
		}
	}
	isSingles := func(p []int) bool {
		a, b := true, true
		for i, k := range p {
			a = a && k == map[bool]int{true: 1, false: 2}[i < 9]
			b = b && k == map[bool]int{true: 2, false: 1}[i < 3]
		}
		return a || b
	}
	allTwo := func(p []int) bool {
		for _, k := range p {
			if k != 2 {
				return false
			}
		}
		return true
	}
	var singles []int
	for k := range pats {
		if isSingles(pats[k]) {
			singles = append(singles, k)
		}
	}
	if len(singles) != 1 || !allTwo(pats[1-singles[0]]) {
		return nil
	}
	s := singles[0]
	var out []labelled
	for i, x := range rows[s] {
		v := i + 1
		if pats[s][0] != 1 {
			v = 12 - i
		}
		out = append(out, labelled{x, v})
	}
	for i, x := range rows[1-s] {
		out = append(out, labelled{x, 25 - out[i].val})
	}
	return out
}

func figuresOf(v int) []int {
	if v < 10 {
		return []int{v}
	}
	return append(figuresOf(v/10), v%10)
}

func glyphFloats(q *[glyphSide * glyphSide]uint8, scale float64) []float64 {
	m := make([]float64, len(q))
	for i, x := range q {
		m[i] = float64(x) * scale
	}
	return m
}

// readNumbers reads every number of the document, setting val. Templates:
// the document's numbering labelled by its shape, the embedded typefaces,
// then, over three rounds, the figures of the numbers read entirely outside
// the numbering, by size class. No pip count takes part.
func readNumbers(all [][]printedNumber) {
	var nums [][]*printedNumber
	for i := range all {
		var ps []*printedNumber
		for j := range all[i] {
			ps = append(ps, &all[i][j])
		}
		nums = append(nums, ps)
	}
	type unit struct {
		u     []float64
		holes int
		holeY float64
	}
	cache := map[*[glyphSide * glyphSide]uint8]unit{}
	get := func(q *[glyphSide * glyphSide]uint8) unit {
		if u, ok := cache[q]; ok {
			return u
		}
		m := glyphFloats(q, 1)
		h, hy := squareHoles(m)
		for i := range m {
			m[i] /= 255
		}
		u := unit{unitVector(m), h, hy}
		cache[q] = u
		return u
	}
	samples := map[int][][]float64{}
	var order []int
	numbered := map[*printedNumber]bool{}
	var heights []float64
	for _, ps := range nums {
		for _, l := range labelNumbering(ps) {
			numbered[l.n] = true
			heights = append(heights, float64(l.n.h))
			for k, d := range figuresOf(l.val) {
				if k >= len(l.n.glyphs) {
					break
				}
				if _, ok := samples[d]; !ok {
					order = append(order, d)
				}
				samples[d] = append(samples[d], get(&l.n.glyphs[k]).u)
			}
		}
	}
	var base []template
	for _, d := range order {
		if len(samples[d]) < 5 {
			continue
		}
		base = append(base, template{d, unitVector(meanOf(samples[d]))})
	}
	base = append(base, fontTemplates()...)
	hnum := 1.0
	if len(heights) > 0 {
		hnum = median(heights)
	}
	type key struct {
		d   int
		big bool
	}
	var auto []template
	for round := 0; round < 3; round++ {
		ts := append(append([]template(nil), base...), auto...)
		acc := map[key][][]float64{}
		var keys []key
		for _, ps := range nums {
			for _, x := range ps {
				x.val = -1
				read := make([]int, len(x.glyphs))
				all := true
				for k := range x.glyphs {
					u := get(&x.glyphs[k])
					read[k] = readFigure(ts, u.u, u.holes, u.holeY)
					all = all && read[k] >= 0
				}
				if !x.alone || x.h < figureMinPx || x.cut || !all || (len(read) > 1 && read[0] == 0) {
					continue
				}
				v := 0
				for _, d := range read {
					v = v*10 + d
				}
				x.val = v
				if numbered[x] {
					continue
				}
				for k, d := range read {
					kk := key{d, float64(x.h) > 1.25*hnum}
					if _, ok := acc[kk]; !ok {
						keys = append(keys, kk)
					}
					acc[kk] = append(acc[kk], get(&x.glyphs[k]).u)
				}
			}
		}
		auto = auto[:0:0]
		for _, k := range keys {
			if len(acc[k]) >= 3 {
				auto = append(auto, template{k.d, centred(meanOf(acc[k]))})
			}
		}
	}
}

func meanOf(vs [][]float64) []float64 {
	out := make([]float64, len(vs[0]))
	for _, v := range vs {
		for i, x := range v {
			out[i] += x
		}
	}
	for i := range out {
		out[i] /= float64(len(vs))
	}
	return out
}

// pipSlot is a place, relative to a diagram's box, where the document prints
// a pip count.
type pipSlot struct{ x, y float64 }

// learnPipSlots finds where the document prints its pip counts, by a vote
// over its diagrams read with fifteen checkers a side and no doubt: a number
// read that equals one of the two pip counts of the checkers votes for its
// place (cells of 0.04). Two places, top and bottom, each with at least ten
// votes and 30 % of the voting diagrams, or none. The pips only say where to
// look, over the whole document: no figure is chosen after them, and a
// diagram's own pips never decide what is read beside it.
func learnPipSlots(nums [][]printedNumber, ds []Diagram) []pipSlot {
	const q = 0.04
	type cell struct{ i, j int }
	votes := map[cell]int{}
	var order []cell
	pos := map[cell][][2]float64{}
	voters := 0
	for i, d := range ds {
		if !d.Complete || d.Pips[0] == d.Pips[1] {
			continue
		}
		hit := false
		for _, x := range nums[i] {
			if x.val >= 0 && (x.val == d.Pips[0] || x.val == d.Pips[1]) {
				k := cell{roundHalfEven(x.rel[0] / q), roundHalfEven(x.rel[1] / q)}
				if _, ok := votes[k]; !ok {
					order = append(order, k)
				}
				votes[k]++
				pos[k] = append(pos[k], x.rel)
				hit = true
			}
		}
		if hit {
			voters++
		}
	}
	smooth := map[cell]int{}
	var sorder []cell
	for _, k := range order {
		for di := -1; di <= 1; di++ {
			for dj := -1; dj <= 1; dj++ {
				c := cell{k.i + di, k.j + dj}
				if _, ok := smooth[c]; !ok {
					sorder = append(sorder, c)
				}
				smooth[c] += votes[k]
			}
		}
	}
	if len(sorder) == 0 {
		return nil
	}
	argmax := func(cs []cell) cell {
		b := cs[0]
		for _, c := range cs {
			if smooth[c] > smooth[b] {
				b = c
			}
		}
		return b
	}
	k1 := argmax(sorder)
	var other []cell
	for _, c := range sorder {
		if math.Abs(float64(c.j-k1.j))*q >= 0.3 {
			other = append(other, c)
		}
	}
	if len(other) == 0 {
		return nil
	}
	var slots []pipSlot
	for _, k := range []cell{k1, argmax(other)} {
		if float64(smooth[k]) < max(10, 0.3*float64(voters)) {
			continue
		}
		var xs, ys []float64
		for _, c := range order {
			if abs(c.i-k.i) <= 1 && abs(c.j-k.j) <= 1 {
				for _, p := range pos[c] {
					xs, ys = append(xs, p[0]), append(ys, p[1])
				}
			}
		}
		slots = append(slots, pipSlot{median(xs), median(ys)})
	}
	if len(slots) != 2 {
		return nil
	}
	if slots[1].y < slots[0].y {
		slots[0], slots[1] = slots[1], slots[0]
	}
	return slots
}

// imagePips reads the pip counts printed at the document's two places
// beside a diagram (top then bottom): the one number within 1.5 figure
// heights across and 1.0 along of each place, read entirely; nil when a
// place holds none, more than one, or a number not read.
func imagePips(nums []printedNumber, b box, slots []pipSlot) *[2]int {
	if len(slots) != 2 {
		return nil
	}
	X0, Y0 := float64(b.x0), float64(b.y0)
	W, H := float64(b.w()), float64(b.h())
	var out [2]int
	for k, s := range slots {
		cx, cy := X0+s.x*W, Y0+s.y*H
		found := -1
		count := 0
		for _, x := range nums {
			if math.Abs(x.cx()-cx) <= 1.5*float64(x.h) && math.Abs(x.cy()-cy) <= float64(x.h) {
				count++
				found = x.val
			}
		}
		if count != 1 || found < 0 {
			return nil
		}
		out[k] = found
	}
	return &out
}
