package jbig2

import (
	"context"
	"errors"
	"fmt"
)

var (
	// ErrUnsupported: a valid JBIG2 stream this decoder does not take.
	ErrUnsupported = errors.New("jbig2: unsupported")
	// ErrMalformed: bytes that are not a JBIG2 stream.
	ErrMalformed = errors.New("jbig2: malformed")
	// ErrBudget: the pixels a document may decode in JBIG2 are spent.
	ErrBudget = errors.New("jbig2: pixel budget of the document exhausted")
)

func unsupported(format string, a ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrUnsupported}, a...)...)
}

func malformed(format string, a ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrMalformed}, a...)...)
}

// Info is what the segment headers of a stream say.
type Info struct {
	Width, Height int
	// Segments counts the segments read, Regions the region segments;
	// Symbols the new symbols the dictionaries announce, Instances the
	// symbols the text regions announce they place.
	Segments, Regions, Symbols, Instances int
}

// Budget bounds the pixels a document decodes in JBIG2: its page and
// regions, each symbol of its dictionaries and each symbol its text regions
// place. The zero value allows nothing; a nil *Budget allows everything.
type Budget struct{ Left int64 }

// Page is a decoded page: rows of Stride bytes, the most significant bit
// first, a 1 bit black.
type Page struct {
	Width, Height, Stride int
	Data                  []byte
}

// Decode parses the stream of a PDF's JBIG2Decode filter (data, with the
// segments of its JBIG2Globals in globals, nil when it has none), spends the
// pixels of its page and regions from budget, and decodes it, checking ctx
// before each segment and each region. width and height, when positive, are
// the dimensions the image dictionary announces: another page is refused.
func Decode(ctx context.Context, data, globals []byte, width, height int, budget *Budget) (*Page, Info, error) {
	pl, err := parseSafe(globals, data, width, height)
	if err != nil {
		return nil, pl.info, err
	}
	dc := &decoder{ctx: ctx, budget: budget}
	if err := dc.spend(pl.pixels); err != nil {
		return nil, pl.info, err
	}
	m, err := dc.run(pl)
	if err != nil {
		return nil, pl.info, err
	}
	return &Page{Width: m.w, Height: m.h, Stride: m.stride, Data: m.b}, pl.info, nil
}

// Inspect parses the segment headers of a stream and refuses what the
// package doc lists, decoding nothing.
func Inspect(data, globals []byte, width, height int) (Info, error) {
	pl, err := parseSafe(globals, data, width, height)
	return pl.info, err
}

func parseSafe(globals, data []byte, width, height int) (pl *plan, err error) {
	defer func() {
		if e := recover(); e != nil {
			pl, err = &plan{}, malformed("%v", e)
		}
	}()
	return parse(globals, data, width, height)
}

type decoder struct {
	ctx     context.Context
	budget  *Budget
	symbols int64 // pixels of the symbols decoded
}

func (dc *decoder) check() error {
	if dc.ctx == nil {
		return nil
	}
	return dc.ctx.Err()
}

func (dc *decoder) spend(n int64) error {
	if dc.budget == nil {
		return nil
	}
	if n > dc.budget.Left {
		return ErrBudget
	}
	dc.budget.Left -= n
	return nil
}

// symbolSize bounds a symbol before it is decoded, and spends its pixels.
func (dc *decoder) symbolSize(w, h int) error {
	if w < 0 || h < 0 || w > maxSide || h > maxSide || int64(w)*int64(h) > MaxPixels {
		return malformed("a symbol of %d×%d pixels", w, h)
	}
	if dc.symbols += int64(w * h); dc.symbols > maxStreamPixels {
		return unsupported("symbols of more than %d pixels", maxStreamPixels)
	}
	return dc.spend(int64(max(w*h, 1)))
}

func (dc *decoder) run(pl *plan) (m *bitmap, err error) {
	defer func() {
		if e := recover(); e != nil {
			m, err = nil, malformed("%v", e)
		}
	}()
	page := newBitmap(pl.w, pl.h)
	if pl.defpix == 1 {
		page.fill(1)
	}
	for _, s := range pl.segs {
		if err := dc.check(); err != nil {
			return nil, err
		}
		if err := dc.segment(s, page); err != nil {
			return nil, err
		}
	}
	return page, nil
}

func (dc *decoder) segment(s *segment, page *bitmap) error {
	var m *bitmap
	var err error
	op := s.region.op
	switch s.typ {
	case segSymbolDict:
		return dc.decodeSymbolDict(s)
	case segTextIntermediate, segText, segTextLossless:
		p := s.tp
		for _, t := range s.refs {
			if t.typ == segSymbolDict {
				p.syms = append(p.syms, t.exported...)
			}
		}
		d := newMQ(s.data)
		m, err = dc.decodeText(d, newIA(p.idLen), make(contexts, refineCX), &p)
	case segGenericIntermed, segGeneric, segGenericLossless:
		if s.gp.mmr {
			m = decodeMMR(s.data, s.gp.w, s.gp.h)
		} else {
			m, err = decodeGeneric(newMQ(s.data), make(contexts, genericCX[s.gp.template]), s.gp, dc.check)
		}
	default: // refinement
		p := s.rp
		if len(s.refs) == 1 {
			p.ref = s.refs[0].result
			if p.ref == nil {
				return malformed("refinement segment %d of a region not decoded", s.num)
			}
		} else {
			p.ref = page.crop(s.region.x, s.region.y, p.w, p.h)
			op = opREPLACE
		}
		m, err = decodeRefine(newMQ(s.data), make(contexts, refineCX), p, dc.check)
	}
	if err != nil {
		return err
	}
	switch s.typ {
	case segTextIntermediate, segGenericIntermed, segRefineIntermediate:
		s.result = m
	default:
		page.compose(m, s.region.x, s.region.y, op)
	}
	return nil
}
