package jbig2

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readFile(t testing.TB, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// readPBM reads a raw (P4) bitmap, 1 for black, as jbig2dec writes it.
func readPBM(t *testing.T, name string) (w, h int, data []byte) {
	b := readFile(t, name)
	if _, err := fmt.Sscanf(string(b), "P4\n%d %d\n", &w, &h); err != nil {
		t.Fatal(err)
	}
	n := 0
	for k := 0; k < 2; k++ {
		n += bytes.IndexByte(b[n:], '\n') + 1
	}
	return w, h, b[n:]
}

// TestOracle: streams of a third-party encoder (jbig2enc) and the example of
// T.88 Annex H decode to what a third-party decoder (jbig2dec) draws. The
// fixtures are made outside the CI; testdata/README says how.
func TestOracle(t *testing.T) {
	for _, c := range []struct{ name, globals, data, want string }{
		{"generic region, template 0", "", "generic.jb2", "generic.pbm"},
		{"generic region, typical prediction", "", "tpgd.jb2", "tpgd.pbm"},
		{"text region, global and page dictionaries", "sym.sym", "sym0.jb2", "sym0.pbm"},
		{"text region, second page of the same globals", "sym.sym", "sym1.jb2", "sym1.pbm"},
		{"refinement and aggregation, template 1 and 2", "annexh-p3.sym", "annexh-p3.jb2", "annexh-p3.pbm"},
		{"generic region, MMR", "", "annexh-mmr.jb2", "annexh-mmr.pbm"},
		{"generic region, prediction with adaptive pixels", "", "annexh-tpgd.jb2", "annexh-tpgd.pbm"},
	} {
		var g []byte
		if c.globals != "" {
			g = readFile(t, c.globals)
		}
		w, h, want := readPBM(t, c.want)
		p, _, err := Decode(context.Background(), readFile(t, c.data), g, w, h, nil)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if p.Width != w || p.Height != h || !bytes.Equal(p.Data, want) {
			diff := 0
			for i := range want {
				if i < len(p.Data) && p.Data[i] != want[i] {
					diff++
				}
			}
			t.Errorf("%s: %d×%d, %d bytes of %d differ", c.name, p.Width, p.Height, diff, len(want))
		}
	}
}

// TestNominalPaths: with the adaptive pixels at their nominal place, the
// running contexts give what the context of 6.2.5.3, read pixel by pixel,
// gives, for every template.
func TestNominalPaths(t *testing.T) {
	data := readFile(t, "generic.jb2")
	for tm := 0; tm < 4; tm++ {
		p := genericParams{w: 61, h: 23, template: tm, at: nominalAT[tm]}
		fast, _ := decodeGeneric(newMQ(data), make(contexts, genericCX[tm]), p, nil)
		// One adaptive pixel moved and back: the same place, the slow path.
		var slow *bitmap
		{
			m := newBitmap(p.w, p.h)
			d, cx := newMQ(data), make(contexts, genericCX[tm])
			for y := 0; y < p.h; y++ {
				for x := 0; x < p.w; x++ {
					if d.bit(cx, contextSlow(m, x, y, &p)) == 1 {
						m.set(x, y, 1)
					}
				}
			}
			slow = m
		}
		if !bytes.Equal(fast.b, slow.b) {
			t.Errorf("template %d: the running context differs", tm)
		}
	}
}

// seg builds a segment header (short form) and its data.
func seg(num uint32, typ int, refs []byte, data []byte) []byte {
	b := binary.BigEndian.AppendUint32(nil, num)
	b = append(b, byte(typ), byte(len(refs)<<5))
	b = append(b, refs...)
	b = append(b, 1)
	b = binary.BigEndian.AppendUint32(b, uint32(len(data)))
	return append(b, data...)
}

func pageInfo(w, h uint32) []byte {
	b := binary.BigEndian.AppendUint32(nil, w)
	b = binary.BigEndian.AppendUint32(b, h)
	return append(b, make([]byte, 11)...)
}

func region(w, h uint32, extra ...byte) []byte {
	b := binary.BigEndian.AppendUint32(nil, w)
	b = binary.BigEndian.AppendUint32(b, h)
	b = append(b, make([]byte, 9)...)
	return append(b, extra...)
}

var nominal0 = []byte{3, 0xff, 0xfd, 0xff, 2, 0xfe, 0xfe, 0xfe}

// TestRefusals: what the decoder does not take is refused by the parser,
// before anything is decoded, and named.
func TestRefusals(t *testing.T) {
	page := seg(0, segPageInfo, nil, pageInfo(64, 32))
	dict := func(flags uint16, nexp, nnew uint32) []byte {
		b := binary.BigEndian.AppendUint16(nil, flags)
		b = append(b, nominal0...)
		b = binary.BigEndian.AppendUint32(b, nexp)
		return binary.BigEndian.AppendUint32(b, nnew)
	}
	text := func(flags uint16, n uint32) []byte {
		b := binary.BigEndian.AppendUint16(region(64, 32), flags)
		return binary.BigEndian.AppendUint32(b, n)
	}
	unknown := append(seg(1, segGeneric, nil, nil)[:7], 0xff, 0xff, 0xff, 0xff)
	for _, c := range []struct {
		name    string
		data    []byte
		w, h    int
		err     error
		message string
	}{
		{"a halftone region", append(page, seg(1, segHalftone, nil, region(8, 8))...), 0, 0, ErrUnsupported, "halftone"},
		{"a Huffman dictionary", append(page, seg(1, segSymbolDict, nil, dict(1, 0, 0))...), 0, 0, ErrUnsupported, "Huffman"},
		{"a Huffman text region", append(page, seg(1, segText, nil, text(1, 0))...), 0, 0, ErrUnsupported, "Huffman"},
		{"an extended template", append(page, seg(1, segGeneric, nil, region(8, 8, 0x10))...), 0, 0, ErrUnsupported, "extended"},
		{"a necessary extension", append(page, seg(1, segExtension, nil, []byte{0x80, 0, 0, 1})...), 0, 0, ErrUnsupported, "necessary"},
		{"a segment of unknown length", append(page, unknown...), 0, 0, ErrUnsupported, "unknown length"},
		{"another size than the container's", page, 65, 32, ErrUnsupported, "container says"},
		{"a page over MaxPixels", seg(0, segPageInfo, nil, pageInfo(1<<14, 1<<14)), 0, 0, ErrUnsupported, "at most"},
		{"a page side over its bound", seg(0, segPageInfo, nil, pageInfo(maxSide+1, 1)), 0, 0, ErrUnsupported, "at most"},
		{"a region over MaxPixels", append(page, seg(1, segGeneric, nil, region(1<<14, 1<<14, 0))...), 0, 0, ErrUnsupported, "region of"},
		{"too many new symbols", append(page, seg(1, segSymbolDict, nil, dict(0, 0, maxSymbols+1))...), 0, 0, ErrUnsupported, "symbol dictionary of"},
		{"more exported than held", append(page, seg(1, segSymbolDict, nil, dict(0, 3, 2))...), 0, 0, ErrMalformed, "exports"},
		{"too many instances", append(page, seg(1, segText, nil, text(0, maxInstances+1))...), 0, 0, ErrUnsupported, "text region of"},
		{"instances without symbols", append(page, seg(1, segText, nil, text(0, 1))...), 0, 0, ErrMalformed, "no dictionary"},
		{"a segment longer than the stream", append(page, seg(1, segGeneric, nil, region(8, 8, 0))[:20]...), 0, 0, ErrMalformed, "remain"},
		{"a reference forward", append(page, seg(1, segGeneric, []byte{5}, region(8, 8, 0))...), 0, 0, ErrMalformed, "refers"},
		{"a region before the page", seg(1, segGeneric, nil, region(8, 8, 0)), 0, 0, ErrMalformed, "before the page"},
		{"no page", nil, 0, 0, ErrMalformed, "no page"},
		{"a referred count of 5", append([]byte{0, 0, 0, 1, segGeneric, 5 << 5}, make([]byte, 10)...), 0, 0, ErrMalformed, "referred-to count"},
	} {
		_, err := Inspect(c.data, nil, c.w, c.h)
		if !errors.Is(err, c.err) || !strings.Contains(err.Error(), c.message) {
			t.Errorf("%s: %v", c.name, err)
		}
	}
}

// TestBudget: a document's budget is spent by its pages, regions and
// symbols, and refused once spent.
func TestBudget(t *testing.T) {
	data := readFile(t, "generic.jb2")
	b := &Budget{Left: 480 * 200 * 2}
	if _, _, err := Decode(context.Background(), data, nil, 0, 0, b); err != nil || b.Left != 0 {
		t.Fatalf("%v, %d left", err, b.Left)
	}
	if _, _, err := Decode(context.Background(), data, nil, 0, 0, b); !errors.Is(err, ErrBudget) {
		t.Errorf("a spent budget: %v", err)
	}
	g, d := readFile(t, "sym.sym"), readFile(t, "sym0.jb2")
	b = &Budget{Left: 480*200*2 + 10}
	if _, _, err := Decode(context.Background(), d, g, 0, 0, b); !errors.Is(err, ErrBudget) {
		t.Errorf("symbols over the budget: %v", err)
	}
}

// TestContext: a cancelled context stops the decode.
func TestContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := Decode(ctx, readFile(t, "generic.jb2"), nil, 0, 0, nil); !errors.Is(err, context.Canceled) {
		t.Errorf("%v", err)
	}
}

// TestMalformed: every truncation and many corruptions of the fixtures are
// refused or decoded, never a panic, within the budget.
func TestMalformed(t *testing.T) {
	for _, c := range [][2]string{{"", "generic.jb2"}, {"sym.sym", "sym0.jb2"}, {"annexh-p3.sym", "annexh-p3.jb2"}, {"", "annexh-mmr.jb2"}} {
		var g []byte
		if c[0] != "" {
			g = readFile(t, c[0])
		}
		data := readFile(t, c[1])
		for n := 0; n < len(data); n += max(1, len(data)/97) {
			try(t, data[:n], g)
			for _, x := range []byte{0x01, 0x80, 0xff} {
				m := append([]byte(nil), data...)
				m[n] ^= x
				try(t, m, g)
			}
		}
		for n := 0; n < len(g); n++ {
			m := append([]byte(nil), g...)
			m[n] ^= 0x55
			try(t, data, m)
		}
	}
}

func try(t *testing.T, data, globals []byte) {
	t.Helper()
	b := &Budget{Left: 1 << 22}
	_, _, err := Decode(context.Background(), data, globals, 0, 0, b)
	if err != nil && !errors.Is(err, ErrMalformed) && !errors.Is(err, ErrUnsupported) && !errors.Is(err, ErrBudget) {
		t.Fatalf("an error of no kind: %v", err)
	}
}

func FuzzDecode(f *testing.F) {
	for _, c := range [][2]string{{"", "generic.jb2"}, {"", "tpgd.jb2"}, {"sym.sym", "sym0.jb2"}, {"annexh-p3.sym", "annexh-p3.jb2"}, {"", "annexh-mmr.jb2"}} {
		var g []byte
		if c[0] != "" {
			g = readFile(f, c[0])
		}
		f.Add(readFile(f, c[1]), g)
	}
	f.Fuzz(func(t *testing.T, data, globals []byte) {
		b := &Budget{Left: 1 << 22}
		_, _, err := Decode(context.Background(), data, globals, 0, 0, b)
		if err != nil && !errors.Is(err, ErrMalformed) && !errors.Is(err, ErrUnsupported) && !errors.Is(err, ErrBudget) {
			t.Fatalf("an error of no kind: %v", err)
		}
	})
}

// TestEmptyHeightClasses: a dictionary whose height classes hold no symbol
// stops, its coded data spent or not.
func TestEmptyHeightClasses(t *testing.T) {
	b := binary.BigEndian.AppendUint16(nil, 0)
	b = append(b, nominal0...)
	b = binary.BigEndian.AppendUint32(b, 0)
	b = binary.BigEndian.AppendUint32(b, 3)
	for _, coded := range [][]byte{nil, {0, 0, 0, 0}, bytes.Repeat([]byte{0xff, 0x7f}, 64)} {
		data := append(seg(0, segPageInfo, nil, pageInfo(8, 8)), seg(1, segSymbolDict, nil, append(b, coded...))...)
		_, _, err := Decode(context.Background(), data, nil, 0, 0, &Budget{Left: 1 << 20})
		if err != nil && !errors.Is(err, ErrMalformed) {
			t.Errorf("%x: %v", coded, err)
		}
	}
}

// TestStreamPixels: the regions of a stream, together, are bounded.
func TestStreamPixels(t *testing.T) {
	data := seg(0, segPageInfo, nil, pageInfo(1<<13, 1<<13))
	for i := uint32(1); i <= 4; i++ {
		data = append(data, seg(i, segGenericIntermed, nil, region(1<<13, 1<<13, 1))...)
	}
	if _, err := Inspect(data, nil, 0, 0); !errors.Is(err, ErrUnsupported) || !strings.Contains(err.Error(), "page and regions") {
		t.Errorf("%v", err)
	}
}
