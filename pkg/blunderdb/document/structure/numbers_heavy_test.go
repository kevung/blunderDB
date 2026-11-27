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
			ts := append(fontTemplates(), template{digit: 1, v: unit(one), learned: true, weight: strokeWeight(one, 255)})
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

func heavyUnit(v []float64) []float64 {
	m := append([]float64(nil), v...)
	for i := range m {
		m[i] /= 255
	}
	return unitVector(m)
}

// TestLearnedRivalIsNoAlibi: a rival figure the document also taught does
// not make a learned win fair — the light 2 of a numbering against a heavy
// 1 learned elsewhere — nor does a 1 already fed one misread 2.
func TestLearnedRivalIsNoAlibi(t *testing.T) {
	oneFig := heavyFigure(t, "1", 18, 2)
	one, oneW := heavyUnit(oneFig), strokeWeight(oneFig, 255)
	light2 := fontTemplates()[2]
	light2.learned = true
	ts := append(fontTemplates(), template{digit: 1, v: one, learned: true, weight: oneW}, light2)
	two := heavyFigure(t, "2", 18, 2)
	h, hy := squareHoles(two)
	if readFigure(ts, heavyUnit(two), h, hy) == 1 {
		t.Error("18 pt, thickened 2: a heavy 2 read 1 against a taught light 2")
	}
	// The learned 1 fed a heavy 2 as if it were a 1.
	fed := centred(meanOf([][]float64{one, heavyUnit(two)}))
	fedW := (oneW + strokeWeight(two, 255)) / 2
	ts = append(fontTemplates(), template{digit: 1, v: fed, learned: true, weight: fedW}, light2)
	other := heavyFigure(t, "2", 26, 1)
	h, hy = squareHoles(other)
	if readFigure(ts, heavyUnit(other), h, hy) == 1 {
		t.Error("26 pt, thickened 1: a heavy 2 read 1 by a 1 fed a misread 2")
	}
}

// TestMisreadNeverTeaches: over readNumbers' rounds, heavy 1s alone teach
// the document's 1, light 2s its 2, and heavy 2s alone come to be compared
// with both; a heavy 2 is never read 1, so never fed back into the 1 it
// would bend for the next round.
func TestMisreadNeverTeaches(t *testing.T) {
	var all [][]printedNumber
	add := func(s string, size float64, thicken int) {
		var q [glyphSide * glyphSide]uint8
		for k, x := range heavyFigure(t, s, size, thicken) {
			q[k] = uint8(x)
		}
		h := int(size)
		y := 40 * len(all)
		all = append(all, []printedNumber{{x0: 0, y0: y, x1: h, y1: y + h, h: h, alone: true, glyphs: [][glyphSide * glyphSide]uint8{q}, val: -1}})
	}
	for _, size := range []float64{24, 26, 28, 30, 32, 34} {
		for thicken := 1; thicken <= 2; thicken++ {
			for range 3 {
				add("1", size, thicken)
			}
		}
	}
	for _, size := range []float64{18, 20, 22} {
		for range 3 {
			add("2", size, 0)
		}
	}
	type drawn struct{ size, thicken int }
	var twos []drawn
	for _, size := range []float64{18, 18, 18, 20, 22, 24, 26} {
		for thicken := 2; thicken <= 3; thicken++ {
			add("2", size, thicken)
			twos = append(twos, drawn{int(size), thicken})
		}
	}
	readNumbers(all)
	if testing.Verbose() {
		var vs []int
		for _, n := range all {
			vs = append(vs, n[0].val)
		}
		t.Log(vs)
	}
	for i, d := range twos {
		if n := all[len(all)-len(twos)+i][0]; n.val == 1 {
			t.Errorf("%d pt, thickened %d: a heavy 2 read 1", d.size, d.thicken)
		}
	}
}

// TestTypefaceBoldIsNoAlibi: a typeface's bold 2, of the learned heavy 1's
// weight, does not make the 1's win fair — it says nothing of how this
// document draws its 2s — so a heavy 2 is still not read 1.
func TestTypefaceBoldIsNoAlibi(t *testing.T) {
	oneFig := heavyFigure(t, "1", 16, 0)
	oneW := strokeWeight(oneFig, 255)
	bold := false
	for _, f := range fontTemplates() {
		bold = bold || (f.digit == 2 && likeWeight(f.weight, oneW))
	}
	if !bold {
		t.Fatal("no typeface 2 of the heavy 1's weight: the case is not posed")
	}
	ts := append(fontTemplates(), template{digit: 1, v: heavyUnit(oneFig), learned: true, weight: oneW})
	two := heavyFigure(t, "2", 16, 2)
	h, hy := squareHoles(two)
	if readFigure(ts, heavyUnit(two), h, hy) == 1 {
		t.Error("16 pt, thickened 2: a heavy 2 read 1 against a typeface bold 2")
	}
}

// TestTenTaughtFiguresKeepTheGuard: a numbering that taught all ten figures,
// light, does not lift the guard — its 2 is not of a heavy 1's weight — so a
// heavy 2 is not read 1, and the heavy 1 itself is read as it is without
// the numbering.
func TestTenTaughtFiguresKeepTheGuard(t *testing.T) {
	var numbering []template
	for _, f := range fontTemplates()[:10] {
		f.learned = true
		numbering = append(numbering, f)
	}
	for _, size := range []float64{16, 18, 20, 22, 24, 26, 28} {
		for _, thicken := range []int{2, 3} {
			oneFig := heavyFigure(t, "1", size, thicken)
			one := template{digit: 1, v: heavyUnit(oneFig), learned: true, weight: strokeWeight(oneFig, 255)}
			ts := append(append(fontTemplates(), numbering...), one)
			two := heavyFigure(t, "2", size, thicken)
			h, hy := squareHoles(two)
			if readFigure(ts, heavyUnit(two), h, hy) == 1 {
				t.Errorf("%v pt, thickened %d: a heavy 2 read 1 though all ten figures were taught", size, thicken)
			}
			h, hy = squareHoles(oneFig)
			alone := readFigure(append(fontTemplates(), one), heavyUnit(oneFig), h, hy)
			if d := readFigure(ts, heavyUnit(oneFig), h, hy); d != alone {
				t.Errorf("%v pt, thickened %d: the heavy 1 learned read %d, %d without the numbering", size, thicken, d, alone)
			}
		}
	}
}
