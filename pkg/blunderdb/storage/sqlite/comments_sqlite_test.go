package sqlite_test

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// TestCommentAddUpdateDelete covers Add, Update, Delete and that a comment's
// id is stable across an update.
func TestCommentAddUpdateDelete(t *testing.T) {
	ctx := context.Background()
	s := openMem(t)
	pos := savePos(t, s, domain.CheckerAction)

	id, err := s.Comments().Add(ctx, "", pos, "first note")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if id == 0 {
		t.Fatal("Add returned id 0")
	}

	if err := s.Comments().Update(ctx, "", id, "edited note"); err != nil {
		t.Fatalf("Update: %v", err)
	}
	text, err := s.Comments().Text(ctx, "", pos)
	if err != nil {
		t.Fatalf("Text: %v", err)
	}
	if text != "edited note" {
		t.Errorf("Text after Update: got %q, want %q", text, "edited note")
	}

	if err := s.Comments().Delete(ctx, "", id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	text, _ = s.Comments().Text(ctx, "", pos)
	if text != "" {
		t.Errorf("Text after Delete: got %q, want empty", text)
	}
}

// TestCommentTextConcatenation checks that Text joins several entries of one
// position with blank lines and skips empty entries.
func TestCommentTextConcatenation(t *testing.T) {
	ctx := context.Background()
	s := openMem(t)
	pos := savePos(t, s, domain.CheckerAction)

	if _, err := s.Comments().Add(ctx, "", pos, "alpha"); err != nil {
		t.Fatalf("Add alpha: %v", err)
	}
	if _, err := s.Comments().Add(ctx, "", pos, ""); err != nil {
		t.Fatalf("Add empty: %v", err)
	}
	if _, err := s.Comments().Add(ctx, "", pos, "beta"); err != nil {
		t.Fatalf("Add beta: %v", err)
	}

	text, err := s.Comments().Text(ctx, "", pos)
	if err != nil {
		t.Fatalf("Text: %v", err)
	}
	if text != "alpha\n\nbeta" {
		t.Errorf("Text: got %q, want %q", text, "alpha\n\nbeta")
	}
}

// TestCommentByPositionAndDeleteForPosition covers ByPosition (most recent
// first) and the bulk DeleteForPosition.
func TestCommentByPositionAndDeleteForPosition(t *testing.T) {
	ctx := context.Background()
	s := openMem(t)
	pos := savePos(t, s, domain.CheckerAction)

	first, _ := s.Comments().Add(ctx, "", pos, "first")
	second, _ := s.Comments().Add(ctx, "", pos, "second")

	var ids []int64
	for e, err := range s.Comments().ByPosition(ctx, "", pos) {
		if err != nil {
			t.Fatalf("ByPosition: %v", err)
		}
		ids = append(ids, e.ID)
	}
	if len(ids) != 2 || ids[0] != second || ids[1] != first {
		t.Errorf("ByPosition order: got %v, want [%d %d]", ids, second, first)
	}

	if err := s.Comments().DeleteForPosition(ctx, "", pos); err != nil {
		t.Fatalf("DeleteForPosition: %v", err)
	}
	n := 0
	for _, err := range s.Comments().ByPosition(ctx, "", pos) {
		if err != nil {
			t.Fatalf("ByPosition: %v", err)
		}
		n++
	}
	if n != 0 {
		t.Errorf("comments after DeleteForPosition: got %d, want 0", n)
	}
}

// TestCommentListAllAndSearch covers ListAll across positions and the LIKE
// search.
func TestCommentListAllAndSearch(t *testing.T) {
	ctx := context.Background()
	s := openMem(t)
	posA := savePos(t, s, domain.CheckerAction)
	posB := savePos(t, s, domain.CubeAction)

	if _, err := s.Comments().Add(ctx, "", posA, "blunder on the bar point"); err != nil {
		t.Fatalf("Add A: %v", err)
	}
	if _, err := s.Comments().Add(ctx, "", posB, "clean take"); err != nil {
		t.Fatalf("Add B: %v", err)
	}

	n := 0
	for _, err := range s.Comments().ListAll(ctx, "") {
		if err != nil {
			t.Fatalf("ListAll: %v", err)
		}
		n++
	}
	if n != 2 {
		t.Errorf("ListAll count: got %d, want 2", n)
	}

	var hits []int64
	for e, err := range s.Comments().Search(ctx, "", "blunder") {
		if err != nil {
			t.Fatalf("Search: %v", err)
		}
		hits = append(hits, e.PositionID)
	}
	if len(hits) != 1 || hits[0] != posA {
		t.Errorf("Search: got %v, want [%d]", hits, posA)
	}
}
