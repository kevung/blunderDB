package structure

import (
	"math"
	"sort"
)

// hole is the white inside of a light checker: (cx, cy) its centre, w×h its
// box, in pixels.
type hole struct{ cx, cy, w, h float64 }

// holes are the elliptical closed holes of the field: the insides of the
// light checkers, whatever their colours or strokes.
func holes(ink *mask, f frame) []hole {
	p := f.pointWidth()
	var out []hole
	for _, half := range [2][2]int{{f.a, f.b}, {f.c, f.d}} {
		xa, xb := half[0], half[1]
		sub := ink.sub(xa, f.y0, xb, f.y1)
		for i := range sub.p {
			sub.p[i] = !sub.p[i]
		}
		lab, bs, areas := label(sub)
		for i, b := range bs {
			w, h := float64(b.w()), float64(b.h())
			if w < 0.35*p || w > 1.05*p || h < 0.2*p || h > 1.1*p {
				continue
			}
			if b.y0 == 0 || b.y1 == f.y1-f.y0 || b.x0 == 0 || b.x1 == xb-xa {
				continue
			}
			area := float64(areas[i])
			if fill := area / (w * h); fill < 0.70 || fill > 0.84 {
				continue
			}
			// Symmetric top and bottom: a point's triangle is not.
			hh := b.h() / 2
			upper, lower := 0, 0
			for y := b.y0; y < b.y1; y++ {
				for x := b.x0; x < b.x1; x++ {
					if lab[y*sub.w+x] == int32(i+1) {
						if y-b.y0 < hh {
							upper++
						}
						if y-b.y0 >= b.h()-hh {
							lower++
						}
					}
				}
			}
			if math.Abs(float64(upper-lower)) > 0.12*area {
				continue
			}
			out = append(out, hole{float64(b.x0+b.x1)/2 + float64(xa), float64(b.y0+b.y1)/2 + float64(f.y0), w, h})
		}
	}
	return out
}

// geometry is the checker the document draws, in units of the point width:
// the ellipse's half axes, the stacking step and the acceptance threshold.
type geometry struct {
	rx, ry, step, thr float64
	learnedOn         int
}

// estimateGeometry learns the checker from the holes of a document's first
// diagrams: their median width and height give the ellipse, the gap between
// neighbouring holes of one column the step.
func estimateGeometry(samples [][]hole, ps []float64) (geometry, bool) {
	var ws, hs, steps []float64
	for si, hl := range samples {
		p := ps[si]
		for _, x := range hl {
			ws = append(ws, x.w/p)
			hs = append(hs, x.h/p)
		}
		for i := range hl {
			for j := range hl {
				if i == j {
					continue
				}
				dy := hl[j].cy - hl[i].cy
				if math.Abs(hl[i].cx-hl[j].cx) < 0.15*p && dy > 0 && dy < 1.6*max(hl[i].h, hl[j].h) {
					steps = append(steps, dy/p)
				}
			}
		}
	}
	if len(ws) < 5 || len(steps) == 0 {
		return geometry{}, false
	}
	r := func(x float64) float64 { return math.Round(x*1000) / 1000 }
	return geometry{
		rx: r(median(ws)/2 + 0.03), ry: r(median(hs)/2 + 0.03), step: r(median(steps)), thr: 0.6,
		learnedOn: len(samples),
	}, true
}

// pNorm is the point width, in pixels, every diagram is read at.
const pNorm = 50

// The 72 angles of an ellipse's outline.
var cosA, sinA [72]float64

func init() {
	for i := range cosA {
		a := 2 * math.Pi * float64(i) / 72
		cosA[i], sinA[i] = math.Cos(a), math.Sin(a)
	}
}

// edgeMap marks the edges of a raster: a gradient of the lightly smoothed
// grey above a fifth of the local contrast (range over about 1.5 points, a
// pale corner of a scan keeps its outlines), never below 0.3 of it (paper
// grain is not an edge), grown by one pixel.
func edgeMap(g *raster) *mask {
	W, H := g.w, g.h
	s := g.clone()
	for y := 1; y < H-1; y++ {
		for x := 1; x < W-1; x++ {
			s.p[y*W+x] = (g.at(x, y-1) + g.at(x, y+1) + g.at(x-1, y) + g.at(x+1, y) + g.at(x, y)) / 5
		}
	}
	sv := make([]float64, len(s.p))
	for i, v := range s.p {
		sv[i] = float64(v)
	}
	mx := windowFilter(sv, W, H, 75, true)
	mn := windowFilter(sv, W, H, 75, false)
	e := newMask(W, H)
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			var gx, gy float32
			if x > 0 && x < W-1 {
				gx = s.at(x+1, y) - s.at(x-1, y)
			}
			if y > 0 && y < H-1 {
				gy = s.at(x, y+1) - s.at(x, y-1)
			}
			i := y*W + x
			rng := float32(mx[i] - mn[i])
			thr := 0.2 * max(min(rng, 1), 0.3)
			e.p[i] = float32(math.Hypot(float64(gx), float64(gy))) > thr
		}
	}
	return e.dilate(1)
}

// ring samples the edge map on the ellipse k·(rx, ry) about (x, y), one value
// per angle, clipped to the image.
func ring(E *mask, x, y int, rx, ry, k float64, out *[72]bool) {
	for i := 0; i < 72; i++ {
		dx := roundHalfEven(k * rx * cosA[i])
		dy := roundHalfEven(k * ry * sinA[i])
		yy := min(max(y+dy, 0), E.h-1)
		xx := min(max(x+dx, 0), E.w-1)
		out[i] = E.p[yy*E.w+xx]
	}
}

// cover is, at one centre, the edge coverage of the eight sectors of the
// outline (0 right, 2 down, 4 left, 6 up, in image coordinates) over the
// ring 0.85..1.15, and the share of edges inside the disc.
type cover struct {
	sect  [8]float64
	inner float64
}

func ellipseCover(E *mask, x, y int, rx, ry float64) cover {
	var out, v [72]bool
	for i := 0; i < 7; i++ {
		ring(E, x, y, rx, ry, 0.85+0.05*float64(i), &v)
		for a := range out {
			out[a] = out[a] || v[a]
		}
	}
	var c cover
	for _, k := range [4]float64{0.2, 0.35, 0.5, 0.6} {
		ring(E, x, y, rx, ry, k, &v)
		n := 0
		for _, b := range v {
			if b {
				n++
			}
		}
		c.inner += float64(n) / 72 / 4
	}
	for s := 0; s < 8; s++ {
		n := 0
		for d := -4; d <= 4; d++ {
			if out[(9*s+d+72)%72] {
				n++
			}
		}
		c.sect[s] = float64(n) / 9
	}
	return c
}

// score is min(mean, worst + 0.3) of the kept sectors, less 1.5 times the
// inner edges.
func (c cover) score(keep []int, inner bool) float64 {
	sum, worst := 0.0, math.Inf(1)
	for _, s := range keep {
		sum += c.sect[s]
		worst = min(worst, c.sect[s])
	}
	v := min(sum/float64(len(keep)), worst+0.3)
	if inner {
		v -= 1.5 * c.inner
	}
	return v
}

func density(E *mask, x, y int, rx, ry float64, ks []float64) float64 {
	var v [72]bool
	d := 0.0
	for _, k := range ks {
		ring(E, x, y, rx, ry, k, &v)
		n := 0
		for _, b := range v {
			if b {
				n++
			}
		}
		d += float64(n) / 72 / float64(len(ks))
	}
	return d
}

// innerStats is the mean and deviation of the grey inside 0.6 of the
// ellipse; false when the ellipse's box leaves the image.
func innerStats(g *raster, x, y int, rx, ry float64) (float64, float64, bool) {
	xr, yr := int(math.Ceil(rx)), int(math.Ceil(ry))
	if y-yr < 0 || x-xr < 0 || y+yr+1 > g.h || x+xr+1 > g.w {
		return 0, 0, false
	}
	var vs []float64
	for dy := -yr; dy <= yr; dy++ {
		for dx := -xr; dx <= xr; dx++ {
			if (float64(dx)/rx)*(float64(dx)/rx)+(float64(dy)/ry)*(float64(dy)/ry) <= 0.36 {
				vs = append(vs, float64(g.at(x+dx, y+dy)))
			}
		}
	}
	if len(vs) == 0 {
		return 0, 0, false
	}
	m := 0.0
	for _, v := range vs {
		m += v
	}
	m /= float64(len(vs))
	sd := 0.0
	for _, v := range vs {
		sd += (v - m) * (v - m)
	}
	return m, math.Sqrt(sd / float64(len(vs))), true
}

// disc is a checker seen in a column.
type disc struct {
	y           int
	score, mean float64
	std         float64
	marked      bool // a figure in it, perhaps the stack's height
}

// column is what one point's column holds, from its base.
type column struct {
	top   bool
	j     int
	discs []disc
	// slotMeans is the inner grey of every slot followed from the base, for
	// counting a dark stack by its length.
	slotMeans []float64
}

type barDisc struct {
	y         int
	mean, std float64
	top       bool
}

// boardReading is the raw reading of one diagram, before the document's
// classes and orientation are known.
type boardReading struct {
	cols   []column // top row j=0..11, then bottom row
	bar    []barDisc
	n      int    // slots a column can hold
	rows   [2]int // points attested in the top and bottom rows
	digits [2][12]int
	cubes  []square
	dice   []square
	trays  [2]int // borne off in the top and bottom rows' trays, 0 when not read
	// inkGrey is the grey this diagram prints its strokes in (paper is 1).
	inkGrey float64
	field   [6]float64
	// cubeFigs are the numbers read in a square beside the field, kept
	// once the document's figures are read.
	cubeFigs []printedNumber
	page     int
	bbox     box
	pageW    int
	pageH    int
	// numbers printed around the diagram, kept until the document's
	// figures are read
	numbers []printedNumber
	// captions are the lines printed above and below it on a page without
	// a text layer, kept until the document's figures are read
	captions [][]captionWord
}

type slot struct {
	y                            int
	m                            float64
	full, ring, term, first, mid float64
}

var allSectors = []int{0, 1, 2, 3, 4, 5, 6, 7}

func without(s []int, drop ...[]int) []int {
	var out []int
outer:
	for _, x := range s {
		for _, d := range drop {
			for _, y := range d {
				if x == y {
					continue outer
				}
			}
		}
		out = append(out, x)
	}
	return out
}

// okBelow: a checker under another is judged without its tip side when it is
// the first; on its sides, or without one of its ends, when it is between.
// The first may also lose its base side (a checker against the field's edge
// when the frame melts into it), its sides then required.
func okBelow(slots []slot, i int, thr float64) bool {
	s := slots[i]
	if i == 0 {
		return s.first >= thr || (s.term >= thr && s.mid >= thr)
	}
	return max(s.mid, s.first, s.term) >= thr
}

// readBoard reads the checkers of a located diagram with the document's
// checker geometry.
func readBoard(z zone, geo geometry) boardReading {
	g, f := z.crop, z.frame
	gi := hysteresis(g)
	out := boardReading{rows: rowEvidence(gi, f, g), inkGrey: inkGrey(g, gi)}
	// Rescaled so a point is pNorm pixels wide.
	if p := f.pointWidth(); math.Abs(p-pNorm) > 2 {
		s := pNorm / p
		g = g.resize(max(1, roundHalfEven(float64(g.w)*s)), max(1, roundHalfEven(float64(g.h)*s)))
		if f2, ok := findFrame(hysteresis(g)); ok {
			f = f2
		} else {
			sc := func(v int) int { return roundHalfEven(float64(v) * s) }
			f = frame{sc(f.a), sc(f.b), sc(f.c), sc(f.d), sc(f.y0), sc(f.y1)}
		}
	}
	// A white margin: a checker against the zone's edge stays readable.
	m := pNorm / 2
	pad := newRaster(g.w+2*m, g.h+2*m)
	for i := range pad.p {
		pad.p[i] = 1
	}
	for y := 0; y < g.h; y++ {
		copy(pad.p[(y+m)*pad.w+m:(y+m)*pad.w+m+g.w], g.p[y*g.w:(y+1)*g.w])
	}
	g, f = pad, f.shift(m, m)
	p := f.pointWidth()
	rx, ry, step := geo.rx*p, geo.ry*p, geo.step*p
	thr := geo.thr
	// A stack may pass mid-height: slots are followed over the whole field,
	// and the top and bottom stacks are shared at the relevé.
	nmax := trunc((float64(f.y1-f.y0)-2*ry)/step) + 1
	out.n = nmax
	strokeInk := hysteresis(g)
	sts, dirs := arrowStrokes(strokeInk, f, p)
	inpaintStrokes(g, strokeInk, sts, dirs)
	E := edgeMap(g)
	for _, top := range []bool{true, false} {
		for j := 0; j < 12; j++ {
			base := f.a
			if j >= 6 {
				base = f.c
			}
			x := roundHalfEven(float64(base) + (float64(j%6)+0.5)*p)
			yc := float64(f.y1) - ry
			dirn := -1.0
			tip, baseS := []int{5, 6, 7}, []int{1, 2, 3}
			if top {
				yc, dirn = float64(f.y0)+ry, 1
				tip, baseS = []int{1, 2, 3}, []int{5, 6, 7}
			}
			// A column against a vertical stroke (frame, bar): the sector on
			// that side may melt into the stroke and is not required.
			var wall []int
			switch j {
			case 0, 6:
				wall = []int{4}
			case 5, 11:
				wall = []int{0}
			}
			allC := without(allSectors, wall)
			sides := without(allC, tip, baseS)
			noTip := without(allC, tip)
			noBase := without(allC, baseS)
			slots := make([]slot, 0, max(nmax, 0))
			for k := 0; k < nmax; k++ {
				ylo, yhi := trunc(yc-0.3*step), trunc(yc+0.3*step)
				covers := make([]cover, 0, yhi-ylo+1)
				bestFull, iFull := math.Inf(-1), 0
				bestAny, iAny := math.Inf(-1), 0
				for y := ylo; y <= yhi; y++ {
					c := ellipseCover(E, x, y, rx, ry)
					covers = append(covers, c)
					if v := c.score(allC, true); v > bestFull {
						bestFull, iFull = v, len(covers)-1
					}
					if v := c.score(noTip, true); v > bestAny {
						bestAny, iAny = v, len(covers)-1
					}
				}
				i := iAny
				if bestFull >= thr {
					i = iFull
				}
				y := roundHalfEven(yc)
				if max(covers[i].score(allC, true), covers[i].score(noTip, true)) >= thr-0.2 {
					y = ylo + i
				}
				ii := min(max(y-ylo, 0), len(covers)-1)
				c := covers[ii]
				mean, _, ok := innerStats(g, x, y, rx, ry)
				if !ok {
					mean = 1
				}
				slots = append(slots, slot{
					y: y, m: mean,
					full: c.score(allC, true), ring: c.score(noBase, false),
					term: c.score(noBase, true), first: c.score(noTip, true), mid: c.score(sides, true),
				})
				yc = float64(y) + dirn*step
			}
			n := 0
			for k := 0; k < nmax; k++ {
				okEnd := slots[k].term >= thr
				if k == 0 {
					okEnd = slots[0].full >= thr
				}
				if !okEnd {
					continue
				}
				all := true
				for i := 0; i < k; i++ {
					if !okBelow(slots, i, thr) {
						all = false
						break
					}
				}
				if all {
					n = k + 1
				}
			}
			col := column{top: top, j: j}
			for k := 0; k < n; k++ {
				mean, std, ok := innerStats(g, x, slots[k].y, rx, ry)
				if !ok {
					break
				}
				col.discs = append(col.discs, disc{y: slots[k].y, score: round3(slots[k].full), mean: round3(mean), std: round3(std)})
			}
			// A marked checker at the top (a figure): a full outline, a clean
			// crown at 0.68-0.78, a loaded centre — a hatched point has edges
			// up to its crown.
			if n < nmax && len(col.discs) == n {
				sl := slots[n]
				all := true
				for i := 0; i < n; i++ {
					if !okBelow(slots, i, thr) {
						all = false
						break
					}
				}
				if sl.ring >= thr+0.15 && all {
					ann := density(E, x, sl.y, rx, ry, []float64{0.68, 0.73, 0.78})
					cen := density(E, x, sl.y, rx, ry, []float64{0.15, 0.3, 0.45})
					mean, std, ok := innerStats(g, x, sl.y, rx, ry)
					if ann <= 0.1 && cen >= 0.15 && ok {
						col.discs = append(col.discs, disc{y: sl.y, score: round3(sl.full), mean: round3(mean), std: round3(std), marked: true})
					}
				}
			}
			for _, s := range slots {
				col.slotMeans = append(col.slotMeans, round3(s.m))
			}
			out.cols = append(out.cols, col)
		}
	}
	// The bar: ellipses anywhere in its column.
	bx := roundHalfEven(float64(f.b+f.c) / 2)
	if float64(f.c-f.b) > 1.6*rx {
		ylo, yhi := trunc(float64(f.y0)+ry), trunc(float64(f.y1)-ry)
		type cand struct {
			y int
			s float64
		}
		var cs []cand
		for y := ylo; y < yhi; y++ {
			cs = append(cs, cand{y, ellipseCover(E, bx, y, rx, ry).score(allSectors, true)})
		}
		sort.SliceStable(cs, func(i, j int) bool { return cs[i].s > cs[j].s })
		var taken []int
		for _, c := range cs {
			if c.s < thr {
				break
			}
			free := true
			for _, t := range taken {
				if math.Abs(float64(c.y-t)) < 1.8*ry {
					free = false
					break
				}
			}
			if free {
				taken = append(taken, c.y)
			}
		}
		sort.Ints(taken)
		for _, y := range taken {
			if mean, std, ok := innerStats(g, bx, y, rx, ry); ok {
				out.bar = append(out.bar, barDisc{y: y, mean: round3(mean), std: round3(std), top: float64(y) < float64(f.y0+f.y1)/2})
			}
		}
	}
	return out
}

// inkGrey is the grey a diagram prints in: the tenth percentile of the grey
// under its ink, so a faded scan's strokes are not taken for black. A
// diagram without ink prints in black (0).
func inkGrey(g *raster, ink *mask) float64 {
	var v []float64
	for i, on := range ink.p {
		if on {
			v = append(v, float64(g.p[i]))
		}
	}
	if len(v) == 0 {
		return 0
	}
	sort.Float64s(v)
	return v[len(v)/10]
}
