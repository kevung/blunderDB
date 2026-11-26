// Package j2k decodes JPEG 2000 images (ISO/IEC 15444-1): a JP2 file or a
// bare codestream. It is the decoder of github.com/d-fuji/jpeg2000, copied
// into this repository (see NOTICE) so that the bounds an untrusted input
// must hold are checked by the parser that reads it, in the same pass,
// rather than by a second reader that could disagree with it on where a
// marker segment ends.
//
// It registers nothing with package image: only package jpx reaches it.
package j2k

import (
	"context"
	"image"
	"math"
)

// MaxPixels is the largest reference grid Parse accepts, in pixels.
const MaxPixels = 12_000_000

// Image is a parsed codestream whose headers hold the parser's bounds.
type Image struct {
	cs   *codestream
	meta *jp2Meta
}

// Parse reads the boxes of a JP2 file, or takes a bare codestream, and
// parses every header of the codestream, refusing what it does not decode
// before anything proportional to the image is allocated.
func Parse(data []byte) (*Image, error) {
	j2k, meta, err := extractCodestream(data)
	if err != nil {
		return nil, err
	}
	cs, err := parseCodestream(j2k)
	if err != nil {
		return nil, err
	}
	return &Image{cs: cs, meta: meta}, nil
}

// Width and Height are the image's, on its reference grid.
func (im *Image) Width() int  { return im.cs.siz.xsiz - im.cs.siz.xosiz }
func (im *Image) Height() int { return im.cs.siz.ysiz - im.cs.siz.yosiz }

// Components is the number of components, one or three.
func (im *Image) Components() int { return len(im.cs.siz.comps) }

// Tiles is the number of tiles of the grid.
func (im *Image) Tiles() int { return len(im.cs.tiles) }

// Reversible is true when the main coding style names the 5/3 transform.
func (im *Image) Reversible() bool { return im.cs.cod.cp.transform == 1 }

// Decode decodes the image, reading ctx before each tile, each packet and
// each code-block's decoding: the units whose cost the headers do not
// bound. A tile's inverse transform runs unchecked.
func (im *Image) Decode(ctx context.Context) (image.Image, error) {
	return decodeImage(ctx, im.cs, im.meta)
}

func imageSize(siz *sizInfo) (int, int, error) {
	c0 := &siz.comps[0]
	w := ceilDiv(siz.xsiz, c0.dx) - ceilDiv(siz.xosiz, c0.dx)
	h := ceilDiv(siz.ysiz, c0.dy) - ceilDiv(siz.yosiz, c0.dy)
	for i := range siz.comps {
		c := &siz.comps[i]
		if c.dx != c0.dx || c.dy != c0.dy {
			return 0, 0, UnsupportedError("subsampled components")
		}
	}
	return w, h, nil
}

func maxDepth(siz *sizInfo) int {
	d := 0
	for i := range siz.comps {
		d = max(d, siz.comps[i].depth)
	}
	return d
}

func decodeImage(ctx context.Context, cs *codestream, meta *jp2Meta) (image.Image, error) {
	siz := &cs.siz
	w, h, err := imageSize(siz)
	if err != nil {
		return nil, err
	}
	nc := len(siz.comps)
	if nc > 4 {
		return nil, UnsupportedError("more than 4 components")
	}

	alpha := meta.alphaChan
	if alpha < 0 && (nc == 2 || nc == 4) {
		// Without container metadata, assume the extra channel of a
		// gray+alpha or RGB+alpha layout is opacity.
		alpha = nc - 1
	}

	deep := maxDepth(siz) > 8
	var img image.Image
	var setPix func(x, y int, px [4]int64)
	// px holds R,G,B,A (or gray replicated) scaled to the target depth.
	switch {
	case nc == 1 && !deep:
		m := image.NewGray(image.Rect(0, 0, w, h))
		img = m
		setPix = func(x, y int, px [4]int64) {
			m.Pix[y*m.Stride+x] = uint8(px[0])
		}
	case nc == 1 && deep:
		m := image.NewGray16(image.Rect(0, 0, w, h))
		img = m
		setPix = func(x, y int, px [4]int64) {
			i := y*m.Stride + 2*x
			m.Pix[i] = uint8(px[0] >> 8)
			m.Pix[i+1] = uint8(px[0])
		}
	case !deep:
		m := image.NewNRGBA(image.Rect(0, 0, w, h))
		img = m
		setPix = func(x, y int, px [4]int64) {
			i := y*m.Stride + 4*x
			m.Pix[i+0] = uint8(px[0])
			m.Pix[i+1] = uint8(px[1])
			m.Pix[i+2] = uint8(px[2])
			m.Pix[i+3] = uint8(px[3])
		}
	default:
		m := image.NewNRGBA64(image.Rect(0, 0, w, h))
		img = m
		setPix = func(x, y int, px [4]int64) {
			i := y*m.Stride + 8*x
			for c := 0; c < 4; c++ {
				m.Pix[i+2*c] = uint8(px[c] >> 8)
				m.Pix[i+2*c+1] = uint8(px[c])
			}
		}
	}

	outMax := int64(255)
	if deep {
		outMax = 65535
	}
	imgX0 := ceilDiv(siz.xosiz, siz.comps[0].dx)
	imgY0 := ceilDiv(siz.yosiz, siz.comps[0].dy)

	for _, td := range cs.tiles {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		d, err := newTileDecoder(cs, td)
		if err != nil {
			return nil, err
		}
		d.ctx = ctx
		if err := d.decode(); err != nil {
			return nil, err
		}
		planes, err := tileSamples(d)
		if err != nil {
			return nil, err
		}

		tc0 := d.comps[0]
		tw, th := tc0.w(), tc0.h()
		// Per-component scaling from its bit depth to the output depth.
		scale := make([]func(int32) int64, nc)
		for c := 0; c < nc; c++ {
			depth := siz.comps[c].depth
			inMax := int64(1)<<uint(depth) - 1
			scale[c] = func(v int32) int64 {
				return (int64(v)*outMax + inMax/2) / inMax
			}
		}
		for y := 0; y < th; y++ {
			for x := 0; x < tw; x++ {
				var px [4]int64
				px[3] = outMax
				i := y*tw + x
				switch nc {
				case 1:
					px[0] = scale[0](planes[0][i])
				case 2:
					g := scale[0](planes[0][i])
					px[0], px[1], px[2] = g, g, g
					px[3] = scale[1](planes[1][i])
				case 3:
					px[0] = scale[0](planes[0][i])
					px[1] = scale[1](planes[1][i])
					px[2] = scale[2](planes[2][i])
				case 4:
					px[0] = scale[0](planes[0][i])
					px[1] = scale[1](planes[1][i])
					px[2] = scale[2](planes[2][i])
					if alpha == 3 {
						px[3] = scale[3](planes[3][i])
					}
				}
				setPix(tc0.x0-imgX0+x, tc0.y0-imgY0+y, px)
			}
		}
	}
	return img, nil
}

// tileSamples applies the inverse component transform and DC level
// shift, returning clamped integer sample planes for one tile.
func tileSamples(d *tileDecoder) ([][]int32, error) {
	siz := &d.cs.siz
	nc := len(d.comps)
	planes := make([][]int32, nc)

	mct := d.th.mct && nc >= 3
	if mct {
		for c := 0; c < 3; c++ {
			if d.comps[c].w() != d.comps[0].w() || d.comps[c].h() != d.comps[0].h() ||
				d.comps[c].reversible != d.comps[0].reversible {
				return nil, errFormat("inconsistent components for MCT")
			}
		}
	}

	if mct && d.comps[0].reversible {
		// Inverse RCT (G.2).
		y, u, v := d.comps[0].samplesI, d.comps[1].samplesI, d.comps[2].samplesI
		for i := range y {
			g := y[i] - (u[i]+v[i])>>2
			r := v[i] + g
			b := u[i] + g
			y[i], u[i], v[i] = r, g, b
		}
	} else if mct {
		// Inverse ICT (G.3).
		y, cb, cr := d.comps[0].samplesF, d.comps[1].samplesF, d.comps[2].samplesF
		for i := range y {
			r := float64(y[i]) + 1.402*float64(cr[i])
			g := float64(y[i]) - 0.344136*float64(cb[i]) - 0.714136*float64(cr[i])
			b := float64(y[i]) + 1.772*float64(cb[i])
			y[i], cb[i], cr[i] = float32(r), float32(g), float32(b)
		}
	}

	for c := 0; c < nc; c++ {
		tc := d.comps[c]
		sc := &siz.comps[c]
		n := tc.w() * tc.h()
		out := make([]int32, n)
		// Reconstruct unsigned sample values: DC level shift for
		// unsigned data, plain offset into display range for signed.
		shift := int32(1) << uint(sc.depth-1)
		hi := int32(1)<<uint(sc.depth) - 1
		if tc.reversible {
			s := tc.samplesI
			for i := 0; i < n; i++ {
				out[i] = clampI32(s[i]+shift, 0, hi)
			}
		} else {
			s := tc.samplesF
			for i := 0; i < n; i++ {
				v := int32(math.Floor(float64(s[i]) + 0.5))
				out[i] = clampI32(v+shift, 0, hi)
			}
		}
		planes[c] = out
	}
	return planes, nil
}

func clampI32(v, lo, hi int32) int32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
