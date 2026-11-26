// Package pdftest builds small PDF files for tests: fonts with a ToUnicode
// table and widths, pages of hand-written content, an outline and the
// document information. The output is deterministic, so a file built from it
// can be kept under testdata and compared.
package pdftest

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"strings"
)

// Font is a font the pages may name.
type Font struct {
	// BaseFont is the font's name.
	BaseFont string
	// Type0 makes a composite font with two-byte codes (Identity-H) whose
	// code is the character's Unicode value; otherwise a simple WinAnsi font.
	Type0 bool
	// Widths gives a code's width in thousandths of the font size; others
	// are 1000 for a Type0 font and 500 for a simple one.
	Widths map[rune]int
	// ToUnicode maps codes First..Last to the same Unicode values (Type0).
	First, Last rune
}

// Page is one page: its size and content stream; fonts are named /F1, /F2…
// in the order of Doc.Fonts.
type Page struct {
	Width, Height float64
	Content       string
	// Compress stores the content with the Flate filter.
	Compress bool
}

// Bookmark is a top-level outline item opening a page (from 1).
type Bookmark struct {
	Title string
	Page  int
}

// Doc is the whole file.
type Doc struct {
	Title, Author string
	Fonts         []Font
	Pages         []Page
	Outline       []Bookmark
}

// Hex returns s as a hex string of two-byte codes, for a Type0 font.
func Hex(s string) string {
	var b strings.Builder
	b.WriteByte('<')
	for _, r := range s {
		fmt.Fprintf(&b, "%04X", r)
	}
	b.WriteByte('>')
	return b.String()
}

// Literal returns s as a literal string for a simple font.
func Literal(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `(`, `\(`, `)`, `\)`)
	return "(" + r.Replace(s) + ")"
}

// Bytes writes the file.
func (d Doc) Bytes() []byte {
	var objs []string // objs[i] is object i+1
	add := func(s string) int {
		objs = append(objs, s)
		return len(objs)
	}
	reserve := func() int { return add("") }
	stream := func(dict string, data []byte) string {
		return fmt.Sprintf("<<%s /Length %d>>\nstream\n%s\nendstream", dict, len(data), data)
	}
	catalog := reserve()
	pages := reserve()
	info := add(fmt.Sprintf("<</Title %s /Author %s /Producer (pdftest)>>", Literal(d.Title), Literal(d.Author)))
	var fontRefs []int
	for _, f := range d.Fonts {
		if !f.Type0 {
			var w strings.Builder
			for c := 32; c <= 255; c++ {
				v := 500
				if x, ok := f.Widths[rune(c)]; ok {
					v = x
				}
				fmt.Fprintf(&w, " %d", v)
			}
			fontRefs = append(fontRefs, add(fmt.Sprintf("<</Type /Font /Subtype /Type1 /BaseFont /%s /Encoding /WinAnsiEncoding /FirstChar 32 /LastChar 255 /Widths [%s]>>", f.BaseFont, w.String())))
			continue
		}
		cmap := fmt.Sprintf("/CIDInit /ProcSet findresource begin\n12 dict begin\nbegincmap\n1 begincodespacerange\n<0000> <FFFF>\nendcodespacerange\n1 beginbfrange\n<%04X> <%04X> <%04X>\nendbfrange\nendcmap\nend\nend", f.First, f.Last, f.First)
		tu := add(stream("", []byte(cmap)))
		var w strings.Builder
		for c := f.First; c <= f.Last; c++ {
			if x, ok := f.Widths[c]; ok {
				fmt.Fprintf(&w, " %d [%d]", c, x)
			}
		}
		desc := add(fmt.Sprintf("<</Type /Font /Subtype /CIDFontType2 /BaseFont /%s /CIDSystemInfo <</Registry (Adobe) /Ordering (Identity) /Supplement 0>> /DW 1000 /W [%s]>>", f.BaseFont, w.String()))
		fontRefs = append(fontRefs, add(fmt.Sprintf("<</Type /Font /Subtype /Type0 /BaseFont /%s /Encoding /Identity-H /DescendantFonts [%d 0 R] /ToUnicode %d 0 R>>", f.BaseFont, desc, tu)))
	}
	var fonts strings.Builder
	for i, r := range fontRefs {
		fmt.Fprintf(&fonts, " /F%d %d 0 R", i+1, r)
	}
	var kids []int
	for _, p := range d.Pages {
		var content int
		if p.Compress {
			var z bytes.Buffer
			zw := zlib.NewWriter(&z)
			_, _ = zw.Write([]byte(p.Content))
			_ = zw.Close()
			content = add(stream(" /Filter /FlateDecode", z.Bytes()))
		} else {
			content = add(stream("", []byte(p.Content)))
		}
		kids = append(kids, add(fmt.Sprintf("<</Type /Page /Parent %d 0 R /MediaBox [0 0 %g %g] /Resources <</Font <<%s>>>> /Contents %d 0 R>>", pages, p.Width, p.Height, fonts.String(), content)))
	}
	var kidList strings.Builder
	for _, k := range kids {
		fmt.Fprintf(&kidList, " %d 0 R", k)
	}
	objs[pages-1] = fmt.Sprintf("<</Type /Pages /Kids [%s] /Count %d>>", kidList.String(), len(kids))
	objs[catalog-1] = fmt.Sprintf("<</Type /Catalog /Pages %d 0 R>>", pages)
	if len(d.Outline) > 0 {
		root := reserve()
		first := len(objs) + 1
		for i, b := range d.Outline {
			n := first + i
			links := ""
			if i > 0 {
				links += fmt.Sprintf(" /Prev %d 0 R", n-1)
			}
			if i < len(d.Outline)-1 {
				links += fmt.Sprintf(" /Next %d 0 R", n+1)
			}
			add(fmt.Sprintf("<</Title %s /Parent %d 0 R /Dest [%d 0 R /Fit]%s>>", Literal(b.Title), root, kids[b.Page-1], links))
		}
		objs[root-1] = fmt.Sprintf("<</Type /Outlines /First %d 0 R /Last %d 0 R /Count %d>>", first, len(objs), len(d.Outline))
		objs[catalog-1] = fmt.Sprintf("<</Type /Catalog /Pages %d 0 R /Outlines %d 0 R>>", pages, root)
	}
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n%\xe2\xe3\xcf\xd3\n")
	offs := make([]int, len(objs))
	for i, o := range objs {
		offs[i] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", i+1, o)
	}
	xref := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n0000000000 65535 f \n", len(objs)+1)
	for _, o := range offs {
		fmt.Fprintf(&buf, "%010d 00000 n \n", o)
	}
	fmt.Fprintf(&buf, "trailer\n<</Size %d /Root %d 0 R /Info %d 0 R>>\nstartxref\n%d\n%%%%EOF\n", len(objs)+1, catalog, info, xref)
	return buf.Bytes()
}
