package jpx

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"os"
	"testing"
	"time"
)

type comp struct{ dx, dy int }

type cs struct{ bytes.Buffer }

func (b *cs) u16(v int) { _ = binary.Write(b, binary.BigEndian, uint16(v)) }
func (b *cs) u32(v int) { _ = binary.Write(b, binary.BigEndian, uint32(v)) }

type spec struct {
	w, h, tw, th int
	rsiz         int
	comps        []comp
	cbw, cbh     int
	style        int
	precincts    []byte // nil: maximal precincts
	cap          bool
	tileParts    int
}

func gray(w, h int) spec {
	return spec{w: w, h: h, tw: w, th: h, comps: []comp{{1, 1}}, cbw: 4, cbh: 4}
}

// build writes a codestream header, with tile-parts that hold no packets:
// enough for the guards, which read headers only.
func build(s spec) []byte {
	var b cs
	b.u16(0xFF4F)
	b.u16(0xFF51)
	b.u16(38 + 3*len(s.comps))
	b.u16(s.rsiz)
	for _, v := range []int{s.w, s.h, 0, 0, s.tw, s.th, 0, 0} {
		b.u32(v)
	}
	b.u16(len(s.comps))
	for _, c := range s.comps {
		b.WriteByte(7)
		b.WriteByte(byte(c.dx))
		b.WriteByte(byte(c.dy))
	}
	if s.cap {
		b.u16(0xFF50)
		b.u16(6)
		b.u32(0)
	}
	b.u16(0xFF52)
	b.u16(12 + len(s.precincts))
	if s.precincts != nil {
		b.WriteByte(1)
	} else {
		b.WriteByte(0)
	}
	b.WriteByte(0)
	b.u16(1)
	b.WriteByte(0)
	b.WriteByte(byte(len(s.precincts) - 1)) // levels; 0 when maximal
	if s.precincts == nil {
		b.Truncate(b.Len() - 1)
		b.WriteByte(0)
	}
	b.WriteByte(byte(s.cbw))
	b.WriteByte(byte(s.cbh))
	b.WriteByte(byte(s.style))
	b.WriteByte(1)
	b.Write(s.precincts)
	b.u16(0xFF5C)
	b.u16(5)
	b.WriteByte(2 << 5)
	b.WriteByte(8 << 3)
	b.WriteByte(8 << 3)
	for i := 0; i < max(s.tileParts, 1); i++ {
		b.u16(0xFF90)
		b.u16(10)
		b.u16(i % 65536)
		b.u32(14)
		b.WriteByte(0)
		b.WriteByte(0)
		b.u16(0xFF93)
	}
	b.u16(0xFFD9)
	return b.Bytes()
}

func TestInspectRefuses(t *testing.T) {
	big := make([]comp, 16384)
	for i := range big {
		big[i] = comp{1, 1}
	}
	with := func(f func(*spec)) spec { s := gray(256, 256); f(&s); return s }
	for name, s := range map[string]spec{
		"precincts of 1×1":   with(func(s *spec) { s.w, s.h, s.tw, s.th = 2000, 2000, 2000, 2000; s.precincts = []byte{0x00} }),
		"a precinct of 16":   with(func(s *spec) { s.precincts = []byte{0x44} }),
		"16384 components":   with(func(s *spec) { s.comps, s.tw, s.th, s.tileParts = big, 1, 1, 5000 }),
		"two components":     with(func(s *spec) { s.comps = []comp{{1, 1}, {1, 1}} }),
		"four components":    with(func(s *spec) { s.comps = []comp{{1, 1}, {1, 1}, {1, 1}, {1, 1}} }),
		"dx 255":             with(func(s *spec) { s.comps = []comp{{255, 255}} }),
		"a chroma subsample": with(func(s *spec) { s.comps = []comp{{1, 1}, {2, 2}, {2, 2}} }),
		"tiles of 1×1":       with(func(s *spec) { s.tw, s.th = 1, 1 }),
		"too many pixels":    with(func(s *spec) { s.w, s.h, s.tw, s.th = 5000, 5000, 5000, 5000 }),
		"a code-block of 8":  with(func(s *spec) { s.cbw = 1 }),
		"HT in Rsiz":         with(func(s *spec) { s.rsiz = 0x4000 }),
		"a CAP marker":       with(func(s *spec) { s.cap = true }),
		"the HT block style": with(func(s *spec) { s.style = 0x40 }),
	} {
		start := time.Now()
		_, _, err := Decode(context.Background(), build(s), 0, 0, &Budget{Left: 1 << 40})
		if !errors.Is(err, ErrUnsupported) {
			t.Errorf("%s: %v", name, err)
		}
		if time.Since(start) > time.Second {
			t.Errorf("%s: refused after %v", name, time.Since(start))
		}
	}
}

func TestInspectAccepts(t *testing.T) {
	for _, f := range []string{"lossless53.jp2", "lossy97.jp2"} {
		data, err := os.ReadFile("testdata/" + f)
		if err != nil {
			t.Fatal(err)
		}
		info, err := Inspect(data)
		if err != nil || info.Width != 320 || info.Height != 40 || info.Components != 1 {
			t.Errorf("%s: %+v %v", f, info, err)
		}
		if info.Reversible != (f == "lossless53.jp2") {
			t.Errorf("%s: reversible %v", f, info.Reversible)
		}
	}
}

func TestDecodeContainer(t *testing.T) {
	data, _ := os.ReadFile("testdata/lossy97.jp2")
	if _, _, err := Decode(context.Background(), data, 321, 40, nil); !errors.Is(err, ErrUnsupported) {
		t.Errorf("another reference grid than the container's: %v", err)
	}
	if _, _, err := Decode(context.Background(), data, 320, 40, &Budget{Left: 1000}); !errors.Is(err, ErrBudget) {
		t.Errorf("budget: %v", err)
	}
	b := &Budget{Left: 320 * 40 * 2}
	for i := 0; i < 2; i++ {
		if _, _, err := Decode(context.Background(), data, 320, 40, b); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, err := Decode(context.Background(), data, 320, 40, b); !errors.Is(err, ErrBudget) {
		t.Errorf("the third decode: %v", err)
	}
}

func TestJP2Boxes(t *testing.T) {
	data, _ := os.ReadFile("testdata/lossy97.jp2")
	box := func(typ string, body []byte) []byte {
		var b cs
		b.u32(8 + len(body))
		b.WriteString(typ)
		b.Write(body)
		return b.Bytes()
	}
	// jp2h is the second box, after the signature and ftyp.
	sigLen := 12
	ftypLen := int(binary.BigEndian.Uint32(data[sigLen:]))
	for _, typ := range []string{"pclr", "cmap", "cdef"} {
		jp2h := box("jp2h", box(typ, make([]byte, 8)))
		d := append(append(append([]byte{}, data[:sigLen+ftypLen]...), jp2h...), data[sigLen+ftypLen:]...)
		if _, err := Inspect(d); !errors.Is(err, ErrUnsupported) {
			t.Errorf("%s: %v", typ, err)
		}
	}
}

// TestHostile: truncated and corrupted streams are refused or read, never a
// panic, and a PDF that holds none of them is not slowed.
func TestHostile(t *testing.T) {
	data, _ := os.ReadFile("testdata/lossy97.jp2")
	for n := 0; n < len(data); n += 3 {
		_, _, _ = Decode(context.Background(), data[:n], 0, 0, nil)
	}
	for i := 0; i < len(data); i += 3 {
		b := append([]byte(nil), data...)
		b[i] ^= 0xff
		_, _, _ = Decode(context.Background(), b, 0, 0, &Budget{Left: 1 << 30})
	}
}

func FuzzDecode(f *testing.F) {
	data, _ := os.ReadFile("testdata/lossy97.jp2")
	f.Add(data)
	f.Add(build(gray(64, 64)))
	f.Fuzz(func(t *testing.T, b []byte) {
		_, _, _ = Decode(context.Background(), b, 0, 0, &Budget{Left: 1 << 22})
	})
}
