package arkqr

import "math/bits"

// An integer arithmetic coder on 48-bit registers. Every quantity is an
// integer and every product is taken on 128 bits, so a stream decodes the
// same on every platform.
const (
	precision = 48
	top       = uint64(1)<<precision - 1
	halfRange = uint64(1) << (precision - 1)
	quarter   = uint64(1) << (precision - 2)
)

// mulDiv is a·b/c without overflow; the quotient always fits, as b ≤ c.
func mulDiv(a, b, c uint64) uint64 {
	hi, lo := bits.Mul64(a, b)
	q, _ := bits.Div64(hi, lo, c)
	return q
}

// coder is the one interface the encoder and the decoder share, so that the
// format is written once: sym codes symbol v under the frequencies fr (the
// encoder) or returns the symbol it reads (the decoder, which ignores v).
type coder interface {
	sym(fr []uint64, v int) int
	decoding() bool
}

type encoder struct {
	lo, hi uint64
	pend   int
	out    []byte
	acc    byte
	nbits  int
}

func newEncoder() *encoder { return &encoder{hi: top} }

func (e *encoder) decoding() bool { return false }

func (e *encoder) bit(b uint8) {
	e.acc = e.acc<<1 | b
	e.nbits++
	if e.nbits == 8 {
		e.out = append(e.out, e.acc)
		e.acc, e.nbits = 0, 0
	}
}

func (e *encoder) emit(b uint8) {
	e.bit(b)
	for ; e.pend > 0; e.pend-- {
		e.bit(1 - b)
	}
}

func (e *encoder) sym(fr []uint64, v int) int {
	var cl, tot uint64
	for i, f := range fr {
		if i < v {
			cl += f
		}
		tot += f
	}
	ch := cl + fr[v]
	r := e.hi - e.lo + 1
	e.hi = e.lo + mulDiv(r, ch, tot) - 1
	e.lo += mulDiv(r, cl, tot)
	for {
		switch {
		case e.hi < halfRange:
			e.emit(0)
		case e.lo >= halfRange:
			e.emit(1)
			e.lo -= halfRange
			e.hi -= halfRange
		case e.lo >= quarter && e.hi < 3*quarter:
			e.pend++
			e.lo -= quarter
			e.hi -= quarter
		default:
			return v
		}
		e.lo <<= 1
		e.hi = e.hi<<1 | 1
	}
}

// finish flushes the stream, padded to a whole byte with zeros.
func (e *encoder) finish() []byte {
	e.pend++
	if e.lo < quarter {
		e.emit(0)
	} else {
		e.emit(1)
	}
	for e.nbits != 0 {
		e.bit(0)
	}
	// Trailing zero bytes carry nothing: the decoder reads zeros past the end.
	for len(e.out) > 0 && e.out[len(e.out)-1] == 0 {
		e.out = e.out[:len(e.out)-1]
	}
	return e.out
}

type decoder struct {
	lo, hi, val uint64
	data        []byte
	pos         int
	// budget bounds the symbols a damaged stream can make the decoder read.
	budget int
	failed bool
}

func newDecoder(data []byte) *decoder {
	d := &decoder{hi: top, data: data, budget: 64 + 64*len(data)}
	for range precision {
		d.val = d.val<<1 | d.in()
	}
	return d
}

func (d *decoder) decoding() bool { return true }

func (d *decoder) in() uint64 {
	var b uint64
	if i := d.pos >> 3; i < len(d.data) {
		b = uint64(d.data[i]>>(7-d.pos&7)) & 1
	}
	d.pos++
	return b
}

func (d *decoder) sym(fr []uint64, _ int) int {
	d.budget--
	if d.budget < 0 {
		d.failed = true
	}
	var tot uint64
	for _, f := range fr {
		tot += f
	}
	r := d.hi - d.lo + 1
	off := d.val - d.lo + 1
	h, l := bits.Mul64(off, tot)
	if l == 0 {
		h--
	}
	l--
	t, _ := bits.Div64(h, l, r)
	var cl uint64
	v := 0
	for ; v < len(fr)-1; v++ {
		if cl+fr[v] > t {
			break
		}
		cl += fr[v]
	}
	ch := cl + fr[v]
	d.hi = d.lo + mulDiv(r, ch, tot) - 1
	d.lo += mulDiv(r, cl, tot)
	for {
		switch {
		case d.hi < halfRange:
		case d.lo >= halfRange:
			d.lo -= halfRange
			d.hi -= halfRange
			d.val -= halfRange
		case d.lo >= quarter && d.hi < 3*quarter:
			d.lo -= quarter
			d.hi -= quarter
			d.val -= quarter
		default:
			return v
		}
		d.lo <<= 1
		d.hi = d.hi<<1 | 1
		d.val = d.val<<1 | d.in()
	}
}

var fair = []uint64{1, 1}

// raw codes the k low bits of v, most significant first.
func raw(c coder, k int, v uint64) uint64 {
	var x uint64
	for i := k - 1; i >= 0; i-- {
		x = x<<1 | uint64(c.sym(fair, int(v>>uint(i)&1)))
	}
	return x
}

// uvar codes an unsigned integer in an Elias-gamma form: its bit length, then
// its bits below the leading one.
func uvar(c coder, v uint64) uint64 {
	x := v + 1
	n := uint64(bits.Len64(x) - 1)
	n = raw(c, 6, n)
	return (uint64(1)<<n | raw(c, int(n), x)) - 1
}

// svar codes a signed integer, zigzagged.
func svar(c coder, v int64) int64 {
	z := uvar(c, uint64(v<<1^(v>>63)))
	return int64(z>>1) ^ -int64(z&1)
}

// flag codes a fixed-probability bit: freq0 against freq1.
func flag(c coder, freq0, freq1 uint64, v bool) bool {
	b := 0
	if v {
		b = 1
	}
	return c.sym([]uint64{freq0, freq1}, b) == 1
}

// adaptive is a bit whose probability follows the bits already coded with it.
type adaptive struct{ n [2]uint64 }

func newAdaptive(n0, n1 uint64) *adaptive { return &adaptive{[2]uint64{n0, n1}} }

func (a *adaptive) code(c coder, v bool) bool {
	b := 0
	if v {
		b = 1
	}
	b = c.sym(a.n[:], b)
	a.n[b] += 16
	if a.n[0]+a.n[1] > 1<<16 {
		a.n[0] = (a.n[0] + 1) / 2
		a.n[1] = (a.n[1] + 1) / 2
	}
	return b == 1
}
