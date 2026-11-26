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

// compose draws src with its top-left corner at (x, y), clipped to m, a
// byte of m at a time: the eight source bits under it are gathered by
// shifting, whatever x is.
func (m *bitmap) compose(src *bitmap, x, y, op int) {
	x0, y0 := max(x, 0), max(y, 0)
	x1, y1 := min(x+src.w, m.w), min(y+src.h, m.h)
	if x0 >= x1 || y0 >= y1 {
		return
	}
	j0, j1 := x0>>3, (x1-1)>>3
	for yy := y0; yy < y1; yy++ {
		dr := m.b[yy*m.stride : (yy+1)*m.stride]
		sr := src.b[(yy-y)*src.stride : (yy-y+1)*src.stride]
		for j := j0; j <= j1; j++ {
			mask := byte(0xFF)
			if j == j0 {
				mask &= 0xFF >> uint(x0&7)
			}
			if j == j1 {
				mask &= 0xFF << uint(7-(x1-1)&7)
			}
			dr[j] = dr[j]&^mask | combine(dr[j], bits8(sr, j*8-x), op)&mask
		}
	}
}

// bits8 is the eight bits of row r from bit b on (b may be negative), 0
// outside the row.
func bits8(r []byte, b int) byte {
	if b < 0 {
		if b <= -8 {
			return 0
		}
		return r[0] >> uint(-b)
	}
	q, k := b>>3, uint(b&7)
	var v byte
	if q < len(r) {
		v = r[q] << k
	}
	if k != 0 && q+1 < len(r) {
		v |= r[q+1] >> (8 - k)
	}
	return v
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
