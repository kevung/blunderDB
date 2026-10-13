package capture

import (
	"fmt"
	"image"
	"math"
	"slices"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// screenStack is what one point shows: a run of one colour from its base.
type screenStack struct {
	cls     class
	count   int
	doubt   string
	col     int
	top     bool
	point   int // blunderDB index once the orientation is known
	tallest bool
}

func (t *theme) read(r *raster, g registration) Reading {
	var rd Reading
	rd.Software = t.software
	rd.Board = image.Rect(int(g.x(0)), int(g.y(0)), int(g.x(t.fieldW())), int(g.y(t.fieldH)))

	// A point whose triangle does not show where it should is hidden (a
	// dialog, the pointer) or drawn in another way: its reading is a doubt.
	misses := map[[2]int]int{}
	for _, q := range t.probes() {
		if q.col >= 0 && !t.agrees(r, q, g) {
			misses[[2]int{q.col, btoi(q.top)}]++
		}
	}
	var stacks []screenStack
	for _, top := range []bool{true, false} {
		for c := 0; c < 12; c++ {
			s := t.readStack(r, g, c, top)
			if misses[[2]int{c, btoi(top)}] >= 2 {
				s.doubt = "the point is hidden or unclear"
			}
			stacks = append(stacks, s)
		}
	}
	barA, barB, barDoubt := t.readBar(r, g)

	homeLeft, labelsSeen := t.readHomeSide(r, g)
	if !labelsSeen {
		// The point numbers are hidden, so the theme's default is a guess:
		// the board may be mirrored, and a mirrored board is never settled
		// without the user seeing it.
		homeLeft = t.homeLeft
		rd.Doubts = append(rd.Doubts, Doubt{Field: "orientation", Reason: "the point numbers are not visible, so the board's direction is assumed"})
	}

	// Which colour plays from the bottom. The theme's default holds unless
	// the pip counts say plainly that it is the other way round: each side
	// is nearer its own home than its opponent's in all but freak positions.
	pipsIfAisBottom, total := 0, 0
	for i := range stacks {
		s := &stacks[i]
		s.point = pointOf(s.col, s.top, homeLeft)
		switch s.cls {
		case clsCheckerA:
			pipsIfAisBottom += s.count * s.point
			total += s.count * 25
		case clsCheckerB:
			pipsIfAisBottom += s.count * (25 - s.point)
			total += s.count * 25
		}
	}
	aIsBottom := t.bottomIsA
	if total > 0 {
		share := float64(pipsIfAisBottom) / float64(total)
		switch {
		case share < 0.4:
			aIsBottom = true
		case share > 0.6:
			aIsBottom = false
		case aIsBottom != (share < 0.5):
			rd.Doubts = append(rd.Doubts, Doubt{Field: "orientation", Reason: "the side each colour plays from is not clear"})
		}
	}
	colourOf := func(c class) int {
		if (c == clsCheckerA) == aIsBottom {
			return domain.Black
		}
		return domain.White
	}

	pos := &rd.Position
	for i := range pos.Board.Points {
		pos.Board.Points[i] = domain.Point{Checkers: 0, Color: domain.None}
	}
	var onBoard [2]int
	for _, s := range stacks {
		if s.count > 0 {
			col := colourOf(s.cls)
			pos.Board.Points[s.point] = domain.Point{Checkers: s.count, Color: col}
			onBoard[col] += s.count
		}
		if s.doubt != "" {
			rd.Doubts = append(rd.Doubts, Doubt{Field: "point", Point: s.point, Reason: s.doubt})
		}
	}
	for _, b := range []struct {
		cls   class
		count int
	}{{clsCheckerA, barA}, {clsCheckerB, barB}} {
		if b.count == 0 {
			continue
		}
		col := colourOf(b.cls)
		idx := domain.WhiteBar
		if col == domain.Black {
			idx = domain.BlackBar
		}
		pos.Board.Points[idx] = domain.Point{Checkers: b.count, Color: col}
		onBoard[col] += b.count
	}
	if barDoubt != "" {
		for _, idx := range []int{domain.WhiteBar, domain.BlackBar} {
			if pos.Board.Points[idx].Checkers > 0 {
				rd.Doubts = append(rd.Doubts, Doubt{Field: "bar", Point: idx, Reason: barDoubt})
			}
		}
	}
	for col := 0; col < 2; col++ {
		if onBoard[col] > 15 {
			rd.Doubts = append(rd.Doubts, Doubt{Field: "checkers", Reason: fmt.Sprintf("%d checkers of one colour on the board", onBoard[col])})
			continue
		}
		pos.Board.Bearoff[col] = 15 - onBoard[col]
	}

	var centres []image.Point
	for _, s := range stacks {
		for k := 0; k < min(s.count, t.slots); k++ {
			centres = append(centres, image.Pt(int(g.x(t.colX(s.col))), int(g.y(t.slotY(k, s.top)))))
		}
	}
	t.readExtras(r, g, &rd, aIsBottom, centres)
	return rd
}

// pointOf numbers a screen column for the bottom player (blunderDB's Black),
// whose home board is the lower-left or lower-right quarter.
func pointOf(col int, top, homeLeft bool) int {
	if !homeLeft {
		col = 11 - col
	}
	if top {
		return 24 - col
	}
	return col + 1
}

// readStack reads one point from its base: a run of one colour, then nothing.
// A checker past a gap, a second colour or a colour near no swatch is a doubt.
func (t *theme) readStack(r *raster, g registration, c int, top bool) screenStack {
	s := screenStack{col: c, top: top}
	hw := 0.25 * g.p
	ended := false
	for k := 0; k < t.slots; k++ {
		cx, cy := g.x(t.colX(c)), g.y(t.slotY(k, top))
		col := r.mean(cx-hw, cy-hw, cx+hw, cy+hw)
		cl, d := t.nearest(col)
		isChecker := cl == clsCheckerA || cl == clsCheckerB
		switch {
		case !isChecker:
			if cl == clsOther && !ended && t.nearestAny(col).isChecker() {
				s.doubt = fmt.Sprintf("slot %d may hold a checker", k+1)
			}
			ended = true
		case ended:
			s.doubt = "checker after an empty slot"
		case s.count > 0 && cl != s.cls:
			s.doubt = "two colours on one point"
			ended = true
		default:
			s.cls = cl
			s.count++
			if d > 0.7*t.maxDist {
				s.doubt = fmt.Sprintf("checker %d is barely recognisable", k+1)
			}
		}
	}
	if s.count == t.slots {
		s.tallest = true
		if n, ok := t.readStackLabel(r, g, c, top, s.cls); ok {
			s.count = n
		} else {
			s.doubt = fmt.Sprintf("%d checkers or more", t.slots)
		}
	}
	return s
}

// readStackLabel reads how many checkers a full stack holds when the program
// prints the number on its last checker. Not implemented for any theme yet.
func (t *theme) readStackLabel(_ *raster, _ registration, _ int, _ bool, _ class) (int, bool) {
	return 0, false
}

// readBar counts the checkers on the bar from the length of each run of one
// colour down the bar's centre line. A run shorter than most of a checker is
// a hinge or an ornament, not a checker.
func (t *theme) readBar(r *raster, g registration) (a, b int, doubt string) {
	cx := g.x(6 + t.barW/2)
	hw := 0.3 * g.p
	const step = 0.1
	counts := map[class]int{}
	run, runCls := 0.0, clsOther
	flush := func() {
		if run >= 0.7*t.step {
			n := run / t.step
			rounded := math.Round(n)
			if math.Abs(n-rounded) > 0.35 {
				doubt = "the bar's checkers overlap"
			}
			counts[runCls] += int(rounded)
		}
		run = 0
	}
	for v := 0.0; v < t.fieldH; v += step {
		if t.onHinge(v) {
			continue
		}
		cy := g.y(v)
		// A checker's rings average out over a taller box; between a
		// checker and the bar, only the checker colours and the bar's count.
		cl := t.nearestOf(r.mean(cx-hw, cy-0.12*g.p, cx+hw, cy+0.12*g.p), clsCheckerA, clsCheckerB, clsField)
		if cl != runCls {
			flush()
			runCls = cl
		}
		if cl == clsCheckerA || cl == clsCheckerB {
			run += step
		}
	}
	flush()
	return counts[clsCheckerA], counts[clsCheckerB], doubt
}

// readHomeSide tells the bottom player's home quarter from the point labels
// under the lower row: points 10, 11 and 12 take two digits, 1 to 9 one, so
// the wide labels lie at the far end from home. ok is false when the capture
// shows no labels there.
func (t *theme) readHomeSide(r *raster, g registration) (homeLeft, ok bool) {
	width := func(c int) float64 {
		cx := g.x(t.colX(c))
		y0, y1 := g.y(t.fieldH)+0.1*g.p, g.y(t.fieldH)+0.75*g.p
		bg := luma(r.mean(cx-0.5*g.p, y0, cx+0.5*g.p, y1))
		n := 0
		for x := cx - 0.5*g.p; x < cx+0.5*g.p; x++ {
			col := r.mean(x, y0, x+1, y1)
			// A column crossing a glyph is darker or lighter than the strip.
			lo, hi := 255.0, 0.0
			for y := y0; y < y1; y++ {
				l := luma(r.at(int(x), int(y)))
				lo, hi = math.Min(lo, l), math.Max(hi, l)
			}
			_ = col
			if bg-lo > 40 || hi-bg > 40 {
				n++
			}
		}
		return float64(n)
	}
	left := width(0) + width(1) + width(2)
	right := width(9) + width(10) + width(11)
	mid := width(3) + width(4) + width(8)
	if left+right < 3 {
		return false, false
	}
	// Wide labels must be clearly wider than the single-digit ones.
	switch {
	case right > 1.4*left && right > 1.3*mid:
		return true, true
	case left > 1.4*right && left > 1.3*mid:
		return false, true
	}
	return false, false
}

// nearest is classify with the distance to the chosen swatch.
func (t *theme) nearest(c [3]float64) (class, float64) {
	best, bestD := clsOther, math.Inf(1)
	for _, s := range t.palette {
		dr, dg, db := c[0]-s.rgb[0], c[1]-s.rgb[1], c[2]-s.rgb[2]
		if d := dr*dr + dg*dg + db*db; d < bestD {
			best, bestD = s.cls, d
		}
	}
	d := math.Sqrt(bestD)
	if d > t.maxDist {
		return clsOther, d
	}
	return best, d
}

func (c class) isChecker() bool { return c == clsCheckerA || c == clsCheckerB }

// nearestAny is the nearest swatch however far it is.
func (t *theme) nearestAny(c [3]float64) class {
	best, bestD := clsOther, math.Inf(1)
	for _, s := range t.palette {
		dr, dg, db := c[0]-s.rgb[0], c[1]-s.rgb[1], c[2]-s.rgb[2]
		if d := dr*dr + dg*dg + db*db; d < bestD {
			best, bestD = s.cls, d
		}
	}
	return best
}

// nearestOf is the nearest swatch among some classes, however far.
func (t *theme) nearestOf(c [3]float64, among ...class) class {
	best, bestD := clsOther, math.Inf(1)
	for _, s := range t.palette {
		if !slices.Contains(among, s.cls) {
			continue
		}
		dr, dg, db := c[0]-s.rgb[0], c[1]-s.rgb[1], c[2]-s.rgb[2]
		if d := dr*dr + dg*dg + db*db; d < bestD {
			best, bestD = s.cls, d
		}
	}
	return best
}

// onHinge says whether a height on the bar shows one of the board's hinges.
func (t *theme) onHinge(v float64) bool {
	for _, h := range t.hinges {
		if v >= h[0] && v <= h[1] {
			return true
		}
	}
	return false
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}
