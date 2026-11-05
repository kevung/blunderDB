package ingest

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/issuance"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// libraryCounts is what a transfer may change in a library, row family by
// row family.
type libraryCounts struct {
	Positions, Matches, Collections, Members, Comments, Metadata int
}

func countLibrary(t *testing.T, s storage.Storage) libraryCounts {
	t.Helper()
	ctx := context.Background()
	var c libraryCounts
	var ids []int64
	for p, err := range s.Positions().List(ctx, "", storage.ListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, p.ID)
	}
	c.Positions = len(ids)
	for _, err := range s.Matches().List(ctx, "", storage.MatchListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		c.Matches++
	}
	var colls []int64
	for col, err := range s.Collections().List(ctx, "") {
		if err != nil {
			t.Fatal(err)
		}
		colls = append(colls, col.ID)
	}
	c.Collections = len(colls)
	for _, id := range colls {
		for _, err := range s.Collections().Members(ctx, "", id) {
			if err != nil {
				t.Fatal(err)
			}
			c.Members++
		}
	}
	comments, err := s.Comments().ByPositions(ctx, "", ids)
	if err != nil {
		t.Fatal(err)
	}
	for _, cs := range comments {
		c.Comments += len(cs)
	}
	meta, err := s.Metadata().Load(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	c.Metadata = len(meta)
	return c
}

// seedLibrary opens a library at a fresh path, fills it from an XG fixture and
// files its positions under a collection named collection.
func seedLibrary(t *testing.T, fixture, collection string) *sqlite.Storage {
	t.Helper()
	ctx := context.Background()
	s, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "lib.db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	xg := filepath.Join("..", "..", "..", "testdata", fixture)
	if _, err := (XGImporter{S: s}).Import(ctx, "", Source{Format: FormatXG, Path: xg}, nil); err != nil {
		t.Fatalf("seed %s: %v", fixture, err)
	}
	var ids []int64
	for p, err := range s.Positions().List(ctx, "", storage.ListOpts{Limit: 5}) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, p.ID)
	}
	col, err := s.Collections().Create(ctx, "", collection, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Collections().AddPositions(ctx, "", col, ids); err != nil {
		t.Fatal(err)
	}
	return s
}

// TestTransfer_BothWaysCreatesNoDuplicate: A → B then B → A leaves each
// library holding the union once, and a second round changes nothing.
func TestTransfer_BothWaysCreatesNoDuplicate(t *testing.T) {
	ctx := context.Background()
	a := seedLibrary(t, "match_with_comment.xg", "from A")
	b := seedLibrary(t, "test.xg", "from B")
	metaA, metaB := countLibrary(t, a).Metadata, countLibrary(t, b).Metadata

	whole := WholeLibraryOptions()
	if _, err := Transfer(ctx, TransferEnd{S: a}, TransferEnd{S: b}, whole, nil); err != nil {
		t.Fatalf("A → B: %v", err)
	}
	if _, err := Transfer(ctx, TransferEnd{S: b}, TransferEnd{S: a}, whole, nil); err != nil {
		t.Fatalf("B → A: %v", err)
	}
	ca, cb := countLibrary(t, a), countLibrary(t, b)
	if ca != cb {
		t.Fatalf("after A → B → A the libraries differ: A %+v, B %+v", ca, cb)
	}
	if ca.Collections != 2 || ca.Matches != 2 {
		t.Errorf("union = %+v, want 2 collections and 2 matches", ca)
	}

	for _, dir := range [][2]*sqlite.Storage{{a, b}, {b, a}} {
		rep, err := Transfer(ctx, TransferEnd{S: dir[0]}, TransferEnd{S: dir[1]}, whole, nil)
		if err != nil {
			t.Fatal(err)
		}
		if rep.Imported.Collections != 0 {
			t.Errorf("a second round changed %d collections", rep.Imported.Collections)
		}
	}
	if got := countLibrary(t, a); got != ca {
		t.Errorf("a second round changed A: %+v, was %+v", got, ca)
	}
	if got := countLibrary(t, b); got != cb {
		t.Errorf("a second round changed B: %+v, was %+v", got, cb)
	}

	// Nothing says where a copy came from: neither side gained metadata.
	if ca.Metadata != metaA || cb.Metadata != metaB {
		t.Errorf("metadata rows A %d→%d, B %d→%d: a transfer writes none", metaA, ca.Metadata, metaB, cb.Metadata)
	}
	batches, err := a.ImportBatches().List(ctx, "", storage.ListOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(batches) != 0 {
		t.Errorf("A holds %d import batches: a transfer opens none", len(batches))
	}
}

// TestTransfer_Selection: a selection of one collection carries that
// collection and its members, nothing else.
func TestTransfer_Selection(t *testing.T) {
	ctx := context.Background()
	a := seedLibrary(t, "match_with_comment.xg", "picked")
	b, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	var col int64
	for c, err := range a.Collections().List(ctx, "") {
		if err != nil {
			t.Fatal(err)
		}
		col = c.ID
	}
	opts := domain.ExportOptions{
		IncludeCollections: true, CollectionIDs: []int64{col}, IncludeCollectionPositions: true,
		IncludeAnalysis: true, IncludeComments: true, IncludePlayedMoves: true,
	}
	if _, err := Transfer(ctx, TransferEnd{S: a}, TransferEnd{S: b}, opts, nil); err != nil {
		t.Fatal(err)
	}
	got := countLibrary(t, b)
	if got.Collections != 1 || got.Members != 5 || got.Positions != 5 {
		t.Errorf("one collection of 5 transferred = %+v", got)
	}
}

// TestTransfer_Refusals: what would write on the source, mark the copy, or
// copy a library into itself is refused before anything is read.
func TestTransfer_Refusals(t *testing.T) {
	ctx := context.Background()
	a := seedLibrary(t, "match_with_comment.xg", "c")
	b, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	for name, mut := range map[string]func(*domain.ExportOptions){
		"incremental": func(o *domain.ExportOptions) { o.Incremental = true },
		"watermark":   func(o *domain.ExportOptions) { o.Watermark = "Cours" },
		"password":    func(o *domain.ExportOptions) { o.Password = "x" },
		"metadata":    func(o *domain.ExportOptions) { o.Metadata = map[string]string{"user": "x"} },
	} {
		opts := WholeLibraryOptions()
		mut(&opts)
		if _, err := Transfer(ctx, TransferEnd{S: a}, TransferEnd{S: b}, opts, nil); !errors.Is(err, storage.ErrInvalid) {
			t.Errorf("%s: err = %v, want ErrInvalid", name, err)
		}
	}
	if _, err := Transfer(ctx, TransferEnd{S: a}, TransferEnd{S: a}, WholeLibraryOptions(), nil); !errors.Is(err, ErrSelfTransfer) {
		t.Errorf("self transfer: err = %v, want ErrSelfTransfer", err)
	}
	if got := countLibrary(t, b); got.Positions != 0 {
		t.Errorf("a refused transfer wrote %d positions", got.Positions)
	}
}

// memberIDs lists the positions filed under the library's only collection,
// and the position ids outside it.
func memberIDs(t *testing.T, s storage.Storage) (col int64, in, out []int64) {
	t.Helper()
	ctx := context.Background()
	for c, err := range s.Collections().List(ctx, "") {
		if err != nil {
			t.Fatal(err)
		}
		col = c.ID
	}
	ids, err := s.Collections().PositionIDs(ctx, "", col, storage.ListOpts{})
	if err != nil {
		t.Fatal(err)
	}
	member := toSet(ids)
	for p, err := range s.Positions().List(ctx, "", storage.ListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		if member[p.ID] {
			in = append(in, p.ID)
		} else {
			out = append(out, p.ID)
		}
	}
	return col, in, out
}

func emptyLibrary(t *testing.T) *sqlite.Storage {
	t.Helper()
	b, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { b.Close() })
	return b
}

// TestTransfer_NamedCollectionCarriesOnlyItsPositions: the whole-library
// options with one collection named and the other families off copy that
// collection's positions and their comments, not the rest of the library —
// least of all a comment written on a position nobody picked.
func TestTransfer_NamedCollectionCarriesOnlyItsPositions(t *testing.T) {
	ctx := context.Background()
	a := seedLibrary(t, "match_with_comment.xg", "picked")
	col, in, out := memberIDs(t, a)
	if len(out) == 0 {
		t.Fatal("fixture: every position is in the collection")
	}
	if _, err := a.Comments().Add(ctx, "", out[0], "private note"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Comments().Add(ctx, "", in[0], "shared note"); err != nil {
		t.Fatal(err)
	}
	b := emptyLibrary(t)
	opts := WholeLibraryOptions()
	opts.CollectionIDs = []int64{col}
	opts.IncludeMatches, opts.IncludeTournaments, opts.IncludeLessons, opts.IncludeAnkiDecks = false, false, false, false
	if _, err := Transfer(ctx, TransferEnd{S: a}, TransferEnd{S: b}, opts, nil); err != nil {
		t.Fatal(err)
	}
	got := countLibrary(t, b)
	if got.Positions != len(in) || got.Collections != 1 || got.Members != len(in) || got.Matches != 0 {
		t.Errorf("one collection of %d transferred = %+v", len(in), got)
	}
	texts, err := b.Comments().ByPositions(ctx, "", allPositionIDs(t, b))
	if err != nil {
		t.Fatal(err)
	}
	for _, cs := range texts {
		for _, c := range cs {
			if c.Text == "private note" {
				t.Error("the comment of a position outside the selection was copied")
			}
		}
	}
}

func allPositionIDs(t *testing.T, s storage.Storage) []int64 {
	t.Helper()
	var ids []int64
	for p, err := range s.Positions().List(context.Background(), "", storage.ListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, p.ID)
	}
	return ids
}

// TestTransfer_CollectionIDsAloneCarriesMembers: the daemon's list form
// (wholeLibrary off, collectionIDs named, no position) copies the collection
// with its members, never an empty shell.
func TestTransfer_CollectionIDsAloneCarriesMembers(t *testing.T) {
	ctx := context.Background()
	a := seedLibrary(t, "match_with_comment.xg", "picked")
	col, in, _ := memberIDs(t, a)
	b := emptyLibrary(t)
	opts := WholeLibraryOptions()
	opts.WholeLibrary = false
	opts.CollectionIDs = []int64{col}
	opts.IncludeLessons = false
	if _, err := Transfer(ctx, TransferEnd{S: a}, TransferEnd{S: b}, opts, nil); err != nil {
		t.Fatal(err)
	}
	if got := countLibrary(t, b); got.Collections != 1 || got.Members != len(in) || got.Positions != len(in) {
		t.Errorf("collection of %d transferred = %+v", len(in), got)
	}
}

// TestTransfer_NamedMatchBringsItsTournamentAlone: a match named by id
// arrives with its tournament's record and its own positions, not with the
// tournament's other matches.
func TestTransfer_NamedMatchBringsItsTournamentAlone(t *testing.T) {
	ctx := context.Background()
	a := seedLibrary(t, "match_with_comment.xg", "c")
	if _, err := (XGImporter{S: a}).Import(ctx, "", Source{Format: FormatXG, Path: filepath.Join("..", "..", "..", "testdata", "test.xg")}, nil); err != nil {
		t.Fatal(err)
	}
	tid, err := a.Tournaments().Create(ctx, "", "Open", "2026-01-01", "Paris")
	if err != nil {
		t.Fatal(err)
	}
	var matches []int64
	for m, err := range a.Matches().List(ctx, "", storage.MatchListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		matches = append(matches, m.ID)
		if err := a.Tournaments().AddMatch(ctx, "", tid, m.ID); err != nil {
			t.Fatal(err)
		}
	}
	if len(matches) != 2 {
		t.Fatalf("fixture holds %d matches, want 2", len(matches))
	}
	b := emptyLibrary(t)
	opts := WholeLibraryOptions()
	opts.MatchIDs = matches[:1]
	opts.IncludeCollections, opts.IncludeLessons, opts.IncludeAnkiDecks = false, false, false
	if _, err := Transfer(ctx, TransferEnd{S: a}, TransferEnd{S: b}, opts, nil); err != nil {
		t.Fatal(err)
	}
	got := countLibrary(t, b)
	if got.Matches != 1 || got.Positions == 0 || got.Positions >= countLibrary(t, a).Positions {
		t.Errorf("one match transferred = %+v", got)
	}
	var tournaments int
	for tr, err := range b.Tournaments().List(ctx, "", storage.ListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		tournaments++
		if tr.Name != "Open" {
			t.Errorf("tournament %q arrived, want Open", tr.Name)
		}
	}
	if tournaments != 1 {
		t.Errorf("%d tournaments arrived, want the match's own", tournaments)
	}
}

// TestTransfer_AbsentIDRefused: an id the source does not hold is refused
// before anything is read, whatever its family.
func TestTransfer_AbsentIDRefused(t *testing.T) {
	ctx := context.Background()
	a := seedLibrary(t, "match_with_comment.xg", "c")
	b := emptyLibrary(t)
	for name, mut := range map[string]func(*domain.ExportOptions){
		"collection": func(o *domain.ExportOptions) { o.CollectionIDs = []int64{999} },
		"match":      func(o *domain.ExportOptions) { o.MatchIDs = []int64{999} },
		"tournament": func(o *domain.ExportOptions) { o.TournamentIDs = []int64{999} },
		"lesson":     func(o *domain.ExportOptions) { o.LessonIDs = []int64{999} },
		"deck":       func(o *domain.ExportOptions) { o.DeckIDs = []int64{999} },
		"position":   func(o *domain.ExportOptions) { o.WholeLibrary = false; o.PositionIDs = []int64{999999} },
	} {
		opts := WholeLibraryOptions()
		mut(&opts)
		if _, err := Transfer(ctx, TransferEnd{S: a}, TransferEnd{S: b}, opts, nil); !errors.Is(err, storage.ErrInvalid) {
			t.Errorf("%s 999: err = %v, want ErrInvalid", name, err)
		}
	}
	if got := countLibrary(t, b); got.Positions != 0 {
		t.Errorf("a refused transfer wrote %d positions", got.Positions)
	}
}

// firstMatch is the only match of s.
func firstMatch(t *testing.T, s storage.Storage) *domain.Match {
	t.Helper()
	for m, err := range s.Matches().List(context.Background(), "", storage.MatchListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		return m
	}
	t.Fatal("no match")
	return nil
}

// TestTransfer_WithoutCommentsCarriesNoMatchNote: a match's own note and the
// name of its file are the owner's; a transfer that leaves comments out leaves
// them too, while the content hash, which deduplicates, still arrives.
func TestTransfer_WithoutCommentsCarriesNoMatchNote(t *testing.T) {
	ctx := context.Background()
	for _, withComments := range []bool{false, true} {
		a := seedLibrary(t, "match_with_comment.xg", "from A")
		src := firstMatch(t, a)
		if err := a.Matches().UpdateComment(ctx, "", src.ID, "private note"); err != nil {
			t.Fatal(err)
		}
		if err := a.Matches().SetVideoSource(ctx, "", src.ID, "https://www.dropbox.com/s/SECRETTOKEN/cours.mp4"); err != nil {
			t.Fatal(err)
		}
		src = firstMatch(t, a)
		if src.FilePath == "" || src.MatchHash == "" {
			t.Fatalf("fixture lacks a file name or a hash: %+v", src)
		}
		b := emptyLibrary(t)
		opts := WholeLibraryOptions()
		opts.IncludeComments = withComments
		if _, err := Transfer(ctx, TransferEnd{S: a}, TransferEnd{S: b}, opts, nil); err != nil {
			t.Fatal(err)
		}
		got := firstMatch(t, b)
		if got.MatchHash != src.MatchHash {
			t.Errorf("withComments=%v: match hash %q, want %q: deduplication needs it", withComments, got.MatchHash, src.MatchHash)
		}
		if withComments {
			if got.Comment != "private note" || got.FilePath == "" {
				t.Errorf("with comments the note and file name travel, got %q %q", got.Comment, got.FilePath)
			}
			continue
		}
		if got.Comment != "" || got.CommentAuthor != "" || got.FilePath != issuance.AnonymousFileName(src.FilePath, false) {
			t.Errorf("without comments the target holds note %q author %q file %q", got.Comment, got.CommentAuthor, got.FilePath)
		}
		if got.VideoSource != nil && *got.VideoSource != "" {
			t.Errorf("without comments the target holds a video source %q", *got.VideoSource)
		}
		if got.FilePath == "" {
			t.Errorf("a received imported match carries no file marker: transcription.IsImported would read it as typed in")
		}
		if strings.Contains(got.FilePath, strings.TrimSuffix(src.FilePath, filepath.Ext(src.FilePath))) {
			t.Errorf("the file name leaks in %q", got.FilePath)
		}
	}
}

// TestTransfer_PositionsOutsideTheirMatchAreIndividual: five positions of a
// match, sent as a collection without the match, are the recipient's own
// positions; sent with their match they stay match positions.
func TestTransfer_PositionsOutsideTheirMatchAreIndividual(t *testing.T) {
	ctx := context.Background()
	individual := func(s storage.Storage) (n, total int) {
		for p, err := range s.Positions().List(ctx, "", storage.ListOpts{}) {
			if err != nil {
				t.Fatal(err)
			}
			total++
			if p.IndividuallyImported {
				n++
			}
		}
		return
	}
	a := seedLibrary(t, "match_with_comment.xg", "picked")
	if n, _ := individual(a); n != 0 {
		t.Fatalf("source already holds %d individual positions", n)
	}
	var col int64
	for c, err := range a.Collections().List(ctx, "") {
		if err != nil {
			t.Fatal(err)
		}
		col = c.ID
	}
	b := emptyLibrary(t)
	opts := domain.ExportOptions{IncludeCollections: true, CollectionIDs: []int64{col}, IncludeCollectionPositions: true}
	if _, err := Transfer(ctx, TransferEnd{S: a}, TransferEnd{S: b}, opts, nil); err != nil {
		t.Fatal(err)
	}
	if n, total := individual(b); total != 5 || n != 5 {
		t.Errorf("collection alone: %d of %d positions individual, want 5 of 5", n, total)
	}

	whole := emptyLibrary(t)
	if _, err := Transfer(ctx, TransferEnd{S: a}, TransferEnd{S: whole}, WholeLibraryOptions(), nil); err != nil {
		t.Fatal(err)
	}
	if n, total := individual(whole); total < 6 || n != 0 {
		t.Errorf("whole library: %d of %d positions individual, want none (they belong to the match)", n, total)
	}
}
