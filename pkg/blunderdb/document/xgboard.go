package document

import (
	"fmt"
	"math"
	"sort"

	"github.com/kevung/blunderdb/pkg/blunderdb/capture"
)

// eXtreme Gammon prints a diagram as text: thirteen lines of glyphs of its
// board font, one glyph per point and rank. The table below was established
// on the prints of XG 2.10 and checked against the printed pip counts of
// 1 518 diagrams; what those diagrams never showed is decoded but flagged
// unproven, so it is never declared sure on the strength of an extrapolation.
//
// Grid: line 0 is the top border, lines 1-5 the top half (1 against the
// border), 6 the middle, 7-11 the bottom half (11 against the border), 12 the
// bottom border. Columns, as offsets of a 24-unit glyph from the line start:
// 0 the cube's margin, 1 the left frame, 2-7 points, 8 the bar's left edge,
// 9 the bar, 10 its right edge, 11-16 points, 17 the right frame, 18 the
// bear-off tray, 19 the right margin. Points are printed 13-24 along the top
// from the left and 12-1 along the bottom, the bottom player's home board at
// the bottom right.

// xgCols are the column offsets of a board drawn at 24 points; the frame
// glyph ð is a quarter glyph wide and the bar's edge @ a twentieth.
var xgCols = [20]float64{0, 24, 30, 54, 78, 102, 126, 150, 174, 175.2, 199.2, 200.4, 224.4, 248.4, 272.4, 296.4, 320.4, 344.4, 350.4, 374.4}

const (
	xgGrid      = 24.0
	xgWidth     = 398.4 // the right margin's far edge
	xgTopBorder = " ñ¡²±²¢ò"
	xgBotBorder = " ó£´³´¤ô"
	white       = 'W'
	black       = 'B'
)

// xgFrameRunes are drawn by every diagram: a font that maps all of them is a
// board font, whatever its name.
var xgFrameRunes = []rune{'ñ', 'ò', 'ó', 'ô', 'ð'}

type pointGlyph struct {
	color byte // 0 for an empty cell
	rank  int
}

var xgTop, xgBottom = func() (map[rune]pointGlyph, map[rune]pointGlyph) {
	top, bot := map[rune]pointGlyph{}, map[rune]pointGlyph{}
	sets := []struct {
		top, bot string
		color    byte
	}{
		{"ABCDE", "abcde", 0}, {"FGHIJ", "fghij", 0}, // empty, plain and hatched point
		{"KLMNO", "klmno", white}, {"PQRST", "pqrst", white},
		{"UVWXY", "uvwxy", black}, {"Z[\\]^", "z{|}~", black},
	}
	for _, s := range sets {
		for i, c := range s.top {
			top[c] = pointGlyph{s.color, i + 1}
		}
		for i, c := range s.bot {
			bot[c] = pointGlyph{s.color, i + 1}
		}
	}
	return top, bot
}()

// stackGlyph decodes the fifth rank of a point, or a checker on the bar: Ð+k
// is k+5 white checkers with the number printed when k > 0, Û+k the same in
// black.
func stackGlyph(c rune) (byte, int, bool) {
	switch {
	case c >= 0xD0 && c <= 0xDA:
		return white, int(c-0xD0) + 5, true
	case c >= 0xDB && c <= 0xE5:
		return black, int(c-0xDB) + 5, true
	}
	return 0, 0, false
}

// offGlyph decodes the tray: æ=5 … ê=1 white, ë=5 … ï=1 black, stacked by
// fives from the border.
func offGlyph(c rune) (byte, int, bool) {
	switch {
	case c >= 0xE6 && c <= 0xEA:
		return white, 5 - int(c-0xE6), true
	case c >= 0xEB && c <= 0xEF:
		return black, 5 - int(c-0xEB), true
	}
	return 0, 0, false
}

// cubeGlyph decodes the cube: ' is the centred cube (printed 64, worth 1),
// " # $ % & are 2 to 32 on one side.
func cubeGlyph(c rune) (int, bool) {
	switch c {
	case '\'':
		return 1, true
	case '"', '#', '$', '%', '&':
		return 1 << (int(c-'"') + 1), true
	}
	return 0, false
}

// Proven limits: the largest of each that the reference diagrams showed.
const (
	provenStack     = 9
	provenBarWhite  = 2
	provenBarBlack  = 3
	provenCubeOwned = 8
)

type bglyph struct {
	x, y, size float64
	text       string
}

// xgGridRead is one diagram's glyphs placed on the grid.
type xgGridRead struct {
	x0, y0, u      float64 // left edge and top baseline, glyph size
	cells          [13][20]rune
	top, bottom    string
	misplaced      int
	conflicts      int
	unexplained    int
	turnMark       rune
	topMark        rune
	bottomMark     rune
	markCandidates int
}

// findGrids places the board font's glyphs of one page on the grid of each
// diagram, a diagram being found at each top-left corner ñ.
func findGrids(gs []bglyph) []*xgGridRead {
	var grids []*xgGridRead
	for _, c := range gs {
		if c.text != "ñ" || c.size <= 0 {
			continue
		}
		u := c.size
		s := u / xgGrid
		g := &xgGridRead{x0: c.x - xgCols[1]*s, y0: c.y, u: u}
		type row struct {
			x float64
			t string
		}
		borders := [2][]row{}
		for _, b := range gs {
			if math.Abs(b.size-u) > 0.02*u {
				continue
			}
			dy := (g.y0 - b.y) / u
			r := math.Round(dy)
			dx := b.x - g.x0
			if math.Abs(dy-r) > 0.05 || r < 0 || r > 12 || dx < -u/2 || dx > (xgWidth+xgGrid/2)*s {
				continue
			}
			ri := int(r)
			if ri == 0 || ri == 12 {
				// Border glyphs are wide and sit off the grid; only the
				// cube's margin is a cell on these lines.
				borders[ri/12] = append(borders[ri/12], row{b.x, b.text})
				if dx > xgCols[1]*s-3*s {
					continue
				}
			}
			best, bestD := -1, math.Inf(1)
			for k := range xgCols {
				if (b.text == "@") != (k == 8 || k == 10) {
					continue
				}
				if d := math.Abs(xgCols[k]*s - dx); d < bestD {
					best, bestD = k, d
				}
			}
			if best < 0 || bestD > 3*s {
				if ri != 0 && ri != 12 { // border glyphs are wide and sit off the grid
					g.misplaced++
				}
				continue
			}
			rs := []rune(b.text)
			if len(rs) != 1 {
				g.misplaced++
				continue
			}
			if prev := g.cells[ri][best]; prev != 0 && prev != rs[0] {
				g.conflicts++
			}
			g.cells[ri][best] = rs[0]
		}
		for i, bs := range borders {
			sort.Slice(bs, func(a, b int) bool { return bs[a].x < bs[b].x })
			var str string
			for _, b := range bs {
				str += b.t
			}
			if i == 0 {
				g.top = str
			} else {
				g.bottom = str
			}
		}
		g.marks(gs)
		grids = append(grids, g)
	}
	sort.SliceStable(grids, func(a, b int) bool {
		ga, gb := grids[a], grids[b]
		if math.Abs(ga.x0-gb.x0) > ga.u {
			return ga.x0 < gb.x0
		}
		return ga.y0 > gb.y0
	})
	return grids
}

func (g *xgGridRead) right() float64 { return g.x0 + xgWidth*g.u/xgGrid }

// marks reads the small checker marks printed with the board font: one to
// the right of each half (the colour of its player) and one under the board
// (the player on roll). Any other board-font glyph inside the diagram is
// counted as unexplained: dice, or a drawing this table does not know.
func (g *xgGridRead) marks(gs []bglyph) {
	u := g.u
	right := g.right()
	bottom := g.y0 - 12*u
	var topY, botY, turnY = math.Inf(-1), math.Inf(1), math.Inf(-1)
	for _, b := range gs {
		big := math.Abs(b.size-u) <= 0.02*u
		isMark := (b.text == "Ð" || b.text == "Û") && b.size < 0.9*u && b.size > 0.4*u
		rs := []rune(b.text)
		switch {
		case isMark && b.x >= right-u && b.x < right+3*u && b.y <= g.y0+u && b.y >= bottom-u:
			if b.y > topY {
				topY, g.topMark = b.y, rs[0]
			}
			if b.y < botY {
				botY, g.bottomMark = b.y, rs[0]
			}
			g.markCandidates++
		case isMark && math.Abs(b.x-g.x0) < u && b.y < bottom && b.y > bottom-6*u:
			if b.y > turnY {
				turnY, g.turnMark = b.y, rs[0]
			}
		case !big && b.text != " " && b.text != "" && b.x >= g.x0 && b.x < right && b.y <= g.y0+u && b.y >= bottom-u/2:
			g.unexplained++
		}
	}
	if g.markCandidates < 2 {
		g.topMark, g.bottomMark = 0, 0
	}
}

// doubt is a capture.Doubt with its weight: a hard doubt makes the entry
// uncertain (the diagram is malformed), a soft one to-verify.
type doubt struct {
	capture.Doubt
	hard bool
}

// xgBoard is what one diagram shows, in the bottom player's terms.
type xgBoard struct {
	// points[p] for the printed point p (1-24): colour and count.
	points [25]struct {
		color byte
		n     int
	}
	bar, off    map[byte]int
	cube        int
	cubeOwner   int // 0 centred, 1 the bottom player, -1 the top player
	bottomColor byte
	turn        int // 1 the bottom player, -1 the top player
	doubts      []doubt
}

func (b *xgBoard) doubt(hard bool, field string, point int, format string, args ...any) {
	b.doubts = append(b.doubts, doubt{capture.Doubt{Field: field, Point: point, Reason: fmt.Sprintf(format, args...)}, hard})
}

// decode reads the diagram's grid with the table.
func (g *xgGridRead) decode() *xgBoard {
	b := &xgBoard{bar: map[byte]int{}, off: map[byte]int{}, cube: 1, bottomColor: white, turn: 1}
	if g.top != xgTopBorder || g.bottom != xgBotBorder {
		b.doubt(true, "orientation", 0, "border glyphs %q / %q are not the 13-24 over 12-1 numbering this reader knows", g.top, g.bottom)
	}
	if g.misplaced > 0 || g.conflicts > 0 {
		b.doubt(true, "frame", 0, "%d glyphs off the grid, %d cells drawn twice", g.misplaced, g.conflicts)
	}
	if g.unexplained > 0 {
		b.doubt(false, "dice", 0, "%d board-font glyphs inside the diagram that the table does not read (dice?)", g.unexplained)
	}
	for r := 1; r <= 11; r++ {
		if g.cells[r][8] != '@' || g.cells[r][10] != '@' || g.cells[r][1] != 'ð' || g.cells[r][17] != 'ð' {
			b.doubt(true, "frame", 0, "line %d lacks the frame or the bar's edges", r)
			break
		}
	}
	for k := range 12 {
		c := 2 + k
		if k >= 6 {
			c = 11 + k - 6
		}
		var top, bot [5]rune
		for r := range 5 {
			top[r] = g.cells[1+r][c]
			bot[r] = g.cells[11-r][c]
		}
		b.column(13+k, top, xgTop)
		b.column(12-k, bot, xgBottom)
	}
	for r := 1; r <= 11; r++ {
		c := g.cells[r][9]
		if c == 0 || c == ' ' {
			continue
		}
		col, n, ok := stackGlyph(c)
		if !ok {
			b.doubt(true, "bar", 0, "glyph %q on the bar", c)
			continue
		}
		if n > 5 {
			b.doubt(false, "bar", 0, "a numbered checker on the bar, never seen in the reference prints")
		}
		b.bar[col] += n - 4
	}
	if b.bar[white] > provenBarWhite || b.bar[black] > provenBarBlack {
		b.doubt(false, "bar", 0, "%d white and %d black on the bar, more than the reference prints showed", b.bar[white], b.bar[black])
	}
	for r := 1; r <= 11; r++ {
		c := g.cells[r][18]
		if c == 0 || c == ' ' {
			continue
		}
		col, n, ok := offGlyph(c)
		if !ok {
			b.doubt(true, "checkers", 0, "glyph %q in the bear-off tray", c)
			continue
		}
		b.off[col] += n
	}
	b.decodeCube(g)
	for r := 1; r <= 11; r++ {
		if c := g.cells[r][19]; c != 0 && c != ' ' {
			b.doubt(true, "frame", 0, "glyph %q right of the tray", c)
		}
	}
	switch {
	case g.topMark == 'Û' && g.bottomMark == 'Ð':
	case g.topMark == 'Ð' && g.bottomMark == 'Û':
		b.bottomColor = black
		b.doubt(false, "orientation", 0, "black at the bottom, never seen in the reference prints")
	default:
		b.doubt(false, "orientation", 0, "the players' colour marks are missing")
	}
	switch {
	case g.turnMark == 0:
		b.doubt(false, "turn", 0, "no mark of the player on roll under the diagram")
	case (g.turnMark == 'Ð') != (b.bottomColor == white):
		b.turn = -1
		b.doubt(false, "turn", 0, "the top player on roll, never seen in the reference prints")
	}
	return b
}

func (b *xgBoard) decodeCube(g *xgGridRead) {
	seen := 0
	for r := range 13 {
		c := g.cells[r][0]
		if c == 0 || c == ' ' {
			continue
		}
		v, ok := cubeGlyph(c)
		seen++
		if !ok || seen > 1 {
			b.doubt(true, "cube", 0, "glyph %q in the cube's margin", c)
			continue
		}
		switch {
		case r == 6 && v == 1:
			b.cube, b.cubeOwner = 1, 0
		case r == 11 && v > 1:
			b.cube, b.cubeOwner = v, 1
			if v > provenCubeOwned {
				b.doubt(false, "cube", 0, "cube at %d, more than the reference prints showed", v)
			}
		case r == 1 && v > 1:
			b.cube, b.cubeOwner = v, -1
			b.doubt(false, "cube", 0, "cube on the top player's side, never seen in the reference prints")
		default:
			b.doubt(true, "cube", 0, "cube glyph %q on line %d", c, r)
		}
	}
	if seen == 0 {
		b.doubt(false, "cube", 0, "no cube drawn")
	}
}

// column reads one point from the border to the fifth rank.
func (b *xgBoard) column(point int, cells [5]rune, table map[rune]pointGlyph) {
	var color byte
	n := 0
	empty := false
	for i, c := range cells {
		rank := i + 1
		var col byte
		var k int
		if sc, sn, ok := stackGlyph(c); ok && rank == 5 {
			col, k = sc, sn
			if sn > provenStack {
				b.doubt(false, "point", point, "a stack of %d, more than the reference prints showed", sn)
			}
		} else if pg, ok := table[c]; ok {
			if pg.rank != rank {
				b.doubt(true, "point", point, "glyph %q of rank %d on rank %d", c, pg.rank, rank)
				return
			}
			col = pg.color
			if col != 0 {
				k = n + 1
				if rank == 5 {
					b.doubt(false, "point", point, "a plain fifth checker, never seen in the reference prints")
				}
			}
		} else {
			b.doubt(true, "point", point, "unknown glyph %q on rank %d", c, rank)
			return
		}
		if col == 0 {
			empty = true
			continue
		}
		if empty || (color != 0 && col != color) {
			b.doubt(true, "point", point, "checkers after an empty cell or of two colours")
			return
		}
		color, n = col, k
	}
	b.points[point].color, b.points[point].n = color, n
}

// counts returns the checkers of a colour on the points, on the bar and in
// the tray.
func (b *xgBoard) counts(col byte) (onBoard, total int) {
	for p := 1; p <= 24; p++ {
		if b.points[p].color == col {
			onBoard += b.points[p].n
		}
	}
	onBoard += b.bar[col]
	return onBoard, onBoard + b.off[col]
}

// pips returns the bottom player's and the top player's pip counts.
func (b *xgBoard) pips() (bottom, top int) {
	tc := otherColor(b.bottomColor)
	for p := 1; p <= 24; p++ {
		switch b.points[p].color {
		case b.bottomColor:
			bottom += b.points[p].n * p
		case tc:
			top += b.points[p].n * (25 - p)
		}
	}
	return bottom + 25*b.bar[b.bottomColor], top + 25*b.bar[tc]
}

func otherColor(c byte) byte {
	if c == white {
		return black
	}
	return white
}

// boardXGID is the XGID's board field: the bottom player in capitals on the
// printed points, its bar at 25, the top player's bar at 0.
func (b *xgBoard) boardXGID() (string, bool) {
	s := []byte("--------------------------")
	tc := otherColor(b.bottomColor)
	for p := 1; p <= 24; p++ {
		pt := b.points[p]
		if pt.n == 0 {
			continue
		}
		if pt.n > 15 {
			return "", false
		}
		if pt.color == b.bottomColor {
			s[p] = byte('A' + pt.n - 1)
		} else {
			s[p] = byte('a' + pt.n - 1)
		}
	}
	if n := b.bar[b.bottomColor]; n > 0 {
		if n > 15 {
			return "", false
		}
		s[25] = byte('A' + n - 1)
	}
	if n := b.bar[tc]; n > 0 {
		if n > 15 {
			return "", false
		}
		s[0] = byte('a' + n - 1)
	}
	return string(s), true
}
