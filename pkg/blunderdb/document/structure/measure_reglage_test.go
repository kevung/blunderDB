package structure

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/document/pdf"
)

// The measure of the scanned books on their tuning pages only. A draw kept
// outside the repository (BLUNDERDB_DOCUMENT_TIRAGE, a JSON file naming each
// book's reserved pages) splits each book once; the reserved pages are held
// back to judge the reader at the end, so this measure never decodes one.
// BLUNDERDB_DOCUMENT_LIVRES is the books' folder, BLUNDERDB_DOCUMENT_ONLY a
// comma list of substrings of their file names. BLUNDERDB_DOCUMENT_OUT, when
// set, receives one JSON line a diagram to compare two readers;
// BLUNDERDB_DOCUMENT_CROP, a folder, receives the diagrams of the pages listed
// in BLUNDERDB_DOCUMENT_PAGES (tuning pages only) as PNG, to look at.
func TestMeasureReglage(t *testing.T) {
	tiragePath, dir := os.Getenv("BLUNDERDB_DOCUMENT_TIRAGE"), os.Getenv("BLUNDERDB_DOCUMENT_LIVRES")
	if tiragePath == "" || dir == "" {
		t.Skip("BLUNDERDB_DOCUMENT_TIRAGE and BLUNDERDB_DOCUMENT_LIVRES are not set")
	}
	data, err := os.ReadFile(tiragePath)
	if err != nil {
		t.Fatal(err)
	}
	var tirage struct {
		Livres map[string]struct {
			Chemin  string
			Pages   int
			Reserve []int
		}
	}
	if err := json.Unmarshal(data, &tirage); err != nil {
		t.Fatal(err)
	}
	var only []string
	for _, b := range strings.Split(os.Getenv("BLUNDERDB_DOCUMENT_ONLY"), ",") {
		if b != "" {
			only = append(only, b)
		}
	}
	var out *json.Encoder
	if p := os.Getenv("BLUNDERDB_DOCUMENT_OUT"); p != "" {
		f, err := os.Create(p)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = f.Close() }()
		out = json.NewEncoder(f)
	}
	cropDir := os.Getenv("BLUNDERDB_DOCUMENT_CROP")
	cropPages := map[int]bool{}
	for _, s := range strings.Split(os.Getenv("BLUNDERDB_DOCUMENT_PAGES"), ",") {
		var n int
		if _, err := fmt.Sscan(s, &n); err == nil {
			cropPages[n] = true
		}
	}
	names := make([]string, 0, len(tirage.Livres))
	for k := range tirage.Livres {
		names = append(names, k)
	}
	sort.Strings(names)
	digits := regexp.MustCompile(`[0-9]+`)
	for _, name := range names {
		book := tirage.Livres[name]
		if !strings.HasSuffix(strings.ToLower(book.Chemin), ".pdf") {
			continue
		}
		keep := len(only) == 0
		for _, o := range only {
			keep = keep || strings.Contains(name, o)
		}
		if !keep {
			continue
		}
		reserved := map[int]bool{}
		for _, p := range book.Reserve {
			reserved[p] = true
		}
		var list []int
		for p := 1; p <= book.Pages; p++ {
			if !reserved[p] {
				list = append(list, p)
			}
		}
		for p := range cropPages {
			if reserved[p] {
				t.Fatalf("%s: page %d is reserved", name, p)
			}
		}
		raw, err := os.ReadFile(filepath.Join(dir, book.Chemin))
		if err != nil {
			t.Logf("%s: %v", name, err)
			continue
		}
		r, err := pdf.Open(raw, pdf.DefaultLimits)
		if err != nil {
			t.Logf("%s: %v", name, err)
			continue
		}
		pages := listedPages{scannedPages{r}, list}
		res, err := Read(context.Background(), pages)
		if err != nil {
			t.Logf("%s: %v", name, err)
			continue
		}
		var sure, complete, verify, uncertain, borne int
		reasons := map[string]int{}
		examples := map[string][]int{}
		for _, d := range res.Diagrams {
			pg := list[d.Page]
			switch {
			case d.State == Confirmed:
				sure++
			case d.State == Uncertain:
				uncertain++
			case d.Complete:
				complete++
			default:
				verify++
			}
			seen := map[string]bool{}
			for _, x := range d.Doubts {
				k := x.Field + ": " + digits.ReplaceAllString(x.Reason, "N")
				if strings.Contains(x.Reason, "borne off") {
					borne++
				}
				if seen[k] {
					continue
				}
				seen[k] = true
				reasons[k]++
				if len(examples[k]) < 6 {
					examples[k] = append(examples[k], pg)
				}
			}
			if out != nil {
				_ = out.Encode(map[string]any{"book": name, "page": pg, "rect": d.Rect.String(), "board": d.Board,
					"state": d.State, "complete": d.Complete, "checkers": d.Checkers, "pips": d.Pips, "printed": d.Printed,
					"text": d.TextPips, "image": d.ImagePips, "doubts": d.Doubts})
			}
		}
		t.Logf("%s: %d tuning pages, %d diagrams: sure %d / complete %d / verify %d / uncertain %d; borne-off doubts %d; profile %+v",
			name, len(list), len(res.Diagrams), sure, complete, verify, uncertain, borne, res.Profile)
		ks := make([]string, 0, len(reasons))
		for k := range reasons {
			ks = append(ks, k)
		}
		sort.Slice(ks, func(i, j int) bool { return reasons[ks[i]] > reasons[ks[j]] })
		for _, k := range ks[:min(14, len(ks))] {
			t.Logf("   %4d %s %v", reasons[k], k, examples[k])
		}
		if cropDir != "" && len(cropPages) > 0 {
			for i, p := range list {
				if !cropPages[p] {
					continue
				}
				pg, err := pages.Page(i)
				if err != nil || pg.Image == nil {
					continue
				}
				for k, d := range res.Diagrams {
					if d.Page != i {
						continue
					}
					sub, ok := pg.Image.(interface {
						SubImage(image.Rectangle) image.Image
					})
					if !ok {
						continue
					}
					f, err := os.Create(filepath.Join(cropDir, fmt.Sprintf("p%d_%d.png", p, k)))
					if err != nil {
						t.Fatal(err)
					}
					_ = png.Encode(f, sub.SubImage(d.Rect.Inset(-40).Intersect(pg.Image.Bounds())))
					_ = f.Close()
					t.Logf("p%d d%d: %s state %d checkers %v pips %v printed %v text %v image %v doubts %v", p, k, d.Board, d.State, d.Checkers, d.Pips, d.Printed, d.TextPips, d.ImagePips, d.Doubts)
				}
			}
		}
	}
}
