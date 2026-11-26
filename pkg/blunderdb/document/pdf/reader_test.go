package pdf_test

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/document/pdf"
	"github.com/kevung/blunderdb/pkg/blunderdb/document/pdf/pdftest"
)

func sample(compress bool) []byte {
	return pdftest.Doc{
		Title:  "A (test) book",
		Author: "Someone",
		Fonts: []pdftest.Font{
			{BaseFont: "ABCDEF+Board", Type0: true, First: 0x20, Last: 0xFF, Widths: map[rune]int{'@': 50, 'ð': 250}},
			{BaseFont: "Helvetica", Widths: map[rune]int{' ': 250}},
		},
		Pages: []pdftest.Page{
			{Width: 600, Height: 800, Content: "BT /F2 10 Tf 72 700 Td (Title page) Tj ET"},
			{Width: 600, Height: 800, Compress: compress, Content: "q 1 0 0 1 10 20 cm BT /F1 24 Tf 50 500 Td " +
				pdftest.Hex(" ð@x") + " Tj 0 -24 Td [" + pdftest.Hex("A") + " -1000 " + pdftest.Hex("B") + "] TJ ET Q " +
				"BT /F2 12 Tf 100 100 Td 2 Tc (a b) Tj T* ET"},
		},
		Outline: []pdftest.Bookmark{{Title: "Second", Page: 2}, {Title: "First", Page: 1}},
	}.Bytes()
}

func TestReadsInfoOutlineAndGlyphs(t *testing.T) {
	for _, compress := range []bool{false, true} {
		r, err := pdf.Open(sample(compress), pdf.DefaultLimits)
		if err != nil {
			t.Fatal(err)
		}
		if title, author := r.Info(); title != "A (test) book" || author != "Someone" {
			t.Errorf("info %q %q", title, author)
		}
		if got := fmt.Sprint(r.Outline()); got != "[{Second 2} {First 1}]" {
			t.Errorf("outline %s", got)
		}
		p, err := r.Page(2)
		if err != nil {
			t.Fatal(err)
		}
		gs, err := r.Glyphs(p)
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for _, g := range gs {
			got = append(got, fmt.Sprintf("%s@%.1f,%.1f/%.1f", g.Text, g.X, g.Y, g.Size))
		}
		want := []string{
			" @60.0,520.0/24.0", "ð@84.0,520.0/24.0", "@@90.0,520.0/24.0", "x@91.2,520.0/24.0",
			"A@60.0,496.0/24.0", "B@108.0,496.0/24.0",
			"a@100.0,100.0/12.0", " @108.0,100.0/12.0", "b@113.0,100.0/12.0",
		}
		if strings.Join(got, " ") != strings.Join(want, " ") {
			t.Errorf("glyphs\n got %v\nwant %v", got, want)
		}
		if !gs[1].Font.Maps('ð', '@') || gs[1].Font.BaseFont != "Board" || gs[6].Font.Maps('ð') {
			t.Errorf("font detection by its characters")
		}
	}
}

func TestRepairsWrongOffsets(t *testing.T) {
	data := sample(true)
	i := bytes.LastIndex(data, []byte("startxref"))
	broken := append(append([]byte{}, data[:i]...), []byte("startxref\n7\n%%EOF\n")...)
	// Shift every object: the table's offsets are now all wrong.
	broken = append([]byte("%PDF-1.4\n% padding padding\n"), broken[9:]...)
	r, err := pdf.Open(broken, pdf.DefaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	if r.NumPages() != 2 {
		t.Fatalf("pages %d", r.NumPages())
	}
	p, _ := r.Page(2)
	if gs, err := r.Glyphs(p); err != nil || len(gs) != 9 {
		t.Fatalf("glyphs %d %v", len(gs), err)
	}
}

// TestCrossReferenceStream reads a PDF 1.5 file whose objects sit in an
// object stream, indexed by a cross-reference stream.
func TestCrossReferenceStream(t *testing.T) {
	var b bytes.Buffer
	b.WriteString("%PDF-1.5\n")
	content := "BT /F1 10 Tf 5 6 Td (hi) Tj ET"
	off3 := b.Len()
	fmt.Fprintf(&b, "3 0 obj\n<</Length %d>>\nstream\n%s\nendstream\nendobj\n", len(content), content)
	objs := []string{
		"<</Type /Catalog /Pages 2 0 R>>",
		"<</Type /Pages /Kids [4 0 R] /Count 1>>",
		"<</Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] /Contents 3 0 R /Resources <</Font <</F1 6 0 R>>>>>>",
		"<</Type /Font /Subtype /Type1 /BaseFont /Helvetica>>",
	}
	nums := []int{1, 2, 4, 6}
	var head, body strings.Builder
	for i, o := range objs {
		fmt.Fprintf(&head, "%d %d ", nums[i], body.Len())
		body.WriteString(o + " ")
	}
	stm := head.String() + body.String()
	off5 := b.Len()
	fmt.Fprintf(&b, "5 0 obj\n<</Type /ObjStm /N 4 /First %d /Length %d>>\nstream\n%s\nendstream\nendobj\n", head.Len(), len(stm), stm)
	off7 := b.Len()
	row := func(t, f2, f3 int) []byte { return []byte{byte(t), byte(f2 >> 8), byte(f2), byte(f3)} }
	var x []byte
	x = append(x, row(0, 0, 255)...)
	x = append(x, row(2, 5, 0)...)
	x = append(x, row(2, 5, 1)...)
	x = append(x, row(1, off3, 0)...)
	x = append(x, row(2, 5, 2)...)
	x = append(x, row(1, off5, 0)...)
	x = append(x, row(2, 5, 3)...)
	x = append(x, row(1, off7, 0)...)
	fmt.Fprintf(&b, "7 0 obj\n<</Type /XRef /Size 8 /W [1 2 1] /Root 1 0 R /Length %d>>\nstream\n", len(x))
	b.Write(x)
	fmt.Fprintf(&b, "\nendstream\nendobj\nstartxref\n%d\n%%%%EOF\n", off7)
	r, err := pdf.Open(b.Bytes(), pdf.DefaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	p, _ := r.Page(1)
	gs, err := r.Glyphs(p)
	if err != nil || len(gs) != 2 || gs[0].Text != "h" || gs[0].X != 5 {
		t.Fatalf("glyphs %+v %v", gs, err)
	}
}

func TestRefusals(t *testing.T) {
	if _, err := pdf.Open([]byte("hello"), pdf.DefaultLimits); !errors.Is(err, pdf.ErrNotPDF) {
		t.Errorf("not a PDF: %v", err)
	}
	enc := bytes.Replace(sample(false), []byte("/Info"), []byte("/Encrypt 3 0 R /Info"), 1)
	if _, err := pdf.Open(enc, pdf.DefaultLimits); !errors.Is(err, pdf.ErrEncrypted) {
		t.Errorf("encrypted: %v", err)
	}
	lim := pdf.DefaultLimits
	lim.MaxFileSize = 100
	if _, err := pdf.Open(sample(false), lim); !errors.Is(err, pdf.ErrTooLarge) {
		t.Errorf("file size: %v", err)
	}
	lim = pdf.DefaultLimits
	lim.MaxPages = 1
	if _, err := pdf.Open(sample(false), lim); !errors.Is(err, pdf.ErrTooLarge) {
		t.Errorf("pages: %v", err)
	}
	lim = pdf.DefaultLimits
	lim.MaxStreamSize = 20
	r, err := pdf.Open(sample(true), lim)
	if err != nil {
		t.Fatal(err)
	}
	p, _ := r.Page(2)
	if _, err := r.Glyphs(p); !errors.Is(err, pdf.ErrTooLarge) {
		t.Errorf("stream size: %v", err)
	}
	lim = pdf.DefaultLimits
	lim.MaxGlyphs = 3
	r, _ = pdf.Open(sample(false), lim)
	p, _ = r.Page(2)
	if _, err := r.Glyphs(p); !errors.Is(err, pdf.ErrTooLarge) {
		t.Errorf("glyph count: %v", err)
	}
}

func TestTextString(t *testing.T) {
	if got := pdf.TextString(pdf.String("\xfe\xff\x00D\x00\xe9\x00p")); got != "Dép" {
		t.Errorf("UTF-16: %q", got)
	}
	if got := pdf.TextString(pdf.String("D\xe9p\n")); got != "Dép" {
		t.Errorf("PDFDocEncoding: %q", got)
	}
}

// TestHostileStructures: loops and absurd numbers end in an error or an
// empty answer, never a hang.
func TestHostileStructures(t *testing.T) {
	cases := []string{
		"%PDF-1.4\n1 0 obj <</Type /Catalog /Pages 2 0 R>> endobj 2 0 obj <</Type /Pages /Kids [2 0 R 2 0 R]>> endobj trailer <</Root 1 0 R>>",
		"%PDF-1.4\n1 0 obj <</Type /Catalog /Pages 1 0 R /Outlines 1 0 R /First 1 0 R>> endobj trailer <</Root 1 0 R>>",
		"%PDF-1.4\n1 0 obj 2 0 R endobj 2 0 obj 1 0 R endobj trailer <</Root 1 0 R>>",
		"%PDF-1.4\n1 0 obj [[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[ endobj trailer <</Root 1 0 R>>",
		"%PDF-1.4\nxref\n0 99999999999\ntrailer <<>>\nstartxref\n9\n%%EOF",
		"%PDF-1.4\n1 0 obj <</Length 3 0 R>> stream\nabc\nendstream endobj 3 0 obj <</Length 1 0 R>> stream\nx\nendstream endobj trailer <</Root 1 0 R>>",
	}
	for i, c := range cases {
		r, err := pdf.Open([]byte(c), pdf.DefaultLimits)
		if err == nil {
			r.Outline()
			for n := 1; n <= r.NumPages(); n++ {
				p, _ := r.Page(n)
				_, _ = r.Glyphs(p)
			}
		}
		_ = i
	}
}

func TestMatrixHonoursRotation(t *testing.T) {
	data := pdftest.Doc{
		Fonts: []pdftest.Font{{BaseFont: "Helvetica"}},
		Pages: []pdftest.Page{{Width: 100, Height: 100, Content: "BT /F1 10 Tf 0 1 -1 0 50 50 Tm (a) Tj ET"}},
	}.Bytes()
	r, err := pdf.Open(data, pdf.DefaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	p, _ := r.Page(1)
	gs, _ := r.Glyphs(p)
	if len(gs) != 1 || gs[0].Upright || math.Abs(gs[0].Size-10) > 1e-9 {
		t.Fatalf("rotated glyph %+v", gs)
	}
}
