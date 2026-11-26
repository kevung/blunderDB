package document

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestMeasureDeepCube measures the glyph reader on the three volumes of a
// book printed by eXtreme Gammon, against the truth established outside the
// repository (one JSON line per diagram: vol, page, xgid). No book and no
// page of one is kept in the repository: the test runs only when
// BLUNDERDB_DOCUMENT_BOOKS names the folder of the PDF files and
// BLUNDERDB_DOCUMENT_TRUTH the truth file.
//
// Required: every diagram found, every one sure, every XGID identical.
func TestMeasureDeepCube(t *testing.T) {
	books, truth := os.Getenv("BLUNDERDB_DOCUMENT_BOOKS"), os.Getenv("BLUNDERDB_DOCUMENT_TRUTH")
	if books == "" || truth == "" {
		t.Skip("BLUNDERDB_DOCUMENT_BOOKS and BLUNDERDB_DOCUMENT_TRUTH are not set")
	}
	want := map[[2]int]string{}
	f, err := os.Open(truth)
	if err != nil {
		t.Fatal(err)
	}
	sc := bufio.NewScanner(f)
	sc.Buffer(nil, 1<<20)
	for sc.Scan() {
		var x struct {
			Vol, Page int
			Xgid      string
		}
		if err := json.Unmarshal(sc.Bytes(), &x); err != nil {
			t.Fatal(err)
		}
		want[[2]int{x.Vol, x.Page}] = x.Xgid
	}
	_ = f.Close()
	total, sure, same := 0, 0, 0
	for vol := 1; vol <= 3; vol++ {
		paths, _ := filepath.Glob(filepath.Join(books, fmt.Sprintf("*Deep Cube Vol. %d*.pdf", vol)))
		if len(paths) != 1 {
			t.Fatalf("volume %d: %d files in %s", vol, len(paths), books)
		}
		data, err := os.ReadFile(paths[0])
		if err != nil {
			t.Fatal(err)
		}
		t0 := time.Now()
		r, err := AnalyzePDF(data, AnalyzeOptions{})
		if err != nil {
			t.Fatalf("volume %d: %v", vol, err)
		}
		elapsed := time.Since(t0)
		volSure := 0
		for _, e := range r.Entries {
			key := [2]int{vol, e.Page}
			x, ok := want[key]
			switch {
			case !ok:
				t.Errorf("volume %d page %d: a diagram the truth does not know", vol, e.Page)
			case x != e.Position:
				t.Errorf("volume %d page %d: read %s, truth %s", vol, e.Page, e.Position, x)
			default:
				same++
			}
			if r.StateOf(e) == Sure {
				sure++
				volSure++
			} else {
				t.Errorf("volume %d page %d: %s %+v", vol, e.Page, r.StateOf(e), e.Doubts)
			}
			delete(want, key)
			total++
		}
		t.Logf("volume %d: %d diagrams, %d sure, read in %v", vol, len(r.Entries), volSure, elapsed.Round(time.Millisecond))
	}
	for k := range want {
		t.Errorf("volume %d page %d: a diagram of the truth not found", k[0], k[1])
	}
	t.Logf("total: %d diagrams, %d sure, %d identical to the truth", total, sure, same)
}
