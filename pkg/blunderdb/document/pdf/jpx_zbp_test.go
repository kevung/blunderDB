package pdf_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/document/pdf"
	"github.com/kevung/blunderdb/pkg/blunderdb/document/pdf/pdftest"
)

// zeroBitPlanes is a codestream of 2048 code-blocks of 16×16 in one
// precinct whose packet header, after the first inclusion, runs zeros: a
// count of zero bit-planes without end, which no context check interrupts.
func zeroBitPlanes(zeros int) []byte {
	var b bytes.Buffer
	u16 := func(v int) { _ = binary.Write(&b, binary.BigEndian, uint16(v)) }
	u32 := func(v int) { _ = binary.Write(&b, binary.BigEndian, uint32(v)) }
	const w, h = 2048 * 16, 16
	u16(0xFF4F)
	u16(0xFF51) // SIZ: one grey component, one tile
	u16(41)
	u16(0)
	for _, v := range []int{w, h, 0, 0, w, h, 0, 0} {
		u32(v)
	}
	u16(1)
	b.Write([]byte{7, 1, 1})
	u16(0xFF52) // COD: no decomposition, code-blocks of 16×16, 5/3
	u16(12)
	b.Write([]byte{0, 0, 0, 1, 0, 0, 2, 2, 0, 1})
	u16(0xFF5C) // QCD: two guard bits, exponent 8
	u16(4)
	b.Write([]byte{2 << 5, 8 << 3})
	// The packet header: not empty, the first code-block included (12
	// levels of the inclusion tree), 11 levels of the zero bit-plane tree
	// known, then zeros. 24 ones are three 0xFF bytes, each followed by a
	// stuffed bit, the zeros' first.
	body := append([]byte{0xFF, 0x7F, 0xBF, 0xC0}, make([]byte, zeros)...)
	u16(0xFF90)
	u16(10)
	u16(0)
	u32(14 + len(body))
	b.Write([]byte{0, 1})
	u16(0xFF93)
	b.Write(body)
	u16(0xFFD9)
	return b.Bytes()
}

// TestJPXZeroBitPlanesBounded: a count of zero bit-planes is bounded by the
// band's bit-planes, so a small PDF ([/FlateDecode /JPXDecode]) cannot buy
// seconds of decoding past its caller's deadline.
func TestJPXZeroBitPlanesBounded(t *testing.T) {
	cs := zeroBitPlanes(16 << 20)
	im := pdftest.Image{Width: 2048 * 16, Height: 16, BPC: 8, ColorSpace: "/DeviceGray", Filter: "[/FlateDecode /JPXDecode]", Data: flate(cs)}
	d := pdftest.Doc{Pages: []pdftest.Page{{Width: 2048 * 16, Height: 16, Content: "32768 0 0 16 0 0 cm /Im1 Do", Images: []pdftest.Image{im}}}}
	data := d.Bytes()
	if len(data) > 64<<10 {
		t.Fatalf("a PDF of %d bytes", len(data))
	}
	r, err := pdf.Open(data, pdf.DefaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	p, err := r.Page(1)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	sc, err := r.ScanContext(ctx, p)
	if d := time.Since(start); d > 2*time.Second {
		t.Errorf("refused after %v", d)
	}
	if err == nil && len(sc.Skipped) == 0 {
		t.Error("the image is painted")
	}
}
