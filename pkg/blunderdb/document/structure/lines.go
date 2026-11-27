package structure

import "math"

// A move drawn as an arrow is a long thin straight stroke across the field.
// Against the frame and a point's sides it can close the outline of a
// checker that is not there, and through a checker it fills the inside of
// one that is. It is painted over before the checkers are read, where it
// runs at its own width; where it crosses an outline it thickens, and the
// outline is kept (see inpaintStrokes).

// strokeSample is one pixel of a stroke and its thickness across it.
type strokeSample struct {
	x, y float64
	t    int
}

// arrowStrokes finds, in the field of a diagram read at point width p, the
// straight strokes slanted away from the frame's rows and the points' sides
// that run thin over at least 2.5 points. A point's side leans less than
// atan(1/10) from the vertical and its hatching stops at its sides, so
// neither is taken; a stroke drawn level or upright is not either.
func arrowStrokes(ink *mask, f frame, p float64) ([][]strokeSample, [][2]float64) {
	x0, x1, y0, y1 := f.a, f.d, f.y0, f.y1
	if x1-x0 < 4 || y1-y0 < 4 {
		return nil, nil
	}
	const minFromRow, minFromSide = 8.0, 16.0
	var angles []float64
	for deg := minFromRow; deg <= 180-minFromRow; deg++ {
		if math.Abs(deg-90) < minFromSide {
			continue
		}
		angles = append(angles, deg*math.Pi/180)
	}
	diag := math.Hypot(float64(x1-x0), float64(y1-y0))
	nr := int(2*diag) + 3
	acc := make([][]int32, len(angles))
	for i := range acc {
		acc[i] = make([]int32, nr)
	}
	cs, sn := make([]float64, len(angles)), make([]float64, len(angles))
	for i, a := range angles {
		cs[i], sn[i] = math.Cos(a), math.Sin(a)
	}
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			if !ink.at(x, y) {
				continue
			}
			fx, fy := float64(x-x0), float64(y-y0)
			for i := range angles {
				r := roundHalfEven(-fx*sn[i]+fy*cs[i]) + nr/2
				acc[i][r]++
			}
		}
	}
	minLen := 1.8 * p
	maxThick := max(3, roundHalfEven(0.16*p))
	maxGap := 1.1 * p
	var out [][]strokeSample
	var dirs [][2]float64
	for i := range angles {
		for r := 1; r < nr-1; r++ {
			v := acc[i][r]
			if float64(v) < minLen || v < acc[i][r-1] || v < acc[i][r+1] {
				continue
			}
			if i > 0 && v < acc[i-1][r] || i+1 < len(angles) && v < acc[i+1][r] {
				continue
			}
			rho := float64(r - nr/2)
			for _, st := range walkStroke(ink, x0, y0, x1, y1, cs[i], sn[i], rho, diag, maxThick, maxGap, minLen) {
				out, dirs = append(out, st), append(dirs, [2]float64{cs[i], sn[i]})
			}
		}
	}
	return out, dirs
}

// walkStroke follows the line of direction (c, s) at distance rho from the
// field's corner and returns its thin runs of at least minLen, gaps of at
// most maxGap bridged where the stroke is hidden under thicker ink, none
// where it leaves the ink.
func walkStroke(ink *mask, x0, y0, x1, y1 int, c, s, rho, diag float64, maxThick int, maxGap, minLen float64) [][]strokeSample {
	nx, ny := -s, c
	thick := func(px, py float64) (int, bool, float64, float64, bool) {
		// The ink nearest the line, across it, within a pixel.
		for _, o := range []float64{0, -1, 1} {
			qx, qy := px+o*nx, py+o*ny
			if !inkAt(ink, qx, qy) {
				continue
			}
			lo, hi := 0, 0
			for inkAt(ink, qx-float64(lo+1)*nx, qy-float64(lo+1)*ny) && lo <= maxThick {
				lo++
			}
			for inkAt(ink, qx+float64(hi+1)*nx, qy+float64(hi+1)*ny) && hi <= maxThick {
				hi++
			}
			t := lo + hi + 1
			// Paper on both sides, where a hatching has its next line.
			alone := true
			for k := 2; k <= 3; k++ {
				if inkAt(ink, qx-float64(lo+k)*nx, qy-float64(lo+k)*ny) || inkAt(ink, qx+float64(hi+k)*nx, qy+float64(hi+k)*ny) {
					alone = false
				}
			}
			cx, cy := qx+float64(hi-lo)/2*nx, qy+float64(hi-lo)/2*ny
			return t, alone, cx, cy, true
		}
		return 0, false, 0, 0, false
	}
	var out [][]strokeSample
	var run []strokeSample
	var thin, blanks, alone, off float64
	blank, crossed := 0, 0.0
	flush := func() {
		// A drawn stroke leaves the ink only for a scan's dropped pixel; a
		// texture crossed (a point's hatching) leaves it at every line. A
		// stroke runs alone over the paper for most of its length, where a
		// walk along a dense hatching always has the next line beside it.
		if thin >= minLen && blanks <= 0.08*thin && alone >= 0.5*thin {
			out = append(out, run)
		}
		run, thin, blanks, alone, blank, crossed, off = nil, 0, 0, 0, 0, 0, 0
	}
	for d := -diag; d <= diag; d++ {
		// The walk follows the stroke it is on, two pixels aside at most:
		// the vote's angle is a degree apart at most, a stroke's few pixels
		// over its length, where a hatching's next line is farther.
		px, py := float64(x0)+d*c+(rho+off)*nx, float64(y0)+d*s+(rho+off)*ny
		if px < float64(x0) || px > float64(x1) || py < float64(y0) || py > float64(y1) {
			if run != nil {
				flush()
			}
			continue
		}
		t, isAlone, cx, cy, ok := thick(px, py)
		switch {
		case ok && t <= maxThick:
			run = append(run, strokeSample{cx, cy, t})
			off = max(-2, min(2, off+max(-0.25, min(0.25, (cx-px)*nx+(cy-py)*ny))))
			thin++
			if isAlone {
				alone++
			}
			blank, crossed = 0, 0
		case run == nil:
		case ok:
			// Across an outline or a filled checker the stroke is hidden.
			if crossed++; crossed > maxGap {
				flush()
			}
		default:
			// A drawn stroke is continuous: a hatching crossed is not one.
			blanks++
			if blank++; blank > 2 {
				flush()
			}
		}
	}
	if run != nil {
		flush()
	}
	return out
}

func inkAt(m *mask, x, y float64) bool {
	xi, yi := roundHalfEven(x), roundHalfEven(y)
	return xi >= 0 && yi >= 0 && xi < m.w && yi < m.h && m.at(xi, yi)
}

// inpaintStrokes paints the strokes over with the grey found on either side
// of them, so neither the edges nor the inside of a checker see the stroke.
// Only the stroke itself is painted, where it runs at its own width: a
// sample is left within two pixels of a crossing (where the stroke leaves
// its line or thickens against another stroke; its two ends are its own),
// or when the grey beside it is ink; and of the pixels around a sample, only those of the stroke's
// width, or paper next to it, are painted. An outline the stroke crosses or
// passes close to is kept.
func inpaintStrokes(g *raster, ink *mask, strokes [][]strokeSample, dirs [][2]float64) {
	const margin = 2
	for k, st := range strokes {
		dx, dy := dirs[k][0], dirs[k][1]
		nx, ny := -dy, dx
		ts := make([]float64, len(st))
		for i, s := range st {
			ts[i] = float64(s.t)
		}
		own := median(ts) + 1
		// ok: the sample runs at the stroke's width, the next one a pixel on.
		ok := make([]bool, len(st))
		for i, s := range st {
			ok[i] = float64(s.t) <= own
			if i+1 < len(st) && math.Hypot(st[i+1].x-s.x, st[i+1].y-s.y) > 1.5 {
				ok[i] = false
			}
		}
		for i, s := range st {
			paint := true
			for j := max(i-margin, 0); j <= min(i+margin, len(st)-1) && paint; j++ {
				paint = (j == i+margin || ok[j]) && float64(st[j].t) <= own
			}
			if !paint {
				continue
			}
			h := float64(s.t) / 2
			ax, ay := s.x-(h+2)*nx, s.y-(h+2)*ny
			bx, by := s.x+(h+2)*nx, s.y+(h+2)*ny
			if !inBounds(g, ax, ay) || !inBounds(g, bx, by) || inkAt(ink, ax, ay) || inkAt(ink, bx, by) {
				continue
			}
			a := g.at(roundHalfEven(ax), roundHalfEven(ay))
			b := g.at(roundHalfEven(bx), roundHalfEven(by))
			r := int(math.Ceil(h + 2))
			cx, cy := roundHalfEven(s.x), roundHalfEven(s.y)
			for yy := cy - r; yy <= cy+r; yy++ {
				for xx := cx - r; xx <= cx+r; xx++ {
					if xx < 0 || yy < 0 || xx >= g.w || yy >= g.h {
						continue
					}
					ox, oy := float64(xx)-s.x, float64(yy)-s.y
					along, across := ox*dx+oy*dy, ox*nx+oy*ny
					if math.Abs(along) > 0.75 {
						continue
					}
					// The stroke's own pixels, or the paper's grey beside it.
					core := math.Abs(across) <= h+0.5
					if !core && (math.Abs(across) > h+1.5 || ink.at(xx, yy)) {
						continue
					}
					f := float32(min(max((across+h+2)/(2*h+4), 0), 1))
					g.p[yy*g.w+xx] = a + (b-a)*f
				}
			}
		}
	}
}

func inBounds(g *raster, x, y float64) bool {
	xi, yi := roundHalfEven(x), roundHalfEven(y)
	return xi >= 0 && yi >= 0 && xi < g.w && yi < g.h
}
