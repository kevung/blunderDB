package document

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/document/pdf"
	"github.com/kevung/blunderdb/pkg/blunderdb/document/structure"
)

// TestMeasureContextOnPages reads the context of a scanned book's diagrams
// on the pages a draw kept for tuning, and only those: a held-out page is
// never composed. It runs only where the books and the draw are, outside the
// repository: BLUNDERDB_MEASURE_DRAW is the draw (JSON, per book its
// "reglage" pages), BLUNDERDB_MEASURE_BOOKS the folder of the PDFs,
// BLUNDERDB_MEASURE_BOOK a substring naming the book, BLUNDERDB_MEASURE_OUT
// the JSON lines written (one a diagram, then one of counts).
func TestMeasureContextOnPages(t *testing.T) {
	draw, books, name, out := os.Getenv("BLUNDERDB_MEASURE_DRAW"), os.Getenv("BLUNDERDB_MEASURE_BOOKS"), os.Getenv("BLUNDERDB_MEASURE_BOOK"), os.Getenv("BLUNDERDB_MEASURE_OUT")
	if draw == "" || books == "" || name == "" || out == "" {
		t.Skip("BLUNDERDB_MEASURE_DRAW, _BOOKS, _BOOK and _OUT are not set")
	}
	raw, err := os.ReadFile(draw)
	if err != nil {
		t.Fatal(err)
	}
	var d struct {
		Livres map[string]struct {
			Chemin  string
			Reglage []int
		}
	}
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	var file string
	var pages []int
	for k, v := range d.Livres {
		if strings.Contains(k, name) {
			if file != "" {
				t.Fatalf("%q names more than one book", name)
			}
			file, pages = filepath.Join(books, v.Chemin), v.Reglage
		}
	}
	if file == "" {
		t.Fatalf("no book named %q", name)
	}
	sort.Ints(pages)
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	rd, err := pdf.Open(data, pdf.DefaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	res, err := structure.Read(t.Context(), pickedScans{pdfScans{t.Context(), rd, rd.NumPages()}, pages})
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	enc := json.NewEncoder(f)
	count := map[string]int{"diagrams": len(res.Diagrams)}
	for _, x := range res.Diagrams {
		c := x.Context
		_ = enc.Encode(map[string]any{"page": pages[x.Page], "rect": x.Rect, "context": c, "doubts": x.Doubts,
			"board": x.Board, "sure": x.CheckersSure, "complete": x.Complete, "pips": x.Pips, "printed": x.Printed, "textPips": x.TextPips, "imagePips": x.ImagePips})
		for k, v := range map[string]bool{"cube read": c.CubeRead, "dice read": c.DiceRead, "turn read": c.TurnRead, "context read": c.Read, "score read": c.Score != nil} {
			if v {
				count[k]++
			}
		}
		for _, y := range x.Doubts {
			switch y.Field {
			case "cube", "dice", "turn", "score":
				count[y.Field+" doubts"]++
			}
		}
	}
	_ = enc.Encode(count)
	t.Log(fmt.Sprint(count))
}

// pickedScans are some pages of a PDF, renumbered from 0 in their order.
type pickedScans struct {
	all   pdfScans
	pages []int // 1-based pages of the PDF
}

func (s pickedScans) Len() int { return len(s.pages) }

func (s pickedScans) Page(i int) (structure.Page, error) { return s.all.Page(s.pages[i] - 1) }
