package j2k

// Tile decoding: geometry construction (resolutions, sub-bands,
// precincts, code-blocks per ISO/IEC 15444-1 Annex B), followed by
// packet parsing, Tier-1 coefficient decoding, dequantization and the
// inverse wavelet transform.

import "context"

type segment struct {
	data   []byte
	passes int
}

type codeblock struct {
	x0, y0, x1, y1 int // in sub-band coordinates
	seenIncl       bool
	lblock         int
	zbp            int
	totalPasses    int
	segs           []segment
}

// precBand is the portion of one sub-band covered by one precinct: a
// grid of code-blocks plus the two tag trees used by packet headers.
type precBand struct {
	ncbx, ncby int
	cbs        []*codeblock
	incl, zbps *tagTree
}

type band struct {
	orient         int // 0: LL, 1: HL, 2: LH, 3: HH
	x0, y0, x1, y1 int
	mb             int     // max bit-planes (guard bits + exponent - 1)
	step           float64 // quantization step (irreversible only)
	coeffI         []int32
	coeffF         []float32
}

func (b *band) w() int { return b.x1 - b.x0 }
func (b *band) h() int { return b.y1 - b.y0 }

type resolution struct {
	x0, y0, x1, y1 int
	ppx, ppy       uint // precinct size exponents
	npw, nph       int  // precinct grid size
	bands          []*band
	prec           [][]*precBand // [precinct][band]
}

type tileComp struct {
	idx            int
	x0, y0, x1, y1 int
	cp             *codingParams
	qp             *quantParams
	reversible     bool
	res            []*resolution
	samplesI       []int32
	samplesF       []float32
}

func (tc *tileComp) w() int { return tc.x1 - tc.x0 }
func (tc *tileComp) h() int { return tc.y1 - tc.y0 }

type tileDecoder struct {
	ctx            context.Context
	cs             *codestream
	td             *tileData
	th             *codHeader
	x0, y0, x1, y1 int // tile rectangle on the reference grid
	comps          []*tileComp
	pos            int // read position in td.data
}

var bandGain = [4]int{0, 1, 1, 2}

func newTileDecoder(cs *codestream, td *tileData) (*tileDecoder, error) {
	d := &tileDecoder{cs: cs, td: td, th: cs.tileHeader(td)}
	siz := &cs.siz
	p := td.index % cs.numXTiles
	q := td.index / cs.numXTiles
	d.x0 = max(siz.xtosiz+p*siz.xtsiz, siz.xosiz)
	d.y0 = max(siz.ytosiz+q*siz.ytsiz, siz.yosiz)
	d.x1 = min(siz.xtosiz+(p+1)*siz.xtsiz, siz.xsiz)
	d.y1 = min(siz.ytosiz+(q+1)*siz.ytsiz, siz.ysiz)

	d.comps = make([]*tileComp, len(siz.comps))
	for c := range siz.comps {
		if err := d.buildComp(c); err != nil {
			return nil, err
		}
	}
	return d, nil
}

func (d *tileDecoder) buildComp(c int) error {
	sc := &d.cs.siz.comps[c]
	cp := d.cs.codingFor(d.td, c)
	qp := d.cs.quantFor(d.td, c)
	if cp.cbStyle&cbBypass != 0 {
		return UnsupportedError("arithmetic coder bypass (lazy) mode")
	}
	tc := &tileComp{
		idx:        c,
		cp:         cp,
		qp:         qp,
		reversible: cp.transform == 1,
		x0:         ceilDiv(d.x0, sc.dx),
		y0:         ceilDiv(d.y0, sc.dy),
		x1:         ceilDiv(d.x1, sc.dx),
		y1:         ceilDiv(d.y1, sc.dy),
	}
	d.comps[c] = tc

	nl := cp.numDecomps
	tc.res = make([]*resolution, nl+1)
	for r := 0; r <= nl; r++ {
		s := uint(nl - r)
		res := &resolution{
			x0:  ceilDiv(tc.x0, 1<<s),
			y0:  ceilDiv(tc.y0, 1<<s),
			x1:  ceilDiv(tc.x1, 1<<s),
			y1:  ceilDiv(tc.y1, 1<<s),
			ppx: cp.precW[r],
			ppy: cp.precH[r],
		}
		tc.res[r] = res
		if res.x1 > res.x0 {
			res.npw = ceilDiv(res.x1, 1<<res.ppx) - res.x0>>res.ppx
		}
		if res.y1 > res.y0 {
			res.nph = ceilDiv(res.y1, 1<<res.ppy) - res.y0>>res.ppy
		}

		// Sub-bands of this resolution level.
		if r == 0 {
			res.bands = []*band{{orient: 0, x0: res.x0, y0: res.y0, x1: res.x1, y1: res.y1}}
		} else {
			nb := uint(nl - r + 1) // decomposition level of these bands
			res.bands = make([]*band, 3)
			for i, orient := range [3]int{1, 2, 3} {
				xo := orient & 1
				yo := orient >> 1
				res.bands[i] = &band{
					orient: orient,
					x0:     ceilDiv(tc.x0-(1<<(nb-1))*xo, 1<<nb),
					y0:     ceilDiv(tc.y0-(1<<(nb-1))*yo, 1<<nb),
					x1:     ceilDiv(tc.x1-(1<<(nb-1))*xo, 1<<nb),
					y1:     ceilDiv(tc.y1-(1<<(nb-1))*yo, 1<<nb),
				}
			}
		}
		for i, b := range res.bands {
			if err := d.quantizeBand(tc, b, r, i); err != nil {
				return err
			}
			if tc.reversible {
				b.coeffI = make([]int32, b.w()*b.h())
			} else {
				b.coeffF = make([]float32, b.w()*b.h())
			}
		}

		// Precincts and their code-block grids.
		ppbx, ppby := res.ppx, res.ppy
		if r > 0 {
			ppbx, ppby = ppbx-1, ppby-1
		}
		cbw := min(cp.cbW, ppbx)
		cbh := min(cp.cbH, ppby)
		res.prec = make([][]*precBand, res.npw*res.nph)
		pgx0 := (res.x0 >> res.ppx) << res.ppx
		pgy0 := (res.y0 >> res.ppy) << res.ppy
		for py := 0; py < res.nph; py++ {
			for px := 0; px < res.npw; px++ {
				prx0 := max(res.x0, pgx0+px<<res.ppx)
				pry0 := max(res.y0, pgy0+py<<res.ppy)
				prx1 := min(res.x1, pgx0+(px+1)<<res.ppx)
				pry1 := min(res.y1, pgy0+(py+1)<<res.ppy)
				pbs := make([]*precBand, len(res.bands))
				for bi, b := range res.bands {
					// Map the precinct rectangle into band coordinates.
					bx0, by0, bx1, by1 := prx0, pry0, prx1, pry1
					if r > 0 {
						xo := b.orient & 1
						yo := b.orient >> 1
						bx0 = ceilDiv(prx0-xo, 2)
						by0 = ceilDiv(pry0-yo, 2)
						bx1 = ceilDiv(prx1-xo, 2)
						by1 = ceilDiv(pry1-yo, 2)
					}
					bx0, by0 = max(bx0, b.x0), max(by0, b.y0)
					bx1, by1 = min(bx1, b.x1), min(by1, b.y1)
					pb := &precBand{}
					if bx1 > bx0 && by1 > by0 {
						cbx0, cby0 := bx0>>cbw, by0>>cbh
						cbx1, cby1 := ceilDiv(bx1, 1<<cbw), ceilDiv(by1, 1<<cbh)
						pb.ncbx, pb.ncby = cbx1-cbx0, cby1-cby0
						pb.cbs = make([]*codeblock, pb.ncbx*pb.ncby)
						for cy := 0; cy < pb.ncby; cy++ {
							for cx := 0; cx < pb.ncbx; cx++ {
								pb.cbs[cy*pb.ncbx+cx] = &codeblock{
									x0:     max(bx0, (cbx0+cx)<<cbw),
									y0:     max(by0, (cby0+cy)<<cbh),
									x1:     min(bx1, (cbx0+cx+1)<<cbw),
									y1:     min(by1, (cby0+cy+1)<<cbh),
									lblock: 3,
								}
							}
						}
						pb.incl = newTagTree(pb.ncbx, pb.ncby)
						pb.zbps = newTagTree(pb.ncbx, pb.ncby)
					}
					pbs[bi] = pb
				}
				res.prec[py*res.npw+px] = pbs
			}
		}
	}
	return nil
}

// quantizeBand computes the maximum bit-plane count and quantization
// step for a band from the QCD/QCC parameters (Annex E).
func (d *tileDecoder) quantizeBand(tc *tileComp, b *band, r, bi int) error {
	qp := tc.qp
	nl := tc.cp.numDecomps
	var eps, mant int
	switch qp.style {
	case 0, 2:
		idx := 0
		if r > 0 {
			idx = 3*(r-1) + bi + 1
		}
		if idx >= len(qp.exps) {
			return errFormat("too few quantization values for band")
		}
		eps = qp.exps[idx]
		if qp.style == 2 {
			mant = qp.mants[idx]
		}
	case 1:
		// Scalar derived: values for all bands derive from the LL entry.
		nb := nl
		if r > 0 {
			nb = nl - r + 1
		}
		eps = qp.exps[0] - nl + nb
		mant = qp.mants[0]
	}
	sc := &d.cs.siz.comps[tc.idx]
	b.mb = qp.guardBits + eps - 1
	if b.mb < 0 {
		b.mb = 0
	}
	if b.mb > 31 {
		return errFormat("bit-plane count out of range")
	}
	rb := sc.depth + bandGain[b.orient]
	b.step = float64(int64(1)<<uint(rb)) / float64(int64(1)<<uint(eps)) *
		(1 + float64(mant)/2048)
	return nil
}

// decode runs the full tile pipeline and leaves the reconstructed
// samples in each tileComp.
func (d *tileDecoder) decode() error {
	if err := d.readPackets(); err != nil {
		return err
	}
	for _, tc := range d.comps {
		for _, res := range tc.res {
			for bi, b := range res.bands {
				for _, pbs := range res.prec {
					for _, cb := range pbs[bi].cbs {
						if err := d.ctx.Err(); err != nil {
							return err
						}
						if err := decodeCodeblock(cb, b, tc); err != nil {
							return err
						}
					}
				}
			}
		}
		inverseDWT(tc)
	}
	return nil
}
