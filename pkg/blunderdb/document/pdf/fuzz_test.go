package pdf_test

import (
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/document/pdf"
)

// FuzzOpen holds the reader to its contract on a downloaded file: whatever
// the bytes, an error or an answer, never a panic and never a hang.
func FuzzOpen(f *testing.F) {
	f.Add(sample(false))
	f.Add(sample(true))
	s := sample(true)
	f.Add(s[:len(s)/2])
	f.Add([]byte("%PDF-1.4\n1 0 obj <</Type /Catalog /Pages 2 0 R>> endobj trailer <</Root 1 0 R>>"))
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
		}
	})
}
