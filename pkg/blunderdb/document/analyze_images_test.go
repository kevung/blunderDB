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
	"slices"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/capture"
	"github.com/kevung/blunderdb/pkg/blunderdb/document/structure"
)

func TestStructureEntry(t *testing.T) {
	confirmed := structure.Diagram{Page: 2, Rect: image.Rect(100, 200, 600, 650), PageSize: [2]int{1000, 1300},
		Board: "-b----E-C---eE---c-e----B-", State: structure.Confirmed, CheckersSure: true}
	e := structureEntry(confirmed, 1)
	one := &Releve{Version: Version, Document: Source{Defaults: structureDefaults()}}
	if one.StateOf(e) != ToVerify || e.Checkers != Sure || e.Context != ContextDefault || e.Page != 3 || e.Position != "XGID=-b----E-C---eE---c-e----B-:0:0:1:00:0:0:0:0:10" || e.Rect == nil || e.Rect.W != 0.5 {
		t.Errorf("confirmed entry %+v", e)
	}
	if len(e.Doubts) != 1 || e.Doubts[0] != ContextDoubt || !one.CheckersConfirmed(e) {
		t.Errorf("confirmed checkers: doubts %+v, checkers confirmed %v", e.Doubts, one.CheckersConfirmed(e))
	}
	// A drawn cube holds the checkers confirmed; a lead on how a stack was
	// counted is not a doubt once the pips confirm the checkers.
	cube := capture.Doubt{Field: "cube", Reason: "a cube is drawn on the top side"}
	lead := capture.Doubt{Field: "point", Point: 6, Reason: "a dark stack counted by its length"}
	drawn := structureEntry(structure.Diagram{Board: confirmed.Board, State: structure.ToVerify, CheckersSure: true, Doubts: []capture.Doubt{cube, lead}}, 2)
	if drawn.Context != ContextToVerify || len(drawn.Doubts) != 2 || drawn.Doubts[0] != ContextDoubt || drawn.Doubts[1] != cube || !one.CheckersConfirmed(drawn) {
		t.Errorf("drawn cube %+v", drawn)
	}
	pips := capture.Doubt{Field: "pips", Reason: "no printed pip count was found next to the diagram"}
	tv := structureEntry(structure.Diagram{Board: confirmed.Board, State: structure.ToVerify, Doubts: []capture.Doubt{pips, cube}}, 3)
	if one.StateOf(tv) != ToVerify || tv.Checkers != ToVerify || len(tv.Doubts) != 3 || tv.Doubts[0] != ContextDoubt || tv.Doubts[1] != pips || one.CheckersConfirmed(tv) {
		t.Errorf("to-verify entry %+v", tv)
	}
	// No checker of one side: not a position, the entry keeps its place.
	empty := structureEntry(structure.Diagram{Board: "-B----E-C---E-------------", State: structure.ToVerify}, 4)
	if one.StateOf(empty) != Uncertain || empty.Position != "" || len(empty.Doubts) != 2 || empty.Doubts[0] != ContextDoubt || one.CheckersConfirmed(empty) {
		t.Errorf("empty side %+v", empty)
	}
	r := &Releve{Format: Format, Version: Version, Document: Source{Title: "t", Defaults: structureDefaults()}, Entries: []Entry{e, drawn, tv, empty}}
	for i := range r.Entries {
		r.Entries[i].Page = 3
	}
	if err := r.Validate(); err != nil {
		t.Error(err)
	}
	if s := Summarize(r); s.Sure != 0 || s.ToVerify != 3 || s.CheckersConfirmed != 2 || s.Uncertain != 1 {
		t.Errorf("summary %+v", s)
	}
}

// Whatever the structure reader says of a diagram, the entry it gives is
// never sure while its context is neither read nor under confirmed
// defaults; once they are confirmed, only an entry whose checkers the pips
// confirmed and with no cube or dice drawn is.
func TestStructureEntryNeverSure(t *testing.T) {
	boards := []string{"-b----E-C---eE---c-e----B-", "-B----E-C---E-------------", ""}
	doubts := [][]capture.Doubt{nil, {{Field: "cube", Reason: "drawn"}}, {{Field: "dice", Reason: "drawn"}}, {{Field: "pips", Reason: "none"}}}
	for _, st := range []structure.State{structure.Confirmed, structure.ToVerify, structure.Uncertain} {
		for _, cs := range []bool{true, false} {
			for _, b := range boards {
				for _, ds := range doubts {
					for _, complete := range []bool{true, false} {
						e := structureEntry(structure.Diagram{Board: b, State: st, CheckersSure: cs, Complete: complete, Doubts: ds}, 1)
						r := &Releve{Format: Format, Version: Version, Document: Source{Title: "t", Defaults: structureDefaults()}, Entries: []Entry{e}}
						if err := r.Validate(); err != nil {
							t.Errorf("state %v, checkers sure %v, board %q, doubts %v: %v", st, cs, b, ds, err)
						}
						if r.StateOf(e) == Sure || r.Importable(e, true) || len(e.Doubts) == 0 || e.Doubts[0] != ContextDoubt {
							t.Errorf("state %v, checkers sure %v, board %q, doubts %v: entry %+v", st, cs, b, ds, e)
						}
						r.Document.Defaults.Confirmed = true
						want := cs && st != structure.Uncertain && e.Position != "" && !drawnContext(ds)
						if got := r.StateOf(e) == Sure; got != want {
							t.Errorf("confirmed defaults, state %v, checkers sure %v, board %q, doubts %v: sure %v, want %v", st, cs, b, ds, got, want)
						}
					}
				}
			}
		}
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

// A real reading of a book without printed pips: fifteen checkers a side
// with no doubt are complete, under the defaults; once the defaults are
// confirmed they pass an import accepting complete entries, the diagram with
// a cube drawn on the top side aside.
func TestAnalyzeCompleteWithoutPips(t *testing.T) {
	r, err := AnalyzeFile("testdata/sans-pips", AnalyzeOptions{Title: "Sans pips"})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Entries) != 8 {
		t.Fatalf("%d entries, want 8", len(r.Entries))
	}
	var cube []int
	for _, e := range r.Entries {
		if !e.Complete || e.Checkers != ToVerify || !hasDoubt(e.Doubts, NoPipsDoubt) {
			t.Errorf("rank %d not complete: %+v", e.Rank, e)
		}
		if e.Context == ContextToVerify {
			cube = append(cube, e.Rank)
		}
	}
	if !slices.Equal(cube, []int{2}) {
		t.Errorf("context to verify on ranks %v, want the drawn cube's 2", cube)
	}
	if got := Unverified(r, true); len(got) != 8 {
		t.Errorf("unconfirmed defaults: unverified %v, want every rank", got)
	}
	r.Document.Defaults.Confirmed = true
	if got := Unverified(r, false); len(got) != 8 {
		t.Errorf("complete entries not accepted: unverified %v, want every rank", got)
	}
	if got := Unverified(r, true); !slices.Equal(got, []int{2}) {
		t.Errorf("complete entries accepted: unverified %v, want [2]", got)
	}
	if _, err := r.Encode(); err != nil {
		t.Errorf("the relevé does not encode: %v", err)
	}
}

// Pips printed and disagreeing are a doubt on the checkers: never complete.
func TestDisagreeingPipsNeverComplete(t *testing.T) {
	board := "-b----E-C---eE---c-e----B-"
	for _, pips := range []capture.Doubt{
		{Field: "pips", Reason: "pips read 167 (bottom) and 167 (top), printed 160 and 167"},
		{Field: "pips", Reason: "printed pips read 167 and 167 in the text layer, 160 and 167 in the image: the two disagree"},
	} {
		e := structureEntry(structure.Diagram{Board: board, State: structure.ToVerify, Complete: true, Doubts: []capture.Doubt{pips}}, 1)
		if e.Complete {
			t.Errorf("%q: complete", pips.Reason)
		}
	}
	e := structureEntry(structure.Diagram{Board: board, State: structure.ToVerify, Complete: true, Doubts: []capture.Doubt{NoPipsDoubt, AssumedDirectionDoubt}}, 1)
	if !e.Complete {
		t.Errorf("no pips printed: not complete: %+v", e)
	}
}
