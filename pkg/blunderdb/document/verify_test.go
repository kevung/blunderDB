package document

import (
	"bytes"
	"context"
	"errors"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func copyDir(t *testing.T, src string) string {
	t.Helper()
	dst := filepath.Join(t.TempDir(), filepath.Base(src))
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	ents, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range ents {
		data, err := os.ReadFile(filepath.Join(src, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dst, e.Name()), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dst
}

func TestSheetAndDiagramImage(t *testing.T) {
	r := sansPips(t)
	sheet := Sheet(r)
	if len(sheet) != len(r.Entries) {
		t.Fatalf("sheet has %d entries, relevé %d", len(sheet), len(r.Entries))
	}
	s := Summarize(r)
	counts := map[State]int{}
	for _, e := range sheet {
		counts[e.State]++
		if e.State != Sure && len(e.Fields) == 0 {
			t.Errorf("rank %d %s names no field to compare", e.Rank, e.State)
		}
	}
	if counts[Sure] != s.Sure || counts[ToVerify] != s.ToVerify || counts[Uncertain] != s.Uncertain {
		t.Errorf("sheet counts %v, summary %d/%d/%d", counts, s.Sure, s.ToVerify, s.Uncertain)
	}
	e := sheet[0]
	if e.Rect == nil {
		t.Fatal("no rect on the first entry")
	}
	data, err := DiagramImage(context.Background(), "testdata/sans-pips", e.Page, *e.Rect, 300)
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if w := img.Bounds().Dx(); w == 0 || w > 300 {
		t.Errorf("thumbnail %d pixels wide, want 1..300", w)
	}
	if _, err := DiagramImage(context.Background(), "testdata/sans-pips", 999, *e.Rect, 300); !errors.Is(err, ErrInvalid) {
		t.Errorf("page out of range: %v, want ErrInvalid", err)
	}
}

func TestDiagramImageBoardFont(t *testing.T) {
	_, err := DiagramImage(context.Background(), "testdata/xg-diagrammes.pdf", 1, Rect{X: 0.1, Y: 0.1, W: 0.5, H: 0.3}, 300)
	if !errors.Is(err, ErrNoPageImage) {
		t.Errorf("a page drawn with a board font: %v, want ErrNoPageImage", err)
	}
}

func TestSetVerified(t *testing.T) {
	r := sansPips(t)
	rank := 0
	for _, e := range r.Entries {
		if !r.Importable(e, false) && e.Position != "" {
			rank = e.Rank
			break
		}
	}
	if rank == 0 {
		t.Skip("no entry to verify in the fixture")
	}
	if err := SetVerified(r, rank, true, ""); err != nil {
		t.Fatal(err)
	}
	if e := r.Entries[rank-1]; !e.Verified || !r.Importable(e, false) {
		t.Fatalf("rank %d not importable once verified", rank)
	}
	if err := SetVerified(r, rank, false, ""); err != nil || r.Entries[rank-1].Verified {
		t.Fatalf("unverify: %v", err)
	}
	// A correction of the context leaves the defaults: the entry keeps its own.
	corrected := "XGID=-b----E-C---eE---c-e----B-:1:1:1:52:0:0:0:5:10"
	if err := SetVerified(r, rank, true, corrected); err != nil {
		t.Fatal(err)
	}
	e := r.Entries[rank-1]
	if e.Position != corrected[len("XGID="):] || !e.Verified {
		t.Fatalf("correction not written: %+v", e)
	}
	before := r.Entries[rank-1]
	if err := SetVerified(r, rank, true, "XGID=nonsense"); !errors.Is(err, ErrInvalid) {
		t.Errorf("a malformed correction: %v, want ErrInvalid", err)
	}
	if r.Entries[rank-1].Position != before.Position {
		t.Error("a refused correction changed the entry")
	}
	if err := SetVerified(r, 10_000, true, ""); !errors.Is(err, ErrNoEntry) {
		t.Errorf("unknown rank: %v", err)
	}
}

func TestSidecar(t *testing.T) {
	dir := copyDir(t, "testdata/sans-pips")
	r, err := AnalyzeFile(dir, AnalyzeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got, err := LoadSidecar(dir); err != nil || got != nil {
		t.Fatalf("no sidecar yet: %v %v", got, err)
	}
	if err := SetVerified(r, 1, true, ""); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(SidecarPath(dir), r); err != nil {
		t.Fatal(err)
	}
	got, err := LoadSidecar(dir)
	if err != nil || got == nil || !got.Entries[0].Verified {
		t.Fatalf("sidecar not read back: %v %v", got, err)
	}
	if SourceOf(SidecarPath(dir)) != filepath.Clean(dir) {
		t.Errorf("SourceOf(%s) = %q", SidecarPath(dir), SourceOf(SidecarPath(dir)))
	}
	// Another version of the document is read anew.
	if err := os.WriteFile(filepath.Join(dir, "p001.png"), []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := LoadSidecar(dir); err != nil || got != nil {
		t.Errorf("a sidecar of another version: %v %v, want none", got, err)
	}
}
