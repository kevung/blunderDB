package ingest

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
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
