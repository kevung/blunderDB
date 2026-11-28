package document

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"

	"golang.org/x/image/draw"

	"github.com/kevung/blunderdb/pkg/blunderdb/document/jpx"
	"github.com/kevung/blunderdb/pkg/blunderdb/document/pdf"
)

// SheetEntry is one entry of a Relevé as the verification sheet shows it:
// where its diagram sits in the document, what was read, how far it is
// trusted and what to compare against the diagram.
type SheetEntry struct {
	Rank      int    `json:"rank"`
	Page      int    `json:"page"`
	Reference string `json:"reference"`
	Position  string `json:"position"`
	// State is the entry's certainty (Releve.StateOf); Verified is set once
	// a person compared it with its diagram, whatever its state.
	State    State `json:"state"`
	Verified bool  `json:"verified"`
	// Fields names what to compare, as ReviewItem.Fields does; empty for a
	// sure entry.
	Fields []string `json:"fields"`
	Rect   *Rect    `json:"rect,omitempty"`
}

// Sheet lists every entry of the Relevé, in its order, for the verification
// sheet; the counts of a Summary follow from the same states.
func Sheet(r *Releve) []SheetEntry {
	out := make([]SheetEntry, 0, len(r.Entries))
	for _, e := range r.Entries {
		st := r.StateOf(e)
		fields := []string{}
		if st != Sure {
			fields, _ = reviewFields(e)
		}
		out = append(out, SheetEntry{
			Rank: e.Rank, Page: e.Page, Reference: e.Reference(), Position: e.Position,
			State: st, Verified: e.Verified, Fields: fields, Rect: e.Rect,
		})
	}
	return out
}

// ErrNoEntry is the refusal of a rank the Relevé does not hold.
var ErrNoEntry = fmt.Errorf("document: no such entry: %w", ErrInvalid)

// SetVerified records a person's verdict on the entry of the given rank:
// verified, or no longer so. A non-empty position is the person's
// correction of what was read, written in place of it; an entry whose
// corrected context leaves the document's defaults keeps it as its own, to
// verify, so that the import never puts the defaults over it. The Relevé is
// left unchanged when the result would not be valid.
func SetVerified(r *Releve, rank int, verified bool, position string) error {
	i := slices.IndexFunc(r.Entries, func(e Entry) bool { return e.Rank == rank })
	if i < 0 {
		return fmt.Errorf("%w: rank %d", ErrNoEntry, rank)
	}
	old := r.Entries[i]
	e := old
	e.Verified = verified
	if p := strings.TrimPrefix(strings.TrimSpace(position), "XGID="); p != "" {
		if _, err := ParsePosition(p); err != nil {
			return fmt.Errorf("%w: rank %d: %w", ErrInvalid, rank, err)
		}
		e.Position = p
		if r.Version >= 2 && e.Context == ContextDefault && (r.Document.Defaults == nil || !r.Document.Defaults.Covers(positionContext(p))) {
			e.Context = ContextToVerify
		}
	}
	r.Entries[i] = e
	if err := r.Validate(); err != nil {
		r.Entries[i] = old
		return err
	}
	return nil
}

// SidecarSuffix ends the name of the Relevé kept beside its document: the
// verification of a book spans several sittings, and its progress lives
// with the book rather than in a database (ADR-0117, no image kept).
const SidecarSuffix = ".releve.json"

// SidecarPath is where the Relevé of the document at path is kept.
func SidecarPath(path string) string {
	return filepath.Clean(path) + SidecarSuffix
}

// SourceOf is the document a Relevé file kept beside it was read from, or
// "" when the file is not named as a sidecar or the document is gone.
func SourceOf(relevePath string) string {
	if !strings.HasSuffix(relevePath, SidecarSuffix) {
		return ""
	}
	src := strings.TrimSuffix(relevePath, SidecarSuffix)
	if _, err := os.Stat(src); err != nil {
		return ""
	}
	return src
}

// LoadSidecar reads the Relevé kept beside the document at path. It returns
// nil, and no error, when there is none or when it was read from another
// version of the file (its SHA-256 differs): the document is then read anew.
func LoadSidecar(path string) (*Releve, error) {
	data, err := os.ReadFile(SidecarPath(path))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r, err := Decode(data)
	if err != nil {
		return nil, err
	}
	sum, err := SourceSHA256(path)
	if err != nil {
		return nil, err
	}
	if r.Document.SHA256 != "" && r.Document.SHA256 != sum {
		return nil, nil
	}
	return r, nil
}

// WriteFile writes the Relevé to path whole or not at all: an interrupted
// write never leaves a half Relevé where the progress of a book is kept.
func WriteFile(path string, r *Releve) error {
	data, err := r.Encode()
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".releve-*.json")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	return nil
}

// SourceSHA256 is the fingerprint AnalyzeFile gives the document at path:
// the file's SHA-256, or that of a folder's page images in their order.
func SourceSHA256(path string) (string, error) {
	paths, err := sourcePages(path)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			return "", err
		}
		_, err = io.Copy(h, f)
		_ = f.Close()
		if err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// sourcePages lists the files of a document as AnalyzeFile reads them: a
// folder's page images sorted, else the file itself.
func sourcePages(path string) ([]string, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !st.IsDir() {
		return []string{path}, nil
	}
	ents, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, e := range ents {
		if e.Type().IsRegular() && IsPageImage(e.Name()) {
			paths = append(paths, filepath.Join(path, e.Name()))
		}
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrNoImage, path)
	}
	sort.Strings(paths)
	return paths, nil
}

// ErrNoPageImage is the answer for a diagram drawn with a board font: its
// page has no image to crop, and its reading is exact.
var ErrNoPageImage = errors.New("document: the page has no image to crop")

// pageCache keeps the last page decoded, and the PDF it came from: a sheet
// goes from one diagram to the next, mostly on the same page, and a scanned
// book takes seconds to open. Nothing of it is written anywhere.
var pageCache struct {
	sync.Mutex
	path  string
	mod   int64
	pdf   *pdf.Reader
	page  int
	image image.Image
}

// DiagramImage crops the diagram at rect on the given page (from 1) of the
// document at path — a PDF of scanned pages, a page image or a folder of
// them — and returns it as a PNG at most maxWidth pixels wide, redrawn from
// the file each time: no image is kept with the Relevé.
func DiagramImage(ctx context.Context, path string, page int, rect Rect, maxWidth int) ([]byte, error) {
	if rect.W <= 0 || rect.H <= 0 || rect.X < 0 || rect.Y < 0 || rect.X+rect.W > 1.0001 || rect.Y+rect.H > 1.0001 {
		return nil, fmt.Errorf("%w: rect %+v", ErrInvalid, rect)
	}
	img, err := pageImage(ctx, path, page)
	if err != nil {
		return nil, err
	}
	b := img.Bounds()
	// A margin shows the print around the diagram: the pips, the cube, the
	// label that a reading may have missed.
	const margin = 0.04
	x0 := b.Min.X + int((rect.X-margin*rect.W)*float64(b.Dx()))
	y0 := b.Min.Y + int((rect.Y-margin*rect.H)*float64(b.Dy()))
	x1 := b.Min.X + int((rect.X+rect.W*(1+margin))*float64(b.Dx())+0.5)
	y1 := b.Min.Y + int((rect.Y+rect.H*(1+margin))*float64(b.Dy())+0.5)
	src := image.Rect(x0, y0, x1, y1).Intersect(b)
	if src.Empty() {
		return nil, fmt.Errorf("%w: rect %+v outside the page", ErrInvalid, rect)
	}
	w, h := src.Dx(), src.Dy()
	if maxWidth > 0 && w > maxWidth {
		h = h * maxWidth / w
		w = maxWidth
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, max(h, 1)))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, src, draw.Src, nil)
	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func pageImage(ctx context.Context, path string, page int) (image.Image, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	pageCache.Lock()
	defer pageCache.Unlock()
	if pageCache.path != path || pageCache.mod != st.ModTime().UnixNano() {
		pageCache.path, pageCache.mod, pageCache.pdf, pageCache.page, pageCache.image = path, st.ModTime().UnixNano(), nil, 0, nil
	}
	if pageCache.page == page && pageCache.image != nil {
		return pageCache.image, nil
	}
	img, err := decodePage(ctx, path, st.IsDir(), page)
	if err != nil {
		return nil, err
	}
	pageCache.page, pageCache.image = page, img
	return img, nil
}

// decodePage decodes one page as the analysis read it; the caller holds
// pageCache.
func decodePage(ctx context.Context, path string, dir bool, page int) (image.Image, error) {
	if page < 1 {
		return nil, fmt.Errorf("%w: page %d", ErrInvalid, page)
	}
	if dir || IsPageImage(path) {
		paths, err := sourcePages(path)
		if err != nil {
			return nil, err
		}
		if page > len(paths) {
			return nil, fmt.Errorf("%w: page %d of %d", ErrInvalid, page, len(paths))
		}
		f := &imageFiles{ctx: ctx, paths: paths[page-1 : page], jpx: jpx.Budget{Left: MaxJPXPixels}}
		p, err := f.Page(0)
		if err != nil {
			return nil, err
		}
		return p.Image, nil
	}
	if pageCache.pdf == nil {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		rd, err := pdf.Open(data, pdf.DefaultLimits)
		if err != nil {
			return nil, fmt.Errorf("document: %w", err)
		}
		pageCache.pdf = rd
	}
	rd := pageCache.pdf
	if page > rd.NumPages() {
		return nil, fmt.Errorf("%w: page %d of %d", ErrInvalid, page, rd.NumPages())
	}
	p, err := rd.Page(page)
	if err != nil {
		return nil, fmt.Errorf("document: %w", err)
	}
	sc, err := rd.ScanContext(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNoPageImage, err)
	}
	return sc.Image, nil
}
