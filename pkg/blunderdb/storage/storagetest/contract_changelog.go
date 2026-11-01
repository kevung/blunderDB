// Contract case: the change journal says, per object, the last thing that
// happened to it.
package storagetest

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/trash"
)

func testChangeJournal(t *testing.T, s storage.Storage) {
	prov, ok := s.(storage.ChangeJournalProvider)
	uprov, uok := s.(storage.ObjectUUIDProvider)
	if !ok || !uok {
		t.Skip("backend keeps no change journal")
	}
	ctx := context.Background()
	j := prov.ChangeJournal()

	// last is the journal entry of one object, and the cursor after it.
	last := func(u string) (storage.Change, bool) {
		t.Helper()
		cs, err := j.Changes(ctx, "", 0, 100000)
		if err != nil {
			t.Fatalf("Changes: %v", err)
		}
		for _, c := range cs {
			if c.UUID == u {
				return c, true
			}
		}
		return storage.Change{}, false
	}
	wantOp := func(what, u, op string) storage.Change {
		t.Helper()
		c, ok := last(u)
		if !ok || c.Op != op {
			t.Fatalf("%s: journal entry of %s = %+v (found %v), want op %s", what, u, c, ok, op)
		}
		return c
	}
	uuidOf := func(family string, id int64) string {
		t.Helper()
		u, err := uprov.ObjectUUIDs().ObjectUUID(ctx, "", family, id)
		if err != nil || u == "" {
			t.Fatalf("uuid of %s %d = %q, %v", family, id, u, err)
		}
		return u
	}

	colID, err := s.Collections().Create(ctx, "", "Journaled", "")
	if err != nil {
		t.Fatal(err)
	}
	u := uuidOf("collection", colID)
	created := wantOp("create", u, storage.ChangeCreate)
	if created.Family != "collection" || created.ChangedAt == "" || created.Seq <= 0 {
		t.Errorf("created entry malformed: %+v", created)
	}

	if err := s.Collections().Update(ctx, "", colID, "Journaled again", ""); err != nil {
		t.Fatal(err)
	}
	updated := wantOp("update", u, storage.ChangeUpdate)
	if updated.Seq <= created.Seq {
		t.Errorf("an update must move the object past its creation: %d <= %d", updated.Seq, created.Seq)
	}
	if after, _ := j.Changes(ctx, "", updated.Seq, 10); len(after) != 0 {
		t.Errorf("nothing follows the last change, got %+v", after)
	}

	// Through the trash: a trash, then back as a create with the same uuid.
	tid, err := trash.Collection(ctx, s, "", colID)
	if err != nil {
		t.Fatal(err)
	}
	wantOp("trash", u, storage.ChangeTrash)
	r, err := trash.Restore(ctx, s, "", tid)
	if err != nil {
		t.Fatal(err)
	}
	if got := uuidOf("collection", r.ID); got != u {
		t.Fatalf("restored uuid %s, want %s", got, u)
	}
	wantOp("restore", u, storage.ChangeCreate)
	cs, _ := j.Changes(ctx, "", 0, 100000)
	seen := map[string]int{}
	for _, c := range cs {
		seen[c.UUID]++
	}
	for k, n := range seen {
		if n != 1 {
			t.Errorf("uuid %s appears %d times: the journal keeps one entry per object", k, n)
		}
	}

	// A hard delete is a delete.
	if err := s.Collections().Delete(ctx, "", r.ID); err != nil {
		t.Fatal(err)
	}
	wantOp("delete", u, storage.ChangeDelete)

	// A position with a note, both gone through the trash.
	p := provenancePos(62)
	pid, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatal(err)
	}
	cid, err := s.Comments().AddFrom(ctx, "", pid, "journaled note", domain.CommentOrigin(""))
	if err != nil {
		t.Fatal(err)
	}
	pu, cu := uuidOf("position", pid), uuidOf("comment", cid)
	wantOp("position create", pu, storage.ChangeCreate)
	wantOp("note create", cu, storage.ChangeCreate)
	if _, err := trash.Position(ctx, s, "", pid); err != nil {
		t.Fatal(err)
	}
	wantOp("position trash", pu, storage.ChangeTrash)
	wantOp("note trash", cu, storage.ChangeTrash)
}
