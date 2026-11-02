package storagetest

import (
	"context"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/changeset"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// pagedSend runs one whole export of a into b, a page of limit at a time.
func (p *csPair) pagedSend(limit int) changeset.Report {
	p.t.Helper()
	var tot changeset.Report
	for page := 0; ; page++ {
		if page > 500 {
			p.t.Fatal("the export never ends")
		}
		set, err := changeset.Export(p.ca, p.a, p.sa, p.cursorFrom, limit)
		if err != nil {
			p.t.Fatalf("export: %v", err)
		}
		if len(set.Records) > limit {
			p.t.Fatalf("page of %d records, limit %d", len(set.Records), limit)
		}
		p.cursorFrom = set.Cursor
		rep, err := changeset.Import(p.cb, p.b, p.sb, set)
		if err != nil {
			p.t.Fatalf("import: %v", err)
		}
		tot.Applied += rep.Applied
		tot.Added += rep.Added
		tot.Modified += rep.Modified
		tot.Removed += rep.Removed
		tot.Older += rep.Older
		tot.Conflicts = append(tot.Conflicts, rep.Conflicts...)
		tot.Missing = append(tot.Missing, rep.Missing...)
		if !set.More {
			return tot
		}
	}
}

func (p *csPair) membersOf(s storage.Storage, ctx context.Context, scope string, id int64) []int64 {
	p.t.Helper()
	got, err := s.Collections().PositionIDs(ctx, scope, id, storage.ListOpts{})
	if err != nil {
		p.t.Fatal(err)
	}
	return got
}

// saved stores boards n, n+1, ... in a.
func (p *csPair) saved(from, count int) []int64 {
	p.t.Helper()
	var ids []int64
	for n := from; n < from+count; n++ {
		q := syncPos(n)
		id, err := p.a.Positions().Save(p.ca, p.sa, &q)
		if err != nil {
			p.t.Fatal(err)
		}
		ids = append(ids, id)
	}
	return ids
}

// syncPos is a board of its own: provenancePos's score is capped by the hash
// past a point, and those numbers are other scenarios'.
func syncPos(n int) domain.Position {
	p := domain.InitializePosition()
	p.DecisionType = domain.CheckerAction
	p.Score = [2]int{n%25 + 1, n/25 + 1}
	return p
}

func doublingAnalysis() *domain.PositionAnalysis {
	return &domain.PositionAnalysis{AnalysisType: "DoublingCube", AnalysisEngineVersion: "x"}
}

// CheckChangeSetFirstSync: a library with no journal, whatever the size of the
// pages, reaches a peer whole and comes back whole.
func CheckChangeSetFirstSync(t *testing.T, a storage.Storage, ca context.Context, sa string, b storage.Storage, cb context.Context, sb string, forget func()) {
	for i, limit := range []int{1, 2, 3, 1000} {
		t.Run("limit "+strconv.Itoa(limit), func(t *testing.T) {
			p := &csPair{t: t, a: a, ca: ca, sa: sa, b: b, cb: cb, sb: sb}
			from := 200 + 10*i
			ids := p.saved(from, 3)
			col, err := a.Collections().Create(ca, sa, "Col "+strconv.Itoa(limit), "")
			if err != nil {
				t.Fatal(err)
			}
			if err := a.Collections().AddPositions(ca, sa, col, ids); err != nil {
				t.Fatal(err)
			}
			if _, err := a.Comments().Add(ca, sa, ids[0], "ma note"); err != nil {
				t.Fatal(err)
			}
			if err := a.Analyses().Save(ca, sa, ids[0], doublingAnalysis()); err != nil {
				t.Fatal(err)
			}
			cu := famUUID(t, a, ca, sa, "collection", col)
			forget()
			if rep := p.pagedSend(limit); len(rep.Missing) != 0 {
				t.Errorf("first sync reports missing references: %+v", rep.Missing)
			}
			bid, ok := famID(b, cb, sb, "collection", cu)
			if !ok {
				t.Fatal("the collection did not arrive")
			}
			if got := p.membersOf(b, cb, sb, bid); len(got) != 3 {
				t.Errorf("members in b: %d, want 3", len(got))
			}
			q := syncPos(from)
			pid, found, err := b.Positions().Exists(cb, sb, storage.ZobristOf(&q))
			if err != nil || !found {
				t.Fatalf("position in b: %v, %v", found, err)
			}
			if _, err := b.Analyses().Load(cb, sb, pid); err != nil {
				t.Errorf("the analysis lost against the note's board: %v", err)
			}
			back := p.swapped()
			back.cursorFrom = storage.Cursor{}
			back.pagedSend(1000)
			if got := p.membersOf(a, ca, sa, col); len(got) != 3 {
				t.Errorf("members in a after the round trip: %d, want 3", len(got))
			}
		})
	}
}

// CheckChangeSetSync: what a paged, journaled exchange must keep.
func CheckChangeSetSync(t *testing.T, a storage.Storage, ca context.Context, sa string, b storage.Storage, cb context.Context, sb string) {
	pair := func(t *testing.T) *csPair { return &csPair{t: t, a: a, ca: ca, sa: sa, b: b, cb: cb, sb: sb} }
	t.Run("PagesCutBetweenACollectionAndItsMembers", func(t *testing.T) { scenarioPagedJournal(pair(t)) })
	t.Run("ExportResumesWithAPositionMovedDuringTheRun", func(t *testing.T) { scenarioMovedDuringRun(pair(t)) })
	t.Run("FallbackBoardNeverBeatsTheRealPosition", func(t *testing.T) { scenarioFallbackBoard(pair(t)) })
	t.Run("AbsentDependencyEmptiesNothingAndIsReported", func(t *testing.T) { scenarioAbsentDependency(pair(t)) })
	t.Run("TieOnACollectionKeepsEveryMember", func(t *testing.T) { scenarioTieKeepsMembers(pair(t)) })
	t.Run("ViewingAMatchDatesNothing", func(t *testing.T) { scenarioMatchViewed(pair(t)) })
	t.Run("SwapPlayersTravelsWhole", func(t *testing.T) { scenarioMatchSwapped(pair(t)) })
	t.Run("RewrittenMatchTravelsWhole", func(t *testing.T) { scenarioMatchRewritten(pair(t)) })
	t.Run("RewriteOntoAHeldHashIsAConflict", func(t *testing.T) { scenarioMatchHashTaken(pair(t)) })
	t.Run("EmptyPublicIDOverwritesNothing", func(t *testing.T) { scenarioEmptyPublicID(pair(t)) })
	t.Run("ReportSplitsAddedModifiedRemoved", func(t *testing.T) { scenarioReportKinds(pair(t)) })
	t.Run("OverwrittenNoteKeepsAVersion", func(t *testing.T) { scenarioNoteVersion(pair(t)) })
}

// The position is re-analysed after it joined the collection: its entry sorts
// after the collection's, and a limit of 3 falls between the two.
func scenarioPagedJournal(p *csPair) {
	t := p.t
	p.catchUp()
	ids := p.saved(300, 3)
	col, _ := p.a.Collections().Create(p.ca, p.sa, "Paged", "")
	if err := p.a.Collections().AddPositions(p.ca, p.sa, col, ids); err != nil {
		t.Fatal(err)
	}
	tick()
	if err := p.a.Analyses().Save(p.ca, p.sa, ids[0], doublingAnalysis()); err != nil {
		t.Fatal(err)
	}
	if rep := p.pagedSend(3); len(rep.Missing) != 0 {
		t.Errorf("missing references: %+v", rep.Missing)
	}
	bid, ok := famID(p.b, p.cb, p.sb, "collection", famUUID(t, p.a, p.ca, p.sa, "collection", col))
	if !ok {
		t.Fatal("the collection did not arrive")
	}
	if got := p.membersOf(p.b, p.cb, p.sb, bid); len(got) != 3 {
		t.Errorf("members in b: %d, want 3", len(got))
	}
}

// An export is several calls: what changes between two of them still arrives
// before what points at it, or in the next run.
func scenarioMovedDuringRun(p *csPair) {
	t := p.t
	p.catchUp()
	ids := p.saved(310, 3)
	col, _ := p.a.Collections().Create(p.ca, p.sa, "Moved", "")
	if err := p.a.Collections().AddPositions(p.ca, p.sa, col, ids[:2]); err != nil {
		t.Fatal(err)
	}
	set, err := changeset.Export(p.ca, p.a, p.sa, p.cursorFrom, 1)
	if err != nil {
		t.Fatal(err)
	}
	p.cursorFrom = set.Cursor
	if _, err := changeset.Import(p.cb, p.b, p.sb, set); err != nil {
		t.Fatal(err)
	}
	// Mid-run: a position not read yet moves to the end of the journal, and a
	// new one joins the collection.
	tick()
	if err := p.a.Analyses().Save(p.ca, p.sa, ids[1], doublingAnalysis()); err != nil {
		t.Fatal(err)
	}
	if err := p.a.Collections().AddPositions(p.ca, p.sa, col, ids[2:]); err != nil {
		t.Fatal(err)
	}
	p.pagedSend(1) // the end of the run
	p.pagedSend(1) // the next one
	bid, ok := famID(p.b, p.cb, p.sb, "collection", famUUID(t, p.a, p.ca, p.sa, "collection", col))
	if !ok {
		t.Fatal("the collection did not arrive")
	}
	if got := p.membersOf(p.b, p.cb, p.sb, bid); len(got) != 3 {
		t.Errorf("members in b: %d, want 3", len(got))
	}
	// A cursor at rest brings nothing more.
	set, err = changeset.Export(p.ca, p.a, p.sa, p.cursorFrom, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if rep, err := changeset.Import(p.cb, p.b, p.sb, set); err != nil || rep.Applied != 0 {
		t.Errorf("a run at rest applied %+v, %v", rep, err)
	}
}

// A note reaches b before its position: the board it carries stands in, and the
// real record wins whatever its date, the floor included.
func scenarioFallbackBoard(p *csPair) {
	t := p.t
	p.catchUp()
	for _, floorDated := range []bool{false, true} {
		n := 320
		if floorDated {
			n = 321
		}
		id := p.saved(n, 1)[0]
		if err := p.a.Analyses().Save(p.ca, p.sa, id, doublingAnalysis()); err != nil {
			t.Fatal(err)
		}
		if _, err := p.a.Comments().Add(p.ca, p.sa, id, "avant sa position"); err != nil {
			t.Fatal(err)
		}
		set, err := changeset.Export(p.ca, p.a, p.sa, p.cursorFrom, 1000)
		if err != nil {
			t.Fatal(err)
		}
		p.cursorFrom = set.Cursor
		byFamily := func(family string) changeset.Set {
			var out changeset.Set
			for _, r := range set.Records {
				if r.Family == family {
					if floorDated && family == "position" {
						r.ChangedAt = "1970-01-01T00:00:00.000Z"
					}
					out.Records = append(out.Records, r)
				}
			}
			return out
		}
		if _, err := changeset.Import(p.cb, p.b, p.sb, byFamily("comment")); err != nil {
			t.Fatal(err)
		}
		// The time of the fallback board is not now: it would beat the record.
		tick()
		if rep, err := changeset.Import(p.cb, p.b, p.sb, byFamily("position")); err != nil || rep.Applied != 1 {
			t.Errorf("floor=%v: the real position lost against its fallback: %+v, %v", floorDated, rep, err)
		}
		q := syncPos(n)
		pid, found, _ := p.b.Positions().Exists(p.cb, p.sb, storage.ZobristOf(&q))
		if !found {
			t.Fatal("no position in b")
		}
		if _, err := p.b.Analyses().Load(p.cb, p.sb, pid); err != nil {
			t.Errorf("floor=%v: the analysis did not arrive: %v", floorDated, err)
		}
	}
}

// A collection record naming a position b lacks: the members b holds stay, and
// the report names the one it could not place.
func scenarioAbsentDependency(p *csPair) {
	t := p.t
	p.catchUp()
	held := p.saved(330, 1)
	col, _ := p.a.Collections().Create(p.ca, p.sa, "Absent", "")
	if err := p.a.Collections().AddPositions(p.ca, p.sa, col, held); err != nil {
		t.Fatal(err)
	}
	p.pagedSend(1000)
	cu := famUUID(t, p.a, p.ca, p.sa, "collection", col)
	bid, _ := famID(p.b, p.cb, p.sb, "collection", cu)
	tick()
	if err := p.a.Collections().AddPositions(p.ca, p.sa, col, p.saved(331, 1)); err != nil {
		t.Fatal(err)
	}
	set, err := changeset.Export(p.ca, p.a, p.sa, p.cursorFrom, 1000)
	if err != nil {
		t.Fatal(err)
	}
	only := changeset.Set{}
	for _, r := range set.Records {
		if r.Family == "collection" && r.UUID == cu {
			only.Records = append(only.Records, r)
		}
	}
	rep, err := changeset.Import(p.cb, p.b, p.sb, only)
	if err != nil {
		t.Fatal(err)
	}
	if got := p.membersOf(p.b, p.cb, p.sb, bid); len(got) != 1 {
		t.Errorf("members in b: %d, want the 1 it holds", len(got))
	}
	if len(rep.Missing) != 1 || rep.Missing[0].UUID != cu || rep.Missing[0].Family != "collection" {
		t.Errorf("report: %+v, want one missing member of %s", rep, cu)
	}
}

// The same collection dated alike on both sides (two libraries adopted at the
// floor): both end with every member, whichever content the tie favours.
func scenarioTieKeepsMembers(p *csPair) {
	t := p.t
	p.catchUp()
	ids := p.saved(340, 3)
	col, _ := p.a.Collections().Create(p.ca, p.sa, "Tie", "")
	if err := p.a.Collections().AddPositions(p.ca, p.sa, col, ids[:1]); err != nil {
		t.Fatal(err)
	}
	p.pagedSend(1000)
	cu := famUUID(t, p.a, p.ca, p.sa, "collection", col)
	bid, _ := famID(p.b, p.cb, p.sb, "collection", cu)
	tick()
	if err := p.a.Collections().AddPositions(p.ca, p.sa, col, ids[1:]); err != nil {
		t.Fatal(err)
	}
	p.pagedSend(1000)
	if err := p.b.Collections().RemovePositions(p.cb, p.sb, bid, p.membersOf(p.b, p.cb, p.sb, bid)[1:]); err != nil {
		t.Fatal(err)
	}
	const same = "2024-06-01T10:00:00.000Z"
	p.mark(p.a, p.ca, p.sa, cu, storage.ChangeUpdate, same)
	p.mark(p.b, p.cb, p.sb, cu, storage.ChangeUpdate, same)
	send := func(q *csPair) {
		t.Helper()
		q.cursorFrom = storage.Cursor{}
		q.pagedSend(1000)
	}
	send(p.swapped())
	if got := p.membersOf(p.a, p.ca, p.sa, col); len(got) != 3 {
		t.Errorf("a after b's tied copy: %d members, want 3", len(got))
	}
	send(p)
	if got := p.membersOf(p.b, p.cb, p.sb, bid); len(got) != 3 {
		t.Errorf("b after a's tied copy: %d members, want 3", len(got))
	}
	send(p.swapped())
	send(p)
	if ga, gb := p.membersOf(p.a, p.ca, p.sa, col), p.membersOf(p.b, p.cb, p.sb, bid); len(ga) != 3 || len(gb) != 3 {
		t.Errorf("after both ways again: a %d, b %d", len(ga), len(gb))
	}
}

// syncedMatch stores a match of two games in a and brings it to b.
func (p *csPair) syncedMatch(hash string, board int) (aID, bID int64) {
	t := p.t
	ms := p.a.Matches()
	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob", Event: "Open", MatchLength: 7,
		MatchDate: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), FilePath: "m.xg", GameCount: 2,
		MatchHash: hash, CanonicalHash: hash + "-c", Comment: "note A"}
	mid, err := ms.Save(p.ca, p.sa, &m)
	if err != nil {
		t.Fatal(err)
	}
	for g := int32(1); g <= 2; g++ {
		gid, err := ms.CreateGame(p.ca, p.sa, &domain.Game{MatchID: mid, GameNumber: g, Winner: 1, PointsWon: 2, MoveCount: 1})
		if err != nil {
			t.Fatal(err)
		}
		q := syncPos(board + int(g))
		pid, _ := p.a.Positions().Save(p.ca, p.sa, &q)
		if _, err := ms.CreateMove(p.ca, p.sa, &domain.Move{GameID: gid, MoveNumber: 1, MoveType: "checker", PositionID: pid,
			Player: 1, Dice: [2]int32{3, 1}, CheckerMove: "8/5 6/5"}); err != nil {
			t.Fatal(err)
		}
	}
	p.pagedSend(1000)
	bid, ok := famID(p.b, p.cb, p.sb, "match", famUUID(t, p.a, p.ca, p.sa, "match", mid))
	if !ok {
		t.Fatal("the match did not arrive")
	}
	return mid, bid
}

// body is what a match holds, read to compare two libraries.
type matchBody struct {
	Header  string
	Games   []int32
	Winners []int32
	Players []int32
	Dice    [][2]int32
}

func readBody(t *testing.T, s storage.Storage, ctx context.Context, scope string, id int64) matchBody {
	t.Helper()
	m, err := s.Matches().Get(ctx, scope, id)
	if err != nil {
		t.Fatal(err)
	}
	b := matchBody{Header: m.Player1Name + "|" + m.Player2Name + "|" + m.MatchHash + "|" + strconv.Itoa(int(m.GameCount))}
	for g, err := range s.Matches().Games(ctx, scope, id) {
		if err != nil {
			t.Fatal(err)
		}
		b.Games = append(b.Games, g.GameNumber)
		b.Winners = append(b.Winners, g.Winner)
	}
	for mv, err := range s.Matches().MovesByMatch(ctx, scope, id) {
		if err != nil {
			t.Fatal(err)
		}
		b.Players = append(b.Players, mv.Player)
		b.Dice = append(b.Dice, mv.Dice)
	}
	return b
}

func (b matchBody) equal(o matchBody) bool {
	return b.Header == o.Header && slices.Equal(b.Games, o.Games) && slices.Equal(b.Winners, o.Winners) &&
		slices.Equal(b.Players, o.Players) && slices.Equal(b.Dice, o.Dice)
}

// Only viewing a match on a must not date it over a later note on b.
func scenarioMatchViewed(p *csPair) {
	t := p.t
	p.catchUp()
	mid, bid := p.syncedMatch("viewed", 350)
	u := famUUID(t, p.a, p.ca, p.sa, "match", mid)
	before := p.entry(p.a, p.ca, p.sa, u)
	tick()
	if err := p.b.Matches().UpdateComment(p.cb, p.sb, bid, "note B, later"); err != nil {
		t.Fatal(err)
	}
	tick()
	if err := p.a.Matches().SetLastVisitedPosition(p.ca, p.sa, mid, 3); err != nil {
		t.Fatal(err)
	}
	if after := p.entry(p.a, p.ca, p.sa, u); after.ChangedAt != before.ChangedAt || after.Seq != before.Seq {
		t.Errorf("viewing moved the entry: %+v, was %+v", after, before)
	}
	p.pagedSend(1000)
	if got, _ := p.b.Matches().Get(p.cb, p.sb, bid); got.Comment != "note B, later" {
		t.Errorf("note in b = %q", got.Comment)
	}
	if v, _ := p.a.Matches().Get(p.ca, p.sa, mid); v.LastVisitedPosition != 3 {
		t.Errorf("the view itself was lost: %d", v.LastVisitedPosition)
	}
}

func scenarioMatchSwapped(p *csPair) {
	t := p.t
	p.catchUp()
	mid, bid := p.syncedMatch("swapped", 360)
	tick()
	if err := p.a.Matches().SwapPlayers(p.ca, p.sa, mid); err != nil {
		t.Fatal(err)
	}
	if rep := p.pagedSend(1000); len(rep.Conflicts) != 0 {
		t.Errorf("conflicts: %+v", rep)
	}
	ba, bb := readBody(t, p.a, p.ca, p.sa, mid), readBody(t, p.b, p.cb, p.sb, bid)
	if !ba.equal(bb) {
		t.Errorf("after the swap a holds %+v, b %+v", ba, bb)
	}
	if bb.Players[0] == 1 {
		t.Errorf("the moves did not turn: %+v", bb)
	}
}

// A transcription saved again rewrites the games in place, with new hashes.
func scenarioMatchRewritten(p *csPair) {
	t := p.t
	p.catchUp()
	mid, bid := p.syncedMatch("rewritten", 370)
	tick()
	ms := p.a.Matches()
	if _, err := ms.DeleteGames(p.ca, p.sa, mid); err != nil {
		t.Fatal(err)
	}
	for g := int32(1); g <= 3; g++ {
		gid, err := ms.CreateGame(p.ca, p.sa, &domain.Game{MatchID: mid, GameNumber: g, Winner: -1, PointsWon: 1, MoveCount: 1})
		if err != nil {
			t.Fatal(err)
		}
		q := syncPos(380 + int(g))
		pid, _ := p.a.Positions().Save(p.ca, p.sa, &q)
		if _, err := ms.CreateMove(p.ca, p.sa, &domain.Move{GameID: gid, MoveNumber: 1, MoveType: "checker", PositionID: pid,
			Player: -1, Dice: [2]int32{6, 5}, CheckerMove: "24/13"}); err != nil {
			t.Fatal(err)
		}
	}
	h, err := ms.Get(p.ca, p.sa, mid)
	if err != nil {
		t.Fatal(err)
	}
	h.MatchHash, h.CanonicalHash, h.GameCount = "rewritten-2", "rewritten-2-c", 3
	if err := ms.ReplaceHeader(p.ca, p.sa, mid, h); err != nil {
		t.Fatal(err)
	}
	p.pagedSend(1000)
	ba, bb := readBody(t, p.a, p.ca, p.sa, mid), readBody(t, p.b, p.cb, p.sb, bid)
	if !ba.equal(bb) {
		t.Errorf("after the rewrite a holds %+v, b %+v", ba, bb)
	}
	if got, _ := p.b.Matches().Get(p.cb, p.sb, bid); got.Comment != "note A" {
		t.Errorf("the rewrite lost the note: %q", got.Comment)
	}
}

// b holds another match under the hash a's rewrite takes: refused, reported,
// and b's match stays as it was.
func scenarioMatchHashTaken(p *csPair) {
	t := p.t
	p.catchUp()
	mid, bid := p.syncedMatch("taken-1", 390)
	other := domain.Match{Player1Name: "C", Player2Name: "D", MatchLength: 1, MatchDate: time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC),
		FilePath: "o.xg", MatchHash: "taken-2", CanonicalHash: "taken-2-c"}
	if _, err := p.b.Matches().Save(p.cb, p.sb, &other); err != nil {
		t.Fatal(err)
	}
	before := readBody(t, p.b, p.cb, p.sb, bid)
	tick()
	h, _ := p.a.Matches().Get(p.ca, p.sa, mid)
	h.MatchHash, h.CanonicalHash = "taken-2", "taken-2-c"
	if err := p.a.Matches().ReplaceHeader(p.ca, p.sa, mid, h); err != nil {
		t.Fatal(err)
	}
	rep := p.pagedSend(1000)
	if len(rep.Conflicts) != 1 {
		t.Errorf("report: %+v, want the one conflict", rep)
	}
	if after := readBody(t, p.b, p.cb, p.sb, bid); !before.equal(after) {
		t.Errorf("a refused update changed b: %+v, was %+v", after, before)
	}
}

// b publishes a position; a, knowing nothing of it, re-analyses it later.
func scenarioEmptyPublicID(p *csPair) {
	t := p.t
	p.catchUp()
	pid := p.saved(400, 1)[0]
	p.pagedSend(1000)
	q := syncPos(400)
	bid, found, _ := p.b.Positions().Exists(p.cb, p.sb, storage.ZobristOf(&q))
	if !found {
		t.Fatal("no position in b")
	}
	tick()
	if err := p.b.(storage.PublicIDProvider).PublicIDs().SetPublicID(p.cb, p.sb, bid, "pub-1"); err != nil {
		t.Fatal(err)
	}
	tick()
	if err := p.a.Analyses().Save(p.ca, p.sa, pid, doublingAnalysis()); err != nil {
		t.Fatal(err)
	}
	if rep := p.pagedSend(1000); rep.Applied == 0 {
		t.Fatalf("the analysis did not travel: %+v", rep)
	}
	if got := publicIDOf(t, p.b, p.cb, p.sb, bid); got != "pub-1" {
		t.Errorf("public_id in b = %q, want pub-1", got)
	}
	if _, err := p.b.Analyses().Load(p.cb, p.sb, bid); err != nil {
		t.Errorf("the analysis did not arrive: %v", err)
	}
}

func scenarioReportKinds(p *csPair) {
	t := p.t
	p.catchUp()
	col, _ := p.a.Collections().Create(p.ca, p.sa, "Kinds", "")
	if rep := p.pagedSend(1000); rep.Added != 1 || rep.Modified != 0 || rep.Removed != 0 || rep.Applied != 1 {
		t.Errorf("create: %+v", rep)
	}
	tick()
	if err := p.a.Collections().Update(p.ca, p.sa, col, "Kinds 2", ""); err != nil {
		t.Fatal(err)
	}
	if rep := p.pagedSend(1000); rep.Added != 0 || rep.Modified != 1 || rep.Removed != 0 || rep.Applied != 1 {
		t.Errorf("update: %+v", rep)
	}
	tick()
	if err := p.a.Collections().Delete(p.ca, p.sa, col); err != nil {
		t.Fatal(err)
	}
	if rep := p.pagedSend(1000); rep.Added != 0 || rep.Modified != 0 || rep.Removed != 1 || rep.Applied != 1 {
		t.Errorf("delete: %+v", rep)
	}
}

// Last write wins on a note, and the one it overwrites stays in the trash,
// where the purge window keeps it (decision 2 of the interop card).
func scenarioNoteVersion(p *csPair) {
	t := p.t
	p.catchUp()
	pid := p.saved(410, 1)[0]
	cid, err := p.a.Comments().Add(p.ca, p.sa, pid, "note v1")
	if err != nil {
		t.Fatal(err)
	}
	p.pagedSend(1000)
	count := func() (n int, v1 bool) {
		list, err := p.b.Trash().List(p.cb, p.sb, domain.TrashComment, storage.ListOpts{Limit: 100})
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range list {
			n++
			v1 = v1 || e.Label == "note v1"
		}
		return n, v1
	}
	n0, _ := count()
	tick()
	if err := p.a.Comments().Update(p.ca, p.sa, cid, "note v2"); err != nil {
		t.Fatal(err)
	}
	p.pagedSend(1000)
	n1, v1 := count()
	if n1 != n0+1 || !v1 {
		t.Errorf("the overwritten note was not kept: %d entries (was %d), v1 %v", n1, n0, v1)
	}
	// A replay overwrites nothing and keeps nothing more.
	p.replay()
	if n2, _ := count(); n2 != n1 {
		t.Errorf("a replay kept a version: %d, was %d", n2, n1)
	}
}
