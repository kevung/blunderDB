package document

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/document/pdf"
)

// ErrUnknownBoardFont refuses a PDF whose diagrams are drawn with a board
// font whose glyphs cannot be read by their shapes; the error names it.
var ErrUnknownBoardFont = errors.New("document: board font not recognised")

// ErrNoGlyphDiagram refuses a PDF in which no diagram is drawn with a board
// font and no page is made of images: a page drawn with vectors is neither
// read by its glyphs nor rendered.
var ErrNoGlyphDiagram = errors.New("document: no diagram drawn with eXtreme Gammon's board font, and no page made of images (a page drawn with vectors is not rendered); give its pages as images: a folder of PNG or JPEG files, one a page")

// AnalyzeOptions are a caller's choices for AnalyzePDF.
type AnalyzeOptions struct {
	// Title and Author replace the PDF's own metadata when set.
	Title  string `json:"title"`
	Author string `json:"author"`
	// FileName names the document when neither Title nor the PDF gives a
	// title; its extension is dropped.
	FileName string `json:"fileName"`
	// Warn, when set, receives what the analysis passed over and why (the
	// pages of a scanned PDF that could not be composed, the board fonts
	// whose diagrams are not read while others are).
	Warn func(string) `json:"-"`
	// Context stops the reading: checked before each page and each
	// diagram drawn in a Snowie board font, and, in a scanned document,
	// between packets and code-blocks.
	// Every caller gives one (CancelImport, Ctrl-C); nil, for a test, is
	// context.Background.
	Context context.Context `json:"-"`
	// Progress, when set, receives the pages read so far after each page
	// (never decreasing, the last one always delivered); wrap it with
	// ThrottledProgress to bound its rate.
	Progress func(Progress) `json:"-"`
}

// Progress is how far a reading has gone: Page of Pages done, Diagrams found
// so far.
type Progress struct {
	Page     int `json:"page"`
	Pages    int `json:"pages"`
	Diagrams int `json:"diagrams"`
}

// ThrottledProgress wraps f so that it is called at most once per interval,
// except for the final report (Page == Pages), which is always delivered.
func ThrottledProgress(f func(Progress), interval time.Duration) func(Progress) {
	if f == nil {
		return nil
	}
	var last time.Time
	return func(p Progress) {
		now := time.Now()
		if p.Page < p.Pages && !last.IsZero() && now.Sub(last) < interval {
			return
		}
		last = now
		f(p)
	}
}

func (o AnalyzeOptions) report(page, pages, diagrams int) {
	if o.Progress != nil {
		o.Progress(Progress{Page: page, Pages: pages, Diagrams: diagrams})
	}
}

// pageHook adapts the progress to the structure reader's per-page report.
func (o AnalyzeOptions) pageHook(pages int) func(done, diagrams int) {
	if o.Progress == nil {
		return nil
	}
	return func(done, diagrams int) { o.report(done, pages, diagrams) }
}

func (o AnalyzeOptions) context() context.Context {
	if o.Context == nil {
		return context.Background()
	}
	return o.Context
}

// AnalyzePDFFile reads the PDF file at path, refusing one larger than the
// reader's limit before reading it, and analyses it with AnalyzePDF. The
// file's name names the document when nothing else does.
func AnalyzePDFFile(path string, opts AnalyzeOptions) (*Releve, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	limit := pdf.DefaultLimits.MaxFileSize
	data, err := io.ReadAll(io.LimitReader(f, int64(limit)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > limit {
		return nil, fmt.Errorf("document: %w: %s is larger than %d bytes", pdf.ErrTooLarge, path, limit)
	}
	if opts.FileName == "" {
		opts.FileName = path
	}
	return AnalyzePDF(data, opts)
}

// AnalyzePDF reads every diagram a PDF draws with eXtreme Gammon's board
// font and returns the Relevé, in the document's order. It writes nothing.
// A PDF without a page drawn with that font is read as a scanned book: its
// pages composed from their images, with their text layer, are read by the
// structure reader (package structure).
//
// Each diagram is read exactly from its glyphs, then cross-checked against
// what the print says next to it: fifteen checkers a side and the printed
// pip counts of both players. An entry is sure only when every check holds
// and nothing in the diagram lies outside what the glyph table was proven
// on; otherwise it is to-verify, or uncertain when the diagram is malformed,
// with each doubt named.
func AnalyzePDF(data []byte, opts AnalyzeOptions) (*Releve, error) {
	rd, err := pdf.Open(data, pdf.DefaultLimits)
	if err != nil {
		return nil, fmt.Errorf("document: %w", err)
	}
	sum := sha256.Sum256(data)
	title, author := rd.Info()
	if t := strings.TrimSpace(opts.Title); t != "" {
		title = t
	}
	if a := strings.TrimSpace(opts.Author); a != "" {
		author = a
	}
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(opts.FileName), filepath.Ext(opts.FileName))
	}
	if title == "" || title == "." {
		title = "Untitled document"
	}
	r := &Releve{Format: Format, Version: Version, Document: Source{
		Title: title, Author: author, SHA256: hex.EncodeToString(sum[:]),
	}}
	outline := rd.Outline()
	sort.SliceStable(outline, func(a, b int) bool { return outline[a].Page < outline[b].Page })
	boardFont := false
	sr := newSnowieRead(opts.context())
	unreadFonts := map[string]bool{}
	for n := 1; n <= rd.NumPages(); n++ {
		if err := opts.context().Err(); err != nil {
			return nil, fmt.Errorf("document: %w", err)
		}
		page, err := rd.Page(n)
		if err != nil {
			return nil, fmt.Errorf("document: %w", err)
		}
		listed := pageHasBoardFont(rd.Fonts(page))
		gs, err := rd.Glyphs(page)
		if err != nil {
			if listed {
				return nil, fmt.Errorf("document: page %d draws with a board font but cannot be read: %w", n, err)
			}
			continue
		}
		// A font that maps every frame code but is only drawn invisibly is
		// an OCR layer's, not a board's.
		gs = visibleGlyphs(gs)
		if listed && drawsBoardFont(gs) {
			boardFont = true
		}
		entries, profile := readPage(gs, page)
		if len(entries) == 0 {
			var unread []string
			entries, profile, unread, err = readSnowiePage(gs, page, sr)
			if err != nil {
				return nil, fmt.Errorf("document: %w", err)
			}
			if len(entries) > 0 || len(unread) > 0 {
				boardFont = true
			}
			for _, f := range unread {
				unreadFonts[f] = true
			}
		}
		if r.Document.Profile == nil && profile != nil {
			r.Document.Profile = profile
		}
		for _, e := range entries {
			e.Rank = len(r.Entries) + 1
			e.Page = n
			e.Chapter = chapterOf(outline, n)
			r.Entries = append(r.Entries, e)
		}
		// A scanned book has no entry here: its progress is the structure
		// reader's, so as not to count its pages twice.
		if boardFont || len(r.Entries) > 0 {
			opts.report(n, rd.NumPages(), len(r.Entries))
		}
	}
	if len(r.Entries) == 0 && !boardFont {
		return analyzeScans(rd, r, opts, outline)
	}
	names := make([]string, 0, len(unreadFonts))
	for f := range unreadFonts {
		names = append(names, strconv.Quote(f))
	}
	sort.Strings(names)
	if len(r.Entries) > 0 && len(names) > 0 && opts.Warn != nil {
		opts.Warn(fmt.Sprintf("diagrams drawn with the board font %s are not read: its glyph shapes are not embedded in a form this reader knows (a TrueType program)", strings.Join(names, ", ")))
	}
	if len(r.Entries) == 0 && len(names) > 0 {
		return nil, fmt.Errorf("%w: the diagrams are drawn with the board font %s, whose glyph shapes are not embedded in a form this reader knows (a TrueType program)", ErrUnknownBoardFont, strings.Join(names, ", "))
	}
	if len(r.Entries) == 0 {
		return nil, ErrNoGlyphDiagram
	}
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return r, nil
}

// visibleGlyphs drops the invisible text layer.
func visibleGlyphs(gs []pdf.Glyph) []pdf.Glyph {
	out := gs[:0:0]
	for _, g := range gs {
		if !g.Invisible {
			out = append(out, g)
		}
	}
	return out
}

// drawsBoardFont reports whether a glyph is drawn in a font that maps the
// frame codes.
func drawsBoardFont(gs []pdf.Glyph) bool {
	for _, g := range gs {
		if g.Font.Maps(xgFrameRunes...) {
			return true
		}
	}
	return false
}

func pageHasBoardFont(fs []*pdf.Font) bool {
	for _, f := range fs {
		if f.Maps(xgFrameRunes...) {
			return true
		}
	}
	return false
}

func chapterOf(outline []pdf.OutlineItem, page int) string {
	ch := ""
	for _, it := range outline {
		if it.Page > page {
			break
		}
		ch = it.Title
	}
	return ch
}

// textLine is a run of text glyphs sharing a baseline.
type textLine struct {
	x, y, size float64
	text       string
}

// textLines groups the glyphs of every font but the board font into lines,
// top to bottom, left to right.
func textLines(gs []pdf.Glyph) []textLine {
	byY := map[float64][]pdf.Glyph{}
	for _, g := range gs {
		if !g.Upright || g.Font.Maps(xgFrameRunes...) || g.Text == "" {
			continue
		}
		y := math.Round(g.Y*2) / 2
		byY[y] = append(byY[y], g)
	}
	var lines []textLine
	for y, row := range byY {
		sort.SliceStable(row, func(a, b int) bool { return row[a].X < row[b].X })
		var sb strings.Builder
		start := 0
		for i, g := range row {
			if i > 0 {
				prev := row[i-1]
				gap := g.X - (prev.X + prev.W)
				if gap > 2*g.Size { // another column of text on the same baseline
					lines = append(lines, textLine{row[start].X, y, row[start].Size, strings.TrimSpace(sb.String())})
					sb.Reset()
					start = i
				} else if gap > 0.25*g.Size && !strings.HasSuffix(sb.String(), " ") {
					sb.WriteByte(' ')
				}
			}
			sb.WriteString(g.Text)
		}
		lines = append(lines, textLine{row[start].X, y, row[start].Size, strings.TrimSpace(sb.String())})
	}
	sort.Slice(lines, func(a, b int) bool {
		if lines[a].y != lines[b].y {
			return lines[a].y > lines[b].y
		}
		return lines[a].x < lines[b].x
	})
	return lines
}

var (
	panelRE = regexp.MustCompile(`(?i)score:\s*(\d+)|pip:\s*(\d+)|(\d+)\s*point match|(crawford)`)
	labelRE = regexp.MustCompile(`^(?i:problem|position|diagram|quiz|exercise|puzzle)\s+#?\d+[a-z]?[.:]?$`)
)

// panel is what XG prints right of a diagram: from the top, the top player's
// score and pips, the match length, the bottom player's pips and score.
type panel struct {
	kinds            string
	scoreTop, pipTop int
	length           int
	pipBot, scoreBot int
	crawford         bool
}

func readPanel(lines []textLine) panel {
	var p panel
	var vals []int
	for _, l := range lines {
		for _, m := range panelRE.FindAllStringSubmatch(l.text, -1) {
			switch {
			case m[1] != "":
				p.kinds += "s"
				vals = append(vals, atoi(m[1]))
			case m[2] != "":
				p.kinds += "p"
				vals = append(vals, atoi(m[2]))
			case m[3] != "":
				p.kinds += "m"
				vals = append(vals, atoi(m[3]))
			default:
				p.crawford = true
			}
		}
	}
	if p.kinds == "spmps" {
		p.scoreTop, p.pipTop, p.length, p.pipBot, p.scoreBot = vals[0], vals[1], vals[2], vals[3], vals[4]
	}
	return p
}

func atoi(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil || n > 1<<20 {
		return -1
	}
	return n
}

func boardGlyphs(gs []pdf.Glyph) []bglyph {
	var out []bglyph
	for _, g := range gs {
		if g.Upright && g.Font.Maps(xgFrameRunes...) {
			out = append(out, bglyph{g.X, g.Y, g.Size, g.Text})
		}
	}
	return out
}

// readPage returns one entry per diagram of the page, in reading order, and
// the profile of the first.
func readPage(gs []pdf.Glyph, page pdf.Page) ([]Entry, *Profile) {
	grids := findGrids(boardGlyphs(gs))
	if len(grids) == 0 {
		return nil, nil
	}
	lines := textLines(gs)
	var entries []Entry
	var profile *Profile
	for i, g := range grids {
		b := g.decode()
		limit := math.Inf(1)
		for j, o := range grids {
			if j != i && o.x0 > g.x0+g.u && o.x0 < limit && o.y0 > g.y0-13*g.u && o.y0 < g.y0+13*g.u {
				limit = o.x0
			}
		}
		var side []textLine
		for _, l := range lines {
			if l.x >= g.right()-g.u && l.x < limit && l.y <= g.y0+g.u && l.y >= g.y0-12.5*g.u {
				side = append(side, l)
			}
		}
		e := entryOf(b, readPanel(side))
		e.Label = labelAbove(lines, g, limit)
		e.Rect = rectOf(g, page.Box)
		entries = append(entries, e)
		if profile == nil {
			bottom := "white"
			if b.bottomColor == black {
				bottom = "black"
			}
			profile = &Profile{
				Home: "right", Bottom: bottom,
				Pips:      "printed right of the diagram, with the score and the match length",
				Cube:      "left of the diagram",
				Numbering: "13-24 along the top, 12-1 along the bottom",
			}
		}
	}
	return entries, profile
}

// labelAbove is the nearest label line ("Problem 191") above the diagram,
// within its column.
func labelAbove(lines []textLine, g *xgGridRead, limit float64) string {
	best := ""
	bestY := math.Inf(1)
	for _, l := range lines {
		if l.y > g.y0+g.u/2 && l.x >= g.x0-g.u && l.x < limit && l.y < bestY && labelRE.MatchString(l.text) {
			best, bestY = l.text, l.y
		}
	}
	return best
}

func rectOf(g *xgGridRead, box [4]float64) *Rect {
	w, h := box[2]-box[0], box[3]-box[1]
	if w <= 0 || h <= 0 {
		return nil
	}
	clamp := func(v float64) float64 { return math.Max(0, math.Min(1, v)) }
	x0 := clamp((g.x0 - box[0]) / w)
	x1 := clamp((g.right() - box[0]) / w)
	y0 := clamp((box[3] - (g.y0 + g.u)) / h)
	y1 := clamp((box[3] - (g.y0 - 12.25*g.u)) / h)
	if x1 <= x0 || y1 <= y0 {
		return nil
	}
	return &Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

// entryOf cross-checks a decoded diagram against its printed panel and
// builds its entry.
func entryOf(b *xgBoard, p panel) Entry {
	tc := otherColor(b.bottomColor)
	for _, c := range []byte{b.bottomColor, tc} {
		if _, total := b.counts(c); total != 15 {
			b.doubt(false, "checkers", 0, "%s has %d checkers on the board, the bar and the tray, not 15", colorName(c), total)
		}
	}
	length, s1, s2, crawford := 0, 0, 0, 0
	if p.kinds != "spmps" {
		b.doubt(false, "score", 0, "the printed score, pips and match length were not found next to the diagram")
	} else {
		pb, pt := b.pips()
		if pb != p.pipBot || pt != p.pipTop {
			b.doubt(false, "pips", 0, "pips read %d/%d, printed %d/%d (bottom/top)", pb, pt, p.pipBot, p.pipTop)
		}
		switch {
		case p.length < 1 || p.length > 64 || p.scoreBot < 0 || p.scoreTop < 0 || p.scoreBot >= p.length || p.scoreTop >= p.length:
			b.doubt(true, "score", 0, "printed score %d-%d to %d is not a match score", p.scoreBot, p.scoreTop, p.length)
		default:
			length, s1, s2 = p.length, p.scoreBot, p.scoreTop
			if p.crawford {
				crawford = 1
				b.doubt(false, "score", 0, "a Crawford game, never seen in the reference prints")
			} else if s1 == length-1 || s2 == length-1 {
				b.doubt(false, "score", 0, "a player 1-away: the Crawford game and the games after it are not told apart")
			}
		}
	}
	return entryFromBoard(b, "00", length, s1, s2, crawford, p.kinds == "spmps")
}

// entryFromBoard builds the entry of a decoded diagram: its XGID, and its
// two certainties lowered by its doubts. pipsChecked says whether the
// printed pip counts were found to confirm the checkers.
func entryFromBoard(b *xgBoard, dice string, length, s1, s2, crawford int, pipsChecked bool) Entry {
	e := Entry{}
	board, ok := b.boardXGID()
	if ok {
		lg := 0
		for v := b.cube; v > 1; v >>= 1 {
			lg++
		}
		xgid := fmt.Sprintf("XGID=%s:%d:%d:%d:%s:%d:%d:%d:%d:10", board, lg, b.cubeOwner, b.turn, dice, s1, s2, crawford, length)
		if _, err := ParsePosition(xgid); err != nil {
			b.doubt(true, "position", 0, "the reading is not a legal position: %v", err)
		} else {
			e.Position = xgid
		}
	} else {
		b.doubt(true, "position", 0, "more than 15 checkers on one point")
	}
	// The print's band gives the context: it is read. A doubt lowers the
	// part of the reading it is about; without the band, the pips that
	// confirm the checkers are missing too.
	e.Checkers, e.Context = Sure, ContextRead
	lower := func(s *State, hard bool) {
		if hard {
			*s = Uncertain
		} else if *s == Sure {
			*s = ToVerify
		}
	}
	for _, d := range b.doubts {
		e.Doubts = append(e.Doubts, d.Doubt)
		if !contextField[d.Field] {
			lower(&e.Checkers, d.hard)
			continue
		}
		c := State(Sure)
		if e.Context == ContextToVerify {
			c = ToVerify
		} else if e.Context == ContextUncertain {
			c = Uncertain
		}
		lower(&c, d.hard)
		e.Context = map[State]ContextState{Sure: ContextRead, ToVerify: ContextToVerify, Uncertain: ContextUncertain}[c]
	}
	if !pipsChecked {
		lower(&e.Checkers, false)
	}
	if e.Position == "" {
		e.Checkers = Uncertain
	}
	return e
}

func colorName(c byte) string {
	if c == white {
		return "white"
	}
	return "black"
}
