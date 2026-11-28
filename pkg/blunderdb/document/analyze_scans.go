package document

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/document/pdf"
	"github.com/kevung/blunderdb/pkg/blunderdb/document/structure"
)

// analyzeScans reads a PDF whose diagrams are not drawn with a board font:
// each page is composed from its images (pdf.Scan) and given, with the words
// of its text layer, to the structure reader. A page that cannot be composed
// is passed over and named through opts.Warn; when none can, the error says
// why for the first.
func analyzeScans(rd *pdf.Reader, r *Releve, opts AnalyzeOptions, outline []pdf.OutlineItem) (*Releve, error) {
	n := rd.NumPages()
	if n > structure.MaxPages {
		return nil, fmt.Errorf("document: %d pages, at most %d", n, structure.MaxPages)
	}
	res, err := structure.ReadProgress(opts.context(), pdfScans{opts.context(), rd, n}, opts.pageHook(n))
	if errors.Is(err, structure.ErrNoPage) && errors.Is(err, pdf.ErrNoPageImage) {
		return nil, fmt.Errorf("%w; %w", ErrNoGlyphDiagram, err)
	}
	if errors.Is(err, structure.ErrNoDiagram) {
		return nil, fmt.Errorf("document: no backgammon diagram found in the PDF's pages: %w", err)
	}
	if err != nil {
		return nil, fmt.Errorf("document: %w", err)
	}
	if opts.Warn != nil && len(res.Unread) > 0 {
		opts.Warn(unreadSummary(res.Unread))
	}
	r.Document.Profile = structureProfile(res.Profile)
	r.Document.Defaults = learnDefaults(res.Diagrams, res.Profile)
	for _, d := range res.Diagrams {
		e := structureEntry(d, len(r.Entries)+1, r.Document.Defaults)
		e.Chapter = chapterOf(outline, e.Page)
		r.Entries = append(r.Entries, e)
	}
	if len(r.Entries) == 0 {
		return nil, fmt.Errorf("document: no backgammon diagram found in the PDF's pages: %w", structure.ErrNoDiagram)
	}
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return r, nil
}

// unreadSummary names the pages passed over, grouped by reason.
func unreadSummary(us []structure.PageError) string {
	byReason := map[string][]string{}
	var order []string
	for _, u := range us {
		k := u.Err.Error()
		if _, ok := byReason[k]; !ok {
			order = append(order, k)
		}
		byReason[k] = append(byReason[k], fmt.Sprint(u.Page+1))
	}
	sort.SliceStable(order, func(a, b int) bool { return len(byReason[order[a]]) > len(byReason[order[b]]) })
	var parts []string
	for _, k := range order {
		parts = append(parts, fmt.Sprintf("p. %s: %s", strings.Join(byReason[k], ", "), k))
	}
	return fmt.Sprintf("%d pages not read; %s", len(us), strings.Join(parts, "; "))
}

// pdfScans are a PDF's pages, each composed from its images when the
// structure reader asks for it and dropped once read.
type pdfScans struct {
	ctx context.Context
	r   *pdf.Reader
	n   int
}

func (s pdfScans) Len() int { return s.n }

func (s pdfScans) Page(i int) (structure.Page, error) {
	p, err := s.r.Page(i + 1)
	if err != nil {
		return structure.Page{}, err
	}
	sc, err := s.r.ScanContext(s.ctx, p)
	if err != nil {
		return structure.Page{Unread: err}, nil
	}
	b := sc.Image.Rect
	if b.Dx() < 32 || b.Dy() < 32 || b.Dx()*b.Dy() > structure.MaxPixels || sc.DPI < 50 {
		return structure.Page{Unread: fmt.Errorf("a page of %d×%d pixels at %.0f dpi", b.Dx(), b.Dy(), sc.DPI)}, nil
	}
	ws := make([]structure.Word, len(sc.Words))
	for k, w := range sc.Words {
		ws[k] = structure.Word{X0: w.X0, Y0: w.Y0, X1: w.X1, Y1: w.Y1, Text: w.Text}
	}
	return structure.Page{Image: sc.Image, DPI: sc.DPI, Words: ws}, nil
}
