// Shared scenario of the changeset package: two libraries exchanging the
// change journal, the last write winning.
package storagetest

import (
	"context"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/changeset"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// CheckChangeSet exchanges changes between library a (scope sa) and library b
// (scope sb). They may be the same backend under two tenants.
func CheckChangeSet(t *testing.T, a storage.Storage, ca context.Context, sa string, b storage.Storage, cb context.Context, sb string) {
	ctxOf := func(scope string) context.Context {
		if scope == sa {
			return ca
		}
		return cb
	}
	tick := func() { time.Sleep(5 * time.Millisecond) }
	var cursor storage.Cursor
	push := func(what string) changeset.Report {
		t.Helper()
		set, err := changeset.Export(ca, a, sa, cursor, 1000)
		if err != nil {
			t.Fatalf("%s: export: %v", what, err)
		}
		cursor = set.Cursor
		rep, err := changeset.Import(cb, b, sb, set)
		if err != nil {
			t.Fatalf("%s: import: %v", what, err)
		}
		return rep
	}
	idOf := func(s storage.Storage, scope, u string) (int64, error) {
		return s.(storage.ObjectUUIDProvider).ObjectUUIDs().ObjectID(ctxOf(scope), scope, "collection", u)
	}
	entry := func(s storage.Storage, scope, u string) storage.Change {
		t.Helper()
		c, err := s.(storage.ChangeJournalProvider).ChangeJournal().Entry(ctxOf(scope), scope, u)
		if err != nil {
			t.Fatalf("journal entry of %s: %v", u, err)
		}
		return c
	}

	// A create travels with its uuid, its content and its time.
	id, err := a.Collections().Create(ca, sa, "Openings", "first")
	if err != nil {
		t.Fatal(err)
	}
	u, _ := a.(storage.ObjectUUIDProvider).ObjectUUIDs().ObjectUUID(ca, sa, "collection", id)
	if rep := push("create"); rep.Applied != 1 {
		t.Fatalf("create: %+v", rep)
	}
	bid, err := idOf(b, sb, u)
	if err != nil {
		t.Fatalf("the collection did not arrive under its uuid: %v", err)
	}
	if c, _ := b.Collections().Get(cb, sb, bid); c.Name != "Openings" || c.Description != "first" {
		t.Errorf("content = %+v", c)
	}
	if ea, eb := entry(a, sa, u), entry(b, sb, u); ea.ChangedAt != eb.ChangedAt || eb.Op != storage.ChangeCreate {
		t.Errorf("journals disagree after the import: %+v vs %+v", ea, eb)
	}

	// Importing again, or importing what came from there, changes nothing.
	set, _ := changeset.Export(ca, a, sa, storage.Cursor{}, 1000)
	if rep, _ := changeset.Import(cb, b, sb, set); rep.Applied != 0 || rep.Older == 0 {
		t.Errorf("a replay applied something: %+v", rep)
	}

	// The later write wins: a renames, nothing newer in b.
	tick()
	if err := a.Collections().Update(ca, sa, id, "Openings 2", "second"); err != nil {
		t.Fatal(err)
	}
	if rep := push("update"); rep.Applied != 1 {
		t.Fatalf("update: %+v", rep)
	}
	if c, _ := b.Collections().Get(cb, sb, bid); c.Name != "Openings 2" {
		t.Errorf("update did not travel: %+v", c)
	}

	// A conflict: b writes after a did, so b stays.
	tick()
	if err := a.Collections().Update(ca, sa, id, "From a", ""); err != nil {
		t.Fatal(err)
	}
	tick()
	if err := b.Collections().Update(cb, sb, bid, "From b", ""); err != nil {
		t.Fatal(err)
	}
	if rep := push("conflict, b newer"); rep.Applied != 0 || rep.Older != 1 {
		t.Fatalf("b was newer: %+v", rep)
	}
	if c, _ := b.Collections().Get(cb, sb, bid); c.Name != "From b" {
		t.Errorf("the newer local write was lost: %+v", c)
	}
	// ... and the other way round: a writes after b did.
	tick()
	if err := a.Collections().Update(ca, sa, id, "From a, later", ""); err != nil {
		t.Fatal(err)
	}
	if rep := push("conflict, a newer"); rep.Applied != 1 {
		t.Fatalf("a was newer: %+v", rep)
	}
	if c, _ := b.Collections().Get(cb, sb, bid); c.Name != "From a, later" {
		t.Errorf("the newer remote write did not win: %+v", c)
	}

	// A trash travels as a trash, and a later restore beats it.
	tick()
	tid, err := trashCollection(ca, a, sa, id)
	if err != nil {
		t.Fatal(err)
	}
	if rep := push("trash"); rep.Applied != 1 {
		t.Fatalf("trash: %+v", rep)
	}
	if _, err := idOf(b, sb, u); err == nil {
		t.Error("the trashed collection is still in b")
	}
	if e := entry(b, sb, u); e.Op != storage.ChangeTrash {
		t.Errorf("b's journal says %s, want trash", e.Op)
	}
	tick()
	if err := restoreCollection(ca, a, sa, tid); err != nil {
		t.Fatal(err)
	}
	if rep := push("restore"); rep.Applied != 1 {
		t.Fatalf("restore: %+v", rep)
	}
	bid, err = idOf(b, sb, u)
	if err != nil {
		t.Fatalf("the restore did not beat the older trash: %v", err)
	}

	// A hard delete removes it for good, and an older create cannot bring it back.
	tick()
	aid, _ := idOf(a, sa, u)
	if err := a.Collections().Delete(ca, sa, aid); err != nil {
		t.Fatal(err)
	}
	if rep := push("delete"); rep.Applied != 1 {
		t.Fatalf("delete: %+v", rep)
	}
	if _, err := b.Collections().Get(cb, sb, bid); err == nil {
		t.Error("the deleted collection is still in b")
	}
	if rep, _ := changeset.Import(cb, b, sb, set); rep.Applied != 0 {
		t.Errorf("an older create resurrected a deleted object: %+v", rep)
	}

	// Names are not identities: a collection named like a local one arrives
	// as a second collection, not as a merge.
	if _, err := b.Collections().Create(cb, sb, "Taken", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Collections().Create(ca, sa, "Taken", ""); err != nil {
		t.Fatal(err)
	}
	if rep := push("same name"); rep.Applied != 1 || len(rep.Conflicts) != 0 {
		t.Errorf("same name: %+v", rep)
	}

	CheckChangeSetScenarios(t, a, ca, sa, b, cb, sb)
}
