// Contract case: what the trash gives back is the same object, identifier
// included.
package storagetest

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/trash"
)

func testTrashKeepsUUIDs(t *testing.T, s storage.Storage) {
	prov, ok := s.(storage.ObjectUUIDProvider)
	if !ok {
		t.Skip("backend keeps no object uuid")
	}
	ctx := context.Background()
	uu := prov.ObjectUUIDs()
	uuidOf := func(family string, id int64) string {
		t.Helper()
		u, err := uu.ObjectUUID(ctx, "", family, id)
		if err != nil || u == "" {
			t.Fatalf("uuid of %s %d = %q, %v", family, id, u, err)
		}
		return u
	}

	p := provenancePos(61)
	pid, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	cid, err := s.Comments().AddFrom(ctx, "", pid, "kept note", domain.CommentOrigin(""))
	if err != nil {
		t.Fatalf("AddFrom: %v", err)
	}
	colID, err := s.Collections().Create(ctx, "", "Kept", "")
	if err != nil {
		t.Fatalf("Create collection: %v", err)
	}
	wantPos, wantCom, wantCol := uuidOf("position", pid), uuidOf("comment", cid), uuidOf("collection", colID)

	// A single note first, then the collection, then the position and the
	// notes that went with it.
	tid, err := trash.CommentEntry(ctx, s, "", cid)
	if err != nil {
		t.Fatalf("trash comment: %v", err)
	}
	r, err := trash.Restore(ctx, s, "", tid)
	if err != nil {
		t.Fatalf("restore comment: %v", err)
	}
	if got := uuidOf("comment", r.ID); got != wantCom {
		t.Errorf("a restored note has uuid %s, want %s", got, wantCom)
	}

	tid, err = trash.Collection(ctx, s, "", colID)
	if err != nil {
		t.Fatalf("trash collection: %v", err)
	}
	r, err = trash.Restore(ctx, s, "", tid)
	if err != nil {
		t.Fatalf("restore collection: %v", err)
	}
	if got := uuidOf("collection", r.ID); got != wantCol {
		t.Errorf("a restored collection has uuid %s, want %s", got, wantCol)
	}

	noteIDs := func(position int64) []int64 {
		var ids []int64
		for c, err := range s.Comments().ByPosition(ctx, "", position) {
			if err != nil {
				t.Fatal(err)
			}
			ids = append(ids, c.ID)
		}
		return ids
	}
	var noteUUID string
	for _, id := range noteIDs(pid) {
		noteUUID = uuidOf("comment", id)
	}
	tid, err = trash.Position(ctx, s, "", pid)
	if err != nil {
		t.Fatalf("trash position: %v", err)
	}
	r, err = trash.Restore(ctx, s, "", tid)
	if err != nil {
		t.Fatalf("restore position: %v", err)
	}
	if got := uuidOf("position", r.ID); got != wantPos {
		t.Errorf("a restored position has uuid %s, want %s", got, wantPos)
	}
	for _, id := range noteIDs(r.ID) {
		if got := uuidOf("comment", id); got != noteUUID {
			t.Errorf("a note restored with its position has uuid %s, want %s", got, noteUUID)
		}
	}
}
