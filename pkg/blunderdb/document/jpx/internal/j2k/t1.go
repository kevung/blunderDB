package j2k

// EBCOT Tier-1 decoding (ISO/IEC 15444-1 Annex D): each code-block's
// coefficients are recovered bit-plane by bit-plane through three
// context-modeled coding passes driven by the MQ decoder.

// Context indices.
const (
	ctxRunLength = 17
	ctxUniform   = 18
	numContexts  = 19
)

// Coefficient state flags. Beside its own state, each coefficient keeps
// the significance of its eight neighbours and the sign of its four
// cardinal ones, updated when a neighbour becomes significant: the
// contexts of Tables D.1 and D.3 are then a table lookup.
const (
	fSig uint32 = 1 << iota
	fVisited
	fRefined
	fSign

	nW // significance of the neighbours
	nE
	nN
	nS
	nNW
	nNE
	nSW
	nSE

	sW // sign of the cardinal neighbours, with their significance
	sE
	sN
	sS
)

const (
	nShift = 4  // the neighbours' significance, nW to nSE
	sShift = 12 // the neighbours' signs, sW to sS
	// below is what the vertically causal mode leaves out on the last row
	// of a stripe.
	below = nS | nSW | nSE | sS
	// nAll is every neighbour's significance.
	nAll = nW | nE | nN | nS | nNW | nNE | nSW | nSE
)

type t1Decoder struct {
	w, h    int
	stride  int
	flags   []uint32 // (w+2) x (h+2), one-pixel border
	mag     []uint32
	lastBP  []int8 // last bit-plane at which each coefficient was coded
	mq      mqDecoder
	cx      [numContexts]mqContext
	sigTab  *[256]uint8
	causal  bool
	resetCx bool
	all     []byte // the segments of a code-block, when one codeword spans them
}

func (t *t1Decoder) resetContexts() {
	for i := range t.cx {
		t.cx[i] = mqContext{}
	}
	t.cx[0].index = 4
	t.cx[ctxRunLength].index = 3
	t.cx[ctxUniform].index = 46
}

// sigTabs is the significance context of Table D.1, by orientation (LL and
// LH share one) and by the significance of the eight neighbours, bit k of
// the index for flag nW<<k.
var sigTabs [4][256]uint8

// signTab is the sign context of Table D.3 with its XOR bit (bit 7), by the
// significance of the cardinal neighbours (bits 0-3, W E N S) and their
// signs (bits 4-7).
var signTab [256]uint8

func init() {
	for o := range sigTabs {
		for n := range 256 {
			bit := func(f uint32) int { return int(uint32(n)<<nShift&f) / int(f) }
			h := bit(nW) + bit(nE)
			v := bit(nN) + bit(nS)
			d := bit(nNW) + bit(nNE) + bit(nSW) + bit(nSE)
			sigTabs[o][n] = uint8(sigContext(o, h, v, d))
		}
	}
	for n := range 256 {
		contrib := func(k int) int {
			if n>>k&1 == 0 {
				return 0
			}
			if n>>(k+4)&1 != 0 {
				return -1
			}
			return 1
		}
		h := max(-1, min(1, contrib(0)+contrib(1)))
		v := max(-1, min(1, contrib(2)+contrib(3)))
		ctx, xor := signContext(h, v)
		signTab[n] = uint8(ctx) | xor<<7
	}
}

// sigContext is Table D.1: the context of h horizontal, v vertical and d
// diagonal significant neighbours of a coefficient of band orient.
func sigContext(orient, h, v, d int) int {
	switch orient {
	case 1: // HL: transpose
		h, v = v, h
	case 3: // HH: diagonal-dominant table
		hv := h + v
		switch {
		case d >= 3:
			return 8
		case d == 2:
			if hv >= 1 {
				return 7
			}
			return 6
		case d == 1:
			if hv >= 2 {
				return 5
			}
			return 3 + hv
		default:
			if hv >= 2 {
				return 2
			}
			return hv
		}
	}
	switch {
	case h == 2:
		return 8
	case h == 1:
		if v >= 1 {
			return 7
		}
		if d >= 1 {
			return 6
		}
		return 5
	case v == 2:
		return 4
	case v == 1:
		return 3
	case d >= 2:
		return 2
	default:
		return d
	}
}

// signContext is Table D.3: the context and XOR bit of the horizontal and
// vertical contributions h and v, each in -1..1.
func signContext(h, v int) (int, uint8) {
	switch {
	case h == 1:
		switch {
		case v == 1:
			return 13, 0
		case v == 0:
			return 12, 0
		default:
			return 11, 0
		}
	case h == 0:
		switch {
		case v == 1:
			return 10, 0
		case v == 0:
			return 9, 0
		default:
			return 10, 1
		}
	default:
		switch {
		case v == 1:
			return 11, 1
		case v == 0:
			return 12, 1
		default:
			return 13, 1
		}
	}
}

// neighbours is the state of a coefficient's neighbours the contexts read:
// on the last row of a stripe, the vertically causal mode drops the row
// below.
func (t *t1Decoder) neighbours(f uint32, y int) uint32 {
	if t.causal && y&3 == 3 {
		return f &^ below
	}
	return f
}

// sigCtx is the significance context of a coefficient whose flags,
// neighbours as the mode reads them, are f.
func (t *t1Decoder) sigCtx(f uint32) int {
	return int(t.sigTab[f>>nShift&0xff])
}

// decodeSign decodes the sign of a newly significant coefficient whose
// flags, neighbours as the mode reads them, are f.
func (t *t1Decoder) decodeSign(f uint32) uint32 {
	e := signTab[f>>nShift&0xf|f>>sShift<<4&0xf0]
	return uint32(t.mq.decode(&t.cx[e&0x7f]) ^ e>>7)
}

// setSignificant marks coefficient (x, y) at flags index idx significant
// at bit-plane bp and tells its neighbours.
func (t *t1Decoder) setSignificant(x, y, idx int, bp uint, negative uint32) {
	t.mag[y*t.w+x] |= 1 << bp
	t.lastBP[y*t.w+x] = int8(bp)
	s := t.stride
	f := t.flags
	f[idx] |= fSig | negative*fSign
	f[idx-s-1] |= nSE
	f[idx-s] |= nS | negative*sS
	f[idx-s+1] |= nSW
	f[idx-1] |= nE | negative*sE
	f[idx+1] |= nW | negative*sW
	f[idx+s-1] |= nNE
	f[idx+s] |= nN | negative*sN
	f[idx+s+1] |= nNW
}

// significancePass is the first pass of each bit-plane: coefficients
// that are not yet significant but have a significant neighbor.
func (t *t1Decoder) significancePass(bp uint) {
	for y0 := 0; y0 < t.h; y0 += 4 {
		yEnd := min(y0+4, t.h)
		for x := 0; x < t.w; x++ {
			idx := (y0+1)*t.stride + x + 1
			for y := y0; y < yEnd; y, idx = y+1, idx+t.stride {
				f := t.flags[idx]
				if f&fSig != 0 || f&nAll == 0 {
					continue
				}
				f = t.neighbours(f, y)
				ctx := t.sigCtx(f)
				if ctx == 0 {
					continue
				}
				if t.mq.decode(&t.cx[ctx]) == 1 {
					t.setSignificant(x, y, idx, bp, t.decodeSign(f))
				}
				t.flags[idx] |= fVisited
			}
		}
	}
}

// refinementPass refines coefficients that were already significant
// before this bit-plane.
func (t *t1Decoder) refinementPass(bp uint) {
	one := uint32(1) << bp
	for y0 := 0; y0 < t.h; y0 += 4 {
		yEnd := min(y0+4, t.h)
		for x := 0; x < t.w; x++ {
			idx := (y0+1)*t.stride + x + 1
			for y := y0; y < yEnd; y, idx = y+1, idx+t.stride {
				f := t.flags[idx]
				if f&(fSig|fVisited) != fSig {
					continue
				}
				ctx := 16
				if f&fRefined == 0 {
					ctx = 15
					if t.neighbours(f, y)&nAll == 0 {
						ctx = 14
					}
				}
				if t.mq.decode(&t.cx[ctx]) == 1 {
					t.mag[y*t.w+x] |= one
				}
				t.lastBP[y*t.w+x] = int8(bp)
				t.flags[idx] |= fRefined
			}
		}
	}
}

// cleanupPass handles all remaining coefficients, with the run-length
// shortcut for all-insignificant columns.
func (t *t1Decoder) cleanupPass(bp uint) {
	s := t.stride
	for y0 := 0; y0 < t.h; y0 += 4 {
		yEnd := min(y0+4, t.h)
		for x := 0; x < t.w; x++ {
			y := y0
			idx := (y0+1)*s + x + 1

			// Run-length mode: a full stripe column where nothing is
			// significant, visited, or has significant neighbors.
			if yEnd == y0+4 {
				last := fSig | fVisited | nAll
				if t.causal {
					last &^= below
				}
				fl := t.flags
				if (fl[idx]|fl[idx+s]|fl[idx+2*s])&(fSig|fVisited|nAll) == 0 && fl[idx+3*s]&last == 0 {
					if t.mq.decode(&t.cx[ctxRunLength]) == 0 {
						continue // whole column stays insignificant
					}
					k := t.mq.decode(&t.cx[ctxUniform])<<1 |
						t.mq.decode(&t.cx[ctxUniform])
					y = y0 + int(k)
					idx += int(k) * s
					f := t.neighbours(t.flags[idx], y)
					t.setSignificant(x, y, idx, bp, t.decodeSign(f))
					y++
					idx += s
				}
			}

			for ; y < yEnd; y, idx = y+1, idx+s {
				f := t.flags[idx]
				if f&(fSig|fVisited) != 0 {
					continue
				}
				f = t.neighbours(f, y)
				if t.mq.decode(&t.cx[t.sigCtx(f)]) == 1 {
					t.setSignificant(x, y, idx, bp, t.decodeSign(f))
				}
			}
		}
	}
	// Clear per-bit-plane visited marks.
	for i := range t.flags {
		t.flags[i] &^= fVisited
	}
}

// decodeCodeblock runs Tier-1 on one code-block and writes the
// dequantized coefficients into the band buffer. t is scratch, reused from
// one code-block to the next.
func decodeCodeblock(t *t1Decoder, cb *codeblock, b *band, tc *tileComp) error {
	w, h := cb.x1-cb.x0, cb.y1-cb.y0
	if w <= 0 || h <= 0 || cb.totalPasses == 0 {
		return nil
	}
	numbps := b.mb - cb.zbp
	if numbps <= 0 {
		return nil
	}

	t.w, t.h, t.stride = w, h, w+2
	t.flags = resize(t.flags, (w+2)*(h+2))
	t.mag = resize(t.mag, w*h)
	t.lastBP = resize(t.lastBP, w*h)
	t.sigTab = &sigTabs[b.orient]
	t.causal = tc.cp.cbStyle&cbVertCausal != 0
	t.resetCx = tc.cp.cbStyle&cbResetCtx != 0
	t.resetContexts()

	termAll := tc.cp.cbStyle&cbTermAll != 0
	var segs []segment
	var one [1]segment
	if termAll {
		segs = cb.segs
	} else {
		// All passes share one MQ codeword spanning the layers.
		all := t.all[:0]
		total := 0
		for _, s := range cb.segs {
			all = append(all, s.data...)
			total += s.passes
		}
		t.all = all
		one[0] = segment{data: all, passes: total}
		segs = one[:]
	}

	bp := numbps - 1
	passType := 2 // the first pass of the MSB plane is a cleanup pass
	for _, seg := range segs {
		t.mq.init(seg.data)
		for i := 0; i < seg.passes; i++ {
			if bp < 0 {
				break
			}
			if t.resetCx && !(bp == numbps-1 && passType == 2) {
				t.resetContexts()
			}
			switch passType {
			case 0:
				t.significancePass(uint(bp))
			case 1:
				t.refinementPass(uint(bp))
			case 2:
				t.cleanupPass(uint(bp))
			}
			if passType == 2 {
				passType = 0
				bp--
			} else {
				passType++
			}
		}
	}

	// Dequantize into the band buffer (Annex E), reconstructing each
	// coefficient at the midpoint of its remaining uncertainty
	// interval: half of the last bit-plane at which it was coded.
	bw := b.w()
	ox, oy := cb.x0-b.x0, cb.y0-b.y0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			mag := t.mag[y*w+x]
			if mag == 0 {
				continue
			}
			neg := t.flags[(y+1)*t.stride+x+1]&fSign != 0
			half := int64(1) << uint(t.lastBP[y*w+x]) >> 1
			di := (oy+y)*bw + ox + x
			if tc.reversible {
				v := int32(mag) + int32(half)
				if neg {
					v = -v
				}
				b.coeffI[di] = v
			} else {
				v := (float64(mag) + 0.5*float64(int64(1)<<uint(t.lastBP[y*w+x]))) * b.step
				if neg {
					v = -v
				}
				b.coeffF[di] = float32(v)
			}
		}
	}
	return nil
}

// resize returns s with n zero elements, reusing its array when it holds
// them.
func resize[T any](s []T, n int) []T {
	if cap(s) < n {
		return make([]T, n)
	}
	s = s[:n]
	clear(s)
	return s
}
