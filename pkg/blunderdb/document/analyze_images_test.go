package document

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sync/atomic"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/capture"
	"github.com/kevung/blunderdb/pkg/blunderdb/document/jpx"
	"github.com/kevung/blunderdb/pkg/blunderdb/document/structure"
)

func TestStructureEntry(t *testing.T) {
	confirmed := structure.Diagram{Page: 2, Rect: image.Rect(100, 200, 600, 650), PageSize: [2]int{1000, 1300},
		Board: "-b----E-C---eE---c-e----B-", State: structure.Confirmed, CheckersSure: true}
	e := structureEntry(confirmed, 1, plainDefaults)
	one := &Releve{Version: Version, Document: Source{Defaults: structureDefaults(structure.Profile{DirectionVoted: true})}}
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
	drawn := structureEntry(structure.Diagram{Board: confirmed.Board, State: structure.ToVerify, CheckersSure: true, Doubts: []capture.Doubt{cube, lead}}, 2, plainDefaults)
	if drawn.Context != ContextToVerify || len(drawn.Doubts) != 2 || drawn.Doubts[0] != ContextDoubt || drawn.Doubts[1] != cube || !one.CheckersConfirmed(drawn) {
		t.Errorf("drawn cube %+v", drawn)
	}
	pips := capture.Doubt{Field: "pips", Reason: "no printed pip count was found next to the diagram"}
	tv := structureEntry(structure.Diagram{Board: confirmed.Board, State: structure.ToVerify, Doubts: []capture.Doubt{pips, cube}}, 3, plainDefaults)
	if one.StateOf(tv) != ToVerify || tv.Checkers != ToVerify || len(tv.Doubts) != 3 || tv.Doubts[0] != ContextDoubt || tv.Doubts[1] != pips || one.CheckersConfirmed(tv) {
		t.Errorf("to-verify entry %+v", tv)
	}
	// No checker of one side: not a position, the entry keeps its place.
	empty := structureEntry(structure.Diagram{Board: "-B----E-C---E-------------", State: structure.ToVerify}, 4, plainDefaults)
	if one.StateOf(empty) != Uncertain || empty.Position != "" || len(empty.Doubts) != 2 || empty.Doubts[0] != ContextDoubt || one.CheckersConfirmed(empty) {
		t.Errorf("empty side %+v", empty)
	}
	r := &Releve{Format: Format, Version: Version, Document: Source{Title: "t", Defaults: structureDefaults(structure.Profile{DirectionVoted: true})}, Entries: []Entry{e, drawn, tv, empty}}
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
						e := structureEntry(structure.Diagram{Board: b, State: st, CheckersSure: cs, Complete: complete, Doubts: ds}, 1, plainDefaults)
						r := &Releve{Format: Format, Version: Version, Document: Source{Title: "t", Defaults: structureDefaults(structure.Profile{DirectionVoted: true})}, Entries: []Entry{e}}
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

// A context read whole makes the entry sure with its checkers, defaults
// unconfirmed; a centred cube read stays under the defaults; an owned cube
// read is written in the position, outside the defaults: to verify.
func TestStructureEntryContext(t *testing.T) {
	board := "-b----E-C---eE---c-e----B-"
	read := structure.Context{Cube: 2, CubeOwner: -1, CubeRead: true, Dice: [2]int{4, 1}, DiceRead: true, Turn: 1, TurnRead: true,
		Score: &structure.Score{Length: 7, Bottom: 1, Top: 3}, Read: true}
	r := &Releve{Format: Format, Version: Version, Document: Source{Title: "t", Defaults: structureDefaults(structure.Profile{DirectionVoted: true})}}
	e := structureEntry(structure.Diagram{Board: board, State: structure.Confirmed, CheckersSure: true, Context: read}, 1, plainDefaults)
	if e.Context != ContextRead || len(e.Doubts) != 0 || e.Position != "XGID="+board+":1:-1:1:41:1:3:0:7:10" || r.StateOf(e) != Sure {
		t.Errorf("context read: %+v", e)
	}
	centred := structure.Context{Cube: 1, CubeRead: true}
	d := structureEntry(structure.Diagram{Board: board, State: structure.Confirmed, CheckersSure: true, Context: centred}, 2, plainDefaults)
	if d.Context != ContextDefault || len(d.Doubts) != 1 || d.Doubts[0] != ContextDoubt || d.Position != "XGID="+board+":0:0:1:00:0:0:0:0:10" {
		t.Errorf("centred cube read under the defaults: %+v", d)
	}
	owned := structureEntry(structure.Diagram{Board: board, State: structure.Confirmed, CheckersSure: true,
		Context: structure.Context{Cube: 4, CubeOwner: 1, CubeRead: true}}, 4, plainDefaults)
	if owned.Context != ContextToVerify || len(owned.Doubts) != 2 || owned.Position != "XGID="+board+":2:1:1:00:0:0:0:0:10" {
		t.Errorf("owned cube read: %+v", owned)
	}
	turn := capture.Doubt{Field: "turn", Reason: "owned by the top player"}
	top := structureEntry(structure.Diagram{Board: board, State: structure.ToVerify, CheckersSure: true, Doubts: []capture.Doubt{turn},
		Context: structure.Context{Cube: 2, CubeOwner: -1, CubeRead: true, Turn: -1}}, 3, plainDefaults)
	if top.Context != ContextToVerify || top.Position != "XGID="+board+":1:-1:-1:00:0:0:0:0:10" {
		t.Errorf("cube owned by the top player, no dice: %+v", top)
	}
	r.Entries = []Entry{e, d, top, owned}
	for i := range r.Entries {
		r.Entries[i].Page = 1
	}
	if err := r.Validate(); err != nil {
		t.Error(err)
	}
	r.Document.Defaults.Confirmed = true
	if r.StateOf(d) != Sure || r.StateOf(top) != ToVerify || r.StateOf(owned) != ToVerify {
		t.Errorf("confirmed defaults: centred cube %v, top's cube %v, owned cube %v", r.StateOf(d), r.StateOf(top), r.StateOf(owned))
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
	if got := Unverified(r, true); len(got) != 8 {
		t.Errorf("direction of play assumed and not confirmed: unverified %v, want every rank", got)
	}
	if d := r.Document.Defaults.Direction; d == nil || d.Home != r.Document.Profile.Home || !hasDoubt(r.Entries[0].Doubts, AssumedDirectionDoubt) {
		t.Fatalf("direction default %+v", d)
	}
	r.Document.Defaults.Direction.Confirmed = true
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
		e := structureEntry(structure.Diagram{Board: board, State: structure.ToVerify, Complete: true, Doubts: []capture.Doubt{pips}}, 1, plainDefaults)
		if e.Complete {
			t.Errorf("%q: complete", pips.Reason)
		}
	}
	e := structureEntry(structure.Diagram{Board: board, State: structure.ToVerify, Complete: true, Doubts: []capture.Doubt{NoPipsDoubt, AssumedDirectionDoubt}}, 1, plainDefaults)
	if !e.Complete {
		t.Errorf("no pips printed: not complete: %+v", e)
	}
}

// plainDefaults are the defaults of a document that prints no context field.
var plainDefaults = structureDefaults(structure.Profile{DirectionVoted: true})

// A match book prints a score beside each diagram and leaves its cube,
// dice and player on roll to the defaults: the reader learns that the score
// is printed, each entry carries its own under the defaults, and one
// confirmation certifies them all; a diagram whose score is not read is not
// taken as money play, and is offered the fallback once the defaults stand.
func TestLearnedDefaultsMatchScore(t *testing.T) {
	board := "-b----E-C---eE---c-e----B-"
	score := func(b, top, length int) structure.Diagram {
		return structure.Diagram{Board: board, State: structure.Confirmed, CheckersSure: true,
			Context: structure.Context{Score: &structure.Score{Length: length, Bottom: b, Top: top}}}
	}
	ds := []structure.Diagram{score(2, 4, 7), score(0, 0, 5), score(3, 1, 9), {Board: board, State: structure.Confirmed, CheckersSure: true}}
	d := learnDefaults(ds, structure.Profile{DirectionVoted: true})
	if !slices.Equal(d.Printed, []string{"score"}) || d.Confirmed || d.Context != StructureDefaults.Context {
		t.Fatalf("learned defaults %+v", d)
	}
	r := &Releve{Format: Format, Version: Version, Document: Source{Title: "Match book", Defaults: d}}
	for i, x := range ds {
		x.Page = i
		r.Entries = append(r.Entries, structureEntry(x, i+1, d))
	}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, e := range r.Entries[:3] {
		if e.Context != ContextDefault || len(e.Doubts) != 1 || e.Doubts[0] != ContextDoubt {
			t.Errorf("rank %d, score read: %+v", e.Rank, e)
		}
	}
	if p := r.Entries[0].Position; p != "XGID="+board+":0:0:1:00:2:4:0:7:10" {
		t.Errorf("rank 1 position %s", p)
	}
	missed := r.Entries[3]
	if missed.Context != ContextToVerify || !hasDoubt(missed.Doubts, MissingScoreDoubt) {
		t.Errorf("score not read in a match book: %+v", missed)
	}
	if got := Unverified(r, false); len(got) != 4 || len(FallbackRanks(r, false)) != 0 {
		t.Errorf("unconfirmed defaults: unverified %v, fallback %v", got, FallbackRanks(r, false))
	}
	r.Document.Defaults.Confirmed = true
	if got := Unverified(r, false); !slices.Equal(got, []int{4}) || !slices.Equal(FallbackRanks(r, false), []int{4}) {
		t.Errorf("confirmed defaults: unverified %v, fallback %v", got, FallbackRanks(r, false))
	}
	if n := missed.FallbackNote(); n != "context not read, to check against the diagram: score ("+MissingScoreDoubt.Reason+")" {
		t.Errorf("fallback note %q", n)
	}
	// Without the score learned as printed, a score read stays outside the
	// defaults, as before any learning.
	alone := structureEntry(ds[0], 1, plainDefaults)
	if alone.Context != ContextToVerify {
		t.Errorf("score read, defaults printing nothing: %+v", alone)
	}
	// A score not read beside dice the text contradicts: both fields are
	// named in the fallback note, the contradiction quoted.
	dice := capture.Doubt{Field: "dice", Reason: "dice 4-1 counted in the image, the text says otherwise"}
	both := structureEntry(structure.Diagram{Board: board, State: structure.ToVerify, CheckersSure: true, Doubts: []capture.Doubt{dice}}, 5, d)
	if both.Context != ContextToVerify || !hasDoubt(both.Doubts, MissingScoreDoubt) || !r.FallbackCertified(both, false) {
		t.Errorf("score and dice not read: %+v", both)
	}
	if n := both.FallbackNote(); n != "context not read, to check against the diagram: dice ("+dice.Reason+"); score ("+MissingScoreDoubt.Reason+")" {
		t.Errorf("fallback note %q", n)
	}
	r.Document.Defaults.Printed = []string{"score", "score"}
	if err := r.Validate(); !errors.Is(err, ErrInvalid) {
		t.Errorf("a field printed twice: err = %v", err)
	}
}

// The fixture without printed pips: once its defaults and direction are
// confirmed, the one diagram whose cube is drawn and not read is offered the
// fallback with complete entries accepted, instead of a refusal.
func TestAnalyzeFallbackWithoutPips(t *testing.T) {
	r, err := AnalyzeFile("testdata/sans-pips", AnalyzeOptions{Title: "Sans pips"})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Document.Defaults.Printed) != 0 {
		t.Errorf("printed fields learned on a book that prints none: %v", r.Document.Defaults.Printed)
	}
	r.Document.Defaults.Confirmed = true
	r.Document.Defaults.Direction.Confirmed = true
	if got := FallbackRanks(r, false); len(got) != 0 {
		t.Errorf("fallback without complete entries accepted: %v", got)
	}
	if got := FallbackRanks(r, true); !slices.Equal(got, []int{2}) {
		t.Errorf("fallback with complete entries accepted: %v, want [2]", got)
	}
	if s := Summarize(r); !slices.Equal(s.FallbackComplete, []int{2}) || len(s.Fallback) != 0 {
		t.Errorf("summary fallback %v / %v", s.Fallback, s.FallbackComplete)
	}
}

// Each printed field frees its own places in the context, and only them.
func TestDefaultsCoversPrinted(t *testing.T) {
	for _, c := range []struct {
		printed []string
		ctx     string
		want    bool
	}{
		{nil, "0:0:1:00:0:0:0:0:10", true},
		{nil, "2:1:1:00:0:0:0:0:10", false},
		{[]string{"cube"}, "2:1:1:00:0:0:0:0:10", true},
		{[]string{"cube"}, "2:-1:1:41:0:0:0:0:10", false},
		{[]string{"dice"}, "0:0:-1:41:0:0:0:0:10", true},
		{[]string{"dice"}, "1:1:1:41:0:0:0:0:10", false},
		{[]string{"dice"}, "0:0:1:00:2:4:0:7:10", false},
		{[]string{"cube", "dice"}, "2:-1:-1:63:0:0:0:0:10", true},
		{[]string{"score"}, "0:0:1:00:2:4:0:7:10", true},
		{[]string{"score"}, "0:0:1:00:2:4:0:7:8", false},
	} {
		d := Defaults{Context: StructureDefaults.Context, Printed: c.printed}
		if got := d.Covers(c.ctx); got != c.want {
			t.Errorf("printed %v covers %s = %v, want %v", c.printed, c.ctx, got, c.want)
		}
	}
}

// A JPEG 2000 page file reads as the same document as its PNG.
func TestAnalyzeFileJPEG2000(t *testing.T) {
	if !IsPageImage("p1.JP2") || !IsPageImage("p1.j2k") {
		t.Fatal("a JPEG 2000 file is not a page image")
	}
	png, err := os.ReadFile("testdata/sans-pips/p001.png")
	if err != nil {
		t.Fatal(err)
	}
	jp2, err := os.ReadFile("testdata/jpx/p001.jp2")
	if err != nil {
		t.Fatal(err)
	}
	read := func(name string, data []byte) *Releve {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
		r, err := AnalyzeFile(dir, AnalyzeOptions{Title: "x"})
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	a, b := read("p001.png", png), read("p001.jp2", jp2)
	if len(a.Entries) == 0 {
		t.Fatal("the PNG page holds no diagram")
	}
	if !reflect.DeepEqual(a.Entries, b.Entries) {
		t.Errorf("the JPEG 2000 page reads differently:\n%+v\n%+v", a.Entries, b.Entries)
	}
}

// A JPEG 2000 file the guards refuse is named, not read as another error.
func TestAnalyzeFileJPEG2000Refused(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "p001.j2k"), []byte("\xff\x4f\xff\x51 garbage"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := AnalyzeFile(dir, AnalyzeOptions{}); !errors.Is(err, jpx.ErrMalformed) {
		t.Errorf("a broken JPEG 2000 file: %v", err)
	}
}

// A page image is decoded only as the PNG or JPEG it claims to be: a JP2
// file named .png reaches no decoder that another package may register.
func TestAnalyzeFilePNGNamedJP2(t *testing.T) {
	sig := "\x00\x00\x00\x0cjP  \r\n\x87\n"
	var decoded atomic.Bool
	image.RegisterFormat("jp2", sig, func(io.Reader) (image.Image, error) {
		decoded.Store(true)
		return image.NewGray(image.Rect(0, 0, 2000, 2000)), nil
	}, func(io.Reader) (image.Config, error) {
		return image.Config{ColorModel: color.GrayModel, Width: 2000, Height: 2000}, nil
	})
	jp2, err := os.ReadFile("testdata/jpx/p001.jp2")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "p001.png"), jp2, 0o644); err != nil {
		t.Fatal(err)
	}
	if r, err := AnalyzeFile(dir, AnalyzeOptions{}); err == nil {
		t.Errorf("a JP2 file named .png reads %d entries", len(r.Entries))
	}
	if decoded.Load() {
		t.Error("a registered JPEG 2000 decoder is reached")
	}
}
