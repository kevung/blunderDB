package structure

import (
	"image"
	"image/draw"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// wordFonts are the typefaces words.bin and figures.bin are drawn from, in
// their order, as a Linux distribution installs them under /usr/share/fonts:
// Liberation 2 (SIL Open Font License 1.1), C059 of URW base35 (AGPL 3 with
// the font exception; Century Schoolbook's metrics) and DejaVu (Bitstream
// Vera licence). Only the drawings derived from them are committed.
var wordFonts = []string{
	"liberation/LiberationSans-Regular.ttf",
	"liberation/LiberationSans-Bold.ttf",
	"liberation/LiberationSans-Italic.ttf",
	"liberation/LiberationSerif-Regular.ttf",
	"liberation/LiberationSerif-Bold.ttf",
	"liberation/LiberationSerif-Italic.ttf",
	"gsfonts/C059-Roman.otf",
	"gsfonts/C059-Bold.otf",
	"TTF/DejaVuSans.ttf",
	"TTF/DejaVuSans-Bold.ttf",
}

// TestGenerateWords writes words.bin, and figures.bin beside it, to the
// path in BLUNDERDB_GEN_WORDS, from wordFonts found under the folder in
// BLUNDERDB_GEN_WORDS_FONTS (/usr/share/fonts when empty). It runs only
// where BLUNDERDB_GEN_WORDS is set: the files are committed, the fonts are
// not.
func TestGenerateWords(t *testing.T) {
	const W, H = 640, 200
	out, dir := os.Getenv("BLUNDERDB_GEN_WORDS"), os.Getenv("BLUNDERDB_GEN_WORDS_FONTS")
	if out == "" {
		t.Skip("BLUNDERDB_GEN_WORDS is not set")
	}
	if dir == "" {
		dir = "/usr/share/fonts"
	}
	var buf, figs []byte
	for _, name := range wordFonts {
		path := filepath.Join(dir, name)
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		f, err := opentype.Parse(raw)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		face, err := opentype.NewFace(f, &opentype.FaceOptions{Size: 100, DPI: 72, Hinting: font.HintingNone})
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range "0123456789" {
			img := image.NewGray(image.Rect(0, 0, W, H))
			draw.Draw(img, img.Bounds(), image.White, image.Point{}, draw.Src)
			d := font.Drawer{Dst: img, Src: image.Black, Face: face, Dot: fixed.P(40, 140)}
			d.DrawString(string(r))
			x0, y0, x1, y1 := inkBox(img)
			v := normSquare(x1-x0, y1-y0, func(x, y int) bool { return img.GrayAt(x0+x, y0+y).Y < 128 })
			for i := 0; i < len(v); i += 2 {
				figs = append(figs, byte(min(15, int(v[i]/17+0.5)))<<4|byte(min(15, int(v[i+1]/17+0.5))))
			}
		}
		for _, word := range wordList {
			img := image.NewGray(image.Rect(0, 0, W, H))
			draw.Draw(img, img.Bounds(), image.White, image.Point{}, draw.Src)
			d := font.Drawer{Dst: img, Src: image.Black, Face: face, Dot: fixed.P(40, 140)}
			d.DrawString(word)
			x0, y0, x1, y1 := inkBox(img)
			if x1 <= x0 {
				t.Fatalf("%s: %q drew nothing", path, word)
			}
			v := normWord(x1-x0, y1-y0, func(x, y int) bool { return img.GrayAt(x0+x, y0+y).Y < 128 })
			buf = append(buf, byte(min(255, int(32*float64(x1-x0)/float64(y1-y0)+0.5))))
			for i := 0; i < len(v); i += 2 {
				buf = append(buf, byte(min(15, int(v[i]/17+0.5)))<<4|byte(min(15, int(v[i+1]/17+0.5))))
			}
		}
	}
	if err := os.WriteFile(out, buf, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(out), "figures.bin"), figs, 0o644); err != nil {
		t.Fatal(err)
	}
}

// inkBox is the box of an image's ink.
func inkBox(img *image.Gray) (x0, y0, x1, y1 int) {
	b := img.Bounds()
	x0, y0 = b.Dx(), b.Dy()
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			if img.GrayAt(x, y).Y < 128 {
				x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x+1), max(y1, y+1)
			}
		}
	}
	return
}
