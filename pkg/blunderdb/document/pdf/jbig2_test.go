package pdf_test

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/document/pdf"
	"github.com/kevung/blunderdb/pkg/blunderdb/document/pdf/pdftest"
)

// jbig2Halftone is a page of 8×8 pixels holding a halftone region, which the
// decoder refuses.
var jbig2Halftone = append(
	[]byte{0, 0, 0, 0, 48, 0, 1, 0, 0, 0, 19, 0, 0, 0, 8, 0, 0, 0, 8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	append([]byte{0, 0, 0, 1, 22, 0, 1, 0, 0, 0, 17, 0, 0, 0, 8, 0, 0, 0, 8}, make([]byte, 9)...)...)

// pbm reads a raw bitmap of the jbig2 package's fixtures, 1 for black.
func pbm(t *testing.T, name string) (w, h int, bits []byte) {
	b := readFile(t, "../../jbig2/testdata/"+name)
	if _, err := fmt.Sscanf(string(b), "P4\n%d %d\n", &w, &h); err != nil {
		t.Fatal(err)
	}
	n := bytes.IndexByte(b, '\n') + 1
	n += bytes.IndexByte(b[n:], '\n') + 1
	return w, h, b[n:]
}

// TestScanJBIG2: a JBIG2 image with its globals, as an image, a stencil
// mask and a soft mask, paints the page jbig2dec draws, black where JBIG2
// codes 1.
func TestScanJBIG2(t *testing.T) {
	w, h, want := pbm(t, "sym0.pbm")
	jb := pdftest.Image{Width: w, Height: h, BPC: 1, ColorSpace: "/DeviceGray", Filter: "/JBIG2Decode",
		Data: readFile(t, "../../jbig2/testdata/sym0.jb2"), Globals: readFile(t, "../../jbig2/testdata/sym.sym")}
	stencil := jb
	stencil.ColorSpace, stencil.ImageMask = "", true
	black := pdftest.Image{Width: 1, Height: 1, BPC: 8, ColorSpace: "/DeviceGray", Data: []byte{0}}
	// A soft mask opaque where JBIG2 codes 0, over a white page: the black
	// image shows where the bitmap is white, as in a scan whose layers
	// a mask separates.
	soft := black
	soft.SMask = &jb
	pw, ph := float64(w)*72/150, float64(h)*72/150
	content := fmt.Sprintf("%g 0 0 %g 0 0 cm /Im1 Do", pw, ph)
	for _, c := range []struct {
		name  string
		im    pdftest.Image
		paint byte // the grey where JBIG2 codes 1
		other byte
	}{
		{"an image", jb, 0, 255},
		{"a stencil mask", stencil, 0, 255},
		{"a soft mask", soft, 255, 0},
	} {
		d := pdftest.Doc{Pages: []pdftest.Page{{Width: pw, Height: ph, Content: content, Images: []pdftest.Image{c.im}}}}
		s, err := scanOf(t, d, 1)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if b := s.Image.Bounds(); b.Dx() != w || b.Dy() != h {
			t.Fatalf("%s: a page of %v", c.name, b)
		}
		bad := 0
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				v := c.other
				if want[y*((w+7)/8)+x/8]>>(7-x%8)&1 == 1 {
					v = c.paint
				}
				if at(s, x, y) != v {
					bad++
				}
			}
		}
		if bad != 0 {
			t.Errorf("%s: %d pixels differ", c.name, bad)
		}
	}
}

// TestScanJBIG2Refused: a refused JBIG2 image leaves its page unread and
// named, and the page's other images undecoded: the JPEG 2000 budget the
// first page would spend is still whole for the second.
func TestScanJBIG2Refused(t *testing.T) {
	jpx := pdftest.Image{Width: 320, Height: 40, BPC: 8, ColorSpace: "/DeviceGray", Filter: "/JPXDecode", Data: readFile(t, "fixture.jp2")}
	ht := pdftest.Image{Width: 8, Height: 8, BPC: 1, ColorSpace: "/DeviceGray", Filter: "/JBIG2Decode", Data: jbig2Halftone}
	draw := "153.6 0 0 19.2 0 0 cm /Im1 Do /Im2 Do"
	d := pdftest.Doc{Pages: []pdftest.Page{
		{Width: 153.6, Height: 19.2, Content: draw, Images: []pdftest.Image{jpx, ht}},
		{Width: 153.6, Height: 19.2, Content: "153.6 0 0 19.2 0 0 cm /Im1 Do", Images: []pdftest.Image{jpx}},
	}}
	lim := pdf.DefaultLimits
	lim.MaxJPXPixels = 320 * 40
	r, err := pdf.Open(d.Bytes(), lim)
	if err != nil {
		t.Fatal(err)
	}
	p1, _ := r.Page(1)
	if _, err := r.Scan(p1); !errors.Is(err, pdf.ErrImageCodec) || !strings.Contains(err.Error(), "halftone") {
		t.Errorf("page 1: %v", err)
	}
	p2, _ := r.Page(2)
	if _, err := r.Scan(p2); err != nil {
		t.Errorf("page 2, its JPEG 2000 budget spent by page 1: %v", err)
	}
}
