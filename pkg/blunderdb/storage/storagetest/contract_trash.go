// Contract cases for the trash: a snapshot is written, read back, discarded,
// and purged by age — and nothing else in the schema notices it exists.
// The table that runs them lives in contract.go.
package storagetest

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testTrashPutListDiscard pins the life cycle: Put returns an id, List reads
// it back most-recent-first with its payload intact, Load finds one, Discard
// removes it, and both reads report ErrNotFound for an id that is not there.
func testTrashPutListDiscard(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	tr := s.Trash()

	if n, err := tr.Count(ctx, ""); err != nil || n != 0 {
		t.Fatalf("empty trash: count %d (%v), want 0", n, err)
	}

	payload, _ := json.Marshal(domain.TrashCollectionPayload{Name: "Blunders", PositionIDs: []int64{7, 9}})
	first, err := tr.Put(ctx, "", domain.TrashCollection, "Blunders", payload)
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	second, err := tr.Put(ctx, "", domain.TrashComment, "a note", []byte(`{"comment":{"id":3}}`))
	if err != nil {
		t.Fatalf("Put (second): %v", err)
	}

	all, err := tr.List(ctx, "", "", storage.ListOpts{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("List: %d entries, want 2", len(all))
	}
	if all[0].ID != second {
		t.Errorf("List is not most-recent-first: got %d then %d", all[0].ID, all[1].ID)
	}
	if all[0].DeletedAt == "" {
		t.Error("a trash entry carries no deletion stamp")
	}

	// The payload must come back byte-for-byte: it is the only thing that can
	// put the deleted object back.
	one, err := tr.Load(ctx, "", first)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	var back domain.TrashCollectionPayload
	if err := json.Unmarshal(one.Payload, &back); err != nil {
		t.Fatalf("the payload did not survive the round trip: %v", err)
	}
	if back.Name != "Blunders" || len(back.PositionIDs) != 2 || back.PositionIDs[1] != 9 {
		t.Errorf("payload came back as %+v", back)
	}
	if one.Label != "Blunders" || one.Kind != domain.TrashCollection {
		t.Errorf("entry came back as kind %q label %q", one.Kind, one.Label)
	}

	// Narrowing by kind.
	comments, err := tr.List(ctx, "", domain.TrashComment, storage.ListOpts{})
	if err != nil {
		t.Fatalf("List (by kind): %v", err)
	}
	if len(comments) != 1 || comments[0].ID != second {
		t.Errorf("List by kind returned %d entries", len(comments))
	}

	if err := tr.Discard(ctx, "", first); err != nil {
		t.Fatalf("Discard: %v", err)
	}
	if _, err := tr.Load(ctx, "", first); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("loading a discarded entry: got %v, want ErrNotFound", err)
	}
	if err := tr.Discard(ctx, "", first); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("discarding twice: got %v, want ErrNotFound", err)
	}
	if n, err := tr.Count(ctx, ""); err != nil || n != 1 {
		t.Errorf("after one discard: count %d (%v), want 1", n, err)
	}
}

// testTrashPurgeByAge pins what `vacuum` runs: a purge with a positive age
// spares what was deleted today, and a purge with age 0 empties the trash.
//
// It cannot age a row by waiting, so it checks the boundary the other way
// round: everything here was written now, so a 30-day purge must drop nothing
// and a 0-day purge must drop everything. A purge that ignored its age
// argument would pass one of those two and fail the other.
func testTrashPurgeByAge(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	tr := s.Trash()

	for i := 0; i < 3; i++ {
		if _, err := tr.Put(ctx, "", domain.TrashComment, "note", []byte(`{}`)); err != nil {
			t.Fatalf("Put: %v", err)
		}
	}

	n, err := tr.Purge(ctx, "", domain.TrashRetentionDays)
	if err != nil {
		t.Fatalf("Purge (30 days): %v", err)
	}
	if n != 0 {
		t.Errorf("a 30-day purge dropped %d entries deleted today", n)
	}
	if got, _ := tr.Count(ctx, ""); got != 3 {
		t.Errorf("after the 30-day purge: %d entries, want 3", got)
	}

	n, err = tr.Purge(ctx, "", 0)
	if err != nil {
		t.Fatalf("Purge (all): %v", err)
	}
	if n != 3 {
		t.Errorf("emptying the trash dropped %d entries, want 3", n)
	}
	if got, _ := tr.Count(ctx, ""); got != 0 {
		t.Errorf("after emptying: %d entries, want 0", got)
	}
}
