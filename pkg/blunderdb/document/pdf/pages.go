package pdf

import (
	"bytes"
	"fmt"
	"strings"
	"unicode/utf16"
)

// Page is one page of the document.
type Page struct {
	// Number is the page's place in the document, from 1.
	Number    int
	num       int // object number of the page dictionary
	dict      Dict
	resources Dict
	// Box is the visible area (CropBox, else MediaBox): x0, y0, x1, y1.
	Box [4]float64
}

// NumPages returns the number of pages.
func (r *Reader) NumPages() int { return len(r.pages) }

// Page returns page n, from 1.
func (r *Reader) Page(n int) (Page, error) {
	if n < 1 || n > len(r.pages) {
		return Page{}, fmt.Errorf("pdf: page %d of %d", n, len(r.pages))
	}
	return r.pages[n-1], nil
}

func (r *Reader) loadPages() error {
	root := r.rootDict()
	type frame struct {
		node      Object
		resources Dict
		box       Object
	}
	stack := []frame{{node: root["Pages"]}}
	visited := map[int]bool{}
	for len(stack) > 0 {
		f := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		num := -1
		if ref, ok := f.node.(Ref); ok {
			if visited[ref.Num] {
				continue
			}
			visited[ref.Num] = true
			num = ref.Num
		}
		d, ok := r.Resolve(f.node).(Dict)
		if !ok {
			continue
		}
		res := f.resources
		if rd, ok := r.Resolve(d["Resources"]).(Dict); ok {
			res = rd
		}
		box := f.box
		if b := d["MediaBox"]; b != nil {
			box = b
		}
		if b := d["CropBox"]; b != nil {
			box = b
		}
		kids, isNode := r.Resolve(d["Kids"]).(Array)
		if d["Type"] == Name("Page") || !isNode {
			if len(r.pages) >= r.lim.MaxPages {
				return fmt.Errorf("%w: more than %d pages", ErrTooLarge, r.lim.MaxPages)
			}
			p := Page{Number: len(r.pages) + 1, num: num, dict: d, resources: res, Box: [4]float64{0, 0, 612, 792}}
			if a, ok := r.Resolve(box).(Array); ok && len(a) == 4 {
				for i := range 4 {
					p.Box[i] = r.num(a[i])
				}
				if p.Box[0] > p.Box[2] {
					p.Box[0], p.Box[2] = p.Box[2], p.Box[0]
				}
				if p.Box[1] > p.Box[3] {
					p.Box[1], p.Box[3] = p.Box[3], p.Box[1]
				}
			}
			r.pages = append(r.pages, p)
			continue
		}
		if len(stack)+len(kids) > r.lim.MaxPages*2 {
			return fmt.Errorf("%w: page tree too wide", ErrTooLarge)
		}
		for i := len(kids) - 1; i >= 0; i-- {
			stack = append(stack, frame{node: kids[i], resources: res, box: box})
		}
	}
	if len(r.pages) == 0 {
		return malformed("no page")
	}
	return nil
}

func (r *Reader) num(o Object) float64 {
	switch v := r.Resolve(o).(type) {
	case int64:
		return float64(v)
	case float64:
		return v
	}
	return 0
}

// content returns the page's content streams, decoded and joined.
func (r *Reader) content(p Page) ([]byte, error) {
	var parts Array
	switch c := p.dict["Contents"].(type) {
	case Array:
		parts = c
	default:
		switch rc := r.Resolve(c).(type) {
		case Array:
			parts = rc
		case *Stream:
			parts = Array{rc}
		}
	}
	var buf bytes.Buffer
	for _, part := range parts {
		s, ok := r.Resolve(part).(*Stream)
		if !ok {
			continue
		}
		data, err := r.Decode(s)
		if err != nil {
			return nil, err
		}
		buf.Write(data)
		buf.WriteByte('\n')
	}
	return buf.Bytes(), nil
}

// Info returns the document's title and author from its information
// dictionary, empty when absent.
func (r *Reader) Info() (title, author string) {
	defer func() { _ = recover() }()
	d, ok := r.Resolve(r.trailer["Info"]).(Dict)
	if !ok {
		return "", ""
	}
	return TextString(r.Resolve(d["Title"])), TextString(r.Resolve(d["Author"]))
}

// TextString decodes a PDF text string: UTF-16 with a byte-order mark, or
// PDFDocEncoding, read as Latin-1 — they differ only in a few punctuation
// marks. Control characters are dropped.
func TextString(o Object) string {
	s, ok := o.(String)
	if !ok {
		return ""
	}
	b := []byte(s)
	var out string
	if len(b) >= 2 && b[0] == 0xFE && b[1] == 0xFF {
		u := make([]uint16, 0, len(b)/2)
		for i := 2; i+1 < len(b); i += 2 {
			u = append(u, uint16(b[i])<<8|uint16(b[i+1]))
		}
		out = string(utf16.Decode(u))
	} else {
		rs := make([]rune, len(b))
		for i, c := range b {
			rs[i] = rune(c)
		}
		out = string(rs)
	}
	return strings.TrimSpace(strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7F {
			return -1
		}
		return r
	}, out))
}

// OutlineItem is a top-level bookmark: its title and the page it opens.
type OutlineItem struct {
	Title string
	Page  int
}

// Outline returns the top-level bookmarks that open a page, in their order.
func (r *Reader) Outline() (items []OutlineItem) {
	defer func() {
		if recover() != nil {
			items = nil
		}
	}()
	root := r.rootDict()
	o, ok := r.Resolve(root["Outlines"]).(Dict)
	if !ok {
		return nil
	}
	byNum := map[int]int{}
	for _, p := range r.pages {
		if p.num >= 0 {
			byNum[p.num] = p.Number
		}
	}
	visited := map[int]bool{}
	cur := o["First"]
	for range 10000 {
		ref, ok := cur.(Ref)
		if !ok || visited[ref.Num] {
			break
		}
		visited[ref.Num] = true
		d, ok := r.Resolve(ref).(Dict)
		if !ok {
			break
		}
		if page := r.destPage(d, byNum); page > 0 {
			items = append(items, OutlineItem{Title: TextString(r.Resolve(d["Title"])), Page: page})
		}
		cur = d["Next"]
	}
	return items
}

func (r *Reader) destPage(item Dict, byNum map[int]int) int {
	dest := r.Resolve(item["Dest"])
	if dest == nil {
		if a, ok := r.Resolve(item["A"]).(Dict); ok && a["S"] == Name("GoTo") {
			dest = r.Resolve(a["D"])
		}
	}
	for range 4 { // a named destination resolves to an array or a dictionary with /D
		switch d := dest.(type) {
		case Array:
			if len(d) == 0 {
				return 0
			}
			switch p := d[0].(type) {
			case Ref:
				return byNum[p.Num]
			case int64: // a page index, in a remote-style destination
				if p >= 0 && int(p) < len(r.pages) {
					return int(p) + 1
				}
			}
			return 0
		case Dict:
			dest = r.Resolve(d["D"])
		case Name:
			dest = r.named(string(d))
		case String:
			dest = r.named(string(d))
		default:
			return 0
		}
	}
	return 0
}

// named looks a destination up in the catalog's /Dests dictionary or its
// /Names /Dests name tree.
func (r *Reader) named(name string) Object {
	root := r.rootDict()
	if d, ok := r.Resolve(root["Dests"]).(Dict); ok {
		if o := d[Name(name)]; o != nil {
			return r.Resolve(o)
		}
	}
	names, ok := r.Resolve(root["Names"]).(Dict)
	if !ok {
		return nil
	}
	stack := []Object{names["Dests"]}
	visited := map[int]bool{}
	for steps := 0; len(stack) > 0 && steps < 10000; steps++ {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if ref, ok := n.(Ref); ok {
			if visited[ref.Num] {
				continue
			}
			visited[ref.Num] = true
		}
		d, ok := r.Resolve(n).(Dict)
		if !ok {
			continue
		}
		if a, ok := r.Resolve(d["Names"]).(Array); ok {
			for i := 0; i+1 < len(a); i += 2 {
				if k, ok := r.Resolve(a[i]).(String); ok && string(k) == name {
					return r.Resolve(a[i+1])
				}
			}
		}
		if kids, ok := r.Resolve(d["Kids"]).(Array); ok && len(kids) < 10000 {
			stack = append(stack, kids...)
		}
	}
	return nil
}

// Fonts returns the fonts of the page's resources (not those of its form
// XObjects), so a caller can tell whether a page may draw with a font before
// reading its content.
func (r *Reader) Fonts(p Page) (fs []*Font) {
	defer func() {
		if recover() != nil {
			fs = nil
		}
	}()
	d, _ := r.Resolve(p.resources["Font"]).(Dict)
	for _, o := range d {
		fs = append(fs, r.font(o))
	}
	return fs
}
