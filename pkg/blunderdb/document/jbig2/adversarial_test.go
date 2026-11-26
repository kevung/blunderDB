package jbig2

import (
	"context"
	"encoding/binary"
	"errors"
	"math/rand"
	"strings"
	"testing"
	"time"
)

// An MQ encoder (T.88 E.2), to build adversarial streams in the tests.
type enc struct {
	a, c uint32
	ct   int
	out  []byte // out[0] is a dummy byte before BP start
}

func newEnc() *enc { return &enc{a: 0x8000, ct: 12, out: []byte{0}} }

func (e *enc) byteOut() {
	b := &e.out[len(e.out)-1]
	if *b == 0xFF {
		e.out = append(e.out, byte(e.c>>20))
		e.c &= 0xFFFFF
		e.ct = 7
		return
	}
	if e.c < 0x8000000 {
		e.out = append(e.out, byte(e.c>>19))
		e.c &= 0x7FFFF
		e.ct = 8
		return
	}
	*b++
	if *b == 0xFF {
		e.c &= 0x7FFFFFF
		e.out = append(e.out, byte(e.c>>20))
		e.c &= 0xFFFFF
		e.ct = 7
	} else {
		e.out = append(e.out, byte(e.c>>19))
		e.c &= 0x7FFFF
		e.ct = 8
	}
}

func (e *enc) renorm() {
	for {
		e.a <<= 1
		e.c <<= 1
		e.ct--
		if e.ct == 0 {
			e.byteOut()
		}
		if e.a&0x8000 != 0 {
			break
		}
	}
}

func (e *enc) bit(cx contexts, i int, d int) {
	s := cx[i]
	idx, mps := s>>1, int(s&1)
	st := &mqTable[idx]
	qe := st.qe
	if d == mps {
		e.a -= qe
		if e.a&0x8000 == 0 {
			if e.a < qe {
				e.a = qe
			} else {
				e.c += qe
			}
			idx = st.nmps
			cx[i] = idx<<1 | uint8(mps)
			e.renorm()
		} else {
			e.c += qe
		}
		return
	}
	e.a -= qe
	if e.a < qe {
		e.c += qe
	} else {
		e.a = qe
	}
	if st.sw == 1 {
		mps = 1 - mps
	}
	idx = st.nlps
	cx[i] = idx<<1 | uint8(mps)
	e.renorm()
}

func (e *enc) flush() []byte {
	t := e.c + e.a
	e.c |= 0xFFFF
	if e.c >= t {
		e.c -= 0x8000
	}
	e.c <<= uint(e.ct)
	e.byteOut()
	e.c <<= uint(e.ct)
	e.byteOut()
	if e.out[len(e.out)-1] != 0xFF {
		e.out = append(e.out, 0xFF)
	}
	e.out = append(e.out, 0xAC)
	return e.out[1:]
}

func (e *enc) integer(cx contexts, v int, oob bool) {
	prev := 1
	put := func(n int, x uint64) {
		for i := n - 1; i >= 0; i-- {
			b := int(x>>uint(i)) & 1
			e.bit(cx, prev, b)
			if prev < 256 {
				prev = prev<<1 | b
			} else {
				prev = (prev<<1|b)&511 | 256
			}
		}
	}
	s := 0
	if oob {
		put(1, 1)
		put(1, 0)
		put(2, 0)
		return
	}
	if v < 0 {
		s, v = 1, -v
	}
	put(1, uint64(s))
	switch {
	case v < 4:
		put(1, 0)
		put(2, uint64(v))
	case v < 20:
		put(2, 2)
		put(4, uint64(v-4))
	case v < 84:
		put(3, 6)
		put(6, uint64(v-20))
	case v < 340:
		put(4, 14)
		put(8, uint64(v-84))
	case v < 4436:
		put(5, 30)
		put(12, uint64(v-340))
	default:
		put(5, 31)
		put(32, uint64(v-4436))
	}
}

func hdr(num uint32, typ int, refs []byte, data []byte) []byte {
	b := binary.BigEndian.AppendUint32(nil, num)
	b = append(b, byte(typ), byte(len(refs)<<5))
	b = append(b, refs...)
	b = append(b, 1)
	b = binary.BigEndian.AppendUint32(b, uint32(len(data)))
	return append(b, data...)
}

func pinfo(w, h uint32) []byte {
	b := binary.BigEndian.AppendUint32(nil, w)
	b = binary.BigEndian.AppendUint32(b, h)
	return append(b, make([]byte, 11)...)
}

func rinfo(w, h, x, y uint32, op byte) []byte {
	b := binary.BigEndian.AppendUint32(nil, w)
	b = binary.BigEndian.AppendUint32(b, h)
	b = binary.BigEndian.AppendUint32(b, x)
	b = binary.BigEndian.AppendUint32(b, y)
	return append(b, op)
}

var nom0 = []byte{3, 0xff, 0xfd, 0xff, 2, 0xfe, 0xfe, 0xfe}

func sdHead(nexp, nnew uint32) []byte {
	b := binary.BigEndian.AppendUint16(nil, 0)
	b = append(b, nom0...)
	b = binary.BigEndian.AppendUint32(b, nexp)
	return binary.BigEndian.AppendUint32(b, nnew)
}

func ctx512() contexts { return make(contexts, 512) }

// emptySymbols codes a dictionary of n empty symbols, all exported.
func emptySymbols(n int) []byte {
	e := newEnc()
	dh, dw, ex := ctx512(), ctx512(), ctx512()
	e.integer(dh, 0, false)
	for i := 0; i < n; i++ {
		e.integer(dw, 0, false)
	}
	e.integer(dw, 0, true)
	e.integer(ex, 0, false)
	e.integer(ex, n, false)
	return append(sdHead(uint32(n), uint32(n)), e.flush()...)
}

// TestReexports: dictionaries re-exporting another's symbols hold them
// again; the stream bounds them together, before decoding.
func TestReexports(t *testing.T) {
	const n = 1 << 19
	e := newEnc()
	ex := ctx512()
	e.integer(ex, 0, false)
	e.integer(ex, n, false)
	again := append(sdHead(n, 0), e.flush()...)
	data := hdr(0, segPageInfo, nil, pinfo(1, 1))
	data = append(data, hdr(1, segSymbolDict, nil, emptySymbols(n))...)
	for k := 2; k < 200; k++ {
		data = append(data, hdr(uint32(k), segSymbolDict, []byte{1}, again)...)
	}
	if _, err := Inspect(data, nil, 0, 0); !errors.Is(err, ErrUnsupported) || !strings.Contains(err.Error(), "together") {
		t.Errorf("re-exports: %v", err)
	}
	// One dictionary of empty symbols costs minCost a symbol, and as much
	// again for its exports.
	data = append(hdr(0, segPageInfo, nil, pinfo(1, 1)), hdr(1, segSymbolDict, nil, emptySymbols(1000))...)
	b := &Budget{Left: 1 << 30}
	if _, _, err := Decode(context.Background(), data, nil, 0, 0, b); err != nil || 1<<30-b.Left != 1+2*1000*minCost {
		t.Errorf("empty symbols: %v, spent %d", err, 1<<30-b.Left)
	}
}

// bigSymbolText is a page s×s, a dictionary of one black-free symbol s×s,
// and a text region XOR-placing it n times at x = 1, 2, 3…
func bigSymbolText(s, n int) []byte {
	e := newEnc()
	dh, dw, ex := ctx512(), ctx512(), ctx512()
	gb := make(contexts, genericCX[0])
	e.integer(dh, s, false)
	e.integer(dw, s, false)
	for i := 0; i < s*s; i++ {
		e.bit(gb, 0, 0)
	}
	e.integer(dw, 0, true)
	e.integer(ex, 0, false)
	e.integer(ex, 1, false)
	d1 := e.flush()
	e = newEnc()
	dt, fs, ds := ctx512(), ctx512(), ctx512()
	e.integer(dt, 0, false)
	e.integer(dt, 0, false)
	e.integer(fs, 1, false)
	for k := 1; k < n; k++ {
		e.integer(ds, -(s - 1), false)
	}
	e.integer(ds, 0, true)
	tb := rinfo(uint32(s), uint32(s), 0, 0, 0)
	tb = binary.BigEndian.AppendUint16(tb, 1<<4|2<<7)
	tb = binary.BigEndian.AppendUint32(tb, uint32(n))
	data := hdr(0, segPageInfo, nil, pinfo(uint32(s), uint32(s)))
	data = append(data, hdr(1, segSymbolDict, nil, append(sdHead(1, 1), d1...))...)
	return append(data, hdr(2, segText, []byte{1}, append(tb, e.flush()...))...)
}

// TestTextContext: a text region composing large symbols reads the context
// as it composes, not once every so many symbols.
func TestTextContext(t *testing.T) {
	data := bigSymbolText(2048, 1<<12)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	t0 := time.Now()
	_, _, err := Decode(ctx, data, nil, 0, 0, nil)
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(t0) > time.Duration(slack())*time.Second {
		t.Errorf("%v after %v", err, time.Since(t0))
	}
}

// TestAggregation: the symbols an aggregation places count with the
// stream's, at minCost each at least.
func TestAggregation(t *testing.T) {
	agg := func(ni int) []byte {
		e := newEnc()
		dh, dw, ex := ctx512(), ctx512(), ctx512()
		ai, dt, fs, ds, ri := ctx512(), ctx512(), ctx512(), ctx512(), ctx512()
		id := make(contexts, 4)
		e.integer(dh, 1, false)
		e.integer(dw, 1, false)
		e.integer(ai, ni, false)
		e.integer(dt, 0, false)
		e.integer(dt, 0, false)
		e.integer(fs, 0, false)
		for k := 0; k < ni; k++ {
			e.bit(id, 1, 0)
			e.integer(ri, 0, false)
			e.integer(ds, 0, k == ni-1)
		}
		e.integer(dw, 0, true)
		e.integer(ex, 1, false)
		e.integer(ex, 1, false)
		h := binary.BigEndian.AppendUint16(nil, 2|1<<12)
		h = append(h, nom0...)
		h = binary.BigEndian.AppendUint32(h, 1)
		h = binary.BigEndian.AppendUint32(h, 1)
		data := hdr(0, segPageInfo, nil, pinfo(1, 1))
		data = append(data, hdr(1, segSymbolDict, nil, emptySymbols(1))...)
		return append(data, hdr(2, segSymbolDict, []byte{1}, append(h, e.flush()...))...)
	}
	if _, _, err := Decode(context.Background(), agg(maxInstances+1), nil, 0, 0, nil); !errors.Is(err, ErrUnsupported) {
		t.Errorf("over the bound: %v", err)
	}
	b := &Budget{Left: 1 << 30}
	if _, _, err := Decode(context.Background(), agg(1000), nil, 0, 0, b); err != nil || 1<<30-b.Left < 1000*minCost {
		t.Errorf("1000 instances: %v, spent %d", err, 1<<30-b.Left)
	}
}

// TestRefineSelfReference: a refinement without a reference spends the
// copy of the page it refines as well as its own pixels.
func TestRefineSelfReference(t *testing.T) {
	data := append(hdr(0, segPageInfo, nil, pinfo(64, 64)), hdr(1, segRefine, nil, append(rinfo(64, 64, 0, 0, 0), 1))...)
	b := &Budget{Left: 1 << 20}
	if _, _, err := Decode(context.Background(), data, nil, 0, 0, b); err != nil || 1<<20-b.Left != 3*64*64 {
		t.Errorf("%v, spent %d", err, 1<<20-b.Left)
	}
}

// TestRefineSLTP: the typical-prediction context is the one where only the
// centre of the reference is black, for both templates.
func TestRefineSLTP(t *testing.T) {
	ref := newBitmap(3, 3)
	ref.set(1, 1, 1)
	for tm := 0; tm < 2; tm++ {
		p := refineParams{w: 3, h: 3, template: tm, ref: ref, at: [2][2]int{{-1, -1}, {-1, -1}}}
		if got := refineContext(newBitmap(3, 3), 1, 1, &p); got != refineSLTP[tm] {
			t.Errorf("template %d: %#x, want %#x", tm, refineSLTP[tm], got)
		}
	}
}

// TestCompose: a composition at any x and with any operator gives what the
// pixels one by one give.
func TestCompose(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 500; i++ {
		dst, src := newBitmap(1+rng.Intn(40), 1+rng.Intn(6)), newBitmap(1+rng.Intn(30), 1+rng.Intn(6))
		rng.Read(dst.b)
		rng.Read(src.b)
		x, y, op := rng.Intn(60)-25, rng.Intn(10)-4, rng.Intn(5)
		want := newBitmap(dst.w, dst.h)
		copy(want.b, dst.b)
		for yy := 0; yy < dst.h; yy++ {
			for xx := 0; xx < dst.w; xx++ {
				if sx, sy := xx-x, yy-y; sx >= 0 && sy >= 0 && sx < src.w && sy < src.h {
					want.set(xx, yy, int(combine(byte(dst.get(xx, yy)), byte(src.get(sx, sy)), op)&1))
				}
			}
		}
		dst.compose(src, x, y, op)
		if string(dst.b) != string(want.b) {
			t.Fatalf("x %d y %d op %d: differs", x, y, op)
		}
	}
}

// TestTextRegionsDrawingFromMany: text regions placing nothing from a large
// dictionary spend what building their symbols and contexts costs, so their
// number is bounded by the budget, not by the stream's length.
func TestTextRegionsDrawingFromMany(t *testing.T) {
	const n = 1 << 16
	data := hdr(0, segPageInfo, nil, pinfo(1, 1))
	data = append(data, hdr(1, segSymbolDict, nil, emptySymbols(n))...)
	const k = 100
	for i := uint32(2); i < 2+k; i++ {
		tb := rinfo(0, 0, 0, 0, 0)
		tb = binary.BigEndian.AppendUint16(tb, 1<<4)
		tb = binary.BigEndian.AppendUint32(tb, 0)
		data = append(data, hdr(i, segText, []byte{1}, tb)...)
	}
	region := int64(minCost*n + 2<<16)
	b := &Budget{Left: 1 << 30}
	if _, _, err := Decode(context.Background(), data, nil, 0, 0, b); err != nil || 1<<30-b.Left < k*region {
		t.Errorf("%v, spent %d, want %d at least", err, 1<<30-b.Left, k*region)
	}
	b = &Budget{Left: 1 << 26}
	if _, _, err := Decode(context.Background(), data, nil, 0, 0, b); !errors.Is(err, ErrBudget) {
		t.Errorf("a smaller budget: %v", err)
	}
}
