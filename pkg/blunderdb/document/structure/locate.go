package structure

import (
	"math"
	"slices"
	"sort"
)

func sortFloats(s []float64) { sort.Float64s(s) }

// frame is the playing field of a straightened diagram: the inner left and
// right edges of each half (a..b, c..d) and the field's top and bottom
// (y0..y1), in the pixels of its zone.
type frame struct{ a, b, c, d, y0, y1 int }

// pointWidth is the width of one point: twelve points over the two halves.
func (f frame) pointWidth() float64 { return float64((f.b-f.a)+(f.d-f.c)) / 12 }

func (f frame) shift(dx, dy int) frame {
	return frame{f.a + dx, f.b + dx, f.c + dx, f.d + dx, f.y0 + dy, f.y1 + dy}
}

// fieldRows bounds the field: the innermost horizontal stroke spanning three
// quarters of the zone in its top sixth, and in its bottom sixth. A stroke cut
// by the scan falls back on the first band inked at 60 %, an erased frame on
// the first row of ink.
func fieldRows(ink *mask) (int, int) {
	H, W := ink.h, ink.w
	v := ink.thicken(true)
	run := longestRuns(v, false)
	top, bot := -1, -1
	for y, r := range run {
		if float64(r) >= 0.75*float64(W) {
			if float64(y) < float64(H)/6 {
				top = y
			}
			if float64(y) > 5*float64(H)/6 && bot < 0 {
				bot = y
			}
		}
	}
	frac := make([]float64, H)
	first, last := -1, -1
	for y := 0; y < H; y++ {
		n := 0
		any := false
		for x := 0; x < W; x++ {
			if v.p[y*W+x] {
				n++
			}
			if ink.p[y*W+x] {
				any = true
			}
		}
		frac[y] = float64(n) / float64(W)
		if any {
			if first < 0 {
				first = y
			}
			last = y
		}
	}
	firstBand := func(from, to, step int) []int {
		var band []int
		for y := from; y != to; y += step {
			if frac[y] >= 0.6 {
				band = append(band, y)
			} else if len(band) > 0 {
				break
			}
		}
		return band
	}
	var y0, y1 int
	if top >= 0 {
		y0 = top
	} else if b := firstBand(0, trunc(float64(H)/6), 1); len(b) > 0 {
		y0 = b[len(b)-1] + 1
	} else {
		y0 = first
	}
	if bot >= 0 {
		y1 = bot + 1
	} else if b := firstBand(H-1, trunc(5*float64(H)/6), -1); len(b) > 0 {
		y1 = b[len(b)-1]
	} else {
		y1 = last + 1
	}
	return y0, y1
}

type line struct{ lo, hi int }

// verticalLines are the vertical strokes of the field: a run of ink over
// 85 % of its height. When the frame is erased the zone's edges stand for it.
func verticalLines(ink *mask, y0, y1 int) []line {
	f := ink.sub(0, y0, ink.w, y1).thicken(false)
	run := longestRuns(f, true)
	var cand []int
	for x, r := range run {
		if float64(r) >= 0.85*float64(y1-y0) {
			cand = append(cand, x)
		}
	}
	var out []line
	for i := 0; i < len(cand); {
		j := i
		for j+1 < len(cand) && cand[j+1]-cand[j] <= 2 {
			j++
		}
		out = append(out, line{cand[i] + 1, cand[j] - 1})
		i = j + 1
	}
	W := ink.w
	if len(out) == 0 || float64(out[0].lo) > 0.02*float64(W) {
		out = append([]line{{-1, -1}}, out...)
	}
	if float64(out[len(out)-1].hi) < 0.98*float64(W) {
		out = append(out, line{W, W})
	}
	return out
}

// findFrame finds two halves of equal width (3 %) separated by a bar narrower
// than a third of a half, with no full stroke inside either half; the widest
// such pair wins.
func findFrame(ink *mask) (frame, bool) {
	H, W := ink.h, ink.w
	y0, y1 := fieldRows(ink)
	if float64(y1-y0) < 0.5*float64(H) {
		return frame{}, false
	}
	lines := verticalLines(ink, y0, y1)
	n := len(lines)
	best, found := frame{}, false
	bestW := 0
	inside := func(lo, hi int) bool {
		for _, l := range lines {
			if lo+2 < l.lo && l.hi < hi-2 {
				return true
			}
		}
		return false
	}
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			for k := j; k < n; k++ {
				for m := k + 1; m < n; m++ {
					a, b, c, d := lines[i].hi+1, lines[j].lo, lines[k].hi+1, lines[m].lo
					w1, w2, bar := b-a, d-c, c-b
					if float64(w1) < 0.25*float64(W) || math.Abs(float64(w1-w2)) > 0.03*float64(w1) || bar < 0 || float64(bar) >= 0.35*float64(w1) {
						continue
					}
					if inside(a, b) || inside(c, d) {
						continue
					}
					if !found || w1 > bestW {
						best, bestW, found = frame{a, b, c, d, y0, y1}, w1, true
					}
				}
			}
		}
	}
	return best, found
}

// pointEvidence counts, per row, the columns whose band at lo..hi of the
// field's height from the base looks like a point: inked at its centre, empty
// at its edges; a point drawn too pale for the ink threshold is looked for in
// the grey.
func pointEvidence(ink *mask, f frame, gray *raster, lo, hi float64) [2]int {
	p := f.pointWidth()
	var per [2]int
	h := float64(f.y1 - f.y0)
	for r, top := range []bool{true, false} {
		for j := 0; j < 12; j++ {
			x0 := float64(f.a) + float64(j%6)*p
			if j >= 6 {
				x0 = float64(f.c) + float64(j%6)*p
			}
			var ya, yb int
			if top {
				ya, yb = trunc(float64(f.y0)+lo*h), trunc(float64(f.y0)+hi*h)
			} else {
				ya, yb = trunc(float64(f.y1)-hi*h), trunc(float64(f.y1)-lo*h)
			}
			xa, xb := trunc(x0+0.08*p), trunc(x0+0.92*p)
			ya, yb = max(ya, 0), min(yb, ink.h)
			xa, xb = max(xa, 0), min(xb, ink.w)
			if ya >= yb || xa >= xb {
				continue
			}
			m := xb - xa
			prof := make([]float64, m)
			gp := make([]float64, m)
			for x := xa; x < xb; x++ {
				s, g := 0, 0.0
				for y := ya; y < yb; y++ {
					if ink.at(x, y) {
						s++
					}
					if gray != nil {
						g += float64(gray.at(x, y))
					}
				}
				prof[x-xa] = float64(s) / float64(yb-ya)
				gp[x-xa] = 1 - g/float64(yb-ya)
			}
			if edgeMean(prof) < 0.15 && slices.Max(prof) > 0.3 {
				per[r]++
			} else if gray != nil && edgeMean(gp) < 0.05 && slices.Max(gp) > 0.15 {
				per[r]++
			}
		}
	}
	return per
}

// edgeMean is the mean of the outer fifths of a profile (the whole profile
// when it is too short to have fifths).
func edgeMean(v []float64) float64 {
	k := len(v) / 5
	var e []float64
	if k == 0 {
		e = v
	} else {
		e = append(append([]float64(nil), v[:k]...), v[len(v)-k:]...)
	}
	s := 0.0
	for _, x := range e {
		s += x
	}
	return s / float64(len(e))
}

var pointBands = [2][2]float64{{0.28, 0.36}, {0.16, 0.24}}

// triangleEvidence is the share of the 24 columns that look like points,
// best of a band near the middle of a point and one near its base.
func triangleEvidence(ink *mask, f frame, gray *raster) float64 {
	best := 0.0
	for _, b := range pointBands {
		e := pointEvidence(ink, f, gray, b[0], b[1])
		best = max(best, float64(e[0]+e[1])/24)
	}
	return best
}

// rowEvidence is the number of points attested in each row (top, bottom): a
// row with none is a half board, one row printed alone.
func rowEvidence(ink *mask, f frame, gray *raster) [2]int {
	a := pointEvidence(ink, f, gray, pointBands[0][0], pointBands[0][1])
	b := pointEvidence(ink, f, gray, pointBands[1][0], pointBands[1][1])
	return [2]int{max(a[0], b[0]), max(a[1], b[1])}
}

// skew is the angle, in degrees, that makes the horizontal projection of the
// component's ink the most peaked (sum of squares of the sheared histogram),
// searched over ±3 degrees by quarters then by twentieths.
func skew(m *mask) float64 {
	var xs, ys []float64
	for y := 0; y < m.h; y++ {
		for x := 0; x < m.w; x++ {
			if m.p[y*m.w+x] {
				xs = append(xs, float64(x))
				ys = append(ys, float64(y))
			}
		}
	}
	if len(xs) < 50 {
		return 0
	}
	mean := 0.0
	for _, x := range xs {
		mean += x
	}
	mean /= float64(len(xs))
	for i := range xs {
		xs[i] -= mean
	}
	yp := make([]int, len(xs))
	score := func(a float64) float64 {
		t := math.Tan(a * math.Pi / 180)
		lo, hi := math.MaxInt, math.MinInt
		for i := range xs {
			yp[i] = roundHalfEven(ys[i] - xs[i]*t)
			lo, hi = min(lo, yp[i]), max(hi, yp[i])
		}
		hist := make([]int, hi-lo+1)
		for _, y := range yp {
			hist[y-lo]++
		}
		s := 0.0
		for _, c := range hist {
			s += float64(c) * float64(c)
		}
		return s
	}
	search := func(start, stop, step float64) float64 {
		best, bs := 0.0, -1.0
		n := int(math.Ceil((stop - start) / step))
		for i := 0; i < n; i++ {
			a := start + float64(i)*step
			if s := score(a); s > bs {
				best, bs = a, s
			}
		}
		return best
	}
	best := search(-3, 3.01, 0.25)
	return search(best-0.25, best+0.26, 0.05)
}

// zone is one candidate diagram of a page.
type zone struct {
	bbox   box // in the page's pixels, before straightening
	angle  float64
	reject string
	points float64
	frame  frame
	crop   *raster // straightened, tightly cropped, neighbours whitened
	ink    *mask
	digits [2][12]int
	cubes  []square // the cubes drawn
	dice   []square // the dice drawn
	trays  [2]int   // checkers in the trays of the top and bottom rows, 0 when none is read
	// field is the field's frame (a, b, c, d, y0, y1) in the page's pixels,
	// the straightening neglected: where a cube's figure or a word lies.
	field [6]float64
}

// locate finds the diagrams of a normalised page: large components of ink,
// straightened, framed and proven to carry points.
func locate(g *raster, dpi float64) []zone {
	ink := hysteresis(g)
	H, W := g.h, g.w
	d := ink.dilate(1)
	lab, boxes, _ := label(d)
	minSide := trunc(0.8 * dpi)
	type comp struct {
		ids []int32
		b   box
	}
	var comps []comp
	for i, b := range boxes {
		if b.h() >= minSide && b.w() >= minSide {
			comps = append(comps, comp{[]int32{int32(i + 1)}, b})
		}
	}
	// Two halves drawn each in its own frame, side by side (same height and
	// width, a bar's width apart at most), are one candidate.
	var tall []comp
	for _, c := range comps {
		if float64(c.b.w())/float64(c.b.h()) < 0.6 {
			tall = append(tall, c)
		}
	}
	used := map[int32]bool{}
	var pairs []comp
	for _, u := range tall {
		for _, v := range tall {
			if v.b.x0 <= u.b.x0 || used[u.ids[0]] || used[v.ids[0]] {
				continue
			}
			hu, hv, wu, wv := float64(u.b.h()), float64(v.b.h()), float64(u.b.w()), float64(v.b.w())
			gap := float64(v.b.x0 - u.b.x1)
			if math.Abs(hu-hv) <= 0.05*hu && math.Abs(float64(u.b.y0-v.b.y0)) <= 0.05*hu && math.Abs(wu-wv) <= 0.08*wu && gap >= 0 && gap <= 0.45*wu {
				pairs = append(pairs, comp{[]int32{u.ids[0], v.ids[0]}, box{u.b.x0, min(u.b.y0, v.b.y0), v.b.x1, max(u.b.y1, v.b.y1)}})
				used[u.ids[0]], used[v.ids[0]] = true, true
			}
		}
	}
	var cands []comp
	for _, c := range comps {
		if !used[c.ids[0]] {
			cands = append(cands, c)
		}
	}
	cands = append(cands, pairs...)
	var found []zone
	for _, c := range cands {
		b := c.b
		z := zone{bbox: b}
		if r := float64(b.w()) / float64(b.h()); r < 0.6 || r > 2.6 {
			z.reject = "shape"
			found = append(found, z)
			continue
		}
		cm := newMask(b.w(), b.h())
		for y := b.y0; y < b.y1; y++ {
			for x := b.x0; x < b.x1; x++ {
				if l := lab[y*W+x]; l == c.ids[0] || (len(c.ids) == 2 && l == c.ids[1]) {
					cm.p[(y-b.y0)*cm.w+x-b.x0] = true
				}
			}
		}
		ang := skew(cm)
		z.angle = ang
		mg := trunc(0.1*float64(max(b.h(), b.w()))) + 4
		X0, Y0, X1, Y1 := max(0, b.x0-mg), max(0, b.y0-mg), min(W, b.x1+mg), min(H, b.y1+mg)
		crop := g.crop(X0, Y0, X1, Y1)
		raw := rotate(crop, ang)
		lm := newMask(crop.w, crop.h)
		for y := 0; y < cm.h; y++ {
			for x := 0; x < cm.w; x++ {
				if cm.p[y*cm.w+x] {
					lm.p[(y+b.y0-Y0)*lm.w+x+b.x0-X0] = true
				}
			}
		}
		lm = lm.dilate(2)
		for i, keep := range lm.p {
			if !keep {
				crop.p[i] = 1
			}
		}
		rc := rotate(crop, ang)
		ri := hysteresis(rc)
		r0, r1, c0, c1 := -1, -1, -1, -1
		for y := 0; y < ri.h; y++ {
			for x := 0; x < ri.w; x++ {
				if ri.p[y*ri.w+x] {
					if r0 < 0 {
						r0 = y
					}
					r1 = y
					if c0 < 0 || x < c0 {
						c0 = x
					}
					c1 = max(c1, x)
				}
			}
		}
		if r0 < 0 {
			continue
		}
		rc = rc.crop(c0, r0, c1+1, r1+1)
		ri = ri.sub(c0, r0, c1+1, r1+1)
		fb, ok := findFrame(ri)
		if !ok {
			z.reject = "frame"
		} else {
			z.points = triangleEvidence(ri, fb, rc)
			z.frame = fb
			need := 0.4
			if len(c.ids) == 2 {
				need = 0.3
			}
			if z.points < need {
				z.reject = "points"
			} else {
				rawInk := hysteresis(raw)
				z.digits = labelDigits(rawInk, r0, c0, fb)
				z.cubes, z.dice = squares(raw, rawInk, r0, c0, fb, X0, Y0)
				z.trays = trays(rawInk, r0, c0, fb, z.cubes, X0, Y0)
				z.field = [6]float64{float64(X0 + c0 + fb.a), float64(X0 + c0 + fb.b), float64(X0 + c0 + fb.c),
					float64(X0 + c0 + fb.d), float64(Y0 + r0 + fb.y0), float64(Y0 + r0 + fb.y1)}
			}
		}
		z.crop, z.ink = rc, ri
		found = append(found, z)
	}
	return found
}

// labelDigits counts the glyphs printed above and below each column, in a
// band 0.1 to 1 point outside the field: numbers 1-9 have one glyph, 10-24
// two, so the row and side of the one-glyph numbers give the corner of point
// 1 without reading a digit.
func labelDigits(ink *mask, oy, ox int, f frame) [2][12]int {
	p := f.pointWidth()
	var out [2][12]int
	for r, top := range []bool{true, false} {
		var ya, yb int
		if top {
			ya, yb = trunc(float64(oy+f.y0)-1.0*p), trunc(float64(oy+f.y0)-0.1*p)
		} else {
			ya, yb = trunc(float64(oy+f.y1)+0.1*p), trunc(float64(oy+f.y1)+1.0*p)
		}
		ya, yb = max(ya, 0), min(yb, ink.h)
		for j := 0; j < 12; j++ {
			base := f.a
			if j >= 6 {
				base = f.c
			}
			xa := trunc(float64(ox+base) + float64(j%6)*p)
			xb := trunc(float64(xa) + p)
			if ya >= yb || max(xa, 0) >= min(max(xb, 0), ink.w) {
				continue
			}
			band := ink.sub(max(xa, 0), ya, max(xb, 0), yb)
			_, bs, _ := label(band)
			for _, b := range bs {
				if float64(b.h()) >= 0.15*p {
					out[r][j]++
				}
			}
		}
	}
	return out
}

// square is a square drawn about the board: a cube, with its height, or a
// die, with its colour and the pips counted on it.
type square struct {
	place string // a cube's: "top", "middle", "bottom"
	dark  bool   // a die filled with ink
	pips  int    // a die's pips, 0 when they are not counted clearly
	// the square in the page's pixels, the straightening neglected
	x0, y0, x1, y1 float64
}

// squares finds the squares drawn about the board by their shape alone (ink
// in the four corners of their box, which a round checker has not; a side of
// 0.45 to 1.4 point): in the field near mid-height, a die, whose pips are
// counted; on the bar or beside the field, a cube, with its height. The
// cube's value is read with the document's figures (cubeFigures).
func squares(g *raster, ink *mask, oy, ox int, f frame, px, py int) ([]square, []square) {
	p := f.pointWidth()
	h := float64(f.y1 - f.y0)
	lab, bs, _ := label(ink)
	var cubes, dice []square
	for i, b := range bs {
		hh, ww := float64(b.h()), float64(b.w())
		if hh < 0.45*p || hh > 1.4*p || ww/hh < 0.75 || ww/hh > 1.33 {
			continue
		}
		k := max(1, trunc(0.12*min(hh, ww)))
		corner := func(x0, y0 int) bool {
			for y := y0; y < y0+k; y++ {
				for x := x0; x < x0+k; x++ {
					if ink.at(x, y) {
						return true
					}
				}
			}
			return false
		}
		if !corner(b.x0, b.y0) || !corner(b.x1-k, b.y0) || !corner(b.x0, b.y1-k) || !corner(b.x1-k, b.y1-k) {
			continue
		}
		cy := float64(b.y0+b.y1)/2 - float64(oy)
		cx := float64(b.x0+b.x1)/2 - float64(ox)
		if cy < float64(f.y0)-0.5*p || cy > float64(f.y1)+0.5*p {
			continue
		}
		sq := square{x0: float64(px + b.x0), y0: float64(py + b.y0), x1: float64(px + b.x1), y1: float64(py + b.y1)}
		inBar := float64(f.b) <= cx && cx <= float64(f.c)
		inField := (float64(f.a) < cx && cx < float64(f.b)) || (float64(f.c) < cx && cx < float64(f.d))
		switch {
		case inField && float64(f.y0)+0.3*h <= cy && cy <= float64(f.y1)-0.3*h:
			sq.dark, sq.pips = diePips(g, ink, lab, int32(i+1), b)
			dice = append(dice, sq)
		case inBar || cx < float64(f.a) || cx > float64(f.d):
			switch {
			case cy < float64(f.y0)+0.3*h:
				sq.place = "top"
			case cy > float64(f.y1)-0.3*h:
				sq.place = "bottom"
			default:
				sq.place = "middle"
			}
			cubes = append(cubes, sq)
		}
	}
	return cubes, dice
}

// diePips tells a die's colour (ink over most of its inside) and counts its
// pips: the spots of the other colour inside it, away from its edge, of one
// size. It gives 0 pips unless one to six spots of one size are found.
func diePips(g *raster, ink *mask, lab []int32, l int32, b box) (bool, int) {
	side := float64(min(b.w(), b.h()))
	m := max(1, trunc(0.1*side))
	x0, y0, x1, y1 := b.x0+m, b.y0+m, b.x1-m, b.y1-m
	if x1-x0 < 4 || y1-y0 < 4 {
		return false, 0
	}
	inked := 0
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if ink.at(x, y) {
				inked++
			}
		}
	}
	dark := 2*inked > (x1-x0)*(y1-y0)
	// A spot: on a dark die, paper; on a light one, ink of another
	// component than the die's outline. Spots are taken within a thin
	// border only, a pip may lie close to the edge; the paper outside a
	// rounded corner touches the box and is left out.
	m = max(1, trunc(0.04*side))
	x0, y0, x1, y1 = b.x0+m, b.y0+m, b.x1-m, b.y1-m
	spot := newMask(x1-x0, y1-y0)
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			// The grey, not the ink mask: a small pip on a dark die
			// is closed by the ink's hysteresis.
			on := g.at(x, y) < 0.5
			if dark {
				spot.p[(y-y0)*spot.w+x-x0] = !on
			} else {
				spot.p[(y-y0)*spot.w+x-x0] = on && lab[y*ink.w+x] != l
			}
		}
	}
	_, ss, areas := label(spot)

	var kept []float64
	for j, s := range ss {
		if s.x0 == 0 || s.y0 == 0 || s.x1 == spot.w || s.y1 == spot.h {
			continue
		}
		d := float64(max(s.w(), s.h()))
		if d < 0.06*side {
			continue
		}
		if d > 0.35*side || s.w() > 2*s.h() || s.h() > 2*s.w() {
			return dark, 0
		}
		kept = append(kept, float64(areas[j]))
	}
	if len(kept) < 1 || len(kept) > 6 {
		return dark, 0
	}
	lo, hi := kept[0], kept[0]
	for _, a := range kept {
		lo, hi = min(lo, a), max(hi, a)
	}
	if hi > 2.5*lo {
		return dark, 0
	}
	return dark, len(kept)
}

// readingOrder sorts a page's diagrams as a reader meets them: the left page
// of a spread before the right (a diagram wholly on one side of the middle),
// then columns (overlapping in x), then top to bottom.
func readingOrder(zs []zone, W int) []zone {
	side := func(z zone) int {
		if float64(z.bbox.x1) <= float64(W)/2 {
			return 0
		}
		if float64(z.bbox.x0) >= float64(W)/2 {
			return 1
		}
		return 0
	}
	var out []zone
	for s := 0; s < 2; s++ {
		var bs []zone
		for _, z := range zs {
			if side(z) == s {
				bs = append(bs, z)
			}
		}
		sort.SliceStable(bs, func(i, j int) bool { return bs[i].bbox.x0 < bs[j].bbox.x0 })
		var cols [][]zone
		for _, z := range bs {
			placed := false
			for ci, col := range cols {
				last := col[len(col)-1].bbox
				if float64(min(last.x1, z.bbox.x1)-max(last.x0, z.bbox.x0)) > 0.3*float64(z.bbox.w()) {
					cols[ci] = append(cols[ci], z)
					placed = true
					break
				}
			}
			if !placed {
				cols = append(cols, []zone{z})
			}
		}
		for _, col := range cols {
			sort.SliceStable(col, func(i, j int) bool { return col[i].bbox.y0 < col[j].bbox.y0 })
			out = append(out, col...)
		}
	}
	return out
}

// trays counts the checkers drawn borne off, seen edge-on as thin bars
// stacked in a tray: a band beside the field bounded by two vertical strokes
// (the field's edge and the board's, or a box of its own), on the side of
// either row. A bar is a run of rows inked across the band, thinner than 0.4
// point, away from the band's own ends; a run thicker is a divider or a
// stack whose bars touch. The bars must also look like checkers seen
// edge-on: of one thickness, at one pitch, and not the edges of hollow
// rectangles (two bars joined at both ends by side strokes). A band that a
// square (a cube) overlaps is not read. A row's tray is read only when
// exactly one band beside it holds bars; 0 is "not read", never "empty".
func trays(ink *mask, oy, ox int, f frame, cubes []square, px, py int) [2]int {
	p := f.pointWidth()
	var out [2]int
	mid := (f.y0 + f.y1) / 2
	minLine := max(4, trunc(1.5*p))
	for row := 0; row < 2; row++ {
		ya, yb := oy+f.y0-trunc(0.3*p), oy+mid
		if row == 1 {
			ya, yb = oy+mid, oy+f.y1+trunc(0.3*p)
		}
		ya, yb = max(ya, 0), min(yb, ink.h)
		if yb-ya < minLine {
			continue
		}
		found, count := 0, 0
		for _, side := range []int{-1, 1} {
			// x from the field's edge outward, 2.2 points at most.
			edge := ox + f.d
			if side < 0 {
				edge = ox + f.a
			}
			type vline struct{ lo, hi, y0, y1 int }
			var lines []vline
			for k := 0; float64(k) <= 2.2*p; k++ {
				x := edge + side*k
				if x < 0 || x >= ink.w {
					break
				}
				best, b0, run, r0 := 0, 0, 0, 0
				for y := ya; y < yb; y++ {
					if ink.at(x, y) {
						if run == 0 {
							r0 = y
						}
						run++
						if run > best {
							best, b0 = run, r0
						}
					} else {
						run = 0
					}
				}
				if best < minLine {
					continue
				}
				if n := len(lines); n > 0 && abs(lines[n-1].hi-x) <= 1 || n > 0 && abs(lines[n-1].lo-x) <= 1 {
					l := &lines[n-1]
					l.lo, l.hi = min(l.lo, x), max(l.hi, x)
					l.y0, l.y1 = max(l.y0, b0), min(l.y1, b0+best)
					continue
				}
				lines = append(lines, vline{x, x, b0, b0 + best})
			}
			for i := 0; i+1 < len(lines); i++ {
				l, r := lines[i], lines[i+1]
				if side < 0 {
					l, r = r, l
				}
				x0, x1 := l.hi+2, r.lo-1
				if float64(x1-x0) < 0.25*p || float64(x1-x0) > 1.4*p {
					continue
				}
				y0, y1 := max(l.y0, r.y0), min(l.y1, r.y1)
				if y1-y0 < minLine {
					continue
				}
				// A run at an end of the band holds its border, as thick as
				// its vertical strokes, and a bar only beyond that.
				wt := max(l.hi-l.lo, r.hi-r.lo) + 1
				covered := false
				for _, q := range cubes {
					qx0, qx1, qy0, qy1 := q.x0-float64(px), q.x1-float64(px), q.y0-float64(py), q.y1-float64(py)
					if qx1 >= float64(l.lo) && qx0 <= float64(r.hi) && qy1 >= float64(y0) && qy0 <= float64(y1) {
						covered = true
					}
				}
				if covered {
					found += 2
					continue
				}
				var bs []trayBar
				run := 0
				end := func(y int) {
					n, edge := run, y-run <= y0+2 || y >= y1-2
					// A run cut by the split between the rows is a divider
					// or a stack across it, not a bar of this row.
					if row == 0 && y >= yb-2 || row == 1 && y-run <= ya+2 {
						n = 0
					}
					if edge && n > 0 {
						n -= wt + 1
					}
					if n > 0 && float64(n) < 0.4*p {
						bs = append(bs, trayBar{y - n, n, edge})
					}
					run = 0
				}
				for y := y0; y < y1; y++ {
					n := 0
					for x := x0; x < x1; x++ {
						if ink.at(x, y) {
							n++
						}
					}
					if float64(n) >= 0.6*float64(x1-x0) {
						run++
					} else {
						end(y)
					}
				}
				end(y1)
				if len(bs) == 0 {
					continue
				}
				if !edgeOnBars(ink, bs, x0, x1) {
					found += 2
					continue
				}
				found++
				count = len(bs)
			}
		}
		if found == 1 {
			out[row] = count
		}
	}
	return out
}

// edgeOnBars tells bars that are checkers seen edge-on: one thickness, one
// pitch, and no two of them joined at both ends by side strokes, which would
// make them the edges of hollow rectangles.
// trayBar is a bar of a tray: its first row and its thickness.
type trayBar struct {
	y, h int
	edge bool // merged with the band's border: its thickness is an estimate
}

func edgeOnBars(ink *mask, bs []trayBar, x0, x1 int) bool {
	hs, pitches := []float64{}, []float64{}
	for i, b := range bs {
		if !b.edge {
			hs = append(hs, float64(b.h))
		}
		if i > 0 && !b.edge && !bs[i-1].edge {
			pitches = append(pitches, float64(b.y-bs[i-1].y))
		}
	}
	same := func(v []float64) bool {
		m := median(append([]float64(nil), v...))
		for _, x := range v {
			if math.Abs(x-m) > max(1, 0.35*m) {
				return false
			}
		}
		return true
	}
	if len(hs) > 0 && !same(hs) || len(pitches) > 1 && !same(pitches) {
		return false
	}
	for i := 1; i < len(bs); i++ {
		a, b := bs[i-1], bs[i]
		ya, yb := a.y+a.h, b.y
		if yb-ya < 2 {
			continue
		}
		sides := 0
		for y := ya; y < yb; y++ {
			if (ink.at(x0, y) || ink.at(x0+1, y)) && (ink.at(x1-1, y) || ink.at(x1-2, y)) {
				sides++
			}
		}
		if float64(sides) >= 0.7*float64(yb-ya) {
			return false
		}
	}
	return true
}
