package sqlite_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// savePos stores a fresh position with the given decision type and returns its
// id. The decision type is part of the Zobrist hash, so distinct types yield
// distinct rows.
func savePos(t *testing.T, s *sqlite.Storage, decision int) int64 {
	t.Helper()
	p := domain.InitializePosition()
	p.DecisionType = decision
	id, err := s.Positions().Save(context.Background(), "", &p)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}
	return id
}

// TestCollectionCRUD covers Create, Get, List, Update, Delete and Reorder.
func TestCollectionCRUD(t *testing.T) {
	ctx := context.Background()
	s := openMem(t)

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
	s := openMem(t)

	cID, err := s.Collections().Create(ctx, "", "C", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	p1 := savePos(t, s, domain.CheckerAction)
	p2 := savePos(t, s, domain.CubeAction)

	if err := s.Collections().AddPosition(ctx, "", cID, p1); err != nil {
		t.Fatalf("AddPosition: %v", err)
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

// TestCollectionPositionIndexMap checks the id→1-based-index mapping.
func TestCollectionPositionIndexMap(t *testing.T) {
	ctx := context.Background()
	s := openMem(t)

	p1 := savePos(t, s, domain.CheckerAction)
	p2 := savePos(t, s, domain.CubeAction)

	idx, err := s.Collections().PositionIndexMap(ctx, "")
	if err != nil {
		t.Fatalf("PositionIndexMap: %v", err)
	}
	if idx[p1] != 1 || idx[p2] != 2 {
		t.Errorf("PositionIndexMap: got %v, want {%d:1 %d:2}", idx, p1, p2)
	}
}

// TestCollectionDeleteCascadesMemberships verifies that deleting a collection
// drops its collection_position rows.
func TestCollectionDeleteCascadesMemberships(t *testing.T) {
	ctx := context.Background()
	s := openMem(t)

	cID, _ := s.Collections().Create(ctx, "", "C", "")
	p := savePos(t, s, domain.CheckerAction)
	if err := s.Collections().AddPosition(ctx, "", cID, p); err != nil {
		t.Fatalf("AddPosition: %v", err)
	}
	if err := s.Collections().Delete(ctx, "", cID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	n := 0
	for _, err := range s.Collections().CollectionsOf(ctx, "", p) {
		if err != nil {
			t.Fatalf("CollectionsOf: %v", err)
		}
		n++
	}
	if n != 0 {
		t.Errorf("memberships after collection delete: got %d, want 0", n)
	}
}
