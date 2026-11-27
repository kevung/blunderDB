package structure

import (
	"math"
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

func TestArrowStrokeFoundAndPaintedOver(t *testing.T) {
	c, f := fieldCanvas()
	c.disc(80, 55, 24, 1)
	// A move from the checker's centre across two points of the top row.
	c.segment(80, 55, 300, 140, 2)
	g := canvasRaster(c)
	sts, dirs := arrowStrokes(hysteresis(g), f, 50)
	if len(sts) == 0 {
		t.Fatal("the arrow is not found")
	}
	inpaintStrokes(g, sts, dirs)
	for _, pt := range [][2]int{{150, 82}, {240, 117}, {90, 59}} {
		if v := g.at(pt[0], pt[1]); v < 0.9 {
			t.Errorf("(%d,%d) still %.2f after painting the stroke over", pt[0], pt[1], v)
		}
	}
	// The checker's outline away from the stroke is kept.
	if v := g.at(80, 55-23); v > 0.1 {
		t.Errorf("the checker's outline is painted over: %.2f", v)
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
