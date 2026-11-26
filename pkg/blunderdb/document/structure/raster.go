package structure

import (
	"image"
	"image/color"
	"math"
)

// raster is a grey image, 0 black to 1 white, one float32 per pixel. Every
// operation of the reader is a plain loop over it: no image library.
type raster struct {
	w, h int
	p    []float32
}

func newRaster(w, h int) *raster { return &raster{w: w, h: h, p: make([]float32, w*h)} }

func (r *raster) at(x, y int) float32 { return r.p[y*r.w+x] }

// fromImage converts a decoded page to grey. A grey image keeps its exact
// values; any other is converted by the standard luminance weights.
func fromImage(img image.Image) *raster {
	b := img.Bounds()
	r := newRaster(b.Dx(), b.Dy())
	plane := func(pix []uint8, stride, off int) *raster {
		for y := 0; y < r.h; y++ {
			row := pix[off+y*stride : off+y*stride+r.w]
			for x, v := range row {
				r.p[y*r.w+x] = float32(v) / 255
			}
		}
		return r
	}
	switch g := img.(type) {
	case *image.Gray:
		return plane(g.Pix, g.Stride, 0)
	case *image.YCbCr:
		// The luma of a JPEG is its grey.
		return plane(g.Y, g.YStride, g.YOffset(b.Min.X, b.Min.Y))
	case *image.RGBA:
		for y := 0; y < r.h; y++ {
			for x := 0; x < r.w; x++ {
				i := y*g.Stride + 4*x
				v := (19595*uint32(g.Pix[i])*0x101 + 38470*uint32(g.Pix[i+1])*0x101 + 7471*uint32(g.Pix[i+2])*0x101 + 1<<15) >> 24
				r.p[y*r.w+x] = float32(v) / 255
			}
		}
		return r
	case *image.Paletted:
		lut := make([]float32, len(g.Palette))
		for i, c := range g.Palette {
			lut[i] = float32(color.GrayModel.Convert(c).(color.Gray).Y) / 255
		}
		for y := 0; y < r.h; y++ {
			for x := 0; x < r.w; x++ {
				r.p[y*r.w+x] = lut[g.Pix[y*g.Stride+x]]
			}
		}
		return r
	}
	for y := 0; y < r.h; y++ {
		for x := 0; x < r.w; x++ {
			v := color.GrayModel.Convert(img.At(b.Min.X+x, b.Min.Y+y)).(color.Gray).Y
			r.p[y*r.w+x] = float32(v) / 255
		}
	}
	return r
}

// crop copies the rectangle [x0,x1)×[y0,y1), clipped to the raster.
func (r *raster) crop(x0, y0, x1, y1 int) *raster {
	x0, y0 = max(x0, 0), max(y0, 0)
	x1, y1 = min(x1, r.w), min(y1, r.h)
	if x1 < x0 {
		x1 = x0
	}
	if y1 < y0 {
		y1 = y0
	}
	out := newRaster(x1-x0, y1-y0)
	for y := y0; y < y1; y++ {
		copy(out.p[(y-y0)*out.w:(y-y0+1)*out.w], r.p[y*r.w+x0:y*r.w+x1])
	}
	return out
}

func (r *raster) clone() *raster {
	out := &raster{w: r.w, h: r.h, p: make([]float32, len(r.p))}
	copy(out.p, r.p)
	return out
}

// quantized is the raster as 8-bit grey, truncated like a conversion of
// floats to bytes, so the resampling steps see what a printed image holds.
func (r *raster) quantized() []float64 {
	q := make([]float64, len(r.p))
	for i, v := range r.p {
		q[i] = float64(uint8(v * 255))
	}
	return q
}

// normalize divides the page by its paper: the maximum over blocks of 4
// pixels, a 25-block maximum filter, a 9-block mean, enlarged back. A binding
// shadow or a yellowed page becomes white paper.
func normalize(g *raster) *raster {
	h4, w4 := g.h/4, g.w/4
	if h4 == 0 || w4 == 0 {
		return g.clone()
	}
	small := make([]float64, w4*h4)
	for y := 0; y < h4; y++ {
		for x := 0; x < w4; x++ {
			m := float32(0)
			for dy := 0; dy < 4; dy++ {
				for dx := 0; dx < 4; dx++ {
					m = max(m, g.p[(4*y+dy)*g.w+4*x+dx])
				}
			}
			small[y*w4+x] = float64(m)
		}
	}
	bg := windowFilter(small, w4, h4, 25, true)
	bg = uniformFilter(bg, w4, h4, 9)
	for i := range bg {
		bg[i] = float64(float32(bg[i]))
	}
	up := resample(bg, w4, h4, g.w, g.h, triangle, 1, false)
	out := newRaster(g.w, g.h)
	for i, v := range g.p {
		d := max(float32(up[i]), 0.05)
		out.p[i] = min(max(v/d, 0), 1)
	}
	return out
}

// windowFilter is the maximum (or minimum) over a k×k window centred on each
// pixel, the window clipped to the image (a mirrored border adds nothing to
// an extremum).
func windowFilter(v []float64, w, h, k int, isMax bool) []float64 {
	tmp := make([]float64, len(v))
	out := make([]float64, len(v))
	line := func(src, dst []float64, n, stride, off int) {
		half := k / 2
		dq := make([]int, 0, n)
		better := func(a, b float64) bool {
			if isMax {
				return a >= b
			}
			return a <= b
		}
		next := 0
		for i := 0; i < n; i++ {
			hi := min(i+half, n-1)
			for ; next <= hi; next++ {
				x := src[off+next*stride]
				for len(dq) > 0 && better(x, src[off+dq[len(dq)-1]*stride]) {
					dq = dq[:len(dq)-1]
				}
				dq = append(dq, next)
			}
			for dq[0] < i-half {
				dq = dq[1:]
			}
			dst[off+i*stride] = src[off+dq[0]*stride]
		}
	}
	for y := 0; y < h; y++ {
		line(v, tmp, w, 1, y*w)
	}
	for x := 0; x < w; x++ {
		line(tmp, out, h, w, x)
	}
	return out
}

// uniformFilter is the mean over a k×k window, the border mirrored
// (d c b a | a b c d).
func uniformFilter(v []float64, w, h, k int) []float64 {
	half := k / 2
	refl := func(i, n int) int {
		for i < 0 || i >= n {
			if i < 0 {
				i = -i - 1
			}
			if i >= n {
				i = 2*n - i - 1
			}
		}
		return i
	}
	tmp := make([]float64, len(v))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			s := 0.0
			for d := -half; d <= half; d++ {
				s += v[y*w+refl(x+d, w)]
			}
			tmp[y*w+x] = float64(float32(s / float64(k)))
		}
	}
	out := make([]float64, len(v))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			s := 0.0
			for d := -half; d <= half; d++ {
				s += tmp[refl(y+d, h)*w+x]
			}
			out[y*w+x] = s / float64(k)
		}
	}
	return out
}

func triangle(x float64) float64 {
	x = math.Abs(x)
	if x < 1 {
		return 1 - x
	}
	return 0
}

func sinc(x float64) float64 {
	if x == 0 {
		return 1
	}
	x *= math.Pi
	return math.Sin(x) / x
}

func lanczos(x float64) float64 {
	if -3 <= x && x < 3 {
		return sinc(x) * sinc(x/3)
	}
	return 0
}

// resample resizes v (w×h) to ow×oh by a separable convolution, rows first,
// each output pixel centred on its own place in the input, the kernel widened
// when shrinking. bytes rounds and clips each pass to 0..255, as an 8-bit
// image would.
func resample(v []float64, w, h, ow, oh int, kernel func(float64) float64, support float64, bytes bool) []float64 {
	type coeffs struct {
		lo int
		k  []float64
	}
	pre := func(in, out int) []coeffs {
		scale := float64(in) / float64(out)
		fs := max(scale, 1)
		sup := support * fs
		cs := make([]coeffs, out)
		for xx := 0; xx < out; xx++ {
			center := (float64(xx) + 0.5) * scale
			lo := max(int(center-sup+0.5), 0)
			hi := min(int(center+sup+0.5), in)
			k := make([]float64, max(hi-lo, 0))
			s := 0.0
			for x := range k {
				k[x] = kernel((float64(x+lo) - center + 0.5) / fs)
				s += k[x]
			}
			if s != 0 {
				for x := range k {
					k[x] /= s
				}
			}
			cs[xx] = coeffs{lo, k}
		}
		return cs
	}
	clip := func(x float64) float64 {
		if !bytes {
			return float64(float32(x))
		}
		return min(max(math.Floor(x+0.5), 0), 255)
	}
	cur, cw := v, w
	if ow != w {
		cs := pre(w, ow)
		nx := make([]float64, ow*h)
		for y := 0; y < h; y++ {
			for xx, c := range cs {
				s := 0.0
				for i, k := range c.k {
					s += k * cur[y*w+c.lo+i]
				}
				nx[y*ow+xx] = clip(s)
			}
		}
		cur, cw = nx, ow
	}
	if oh != h {
		cs := pre(h, oh)
		ny := make([]float64, cw*oh)
		for yy, c := range cs {
			for x := 0; x < cw; x++ {
				s := 0.0
				for i, k := range c.k {
					s += k * cur[(c.lo+i)*cw+x]
				}
				ny[yy*cw+x] = clip(s)
			}
		}
		cur = ny
	}
	return cur
}

// rotate turns the raster by deg degrees counter-clockwise about its centre,
// bilinear, keeping its size and filling with white; an angle under 0.05
// degree leaves it as it is.
func rotate(g *raster, deg float64) *raster {
	if math.Abs(deg) < 0.05 {
		return g.clone()
	}
	q := g.quantized()
	ang := -deg * math.Pi / 180
	a, b := math.Cos(ang), math.Sin(ang)
	d, e := -math.Sin(ang), math.Cos(ang)
	cx, cy := float64(g.w)/2, float64(g.h)/2
	c := a*(-cx) + b*(-cy) + cx
	f := d*(-cx) + e*(-cy) + cy
	out := newRaster(g.w, g.h)
	clampX := func(x int) int { return min(max(x, 0), g.w-1) }
	for y := 0; y < g.h; y++ {
		for x := 0; x < g.w; x++ {
			xin := a*(float64(x)+0.5) + b*(float64(y)+0.5) + c
			yin := d*(float64(x)+0.5) + e*(float64(y)+0.5) + f
			v := 255.0
			if xin >= 0 && xin < float64(g.w) && yin >= 0 && yin < float64(g.h) {
				xin -= 0.5
				yin -= 0.5
				x0, y0 := int(math.Floor(xin)), int(math.Floor(yin))
				dx, dy := xin-float64(x0), yin-float64(y0)
				xa, xb := clampX(x0), clampX(x0+1)
				row := max(y0, 0) * g.w
				v = q[row+xa] + (q[row+xb]-q[row+xa])*dx
				if y0+1 >= 0 && y0+1 < g.h {
					r2 := (y0 + 1) * g.w
					v2 := q[r2+xa] + (q[r2+xb]-q[r2+xa])*dx
					v += (v2 - v) * dy
				}
				v = float64(uint8(v))
			}
			out.p[y*g.w+x] = float32(v) / 255
		}
	}
	return out
}

// resize scales an 8-bit view of the raster to w×h with a Lanczos kernel.
func (r *raster) resize(w, h int) *raster {
	v := resample(r.quantized(), r.w, r.h, w, h, lanczos, 3, true)
	out := newRaster(w, h)
	for i, x := range v {
		out.p[i] = float32(x) / 255
	}
	return out
}

// mask is a binary image.
type mask struct {
	w, h int
	p    []bool
}

func newMask(w, h int) *mask { return &mask{w: w, h: h, p: make([]bool, w*h)} }

func (m *mask) at(x, y int) bool { return m.p[y*m.w+x] }

// Ink thresholds of a normalised page: a pixel darker than inkStrong is ink;
// one darker than inkWeak is ink when it touches strong ink.
const (
	inkStrong = 0.55
	inkWeak   = 0.85
)

// hysteresis is the ink of a normalised raster: a strong pixel, or a weak one
// 8-connected to a strong one, so a thin pale stroke of a scan stays whole
// while isolated paper grain is dropped.
func hysteresis(g *raster) *mask {
	m := newMask(g.w, g.h)
	stack := []int{}
	for i, v := range g.p {
		if v < inkStrong && !m.p[i] {
			m.p[i] = true
			stack = append(stack[:0], i)
			for len(stack) > 0 {
				j := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				x, y := j%g.w, j/g.w
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						xx, yy := x+dx, y+dy
						if xx < 0 || yy < 0 || xx >= g.w || yy >= g.h {
							continue
						}
						k := yy*g.w + xx
						if !m.p[k] && g.p[k] < inkWeak {
							m.p[k] = true
							stack = append(stack, k)
						}
					}
				}
			}
		}
	}
	return m
}

// dilate grows the mask by one pixel in the four directions, n times.
func (m *mask) dilate(n int) *mask {
	cur := m
	for ; n > 0; n-- {
		out := newMask(cur.w, cur.h)
		for y := 0; y < cur.h; y++ {
			for x := 0; x < cur.w; x++ {
				i := y*cur.w + x
				out.p[i] = cur.p[i] || (x > 0 && cur.p[i-1]) || (x+1 < cur.w && cur.p[i+1]) ||
					(y > 0 && cur.p[i-cur.w]) || (y+1 < cur.h && cur.p[i+cur.w])
			}
		}
		cur = out
	}
	return cur
}

// box is a component's bounding box, [x0,x1)×[y0,y1).
type box struct{ x0, y0, x1, y1 int }

func (b box) w() int { return b.x1 - b.x0 }
func (b box) h() int { return b.y1 - b.y0 }

// label numbers the 4-connected components of m from 1, in the order a
// raster scan meets them, and returns each one's box (boxes[i-1] for label i)
// and pixel count.
func label(m *mask) ([]int32, []box, []int) {
	lab := make([]int32, len(m.p))
	var boxes []box
	var areas []int
	stack := []int{}
	for i, v := range m.p {
		if !v || lab[i] != 0 {
			continue
		}
		n := int32(len(boxes) + 1)
		b := box{m.w, m.h, 0, 0}
		area := 0
		lab[i] = n
		stack = append(stack[:0], i)
		for len(stack) > 0 {
			j := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			x, y := j%m.w, j/m.w
			area++
			b.x0, b.y0 = min(b.x0, x), min(b.y0, y)
			b.x1, b.y1 = max(b.x1, x+1), max(b.y1, y+1)
			for _, k := range [4]int{j - 1, j + 1, j - m.w, j + m.w} {
				switch {
				case k == j-1 && x == 0, k == j+1 && x == m.w-1, k < 0, k >= len(m.p):
					continue
				}
				if m.p[k] && lab[k] == 0 {
					lab[k] = n
					stack = append(stack, k)
				}
			}
		}
		boxes = append(boxes, b)
		areas = append(areas, area)
	}
	return lab, boxes, areas
}

// sub copies the rectangle [x0,x1)×[y0,y1) of the mask, clipped.
func (m *mask) sub(x0, y0, x1, y1 int) *mask {
	x0, y0 = max(x0, 0), max(y0, 0)
	x1, y1 = min(x1, m.w), min(y1, m.h)
	if x1 < x0 {
		x1 = x0
	}
	if y1 < y0 {
		y1 = y0
	}
	out := newMask(x1-x0, y1-y0)
	for y := y0; y < y1; y++ {
		copy(out.p[(y-y0)*out.w:(y-y0+1)*out.w], m.p[y*m.w+x0:y*m.w+x1])
	}
	return out
}

// longestRuns is, for each column (byColumn) or each row, the longest run of
// set pixels along it after gaps of up to two pixels are closed (a scanned
// stroke, or one drawn in pieces).
func longestRuns(m *mask, byColumn bool) []int {
	n, length := m.h, m.w
	get := func(line, i int) bool { return m.p[line*m.w+i] }
	if byColumn {
		n, length = m.w, m.h
		get = func(line, i int) bool { return m.p[i*m.w+line] }
	}
	best := make([]int, n)
	for line := 0; line < n; line++ {
		run := 0
		for i := 0; i < length; i++ {
			c := get(line, i) || get(line, min(i+1, length-1)) || get(line, max(i-1, 0))
			if c {
				run++
				best[line] = max(best[line], run)
			} else {
				run = 0
			}
		}
	}
	return best
}

// thicken ORs each pixel with its neighbours across the lines (vertical:
// above and below), the border repeated.
func (m *mask) thicken(vertical bool) *mask {
	out := newMask(m.w, m.h)
	for y := 0; y < m.h; y++ {
		for x := 0; x < m.w; x++ {
			if vertical {
				out.p[y*m.w+x] = m.at(x, y) || m.at(x, min(y+1, m.h-1)) || m.at(x, max(y-1, 0))
			} else {
				out.p[y*m.w+x] = m.at(x, y) || m.at(min(x+1, m.w-1), y) || m.at(max(x-1, 0), y)
			}
		}
	}
	return out
}

// roundHalfEven rounds like the reference arithmetic: halves to even.
func roundHalfEven(x float64) int { return int(math.RoundToEven(x)) }

// trunc is a conversion of a float to an integer, toward zero.
func trunc(x float64) int { return int(x) }

func round3(x float64) float64 { return math.Round(x*1000) / 1000 }

func median(v []float64) float64 {
	if len(v) == 0 {
		return math.NaN()
	}
	s := append([]float64(nil), v...)
	sortFloats(s)
	n := len(s)
	if n%2 == 1 {
		return s[n/2]
	}
	return (s[n/2-1] + s[n/2]) / 2
}
