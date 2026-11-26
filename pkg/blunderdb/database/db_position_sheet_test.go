package database

import (
	"errors"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/printsheet"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

var sheetLineRe = regexp.MustCompile(`<div class="de"><span class="no">(\d+)</span><span class="nt">([^<]+)</span><span class="an"><span class="wh">(\d+)–(\d+)</span>`)

// TestPositionSheetOracle prints every position of an analysed match and
// reads each line back: its ArkID decodes to the stored position, whose pips
// on roll and of the opponent are the ones printed; the list is the whole
// list in its order, the analysis is on the line or on the back under the
// same number, text is escaped, and nothing is written.
func TestPositionSheetOracle(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	if _, err := db.ImportXGMatch(filepath.Join("testdata", "match_with_comment.xg")); err != nil {
		t.Fatal(err)
	}
	ids, err := db.LoadPositionIDsByFilters(SearchFilters{})
	if err != nil || len(ids) < 10 {
		t.Fatalf("ids %d, err %v", len(ids), err)
	}
	if err := db.AddComment(ids[3], `<script>alert(1)</script> à revoir`); err != nil {
		t.Fatal(err)
	}
	before := tableCounts(t, db)

	page, err := db.PositionSheetHTML(PositionSheetSource{IDs: ids}, PositionSheetOptions{Language: "fr", Title: "Mes <b>", Analysis: true, Comments: true})
	if err != nil {
		t.Fatal(err)
	}
	lines := sheetLineRe.FindAllStringSubmatch(page, -1)
	if len(lines) != len(ids) {
		t.Fatalf("%d lines printed, want %d", len(lines), len(ids))
	}
	stored, err := db.LoadPositionsByIDs(ids)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[int64]*domain.Position{}
	for i := range stored {
		byID[stored[i].ID] = &stored[i]
	}
	for i, l := range lines {
		if l[1] != strconv.Itoa(i+1) {
			t.Fatalf("line %d numbered %s", i+1, l[1])
		}
		ark := strings.ReplaceAll(l[2], "&#34;", `"`)
		dec, err := domain.DecodeArkID(ark)
		if err != nil {
			t.Fatalf("line %d: %q does not read back: %v", i+1, ark, err)
		}
		want, _ := domain.EncodeArkID(byID[ids[i]])
		if again, _ := domain.EncodeArkID(&dec); again != want || ark != want {
			t.Errorf("line %d: printed %q, reads back as %q, stored %q", i+1, ark, again, want)
		}
		printed := [2]int{atoi(t, l[3]), atoi(t, l[4])}
		if got := printsheet.OnRollPips(&dec); got != printed {
			t.Errorf("line %d: printed pips %v, the ArkID has %v", i+1, printed, got)
		}
		if got := printsheet.OnRollPips(byID[ids[i]]); got != printed {
			t.Errorf("line %d: printed pips %v, the stored position has %v", i+1, printed, got)
		}
	}
	for _, want := range []string{`Mes &lt;b&gt;`, `&lt;script&gt;alert(1)&lt;/script&gt; à revoir`, `<b>`, `data-fill`} {
		if !strings.Contains(page, want) {
			t.Errorf("sheet lacks %q", want)
		}
	}
	if n := strings.Count(page, `</span> <b>`); n < len(ids)/2 {
		t.Errorf("%d analysed lines of %d", n, len(ids))
	}
	if strings.Contains(page, `<script>alert`) || strings.Contains(page, "−0.") {
		t.Error("an unescaped comment or a loss written with its 0.")
	}

	// On the back: the front holds no analysis, the back one line per
	// position under the same numbers, with the comments.
	verso, err := db.PositionSheetHTML(PositionSheetSource{IDs: ids}, PositionSheetOptions{Analysis: true, Verso: true, Comments: true})
	if err != nil {
		t.Fatal(err)
	}
	front, back, ok := strings.Cut(verso, "<div data-verso>")
	if !ok {
		t.Fatal("no back")
	}
	if n := strings.Count(back, `<div class="de"><span class="no">`); n != len(ids) {
		t.Errorf("%d answers on the back, want %d", n, len(ids))
	}
	if strings.Contains(front, "<b>") {
		t.Error("the front gives the answers away")
	}
	if strings.Contains(front, "alert(1)") || !strings.Contains(back, "alert(1)") {
		t.Error("the comment belongs on the back")
	}

	if after := tableCounts(t, db); after != before {
		t.Errorf("printing wrote to the library: %s, then %s", before, after)
	}
}

// TestPositionSheetSources prints a collection and a query as the same list
// of ids prints, and refuses what names no list, or too long a one.
func TestPositionSheetSources(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	if _, err := db.ImportXGMatch(filepath.Join("testdata", "match_with_comment.xg")); err != nil {
		t.Fatal(err)
	}
	ids, err := db.LoadPositionIDsByFilters(SearchFilters{})
	if err != nil {
		t.Fatal(err)
	}
	cid, err := db.CreateCollection("Révisions", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AddPositionsToCollection(cid, ids[:5]); err != nil {
		t.Fatal(err)
	}
	page, err := db.PositionSheetHTML(PositionSheetSource{Collection: cid}, printsheet.DefaultPositionsOptions())
	if err != nil {
		t.Fatal(err)
	}
	if n := len(sheetLineRe.FindAllString(page, -1)); n != 5 || !strings.Contains(page, "Révisions") {
		t.Errorf("collection: %d lines, title present %v", n, strings.Contains(page, "Révisions"))
	}
	page, err = db.PositionSheetHTML(PositionSheetSource{Query: "s"}, printsheet.DefaultPositionsOptions())
	if err != nil {
		t.Fatal(err)
	}
	if n := len(sheetLineRe.FindAllString(page, -1)); n == 0 {
		t.Error("query: no line")
	}

	tooMany := make([]int64, printsheet.MaxPositions+1)
	for i := range tooMany {
		tooMany[i] = ids[0]
	}
	for name, src := range map[string]PositionSheetSource{
		"nothing":  {},
		"two":      {IDs: ids[:1], Collection: cid},
		"too many": {IDs: tooMany},
		"unknown":  {Query: "zzzunknowntoken"},
	} {
		if _, err := db.PositionSheetHTML(src, printsheet.DefaultPositionsOptions()); !errors.Is(err, storage.ErrInvalid) {
			t.Errorf("%s: err %v, want ErrInvalid", name, err)
		}
	}
	if _, err := db.PositionSheetHTML(PositionSheetSource{IDs: []int64{1 << 40}}, printsheet.DefaultPositionsOptions()); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("unknown id: err %v, want ErrNotFound", err)
	}
}

func atoi(t *testing.T, s string) int {
	t.Helper()
	n, err := strconv.Atoi(s)
	if err != nil {
		t.Fatal(err)
	}
	return n
}
