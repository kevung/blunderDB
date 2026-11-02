package storagetest

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/changeset"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/trash"
)

var linkMarks = []domain.DrawingMark{{Kind: domain.DrawingArrow, From: 13, To: 7, Color: "red"}}

func (p *csPair) setLink(s storage.Storage, c context.Context, scope string, id int64, d *domain.Derivation) {
	p.t.Helper()
	if err := s.(storage.DerivationProvider).PositionDerivations().SetDerivedFrom(c, scope, id, d); err != nil {
		p.t.Fatal(err)
	}
}

func (p *csPair) link(s storage.Storage, c context.Context, scope string, id int64) *domain.Derivation {
	p.t.Helper()
	d, err := s.(storage.DerivationProvider).PositionDerivations().DerivedFrom(c, scope, id)
	if err != nil {
		p.t.Fatal(err)
	}
	return d
}

func (p *csPair) draw(s storage.Storage, c context.Context, scope string, id int64, m []domain.DrawingMark) {
	p.t.Helper()
	if err := s.(storage.DrawingProvider).PositionDrawings().SetDrawing(c, scope, id, m); err != nil {
		p.t.Fatal(err)
	}
}

func (p *csPair) drawing(s storage.Storage, c context.Context, scope string, id int64) []domain.DrawingMark {
	p.t.Helper()
	m, err := s.(storage.DrawingProvider).PositionDrawings().Drawing(c, scope, id)
	if err != nil {
		p.t.Fatal(err)
	}
	return m
}

// onB is the id in b of the position a holds as id.
func (p *csPair) onB(id int64) int64 {
	p.t.Helper()
	q, err := p.a.Positions().Load(p.ca, p.sa, id)
	if err != nil {
		p.t.Fatal(err)
	}
	bid, found, err := p.b.Positions().Exists(p.cb, p.sb, storage.ZobristOf(q))
	if err != nil || !found {
		p.t.Fatalf("position %d of a is not in b (%v)", id, err)
	}
	return bid
}

// CheckChangeSetLinks is what a link or a drawing must survive between two
// libraries. pair gives two libraries that have never met.
func CheckChangeSetLinks(t *testing.T, pair func(t *testing.T) *ChangeSetPair) {
	for _, sc := range []struct {
		name string
		run  func(*csPair)
	}{
		{"LinkOfAVariantHeldUnderAnOlderUUID", scenarioLinkOlderUUID},
		{"LinkWaitsForAnOriginOfALaterPage", scenarioLinkAcrossPages},
		{"LinkWaitsForAnOriginOfALaterSet", scenarioLinkAcrossSets},
		{"ErasedWaitingLinkStaysErased", scenarioErasedWaitingLink},
		{"WaitingLinkErasedMeanwhileStaysErased", scenarioWaitingLinkErasedMeanwhile},
		{"TrashKeepsPartFacts", scenarioTrashKeepsPartFacts},
		{"RefusedLoopKeepsTheLink", scenarioLoopKeepsLink},
		{"FirstMeetKeepsWhatThePeerNeverSaw", scenarioFirstMeet},
		{"TieAtTheFloorLosesNothing", scenarioFloorTie},
		{"ErasureTravelsAndIsDated", scenarioErasure},
		{"ImportDatesNoVariantThroughTheCascade", scenarioImportCascade},
		{"PositionOfAMatchWaitsForItsRealRecord", scenarioMatchPositionRecord},
	} {
		t.Run(sc.name, func(t *testing.T) { p := pair(t); sc.run(p) })
	}
}

func scenarioLinkOlderUUID(p *csPair) {
	t := p.t
	qv := syncPos(311)
	bv, err := p.b.Positions().Save(p.cb, p.sb, &qv)
	if err != nil {
		t.Fatal(err)
	}
	ubv := famUUID(t, p.b, p.cb, p.sb, "position", bv)
	tick()
	ids := p.saved(310, 2)
	o, v := ids[0], ids[1]
	p.setLink(p.a, p.ca, p.sa, v, &domain.Derivation{FromID: o, Move: "m"})
	tick()
	p.draw(p.a, p.ca, p.sa, o, linkMarks) // the origin now comes after the variant
	uav := famUUID(t, p.a, p.ca, p.sa, "position", v)
	if ubv >= uav {
		t.Fatalf("the variant must be older in b: %s, %s", ubv, uav)
	}
	set, err := changeset.Export(p.ca, p.a, p.sa, storage.Cursor{}, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if rep, err := changeset.Import(p.cb, p.b, p.sb, set); err != nil {
		t.Fatalf("import stopped on the kept uuid: %v (%+v)", err, rep)
	}
	if d := p.link(p.b, p.cb, p.sb, bv); d == nil || d.FromID != p.onB(o) || d.Move != "m" {
		t.Errorf("link in b = %+v, want position %d", d, p.onB(o))
	}
	if eb := p.entry(p.b, p.cb, p.sb, ubv); eb.ChangedAt != p.entry(p.a, p.ca, p.sa, uav).ChangedAt {
		t.Errorf("the kept uuid's entry is dated %s, want the record's", eb.ChangedAt)
	}
}

func scenarioLinkAcrossPages(p *csPair) {
	t := p.t
	ids := p.saved(300, 2)
	o, v := ids[0], ids[1]
	p.setLink(p.a, p.ca, p.sa, v, &domain.Derivation{FromID: o, Move: "m"})
	tick()
	p.draw(p.a, p.ca, p.sa, o, linkMarks)
	p.pagedSend(1)
	bv, bo := p.onB(v), p.onB(o)
	if d := p.link(p.b, p.cb, p.sb, bv); d == nil || d.FromID != bo || d.Move != "m" {
		t.Errorf("link in b after pages of one = %+v, want position %d", d, bo)
	}
	uv := famUUID(t, p.a, p.ca, p.sa, "position", v)
	if ea, eb := p.entry(p.a, p.ca, p.sa, uv), p.entry(p.b, p.cb, p.sb, uv); ea.ChangedAt != eb.ChangedAt {
		t.Errorf("rebuilding the link dated the variant: %s, %s", ea.ChangedAt, eb.ChangedAt)
	}
	if rep := p.replay(); rep.Applied != 0 || len(rep.Missing) != 0 {
		t.Errorf("a replay applied or missed something: %+v", rep)
	}
	back := p.swapped()
	back.cursorFrom = storage.Cursor{}
	if rep := back.pagedSend(1); rep.Applied != 0 || len(rep.Missing) != 0 {
		t.Errorf("the way back applied or missed something: %+v", rep)
	}
}

func scenarioLinkAcrossSets(p *csPair) {
	t := p.t
	ids := p.saved(303, 2)
	o, v := ids[0], ids[1]
	p.setLink(p.a, p.ca, p.sa, v, &domain.Derivation{FromID: o})
	uv := famUUID(t, p.a, p.ca, p.sa, "position", v)
	set, err := changeset.Export(p.ca, p.a, p.sa, storage.Cursor{}, 1000)
	if err != nil {
		t.Fatal(err)
	}
	var onlyV, onlyO changeset.Set
	for _, r := range set.Records {
		if r.UUID == uv {
			onlyV.Records = append(onlyV.Records, r)
		} else {
			onlyO.Records = append(onlyO.Records, r)
		}
	}
	rep, err := changeset.Import(p.cb, p.b, p.sb, onlyV)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Missing) != 1 || rep.Missing[0].UUID != uv {
		t.Errorf("the absent origin is not reported: %+v", rep.Missing)
	}
	// The way back holds the link while b waits: nothing to lose there.
	bv := p.onB(v)
	if d := p.link(p.b, p.cb, p.sb, bv); d != nil {
		t.Fatalf("a link to a position b does not hold: %+v", d)
	}
	back := p.swapped()
	back.cursorFrom = storage.Cursor{}
	if rep := back.pagedSend(1000); rep.Modified != 0 {
		t.Errorf("a waiting link changed the peer: %+v", rep)
	}
	if d := p.link(p.a, p.ca, p.sa, v); d == nil || d.FromID != o {
		t.Errorf("a waiting link erased the peer's: %+v", d)
	}
	// Replayed alone, it still waits, and says so.
	if rep, err := changeset.Import(p.cb, p.b, p.sb, onlyV); err != nil || rep.Applied != 0 {
		t.Errorf("replay of the variant: %+v, %v", rep, err)
	}
	if rep, err := changeset.Import(p.cb, p.b, p.sb, onlyO); err != nil || len(rep.Missing) != 0 {
		t.Errorf("the origin's set: %+v, %v", rep, err)
	}
	if d := p.link(p.b, p.cb, p.sb, bv); d == nil || d.FromID != p.onB(o) {
		t.Errorf("the link did not come back with its origin: %+v", d)
	}
	if ea, eb := p.entry(p.a, p.ca, p.sa, uv), p.entry(p.b, p.cb, p.sb, uv); ea.ChangedAt != eb.ChangedAt {
		t.Errorf("rebuilding the link dated the variant: %s, %s", ea.ChangedAt, eb.ChangedAt)
	}
}

func scenarioLoopKeepsLink(p *csPair) {
	t := p.t
	ids := p.saved(320, 3)
	o, v, x := ids[0], ids[1], ids[2]
	p.pagedSend(1000)
	bo, bv, bx := p.onB(o), p.onB(v), p.onB(x)
	p.setLink(p.b, p.cb, p.sb, bo, &domain.Derivation{FromID: bv})
	p.setLink(p.b, p.cb, p.sb, bv, &domain.Derivation{FromID: bx, Move: "keep"})
	tick()
	p.setLink(p.a, p.ca, p.sa, v, &domain.Derivation{FromID: o})
	rep := p.pagedSend(1000)
	if d := p.link(p.b, p.cb, p.sb, bv); d == nil || d.FromID != bx || d.Move != "keep" {
		t.Errorf("the refused loop took the link: %+v", d)
	}
	if len(rep.Missing) != 1 {
		t.Errorf("the refused link is not reported: %+v", rep.Missing)
	}
}

func scenarioFirstMeet(p *csPair) {
	t := p.t
	qx, qy, qo := syncPos(331), syncPos(332), syncPos(333)
	bx, _ := p.b.Positions().Save(p.cb, p.sb, &qx)
	by, _ := p.b.Positions().Save(p.cb, p.sb, &qy)
	bo, _ := p.b.Positions().Save(p.cb, p.sb, &qo)
	p.draw(p.b, p.cb, p.sb, bx, linkMarks)
	p.setLink(p.b, p.cb, p.sb, by, &domain.Derivation{FromID: bo, Move: "m"})
	tick()
	ax, _ := p.a.Positions().Save(p.ca, p.sa, &qx)
	ay, _ := p.a.Positions().Save(p.ca, p.sa, &qy)
	p.pagedSend(1000)
	if m := p.drawing(p.b, p.cb, p.sb, bx); !reflect.DeepEqual(m, linkMarks) {
		t.Errorf("a peer that never saw the drawing erased it: %+v", m)
	}
	if d := p.link(p.b, p.cb, p.sb, by); d == nil || d.FromID != bo {
		t.Errorf("a peer that never saw the link erased it: %+v", d)
	}
	// Both sides end up with it, then nothing moves any more.
	back := p.swapped()
	back.cursorFrom = storage.Cursor{}
	back.pagedSend(1000)
	if m := p.drawing(p.a, p.ca, p.sa, ax); !reflect.DeepEqual(m, linkMarks) {
		t.Errorf("the drawing did not reach a: %+v", m)
	}
	if d := p.link(p.a, p.ca, p.sa, ay); d == nil {
		t.Error("the link did not reach a")
	}
	p.pagedSend(1000)
	back.pagedSend(1000)
	if rep := p.pagedSend(1000); rep.Applied != 0 {
		t.Errorf("the two libraries keep exchanging: %+v", rep)
	}
	if rep := back.pagedSend(1000); rep.Applied != 0 {
		t.Errorf("the two libraries keep exchanging, back: %+v", rep)
	}
}

func scenarioFloorTie(p *csPair) {
	t := p.t
	q := syncPos(334)
	aid, _ := p.a.Positions().Save(p.ca, p.sa, &q)
	p.pagedSend(1000)
	bid := p.onB(aid)
	u := famUUID(t, p.a, p.ca, p.sa, "position", aid)
	p.draw(p.b, p.cb, p.sb, bid, linkMarks)
	const floor = "1970-01-01T00:00:00.000Z"
	if err := p.a.(storage.ChangeJournalProvider).ChangeJournal().Mark(p.ca, p.sa, "position", u, storage.ChangeUpdate, floor); err != nil {
		t.Fatal(err)
	}
	if err := p.b.(storage.ChangeJournalProvider).ChangeJournal().Mark(p.cb, p.sb, "position", u, storage.ChangeUpdate, floor); err != nil {
		t.Fatal(err)
	}
	p.cursorFrom = storage.Cursor{}
	p.pagedSend(1000)
	if m := p.drawing(p.b, p.cb, p.sb, bid); !reflect.DeepEqual(m, linkMarks) {
		t.Errorf("a tie at the floor erased the drawing: %+v", m)
	}
	back := p.swapped()
	back.cursorFrom = storage.Cursor{}
	back.pagedSend(1000)
	if m := p.drawing(p.a, p.ca, p.sa, aid); !reflect.DeepEqual(m, linkMarks) {
		t.Errorf("a tie at the floor did not bring the drawing to a: %+v", m)
	}
}

func scenarioErasure(p *csPair) {
	t := p.t
	ids := p.saved(335, 3)
	x, o, v := ids[0], ids[1], ids[2]
	p.draw(p.a, p.ca, p.sa, x, linkMarks)
	p.setLink(p.a, p.ca, p.sa, v, &domain.Derivation{FromID: o})
	p.pagedSend(1000)
	bx, bv := p.onB(x), p.onB(v)

	// Erased at one end, erased at the other: the record says so.
	tick()
	p.draw(p.a, p.ca, p.sa, x, nil)
	p.setLink(p.a, p.ca, p.sa, v, nil)
	p.pagedSend(1000)
	if m := p.drawing(p.b, p.cb, p.sb, bx); len(m) != 0 {
		t.Errorf("the erasure did not travel: %+v", m)
	}
	if d := p.link(p.b, p.cb, p.sb, bv); d != nil {
		t.Errorf("the erased link did not travel: %+v", d)
	}

	// An erasure older than the drawing it meets takes nothing, even when a
	// later edit carries it.
	want := []domain.DrawingMark{{Kind: domain.DrawingSquare, From: 6, To: 6, Color: "blue"}}
	tick()
	p.draw(p.a, p.ca, p.sa, x, linkMarks)
	p.pagedSend(1000)
	tick()
	p.draw(p.a, p.ca, p.sa, x, nil)
	tick()
	p.draw(p.b, p.cb, p.sb, bx, want)
	tick()
	if err := p.a.Analyses().Save(p.ca, p.sa, x, doublingAnalysis()); err != nil {
		t.Fatal(err)
	}
	p.pagedSend(1000)
	if m := p.drawing(p.b, p.cb, p.sb, bx); !reflect.DeepEqual(m, want) {
		t.Errorf("an older erasure took a later drawing: %+v", m)
	}
	if _, err := p.b.Analyses().Load(p.cb, p.sb, bx); err != nil {
		t.Errorf("the later edit did not arrive: %v", err)
	}
}

func scenarioImportCascade(p *csPair) {
	t := p.t
	ids := p.saved(340, 2)
	o, v := ids[0], ids[1]
	p.setLink(p.a, p.ca, p.sa, v, &domain.Derivation{FromID: o})
	p.pagedSend(1000)
	uv := famUUID(t, p.a, p.ca, p.sa, "position", v)
	tick()
	// b applies the origin's delete alone: its cascade must date nothing.
	if err := p.a.Positions().Delete(p.ca, p.sa, o); err != nil {
		t.Fatal(err)
	}
	before := p.entry(p.b, p.cb, p.sb, uv)
	set, err := changeset.Export(p.ca, p.a, p.sa, p.cursorFrom, 1000)
	if err != nil {
		t.Fatal(err)
	}
	var del changeset.Set
	for _, r := range set.Records {
		if r.Op == storage.ChangeDelete {
			del.Records = append(del.Records, r)
		}
	}
	if len(del.Records) != 1 {
		t.Fatalf("delete records: %+v", set.Records)
	}
	if _, err := changeset.Import(p.cb, p.b, p.sb, del); err != nil {
		t.Fatal(err)
	}
	if e := p.entry(p.b, p.cb, p.sb, uv); e.ChangedAt != before.ChangedAt || e.Seq != before.Seq {
		t.Errorf("the cascade of an import touched the variant's entry: %+v, was %+v", e, before)
	}

	// The whole round trip: the author draws on the variant, the peer gets it,
	// and the way back takes nothing from the author.
	tick()
	p.draw(p.a, p.ca, p.sa, v, linkMarks)
	tick()
	p.pagedSend(1000)
	bv := p.onB(v)
	if m := p.drawing(p.b, p.cb, p.sb, bv); !reflect.DeepEqual(m, linkMarks) {
		t.Errorf("the drawing did not reach b: %+v", m)
	}
	back := p.swapped()
	back.cursorFrom = storage.Cursor{}
	back.pagedSend(1000)
	if m := p.drawing(p.a, p.ca, p.sa, v); !reflect.DeepEqual(m, linkMarks) {
		t.Errorf("the way back erased the author's drawing: %+v", m)
	}
}

func scenarioMatchPositionRecord(p *csPair) {
	t := p.t
	p.saved(350, 2)
	ms := p.a.Matches()
	write := func(hash string, board int) int64 {
		m := domain.Match{Player1Name: "A", Player2Name: "B", Event: "O", MatchLength: 7,
			MatchDate: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), FilePath: "m.xg", GameCount: 1,
			MatchHash: hash, CanonicalHash: hash + "-c"}
		mid, err := ms.Save(p.ca, p.sa, &m)
		if err != nil {
			t.Fatal(err)
		}
		gid, err := ms.CreateGame(p.ca, p.sa, &domain.Game{MatchID: mid, GameNumber: 1, Winner: 1, PointsWon: 2, MoveCount: 1})
		if err != nil {
			t.Fatal(err)
		}
		q := syncPos(board)
		pid, err := p.a.Positions().Save(p.ca, p.sa, &q)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ms.CreateMove(p.ca, p.sa, &domain.Move{GameID: gid, MoveNumber: 1, MoveType: "checker", PositionID: pid,
			Player: 1, Dice: [2]int32{3, 1}, CheckerMove: "8/5 6/5"}); err != nil {
			t.Fatal(err)
		}
		return pid
	}
	write("h1", 360)
	var fresh int64
	for page := 0; page < 50; page++ {
		set, err := changeset.Export(p.ca, p.a, p.sa, p.cursorFrom, 1)
		if err != nil {
			t.Fatal(err)
		}
		p.cursorFrom = set.Cursor
		if fresh == 0 && set.Cursor.Family == "match" {
			fresh = write("h2", 361)
			if err := p.a.Analyses().Save(p.ca, p.sa, fresh, doublingAnalysis()); err != nil {
				t.Fatal(err)
			}
			p.draw(p.a, p.ca, p.sa, fresh, linkMarks)
			tick()
		}
		if _, err := changeset.Import(p.cb, p.b, p.sb, set); err != nil {
			t.Fatal(err)
		}
		if !set.More {
			break
		}
	}
	if fresh == 0 {
		t.Fatal("the run never reached the match family")
	}
	tick()
	p.pagedSend(1000)
	bid := p.onB(fresh)
	if _, err := p.b.Analyses().Load(p.cb, p.sb, bid); err != nil {
		t.Errorf("analysis of the match's position: %v", err)
	}
	if m := p.drawing(p.b, p.cb, p.sb, bid); !reflect.DeepEqual(m, linkMarks) {
		t.Errorf("drawing of the match's position: %+v", m)
	}
}

// ChangeSetPair is two libraries that exchange change sets.
type ChangeSetPair = csPair

// NewChangeSetPair names two libraries for CheckChangeSetLinks.
func NewChangeSetPair(t *testing.T, a storage.Storage, ca context.Context, sa string, b storage.Storage, cb context.Context, sb string) *ChangeSetPair {
	return &csPair{t: t, a: a, ca: ca, sa: sa, b: b, cb: cb, sb: sb}
}

// waitingOnB leaves on b a variant whose origin is in the set it returns.
func (p *csPair) waitingOnB(from int) (variant int64, origins changeset.Set) {
	t := p.t
	ids := p.saved(from, 2)
	o, v := ids[0], ids[1]
	p.setLink(p.a, p.ca, p.sa, v, &domain.Derivation{FromID: o})
	uv := famUUID(t, p.a, p.ca, p.sa, "position", v)
	set, err := changeset.Export(p.ca, p.a, p.sa, storage.Cursor{}, 1000)
	if err != nil {
		t.Fatal(err)
	}
	var onlyV changeset.Set
	for _, r := range set.Records {
		if r.UUID == uv {
			onlyV.Records = append(onlyV.Records, r)
		} else {
			origins.Records = append(origins.Records, r)
		}
	}
	if _, err := changeset.Import(p.cb, p.b, p.sb, onlyV); err != nil {
		t.Fatal(err)
	}
	return p.onB(v), origins
}

// Erasing a link that waits for its origin is an erasure like another: dated,
// journaled, and the origin's arrival does not bring the link back.
func scenarioErasedWaitingLink(p *csPair) {
	t := p.t
	bv, origins := p.waitingOnB(340)
	u := famUUID(t, p.b, p.cb, p.sb, "position", bv)
	before := p.entry(p.b, p.cb, p.sb, u)
	tick()
	p.setLink(p.b, p.cb, p.sb, bv, nil)
	after := p.entry(p.b, p.cb, p.sb, u)
	if after.ChangedAt <= before.ChangedAt {
		t.Errorf("the erasure left the journal at %s (was %s)", after.ChangedAt, before.ChangedAt)
	}
	f, known, err := p.b.(storage.ChangeJournalProvider).ChangeJournal().PartFact(p.cb, p.sb, bv, storage.PartDerivation)
	if err != nil || !known || f.PendingOrigin != "" || f.At != after.ChangedAt {
		t.Errorf("the fact after the erasure: %+v, known %v, %v; want no origin, dated %s", f, known, err, after.ChangedAt)
	}
	if _, err := changeset.Import(p.cb, p.b, p.sb, origins); err != nil {
		t.Fatal(err)
	}
	if d := p.link(p.b, p.cb, p.sb, bv); d != nil {
		t.Errorf("the origin's arrival brought an erased link back: %+v", d)
	}
}

// racingStores erases, right after the import has read the waiting links, what
// a writer on another connection would: settleLinks holds a stale list.
type racingStores struct {
	storage.Storage
	race func()
}

func (r racingStores) ChangeJournal() storage.ChangeJournalStore {
	return racingJournal{r.Storage.(storage.ChangeJournalProvider).ChangeJournal(), r.race}
}

func (r racingStores) ObjectUUIDs() storage.ObjectUUIDStore {
	return r.Storage.(storage.ObjectUUIDProvider).ObjectUUIDs()
}

type racingJournal struct {
	storage.ChangeJournalStore
	race func()
}

func (j racingJournal) PendingLinks(ctx context.Context, scope string) ([]storage.PartFact, error) {
	list, err := j.ChangeJournalStore.PendingLinks(ctx, scope)
	j.race()
	return list, err
}

// A link erased between the reading of the waiting list and its settling stays
// erased: the transaction goes by what it reads, not by the list.
func scenarioWaitingLinkErasedMeanwhile(p *csPair) {
	bv, origins := p.waitingOnB(350)
	stores := racingStores{p.b, func() { p.setLink(p.b, p.cb, p.sb, bv, nil) }}
	if _, err := changeset.Import(p.cb, stores, p.sb, origins); err != nil {
		p.t.Fatal(err)
	}
	if d := p.link(p.b, p.cb, p.sb, bv); d != nil {
		p.t.Errorf("a link erased meanwhile was made from the stale list: %+v", d)
	}
}

// The trash keeps what the library knew of the parts of a position: a link
// still waiting waits again after the restore, and restoring is no new write.
func scenarioTrashKeepsPartFacts(p *csPair) {
	t := p.t
	bv, origins := p.waitingOnB(360)
	journal := p.b.(storage.ChangeJournalProvider).ChangeJournal()
	waiting, _, err := journal.PartFact(p.cb, p.sb, bv, storage.PartDerivation)
	if err != nil || waiting.PendingOrigin == "" {
		t.Fatalf("the scenario needs a waiting link: %+v, %v", waiting, err)
	}
	drawn := []domain.DrawingMark{{Kind: domain.DrawingSquare, From: 6, To: 6, Color: "green"}}
	drawings := p.b.(storage.DrawingProvider).PositionDrawings()
	if err := drawings.SetDrawing(p.cb, p.sb, bv, drawn); err != nil {
		t.Fatal(err)
	}
	if err := drawings.SetDrawing(p.cb, p.sb, bv, nil); err != nil {
		t.Fatal(err)
	}
	erased, known, err := journal.PartFact(p.cb, p.sb, bv, storage.PartDrawing)
	if err != nil || !known {
		t.Fatalf("the scenario needs an erased drawing: %+v, %v, %v", erased, known, err)
	}

	entry, err := trash.Position(p.cb, p.b, p.sb, bv)
	if err != nil {
		t.Fatal(err)
	}
	tick()
	res, err := trash.Restore(p.cb, p.b, p.sb, entry)
	if err != nil {
		t.Fatal(err)
	}
	if got, _, err := journal.PartFact(p.cb, p.sb, res.ID, storage.PartDerivation); err != nil || got.PendingOrigin != waiting.PendingOrigin || got.At != waiting.At {
		t.Errorf("the waiting link after the restore: %+v, %v; want %+v", got, err, waiting)
	}
	if got, known, err := journal.PartFact(p.cb, p.sb, res.ID, storage.PartDrawing); err != nil || !known || got.At != erased.At {
		t.Errorf("the erased drawing after the restore: %+v, known %v, %v; want dated %s", got, known, err, erased.At)
	}
	if _, err := changeset.Import(p.cb, p.b, p.sb, origins); err != nil {
		t.Fatal(err)
	}
	if d := p.link(p.b, p.cb, p.sb, res.ID); d == nil {
		t.Error("the restored variant did not get its link when the origin arrived")
	}
}
