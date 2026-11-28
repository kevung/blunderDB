package database

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/document"
)

func TestAnalyzeDocumentReportsProgressAndIsTracked(t *testing.T) {
	d := newTestDB(t)
	var got []document.Progress
	SetDocumentProgress(d, func(p document.Progress) { got = append(got, p) })
	var kinds []string
	SetTaskTracker(d, func(kind, label string, cancel func()) func(error) {
		kinds = append(kinds, kind)
		if cancel == nil {
			t.Error("a document reading must be cancellable")
		}
		return func(error) {}
	})
	out, err := d.AnalyzeDocument("pkg/blunderdb/document/testdata/xg-diagrammes.pdf", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "entries") {
		t.Fatalf("no relevé: %.80s", out)
	}
	if len(got) == 0 || got[len(got)-1].Page != got[len(got)-1].Pages {
		t.Fatalf("progress = %+v, want it to end on the last page", got)
	}
	if len(kinds) != 1 || kinds[0] != TaskImport {
		t.Fatalf("tracked kinds = %v", kinds)
	}
}

func TestDocumentVerificationKeepsItsProgressBesideTheDocument(t *testing.T) {
	d := newTestDB(t)
	data, err := os.ReadFile("pkg/blunderdb/document/testdata/xg-diagrammes.pdf")
	if err != nil {
		t.Fatal(err)
	}
	pdf := filepath.Join(t.TempDir(), "livre.pdf")
	if err := os.WriteFile(pdf, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := d.LoadDocumentReleve(pdf); err != nil || got != "" {
		t.Fatalf("no relevé yet: %q %v", got, err)
	}
	releve, err := d.AnalyzeDocument(pdf, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if dst, err := d.SaveDocumentReleve(pdf, releve); err != nil || dst != pdf+document.SidecarSuffix {
		t.Fatalf("saved to %q: %v", dst, err)
	}
	sheet, err := d.DocumentSheet(releve)
	if err != nil || len(sheet) == 0 {
		t.Fatalf("sheet: %v %v", sheet, err)
	}
	out, err := d.VerifyDocumentEntry(pdf, releve, sheet[0].Rank, true, "")
	if err != nil {
		t.Fatal(err)
	}
	kept, err := d.LoadDocumentReleve(pdf)
	if err != nil || kept != out {
		t.Fatalf("the verdict was not kept beside the document: %v", err)
	}
	if again, _ := d.DocumentSheet(kept); !again[0].Verified {
		t.Error("rank 1 not verified once read back")
	}
	if _, err := d.DocumentDiagramImage(pdf, 1, 0.1, 0.1, 0.5, 0.3); !errors.Is(err, document.ErrNoPageImage) {
		t.Errorf("a board-font page has no image: %v", err)
	}
	url, err := d.DocumentDiagramImage("pkg/blunderdb/document/testdata/sans-pips", 1, 0.1, 0.1, 0.5, 0.3)
	if err != nil || !strings.HasPrefix(url, "data:image/png;base64,") {
		t.Errorf("crop of a page image: %.40q %v", url, err)
	}
}
