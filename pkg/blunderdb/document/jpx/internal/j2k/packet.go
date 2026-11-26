package j2k

import (
	"math/bits"
	"sort"
)

// Packet parsing (ISO/IEC 15444-1 B.9-B.10): iterate packets in the
// tile's progression order, decode each packet header, and attach the
// code-block data segments it contributes.

// pktRef identifies one precinct of one component-resolution, with the
// projected reference-grid position used by position-based orders.
type pktRef struct {
	c, r, p int
	x, y    int
}

func (d *tileDecoder) packetRefs() []pktRef {
	var refs []pktRef
	for c, tc := range d.comps {
		sc := &d.cs.siz.comps[c]
		nl := tc.cp.numDecomps
		for r, res := range tc.res {
			pgx0 := (res.x0 >> res.ppx) << res.ppx
			pgy0 := (res.y0 >> res.ppy) << res.ppy
			for py := 0; py < res.nph; py++ {
				for px := 0; px < res.npw; px++ {
					x := sc.dx * ((pgx0 + px<<res.ppx) << uint(nl-r))
					y := sc.dy * ((pgy0 + py<<res.ppy) << uint(nl-r))
					refs = append(refs, pktRef{
						c: c, r: r, p: py*res.npw + px,
						x: max(x, d.x0), y: max(y, d.y0),
					})
				}
			}
		}
	}
	return refs
}

func (d *tileDecoder) readPackets() error {
	refs := d.packetRefs()
	layers := d.th.numLayers

	emit := func(ref pktRef, layer int) error {
		if err := d.ctx.Err(); err != nil {
			return err
		}
		return d.decodePacket(ref.c, ref.r, ref.p, layer)
	}

	switch d.th.progression {
	case progLRCP, progRLCP:
		sort.SliceStable(refs, func(i, j int) bool {
			a, b := refs[i], refs[j]
			if a.r != b.r {
				return a.r < b.r
			}
			if a.c != b.c {
				return a.c < b.c
			}
			return a.p < b.p
		})
		if d.th.progression == progLRCP {
			for l := 0; l < layers; l++ {
				for _, ref := range refs {
					if err := emit(ref, l); err != nil {
						return err
					}
				}
			}
		} else {
			maxR := 0
			for _, ref := range refs {
				maxR = max(maxR, ref.r)
			}
			for r := 0; r <= maxR; r++ {
				for l := 0; l < layers; l++ {
					for _, ref := range refs {
						if ref.r == r {
							if err := emit(ref, l); err != nil {
								return err
							}
						}
					}
				}
			}
		}
	case progRPCL, progPCRL, progCPRL:
		key := func(a pktRef) [4]int {
			switch d.th.progression {
			case progRPCL:
				return [4]int{a.r, a.y, a.x, a.c}
			case progPCRL:
				return [4]int{a.y, a.x, a.c, a.r}
			default: // CPRL
				return [4]int{a.c, a.y, a.x, a.r}
			}
		}
		sort.SliceStable(refs, func(i, j int) bool {
			ka, kb := key(refs[i]), key(refs[j])
			for n := 0; n < 4; n++ {
				if ka[n] != kb[n] {
					return ka[n] < kb[n]
				}
			}
			return false
		})
		for _, ref := range refs {
			for l := 0; l < layers; l++ {
				if err := emit(ref, l); err != nil {
					return err
				}
			}
		}
	default:
		return errFormat("unknown progression order")
	}
	return nil
}

// cbContribution records one code-block's contribution announced by a
// packet header, so the body bytes can be attached afterwards.
type cbContribution struct {
	cb   *codeblock
	segs []segLen
}

type segLen struct {
	length, passes int
}

func (d *tileDecoder) decodePacket(c, r, p, layer int) error {
	data := d.td.data
	res := d.comps[c].res[r]
	cp := d.comps[c].cp
	termAll := cp.cbStyle&cbTermAll != 0

	// Optional SOP marker segment before the packet.
	if d.th.sop && d.pos+6 <= len(data) &&
		be16(data[d.pos:]) == mSOP {
		d.pos += 6
	}

	if d.pos >= len(data) {
		return errShortPacket
	}
	br := &bitReader{data: data[d.pos:]}
	nonEmpty, err := br.bit()
	if err != nil {
		return err
	}

	var contribs []*cbContribution
	if nonEmpty == 1 {
		for bi := range res.bands {
			pb := res.prec[p][bi]
			for cbIdx, cb := range pb.cbs {
				contrib, err := d.readCBHeader(br, pb, cb, cbIdx, layer, termAll, res.bands[bi].mb)
				if err != nil {
					return err
				}
				if contrib != nil {
					contribs = append(contribs, contrib)
				}
			}
		}
	}
	br.alignFlush()
	d.pos += br.pos

	// Optional EPH marker after the packet header.
	if d.th.eph {
		if d.pos+2 > len(data) || be16(data[d.pos:]) != mEPH {
			return errFormat("missing EPH marker")
		}
		d.pos += 2
	}

	// Packet body: code-block segments in header order. The segments
	// alias the tile data rather than copying it.
	for _, ct := range contribs {
		for _, sl := range ct.segs {
			if sl.length > len(data)-d.pos {
				return errShortPacket
			}
			ct.cb.segs = append(ct.cb.segs, segment{
				data:   data[d.pos : d.pos+sl.length],
				passes: sl.passes,
			})
			d.pos += sl.length
			ct.cb.totalPasses += sl.passes
		}
	}
	return nil
}

func (d *tileDecoder) readCBHeader(br *bitReader, pb *precBand, cb *codeblock, cbIdx, layer int, termAll bool, mb int) (*cbContribution, error) {
	// Inclusion.
	if !cb.seenIncl {
		incl, err := pb.incl.decode(br, cbIdx, int32(layer)+1)
		if err != nil {
			return nil, err
		}
		if !incl {
			return nil, nil
		}
	} else {
		bit, err := br.bit()
		if err != nil {
			return nil, err
		}
		if bit == 0 {
			return nil, nil
		}
	}

	// Zero bit-planes on first inclusion.
	if !cb.seenIncl {
		// A code-block has at most mb bit-planes, all of them possibly
		// zero: a larger count is malformed, and reading on would let the
		// packet header's bits alone set the work (each threshold rewalks
		// the tag tree).
		for k := int32(1); ; k++ {
			if k > int32(mb)+1 {
				return nil, errFormat("more zero bit-planes than the band has")
			}
			ok, err := pb.zbps.decode(br, cbIdx, k)
			if err != nil {
				return nil, err
			}
			if ok {
				break
			}
		}
		cb.zbp = int(pb.zbps.value(cbIdx))
		cb.seenIncl = true
	}

	// Number of coding passes (Table B.4).
	numPasses, err := decodeNumPasses(br)
	if err != nil {
		return nil, err
	}
	if cb.totalPasses+numPasses > 109 {
		return nil, errFormat("too many coding passes")
	}

	// Lblock update.
	for {
		bit, err := br.bit()
		if err != nil {
			return nil, err
		}
		if bit == 0 {
			break
		}
		cb.lblock++
		if cb.lblock > 32 {
			return nil, errFormat("invalid Lblock")
		}
	}

	// Codeword segment lengths. Without pass termination all passes of
	// this packet share a single codeword segment.
	ct := &cbContribution{cb: cb}
	readSeg := func(passes int) error {
		n := uint(cb.lblock + bits.Len(uint(passes)) - 1)
		if n > 31 {
			return errFormat("segment length too long")
		}
		v, err := br.bits(n)
		if err != nil {
			return err
		}
		ct.segs = append(ct.segs, segLen{length: int(v), passes: passes})
		return nil
	}
	if termAll {
		for i := 0; i < numPasses; i++ {
			if err := readSeg(1); err != nil {
				return nil, err
			}
		}
	} else {
		if err := readSeg(numPasses); err != nil {
			return nil, err
		}
	}
	return ct, nil
}

func decodeNumPasses(br *bitReader) (int, error) {
	bit, err := br.bit()
	if err != nil {
		return 0, err
	}
	if bit == 0 {
		return 1, nil
	}
	if bit, err = br.bit(); err != nil {
		return 0, err
	}
	if bit == 0 {
		return 2, nil
	}
	v, err := br.bits(2)
	if err != nil {
		return 0, err
	}
	if v < 3 {
		return int(v) + 3, nil
	}
	if v, err = br.bits(5); err != nil {
		return 0, err
	}
	if v < 31 {
		return int(v) + 6, nil
	}
	if v, err = br.bits(7); err != nil {
		return 0, err
	}
	return int(v) + 37, nil
}
