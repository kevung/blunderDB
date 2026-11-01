// Scenarios of the changeset package for the families beyond the collection:
// each runs on the pair of libraries the contract hands over.
package storagetest

import (
	"context"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/changeset"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/trash"
)

func tick() { time.Sleep(5 * time.Millisecond) }

// catchUp moves the pair's cursor past everything a holds now, so a scenario
// sees its own changes only.
func (p *csPair) catchUp() {
	p.t.Helper()
	set, err := changeset.Export(p.ca, p.a, p.sa, storage.Cursor{}, 1_000_000)
	if err != nil {
		p.t.Fatal(err)
	}
	p.cursorFrom = set.Cursor
}

// replay sends everything a holds, from 0, to b.
func (p *csPair) replay() changeset.Report {
	p.t.Helper()
	set, err := changeset.Export(p.ca, p.a, p.sa, storage.Cursor{}, 1_000_000)
	if err != nil {
		p.t.Fatal(err)
	}
	rep, err := changeset.Import(p.cb, p.b, p.sb, set)
	if err != nil {
		p.t.Fatal(err)
	}
	return rep
}

func famUUID(t *testing.T, s storage.Storage, ctx context.Context, scope, family string, id int64) string {
	t.Helper()
	u, err := s.(storage.ObjectUUIDProvider).ObjectUUIDs().ObjectUUID(ctx, scope, family, id)
	if err != nil || u == "" {
		t.Fatalf("uuid of %s %d: %q, %v", family, id, u, err)
	}
	return u
}

func famID(s storage.Storage, ctx context.Context, scope, family, uuid string) (int64, bool) {
	id, err := s.(storage.ObjectUUIDProvider).ObjectUUIDs().ObjectID(ctx, scope, family, uuid)
	return id, err == nil
}

// A note travels with its text, to the position holding the same board; an
// edit, a trash and a delete follow, the later write winning.
func scenarioComment(p *csPair) {
	t := p.t
	p.catchUp()
	qa, qb := provenancePos(41), provenancePos(41)
	pa, err := p.a.Positions().Save(p.ca, p.sa, &qa)
	if err != nil {
		t.Fatal(err)
	}
	pb, err := p.b.Positions().Save(p.cb, p.sb, &qb)
	if err != nil {
		t.Fatal(err)
	}
	cid, _ := p.a.Comments().Add(p.ca, p.sa, pa, "first note")
	u := famUUID(t, p.a, p.ca, p.sa, "comment", cid)
	p.send()
	bid, ok := famID(p.b, p.cb, p.sb, "comment", u)
	if !ok {
		t.Fatal("the note did not arrive under its uuid")
	}
	if e, err := p.b.Comments().Get(p.cb, p.sb, bid); err != nil || e.PositionID != pb || e.Text != "first note" {
		t.Fatalf("note in b = %+v, %v (position %d)", e, err, pb)
	}
	if ea, eb := p.entry(p.a, p.ca, p.sa, u), p.entry(p.b, p.cb, p.sb, u); ea.ChangedAt != eb.ChangedAt {
		t.Errorf("journals disagree: %+v vs %+v", ea, eb)
	}
	if rep := p.replay(); rep.Applied != 0 {
		t.Errorf("a replay applied %+v", rep)
	}

	tick()
	if err := p.a.Comments().Update(p.ca, p.sa, cid, "second note"); err != nil {
		t.Fatal(err)
	}
	p.send()
	if e, _ := p.b.Comments().Get(p.cb, p.sb, bid); e.Text != "second note" {
		t.Errorf("the edit did not travel: %+v", e)
	}
	// b edits after a did: b stays.
	tick()
	_ = p.a.Comments().Update(p.ca, p.sa, cid, "from a")
	tick()
	_ = p.b.Comments().Update(p.cb, p.sb, bid, "from b")
	p.send()
	if e, _ := p.b.Comments().Get(p.cb, p.sb, bid); e.Text != "from b" {
		t.Errorf("the newer local edit was lost: %+v", e)
	}

	tick()
	if _, err := trash.CommentEntry(p.ca, p.a, p.sa, cid); err != nil {
		t.Fatal(err)
	}
	p.send()
	if _, ok := famID(p.b, p.cb, p.sb, "comment", u); ok {
		t.Error("the trashed note is still in b")
	}
	if e := p.entry(p.b, p.cb, p.sb, u); e.Op != storage.ChangeTrash {
		t.Errorf("b's journal says %s, want trash", e.Op)
	}

	c2, _ := p.a.Comments().Add(p.ca, p.sa, pa, "to delete")
	u2 := famUUID(t, p.a, p.ca, p.sa, "comment", c2)
	p.send()
	if _, ok := famID(p.b, p.cb, p.sb, "comment", u2); !ok {
		t.Fatal("second note did not arrive")
	}
	tick()
	_ = p.a.Comments().Delete(p.ca, p.sa, c2)
	p.send()
	if _, ok := famID(p.b, p.cb, p.sb, "comment", u2); ok {
		t.Error("the deleted note is still in b")
	}
}

func (p *csPair) swapped() *csPair {
	return &csPair{t: p.t, a: p.b, ca: p.cb, sa: p.sb, b: p.a, cb: p.ca, sb: p.sa}
}

// A position travels with its uuid. The same board under two uuids ends under
// the older one on both sides whichever way the exchange goes, and what hangs
// on the position (its notes, its place in a collection) stays on it.
func scenarioPosition(p *csPair) {
	t := p.t
	p.catchUp()
	q1, q2, q3 := provenancePos(51), provenancePos(51), provenancePos(52)
	pa, _ := p.a.Positions().Save(p.ca, p.sa, &q1)
	pb, _ := p.b.Positions().Save(p.cb, p.sb, &q2)
	solo, _ := p.a.Positions().Save(p.ca, p.sa, &q3)
	ua, ub := famUUID(t, p.a, p.ca, p.sa, "position", pa), famUUID(t, p.b, p.cb, p.sb, "position", pb)
	if ua == ub {
		t.Fatal("the scenario needs two uuids for one board")
	}
	older := min(ua, ub)
	// What hangs on b's row.
	if _, err := p.b.Comments().Add(p.cb, p.sb, pb, "kept on the row"); err != nil {
		t.Fatal(err)
	}
	col, _ := p.b.Collections().Create(p.cb, p.sb, "Holds the board", "")
	if err := p.b.Collections().AddPositions(p.cb, p.sb, col, []int64{pb}); err != nil {
		t.Fatal(err)
	}

	p.send()
	p.swapped().send()
	for _, side := range []struct {
		s  storage.Storage
		c  context.Context
		sc string
		id int64
	}{{p.a, p.ca, p.sa, pa}, {p.b, p.cb, p.sb, pb}} {
		if got := famUUID(t, side.s, side.c, side.sc, "position", side.id); got != older {
			t.Errorf("a board kept the uuid %s, want the older %s", got, older)
		}
	}
	if es := p.b.Comments(); es != nil {
		n := 0
		for e, err := range p.b.Comments().ByPosition(p.cb, p.sb, pb) {
			if err != nil {
				t.Fatal(err)
			}
			if e.Text == "kept on the row" {
				n++
			}
		}
		if n != 1 {
			t.Errorf("the note left its position: %d", n)
		}
	}
	if ids, _ := p.b.Collections().PositionIDs(p.cb, p.sb, col, storage.ListOpts{}); len(ids) != 1 || ids[0] != pb {
		t.Errorf("the collection lost its member: %v", ids)
	}

	// A board only a holds arrives under its uuid, and a replay changes nothing.
	us := famUUID(t, p.a, p.ca, p.sa, "position", solo)
	if bid, ok := famID(p.b, p.cb, p.sb, "position", us); !ok {
		t.Error("a position did not arrive under its uuid")
	} else if l, err := p.b.Positions().Load(p.cb, p.sb, bid); err != nil || l.Score != [2]int{52, 0} {
		t.Errorf("content = %+v, %v", l, err)
	}
	if rep := p.replay(); rep.Applied != 0 {
		t.Errorf("a replay applied %+v", rep)
	}

	// A delete travels.
	tick()
	if err := p.a.Positions().Delete(p.ca, p.sa, solo); err != nil {
		t.Fatal(err)
	}
	p.send()
	if _, ok := famID(p.b, p.cb, p.sb, "position", us); ok {
		t.Error("the deleted position is still in b")
	}
}

// A collection whose journal entry comes before its members' still gets them:
// positions are applied first.
func scenarioMembersFirst(p *csPair) {
	t := p.t
	p.catchUp()
	var ids []int64
	for n := 61; n < 64; n++ {
		q := provenancePos(n)
		id, _ := p.a.Positions().Save(p.ca, p.sa, &q)
		ids = append(ids, id)
	}
	col, _ := p.a.Collections().Create(p.ca, p.sa, "Members after", "")
	if err := p.a.Collections().AddPositions(p.ca, p.sa, col, []int64{ids[2], ids[0], ids[1]}); err != nil {
		t.Fatal(err)
	}
	// The positions are touched after the collection: their entries sort later.
	tick()
	for _, id := range ids {
		u := famUUID(t, p.a, p.ca, p.sa, "position", id)
		if err := p.a.(storage.ChangeJournalProvider).ChangeJournal().Mark(p.ca, p.sa, "position", u, storage.ChangeUpdate, ""); err != nil {
			t.Fatal(err)
		}
	}
	set, _ := changeset.Export(p.ca, p.a, p.sa, p.cursorFrom, 1000)
	ci, pi := -1, -1
	for i, r := range set.Records {
		if r.Family == "collection" {
			ci = i
		}
		if r.Family == "position" {
			pi = i
		}
	}
	if ci < 0 || pi < ci {
		t.Fatalf("the scenario needs the collection (%d) before a position (%d)", ci, pi)
	}
	p.send()
	cu := famUUID(t, p.a, p.ca, p.sa, "collection", col)
	bid, ok := famID(p.b, p.cb, p.sb, "collection", cu)
	if !ok {
		t.Fatal("the collection did not arrive")
	}
	got, _ := p.b.Collections().PositionIDs(p.cb, p.sb, bid, storage.ListOpts{})
	if len(got) != 3 {
		t.Errorf("members that arrived with the collection: %v, want 3", got)
	}
}

// A lesson travels with its steps, which name a collection and a board.
func scenarioLesson(p *csPair) {
	t := p.t
	p.catchUp()
	q := provenancePos(71)
	pa, _ := p.a.Positions().Save(p.ca, p.sa, &q)
	col, _ := p.a.Collections().Create(p.ca, p.sa, "For the lesson", "")
	lid, err := p.a.Lessons().Create(p.ca, p.sa, "Bear-off", "basics")
	if err != nil {
		t.Fatal(err)
	}
	for i, st := range []domain.LessonStep{
		{Title: "Look", Text: "first", PositionID: pa},
		{Title: "Practise", Text: "second", CollectionID: col},
		{Title: "Words", Text: "third"},
	} {
		if _, err := p.a.Lessons().AddStep(p.ca, p.sa, lid, st); err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
	}
	u := famUUID(t, p.a, p.ca, p.sa, "lesson", lid)
	p.send()
	bid, ok := famID(p.b, p.cb, p.sb, "lesson", u)
	if !ok {
		t.Fatal("the lesson did not arrive under its uuid")
	}
	l, err := p.b.Lessons().Get(p.cb, p.sb, bid)
	if err != nil || l.Name != "Bear-off" || len(l.Steps) != 3 {
		t.Fatalf("lesson in b = %+v, %v", l, err)
	}
	if l.Steps[0].PositionID == 0 || l.Steps[1].CollectionID == 0 || l.Steps[2].Text != "third" {
		t.Errorf("steps = %+v", l.Steps)
	}
	if rep := p.replay(); rep.Applied != 0 {
		t.Errorf("a replay applied %+v", rep)
	}

	tick()
	if err := p.a.Lessons().Update(p.ca, p.sa, lid, "Bear-off 2", "more"); err != nil {
		t.Fatal(err)
	}
	p.send()
	if l, _ := p.b.Lessons().Get(p.cb, p.sb, bid); l.Name != "Bear-off 2" {
		t.Errorf("the rename did not travel: %+v", l)
	}
	tick()
	_ = p.a.Lessons().Delete(p.ca, p.sa, lid)
	p.send()
	if _, ok := famID(p.b, p.cb, p.sb, "lesson", u); ok {
		t.Error("the deleted lesson is still in b")
	}
}

// A match travels whole: header, games, moves, boards; its note and its
// deletion follow. Where the user last looked stays where it was.
func scenarioMatch(p *csPair) {
	t := p.t
	p.catchUp()
	ms := p.a.Matches()
	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob", Event: "Open", MatchLength: 7,
		MatchDate: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), FilePath: "m.xg", GameCount: 1,
		MatchHash: "cs-hash", CanonicalHash: "cs-canon", Comment: "a good one"}
	mid, err := ms.Save(p.ca, p.sa, &m)
	if err != nil {
		t.Fatal(err)
	}
	gid, _ := ms.CreateGame(p.ca, p.sa, &domain.Game{MatchID: mid, GameNumber: 1, Winner: 1, PointsWon: 2, MoveCount: 2})
	for n := int32(1); n <= 2; n++ {
		q := provenancePos(80 + int(n))
		pid, _ := p.a.Positions().Save(p.ca, p.sa, &q)
		if _, err := ms.CreateMove(p.ca, p.sa, &domain.Move{GameID: gid, MoveNumber: n, MoveType: "checker", PositionID: pid,
			Player: 1, Dice: [2]int32{3, 1}, CheckerMove: "8/5 6/5"}); err != nil {
			t.Fatal(err)
		}
	}
	u := famUUID(t, p.a, p.ca, p.sa, "match", mid)
	p.send()
	bid, ok := famID(p.b, p.cb, p.sb, "match", u)
	if !ok {
		t.Fatal("the match did not arrive under its uuid")
	}
	got, err := p.b.Matches().Get(p.cb, p.sb, bid)
	if err != nil || got.Player1Name != "Alice" || got.Comment != "a good one" || got.MatchHash != "cs-hash" {
		t.Fatalf("match in b = %+v, %v", got, err)
	}
	nGames, nMoves, nBoards := 0, 0, 0
	for g, err := range p.b.Matches().Games(p.cb, p.sb, bid) {
		if err != nil {
			t.Fatal(err)
		}
		nGames++
		_ = g
	}
	for mv, err := range p.b.Matches().MovesByMatch(p.cb, p.sb, bid) {
		if err != nil {
			t.Fatal(err)
		}
		nMoves++
		if mv.PositionID != 0 {
			nBoards++
		}
	}
	if nGames != 1 || nMoves != 2 || nBoards != 2 {
		t.Errorf("games %d, moves %d, boards %d", nGames, nMoves, nBoards)
	}
	if rep := p.replay(); rep.Applied != 0 {
		t.Errorf("a replay applied %+v", rep)
	}

	tick()
	if err := ms.UpdateComment(p.ca, p.sa, mid, "better note"); err != nil {
		t.Fatal(err)
	}
	p.send()
	if got, _ := p.b.Matches().Get(p.cb, p.sb, bid); got.Comment != "better note" {
		t.Errorf("the note did not travel: %q", got.Comment)
	}
	tick()
	if _, err := trash.Match(p.ca, p.a, p.sa, mid); err != nil {
		t.Fatal(err)
	}
	p.send()
	if _, ok := famID(p.b, p.cb, p.sb, "match", u); ok {
		t.Error("the trashed match is still in b")
	}
	if e := p.entry(p.b, p.cb, p.sb, u); e.Op != storage.ChangeTrash {
		t.Errorf("b's journal says %s, want trash", e.Op)
	}
}

func publicIDOf(t *testing.T, s storage.Storage, ctx context.Context, scope string, id int64) string {
	t.Helper()
	v, err := s.(storage.PublicIDProvider).PublicIDs().PublicID(ctx, scope, id)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// A position carries a public_id the engine stores and returns as given: it
// never reads it. It travels with the position, an edit and a clearing follow,
// and writing it is an edit the journal shows.
func scenarioPublicID(p *csPair) {
	t := p.t
	p.catchUp()
	q := provenancePos(81)
	pa, _ := p.a.Positions().Save(p.ca, p.sa, &q)
	ids := p.a.(storage.PublicIDProvider).PublicIDs()
	if got := publicIDOf(t, p.a, p.ca, p.sa, pa); got != "" {
		t.Fatalf("a fresh position holds public_id %q", got)
	}
	u := famUUID(t, p.a, p.ca, p.sa, "position", pa)
	p.send()
	bid, ok := famID(p.b, p.cb, p.sb, "position", u)
	if !ok {
		t.Fatal("the position did not arrive")
	}
	if got := publicIDOf(t, p.b, p.cb, p.sb, bid); got != "" {
		t.Errorf("a position without public_id arrived with %q", got)
	}

	// Any text, kept byte for byte.
	const opaque = "Pos 7/ é 日本"
	before := p.entry(p.a, p.ca, p.sa, u)
	tick()
	if err := ids.SetPublicID(p.ca, p.sa, pa, opaque); err != nil {
		t.Fatal(err)
	}
	if after := p.entry(p.a, p.ca, p.sa, u); after.ChangedAt <= before.ChangedAt || after.Op != storage.ChangeUpdate {
		t.Errorf("setting public_id left the journal at %+v (was %+v)", after, before)
	}
	p.send()
	if got := publicIDOf(t, p.b, p.cb, p.sb, bid); got != opaque {
		t.Errorf("public_id in b = %q, want %q", got, opaque)
	}
	if rep := p.replay(); rep.Applied != 0 {
		t.Errorf("a replay applied %+v", rep)
	}
	if err := ids.SetPublicID(p.ca, p.sa, 1<<40, "x"); err == nil {
		t.Error("public_id set on a position that does not exist")
	}

	tick()
	if err := ids.SetPublicID(p.ca, p.sa, pa, ""); err != nil {
		t.Fatal(err)
	}
	p.send()
	if got := publicIDOf(t, p.b, p.cb, p.sb, bid); got != "" {
		t.Errorf("the cleared public_id is still %q in b", got)
	}

	// A position arriving with one (created after it was set).
	q2 := provenancePos(82)
	p2, _ := p.a.Positions().Save(p.ca, p.sa, &q2)
	_ = ids.SetPublicID(p.ca, p.sa, p2, "second")
	u2 := famUUID(t, p.a, p.ca, p.sa, "position", p2)
	p.send()
	if id2, ok := famID(p.b, p.cb, p.sb, "position", u2); !ok || publicIDOf(t, p.b, p.cb, p.sb, id2) != "second" {
		t.Error("a position created with a public_id lost it on the way")
	}
}
