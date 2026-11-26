package jpx

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"image"
	"os"
	"runtime"
	"testing"
)

// fixtures: every stream of testdata, with the SHA-256 of what this decoder
// draws from it. The digests pin the output bit for bit, so that a faster
// inner loop cannot change a sample. big97, rgb97tiles and rev53 are
// opj_compress on plasma noise (1024² grey at -r 20; 512² RGB at -r 20 in
// 256² tiles; 512² grey, 5/3 reversible, at -r 4). The 9/7 path computes in
// floating point, which a compiler may fuse into multiply-adds off amd64:
// its digests hold there only.
var fixtures = []struct {
	name, digest string
	float        bool
}{
	{"lossless53.jp2", "8da84c97982a2d079b2ee540264ddc78d8a795bc8f52a9b37a546d923237c103", false},
	{"lossy97.jp2", "6083cda102575f4d96ecf4da12188f9420190e2dfa9ae84b1b52f912809850b3", true},
	{"big97.jp2", "a04971648b066cad6ab51615761a3d0a09630df7050124f8a7ae5bc6816daa52", true},
	{"rgb97tiles.jp2", "de056be65c635573f0131601e84a07c123cde60baf2235003aabde275520bd04", true},
	{"rev53.jp2", "8cdbbea2ea12532129fea7c60950c9fdcc7b3ed144b6f6c718fc1da5c7b4654c", false},
}

func digest(img image.Image) string {
	h := sha256.New()
	b := img.Bounds()
	fmt.Fprintf(h, "%T %v ", img, b)
	switch m := img.(type) {
	case *image.Gray:
		h.Write(m.Pix)
	case *image.Gray16:
		h.Write(m.Pix)
	case *image.NRGBA:
		h.Write(m.Pix)
	case *image.NRGBA64:
		h.Write(m.Pix)
	default:
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				r, g, bl, a := img.At(x, y).RGBA()
				_ = binary.Write(h, binary.BigEndian, [4]uint32{r, g, bl, a})
			}
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}

func TestGolden(t *testing.T) {
	for _, f := range fixtures {
		data, err := os.ReadFile("testdata/" + f.name)
		if err != nil {
			t.Fatal(err)
		}
		img, _, err := Decode(context.Background(), data, 0, 0, nil)
		if err != nil {
			t.Errorf("%s: %v", f.name, err)
			continue
		}
		if f.float && runtime.GOARCH != "amd64" {
			continue
		}
		if got := digest(img); got != f.digest {
			t.Errorf("%s: digest %s, want %s", f.name, got, f.digest)
		}
	}
}

func BenchmarkDecode(b *testing.B) {
	for _, f := range fixtures {
		data, err := os.ReadFile("testdata/" + f.name)
		if err != nil {
			b.Fatal(err)
		}
		b.Run(f.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(data)))
			for b.Loop() {
				if _, _, err := Decode(context.Background(), data, 0, 0, nil); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
