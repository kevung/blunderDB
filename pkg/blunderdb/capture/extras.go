package capture

import (
	"image"
	"math"
	"sort"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// readExtras reads the dice, the turn and the cube. The score is never on the
// board itself: it is left to the user.
func (t *theme) readExtras(r *raster, g registration, rd *Reading, aIsBottom bool, checkers []image.Point) {
	rd.Unread = append(rd.Unread, "score")
	pos := &rd.Position
	// Unread, the score stays at money play — the one value that makes no
	// claim about a match — until the user sets it.
	pos.Score = [2]int{-1, -1}
	playerOf := func(c class) int {
		if (c == clsCheckerA) == aIsBottom {
			return domain.Black
		}
		return domain.White
	}

	dice, owner, ok := t.readDice(r, g, checkers)
	switch {
	case ok:
		pos.Dice = dice
		pos.PlayerOnRoll = playerOf(owner)
		pos.DecisionType = domain.CheckerAction
	case owner != clsOther:
		// Dice of one colour are on the board but cannot be read.
		pos.PlayerOnRoll = playerOf(owner)
		pos.DecisionType = domain.CheckerAction
		rd.Doubts = append(rd.Doubts, Doubt{Field: "dice", Reason: "the dice could not be read"})
	default:
		pos.DecisionType = domain.CubeAction
		pos.PlayerOnRoll = domain.Black
		rd.Unread = append(rd.Unread, "turn")
	}

	cube, doubt := t.readCube(r, g)
	pos.Cube = cube
	if doubt != "" {
		rd.Doubts = append(rd.Doubts, Doubt{Field: "cube", Reason: doubt})
	}
}

// blob is a connected set of pixels.
type blob struct {
	n    int
	box  image.Rectangle
	edge bool // touches the region's border
}

// blobs labels the 4-connected components of a mask over a w×h region.
func blobs(mask []bool, w, h int) []blob {
	seen := make([]bool, len(mask))
	var out []blob
	stack := []int{}
	for i, m := range mask {
		if !m || seen[i] {
			continue
		}
		// A literal: image.Rect would reorder the inverted corners.
		b := blob{box: image.Rectangle{Min: image.Pt(w, h), Max: image.Pt(-1, -1)}}
		stack = append(stack[:0], i)
		seen[i] = true
		for len(stack) > 0 {
			j := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			x, y := j%w, j/w
			b.n++
			b.box.Min.X, b.box.Min.Y = min(b.box.Min.X, x), min(b.box.Min.Y, y)
			b.box.Max.X, b.box.Max.Y = max(b.box.Max.X, x+1), max(b.box.Max.Y, y+1)
			if x == 0 || y == 0 || x == w-1 || y == h-1 {
				b.edge = true
			}
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, ny := x+d[0], y+d[1]
				if nx < 0 || ny < 0 || nx >= w || ny >= h {
					continue
				}
				k := ny*w + nx
				if mask[k] && !seen[k] {
					seen[k] = true
					stack = append(stack, k)
				}
			}
		}
		out = append(out, b)
	}
	return out
}

// region is a rectangle of the capture copied out, in pixels.
type region struct {
	x0, y0, w, h int
	col          [][3]float64
}

func (r *raster) region(x0, y0, x1, y1 float64) region {
	rg := region{x0: int(math.Max(0, x0)), y0: int(math.Max(0, y0))}
	rg.w = int(math.Min(float64(r.w), x1)) - rg.x0
	rg.h = int(math.Min(float64(r.h), y1)) - rg.y0
	if rg.w <= 0 || rg.h <= 0 {
		rg.w, rg.h = 0, 0
		return rg
	}
	rg.col = make([][3]float64, rg.w*rg.h)
	for y := 0; y < rg.h; y++ {
		for x := 0; x < rg.w; x++ {
			rg.col[y*rg.w+x] = r.at(rg.x0+x, rg.y0+y)
		}
	}
	return rg
}

// readDice looks for the dice in the band across the middle of the board: two
// squares in one player's colour, each read by counting its pips. owner is
// the colour of the dice found, clsOther when there are none.
func (t *theme) readDice(r *raster, g registration, checkers []image.Point) (dice [2]int, owner class, ok bool) {
	mid := t.fieldH / 2
	rg := r.region(g.x(0), g.y(mid-1.6), g.x(t.fieldW()), g.y(mid+1.6))
	if rg.w == 0 {
		return dice, clsOther, false
	}
	type die struct {
		cls  class
		pips int
		x    int
	}
	barL, barR := g.x(6)-float64(rg.x0), g.x(6+t.barW)-float64(rg.x0)
	var found []die
	for _, cls := range []class{clsCheckerA, clsCheckerB} {
		mask := make([]bool, len(rg.col))
		for i, c := range rg.col {
			// A die is drawn darker or lighter than the checkers of its colour.
			mask[i] = t.nearestAny(c) == cls
		}
		for _, b := range blobs(mask, rg.w, rg.h) {
			bw, bh := float64(b.box.Dx())/g.p, float64(b.box.Dy())/g.p
			// A die is smaller than a checker and roughly square, even
			// tilted; its face fills most of its box.
			if bw < 0.45 || bh < 0.45 || bw > 0.85 || bh > 0.85 || bw/bh > 1.5 || bh/bw > 1.5 {
				continue
			}
			if float64(b.n) < 0.45*float64(b.box.Dx()*b.box.Dy()) {
				continue
			}
			if float64(b.box.Max.X) > barL && float64(b.box.Min.X) < barR {
				continue // a checker on the bar
			}
			cx, cy := float64(rg.x0)+float64(b.box.Min.X+b.box.Max.X)/2, float64(rg.y0)+float64(b.box.Min.Y+b.box.Max.Y)/2
			onChecker := false
			for _, c := range checkers {
				if math.Hypot(cx-float64(c.X), cy-float64(c.Y)) < 0.45*g.p {
					onChecker = true
				}
			}
			if onChecker {
				continue // the top of a tall stack reaching the middle
			}
			pips := countPips(rg, mask, b.box, cls == clsCheckerA)
			found = append(found, die{cls, pips, b.box.Min.X})
		}
	}
	if len(found) == 0 {
		return dice, clsOther, false
	}
	sort.Slice(found, func(i, j int) bool { return found[i].x < found[j].x })
	// Shadows and ornaments can pass for a die of the other colour; the
	// colour that shows two dice with readable pips is the roller's.
	byCls := map[class][]die{}
	for _, d := range found {
		if d.pips >= 1 && d.pips <= 6 {
			byCls[d.cls] = append(byCls[d.cls], d)
		}
	}
	switch {
	case len(byCls[clsCheckerA]) == 2 && len(byCls[clsCheckerB]) != 2:
		found = byCls[clsCheckerA]
	case len(byCls[clsCheckerB]) == 2 && len(byCls[clsCheckerA]) != 2:
		found = byCls[clsCheckerB]
	}
	owner = found[0].cls
	if len(found) != 2 || found[0].cls != found[1].cls {
		return dice, owner, false
	}
	for i, d := range found {
		if d.pips < 1 || d.pips > 6 {
			return dice, owner, false
		}
		dice[i] = d.pips
	}
	return dice, owner, true
}

// countPips counts the spots on one die as the local extremes of its
// luminance that stand out from the ring of pixels two steps around them:
// darkest points on a light die, lightest on a dark one. At a few pixels per
// spot, spots run together under any threshold, but each keeps its own
// extreme; the shadow beside a die is no darker than its surroundings.
func countPips(rg region, faceMask []bool, box image.Rectangle, lightFace bool) int {
	var faceLums []float64
	lum := func(x, y int) float64 {
		x, y = clampInt(x, 0, rg.w-1), clampInt(y, 0, rg.h-1)
		l := luma(rg.col[y*rg.w+x])
		if !lightFace {
			l = 255 - l // spots are then the dark extremes in both cases
		}
		return l
	}
	for y := box.Min.Y; y < box.Max.Y; y++ {
		for x := box.Min.X; x < box.Max.X; x++ {
			if faceMask[y*rg.w+x] {
				faceLums = append(faceLums, lum(x, y))
			}
		}
	}
	if len(faceLums) == 0 {
		return 0
	}
	sort.Float64s(faceLums)
	face := faceLums[len(faceLums)/2]
	isFace := func(x, y int) bool { return math.Abs(lum(x, y)-face) < 35 }
	// The counter's distances follow the die's size, so that a die drawn
	// large or shrunk by a fractional factor is read like one drawn small:
	// m, the reach of a local extreme, is about a spot's radius; R, the ring
	// a spot must stand out from, lies between a spot and its neighbours.
	side := min(box.Dx(), box.Dy())
	m := max(1, int(math.Round(float64(side)/16)))
	R := max(2, int(math.Round(float64(side)/8)))
	box = box.Inset(-m).Intersect(image.Rect(0, 0, rg.w, rg.h))
	var spots []image.Point
	for y := box.Min.Y; y < box.Max.Y; y++ {
		for x := box.Min.X; x < box.Max.X; x++ {
			l := lum(x, y)
			isMin := true
			for dy := -m; dy <= m && isMin; dy++ {
				for dx := -m; dx <= m; dx++ {
					if dx == 0 && dy == 0 {
						continue
					}
					v := lum(x+dx, y+dy)
					// Ties go to the first pixel in reading order.
					if v < l || (v == l && (dy < 0 || (dy == 0 && dx < 0))) {
						isMin = false
						break
					}
				}
			}
			if !isMin {
				continue
			}
			ring, k := 0.0, 0
			for d := -R; d <= R; d++ {
				ring += lum(x+d, y-R) + lum(x+d, y+R)
				k += 2
				if d > -R && d < R {
					ring += lum(x-R, y+d) + lum(x+R, y+d)
					k += 2
				}
			}
			if l >= ring/float64(k)-60 {
				continue
			}
			// A spot lies on the face: most of the eight compass points
			// two spot-widths away show the face's colour, not the board's.
			around := 0
			for _, d := range [8][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}} {
				if isFace(x+R*d[0], y+R*d[1]) {
					around++
				}
			}
			if around < 5 {
				continue
			}
			near := false
			for _, q := range spots {
				if abs(q.X-x) <= m && abs(q.Y-y) <= m {
					near = true
				}
			}
			if !near {
				spots = append(spots, image.Pt(x, y))
			}
		}
	}
	if !pointSymmetric(spots, math.Max(1.5, float64(side)/8)) {
		return 0
	}
	return len(spots)
}

// pointSymmetric says whether spots could be a die's face: every face from
// one to six is symmetric about its centre, which holds a spot exactly when
// the count is odd. A single spot missed or added by the counter breaks that,
// and the die is then unread rather than misread. A pair of opposite spots
// lost or gained together keeps the symmetry and passes: 6↔4, 6↔2, 5↔3,
// 5↔1, 3↔1 (and 4↔2) cannot be told apart by it.
func pointSymmetric(spots []image.Point, tol float64) bool {
	if len(spots) == 0 {
		return true
	}
	var cx, cy float64
	for _, q := range spots {
		cx += float64(q.X)
		cy += float64(q.Y)
	}
	cx /= float64(len(spots))
	cy /= float64(len(spots))
	centred := 0
	for _, q := range spots {
		mx, my := 2*cx-float64(q.X), 2*cy-float64(q.Y)
		if math.Hypot(mx-float64(q.X), my-float64(q.Y)) <= tol {
			centred++
			continue
		}
		mirrored := false
		for _, o := range spots {
			if math.Hypot(mx-float64(o.X), my-float64(o.Y)) <= tol {
				mirrored = true
				break
			}
		}
		if !mirrored {
			return false
		}
	}
	return centred == len(spots)%2
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// readCube finds the cube in the tray right of the board: in the middle when
// nobody owns it, at the top or the bottom next to its owner. Owned, its
// value is read from its digits: how many glyphs and how many holes in them
// (2 has none, 4 one, 8 two; 16 one, 32 none, 64 two).
func (t *theme) readCube(r *raster, g registration) (domain.Cube, string) {
	cube := domain.Cube{Owner: domain.None, Value: 0}
	if t.trayW == 0 {
		return cube, ""
	}
	rg := r.region(g.x(t.fieldW()+0.15), g.y(0), g.x(t.fieldW()+t.trayW), g.y(t.fieldH))
	if rg.w == 0 {
		return cube, "the cube is outside the capture"
	}
	mask := make([]bool, len(rg.col))
	for i, c := range rg.col {
		mask[i] = luma(c) > 205 && math.Abs(c[0]-c[2]) < 30
	}
	var best blob
	for _, b := range blobs(mask, rg.w, rg.h) {
		if b.n > best.n {
			best = b
		}
	}
	bw, bh := float64(best.box.Dx())/g.p, float64(best.box.Dy())/g.p
	if bw < 0.4 || bh < 0.4 || bw > 1.4 || bh > 1.4 {
		return cube, "no cube found"
	}
	v := (float64(rg.y0+best.box.Min.Y+best.box.Max.Y)/2 - g.y0) / (g.p * g.hs)
	switch {
	case v < t.fieldH/3:
		// The top of the screen is blunderDB's White whatever its colour.
		cube.Owner = domain.White
	case v > 2*t.fieldH/3:
		cube.Owner = domain.Black
	default:
		return cube, ""
	}
	value, ok := cubeDigits(rg, best.box)
	if !ok {
		cube.Value = 1
		return cube, "the cube's value could not be read"
	}
	cube.Value = value
	return cube, ""
}

// cubeDigits reads an owned cube's value as a log2 exponent.
func cubeDigits(rg region, box image.Rectangle) (int, bool) {
	w, h := box.Dx(), box.Dy()
	ink := make([]bool, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			ink[y*w+x] = luma(rg.col[(box.Min.Y+y)*rg.w+box.Min.X+x]) < 120
		}
	}
	glyphs := 0
	for _, b := range blobs(ink, w, h) {
		if !b.edge && b.box.Dy() >= h/4 {
			glyphs++
		}
	}
	paper := make([]bool, w*h)
	for i, v := range ink {
		paper[i] = !v
	}
	holes := 0
	for _, b := range blobs(paper, w, h) {
		if !b.edge {
			holes++
		}
	}
	table := map[[2]int]int{{1, 0}: 1, {1, 1}: 2, {1, 2}: 3, {2, 1}: 4, {2, 0}: 5, {2, 2}: 6}
	v, ok := table[[2]int{glyphs, holes}]
	if !ok {
		return 0, false
	}
	return v, true
}
