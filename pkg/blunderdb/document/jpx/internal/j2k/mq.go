package j2k

import "math/bits"

// MQ arithmetic decoder as specified in ISO/IEC 15444-1 Annex C.

// mqState is one row of the probability state transition table
// (Table C.2): the LPS probability estimate Qe, the next states after
// an MPS or LPS renormalization, and whether the MPS sense switches.
type mqState struct {
	qe         uint16
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

// mqContext is the adaptive state of one coding context: an index into
// mqTable plus the current most-probable-symbol sense.
type mqContext struct {
	index uint8
	mps   uint8
}

type mqDecoder struct {
	data []byte
	bp   int
	c    uint32
	a    uint32
	ct   int
}

func (d *mqDecoder) byteAt(i int) uint32 {
	if i < len(d.data) {
		return uint32(d.data[i])
	}
	// Past the end of the segment the decoder behaves as if it read
	// 0xFF 0xFF..., which makes bytein insert 1-bits without consuming.
	return 0xFF
}

func (d *mqDecoder) init(data []byte) {
	d.data = data
	d.bp = 0
	d.c = d.byteAt(0) << 16
	d.bytein()
	d.c <<= 7
	d.ct -= 7
	d.a = 0x8000
}

func (d *mqDecoder) bytein() {
	if d.byteAt(d.bp) == 0xFF {
		if d.byteAt(d.bp+1) > 0x8F {
			d.c += 0xFF00
			d.ct = 8
		} else {
			d.bp++
			d.c += d.byteAt(d.bp) << 9
			d.ct = 7
		}
	} else {
		d.bp++
		d.c += d.byteAt(d.bp) << 8
		d.ct = 8
	}
}

// decode returns the next binary decision for the given context.
func (d *mqDecoder) decode(cx *mqContext) uint8 {
	st := &mqTable[cx.index]
	qe := uint32(st.qe)
	d.a -= qe

	var bit uint8
	if (d.c >> 16) < qe {
		// LPS exchange path.
		if d.a < qe {
			bit = cx.mps
			cx.index = st.nmps
		} else {
			bit = 1 - cx.mps
			if st.sw == 1 {
				cx.mps = 1 - cx.mps
			}
			cx.index = st.nlps
		}
		d.a = qe
		d.renorm()
	} else {
		d.c -= qe << 16
		if d.a&0x8000 == 0 {
			// MPS exchange path.
			if d.a < qe {
				bit = 1 - cx.mps
				if st.sw == 1 {
					cx.mps = 1 - cx.mps
				}
				cx.index = st.nlps
			} else {
				bit = cx.mps
				cx.index = st.nmps
			}
			d.renorm()
		} else {
			bit = cx.mps
		}
	}
	return bit
}

// renorm shifts until a's bit 15 is set, as many bits at once as the byte
// read holds, reading the next byte where the bit-by-bit loop of C.3.3
// reads it.
func (d *mqDecoder) renorm() {
	n := bits.LeadingZeros32(d.a) - 16
	for n > 0 {
		if d.ct == 0 {
			d.bytein()
		}
		t := min(n, d.ct)
		d.a <<= uint(t)
		d.c <<= uint(t)
		d.ct -= t
		n -= t
	}
}
