package document

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/capture"
	"github.com/kevung/blunderdb/pkg/blunderdb/document/structure"
)

func TestStructureEntry(t *testing.T) {
	sure := structure.Diagram{Page: 2, Rect: image.Rect(100, 200, 600, 650), PageSize: [2]int{1000, 1300},
		Board: "-b----E-C---eE---c-e----B-", State: structure.Sure}
	e := structureEntry(sure, 1)
	if e.State != Sure || e.Page != 3 || e.Position != "XGID=-b----E-C---eE---c-e----B-:0:0:1:00:0:0:0:0:10" || e.Rect == nil || e.Rect.W != 0.5 {
		t.Errorf("sure entry %+v", e)
	}
	doubt := []capture.Doubt{{Field: "pips", Reason: "no printed pip count was found next to the diagram"}}
	tv := structureEntry(structure.Diagram{Board: sure.Board, State: structure.ToVerify, Doubts: doubt}, 2)
	if tv.State != ToVerify || len(tv.Doubts) != 1 {
		t.Errorf("to-verify entry %+v", tv)
	}
	// No checker of one side: not a position, the entry keeps its place.
	empty := structureEntry(structure.Diagram{Board: "-B----E-C---E-------------", State: structure.ToVerify}, 3)
	if empty.State != Uncertain || empty.Position != "" || len(empty.Doubts) != 1 {
		t.Errorf("empty side %+v", empty)
	}
	r := &Releve{Format: Format, Version: Version, Document: Source{Title: "t"}, Entries: []Entry{e, tv, empty}}
	r.Entries[1].Page, r.Entries[2].Page = 3, 3
	if err := r.Validate(); err != nil {
		t.Error(err)
	}
}

func pngWithDPI(t *testing.T, img image.Image, ppm uint32) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	data := b.Bytes()
	chunk := make([]byte, 0, 21)
	chunk = binary.BigEndian.AppendUint32(chunk, 9)
	body := []byte("pHYs")
	body = binary.BigEndian.AppendUint32(body, ppm)
	body = binary.BigEndian.AppendUint32(body, ppm)
	body = append(body, 1)
	chunk = append(chunk, body...)
	chunk = binary.BigEndian.AppendUint32(chunk, crc32.ChecksumIEEE(body))
	ihdrEnd := 8 + 8 + 13 + 4
	return append(append(append([]byte(nil), data[:ihdrEnd]...), chunk...), data[ihdrEnd:]...)
}

func TestImageDPI(t *testing.T) {
	data := pngWithDPI(t, image.NewGray(image.Rect(0, 0, 4, 4)), 5906)
	if d := imageDPI(data); d < 149.9 || d > 150.1 {
		t.Errorf("pHYs dpi %v", d)
	}
	jfif := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0, 16, 'J', 'F', 'I', 'F', 0, 1, 1, 1, 0, 200, 0, 200, 0, 0, 0}
	if d := imageDPI(jfif); d != 200 {
		t.Errorf("JFIF dpi %v", d)
	}
	if d := imageDPI([]byte("not an image at all")); d != 0 {
		t.Errorf("dpi %v", d)
	}
	if d := estimatedDPI(1700, 2200); d != 200 {
		t.Errorf("estimated %v", d)
	}
}

func TestAnalyzeFileImages(t *testing.T) {
	dir := t.TempDir()
	if _, err := AnalyzeFile(dir, AnalyzeOptions{}); !errors.Is(err, ErrNoImage) {
		t.Errorf("empty folder: %v", err)
	}
	blank := image.NewGray(image.Rect(0, 0, 400, 500))
	for i := range blank.Pix {
		blank.Pix[i] = 255
	}
	if err := os.WriteFile(filepath.Join(dir, "p001.png"), pngWithDPI(t, blank, 5906), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := AnalyzeFile(dir, AnalyzeOptions{}); !errors.Is(err, structure.ErrNoDiagram) {
		t.Errorf("blank page: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "p002.png"), []byte("\x89PNG\r\n\x1a\nbroken"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := AnalyzeFile(dir, AnalyzeOptions{}); err == nil {
		t.Errorf("a broken image must be refused")
	}
}
