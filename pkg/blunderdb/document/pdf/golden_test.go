package pdf_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"runtime"
	"slices"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/document/pdf"
	"github.com/kevung/blunderdb/pkg/blunderdb/document/pdf/pdftest"
)

// scanCase is a one-image page, the image filling it at dpi.
type scanCase struct {
	name string
	// digest is the SHA-256 of the composed page; empty when the bytes come
	// from a codec of the standard library, whose output is not ours to pin.
	digest string
	im     pdftest.Image
	dpi    float64
	// cm, when set, places the image on a page of its size at dpi.
	cm string
}

func mustRead(name string) []byte {
	b, err := os.ReadFile(name)
	if err != nil {
		panic(err)
	}
	return b
}

// texture is a deterministic w×h image of n channels, smooth enough for a
// codec and varied enough for every sample value to occur.
func texture(w, h, n int) []byte {
	b := make([]byte, w*h*n)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			for k := 0; k < n; k++ {
				b[(y*w+x)*n+k] = byte((x*(k+1) + y*(3-k) + (x*y)>>(6+k)) & 0xff)
			}
		}
	}
	return b
}

func scanCases() []scanCase {
	const fw, fh = 2480, 3508
	rgb := texture(1240, 1754, 3)
	idx := texture(1240, 1754, 1)
	palette := texture(256, 1, 3)
	colour := image.NewRGBA(image.Rect(0, 0, 1240, 1754))
	for i := 0; i < 1240*1754; i++ {
		copy(colour.Pix[4*i:], rgb[3*i:3*i+3])
		colour.Pix[4*i+3] = 255
	}
	var jb bytes.Buffer
	_ = jpeg.Encode(&jb, colour, &jpeg.Options{Quality: 90})
	fax := pdftest.Image{Width: fw, Height: fh, BPC: 1, ColorSpace: "/DeviceGray",
		Filter: "/CCITTFaxDecode", DecodeParms: fmt.Sprintf("<</K -1 /Columns %d /Rows %d>>", fw, fh), Data: mustRead("testdata/a4.g4")}
	stencil := fax
	stencil.ColorSpace, stencil.ImageMask = "", true
	return []scanCase{
		{name: "fax A4", digest: "76c62192f7abbde025bb40fd63e1d9652029251eadee7fdec5128410f13cc2d2", dpi: 300, im: pdftest.Image{Width: fw, Height: fh, BPC: 1, ColorSpace: "/DeviceGray",
			Filter: "/CCITTFaxDecode", DecodeParms: fmt.Sprintf("<</K -1 /Columns %d /Rows %d>>", fw, fh), Data: mustRead("testdata/a4.g4")}},
		{name: "fax stencil", digest: "76c62192f7abbde025bb40fd63e1d9652029251eadee7fdec5128410f13cc2d2", dpi: 300, im: pdftest.Image{Width: fw, Height: fh, BPC: 1, ImageMask: true,
			Filter: "/CCITTFaxDecode", DecodeParms: fmt.Sprintf("<</K -1 /Columns %d /Rows %d>>", fw, fh), Data: mustRead("testdata/a4.g4")}},
		{name: "jpx grey", digest: "2f8b2702575f6d17fe2f09f421affe85e8caa481e249f7d4be63643fd5c8456c", dpi: 300, im: pdftest.Image{Width: 1024, Height: 1024, BPC: 8, ColorSpace: "/DeviceGray",
			Filter: "/JPXDecode", Data: mustRead("../jpx/testdata/big97.jp2")}},
		{name: "jpx rgb tiles", digest: "dea350df1b980467eb9f5e03f7c2ef6f6eb549107cbe3a7366f061e1171091d6", dpi: 150, im: pdftest.Image{Width: 512, Height: 512, BPC: 8, ColorSpace: "/DeviceRGB",
			Filter: "/JPXDecode", Data: mustRead("../jpx/testdata/rgb97tiles.jp2")}},
		{name: "jpx reversible", digest: "47f0373b021c85bf937ae746eca6c1a39d83f7dfdfda2945d5e7982deda975f9", dpi: 150, im: pdftest.Image{Width: 512, Height: 512, BPC: 8, ColorSpace: "/DeviceGray",
			Filter: "/JPXDecode", Data: mustRead("../jpx/testdata/rev53.jp2")}},
		{name: "jbig2 generic", digest: "76740828d700cb59de82289e95237829f9acb45c83af8469c8397d217757b54d", dpi: 150, im: pdftest.Image{Width: 480, Height: 200, BPC: 1, ColorSpace: "/DeviceGray",
			Filter: "/JBIG2Decode", Data: mustRead("../jbig2/testdata/generic.jb2")}},
		{name: "jbig2 text", digest: "2e6a9a2461db0d5072cde2855be999cd8ebbb7b328017ed90ac1e16be38e815a", dpi: 150, im: pdftest.Image{Width: 480, Height: 200, BPC: 1, ColorSpace: "/DeviceGray",
			Filter: "/JBIG2Decode", Data: mustRead("../jbig2/testdata/sym0.jb2"), Globals: mustRead("../jbig2/testdata/sym.sym")}},
		{name: "flate rgb", digest: "f539c670674faefd54300f76cb8f480a84c6776d4f1218bc04bc6a2ca5245997", dpi: 150, im: pdftest.Image{Width: 1240, Height: 1754, BPC: 8, ColorSpace: "/DeviceRGB",
			Filter: "/FlateDecode", Data: flate(rgb)}},
		{name: "flate indexed", digest: "83e11351102993bfe187c6ce318bfbd10362d809cc2d252ca7a350b125e9e8b1", dpi: 150, im: pdftest.Image{Width: 1240, Height: 1754, BPC: 8,
			ColorSpace: "[/Indexed /DeviceRGB 255 <" + hex.EncodeToString(palette) + ">]", Filter: "/FlateDecode", Data: flate(idx)}},
		{name: "dct rgb", dpi: 150, im: pdftest.Image{Width: 1240, Height: 1754, BPC: 8, ColorSpace: "/DeviceRGB",
			Filter: "/DCTDecode", Data: jb.Bytes()}},
		// Images finer than the page, reduced by 2×2 and 3×3 samples, a
		// soft mask, a turned stencil and samples of 4 bits decoded.
		{name: "fax 600 dpi", digest: "e4a5217844de0299caea2f6cceb537b337f6c9ec60cc4a72122a72ddc9b2d411", dpi: 600, im: fax},
		{name: "jpx 900 dpi", digest: "86eebf1e958fb1a0de5dd0de8bcb864c46f3a9bfbddd818641c5a8776a57ccc1", dpi: 900, im: pdftest.Image{Width: 1024, Height: 1024, BPC: 8, ColorSpace: "/DeviceGray",
			Filter: "/JPXDecode", Data: mustRead("../jpx/testdata/big97.jp2")}},
		{name: "soft mask 450 dpi", digest: "c79af293a23161f331e73770abd8e81a88b2d0fc6ed46710630c27ac791a67ec", dpi: 450, im: pdftest.Image{Width: 1240, Height: 1754, BPC: 8, ColorSpace: "/DeviceGray",
			Filter: "/FlateDecode", Data: flate(idx), SMask: &pdftest.Image{Width: 620, Height: 877, BPC: 8, ColorSpace: "/DeviceGray",
				Filter: "/FlateDecode", Data: flate(texture(620, 877, 1)[1:])}}},
		{name: "turned stencil", digest: "5894a220447f4ca7c241d7205cb0a86caa4e1e2d6cba59b9602a3dd0d82ea0eb", dpi: 300, cm: "500 120 -170 700 150 40", im: stencil},
		{name: "grey 4 bits decoded", digest: "7f3e745cb09137227a553e5b4d40afd7312fbea1e0baf76272cb439b5c3f11ed", dpi: 150, im: pdftest.Image{Width: 1240, Height: 1754, BPC: 4, ColorSpace: "/DeviceGray",
			Extra: "/Decode [1 0]", Filter: "/FlateDecode", Data: flate(idx[:620*1754])}},
	}
}

func (c scanCase) doc() []byte {
	pw, ph := float64(c.im.Width)*72/c.dpi, float64(c.im.Height)*72/c.dpi
	content := fmt.Sprintf("q %g 0 0 %g 0 0 cm /Im1 Do Q", pw, ph)
	if c.cm != "" {
		content = "q " + c.cm + " cm /Im1 Do Q"
	}
	return pdftest.Doc{Pages: []pdftest.Page{{Width: pw, Height: ph, Content: content, Images: []pdftest.Image{c.im}}}}.Bytes()
}

func scanBytes(doc []byte) (*pdf.Scan, error) {
	r, err := pdf.Open(doc, pdf.DefaultLimits)
	if err != nil {
		return nil, err
	}
	p, err := r.Page(1)
	if err != nil {
		return nil, err
	}
	return r.Scan(p)
}

func scanDigest(s *pdf.Scan) string {
	h := sha256.New()
	fmt.Fprintf(h, "%v %v %q ", s.Image.Rect, s.DPI, s.Skipped)
	h.Write(s.Image.Pix)
	return hex.EncodeToString(h.Sum(nil))
}

// TestScanGolden pins, bit for bit, the page each codec composes. Painting
// computes in floating point, which a compiler may fuse into multiply-adds
// off amd64: the digests hold there only.
func TestScanGolden(t *testing.T) {
	for _, c := range scanCases() {
		s, err := scanBytes(c.doc())
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if lo, hi := slices.Min(s.Image.Pix), slices.Max(s.Image.Pix); lo == hi {
			t.Errorf("%s: a uniform page of %d", c.name, lo)
		}
		if c.digest == "" || runtime.GOARCH != "amd64" {
			t.Logf("%s: %s", c.name, scanDigest(s))
			continue
		}
		if got := scanDigest(s); got != c.digest {
			t.Errorf("%s: digest %s, want %s", c.name, got, c.digest)
		}
	}
}

func BenchmarkScan(b *testing.B) {
	for _, c := range scanCases() {
		doc := c.doc()
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := scanBytes(doc); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
