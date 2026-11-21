// Contract cases for copying a whole collection (storage.CopyCollection).
package storagetest

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// saveCopyFixture saves n distinct positions in scope and returns their ids.
func saveCopyFixture(t *testing.T, ctx context.Context, s storage.Storage, scope string, n int) []int64 {
	t.Helper()
	ids := make([]int64, n)
	for i := range ids {
		p := provenancePos(i + 1)
		id, err := s.Positions().Save(ctx, scope, &p)
		if err != nil {
			t.Fatalf("Save position %d: %v", i, err)
		}
		ids[i] = id
	}
	return ids
}

// testCollectionCopyWhole: a copy to a new name makes a hand-made collection
// with the source's description and order; a copy into an existing one
// appends the missing positions after its own, in the source's order, and
// skips those it already holds. The source is left as it was.
func testCollectionCopyWhole(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	cs := s.Collections()
	p := saveCopyFixture(t, ctx, s, "", 4)
	src, err := cs.Create(ctx, "", "src", "the source")
	if err != nil {
		t.Fatalf("Create src: %v", err)
	}
	srcOrder := []int64{p[2], p[0], p[3]}
	if err := cs.AddPositions(ctx, "", src, srcOrder); err != nil {
		t.Fatalf("AddPositions src: %v", err)
	}

	res, err := storage.CopyCollection(ctx, s, "", src, storage.CollectionCopyTarget{Name: "  copy  "})
	if err != nil {
		t.Fatalf("CopyCollection to a new name: %v", err)
	}
	if res.Copied != 3 || res.ID == src || res.ID == 0 {
		t.Errorf("CopyCollection to a new name = %+v, want a new id and 3 copied", res)
	}
	c, err := cs.Get(ctx, "", res.ID)
	if err != nil {
		t.Fatalf("Get copy: %v", err)
	}
	if c.Name != "copy" || c.Description != "the source" || c.FilterQuery != "" {
		t.Errorf("copy = name %q description %q query %q, want copy / the source / none", c.Name, c.Description, c.FilterQuery)
	}
	if got := collectionPositionIDs(t, s, res.ID); !slices.Equal(got, srcOrder) {
		t.Errorf("copy holds %v, want the source's order %v", got, srcOrder)
	}

	dst, err := cs.Create(ctx, "", "dst", "")
	if err != nil {
		t.Fatalf("Create dst: %v", err)
	}
	if err := cs.AddPositions(ctx, "", dst, []int64{p[1], p[0]}); err != nil {
		t.Fatalf("AddPositions dst: %v", err)
	}
	res, err = storage.CopyCollection(ctx, s, "", src, storage.CollectionCopyTarget{ToID: dst})
	if err != nil {
		t.Fatalf("CopyCollection into dst: %v", err)
	}
	if res.ID != dst || res.Copied != 2 {
		t.Errorf("CopyCollection into dst = %+v, want id %d and 2 copied", res, dst)
	}
	if got, want := collectionPositionIDs(t, s, dst), []int64{p[1], p[0], p[2], p[3]}; !slices.Equal(got, want) {
		t.Errorf("dst holds %v, want %v", got, want)
	}
	res, err = storage.CopyCollection(ctx, s, "", src, storage.CollectionCopyTarget{ToID: dst})
	if err != nil || res.Copied != 0 {
		t.Errorf("copying again = %+v, %v; want 0 copied", res, err)
	}
	if got := collectionPositionIDs(t, s, src); !slices.Equal(got, srcOrder) {
		t.Errorf("source holds %v after the copies, want %v", got, srcOrder)
	}

	empty, err := cs.Create(ctx, "", "empty", "")
	if err != nil {
		t.Fatalf("Create empty: %v", err)
	}
	if res, err := storage.CopyCollection(ctx, s, "", empty, storage.CollectionCopyTarget{Name: "empty copy"}); err != nil || res.Copied != 0 {
		t.Errorf("copying an empty collection = %+v, %v; want a new empty collection", res, err)
	}
}

// testCollectionCopyRefused: a malformed target, an unknown collection on
// either side and a living collection on either side are refused, and
// nothing is created.
func testCollectionCopyRefused(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	cs := s.Collections()
	p := saveCopyFixture(t, ctx, s, "", 1)
	src, err := cs.Create(ctx, "", "src", "")
	if err != nil {
		t.Fatalf("Create src: %v", err)
	}
	if err := cs.AddPositions(ctx, "", src, p); err != nil {
		t.Fatalf("AddPositions: %v", err)
	}
	living, err := storage.CreateCollection(ctx, s, "", "living", "", "E>80")
	if err != nil {
		t.Fatalf("CreateCollection living: %v", err)
	}
	before := collectionIDs(t, s)

	for _, tc := range []struct {
		name   string
		from   int64
		target storage.CollectionCopyTarget
		want   error
	}{
		{"no target", src, storage.CollectionCopyTarget{}, storage.ErrInvalid},
		{"blank name", src, storage.CollectionCopyTarget{Name: "  "}, storage.ErrInvalid},
		{"both targets", src, storage.CollectionCopyTarget{ToID: living, Name: "x"}, storage.ErrInvalid},
		{"onto itself", src, storage.CollectionCopyTarget{ToID: src}, storage.ErrInvalid},
		{"unknown source", 999999, storage.CollectionCopyTarget{Name: "x"}, storage.ErrNotFound},
		{"unknown target", src, storage.CollectionCopyTarget{ToID: 999999}, storage.ErrNotFound},
		{"living source", living, storage.CollectionCopyTarget{Name: "x"}, storage.ErrInvalid},
		{"living target", src, storage.CollectionCopyTarget{ToID: living}, storage.ErrInvalid},
	} {
		if _, err := storage.CopyCollection(ctx, s, "", tc.from, tc.target); !errors.Is(err, tc.want) {
			t.Errorf("%s: got %v, want %v", tc.name, err, tc.want)
		}
	}
	if after := collectionIDs(t, s); !slices.Equal(after, before) {
		t.Errorf("a refused copy changed the collections: %v, want %v", after, before)
	}
}

// failingAddStorage makes every transaction's AddPositions fail, after the
// copy has already created its target inside it.
type failingAddStorage struct{ storage.Storage }

func (f failingAddStorage) BeginTx(ctx context.Context) (storage.Tx, error) {
	tx, err := f.Storage.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	return failingAddTx{tx}, nil
}

type failingAddTx struct{ storage.Tx }

func (f failingAddTx) Collections() storage.CollectionStore {
	return failingAddCollections{f.Tx.Collections()}
}

type failingAddCollections struct{ storage.CollectionStore }

var errInjectedAdd = errors.New("injected AddPositions failure")

func (failingAddCollections) AddPositions(context.Context, string, int64, []int64) error {
	return errInjectedAdd
}

// testCollectionCopyAtomic: when the copy fails after creating its target,
// the target is not left behind — the copy is one transaction.
func testCollectionCopyAtomic(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	cs := s.Collections()
	p := saveCopyFixture(t, ctx, s, "", 2)
	src, err := cs.Create(ctx, "", "src", "")
	if err != nil {
		t.Fatalf("Create src: %v", err)
	}
	if err := cs.AddPositions(ctx, "", src, p); err != nil {
		t.Fatalf("AddPositions: %v", err)
	}
	before := collectionIDs(t, s)
	if _, err := storage.CopyCollection(ctx, failingAddStorage{s}, "", src, storage.CollectionCopyTarget{Name: "half"}); !errors.Is(err, errInjectedAdd) {
		t.Fatalf("CopyCollection with a failing add: got %v, want the injected failure", err)
	}
	if after := collectionIDs(t, s); !slices.Equal(after, before) {
		t.Errorf("a failed copy left collections %v, want %v", after, before)
	}
}
