// Package structure reads the backgammon diagrams of page images by their
// structure, not by a style: two rows of twelve points cut by a bar, and
// checkers that are discs of two classes stacked from a point's base. A
// document draws all its diagrams the same way, so the reader learns its
// style from the document itself, in two passes, and keeps nothing from one
// document to the next:
//
//  1. Profile: the checker's ellipse and stacking step, learned on the holes
//     of the light checkers of the first diagrams found (at least 30).
//  2. Reading: every page is normalised (divided by its paper), its large
//     components of ink straightened and framed (two halves, a bar, points
//     proven at their mid-height), and each column followed slot by slot.
//  3. Relevé: the two checker classes, the printed numbering, where the pip
//     counts are printed and the direction of play are learned over all the
//     diagrams, then each diagram is judged.
//
// The printed pip counts come from the text layer and from the image
// itself (numbers.go): numbers standing alone around a diagram are read
// figure by figure — against the document's own numbering, labelled by its
// shape, eight embedded typefaces, then the figures read elsewhere in the
// document — and a number is read only when every figure is clear; the two
// places where the document prints its pips are learned by a vote over the
// diagrams read with fifteen checkers a side. No figure is chosen after a
// diagram's pips, and nothing is kept from one document to the next. When
// the text and the image give different pips, neither is taken and the
// disagreement is a named doubt.
//
// A diagram is sure only when the pip counts printed beside it agree with
// both sides of the reading, no side has more than fifteen checkers and
// nothing is in doubt. The printed pips may also fix the height of one
// ambiguous stack, and the reading is then sure only if that side reaches
// fifteen checkers, a second equation the pips did not give. The checkers
// borne off are counted in the tray when the document draws them there as
// bars: a side's drawn and tray checkers make fifteen, or the gap is a named
// doubt. With no tray read they are taken as fifteen less those drawn, and
// the direction of play when no pips confirm it is a supposition: both are
// named doubts. A document whose two checker classes are both lighter than
// mid-grey draws no filled checker (letters X and O): it is not read, each
// diagram uncertain with that cause.
//
// The context — cube, dice, player on roll, score — is read apart
// (context.go, Diagram.Context), a field certified only by a cross-check: a
// cube's value read in its square or in the text, coherent with its place;
// dice counted in the image and confirmed by the text, their colour and their
// half of the board telling the same player on roll; a score panel whose
// scores the printed pips assign. A cube not found is not centred, and what is
// drawn or printed and not certified is a named doubt, never invented.
//
// Pages and text come from the caller: an image per page (grey or colour,
// 150 to 300 dpi) and, when the document has one, its text layer as words
// with their boxes in the image's pixels; a page the caller cannot give is
// passed over with its reason (Page.Unread, Result.Unread). Images are bounded (MaxPixels) and
// never kept: a page is decoded, read and dropped, twice (once per pass).
package structure

import (
	"context"
	"errors"
	"fmt"
	"image"

	"github.com/kevung/blunderdb/pkg/blunderdb/capture"
)

// MaxPixels bounds one page image: a letter page at 300 dpi is 8.4 million
// pixels; twelve times as many is not a page.
const MaxPixels = 100_000_000

// MaxPages bounds a document.
const MaxPages = 10_000

// minLearned is how many diagrams the checker's shape is learned on.
const minLearned = 30

// ErrNoDiagram is the refusal of a document in which no diagram was found.
var ErrNoDiagram = errors.New("structure: no backgammon diagram was found in the pages")

// ErrNoCheckerShape is the refusal of a document whose diagrams show too few
// light checkers to learn their shape.
var ErrNoCheckerShape = errors.New("structure: too few light checkers to learn how the document draws them")

// ErrPage is the refusal of a page the reader does not take.
var ErrPage = errors.New("structure: page refused")

// ErrNoPage is the refusal of a document none of whose pages could be read.
var ErrNoPage = errors.New("structure: no page of the document could be read")

// errUnread marks a page passed over (Page.Unread).
var errUnread = errors.New("structure: page passed over")

// inspectSources, set by the measure only, sees the diagrams judged with the
// text layer's pips alone and with the image's alone, beside the reading.
var inspectSources func(text, image, both []Diagram)

// Word is a word of a page's text layer, its box in the page image's pixels.
type Word struct {
	X0, Y0, X1, Y1 float64
	Text           string
}

// Page is one page of a document.
type Page struct {
	Image image.Image
	// DPI is the image's resolution; a diagram is at least 0.8 inch wide.
	DPI   float64
	Words []Word
	// Unread, when set, says why the page has no image to read: the reader
	// passes over it and reads the others.
	Unread error
}

// Pages are a document's pages, in order. Page may be called more than once
// for one page: the reader reads the document twice and keeps no image.
type Pages interface {
	Len() int
	Page(i int) (Page, error)
}

// State is how far a diagram's checkers can be trusted; the context has its
// own certainty (Context.Read), and a context doubt holds the state to verify.
type State int

const (
	ToVerify State = iota
	// Confirmed — the printed pip counts confirm the checkers and nothing
	// drawn (cube off the centre, dice) is left unread.
	Confirmed
	Uncertain
)

// Diagram is the reading of one diagram.
type Diagram struct {
	Page     int             // index of the page, from 0
	Rect     image.Rectangle // where the diagram lies on its page, in pixels
	PageSize [2]int          // the page image's width and height
	// Board is the XGID board (26 characters): the player whose home is in
	// the bottom row is the upper-case one. Empty when the reading is not a
	// position.
	Board    string
	State    State
	Doubts   []capture.Doubt
	Families []string // the named causes, to verify diagrams by family
	Checkers [2]int   // checkers drawn, bottom player then top player
	Pips     [2]int   // pips read, bottom then top
	Printed  *[2]int  // pips printed, the upper number then the lower one
	// TextPips and ImagePips are the printed pips as the text layer and the
	// image give them; Printed is the one taken, nil when they disagree.
	TextPips, ImagePips *[2]int
	// Complete is fifteen checkers a side read with no doubt: not a state,
	// kept as information.
	Complete bool
	// CheckersSure is a reading whose checkers would be sure; only a cube
	// or dice drawn and not read hold it to verify.
	CheckersSure bool
	// Context is the cube, dice, player on roll and score as the diagram
	// prints them, each given only when read (readContext).
	Context Context
}

// NoPipsReason is the reason of the "pips" doubt of a diagram next to which
// no pip count was found — absent, as against printed and disagreeing.
const NoPipsReason = "no printed pip count was found next to the diagram"

// AssumedDirectionReason is the reason of the "orientation" doubt of a
// document whose direction of play no printed pips voted for.
const AssumedDirectionReason = "the direction of play is assumed (no printed pips confirm it): checkers taken to move toward the side with the fewer pips"

// Profile is what the reader learned of the document.
type Profile struct {
	Rx, Ry, Step   float64 // checker half axes and stacking step, in point widths
	LearnedOn      int     // diagrams the checker was learned on
	Classes        [2]float64
	HomeSide       byte // 'L' or 'R': side of the bottom player's home
	BottomIsDark   bool
	DirectionVoted bool   // false: assumed from the fewer pips
	Numbering      string // corner of point 1 per the printed numbering ("BL"…), or ""
	PipPlace       byte   // 'G' left, 'D' right, 'L' tagged "pips", 'B' at the bar, or 0
	// PipsInImage: the two places where the document prints its pip counts
	// were learned on its images, and the pips are read there.
	PipsInImage bool
}

// Result is the reading of a document.
type Result struct {
	Profile  Profile
	Diagrams []Diagram
	Rejected int // components shaped or framed unlike a diagram
	// Unread are the pages passed over, each with its reason (Page.Unread).
	Unread []PageError
}

// PageError is a page the reader passed over.
type PageError struct {
	Page int // from 0
	Err  error
}

// inspectReading, set by the measure only, sees each raw reading.
var inspectReading func(*boardReading)

func image2Rect(b box) image.Rectangle { return image.Rect(b.x0, b.y0, b.x1, b.y1) }

func loadPage(pages Pages, i int) (*raster, Page, error) {
	pg, err := pages.Page(i)
	if err != nil {
		return nil, pg, err
	}
	if pg.Unread != nil {
		return nil, pg, errUnread
	}
	if pg.Image == nil {
		return nil, pg, fmt.Errorf("%w: page %d has no image", ErrPage, i+1)
	}
	b := pg.Image.Bounds()
	if b.Dx() < 32 || b.Dy() < 32 || b.Dx()*b.Dy() > MaxPixels || b.Dx() > 40_000 || b.Dy() > 40_000 {
		return nil, pg, fmt.Errorf("%w: page %d is %d×%d pixels", ErrPage, i+1, b.Dx(), b.Dy())
	}
	if pg.DPI < 50 || pg.DPI > 1200 {
		return nil, pg, fmt.Errorf("%w: page %d at %.0f dpi, want 50 to 1200", ErrPage, i+1, pg.DPI)
	}
	return normalize(fromImage(pg.Image)), pg, nil
}

func diagramsOf(g *raster, dpi float64) ([]zone, int) {
	var ok []zone
	rejected := 0
	for _, z := range locate(g, dpi) {
		if z.reject == "" {
			ok = append(ok, z)
		} else {
			rejected++
		}
	}
	return readingOrder(ok, g.w), rejected
}

// readPage is what reading a page keeps of it: its diagrams' zones, the
// figures printed in each, and the page's size.
type readPage struct {
	pg       Page
	zs       []zone
	rejected int
	numbers  [][]printedNumber
	captions [][][]captionWord
	w, h     int
}

// readPageAt loads page i and reads its zones; an unread page is returned
// with pg.Unread set and no error.
func readPageAt(pages Pages, i int) (*readPage, error) {
	g, pg, err := loadPage(pages, i)
	if errors.Is(err, errUnread) {
		return &readPage{pg: pg}, nil
	}
	if err != nil {
		return nil, err
	}
	pg.Image = nil
	rp := &readPage{pg: pg, w: g.w, h: g.h}
	rp.zs, rp.rejected = diagramsOf(g, pg.DPI)
	for _, z := range rp.zs {
		rp.numbers = append(rp.numbers, extractNumbers(g, z.bbox))
		if len(pg.Words) == 0 {
			rp.captions = append(rp.captions, extractCaptions(g, z.bbox))
		}
	}
	return rp, nil
}

// Read reads every diagram of the pages, in the document's order.
func Read(ctx context.Context, pages Pages) (*Result, error) {
	n := pages.Len()
	if n > MaxPages {
		return nil, fmt.Errorf("%w: %d pages, at most %d", ErrPage, n, MaxPages)
	}
	var samples [][]hole
	unread, firstUnread := 0, error(nil)
	var ps []float64
	// The pages the shape is learned on are kept as read, so that the
	// second pass decodes no page twice: what it needs of a page is its
	// zones and the figures printed in them, never the page's raster.
	memo := map[int]*readPage{}
	for i := 0; i < n && len(samples) < minLearned; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		rp, err := readPageAt(pages, i)
		if err != nil {
			return nil, err
		}
		memo[i] = rp
		if rp.pg.Unread != nil {
			if unread++; firstUnread == nil {
				firstUnread = fmt.Errorf("page %d: %w", i+1, rp.pg.Unread)
			}
			continue
		}
		for _, z := range rp.zs {
			samples = append(samples, holes(z.ink, z.frame))
			ps = append(ps, z.frame.pointWidth())
		}
	}
	if len(samples) == 0 {
		if n > 0 && unread == n {
			return nil, fmt.Errorf("%w: %w", ErrNoPage, firstUnread)
		}
		return nil, ErrNoDiagram
	}
	geo, ok := estimateGeometry(samples, ps)
	if !ok {
		return nil, ErrNoCheckerShape
	}
	res := &Result{Profile: Profile{Rx: geo.rx, Ry: geo.ry, Step: geo.step, LearnedOn: geo.learnedOn}}
	var rds []*boardReading
	words := map[int][]Word{}
	for i := 0; i < n; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		rp := memo[i]
		delete(memo, i)
		if rp == nil {
			var err error
			if rp, err = readPageAt(pages, i); err != nil {
				return nil, err
			}
		}
		pg, zs := rp.pg, rp.zs
		if pg.Unread != nil {
			res.Unread = append(res.Unread, PageError{i, pg.Unread})
			continue
		}
		res.Rejected += rp.rejected
		for k, z := range zs {
			rd := readBoard(z, geo)
			rd.digits, rd.cubes, rd.dice, rd.field, rd.trays = z.digits, z.cubes, z.dice, z.field, z.trays
			rd.page, rd.bbox, rd.pageW, rd.pageH = i, z.bbox, rp.w, rp.h
			rd.numbers = rp.numbers[k]
			if k < len(rp.captions) {
				rd.captions = rp.captions[k]
			}
			rds = append(rds, &rd)
			if inspectReading != nil {
				inspectReading(&rd)
			}
		}
		if len(zs) > 0 && len(pg.Words) > 0 {
			words[i] = pg.Words
		}
	}
	// The pips printed in the image: the document's figures read, then the
	// two places of its pip counts learned on the readings judged without
	// any printed pips, under the direction of play that gathers the most
	// voters, then read beside each diagram.
	all := make([][]printedNumber, len(rds))
	for i, rd := range rds {
		all[i] = rd.numbers
	}
	digits := readNumbers(all)
	var slots []pipSlot
	voters := 0
	for _, h := range corners {
		bare, _ := releve(rds, nil, nil, h)
		if s, v := learnPipSlots(all, bare); s != nil && v > voters {
			slots, voters = s, v
		}
	}
	stretch := 1.0
	if geo.rx > 0.03 && geo.ry > 0.03 {
		stretch = (geo.rx - 0.03) / (geo.ry - 0.03)
	}
	keepSquares(rds, stretch)
	img := make([]*[2]int, len(rds))
	for i, rd := range rds {
		img[i] = imagePips(rd.numbers, rd.bbox, slots)
		rd.cubeFigs = cubeFigures(rd)
		rd.numbers = nil
	}
	ds, prof := releve(rds, words, img, "")
	// A page without a text layer gives the words read in its captions,
	// to the context only: the pips are read in the image.
	captions := map[int][][][]captionWord{}
	for _, rd := range rds {
		if rd.captions != nil {
			captions[rd.page] = append(captions[rd.page], rd.captions)
			rd.captions = nil
		}
	}
	text := map[int][]Word{}
	for pg, ls := range captions {
		text[pg] = captionWords(ls, captionFigures(digits))
	}
	envs := contextEnvs(rds)
	for i := range ds {
		ws := words[rds[i].page]
		if ws == nil {
			ws = text[rds[i].page]
		}
		readContext(rds[i], &ds[i], ws, prof, envs[i])
	}
	missingCubes(rds, ds)
	if inspectSources != nil {
		text, _ := releve(rds, words, nil, "")
		image, _ := releve(rds, nil, img, "")
		inspectSources(text, image, ds)
	}
	prof.PipsInImage = slots != nil
	prof.Rx, prof.Ry, prof.Step, prof.LearnedOn = geo.rx, geo.ry, geo.step, geo.learnedOn
	res.Profile, res.Diagrams = prof, ds
	return res, nil
}
