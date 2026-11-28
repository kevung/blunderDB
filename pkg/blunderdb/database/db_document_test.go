package database

import (
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
