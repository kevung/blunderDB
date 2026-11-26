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

// Coefficient state flags.
const (
	fSig uint8 = 1 << iota
	fVisited
	fRefined
	fSign
)

type t1Decoder struct {
	w, h    int
	stride  int
	flags   []uint8 // (w+2) x (h+2), one-pixel border
	mag     []uint32
	lastBP  []int8 // last bit-plane at which each coefficient was coded
	mq      mqDecoder
	cx      [numContexts]mqContext
	orient  int
	causal  bool
	resetCx bool
}

func (t *t1Decoder) resetContexts() {
	for i := range t.cx {
		t.cx[i] = mqContext{}
	}
	t.cx[0].index = 4
	t.cx[ctxRunLength].index = 3
	t.cx[ctxUniform].index = 46
}

// Zero-or-one significance of a neighbor.
func (t *t1Decoder) sig(idx int) int {
	return int(t.flags[idx] & fSig)
}

// sigCtx computes the significance coding context (Table D.1) for the
// flags index idx. noBelow implements the vertically causal mode.
func (t *t1Decoder) sigCtx(idx int, noBelow bool) int {
	s := t.stride
	h := t.sig(idx-1) + t.sig(idx+1)
	v := t.sig(idx - s)
	d := t.sig(idx-s-1) + t.sig(idx-s+1)
	if !noBelow {
		v += t.sig(idx + s)
		d += t.sig(idx+s-1) + t.sig(idx+s+1)
	}
	switch t.orient {
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

// decodeSign decodes the sign of a newly significant coefficient using
// the sign contexts of Table D.3.
func (t *t1Decoder) decodeSign(idx int, noBelow bool) uint8 {
	s := t.stride
	contrib := func(i int) int {
		f := t.flags[i]
		if f&fSig == 0 {
			return 0
		}
		if f&fSign != 0 {
			return -1
		}
		return 1
	}
	h := contrib(idx-1) + contrib(idx+1)
	v := contrib(idx - s)
	if !noBelow {
		v += contrib(idx + s)
	}
	h = max(-1, min(1, h))
	v = max(-1, min(1, v))

	var ctx int
	var xor uint8
	switch {
	case h == 1:
		switch {
		case v == 1:
			ctx = 13
		case v == 0:
			ctx = 12
		default:
			ctx = 11
		}
	case h == 0:
		switch {
		case v == 1:
			ctx = 10
		case v == 0:
			ctx = 9
		default:
			ctx, xor = 10, 1
		}
	default:
		switch {
		case v == 1:
			ctx, xor = 11, 1
		case v == 0:
			ctx, xor = 12, 1
		default:
			ctx, xor = 13, 1
		}
	}
	return t.mq.decode(&t.cx[ctx]) ^ xor
}

func (t *t1Decoder) setSignificant(x, y, idx int, bp uint, negative uint8) {
	t.mag[y*t.w+x] |= 1 << bp
	t.lastBP[y*t.w+x] = int8(bp)
	t.flags[idx] |= fSig
	if negative != 0 {
		t.flags[idx] |= fSign
	}
}

// significancePass is the first pass of each bit-plane: coefficients
// that are not yet significant but have a significant neighbor.
func (t *t1Decoder) significancePass(bp uint) {
	for y0 := 0; y0 < t.h; y0 += 4 {
		for x := 0; x < t.w; x++ {
			for y := y0; y < min(y0+4, t.h); y++ {
				idx := (y+1)*t.stride + x + 1
				f := t.flags[idx]
				if f&fSig != 0 {
					continue
				}
				noBelow := t.causal && y&3 == 3
				ctx := t.sigCtx(idx, noBelow)
				if ctx == 0 {
					continue
				}
				if t.mq.decode(&t.cx[ctx]) == 1 {
					neg := t.decodeSign(idx, noBelow)
					t.setSignificant(x, y, idx, bp, neg)
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
		for x := 0; x < t.w; x++ {
			for y := y0; y < min(y0+4, t.h); y++ {
				idx := (y+1)*t.stride + x + 1
				f := t.flags[idx]
				if f&fSig == 0 || f&fVisited != 0 {
					continue
				}
				var ctx int
				if f&fRefined != 0 {
					ctx = 16
				} else {
					noBelow := t.causal && y&3 == 3
					if t.sigCtx(idx, noBelow) == 0 {
						ctx = 14
					} else {
						ctx = 15
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
	for y0 := 0; y0 < t.h; y0 += 4 {
		for x := 0; x < t.w; x++ {
			y := y0
			yEnd := min(y0+4, t.h)

			// Run-length mode: a full stripe column where nothing is
			// significant, visited, or has significant neighbors.
			if yEnd == y0+4 {
				runMode := true
				for yy := y0; yy < yEnd; yy++ {
					idx := (yy+1)*t.stride + x + 1
					noBelow := t.causal && yy&3 == 3
					if t.flags[idx]&(fSig|fVisited) != 0 || t.sigCtx(idx, noBelow) != 0 {
						runMode = false
						break
					}
				}
				if runMode {
					if t.mq.decode(&t.cx[ctxRunLength]) == 0 {
						continue // whole column stays insignificant
					}
					k := t.mq.decode(&t.cx[ctxUniform])<<1 |
						t.mq.decode(&t.cx[ctxUniform])
					y = y0 + int(k)
					idx := (y+1)*t.stride + x + 1
					noBelow := t.causal && y&3 == 3
					neg := t.decodeSign(idx, noBelow)
					t.setSignificant(x, y, idx, bp, neg)
					y++
				}
			}

			for ; y < yEnd; y++ {
				idx := (y+1)*t.stride + x + 1
				f := t.flags[idx]
				if f&(fSig|fVisited) != 0 {
					continue
				}
				noBelow := t.causal && y&3 == 3
				ctx := t.sigCtx(idx, noBelow)
				if t.mq.decode(&t.cx[ctx]) == 1 {
					neg := t.decodeSign(idx, noBelow)
					t.setSignificant(x, y, idx, bp, neg)
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
// dequantized coefficients into the band buffer.
func decodeCodeblock(cb *codeblock, b *band, tc *tileComp) error {
	w, h := cb.x1-cb.x0, cb.y1-cb.y0
	if w <= 0 || h <= 0 || cb.totalPasses == 0 {
		return nil
	}
	numbps := b.mb - cb.zbp
	if numbps <= 0 {
		return nil
	}

	t := &t1Decoder{
		w: w, h: h, stride: w + 2,
		flags:   make([]uint8, (w+2)*(h+2)),
		mag:     make([]uint32, w*h),
		lastBP:  make([]int8, w*h),
		orient:  b.orient,
		causal:  tc.cp.cbStyle&cbVertCausal != 0,
		resetCx: tc.cp.cbStyle&cbResetCtx != 0,
	}
	t.resetContexts()

	termAll := tc.cp.cbStyle&cbTermAll != 0
	var segs []segment
	if termAll {
		segs = cb.segs
	} else {
		// All passes share one MQ codeword spanning the layers.
		var all []byte
		total := 0
		for _, s := range cb.segs {
			all = append(all, s.data...)
			total += s.passes
		}
		segs = []segment{{data: all, passes: total}}
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
