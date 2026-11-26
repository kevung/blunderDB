package j2k

// Inverse discrete wavelet transform (ISO/IEC 15444-1 Annex F):
// reversible 5/3 integer lifting and irreversible 9/7 float lifting,
// both with periodic symmetric extension. Coordinates are kept on the
// absolute reference grid so that odd tile origins interleave
// correctly.

// 9/7 lifting parameters (Table F.4).
const (
	dwtAlpha = -1.586134342059924
	dwtBeta  = -0.052980118572961
	dwtGamma = 0.882911075530934
	dwtDelta = 0.443506852043971
	dwtK     = 1.230174104914001
)

// refl maps an absolute index into [i0, i1) by symmetric reflection
// about the interval end points.
func refl(i, i0, i1 int) int {
	if i1-i0 == 1 {
		return i0
	}
	for i < i0 || i >= i1 {
		if i < i0 {
			i = 2*i0 - i
		} else {
			i = 2*(i1-1) - i
		}
	}
	return i
}

// sr53 performs the 1D reversible synthesis of F.3.8.2 in place.
// x[k] holds the interleaved sample at absolute index i0+k: even
// absolute indices are low-pass, odd are high-pass.
func sr53(x []int32, i0, i1 int) {
	n := i1 - i0
	if n <= 0 {
		return
	}
	if n == 1 {
		if i0&1 == 1 {
			x[0] /= 2
		}
		return
	}
	const margin = 2
	base := i0 - margin
	ext := make([]int32, n+2*margin)
	for a := base; a < i1+margin; a++ {
		ext[a-base] = x[refl(a, i0, i1)-i0]
	}
	// Even (low-pass) samples first, over one extra position each side
	// so the odd step has both neighbors.
	e := i0 - 1
	if e&1 != 0 {
		e++
	}
	for ; e < i1+1; e += 2 {
		k := e - base
		ext[k] -= (ext[k-1] + ext[k+1] + 2) >> 2
	}
	o := i0
	if o&1 == 0 {
		o++
	}
	for ; o < i1; o += 2 {
		k := o - base
		ext[k] += (ext[k-1] + ext[k+1]) >> 1
	}
	copy(x, ext[margin:margin+n])
}

// sr97 performs the 1D irreversible synthesis of F.3.8.2 in place,
// with the same interleaving convention as sr53.
func sr97(x []float32, i0, i1 int) {
	n := i1 - i0
	if n <= 0 {
		return
	}
	if n == 1 {
		if i0&1 == 1 {
			x[0] /= 2
		}
		return
	}
	const margin = 4
	base := i0 - margin
	ext := make([]float32, n+2*margin)
	for a := base; a < i1+margin; a++ {
		v := x[refl(a, i0, i1)-i0]
		// Undo the analysis gain: reflection preserves index parity.
		if a&1 == 0 {
			v *= dwtK
		} else {
			v *= 1 / dwtK
		}
		ext[a-base] = v
	}
	lift := func(parity, lo, hi int, c float32) {
		s := lo
		if s&1 != parity {
			s++
		}
		for i := s; i < hi; i += 2 {
			k := i - base
			ext[k] -= c * (ext[k-1] + ext[k+1])
		}
	}
	lift(0, i0-3, i1+3, dwtDelta)
	lift(1, i0-2, i1+2, dwtGamma)
	lift(0, i0-1, i1+1, dwtBeta)
	lift(1, i0, i1, dwtAlpha)
	copy(x, ext[margin:margin+n])
}

// inverseDWT reconstructs the tile-component samples from its
// sub-band coefficients, one resolution level at a time.
func inverseDWT(tc *tileComp) {
	if tc.reversible {
		tc.samplesI = idwtInt(tc)
	} else {
		tc.samplesF = idwtFloat(tc)
	}
}

func idwtInt(tc *tileComp) []int32 {
	cur := tc.res[0].bands[0].coeffI
	for r := 1; r < len(tc.res); r++ {
		res := tc.res[r]
		prev := tc.res[r-1]
		w, h := res.x1-res.x0, res.y1-res.y0
		out := make([]int32, w*h)
		interleave(res, prev, func(dst, src, band int) {
			var v int32
			if band < 0 {
				v = cur[src]
			} else {
				v = res.bands[band].coeffI[src]
			}
			out[dst] = v
		})
		for y := 0; y < h; y++ {
			sr53(out[y*w:y*w+w], res.x0, res.x1)
		}
		col := make([]int32, h)
		for x := 0; x < w; x++ {
			for y := 0; y < h; y++ {
				col[y] = out[y*w+x]
			}
			sr53(col, res.y0, res.y1)
			for y := 0; y < h; y++ {
				out[y*w+x] = col[y]
			}
		}
		cur = out
	}
	return cur
}

func idwtFloat(tc *tileComp) []float32 {
	cur := tc.res[0].bands[0].coeffF
	for r := 1; r < len(tc.res); r++ {
		res := tc.res[r]
		prev := tc.res[r-1]
		w, h := res.x1-res.x0, res.y1-res.y0
		out := make([]float32, w*h)
		interleave(res, prev, func(dst, src, band int) {
			var v float32
			if band < 0 {
				v = cur[src]
			} else {
				v = res.bands[band].coeffF[src]
			}
			out[dst] = v
		})
		for y := 0; y < h; y++ {
			sr97(out[y*w:y*w+w], res.x0, res.x1)
		}
		col := make([]float32, h)
		for x := 0; x < w; x++ {
			for y := 0; y < h; y++ {
				col[y] = out[y*w+x]
			}
			sr97(col, res.y0, res.y1)
			for y := 0; y < h; y++ {
				out[y*w+x] = col[y]
			}
		}
		cur = out
	}
	return cur
}

// interleave calls set for every sample of the resolution's
// interleaved array (F.3.3 2D_INTERLEAVE): band -1 denotes the
// lower-resolution LL image, 0..2 the HL/LH/HH bands.
func interleave(res, prev *resolution, set func(dst, src, band int)) {
	w := res.x1 - res.x0
	prevW := prev.x1 - prev.x0
	for v := res.y0; v < res.y1; v++ {
		for u := res.x0; u < res.x1; u++ {
			dst := (v-res.y0)*w + (u - res.x0)
			switch {
			case u&1 == 0 && v&1 == 0:
				set(dst, (v/2-prev.y0)*prevW+(u/2-prev.x0), -1)
			case u&1 == 1 && v&1 == 0:
				b := res.bands[0] // HL
				set(dst, (v/2-b.y0)*b.w()+(u>>1-b.x0), 0)
			case u&1 == 0 && v&1 == 1:
				b := res.bands[1] // LH
				set(dst, (v>>1-b.y0)*b.w()+(u/2-b.x0), 1)
			default:
				b := res.bands[2] // HH
				set(dst, (v>>1-b.y0)*b.w()+(u>>1-b.x0), 2)
			}
		}
	}
}
