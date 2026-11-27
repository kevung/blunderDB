package structure

import (
	"image"
	"image/draw"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// heavyFigure draws the figure s in Go Bold, size points, its stroke grown
// by thicken pixels a side (a scan's ink spread), and gives its normalised
// square as a figure read from a page is.
func heavyFigure(t *testing.T, s string, size float64, thicken int) []float64 {
	t.Helper()
	f, err := opentype.Parse(gobold.TTF)
	if err != nil {
		t.Fatal(err)
	}
	face, err := opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewGray(image.Rect(0, 0, 100, 100))
	draw.Draw(img, img.Bounds(), image.White, image.Point{}, draw.Src)
	d := font.Drawer{Dst: img, Src: image.Black, Face: face, Dot: fixed.P(30, 60)}
	d.DrawString(s)
	for k := 0; k < thicken; k++ {
		c := image.NewGray(img.Bounds())
		copy(c.Pix, img.Pix)
		for y := 1; y < 99; y++ {
			for x := 1; x < 99; x++ {
				if img.GrayAt(x-1, y).Y < 128 || img.GrayAt(x+1, y).Y < 128 || img.GrayAt(x, y-1).Y < 128 || img.GrayAt(x, y+1).Y < 128 {
					c.Pix[y*c.Stride+x] = 0
				}
			}
		}
		img = c
	}
	g := normalize(fromImage(img))
	lab, comps := label8(hysteresis(g))
	if len(comps) != 1 {
		t.Fatalf("%q drawn in %d pieces", s, len(comps))
	}
	c := comps[0]
	return normSquare(c.w(), c.h(), func(x, y int) bool { return lab[(c.y0+y)*g.w+c.x0+x] == 1 })
}

// TestHeavyTwoNotReadAsOne: in a document whose own heavy 1s were learned
// and whose 2s were not, a heavy 2 is read 2 or not read, never 1 — the
// learned 1 beats the typefaces' light 2 only by its weight — while the
// heavy 1 itself is still read, short of a blot (thickened by 3 pixels, where
// the typefaces no longer tell a 1 from its neighbours).
func TestHeavyTwoNotReadAsOne(t *testing.T) {
	for _, thicken := range []int{1, 2, 3} {
		for _, size := range []float64{18, 22, 26, 30, 36} {
			unit := func(v []float64) []float64 {
				m := append([]float64(nil), v...)
				for i := range m {
					m[i] /= 255
				}
				return unitVector(m)
			}
			one := heavyFigure(t, "1", size, thicken)
			ts := append(fontTemplates(), template{digit: 1, v: unit(one), learned: true})
			two := heavyFigure(t, "2", size, thicken)
			h, hy := squareHoles(two)
			if d := readFigure(ts, unit(two), h, hy); d == 1 {
				t.Errorf("size %v, thickened %d: a heavy 2 read 1", size, thicken)
			}
			h, hy = squareHoles(one)
			if d := readFigure(ts, unit(one), h, hy); d != 1 && thicken < 3 {
				t.Errorf("size %v, thickened %d: the heavy 1 learned read %d", size, thicken, d)
			}
		}
	}
}
