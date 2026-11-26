package structure

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

// TestMeasureContext dumps, for a PDF's pages, each diagram's board and
// context as read, one JSON line a diagram, for comparison with a truth
// kept outside the repository. Skipped unless BLUNDERDB_CONTEXT_PDF (the
// PDF), BLUNDERDB_CONTEXT_PAGES ("first-last") and BLUNDERDB_CONTEXT_OUT (the
// file written) are set.
func TestMeasureContext(t *testing.T) {
	pdf, pages, out := os.Getenv("BLUNDERDB_CONTEXT_PDF"), os.Getenv("BLUNDERDB_CONTEXT_PAGES"), os.Getenv("BLUNDERDB_CONTEXT_OUT")
	if pdf == "" || pages == "" || out == "" {
		t.Skip("BLUNDERDB_CONTEXT_PDF, BLUNDERDB_CONTEXT_PAGES and BLUNDERDB_CONTEXT_OUT are not set")
	}
	for _, tool := range []string{"pdftoppm", "pdftotext", "pdfimages"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s is not installed", tool)
		}
	}
	a, b, _ := strings.Cut(pages, "-")
	first, _ := strconv.Atoi(a)
	last, _ := strconv.Atoi(b)
	var list []int
	for p := first; p <= last; p++ {
		list = append(list, p)
	}
	res, err := Read(context.Background(), newPDFPages(t, pdf, list))
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	enc := json.NewEncoder(f)
	for _, d := range res.Diagrams {
		_ = enc.Encode(map[string]any{"page": list[d.Page], "rect": d.Rect, "board": d.Board, "state": d.State,
			"sure": d.CheckersSure, "pips": d.Pips, "context": d.Context, "doubts": d.Doubts})
	}
	t.Logf("%d diagrams, profile %+v", len(res.Diagrams), res.Profile)
}
