// Contract case: the change journal says, per object, the last thing that
// happened to it.
package storagetest

import (
	"context"
	"testing"
	"time"

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

// A match sent to the trash is restorable, so the journal must say trash for
// the match, for the positions only it held and for their notes — a delete
// would make a peer remove them for good.
func testChangeJournalTrashMatch(t *testing.T, s storage.Storage) {
	prov, ok := s.(storage.ChangeJournalProvider)
	uprov, uok := s.(storage.ObjectUUIDProvider)
	if !ok || !uok {
		t.Skip("backend keeps no change journal")
	}
	ctx := context.Background()
	uuidOf := func(family string, id int64) string {
		t.Helper()
		u, err := uprov.ObjectUUIDs().ObjectUUID(ctx, "", family, id)
		if err != nil || u == "" {
			t.Fatalf("uuid of %s %d = %q, %v", family, id, u, err)
		}
		return u
	}
	opOf := func(u string) string {
		t.Helper()
		c, err := prov.ChangeJournal().Entry(ctx, "", u)
		if err != nil {
			t.Fatalf("journal entry of %s: %v", u, err)
		}
		return c.Op
	}

	ms := s.Matches()
	mid, err := ms.Save(ctx, "", &domain.Match{Player1Name: "Jo", Player2Name: "Flo", MatchLength: 5,
		MatchDate: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), MatchHash: "jt-h", CanonicalHash: "jt-c", GameCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	gid, err := ms.CreateGame(ctx, "", &domain.Game{MatchID: mid, GameNumber: 1, MoveCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	p := provenancePos(63)
	pid, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatal(err)
	}
	cid, err := s.Comments().AddFrom(ctx, "", pid, "only this match holds me", domain.CommentOrigin(""))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ms.CreateMove(ctx, "", &domain.Move{GameID: gid, MoveNumber: 1, MoveType: "checker", PositionID: pid,
		Player: 1, Dice: [2]int32{3, 1}, CheckerMove: "8/5 6/5"}); err != nil {
		t.Fatal(err)
	}
	mu, pu, cu := uuidOf("match", mid), uuidOf("position", pid), uuidOf("comment", cid)

	tid, err := trash.Match(ctx, s, "", mid)
	if err != nil {
		t.Fatal(err)
	}
	for what, u := range map[string]string{"match": mu, "orphan position": pu, "its note": cu} {
		if got := opOf(u); got != storage.ChangeTrash {
			t.Errorf("%s: journal says %s after a trash, want trash", what, got)
		}
	}
	r, err := trash.Restore(ctx, s, "", tid)
	if err != nil {
		t.Fatal(err)
	}
	if got := uuidOf("match", r.ID); got != mu {
		t.Errorf("restored match uuid %s, want %s", got, mu)
	}
	// Later statements of the restore (video, tournament slot) may leave an
	// update: an upsert for a reader, which is all a restore must be.
	if got := opOf(mu); got != storage.ChangeCreate && got != storage.ChangeUpdate {
		t.Errorf("restored match: journal says %s, want a create or an update", got)
	}
}
