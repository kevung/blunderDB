package storagetest

import (
	"context"
	"errors"
	"maps"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// cappedSourcePositions is how many positions cappedSource holds.
const cappedSourcePositions = 6

// cappedSourceCap is the cap the capped tests import cappedSource with.
const cappedSourceCap = 4

// cappedSource writes a native database of cappedSourcePositions positions,
// p1..p6 in ascending id, and returns its path. Under cappedSourceCap, p5
// and p6 are cut, and with them what reaches them:
//   - match "first" stands on p1 and is carried, holding p1;
//   - p2 is individually imported, so it holds itself;
//   - collection "keep" holds p3 and is carried;
//   - match "last" stands on p4 and p6: left out, it leaves p4 held by
//     nothing, so p4 is left out too;
//   - collection "cut" (p2, p5), deck "deck" (p5) and lesson "lesson" (a step
//     on p6) are left out whole.
func cappedSource(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "capped.db")
	src, err := sqlite.Open(ctx, path, nil)
	if err != nil {
		t.Fatalf("open source: %v", err)
	}
	defer src.Close()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("build source: %v", err)
		}
	}
	p := make([]int64, cappedSourcePositions+1)
	for n := 1; n <= cappedSourcePositions; n++ {
		pos := provenancePos(n)
		pos.IndividuallyImported = n == 2
		p[n], err = src.Positions().Save(ctx, "", &pos)
		must(err)
	}
	for hash, on := range map[string][]int64{"first": {p[1]}, "last": {p[4], p[6]}} {
		match := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 5, MatchHash: hash}
		mid, err := src.Matches().Save(ctx, "", &match)
		must(err)
		gid, err := src.Matches().CreateGame(ctx, "", &domain.Game{MatchID: mid, GameNumber: 1})
		must(err)
		for i, pid := range on {
			mv := domain.Move{GameID: gid, MoveNumber: int32(i + 1), MoveType: "checker", PositionID: pid, Player: 1, Dice: [2]int32{3, 1}, CheckerMove: "8/5 6/5"}
			_, err := src.Matches().CreateMove(ctx, "", &mv)
			must(err)
		}
	}
	keep, err := storage.CreateCollection(ctx, src, "", "keep", "", "")
	must(err)
	must(src.Collections().AddPositions(ctx, "", keep, []int64{p[3]}))
	cut, err := storage.CreateCollection(ctx, src, "", "cut", "", "")
	must(err)
	must(src.Collections().AddPositions(ctx, "", cut, []int64{p[2], p[5]}))
	deck, err := src.Anki().CreateDeck(ctx, "", "deck", "", domain.AnkiSourceCollection, cut, "")
	must(err)
	must(src.Anki().SyncWithPositions(ctx, "", deck, []int64{p[5]}))
	lesson, err := src.Lessons().Create(ctx, "", "lesson", "")
	must(err)
	_, err = src.Lessons().AddStep(ctx, "", lesson, domain.LessonStep{Title: "six", PositionID: p[6]})
	must(err)
	return path
}

func countRows(t *testing.T, s storage.Storage) (positions, matches int) {
	t.Helper()
	ctx := context.Background()
	positions, err := s.Positions().Count(ctx, "")
	if err != nil {
		t.Fatalf("count positions: %v", err)
	}
	if matches, err = s.Matches().Count(ctx, "", storage.MatchListOpts{}); err != nil {
		t.Fatalf("count matches: %v", err)
	}
	return positions, matches
}

// libraryShape names the target's collections with their member counts, its
// lessons with their step counts and its decks with their position counts.
func libraryShape(t *testing.T, s storage.Storage) map[string]int {
	t.Helper()
	ctx := context.Background()
	shape := map[string]int{}
	var colls []storage.Collection
	for c, err := range s.Collections().List(ctx, "") {
		if err != nil {
			t.Fatal(err)
		}
		colls = append(colls, *c)
	}
	for _, c := range colls {
		n := 0
		for _, err := range s.Collections().Members(ctx, "", c.ID) {
			if err != nil {
				t.Fatal(err)
			}
			n++
		}
		shape["collection "+c.Name] = n
	}
	lessons, err := s.Lessons().List(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range lessons {
		full, err := s.Lessons().Get(ctx, "", l.ID)
		if err != nil {
			t.Fatal(err)
		}
		shape["lesson "+l.Name] = len(full.Steps)
	}
	var decks []domain.AnkiDeck
	for d, err := range s.Anki().ListDecks(ctx, "") {
		if err != nil {
			t.Fatal(err)
		}
		decks = append(decks, *d)
	}
	for _, d := range decks {
		n, err := s.Anki().DeckPositionCount(ctx, "", d.ID)
		if err != nil {
			t.Fatal(err)
		}
		shape["deck "+d.Name] = n
	}
	return shape
}

// testDBImportCapped: a capped native import carries at most the cap, the
// first positions in source id order that something holds once imported;
// it announces that total before writing, leaves out whole every match,
// collection, Lesson and deck reaching a position it did not carry, and an
// uncapped import of the same file afterwards brings all of them complete.
func testDBImportCapped(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	path := cappedSource(t)
	var first *ingest.Progress
	var totals []int
	sum, err := ingest.DBImporter{S: s}.Import(ctx, "", ingest.Source{Format: ingest.FormatNativeDB, Path: path, MaxPositions: cappedSourceCap}, func(p ingest.Progress) {
		if first == nil {
			first = &p
		}
		totals = append(totals, p.TotalPositions)
	})
	if err != nil {
		t.Fatalf("capped import: %v", err)
	}
	if first == nil || first.Positions != 0 || first.TotalPositions != 3 {
		t.Fatalf("first progress = %+v, want the total 3 before any position", first)
	}
	for _, n := range totals {
		if n != 3 {
			t.Fatalf("progress totals %v, want 3 throughout", totals)
		}
	}
	got := [6]int{sum.SavedPositions, sum.PositionsLeftOut, sum.MatchesLeftOut, sum.CollectionsLeftOut, sum.LessonsLeftOut, sum.DecksLeftOut}
	if want := [6]int{3, 3, 1, 1, 1, 1}; got != want {
		t.Fatalf("capped summary (saved, left out: positions, matches, collections, lessons, decks) = %v, want %v", got, want)
	}
	if p, m := countRows(t, s); p != 3 || m != 1 {
		t.Fatalf("after capped import: %d positions, %d matches; want 3, 1", p, m)
	}
	scores := map[int]bool{}
	for p, err := range s.Positions().List(ctx, "", storage.ListOpts{}) {
		if err != nil {
			t.Fatalf("list positions: %v", err)
		}
		scores[p.Score[0]] = true
	}
	if !scores[1] || !scores[2] || !scores[3] {
		t.Fatalf("carried positions of scores %v, want 1, 2 and 3", scores)
	}
	if shape, want := libraryShape(t, s), map[string]int{"collection keep": 1}; !maps.Equal(shape, want) {
		t.Fatalf("after capped import: %v, want %v", shape, want)
	}

	sum, err = ingest.DBImporter{S: s}.Import(ctx, "", ingest.Source{Format: ingest.FormatNativeDB, Path: path}, nil)
	if err != nil {
		t.Fatalf("uncapped import: %v", err)
	}
	if sum.PositionsLeftOut != 0 || sum.MatchesLeftOut != 0 || sum.Matches != 1 {
		t.Fatalf("uncapped summary = %+v; want the remaining match, nothing left out", sum)
	}
	if p, m := countRows(t, s); p != cappedSourcePositions || m != 2 {
		t.Fatalf("after uncapped import: %d positions, %d matches; want %d, 2", p, m, cappedSourcePositions)
	}
	want := map[string]int{"collection keep": 1, "collection cut": 2, "lesson lesson": 1, "deck deck": 1}
	if shape := libraryShape(t, s); !maps.Equal(shape, want) {
		t.Fatalf("after uncapped import: %v, want everything complete: %v", shape, want)
	}
}

// testDBImportCancelLeavesNothing: an import cancelled midway, or failing,
// writes no row at all.
func testDBImportCancelLeavesNothing(t *testing.T, s storage.Storage) {
	path := cappedSource(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err := ingest.DBImporter{S: s}.Import(ctx, "", ingest.Source{Format: ingest.FormatNativeDB, Path: path}, func(p ingest.Progress) {
		if p.Positions == 2 {
			cancel()
		}
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled import: err = %v, want context.Canceled", err)
	}
	if p, m := countRows(t, s); p != 0 || m != 0 {
		t.Fatalf("after a cancelled import: %d positions, %d matches; want none", p, m)
	}

	failing := ingest.DBImporter{S: s, BeforeCommit: func(context.Context, storage.Tx, ingest.Summary) error {
		return errors.New("refused")
	}}
	if _, err := failing.Import(context.Background(), "", ingest.Source{Format: ingest.FormatNativeDB, Path: path}, nil); err == nil {
		t.Fatal("failing import: no error")
	}
	if p, m := countRows(t, s); p != 0 || m != 0 {
		t.Fatalf("after a failed import: %d positions, %d matches; want none", p, m)
	}
}

// testDBAnalyzeWritesNothing: the analysis counts what the file would bring
// against what the target holds, and writes nothing.
func testDBAnalyzeWritesNothing(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	path := cappedSource(t)
	held := provenancePos(1)
	if _, err := s.Positions().Save(ctx, "", &held); err != nil {
		t.Fatalf("save held position: %v", err)
	}
	a, err := ingest.AnalyzeDB(ctx, s, "", path)
	if err != nil {
		t.Fatalf("AnalyzeDB: %v", err)
	}
	if a.Positions != cappedSourcePositions || a.PositionsHeld != 1 || a.Matches != 2 || a.MatchesKnown != 0 || a.SourceVersion == "" {
		t.Fatalf("analysis = %+v; want %d positions, 1 held, 2 matches, a version", a, cappedSourcePositions)
	}
	if p, m := countRows(t, s); p != 1 || m != 0 {
		t.Fatalf("after analysis: %d positions, %d matches; want the 1 held, no match", p, m)
	}
	if _, err := ingest.AnalyzeDB(ctx, s, "", filepath.Join(t.TempDir(), "missing.db")); err == nil {
		t.Fatal("AnalyzeDB of a missing file: no error")
	}
}
