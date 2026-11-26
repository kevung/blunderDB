package pdf

import (
	"strconv"
	"strings"
	"sync"
	"unicode/utf16"
)

// Font is what the reader knows of a font: how its strings split into codes,
// each code's width and text.
type Font struct {
	// BaseFont is the font's name, its subset tag ("ABCDEF+") removed.
	BaseFont string
	twoByte  bool
	toUni    map[uint32]string
	runes    map[rune]bool
	width    map[uint32]float64 // text-space units per unit of font size
	dw       float64
	enc      *[256]rune
	// program reads the embedded TrueType program once, on demand.
	program func() []byte
}

// Program returns the font's embedded TrueType program (FontFile2) of a
// simple font, decoded, or nil: a font not embedded, of another kind, or
// whose stream does not decode within the reader's limits. A reader that
// knows a font by the shapes it draws reads them here.
func (f *Font) Program() []byte {
	if f.program == nil {
		return nil
	}
	return f.program()
}

// Maps reports whether the font's ToUnicode table gives some code each of
// these characters — how a font is recognised by what it draws rather than
// by its name.
func (f *Font) Maps(rs ...rune) bool {
	for _, r := range rs {
		if !f.runes[r] {
			return false
		}
	}
	return true
}

func (f *Font) text(code uint32) string {
	if s, ok := f.toUni[code]; ok {
		return s
	}
	if f.enc != nil && code < 256 {
		if r := f.enc[code]; r != 0 {
			return string(r)
		}
	}
	return ""
}

func (f *Font) advance(code uint32) float64 {
	if w, ok := f.width[code]; ok {
		return w
	}
	return f.dw
}

const maxCMapEntries = 1 << 17

// font reads the font object o (a reference or an inline dictionary).
func (r *Reader) font(o Object) *Font {
	ref, isRef := o.(Ref)
	if isRef {
		if f, ok := r.fonts[ref.Num]; ok {
			return f
		}
	}
	f := &Font{width: map[uint32]float64{}, dw: 0.5}
	if isRef {
		r.fonts[ref.Num] = f
	}
	d, ok := r.Resolve(o).(Dict)
	if !ok {
		return f
	}
	base, _ := r.Resolve(d["BaseFont"]).(Name)
	if i := strings.IndexByte(string(base), '+'); i == 6 {
		base = base[7:]
	}
	f.BaseFont = string(base)
	if s, ok := r.Resolve(d["ToUnicode"]).(*Stream); ok {
		if data, err := r.Decode(s); err == nil {
			f.toUni = parseToUnicode(data)
		}
	}
	f.runes = map[rune]bool{}
	for _, s := range f.toUni {
		for _, c := range s {
			f.runes[c] = true
		}
	}
	if d["Subtype"] == Name("Type0") {
		f.twoByte = true
		f.dw = 1
		desc, _ := r.Resolve(d["DescendantFonts"]).(Array)
		if len(desc) > 0 {
			if cd, ok := r.Resolve(desc[0]).(Dict); ok {
				if v, ok := r.Resolve(cd["DW"]).(int64); ok {
					f.dw = float64(v) / 1000
				}
				r.cidWidths(f, r.Resolve(cd["W"]))
			}
		}
		return f
	}
	scale := 0.001
	if d["Subtype"] == Name("Type3") {
		if m, ok := r.Resolve(d["FontMatrix"]).(Array); ok && len(m) == 6 {
			scale = r.num(m[0])
		}
	}
	first, _ := r.Resolve(d["FirstChar"]).(int64)
	if ws, ok := r.Resolve(d["Widths"]).(Array); ok && first >= 0 && first < 256 {
		for i, w := range ws {
			if int(first)+i > 255 {
				break
			}
			f.width[uint32(int(first)+i)] = r.num(w) * scale
		}
	}
	if fd, ok := r.Resolve(d["FontDescriptor"]).(Dict); ok {
		if mw := r.num(fd["MissingWidth"]); mw > 0 {
			f.dw = mw * scale
		}
		if s, ok := r.Resolve(fd["FontFile2"]).(*Stream); ok && d["Subtype"] == Name("TrueType") {
			f.program = sync.OnceValue(func() []byte {
				data, err := r.Decode(s)
				if err != nil {
					return nil
				}
				return data
			})
		}
	}
	f.enc = simpleEncoding(r, r.Resolve(d["Encoding"]))
	return f
}

func (r *Reader) cidWidths(f *Font, o Object) {
	w, ok := o.(Array)
	if !ok {
		return
	}
	n := 0
	for i := 0; i < len(w) && n < maxCMapEntries; {
		c1, ok := r.Resolve(w[i]).(int64)
		if !ok || i+1 >= len(w) {
			return
		}
		if list, ok := r.Resolve(w[i+1]).(Array); ok {
			for k, v := range list {
				f.width[uint32(c1)+uint32(k)] = r.num(v) / 1000
				n++
			}
			i += 2
			continue
		}
		c2, ok := r.Resolve(w[i+1]).(int64)
		if !ok || i+2 >= len(w) || c2 < c1 || c2-c1 > 65535 {
			return
		}
		v := r.num(w[i+2]) / 1000
		for c := c1; c <= c2; c++ {
			f.width[uint32(c)] = v
			n++
		}
		i += 3
	}
}

func codeOf(s String) uint32 {
	var c uint32
	for i := 0; i < len(s) && i < 4; i++ {
		c = c<<8 | uint32(s[i])
	}
	return c
}

func utf16BE(s String) string {
	if len(s)%2 == 1 {
		return string(s)
	}
	u := make([]uint16, len(s)/2)
	for i := range u {
		u[i] = uint16(s[2*i])<<8 | uint16(s[2*i+1])
	}
	return string(utf16.Decode(u))
}

// parseToUnicode reads the bfchar and bfrange sections of a ToUnicode CMap.
func parseToUnicode(data []byte) map[uint32]string {
	m := map[uint32]string{}
	l := &lexer{b: data}
	for {
		t := l.token()
		if _, ok := t.(eof); ok || len(m) > maxCMapEntries {
			return m
		}
		k, ok := t.(keyword)
		if !ok {
			continue
		}
		switch k {
		case "beginbfchar":
			for {
				src, ok := l.token().(String)
				if !ok {
					break
				}
				switch dst := l.token().(type) {
				case String:
					m[codeOf(src)] = utf16BE(dst)
				case Name:
					m[codeOf(src)] = string(dst)
				}
			}
		case "beginbfrange":
			for {
				lo, ok1 := l.token().(String)
				if !ok1 {
					break
				}
				hi, ok2 := l.token().(String)
				dst, err := l.object(false, 0)
				if !ok2 || err != nil {
					break
				}
				a, b := codeOf(lo), codeOf(hi)
				if b < a || b-a > 65535 || len(m) > maxCMapEntries {
					continue
				}
				switch d := dst.(type) {
				case String:
					base := []rune(utf16BE(d))
					if len(base) == 0 {
						continue
					}
					for c := a; c <= b; c++ {
						rs := append([]rune(nil), base...)
						rs[len(rs)-1] += rune(c - a)
						m[c] = string(rs)
					}
				case Array:
					for i, v := range d {
						if s, ok := v.(String); ok && a+uint32(i) <= b {
							m[a+uint32(i)] = utf16BE(s)
						}
					}
				}
			}
		}
	}
}

// cp1252 is WinAnsiEncoding where it differs from Latin-1.
var cp1252 = map[byte]rune{
	0x80: '€', 0x82: '‚', 0x83: 'ƒ', 0x84: '„', 0x85: '…', 0x86: '†', 0x87: '‡', 0x88: 'ˆ',
	0x89: '‰', 0x8A: 'Š', 0x8B: '‹', 0x8C: 'Œ', 0x8E: 'Ž', 0x91: '‘', 0x92: '’', 0x93: '“',
	0x94: '”', 0x95: '•', 0x96: '–', 0x97: '—', 0x98: '˜', 0x99: '™', 0x9A: 'š', 0x9B: '›',
	0x9C: 'œ', 0x9E: 'ž', 0x9F: 'Ÿ',
}

// glyphNames are the glyph names a /Differences array uses for the
// characters a diagram's caption needs: letters, digits and punctuation.
var glyphNames = map[string]rune{
	"space": ' ', "zero": '0', "one": '1', "two": '2', "three": '3', "four": '4', "five": '5',
	"six": '6', "seven": '7', "eight": '8', "nine": '9', "period": '.', "comma": ',',
	"colon": ':', "semicolon": ';', "hyphen": '-', "slash": '/', "parenleft": '(',
	"parenright": ')', "percent": '%', "plus": '+', "equal": '=', "numbersign": '#',
	"quoteright": '’', "quotesingle": '\'', "exclam": '!', "question": '?',
}

// simpleEncoding approximates a simple font's encoding: WinAnsi as the base
// (Latin-1 elsewhere), then the /Differences the font names.
func simpleEncoding(r *Reader, o Object) *[256]rune {
	var enc [256]rune
	for i := 32; i < 256; i++ {
		enc[i] = rune(i)
		if c, ok := cp1252[byte(i)]; ok {
			enc[i] = c
		}
	}
	d, ok := o.(Dict)
	if !ok {
		return &enc
	}
	diffs, _ := r.Resolve(d["Differences"]).(Array)
	code := -1
	for _, v := range diffs {
		switch x := v.(type) {
		case int64:
			code = int(x)
		case Name:
			if code >= 0 && code < 256 {
				enc[code] = glyphRune(string(x))
			}
			code++
		}
	}
	return &enc
}

func glyphRune(name string) rune {
	if r, ok := glyphNames[name]; ok {
		return r
	}
	if rs := []rune(name); len(rs) == 1 {
		return rs[0]
	}
	if strings.HasPrefix(name, "uni") && len(name) == 7 {
		if v, err := strconv.ParseUint(name[3:], 16, 32); err == nil {
			return rune(v)
		}
	}
	return 0
}
