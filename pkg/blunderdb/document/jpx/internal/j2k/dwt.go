package j2k

// Inverse discrete wavelet transform (ISO/IEC 15444-1 Annex F):
// reversible 5/3 integer lifting and irreversible 9/7 float lifting,
// both with periodic symmetric extension. Coordinates are kept on the
// absolute reference grid so that odd tile origins interleave
// correctly.

import "github.com/kevung/blunderdb/pkg/blunderdb/document/internal/parallel"

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
// absolute indices are low-pass, odd are high-pass. ext is scratch of
// at least i1-i0+4 samples.
func sr53(x []int32, i0, i1 int, ext []int32) {
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
	ext = ext[:n+2*margin]
	for a := base; a < i0; a++ {
		ext[a-base] = x[refl(a, i0, i1)-i0]
	}
	copy(ext[margin:], x[:n])
	for a := i1; a < i1+margin; a++ {
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

// vr53 is sr53 down the columns [c0, c1) of the w-wide rows of x, a row
// of them at a time; ext is scratch of (i1-i0+4)*(c1-c0) samples. Each
// sample takes the operations sr53 gives it, in the same order.
func vr53(x []int32, w, c0, c1, i0, i1 int, ext []int32) {
	n, cw := i1-i0, c1-c0
	if n <= 0 || cw <= 0 {
		return
	}
	if n == 1 {
		if i0&1 == 1 {
			for j := c0; j < c1; j++ {
				x[j] /= 2
			}
		}
		return
	}
	const margin = 2
	base := i0 - margin
	for a := base; a < i1+margin; a++ {
		src := (refl(a, i0, i1) - i0) * w
		copy(ext[(a-base)*cw:(a-base+1)*cw], x[src+c0:src+c1])
	}
	e := i0 - 1
	if e&1 != 0 {
		e++
	}
	for ; e < i1+1; e += 2 {
		row, up, dn := rows3(ext, e-base, cw)
		for j := range row {
			row[j] -= (up[j] + dn[j] + 2) >> 2
		}
	}
	o := i0
	if o&1 == 0 {
		o++
	}
	for ; o < i1; o += 2 {
		row, up, dn := rows3(ext, o-base, cw)
		for j := range row {
			row[j] += (up[j] + dn[j]) >> 1
		}
	}
	for k := 0; k < n; k++ {
		copy(x[k*w+c0:k*w+c1], ext[(margin+k)*cw:(margin+k+1)*cw])
	}
}

// rows3 is row k of the cw-wide rows of ext, with the rows above and below,
// of one length so that indexing them by one index is checked once.
func rows3[T int32 | float32](ext []T, k, cw int) (row, up, dn []T) {
	row = ext[k*cw : (k+1)*cw]
	up = ext[(k-1)*cw : k*cw]
	dn = ext[(k+1)*cw : (k+2)*cw]
	return row, up[:len(row)], dn[:len(row)]
}

// sr97 performs the 1D irreversible synthesis of F.3.8.2 in place,
// with the same interleaving convention as sr53. ext is scratch of at
// least i1-i0+8 samples.
func sr97(x []float32, i0, i1 int, ext []float32) {
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
	ext = ext[:n+2*margin]
	// Undo the analysis gain: reflection preserves index parity.
	gain := func(a int, v float32) float32 {
		if a&1 == 0 {
			return v * dwtK
		}
		return v * (1 / dwtK)
	}
	for a := base; a < i0; a++ {
		ext[a-base] = gain(a, x[refl(a, i0, i1)-i0])
	}
	for a := i1; a < i1+margin; a++ {
		ext[a-base] = gain(a, x[refl(a, i0, i1)-i0])
	}
	mid := ext[margin : margin+n]
	for k := i0 & 1; k < n; k += 2 {
		mid[k] = x[k] * dwtK
	}
	for k := 1 - i0&1; k < n; k += 2 {
		mid[k] = x[k] * (1 / dwtK)
	}
	lift := func(parity, lo, hi int, c float32) {
		s := lo
		if s&1 != parity {
			s++
		}
		for k := s - base; k < hi-base; k += 2 {
			ext[k] -= c * (ext[k-1] + ext[k+1])
		}
	}
	lift(0, i0-3, i1+3, dwtDelta)
	lift(1, i0-2, i1+2, dwtGamma)
	lift(0, i0-1, i1+1, dwtBeta)
	lift(1, i0, i1, dwtAlpha)
	copy(x, mid)
}

// vr97 is sr97 down the columns [c0, c1) of the w-wide rows of x, as vr53
// is sr53; ext is scratch of (i1-i0+8)*(c1-c0) samples.
func vr97(x []float32, w, c0, c1, i0, i1 int, ext []float32) {
	n, cw := i1-i0, c1-c0
	if n <= 0 || cw <= 0 {
		return
	}
	if n == 1 {
		if i0&1 == 1 {
			for j := c0; j < c1; j++ {
				x[j] /= 2
			}
		}
		return
	}
	const margin = 4
	base := i0 - margin
	for a := base; a < i1+margin; a++ {
		src := x[(refl(a, i0, i1)-i0)*w+c0:][:cw]
		dst := ext[(a-base)*cw:][:cw]
		if a&1 == 0 {
			for j, v := range src {
				dst[j] = v * dwtK
			}
		} else {
			for j, v := range src {
				dst[j] = v * (1 / dwtK)
			}
		}
	}
	lift := func(parity, lo, hi int, c float32) {
		s := lo
		if s&1 != parity {
			s++
		}
		for i := s; i < hi; i += 2 {
			row, up, dn := rows3(ext, i-base, cw)
			for j := range row {
				row[j] -= c * (up[j] + dn[j])
			}
		}
	}
	lift(0, i0-3, i1+3, dwtDelta)
	lift(1, i0-2, i1+2, dwtGamma)
	lift(0, i0-1, i1+1, dwtBeta)
	lift(1, i0, i1, dwtAlpha)
	for k := 0; k < n; k++ {
		copy(x[k*w+c0:k*w+c1], ext[(margin+k)*cw:(margin+k+1)*cw])
	}
}

// strip is the columns the vertical pass takes at once: its scratch, a few
// dozen rows of them, stays in the cache.
const strip = 64

// inverseDWT reconstructs the tile-component samples from its
// sub-band coefficients, one resolution level at a time.
func inverseDWT(tc *tileComp) {
	if tc.reversible {
		tc.samplesI = idwt(tc, func(b *band) []int32 { return b.coeffI }, sr53, vr53, 2)
	} else {
		tc.samplesF = idwt(tc, func(b *band) []float32 { return b.coeffF }, sr97, vr97, 4)
	}
}

// idwt runs the levels of tc with the 1D synthesis hr along the rows and vr
// down the columns, whose symmetric extension is margin samples a side.
// Rows, and strips of columns, are independent: they run in parallel.
func idwt[T int32 | float32](tc *tileComp, coeff func(*band) []T,
	hr func(x []T, i0, i1 int, ext []T),
	vr func(x []T, w, c0, c1, i0, i1 int, ext []T), margin int) []T {
	cur := coeff(tc.res[0].bands[0])
	for r := 1; r < len(tc.res); r++ {
		res := tc.res[r]
		w, h := res.x1-res.x0, res.y1-res.y0
		out := make([]T, w*h)
		interleave(out, res, tc.res[r-1], cur, [3][]T{coeff(res.bands[0]), coeff(res.bands[1]), coeff(res.bands[2])})
		parallel.For(h, max(1, 1<<14/max(w, 1)), func(lo, hi int) {
			ext := make([]T, w+2*margin)
			for y := lo; y < hi; y++ {
				hr(out[y*w:y*w+w], res.x0, res.x1, ext)
			}
		})
		strips := (w + strip - 1) / strip
		parallel.For(strips, max(1, 1<<14/max(h*strip, 1)), func(lo, hi int) {
			ext := make([]T, (h+2*margin)*strip)
			for s := lo; s < hi; s++ {
				vr(out, w, s*strip, min((s+1)*strip, w), res.y0, res.y1, ext)
			}
		})
		cur = out
	}
	return cur
}

// interleave fills out with the resolution's interleaved array (F.3.3
// 2D_INTERLEAVE): the lower resolution's LL image ll at even (u, v), the
// HL, LH and HH bands at the others.
func interleave[T int32 | float32](out []T, res, prev *resolution, ll []T, bands [3][]T) {
	w := res.x1 - res.x0
	prevW := prev.x1 - prev.x0
	for v := res.y0; v < res.y1; v++ {
		row := out[(v-res.y0)*w : (v-res.y0+1)*w]
		var even, odd []T // the sources of even and odd u on row v
		var e0, o0 int    // their first column
		if v&1 == 0 {
			b := res.bands[0] // HL
			even, e0 = ll[(v/2-prev.y0)*prevW:], prev.x0
			odd, o0 = bands[0][(v/2-b.y0)*b.w():], b.x0
		} else {
			lh, hh := res.bands[1], res.bands[2]
			even, e0 = bands[1][(v>>1-lh.y0)*lh.w():], lh.x0
			odd, o0 = bands[2][(v>>1-hh.y0)*hh.w():], hh.x0
		}
		ue := res.x0 + res.x0&1 // the first even u
		for u := ue; u < res.x1; u += 2 {
			row[u-res.x0] = even[u/2-e0]
		}
		for u := res.x0 + 1 - res.x0&1; u < res.x1; u += 2 {
			row[u-res.x0] = odd[u>>1-o0]
		}
	}
}
