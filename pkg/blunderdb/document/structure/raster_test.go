package structure

import (
	"math"
	"math/rand"
	"testing"
)

func maskOf(rows ...string) *mask {
	m := newMask(len(rows[0]), len(rows))
	for y, r := range rows {
		for x, c := range r {
			m.p[y*m.w+x] = c == '#'
		}
	}
	return m
}

func TestHysteresis(t *testing.T) {
	g := newRaster(8, 3)
	for i := range g.p {
		g.p[i] = 1
	}
	// a strong pixel with a weak tail; an isolated weak pixel
	g.p[0], g.p[1], g.p[2] = 0.3, 0.8, 0.8
	g.p[2*8+7] = 0.7
	m := hysteresis(g)
	if !m.p[0] || !m.p[1] || !m.p[2] || m.p[3] || m.p[2*8+7] {
		t.Errorf("hysteresis %v", m.p)
	}
}

func TestLabelOrderAndBoxes(t *testing.T) {
	m := maskOf(
		"..##....",
		"..#...#.",
		"......#.",
		"#.......",
	)
	_, bs, areas := label(m)
	want := []box{{2, 0, 4, 2}, {6, 1, 7, 3}, {0, 3, 1, 4}}
	if len(bs) != 3 {
		t.Fatalf("%d components", len(bs))
	}
	for i, b := range bs {
		if b != want[i] {
			t.Errorf("component %d: %+v, want %+v", i+1, b, want[i])
		}
	}
	if areas[0] != 3 || areas[1] != 2 || areas[2] != 1 {
		t.Errorf("areas %v", areas)
	}
	// 4-connectivity: a diagonal is two components.
	if _, bs, _ := label(maskOf("#.", ".#")); len(bs) != 2 {
		t.Errorf("diagonal: %d components", len(bs))
	}
}

func TestLongestRunsCloseGaps(t *testing.T) {
	m := maskOf("###..###.....#")
	if r := longestRuns(m, false)[0]; r != 9 {
		t.Errorf("run %d, want 9 (gap of two closed, inner end grown)", r)
	}
	m = maskOf("##...##")
	if r := longestRuns(m, false)[0]; r != 3 {
		t.Errorf("run %d, want 3 (a gap of three stays open)", r)
	}
}

func TestWindowAndUniformFilters(t *testing.T) {
	r := rand.New(rand.NewSource(3))
	w, h, k := 13, 9, 5
	v := make([]float64, w*h)
	for i := range v {
		v[i] = r.Float64()
	}
	mx := windowFilter(v, w, h, k, true)
	mn := windowFilter(v, w, h, k, false)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			bmax, bmin := math.Inf(-1), math.Inf(1)
			for yy := max(y-k/2, 0); yy <= min(y+k/2, h-1); yy++ {
				for xx := max(x-k/2, 0); xx <= min(x+k/2, w-1); xx++ {
					bmax, bmin = max(bmax, v[yy*w+xx]), min(bmin, v[yy*w+xx])
				}
			}
			if mx[y*w+x] != bmax || mn[y*w+x] != bmin {
				t.Fatalf("(%d,%d): %v %v, want %v %v", x, y, mx[y*w+x], mn[y*w+x], bmax, bmin)
			}
		}
	}
	// a mirrored border: the mean of a constant stays the constant, and of
	// a ramp at its first sample uses the mirror (0,0,1 | 0,1,2 ...).
	ramp := []float64{0, 1, 2, 3, 4}
	u := uniformFilter(ramp, 5, 1, 3)
	if math.Abs(u[0]-1.0/3) > 1e-6 || math.Abs(u[2]-2) > 1e-6 || math.Abs(u[4]-11.0/3) > 1e-6 {
		t.Errorf("uniform %v", u)
	}
}

func TestResample(t *testing.T) {
	v := []float64{0, 100, 200, 100}
	if out := resample(v, 4, 1, 4, 1, triangle, 1, false); out[1] != 100 {
		t.Errorf("same size changed values: %v", out)
	}
	up := resample(v, 4, 1, 8, 1, triangle, 1, false)
	if up[0] != 0 || math.Abs(up[1]-25) > 1e-4 || math.Abs(up[2]-75) > 1e-4 || up[7] != 100 {
		t.Errorf("bilinear x2: %v", up)
	}
	down := resample([]float64{10, 10, 10, 10, 10, 10}, 6, 1, 2, 1, lanczos, 3, true)
	if down[0] != 10 || down[1] != 10 {
		t.Errorf("lanczos of a constant: %v", down)
	}
}

func TestNormalizeEffacesShadow(t *testing.T) {
	g := newRaster(1000, 120)
	for y := 0; y < g.h; y++ {
		for x := 0; x < g.w; x++ {
			g.p[y*g.w+x] = float32(0.6 + 0.4*float64(x)/float64(g.w)) // binding shadow
		}
	}
	for x := 20; x < 980; x++ {
		g.p[60*g.w+x] *= 0.3 // a stroke across it
	}
	n := normalize(g)
	if v := n.at(10, 10); v < 0.95 {
		t.Errorf("paper in the shadow %v, want white", v)
	}
	if v := n.at(30, 60); v > 0.4 {
		t.Errorf("stroke in the shadow %v, want ink", v)
	}
}

func TestSkewAndRotate(t *testing.T) {
	g := newRaster(300, 200)
	for i := range g.p {
		g.p[i] = 1
	}
	for _, y0 := range []int{40, 100, 160} {
		for x := 0; x < 300; x++ {
			y := y0 + int(math.Round(float64(x-150)*math.Tan(1.5*math.Pi/180)))
			for d := 0; d < 2; d++ {
				g.p[(y+d)*g.w+x] = 0
			}
		}
	}
	m := hysteresis(g)
	a := skew(m)
	if math.Abs(a-1.5) > 0.06 {
		t.Fatalf("skew %v, want 1.5", a)
	}
	straight := rotate(g, a)
	if a2 := skew(hysteresis(straight)); math.Abs(a2) > 0.2 {
		t.Errorf("after rotation the skew is %v", a2)
	}
	if r := rotate(g, 0.01); r.p[0] != g.p[0] || &r.p[0] == &g.p[0] {
		t.Errorf("a negligible angle must copy the raster unchanged")
	}
}

func TestPointNumber(t *testing.T) {
	for _, c := range []struct {
		top  bool
		j    int
		home string
		want int
	}{
		{false, 11, "BR", 1}, {false, 0, "BR", 12}, {true, 0, "BR", 13}, {true, 11, "BR", 24},
		{false, 0, "BL", 1}, {true, 11, "TR", 1}, {false, 11, "TR", 24}, {true, 0, "TL", 1},
	} {
		if got := pointNumber(c.top, c.j, c.home); got != c.want {
			t.Errorf("pointNumber(%v,%d,%s) = %d, want %d", c.top, c.j, c.home, got, c.want)
		}
	}
}

func TestReadingOrderSpread(t *testing.T) {
	z := func(x0, y0 int) zone { return zone{bbox: box{x0, y0, x0 + 300, y0 + 250}} }
	// right page first in the input; two columns on the left page
	got := readingOrder([]zone{z(1400, 100), z(500, 600), z(100, 600), z(100, 100)}, 2000)
	want := []int{100, 100, 500, 1400}
	wantY := []int{100, 600, 600, 100}
	for i, g := range got {
		if g.bbox.x0 != want[i] || g.bbox.y0 != wantY[i] {
			t.Errorf("rank %d: %+v", i+1, g.bbox)
		}
	}
}

func TestTextFeaturesPipsAtBar(t *testing.T) {
	b := box{0, 100, 1300, 1100}
	w := func(s string, cx, cy float64) Word { return Word{cx - 10, cy - 10, cx + 10, cy + 10, s} }
	var ws []Word
	for j := 0; j < 12; j++ {
		cx := 50 + float64(j)*100
		if j >= 6 {
			cx += 100
		}
		ws = append(ws, w(itoa(13+j), cx, 60))
	}
	ws = append(ws, w("57", 650, 60))
	ws = append(ws, w("134", -150, 300), w("97", -150, 900))
	num, side := textFeatures(b, ws)
	if len(side['B']) != 1 || side['B'][0].v != 57 {
		t.Errorf("pips at the bar %v", side['B'])
	}
	if len(side['G']) != 2 || len(num) != 12 {
		t.Errorf("left %v, numbering %d", side['G'], len(num))
	}
}

func itoa(n int) string { return string(rune('0'+n/10)) + string(rune('0'+n%10)) }

func TestReadColumn(t *testing.T) {
	cl := classes{c0: 0.2, c1: 1, t0: 0.08, t1: 0.08, split: 0.6, ok: true}
	d := func(m float64) disc { return disc{mean: m} }
	dark := []float64{0.2, 0.2, 0.2, 0.2, 1, 1}
	// a dark stack of four seen as two, counted by its length
	ct := column{top: true, discs: []disc{d(0.2), d(0.2)}, slotMeans: dark}
	out, doubt, _, how := readColumn(ct, column{slotMeans: []float64{1, 1}}, cl, 9)
	if doubt != "" || out[0] == nil || out[0].n != 4 || len(how) != 1 || how[0].kind != "length" {
		t.Errorf("length: %+v %q %+v", out[0], doubt, how)
	}
	// one chain from top to bottom, shared at its change of class
	chain := func(cs ...float64) column {
		c := column{slotMeans: []float64{1}}
		for _, m := range cs {
			c.discs = append(c.discs, d(m))
		}
		return c
	}
	out, doubt, _, _ = readColumn(chain(0.2, 0.2, 0.2, 1, 1, 1), chain(1, 1, 1, 0.2, 0.2, 0.2), cl, 9)
	if doubt != "" || out[0].n != 3 || out[0].class != 0 || out[1].n != 3 || out[1].class != 1 {
		t.Errorf("share: %+v %+v %q", out[0], out[1], doubt)
	}
	out, doubt, _, _ = readColumn(chain(0.2, 1), chain(), cl, 9)
	if doubt != "mixed classes" {
		t.Errorf("mixed: %q %+v", doubt, out)
	}
	_, doubt, _, _ = readColumn(chain(0.2, 0.45), chain(), cl, 9)
	if doubt != "" {
		t.Errorf("a disc off classes on top is dropped, got %q", doubt)
	}
	_, doubt, _, _ = readColumn(chain(0.45, 0.2), chain(), cl, 9)
	if doubt != "off classes" {
		t.Errorf("a disc off classes under a checker: %q", doubt)
	}
}

func TestSolveByPips(t *testing.T) {
	a := assembled{pts: [2]map[int]int{{6: 4, 8: 3, 13: 5, 24: 2}, {6: 5, 8: 3, 13: 5, 24: 2}}}
	pr := &[2]int{pipsOf(a.pts, a.bar, 0) + 6, pipsOf(a.pts, a.bar, 1)}
	pts, k, pt, add, fifteen, ok := solveByPips(a, pr, "top=class0")
	if !ok || k != 0 || pt != 6 || add != 1 || !fifteen || pts[0][6] != 5 {
		t.Errorf("solve: %v %d %d %d %v %v", pts, k, pt, add, fifteen, ok)
	}
	// 12 short with two high stacks (the 6 and the 12 point, three each):
	// two pips solutions, so none is taken.
	b := assembled{pts: [2]map[int]int{{6: 3, 12: 3, 13: 5, 24: 2}, a.pts[1]}}
	pr2 := &[2]int{pipsOf(b.pts, b.bar, 0) + 12, pipsOf(b.pts, b.bar, 1)}
	if _, _, _, _, _, ok := solveByPips(b, pr2, "top=class0"); ok {
		t.Errorf("two stacks explain the gap: must not solve")
	}
	if _, _, _, _, _, ok := solveByPips(a, pr, ""); ok {
		t.Errorf("without the attribution of the printed pips, nothing is solved")
	}
}
