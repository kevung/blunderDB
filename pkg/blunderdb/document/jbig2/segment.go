package jbig2

import "fmt"

// Bounds the parser sets before anything proportional to them is allocated.
const (
	// MaxPixels bounds a page, and any one region or symbol of it.
	MaxPixels = 1 << 26
	// maxSide bounds a side of a page, a region or a symbol.
	maxSide = 1 << 20
	// maxSymbols bounds the symbols a dictionary holds (its input and its
	// new ones) and those a text region draws from.
	maxSymbols = 1 << 20
	// maxInstances bounds the symbols a text region places.
	maxInstances = 1 << 22
	// maxStreamPixels bounds what a stream holds in memory at once: its
	// page and regions, and apart the symbols of its dictionaries.
	maxStreamPixels = 1 << 28
)

// Segment types (7.3).
const (
	segSymbolDict         = 0
	segTextIntermediate   = 4
	segText               = 6
	segTextLossless       = 7
	segPatternDict        = 16
	segHalftoneIntermed   = 20
	segHalftone           = 22
	segHalftoneLossless   = 23
	segGenericIntermed    = 36
	segGeneric            = 38
	segGenericLossless    = 39
	segRefineIntermediate = 40
	segRefine             = 42
	segRefineLossless     = 43
	segPageInfo           = 48
	segEndOfPage          = 49
	segEndOfStripe        = 50
	segEndOfFile          = 51
	segProfiles           = 52
	segTables             = 53
	segExtension          = 62
)

type regionInfo struct{ w, h, x, y, op int }

type sdParams struct {
	refagg          bool
	used, retained  bool
	template        int
	rtemplate       int
	at              [4][2]int
	rat             [2][2]int
	nexp, nnew, nin int
	idLen           int
}

type segment struct {
	num  uint32
	typ  int
	refs []*segment
	data []byte // what follows the fixed fields of the segment's header

	region regionInfo
	gp     genericParams
	rp     refineParams
	sd     sdParams
	tp     textParams

	// Set while decoding.
	exported []*bitmap
	gb, gr   contexts
	result   *bitmap
}

// plan is a stream parsed, every header read and bounded, nothing decoded.
type plan struct {
	segs   []*segment
	w, h   int
	defpix int
	pixels int64 // pixels of the page and its regions
	info   Info
}

type reader struct {
	b []byte
	p int
}

func (r *reader) need(n int) error {
	if n < 0 || len(r.b)-r.p < n {
		return malformed("%d bytes where %d remain", n, len(r.b)-r.p)
	}
	return nil
}

func (r *reader) u8() int { r.p++; return int(r.b[r.p-1]) }

func (r *reader) s8() int { return int(int8(r.u8())) }

func (r *reader) u16() int { r.p += 2; return int(r.b[r.p-2])<<8 | int(r.b[r.p-1]) }

func (r *reader) u32() uint32 {
	r.p += 4
	b := r.b[r.p-4:]
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
}

// parse reads the segments of globals then of data, in one pass, and
// refuses what the package doc lists before anything is decoded. width and
// height, when positive, are the dimensions the container announces.
func parse(globals, data []byte, width, height int) (*plan, error) {
	pl := &plan{}
	seen := map[uint32]*segment{}
	for i, stream := range [][]byte{globals, data} {
		r := &reader{b: stream}
		for r.p < len(r.b) {
			s, err := pl.header(r, seen, i == 0, width, height)
			if err != nil {
				return pl, err
			}
			if s.typ == segEndOfPage || s.typ == segEndOfFile {
				break
			}
			seen[s.num] = s
			if decoded(s.typ) {
				pl.segs = append(pl.segs, s)
			}
		}
	}
	if pl.w == 0 {
		return pl, malformed("no page information segment")
	}
	if pl.pixels > maxStreamPixels {
		return pl, unsupported("a page and regions of %d pixels, at most %d", pl.pixels, maxStreamPixels)
	}
	pl.info.Width, pl.info.Height = pl.w, pl.h
	return pl, nil
}

// header reads one segment header and the fixed fields of its data; the
// data of a segment to decode is kept, past those fields.
func (pl *plan) header(r *reader, seen map[uint32]*segment, global bool, width, height int) (*segment, error) {
	if err := r.need(6); err != nil {
		return nil, err
	}
	s := &segment{num: r.u32()}
	flags := r.u8()
	s.typ = flags & 63
	c := r.u8()
	n := c >> 5
	switch {
	case n == 7:
		r.p--
		if err := r.need(4); err != nil {
			return nil, err
		}
		n = int(r.u32() & 0x1fffffff)
		if err := r.need((n + 8) / 8); err != nil {
			return nil, err
		}
		r.p += (n + 8) / 8
	case n > 4:
		return nil, malformed("segment %d: referred-to count %d", s.num, n)
	}
	size := 1
	if s.num > 65536 {
		size = 4
	} else if s.num > 256 {
		size = 2
	}
	if err := r.need(n * size); err != nil {
		return nil, err
	}
	for k := 0; k < n; k++ {
		var ref uint32
		switch size {
		case 1:
			ref = uint32(r.u8())
		case 2:
			ref = uint32(r.u16())
		default:
			ref = r.u32()
		}
		t, ok := seen[ref]
		if !ok || ref >= s.num {
			return nil, malformed("segment %d refers to segment %d, not before it", s.num, ref)
		}
		s.refs = append(s.refs, t)
	}
	pa := 1
	if flags&0x40 != 0 {
		pa = 4
	}
	if err := r.need(pa + 4); err != nil {
		return nil, err
	}
	r.p += pa
	length := r.u32()
	if length == 0xffffffff {
		return nil, unsupported("segment %d of unknown length", s.num)
	}
	if err := r.need(int(length)); err != nil {
		return nil, fmt.Errorf("segment %d: %w", s.num, err)
	}
	body := &reader{b: r.b[r.p : r.p+int(length)]}
	r.p += int(length)
	pl.info.Segments++
	if err := pl.body(s, body, global, width, height); err != nil {
		return nil, err
	}
	s.data = body.b[body.p:]
	return s, nil
}

// decoded tells whether a segment of type t is decoded, not only parsed.
func decoded(t int) bool {
	switch t {
	case segSymbolDict, segTextIntermediate, segText, segTextLossless, segGenericIntermed, segGeneric,
		segGenericLossless, segRefineIntermediate, segRefine, segRefineLossless:
		return true
	}
	return false
}

func (pl *plan) body(s *segment, r *reader, global bool, width, height int) error {
	switch s.typ {
	case segPageInfo:
		return pl.pageInfo(s, r, global, width, height)
	case segSymbolDict:
		return pl.symbolDict(s, r)
	case segTextIntermediate, segText, segTextLossless, segGenericIntermed, segGeneric, segGenericLossless,
		segRefineIntermediate, segRefine, segRefineLossless:
		if global {
			return malformed("a region segment in the globals")
		}
		if pl.w == 0 {
			return malformed("segment %d: a region before the page information", s.num)
		}
		if err := pl.regionInfo(s, r); err != nil {
			return err
		}
		switch s.typ {
		case segTextIntermediate, segText, segTextLossless:
			return pl.textRegion(s, r)
		case segGenericIntermed, segGeneric, segGenericLossless:
			return pl.genericRegion(s, r)
		}
		return pl.refineRegion(s, r)
	case segHalftoneIntermed, segHalftone, segHalftoneLossless:
		return unsupported("a halftone region (segment %d)", s.num)
	case segExtension:
		if err := r.need(4); err != nil {
			return err
		}
		if t := r.u32(); t&0x80000000 != 0 {
			return unsupported("extension %#x (segment %d), marked necessary", t, s.num)
		}
	case segPatternDict, segTables, segProfiles, segEndOfStripe, segEndOfPage, segEndOfFile:
		// A pattern dictionary serves only halftones and a table only
		// Huffman coding, both refused where they are used.
	default:
		return malformed("segment %d of type %d", s.num, s.typ)
	}
	return nil
}

func (pl *plan) pageInfo(s *segment, r *reader, global bool, width, height int) error {
	if global || pl.w != 0 {
		return malformed("segment %d: a second page, or a page in the globals", s.num)
	}
	if err := r.need(19); err != nil {
		return err
	}
	w, h := r.u32(), r.u32()
	r.p += 8
	flags := r.u8()
	striping := r.u16()
	if h == 0xffffffff {
		if striping&0x8000 == 0 || height <= 0 {
			return malformed("a page of unknown height, not striped")
		}
		h = uint32(height)
	}
	if w == 0 || h == 0 || w > maxSide || h > maxSide || int64(w)*int64(h) > MaxPixels {
		return unsupported("a page of %d×%d pixels, at most %d", w, h, MaxPixels)
	}
	if (width > 0 && int(w) != width) || (height > 0 && int(h) != height) {
		return unsupported("the page is %d×%d, its container says %d×%d", w, h, width, height)
	}
	pl.w, pl.h = int(w), int(h)
	pl.defpix = flags >> 2 & 1
	pl.pixels += int64(w) * int64(h)
	return nil
}

func (pl *plan) regionInfo(s *segment, r *reader) error {
	if err := r.need(17); err != nil {
		return err
	}
	w, h, x, y := r.u32(), r.u32(), r.u32(), r.u32()
	op := r.u8() & 7
	if op > opREPLACE {
		return malformed("segment %d: combination operator %d", s.num, op)
	}
	if w > maxSide || h > maxSide || int64(w)*int64(h) > MaxPixels {
		return unsupported("a region of %d×%d pixels, at most %d", w, h, MaxPixels)
	}
	// Coordinates are unsigned: past the page, the region is clipped away.
	s.region = regionInfo{int(w), int(h), int(min(x, maxSide*2)), int(min(y, maxSide*2)), op}
	pl.pixels += int64(w) * int64(h)
	pl.info.Regions++
	return nil
}

func readAT(r *reader, at [][2]int) error {
	if err := r.need(2 * len(at)); err != nil {
		return err
	}
	for i := range at {
		at[i][0], at[i][1] = r.s8(), r.s8()
	}
	return nil
}

func (pl *plan) genericRegion(s *segment, r *reader) error {
	if err := r.need(1); err != nil {
		return err
	}
	f := r.u8()
	p := &s.gp
	p.w, p.h = s.region.w, s.region.h
	p.mmr, p.template, p.tpgdon = f&1 != 0, f>>1&3, f&8 != 0
	if f&16 != 0 {
		return unsupported("a generic region with an extended template (segment %d)", s.num)
	}
	if p.mmr {
		return nil
	}
	n := 1
	if p.template == 0 {
		n = 4
	}
	return readAT(r, p.at[:n])
}

func (pl *plan) refineRegion(s *segment, r *reader) error {
	if err := r.need(1); err != nil {
		return err
	}
	f := r.u8()
	p := &s.rp
	p.w, p.h = s.region.w, s.region.h
	p.template, p.tpgron = f&1, f&2 != 0
	if p.template == 0 {
		if err := readAT(r, p.at[:]); err != nil {
			return err
		}
	}
	if len(s.refs) > 1 {
		return malformed("refinement segment %d refers to %d segments", s.num, len(s.refs))
	}
	for _, t := range s.refs {
		if t.typ != segGenericIntermed && t.typ != segRefineIntermediate && t.typ != segTextIntermediate {
			return malformed("refinement segment %d refers to a segment of type %d", s.num, t.typ)
		}
	}
	return nil
}

func ceilLog2(n int) int {
	k := 0
	for 1<<k < n {
		k++
	}
	return k
}

// exports counts the symbols the dictionaries s refers to export.
func exports(s *segment) int {
	n := 0
	for _, t := range s.refs {
		if t.typ == segSymbolDict {
			n += t.sd.nexp
		}
	}
	return n
}

func (pl *plan) symbolDict(s *segment, r *reader) error {
	if err := r.need(2); err != nil {
		return err
	}
	f := r.u16()
	p := &s.sd
	if f&1 != 0 {
		return unsupported("a Huffman-coded symbol dictionary (segment %d)", s.num)
	}
	p.refagg, p.used, p.retained = f&2 != 0, f&0x100 != 0, f&0x200 != 0
	p.template, p.rtemplate = f>>10&3, f>>12&1
	n := 1
	if p.template == 0 {
		n = 4
	}
	if err := readAT(r, p.at[:n]); err != nil {
		return err
	}
	if p.refagg && p.rtemplate == 0 {
		if err := readAT(r, p.rat[:]); err != nil {
			return err
		}
	}
	if err := r.need(8); err != nil {
		return err
	}
	nexp, nnew := r.u32(), r.u32()
	p.nin = exports(s)
	if nnew > maxSymbols || p.nin+int(nnew) > maxSymbols {
		return unsupported("a symbol dictionary of %d+%d symbols, at most %d", p.nin, nnew, maxSymbols)
	}
	p.nexp, p.nnew = int(nexp), int(nnew)
	if p.nexp > p.nin+p.nnew {
		return malformed("symbol dictionary %d exports %d of %d symbols", s.num, nexp, p.nin+p.nnew)
	}
	p.idLen = ceilLog2(p.nin + p.nnew)
	if p.used {
		last := lastDict(s)
		if last == nil || !last.sd.retained || last.sd.template != p.template || last.sd.rtemplate != p.rtemplate {
			return malformed("symbol dictionary %d reuses contexts no dictionary it refers to retains", s.num)
		}
	}
	pl.info.Symbols += p.nnew
	return nil
}

func lastDict(s *segment) *segment {
	for i := len(s.refs) - 1; i >= 0; i-- {
		if s.refs[i].typ == segSymbolDict {
			return s.refs[i]
		}
	}
	return nil
}

func (pl *plan) textRegion(s *segment, r *reader) error {
	if err := r.need(2); err != nil {
		return err
	}
	f := r.u16()
	if f&1 != 0 {
		return unsupported("a Huffman-coded text region (segment %d)", s.num)
	}
	p := &s.tp
	p.w, p.h = s.region.w, s.region.h
	p.refine = f&2 != 0
	p.strips = 1 << (f >> 2 & 3)
	p.corner = f >> 4 & 3
	p.transposed = f&0x40 != 0
	p.op = f >> 7 & 3
	p.defpix = f >> 9 & 1
	p.dsoffset = f >> 10 & 31
	if p.dsoffset >= 16 {
		p.dsoffset -= 32
	}
	p.rtemplate = f >> 15 & 1
	if p.refine && p.rtemplate == 0 {
		if err := readAT(r, p.rat[:]); err != nil {
			return err
		}
	}
	if err := r.need(4); err != nil {
		return err
	}
	n := r.u32()
	nsyms := exports(s)
	if n > maxInstances {
		return unsupported("a text region of %d symbols, at most %d", n, maxInstances)
	}
	if nsyms > maxSymbols {
		return unsupported("a text region drawing from %d symbols, at most %d", nsyms, maxSymbols)
	}
	if n > 0 && nsyms == 0 {
		return malformed("text region %d places symbols from no dictionary", s.num)
	}
	p.ninst, p.idLen = int(n), ceilLog2(nsyms)
	pl.info.Instances += p.ninst
	return nil
}
