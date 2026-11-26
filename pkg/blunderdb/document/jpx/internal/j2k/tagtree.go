package j2k

// Tag trees (ISO/IEC 15444-1 B.10.2) encode a 2D array of non-negative
// integers as a quad-tree of running minimums. Packet headers use them
// for code-block inclusion and zero-bit-plane counts. Decoding is
// incremental: each call refines knowledge of one leaf up to a
// threshold, and state persists across packets and layers.

type tagTreeNode struct {
	parent int32
	low    int32
	known  bool
}

type tagTree struct {
	nodes  []tagTreeNode
	leaves int
}

func newTagTree(w, h int) *tagTree {
	t := &tagTree{leaves: w * h}
	// Build levels from the leaves up until a single root remains.
	type level struct{ w, h, off int }
	var levels []level
	off := 0
	lw, lh := w, h
	for {
		levels = append(levels, level{lw, lh, off})
		off += lw * lh
		if lw == 1 && lh == 1 {
			break
		}
		lw = (lw + 1) / 2
		lh = (lh + 1) / 2
	}
	t.nodes = make([]tagTreeNode, off)
	for i := range t.nodes {
		t.nodes[i].parent = -1
	}
	for li := 0; li < len(levels)-1; li++ {
		cur, next := levels[li], levels[li+1]
		for y := 0; y < cur.h; y++ {
			for x := 0; x < cur.w; x++ {
				p := next.off + (y/2)*next.w + x/2
				t.nodes[cur.off+y*cur.w+x].parent = int32(p)
			}
		}
	}
	return t
}

// decode refines the leaf's value bound using bits from br and reports
// whether the leaf's value is known to be strictly less than threshold.
func (t *tagTree) decode(br *bitReader, leaf int, threshold int32) (bool, error) {
	// Collect the path from root down to the leaf.
	var path [32]int32
	n := 0
	for i := int32(leaf); i >= 0; i = t.nodes[i].parent {
		path[n] = i
		n++
	}
	low := int32(0)
	for i := n - 1; i >= 0; i-- {
		node := &t.nodes[path[i]]
		if node.low < low {
			node.low = low
		}
		for !node.known && node.low < threshold {
			bit, err := br.bit()
			if err != nil {
				return false, err
			}
			if bit == 1 {
				node.known = true
			} else {
				node.low++
			}
		}
		low = node.low
		if !node.known && low >= threshold {
			return false, nil
		}
	}
	return true, nil
}

// value returns the decoded value of a leaf; valid only after decode
// has returned true for it.
func (t *tagTree) value(leaf int) int32 {
	return t.nodes[leaf].low
}

// bitReader reads MSB-first bits from packet headers, honoring the bit
// stuffing rule of B.10.1: a byte following 0xFF carries only 7 bits.
type bitReader struct {
	data []byte
	pos  int
	buf  uint32
	ct   uint
}

func (b *bitReader) bit() (uint32, error) {
	if b.ct == 0 {
		if b.pos >= len(b.data) {
			return 0, errShortPacket
		}
		stuffed := b.buf == 0xFF
		b.buf = uint32(b.data[b.pos])
		b.pos++
		if stuffed {
			if b.buf&0x80 != 0 {
				return 0, errFormat("invalid bit stuffing in packet header")
			}
			b.ct = 7
		} else {
			b.ct = 8
		}
	}
	b.ct--
	return (b.buf >> b.ct) & 1, nil
}

func (b *bitReader) bits(n uint) (uint32, error) {
	var v uint32
	for i := uint(0); i < n; i++ {
		bit, err := b.bit()
		if err != nil {
			return 0, err
		}
		v = v<<1 | bit
	}
	return v, nil
}

// alignFlush consumes padding up to a byte boundary at the end of a
// packet header. If the final header byte is 0xFF, a stuffing byte
// follows and is consumed too.
func (b *bitReader) alignFlush() {
	b.ct = 0
	if b.buf == 0xFF {
		b.pos++
		b.buf = 0
	}
}
