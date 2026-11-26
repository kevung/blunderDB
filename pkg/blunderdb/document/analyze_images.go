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
	"math/bits"
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
// carries no text layer: the printed pip counts that confirm an entry are
// read in the image itself, and an entry without them is not sure.
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
		Profile: structureProfile(res.Profile), Defaults: structureDefaults(res.Profile),
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
	// The reader certifies the checkers by the printed pips and the context
	// field by field (structure.Context): a context read whole is "read";
	// what it does not read takes the document's defaults, which a person
	// confirms once; a cube, dice or score printed and not certified leave
	// the context to verify even then.
	e := Entry{Rank: rank, Page: d.Page + 1, Checkers: ToVerify, Context: ContextDefault}
	if !d.Context.Read {
		e.Doubts = []capture.Doubt{ContextDoubt}
	}
	if d.CheckersSure {
		e.Checkers = Sure
		// The other doubts of a reading whose checkers the pips confirm are
		// leads for checking checkers: only the drawn context stays.
		for _, x := range d.Doubts {
			if contextField[x.Field] {
				e.Doubts = append(e.Doubts, x)
			}
		}
	} else {
		e.Doubts = append(e.Doubts, d.Doubts...)
	}
	// A context read in part and outside the defaults (an owned cube, a
	// match score): the position says what was read, which the defaults do
	// not cover, so it is left to verify.
	tail := contextXGID(d.Context)
	if !d.Context.Read && tail != StructureDefaults.Context && !drawnContext(e.Doubts) {
		e.Doubts = append(e.Doubts, capture.Doubt{Field: "context", Reason: "context read in part (" + tail + "), outside the document's defaults: the rest is not read"})
	}
	switch {
	case d.Context.Read:
		e.Context = ContextRead
	case drawnContext(e.Doubts):
		e.Context = ContextToVerify
	}
	if d.State == structure.Uncertain {
		e.Checkers = Uncertain
	}
	if d.Board != "" {
		xgid := "XGID=" + d.Board + ":" + tail
		if _, err := ParsePosition(xgid); err == nil {
			e.Position = xgid
		} else {
			e.Doubts = append(e.Doubts, doubtPosition(err))
		}
	}
	if e.Position == "" {
		e.Checkers = Uncertain
		if d.Board == "" {
			e.Doubts = append(e.Doubts, doubtPosition(errors.New("the reading is not a position")))
		}
	}
	// Fifteen checkers a side with no doubt about them and no pip count
	// printed to confirm them: the intermediate state an import accepts as a
	// whole on request. Pips printed and disagreeing never make it.
	e.Complete = d.Complete && e.Checkers == ToVerify && e.Position != "" && hasDoubt(e.Doubts, NoPipsDoubt) && onlyCompleteLeads(e.Doubts)
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
	if p.PipsInImage {
		out.Pips = strings.TrimPrefix(out.Pips+"; also read in the image", "; also ")
	}
	if c := p.Numbering; c != "" {
		row := map[byte]string{'B': "bottom", 'T': "top"}[c[0]]
		side := map[byte]string{'L': "left", 'R': "right"}[c[1]]
		out.Numbering = "point 1 printed at the " + row + " " + side
	}
	if !p.DirectionVoted {
		out.Numbering = strings.TrimSpace(out.Numbering + " " + assumedDirectionMark)
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

// contextXGID is the XGID after its board: the context read, the document's
// defaults (StructureDefaults) for every field not read.
func contextXGID(c structure.Context) string {
	cube, owner, turn, dice := 0, c.CubeOwner, 1, "00"
	if c.Cube > 1 {
		cube = bits.Len(uint(c.Cube)) - 1
	}
	if c.Turn != 0 {
		turn = c.Turn
	}
	if c.Dice[0] > 0 && c.Dice[1] > 0 {
		dice = fmt.Sprintf("%d%d", c.Dice[0], c.Dice[1])
	}
	s1, s2, crawford, length := 0, 0, 0, 0
	if s := c.Score; s != nil {
		s1, s2, length = s.Bottom, s.Top, s.Length
		if s.Crawford {
			crawford = 1
		}
	}
	return fmt.Sprintf("%d:%d:%d:%s:%d:%d:%d:%d:10", cube, owner, turn, dice, s1, s2, crawford, length)
}

// ContextDoubt is the doubt every entry read by its structure carries
// unless its context is read whole: its checkers may be confirmed, its cube,
// dice, player on roll and score are not all read. Its field and reason never change, so a Relevé is searched by
// them.
var ContextDoubt = capture.Doubt{Field: "context", Reason: "context not read: cube, dice, player on roll, score"}

// NoPipsDoubt and AssumedDirectionDoubt are the only doubts beside the
// context a complete entry carries: no pip count printed to cross-check the
// checkers, and the document's direction of play taken from the fewer pips
// for want of one. A pip count printed and disagreeing is a doubt on the
// checkers that no complete entry has.
var (
	NoPipsDoubt           = capture.Doubt{Field: "pips", Reason: structure.NoPipsReason}
	AssumedDirectionDoubt = capture.Doubt{Field: "orientation", Reason: structure.AssumedDirectionReason}
)

// completeLead reports a doubt a complete entry may carry beside its
// context.
func completeLead(d capture.Doubt) bool {
	return contextField[d.Field] || d == NoPipsDoubt || d == AssumedDirectionDoubt
}

// contextField names the doubts about what surrounds the checkers.
var contextField = map[string]bool{"context": true, "cube": true, "dice": true, "score": true, "turn": true}

// structureDefaults is a fresh, unconfirmed copy of StructureDefaults, with
// the direction of play the reader assumed when no printed pips voted for
// one.
func structureDefaults(p structure.Profile) *Defaults {
	d := StructureDefaults
	if !p.DirectionVoted {
		d.Direction = assumedDirection(structureProfile(p).Home)
	}
	return &d
}

// assumedDirection is the unconfirmed default of a direction of play taken
// from the fewer pips, the bottom player's home on the side home.
func assumedDirection(home string) *Direction {
	return &Direction{Home: home, Description: "direction of play assumed, no printed pips confirm it: checkers move toward the side with the fewer pips, the bottom player's home on the " + home}
}

func onlyCompleteLeads(ds []capture.Doubt) bool {
	for _, d := range ds {
		if !completeLead(d) {
			return false
		}
	}
	return true
}

func doubtPosition(err error) capture.Doubt {
	return capture.Doubt{Field: "position", Reason: err.Error()}
}
