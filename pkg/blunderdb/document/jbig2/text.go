package jbig2

// textParams are the parameters of the text region decoding procedure
// (6.4), arithmetic only.
type textParams struct {
	w, h       int
	refine     bool
	strips     int
	corner     int // bit 0: top, bit 1: right
	transposed bool
	op         int
	defpix     int
	dsoffset   int
	rtemplate  int
	rat        [2][2]int
	ninst      int
	idLen      int
	syms       []*bitmap
}

// iaContexts are the contexts of the integer and symbol-ID decoders, which a
// symbol dictionary shares with the text regions it decodes.
type iaContexts struct {
	dt, fs, ds, it, ri, rdw, rdh, rdx, rdy, dh, dw, ex, ai contexts
	id                                                     contexts
}

func newIA(idLen int) *iaContexts {
	c := func() contexts { return make(contexts, 512) }
	return &iaContexts{dt: c(), fs: c(), ds: c(), it: c(), ri: c(), rdw: c(), rdh: c(), rdx: c(), rdy: c(),
		dh: c(), dw: c(), ex: c(), ai: c(), id: make(contexts, 2<<idLen)}
}

func (dc *decoder) integer(d *mq, cx contexts, what string) (int, error) {
	v, oob := d.integer(cx)
	if oob {
		return 0, malformed("out-of-band %s", what)
	}
	return v, nil
}

func (dc *decoder) decodeText(d *mq, ia *iaContexts, gr contexts, p *textParams) (*bitmap, error) {
	out := newBitmap(p.w, p.h)
	if p.defpix == 1 {
		out.fill(1)
	}
	v, err := dc.integer(d, ia.dt, "strip")
	if err != nil {
		return nil, err
	}
	stript, firsts := -v, 0
	top, right := p.corner&1 != 0, p.corner&2 != 0
	for n := 0; n < p.ninst; {
		dt, err := dc.integer(d, ia.dt, "strip")
		if err != nil {
			return nil, err
		}
		stript += dt
		dfs, err := dc.integer(d, ia.fs, "first S")
		if err != nil {
			return nil, err
		}
		firsts += dfs
		curs := firsts
		for {
			curt := 0
			if p.strips > 1 {
				if curt, err = dc.integer(d, ia.it, "T"); err != nil {
					return nil, err
				}
			}
			t := stript*p.strips + curt
			id := d.id(ia.id, p.idLen)
			if id >= len(p.syms) {
				return nil, malformed("symbol %d of %d", id, len(p.syms))
			}
			ib := p.syms[id]
			ri := 0
			if p.refine {
				if ri, err = dc.integer(d, ia.ri, "refinement"); err != nil {
					return nil, err
				}
			}
			if ri != 0 {
				var r [4]int
				for k, cx := range []contexts{ia.rdw, ia.rdh, ia.rdx, ia.rdy} {
					if r[k], err = dc.integer(d, cx, "refinement delta"); err != nil {
						return nil, err
					}
				}
				rp := refineParams{w: ib.w + r[0], h: ib.h + r[1], template: p.rtemplate, ref: ib,
					dx: r[0]>>1 + r[2], dy: r[1]>>1 + r[3], at: p.rat}
				if err := dc.symbolSize(rp.w, rp.h); err != nil {
					return nil, err
				}
				if ib, err = decodeRefine(d, gr, rp, dc.check); err != nil {
					return nil, err
				}
			}
			wi, hi := ib.w, ib.h
			if !p.transposed && right {
				curs += wi - 1
			} else if p.transposed && !top {
				curs += hi - 1
			}
			x, y := curs, t
			if p.transposed {
				x, y = t, curs
			}
			if right {
				x -= wi - 1
			}
			if !top {
				y -= hi - 1
			}
			if err := dc.place(1, wi*hi); err != nil {
				return nil, err
			}
			out.compose(ib, x, y, p.op)
			if !p.transposed && !right {
				curs += wi - 1
			} else if p.transposed && top {
				curs += hi - 1
			}
			n++
			ds, oob := d.integer(ia.ds)
			if oob || n >= p.ninst {
				break
			}
			curs += ds + p.dsoffset
		}
	}
	return out, nil
}
