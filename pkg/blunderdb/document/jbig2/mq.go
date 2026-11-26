package jbig2

// The MQ arithmetic decoder of T.88 Annex E (the one JPEG 2000 also uses),
// and the integer and symbol-ID decoders built on it (Annex A).

type mqState struct {
	qe         uint32
	nmps, nlps uint8
	sw         uint8
}

var mqTable = [47]mqState{
	{0x5601, 1, 1, 1}, {0x3401, 2, 6, 0}, {0x1801, 3, 9, 0},
	{0x0AC1, 4, 12, 0}, {0x0521, 5, 29, 0}, {0x0221, 38, 33, 0},
	{0x5601, 7, 6, 1}, {0x5401, 8, 14, 0}, {0x4801, 9, 14, 0},
	{0x3801, 10, 14, 0}, {0x3001, 11, 17, 0}, {0x2401, 12, 18, 0},
	{0x1C01, 13, 20, 0}, {0x1601, 29, 21, 0}, {0x5601, 15, 14, 1},
	{0x5401, 16, 14, 0}, {0x5101, 17, 15, 0}, {0x4801, 18, 16, 0},
	{0x3801, 19, 17, 0}, {0x3401, 20, 18, 0}, {0x3001, 21, 19, 0},
	{0x2801, 22, 19, 0}, {0x2401, 23, 20, 0}, {0x2201, 24, 21, 0},
	{0x1C01, 25, 22, 0}, {0x1801, 26, 23, 0}, {0x1601, 27, 24, 0},
	{0x1401, 28, 25, 0}, {0x1201, 29, 26, 0}, {0x1101, 30, 27, 0},
	{0x0AC1, 31, 28, 0}, {0x09C1, 32, 29, 0}, {0x08A1, 33, 30, 0},
	{0x0521, 34, 31, 0}, {0x0441, 35, 32, 0}, {0x02A1, 36, 33, 0},
	{0x0221, 37, 34, 0}, {0x0141, 38, 35, 0}, {0x0111, 39, 36, 0},
	{0x0085, 40, 37, 0}, {0x0049, 41, 38, 0}, {0x0025, 42, 39, 0},
	{0x0015, 43, 40, 0}, {0x0009, 44, 41, 0}, {0x0005, 45, 42, 0},
	{0x0001, 45, 43, 0}, {0x5601, 46, 46, 0},
}

// A context is its state index shifted left once, ored with its MPS.
type contexts []uint8

type mq struct {
	data  []byte
	bp    int
	chigh uint32
	clow  uint32
	a     uint32
	ct    int
}

// at reads past the end as 0xFF, which feeds 1-bits without consuming: a
// truncated segment decodes to something, bounded by the counts around it.
func (d *mq) at(i int) uint32 {
	if i < len(d.data) {
		return uint32(d.data[i])
	}
	return 0xFF
}

func newMQ(data []byte) *mq {
	d := &mq{data: data, chigh: 0}
	d.chigh = d.at(0)
	d.byteIn()
	d.chigh = (d.chigh<<7)&0xffff | (d.clow>>9)&0x7f
	d.clow = (d.clow << 7) & 0xffff
	d.ct -= 7
	d.a = 0x8000
	return d
}

func (d *mq) byteIn() {
	if d.at(d.bp) == 0xFF {
		if d.at(d.bp+1) > 0x8F {
			d.clow += 0xFF00
			d.ct = 8
		} else {
			d.bp++
			d.clow += d.at(d.bp) << 9
			d.ct = 7
		}
	} else {
		d.bp++
		d.clow += d.at(d.bp) << 8
		d.ct = 8
	}
	if d.clow > 0xffff {
		d.chigh += d.clow >> 16
		d.clow &= 0xffff
	}
}

func (d *mq) bit(cx contexts, i int) int {
	s := cx[i]
	idx, mps := s>>1, int(s&1)
	st := &mqTable[idx]
	qe := st.qe
	a := d.a - qe
	var v int
	if d.chigh < qe {
		if a < qe {
			a = qe
			v = mps
			idx = st.nmps
		} else {
			a = qe
			v = 1 ^ mps
			if st.sw == 1 {
				mps = v
			}
			idx = st.nlps
		}
	} else {
		d.chigh -= qe
		if a&0x8000 != 0 {
			d.a = a
			return mps
		}
		if a < qe {
			v = 1 ^ mps
			if st.sw == 1 {
				mps = v
			}
			idx = st.nlps
		} else {
			v = mps
			idx = st.nmps
		}
	}
	for {
		if d.ct == 0 {
			d.byteIn()
		}
		a <<= 1
		d.chigh = (d.chigh<<1)&0xffff | (d.clow>>15)&1
		d.clow = (d.clow << 1) & 0xffff
		d.ct--
		if a&0x8000 != 0 {
			break
		}
	}
	d.a = a
	cx[i] = idx<<1 | uint8(mps)
	return v
}

// integer decodes one value with the 512 contexts of an IAx procedure
// (A.2); oob is the out-of-band value, a negative zero.
func (d *mq) integer(cx contexts) (v int, oob bool) {
	prev := 1
	read := func(n int) int {
		v := 0
		for i := 0; i < n; i++ {
			b := d.bit(cx, prev)
			if prev < 256 {
				prev = prev<<1 | b
			} else {
				prev = (prev<<1|b)&511 | 256
			}
			v = v<<1 | b
		}
		return v
	}
	s := read(1)
	switch {
	case read(1) == 0:
		v = read(2)
	case read(1) == 0:
		v = read(4) + 4
	case read(1) == 0:
		v = read(6) + 20
	case read(1) == 0:
		v = read(8) + 84
	case read(1) == 0:
		v = read(12) + 340
	default:
		v = read(32) + 4436
	}
	if s == 1 {
		if v == 0 {
			return 0, true
		}
		return -v, false
	}
	return v, false
}

// id decodes a symbol ID of n bits (A.3) with 1<<(n+1) contexts.
func (d *mq) id(cx contexts, n int) int {
	prev := 1
	for i := 0; i < n; i++ {
		prev = prev<<1 | d.bit(cx, prev)
	}
	return prev - 1<<n
}
