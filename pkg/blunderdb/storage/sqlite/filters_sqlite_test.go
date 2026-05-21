package sqlite_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// TestFilterCRUD covers Save, List, Update and Delete, plus the ErrConflict /
// ErrNotFound error paths.
func TestFilterCRUD(t *testing.T) {
	ctx := context.Background()
	s := openMem(t)

	id, err := s.Filters().Save(ctx, "", "blunders", "e>0.1")
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if id == 0 {
		t.Fatal("Save returned id 0")
	}

	// A duplicate name is rejected with ErrConflict.
	if _, err := s.Filters().Save(ctx, "", "blunders", "e>0.2"); !errors.Is(err, storage.ErrConflict) {
		t.Errorf("Save duplicate: got %v, want ErrConflict", err)
	}

	if _, err := s.Filters().Save(ctx, "", "cube spots", "decision cube"); err != nil {
		t.Fatalf("Save second: %v", err)
	}

	if err := s.Filters().Update(ctx, "", id, "big blunders", "e>0.3"); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if err := s.Filters().Update(ctx, "", 999, "x", "y"); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Update missing: got %v, want ErrNotFound", err)
	}

	var got []storage.Filter
	for f, err := range s.Filters().List(ctx, "") {
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		got = append(got, *f)
	}
	if len(got) != 2 {
		t.Fatalf("List count: got %d, want 2", len(got))
	}
	if got[0].ID != id || got[0].Name != "big blunders" || got[0].Command != "e>0.3" {
		t.Errorf("List[0]: %+v", got[0])
	}

	if err := s.Filters().Delete(ctx, "", id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := s.Filters().Delete(ctx, "", id); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Delete missing: got %v, want ErrNotFound", err)
	}
}

// TestFilterEditPosition covers SaveEditPosition and LoadEditPosition.
func TestFilterEditPosition(t *testing.T) {
	ctx := context.Background()
	s := openMem(t)

	// An unknown filter has no edit position and reports no error.
	if pos, err := s.Filters().LoadEditPosition(ctx, "", "missing"); err != nil || pos != "" {
		t.Errorf("LoadEditPosition unknown: got (%q, %v), want (\"\", nil)", pos, err)
	}
	// Storing an edit position for an unknown filter is ErrNotFound.
	if err := s.Filters().SaveEditPosition(ctx, "", "missing", "{}"); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("SaveEditPosition unknown: got %v, want ErrNotFound", err)
	}

	if _, err := s.Filters().Save(ctx, "", "f1", "cmd"); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := s.Filters().SaveEditPosition(ctx, "", "f1", `{"board":1}`); err != nil {
		t.Fatalf("SaveEditPosition: %v", err)
	}
	pos, err := s.Filters().LoadEditPosition(ctx, "", "f1")
	if err != nil {
		t.Fatalf("LoadEditPosition: %v", err)
	}
	if pos != `{"board":1}` {
		t.Errorf("LoadEditPosition: got %q", pos)
	}
}
