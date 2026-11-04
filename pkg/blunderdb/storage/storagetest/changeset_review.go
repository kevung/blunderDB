// Scenarios of the changeset that a review measured red: what a library holds
// for itself must not travel as an edit, and a review of a card must not be
// lost to a mere draw.
package storagetest

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/changeset"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func checkChangeSetReview(t *testing.T, pair func() *csPair) {
	t.Run("AnkiSearchSourceRowIDs", func(t *testing.T) { p := pair(); p.t = t; scenarioAnkiSearchRowIDs(p) })
	t.Run("AnkiSyncIsNoEdit", func(t *testing.T) { p := pair(); p.t = t; scenarioAnkiSyncIsNoEdit(p) })
	t.Run("AnkiTwinKeepsTheReview", func(t *testing.T) { p := pair(); p.t = t; scenarioAnkiTwinKeepsReview(p) })
	t.Run("AnkiTieKeepsTheReview", func(t *testing.T) { p := pair(); p.t = t; scenarioAnkiTieKeepsReview(p) })
	t.Run("CursorBeforeTheFamilies", func(t *testing.T) { p := pair(); p.t = t; scenarioCursorBeforeFamilies(p) })
	t.Run("TournamentMembership", func(t *testing.T) { p := pair(); p.t = t; scenarioTournamentMembership(p) })
	for _, s := range []struct {
		name string
		run  func(*csPair)
	}{
		{"LocalWritesDatedPastHeld", scenarioLocalWritesDatedPastHeld},
		{"NewFamiliesPagedAtOne", scenarioNewFamiliesPagedAtOne},
		{"AnkiTwinConverges", scenarioAnkiTwinConverges},
		{"AnkiTwinLastWrite", scenarioAnkiTwinLastWrite},
		{"TournamentReplay", scenarioTournamentReplay},
		{"TournamentOrderExchange", scenarioTournamentOrderExchange},
		{"TournamentOldFile", scenarioTournamentOldFile},
		{"TournamentCrossedFiles", scenarioTournamentCrossedFiles},
		{"LegacyAttachSurvives", scenarioLegacyAttachSurvives},
		{"LegacyAttachSurvivesLaterEdit", scenarioLegacyAttachSurvivesLaterEdit},
		{"ExplicitDetachStands", scenarioExplicitDetachStands},
	} {
		t.Run(s.name, func(t *testing.T) { p := pair(); p.t = t; s.run(p) })
	}
}

// back sends b's changes, from 0, to a.
func (p *csPair) back() changeset.Report {
	p.t.Helper()
	set, err := changeset.Export(p.cb, p.b, p.sb, storage.Cursor{}, 1_000_000)
	if err != nil {
		p.t.Fatal(err)
	}
	rep, err := changeset.Import(p.ca, p.a, p.sa, set)
	if err != nil {
		p.t.Fatal(err)
	}
	return rep
}

// A search source that names rows by their numbering here (a match, a
// tournament, a position) must not reach a library where those numbers are
// other rows.
func scenarioAnkiSearchRowIDs(p *csPair) {
	t := p.t
	p.catchUp()
	src := `{"command":"ma7 tn3 like5","ids":[],"payload":{"matchIDsFilter":"7","tournamentIDsFilter":"3"}}`
	deck, err := p.a.Anki().CreateDeck(p.ca, p.sa, "Rows", "", domain.AnkiSourceSearch, 0, src)
	if err != nil {
		t.Fatal(err)
	}
	kept, _ := p.a.Anki().CreateDeck(p.ca, p.sa, "Kept", "", domain.AnkiSourceSearch, 0, `{"command":"x","ids":[]}`)
	p.send()
	got := deckOf(p, p.b, p.cb, p.sb, famIDMust(p, p.b, p.cb, p.sb, "anki_deck", famUUID(t, p.a, p.ca, p.sa, "anki_deck", deck))).SourceCommand
	for _, bad := range []string{"ma7", "tn3", "like5", "matchIDsFilter", "tournamentIDsFilter"} {
		if strings.Contains(got, bad) {
			t.Errorf("the source reached the peer with %q: %s", bad, got)
		}
	}
	got = deckOf(p, p.b, p.cb, p.sb, famIDMust(p, p.b, p.cb, p.sb, "anki_deck", famUUID(t, p.a, p.ca, p.sa, "anki_deck", kept))).SourceCommand
	if !strings.Contains(got, `"command":"x"`) {
		t.Errorf("a source that names no row lost its command: %s", got)
	}
}

// Synchronising a deck against its source touches its date and nothing it
// carries: a peer's rename must stand.
func scenarioAnkiSyncIsNoEdit(p *csPair) {
	t := p.t
	p.catchUp()
	q := syncPos(630)
	pa, _ := p.a.Positions().Save(p.ca, p.sa, &q)
	deck, _ := p.a.Anki().CreateDeck(p.ca, p.sa, "Syncing", "", domain.AnkiSourceSearch, 0, "")
	if err := p.a.Anki().SyncWithPositions(p.ca, p.sa, deck, []int64{pa}); err != nil {
		t.Fatal(err)
	}
	du := famUUID(t, p.a, p.ca, p.sa, "anki_deck", deck)
	p.send()
	bdeck := famIDMust(p, p.b, p.cb, p.sb, "anki_deck", du)
	tick()
	if err := p.b.Anki().UpdateDeck(p.cb, p.sb, bdeck, "Renamed by b", ""); err != nil {
		t.Fatal(err)
	}
	tick()
	before := p.entry(p.a, p.ca, p.sa, du)
	if err := p.a.Anki().SyncWithPositions(p.ca, p.sa, deck, []int64{pa}); err != nil {
		t.Fatal(err)
	}
	if after := p.entry(p.a, p.ca, p.sa, du); after.ChangedAt != before.ChangedAt {
		t.Errorf("a synchronisation dated the deck: %s -> %s", before.ChangedAt, after.ChangedAt)
	}
	if rep := p.send(); rep.Applied != 0 {
		t.Errorf("a synchronisation travelled: %+v", rep)
	}
	if d := deckOf(p, p.b, p.cb, p.sb, bdeck); d.Name != "Renamed by b" {
		t.Errorf("the peer's rename was undone: %q", d.Name)
	}
}

func review(p *csPair, s storage.Storage, ctx context.Context, scope string, id int64, times int) {
	p.t.Helper()
	for range times {
		if _, err := s.Anki().ReviewCard(ctx, scope, id, 3); err != nil {
			p.t.Fatal(err)
		}
	}
}

// Two libraries that drew the same card before they met: the one reviewed
// three times keeps its reviews, whichever drew the card later.
func scenarioAnkiTwinKeepsReview(p *csPair) {
	t := p.t
	p.catchUp()
	q := syncPos(640)
	pa, _ := p.a.Positions().Save(p.ca, p.sa, &q)
	q2 := syncPos(640)
	pb, _ := p.b.Positions().Save(p.cb, p.sb, &q2)
	deck, _ := p.a.Anki().CreateDeck(p.ca, p.sa, "Twins", "", domain.AnkiSourceSearch, 0, "")
	du := famUUID(t, p.a, p.ca, p.sa, "anki_deck", deck)
	p.send()
	bdeck := famIDMust(p, p.b, p.cb, p.sb, "anki_deck", du)
	tick()
	if err := p.a.Anki().SyncWithPositions(p.ca, p.sa, deck, []int64{pa}); err != nil {
		t.Fatal(err)
	}
	ac, _ := cardOf(p, p.a, p.ca, p.sa, deck, pa)
	review(p, p.a, p.ca, p.sa, ac, 3)
	tick()
	if err := p.b.Anki().SyncWithPositions(p.cb, p.sb, bdeck, []int64{pb}); err != nil {
		t.Fatal(err)
	}
	bc, _ := cardOf(p, p.b, p.cb, p.sb, bdeck, pb)
	p.send()
	p.back()
	p.send()
	if c, _ := ankiCard(p, p.a, p.ca, p.sa, deck, ac); c.Reps != 3 {
		t.Errorf("a: reps = %d after the exchange, want 3", c.Reps)
	}
	if c, _ := ankiCard(p, p.b, p.cb, p.sb, bdeck, bc); c.Reps != 3 {
		t.Errorf("b: reps = %d after the exchange, want 3", c.Reps)
	}
}

// Both libraries hold the same cards under the same uuids, dated alike (they
// were drawn before the journal): whichever side reviewed a card keeps it, on
// both sides, and no draw of a digest decides.
func scenarioAnkiTieKeepsReview(p *csPair) {
	t := p.t
	p.catchUp()
	const n = 8
	var pos [n]int64
	for i := range pos {
		q := syncPos(650 + i)
		pos[i], _ = p.a.Positions().Save(p.ca, p.sa, &q)
	}
	deck, _ := p.a.Anki().CreateDeck(p.ca, p.sa, "Tie", "", domain.AnkiSourceSearch, 0, "")
	if err := p.a.Anki().SyncWithPositions(p.ca, p.sa, deck, pos[:]); err != nil {
		t.Fatal(err)
	}
	du := famUUID(t, p.a, p.ca, p.sa, "anki_deck", deck)
	p.send()
	bdeck := famIDMust(p, p.b, p.cb, p.sb, "anki_deck", du)
	tick()
	var au, bu [n]int64
	for i := range pos {
		au[i], _ = cardOf(p, p.a, p.ca, p.sa, deck, pos[i])
		bpos := p.onB(pos[i])
		bu[i], _ = cardOf(p, p.b, p.cb, p.sb, bdeck, bpos)
	}
	for i := range pos {
		if i%2 == 0 {
			review(p, p.a, p.ca, p.sa, au[i], 1)
		} else {
			review(p, p.b, p.cb, p.sb, bu[i], 1)
		}
	}
	// The same date on both sides for every card: a tie to the millisecond.
	for i := range pos {
		uu := famUUID(t, p.a, p.ca, p.sa, "anki_card", au[i])
		at := max(p.entry(p.a, p.ca, p.sa, uu).ChangedAt, p.entry(p.b, p.cb, p.sb, uu).ChangedAt)
		p.markAs(p.a, p.ca, p.sa, "anki_card", uu, at)
		p.markAs(p.b, p.cb, p.sb, "anki_card", uu, at)
	}
	p.back()
	p.send()
	for i := range pos {
		ca, _ := ankiCard(p, p.a, p.ca, p.sa, deck, au[i])
		cb, _ := ankiCard(p, p.b, p.cb, p.sb, bdeck, bu[i])
		if ca.Reps != 1 || cb.Reps != 1 {
			t.Errorf("card %d: reps a=%d b=%d, want 1 on both", i, ca.Reps, cb.Reps)
		}
	}
}

// A match belongs to a tournament here and there: the link travels with the
// match, in its place, and a note on the tournament is not dated by it.
func scenarioTournamentMembership(p *csPair) {
	t := p.t
	p.catchUp()
	tid, _ := p.a.Tournaments().Create(p.ca, p.sa, "Membership", "", "")
	tu := famUUID(t, p.a, p.ca, p.sa, "tournament", tid)
	var mids [2]int64
	for i := range mids {
		h := "mem-" + string(rune('a'+i))
		mids[i], _ = p.a.Matches().Save(p.ca, p.sa, &domain.Match{Player1Name: "P", Player2Name: "Q", MatchLength: 3, FilePath: h + ".xg", MatchHash: h, CanonicalHash: h + "c"})
	}
	p.send()
	btid := famIDMust(p, p.b, p.cb, p.sb, "tournament", tu)
	tick()
	if err := p.b.Tournaments().UpdateComment(p.cb, p.sb, btid, "note of b"); err != nil {
		t.Fatal(err)
	}
	tick()
	before := p.entry(p.a, p.ca, p.sa, tu)
	for _, m := range mids {
		if err := p.a.Tournaments().AddMatch(p.ca, p.sa, tid, m); err != nil {
			t.Fatal(err)
		}
	}
	if after := p.entry(p.a, p.ca, p.sa, tu); after.ChangedAt != before.ChangedAt {
		t.Errorf("attaching a match dated the tournament: %s -> %s", before.ChangedAt, after.ChangedAt)
	}
	p.send()
	if g, _ := p.b.Tournaments().Get(p.cb, p.sb, btid); g.Comment != "note of b" {
		t.Errorf("attaching a match undid the peer's note: %q", g.Comment)
	}
	var order []string
	for m, err := range p.b.Tournaments().Matches(p.cb, p.sb, btid) {
		if err != nil {
			t.Fatal(err)
		}
		order = append(order, m.MatchHash)
	}
	if len(order) != 2 || order[0] != "mem-a" || order[1] != "mem-b" {
		t.Fatalf("the matches of the tournament in b: %v, want [mem-a mem-b]", order)
	}
	// The order moves, then a match leaves.
	tick()
	if err := p.a.Tournaments().ReorderMatches(p.ca, p.sa, tid, []int64{mids[1], mids[0]}); err != nil {
		t.Fatal(err)
	}
	p.send()
	order = order[:0]
	for m, err := range p.b.Tournaments().Matches(p.cb, p.sb, btid) {
		if err != nil {
			t.Fatal(err)
		}
		order = append(order, m.MatchHash)
	}
	if len(order) != 2 || order[0] != "mem-b" {
		t.Errorf("the new order did not travel: %v", order)
	}
	tick()
	if err := p.a.Tournaments().RemoveMatch(p.ca, p.sa, mids[0]); err != nil {
		t.Fatal(err)
	}
	p.send()
	bm := famIDMust(p, p.b, p.cb, p.sb, "match", famUUID(t, p.a, p.ca, p.sa, "match", mids[0]))
	if _, err := p.b.Tournaments().TournamentOf(p.cb, p.sb, bm); err == nil {
		t.Error("the match left the tournament in a and stayed in b")
	}
}

func (p *csPair) markAs(s storage.Storage, ctx context.Context, scope, family, uuid, at string) {
	p.t.Helper()
	if err := s.(storage.ChangeJournalProvider).ChangeJournal().Mark(ctx, scope, family, uuid, storage.ChangeUpdate, at); err != nil {
		p.t.Fatal(err)
	}
}

// A cursor taken when fewer families were carried would give nothing for the
// others: it is refused, where a cursor of now goes on.
func scenarioCursorBeforeFamilies(p *csPair) {
	t := p.t
	p.catchUp()
	if _, err := p.a.Tournaments().Create(p.ca, p.sa, "Cursor", "", ""); err != nil {
		t.Fatal(err)
	}
	set, err := changeset.Export(p.ca, p.a, p.sa, p.cursorFrom, 1000)
	if err != nil || len(set.Records) == 0 {
		t.Fatalf("export from the current cursor: %d records, %v", len(set.Records), err)
	}
	old := set.Cursor
	old.Carried = ""
	if got, err := changeset.Export(p.ca, p.a, p.sa, old, 1000); !errors.Is(err, storage.ErrCursorStale) {
		t.Errorf("a cursor of before the families: %d records, %v, want ErrCursorStale", len(got.Records), err)
	}
}
