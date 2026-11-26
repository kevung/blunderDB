package database

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/arkqr"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// arkIDsOf is the set of ArkIDs a library holds, and whether every position
// of it was brought on its own.
func arkIDsOf(t *testing.T, db *Database) (map[string]bool, bool) {
	t.Helper()
	ids, err := db.LoadPositionIDsByFilters(SearchFilters{})
	if err != nil {
		t.Fatal(err)
	}
	ps, err := db.LoadPositionsByIDs(ids)
	if err != nil {
		t.Fatal(err)
	}
	out, individual := map[string]bool{}, true
	for i := range ps {
		a, err := domain.EncodeArkID(&ps[i])
		if err != nil {
			t.Fatal(err)
		}
		out[a] = true
		individual = individual && ps[i].IndividuallyImported
	}
	return out, individual
}

// TestPositionSheetReimport prints a list long enough for several tranches
// with its reimport codes, then reads it back: the whole file, again (every
// position already held), and one tranche alone with a data code lost, which
// its parity rebuilds. Each position comes back as one brought on its own.
func TestPositionSheetReimport(t *testing.T) {
	t.Parallel()
	src := newTestDB(t)
	if _, err := src.ImportXGMatch(filepath.Join("testdata", "match_with_comment.xg")); err != nil {
		t.Fatal(err)
	}
	ids, err := src.LoadPositionIDsByFilters(SearchFilters{})
	if err != nil || len(ids) == 0 {
		t.Fatalf("ids %d, err %v", len(ids), err)
	}
	// The list repeats itself past one tranche: the second tranche holds
	// positions of its own order.
	list := slices.Clone(ids)
	for len(list) <= arkqr.Tranche+10 {
		list = append(list, ids...)
	}
	want, _ := arkIDsOf(t, src)

	page, err := src.PositionSheetHTML(PositionSheetSource{IDs: list}, PositionSheetOptions{Language: "fr", Analysis: true, ReimportCode: true})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "liste.html")
	if err := os.WriteFile(path, []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}

	dst := newTestDB(t)
	res, err := dst.ImportPaperArchive([]string{path})
	if err != nil {
		t.Fatal(err)
	}
	if res.Positions != len(want) || len(res.Failed) > 0 || len(res.Missing) > 0 || len(res.Imported) > 0 {
		t.Fatalf("import: %+v, want %d positions", res, len(want))
	}
	got, individual := arkIDsOf(t, dst)
	if len(got) != len(want) || !individual {
		t.Fatalf("%d positions back (individual %v), want %d", len(got), individual, len(want))
	}
	for a := range want {
		if !got[a] {
			t.Errorf("%s did not come back", a)
		}
	}
	again, err := dst.ImportPaperArchive([]string{path})
	if err != nil || again.Positions != 0 || again.KnownPositions != len(list) {
		t.Fatalf("again: %+v, %v; want %d known", again, err, len(list))
	}

	// One tranche alone, its first data code lost.
	frames, err := arkqr.ReadPage([]byte(page))
	if err != nil {
		t.Fatal(err)
	}
	var archives []uint32
	byArchive := map[uint32][]arkqr.Frame{}
	for _, f := range frames {
		if f.Kind != arkqr.KindPositions {
			t.Fatalf("a list embeds a code of kind %d", f.Kind)
		}
		if byArchive[f.Archive] == nil {
			archives = append(archives, f.Archive)
		}
		byArchive[f.Archive] = append(byArchive[f.Archive], f)
	}
	wantTranches := (len(list) + arkqr.Tranche - 1) / arkqr.Tranche
	if len(archives) != wantTranches {
		t.Fatalf("%d tranches embedded, want %d", len(archives), wantTranches)
	}
	second := byArchive[archives[1]]
	alone := filepath.Join(dir, "tranche.html")
	if err := os.WriteFile(alone, []byte(arkqr.EmbedFrames(second[1:])), 0o644); err != nil {
		t.Fatal(err)
	}
	one := newTestDB(t)
	res, err = one.ImportPaperArchive([]string{alone})
	if err != nil {
		t.Fatal(err)
	}
	if res.Repaired != 1 || len(res.Missing) > 0 || res.Positions+res.KnownPositions != min(arkqr.Tranche, len(list)-arkqr.Tranche) {
		t.Fatalf("tranche alone: %+v", res)
	}
}
