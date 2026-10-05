// Contract cases for the Pile: the collection the "come back to this" gesture
// aims at, designated by a library setting rather than by its name.
package storagetest

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func testPileLifecycle(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	cp := checkerPos()
	id, err := s.Positions().Save(ctx, "", &cp)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}
	stored := cp
	stored.ID = id

	// Reading creates nothing.
	if on, err := storage.PositionOnPile(ctx, s, "", &stored); err != nil || on {
		t.Fatalf("PositionOnPile on a library with no Pile: %v, %v", on, err)
	}
	if n, _ := collectionCount(ctx, s); n != 0 {
		t.Fatalf("reading the Pile created %d collection(s)", n)
	}

	// First use creates it, and the position goes on.
	r, err := storage.TogglePile(ctx, s, "", &stored)
	if err != nil || !r.OnPile || r.Brought || r.PositionID != id {
		t.Fatalf("first toggle: %+v, %v", r, err)
	}
	if on, _ := storage.PositionOnPile(ctx, s, "", &stored); !on {
		t.Fatalf("position not on the Pile after the first toggle")
	}

	// Renaming does not change which collection is the Pile.
	if err := s.Collections().Update(ctx, "", r.CollectionID, "À revoir", ""); err != nil {
		t.Fatalf("rename: %v", err)
	}
	// The same gesture takes it off, on the same collection.
	r2, err := storage.TogglePile(ctx, s, "", &stored)
	if err != nil || r2.OnPile || r2.CollectionID != r.CollectionID {
		t.Fatalf("second toggle: %+v, %v", r2, err)
	}

	// Deleted, the Pile is created again.
	if err := s.Collections().Delete(ctx, "", r.CollectionID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if on, _ := storage.PositionOnPile(ctx, s, "", &stored); on {
		t.Fatalf("position still on a deleted Pile")
	}
	r3, err := storage.TogglePile(ctx, s, "", &stored)
	if err != nil || !r3.OnPile || r3.CollectionID == r.CollectionID {
		t.Fatalf("toggle after delete: %+v, %v", r3, err)
	}
}

func testPileBringsDraft(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	draft := checkerPos()
	draft.ID = 0

	r, err := storage.TogglePile(ctx, s, "", &draft)
	if err != nil || !r.OnPile || !r.Brought || r.PositionID == 0 {
		t.Fatalf("toggle of a draft: %+v, %v", r, err)
	}
	// The marker follows the draft by its hash, though the draft keeps ID 0.
	if on, err := storage.PositionOnPile(ctx, s, "", &draft); err != nil || !on {
		t.Fatalf("draft not seen on the Pile after the toggle: %v, %v", on, err)
	}
	got, err := s.Positions().Load(ctx, "", r.PositionID)
	if err != nil || !got.IndividuallyImported {
		t.Fatalf("draft not written as brought in on its own: %+v, %v", got, err)
	}
	// Asking again changes nothing: the state is read, not toggled.
	if on, _ := storage.PositionOnPile(ctx, s, "", &draft); !on {
		t.Fatalf("a second read turned the marker off")
	}
	// Taking it off leaves the position in the library.
	r2, err := storage.TogglePile(ctx, s, "", &draft)
	if err != nil || r2.OnPile || r2.Brought || r2.PositionID != r.PositionID {
		t.Fatalf("second toggle of the same draft: %+v, %v", r2, err)
	}
	if _, err := s.Positions().Load(ctx, "", r.PositionID); err != nil {
		t.Fatalf("position left the library: %v", err)
	}
	if on, _ := storage.PositionOnPile(ctx, s, "", &draft); on {
		t.Fatalf("draft still on the Pile after being taken off")
	}
}

func collectionCount(ctx context.Context, s storage.Storage) (int, error) {
	n := 0
	for _, err := range s.Collections().List(ctx, "") {
		if err != nil {
			return 0, err
		}
		n++
	}
	return n, nil
}
