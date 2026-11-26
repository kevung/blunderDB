package document

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/document/pdf/pdftest"
)

// The fixtures are synthetic: a TrueType board font generated here, whose
// glyphs are drawn the way a Snowie board font draws them (a checker is a
// ring of ink, black when filled), at the codes the shared tables read, and
// a page drawing a diagram with it. No glyph of a real font is copied.

type poly [][2]float64

func circle(cx, cy, r float64, clockwise bool) poly {
	var p poly
	for k := range 48 {
		th := 2 * math.Pi * float64(k) / 48
		if clockwise {
			th = -th
		}
		p = append(p, [2]float64{cx + r*math.Cos(th), cy + r*math.Sin(th)})
	}
	return p
}

func rect(x0, y0, x1, y1 float64, clockwise bool) poly {
	if clockwise {
		return poly{{x0, y0}, {x0, y1}, {x1, y1}, {x1, y0}}
	}
	return poly{{x0, y0}, {x1, y0}, {x1, y1}, {x0, y1}}
}

// The synthetic glyphs, in a 1000-unit em.
var (
	glyphPoint = []poly{{{100, 0}, {130, 0}, {520, 1000}, {490, 1000}}, {{870, 0}, {900, 0}, {510, 1000}, {480, 1000}}}
	glyphWhite = append([]poly{circle(500, 500, 450, true), circle(500, 500, 420, false)}, glyphPoint...)
	glyphBlack = append([]poly{circle(500, 500, 450, true)}, glyphPoint...)
	discBlack  = []poly{circle(500, 500, 450, true)}
	discWhite  = []poly{circle(500, 500, 450, true), circle(500, 500, 420, false)}
	numWhite   = []poly{circle(500, 500, 450, true), circle(500, 500, 420, false), rect(430, 380, 570, 620, true)}
	numBlack   = []poly{circle(500, 500, 450, true), rect(450, 400, 550, 600, false)}
	small      = []poly{rect(350, 350, 650, 650, true)}
	frame      = []poly{rect(480, 0, 520, 1000, true)}
	cubeBox    = []poly{rect(80, 80, 920, 920, true), rect(110, 110, 890, 890, false), rect(450, 300, 550, 700, true)}
	trayBlack  = []poly{rect(100, 100, 900, 220, true), rect(100, 300, 900, 420, true), rect(100, 500, 900, 620, true)}
	trayWhite  = []poly{rect(100, 100, 900, 220, true), rect(115, 115, 885, 205, false), rect(100, 300, 900, 420, true), rect(115, 315, 885, 405, false), rect(100, 500, 900, 620, true), rect(115, 515, 885, 605, false)}
)

// snowieFontGlyphs maps the codes the fixture draws to their glyphs.
func snowieFontGlyphs() map[byte][]poly {
	m := map[byte][]poly{
		'A': glyphPoint, 'K': glyphWhite, 'S': glyphBlack, 'f': discBlack, ':': discWhite,
		'<': frame, '>': frame, ')': frame, '/': frame, '.': discWhite, '-': discBlack,
		0xAB: small, 0xC3: trayBlack, 0xCF: trayWhite,
	}
	for c := 0xD3; c <= 0xEA; c++ {
		m[byte(c)] = small
	}
	for c := 0xA1; c <= 0xAA; c++ {
		m[byte(c)] = small
	}
	for c := 0xB8; c <= 0xC2; c++ {
		m[byte(c)] = cubeBox
	}
	for _, c := range "!\"#$%&123456" {
		m[byte(c)] = small
	}
	for c := 0xEB; c <= 0xF4; c++ {
		m[byte(c)] = numBlack
	}
	for c := 0xF5; c <= 0xFE; c++ {
		m[byte(c)] = numWhite
	}
	return m
}

// buildTrueType writes a TrueType program with a (1,0) format 0 cmap.
func buildTrueType(glyphs map[byte][]poly) []byte {
	var codes []int
	for c := range 256 {
		if _, ok := glyphs[byte(c)]; ok {
			codes = append(codes, c)
		}
	}
	cmap := make([]byte, 4+8+6+256)
	binary.BigEndian.PutUint16(cmap[2:], 1)
	binary.BigEndian.PutUint16(cmap[4:], 1)
	binary.BigEndian.PutUint32(cmap[8:], 12)
	binary.BigEndian.PutUint16(cmap[14:], 262)
	var glyf bytes.Buffer
	loca := []uint32{0, 0} // glyph 0 is empty
	for i, c := range codes {
		cmap[18+c] = byte(i + 1)
		var g bytes.Buffer
		cs := glyphs[byte(c)]
		w16 := func(v int) { _ = binary.Write(&g, binary.BigEndian, int16(v)) }
		w16(len(cs))
		w16(0)
		w16(0)
		w16(1000)
		w16(1000)
		end := -1
		for _, p := range cs {
			end += len(p)
			w16(end)
		}
		w16(0)
		for _, p := range cs {
			for range p {
				g.WriteByte(1)
			}
		}
		for axis := range 2 {
			last := 0
			for _, p := range cs {
				for _, q := range p {
					v := int(math.Round(q[axis]))
					w16(v - last)
					last = v
				}
			}
		}
		glyf.Write(g.Bytes())
		for glyf.Len()%4 != 0 {
			glyf.WriteByte(0)
		}
		loca = append(loca, uint32(glyf.Len()))
	}
	var locaB bytes.Buffer
	for _, o := range loca {
		_ = binary.Write(&locaB, binary.BigEndian, o)
	}
	head := make([]byte, 54)
	binary.BigEndian.PutUint16(head[18:], 1000)
	binary.BigEndian.PutUint16(head[50:], 1)
	maxp := make([]byte, 6)
	binary.BigEndian.PutUint32(maxp, 0x5000)
	binary.BigEndian.PutUint16(maxp[4:], uint16(len(codes)+1))
	tables := []struct {
		tag  string
		data []byte
	}{{"cmap", cmap}, {"glyf", glyf.Bytes()}, {"head", head}, {"loca", locaB.Bytes()}, {"maxp", maxp}}
	var out bytes.Buffer
	_ = binary.Write(&out, binary.BigEndian, []uint16{1, 0, uint16(len(tables)), 0, 0, 0})
	off := 12 + 16*len(tables)
	for _, t := range tables {
		out.WriteString(t.tag)
		_ = binary.Write(&out, binary.BigEndian, []uint32{0, uint32(off), uint32(len(t.data))})
		off += (len(t.data) + 3) &^ 3
	}
	for _, t := range tables {
		out.Write(t.data)
		for out.Len()%4 != 0 {
			out.WriteByte(0)
		}
	}
	return out.Bytes()
}

// snowieDiagram describes a fixture diagram from its XGID board, in the
// bottom player's terms.
type snowieDiagram struct {
	board       string // the XGID's 26 characters
	bottomBlack bool
	homeLeft    bool
	// numberedForTop prints the numbers for the top player (1-12 on top).
	numberedForTop bool
	// elder draws a black checker as a disc over its point.
	elder          bool
	cube, cubeRow  int
	dice           [2]int
	diceBlack      bool
	pipDelta       int // added to the printed bottom pips
	noPips         bool
	x0, y0         float64 // the top numbers' baseline, at the left column
	dropProgram    bool
	brokenProgram  bool
	fontName       string
	extraLeftCount bool
	// fontRes is the resource the content draws with, F1 by default.
	fontRes string
}

func (d snowieDiagram) content() string {
	const u = 12.0
	var s strings.Builder
	put := func(x, y float64, c byte) {
		res := d.fontRes
		if res == "" {
			res = "F1"
		}
		fmt.Fprintf(&s, "BT /%s 12 Tf %g %g Td <%02X> Tj ET\n", res, x, y, c)
	}
	colX := func(i int) float64 {
		x := d.x0 + u*float64(i)
		if i >= 6 {
			x += u
		}
		return x
	}
	rowY := func(r int) float64 { return d.y0 - u*float64(r) }
	// pt is the point (bottom terms) of column i's top or bottom half.
	pt := func(i int, top bool) int {
		if d.homeLeft {
			if top {
				return 24 - i
			}
			return 1 + i
		}
		if top {
			return 13 + i
		}
		return 12 - i
	}
	bottomColor, topColor := byte(white), byte(black)
	if d.bottomBlack {
		bottomColor, topColor = black, white
	}
	for i := range 12 {
		tl, bl := pt(i, true), pt(i, false)
		if d.numberedForTop {
			tl, bl = 25-tl, 25-bl
		}
		put(colX(i), d.y0, byte(0xEA-tl+1))
		put(colX(i), rowY(12), byte(0xEA-bl+1))
		for _, top := range []bool{true, false} {
			p := pt(i, top)
			ch := d.board[p]
			n, col := 0, byte(0)
			switch {
			case ch >= 'A' && ch <= 'P':
				n, col = int(ch-'A')+1, bottomColor
			case ch >= 'a' && ch <= 'p':
				n, col = int(ch-'a')+1, topColor
			}
			for rank := 1; rank <= 5; rank++ {
				r := rank
				if !top {
					r = 12 - rank
				}
				x, y := colX(i), rowY(r)
				switch {
				case rank > n:
					put(x, y, 'A')
				case rank == 5 && n > 5:
					if col == black {
						put(x, y, byte(0xEB+n-6))
					} else {
						put(x, y, byte(0xF5+n-6))
					}
				case col == white:
					put(x, y, 'K')
				case d.elder:
					put(x, y, 'A')
					put(x, y, 'f')
				default:
					put(x, y, 'S')
				}
			}
		}
	}
	barX := (colX(5) + colX(6)) / 2
	left, right := colX(0)-u, colX(11)+u
	for r := 1; r <= 11; r++ {
		put(left, rowY(r), '<')
		put(right, rowY(r), '>')
	}
	barN := func(ch byte) int {
		switch {
		case ch >= 'A' && ch <= 'P':
			return int(ch-'A') + 1
		case ch >= 'a' && ch <= 'p':
			return int(ch-'a') + 1
		}
		return 0
	}
	for r := 1; r <= 11; r++ {
		c := byte('/')
		if r <= barN(d.board[0]) {
			c = map[bool]byte{true: '.', false: '-'}[topColor == white]
		}
		if 12-r <= barN(d.board[25]) {
			c = map[bool]byte{true: '.', false: '-'}[bottomColor == white]
		}
		put(barX, rowY(r), c)
	}
	// The tray: the top player's at the top, the bottom player's below.
	onBoard := map[bool]int{}
	for p := 0; p < 26; p++ {
		ch := d.board[p]
		if ch >= 'A' && ch <= 'P' {
			onBoard[false] += barN(ch)
		} else if ch >= 'a' && ch <= 'p' {
			onBoard[true] += barN(ch)
		}
	}
	for _, top := range []bool{true, false} {
		off := 15 - onBoard[top]
		if off == 0 {
			continue
		}
		col := topColor
		r, rc := 1, 2
		if !top {
			col, r, rc = bottomColor, 11, 10
		}
		put(left-u, rowY(r), map[bool]byte{true: 0xC3, false: 0xCF}[col == black])
		if off != 3 || d.extraLeftCount {
			put(left-u, rowY(rc), byte(0xEA-off+1))
		}
	}
	if d.cube > 0 {
		put(right+u, rowY(d.cubeRow), byte(0xB8+int(math.Log2(float64(d.cube)))))
	}
	if !d.noPips {
		pb, pt := 0, 0
		for p := 1; p <= 24; p++ {
			ch := d.board[p]
			if ch >= 'A' && ch <= 'P' {
				pb += barN(ch) * p
			} else if ch >= 'a' && ch <= 'p' {
				pt += barN(ch) * (25 - p)
			}
		}
		pb += 25*barN(d.board[25]) + d.pipDelta
		pt += 25 * barN(d.board[0])
		for _, l := range []struct {
			r, v int
		}{{3, pt}, {9, pb}} {
			put(right+u, rowY(l.r), 0xAB)
			for k, ch := range fmt.Sprint(l.v) {
				dg := int(ch - '0')
				if dg == 0 {
					dg = 10
				}
				put(right+2*u+float64(k)*u/2, rowY(l.r), byte(0xA0+dg))
			}
		}
	}
	if d.dice[0] > 0 {
		for k, v := range d.dice {
			c := byte('!' + v - 1)
			if d.diceBlack {
				c = byte('1' + v - 1)
			}
			put(colX(8)+float64(k)*1.5*u, rowY(6), c)
		}
	}
	return s.String()
}

func (d snowieDiagram) pdf() []byte {
	prog := buildTrueType(snowieFontGlyphs())
	if d.dropProgram {
		prog = nil
	}
	if d.brokenProgram {
		prog = prog[:len(prog)/3]
	}
	name := d.fontName
	if name == "" {
		name = "ABCDEF+BoardGlyphs"
	}
	f := pdftest.Font{BaseFont: name, First: 0x21, Last: 0xFE, Program: prog}
	return pdftest.Doc{
		Title: "Fixture", Fonts: []pdftest.Font{f},
		Pages: []pdftest.Page{{Width: 595, Height: 842, Content: d.content()}},
	}.Bytes()
}

func analyzeOne(t *testing.T, d snowieDiagram) Entry {
	t.Helper()
	r, err := AnalyzePDF(d.pdf(), AnalyzeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Entries) != 1 {
		t.Fatalf("%d entries", len(r.Entries))
	}
	return r.Entries[0]
}

// snowieMidgame has a checker on each bar, a numbered stack of six, and
// three and two checkers borne off.
const snowieMidgame = "a-----E-C----D---c-f----bA"

func TestSnowieReadsDiagram(t *testing.T) {
	for _, tc := range []struct {
		name string
		d    snowieDiagram
		want string
	}{
		{"home left, black at the bottom, dice", snowieDiagram{board: snowieMidgame, bottomBlack: true, homeLeft: true, cube: 2, cubeRow: 11, dice: [2]int{3, 1}, diceBlack: true},
			"XGID=" + snowieMidgame + ":1:1:1:31:0:0:0:0:10"},
		{"home right, white at the bottom, top on roll", snowieDiagram{board: snowieMidgame, homeLeft: false, cube: 4, cubeRow: 1, dice: [2]int{6, 5}, diceBlack: true},
			"XGID=" + snowieMidgame + ":2:-1:-1:65:0:0:0:0:10"},
		{"numbered for the top player, centred cube", snowieDiagram{board: snowieMidgame, bottomBlack: true, homeLeft: true, numberedForTop: true, cube: 1, cubeRow: 6},
			"XGID=" + snowieMidgame + ":0:0:1:00:0:0:0:0:10"},
		{"black drawn over the point, numbered stacks", snowieDiagram{board: "-------H-----g-----------B", bottomBlack: true, homeLeft: true, elder: true, extraLeftCount: true},
			"XGID=-------H-----g-----------B:0:0:1:00:0:0:0:0:10"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.d.x0, tc.d.y0 = 155, 700
			e := analyzeOne(t, tc.d)
			if e.Position != tc.want {
				t.Fatalf("read %s, want %s (%v)", e.Position, tc.want, e.Doubts)
			}
			if e.Checkers != Sure {
				t.Fatalf("checkers %s: %v", e.Checkers, e.Doubts)
			}
			if e.Context == ContextRead {
				t.Fatal("the score is not in the board font: the context cannot be read whole")
			}
		})
	}
}

// A reading the print does not confirm is never sure.
func TestSnowieUnconfirmedIsNotSure(t *testing.T) {
	for _, tc := range []struct {
		name string
		d    snowieDiagram
	}{
		{"pips contradict", snowieDiagram{board: snowieMidgame, bottomBlack: true, homeLeft: true, pipDelta: 3}},
		{"no pips", snowieDiagram{board: snowieMidgame, bottomBlack: true, homeLeft: true, noPips: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.d.x0, tc.d.y0 = 155, 700
			if e := analyzeOne(t, tc.d); e.Checkers == Sure {
				t.Fatalf("sure without the print's confirmation: %s", e.Position)
			}
		})
	}
}

// A board font whose shapes cannot be read is named, not guessed at.
func TestSnowieUnreadableFontIsNamed(t *testing.T) {
	for _, d := range []snowieDiagram{
		{board: snowieMidgame, homeLeft: true, dropProgram: true, fontName: "ABCDEF+SomeBoardFont"},
		{board: snowieMidgame, homeLeft: true, brokenProgram: true, fontName: "ABCDEF+SomeBoardFont"},
	} {
		d.x0, d.y0 = 155, 700
		_, err := AnalyzePDF(d.pdf(), AnalyzeOptions{})
		if !errors.Is(err, ErrUnknownBoardFont) || !strings.Contains(fmt.Sprint(err), "SomeBoardFont") {
			t.Fatalf("got %v", err)
		}
	}
}

func TestShapeOfFixtureGlyphs(t *testing.T) {
	fs := newFontShapes(buildTrueType(snowieFontGlyphs()), &shapeCost{left: shapeBudget})
	for code, want := range map[byte]glyphShape{
		'A': {}, 'K': {disc: whiteDisc}, 'S': {disc: blackDisc}, 'f': {disc: blackDisc},
		0xEB: {disc: blackDisc}, 0xF5: {disc: whiteDisc},
		0xC3: {bars: 3, barsBlack: true}, 0xCF: {bars: 3},
	} {
		got, ok := fs.shape(uint32(code))
		tray := code == 0xC3 || code == 0xCF
		if !ok || got.disc != want.disc || (tray && (got.bars != want.bars || got.barsBlack != want.barsBlack)) {
			t.Errorf("code %#x: %+v, want %+v", code, got, want)
		}
	}
	if sh, _ := fs.shape(0xB8); sh.disc != noDisc {
		t.Errorf("a cube read as a checker: %+v", sh)
	}
}

// A malformed program is refused, never a panic.
func FuzzParseTrueType(f *testing.F) {
	f.Add(buildTrueType(snowieFontGlyphs()))
	f.Fuzz(func(t *testing.T, data []byte) {
		cost := &shapeCost{left: fuzzShapeBudget}
		fs := newFontShapes(data, cost)
		if fs == nil {
			return
		}
		calls := 0
		for c := uint32(0); c < 256; c++ {
			fs.shape(c)
			calls++
		}
		// The work is the budget's: what it allows, and no more.
		if cost.left < 0 || cost.left > fuzzShapeBudget || calls != 256 {
			t.Fatalf("shape budget %d of %d", cost.left, fuzzShapeBudget)
		}
	})
}

// fuzzShapeBudget keeps a fuzzed font to a few milliseconds of shapes.
const fuzzShapeBudget = 50_000_000

func (d snowieDiagram) font() pdftest.Font {
	prog := buildTrueType(snowieFontGlyphs())
	if d.dropProgram {
		prog = nil
	}
	name := d.fontName
	if name == "" {
		name = "ABCDEF+BoardGlyphs"
	}
	return pdftest.Font{BaseFont: name, First: 0x21, Last: 0xFE, Program: prog}
}

// The context is checked before each diagram: a cancelled reading stops
// between two diagrams of one page, not once the document is read.
func TestSnowieCancelledBetweenDiagrams(t *testing.T) {
	a := snowieDiagram{board: snowieMidgame, bottomBlack: true, homeLeft: true, x0: 155, y0: 800}
	b := a
	b.y0 = 500
	data := pdftest.Doc{Title: "Fixture", Fonts: []pdftest.Font{a.font()},
		Pages: []pdftest.Page{{Width: 595, Height: 842, Content: a.content() + b.content()}}}.Bytes()
	if r, err := AnalyzePDF(data, AnalyzeOptions{}); err != nil || len(r.Entries) != 2 {
		t.Fatalf("uncancelled: %v", err)
	}
	// The page's check, the first diagram's, then the second's.
	ctx := &errAfter{Context: context.Background(), n: 3}
	if _, err := AnalyzePDF(data, AnalyzeOptions{Context: ctx}); !errors.Is(err, context.Canceled) {
		t.Fatalf("a cancelled document: %v", err)
	}
	if ctx.calls != ctx.n {
		t.Errorf("%d checks of the context after its cancellation", ctx.calls-ctx.n)
	}
}

// A document whose glyphs exceed the shape budget leaves its diagrams
// uncertain, the budget named, in bounded time.
func TestSnowieShapeBudgetExhausted(t *testing.T) {
	saved := shapeBudget
	shapeBudget = 50 * shapeSamples // a point and its frame, not a checker
	defer func() { shapeBudget = saved }()
	d := snowieDiagram{board: snowieMidgame, bottomBlack: true, homeLeft: true, x0: 155, y0: 700}
	e := analyzeOne(t, d)
	if e.Checkers == Sure || !strings.Contains(fmt.Sprint(e.Doubts), "budget") {
		t.Fatalf("checkers %s, doubts %v", e.Checkers, e.Doubts)
	}
}

// Nested composites of empty glyphs each cost: 32 components four deep
// are refused at the budget, not walked 32⁴ times.
func TestCompositeBombIsBounded(t *testing.T) {
	prog := buildTrueType(map[byte][]poly{'A': small})
	tt, err := parseTrueType(prog)
	if err != nil {
		t.Fatal(err)
	}
	// Glyph 1 becomes 32 references to itself: every level the same.
	var g bytes.Buffer
	_ = binary.Write(&g, binary.BigEndian, []int16{-1, 0, 0, 1000, 1000})
	for k := range 32 {
		flags := uint16(1 | 2)
		if k < 31 {
			flags |= 0x20
		}
		_ = binary.Write(&g, binary.BigEndian, []uint16{flags, 1, 0, 0})
	}
	tt.glyf = g.Bytes()
	tt.loca = make([]byte, 12)
	binary.BigEndian.PutUint32(tt.loca[8:], uint32(len(tt.glyf)))
	binary.BigEndian.PutUint32(tt.loca[4:], 0)
	tt.longLoca = true
	fs := &fontShapes{tt: tt, cost: &shapeCost{left: shapeBudget}, cache: map[uint32]glyphShape{}, bad: map[uint32]bool{}}
	start := time.Now()
	if _, ok := fs.shape('A'); ok {
		t.Fatal("a composite bomb read as a glyph")
	}
	if d := time.Since(start); d > time.Second {
		t.Fatalf("refused in %v", d)
	}
}

// A diagram in a font the reader cannot read is named even when others of
// the document are read.
func TestSnowieUnreadFontWarnedBesideReadDiagrams(t *testing.T) {
	a := snowieDiagram{board: snowieMidgame, bottomBlack: true, homeLeft: true, x0: 155, y0: 700}
	b := a
	b.fontRes, b.fontName, b.dropProgram = "F2", "ABCDEF+OtherBoardFont", true
	data := pdftest.Doc{Title: "Fixture", Fonts: []pdftest.Font{a.font(), b.font()},
		Pages: []pdftest.Page{{Width: 595, Height: 842, Content: a.content()}, {Width: 595, Height: 842, Content: b.content()}}}.Bytes()
	var warned []string
	r, err := AnalyzePDF(data, AnalyzeOptions{Warn: func(s string) { warned = append(warned, s) }})
	if err != nil || len(r.Entries) != 1 {
		t.Fatalf("%v", err)
	}
	if !strings.Contains(strings.Join(warned, "\n"), "OtherBoardFont") {
		t.Fatalf("warnings %q", warned)
	}
}
