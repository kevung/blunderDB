package structure

import (
	"strings"
	"testing"
)

// handBoard is a diagram whose field spans x 100 to 750 and y 100 to 650 on
// page pg, shifted down by dy: a point 50 pixels wide, a cube beside the
// field at x 770 to 815.
func handBoard(pg int, dy float64, cubes ...square) *boardReading {
	rd := &boardReading{
		field: [6]float64{100, 400, 450, 750, 100 + dy, 650 + dy},
		bbox:  box{80, 80 + int(dy), 770, 670 + int(dy)},
		page:  pg,
	}
	for _, s := range cubes {
		s.y0, s.y1 = s.y0+dy, s.y1+dy
		rd.cubes = append(rd.cubes, s)
	}
	return rd
}

// cubeAt is a square of aspect 1 beside the field's right edge, at place.
func cubeAt(place string, blank bool) square {
	y := map[string]float64{"top": 105, "middle": 355, "bottom": 600}[place]
	return square{aspect: 1, x0: 770, x1: 815, y0: y, y1: y + 45, place: place, blank: blank}
}

// cubeDoubt reads the cube of each diagram against its document and gives
// its context and the reason of its cube doubt.
func cubeDoubt(rds []*boardReading) ([]Context, []string) {
	envs := contextEnvs(rds)
	var cs []Context
	var why []string
	for i, rd := range rds {
		var d Diagram
		readContext(rd, &d, nil, Profile{}, envs[i])
		cs = append(cs, d.Context)
		why = append(why, doubtsOn(d)["cube"])
	}
	return cs, why
}

// TestKeepSquaresStretch: a square of aspect 1.5 is no cube where the
// checkers are round, and a cube where they measure the same stretch; a
// square drawn square is then no cube.
func TestKeepSquaresStretch(t *testing.T) {
	long := cubeAt("top", true)
	long.aspect = 1.5
	for _, c := range []struct {
		aspect, stretch float64
		kept            bool
	}{{1.5, 1, false}, {1.5, 1.4, true}, {1, 1.4, false}, {1, 1, true}} {
		s := long
		s.aspect = c.aspect
		rd := handBoard(0, 0, s)
		keepSquares([]*boardReading{rd}, c.stretch)
		if got := len(rd.cubes) == 1; got != c.kept {
			t.Errorf("aspect %.2f, stretch %.2f: kept %v, want %v", c.aspect, c.stretch, got, c.kept)
		}
	}
}

// TestKeepSquaresNearer: a square nearer another diagram of the page is
// that diagram's, not the one it was found about.
func TestKeepSquaresNearer(t *testing.T) {
	near := square{aspect: 1, x0: 770, x1: 815, y0: 760, y1: 800, place: "bottom"}
	a, b := handBoard(0, 0, cubeAt("top", false), near), handBoard(0, 700)
	keepSquares([]*boardReading{a, b}, 1)
	if len(a.cubes) != 1 || a.cubes[0].place != "top" {
		t.Errorf("squares kept about the first diagram: %+v", a.cubes)
	}
	other := handBoard(1, 700)
	a = handBoard(0, 0, near)
	keepSquares([]*boardReading{a, other}, 1)
	if len(a.cubes) != 1 {
		t.Errorf("a square claimed by a diagram of another page")
	}
}

// TestBareFace: a faint figure on a washed-out scan is a mark; an outline
// too pale against its paper tells nothing, so the face is not bare.
func TestBareFace(t *testing.T) {
	face := func(paper, ink, figure float64) bool {
		g := newRaster(60, 60)
		for i := range g.p {
			g.p[i] = float32(paper)
		}
		set := func(x0, y0, x1, y1 int, v float64) {
			for y := y0; y < y1; y++ {
				for x := x0; x < x1; x++ {
					g.p[y*g.w+x] = float32(v)
				}
			}
		}
		set(10, 10, 50, 50, ink)
		set(13, 13, 47, 47, paper)
		if figure >= 0 {
			set(26, 22, 34, 38, figure)
		}
		return bareFace(g, box{10, 10, 50, 50})
	}
	if !face(0.85, 0.55, -1) {
		t.Error("a washed-out empty face not bare")
	}
	if face(0.85, 0.55, 0.65) {
		t.Error("a faint figure on a washed-out face read bare")
	}
	if face(0.9, 0.75, -1) {
		t.Error("an outline too pale to judge read bare")
	}
	if face(1, 0, 0) {
		t.Error("a black figure read bare")
	}
}

// TestBareCubePlaces: three lone bare cubes, each at its own place, are no
// drawing the document repeats; three at one place where another diagram
// prints a value are figures the scan may have lost; a 64 printed there
// stops the reading only off the middle, where it is an owned cube.
func TestBareCubePlaces(t *testing.T) {
	cs, why := cubeDoubt([]*boardReading{
		handBoard(0, 0, cubeAt("top", true)),
		handBoard(0, 700, cubeAt("bottom", true)),
		handBoard(1, 0, cubeAt("middle", true)),
	})
	for i := range cs {
		if cs[i].CubeRead || !strings.Contains(why[i], "beside 0 other") {
			t.Errorf("diagram %d: a bare cube at its own place read %+v; doubt %q", i+1, cs[i], why[i])
		}
	}
	three := func(fourth *boardReading) ([]Context, []string) {
		return cubeDoubt([]*boardReading{
			handBoard(0, 0, cubeAt("top", true)),
			handBoard(0, 700, cubeAt("top", true)),
			handBoard(1, 0, cubeAt("top", true)),
			fourth,
		})
	}
	cs, why = three(handBoard(1, 700, cubeAt("top", false)))
	for i := range 3 {
		if cs[i].CubeRead || !strings.Contains(why[i], "print a value") {
			t.Errorf("diagram %d: a bare cube where another prints a value read %+v; doubt %q", i+1, cs[i], why[i])
		}
	}
	sixtyFour := func(place string) *boardReading {
		rd := handBoard(1, 700, cubeAt(place, false))
		s := rd.cubes[0]
		rd.cubeFigs = []printedNumber{{x0: int(s.x0) + 10, y0: int(s.y0) + 10, x1: int(s.x1) - 10, y1: int(s.y1) - 10, val: 64}}
		return rd
	}
	middle := func(fourth *boardReading) ([]Context, []string) {
		return cubeDoubt([]*boardReading{
			handBoard(0, 0, cubeAt("middle", true)),
			handBoard(0, 700, cubeAt("middle", true)),
			handBoard(1, 0, cubeAt("middle", true)),
			fourth,
		})
	}
	cs, why = middle(sixtyFour("middle"))
	for i := range 3 {
		if !cs[i].CubeRead || cs[i].Cube != 1 || cs[i].CubeOwner != 0 || why[i] != "" {
			t.Errorf("diagram %d: a bare cube beside a centred 64 at its place read %+v; doubt %q", i+1, cs[i], why[i])
		}
	}
	// A 64 at the top is an owned cube showing its value: the bare faces
	// at that place may be values the scan lost.
	cs, why = three(sixtyFour("top"))
	for i := range 3 {
		if cs[i].CubeRead || !strings.Contains(why[i], "print a value") {
			t.Errorf("diagram %d: a bare cube beside an owned 64 at its place read %+v; doubt %q", i+1, cs[i], why[i])
		}
	}
	cs, _ = three(handBoard(1, 700, cubeAt("bottom", false)))
	for i := range 3 {
		if !cs[i].CubeRead || cs[i].Cube != 1 {
			t.Errorf("diagram %d: a value printed at another place stopped the reading: %+v", i+1, cs[i])
		}
	}
}

// TestDiceDecision: dice counted are read when the text cites that roll
// twice among others or alone once, a doubt when cited once beside other
// rolls.
func TestDiceDecision(t *testing.T) {
	ws := func(ts ...string) []Word {
		var out []Word
		for i, x := range ts {
			out = append(out, Word{X0: float64(120 + 40*i), X1: float64(150 + 40*i), Y0: 690, Y1: 700, Text: x})
		}
		return out
	}
	for _, c := range []struct {
		words []string
		read  bool
		why   string
	}{
		{[]string{"Dark", "rolls", "a", "41,", "Light", "rolls", "a", "65."}, false, "cited once beside other rolls"},
		{[]string{"Dark", "rolls", "a", "41"}, true, ""},
		{[]string{"rolls", "41,", "rolls", "65,", "plays", "41"}, true, ""},
		{[]string{"rolls", "a", "65"}, false, "the text says otherwise"},
	} {
		rd := handBoard(0, 0)
		rd.dice = []square{
			{aspect: 1, x0: 500, x1: 540, y0: 355, y1: 395, pips: 4, dark: true},
			{aspect: 1, x0: 560, x1: 600, y0: 355, y1: 395, pips: 1, dark: true},
		}
		var d Diagram
		readContext(rd, &d, ws(c.words...), Profile{BottomIsDark: true}, contextEnv{})
		why := doubtsOn(d)["dice"]
		if d.Context.DiceRead != c.read || !strings.Contains(why, c.why) || (c.read && why != "") {
			t.Errorf("%v: dice read %v, doubt %q", c.words, d.Context.DiceRead, why)
		}
	}
}
