// Scenarios of the changeset package beyond the plain exchange: the peer's
// trash, the clock, the tie, the content of a collection, the rows that
// predate the journal.
package storagetest

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/changeset"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// csPair is two libraries (or two tenants of one) exchanging changes.
type csPair struct {
	t          *testing.T
	a, b       storage.Storage
	ca, cb     context.Context
	sa, sb     string
	cursorFrom storage.Cursor
}

func (p *csPair) uuid(s storage.Storage, ctx context.Context, scope string, id int64) string {
	p.t.Helper()
	u, err := s.(storage.ObjectUUIDProvider).ObjectUUIDs().ObjectUUID(ctx, scope, "collection", id)
	if err != nil || u == "" {
		p.t.Fatalf("uuid of collection %d: %q, %v", id, u, err)
	}
	return u
}

func (p *csPair) idOf(s storage.Storage, ctx context.Context, scope, u string) (int64, bool) {
	id, err := s.(storage.ObjectUUIDProvider).ObjectUUIDs().ObjectID(ctx, scope, "collection", u)
	return id, err == nil
}

// send exports a's changes after the pair's cursor and imports them into b.
func (p *csPair) send() changeset.Report {
	p.t.Helper()
	set, err := changeset.Export(p.ca, p.a, p.sa, p.cursorFrom, 1000)
	if err != nil {
		p.t.Fatalf("export: %v", err)
	}
	p.cursorFrom = set.Cursor
	rep, err := changeset.Import(p.cb, p.b, p.sb, set)
	if err != nil {
		p.t.Fatalf("import: %v", err)
	}
	return rep
}

// trashEntriesOf counts the collection trash entries of library b naming u.
func (p *csPair) trashEntriesOf(s storage.Storage, ctx context.Context, scope, u string) int {
	p.t.Helper()
	es, err := s.Trash().List(ctx, scope, domain.TrashCollection, storage.ListOpts{})
	if err != nil {
		p.t.Fatal(err)
	}
	n := 0
	for _, e := range es {
		var v struct {
			UUID string `json:"uuid"`
		}
		if json.Unmarshal(e.Payload, &v) == nil && v.UUID == u {
			n++
		}
	}
	return n
}

func (p *csPair) mark(s storage.Storage, ctx context.Context, scope, u, op, at string) {
	p.t.Helper()
	if err := s.(storage.ChangeJournalProvider).ChangeJournal().Mark(ctx, scope, "collection", u, op, at); err != nil {
		p.t.Fatal(err)
	}
}

func (p *csPair) entry(s storage.Storage, ctx context.Context, scope, u string) storage.Change {
	p.t.Helper()
	c, err := s.(storage.ChangeJournalProvider).ChangeJournal().Entry(ctx, scope, u)
	if err != nil {
		p.t.Fatalf("entry of %s: %v", u, err)
	}
	return c
}

func (p *csPair) name(s storage.Storage, ctx context.Context, scope string, id int64) string {
	c, err := s.Collections().Get(ctx, scope, id)
	if err != nil {
		p.t.Fatal(err)
	}
	return c.Name
}

// CheckChangeSetScenarios runs the scenarios on two libraries.
func CheckChangeSetScenarios(t *testing.T, a storage.Storage, ca context.Context, sa string, b storage.Storage, cb context.Context, sb string) {
	t.Run("RestoreClearsThePeersTrash", func(t *testing.T) { scenarioPeerTrash(&csPair{t: t, a: a, ca: ca, sa: sa, b: b, cb: cb, sb: sb}) })
	t.Run("Clock", func(t *testing.T) { scenarioClock(&csPair{t: t, a: a, ca: ca, sa: sa, b: b, cb: cb, sb: sb}) })
	t.Run("TieToTheMillisecond", func(t *testing.T) { scenarioTie(&csPair{t: t, a: a, ca: ca, sa: sa, b: b, cb: cb, sb: sb}) })
	t.Run("CollectionContent", func(t *testing.T) { scenarioContent(&csPair{t: t, a: a, ca: ca, sa: sa, b: b, cb: cb, sb: sb}) })
	t.Run("Position", func(t *testing.T) { scenarioPosition(&csPair{t: t, a: a, ca: ca, sa: sa, b: b, cb: cb, sb: sb}) })
	t.Run("MembersAfterTheirPositions", func(t *testing.T) { scenarioMembersFirst(&csPair{t: t, a: a, ca: ca, sa: sa, b: b, cb: cb, sb: sb}) })
	t.Run("Match", func(t *testing.T) { scenarioMatch(&csPair{t: t, a: a, ca: ca, sa: sa, b: b, cb: cb, sb: sb}) })
	t.Run("PublicID", func(t *testing.T) { scenarioPublicID(&csPair{t: t, a: a, ca: ca, sa: sa, b: b, cb: cb, sb: sb}) })
	t.Run("DrawingAndDerivation", func(t *testing.T) {
		scenarioDrawingAndDerivation(&csPair{t: t, a: a, ca: ca, sa: sa, b: b, cb: cb, sb: sb})
	})
	t.Run("Lesson", func(t *testing.T) { scenarioLesson(&csPair{t: t, a: a, ca: ca, sa: sa, b: b, cb: cb, sb: sb}) })
	t.Run("Comment", func(t *testing.T) { scenarioComment(&csPair{t: t, a: a, ca: ca, sa: sa, b: b, cb: cb, sb: sb}) })
	t.Run("CursorStopsAtUncarriedFamilies", func(t *testing.T) {
		scenarioUncarried(&csPair{t: t, a: a, ca: ca, sa: sa, b: b, cb: cb, sb: sb})
	})
	CheckChangeSetSync(t, a, ca, sa, b, cb, sb)
}

// A trashes a collection then restores it; b received both. b must end with
// the one collection and no trash entry that would restore a second.
func scenarioPeerTrash(p *csPair) {
	id, _ := p.a.Collections().Create(p.ca, p.sa, "Round trip", "")
	u := p.uuid(p.a, p.ca, p.sa, id)
	p.send()
	time.Sleep(5 * time.Millisecond)
	tid, err := trashCollection(p.ca, p.a, p.sa, id)
	if err != nil {
		p.t.Fatal(err)
	}
	p.send()
	if p.trashEntriesOf(p.b, p.cb, p.sb, u) != 1 {
		p.t.Fatalf("b should hold the trash entry after the trash")
	}
	time.Sleep(5 * time.Millisecond)
	if err := restoreCollection(p.ca, p.a, p.sa, tid); err != nil {
		p.t.Fatal(err)
	}
	if rep := p.send(); rep.Applied != 1 {
		p.t.Fatalf("restore: %+v", rep)
	}
	if _, ok := p.idOf(p.b, p.cb, p.sb, u); !ok {
		p.t.Fatal("b lost the restored collection")
	}
	if n := p.trashEntriesOf(p.b, p.cb, p.sb, u); n != 0 {
		p.t.Errorf("b keeps %d trash entry for a collection that is alive: restoring it would duplicate it", n)
	}
	// A hard delete elsewhere also empties the peer's trash of it.
	id2, _ := p.a.Collections().Create(p.ca, p.sa, "Short lived", "")
	u2 := p.uuid(p.a, p.ca, p.sa, id2)
	p.send()
	time.Sleep(5 * time.Millisecond)
	tid2, _ := trashCollection(p.ca, p.a, p.sa, id2)
	p.send()
	_ = tid2
	time.Sleep(5 * time.Millisecond)
	p.mark(p.a, p.ca, p.sa, u2, storage.ChangeDelete, "")
	p.send()
	if n := p.trashEntriesOf(p.b, p.cb, p.sb, u2); n != 0 {
		p.t.Errorf("b keeps %d trash entry of a collection deleted for good", n)
	}
}

func scenarioClock(p *csPair) {
	id, _ := p.a.Collections().Create(p.ca, p.sa, "Clock", "")
	u := p.uuid(p.a, p.ca, p.sa, id)
	p.send()
	bid, _ := p.idOf(p.b, p.cb, p.sb, u)
	payload := func(name string) json.RawMessage {
		raw, _ := json.Marshal(map[string]any{"name": name, "description": ""})
		return raw
	}
	recv := func(name, at string) changeset.Report {
		rep, err := changeset.Import(p.cb, p.b, p.sb, changeset.Set{Records: []changeset.Record{{
			Change: storage.Change{UUID: u, Family: "collection", Op: storage.ChangeUpdate, ChangedAt: at}, Payload: payload(name)}}})
		if err != nil {
			p.t.Fatalf("import at %s: %v", at, err)
		}
		return rep
	}
	// The local entry is at 10:00:00.999; as text "…:00Z" would beat it.
	p.mark(p.b, p.cb, p.sb, u, storage.ChangeUpdate, "2025-06-01T10:00:00.999Z")
	if rep := recv("older, no fraction", "2025-06-01T10:00:00Z"); rep.Applied != 0 || rep.Older != 1 {
		p.t.Errorf("a date 999 ms older won: %+v", rep)
	}
	if rep := recv("older, offset", "2025-06-01T12:00:00.500+02:00"); rep.Applied != 0 {
		p.t.Errorf("a date in another zone, older, won: %+v", rep)
	}
	if rep := recv("newer, offset", "2025-06-01T12:00:01+02:00"); rep.Applied != 1 {
		p.t.Errorf("a date in another zone, newer, lost: %+v", rep)
	}
	if got := p.entry(p.b, p.cb, p.sb, u).ChangedAt; got != "2025-06-01T10:00:01.000Z" {
		p.t.Errorf("the stored date was not normalised: %s", got)
	}
	if _, err := changeset.Import(p.cb, p.b, p.sb, changeset.Set{Records: []changeset.Record{{
		Change: storage.Change{UUID: u, Family: "collection", Op: storage.ChangeUpdate, ChangedAt: "yesterday"}}}}); err == nil {
		p.t.Error("an unreadable date was accepted")
	}
	// A date in the future is taken as now: a local edit made after the
	// import is newer than it.
	rep := recv("from the future", "2999-01-01T00:00:00.000Z")
	if rep.Applied != 1 || rep.Clamped != 1 {
		p.t.Fatalf("future record: %+v", rep)
	}
	at := p.entry(p.b, p.cb, p.sb, u).ChangedAt
	if at > time.Now().UTC().Add(time.Second).Format(storage.ChangedAtLayout) {
		p.t.Fatalf("the future date was stored: %s", at)
	}
	time.Sleep(5 * time.Millisecond)
	if err := p.b.Collections().Update(p.cb, p.sb, bid, "edited here", ""); err != nil {
		p.t.Fatal(err)
	}
	if after := p.entry(p.b, p.cb, p.sb, u).ChangedAt; after <= at {
		p.t.Errorf("a local edit after the import is not newer: %s <= %s", after, at)
	}
	// The peer's clock is still ahead and it sends the record again: the same
	// change, which must not beat the edit made here since. A really newer
	// change from that peer still does.
	if rep := recv("from the future", "2999-01-01T00:00:00.000Z"); rep.Applied != 0 {
		p.t.Errorf("a future record sent again beat a local edit: %+v", rep)
	}
	if c, _ := p.b.Collections().Get(p.cb, p.sb, bid); c.Name != "edited here" {
		p.t.Errorf("the local edit was overwritten: %q", c.Name)
	}
	time.Sleep(5 * time.Millisecond)
	if rep := recv("later, still ahead", "2999-01-02T00:00:00.000Z"); rep.Applied != 1 {
		p.t.Errorf("a newer future record from the same peer was ignored: %+v", rep)
	}
}

// Same millisecond on both sides, different content: both pick the same one.
func scenarioTie(p *csPair) {
	id, _ := p.a.Collections().Create(p.ca, p.sa, "Tie", "")
	u := p.uuid(p.a, p.ca, p.sa, id)
	p.send()
	bid, _ := p.idOf(p.b, p.cb, p.sb, u)
	const at = "2025-07-01T10:00:00.000Z"
	if err := p.a.Collections().Update(p.ca, p.sa, id, "Tie from a", ""); err != nil {
		p.t.Fatal(err)
	}
	p.mark(p.a, p.ca, p.sa, u, storage.ChangeUpdate, at)
	if err := p.b.Collections().Update(p.cb, p.sb, bid, "Tie from b", ""); err != nil {
		p.t.Fatal(err)
	}
	p.mark(p.b, p.cb, p.sb, u, storage.ChangeUpdate, at)
	fromA, _ := changeset.Export(p.ca, p.a, p.sa, storage.Cursor{}, 1000)
	fromB, _ := changeset.Export(p.cb, p.b, p.sb, storage.Cursor{}, 1000)
	if _, err := changeset.Import(p.cb, p.b, p.sb, fromA); err != nil {
		p.t.Fatal(err)
	}
	if _, err := changeset.Import(p.ca, p.a, p.sa, fromB); err != nil {
		p.t.Fatal(err)
	}
	if na, nb := p.name(p.a, p.ca, p.sa, id), p.name(p.b, p.cb, p.sb, bid); na != nb {
		p.t.Errorf("the libraries diverge on a tie: a=%q b=%q", na, nb)
	}
}

// Membership, living query and rank travel with the collection.
func scenarioContent(p *csPair) {
	// The same board is one position per library, under a uuid of its own:
	// b takes a's uuids, as the position family will.
	var pos [3]int64
	for i := range pos {
		mk := func() domain.Position {
			q := provenancePos(0)
			q.Board.Points[3+i].Checkers = 2
			return q
		}
		qa, qb := mk(), mk()
		pos[i], _ = p.a.Positions().Save(p.ca, p.sa, &qa)
		bid, err := p.b.Positions().Save(p.cb, p.sb, &qb)
		if err != nil {
			p.t.Fatal(err)
		}
		ua, _ := p.a.(storage.ObjectUUIDProvider).ObjectUUIDs().ObjectUUID(p.ca, p.sa, "position", pos[i])
		if err := p.b.(storage.ObjectUUIDProvider).ObjectUUIDs().SetObjectUUID(p.cb, p.sb, "position", bid, ua); err != nil {
			p.t.Fatal(err)
		}
	}
	id, _ := p.a.Collections().Create(p.ca, p.sa, "With members", "")
	if err := p.a.Collections().AddPositions(p.ca, p.sa, id, []int64{pos[2], pos[0], pos[1]}); err != nil {
		p.t.Fatal(err)
	}
	live, _ := p.a.Collections().Create(p.ca, p.sa, "Living", "")
	if err := p.a.Collections().SetFilterQuery(p.ca, p.sa, live, "cube"); err != nil {
		p.t.Fatal(err)
	}
	p.send()
	u := p.uuid(p.a, p.ca, p.sa, id)
	bid, ok := p.idOf(p.b, p.cb, p.sb, u)
	if !ok {
		p.t.Fatal("the collection did not arrive")
	}
	order := func(s storage.Storage, ctx context.Context, scope string, cid int64) []string {
		ids, err := s.Collections().PositionIDs(ctx, scope, cid, storage.ListOpts{})
		if err != nil {
			p.t.Fatal(err)
		}
		var out []string
		for _, i := range ids {
			x, _ := s.(storage.ObjectUUIDProvider).ObjectUUIDs().ObjectUUID(ctx, scope, "position", i)
			out = append(out, x)
		}
		return out
	}
	wa, wb := order(p.a, p.ca, p.sa, id), order(p.b, p.cb, p.sb, bid)
	if len(wb) != 3 || wa[0] != wb[0] || wa[1] != wb[1] || wa[2] != wb[2] {
		p.t.Errorf("membership differs: a=%v b=%v", wa, wb)
	}
	lid, ok := p.idOf(p.b, p.cb, p.sb, p.uuid(p.a, p.ca, p.sa, live))
	if !ok {
		p.t.Fatal("the living collection did not arrive")
	}
	if c, _ := p.b.Collections().Get(p.cb, p.sb, lid); c.FilterQuery != "cube" {
		p.t.Errorf("filter query = %q", c.FilterQuery)
	}
	// A removal travels too.
	time.Sleep(5 * time.Millisecond)
	if err := p.a.Collections().RemovePosition(p.ca, p.sa, id, pos[0]); err != nil {
		p.t.Fatal(err)
	}
	p.send()
	if got := order(p.b, p.cb, p.sb, bid); len(got) != 2 {
		p.t.Errorf("after a removal b holds %v", got)
	}
}

// The cursor stays at the last entry of a carried family, so a family carried
// later is not skipped.
func scenarioUncarried(p *csPair) {
	id, _ := p.a.Collections().Create(p.ca, p.sa, "Before a draft", "")
	u := p.uuid(p.a, p.ca, p.sa, id)
	if _, err := p.a.Transcriptions().Save(p.ca, p.sa, &storage.Transcription{FormatVersion: "1", Label: "Not carried yet", Document: "{}"}); err != nil {
		p.t.Fatal(err)
	}
	set, err := changeset.Export(p.ca, p.a, p.sa, storage.Cursor{}, 1000)
	if err != nil {
		p.t.Fatal(err)
	}
	last := p.entry(p.a, p.ca, p.sa, u).Seq
	if set.Cursor.Seq != last {
		p.t.Errorf("cursor %d passed the carried entries (%d) over an uncarried family", set.Cursor.Seq, last)
	}
	for _, r := range set.Records {
		if r.Family == "transcription" {
			p.t.Errorf("a record of family %s", r.Family)
		}
	}
}

// CheckChangeSetPreJournal exchanges rows that predate the journal: a holds
// collections with no entry (forget is called to take them away), b holds
// one of them, edited since.
func CheckChangeSetPreJournal(t *testing.T, a storage.Storage, ca context.Context, sa string, b storage.Storage, cb context.Context, sb string, forget func()) {
	p := &csPair{t: t, a: a, ca: ca, sa: sa, b: b, cb: cb, sb: sb}
	one, _ := a.Collections().Create(ca, sa, "Old one", "")
	two, _ := a.Collections().Create(ca, sa, "Old two", "")
	u1, u2 := p.uuid(a, ca, sa, one), p.uuid(a, ca, sa, two)
	if rep := p.send(); rep.Applied < 2 {
		t.Fatalf("setup: %+v", rep)
	}
	// b already holds "Old one" and edits it; a loses its journal.
	bid, _ := p.idOf(b, cb, sb, u1)
	time.Sleep(5 * time.Millisecond)
	if err := b.Collections().Update(cb, sb, bid, "Old one, edited in b", ""); err != nil {
		t.Fatal(err)
	}
	b2, _ := p.idOf(b, cb, sb, u2)
	if err := b.Collections().Delete(cb, sb, b2); err != nil {
		t.Fatal(err)
	}
	forget()
	if _, err := a.(storage.ChangeJournalProvider).ChangeJournal().Entry(ca, sa, u1); err == nil {
		t.Fatal("the journal still has an entry: the scenario tests nothing")
	}
	p.cursorFrom = storage.Cursor{}
	rep := p.send()
	if got := p.name(b, cb, sb, bid); got != "Old one, edited in b" {
		t.Errorf("an object that predates the journal overwrote an edit: %q", got)
	}
	if _, ok := p.idOf(b, cb, sb, u2); ok {
		t.Error("an object that predates the journal resurrected a delete")
	}
	_ = rep
	// A library that holds none of them receives them all.
	c := &csPair{t: t, a: a, ca: ca, sa: sa, b: b, cb: cb, sb: sb}
	three, _ := a.Collections().Create(ca, sa, "Old three", "")
	_ = three
	forget()
	if set, err := changeset.Export(ca, a, sa, storage.Cursor{}, 1000); err != nil || len(set.Records) < 3 {
		t.Errorf("export from 0 carries %d records (%v), want every collection", len(set.Records), err)
	}
	_ = c

	// From 0 the export pages like any other: rows that predate the journal
	// come a page at a time, each once, and a page holds at most the limit.
	forget()
	seen, pages, cur := map[string]int{}, 0, storage.Cursor{}
	for pages < 50 {
		set, err := changeset.Export(ca, a, sa, cur, 2)
		if err != nil {
			t.Fatal(err)
		}
		pages++
		if len(set.Records) > 2 {
			t.Fatalf("page %d holds %d records, limit 2", pages, len(set.Records))
		}
		for _, r := range set.Records {
			seen[r.UUID]++
		}
		cur = set.Cursor
		if !set.More {
			break
		}
	}
	if pages < 2 || len(seen) < 3 {
		t.Errorf("%d pages, %d objects: the export did not page", pages, len(seen))
	}
	for u, n := range seen {
		if n != 1 {
			t.Errorf("%s came %d times", u, n)
		}
	}
}
