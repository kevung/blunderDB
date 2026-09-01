// Contract cases for collections.
// The table that runs them lives in contract.go.
package storagetest

import (
	"context"
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
