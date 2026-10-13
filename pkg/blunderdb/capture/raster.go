package capture

import (
	"image"
	"image/draw"
	"math"
)

// raster is an RGB image with summed-area tables, so that the mean colour of
// any box costs four lookups per channel whatever its size: the search samples
// the same image at hundreds of scales.
type raster struct {
	w, h int
	pix  []uint8    // RGB, row-major
	sum  [3][]int64 // (w+1)*(h+1) summed-area table per channel
}

// newRaster copies img into RGB, averaging each f×f block into one pixel.
func newRaster(img image.Image, f int) *raster {
	b := img.Bounds()
	rgba, ok := img.(*image.RGBA)
	if !ok || rgba.Rect.Min != (image.Point{}) {
		rgba = image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
		draw.Draw(rgba, rgba.Rect, img, b.Min, draw.Src)
	}
	W, H := b.Dx(), b.Dy()
	r := &raster{w: (W + f - 1) / f, h: (H + f - 1) / f}
	r.pix = make([]uint8, r.w*r.h*3)
	for y := 0; y < r.h; y++ {
		for x := 0; x < r.w; x++ {
			var sum [3]int
			n := 0
			for yy := y * f; yy < min((y+1)*f, H); yy++ {
				row := rgba.Pix[yy*rgba.Stride:]
				for xx := x * f; xx < min((x+1)*f, W); xx++ {
					sum[0] += int(row[xx*4])
					sum[1] += int(row[xx*4+1])
					sum[2] += int(row[xx*4+2])
					n++
				}
			}
			for c := 0; c < 3; c++ {
				r.pix[(y*r.w+x)*3+c] = uint8(sum[c] / n)
			}
		}
	}
	r.integrate()
	return r
}

func (r *raster) integrate() {
	W := r.w + 1
	for c := 0; c < 3; c++ {
		s := make([]int64, W*(r.h+1))
		for y := 0; y < r.h; y++ {
			var run int64
			for x := 0; x < r.w; x++ {
				run += int64(r.pix[(y*r.w+x)*3+c])
				s[(y+1)*W+x+1] = s[y*W+x+1] + run
			}
		}
		r.sum[c] = s
	}
}

// downscale averages f×f blocks; the search runs on it so that a point is a
// dozen pixels wide whatever the capture's resolution.
func (r *raster) downscale(f int) *raster {
	if f <= 1 {
		return r
	}
	d := &raster{w: r.w / f, h: r.h / f}
	d.pix = make([]uint8, d.w*d.h*3)
	for y := 0; y < d.h; y++ {
		for x := 0; x < d.w; x++ {
			c := r.mean(float64(x*f), float64(y*f), float64(x*f+f), float64(y*f+f))
			for i := 0; i < 3; i++ {
				d.pix[(y*d.w+x)*3+i] = uint8(c[i] + 0.5)
			}
		}
	}
	d.integrate()
	return d
}

// mean is the average colour of the box [x0,x1)×[y0,y1), clipped to the image;
// a box wholly outside returns a colour no palette entry is near.
func (r *raster) mean(x0, y0, x1, y1 float64) [3]float64 {
	ix0, iy0 := clampInt(int(math.Round(x0)), 0, r.w), clampInt(int(math.Round(y0)), 0, r.h)
	ix1, iy1 := clampInt(int(math.Round(x1)), 0, r.w), clampInt(int(math.Round(y1)), 0, r.h)
	if ix1 <= ix0 {
		ix1 = ix0 + 1
	}
	if iy1 <= iy0 {
		iy1 = iy0 + 1
	}
	if ix1 > r.w || iy1 > r.h {
		return [3]float64{-1000, -1000, -1000}
	}
	W := r.w + 1
	n := float64((ix1 - ix0) * (iy1 - iy0))
	var out [3]float64
	for c := 0; c < 3; c++ {
		s := r.sum[c]
		out[c] = float64(s[iy1*W+ix1]-s[iy0*W+ix1]-s[iy1*W+ix0]+s[iy0*W+ix0]) / n
	}
	return out
}

// at is one pixel's colour.
func (r *raster) at(x, y int) [3]float64 {
	if x < 0 || y < 0 || x >= r.w || y >= r.h {
		return [3]float64{-1000, -1000, -1000}
	}
	i := (y*r.w + x) * 3
	return [3]float64{float64(r.pix[i]), float64(r.pix[i+1]), float64(r.pix[i+2])}
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func luma(c [3]float64) float64 { return 0.299*c[0] + 0.587*c[1] + 0.114*c[2] }
