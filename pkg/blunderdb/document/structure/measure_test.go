package structure

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"image"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// The measure runs only on a machine that holds the development books and
// the prototype's outputs, outside the repository; no book, page or image of
// one is kept here. BLUNDERDB_DOCUMENT_BANC names the bench: data/<name>.pdf
// for each book, verite.jsonl and mesure/pages_v<vol>.txt for the volumes
// printed with a board font. BLUNDERDB_DOCUMENT_PARITY names the prototype's
// readings: <name>.txt (the pages it read) and <name>.releve.jsonl.
// BLUNDERDB_DOCUMENT_ONLY, a comma list, restricts the books;
// BLUNDERDB_DOCUMENT_PAGES keeps the first pages of each list. Pages are
// rendered by pdftoppm at the resolution of their largest image (150 to 300
// dpi, 150 for a page of vectors) and their text layer by pdftotext -bbox.
var measured = []string{"v1", "v2", "v3", "trice", "woolsey", "robertie", "olsen", "kageyama", "mochizuki"}

func TestMeasureStructure(t *testing.T) {
	banc, parity := os.Getenv("BLUNDERDB_DOCUMENT_BANC"), os.Getenv("BLUNDERDB_DOCUMENT_PARITY")
	if banc == "" || parity == "" {
		t.Skip("BLUNDERDB_DOCUMENT_BANC and BLUNDERDB_DOCUMENT_PARITY are not set")
	}
	for _, tool := range []string{"pdftoppm", "pdftotext", "pdfimages"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s is not installed", tool)
		}
	}
	only := map[string]bool{}
	for _, b := range strings.Split(os.Getenv("BLUNDERDB_DOCUMENT_ONLY"), ",") {
		if b != "" {
			only[b] = true
		}
	}
	var truth map[[2]int]string
	for _, book := range measured {
		if len(only) > 0 && !only[book] {
			continue
		}
		t.Run(book, func(t *testing.T) {
			pdf := filepath.Join(banc, "data", book+".pdf")
			var pageList []int
			vol := 0
			if strings.HasPrefix(book, "v") {
				vol, _ = strconv.Atoi(book[1:])
				pageList = readPageList(t, filepath.Join(banc, "mesure", "pages_"+book+".txt"))
				if truth == nil {
					truth = readTruth(t, filepath.Join(banc, "verite.jsonl"))
				}
			} else {
				pageList = readPageList(t, filepath.Join(parity, book+".txt"))
			}
			if n, _ := strconv.Atoi(os.Getenv("BLUNDERDB_DOCUMENT_PAGES")); n > 0 && n < len(pageList) {
				pageList = pageList[:n]
			}
			src := newPDFPages(t, pdf, pageList)
			if dump := os.Getenv("BLUNDERDB_DOCUMENT_DUMP"); dump != "" {
				inspectReading = func(rd *boardReading) {
					f, err := os.OpenFile(dump, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
					if err != nil {
						return
					}
					defer func() { _ = f.Close() }()
					fmt.Fprintf(f, "page %d bbox %v n %d bar %+v\n", pageList[rd.page], rd.bbox, rd.n, rd.bar)
					for _, c := range rd.cols {
						if len(c.discs) > 0 {
							fmt.Fprintf(f, "  top=%v j=%d %+v\n", c.top, c.j, c.discs)
						}
					}
				}
				defer func() { inspectReading = nil }()
			}
			t0 := time.Now()
			res, err := Read(context.Background(), src)
			if err != nil {
				t.Fatal(err)
			}
			el := time.Since(t0)
			states := map[State]int{}
			rank := map[int]int{}
			got := map[[2]int]Diagram{}
			for _, d := range res.Diagrams {
				pg := pageList[d.Page]
				got[[2]int{pg, rank[pg]}] = d
				rank[pg]++
				states[d.State]++
			}
			t.Logf("%s: %d pages in %v (%v a page), profile %+v", book, len(pageList), el.Round(time.Second), (el / time.Duration(len(pageList))).Round(time.Millisecond), res.Profile)
			t.Logf("%s: %d diagrams, checkers confirmed %d, to verify %d, uncertain %d, rejected %d", book, len(res.Diagrams), states[Confirmed], states[ToVerify], states[Uncertain], res.Rejected)
			if vol > 0 {
				compareTruth(t, vol, got, truth)
			} else {
				compareProto(t, book, got, filepath.Join(parity, book+".releve.jsonl"))
			}
		})
	}
}

func readPageList(t *testing.T, path string) []int {
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var out []int
	for _, l := range strings.Split(string(data), "\n") {
		f := strings.Fields(l)
		if len(f) == 0 || strings.HasPrefix(l, "#") {
			continue
		}
		n, err := strconv.Atoi(f[len(f)-1])
		if err != nil {
			t.Fatalf("%s: %q", path, l)
		}
		out = append(out, n)
	}
	return out
}

func readTruth(t *testing.T, path string) map[[2]int]string {
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	out := map[[2]int]string{}
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
		out[[2]int{x.Vol, x.Page}] = strings.Split(strings.TrimPrefix(x.Xgid, "XGID="), ":")[0]
	}
	return out
}

// compareTruth: no sure reading may differ from the truth.
func compareTruth(t *testing.T, vol int, got map[[2]int]Diagram, truth map[[2]int]string) {
	sure, sureWrong, exact, checkersSure := 0, 0, 0, 0
	for k, d := range got {
		want, ok := truth[[2]int{vol, k[0]}]
		if !ok || k[1] > 0 {
			t.Errorf("v%d page %d rank %d: a diagram the truth does not know", vol, k[0], k[1])
			continue
		}
		if d.Board == want {
			exact++
		}
		if d.CheckersSure {
			checkersSure++
		}
		if d.State == Confirmed {
			sure++
			if d.Board != want {
				sureWrong++
				t.Errorf("v%d page %d: sure %s, truth %s", vol, k[0], d.Board, want)
			}
		}
	}
	t.Logf("v%d against the truth: %d found, %d exact, %d sure (%d wrong), %d with sure checkers", vol, len(got), exact, sure, sureWrong, checkersSure)
}

// explained are the readings sure here and not in the prototype that were
// checked by eye against the printed diagram: the reading here is right and
// the prototype's is not. Each holds only for the exact board recorded.
var explained = map[string]map[[2]int]struct{ board, why string }{
	"kageyama": {
		{141, 1}: {"-BBBBBBA------A----bbcbbA-", "the prototype misses the dark checker on 14 and calls the column mixed"},
		{143, 0}: {"-BBBBBBA----------AbbcbbA-", "the prototype sees a light checker on an empty grey point"},
		{167, 0}: {"aCBBDbC-----b-a--b-cbb----", "the prototype sees a light checker on an empty point, sixteen light"},
		{189, 1}: {"aBbBCCB-B----A---ccdb-----", "the prototype sees two pale discs on an empty grey point, off classes"},
	},
}

var protoState = map[string]State{"sur": Confirmed, "a_verifier": ToVerify, "incertain": Uncertain}

// compareProto counts the differences with the prototype's relevé; a
// reading sure here that the prototype did not call sure is an error.
func compareProto(t *testing.T, book string, got map[[2]int]Diagram, path string) {
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	type pr struct {
		Page, Rang int
		Board      string
		Etat       string
	}
	proto := map[[2]int]pr{}
	sc := bufio.NewScanner(f)
	sc.Buffer(nil, 1<<22)
	for sc.Scan() {
		var x pr
		if strings.Contains(sc.Text(), `"profil"`) {
			continue
		}
		if err := json.Unmarshal(sc.Bytes(), &x); err != nil {
			t.Fatal(err)
		}
		proto[[2]int{x.Page, x.Rang}] = x
	}
	onlyGo, onlyProto, sameBoard, sameState := 0, 0, 0, 0
	trans := map[string]int{}
	names := map[State]string{Confirmed: "checkers confirmed", ToVerify: "to-verify", Uncertain: "uncertain"}
	for k, d := range got {
		p, ok := proto[k]
		if !ok {
			onlyGo++
			t.Logf("%s p%d r%d: found here only (%s)", book, k[0], k[1], names[d.State])
			continue
		}
		ps := protoState[p.Etat]
		if d.Board == p.Board || (d.Board == "" && ps == Uncertain) {
			sameBoard++
		}
		gs := d.State
		if d.CheckersSure {
			gs = Confirmed
		}
		if gs == ps {
			sameState++
		} else {
			trans[names[ps]+"→"+names[gs]]++
			if x, ok := explained[book][k]; ok && gs == Confirmed && x.board == d.Board {
				t.Logf("%s p%d r%d: sure here, %s in the prototype, explained: %s", book, k[0], k[1], p.Etat, x.why)
			} else if gs == Confirmed {
				t.Errorf("%s p%d r%d: sure here, %s in the prototype (%s against %s)", book, k[0], k[1], p.Etat, d.Board, p.Board)
			} else {
				t.Logf("%s p%d r%d: %s → %s (%s against %s) %v", book, k[0], k[1], p.Etat, names[d.State], d.Board, p.Board, d.Doubts)
			}
		}
	}
	for k := range proto {
		if _, ok := got[k]; !ok {
			onlyProto++
			t.Logf("%s p%d r%d: found by the prototype only", book, k[0], k[1])
		}
	}
	protoSure, unreadHeld := 0, 0
	for _, p := range proto {
		if p.Etat == "sur" {
			protoSure++
		}
	}
	for _, d := range got {
		if d.CheckersSure && d.State != Confirmed {
			unreadHeld++
		}
	}
	t.Logf("%s against the prototype: %d here, %d there, %d only here, %d only there, same board %d, same state %d (prototype sure %d), changes %v; sure checkers held to verify by a drawn cube or dice: %d",
		book, len(got), len(proto), onlyGo, onlyProto, sameBoard, sameState, protoSure, trans, unreadHeld)
}

// pdfPages renders the listed pages of a PDF on demand, a few ahead.
type pdfPages struct {
	t     *testing.T
	pdf   string
	pages []int
	words map[int][]rawWord
	dir   string
	mu    sync.Mutex
	fut   map[int]chan rendered
}

type rawWord struct {
	x0, y0, x1, y1 float64
	text           string
}

type rendered struct {
	img image.Image
	dpi float64
	err error
}

var reWord = regexp.MustCompile(`<word xMin="([\d.]+)" yMin="([\d.]+)" xMax="([\d.]+)" yMax="([\d.]+)">([^<]*)</word>`)

func newPDFPages(t *testing.T, pdf string, pages []int) *pdfPages {
	dir := t.TempDir()
	bbox := filepath.Join(dir, "bbox.html")
	words := map[int][]rawWord{}
	if err := exec.Command("pdftotext", "-bbox", pdf, bbox).Run(); err == nil {
		f, err := os.Open(bbox)
		if err != nil {
			t.Fatal(err)
		}
		sc := bufio.NewScanner(f)
		sc.Buffer(nil, 1<<22)
		page := 0
		for sc.Scan() {
			l := sc.Text()
			if strings.Contains(l, "<page") {
				page++
			}
			if m := reWord.FindStringSubmatch(l); m != nil {
				v := [4]float64{}
				for i := range v {
					v[i], _ = strconv.ParseFloat(m[i+1], 64)
				}
				words[page] = append(words[page], rawWord{v[0], v[1], v[2], v[3], unescape(m[5])})
			}
		}
		_ = f.Close()
		_ = os.Remove(bbox)
	}
	return &pdfPages{t: t, pdf: pdf, pages: pages, words: words, dir: dir, fut: map[int]chan rendered{}}
}

func unescape(s string) string {
	return strings.NewReplacer("&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", `"`, "&#39;", "'").Replace(s)
}

func (p *pdfPages) Len() int { return len(p.pages) }

func (p *pdfPages) start(i int) chan rendered {
	if c, ok := p.fut[i]; ok {
		return c
	}
	c := make(chan rendered, 1)
	p.fut[i] = c
	go func() { c <- p.render(p.pages[i]) }()
	return c
}

func (p *pdfPages) Page(i int) (Page, error) {
	p.mu.Lock()
	c := p.start(i)
	for k := i + 1; k < min(i+4, len(p.pages)); k++ {
		p.start(k)
	}
	delete(p.fut, i)
	p.mu.Unlock()
	r := <-c
	if r.err != nil {
		return Page{}, r.err
	}
	pg := Page{Image: r.img, DPI: r.dpi}
	s := r.dpi / 72
	for _, w := range p.words[p.pages[i]] {
		pg.Words = append(pg.Words, Word{w.x0 * s, w.y0 * s, w.x1 * s, w.y1 * s, w.text})
	}
	return pg, nil
}

// nativeDPI is the resolution of the page's largest image, bounded to 150
// to 300; 150 for a page of vectors.
func (p *pdfPages) nativeDPI(page int) float64 {
	out, err := exec.Command("pdfimages", "-list", "-f", strconv.Itoa(page), "-l", strconv.Itoa(page), p.pdf).Output()
	if err != nil {
		return 150
	}
	best, dpi := 0, 0
	lines := strings.Split(string(out), "\n")
	for _, l := range lines[min(2, len(lines)):] {
		f := strings.Fields(l)
		if len(f) < 14 || (f[2] != "image" && f[2] != "stencil") {
			continue
		}
		w, _ := strconv.Atoi(f[3])
		h, _ := strconv.Atoi(f[4])
		x, _ := strconv.Atoi(f[12])
		if best == 0 || w*h > best {
			best, dpi = w*h, x
		}
	}
	if best == 0 {
		return 150
	}
	return float64(min(300, max(150, dpi)))
}

func (p *pdfPages) render(page int) rendered {
	dpi := p.nativeDPI(page)
	base := filepath.Join(p.dir, fmt.Sprintf("p%d", page))
	cmd := exec.Command("pdftoppm", "-f", strconv.Itoa(page), "-l", strconv.Itoa(page), "-r", strconv.Itoa(int(dpi)), "-gray", "-png", "-singlefile", p.pdf, base)
	if out, err := cmd.CombinedOutput(); err != nil {
		return rendered{err: fmt.Errorf("pdftoppm page %d: %w %s", page, err, out)}
	}
	f, err := os.Open(base + ".png")
	if err != nil {
		return rendered{err: err}
	}
	img, _, err := image.Decode(f)
	_ = f.Close()
	_ = os.Remove(base + ".png")
	return rendered{img: img, dpi: dpi, err: err}
}
