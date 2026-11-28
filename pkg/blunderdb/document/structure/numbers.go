package structure

import (
	_ "embed"
	"math"
	"slices"
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
	boxed          bool // centred in a square of a cube's size
	lettered       bool // alone but for one letter before it (prefixOf)
	farOut         bool // in a far band beside the board (extractNumbers)
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

// numbersNear and numbersFar are how far beside a diagram, in point
// widths, its numbers are looked for: near it, where its figures are
// learned, then on the board's horizontal band out to about half a board's
// width, where some documents print their pip counts.
const numbersNear, numbersFar = 5.0, 10.0

// extractNumbers finds the numbers printed around the diagram whose box is b
// on the normalised page g: in the box widened by numbersNear point widths
// across and 1.8 along, then on either side out to numbersFar. A number of
// the far bands is read, never learned from (farOut): the figures a
// document is learned on stay those beside its diagrams.
func extractNumbers(g *raster, b box) []printedNumber {
	X0, X1 := float64(b.x0), float64(b.x1)
	p := (X1 - X0) / 13
	near, _ := numbersIn(g, b, X0-numbersNear*p, X1+numbersNear*p)
	// A number belongs to the near zone or to a far band by its centre,
	// half a point inside the near zone's edge; the bands reach further
	// in, so that a number at the edge is whole in its band with the letter
	// that may stand before it.
	in0, in1 := X0-(numbersNear-0.5)*p, X1+(numbersNear-0.5)*p
	var far []printedNumber
	var letters []box
	for _, side := range [2][2]float64{{X0 - numbersFar*p, in0 + 2.5*p}, {in1 - 2.5*p, X1 + numbersFar*p}} {
		ns, ls := numbersIn(g, b, side[0], side[1])
		letters = append(letters, ls...)
		for _, n := range ns {
			if !n.cut && (n.cx() < in0 || n.cx() >= in1) {
				n.farOut = true
				far = append(far, n)
			}
		}
	}
	out := near[:0:0]
	for _, n := range near {
		keep := n.cx() >= in0 && n.cx() < in1
		for _, l := range letters {
			if n.x0 < l.x1 && l.x0 < n.x1 && n.y0 < l.y1 && l.y0 < n.y1 {
				keep = false // a band's letter, seen from the near zone
			}
		}
		if keep {
			out = append(out, n)
		}
	}
	return append(out, far...)
}

// numbersIn finds the numbers in the box b widened to [wx0,wx1) across and
// 1.8 point widths along: components of ink of a figure's size, grouped
// into numbers of one to three figures of one height on one baseline.
// It also gives the boxes of the letters written before numbers (prefixOf),
// which are no numbers.
func numbersIn(g *raster, b box, wx0, wx1 float64) ([]printedNumber, []box) {
	X0, Y0, X1, Y1 := float64(b.x0), float64(b.y0), float64(b.x1), float64(b.y1)
	W := X1 - X0
	p := W / 13
	rx0, rx1 := max(0, trunc(wx0)), min(g.w, trunc(wx1))
	ry0, ry1 := max(0, trunc(Y0-1.8*p)), min(g.h, trunc(Y1+1.8*p))
	if rx1-rx0 < 4 || ry1-ry0 < 4 {
		return nil, nil
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
	var figs [][]int
	letters := map[int]bool{}
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
		in := map[int32]bool{}
		for _, i := range grp.figs {
			in[int32(i+1)] = true
		}
		alone = alone && bandClear(lab, comps, sub.w, sub.h, in, gx0, gy0, gx1, gy1)
		// A letter naming the side before a count ("W 164", "B 123") does
		// not keep it from standing alone when the pair stands alone on
		// its line, clear of other ink by 1.5 height: one capital of the
		// figures' height, within one height of their left. The letter is
		// then no number of its own, whatever figure it resembles (a B
		// reads as an 8). A count so lettered is read, never learned from.
		if k := prefixOf(comps, in, gx0, gy0, gx1, gy1); k >= 0 {
			c := comps[k]
			in2 := map[int32]bool{int32(k + 1): true}
			ink2 := map[int]bool{k: true}
			for _, i := range grp.figs {
				in2[int32(i+1)] = true
				ink2[i] = true
			}
			if lineClear(comps, ink2, c.x0, gy0, gx1, gy1, 1.5) && bandClear(lab, comps, sub.w, sub.h, in2, c.x0, gy0, gx1, gy1) {
				n.lettered = !alone
				letters[k] = true
			}
		}
		n.boxed = boxedBy(ink, gx0, gy0, gx1, gy1, p)
		n.x0, n.y0, n.x1, n.y1 = gx0+rx0, gy0+ry0, gx1+rx0, gy1+ry0
		n.h, n.alone = n.y1-n.y0, alone
		n.rel = [2]float64{round4((n.cx() - X0) / W), round4((n.cy() - Y0) / (Y1 - Y0))}
		m := 0.6 * float64(n.h)
		n.cut = float64(n.x0)-wx0 < m || wx1-float64(n.x1) < m || float64(n.y0)-(Y0-1.8*p) < m || (Y1+1.8*p)-float64(n.y1) < m
		out = append(out, n)
		figs = append(figs, grp.figs)
	}
	kept := out[:0]
	var lbs []box
	for k := range letters {
		c := comps[k]
		lbs = append(lbs, box{c.x0 + rx0, c.y0 + ry0, c.x1 + rx0, c.y1 + ry0})
	}
	for i, n := range out {
		if len(figs[i]) != 1 || !letters[figs[i][0]] {
			kept = append(kept, n)
		}
	}
	return kept, lbs
}

// boxedBy tells whether the figures in [x0,x1)×[y0,y1) are centred in a
// square whose side is 0.45 to 1.4 point (p): the first stroke met going out
// from the figures, left, right, up and down, across their middle half.
func boxedBy(ink *mask, x0, y0, x1, y1 int, p float64) bool {
	reach := trunc(0.9 * p)
	hit := func(x, y, dx, dy int) int {
		for d := 1; d <= reach; d++ {
			xx, yy := x+d*dx, y+d*dy
			if xx < 0 || yy < 0 || xx >= ink.w || yy >= ink.h {
				return -1
			}
			if ink.at(xx, yy) {
				return d
			}
		}
		return -1
	}
	// Each side needs the stroke along most of the band it crosses.
	side := func(dx, dy int) int {
		var ds []float64
		if dx != 0 {
			x := x0 - 1
			if dx > 0 {
				x = x1
			}
			for y := y0 + (y1-y0)/4; y < y1-(y1-y0)/4; y++ {
				if d := hit(x, y, dx, 0); d > 0 {
					ds = append(ds, float64(d))
				}
			}
			if 3*len(ds) < 2*max(1, (y1-y0)/2) {
				return -1
			}
		} else {
			y := y0 - 1
			if dy > 0 {
				y = y1
			}
			for x := x0 + (x1-x0)/4; x < x1-(x1-x0)/4; x++ {
				if d := hit(x, y, 0, dy); d > 0 {
					ds = append(ds, float64(d))
				}
			}
			if 3*len(ds) < 2*max(1, (x1-x0)/2) {
				return -1
			}
		}
		return trunc(median(ds))
	}
	l, r, u, d := side(-1, 0), side(1, 0), side(0, -1), side(0, 1)
	if l < 0 || r < 0 || u < 0 || d < 0 {
		return false
	}
	w, h := float64(x1-x0+l+r), float64(y1-y0+u+d)
	if w < 0.45*p || w > 1.4*p || h < 0.45*p || h > 1.4*p || w/h < 0.75 || w/h > 1.33 {
		return false
	}
	// Centred: the margins on either side within a third of the side.
	return math.Abs(float64(l-r)) <= w/3 && math.Abs(float64(u-d)) <= h/3
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
		alone := lineClear(comps, in, x0, y0, x1, y1, 0.6)
		out = append(out, figureGroup{grp, alone})
	}
	return out
}

// lineClear tells whether no letter-sized component lies on the line of
// [x0,x1)×[y0,y1) within reach of its height, the components in aside.
func lineClear(comps []box, in map[int]bool, x0, y0, x1, y1 int, reach float64) bool {
	h := float64(y1 - y0)
	for k, c := range comps {
		if in[k] {
			continue
		}
		ov := float64(min(c.y1, y1) - max(c.y0, y0))
		ch := float64(c.h())
		if ov < 0.5*h || !(0.25*h <= ch && ch <= 1.6*h) {
			continue // a frame's stroke is no letter
		}
		if float64(max(c.x0-x1, x0-c.x1)) < reach*h {
			return false
		}
	}
	return true
}

// bandClear tells whether the ink beside [x0,x1)×[y0,y1) on its line,
// whatever its component (a figure joined to the frame belongs to a large
// one), is at most a long vertical stroke passing by (frame, bar), a dot or
// a colon, the components in aside; and whether the zone's edge is not
// within 0.6 height.
func bandClear(lab []int32, comps []box, w, h int, in map[int32]bool, x0, y0, x1, y1 int) bool {
	hh := float64(y1 - y0)
	if float64(min(y0, x0, h-y1, w-x1)) < 0.6*hh {
		return false
	}
	ya, yb := trunc(float64(y0)+0.2*hh), trunc(float64(y1)-0.2*hh)+1
	for _, band := range [2][2]int{{max(0, trunc(float64(x0)-0.6*hh)), x0}, {x1, min(w, trunc(float64(x1)+0.6*hh)+1)}} {
		xa, xb := band[0], band[1]
		seen := map[int32]bool{}
		for y := max(ya, 0); y < min(yb, h); y++ {
			for x := xa; x < xb; x++ {
				l := lab[y*w+x]
				if l == 0 || in[l] || seen[l] {
					continue
				}
				seen[l] = true
				c := comps[l-1]
				if float64(c.h()) < 0.3*hh && float64(c.w()) < 0.3*hh {
					continue
				}
				if !passingLine(lab, w, h, l, xa, xb, ya, yb, y0, y1, hh) {
					return false
				}
			}
		}
	}
	return true
}

// prefixOf is the component that may be a letter written before the number
// in [x0,x1)×[y0,y1), or -1: a capital of the figures' height and baseline,
// no wider than 1.6 height, ending within one height of their left.
func prefixOf(comps []box, in map[int32]bool, x0, y0, x1, y1 int) int {
	h := float64(y1 - y0)
	best := -1
	for k, c := range comps {
		if in[int32(k+1)] || c.x1 > x0 {
			continue
		}
		ch := float64(c.h())
		if ch < 0.75*h || ch > 1.35*h || float64(c.w()) > 1.6*h {
			continue
		}
		if math.Abs(float64(c.y1-y1)) > 0.2*h || math.Abs(float64(c.y0-y0)) > 0.3*h {
			continue
		}
		if float64(x0-c.x1) > h {
			continue
		}
		if best < 0 || c.x1 > comps[best].x1 {
			best = k
		}
	}
	return best
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
	// learned marks a template drawn from the document's own figures, as
	// against the embedded typefaces.
	learned bool
	weight  float64 // its stroke's thickness over its height (strokeWeight)
}

// strokeWeight is a figure's stroke thickness over its height, on its
// normalised square: twice its ink's area over its outline's length, the
// ink being the levels at least half of full.
func strokeWeight(m []float64, full float64) float64 {
	in := func(x, y int) bool {
		return x >= 0 && y >= 0 && x < glyphSide && y < glyphSide && m[y*glyphSide+x] >= full/2
	}
	area, outline, y0, y1 := 0, 0, glyphSide, 0
	for y := 0; y < glyphSide; y++ {
		for x := 0; x < glyphSide; x++ {
			if !in(x, y) {
				continue
			}
			area++
			y0, y1 = min(y0, y), max(y1, y+1)
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				if !in(x+d[0], y+d[1]) {
					outline++
				}
			}
		}
	}
	if area == 0 {
		return 0
	}
	return 2 * float64(area) / float64(outline) / float64(y1-y0)
}

// likeWeight tells whether two strokes are of one weight, within a third.
func likeWeight(a, b float64) bool { return max(a, b) <= 1.35*min(a, b) }

func fontTemplates() []template { return templatesOf(digitFont) }

// templatesOf reads figures stored as digitFont is.
func templatesOf(font []byte) []template {
	var out []template
	for k := 0; k+200 <= len(font); k += 200 {
		m := make([]float64, glyphSide*glyphSide)
		for i := range m {
			b := font[k+i/2]
			if i%2 == 0 {
				b >>= 4
			}
			m[i] = float64(b & 15)
		}
		out = append(out, template{digit: (k / 200) % 10, v: unitVector(m), weight: strokeWeight(m, 9)})
	}
	return out
}

// figureScores is, for each figure, the best correlation of u with its
// templates (u and the templates centred and of unit length), -Inf for a
// figure no template draws. Pips and captions measure alike by it.
func figureScores(ts []template, u []float64) [10]float64 {
	var best [10]float64
	for i := range best {
		best[i] = math.Inf(-1)
	}
	for _, t := range ts {
		s := 0.0
		for i, x := range t.v {
			s += x * u[i]
		}
		best[t.digit] = max(best[t.digit], s)
	}
	return best
}

// bestFigure is the figure of the highest score, or -1 when it falls below
// least or another figure comes within margin of it.
func bestFigure(best [10]float64, least, margin float64) int {
	d1 := -1
	for d := range best {
		if !math.IsInf(best[d], -1) && (d1 < 0 || best[d] > best[d1]) {
			d1 = d
		}
	}
	if d1 < 0 || best[d1] < least {
		return -1
	}
	for d := range best {
		if d != d1 && best[d] > best[d1]-margin {
			return -1
		}
	}
	return d1
}

// readFigure gives the figure read, or -1: the best correlation reaches
// figureScore, leads every other figure by figureMargin, is a fair win
// (fairWin), and the figure's holes agree (a 6 holed low, a 9 high).
func readFigure(ts []template, u []float64, holes int, holeY float64) int {
	d1 := bestFigure(figureScores(ts, u), figureScore, figureMargin)
	if d1 < 0 || !fairWin(ts, u, d1) || !slices.Contains(holesOf[d1], holes) {
		return -1
	}
	if (d1 == 6 || d1 == 9) && holeY >= 0 && (holeY > 0.5) != (d1 == 6) {
		return -1
	}
	return d1
}

// fairWin tells whether d, read from the templates ts, did not win only
// because its best template, drawn from the document, carries a weight its
// rivals' do not — a heavy 2 against the document's own heavy 1 and only
// light 2s, a typeface's or a numbering's. A rival the document taught no
// template of the winner's weight was never measured against it on equal
// terms: the embedded typefaces alone must then rank d ahead of it by
// figureMargin, else the reading is a doubt. Only a learned template
// excuses a rival: the typefaces' bold figures stand in every document, so
// a typeface 2 of the winner's weight proves nothing about how this
// document draws its 2s, and a numbering's light 2 nothing about its heavy
// ones — so the guard holds when the numbering taught all ten figures.
func fairWin(ts []template, u []float64, d int) bool {
	var win *template
	best := math.Inf(-1)
	for i := range ts {
		if ts[i].digit != d {
			continue
		}
		if s := dot(ts[i].v, u); s > best {
			best, win = s, &ts[i]
		}
	}
	if win == nil || !win.learned {
		return true
	}
	var like [10]bool
	var fonts []template
	for _, t := range ts {
		if t.learned {
			like[t.digit] = like[t.digit] || likeWeight(t.weight, win.weight)
		} else {
			fonts = append(fonts, t)
		}
	}
	fs := figureScores(fonts, u)
	for e := range fs {
		if e != d && !like[e] && fs[e] > fs[d]-figureMargin {
			return false
		}
	}
	return true
}

func dot(a, b []float64) float64 {
	s := 0.0
	for i, x := range a {
		s += x * b[i]
	}
	return s
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
// readNumbers reads the numbers printed around the diagrams and gives the
// document's figures as it learned them, to read other figures by.
func readNumbers(all [][]printedNumber) []template {
	var nums [][]*printedNumber
	for i := range all {
		var ps []*printedNumber
		for j := range all[i] {
			ps = append(ps, &all[i][j])
		}
		nums = append(nums, ps)
	}
	type unit struct {
		u      []float64
		holes  int
		holeY  float64
		weight float64
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
		u := unit{unitVector(m), h, hy, strokeWeight(m, 1)}
		cache[q] = u
		return u
	}
	samples, weights := map[int][][]float64{}, map[int][]float64{}
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
				u := get(&l.n.glyphs[k])
				samples[d] = append(samples[d], u.u)
				weights[d] = append(weights[d], u.weight)
			}
		}
	}
	var base []template
	for _, d := range order {
		if len(samples[d]) < 5 {
			continue
		}
		base = append(base, template{digit: d, v: unitVector(meanOf(samples[d])), learned: true, weight: median(weights[d])})
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
		acc, accW := map[key][][]float64{}, map[key][]float64{}
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
				// A number in a cube's square is not alone, its square
				// standing round it, nor one after a letter: it is read,
				// never learned from.
				if !(x.alone || x.boxed || x.lettered) || x.h < figureMinPx || x.cut || !all || (len(read) > 1 && read[0] == 0) {
					continue
				}
				v := 0
				for _, d := range read {
					v = v*10 + d
				}
				x.val = v
				if numbered[x] || !x.alone || x.farOut {
					continue
				}
				for k, d := range read {
					kk := key{d, float64(x.h) > 1.25*hnum}
					if _, ok := acc[kk]; !ok {
						keys = append(keys, kk)
					}
					u := get(&x.glyphs[k])
					acc[kk] = append(acc[kk], u.u)
					accW[kk] = append(accW[kk], u.weight)
				}
			}
		}
		auto = auto[:0:0]
		for _, k := range keys {
			if len(acc[k]) >= 3 {
				auto = append(auto, template{digit: k.d, v: centred(meanOf(acc[k])), learned: true, weight: median(accW[k])})
			}
		}
	}
	return append(append([]template(nil), base...), auto...)
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
// a pip count, and the span [x0,x1]×[y0,y1] its counts were seen over.
type pipSlot struct{ x, y, x0, x1, y0, y1 float64 }

// learnPipSlots finds where the document prints its pip counts, by a vote
// over its diagrams read with fifteen checkers a side and no doubt, those
// counted in a tray included (Diagram.pipVoter): a number
// read that equals one of the two pip counts of the checkers votes for its
// place (cells of 0.04). Two places, top and bottom, each with at least ten
// votes and 30 % of the voting diagrams, or none; it also returns how many
// diagrams voted. The pips only say where to
// look, over the whole document: no figure is chosen after them, and a
// diagram's own pips never decide what is read beside it.
func learnPipSlots(nums [][]printedNumber, ds []Diagram) ([]pipSlot, int) {
	const q = 0.04
	type cell struct{ i, j int }
	votes := map[cell]int{}
	var order []cell
	pos := map[cell][][2]float64{}
	voters := 0
	for i, d := range ds {
		if !d.pipVoter || d.Pips[0] == d.Pips[1] {
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
		return nil, 0
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
		return nil, 0
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
		// A document that draws its boards at varying places and sizes
		// prints its counts at varying places beside them: the place
		// spreads over the cells voted for at least twice, each within two
		// cells of it, on its own half of the board.
		in := map[cell]bool{}
		for _, c := range order {
			if abs(c.i-k.i) <= 1 && abs(c.j-k.j) <= 1 {
				in[c] = true
			}
		}
		for grown := true; grown; {
			grown = false
			for _, c := range order {
				if in[c] || votes[c] < 2 || (float64(c.j)*q < 0.5) != (float64(k.j)*q < 0.5) {
					continue
				}
				for d := range in {
					if abs(c.i-d.i) <= 2 && abs(c.j-d.j) <= 2 {
						in[c], grown = true, true
						break
					}
				}
			}
		}
		x0, x1, y0, y1 := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
		for _, c := range order {
			if in[c] {
				for _, p := range pos[c] {
					x0, x1, y0, y1 = min(x0, p[0]), max(x1, p[0]), min(y0, p[1]), max(y1, p[1])
				}
			}
		}
		slots = append(slots, pipSlot{median(xs), median(ys), x0, x1, y0, y1})
	}
	if len(slots) != 2 {
		return nil, 0
	}
	if slots[1].y < slots[0].y {
		slots[0], slots[1] = slots[1], slots[0]
	}
	return slots, voters
}

// imagePips reads the pip counts printed at the document's two places
// beside a diagram (top then bottom): the one number within 1.5 figure
// heights across and 1.0 along of each place, or else of its span, read
// entirely; nil when a place holds none, more than one, or a number not
// read.
func imagePips(nums []printedNumber, b box, slots []pipSlot) *[2]int {
	if len(slots) != 2 {
		return nil
	}
	X0, Y0 := float64(b.x0), float64(b.y0)
	W, H := float64(b.w()), float64(b.h())
	var out [2]int
	for k, s := range slots {
		// The place itself first; its span only when nothing stands there,
		// so that a span never hides a count read at the place.
		at := func(x0, x1, y0, y1 float64) (int, int) {
			found, count := -1, 0
			for _, x := range nums {
				h := float64(x.h)
				if x.cx() >= X0+x0*W-1.5*h && x.cx() <= X0+x1*W+1.5*h && x.cy() >= Y0+y0*H-h && x.cy() <= Y0+y1*H+h {
					count++
					found = x.val
				}
			}
			return found, count
		}
		found, count := at(s.x, s.x, s.y, s.y)
		if count == 0 {
			found, count = at(s.x0, s.x1, s.y0, s.y1)
		}
		if count != 1 || found < 0 {
			return nil
		}
		out[k] = found
	}
	return &out
}
