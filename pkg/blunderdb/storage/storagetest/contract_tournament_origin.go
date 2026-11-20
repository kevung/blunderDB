// Contract case for the origin of a tournament: free text, unknown until set,
// kept by the header edit, and read the same by Get and List.
// The table that runs it lives in contract.go.
package storagetest

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func testTournamentOrigin(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ts := s.Tournaments()

	played, err := ts.Create(ctx, "", "Online Cup", "2026-11-01", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	imported, err := ts.Create(ctx, "", "Open de Lyon", "2026-10-01", "Lyon")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got, err := ts.Get(ctx, "", played); err != nil || got.Origin != "" {
		t.Fatalf("a new tournament's origin = %q (%v), want unknown", got.Origin, err)
	}

	if err := ts.UpdateOrigin(ctx, "", played, "  platform \n"); err != nil {
		t.Fatalf("UpdateOrigin: %v", err)
	}
	if err := ts.UpdateOrigin(ctx, "", imported, "imported"); err != nil {
		t.Fatalf("UpdateOrigin: %v", err)
	}
	// The header edit carries no origin and must not erase it.
	if err := ts.Update(ctx, "", played, "Online Cup 2026", "2026-11-01", ""); err != nil {
		t.Fatalf("Update: %v", err)
	}

	want := map[int64]string{played: "platform", imported: "imported"}
	for id, origin := range want {
		got, err := ts.Get(ctx, "", id)
		if err != nil {
			t.Fatalf("Get %d: %v", id, err)
		}
		if got.Origin != origin {
			t.Errorf("Get %d: origin %q, want %q", id, got.Origin, origin)
		}
	}
	listed := 0
	for tr, err := range ts.List(ctx, "", storage.ListOpts{}) {
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if origin, ok := want[tr.ID]; ok {
			listed++
			if tr.Origin != origin {
				t.Errorf("List %d: origin %q, want %q", tr.ID, tr.Origin, origin)
			}
		}
	}
	if listed != len(want) {
		t.Errorf("List held %d of the %d tournaments", listed, len(want))
	}

	if err := ts.UpdateOrigin(ctx, "", played, ""); err != nil {
		t.Fatalf("UpdateOrigin to unknown: %v", err)
	}
	if got, err := ts.Get(ctx, "", played); err != nil || got.Origin != "" {
		t.Errorf("origin after clearing = %q (%v), want unknown", got.Origin, err)
	}
}
