package structure

import (
	"bytes"
	"compress/zlib"
	"context"
	"fmt"
	"image"
	"math/rand"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/document/pdf"
	"github.com/kevung/blunderdb/pkg/blunderdb/document/pdf/pdftest"
)

// scannedPages are the pages of a PDF composed by pdf.Scan.
type scannedPages struct{ r *pdf.Reader }

func (s scannedPages) Len() int { return s.r.NumPages() }

func (s scannedPages) Page(i int) (Page, error) {
	p, err := s.r.Page(i + 1)
	if err != nil {
		return Page{}, err
	}
	sc, err := s.r.Scan(p)
	if err != nil {
		return Page{Unread: err}, nil
	}
	var ws []Word
	for _, w := range sc.Words {
		ws = append(ws, Word{w.X0, w.Y0, w.X1, w.Y1, w.Text})
	}
	return Page{Image: sc.Image, DPI: sc.DPI, Words: ws}, nil
}

// scannedPDF stores each page as a grey image of the whole page at 150 dpi,
// its words as invisible text, as an OCR leaves a scan; a last page is
// drawn with vectors only.
func scannedPDF(src memPages) []byte {
	var pages []pdftest.Page
	for _, pg := range src {
		img := pg.Image.(*image.Gray)
		w, h := img.Rect.Dx(), img.Rect.Dy()
		var z bytes.Buffer
		zw := zlib.NewWriter(&z)
		_, _ = zw.Write(img.Pix)
		_ = zw.Close()
		wpt, hpt := float64(w)*72/150, float64(h)*72/150
		var c strings.Builder
		fmt.Fprintf(&c, "q %g 0 0 %g 0 0 cm /Im1 Do Q BT 3 Tr", wpt, hpt)
		for _, wd := range pg.Words {
			size := (wd.Y1 - wd.Y0) * 72 / 150
			fmt.Fprintf(&c, " /F1 %.2f Tf 1 0 0 1 %.2f %.2f Tm (%s) Tj", size, wd.X0*72/150, hpt-wd.Y1*72/150+0.2*size, wd.Text)
		}
		c.WriteString(" ET")
		pages = append(pages, pdftest.Page{Width: wpt, Height: hpt, Content: c.String(), Compress: true,
			Images: []pdftest.Image{{Width: w, Height: h, BPC: 8, ColorSpace: "/DeviceGray", Filter: "/FlateDecode", Data: z.Bytes()}}})
	}
	pages = append(pages, pdftest.Page{Width: 612, Height: 792, Content: "0 0 m 612 792 l S"})
	return pdftest.Doc{Title: "Scans", Fonts: []pdftest.Font{{BaseFont: "Helvetica"}}, Pages: pages}.Bytes()
}

// TestScannedPDF: a scanned book read from its PDF, its pips printed both
// in the image and in the invisible text: every diagram sure, the two
// sources agreeing; the page of vectors is passed over.
func TestScannedPDF(t *testing.T) {
	rd := rand.New(rand.NewSource(17))
	var sp []synthPage
	for i := 0; i < 8; i++ {
		o := drawOpts{p: 36, numbering: true, pips: true, inkPips: true}
		sp = append(sp, synthPage{pos: []synthPos{randomPos(rd), randomPos(rd)}, opts: []drawOpts{o, o}})
	}
	src, _ := book(t, sp, 2)
	r, err := pdf.Open(scannedPDF(src), pdf.DefaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Read(context.Background(), scannedPages{r})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Unread) != 1 || res.Unread[0].Page != len(src) {
		t.Errorf("unread %v", res.Unread)
	}
	sure := 0
	for _, d := range res.Diagrams {
		if d.State == Confirmed {
			sure++
		}
	}
	if len(res.Diagrams) != 16 || sure < 12 || !res.Profile.PipsInImage {
		t.Errorf("%d diagrams, %d sure, profile %+v", len(res.Diagrams), sure, res.Profile)
	}
	for i, d := range res.Diagrams {
		if d.TextPips == nil || d.ImagePips == nil || *d.TextPips != *d.ImagePips {
			t.Errorf("diagram %d: pips from the text %v, from the image %v", i+1, d.TextPips, d.ImagePips)
		}
	}
}
