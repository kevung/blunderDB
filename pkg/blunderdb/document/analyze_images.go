package document

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg" // pages scanned as JPEG
	_ "image/png"  // pages rendered as PNG
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/capture"
	"github.com/kevung/blunderdb/pkg/blunderdb/document/structure"
)

// MaxImageFileSize bounds one page image file, read before it is decoded.
const MaxImageFileSize = 64 << 20

// ErrNoImage refuses a folder holding no page image.
var ErrNoImage = errors.New("document: no PNG or JPEG page image in the folder")

// IsPageImage reports whether a file name is a page image the reader takes.
func IsPageImage(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".png", ".jpg", ".jpeg":
		return true
	}
	return false
}

// AnalyzeFile reads the document at path: a folder of page images (PNG or
// JPEG, in the lexicographic order of their names), one page image, or a
// PDF (AnalyzePDFFile). It writes nothing.
func AnalyzeFile(path string, opts AnalyzeOptions) (*Releve, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if opts.FileName == "" {
		opts.FileName = path
	}
	if st.IsDir() {
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
		// A folder is named without an extension to drop.
		if opts.Title == "" {
			opts.Title = filepath.Base(filepath.Clean(path))
		}
		return AnalyzeImages(paths, opts)
	}
	if IsPageImage(path) {
		return AnalyzeImages([]string{path}, opts)
	}
	return AnalyzePDFFile(path, opts)
}

// AnalyzeImages reads the diagrams of page images, one file a page in the
// given order, with the structure reader (package structure), and returns
// the Relevé. It writes nothing and keeps no image: each page is decoded when
// read, twice (the reader learns the document's style first). A page image
// carries no text layer, so no printed pip count is read from it and no
// entry is sure; each one says why.
func AnalyzeImages(paths []string, opts AnalyzeOptions) (*Releve, error) {
	if len(paths) == 0 {
		return nil, ErrNoImage
	}
	if len(paths) > structure.MaxPages {
		return nil, fmt.Errorf("document: %d pages, at most %d", len(paths), structure.MaxPages)
	}
	h := sha256.New()
	for _, p := range paths {
		data, err := readBounded(p)
		if err != nil {
			return nil, err
		}
		h.Write(data)
	}
	title := strings.TrimSpace(opts.Title)
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(opts.FileName), filepath.Ext(opts.FileName))
	}
	if title == "" || title == "." {
		title = "Untitled document"
	}
	res, err := structure.Read(context.Background(), imageFiles(paths))
	if errors.Is(err, structure.ErrNoDiagram) {
		return nil, fmt.Errorf("document: no backgammon diagram found in the page images: %w", err)
	}
	if err != nil {
		return nil, fmt.Errorf("document: %w", err)
	}
	r := &Releve{Format: Format, Version: Version, Document: Source{
		Title: title, Author: strings.TrimSpace(opts.Author), SHA256: hex.EncodeToString(h.Sum(nil)),
		Profile: structureProfile(res.Profile),
	}}
	for _, d := range res.Diagrams {
		r.Entries = append(r.Entries, structureEntry(d, len(r.Entries)+1))
	}
	if len(r.Entries) == 0 {
		return nil, fmt.Errorf("document: no backgammon diagram found in the page images: %w", structure.ErrNoDiagram)
	}
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return r, nil
}

func structureEntry(d structure.Diagram, rank int) Entry {
	e := Entry{Rank: rank, Page: d.Page + 1, Doubts: d.Doubts}
	switch d.State {
	case structure.Sure:
		e.State = Sure
	case structure.Uncertain:
		e.State = Uncertain
	default:
		e.State = ToVerify
	}
	if d.Board != "" {
		// The cube, the dice, the player on roll and the score are not read:
		// a centred cube, no dice, the bottom player on roll, money play.
		xgid := "XGID=" + d.Board + ":0:0:1:00:0:0:0:0:10"
		if _, err := ParsePosition(xgid); err == nil {
			e.Position = xgid
		} else {
			e.Doubts = append(e.Doubts, doubtPosition(err))
		}
	}
	if e.Position == "" {
		e.State = Uncertain
		if d.Board == "" {
			e.Doubts = append(e.Doubts, doubtPosition(errors.New("the reading is not a position")))
		}
	}
	if pw, ph := d.PageSize[0], d.PageSize[1]; pw > 0 && ph > 0 {
		rc := d.Rect.Intersect(image.Rect(0, 0, pw, ph))
		if !rc.Empty() {
			e.Rect = &Rect{X: float64(rc.Min.X) / float64(pw), Y: float64(rc.Min.Y) / float64(ph),
				W: float64(rc.Dx()) / float64(pw), H: float64(rc.Dy()) / float64(ph)}
		}
	}
	return e
}

func structureProfile(p structure.Profile) *Profile {
	out := &Profile{Home: "right", Bottom: "white"}
	if p.HomeSide == 'L' {
		out.Home = "left"
	}
	if p.BottomIsDark {
		out.Bottom = "black"
	}
	out.Pips = map[byte]string{
		'G': "left of the board", 'D': "right of the board", 'L': "after the word pips",
		'B': "in the numbering row, at the bar",
	}[p.PipPlace]
	if c := p.Numbering; c != "" {
		row := map[byte]string{'B': "bottom", 'T': "top"}[c[0]]
		side := map[byte]string{'L': "left", 'R': "right"}[c[1]]
		out.Numbering = "point 1 printed at the " + row + " " + side
	}
	if !p.DirectionVoted {
		out.Numbering = strings.TrimSpace(out.Numbering + " (direction of play assumed)")
	}
	return out
}

// imageFiles are page image files decoded on demand, bounded before decoding.
type imageFiles []string

func (f imageFiles) Len() int { return len(f) }

func (f imageFiles) Page(i int) (structure.Page, error) {
	data, err := readBounded(f[i])
	if err != nil {
		return structure.Page{}, err
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return structure.Page{}, fmt.Errorf("%s: %w", f[i], err)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > 40_000 || cfg.Height > 40_000 || cfg.Width*cfg.Height > structure.MaxPixels {
		return structure.Page{}, fmt.Errorf("%w: %s is %d×%d pixels", structure.ErrPage, f[i], cfg.Width, cfg.Height)
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return structure.Page{}, fmt.Errorf("%s: %w", f[i], err)
	}
	dpi := imageDPI(data)
	if dpi < 50 || dpi > 1200 {
		dpi = estimatedDPI(cfg.Width, cfg.Height)
	}
	return structure.Page{Image: img, DPI: dpi}, nil
}

func readBounded(path string) ([]byte, error) {
	fh, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = fh.Close() }()
	st, err := fh.Stat()
	if err != nil {
		return nil, err
	}
	if st.Size() > MaxImageFileSize {
		return nil, fmt.Errorf("%w: %s is larger than %d bytes", structure.ErrPage, path, MaxImageFileSize)
	}
	data := make([]byte, st.Size())
	if _, err := fh.ReadAt(data, 0); err != nil && st.Size() > 0 {
		return nil, err
	}
	return data, nil
}

// estimatedDPI is the resolution of a page whose file does not say it,
// taking its long side for eleven inches (a book page).
func estimatedDPI(w, h int) float64 {
	return min(max(float64(max(w, h))/11, 100), 600)
}

// imageDPI is the resolution a PNG (pHYs, in pixels per metre) or a JPEG
// (JFIF density in dots per inch or per centimetre) records; 0 if none.
func imageDPI(data []byte) float64 {
	if len(data) > 8 && bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")) {
		for i := 8; i+12 <= len(data); {
			n := int(binary.BigEndian.Uint32(data[i:]))
			typ := string(data[i+4 : i+8])
			if n < 0 || i+12+n > len(data) {
				return 0
			}
			if typ == "pHYs" && n == 9 && data[i+16] == 1 {
				return float64(binary.BigEndian.Uint32(data[i+8:])) * 0.0254
			}
			if typ == "IDAT" {
				return 0
			}
			i += 12 + n
		}
		return 0
	}
	if len(data) > 20 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF && data[3] == 0xE0 && string(data[6:11]) == "JFIF\x00" {
		unit, x := data[13], float64(binary.BigEndian.Uint16(data[14:]))
		switch unit {
		case 1:
			return x
		case 2:
			return x * 2.54
		}
	}
	return 0
}

func doubtPosition(err error) capture.Doubt {
	return capture.Doubt{Field: "position", Reason: err.Error()}
}
