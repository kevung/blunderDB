package structure

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/document/pdf"
)

// The measure of scanned books read straight from their PDF (pdf.Scan, no
// external rasteriser), against the prototype's relevés. It runs only on a
// machine that holds the development books and the prototype's outputs,
// outside the repository: BLUNDERDB_DOCUMENT_BANC names the bench
// (data/<name>.pdf, tmp/nb/relI_<name>.jsonl read with the image's pips
// only, tmp/nb/nr_<name>.jsonl or relT_<name>.jsonl with the text's).
// BLUNDERDB_DOCUMENT_ONLY, a comma list, restricts the books. The pages read
// are those the prototype found diagrams on.
var measuredScans = []string{"trice", "woolsey", "kageyama", "olsen", "mochizuki", "robertie"}

// explainedScans are the readings sure here, from one source, and not in
// the prototype, checked by eye on the printed diagram.
var explainedScans = map[string]map[[2]int]struct{ board, why string }{
	"kageyama": {
		{13, 1}: {"aDBBBBC------------ccbbbb-", "point 8 is an empty grey point; the prototype sees a light checker on it"},
	},
	"mochizuki": {
		{137, 0}: {"-ABBBCBBA---b----abcbbba--", "checked by eye: fifteen a side as read here; the prototype misses dark checkers on 17 to 23"},
	},
	"trice": {
		{45, 0}:  {"--BBBbD-C---bB---b-bbbc---", "121 and 92 printed, read here at the page's 300 dpi; the prototype, at 150 dpi, did not read the italic 121"},
		{197, 0}: {"-BbbCCB----------bbcbb----", "156 and 41 printed, read here at 300 dpi; the prototype, at 150 dpi, did not read 41 beside the tray's bars"},
	},
}

type protoEntry struct {
	Page, Rang int
	Board      string
	Etat       string
}

func readProto(t *testing.T, path string) map[[2]int]protoEntry {
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	out := map[[2]int]protoEntry{}
	sc := bufio.NewScanner(f)
	sc.Buffer(nil, 1<<22)
	for sc.Scan() {
		if strings.Contains(sc.Text(), `"profil"`) {
			continue
		}
		var x protoEntry
		if err := json.Unmarshal(sc.Bytes(), &x); err != nil {
			t.Fatal(err)
		}
		out[[2]int{x.Page, x.Rang}] = x
	}
	return out
}

// listedPages are the pages of a PDF given by number (from 1).
type listedPages struct {
	scannedPages
	list []int
}

func (l listedPages) Len() int { return len(l.list) }

func (l listedPages) Page(i int) (Page, error) {
	p, err := l.r.Page(l.list[i])
	if err != nil {
		return Page{}, err
	}
	t0 := time.Now()
	sc, err := l.r.Scan(p)
	scanTime += time.Since(t0)
	if err != nil {
		return Page{Unread: err}, nil
	}
	var ws []Word
	for _, w := range sc.Words {
		ws = append(ws, Word{w.X0, w.Y0, w.X1, w.Y1, w.Text})
	}
	return Page{Image: sc.Image, DPI: sc.DPI, Words: ws}, nil
}

var scanTime time.Duration

func peakRSS() string {
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return "?"
	}
	for _, l := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(l, "VmHWM:") {
			return strings.TrimSpace(strings.TrimPrefix(l, "VmHWM:"))
		}
	}
	return "?"
}

func TestMeasureScans(t *testing.T) {
	banc := os.Getenv("BLUNDERDB_DOCUMENT_BANC")
	if banc == "" {
		t.Skip("BLUNDERDB_DOCUMENT_BANC is not set")
	}
	only := map[string]bool{}
	for _, b := range strings.Split(os.Getenv("BLUNDERDB_DOCUMENT_ONLY"), ",") {
		if b != "" {
			only[b] = true
		}
	}
	for _, book := range measuredScans {
		if len(only) > 0 && !only[book] {
			continue
		}
		t.Run(book, func(t *testing.T) {
			nb := filepath.Join(banc, "tmp", "nb")
			textPath := filepath.Join(nb, "nr_"+book+".jsonl")
			if _, err := os.Stat(textPath); err != nil {
				textPath = filepath.Join(nb, "relT_"+book+".jsonl")
			}
			protoText := readProto(t, textPath)
			var protoImage map[[2]int]protoEntry
			if _, err := os.Stat(filepath.Join(nb, "relI_"+book+".jsonl")); err == nil {
				protoImage = readProto(t, filepath.Join(nb, "relI_"+book+".jsonl"))
			}
			seen := map[int]bool{}
			var list []int
			for k := range protoText {
				if !seen[k[0]] {
					seen[k[0]] = true
					list = append(list, k[0])
				}
			}
			sort.Ints(list)
			data, err := os.ReadFile(filepath.Join(banc, "data", book+".pdf"))
			if err != nil {
				t.Fatal(err)
			}
			r, err := pdf.Open(data, pdf.DefaultLimits)
			if err != nil {
				t.Fatal(err)
			}
			var vText, vImage []Diagram
			inspectSources = func(text, image, both []Diagram) { vText, vImage = text, image }
			defer func() { inspectSources = nil }()
			scanTime = 0
			t0 := time.Now()
			res, err := Read(context.Background(), listedPages{scannedPages{r}, list})
			if err != nil {
				t.Fatal(err)
			}
			el := time.Since(t0)
			key := func(ds []Diagram) map[[2]int]Diagram {
				out := map[[2]int]Diagram{}
				rank := map[int]int{}
				for _, d := range ds {
					pg := list[d.Page]
					out[[2]int{pg, rank[pg]}] = d
					rank[pg]++
				}
				return out
			}
			both, text, image := key(res.Diagrams), key(vText), key(vImage)
			same := 0
			for k, d := range both {
				if p, ok := protoText[k]; ok && (p.Board == d.Board || (d.Board == "" && p.Etat == "incertain")) {
					same++
				}
			}
			count := func(m map[[2]int]Diagram) (sure, checkers int) {
				for _, d := range m {
					if d.State == Confirmed {
						sure++
					}
					if d.CheckersSure {
						checkers++
					}
				}
				return
			}
			contra := 0
			for _, d := range both {
				for _, f := range d.Families {
					if f == "pips text/image" {
						contra++
					}
				}
			}
			ts, tc := count(text)
			is, ic := count(image)
			bs, bc := count(both)
			pSure := func(m map[[2]int]protoEntry) int {
				n := 0
				for _, p := range m {
					if p.Etat == "sur" {
						n++
					}
				}
				return n
			}
			t.Logf("%s: %d pages in %v (%v a page, composing %v a page), peak RSS %s, %d unread", book, len(list), el.Round(time.Second),
				(el / time.Duration(2*len(list))).Round(time.Millisecond), (scanTime / time.Duration(2*len(list))).Round(time.Millisecond), peakRSS(), len(res.Unread))
			t.Logf("%s: %d diagrams (prototype %d), %d same board; checkers confirmed / sure checkers: text %d/%d (prototype %d), image %d/%d (prototype %d), both %d/%d; contradictions %d",
				book, len(both), len(protoText), same, ts, tc, pSure(protoText), is, ic, pSure(protoImage), bs, bc, contra)
			if len(res.Unread) > 0 {
				t.Logf("%s: unread %v", book, res.Unread[:min(5, len(res.Unread))])
			}
			// Every reading sure here by its checkers is sure in one of the
			// prototype's relevés with the same board, or listed.
			for k, d := range both {
				if !d.CheckersSure {
					continue
				}
				pt, pi := protoText[k], protoImage[k]
				if (pt.Etat == "sur" && pt.Board == d.Board) || (pi.Etat == "sur" && pi.Board == d.Board) {
					continue
				}
				if (pt.Etat == "sur" && pt.Board != d.Board) || (pi.Etat == "sur" && pi.Board != d.Board) {
					t.Errorf("%s p%d r%d: sure %s, the prototype sure with another board (%s / %s)", book, k[0], k[1], d.Board, pt.Board, pi.Board)
					continue
				}
				src := "text"
				if d.TextPips == nil {
					src = "image"
				} else if d.ImagePips != nil {
					src = "both"
				}
				if x, ok := explainedScans[book][k]; ok && x.board == d.Board {
					t.Logf("%s p%d r%d: sure here only (%s), explained: %s", book, k[0], k[1], src, x.why)
				} else {
					t.Logf("%s p%d r%d: sure here only (%s): %s, prototype %s/%s %s, text pips %v image %v, printed %v pips %v", book, k[0], k[1], src, d.Board, pt.Etat, pi.Etat, pt.Board, d.TextPips, d.ImagePips, d.Printed, d.Pips)
				}
			}
			var ms runtime.MemStats
			runtime.ReadMemStats(&ms)
			t.Logf("%s: heap now %d MiB", book, ms.HeapAlloc>>20)
		})
	}
}
