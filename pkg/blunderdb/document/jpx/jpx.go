package jpx

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"

	"github.com/d-fuji/jpeg2000"
)

// MaxPixels is the largest image, in pixels of its reference grid.
const MaxPixels = 12_000_000

func init() {
	// The decoder's own bound applies to any caller that reaches it.
	jpeg2000.MaxImagePixels = MaxPixels
}

var (
	// ErrUnsupported: a valid JPEG 2000 image this reader does not take.
	ErrUnsupported = errors.New("jpeg 2000: unsupported")
	// ErrMalformed: bytes that are not a JPEG 2000 image.
	ErrMalformed = errors.New("jpeg 2000: malformed")
	// ErrBudget: the pixels a document may decode in JPEG 2000 are spent.
	ErrBudget = errors.New("jpeg 2000: pixel budget of the document exhausted")
)

func unsupported(format string, a ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrUnsupported}, a...)...)
}

func malformed(format string, a ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrMalformed}, a...)...)
}

// Info is what the headers of an image say.
type Info struct {
	Width, Height int
	Components    int
	// Tiles is the number of tiles; Reversible is true when the first
	// coding style names the 5/3 transform.
	Tiles      int
	Reversible bool
}

// Budget bounds the pixels a document decodes in JPEG 2000, since a decode
// cannot be cancelled. The zero value allows nothing; a nil *Budget allows
// everything.
type Budget struct{ Left int64 }

// Decode inspects data, spends its pixels from budget, and decodes it.
// width and height, when positive, are the dimensions the container
// announces: another reference grid is refused.
func Decode(data []byte, width, height int, budget *Budget) (image.Image, Info, error) {
	info, cs, err := inspect(data)
	if err != nil {
		return nil, info, err
	}
	if (width > 0 && width != info.Width) || (height > 0 && height != info.Height) {
		return nil, info, unsupported("the image is %d×%d, its container says %d×%d", info.Width, info.Height, width, height)
	}
	if budget != nil {
		n := int64(info.Width) * int64(info.Height)
		if n > budget.Left {
			return nil, info, ErrBudget
		}
		budget.Left -= n
	}
	img, err := decodeSafe(cs)
	return img, info, err
}

// Inspect walks the headers of data and refuses what the package doc lists.
func Inspect(data []byte) (Info, error) {
	info, _, err := inspect(data)
	return info, err
}

func decodeSafe(cs []byte) (img image.Image, err error) {
	defer func() {
		if e := recover(); e != nil {
			img, err = nil, malformed("%v", e)
		}
	}()
	img, err = jpeg2000.Decode(bytes.NewReader(cs))
	if err != nil {
		var u jpeg2000.UnsupportedError
		if errors.As(err, &u) {
			return nil, unsupported("%v", err)
		}
		return nil, malformed("%v", err)
	}
	return img, nil
}

var jp2Signature = []byte{0, 0, 0, 12, 'j', 'P', ' ', ' ', 0x0d, 0x0a, 0x87, 0x0a}

// codestream returns the codestream of a JP2 file (its first jp2c box) or
// data itself when it is a bare codestream.
func codestream(data []byte) ([]byte, error) {
	if len(data) >= 4 && data[0] == 0xFF && data[1] == 0x4F && data[2] == 0xFF && data[3] == 0x51 {
		return data, nil
	}
	if !bytes.HasPrefix(data, jp2Signature) {
		return nil, malformed("neither a JP2 file nor a codestream")
	}
	var cs []byte
	var walk func(b []byte, depth int) error
	walk = func(b []byte, depth int) error {
		for pos := 0; pos+8 <= len(b); {
			n := int(binary.BigEndian.Uint32(b[pos:]))
			typ := string(b[pos+4 : pos+8])
			hdr := 8
			switch n {
			case 0:
				n = len(b) - pos
			case 1:
				if pos+16 > len(b) || binary.BigEndian.Uint32(b[pos+8:]) != 0 {
					return malformed("a box too large")
				}
				n, hdr = int(binary.BigEndian.Uint32(b[pos+12:])), 16
			}
			if n < hdr || n > len(b)-pos {
				return malformed("a box of %d bytes", n)
			}
			body := b[pos+hdr : pos+n]
			switch typ {
			case "jp2h":
				if depth > 0 {
					return malformed("nested headers")
				}
				if err := walk(body, depth+1); err != nil {
					return err
				}
			case "pclr", "cmap", "cdef":
				return unsupported("a JP2 %s box (palette, component mapping or channel definition)", typ)
			case "jp2c":
				if cs == nil {
					cs = body
				}
			}
			pos += n
		}
		return nil
	}
	if err := walk(data, 0); err != nil {
		return nil, err
	}
	if cs == nil {
		return nil, malformed("no codestream box")
	}
	return cs, nil
}

const (
	mSOC = 0xFF4F
	mCAP = 0xFF50
	mSIZ = 0xFF51
	mCOD = 0xFF52
	mCOC = 0xFF53
	mSOT = 0xFF90
	mSOD = 0xFF93
	mEOC = 0xFFD9
)

func inspect(data []byte) (Info, []byte, error) {
	var info Info
	cs, err := codestream(data)
	if err != nil {
		return info, nil, err
	}
	be16 := func(p int) int { return int(binary.BigEndian.Uint16(cs[p:])) }
	if len(cs) < 6 || be16(0) != mSOC || be16(2) != mSIZ {
		return info, nil, malformed("no SIZ marker")
	}
	l := be16(4)
	if l < 41 || 4+l > len(cs) {
		return info, nil, malformed("a SIZ of %d bytes", l)
	}
	s := cs[6 : 4+l]
	u32 := func(i int) int64 { return int64(binary.BigEndian.Uint32(s[i:])) }
	rsiz := int(binary.BigEndian.Uint16(s))
	xs, ys, xo, yo := u32(2), u32(6), u32(10), u32(14)
	xt, yt, xto, yto := u32(18), u32(22), u32(26), u32(30)
	nc := int(binary.BigEndian.Uint16(s[34:]))
	if rsiz&0x4000 != 0 {
		return info, nil, unsupported("high-throughput JPEG 2000")
	}
	if nc != 1 && nc != 3 {
		return info, nil, unsupported("%d components (one grey or three colour are read)", nc)
	}
	if len(s) < 36+3*nc {
		return info, nil, malformed("a SIZ too short for its components")
	}
	for i := 0; i < nc; i++ {
		if s[37+3*i] != 1 || s[38+3*i] != 1 {
			return info, nil, unsupported("subsampled components")
		}
	}
	if xs <= xo || ys <= yo || xt <= 0 || yt <= 0 || xto > xo || yto > yo || xto+xt <= xo || yto+yt <= yo {
		return info, nil, malformed("an inconsistent reference grid")
	}
	w, h := xs-xo, ys-yo
	if w > MaxPixels || h > MaxPixels || w*h > MaxPixels {
		return info, nil, unsupported("an image of %d×%d pixels (at most %d)", w, h, MaxPixels)
	}
	info = Info{Width: int(w), Height: int(h), Components: nc}
	ntx, nty := (xs-xto+xt-1)/xt, (ys-yto+yt-1)/yt
	tiles := ntx * nty
	if tiles > 65535 || tiles > max(1, w*h/(64*64)) {
		return info, nil, unsupported("%d tiles for an image of %d×%d pixels", tiles, w, h)
	}
	info.Tiles = int(tiles)

	// A segment's coding style is checked wherever it appears: the main
	// header and each tile-part header may restate it.
	checkCod := func(seg []byte, first bool) error {
		if len(seg) < 10 {
			return malformed("a COD segment of %d bytes", len(seg))
		}
		if first {
			info.Reversible = seg[9] == 1
		}
		return checkSP(seg[5:], seg[0]&1 != 0)
	}
	checkCoc := func(seg []byte) error {
		skip := 1
		if nc > 256 {
			skip = 2
		}
		if len(seg) < skip+6 {
			return malformed("a COC segment of %d bytes", len(seg))
		}
		return checkSP(seg[skip+1:], seg[skip]&1 != 0)
	}

	pos := 4 + l
	sawCOD := false
	// markers reads the marker segments from pos up to the first SOT or
	// SOD, returning the marker it stopped on.
	markers := func(pos int) (int, int, error) {
		for {
			if pos+2 > len(cs) {
				return pos, mEOC, nil
			}
			m := be16(pos)
			if m == mSOT || m == mSOD || m == mEOC {
				return pos, m, nil
			}
			if m < 0xFF30 || pos+4 > len(cs) {
				return pos, 0, malformed("marker %#x", m)
			}
			if m >= 0xFF30 && m <= 0xFF3F { // markers without a segment
				pos += 2
				continue
			}
			n := be16(pos + 2)
			if n < 2 || pos+2+n > len(cs) {
				return pos, 0, malformed("a segment of %d bytes", n)
			}
			seg := cs[pos+4 : pos+2+n]
			var err error
			switch m {
			case mCAP:
				err = unsupported("high-throughput JPEG 2000")
			case mCOD:
				err = checkCod(seg, !sawCOD)
				sawCOD = true
			case mCOC:
				err = checkCoc(seg)
			}
			if err != nil {
				return pos, 0, err
			}
			pos += 2 + n
		}
	}
	pos, m, err := markers(pos)
	if err != nil {
		return info, nil, err
	}
	parts := 0
	for m == mSOT {
		if pos+12 > len(cs) || be16(pos+2) != 10 {
			return info, nil, malformed("a SOT segment")
		}
		parts++
		if parts > 65535 {
			return info, nil, unsupported("more than 65535 tile-parts")
		}
		if idx := be16(pos + 4); int64(idx) >= tiles {
			return info, nil, malformed("tile %d of %d", idx, tiles)
		}
		psot := int(binary.BigEndian.Uint32(cs[pos+6:]))
		start := pos
		if pos, m, err = markers(pos + 12); err != nil {
			return info, nil, err
		}
		if m != mSOD {
			return info, nil, malformed("a tile-part without data")
		}
		if psot == 0 {
			break
		}
		if psot < pos-start+2 || start+psot > len(cs) {
			// A truncated last tile-part is the decoder's to read.
			break
		}
		pos = start + psot
		if pos+2 > len(cs) {
			break
		}
		m = be16(pos)
	}
	if !sawCOD {
		return info, nil, malformed("no COD marker")
	}
	return info, cs, nil
}

// checkSP checks the code-block and precinct sizes of a coding style: seg
// starts at the number of decomposition levels.
func checkSP(sp []byte, precincts bool) error {
	if len(sp) < 5 {
		return malformed("a coding style of %d bytes", len(sp))
	}
	levels := int(sp[0])
	if levels > 32 {
		return malformed("%d decomposition levels", levels)
	}
	if sp[1]&0x0F < 2 || sp[2]&0x0F < 2 {
		return unsupported("a code-block under 16×16")
	}
	if sp[3]&0x40 != 0 {
		return unsupported("high-throughput JPEG 2000")
	}
	if precincts {
		if len(sp) < 5+levels+1 {
			return malformed("a coding style without its precincts")
		}
		for _, b := range sp[5 : 5+levels+1] {
			if b&0x0F < 5 || b>>4 < 5 {
				return unsupported("a precinct under 32×32")
			}
		}
	}
	return nil
}
