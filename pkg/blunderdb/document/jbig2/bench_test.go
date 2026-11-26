package jbig2

import (
	"context"
	"testing"
)

// BenchmarkDecode runs the oracle fixtures, whose output TestOracle pins
// byte for byte.
func BenchmarkDecode(b *testing.B) {
	for _, c := range []struct{ name, globals, data string }{
		{"generic", "", "generic.jb2"},
		{"tpgd", "", "tpgd.jb2"},
		{"text", "sym.sym", "sym0.jb2"},
		{"refine", "annexh-p3.sym", "annexh-p3.jb2"},
		{"mmr", "", "annexh-mmr.jb2"},
	} {
		var g []byte
		if c.globals != "" {
			g = readFile(b, c.globals)
		}
		data := readFile(b, c.data)
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, _, err := Decode(context.Background(), data, g, 0, 0, nil); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
