package jpx

import (
	"bytes"
	"context"
	"errors"
	"image"
	"runtime"
	"testing"
	"time"
)

// The codestreams below are the ones a second header reader, written apart
// from the decoder, let through to it: each declares, where only the
// decoder looks, a coding style that costs gigabytes. One parser reads them
// now; they are refused, or decoded as the headers it read say.

func (b *cs) siz(w, h, tw, th int, comps []comp) {
	b.u16(0xFF51)
	b.u16(38 + 3*len(comps))
	b.u16(0)
	for _, v := range []int{w, h, 0, 0, tw, th, 0, 0} {
		b.u32(v)
	}
	b.u16(len(comps))
	for _, c := range comps {
		b.WriteByte(7)
		b.WriteByte(byte(c.dx))
		b.WriteByte(byte(c.dy))
	}
}

func (b *cs) cod(levels, cbw, cbh int, prec []byte) {
	b.u16(0xFF52)
	b.u16(12 + len(prec))
	if prec != nil {
		b.WriteByte(1)
	} else {
		b.WriteByte(0)
	}
	b.WriteByte(0)
	b.u16(1)
	b.WriteByte(0)
	b.WriteByte(byte(levels))
	b.WriteByte(byte(cbw))
	b.WriteByte(byte(cbh))
	b.WriteByte(0)
	b.WriteByte(1)
	b.Write(prec)
}

func (b *cs) qcd(levels, eps int) {
	n := 1 + 3*levels
	b.u16(0xFF5C)
	b.u16(3 + n)
	b.WriteByte(2 << 5)
	for i := 0; i < n; i++ {
		b.WriteByte(byte(eps << 3))
	}
}

func (b *cs) tile(idx int, body []byte) {
	b.u16(0xFF90)
	b.u16(10)
	b.u16(idx)
	b.u32(14 + len(body))
	b.WriteByte(0)
	b.WriteByte(0)
	b.u16(0xFF93)
	b.Write(body)
}

func grey() []comp { return []comp{{1, 1}} }

// hostile builds the codestream of a bypass for an a×a image.
func hostile(kind string, a int) []byte {
	var b cs
	b.u16(0xFF4F)
	switch kind {
	case "sodprec": // a SOD in the main header, then precincts of 1×1
		b.siz(a, a, a, a, grey())
		b.cod(0, 4, 4, nil)
		b.qcd(0, 8)
		b.u16(0xFF93)
		b.cod(0, 4, 4, []byte{0x00})
		b.tile(0, make([]byte, 16))
	case "sodcomps": // a SOD, then a SIZ of 16384 components and 1×1 tiles
		b.siz(64, 64, 64, 64, grey())
		b.cod(0, 4, 4, nil)
		b.qcd(0, 8)
		b.u16(0xFF93)
		many := make([]comp, 16384)
		for i := range many {
			many[i] = comp{1, 1}
		}
		b.siz(256, 256, 1, 1, many)
		for i := 0; i < 16; i++ {
			b.tile(i, nil)
		}
	case "siz2dx": // a second SIZ, subsampled by 255
		b.siz(a, a, a, a, grey())
		b.siz(a, a, a, a, []comp{{255, 255}})
		b.cod(0, 4, 4, nil)
		b.qcd(0, 8)
		b.tile(0, nil)
	case "ff30": // a reserved marker read with and without a length
		b.siz(a, a, a, a, grey())
		b.cod(0, 4, 4, nil)
		b.qcd(0, 8)
		p := b.Len()
		b.u16(0xFF30)
		b.u16(0xFF64) // a length of 0xFF64, or a COM marker
		var evil cs
		evil.cod(0, 4, 4, []byte{0x00})
		next := p + 2 + 0xFF64
		b.u16(next + evil.Len() - (p + 4)) // the COM covers the evil COD
		for b.Len() < next {
			b.WriteByte(0)
		}
		b.Write(evil.Bytes())
		b.tile(0, make([]byte, 16))
	case "minprec": // the smallest precincts and code-blocks taken
		b.siz(a, a, a, a, []comp{{1, 1}, {1, 1}, {1, 1}})
		b.cod(5, 2, 2, bytes.Repeat([]byte{0x55}, 6))
		b.qcd(5, 8)
		b.tile(0, make([]byte, 16))
	case "cpu": // every code-block coded with as many passes as it may hold
		b.siz(a, a, a, a, grey())
		b.cod(0, 4, 4, nil)
		b.qcd(0, 30)
		b.tile(0, cpuBody(a))
	}
	b.u16(0xFFD9)
	return b.Bytes()
}

// allocated returns the bytes f allocates.
func allocated(f func()) uint64 {
	var m0, m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m0)
	f()
	runtime.ReadMemStats(&m1)
	return m1.TotalAlloc - m0.TotalAlloc
}

func TestHeaderBypasses(t *testing.T) {
	const a = 1024
	for _, kind := range []string{"sodprec", "sodcomps", "siz2dx"} {
		_, _, err := Decode(context.Background(), hostile(kind, a), 0, 0, &Budget{Left: 1 << 40})
		if !errors.Is(err, ErrMalformed) {
			t.Errorf("%s: %v", kind, err)
		}
	}
	// A reserved marker carries no length: the COM that follows covers the
	// precincts of 1×1, which are never read.
	var err error
	n := allocated(func() {
		_, _, err = Decode(context.Background(), hostile("ff30", a), 0, 0, &Budget{Left: 1 << 40})
	})
	if errors.Is(err, ErrUnsupported) || n > 64<<20 {
		t.Errorf("ff30: %v, %d MiB allocated", err, n>>20)
	}
}

// TestSmallestPrecincts: an image of the smallest precincts and code-blocks
// taken costs memory in proportion to its pixels.
func TestSmallestPrecincts(t *testing.T) {
	const a = 2048
	n := allocated(func() {
		_, _, _ = Decode(context.Background(), hostile("minprec", a), 0, 0, nil)
	})
	if perPixel := n / (a * a); perPixel > 40 {
		t.Errorf("%d bytes a pixel", perPixel)
	}
}

// TestWorstCaseStops: the coded data no header bounds is stopped by the
// context, between code-blocks, and refused by the budget before it starts.
func TestWorstCaseStops(t *testing.T) {
	const a = 1024
	data := hostile("cpu", a)
	if _, _, err := Decode(context.Background(), data, 0, 0, &Budget{Left: a*a - 1}); !errors.Is(err, ErrBudget) {
		t.Errorf("budget: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, _, err := Decode(ctx, data, 0, 0, nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("deadline: %v", err)
	}
	if d := time.Since(start); d > time.Second {
		t.Errorf("stopped after %v", d)
	}
}

// BenchmarkWorstCase reports the cost a pixel of the coded data that costs
// the most: the budgets' comments state it (about 1.2 µs).
func BenchmarkWorstCase(b *testing.B) {
	const a = 512
	data := hostile("cpu", a)
	for b.Loop() {
		if _, _, err := Decode(context.Background(), data, 0, 0, nil); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/(a*a), "ns/pixel")
}

// TestNotRegistered: no JPEG 2000 decoder is registered with package image,
// so a JP2 file under another name reaches none.
func TestNotRegistered(t *testing.T) {
	for _, data := range [][]byte{hostile("cpu", 64), append(append([]byte{}, jp2Sig...), 0, 0, 0, 0)} {
		if _, format, err := image.DecodeConfig(bytes.NewReader(data)); !errors.Is(err, image.ErrFormat) {
			t.Errorf("decoded as %q: %v", format, err)
		}
	}
}

var jp2Sig = []byte{0, 0, 0, 12, 'j', 'P', ' ', ' ', 0x0d, 0x0a, 0x87, 0x0a}

// cpuBody codes one layer in which every 64×64 code-block of an a×a tile
// is included with the most coding passes its bit-planes allow.
func cpuBody(a int) []byte {
	n := (a + 63) / 64
	incl, zb := newTagTree(n, n), newTagTree(n, n)
	w := &bitWriter{}
	w.bit(1)
	for i := 0; i < n*n; i++ {
		incl.encode(w, i)
		zb.encode(w, i)
		w.bits(0b1111, 4)
		w.bits(31, 5)
		w.bits(54, 7)
		w.bit(0)
		w.bits(0, 3+7-1)
	}
	return w.flush()
}

// bitWriter writes a packet header, with its bit stuffing after 0xFF.
type bitWriter struct {
	out    []byte
	cur    byte
	n, max int
}

func (w *bitWriter) bit(v int) {
	if w.max == 0 {
		w.max = 8
	}
	w.cur = w.cur<<1 | byte(v&1)
	w.n++
	if w.n == w.max {
		w.out = append(w.out, w.cur)
		w.max = 8
		if w.cur == 0xFF {
			w.max = 7
		}
		w.cur, w.n = 0, 0
	}
}

func (w *bitWriter) bits(v, n int) {
	for i := n - 1; i >= 0; i-- {
		w.bit(v >> i)
	}
}

func (w *bitWriter) flush() []byte {
	for w.n != 0 {
		w.bit(0)
	}
	if len(w.out) > 0 && w.out[len(w.out)-1] == 0xFF {
		w.out = append(w.out, 0)
	}
	return w.out
}

// tagTree encodes a tag tree whose values are all zero.
type tagTree struct {
	parent []int
	known  []bool
}

func newTagTree(w, h int) *tagTree {
	type level struct{ w, h, off int }
	var ls []level
	off := 0
	for lw, lh := w, h; ; lw, lh = (lw+1)/2, (lh+1)/2 {
		ls = append(ls, level{lw, lh, off})
		off += lw * lh
		if lw == 1 && lh == 1 {
			break
		}
	}
	t := &tagTree{parent: make([]int, off), known: make([]bool, off)}
	for i := range t.parent {
		t.parent[i] = -1
	}
	for li := 0; li < len(ls)-1; li++ {
		c, p := ls[li], ls[li+1]
		for y := 0; y < c.h; y++ {
			for x := 0; x < c.w; x++ {
				t.parent[c.off+y*c.w+x] = p.off + (y/2)*p.w + x/2
			}
		}
	}
	return t
}

func (t *tagTree) encode(w *bitWriter, leaf int) {
	var path []int
	for i := leaf; i >= 0; i = t.parent[i] {
		path = append(path, i)
	}
	for i := len(path) - 1; i >= 0; i-- {
		if !t.known[path[i]] {
			w.bit(1)
			t.known[path[i]] = true
		}
	}
}
