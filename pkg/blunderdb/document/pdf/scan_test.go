package pdf_test

import (
	"bytes"
	"compress/zlib"
	"errors"
	"image"
	"image/jpeg"
	"os"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/document/pdf"
	"github.com/kevung/blunderdb/pkg/blunderdb/document/pdf/pdftest"
)

func flate(b []byte) []byte {
	var z bytes.Buffer
	w := zlib.NewWriter(&z)
	_, _ = w.Write(b)
	_ = w.Close()
	return z.Bytes()
}

// greyPage is a page of w×h grey pixels, white with a black square
// [x0,x1)×[y0,y1).
func greyPage(w, h, x0, y0, x1, y1 int) []byte {
	b := bytes.Repeat([]byte{255}, w*h)
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			b[y*w+x] = 0
		}
	}
	return b
}

func scanOf(t *testing.T, d pdftest.Doc, n int) (*pdf.Scan, error) {
	t.Helper()
	r, err := pdf.Open(d.Bytes(), pdf.DefaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	p, err := r.Page(n)
	if err != nil {
		t.Fatal(err)
	}
	return r.Scan(p)
}

func at(s *pdf.Scan, x, y int) uint8 { return s.Image.GrayAt(x, y).Y }

// TestScanFlate: a page made of one grey image is composed at the image's
// resolution, pixel for pixel; the invisible text gives its words in the
// image's pixels.
func TestScanFlate(t *testing.T) {
	img := pdftest.Image{Width: 1275, Height: 1650, BPC: 8, ColorSpace: "/DeviceGray", Filter: "/FlateDecode",
		Data: flate(greyPage(1275, 1650, 300, 400, 500, 450))}
	content := "q 612 0 0 792 0 0 cm /Im1 Do Q BT 3 Tr /F1 12 Tf 100 700 Td (137 pips) Tj ET"
	d := pdftest.Doc{Fonts: []pdftest.Font{{BaseFont: "Helvetica"}}, Pages: []pdftest.Page{{Width: 612, Height: 792, Content: content, Images: []pdftest.Image{img}}}}
	s, err := scanOf(t, d, 1)
	if err != nil {
		t.Fatal(err)
	}
	if s.DPI != 150 || s.Image.Rect.Dx() != 1275 || s.Image.Rect.Dy() != 1650 {
		t.Fatalf("scan at %v dpi, %v", s.DPI, s.Image.Rect)
	}
	if at(s, 400, 425) != 0 || at(s, 299, 425) != 255 || at(s, 500, 425) != 255 || at(s, 400, 399) != 255 || at(s, 300, 400) != 0 {
		t.Errorf("square not where drawn: %d %d %d %d", at(s, 400, 425), at(s, 299, 425), at(s, 500, 425), at(s, 400, 399))
	}
	if len(s.Words) != 2 || s.Words[0].Text != "137" || s.Words[1].Text != "pips" {
		t.Fatalf("words %+v", s.Words)
	}
	w := s.Words[0]
	// x = 100 pt, baseline at 700 pt: 208 and 192 pixels at 150 dpi.
	if w.X0 < 207 || w.X0 > 209 || w.Y1 < 196 || w.Y1 > 198 || w.Y0 > 172 || w.X1 <= w.X0 {
		t.Errorf("word box %+v", w)
	}
}

// TestScanResolution: a page is composed at its images' resolution, bounded
// to 150-300 dpi, and its Rotate turns it.
func TestScanResolution(t *testing.T) {
	big := pdftest.Image{Width: 2448, Height: 3168, BPC: 8, ColorSpace: "/DeviceGray", Filter: "/FlateDecode", Data: flate(greyPage(2448, 3168, 0, 0, 10, 10))}
	small := pdftest.Image{Width: 306, Height: 396, BPC: 8, ColorSpace: "/DeviceGray", Data: greyPage(306, 396, 0, 0, 1, 1)}
	page := func(im pdftest.Image, rot int) pdftest.Page {
		return pdftest.Page{Width: 612, Height: 792, Content: "612 0 0 792 0 0 cm /Im1 Do", Images: []pdftest.Image{im}, Rotate: rot}
	}
	d := pdftest.Doc{Pages: []pdftest.Page{page(big, 0), page(small, 0), page(big, 90)}}
	for i := 0; i < 3; i++ {
		s, err := scanOf(t, d, i+1)
		if err != nil {
			t.Fatal(err)
		}
		switch i {
		case 0:
			if s.DPI != 288 || s.Image.Rect.Dx() != 2448 || s.Image.Rect.Dy() != 3168 {
				t.Errorf("page 1 at %v dpi, %v", s.DPI, s.Image.Rect)
			}
			if at(s, 5, 5) != 0 || at(s, 25, 25) != 255 {
				t.Errorf("page 1 corner %d %d", at(s, 5, 5), at(s, 25, 25))
			}
		case 1:
			if s.DPI != 150 || s.Image.Rect.Dx() != 1275 || s.Image.Rect.Dy() != 1650 {
				t.Errorf("page 2 at %v dpi, %v", s.DPI, s.Image.Rect)
			}
		case 2:
			if s.Image.Rect.Dx() != 3168 || s.Image.Rect.Dy() != 2448 {
				t.Errorf("page 3 rotated: %v", s.Image.Rect)
			}
			// The image's top-left corner, turned a quarter clockwise, is
			// now at the top right.
			if at(s, 3168-5, 5) != 0 || at(s, 5, 5) != 255 {
				t.Errorf("page 3 corners %d %d", at(s, 3168-5, 5), at(s, 5, 5))
			}
		}
	}
}

// g4Bar is a 32×4 fax image (Group 4) of a black bar on columns 10 to 13:
// the first row in horizontal mode (white 10, black 4), the others in
// vertical mode, then the end of the block.
func g4Bar() []byte {
	bits := "001" + "00111" + "011" + "1" + "111" + "111" + "111" + "000000000001" + "000000000001"
	for len(bits)%8 != 0 {
		bits += "0"
	}
	out := make([]byte, len(bits)/8)
	for i, c := range bits {
		if c == '1' {
			out[i/8] |= 0x80 >> (i % 8)
		}
	}
	return out
}

// TestScanMasks: a JPEG background and a fax stencil painted in black over
// it, as scanners store a page; a soft mask blends.
func TestScanMasks(t *testing.T) {
	bg := image.NewGray(image.Rect(0, 0, 320, 40))
	for i := range bg.Pix {
		bg.Pix[i] = 200
	}
	var jb bytes.Buffer
	if err := jpeg.Encode(&jb, bg, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	back := pdftest.Image{Width: 320, Height: 40, BPC: 8, ColorSpace: "/DeviceGray", Filter: "/DCTDecode", Data: jb.Bytes()}
	mask := pdftest.Image{Width: 32, Height: 4, BPC: 1, ImageMask: true, Filter: "/CCITTFaxDecode", DecodeParms: "<</K -1 /Columns 32 /Rows 4>>", Data: g4Bar()}
	soft := pdftest.Image{Width: 2, Height: 2, BPC: 8, ColorSpace: "/DeviceGray", Data: []byte{0, 0, 0, 0},
		SMask: &pdftest.Image{Width: 2, Height: 2, BPC: 8, ColorSpace: "/DeviceGray", Data: []byte{128, 128, 128, 128}}}
	content := "q 153.6 0 0 19.2 0 0 cm /Im1 Do Q q 0 g 153.6 0 0 19.2 0 0 cm /Im2 Do Q q 20 0 0 5 0 30 cm /Im3 Do Q"
	d := pdftest.Doc{Pages: []pdftest.Page{{Width: 153.6, Height: 40, Content: content, Images: []pdftest.Image{back, mask, soft}}}}
	s, err := scanOf(t, d, 1)
	if err != nil {
		t.Fatal(err)
	}
	if s.DPI != 150 {
		t.Fatalf("dpi %v", s.DPI)
	}
	// The page is 320×83 pixels; the stencil's 32 columns span 10 pixels
	// each, its bar on pixels 100 to 140, over rows 63 to 83.
	y := 75
	if g := at(s, 120, y); g > 20 {
		t.Errorf("under the bar: %d", g)
	}
	if g := at(s, 60, y); g < 185 || g > 215 {
		t.Errorf("background: %d", g)
	}
	if g := at(s, 20, 15); g < 110 || g > 145 {
		t.Errorf("half-transparent black on white: %d", g)
	}
}

// TestScanRefusals: a page of vectors, a JBIG2 image, an inline image.
func TestScanRefusals(t *testing.T) {
	jb2 := pdftest.Image{Width: 8, Height: 8, BPC: 1, ColorSpace: "/DeviceGray", Filter: "/JBIG2Decode", Data: []byte{0, 1, 2}}
	inline := "q 72 0 0 72 0 0 cm BI /W 2 /H 1 /BPC 8 /CS /G ID \x00\x00 EI Q"
	d := pdftest.Doc{Pages: []pdftest.Page{
		{Width: 100, Height: 100, Content: "0 0 m 100 100 l S"},
		{Width: 100, Height: 100, Content: "100 0 0 100 0 0 cm /Im1 Do", Images: []pdftest.Image{jb2}},
		{Width: 72, Height: 72, Content: inline},
	}}
	if _, err := scanOf(t, d, 1); !errors.Is(err, pdf.ErrNoPageImage) {
		t.Errorf("vectors: %v", err)
	}
	if _, err := scanOf(t, d, 2); !errors.Is(err, pdf.ErrImageCodec) || !strings.Contains(err.Error(), "JBIG2") {
		t.Errorf("JBIG2: %v", err)
	}
	s, err := scanOf(t, d, 3)
	if err != nil {
		t.Fatal(err)
	}
	if at(s, 10, 10) != 0 || s.DPI != 150 {
		t.Errorf("inline image: %d at %v dpi", at(s, 10, 10), s.DPI)
	}
}

// TestScanJPX: a JPEG 2000 image reads as the same page as the lossless
// grey samples it was made from.
func TestScanJPX(t *testing.T) {
	jp2, err := os.ReadFile("testdata/fixture.jp2")
	if err != nil {
		t.Fatal(err)
	}
	im := pdftest.Image{Width: 320, Height: 40, BPC: 8, ColorSpace: "/DeviceGray", Filter: "/JPXDecode", Data: jp2}
	d := pdftest.Doc{Pages: []pdftest.Page{{Width: 153.6, Height: 19.2, Content: "153.6 0 0 19.2 0 0 cm /Im1 Do", Images: []pdftest.Image{im}}}}
	s, err := scanOf(t, d, 1)
	if err != nil {
		t.Fatal(err)
	}
	if g := at(s, 120, 30); g > 5 {
		t.Errorf("bar: %d", g)
	}
	if g := at(s, 10, 30); g < 40 || g > 60 {
		t.Errorf("gradient start: %d", g)
	}
	if g := at(s, 300, 5); g < 200 {
		t.Errorf("gradient end: %d", g)
	}
	bad := im
	bad.Data = []byte("not a JPEG 2000 stream")
	d.Pages[0].Images = []pdftest.Image{bad}
	if s, err := scanOf(t, d, 1); err != nil || len(s.Skipped) != 1 {
		t.Errorf("a broken JPEG 2000 stream is skipped and named: %v %v", err, s)
	}
}

// jpxWant is the grey image the JPEG 2000 fixtures were encoded from by a
// third-party encoder: a ramp with a black bar.
func jpxWant(x, y int) int {
	if x >= 100 && x < 140 && y >= 20 {
		return 0
	}
	return 40 + x*180/320
}

func jpxPage(t *testing.T, data []byte) (*pdf.Scan, error) {
	t.Helper()
	im := pdftest.Image{Width: 320, Height: 40, BPC: 8, ColorSpace: "/DeviceGray", Filter: "/JPXDecode", Data: data}
	d := pdftest.Doc{Pages: []pdftest.Page{{Width: 153.6, Height: 19.2, Content: "153.6 0 0 19.2 0 0 cm /Im1 Do", Images: []pdftest.Image{im}}}}
	return scanOf(t, d, 1)
}

// TestScanJPXOracle: the reversible 5/3 transform returns the encoder's
// input exactly, the irreversible 9/7 within its quantisation.
func TestScanJPXOracle(t *testing.T) {
	for _, c := range []struct {
		file string
		tol  float64
	}{{"fixture.jp2", 0}, {"lossy97.jp2", 8}} {
		data, err := os.ReadFile("testdata/" + c.file)
		if err != nil {
			t.Fatal(err)
		}
		s, err := jpxPage(t, data)
		if err != nil || len(s.Skipped) != 0 {
			t.Fatalf("%s: %v %v", c.file, err, s)
		}
		sum, worst := 0.0, 0
		for y := 0; y < 40; y++ {
			for x := 0; x < 320; x++ {
				e := int(at(s, x, y)) - jpxWant(x, y)
				if e < 0 {
					e = -e
				}
				sum += float64(e)
				worst = max(worst, e)
			}
		}
		if c.tol == 0 && worst != 0 {
			t.Errorf("%s: worst error %d", c.file, worst)
		}
		if mean := sum / (320 * 40); mean > c.tol {
			t.Errorf("%s: mean error %.2f", c.file, mean)
		}
	}
}

// TestScanJPXHostile: truncated and corrupted streams are skipped or read,
// never a panic.
func TestScanJPXHostile(t *testing.T) {
	data, err := os.ReadFile("testdata/lossy97.jp2")
	if err != nil {
		t.Fatal(err)
	}
	for n := 0; n < len(data); n += 7 {
		_, _ = jpxPage(t, data[:n])
	}
	for i := 0; i < len(data); i += 5 {
		b := append([]byte(nil), data...)
		b[i] ^= 0xff
		_, _ = jpxPage(t, b)
	}
}

func FuzzJPX(f *testing.F) {
	data, err := os.ReadFile("testdata/lossy97.jp2")
	if err != nil {
		f.Fatal(err)
	}
	f.Add(data)
	f.Fuzz(func(t *testing.T, b []byte) { _, _ = jpxPage(t, b) })
}
