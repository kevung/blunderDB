package jbig2

// bitmap is a bi-level image, rows of stride bytes, the most significant
// bit first, 1 for black.
type bitmap struct {
	w, h, stride int
	b            []byte
}

func newBitmap(w, h int) *bitmap {
	s := (w + 7) / 8
	return &bitmap{w: w, h: h, stride: s, b: make([]byte, s*h)}
}

func (m *bitmap) get(x, y int) int {
	if x < 0 || y < 0 || x >= m.w || y >= m.h {
		return 0
	}
	return int(m.b[y*m.stride+x>>3]>>(7-uint(x&7))) & 1
}

func (m *bitmap) set(x, y, v int) {
	i, s := y*m.stride+x>>3, 7-uint(x&7)
	m.b[i] = m.b[i]&^(1<<s) | byte(v&1)<<s
}

func (m *bitmap) fill(v int) {
	f := byte(0)
	if v != 0 {
		f = 0xFF
	}
	for i := range m.b {
		m.b[i] = f
	}
}

// row is row y, nil outside the bitmap.
func (m *bitmap) row(y int) []byte {
	if y < 0 || y >= m.h {
		return nil
	}
	return m.b[y*m.stride : (y+1)*m.stride]
}

// px reads pixel x of a row given by row (nil is white), w pixels wide.
func px(row []byte, x, w int) uint32 {
	if row == nil || x < 0 || x >= w {
		return 0
	}
	return uint32(row[x>>3]>>(7-uint(x&7))) & 1
}

// The combination operators of 7.4.1.5 (region) and 7.4.8.5 (page).
const (
	opOR = iota
	opAND
	opXOR
	opXNOR
	opREPLACE
)

func combine(d, s byte, op int) byte {
	switch op {
	case opAND:
		return d & s
	case opXOR:
		return d ^ s
	case opXNOR:
		return ^(d ^ s)
	case opREPLACE:
		return s
	}
	return d | s
}

// compose draws src with its top-left corner at (x, y), clipped to m.
func (m *bitmap) compose(src *bitmap, x, y, op int) {
	x0, y0 := max(x, 0), max(y, 0)
	x1, y1 := min(x+src.w, m.w), min(y+src.h, m.h)
	if x0 >= x1 || y0 >= y1 {
		return
	}
	if x&7 == 0 && x >= 0 {
		// Byte-aligned: whole bytes, the last one masked to the clip.
		n := (x1 - x0 + 7) / 8
		last := byte(0xFF) << uint(n*8-(x1-x0))
		for yy := y0; yy < y1; yy++ {
			dr := m.b[yy*m.stride+x0>>3:]
			sr := src.b[(yy-y)*src.stride+(x0-x)>>3:]
			for i := 0; i < n; i++ {
				mask := byte(0xFF)
				if i == n-1 {
					mask = last
				}
				dr[i] = dr[i]&^mask | combine(dr[i], sr[i], op)&mask
			}
		}
		return
	}
	for yy := y0; yy < y1; yy++ {
		for xx := x0; xx < x1; xx++ {
			s := byte(src.get(xx-x, yy-y))
			if op == opOR && s == 0 {
				continue
			}
			m.set(xx, yy, int(combine(byte(m.get(xx, yy)), s, op)&1))
		}
	}
}

// crop copies the w×h area of m at (x, y), white outside m.
func (m *bitmap) crop(x, y, w, h int) *bitmap {
	c := newBitmap(w, h)
	for yy := 0; yy < h; yy++ {
		for xx := 0; xx < w; xx++ {
			if m.get(x+xx, y+yy) == 1 {
				c.set(xx, yy, 1)
			}
		}
	}
	return c
}
