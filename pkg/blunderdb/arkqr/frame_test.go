package arkqr

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"reflect"
	"strings"
	"testing"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
)

func corpusPayloads(t *testing.T) [][]byte {
	t.Helper()
	var out [][]byte
	for _, cm := range loadCorpus(t) {
		data, err := Encode(Match{Doc: cm.doc})
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, data)
	}
	return out
}

func TestFramesAssembleAndRepair(t *testing.T) {
	payloads := corpusPayloads(t)
	big := bytes.Repeat([]byte{1, 2, 3, 4, 5, 6, 7}, 400) // too long for one code
	payloads = append(payloads, big)
	frames, owner, err := Frames(payloads)
	if err != nil {
		t.Fatal(err)
	}
	nData := 0
	for i, f := range frames {
		g, err := ParseText(f.Text())
		if err != nil || !reflect.DeepEqual(g, f) {
			t.Fatalf("frame %d does not survive its text: %v", i, err)
		}
		if !f.Parity {
			nData++
		} else if owner[i] != -1 {
			t.Fatalf("parity frame %d owned by a match", i)
		}
	}
	if want := len(payloads) + 2; nData != want {
		t.Fatalf("%d data frames, want %d (one per match, the long one in three)", nData, want)
	}
	if got := Assemble(frames); !reflect.DeepEqual(got.Matches, payloads) || len(got.Missing) != 0 {
		t.Fatalf("assembled %d matches, missing %v", len(got.Matches), got.Missing)
	}

	// One code lost in every group of five: every group is rebuilt.
	var damaged []Frame
	for _, f := range frames {
		if !f.Parity && f.Index%ParityGroup == (f.Index/ParityGroup)%min(ParityGroup, f.Total-f.Index/ParityGroup*ParityGroup) {
			continue
		}
		damaged = append(damaged, f)
	}
	got := Assemble(damaged)
	if !reflect.DeepEqual(got.Matches, payloads) || got.Repaired == 0 {
		t.Fatalf("repair: %d matches, %d repaired, missing %v", len(got.Matches), got.Repaired, got.Missing)
	}

	// Two codes lost in one group: reported, the rest restored.
	var two []Frame
	for _, f := range frames {
		if !f.Parity && (f.Index == 0 || f.Index == 1) {
			continue
		}
		two = append(two, f)
	}
	got = Assemble(two)
	if len(got.Matches) != len(payloads)-2 || len(got.Missing[frames[0].Archive]) != 2 {
		t.Fatalf("two lost: %d matches, missing %v", len(got.Matches), got.Missing)
	}
}

// sheetImage draws frames' codes side by side, as a scanner would see them.
func sheetImage(t *testing.T, frames []Frame, scale int) image.Image {
	t.Helper()
	var all [][][]bool
	w, h, rowH, x := 0, 0, 0, 0
	const width = 1600
	type pos struct{ x, y int }
	var at []pos
	for _, f := range frames {
		m, err := Modules(f)
		if err != nil {
			t.Fatal(err)
		}
		side := len(m) * scale
		if x > 0 && x+side > width {
			x, h, rowH = 0, h+rowH+2*scale, 0
		}
		at = append(at, pos{x, h})
		all = append(all, m)
		x += side + 2*scale
		w = max(w, x)
		rowH = max(rowH, side)
	}
	img := image.NewGray(image.Rect(0, 0, w, h+rowH))
	for i := range img.Pix {
		img.Pix[i] = 255
	}
	for i, m := range all {
		for y, row := range m {
			for xx, dark := range row {
				if !dark {
					continue
				}
				for dy := range scale {
					for dx := range scale {
						img.SetGray(at[i].x+xx*scale+dx, at[i].y+y*scale+dy, color.Gray{})
					}
				}
			}
		}
	}
	return img
}

func TestScanImagePNGAndJPEG(t *testing.T) {
	payloads := corpusPayloads(t)
	frames, _, _ := Frames(payloads)
	img := sheetImage(t, frames, 4)
	var pngBuf, jpgBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		t.Fatal(err)
	}
	if err := jpeg.Encode(&jpgBuf, img, &jpeg.Options{Quality: 80}); err != nil {
		t.Fatal(err)
	}
	for name, buf := range map[string]*bytes.Buffer{"png": &pngBuf, "jpeg": &jpgBuf} {
		read, err := ScanImage(buf)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		got := Assemble(read)
		if !reflect.DeepEqual(got.Matches, payloads) {
			t.Fatalf("%s: read %d codes of %d, %d matches, missing %v", name, len(read), len(frames), len(got.Matches), got.Missing)
		}
	}
}

func TestPageEmbedsFramesAndKeepsFoldsClear(t *testing.T) {
	payloads := corpusPayloads(t)
	payloads = append(payloads, payloads...)
	payloads = append(payloads, payloads...)
	frames, _, _ := Frames(payloads)
	codes := make([]Code, len(frames))
	for i, f := range frames {
		codes[i] = Code{Frame: f, Label: "Joueur 1 – Joueur 2 · 2026-11-26 · 7 pts"}
	}
	page, err := Page("Archive <test>", "Club de Lyon", codes)
	if err != nil {
		t.Fatal(err)
	}
	if !IsPage(page) || !strings.Contains(string(page), "Archive &lt;test&gt;") {
		t.Fatal("page markers missing")
	}
	read, err := ReadPage(page)
	if err != nil || !reflect.DeepEqual(read, frames) {
		t.Fatalf("embedded frames: %v", err)
	}
	sheets, err := layout(codes)
	if err != nil {
		t.Fatal(err)
	}
	if len(sheets) < 2 {
		t.Fatalf("%d sheets for %d codes", len(sheets), len(codes))
	}
	for si, s := range sheets {
		for qi, q := range s {
			for _, p := range q {
				if p.x < 0 || p.y < 0 || p.x+p.side > quadW+1e-9 || p.y+p.side+labelH > quadH+1e-9 {
					t.Fatalf("sheet %d quadrant %d: code at (%.1f, %.1f) side %.1f leaves its quadrant", si, qi, p.x, p.y, p.side)
				}
			}
		}
	}
	if _, err := ReadPage([]byte("<html></html>")); err == nil {
		t.Fatal("a page without archive read")
	}
}

// TestCodeLevelM reads a code back with its metadata: error correction M.
func TestCodeLevelM(t *testing.T) {
	frames, _, _ := Frames(corpusPayloads(t)[:1])
	bmp, err := gozxing.NewBinaryBitmapFromImage(sheetImage(t, frames[:1], 4))
	if err != nil {
		t.Fatal(err)
	}
	res, err := qrcode.NewQRCodeReader().Decode(bmp, nil)
	if err != nil {
		t.Fatal(err)
	}
	if lvl := res.GetResultMetadata()[gozxing.ResultMetadataType_ERROR_CORRECTION_LEVEL]; lvl != "M" {
		t.Fatalf("error correction level %v, want M", lvl)
	}
}

// TestFramesRefuseTooManyCodes: a rank is stated on 16 bits, so an archive
// of more codes is refused rather than written and misread.
func TestFramesRefuseTooManyCodes(t *testing.T) {
	small := make([][]byte, maxFrames+1)
	for i := range small {
		small[i] = []byte{1, byte(i)}
	}
	if _, _, err := Frames(small); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("%d codes: %v", len(small), err)
	}
	if fs, _, err := Frames(small[:maxFrames]); err != nil || fs[len(fs)-1].Total != maxFrames {
		t.Fatalf("%d codes: %v", maxFrames, err)
	}
	if _, _, err := Frames([][]byte{make([]byte, 256*maxPart)}); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("a match of 256 parts: %v", err)
	}
}
