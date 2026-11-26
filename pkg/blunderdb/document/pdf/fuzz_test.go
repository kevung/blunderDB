package pdf_test

import (
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/document/pdf"
	"github.com/kevung/blunderdb/pkg/blunderdb/document/pdf/pdftest"
)

// FuzzOpen holds the reader to its contract on a downloaded file: whatever
// the bytes, an error or an answer, never a panic and never a hang.
func FuzzOpen(f *testing.F) {
	f.Add(sample(false))
	f.Add(sample(true))
	s := sample(true)
	f.Add(s[:len(s)/2])
	f.Add([]byte("%PDF-1.4\n1 0 obj <</Type /Catalog /Pages 2 0 R>> endobj trailer <</Root 1 0 R>>"))
	f.Add(scanSample())
	lim := pdf.Limits{MaxFileSize: 1 << 20, MaxStreamSize: 1 << 20, MaxDecoded: 8 << 20, MaxPages: 64, MaxObjects: 1 << 16, MaxGlyphs: 1 << 14}
	f.Fuzz(func(t *testing.T, data []byte) {
		tm := time.AfterFunc(10*time.Second, func() { panic("an input took more than 10 s") })
		defer tm.Stop()
		r, err := pdf.Open(data, lim)
		if err != nil {
			return
		}
		r.Info()
		r.Outline()
		for n := 1; n <= r.NumPages(); n++ {
			p, err := r.Page(n)
			if err != nil {
				t.Fatalf("page %d of %d: %v", n, r.NumPages(), err)
			}
			r.Fonts(p)
			_, _ = r.Glyphs(p)
			_, _ = r.Scan(p)
		}
	})
}

// scanSample is a scanned page: a grey image, a fax stencil, an inline
// image and an invisible text layer.
func scanSample() []byte {
	img := pdftest.Image{Width: 16, Height: 16, BPC: 8, ColorSpace: "/DeviceGray", Filter: "/FlateDecode", Data: flate(greyPage(16, 16, 2, 2, 8, 8))}
	mask := pdftest.Image{Width: 32, Height: 4, BPC: 1, ImageMask: true, Filter: "/CCITTFaxDecode", DecodeParms: "<</K -1 /Columns 32 /Rows 4>>", Data: g4Bar()}
	content := "q 72 0 0 72 0 0 cm /Im1 Do Q q 0 g 72 0 0 9 0 0 cm /Im2 Do Q q 9 0 0 9 9 9 cm BI /W 2 /H 1 /BPC 8 /CS /G ID \x00\x00 EI Q BT 3 Tr /F1 10 Tf 5 50 Td (12 pips) Tj ET"
	return pdftest.Doc{Fonts: []pdftest.Font{{BaseFont: "Helvetica"}}, Pages: []pdftest.Page{{Width: 72, Height: 72, Content: content, Images: []pdftest.Image{img, mask}}}}.Bytes()
}
