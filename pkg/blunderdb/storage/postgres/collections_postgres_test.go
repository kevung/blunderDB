//go:build postgres

package postgres_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// TestCollectionCRUD covers Create, Get, List, Update, Delete and Reorder.
func TestCollectionCRUD(t *testing.T) {
	ctx := context.Background()
	s, _ := openMatchStore(t)

	id, err := s.Collections().Create(ctx, "", "Openings", "early-game positions")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if id == 0 {
		t.Fatal("Create returned id 0")
	}

	got, err := s.Collections().Get(ctx, "", id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "Openings" || got.Description != "early-game positions" {
		t.Errorf("Get: %+v", got)
	}
	if got.PositionCount != 0 {
		t.Errorf("Get PositionCount: got %d, want 0", got.PositionCount)
	}

	if err := s.Collections().Update(ctx, "", id, "Opening theory", "updated"); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ = s.Collections().Get(ctx, "", id)
	if got.Name != "Opening theory" || got.Description != "updated" {
		t.Errorf("after Update: %+v", got)
	}

	id2, _ := s.Collections().Create(ctx, "", "Endgames", "")
	if err := s.Collections().Reorder(ctx, "", []int64{id2, id}); err != nil {
		t.Fatalf("Reorder: %v", err)
	}
	var order []int64
	for c, err := range s.Collections().List(ctx, "") {
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		order = append(order, c.ID)
	}
	if len(order) != 2 || order[0] != id2 || order[1] != id {
		t.Errorf("List order after Reorder: got %v, want [%d %d]", order, id2, id)
	}

	if err := s.Collections().Delete(ctx, "", id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Collections().Get(ctx, "", id); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("after Delete Get: got %v, want ErrNotFound", err)
	}
}

// TestCollectionMembership covers AddPosition(s), RemovePosition(s),
// ReorderPositions, Positions, CollectionsOf and the position count.
func TestCollectionMembership(t *testing.T) {
	ctx := context.Background()
	s, _ := openMatchStore(t)

	cID, err := s.Collections().Create(ctx, "", "C", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	p1 := savePos(t, s, domain.CheckerAction)
	p2 := savePos(t, s, domain.CubeAction)

	if err := s.Collections().AddPosition(ctx, "", cID, p1); err != nil {
		t.Fatalf("AddPosition: %v", err)
	}
	// Re-adding the same position is a no-op (no duplicate, no error).
	if err := s.Collections().AddPosition(ctx, "", cID, p1); err != nil {
		t.Fatalf("AddPosition again: %v", err)
	}
	if err := s.Collections().AddPositions(ctx, "", cID, []int64{p2}); err != nil {
		t.Fatalf("AddPositions: %v", err)
	}

	got, _ := s.Collections().Get(ctx, "", cID)
	if got.PositionCount != 2 {
		t.Errorf("PositionCount: got %d, want 2", got.PositionCount)
	}

	if err := s.Collections().ReorderPositions(ctx, "", cID, []int64{p2, p1}); err != nil {
		t.Fatalf("ReorderPositions: %v", err)
	}
	var order []int64
	for p, err := range s.Collections().Positions(ctx, "", cID) {
		if err != nil {
			t.Fatalf("Positions: %v", err)
		}
		order = append(order, p.ID)
	}
	if len(order) != 2 || order[0] != p2 || order[1] != p1 {
		t.Errorf("Positions order: got %v, want [%d %d]", order, p2, p1)
	}

	var cols []int64
	for c, err := range s.Collections().CollectionsOf(ctx, "", p1) {
		if err != nil {
			t.Fatalf("CollectionsOf: %v", err)
		}
		cols = append(cols, c.ID)
	}
	if len(cols) != 1 || cols[0] != cID {
		t.Errorf("CollectionsOf: got %v, want [%d]", cols, cID)
	}

	if err := s.Collections().RemovePosition(ctx, "", cID, p1); err != nil {
		t.Fatalf("RemovePosition: %v", err)
	}
	if err := s.Collections().RemovePositions(ctx, "", cID, []int64{p2}); err != nil {
		t.Fatalf("RemovePositions: %v", err)
	}
	got, _ = s.Collections().Get(ctx, "", cID)
	if got.PositionCount != 0 {
		t.Errorf("PositionCount after removal: got %d, want 0", got.PositionCount)
	}
}

// TestCollectionMoveCopy covers MovePosition and CopyPosition.
func TestCollectionMoveCopy(t *testing.T) {
	ctx := context.Background()
	s, _ := openMatchStore(t)

	src, _ := s.Collections().Create(ctx, "", "src", "")
	dst, _ := s.Collections().Create(ctx, "", "dst", "")
	p := savePos(t, s, domain.CheckerAction)

	if err := s.Collections().AddPosition(ctx, "", src, p); err != nil {
		t.Fatalf("AddPosition: %v", err)
	}

	if err := s.Collections().MovePosition(ctx, "", src, dst, p); err != nil {
		t.Fatalf("MovePosition: %v", err)
	}
	if c, _ := s.Collections().Get(ctx, "", src); c.PositionCount != 0 {
		t.Errorf("source count after move: got %d, want 0", c.PositionCount)
	}
	if c, _ := s.Collections().Get(ctx, "", dst); c.PositionCount != 1 {
		t.Errorf("dest count after move: got %d, want 1", c.PositionCount)
	}

	// CopyPosition leaves the position in dst and also adds it to src.
	if err := s.Collections().CopyPosition(ctx, "", src, p); err != nil {
		t.Fatalf("CopyPosition: %v", err)
	}
	if c, _ := s.Collections().Get(ctx, "", src); c.PositionCount != 1 {
		t.Errorf("source count after copy: got %d, want 1", c.PositionCount)
	}
	if c, _ := s.Collections().Get(ctx, "", dst); c.PositionCount != 1 {
		t.Errorf("dest count after copy: got %d, want 1", c.PositionCount)
	}
}

// TestCollectionPositionIndexMap checks the id→1-based-index mapping.
func TestCollectionPositionIndexMap(t *testing.T) {
	ctx := context.Background()
	s, _ := openMatchStore(t)

	p1 := savePos(t, s, domain.CheckerAction)
	p2 := savePos(t, s, domain.CubeAction)

	idx, err := s.Collections().PositionIndexMap(ctx, "")
	if err != nil {
		t.Fatalf("PositionIndexMap: %v", err)
	}
	if len(idx) != 2 {
		t.Fatalf("PositionIndexMap size: got %d, want 2", len(idx))
	}
	if idx[p1] != 1 || idx[p2] != 2 {
		t.Errorf("PositionIndexMap: p1=%d p2=%d, want 1 and 2", idx[p1], idx[p2])
	}
}
