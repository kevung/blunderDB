package document

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/document/pdf"
	"github.com/kevung/blunderdb/pkg/blunderdb/document/pdf/pdftest"
)

var update = flag.Bool("update", false, "rewrite testdata/xg-diagrammes.pdf")

// diagram is a made-up diagram for the test PDF: a position, drawn the way
// eXtreme Gammon prints it, with what it prints beside it.
type diagram struct {
	xgid  string // the expected reading
	label string
	// pips overrides the printed pip counts (bottom, top) to break the
	// cross-check.
	pips *[2]int
	// patch replaces one cell of the drawn grid: line, column, glyph.
	patch *struct {
		r, c int
		g    rune
	}
}

// xgRows draws a position as the thirteen lines of board-font glyphs, the
// inverse of the table in xgboard.go. The bottom player is white.
func xgRows(t testing.TB, xgid string) [13][]rune {
	t.Helper()
	if _, err := ParsePosition(xgid); err != nil && xgid != tooFull {
		t.Fatal(err)
	}
	f := strings.Split(strings.TrimPrefix(xgid, "XGID="), ":")
	board := f[0]
	var rows [13][]rune
	rows[0], rows[12] = []rune(xgTopBorder), []rune(xgBotBorder)
	for r := 1; r <= 11; r++ {
		rows[r] = []rune(strings.Repeat(" ", 20))
		rows[r][1], rows[r][8], rows[r][10], rows[r][17] = 'ð', '@', '@', 'ð'
	}
	count := func(c byte) (byte, int) {
		switch {
		case c >= 'A' && c <= 'O':
			return white, int(c-'A') + 1
		case c >= 'a' && c <= 'o':
			return black, int(c-'a') + 1
		}
		return 0, 0
	}
	for k := range 12 {
		c := 2 + k
		if k >= 6 {
			c = 11 + k - 6
		}
		for half, point := range []int{13 + k, 12 - k} {
			col, n := count(board[point])
			for rank := 1; rank <= 5; rank++ {
				var g rune
				switch {
				case rank == 5 && n >= 5 && col == white:
					g = rune(0xD0 + n - 5)
				case rank == 5 && n >= 5:
					g = rune(0xDB + n - 5)
				case rank <= n && col == white:
					g = 'K' + rune(rank-1)
				case rank <= n:
					g = 'U' + rune(rank-1)
				default:
					g = 'A' + rune(rank-1)
				}
				row := rank
				if half == 1 {
					row = 12 - rank
					if g >= 'A' && g <= 'Z' {
						g = g - 'A' + 'a'
					}
				}
				rows[row][c] = g
			}
		}
	}
	_, wb := count(board[25])
	for i := range wb {
		rows[5-i][9] = 0xD0
	}
	_, bb := count(board[0])
	for i := range bb {
		rows[7+i][9] = 0xDB
	}
	tray := func(n int, firstRow, step int, base rune) {
		for row := firstRow; n > 0; row += step {
			v := min(n, 5)
			rows[row][18] = base + rune(5-v)
			n -= v
		}
	}
	on := map[byte]int{}
	for i := range 26 {
		c, n := count(board[i])
		on[c] += n
	}
	tray(15-on[white], 11, -1, 0xE6)
	tray(15-on[black], 1, 1, 0xEB)
	lg, owner := atoi(f[1]), atoi(f[2])
	switch {
	case owner == 0:
		rows[6][0] = '\''
	case owner == 1:
		rows[11][0] = '"' + rune(lg-1)
	default:
		rows[1][0] = '"' + rune(lg-1)
	}
	return rows
}

func drawDiagram(t testing.TB, d diagram, x0, y0 float64) string {
	rows := xgRows(t, d.xgid)
	if p := d.patch; p != nil {
		rows[p.r][p.c] = p.g
	}
	f := strings.Split(strings.TrimPrefix(d.xgid, "XGID="), ":")
	board := f[0]
	var b xgBoard
	b.bottomColor = white
	b.bar = map[byte]int{}
	if c := board[25]; c != '-' {
		b.bar[white] = int(c-'A') + 1
	}
	if c := board[0]; c != '-' {
		b.bar[black] = int(c-'a') + 1
	}
	for p := 1; p <= 24; p++ {
		c := board[p]
		switch {
		case c >= 'A' && c <= 'O':
			b.points[p].color, b.points[p].n = white, int(c-'A')+1
		case c >= 'a' && c <= 'o':
			b.points[p].color, b.points[p].n = black, int(c-'a')+1
		}
	}
	pb, pt := b.pips()
	if d.pips != nil {
		pb, pt = d.pips[0], d.pips[1]
	}
	var s strings.Builder
	fmt.Fprintf(&s, "BT /F1 24 Tf %g %g Td", x0, y0)
	for i, r := range rows {
		if i > 0 {
			s.WriteString(" 0 -24 Td")
		}
		fmt.Fprintf(&s, " %s Tj", pdftest.Hex(string(r)))
	}
	s.WriteString(" ET\n")
	right := x0 + xgWidth
	fmt.Fprintf(&s, "BT /F1 16 Tf %g %g Td %s Tj ET\n", right, y0-90, pdftest.Hex("Û"))
	fmt.Fprintf(&s, "BT /F1 16 Tf %g %g Td %s Tj ET\n", right, y0-230, pdftest.Hex("Ð"))
	fmt.Fprintf(&s, "BT /F1 16 Tf %g %g Td %s Tj ET\n", x0, y0-12*24-30, pdftest.Hex("Ð"))
	text := []struct {
		dy float64
		s  string
	}{
		{90, "Player 2"}, {100, "score: " + f[6]}, {110, "pip: " + fmt.Sprint(pt)},
		{150, f[8] + " point match"},
		{200, "pip: " + fmt.Sprint(pb)}, {210, "score: " + f[5]}, {230, "Player 1"},
	}
	for _, l := range text {
		fmt.Fprintf(&s, "BT /F2 7.6 Tf %g %g Td %s Tj ET\n", right+20, y0-l.dy, pdftest.Literal(l.s))
	}
	fmt.Fprintf(&s, "BT /F2 12 Tf %g %g Td (Best Cube action: Double / Take) Tj ET\n", x0+20, y0-12*24-30)
	if d.label != "" {
		fmt.Fprintf(&s, "BT /F2 14 Tf %g %g Td %s Tj ET\n", x0, y0+40, pdftest.Literal(d.label))
	}
	return s.String()
}

const (
	start    = "XGID=-b----E-C---eE---c-e----B-:0:0:1:00:0:0:0:5:10"
	stack7   = "XGID=a-----G-C---dB---c-e----bA:2:1:1:00:2:3:0:7:10"
	race     = "XGID=-BCB-DD------------cb-a---:0:0:1:00:0:0:0:3:10"
	topCube  = "XGID=-b----E-C---eE---c-e----B-:1:-1:1:00:1:0:0:5:10"
	tooFull  = "XGID=-b----I-I---eE---c-e----B-:0:0:1:00:0:0:0:5:10"
	unknownG = '?'
)

// fixtureDiagrams are the test PDF's diagrams, page by page; a nil page has
// text only.
var fixtureDiagrams = [][]diagram{
	nil,
	{{xgid: start, label: "Problem 1"}},
	{{xgid: stack7}, {xgid: race}},
	{{xgid: start, pips: &[2]int{160, 167}}},
	{{xgid: start, patch: &struct {
		r, c int
		g    rune
	}{11, 16, unknownG}}},
	{{xgid: topCube}},
}

func fixturePDF(t testing.TB) []byte {
	board := pdftest.Font{BaseFont: "QWERTY+BoardFont", Type0: true, First: 0x20, Last: 0xFF, Widths: map[rune]int{
		'ð': 250, 'ñ': 250, 'ò': 250, 'ó': 250, 'ô': 250, '@': 50, '²': 50, '´': 50,
		'¡': 6000, '¢': 6000, '£': 6000, '¤': 6000,
	}}
	doc := pdftest.Doc{
		Title: "Invented diagrams", Author: "The test suite",
		Fonts:   []pdftest.Font{board, {BaseFont: "Helvetica"}},
		Outline: []pdftest.Bookmark{{Title: "One", Page: 2}, {Title: "Two", Page: 4}},
	}
	for _, page := range fixtureDiagrams {
		p := pdftest.Page{Width: 612, Height: 792, Content: "BT /F2 20 Tf 72 700 Td (A page of prose) Tj ET\n"}
		for i, d := range page {
			p.Content += drawDiagram(t, d, 56.7+float64(i)*520, 680)
		}
		if len(page) > 1 {
			p.Width = 1100
		}
		doc.Pages = append(doc.Pages, p)
	}
	return doc.Bytes()
}

const fixturePath = "testdata/xg-diagrammes.pdf"

func TestFixtureIsCurrent(t *testing.T) {
	want := fixturePDF(t)
	if *update {
		if err := os.WriteFile(fixturePath, want, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s is stale: go test ./pkg/blunderdb/document -run TestFixtureIsCurrent -update", fixturePath)
	}
}

func TestAnalyzeFixture(t *testing.T) {
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	r, err := AnalyzePDF(data, AnalyzeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if r.Document.Title != "Invented diagrams" || r.Document.Author != "The test suite" || len(r.Document.SHA256) != 64 {
		t.Errorf("document %+v", r.Document)
	}
	if p := r.Document.Profile; p == nil || p.Home != "right" || p.Bottom != "white" {
		t.Errorf("profile %+v", p)
	}
	type want struct {
		page        int
		chapter     string
		label, xgid string
		state       State
		doubtFields string
	}
	wants := []want{
		{2, "One", "Problem 1", start, Sure, ""},
		{3, "One", "", stack7, Sure, ""},
		{3, "One", "", race, Sure, ""},
		{4, "Two", "", start, ToVerify, "pips"},
		{5, "Two", "", "XGID=------E-C---eE---c-e----B-:0:0:1:00:0:0:0:5:10", Uncertain, "point checkers pips"},
		{6, "Two", "", topCube, ToVerify, "cube"},
	}
	if len(r.Entries) != len(wants) {
		t.Fatalf("%d entries, want %d: %+v", len(r.Entries), len(wants), r.Entries)
	}
	for i, w := range wants {
		e := r.Entries[i]
		var fields []string
		for _, d := range e.Doubts {
			fields = append(fields, d.Field)
		}
		if e.Rank != i+1 || e.Page != w.page || e.Chapter != w.chapter || e.Label != w.label || e.Position != w.xgid ||
			e.State != w.state || strings.Join(fields, " ") != w.doubtFields || e.Rect == nil || e.Verified {
			t.Errorf("rank %d = %+v (doubts %v), want %+v", i+1, e, e.Doubts, w)
		}
	}
	if r2, _ := AnalyzePDF(data, AnalyzeOptions{Title: "Mine", Author: "Me"}); r2.Document.Title != "Mine" || r2.Document.Author != "Me" {
		t.Errorf("options do not replace the metadata: %+v", r2.Document)
	}
	if _, err := r.Encode(); err != nil {
		t.Errorf("the relevé does not encode: %v", err)
	}
}

func TestAnalyzeRefusesAPDFWithoutBoardFont(t *testing.T) {
	data := pdftest.Doc{Title: "Prose", Fonts: []pdftest.Font{{BaseFont: "Helvetica"}},
		Pages: []pdftest.Page{{Width: 600, Height: 800, Content: "BT /F1 12 Tf 72 700 Td (score: 3 pip: 167) Tj ET"}}}.Bytes()
	if _, err := AnalyzePDF(data, AnalyzeOptions{}); !errors.Is(err, ErrNoGlyphDiagram) {
		t.Errorf("err = %v, want ErrNoGlyphDiagram", err)
	}
	if _, err := AnalyzePDF([]byte("not a pdf"), AnalyzeOptions{}); !errors.Is(err, pdf.ErrNotPDF) {
		t.Errorf("err = %v, want pdf.ErrNotPDF", err)
	}
}

// TestUnreadableDiagramKeepsItsPlace: a diagram that is no legal position is
// an uncertain entry without position, which the import refuses.
func TestUnreadableDiagramKeepsItsPlace(t *testing.T) {
	g := gridOf(xgRows(t, tooFull))
	e := entryOf(g.decode(), panel{})
	if e.State != Uncertain || e.Position != "" {
		t.Fatalf("entry %+v", e)
	}
	r := &Releve{Format: Format, Version: Version, Document: Source{Title: "T"}, Entries: []Entry{{Rank: 1, Page: 1, State: Uncertain}}}
	if err := r.Validate(); err != nil {
		t.Fatalf("an uncertain entry without position is refused: %v", err)
	}
	r.Entries[0].Verified = true
	if err := r.Validate(); err == nil {
		t.Fatalf("a verified entry without position is accepted")
	}
}

// gridOf places drawn rows on a grid directly, as findGrids would.
func gridOf(rows [13][]rune) *xgGridRead {
	g := &xgGridRead{u: xgGrid, top: xgTopBorder, bottom: xgBotBorder, topMark: 'Û', bottomMark: 'Ð', turnMark: 'Ð'}
	for r := 1; r <= 11; r++ {
		copy(g.cells[r][:], rows[r])
	}
	return g
}

func TestTable(t *testing.T) {
	for _, x := range []string{start, stack7, race, topCube} {
		b := gridOf(xgRows(t, x)).decode()
		board, ok := b.boardXGID()
		if want := strings.Split(strings.TrimPrefix(x, "XGID="), ":")[0]; !ok || board != want {
			t.Errorf("%s read as %s", want, board)
		}
	}
	if c, n, ok := stackGlyph('Þ'); !ok || c != black || n != 8 {
		t.Errorf("Þ = %c %d", c, n)
	}
	if c, n, ok := offGlyph('è'); !ok || c != white || n != 3 {
		t.Errorf("è = %c %d", c, n)
	}
	if v, ok := cubeGlyph('$'); !ok || v != 8 {
		t.Errorf("$ = %d", v)
	}
	// What the reference prints never showed is read, but not trusted.
	cases := []struct {
		name  string
		patch func(*xgGridRead)
		field string
	}{
		{"numbered bar", func(g *xgGridRead) { g.cells[5][9] = 0xD1 }, "bar"},
		{"four black on the bar", func(g *xgGridRead) {
			for r := 7; r <= 10; r++ {
				g.cells[r][9] = 0xDB
			}
		}, "bar"},
		{"cube 16", func(g *xgGridRead) { g.cells[6][0] = 0; g.cells[11][0] = '%' }, "cube"},
		{"top player on roll", func(g *xgGridRead) { g.turnMark = 'Û' }, "turn"},
		{"black at the bottom", func(g *xgGridRead) { g.topMark, g.bottomMark = 'Ð', 'Û' }, "orientation"},
	}
	for _, c := range cases {
		g := gridOf(xgRows(t, start))
		c.patch(g)
		b := g.decode()
		found := false
		for _, d := range b.doubts {
			found = found || (d.Field == c.field && !d.hard)
		}
		if !found {
			t.Errorf("%s: doubts %+v, want a soft %q", c.name, b.doubts, c.field)
		}
	}
	g := gridOf(xgRows(t, start))
	g.cells[7][2] = 0xDB + 5 // a stack of ten on point 12, in place of the fifth rank
	g.cells[11][2], g.cells[10][2], g.cells[9][2], g.cells[8][2] = 'u', 'v', 'w', 'x'
	b := g.decode()
	if b.points[12].n != 10 || len(b.doubts) != 1 || b.doubts[0].Field != "point" || b.doubts[0].hard {
		t.Errorf("stack of ten: %+v %+v", b.points[12], b.doubts)
	}
}
