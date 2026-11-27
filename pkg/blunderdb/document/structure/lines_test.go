package structure

import (
	"context"
	"math"
	"math/rand"
	"testing"
)

// fieldCanvas draws an empty field at point width 50: its frame, the bar's
// two sides, and returns it with its frame.
func fieldCanvas() (*canvas, frame) {
	f := frame{a: 30, b: 330, c: 390, d: 690, y0: 30, y1: 560}
	c := newCanvas(720, 590)
	c.rect(f.a-2, f.y0-2, f.d+2, f.y0, 0)
	c.rect(f.a-2, f.y1, f.d+2, f.y1+2, 0)
	for _, x := range []int{f.a, f.b, f.c, f.d} {
		c.rect(x-1, f.y0, x+1, f.y1, 0)
	}
	return c, f
}

func (c *canvas) segment(x0, y0, x1, y1, w float64) {
	n := int(math.Hypot(x1-x0, y1-y0)) * 2
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n)
		x, y := x0+t*(x1-x0), y0+t*(y1-y0)
		for dy := -w / 2; dy <= w/2; dy += 0.5 {
			for dx := -w / 2; dx <= w/2; dx += 0.5 {
				c.set(int(math.Round(x+dx)), int(math.Round(y+dy)), 0)
			}
		}
	}
}

func canvasRaster(c *canvas) *raster {
	r := newRaster(c.w, c.h)
	for i, v := range c.p {
		r.p[i] = float32(v)
	}
	return r
}

// crossedOutline draws a light checker and a stroke across it, paints the
// stroke over and returns the outline's pixels lost and the stroke's pixels
// kept at the given points.
func crossedOutline(t *testing.T, cx, cy, r float64, x0, y0, x1, y1 float64, away [][2]int) {
	t.Helper()
	c, f := fieldCanvas()
	c.disc(cx, cy, r, 1)
	c.segment(x0, y0, x1, y1, 2)
	g := canvasRaster(c)
	before := canvasRaster(c)
	ink := hysteresis(g)
	sts, dirs := arrowStrokes(ink, f, 50)
	if len(sts) == 0 {
		t.Fatal("the arrow is not found")
	}
	inpaintStrokes(g, ink, sts, dirs)
	for _, pt := range away {
		if v := g.at(pt[0], pt[1]); v < 0.9 {
			t.Errorf("(%d,%d) still %.2f after painting the stroke over", pt[0], pt[1], v)
		}
	}
	// Every pixel of the outline, at the crossings first, is kept.
	lost := 0
	for y := int(cy - r - 2); y <= int(cy+r+2); y++ {
		for x := int(cx - r - 2); x <= int(cx+r+2); x++ {
			d := math.Hypot(float64(x)-cx, float64(y)-cy)
			if d > r-2 && d <= r && before.at(x, y) < 0.5 && g.at(x, y) > 0.5 {
				lost++
			}
		}
	}
	if lost > 0 {
		t.Errorf("%d pixels of the crossed outline painted over", lost)
	}
}

func TestArrowStrokeFoundAndPaintedOver(t *testing.T) {
	// A move from the checker's centre across two points of the top row:
	// it crosses the outline at about 20 degrees.
	crossedOutline(t, 80, 55, 24, 80, 55, 300, 140, [][2]int{{150, 82}, {240, 117}, {90, 59}})
}

func TestArrowThroughCheckerKeepsOutline(t *testing.T) {
	for _, c := range []struct {
		name           string
		x0, y0, x1, y1 float64
	}{
		{"45", 120, 220, 290, 390},
		{"30", 100, 242, 300, 358},
		{"60", 150, 213, 250, 387},
	} {
		t.Run(c.name, func(t *testing.T) {
			crossedOutline(t, 200, 300, 24, c.x0, c.y0, c.x1, c.y1, nil)
		})
	}
}

func TestHatchingIsNoArrow(t *testing.T) {
	c, f := fieldCanvas()
	// Hatched points, every other one, from the base: short parallel strokes
	// 4 pixels apart, slanted like an arrow could be.
	for j := 0; j < 6; j += 2 {
		x0 := float64(f.a + 50*j)
		for k := 0.0; k < 240; k += 4 {
			c.segment(x0, float64(f.y0)+k, x0+48, float64(f.y0)+k+30, 1)
		}
	}
	if sts, _ := arrowStrokes(hysteresis(canvasRaster(c)), f, 50); len(sts) != 0 {
		t.Errorf("%d strokes found in a hatching, first from (%.0f,%.0f)", len(sts), sts[0][0].x, sts[0][0].y)
	}
}

func TestArrowLevelOrUprightIsKept(t *testing.T) {
	c, f := fieldCanvas()
	c.segment(60, 300, 320, 300, 2)
	c.segment(450, 60, 452, 500, 2)
	if sts, _ := arrowStrokes(hysteresis(canvasRaster(c)), f, 50); len(sts) != 0 {
		t.Errorf("%d strokes taken along the frame's rows or the points' sides", len(sts))
	}
}

func TestTrayCountsTeachNoPipPlace(t *testing.T) {
	if !trayCounts([]sideNum{{13, 100}, {3, 500}}) {
		t.Error("13 and 3 beside the trays are taken for pips")
	}
	if trayCounts([]sideNum{{13, 100}, {46, 500}}) {
		t.Error("a pip count of 46 is taken for a tray's count")
	}
}

// TestArrowsReadEndToEnd reads a book whose diagrams draw each move as an
// arrow from the outermost checker of a light stack toward the middle of
// the field: the stroke inside the checker must not cost it its place. An
// arrow laid over a point filled in grey is not seen (the stroke does not
// run alone there): the arrows here leave the points' bases.
func TestArrowsReadEndToEnd(t *testing.T) {
	r := rand.New(rand.NewSource(7))
	var pages memPages
	var want []string
	for pg := 0; pg < 12; pg++ {
		c := newCanvas(1275, 1650)
		var words []Word
		for i := 0; i < 2; i++ {
			s := randomPos(r)
			o := plainOpts(36)
			ox, oy := 300, 200+i*int(14*o.p)
			words = append(words, c.draw(s, ox, oy, o)...)
			want = append(want, s.board())
			p, rad := o.p, 0.48*o.p
			a := float64(ox + 3)
			y0, y1 := float64(oy+3), float64(oy+3)+float64(int(11*p))
			arrows := 0
			for q := 1; q <= 24 && arrows < 2; q++ {
				// A stack of three or more: the arrow leaves the points'
				// filled bases at once and runs over the paper.
				n := s.top[q]
				if n < 3 {
					continue
				}
				top, j := q <= 12, q-13
				if top {
					j = 12 - q
				}
				x := a + float64(j%6)*p + p/2
				if j >= 6 {
					x += 7 * p
				}
				y, dir := y1-rad-float64(n-1)*2*rad, -1.0
				if top {
					y, dir = y0+rad+float64(n-1)*2*rad, 1
				}
				dx := math.Cos(65 * math.Pi / 180)
				if j%6 >= 3 {
					dx = -dx
				}
				L := 6 * p
				c.segment(x, y, x+dx*L, y+dir*math.Sin(65*math.Pi/180)*L, 2)
				arrows++
			}
		}
		pages = append(pages, Page{Image: c.image(), DPI: 150, Words: words})
	}
	res, err := Read(context.Background(), pages)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Diagrams) != len(want) {
		t.Fatalf("%d diagrams, want %d", len(res.Diagrams), len(want))
	}
	// An arrow not followed leaves its diagram with a named doubt; never a
	// sure reading that is wrong. The one doubt here is diagram 12: its two
	// arrows run back to back along nearly the same line, each tip inside the
	// other's checker, and where they overlap the ink is wider than one
	// stroke. Drawn alone, either arrow is painted over and the diagram read.
	exact := 0
	for i, d := range res.Diagrams {
		switch {
		case d.Board == want[i] && d.State == Confirmed:
			exact++
		case d.State == Confirmed:
			t.Errorf("diagram %d: want %s, got a wrong sure %s", i, want[i], describe(d))
		}
	}
	if exact < len(want)-1 {
		t.Errorf("%d diagrams of %d read exactly and sure", exact, len(want))
	}
}

// TestTrayCountsAreNotPips: a book prints its pips in the image and, in the
// text layer, the checkers borne off beside each board's trays (13 and 3).
// Those counts do not teach where the pips are printed: the image's pips
// make the diagrams sure.
func TestTrayCountsAreNotPips(t *testing.T) {
	src := inkBook(t, 0)
	p := 36.0
	right := 300 + 6 + 13*p + 1.5*p
	for i := range src {
		for k := 0; k < 2; k++ {
			oy := 200 + float64(k)*float64(int(14*p)) + 3
			for _, w := range []struct {
				s string
				y float64
			}{{"13", oy + 2*p}, {"3", oy + 9*p}} {
				src[i].Words = append(src[i].Words, Word{right - 5, w.y - 7, right + 5, w.y + 7, w.s})
			}
		}
	}
	res, err := Read(context.Background(), src)
	if err != nil {
		t.Fatal(err)
	}
	if res.Profile.PipPlace != 0 {
		t.Errorf("pips taken as printed at %q", res.Profile.PipPlace)
	}
	sure := 0
	for _, d := range res.Diagrams {
		if d.State == Confirmed {
			sure++
		}
	}
	if sure < len(res.Diagrams)*3/4 {
		t.Errorf("%d sure of %d", sure, len(res.Diagrams))
	}
}
