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
// A diagram is sure only when the pip counts printed beside it agree with
// both sides of the reading, no side has more than fifteen checkers and
// nothing is in doubt. The printed pips may also fix the height of one
// ambiguous stack, and the reading is then sure only if that side reaches
// fifteen checkers, a second equation the pips did not give. What the reader
// does not read — the cube's value and owner, the dice, the score, the
// checkers borne off (taken as fifteen less those drawn), the direction of
// play when no pips confirm it — is never invented: a drawn cube off the
// centre, drawn dice, a half board, or a supposition the pips do not lift is
// a named doubt.
//
// Pages and text come from the caller: an image per page (grey or colour,
// 150 to 300 dpi) and, when the document has one, its text layer as words
// with their boxes in the image's pixels. Images are bounded (MaxPixels) and
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
}

// Pages are a document's pages, in order. Page may be called more than once
// for one page: the reader reads the document twice and keeps no image.
type Pages interface {
	Len() int
	Page(i int) (Page, error)
}

// State is how far a diagram's reading can be trusted.
type State int

const (
	ToVerify State = iota
	Sure
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
	// Complete is fifteen checkers a side read with no doubt: not a state,
	// kept as information.
	Complete bool
	// CheckersSure is a reading whose checkers would be sure; only a cube
	// or dice drawn and not read hold it to verify.
	CheckersSure bool
}

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
}

// Result is the reading of a document.
type Result struct {
	Profile  Profile
	Diagrams []Diagram
	Rejected int // components shaped or framed unlike a diagram
}

func image2Rect(b box) image.Rectangle { return image.Rect(b.x0, b.y0, b.x1, b.y1) }

func loadPage(pages Pages, i int) (*raster, Page, error) {
	pg, err := pages.Page(i)
	if err != nil {
		return nil, pg, err
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

// Read reads every diagram of the pages, in the document's order.
func Read(ctx context.Context, pages Pages) (*Result, error) {
	n := pages.Len()
	if n > MaxPages {
		return nil, fmt.Errorf("%w: %d pages, at most %d", ErrPage, n, MaxPages)
	}
	var samples [][]hole
	var ps []float64
	for i := 0; i < n && len(samples) < minLearned; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		g, pg, err := loadPage(pages, i)
		if err != nil {
			return nil, err
		}
		zs, _ := diagramsOf(g, pg.DPI)
		for _, z := range zs {
			samples = append(samples, holes(z.ink, z.frame))
			ps = append(ps, z.frame.pointWidth())
		}
	}
	if len(samples) == 0 {
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
		g, pg, err := loadPage(pages, i)
		if err != nil {
			return nil, err
		}
		zs, rej := diagramsOf(g, pg.DPI)
		res.Rejected += rej
		for _, z := range zs {
			rd := readBoard(z, geo)
			rd.digits, rd.cubes, rd.dice = z.digits, z.cubes, z.dice
			rd.page, rd.bbox, rd.pageW, rd.pageH = i, z.bbox, g.w, g.h
			rds = append(rds, &rd)
		}
		if len(zs) > 0 && len(pg.Words) > 0 {
			words[i] = pg.Words
		}
	}
	ds, prof := releve(rds, words)
	prof.Rx, prof.Ry, prof.Step, prof.LearnedOn = geo.rx, geo.ry, geo.step, geo.learnedOn
	res.Profile, res.Diagrams = prof, ds
	return res, nil
}
