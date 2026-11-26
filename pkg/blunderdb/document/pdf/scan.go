package pdf

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"math"
	"strings"
	"unicode"

	"github.com/kevung/blunderdb/pkg/blunderdb/document/jpx"

	"golang.org/x/image/ccitt"
)

// A scanned page is a page made of images: Scan composes them into one grey
// image of the page and places the words of its text layer (often the
// invisible text of an OCR) on it.

var (
	// ErrNoPageImage: the page draws no image. A page of vectors (a diagram
	// drawn with lines) is not rendered.
	ErrNoPageImage = errors.New("pdf: the page draws no image (a page drawn with vectors is not rendered)")
	// ErrImageCodec: an image of the page is compressed with JBIG2, which
	// this reader does not decode, or in JPEG 2000 beyond what it takes.
	ErrImageCodec = errors.New("pdf: an image of the page is compressed with JBIG2, which is not decoded, or in a JPEG 2000 this reader refuses; give the pages as images (PNG or JPEG files)")
)

const (
	// MinScanDPI and MaxScanDPI bound the resolution a page is composed at:
	// its images' own, within these bounds.
	MinScanDPI = 150
	MaxScanDPI = 300
	// maxScanPixels bounds the composed page (and Limits.MaxStreamSize
	// does too); a larger page is composed at a lower resolution. One image
	// of a page has at most Limits.MaxStreamSize pixels, checked before it
	// is decoded.
	maxScanPixels = 64_000_000
	// maxDraws bounds the images a page draws.
	maxDraws = 4096
	// maxPainted bounds the pixels painted on a page, all images together,
	// as a multiple of the page's own: a background, a mask and a few
	// pictures.
	maxPainted = 6
)

// Word is a word of a page's text, its box in the pixels of the page's Scan.
type Word struct {
	X0, Y0, X1, Y1 float64
	Text           string
}

// Scan is a page composed from its images, with its text.
type Scan struct {
	Image *image.Gray
	DPI   float64
	Words []Word
	// Skipped names what the page draws and the composition leaves out (a
	// colour-keyed mask, an image in an unread colour space…).
	Skipped []string
}

type imageDraw struct {
	m    matrix // the image's unit square in user space
	s    *Stream
	res  Dict
	fill uint8
}

// Scan composes the page from the images it draws, each placed by its
// matrix, at the resolution of the finest image covering a twentieth of the
// page or more (MinScanDPI to MaxScanDPI); and gives the words of its text,
// visible or not, in the image's pixels. A text the reader cannot map to
// Unicode gives no word. A page without any image is ErrNoPageImage; an
// image compressed with JBIG2 is ErrImageCodec. Images are
// decoded one at a time and dropped once painted.
func (r *Reader) Scan(p Page) (sc *Scan, err error) {
	defer func() {
		if e := recover(); e != nil {
			sc, err = nil, malformed("%v", e)
		}
	}()
	data, err := r.content(p)
	if err != nil {
		return nil, err
	}
	in := &interp{r: r, g: gstate{ctm: identity, tz: 1}, tm: identity, tlm: identity, inForm: map[*Stream]bool{}, collect: true}
	if err := in.run(data, p.resources); err != nil {
		return nil, err
	}
	if len(in.draws) == 0 {
		return nil, ErrNoPageImage
	}
	bx0, by0, bx1, by1 := p.Box[0], p.Box[1], p.Box[2], p.Box[3]
	pw, ph := bx1-bx0, by1-by0
	if !(pw > 1 && ph > 1 && pw < 1e5 && ph < 1e5) {
		return nil, malformed("page box %v", p.Box)
	}
	dpi := 0.0
	for pass := 0; pass < 2 && dpi == 0; pass++ {
		for _, d := range in.draws {
			w, h := r.num(d.s.Dict["Width"]), r.num(d.s.Dict["Height"])
			ux, uy := math.Hypot(d.m[0], d.m[1]), math.Hypot(d.m[2], d.m[3])
			if !(ux > 0 && uy > 0) || w <= 0 || h <= 0 {
				continue
			}
			if pass == 0 && ux*uy < pw*ph/20 {
				continue
			}
			dpi = max(dpi, w/(ux/72), h/(uy/72))
		}
	}
	if math.IsNaN(dpi) || dpi == 0 {
		dpi = MinScanDPI
	}
	dpi = min(max(dpi, MinScanDPI), MaxScanDPI)
	if px, most := pw*ph*dpi*dpi/72/72, float64(min(maxScanPixels, r.lim.MaxStreamSize)); px > most {
		dpi *= math.Sqrt(most / px)
	}
	s := dpi / 72
	W0, H0 := int(math.Round(pw*s)), int(math.Round(ph*s))
	if W0 < 1 || H0 < 1 {
		return nil, malformed("page of %d×%d pixels", W0, H0)
	}
	dev := matrix{s, 0, 0, -s, -bx0 * s, by1 * s}
	W, H := W0, H0
	rot, _ := r.Resolve(p.dict["Rotate"]).(int64)
	switch ((rot % 360) + 360) % 360 {
	case 90:
		dev, W, H = dev.mul(matrix{0, 1, -1, 0, float64(H0), 0}), H0, W0
	case 180:
		dev = dev.mul(matrix{-1, 0, 0, -1, float64(W0), float64(H0)})
	case 270:
		dev, W, H = dev.mul(matrix{0, -1, 1, 0, 0, float64(W0)}), H0, W0
	}
	out := &Scan{Image: image.NewGray(image.Rect(0, 0, W, H)), DPI: dpi}
	for i := range out.Image.Pix {
		out.Image.Pix[i] = 255
	}
	budget := maxPainted * W * H
	var codecErr, refused error
	painted := 0
	for _, d := range in.draws {
		pic, why, err := r.picture(d)
		if errors.Is(err, ErrImageCodec) {
			codecErr = err
			continue
		}
		if err != nil {
			out.Skipped = append(out.Skipped, err.Error())
			if errors.Is(err, jpx.ErrUnsupported) || errors.Is(err, jpx.ErrBudget) {
				refused = err
			}
			continue
		}
		painted++
		if why != "" {
			out.Skipped = append(out.Skipped, why)
		}
		if budget -= paint(out.Image, d.m.mul(dev), pic); budget < 0 {
			return nil, fmt.Errorf("%w: a page paints more than %d times its pixels", ErrTooLarge, maxPainted)
		}
	}
	if codecErr != nil {
		return nil, codecErr
	}
	if refused != nil && painted == 0 {
		// A page left blank by a refusal would read as a page without
		// diagram: it is named as unread instead.
		return nil, fmt.Errorf("%w: %w", ErrImageCodec, refused)
	}
	out.Words = words(in.out, dev)
	return out, nil
}

// plane is one channel of an image, 8 bits a sample, row 0 at the top.
type plane struct {
	w, h int
	p    []uint8
}

func (pl *plane) at(u, v float64) uint8 {
	x, y := int(u*float64(pl.w)), int((1-v)*float64(pl.h))
	x, y = min(max(x, 0), pl.w-1), min(max(y, 0), pl.h-1)
	return pl.p[y*pl.w+x]
}

// picture is an image ready to paint: its grey (nil for a stencil mask,
// painted with fill) and its opacity (nil: opaque).
type picture struct {
	grey, alpha *plane
	fill        uint8
}

// picture decodes the image a draw paints, with its mask. why names what
// is left out of it.
func (r *Reader) picture(d imageDraw) (pic picture, why string, err error) {
	dict := d.s.Dict
	pic.fill = d.fill
	if b, _ := r.Resolve(dict["ImageMask"]).(bool); b {
		m, err := r.stencil(d.s, d.res)
		if err != nil {
			return pic, "", err
		}
		pic.alpha = m
		return pic, "", nil
	}
	if pic.grey, err = r.greyPlane(d.s, d.res); err != nil {
		return pic, "", err
	}
	if sm, ok := r.Resolve(dict["SMask"]).(*Stream); ok {
		a, err := r.greyPlane(sm, d.res)
		if err != nil {
			return pic, "", err
		}
		pic.alpha = a
	} else if mk, ok := r.Resolve(dict["Mask"]).(*Stream); ok {
		a, err := r.stencil(mk, d.res)
		if err != nil {
			return pic, "", err
		}
		pic.alpha = a
	} else if _, ok := r.Resolve(dict["Mask"]).(Array); ok {
		why = "a colour-keyed mask is not applied: the image is painted whole"
	}
	return pic, why, nil
}

// stencil decodes a stencil mask as an opacity: 255 where it paints (a 0
// sample, or a 1 when its Decode is [1 0]).
func (r *Reader) stencil(s *Stream, res Dict) (*plane, error) {
	pl, err := r.samples(s, res, true)
	if err != nil {
		return nil, err
	}
	inv := false
	if a, ok := r.Resolve(s.Dict["Decode"]).(Array); ok && len(a) == 2 && r.num(a[0]) > r.num(a[1]) {
		inv = true
	}
	for i, v := range pl.p {
		if (v < 128) != inv {
			pl.p[i] = 255
		} else {
			pl.p[i] = 0
		}
	}
	return pl, nil
}

func (r *Reader) greyPlane(s *Stream, res Dict) (*plane, error) { return r.samples(s, res, false) }

// imageFilters undoes a stream's general filters and stops at an image
// codec, which it names with its parameters.
func (r *Reader) imageFilters(s *Stream) (data []byte, codec Name, parm Dict, err error) {
	var filters Array
	switch f := r.Resolve(s.Dict["Filter"]).(type) {
	case Name:
		filters = Array{f}
	case Array:
		filters = f
	}
	var parms Array
	switch p := r.Resolve(s.Dict["DecodeParms"]).(type) {
	case Dict:
		parms = Array{p}
	case Array:
		parms = p
	}
	data = s.Raw
	for i, f := range filters {
		var p Dict
		if i < len(parms) {
			p, _ = r.Resolve(parms[i]).(Dict)
		}
		name, _ := r.Resolve(f).(Name)
		switch name {
		case "FlateDecode", "Fl":
			data, err = r.inflate(data)
			if err == nil {
				data, err = predict(data, p)
			}
		case "ASCIIHexDecode", "AHx":
			data = asciiHex(data)
		case "ASCII85Decode", "A85":
			data = ascii85(data)
		case "DCTDecode", "DCT", "CCITTFaxDecode", "CCF", "JBIG2Decode", "JPXDecode":
			if i != len(filters)-1 {
				return nil, "", nil, malformed("image codec %s before another filter", name)
			}
			return data, name, p, nil
		default:
			return nil, "", nil, fmt.Errorf("%w: %s", ErrUnsupported, name)
		}
		if err != nil {
			return nil, "", nil, err
		}
		if len(data) > r.lim.MaxStreamSize {
			return nil, "", nil, fmt.Errorf("%w: a stream decodes to more than %d bytes", ErrTooLarge, r.lim.MaxStreamSize)
		}
	}
	return data, "", nil, nil
}

// samples decodes an image to one grey sample a pixel. A stencil is read as
// one bit a pixel, whatever its dictionary says.
func (r *Reader) samples(s *Stream, res Dict, stencil bool) (*plane, error) {
	dict := s.Dict
	w, h := int(r.num(dict["Width"])), int(r.num(dict["Height"]))
	if w <= 0 || h <= 0 || w > 1<<20 || h > 1<<20 || w*h > r.lim.MaxStreamSize {
		return nil, fmt.Errorf("%w: an image of %d×%d pixels", ErrTooLarge, w, h)
	}
	data, codec, parm, err := r.imageFilters(s)
	if err != nil {
		return nil, err
	}
	switch codec {
	case "JBIG2Decode":
		return nil, fmt.Errorf("%w (%s)", ErrImageCodec, codec)
	case "JPXDecode":
		return r.jpxPlane(data, dict, w, h, stencil)
	case "DCTDecode", "DCT":
		return jpegPlane(data, r.lim.MaxStreamSize)
	case "CCITTFaxDecode", "CCF":
		return ccittPlane(data, parm, w, h, r.lim.MaxStreamSize)
	}
	bpc := int(r.num(dict["BitsPerComponent"]))
	if stencil || bpc == 0 {
		bpc = 1
	}
	cs := colourSpace{n: 1}
	if !stencil {
		if cs, err = r.colourSpace(dict["ColorSpace"], res, 0); err != nil {
			return nil, err
		}
	}
	if bpc != 1 && bpc != 2 && bpc != 4 && bpc != 8 && bpc != 16 {
		return nil, malformed("%d bits a component", bpc)
	}
	n := cs.n
	if cs.lookup != nil {
		n = 1
	}
	row := (w*n*bpc + 7) / 8
	if row*h > r.lim.MaxStreamSize {
		return nil, fmt.Errorf("%w: an image of %d×%d pixels", ErrTooLarge, w, h)
	}
	if len(data) < row*h {
		data = append(data, make([]byte, row*h-len(data))...) // a short stream ends in black, as viewers draw it
	}
	var dec [][2]float64
	if a, ok := r.Resolve(dict["Decode"]).(Array); ok && len(a) == 2*n && cs.lookup == nil {
		for k := 0; k < n; k++ {
			dec = append(dec, [2]float64{r.num(a[2*k]), r.num(a[2*k+1])})
		}
	}
	maxv := float64(int(1)<<bpc - 1)
	pl := &plane{w, h, make([]uint8, w*h)}
	comp := make([]float64, n)
	for y := 0; y < h; y++ {
		line := data[y*row : (y+1)*row]
		bit := 0
		for x := 0; x < w; x++ {
			for k := 0; k < n; k++ {
				var v int
				switch bpc {
				case 8:
					v = int(line[bit/8])
				case 16:
					v = int(line[bit/8])<<8 | int(line[bit/8+1])
				default:
					v = int(line[bit/8]>>(8-bpc-bit%8)) & (1<<bpc - 1)
				}
				bit += bpc
				comp[k] = float64(v) / maxv
				if dec != nil {
					comp[k] = dec[k][0] + comp[k]*(dec[k][1]-dec[k][0])
				}
				if cs.lookup != nil {
					comp[k] = float64(v)
				}
			}
			pl.p[y*w+x] = cs.grey(comp)
		}
	}
	return pl, nil
}

func jpegPlane(data []byte, most int) (*plane, error) {
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, malformed("jpeg: %v", err)
	}
	if cfg.Width*cfg.Height > most || cfg.Width <= 0 || cfg.Height <= 0 {
		return nil, fmt.Errorf("%w: a JPEG of %d×%d pixels", ErrTooLarge, cfg.Width, cfg.Height)
	}
	img, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, malformed("jpeg: %v", err)
	}
	return greyPlane(img), nil
}

// jpxPlane decodes a JPEG 2000 image to grey. The image dictionary decides
// what the codestream cannot: an Indexed colour space, an embedded alpha
// (SMaskInData) or a Decode array this reader cannot honour is refused, the
// image skipped and named, rather than read wrongly. BitsPerComponent is
// ignored for this filter, as the PDF standard says.
func (r *Reader) jpxPlane(data []byte, dict Dict, w, h int, stencil bool) (*plane, error) {
	if stencil {
		return nil, malformed("a JPEG 2000 stencil mask")
	}
	if n := r.num(dict["SMaskInData"]); n != 0 {
		return nil, fmt.Errorf("%w: JPEG 2000 with an embedded alpha (SMaskInData %d)", jpx.ErrUnsupported, int(n))
	}
	if isIndexed(r, dict["ColorSpace"]) {
		return nil, fmt.Errorf("%w: JPEG 2000 in an Indexed colour space", jpx.ErrUnsupported)
	}
	img, info, err := jpx.Decode(data, w, h, &r.jpx)
	if err != nil {
		return nil, err
	}
	pl := greyPlane(img)
	if a, ok := r.Resolve(dict["Decode"]).(Array); ok {
		if len(a) != 2*info.Components {
			return nil, malformed("a Decode array of %d numbers for %d components", len(a), info.Components)
		}
		d0, d1 := r.num(a[0]), r.num(a[1])
		for k := 1; k < info.Components; k++ {
			if r.num(a[2*k]) != d0 || r.num(a[2*k+1]) != d1 {
				return nil, fmt.Errorf("%w: JPEG 2000 with a Decode array that differs by component", jpx.ErrUnsupported)
			}
		}
		if d0 != 0 || d1 != 1 {
			for i, v := range pl.p {
				x := d0 + float64(v)/255*(d1-d0)
				pl.p[i] = uint8(min(max(x, 0), 1)*255 + 0.5)
			}
		}
	}
	return pl, nil
}

func isIndexed(r *Reader, cs Object) bool {
	switch v := r.Resolve(cs).(type) {
	case Name:
		return v == "Indexed" || v == "I"
	case Array:
		return len(v) > 0 && isIndexed(r, v[0])
	}
	return false
}

// greyPlane reduces a decoded image to one grey sample a pixel.
func greyPlane(img image.Image) *plane {
	b := img.Bounds()
	pl := &plane{b.Dx(), b.Dy(), make([]uint8, b.Dx()*b.Dy())}
	switch m := img.(type) {
	case *image.Gray:
		for y := 0; y < pl.h; y++ {
			copy(pl.p[y*pl.w:(y+1)*pl.w], m.Pix[y*m.Stride:y*m.Stride+pl.w])
		}
	case *image.YCbCr:
		for y := 0; y < pl.h; y++ {
			copy(pl.p[y*pl.w:(y+1)*pl.w], m.Y[y*m.YStride:y*m.YStride+pl.w])
		}
	case *image.NRGBA:
		for y := 0; y < pl.h; y++ {
			for x := 0; x < pl.w; x++ {
				p := m.Pix[y*m.Stride+4*x:]
				pl.p[y*pl.w+x] = uint8((299*uint32(p[0]) + 587*uint32(p[1]) + 114*uint32(p[2])) / 1000)
			}
		}
	case *image.CMYK:
		for i := 0; i < pl.w*pl.h; i++ {
			c := m.Pix[i*4 : i*4+4]
			k := 0.3*float64(c[0]) + 0.59*float64(c[1]) + 0.11*float64(c[2]) + float64(c[3])
			pl.p[i] = uint8(255 - min(k, 255))
		}
	default:
		for y := 0; y < pl.h; y++ {
			for x := 0; x < pl.w; x++ {
				rr, gg, bb, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
				pl.p[y*pl.w+x] = uint8((299*rr + 587*gg + 114*bb) / 1000 >> 8)
			}
		}
	}
	return pl
}

// ccittPlane decodes a fax-compressed image (Group 3 one-dimensional or
// Group 4) to 0 for black and 255 for white.
func ccittPlane(data []byte, p Dict, w, h, most int) (*plane, error) {
	k, _ := p["K"].(int64)
	cols := w
	if c, ok := p["Columns"].(int64); ok && c > 0 {
		cols = int(c)
	}
	rows := h
	if c, ok := p["Rows"].(int64); ok && c > 0 {
		rows = int(c)
	}
	if cols > 1<<20 || rows > 1<<20 || cols*rows > most {
		return nil, fmt.Errorf("%w: a fax image of %d×%d pixels", ErrTooLarge, cols, rows)
	}
	sf := ccitt.Group4
	if k > 0 {
		return nil, fmt.Errorf("%w: CCITTFaxDecode with mixed one- and two-dimensional coding (K > 0)", ErrUnsupported)
	} else if k == 0 {
		sf = ccitt.Group3
	}
	black1, _ := p["BlackIs1"].(bool)
	align, _ := p["EncodedByteAlign"].(bool)
	dst := image.NewGray(image.Rect(0, 0, cols, rows))
	for i := range dst.Pix {
		dst.Pix[i] = 255
	}
	// A broken or truncated stream keeps the rows decoded before the fault,
	// the others white, as viewers draw it.
	_ = ccitt.DecodeIntoGray(dst, bytes.NewReader(data), ccitt.MSB, sf, &ccitt.Options{Align: align, Invert: black1})
	return &plane{cols, rows, dst.Pix}, nil
}

// colourSpace turns an image's components to grey.
type colourSpace struct {
	n      int
	kind   byte   // 'G' grey, 'R' RGB, 'K' CMYK, 'L' Lab, 'S' a separation (tint)
	lookup []byte // Indexed: the base space's components per index
	base   *colourSpace
}

func (cs colourSpace) grey(c []float64) uint8 {
	if cs.lookup != nil {
		i := int(c[0]) * cs.base.n
		if i < 0 || i+cs.base.n > len(cs.lookup) {
			return 0
		}
		b := make([]float64, cs.base.n)
		for k := range b {
			b[k] = float64(cs.lookup[i+k]) / 255
		}
		return cs.base.grey(b)
	}
	var g float64
	switch cs.kind {
	case 'R':
		g = 0.299*c[0] + 0.587*c[1] + 0.114*c[2]
	case 'K':
		g = 1 - min(1, 0.3*c[0]+0.59*c[1]+0.11*c[2]+c[3])
	case 'L':
		g = c[0] / 100
	case 'S':
		g = 1 - c[0]
	default:
		g = c[0]
	}
	if math.IsNaN(g) {
		return 0
	}
	return uint8(math.Round(min(max(g, 0), 1) * 255))
}

func (r *Reader) colourSpace(o Object, res Dict, depth int) (colourSpace, error) {
	if depth > 4 {
		return colourSpace{}, malformed("colour spaces nested too deep")
	}
	o = r.Resolve(o)
	if n, ok := o.(Name); ok {
		switch n {
		case "DeviceGray", "G", "CalGray", "":
			return colourSpace{n: 1, kind: 'G'}, nil
		case "DeviceRGB", "RGB", "CalRGB":
			return colourSpace{n: 3, kind: 'R'}, nil
		case "DeviceCMYK", "CMYK":
			return colourSpace{n: 4, kind: 'K'}, nil
		}
		named, _ := r.Resolve(res["ColorSpace"]).(Dict)
		if v, ok := named[n]; ok {
			return r.colourSpace(v, res, depth+1)
		}
		return colourSpace{}, fmt.Errorf("%w: colour space %s", ErrUnsupported, n)
	}
	if o == nil {
		return colourSpace{n: 1, kind: 'G'}, nil
	}
	a, ok := o.(Array)
	if !ok || len(a) == 0 {
		return colourSpace{}, malformed("colour space %v", o)
	}
	kind, _ := r.Resolve(a[0]).(Name)
	switch kind {
	case "CalGray", "CalRGB", "DeviceGray", "DeviceRGB", "DeviceCMYK":
		return r.colourSpace(kind, res, depth+1)
	case "Lab":
		return colourSpace{n: 3, kind: 'L'}, nil
	case "ICCBased":
		if len(a) > 1 {
			if s, ok := r.Resolve(a[1]).(*Stream); ok {
				switch int(r.num(s.Dict["N"])) {
				case 1:
					return colourSpace{n: 1, kind: 'G'}, nil
				case 3:
					return colourSpace{n: 3, kind: 'R'}, nil
				case 4:
					return colourSpace{n: 4, kind: 'K'}, nil
				}
			}
		}
	case "Separation":
		return colourSpace{n: 1, kind: 'S'}, nil
	case "Indexed", "I":
		if len(a) == 4 {
			base, err := r.colourSpace(a[1], res, depth+1)
			if err != nil || base.lookup != nil {
				return colourSpace{}, malformed("indexed colour space")
			}
			var lut []byte
			switch v := r.Resolve(a[3]).(type) {
			case String:
				lut = []byte(v)
			case *Stream:
				if lut, err = r.Decode(v); err != nil {
					return colourSpace{}, err
				}
			}
			return colourSpace{n: 1, lookup: lut, base: &base}, nil
		}
	}
	return colourSpace{}, fmt.Errorf("%w: colour space %v", ErrUnsupported, kind)
}

func (m matrix) inverse() (matrix, bool) {
	det := m[0]*m[3] - m[1]*m[2]
	if det == 0 || math.IsNaN(det) || math.IsInf(det, 0) {
		return matrix{}, false
	}
	a, b, c, d := m[3]/det, -m[1]/det, -m[2]/det, m[0]/det
	return matrix{a, b, c, d, -(m[4]*a + m[5]*c), -(m[4]*b + m[5]*d)}, true
}

func (m matrix) apply(x, y float64) (float64, float64) {
	return x*m[0] + y*m[2] + m[4], x*m[1] + y*m[3] + m[5]
}

// paint composes a picture onto the page through t, the image's unit square
// in the page's pixels; each pixel takes the mean of up to 3 × 3 samples, so
// a finer image is reduced rather than decimated. It returns the pixels
// visited.
func paint(dst *image.Gray, t matrix, pic picture) int {
	inv, ok := t.inverse()
	if !ok {
		return 0
	}
	x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, c := range [4][2]float64{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
		x, y := t.apply(c[0], c[1])
		x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
	}
	W, H := dst.Rect.Dx(), dst.Rect.Dy()
	if !(x1 > 0 && y1 > 0 && x0 < float64(W) && y0 < float64(H)) {
		return 0
	}
	ix0, iy0 := max(int(math.Floor(x0)), 0), max(int(math.Floor(y0)), 0)
	ix1, iy1 := min(int(math.Ceil(x1)), W), min(int(math.Ceil(y1)), H)
	src := pic.grey
	if src == nil {
		src = pic.alpha
	}
	// Source pixels a page pixel spans, along each axis of the page.
	du := math.Hypot(inv[0], inv[2]) * float64(src.w)
	dv := math.Hypot(inv[1], inv[3]) * float64(src.h)
	k := min(max(int(math.Ceil(max(du, dv))), 1), 3)
	n := float64(k * k)
	for y := iy0; y < iy1; y++ {
		for x := ix0; x < ix1; x++ {
			var sumG, sumA float64
			for sy := 0; sy < k; sy++ {
				for sx := 0; sx < k; sx++ {
					u, v := inv.apply(float64(x)+(float64(sx)+0.5)/float64(k), float64(y)+(float64(sy)+0.5)/float64(k))
					if u < 0 || u >= 1 || v < 0 || v >= 1 {
						continue
					}
					a := 255.0
					if pic.alpha != nil {
						a = float64(pic.alpha.at(u, v))
					}
					g := float64(pic.fill)
					if pic.grey != nil {
						g = float64(pic.grey.at(u, v))
					}
					sumA += a
					sumG += g * a
				}
			}
			if sumA == 0 {
				continue
			}
			alpha := sumA / n / 255
			g := sumG / sumA
			i := y*dst.Stride + x
			dst.Pix[i] = uint8(math.Round(float64(dst.Pix[i])*(1-alpha) + g*alpha))
		}
	}
	return (ix1 - ix0) * (iy1 - iy0)
}

// words joins a page's glyphs into words: glyphs drawn one after the other
// on one baseline, upright on the page as composed (a page turned by its
// Rotate included), without a space or a gap between them. A word with a
// glyph the font does not map to Unicode is dropped.
func words(gs []Glyph, dev matrix) []Word {
	type placed struct {
		g              Glyph
		ox, oy, x1, sz float64
		box            [4]float64
	}
	var out []Word
	var cur []placed
	flush := func() {
		defer func() { cur = cur[:0] }()
		if len(cur) == 0 {
			return
		}
		var b strings.Builder
		x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
		for _, p := range cur {
			if p.g.Text == "" || strings.ContainsRune(p.g.Text, unicode.ReplacementChar) {
				return
			}
			b.WriteString(p.g.Text)
			x0, y0, x1, y1 = min(x0, p.box[0]), min(y0, p.box[1]), max(x1, p.box[2]), max(y1, p.box[3])
		}
		t := strings.TrimSpace(b.String())
		if t == "" || math.IsNaN(x0+y0+x1+y1) || math.IsInf(x0+y0+x1+y1, 0) {
			return
		}
		out = append(out, Word{x0, y0, x1, y1, t})
	}
	for _, g := range gs {
		M := g.m.mul(dev)
		upright := M[0] > 0 && M[3] < 0 && math.Abs(M[1]) < 1e-3*M[0] && math.Abs(M[2]) < -1e-3*M[3]
		sz := g.size * -M[3]
		if !upright || !(sz > 0) || (strings.TrimSpace(g.Text) == "" && g.Text != "") {
			flush()
			continue
		}
		p := placed{g: g, ox: M[4], oy: M[5], x1: M[4] + g.adv*M[0], sz: sz}
		p.box = [4]float64{p.ox, p.oy - 0.8*sz, p.x1, p.oy + 0.2*sz}
		if n := len(cur); n > 0 {
			q := cur[n-1]
			gap := p.ox - q.x1
			if math.Abs(p.oy-q.oy) > 0.3*q.sz || gap > 0.25*q.sz || gap < -0.5*q.sz {
				flush()
			}
		}
		cur = append(cur, p)
	}
	flush()
	return out
}

// xobject draws a form or, when composing a scan, records an image.
func (in *interp) xobject(name Name, res Dict) error {
	xo, _ := in.r.Resolve(res["XObject"]).(Dict)
	s, ok := in.r.Resolve(xo[name]).(*Stream)
	if !ok {
		return nil
	}
	if s.Dict["Subtype"] == Name("Image") {
		in.record(s, res)
		return nil
	}
	return in.form(name, res)
}

func (in *interp) record(s *Stream, res Dict) {
	if !in.collect || len(in.draws) >= maxDraws {
		return
	}
	in.draws = append(in.draws, imageDraw{m: in.g.ctm, s: s, res: res, fill: in.g.fill})
}

// inlineKeys expands the abbreviations of an inline image's dictionary.
var inlineKeys = map[Name]Name{"W": "Width", "H": "Height", "BPC": "BitsPerComponent", "CS": "ColorSpace", "F": "Filter", "DP": "DecodeParms", "IM": "ImageMask", "D": "Decode", "I": "Interpolate", "L": "Length"}

var inlineNames = map[Name]Name{"G": "DeviceGray", "RGB": "DeviceRGB", "CMYK": "DeviceCMYK", "I": "Indexed", "AHx": "ASCIIHexDecode", "A85": "ASCII85Decode", "Fl": "FlateDecode", "DCT": "DCTDecode", "CCF": "CCITTFaxDecode", "RL": "RunLengthDecode", "LZW": "LZWDecode"}

// inlineImage reads "… ID <data> EI" after BI: its dictionary and data.
func inlineImage(l *lexer) *Stream {
	d := Dict{}
	for i := 0; i < 64; i++ {
		save := l.pos
		t := l.token()
		if k, ok := t.(keyword); ok && k == "ID" {
			break
		}
		if _, ok := t.(eof); ok {
			return nil
		}
		key, ok := t.(Name)
		if !ok {
			l.pos = save + 1
			continue
		}
		v, err := l.object(false, 0)
		if err != nil {
			return nil
		}
		if full, ok := inlineKeys[key]; ok {
			key = full
		}
		switch x := v.(type) {
		case Name:
			if full, ok := inlineNames[x]; ok {
				v = full
			}
		case Array:
			for j, e := range x {
				if n, ok := e.(Name); ok {
					if full, ok := inlineNames[n]; ok {
						x[j] = full
					}
				}
			}
		}
		d[key] = v
	}
	start := l.pos + 1
	for i := start; i+1 < len(l.b); i++ {
		if l.b[i] == 'E' && l.b[i+1] == 'I' && i > 0 && isWhite(l.b[i-1]) && (i+2 >= len(l.b) || isWhite(l.b[i+2])) {
			l.pos = i + 2
			if start > i {
				return nil
			}
			return &Stream{Dict: d, Raw: l.b[start : i-1]}
		}
	}
	l.pos = len(l.b)
	return nil
}

// setFill sets the grey a stencil mask paints with, from a colour operator
// of n operands (grey, RGB or CMYK).
func (in *interp) setFill(ops []any, n int) {
	if len(ops) < n {
		return
	}
	c := make([]float64, n)
	for i := range c {
		c[i] = numArg(ops, len(ops)-n+i)
	}
	kind := map[int]byte{1: 'G', 3: 'R', 4: 'K'}[n]
	in.g.fill = colourSpace{n: n, kind: kind}.grey(c)
}
