// Contract cases for collections: membership, order and the collection rows.
// The table that runs them lives in contract.go.
package storagetest

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func testCollectionMoveBetween(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	cp := checkerPos()
	posID, err := s.Positions().Save(ctx, "", &cp)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}

	src, err := s.Collections().Create(ctx, "", "src", "source")
	if err != nil {
		t.Fatalf("Create src: %v", err)
	}
	dst, err := s.Collections().Create(ctx, "", "dst", "destination")
	if err != nil {
		t.Fatalf("Create dst: %v", err)
	}

	if err := s.Collections().AddPosition(ctx, "", src, posID); err != nil {
		t.Fatalf("AddPosition: %v", err)
	}
	// Adding the same position twice is a no-op, not an error.
	if err := s.Collections().AddPosition(ctx, "", src, posID); err != nil {
		t.Fatalf("AddPosition again: %v", err)
	}
	if c, _ := s.Collections().Get(ctx, "", src); c.PositionCount != 1 {
		t.Errorf("src count after add: got %d, want 1", c.PositionCount)
	}

	if err := s.Collections().MovePosition(ctx, "", src, dst, posID); err != nil {
		t.Fatalf("MovePosition: %v", err)
	}
	if c, _ := s.Collections().Get(ctx, "", src); c.PositionCount != 0 {
		t.Errorf("src count after move: got %d, want 0", c.PositionCount)
	}
	if c, _ := s.Collections().Get(ctx, "", dst); c.PositionCount != 1 {
		t.Errorf("dst count after move: got %d, want 1", c.PositionCount)
	}

	// The moved position is reachable through the destination collection.
	var ids []int64
	for p, err := range s.Collections().Positions(ctx, "", dst) {
		if err != nil {
			t.Fatalf("Positions: %v", err)
		}
		ids = append(ids, p.ID)
	}
	if len(ids) != 1 || ids[0] != posID {
		t.Errorf("dst positions: got %v, want [%d]", ids, posID)
	}

	// CollectionsOf reflects the new membership only.
	var cols []int64
	for c, err := range s.Collections().CollectionsOf(ctx, "", posID) {
		if err != nil {
			t.Fatalf("CollectionsOf: %v", err)
		}
		cols = append(cols, c.ID)
	}
	if len(cols) != 1 || cols[0] != dst {
		t.Errorf("CollectionsOf: got %v, want [%d]", cols, dst)
	}
}

// testCollectionCopyPosition pins CopyPosition against MovePosition: unlike a
// move, a copy leaves the position in the source collection as well as adding
// it to the destination, so it belongs to both afterwards.
func testCollectionCopyPosition(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	cp := checkerPos()
	posID, err := s.Positions().Save(ctx, "", &cp)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}

	src, err := s.Collections().Create(ctx, "", "src", "source")
	if err != nil {
		t.Fatalf("Create src: %v", err)
	}
	dst, err := s.Collections().Create(ctx, "", "dst", "destination")
	if err != nil {
		t.Fatalf("Create dst: %v", err)
	}
	if err := s.Collections().AddPosition(ctx, "", src, posID); err != nil {
		t.Fatalf("AddPosition: %v", err)
	}

	if err := s.Collections().CopyPosition(ctx, "", dst, posID); err != nil {
		t.Fatalf("CopyPosition: %v", err)
	}
	// Copying the same position again is a no-op, not an error (mirrors
	// AddPosition's dedup).
	if err := s.Collections().CopyPosition(ctx, "", dst, posID); err != nil {
		t.Fatalf("CopyPosition again: %v", err)
	}

	if c, _ := s.Collections().Get(ctx, "", src); c.PositionCount != 1 {
		t.Errorf("src count after copy: got %d, want 1 (copy must not remove from source)", c.PositionCount)
	}
	if c, _ := s.Collections().Get(ctx, "", dst); c.PositionCount != 1 {
		t.Errorf("dst count after copy: got %d, want 1", c.PositionCount)
	}

	var cols []int64
	for c, err := range s.Collections().CollectionsOf(ctx, "", posID) {
		if err != nil {
			t.Fatalf("CollectionsOf: %v", err)
		}
		cols = append(cols, c.ID)
	}
	if len(cols) != 2 {
		t.Errorf("CollectionsOf after copy: got %v, want both %d and %d", cols, src, dst)
	}
}

// collectionPositionIDs returns the ids of a collection's positions in
// collection order.
func collectionPositionIDs(t *testing.T, s storage.Storage, collectionID int64) []int64 {
	t.Helper()
	var ids []int64
	for p, err := range s.Collections().Positions(context.Background(), "", collectionID) {
		if err != nil {
			t.Fatalf("Positions of collection %d: %v", collectionID, err)
		}
		ids = append(ids, p.ID)
	}
	return ids
}

// collectionIDs returns the ids of every collection in list order.
func collectionIDs(t *testing.T, s storage.Storage) []int64 {
	t.Helper()
	var ids []int64
	for c, err := range s.Collections().List(context.Background(), "") {
		if err != nil {
			t.Fatalf("List collections: %v", err)
		}
		ids = append(ids, c.ID)
	}
	return ids
}

// collectionsOfIDs returns the ids of the collections a position belongs to.
func collectionsOfIDs(t *testing.T, s storage.Storage, positionID int64) []int64 {
	t.Helper()
	var ids []int64
	for c, err := range s.Collections().CollectionsOf(context.Background(), "", positionID) {
		if err != nil {
			t.Fatalf("CollectionsOf %d: %v", positionID, err)
		}
		ids = append(ids, c.ID)
	}
	return ids
}

// positionCount reads a collection's denormalised member count.
func positionCount(t *testing.T, s storage.Storage, collectionID int64) int {
	t.Helper()
	c, err := s.Collections().Get(context.Background(), "", collectionID)
	if err != nil {
		t.Fatalf("Get collection %d: %v", collectionID, err)
	}
	return c.PositionCount
}

// testCollectionReorderPositions pins the order of a collection's members:
// AddPositions appends in the given order, ReorderPositions rewrites the
// order wholesale, a later AddPositions appends after the reordered tail
// and ignores positions already present, and RemovePosition(s) closes the
// gaps without disturbing the remaining order.
func testCollectionReorderPositions(t *testing.T, s storage.Storage) {
	c := context.Background()
	save := func(n int) int64 {
		p := provenancePos(n)
		id, err := s.Positions().Save(c, "", &p)
		if err != nil {
			t.Fatalf("Save position %d: %v", n, err)
		}
		return id
	}
	a, b, d, e := save(1), save(2), save(3), save(4)

	col, err := s.Collections().Create(c, "", "ordered", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := s.Collections().AddPositions(c, "", col, []int64{a, b, d}); err != nil {
		t.Fatalf("AddPositions: %v", err)
	}
	if got := collectionPositionIDs(t, s, col); !equalIDs(got, []int64{a, b, d}) {
		t.Errorf("after AddPositions: got %v, want [%d %d %d]", got, a, b, d)
	}

	if err := s.Collections().ReorderPositions(c, "", col, []int64{d, a, b}); err != nil {
		t.Fatalf("ReorderPositions: %v", err)
	}
	if got := collectionPositionIDs(t, s, col); !equalIDs(got, []int64{d, a, b}) {
		t.Errorf("after ReorderPositions: got %v, want [%d %d %d]", got, d, a, b)
	}

	// b is already a member: only e is appended, after the reordered tail.
	if err := s.Collections().AddPositions(c, "", col, []int64{b, e}); err != nil {
		t.Fatalf("AddPositions with a duplicate: %v", err)
	}
	if got := collectionPositionIDs(t, s, col); !equalIDs(got, []int64{d, a, b, e}) {
		t.Errorf("after AddPositions [b e]: got %v, want [%d %d %d %d]", got, d, a, b, e)
	}
	if n := positionCount(t, s, col); n != 4 {
		t.Errorf("PositionCount: got %d, want 4", n)
	}

	if err := s.Collections().RemovePositions(c, "", col, []int64{a, e}); err != nil {
		t.Fatalf("RemovePositions: %v", err)
	}
	if got := collectionPositionIDs(t, s, col); !equalIDs(got, []int64{d, b}) {
		t.Errorf("after RemovePositions [a e]: got %v, want [%d %d]", got, d, b)
	}
	if err := s.Collections().RemovePosition(c, "", col, b); err != nil {
		t.Fatalf("RemovePosition: %v", err)
	}
	// Removing a non-member is a no-op, not an error.
	if err := s.Collections().RemovePosition(c, "", col, b); err != nil {
		t.Fatalf("RemovePosition of a non-member: %v", err)
	}
	if got := collectionPositionIDs(t, s, col); !equalIDs(got, []int64{d}) {
		t.Errorf("after RemovePosition b: got %v, want [%d]", got, d)
	}
	if n := positionCount(t, s, col); n != 1 {
		t.Errorf("PositionCount after removals: got %d, want 1", n)
	}
	// The removed positions still exist: membership is not ownership.
	if _, err := s.Positions().Load(c, "", a); err != nil {
		t.Errorf("removed position %d should still load: %v", a, err)
	}
}

// testCollectionRenameAndDelete pins the collection rows themselves: Create
// appends to the list order, Get reports the fields and a zero count, Update
// renames in place, Reorder rewrites the list order, and Delete makes the id
// unknown (ErrNotFound) and drops its memberships while leaving the positions
// and the other collections intact.
func testCollectionRenameAndDelete(t *testing.T, s storage.Storage) {
	c := context.Background()
	cols := s.Collections()

	openings, err := cols.Create(c, "", "Openings", "early-game positions")
	if err != nil {
		t.Fatalf("Create Openings: %v", err)
	}
	endgames, err := cols.Create(c, "", "Endgames", "")
	if err != nil {
		t.Fatalf("Create Endgames: %v", err)
	}
	if openings == 0 || endgames == 0 || openings == endgames {
		t.Fatalf("Create ids: got %d and %d", openings, endgames)
	}

	got, err := cols.Get(c, "", openings)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != openings || got.Name != "Openings" || got.Description != "early-game positions" || got.PositionCount != 0 {
		t.Errorf("Get: %+v", got)
	}
	if got.CreatedAt == "" || got.UpdatedAt == "" {
		t.Errorf("Get timestamps: created %q, updated %q, want both set", got.CreatedAt, got.UpdatedAt)
	}
	if got := collectionIDs(t, s); !equalIDs(got, []int64{openings, endgames}) {
		t.Errorf("List in creation order: got %v, want [%d %d]", got, openings, endgames)
	}

	if err := cols.Update(c, "", openings, "Opening theory", "renamed"); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got, _ := cols.Get(c, "", openings); got.Name != "Opening theory" || got.Description != "renamed" {
		t.Errorf("after Update: %+v", got)
	}
	if got, _ := cols.Get(c, "", endgames); got.Name != "Endgames" || got.Description != "" {
		t.Errorf("Update must not touch another collection: %+v", got)
	}

	if err := cols.Reorder(c, "", []int64{endgames, openings}); err != nil {
		t.Fatalf("Reorder: %v", err)
	}
	if got := collectionIDs(t, s); !equalIDs(got, []int64{endgames, openings}) {
		t.Errorf("List after Reorder: got %v, want [%d %d]", got, endgames, openings)
	}

	// A position in both collections: deleting one collection drops that
	// membership only.
	p := provenancePos(1)
	pos, err := s.Positions().Save(c, "", &p)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}
	if err := cols.AddPosition(c, "", openings, pos); err != nil {
		t.Fatalf("AddPosition openings: %v", err)
	}
	if err := cols.AddPosition(c, "", endgames, pos); err != nil {
		t.Fatalf("AddPosition endgames: %v", err)
	}

	if err := cols.Delete(c, "", openings); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := cols.Get(c, "", openings); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Get after Delete: got %v, want ErrNotFound", err)
	}
	if got := collectionIDs(t, s); !equalIDs(got, []int64{endgames}) {
		t.Errorf("List after Delete: got %v, want [%d]", got, endgames)
	}
	if got := collectionsOfIDs(t, s, pos); !equalIDs(got, []int64{endgames}) {
		t.Errorf("CollectionsOf after Delete: got %v, want [%d]", got, endgames)
	}
	if n := positionCount(t, s, endgames); n != 1 {
		t.Errorf("surviving collection count: got %d, want 1", n)
	}
	if _, err := s.Positions().Load(c, "", pos); err != nil {
		t.Errorf("position must survive its collection: %v", err)
	}
	// Deleting an unknown id is a no-op, not an error.
	if err := cols.Delete(c, "", openings); err != nil {
		t.Errorf("Delete of an unknown collection: %v", err)
	}
}

// testCollectionPositionIndexMap pins PositionIndexMap: every stored
// position — collected or not — maps to its 1-based rank by id, and an empty
// database yields an empty map.
func testCollectionPositionIndexMap(t *testing.T, s storage.Storage) {
	c := context.Background()

	if idx, err := s.Collections().PositionIndexMap(c, ""); err != nil {
		t.Fatalf("PositionIndexMap on empty db: %v", err)
	} else if len(idx) != 0 {
		t.Errorf("PositionIndexMap on empty db: got %v, want empty", idx)
	}

	save := func(n int) int64 {
		p := provenancePos(n)
		id, err := s.Positions().Save(c, "", &p)
		if err != nil {
			t.Fatalf("Save position %d: %v", n, err)
		}
		return id
	}
	p1, p2, p3 := save(1), save(2), save(3)
	col, err := s.Collections().Create(c, "", "some", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := s.Collections().AddPosition(c, "", col, p2); err != nil {
		t.Fatalf("AddPosition: %v", err)
	}

	idx, err := s.Collections().PositionIndexMap(c, "")
	if err != nil {
		t.Fatalf("PositionIndexMap: %v", err)
	}
	if len(idx) != 3 || idx[p1] != 1 || idx[p2] != 2 || idx[p3] != 3 {
		t.Errorf("PositionIndexMap: got %v, want {%d:1 %d:2 %d:3}", idx, p1, p2, p3)
	}

	// Deleting a position closes the gap: the ranks are recomputed, not stored.
	if err := s.Positions().Delete(c, "", p1); err != nil {
		t.Fatalf("Positions.Delete: %v", err)
	}
	idx, err = s.Collections().PositionIndexMap(c, "")
	if err != nil {
		t.Fatalf("PositionIndexMap after delete: %v", err)
	}
	if len(idx) != 2 || idx[p2] != 1 || idx[p3] != 2 {
		t.Errorf("PositionIndexMap after delete: got %v, want {%d:1 %d:2}", idx, p2, p3)
	}
}

// equalIDs reports whether two id slices hold the same ids in the same order.
func equalIDs(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
