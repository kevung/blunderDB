package document

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/document/pdf"
)

// Snowie prints a diagram as text too, in a board font of its own (the
// Atchoum family, "AtchoumSnowie4Normal" and its elders): thirteen lines of
// one-em glyphs, the point numbers along the top and the bottom border, the
// pip counts and the cube beside the board, the tray beyond the frame.
// Versions of the font place their glyphs at different codes and build a
// checker differently — one glyph for a checker on its point, or a disc
// drawn over the point — so the checkers are read from the shapes the
// embedded program draws (ttshape.go), never from a code: a ring of ink is a
// checker, its inside tells its colour. Codes are trusted only for what a
// shape cannot say and what every known version shares: the point numbers
// (Ó 24 … ê 1, also the count beside the tray), the pip digits (¡ 1 … © 9,
// ª 0, after «, "pip:"), the count on a fifth checker (ë … ô black 6-15,
// õ … þ white 6-15), the cube's value and the dice's, each read only where
// it is drawn (the cube beside the board, the dice on its middle line).
//
// A diagram is found by its two rows of point numbers, twelve each at a
// pitch of one glyph with the bar between the halves, one on top of the
// other twelve lines apart: neither the font's name nor the page's layout
// is assumed. The numbers give each column its point and the side of each
// home board; the printed pip counts say which colour the bottom player
// has. A diagram is sure only when those pips confirm the checkers and the
// fifteen of each side are accounted for; the score is not in the board
// font and is never read here.

// snowieLabel is the point a number glyph prints, 0 when it prints none.
func snowieLabel(s string) int {
	r := []rune(s)
	if len(r) != 1 || r[0] < 0xD3 || r[0] > 0xEA {
		return 0
	}
	return 0xEA - int(r[0]) + 1
}

// snowieDigit is a pip digit.
func snowieDigit(s string) (int, bool) {
	r := []rune(s)
	if len(r) != 1 || r[0] < 0xA1 || r[0] > 0xAA {
		return 0, false
	}
	return (int(r[0]) - 0xA0) % 10, true
}

// snowieNumbered is the count a numbered fifth checker prints, 0 for none.
func snowieNumbered(s string) (int, byte) {
	r := []rune(s)
	switch {
	case len(r) != 1:
		return 0, 0
	case r[0] >= 0xEB && r[0] <= 0xF4:
		return int(r[0]) - 0xEB + 6, black
	case r[0] >= 0xF5 && r[0] <= 0xFE:
		return int(r[0]) - 0xF5 + 6, white
	}
	return 0, 0
}

// snowieCube is the value of a cube glyph drawn beside the board: ¸ 1 …
// Â 1024 in the fourth version, " 1 … & 16 in its elder.
func snowieCube(s string) int {
	r := []rune(s)
	switch {
	case len(r) != 1:
	case r[0] >= 0xB8 && r[0] <= 0xC2:
		return 1 << (r[0] - 0xB8)
	case r[0] >= '"' && r[0] <= '&':
		return 1 << (r[0] - '"')
	}
	return 0
}

// snowieDie is a die drawn on the board's middle line: ! … & white and
// 1 … 6 black in the fourth version, Ì … Ñ black in its elder.
func snowieDie(s string) (int, byte) {
	r := []rune(s)
	switch {
	case len(r) != 1:
	case r[0] >= '!' && r[0] <= '&':
		return int(r[0]-'!') + 1, white
	case r[0] >= '1' && r[0] <= '6':
		return int(r[0]-'1') + 1, black
	case r[0] >= 0xCC && r[0] <= 0xD1:
		return int(r[0]-0xCC) + 1, black
	}
	return 0, 0
}

// snowieGrid is one diagram found by its rows of point numbers.
type snowieGrid struct {
	font       *pdf.Font
	u          float64
	topY, botY float64
	// col[i] is the x of the i-th point column from the left; top[i] and
	// bottom[i] the points it prints.
	col         [12]float64
	top, bottom [12]int
	barX        float64
	// row1Y is the baseline of the board's first line, under the top
	// numbers: a document may space the numbers off the board.
	row1Y float64
	// printed are the numbers at the ends of the top and the bottom border.
	printed [4]int
}

func (g *snowieGrid) left() float64  { return g.col[0] - g.u }
func (g *snowieGrid) right() float64 { return g.col[11] + g.u }
func (g *snowieGrid) midY() float64  { return g.row1Y - 5*g.u }

// row is the line a glyph's baseline sits on (1-11 inside the board) and
// whether it sits on one.
func (g *snowieGrid) row(y float64) (int, bool) {
	k := (g.row1Y-y)/g.u + 1
	r := math.Round(k)
	return int(r), math.Abs(k-r) < 0.25
}

// findSnowieGrids finds the diagrams of one page.
func findSnowieGrids(gs []pdf.Glyph) []*snowieGrid {
	type key struct {
		f *pdf.Font
		y float64
	}
	rows := map[key][]pdf.Glyph{}
	for _, g := range gs {
		if g.Upright && g.Font != nil && snowieLabel(g.Text) > 0 {
			k := key{g.Font, math.Round(g.Y*10) / 10}
			rows[k] = append(rows[k], g)
		}
	}
	type labelRow struct {
		f    *pdf.Font
		y, u float64
		x    [12]float64
		v    [12]int
	}
	var tops, bottoms []labelRow
	for k, row := range rows {
		sort.Slice(row, func(i, j int) bool { return row[i].X < row[j].X })
		for i := 0; i+12 <= len(row); i++ {
			r := row[i : i+12]
			u := r[1].X - r[0].X
			if u <= 0 || math.Abs(u-r[0].Size) > 0.25*r[0].Size {
				continue
			}
			ok := true
			for j := 1; j < 12; j++ {
				want := u
				if j == 6 {
					want = 2 * u
				}
				if math.Abs(r[j].X-r[j-1].X-want) > 0.15*u {
					ok = false
					break
				}
			}
			if !ok {
				continue
			}
			var lr labelRow
			lr.f, lr.y, lr.u = k.f, k.y, u
			seen := map[int]bool{}
			lo, hi := 25, 0
			for j := range 12 {
				lr.x[j], lr.v[j] = r[j].X, snowieLabel(r[j].Text)
				seen[lr.v[j]] = true
				lo, hi = min(lo, lr.v[j]), max(hi, lr.v[j])
			}
			if len(seen) != 12 || hi-lo != 11 || (lo != 1 && lo != 13) {
				continue
			}
			// The numbers run along the border, one way or the other.
			step := lr.v[1] - lr.v[0]
			for j := 1; j < 12 && ok; j++ {
				ok = lr.v[j]-lr.v[j-1] == step
			}
			if !ok || (step != 1 && step != -1) {
				continue
			}
			tops = append(tops, lr)
			bottoms = append(bottoms, lr)
			i += 11
		}
	}
	var out []*snowieGrid
	for _, t := range tops {
		for _, b := range bottoms {
			if b.f != t.f || (t.y-b.y)/t.u < 11.7 || (t.y-b.y)/t.u > 13.5 || math.Abs(b.u-t.u) > 0.05*t.u {
				continue
			}
			ok := true
			for j := range 12 {
				if math.Abs(t.x[j]-b.x[j]) > 0.4*t.u || t.v[j]+b.v[j] != 25 {
					ok = false
					break
				}
			}
			if !ok {
				continue
			}
			g := &snowieGrid{font: t.f, u: t.u, topY: t.y, botY: b.y, col: t.x, top: t.v, bottom: b.v, printed: [4]int{t.v[0], t.v[11], b.v[0], b.v[11]}}
			if t.v[0] <= 12 {
				// Numbered for the player at the top: the board is read
				// in the bottom player's terms all the same.
				for j := range 12 {
					g.top[j], g.bottom[j] = 25-t.v[j], 25-b.v[j]
				}
			}
			g.barX = (t.x[5] + t.x[6]) / 2
			if g.drawn(gs) {
				out = append(out, g)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if math.Abs(out[i].topY-out[j].topY) > out[i].u {
			return out[i].topY > out[j].topY
		}
		return out[i].col[0] < out[j].col[0]
	})
	return out
}

// drawn reports whether the board between the two rows of numbers is drawn
// in their font: a glyph in nine cells of the points out of ten. Two
// diagrams one above the other also put a bottom row twelve lines above a
// top row, with nothing of a board between them.
func (g *snowieGrid) drawn(gs []pdf.Glyph) bool {
	g.row1Y = math.Inf(-1)
	for _, gl := range gs {
		if gl.Font == g.font && gl.Y < g.topY-0.5*g.u && gl.Y > g.botY+0.5*g.u && math.Abs(gl.X-g.col[0]) < 0.3*g.u {
			g.row1Y = math.Max(g.row1Y, gl.Y)
		}
	}
	if math.IsInf(g.row1Y, -1) || g.row1Y-10*g.u < g.botY+0.5*g.u {
		return false
	}
	filled := map[[2]int]bool{}
	for _, gl := range gs {
		if gl.Font != g.font {
			continue
		}
		r, on := g.row(gl.Y)
		if !on || r < 1 || r > 11 || r == 6 {
			continue
		}
		for c := range 12 {
			if math.Abs(gl.X-g.col[c]) < 0.3*g.u {
				filled[[2]int{c, r}] = true
			}
		}
	}
	return len(filled) >= 108
}

// snowieRead is what the shape reader keeps for one document: each font's
// shapes, the budget they share, and the caller's context.
type snowieRead struct {
	ctx    context.Context
	shapes map[*pdf.Font]*fontShapes
	cost   shapeCost
}

func newSnowieRead(ctx context.Context) *snowieRead {
	return &snowieRead{ctx: ctx, shapes: map[*pdf.Font]*fontShapes{}, cost: shapeCost{left: shapeBudget}}
}

func (s *snowieRead) of(f *pdf.Font) *fontShapes {
	if fs, ok := s.shapes[f]; ok {
		return fs
	}
	var fs *fontShapes
	if p := f.Program(); p != nil {
		fs = newFontShapes(p, &s.cost)
	}
	s.shapes[f] = fs
	return fs
}

// snowieCell is one cell's reading.
type snowieCell struct {
	color    byte // 0 no checker
	numbered int
	unclear  bool
	// plain are the colours of the unnumbered discs drawn in the cell.
	plain     []byte
	numColor  byte
	numbers   int
	anyUnread bool
}

// resolve settles a cell: a numbered disc carries the count and decides
// the colour, whatever plain disc a version of the font draws with it; two
// plain discs of different colours are a doubt.
func (cl *snowieCell) resolve() {
	switch {
	case cl.anyUnread || cl.numbers > 1:
		cl.unclear = true
	case cl.numbers == 1:
		cl.color = cl.numColor
	default:
		for _, c := range cl.plain {
			if cl.color != 0 && c != cl.color {
				cl.unclear = true
			}
			cl.color = c
		}
	}
}

// decode reads one diagram. It returns nil when its font's shapes cannot be
// read: the font is then named as not recognised.
func (g *snowieGrid) decode(gs []pdf.Glyph, sr *snowieRead) *snowieBoard {
	fs := sr.of(g.font)
	if fs == nil {
		return nil
	}
	b := &xgBoard{bar: map[byte]int{}, off: map[byte]int{}, cube: 1, bottomColor: white, turn: 1}
	u := g.u
	var mine []pdf.Glyph
	for _, gl := range gs {
		if gl.Font == g.font && gl.Upright && gl.X > g.left()-4*u && gl.X < g.right()+6*u && gl.Y < g.topY-0.5*u && gl.Y > g.botY+0.5*u {
			mine = append(mine, gl)
		}
	}
	cells := map[[2]int]*snowieCell{} // [column 0-11 or 12 for the bar, row 1-11]
	unreadable := 0
	cellOf := func(gl pdf.Glyph, c, r int) {
		sh, ok := fs.shape(gl.Code)
		if !ok {
			unreadable++
			return
		}
		if sh.disc == noDisc {
			return
		}
		cl := cells[[2]int{c, r}]
		if cl == nil {
			cl = &snowieCell{}
			cells[[2]int{c, r}] = cl
		}
		var col byte
		switch sh.disc {
		case whiteDisc:
			col = white
		case blackDisc:
			col = black
		default:
			cl.anyUnread = true
			return
		}
		if n, nc := snowieNumbered(gl.Text); n > 0 {
			if nc != col {
				cl.anyUnread = true
			}
			cl.numbered, cl.numColor = n, col
			cl.numbers++
			return
		}
		cl.plain = append(cl.plain, col)
	}
	var outside []snowieSide
	var dice []int
	diceColor := byte(0)
	for _, gl := range mine {
		r, onRow := g.row(gl.Y)
		inBoard := gl.X > g.left()+0.5*u && gl.X < g.right()-0.5*u
		if !inBoard {
			if gl.X <= g.left()-0.5*u || gl.X >= g.right()+0.5*u {
				outside = append(outside, snowieSide{gl, gl.Y > g.midY()})
			}
			continue
		}
		if !onRow || r < 1 || r > 11 {
			continue
		}
		if r == 6 {
			if v, c := snowieDie(gl.Text); v > 0 {
				if diceColor != 0 && diceColor != c {
					b.doubt(true, "dice", 0, "dice of both colours on the middle line")
				}
				diceColor = c
				dice = append(dice, v)
			} else if sh, ok := fs.shape(gl.Code); ok && sh.disc != noDisc {
				b.doubt(true, "frame", 0, "a checker drawn on the middle line")
			}
			continue
		}
		if math.Abs(gl.X-g.barX) < 0.3*u {
			cellOf(gl, 12, r)
			continue
		}
		for c := range 12 {
			if math.Abs(gl.X-g.col[c]) < 0.3*u {
				cellOf(gl, c, r)
				break
			}
		}
	}
	for _, cl := range cells {
		cl.resolve()
	}
	if unreadable > 0 && fs.cost.exhausted {
		b.doubt(true, "frame", 0, "%d glyphs of the board font %q left unread: the document's glyphs exceed the shape reader's budget", unreadable, g.font.BaseFont)
	} else if unreadable > 0 {
		b.doubt(true, "frame", 0, "%d glyphs of the board font %q whose shapes the embedded program does not give", unreadable, g.font.BaseFont)
	}
	g.placeCheckers(b, cells)
	pipTop, pipBot := g.readSide(b, fs, mine, outside)
	return g.finish(b, pipTop, pipBot, dice, diceColor)
}

// snowieSide is a board-font glyph beside the board, in its top or bottom
// half.
type snowieSide struct {
	gl  pdf.Glyph
	top bool
}

// placeCheckers puts the cells' checkers on the points and the bar.
func (g *snowieGrid) placeCheckers(b *xgBoard, cells map[[2]int]*snowieCell) {
	// The points, from the border towards the middle.
	for c := range 12 {
		for _, half := range []struct {
			point int
			rows  [5]int
		}{{g.top[c], [5]int{1, 2, 3, 4, 5}}, {g.bottom[c], [5]int{11, 10, 9, 8, 7}}} {
			n, col := 0, byte(0)
			gap := false
			for k, r := range half.rows {
				cl := cells[[2]int{c, r}]
				if cl == nil {
					gap = true
					continue
				}
				if cl.unclear {
					b.doubt(true, "checkers", half.point, "a checker whose colour the glyphs do not tell")
				}
				if gap {
					b.doubt(true, "checkers", half.point, "a checker drawn away from the others")
				}
				if col != 0 && cl.color != col {
					b.doubt(true, "checkers", half.point, "checkers of both colours")
				}
				col = cl.color
				n++
				if cl.numbered > 0 {
					if k != 4 {
						b.doubt(true, "checkers", half.point, "a numbered checker below the fifth")
					}
					n = cl.numbered
				}
			}
			if n > 0 {
				b.points[half.point].color, b.points[half.point].n = col, n
			}
		}
	}
	for r := 1; r <= 11; r++ {
		if cl := cells[[2]int{12, r}]; cl != nil {
			if cl.unclear {
				b.doubt(true, "checkers", 0, "a checker on the bar whose colour the glyphs do not tell")
			}
			n := 1
			if cl.numbered > 0 {
				n = cl.numbered
			}
			b.bar[cl.color] += n
		}
	}
}

// readSide reads beside the board the pip counts (-1 when not printed),
// the cube and the tray.
func (g *snowieGrid) readSide(b *xgBoard, fs *fontShapes, mine []pdf.Glyph, outside []snowieSide) (pipTop, pipBot int) {
	u := g.u
	// Beside the board: the pip counts, the cube, the tray.
	pipTop, pipBot = -1, -1
	type cubeAt struct {
		v   int
		row int
	}
	var cubes []cubeAt
	trayBars := map[bool][]snowieSide{}
	trayCount := map[bool][]snowieSide{}
	for _, s := range outside {
		gl := s.gl
		r, _ := g.row(gl.Y)
		switch {
		case gl.Text == "«":
			v, ok := g.pipAfter(mine, gl)
			if !ok {
				continue
			}
			if s.top {
				pipTop = v
			} else {
				pipBot = v
			}
		case snowieCube(gl.Text) > 0 && gl.X > g.right():
			cubes = append(cubes, cubeAt{snowieCube(gl.Text), r})
		case snowieLabel(gl.Text) > 0:
			trayCount[s.top] = append(trayCount[s.top], s)
		default:
			if _, isDigit := snowieDigit(gl.Text); isDigit {
				continue
			}
			if sh, ok := fs.shape(gl.Code); ok && sh.bars > 0 && !sh.barsMixed && sh.disc == noDisc {
				trayBars[s.top] = append(trayBars[s.top], s)
			}
		}
	}
	for _, top := range []bool{true, false} {
		bars := trayBars[top]
		if len(bars) == 0 {
			if len(trayCount[top]) > 0 {
				b.doubt(true, "checkers", 0, "a count beside the tray with no checker in it")
			}
			continue
		}
		if len(bars) > 1 {
			b.doubt(true, "checkers", 0, "%d stacks in one half of the tray", len(bars))
		}
		sh, _ := fs.shape(bars[0].gl.Code)
		col := byte(white)
		if sh.barsBlack {
			col = black
		}
		n := sh.bars
		for _, c := range trayCount[top] {
			if math.Abs(c.gl.X-bars[0].gl.X) < 0.3*u {
				n = snowieLabel(c.gl.Text)
			}
		}
		if b.off[col] > 0 {
			b.doubt(true, "checkers", 0, "%s checkers in both halves of the tray", colorName(col))
		}
		b.off[col] += n
	}
	switch len(cubes) {
	case 0:
	case 1:
		c := cubes[0]
		switch {
		case c.row <= 3:
			b.cube, b.cubeOwner = c.v, -1
		case c.row >= 9:
			b.cube, b.cubeOwner = c.v, 1
		case c.v == 1 || c.v == 64:
			b.cube, b.cubeOwner = 1, 0
		default:
			b.doubt(true, "cube", 0, "a centred cube printed %d", c.v)
		}
		if b.cubeOwner != 0 && c.v < 2 {
			b.doubt(true, "cube", 0, "an owned cube printed %d", c.v)
		}
	default:
		b.doubt(true, "cube", 0, "%d cubes drawn beside the diagram", len(cubes))
	}
	return pipTop, pipBot
}

// finish settles which colour is the bottom player's by the printed pips,
// the player on roll by the dice, and names what the diagram does not say.
func (g *snowieGrid) finish(b *xgBoard, pipTop, pipBot int, dice []int, diceColor byte) *snowieBoard {
	// The pips say which colour moves towards the bottom numbers.
	pipsChecked := pipTop >= 0 && pipBot >= 0
	if pipsChecked {
		var fits []byte
		for _, c := range []byte{white, black} {
			b.bottomColor = c
			if pb, pt := b.pips(); pb == pipBot && pt == pipTop {
				fits = append(fits, c)
			}
		}
		switch len(fits) {
		case 1:
			b.bottomColor = fits[0]
		case 2:
			b.bottomColor = white
			b.doubt(false, "pips", 0, "the printed pips %d/%d fit both colours as the bottom player", pipBot, pipTop)
		default:
			b.bottomColor = white
			pw, tw := b.pips()
			b.bottomColor = black
			pk, tk := b.pips()
			b.bottomColor = white
			b.doubt(false, "pips", 0, "pips read %d/%d (white at the bottom) or %d/%d (black), printed %d/%d (bottom/top)", pw, tw, pk, tk, pipBot, pipTop)
		}
	} else {
		b.doubt(false, "pips", 0, "the pip counts are not printed beside the diagram")
	}
	switch {
	case len(dice) == 0:
		b.doubt(false, "turn", 0, "no dice drawn: the player on roll is not read")
	case len(dice) != 2:
		b.doubt(true, "dice", 0, "%d dice drawn", len(dice))
	default:
		if diceColor != b.bottomColor {
			b.turn = -1
		}
	}
	for _, c := range []byte{b.bottomColor, otherColor(b.bottomColor)} {
		if _, total := b.counts(c); total != 15 {
			b.doubt(false, "checkers", 0, "%s has %d checkers on the board, the bar and the tray, not 15", colorName(c), total)
		}
	}
	b.doubt(false, "score", 0, "the board font prints no score: it is not read")
	d := "00"
	if len(dice) == 2 && !hasHard(b) {
		d = fmt.Sprintf("%d%d", dice[0], dice[1])
	}
	home := "left"
	if g.bottom[0] != 1 {
		home = "right"
	}
	bottom := "white"
	if b.bottomColor == black {
		bottom = "black"
	}
	prof := &Profile{
		Home: home, Bottom: bottom,
		Pips:      "printed beside the diagram in its board font",
		Cube:      "beside the diagram, at the side of its owner",
		Numbering: fmt.Sprintf("%d-%d along the top, %d-%d along the bottom", g.printed[0], g.printed[1], g.printed[2], g.printed[3]),
	}
	return &snowieBoard{b, d, pipsChecked, prof}
}

// snowieBoard is a decoded diagram with what its entry needs beyond the
// board: the dice ("00" for none) and whether the printed pips were found.
type snowieBoard struct {
	*xgBoard
	dice        string
	pipsChecked bool
	profile     *Profile
}

func hasHard(b *xgBoard) bool {
	for _, d := range b.doubts {
		if d.hard {
			return true
		}
	}
	return false
}

// pipAfter reads the number printed after a "pip:" glyph on its line.
func (g *snowieGrid) pipAfter(gs []pdf.Glyph, at pdf.Glyph) (int, bool) {
	var line []pdf.Glyph
	for _, gl := range gs {
		if math.Abs(gl.Y-at.Y) < 0.1*g.u && gl.X > at.X && gl.X < at.X+4*g.u {
			line = append(line, gl)
		}
	}
	sort.Slice(line, func(i, j int) bool { return line[i].X < line[j].X })
	v, n := 0, 0
	for _, gl := range line {
		d, ok := snowieDigit(gl.Text)
		if !ok {
			if n > 0 || strings.TrimSpace(gl.Text) != "" {
				break
			}
			continue
		}
		v = v*10 + d
		n++
	}
	return v, n > 0 && n <= 3
}

// readSnowiePage returns the entries of the page's Snowie diagrams, the
// profile of the first, and the names of the board fonts whose diagrams were
// found but whose shapes could not be read. The context is checked before
// each diagram, each costing a bounded part of the shape budget.
func readSnowiePage(gs []pdf.Glyph, page pdf.Page, sr *snowieRead) ([]Entry, *Profile, []string, error) {
	grids := findSnowieGrids(gs)
	if len(grids) == 0 {
		return nil, nil, nil, nil
	}
	lines := textLines(gs)
	var entries []Entry
	var profile *Profile
	var unread []string
	for _, g := range grids {
		if err := sr.ctx.Err(); err != nil {
			return nil, nil, nil, err
		}
		sb := g.decode(gs, sr)
		if sb == nil {
			unread = append(unread, g.font.BaseFont)
			continue
		}
		e := entryFromBoard(sb.xgBoard, sb.dice, 0, 0, 0, 0, sb.pipsChecked)
		e.Label = snowieLabelAbove(lines, g)
		e.Rect = snowieRect(g, page.Box)
		entries = append(entries, e)
		if profile == nil {
			profile = sb.profile
		}
	}
	return entries, profile, unread, nil
}

func snowieLabelAbove(lines []textLine, g *snowieGrid) string {
	best := ""
	bestY := math.Inf(1)
	for _, l := range lines {
		if l.y > g.topY+g.u/2 && l.x >= g.left()-2*g.u && l.x < g.right()+4*g.u && l.y < bestY && labelRE.MatchString(l.text) {
			best, bestY = l.text, l.y
		}
	}
	return best
}

func snowieRect(g *snowieGrid, box [4]float64) *Rect {
	w, h := box[2]-box[0], box[3]-box[1]
	if w <= 0 || h <= 0 {
		return nil
	}
	clamp := func(v float64) float64 { return math.Max(0, math.Min(1, v)) }
	x0 := clamp((g.left() - g.u - box[0]) / w)
	x1 := clamp((g.right() + 4*g.u - box[0]) / w)
	y0 := clamp((box[3] - (g.topY + g.u)) / h)
	y1 := clamp((box[3] - (g.botY - 0.25*g.u)) / h)
	if x1 <= x0 || y1 <= y0 {
		return nil
	}
	return &Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}
