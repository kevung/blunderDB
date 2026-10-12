package capture

// class is what a sampled box shows.
type class uint8

const (
	clsOther class = iota
	clsCheckerA
	clsCheckerB
	clsLightTri
	clsDarkTri
	clsField
)

// swatch is one colour a theme draws, measured on real captures.
type swatch struct {
	cls class
	rgb [3]float64
}

// theme describes one program's board in units of the point width p, from the
// top-left corner of the left playing field. Every length below is in p.
type theme struct {
	software Software
	// barW is the bar's width; fieldH the playing area's height (top edge of
	// the upper points to bottom edge of the lower ones).
	barW, fieldH float64
	// margin is the gap between a field edge and the first checker; step the
	// distance between two stacked checkers' centres; checker the diameter.
	margin, step, checker float64
	// triSlots is how many stacked slots, from the base, the triangle covers
	// at its centre line: below it an empty slot shows the triangle's colour,
	// above it the field's.
	triSlots int
	// slots is how many checkers a point shows before the program draws them
	// overlapped or with a number.
	slots int
	// firstLight says that the leftmost upper point is the light triangle.
	firstLight bool
	// palette holds the colours, checkers first; maxDist is the colour
	// distance beyond which a box matches no swatch.
	palette []swatch
	maxDist float64
	// bottomIsA says that by default the checkers of colour A belong to the
	// player at the bottom of the screen (blunderDB's Black).
	bottomIsA bool
	// homeLeft says that by default the bottom player's home board is the
	// lower-left quarter; the point labels, when the capture shows them,
	// override it.
	homeLeft bool
	// trayW is the width of the tray beyond the right field where the cube
	// and the borne-off checkers sit; zero when the program has none there.
	trayW float64
	// hinges are the heights, top down, where the bar shows a hinge in a
	// checker-like colour.
	hinges [][2]float64
}

// themes lists the boards Read recognises, the most common first.
var themes = []*theme{xgDefault}

// xgDefault is eXtreme Gammon 2's default board (« Classic »): gold checkers
// against dark brown on a grey field, the cube in the right-hand tray.
var xgDefault = &theme{
	software:   SoftwareXG,
	barW:       0.95,
	fieldH:     10.82,
	margin:     0.07,
	step:       0.975,
	checker:    0.93,
	triSlots:   3,
	slots:      5,
	firstLight: true,
	palette: []swatch{
		{clsCheckerA, [3]float64{203, 162, 89}},
		{clsCheckerB, [3]float64{82, 58, 55}},
		// The light triangles are shaded: three tones from their tip to
		// their base.
		{clsLightTri, [3]float64{189, 179, 171}},
		{clsLightTri, [3]float64{214, 201, 188}},
		{clsLightTri, [3]float64{168, 158, 150}},
		{clsDarkTri, [3]float64{70, 70, 70}},
		{clsField, [3]float64{128, 127, 126}},
	},
	maxDist:   38,
	bottomIsA: true,
	homeLeft:  true,
	trayW:     1.6,
	hinges:    [][2]float64{{0.85, 1.8}, {9.05, 9.95}},
}

func (t *theme) fieldW() float64 { return 12 + t.barW }

// colX is the centre of the c-th screen column, 0..11 from left to right.
func (t *theme) colX(c int) float64 {
	if c < 6 {
		return float64(c) + 0.5
	}
	return t.barW + float64(c) + 0.5
}

// slotY is the centre of the k-th stacked slot from the base, on the upper
// row (top) or the lower one.
func (t *theme) slotY(k int, top bool) float64 {
	d := t.margin + (float64(k)+0.5)*t.step
	if top {
		return d
	}
	return t.fieldH - d
}

// light says whether the triangle of screen column c on that row is the light
// one: triangles alternate along a row and face a triangle of the other colour.
func (t *theme) light(c int, top bool) bool {
	l := c%2 == 0
	if !top {
		l = !l
	}
	return l == t.firstLight
}

func (t *theme) classify(c [3]float64) class {
	best, bestD := clsOther, t.maxDist*t.maxDist
	for _, s := range t.palette {
		dr, dg, db := c[0]-s.rgb[0], c[1]-s.rgb[1], c[2]-s.rgb[2]
		if d := dr*dr + dg*dg + db*db; d < bestD {
			best, bestD = s.cls, d
		}
	}
	return best
}
