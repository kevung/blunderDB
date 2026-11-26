package document

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/document/pdf"
	"github.com/kevung/blunderdb/pkg/blunderdb/document/pdf/pdftest"
	"github.com/kevung/blunderdb/pkg/blunderdb/document/structure"
)

// TestAnalyzeScannedPDF: a PDF without a board font goes to the structure
// reader; a page of vectors is passed over and named; a PDF of vectors
// only is refused.
func TestAnalyzeScannedPDF(t *testing.T) {
	blank := pdftest.Image{Width: 200, Height: 200, BPC: 8, ColorSpace: "/DeviceGray", Data: bytes.Repeat([]byte{255}, 200*200)}
	scan := pdftest.Page{Width: 96, Height: 96, Content: "96 0 0 96 0 0 cm /Im1 Do", Images: []pdftest.Image{blank}}
	vectors := pdftest.Page{Width: 96, Height: 96, Content: "0 0 m 96 96 l S"}
	_, err := AnalyzePDF(pdftest.Doc{Pages: []pdftest.Page{scan, vectors}}.Bytes(), AnalyzeOptions{})
	if !errors.Is(err, structure.ErrNoDiagram) {
		t.Errorf("a blank scan: %v, want ErrNoDiagram", err)
	}
	_, err = AnalyzePDF(pdftest.Doc{Pages: []pdftest.Page{vectors, vectors}}.Bytes(), AnalyzeOptions{})
	if !errors.Is(err, ErrNoGlyphDiagram) || !errors.Is(err, pdf.ErrNoPageImage) {
		t.Errorf("vectors only: %v", err)
	}
	s := unreadSummary([]structure.PageError{{Page: 1, Err: pdf.ErrNoPageImage}, {Page: 4, Err: pdf.ErrNoPageImage}, {Page: 6, Err: pdf.ErrImageCodec}})
	if !strings.HasPrefix(s, "3 pages not read; p. 2, 5: pdf: the page draws no image") || !strings.Contains(s, "p. 7: pdf: an image") {
		t.Errorf("summary %q", s)
	}
}

// TestInvisibleTextLayerIsNotABoardFont: an OCR layer (rendering mode 3) in a
// font that maps every code over a scan leaves the page to the scan reader.
func TestInvisibleTextLayerIsNotABoardFont(t *testing.T) {
	blank := pdftest.Image{Width: 200, Height: 200, BPC: 8, ColorSpace: "/DeviceGray", Data: bytes.Repeat([]byte{255}, 200*200)}
	ocr := pdftest.Font{BaseFont: "GlyphLessFont", Type0: true, First: 0x20, Last: 0xFF}
	content := "96 0 0 96 0 0 cm /Im1 Do\nBT 3 Tr /F1 12 Tf 10 50 Td " + pdftest.Hex("ñòóôð text") + " Tj ET"
	data := pdftest.Doc{Fonts: []pdftest.Font{ocr}, Pages: []pdftest.Page{{Width: 96, Height: 96, Content: content, Images: []pdftest.Image{blank}}}}.Bytes()
	_, err := AnalyzePDF(data, AnalyzeOptions{})
	if errors.Is(err, ErrNoGlyphDiagram) || !errors.Is(err, structure.ErrNoDiagram) {
		t.Errorf("invisible OCR layer: %v, want the scan reader's ErrNoDiagram", err)
	}
}
