package migrate

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// The Pile follows the library: the destination's setting names the copy of the
// collection, not the source's id, so a first gesture there does not create a
// second Pile.
func TestRunCarriesThePile(t *testing.T) {
	ctx := context.Background()
	open := func() storage.Storage {
		s, err := sqlite.Open(ctx, ":memory:", nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { s.Close() })
		if err := s.Migrate(ctx); err != nil {
			t.Fatal(err)
		}
		return s
	}
	src, dst := open(), open()

	// A collection before the Pile, so that the two ids differ in the copy order.
	if _, err := src.Collections().Create(ctx, "", "Openings", ""); err != nil {
		t.Fatal(err)
	}
	pos := domain.InitializePosition()
	pos.DecisionType = domain.CheckerAction
	if _, err := storage.TogglePile(ctx, src, "", &pos); err != nil {
		t.Fatal(err)
	}

	if _, err := Run(ctx, src, dst, "", Options{}); err != nil {
		t.Fatal(err)
	}

	id, err := storage.PileCollectionID(ctx, dst, "")
	if err != nil || id == 0 {
		t.Fatalf("the Pile was not carried: id %d, err %v", id, err)
	}
	c, err := dst.Collections().Get(ctx, "", id)
	if err != nil || c.PositionCount != 1 {
		t.Fatalf("the designated collection is not the Pile with its position: %+v, %v", c, err)
	}
	if on, err := storage.PositionOnPile(ctx, dst, "", &pos); err != nil || !on {
		t.Fatalf("the position is not on the migrated Pile: %v, %v", on, err)
	}
	// The first gesture on the destination takes it off the same Pile.
	r, err := storage.TogglePile(ctx, dst, "", &pos)
	if err != nil || r.OnPile || r.CollectionID != id {
		t.Fatalf("toggle on the destination: %+v, %v", r, err)
	}
	n := 0
	for _, err := range dst.Collections().List(ctx, "") {
		if err != nil {
			t.Fatal(err)
		}
		n++
	}
	if n != 2 {
		t.Fatalf("destination holds %d collections, want 2", n)
	}
}
