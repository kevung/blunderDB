package document

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

// The written documents of these tests are made here from page images of
// the package's own fixtures, never taken from a book: each format must give
// back the reading of its pictures read as page images.

var writtenPNGs = []string{"testdata/sans-pips/p001.png", "testdata/sans-pips/p002.png"}

func pngData(t *testing.T) [][]byte {
	t.Helper()
	var out [][]byte
	for _, p := range writtenPNGs {
		d, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, d)
	}
	return out
}

func writeZip(t *testing.T, path string, parts [][2]string) {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, p := range parts {
		method := zip.Deflate
		if p[0] == "mimetype" {
			method = zip.Store
		}
		w, err := zw.CreateHeader(&zip.FileHeader{Name: p[0], Method: method})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(p[1])); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func makeDOCX(t *testing.T, dir string, imgs [][]byte) string {
	var body, rels strings.Builder
	body.WriteString(`<w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:t>Chapter One</w:t></w:r></w:p>`)
	parts := [][2]string{{"[Content_Types].xml", `<?xml version="1.0"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"/>`}}
	for i, d := range imgs {
		// The picture as Word writes it: a DrawingML blip, with the VML
		// fallback older readers take, which must not count twice.
		fmt.Fprintf(&body, `<w:p><w:r><mc:AlternateContent><mc:Choice Requires="wps"><w:drawing><a:graphic><a:graphicData><pic:pic><pic:blipFill><a:blip r:embed="rId%d"/></pic:blipFill></pic:pic></a:graphicData></a:graphic></w:drawing></mc:Choice><mc:Fallback><w:pict><v:imagedata r:id="rId%d"/></w:pict></mc:Fallback></mc:AlternateContent></w:r></w:p><w:p><w:r><w:t>Diagram follows the text.</w:t></w:r></w:p>`, i+1, i+1)
		fmt.Fprintf(&rels, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/image%d.png"/>`, i+1, i+1)
		parts = append(parts, [2]string{fmt.Sprintf("word/media/image%d.png", i+1), string(d)})
	}
	rels.WriteString(`<Relationship Id="rIdX" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="http://example.com/x.png" TargetMode="External"/>`)
	parts = append(parts,
		[2]string{"word/document.xml", `<?xml version="1.0"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:mc="http://schemas.openxmlformats.org/markup-compatibility/2006" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture" xmlns:v="urn:schemas-microsoft-com:vml"><w:body>` + body.String() + `</w:body></w:document>`},
		[2]string{"word/_rels/document.xml.rels", `<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` + rels.String() + `</Relationships>`},
		[2]string{"docProps/core.xml", `<?xml version="1.0"?><cp:coreProperties xmlns:cp="x" xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Docx Title</dc:title><dc:creator>Some Author</dc:creator></cp:coreProperties>`})
	p := filepath.Join(dir, "book.docx")
	writeZip(t, p, parts)
	return p
}

func makeODT(t *testing.T, dir string, imgs [][]byte) string {
	var body strings.Builder
	body.WriteString(`<text:h text:outline-level="1">Chapter One</text:h>`)
	parts := [][2]string{{"mimetype", "application/vnd.oasis.opendocument.text"}}
	for i, d := range imgs {
		// One picture under two formats: the SVG is passed over for its
		// bitmap replacement.
		fmt.Fprintf(&body, `<text:p><draw:frame><draw:image xlink:href="Pictures/p%d.svg"/><draw:image xlink:href="Pictures/p%d.png"/></draw:frame></text:p><text:p>Diagram follows<text:s/>the text.</text:p>`, i+1, i+1)
		parts = append(parts, [2]string{fmt.Sprintf("Pictures/p%d.png", i+1), string(d)},
			[2]string{fmt.Sprintf("Pictures/p%d.svg", i+1), `<svg xmlns="http://www.w3.org/2000/svg"/>`})
	}
	parts = append(parts,
		[2]string{"content.xml", `<?xml version="1.0"?><office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0" xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0" xmlns:draw="urn:oasis:names:tc:opendocument:xmlns:drawing:1.0" xmlns:xlink="http://www.w3.org/1999/xlink"><office:body><office:text>` + body.String() + `</office:text></office:body></office:document-content>`},
		[2]string{"meta.xml", `<?xml version="1.0"?><office:document-meta xmlns:office="o" xmlns:dc="http://purl.org/dc/elements/1.1/"><office:meta><dc:title>Odt Title</dc:title></office:meta></office:document-meta>`})
	p := filepath.Join(dir, "book.odt")
	writeZip(t, p, parts)
	return p
}

func makeEPUB(t *testing.T, dir string, imgs [][]byte) string {
	var body strings.Builder
	body.WriteString(`<h1>Chapter One</h1>`)
	parts := [][2]string{{"mimetype", "application/epub+zip"},
		{"META-INF/container.xml", `<?xml version="1.0"?><container xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>`}}
	for i, d := range imgs {
		fmt.Fprintf(&body, `<figure><img src="../images/p%%2D%d.png" alt=""/><figcaption>Diagram follows the text.</figcaption></figure>`, i+1)
		parts = append(parts, [2]string{fmt.Sprintf("OEBPS/images/p-%d.png", i+1), string(d)})
	}
	parts = append(parts,
		[2]string{"OEBPS/content.opf", `<?xml version="1.0"?><package xmlns="http://www.idpf.org/2007/opf" xmlns:dc="http://purl.org/dc/elements/1.1/"><metadata><dc:title>Epub Title</dc:title></metadata><manifest><item id="c1" href="text/ch1.xhtml" media-type="application/xhtml+xml"/></manifest><spine><itemref idref="c1"/></spine></package>`},
		[2]string{"OEBPS/text/ch1.xhtml", `<?xml version="1.0"?><html xmlns="http://www.w3.org/1999/xhtml"><head><title>c</title></head><body>` + body.String() + `</body></html>`})
	p := filepath.Join(dir, "book.epub")
	writeZip(t, p, parts)
	return p
}

func makeHTML(t *testing.T, dir string, imgs [][]byte) string {
	page := filepath.Join(dir, "page")
	if err := os.MkdirAll(filepath.Join(page, "img"), 0o755); err != nil {
		t.Fatal(err)
	}
	var body strings.Builder
	body.WriteString(`<html><head><title>Html Title</title></head><body><h2>Chapter One</h2>`)
	// A picture on the network and one outside the page's folder are
	// never read.
	body.WriteString(`<p><img src="https://example.com/board.png"><img src="../outside.png"></p>`)
	for i, d := range imgs {
		name := fmt.Sprintf("p%d.png", i+1)
		if err := os.WriteFile(filepath.Join(page, "img", name), d, 0o644); err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&body, `<p><img src="img/%s"></p><p>Diagram follows the text.</p>`, name)
	}
	if err := os.WriteFile(filepath.Join(dir, "outside.png"), imgs[0], 0o644); err != nil {
		t.Fatal(err)
	}
	body.WriteString(`</body></html>`)
	p := filepath.Join(page, "index.html")
	if err := os.WriteFile(p, []byte(body.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// pagesReading is the reading of the pictures as a folder of page images,
// done once.
var pagesReading = func() func(*testing.T) *Releve {
	var once sync.Once
	var r *Releve
	var err error
	return func(t *testing.T) *Releve {
		t.Helper()
		once.Do(func() {
			dir, e := os.MkdirTemp("", "written")
			if e != nil {
				err = e
				return
			}
			defer func() { _ = os.RemoveAll(dir) }()
			for i, p := range writtenPNGs {
				d, e := os.ReadFile(p)
				if e != nil {
					err = e
					return
				}
				if e := os.WriteFile(filepath.Join(dir, fmt.Sprintf("p%d.png", i+1)), d, 0o644); e != nil {
					err = e
					return
				}
			}
			r, err = AnalyzeFile(dir, AnalyzeOptions{Title: "pages"})
		})
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
}()

func TestWrittenDocumentsReadAsTheirPictures(t *testing.T) {
	want := pagesReading(t)
	if len(want.Entries) == 0 {
		t.Fatal("the pictures hold no diagram")
	}
	imgs := pngData(t)
	for _, f := range []struct {
		name, title string
		make        func(*testing.T, string, [][]byte) string
	}{
		{"docx", "Docx Title", makeDOCX}, {"odt", "Odt Title", makeODT},
		{"epub", "Epub Title", makeEPUB}, {"html", "Html Title", makeHTML},
	} {
		t.Run(f.name, func(t *testing.T) {
			path := f.make(t, t.TempDir(), imgs)
			var warns []string
			r, err := AnalyzeFile(path, AnalyzeOptions{Warn: func(s string) { warns = append(warns, s) }})
			if err != nil {
				t.Fatal(err)
			}
			if r.Document.Title != f.title {
				t.Errorf("title %q, want %q", r.Document.Title, f.title)
			}
			if len(r.Entries) != len(want.Entries) {
				t.Fatalf("%d entries, want %d", len(r.Entries), len(want.Entries))
			}
			if ref := r.ReferenceOf(r.Entries[0]); !r.Document.Pictures || ref != fmt.Sprintf("Chapter One, fig. %d", r.Entries[0].Page) {
				t.Errorf("entry cited %q, want its picture's rank under its chapter", ref)
			}
			for i, e := range r.Entries {
				w := want.Entries[i]
				if e.Chapter != "Chapter One" {
					t.Errorf("entry %d: chapter %q", e.Rank, e.Chapter)
				}
				e.Chapter = ""
				if !reflect.DeepEqual(e, w) {
					t.Errorf("entry %d reads differently:\n%+v\n%+v", e.Rank, e, w)
				}
			}
			sum, err := SourceSHA256(path)
			if err != nil || sum != r.Document.SHA256 {
				t.Errorf("source fingerprint %q (%v), relevé %q", sum, err, r.Document.SHA256)
			}
			if f.name == "html" && (len(warns) != 1 || !strings.Contains(warns[0], "1 pictures on the network")) {
				t.Errorf("warnings %q, want the picture on the network named", warns)
			}
			e := r.Entries[0]
			png, err := DiagramImage(t.Context(), path, e.Page, *e.Rect, 200)
			if err != nil || !bytes.HasPrefix(png, []byte("\x89PNG")) {
				t.Errorf("diagram image: %v", err)
			}
		})
	}
}

// A picture drawn with vectors is named, never guessed at, and a document
// holding only such pictures says so in its refusal.
func TestWrittenVectorPictureNamed(t *testing.T) {
	emf := make([]byte, 128)
	emf[0] = 1
	copy(emf[40:], " EMF")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "d.emf"), emf, 0o644); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "doc.html")
	if err := os.WriteFile(p, []byte(`<p><img src="d.emf"></p>`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := AnalyzeFile(p, AnalyzeOptions{})
	if err == nil || !strings.Contains(err.Error(), "EMF picture") {
		t.Fatalf("error %v, want the EMF picture named", err)
	}
}

func TestWrittenLegacyWordRefused(t *testing.T) {
	p := filepath.Join(t.TempDir(), "old.doc")
	if err := os.WriteFile(p, append([]byte("\xD0\xCF\x11\xE0\xA1\xB1\x1A\xE1"), make([]byte, 512)...), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := AnalyzeFile(p, AnalyzeOptions{}); !errors.Is(err, ErrLegacyWord) {
		t.Fatalf("error %v, want ErrLegacyWord", err)
	}
	if !strings.Contains(ErrLegacyWord.Error(), "PDF or DOCX") {
		t.Fatal("the refusal does not say what to save the file as")
	}
}

// A small archive that would unpack without end is refused: too many
// entries, or a part unpacking past its bound.
func TestWrittenZipBombRefused(t *testing.T) {
	dir := t.TempDir()
	many := [][2]string{{"word/document.xml", "<w:document/>"}}
	for i := range MaxDocumentEntries + 1 {
		many = append(many, [2]string{fmt.Sprintf("x/%d", i), ""})
	}
	p := filepath.Join(dir, "many.docx")
	writeZip(t, p, many)
	if _, err := AnalyzeFile(p, AnalyzeOptions{}); !errors.Is(err, ErrDocumentTooLarge) {
		t.Errorf("many entries: %v, want ErrDocumentTooLarge", err)
	}
	big := filepath.Join(dir, "big.docx")
	writeZip(t, big, [][2]string{{"word/document.xml", "<w:document>" + strings.Repeat(" ", MaxDocumentPart+1) + "</w:document>"}})
	if st, err := os.Stat(big); err != nil || st.Size() > 1<<20 {
		t.Fatalf("the bomb is not small: %v", err)
	}
	if _, err := AnalyzeFile(big, AnalyzeOptions{}); !errors.Is(err, ErrDocumentTooLarge) {
		t.Errorf("part unpacking past its bound: %v, want ErrDocumentTooLarge", err)
	}
}

// The text beside a picture is laid out in a column at its right, beyond
// the reach of untagged pips, the text before above the text after, and
// only when it holds a figure.
func TestTextBeside(t *testing.T) {
	if ws := textBeside(1300, 1000, "Black to play", "", "a fine move"); ws != nil {
		t.Errorf("text without a figure laid out: %+v", ws)
	}
	ws := textBeside(1300, 1000, "before 12", "", "Black to play 53")
	if len(ws) != 6 {
		t.Fatalf("%d words, want 6", len(ws))
	}
	for _, w := range ws {
		if cx := (w.X0 + w.X1) / 2; cx < 1300+3.5*100 || cx > 1300+8*100 {
			t.Errorf("%q laid out at x %.0f, out of the column", w.Text, cx)
		}
	}
	if ws[1].Y0 >= ws[2].Y0 {
		t.Errorf("the text before is not above the text after: %+v", ws)
	}
}

// The pips the text beside a picture prints confirm its checkers, as a
// PDF's text layer does: a diagram read without printed pips is sure once
// its caption gives both counts.
func TestWrittenCaptionPipsConfirm(t *testing.T) {
	want := pagesReading(t)
	e := want.Entries[0]
	if e.Checkers == Sure {
		t.Fatal("the picture already prints its pips")
	}
	dir := t.TempDir()
	crop, err := DiagramImage(t.Context(), writtenPNGs[e.Page-1], 1, *e.Rect, 0)
	if err != nil {
		t.Fatal(err)
	}
	pos, err := ParsePosition(e.Position)
	if err != nil {
		t.Fatal(err)
	}
	bottom, top := pos.ComputePipCounts()
	if err := os.WriteFile(filepath.Join(dir, "d.png"), crop, 0o644); err != nil {
		t.Fatal(err)
	}
	read := func(upper, lower int) Entry {
		p := filepath.Join(dir, fmt.Sprintf("d-%d-%d.html", upper, lower))
		page := fmt.Sprintf(`<p>Pips %d</p><p><img src="d.png"></p><p>Pips %d</p>`, upper, lower)
		if err := os.WriteFile(p, []byte(page), 0o644); err != nil {
			t.Fatal(err)
		}
		r, err := AnalyzeFile(p, AnalyzeOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if len(r.Entries) != 1 {
			t.Fatalf("%d entries, want 1", len(r.Entries))
		}
		return r.Entries[0]
	}
	if got := read(top, bottom); got.Checkers != Sure || got.Position != e.Position {
		t.Errorf("pips %d and %d in the text: %s %s, want sure %s", top, bottom, got.Checkers, got.Position, e.Position)
	}
	if got := read(top+1, bottom); got.Checkers == Sure {
		t.Errorf("pips that disagree confirm the checkers: %+v", got)
	}
}
