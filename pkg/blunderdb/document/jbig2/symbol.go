package jbig2

// decodeSymbolDict decodes the new symbols of a dictionary (6.5), arithmetic
// only, and keeps those it exports.
func (dc *decoder) decodeSymbolDict(s *segment) error {
	p := &s.sd
	var in []*bitmap
	for _, t := range s.refs {
		if t.typ == segSymbolDict {
			in = append(in, t.exported...)
		}
	}
	if len(in) != p.nin {
		return malformed("symbol dictionary %d: %d input symbols, %d announced", s.num, len(in), p.nin)
	}
	if err := dc.holdSymbols(int64(minCost) * int64(p.nin+p.nexp)); err != nil {
		return err
	}
	d := newMQ(s.data)
	ia := newIA(p.idLen)
	gb, gr := make(contexts, genericCX[p.template]), make(contexts, refineCX)
	if p.used {
		last := lastDict(s)
		copy(gb, last.gb)
		copy(gr, last.gr)
	}
	all := append(in[:len(in):len(in)], make([]*bitmap, 0, p.nnew)...)
	gp := genericParams{template: p.template, at: p.at}
	hc := 0
	for classes := 0; len(all) < p.nin+p.nnew; classes++ {
		if classes >= p.nnew {
			return malformed("symbol dictionary %d: height classes without symbols", s.num)
		}
		dh, err := dc.integer(d, ia.dh, "height class")
		if err != nil {
			return err
		}
		if hc += dh; hc < 0 || hc > maxSide {
			return malformed("a height class of %d", hc)
		}
		w := 0
		for {
			dw, oob := d.integer(ia.dw)
			if oob {
				break
			}
			if len(all) >= p.nin+p.nnew {
				return malformed("symbol dictionary %d decodes more than %d symbols", s.num, p.nnew)
			}
			if w += dw; w < 0 || w > maxSide {
				return malformed("a symbol %d pixels wide", w)
			}
			if err := dc.symbolSize(w, hc); err != nil {
				return err
			}
			if err := dc.check(); err != nil {
				return err
			}
			var bm *bitmap
			switch {
			case !p.refagg:
				gp.w, gp.h = w, hc
				bm, err = decodeGeneric(d, gb, gp, dc.check)
			default:
				bm, err = dc.aggregate(d, ia, gr, p, all, w, hc)
			}
			if err != nil {
				return err
			}
			all = append(all, bm)
		}
	}
	// The export flags: runs, alternately not exported and exported.
	exp := make([]*bitmap, 0, p.nexp)
	on := false
	for i, runs := 0, 0; i < len(all); runs++ {
		n, oob := d.integer(ia.ex)
		if oob || n < 0 || n > len(all)-i || runs > 2*len(all)+2 {
			return malformed("symbol dictionary %d: export run of %d", s.num, n)
		}
		if on {
			exp = append(exp, all[i:i+n]...)
		}
		i += n
		on = !on
	}
	if len(exp) != p.nexp {
		return malformed("symbol dictionary %d exports %d symbols, %d announced", s.num, len(exp), p.nexp)
	}
	s.exported = exp
	if p.retained {
		s.gb, s.gr = gb, gr
	}
	return nil
}

// aggregate decodes a symbol made by refining one symbol, or by placing
// several (6.5.8.2).
func (dc *decoder) aggregate(d *mq, ia *iaContexts, gr contexts, p *sdParams, syms []*bitmap, w, h int) (*bitmap, error) {
	n, err := dc.integer(d, ia.ai, "instance count")
	if err != nil {
		return nil, err
	}
	if n < 1 {
		return nil, malformed("a symbol aggregating %d symbols", n)
	}
	if n > maxInstances-dc.instances {
		return nil, unsupported("more than %d symbols placed", maxInstances)
	}
	if n == 1 {
		id := d.id(ia.id, p.idLen)
		if id >= len(syms) {
			return nil, malformed("symbol %d of %d", id, len(syms))
		}
		rdx, err := dc.integer(d, ia.rdx, "refinement delta")
		if err != nil {
			return nil, err
		}
		rdy, err := dc.integer(d, ia.rdy, "refinement delta")
		if err != nil {
			return nil, err
		}
		return decodeRefine(d, gr, refineParams{w: w, h: h, template: p.rtemplate, ref: syms[id], dx: rdx, dy: rdy, at: p.rat}, dc.check)
	}
	tp := &textParams{w: w, h: h, refine: true, strips: 1, corner: 1, op: opOR, rtemplate: p.rtemplate,
		rat: p.rat, ninst: n, idLen: p.idLen, syms: syms}
	return dc.decodeText(d, ia, gr, tp)
}
