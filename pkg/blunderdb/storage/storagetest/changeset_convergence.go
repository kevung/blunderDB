package storagetest

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/changeset"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Each local write of a new family, over a held date ahead of the clock (set-back
// clock) or in the same ms: dated past it, and it reaches the peer.
func scenarioLocalWritesDatedPastHeld(p *csPair) {
	t := p.t
	p.catchUp()
	q := syncPos(700)
	pa, _ := p.a.Positions().Save(p.ca, p.sa, &q)
	tid, _ := p.a.Tournaments().Create(p.ca, p.sa, "Rev", "", "")
	deck, _ := p.a.Anki().CreateDeck(p.ca, p.sa, "Revdeck", "", domain.AnkiSourceSearch, 0, "")
	_ = p.a.Anki().SyncWithPositions(p.ca, p.sa, deck, []int64{pa})
	card, _ := cardOf(p, p.a, p.ca, p.sa, deck, pa)
	trid, err := p.a.Transcriptions().Save(p.ca, p.sa, &storage.Transcription{FormatVersion: "1", Label: "l0", Document: "{}"})
	if err != nil {
		t.Fatal(err)
	}
	type W struct {
		fam   string
		id    int64
		write func(i int) error
		check func(i int) bool
	}
	lbl := func(i int) string { return "v" + string(rune('a'+i)) }
	ws := []W{
		{"tournament", tid, func(i int) error { return p.a.Tournaments().Update(p.ca, p.sa, tid, "Rev"+lbl(i), "", "") }, func(i int) bool {
			id := famIDMust(p, p.b, p.cb, p.sb, "tournament", famUUID(t, p.a, p.ca, p.sa, "tournament", tid))
			g, _ := p.b.Tournaments().Get(p.cb, p.sb, id)
			return g.Name == "Rev"+lbl(i)
		}},
		{"tournament", tid, func(i int) error { return p.a.Tournaments().UpdateComment(p.ca, p.sa, tid, lbl(i)) }, func(i int) bool {
			id := famIDMust(p, p.b, p.cb, p.sb, "tournament", famUUID(t, p.a, p.ca, p.sa, "tournament", tid))
			g, _ := p.b.Tournaments().Get(p.cb, p.sb, id)
			return g.Comment == lbl(i)
		}},
		{"anki_deck", deck, func(i int) error { return p.a.Anki().UpdateDeck(p.ca, p.sa, deck, "Revd"+lbl(i), "") }, func(i int) bool {
			id := famIDMust(p, p.b, p.cb, p.sb, "anki_deck", famUUID(t, p.a, p.ca, p.sa, "anki_deck", deck))
			return deckOf(p, p.b, p.cb, p.sb, id).Name == "Revd"+lbl(i)
		}},
		{"anki_card", card, func(i int) error { _, e := p.a.Anki().ReviewCard(p.ca, p.sa, card, 3); return e }, func(i int) bool {
			uu := famUUID(t, p.a, p.ca, p.sa, "anki_card", card)
			ida := famIDMust(p, p.a, p.ca, p.sa, "anki_card", uu)
			idb := famIDMust(p, p.b, p.cb, p.sb, "anki_card", uu)
			ca, _ := p.a.(storage.AnkiStateProvider).AnkiStates().Card(p.ca, p.sa, ida)
			cb, _ := p.b.(storage.AnkiStateProvider).AnkiStates().Card(p.cb, p.sb, idb)
			return ca.Reps == cb.Reps
		}},
		{"anki_card", card, func(i int) error { return p.a.Anki().SetCardSuspended(p.ca, p.sa, card, i%2 == 0) }, func(i int) bool {
			uu := famUUID(t, p.a, p.ca, p.sa, "anki_card", card)
			idb := famIDMust(p, p.b, p.cb, p.sb, "anki_card", uu)
			cb, _ := p.b.(storage.AnkiStateProvider).AnkiStates().Card(p.cb, p.sb, idb)
			return cb.Suspended == (i%2 == 0)
		}},
		{"transcription", trid, func(i int) error {
			_, e := p.a.Transcriptions().Save(p.ca, p.sa, &storage.Transcription{ID: trid, FormatVersion: "1", Label: lbl(i), Document: "{}"})
			return e
		}, func(i int) bool {
			id := famIDMust(p, p.b, p.cb, p.sb, "transcription", famUUID(t, p.a, p.ca, p.sa, "transcription", trid))
			g, _ := p.b.Transcriptions().Get(p.cb, p.sb, id)
			return g.Label == lbl(i)
		}},
	}
	p.send()
	journal := p.a.(storage.ChangeJournalProvider).ChangeJournal()
	journalB := p.b.(storage.ChangeJournalProvider).ChangeJournal()
	for n, w := range ws {
		u := famUUID(t, p.a, p.ca, p.sa, w.fam, w.id)
		// same millisecond: two writes in a row, no pause, peer holding the first
		for i := range 4 {
			if err := w.write(i); err != nil {
				t.Fatal(err)
			}
			rep := p.send()
			if !w.check(i) {
				t.Errorf("#%d %s same-ms write %d did not win: %+v", n, w.fam, i, rep)
			}
		}
		// clock set back: both libraries hold a date 400 ms ahead (within slack)
		ahead := time.Now().UTC().Add(400 * time.Millisecond).Format(storage.ChangedAtLayout)
		_ = journal.Mark(p.ca, p.sa, w.fam, u, storage.ChangeUpdate, ahead)
		_ = journalB.Mark(p.cb, p.sb, w.fam, u, storage.ChangeUpdate, ahead)
		if err := w.write(5); err != nil {
			t.Fatal(err)
		}
		if got := p.entry(p.a, p.ca, p.sa, u).ChangedAt; got <= ahead {
			t.Errorf("#%d %s local write dated %s, held %s", n, w.fam, got, ahead)
		}
		if _, err := changeset.Export(p.ca, p.a, p.sa, p.cursorFrom, 1000); err != nil {
			t.Fatal(err)
		}
		rep := p.send()
		if !w.check(5) {
			t.Errorf("#%d %s older-than-held write lost at peer: %+v", n, w.fam, rep)
		}
		// held an hour ahead, on both: only the local write's stamp orders them
		far := time.Now().UTC().Add(time.Hour).Format(storage.ChangedAtLayout)
		_ = journal.Mark(p.ca, p.sa, w.fam, u, storage.ChangeUpdate, far)
		_ = journalB.Mark(p.cb, p.sb, w.fam, u, storage.ChangeUpdate, far)
		if err := w.write(6); err != nil {
			t.Fatal(err)
		}
		rep = p.send()
		if !w.check(6) {
			t.Errorf("#%d %s far-held write lost at peer: %+v", n, w.fam, rep)
		}
		time.Sleep(time.Millisecond)
	}
	// delete over a held date
	u := famUUID(t, p.a, p.ca, p.sa, "tournament", tid)
	far := time.Now().UTC().Add(time.Hour).Format(storage.ChangedAtLayout)
	_ = journal.Mark(p.ca, p.sa, "tournament", u, storage.ChangeUpdate, far)
	_ = journalB.Mark(p.cb, p.sb, "tournament", u, storage.ChangeUpdate, far)
	_ = p.a.Tournaments().Delete(p.ca, p.sa, tid)
	rep := p.send()
	if _, ok := famID(p.b, p.cb, p.sb, "tournament", u); ok {
		t.Errorf("delete over held date lost at peer: %+v", rep)
	}
}

func exportedPayload(p *csPair, s storage.Storage, ctx context.Context, scope string, u string) string {
	set, err := changeset.Export(ctx, s, scope, storage.Cursor{}, 1_000_000)
	if err != nil {
		p.t.Fatal(err)
	}
	for _, r := range set.Records {
		if r.UUID == u {
			var v any
			_ = json.Unmarshal(r.Payload, &v)
			b, _ := json.Marshal(v)
			return r.ChangedAt + " " + string(b)
		}
	}
	return ""
}

// Replaying one file: import F twice, the second must do nothing. F holds a
// tournament and an attached match; the attach bumps the receiver's updated_at.
func scenarioTournamentReplay(p *csPair) {
	t := p.t
	for round := range 4 {
		p.catchUp()
		tid, _ := p.a.Tournaments().Create(p.ca, p.sa, "RevR"+string(rune('a'+round)), "", "")
		h := "revr-" + string(rune('a'+round))
		mid, _ := p.a.Matches().Save(p.ca, p.sa, &domain.Match{Player1Name: "P", Player2Name: "Q", MatchLength: 3, FilePath: h + ".xg", MatchHash: h, CanonicalHash: h + "c"})
		_ = p.a.Tournaments().AddMatch(p.ca, p.sa, tid, mid)
		tu := famUUID(t, p.a, p.ca, p.sa, "tournament", tid)
		set, err := changeset.Export(p.ca, p.a, p.sa, p.cursorFrom, 1000)
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(1100 * time.Millisecond) // datetime('now') has second grain
		if _, err := changeset.Import(p.cb, p.b, p.sb, set); err != nil {
			t.Fatal(err)
		}
		r2, err := changeset.Import(p.cb, p.b, p.sb, set)
		if err != nil {
			t.Fatal(err)
		}
		pa, pb := exportedPayload(p, p.a, p.ca, p.sa, tu), exportedPayload(p, p.b, p.cb, p.sb, tu)
		if r2.Applied != 0 {
			t.Errorf("round %d: replay of the same file applied %d", round, r2.Applied)
		}
		if pa != pb {
			t.Errorf("round %d: tournament differs after one-way sync + replay", round)
		}
	}
}

// Both attach a different match to one tournament, then exchange incrementally.
func scenarioTournamentOrderExchange(p *csPair) {
	t := p.t
	p.catchUp()
	tid, _ := p.a.Tournaments().Create(p.ca, p.sa, "RevO", "", "")
	tu := famUUID(t, p.a, p.ca, p.sa, "tournament", tid)
	mk := func(s storage.Storage, c context.Context, sc, h string) int64 {
		id, err := s.Matches().Save(c, sc, &domain.Match{Player1Name: "P", Player2Name: "Q", MatchLength: 3, FilePath: h + ".xg", MatchHash: h, CanonicalHash: h + "c"})
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	p.send()
	btid := famIDMust(p, p.b, p.cb, p.sb, "tournament", tu)
	sw := p.swapped()
	sw.t = t
	sw.catchUp()
	ma := mk(p.a, p.ca, p.sa, "revo-a")
	tick()
	mb := mk(p.b, p.cb, p.sb, "revo-b")
	_ = p.a.Tournaments().AddMatch(p.ca, p.sa, tid, ma)
	tick()
	_ = p.b.Tournaments().AddMatch(p.cb, p.sb, btid, mb)
	p.send()
	sw.send()
	order := func(s storage.Storage, c context.Context, sc string, id int64) (o []string) {
		for m, err := range s.Tournaments().Matches(c, sc, id) {
			if err != nil {
				t.Fatal(err)
			}
			o = append(o, m.MatchHash)
		}
		return
	}
	p.send()
	sw.send()
	oa, ob := order(p.a, p.ca, p.sa, tid), order(p.b, p.cb, p.sb, btid)
	if len(oa) != 2 || !slices.Equal(oa, ob) {
		t.Errorf("order diverges for good on an incremental exchange: a=%v b=%v", oa, ob)
	}
}

// Two draws of one card; a suspends after both draws, no review anywhere. It is
// the last write of the object.
func scenarioAnkiTwinLastWrite(p *csPair) {
	t := p.t
	for round := range 6 {
		p.catchUp()
		q := syncPos(720 + round)
		pa, _ := p.a.Positions().Save(p.ca, p.sa, &q)
		q2 := syncPos(720 + round)
		pb, _ := p.b.Positions().Save(p.cb, p.sb, &q2)
		deck, _ := p.a.Anki().CreateDeck(p.ca, p.sa, "RevT"+string(rune('a'+round)), "", domain.AnkiSourceSearch, 0, "")
		du := famUUID(t, p.a, p.ca, p.sa, "anki_deck", deck)
		p.send()
		bdeck := famIDMust(p, p.b, p.cb, p.sb, "anki_deck", du)
		// draws: one side first, by round
		if round%2 == 0 {
			_ = p.a.Anki().SyncWithPositions(p.ca, p.sa, deck, []int64{pa})
			time.Sleep(1100 * time.Millisecond)
			_ = p.b.Anki().SyncWithPositions(p.cb, p.sb, bdeck, []int64{pb})
		} else {
			_ = p.b.Anki().SyncWithPositions(p.cb, p.sb, bdeck, []int64{pb})
			time.Sleep(1100 * time.Millisecond)
			_ = p.a.Anki().SyncWithPositions(p.ca, p.sa, deck, []int64{pa})
		}
		ac, _ := cardOf(p, p.a, p.ca, p.sa, deck, pa)
		bc, _ := cardOf(p, p.b, p.cb, p.sb, bdeck, pb)
		tick()
		if err := p.a.Anki().SetCardSuspended(p.ca, p.sa, ac, true); err != nil {
			t.Fatal(err)
		}
		p.send()
		p.back()
		p.send()
		ca, _ := ankiCard(p, p.a, p.ca, p.sa, deck, ac)
		cb, _ := ankiCard(p, p.b, p.cb, p.sb, bdeck, bc)
		if !ca.Suspended || !cb.Suspended {
			t.Errorf("round %d: last write (suspend on a) lost: a=%v b=%v", round, ca.Suspended, cb.Suspended)
		}
	}
}

// Paging at 1 over every new family gives the same result as a whole export.
func scenarioNewFamiliesPagedAtOne(p *csPair) {
	t := p.t
	p.catchUp()
	q := syncPos(760)
	pa, _ := p.a.Positions().Save(p.ca, p.sa, &q)
	tid, _ := p.a.Tournaments().Create(p.ca, p.sa, "RevP", "", "")
	mid, _ := p.a.Matches().Save(p.ca, p.sa, &domain.Match{Player1Name: "P", Player2Name: "Q", MatchLength: 3, FilePath: "revp.xg", MatchHash: "revp", CanonicalHash: "revpc"})
	_ = p.a.Tournaments().AddMatch(p.ca, p.sa, tid, mid)
	trid, _ := p.a.Transcriptions().Save(p.ca, p.sa, &storage.Transcription{FormatVersion: "1", MatchID: mid, Label: "revp", Document: "{}"})
	deck, _ := p.a.Anki().CreateDeck(p.ca, p.sa, "RevP", "", domain.AnkiSourceSearch, 0, "")
	_ = p.a.Anki().SyncWithPositions(p.ca, p.sa, deck, []int64{pa})
	rep := p.pagedSend(1)
	if len(rep.Conflicts) != 0 || len(rep.Missing) != 0 {
		t.Errorf("paged at 1: %+v", rep)
	}
	for _, f := range []struct {
		fam string
		id  int64
	}{{"tournament", tid}, {"transcription", trid}, {"anki_deck", deck}} {
		famIDMust(p, p.b, p.cb, p.sb, f.fam, famUUID(t, p.a, p.ca, p.sa, f.fam, f.id))
	}
	bt := famIDMust(p, p.b, p.cb, p.sb, "transcription", famUUID(t, p.a, p.ca, p.sa, "transcription", trid))
	if g, _ := p.b.Transcriptions().Get(p.cb, p.sb, bt); g.MatchID == 0 {
		t.Error("transcription lost its match under paging")
	}
}

// One library draws first, the other draws and reviews, the first then suspends:
// the later gesture is the suspension, the further reviews are the other's. Both
// end on one state holding the two, whichever uuid is the older.
func scenarioAnkiTwinConverges(p *csPair) {
	t := p.t
	for round := range 2 {
		p.catchUp()
		q := syncPos(780 + round)
		pa, _ := p.a.Positions().Save(p.ca, p.sa, &q)
		q2 := syncPos(780 + round)
		pb, _ := p.b.Positions().Save(p.cb, p.sb, &q2)
		deck, _ := p.a.Anki().CreateDeck(p.ca, p.sa, "RevD"+string(rune('a'+round)), "", domain.AnkiSourceSearch, 0, "")
		du := famUUID(t, p.a, p.ca, p.sa, "anki_deck", deck)
		p.send()
		bdeck := famIDMust(p, p.b, p.cb, p.sb, "anki_deck", du)
		first, second := func() { _ = p.a.Anki().SyncWithPositions(p.ca, p.sa, deck, []int64{pa}) },
			func() { _ = p.b.Anki().SyncWithPositions(p.cb, p.sb, bdeck, []int64{pb}) }
		if round == 1 {
			first, second = second, first
		}
		first()
		time.Sleep(20 * time.Millisecond)
		second()
		ac, _ := cardOf(p, p.a, p.ca, p.sa, deck, pa)
		bc, _ := cardOf(p, p.b, p.cb, p.sb, bdeck, pb)
		review(p, p.b, p.cb, p.sb, bc, 3)
		tick()
		if err := p.a.Anki().SetCardSuspended(p.ca, p.sa, ac, true); err != nil {
			t.Fatal(err)
		}
		for range 3 {
			p.send()
			p.back()
		}
		ca, _ := ankiCard(p, p.a, p.ca, p.sa, deck, ac)
		cb, _ := ankiCard(p, p.b, p.cb, p.sb, bdeck, bc)
		if ca.Reps != 3 || cb.Reps != 3 {
			t.Errorf("round %d: a review was lost: reps a=%d b=%d, want 3", round, ca.Reps, cb.Reps)
		}
		if !ca.Suspended || !cb.Suspended {
			t.Errorf("round %d: the suspension, the last gesture, was lost: a=%v b=%v", round, ca.Suspended, cb.Suspended)
		}
	}
}

// An old file replayed after a match left the tournament.
func scenarioTournamentOldFile(p *csPair) {
	t := p.t
	for round := range 6 {
		p.catchUp()
		r := string(rune('a' + round))
		tid, _ := p.a.Tournaments().Create(p.ca, p.sa, "RevF"+r, "", "")
		tu := famUUID(t, p.a, p.ca, p.sa, "tournament", tid)
		var m [3]int64
		for i := range m {
			h := "revf-" + r + string(rune('0'+i))
			m[i], _ = p.a.Matches().Save(p.ca, p.sa, &domain.Match{Player1Name: "P", Player2Name: "Q", MatchLength: 3, FilePath: h + ".xg", MatchHash: h, CanonicalHash: h + "c"})
			_ = p.a.Tournaments().AddMatch(p.ca, p.sa, tid, m[i])
		}
		f0, err := changeset.Export(p.ca, p.a, p.sa, p.cursorFrom, 1000)
		if err != nil {
			t.Fatal(err)
		}
		p.send()
		btid := famIDMust(p, p.b, p.cb, p.sb, "tournament", tu)
		tick()
		_ = p.a.Tournaments().RemoveMatch(p.ca, p.sa, m[0])
		p.send()
		order := func(s storage.Storage, c context.Context, sc string, id int64) (o []string) {
			for m, err := range s.Tournaments().Matches(c, sc, id) {
				if err != nil {
					t.Fatal(err)
				}
				o = append(o, m.MatchHash)
			}
			return
		}
		if _, err := changeset.Import(p.cb, p.b, p.sb, f0); err != nil {
			t.Fatal(err)
		}
		oa, ob := order(p.a, p.ca, p.sa, tid), order(p.b, p.cb, p.sb, btid)
		if !slices.Equal(oa, ob) {
			t.Errorf("round %d: replaying an old file reordered the tournament: a=%v b=%v", round, oa, ob)
		}
	}
}

// Files crossing: each exports before seeing the other.
func scenarioTournamentCrossedFiles(p *csPair) {
	t := p.t
	p.catchUp()
	tid, _ := p.a.Tournaments().Create(p.ca, p.sa, "RevC", "", "")
	tu := famUUID(t, p.a, p.ca, p.sa, "tournament", tid)
	p.send()
	btid := famIDMust(p, p.b, p.cb, p.sb, "tournament", tu)
	sw := p.swapped()
	sw.t = t
	sw.catchUp()
	mk := func(s storage.Storage, c context.Context, sc, h string) int64 {
		id, err := s.Matches().Save(c, sc, &domain.Match{Player1Name: "P", Player2Name: "Q", MatchLength: 3, FilePath: h + ".xg", MatchHash: h, CanonicalHash: h + "c"})
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	ma := mk(p.a, p.ca, p.sa, "revc-a")
	mb := mk(p.b, p.cb, p.sb, "revc-b")
	_ = p.a.Tournaments().AddMatch(p.ca, p.sa, tid, ma)
	_ = p.b.Tournaments().AddMatch(p.cb, p.sb, btid, mb)
	fa, _ := changeset.Export(p.ca, p.a, p.sa, p.cursorFrom, 1000)
	fb, _ := changeset.Export(sw.ca, sw.a, sw.sa, sw.cursorFrom, 1000)
	p.cursorFrom, sw.cursorFrom = fa.Cursor, fb.Cursor
	_, _ = changeset.Import(p.cb, p.b, p.sb, fa)
	_, _ = changeset.Import(p.ca, p.a, p.sa, fb)
	order := func(s storage.Storage, c context.Context, sc string, id int64) (o []string) {
		for m, err := range s.Tournaments().Matches(c, sc, id) {
			if err != nil {
				t.Fatal(err)
			}
			o = append(o, m.MatchHash)
		}
		return
	}
	for range 3 {
		p.send()
		sw.send()
	}
	oa, ob := order(p.a, p.ca, p.sa, tid), order(p.b, p.cb, p.sb, btid)
	if len(oa) != 2 || !slices.Equal(oa, ob) {
		t.Errorf("order diverges: a=%v b=%v", oa, ob)
	}
}

// tournamentOfMatch is the uuid of the tournament a match belongs to, "" for none.
func tournamentOfMatch(p *csPair, s storage.Storage, ctx context.Context, scope string, match int64) string {
	p.t.Helper()
	got, err := s.Tournaments().TournamentOf(ctx, scope, match)
	if errors.Is(err, storage.ErrNotFound) {
		return ""
	}
	if err != nil {
		p.t.Fatal(err)
	}
	return famUUID(p.t, s, ctx, scope, "tournament", got.ID)
}

// heldBeforeTheAttach is a match both libraries hold under one uuid and date,
// attached on a by a gesture the journal of that schema did not date, and the
// tournament it joined.
func heldBeforeTheAttach(p *csPair, tag string) (ma, mb int64, mu, tu string) {
	t := p.t
	p.catchUp()
	tid, err := p.a.Tournaments().Create(p.ca, p.sa, "RevL"+tag, "", "")
	if err != nil {
		t.Fatal(err)
	}
	ma, err = p.a.Matches().Save(p.ca, p.sa, &domain.Match{Player1Name: "P", Player2Name: "Q", MatchLength: 3, FilePath: tag + ".xg", MatchHash: "revl-" + tag, CanonicalHash: "revl-" + tag + "c"})
	if err != nil {
		t.Fatal(err)
	}
	mu, tu = famUUID(t, p.a, p.ca, p.sa, "match", ma), famUUID(t, p.a, p.ca, p.sa, "tournament", tid)
	p.send()
	mb = famIDMust(p, p.b, p.cb, p.sb, "match", mu)
	at := p.entry(p.a, p.ca, p.sa, mu).ChangedAt
	if err := p.a.Tournaments().AddMatch(p.ca, p.sa, tid, ma); err != nil {
		t.Fatal(err)
	}
	p.markAs(p.a, p.ca, p.sa, "match", mu, at)
	return ma, mb, mu, tu
}

// A match attached under the older schema, undated: the exchange keeps it, on
// both sides, whatever the digests of the two states say.
func scenarioLegacyAttachSurvives(p *csPair) {
	t := p.t
	for round := range 6 {
		ma, mb, _, tu := heldBeforeTheAttach(p, "s"+string(rune('a'+round)))
		for range 2 {
			p.send()
			p.back()
		}
		if got := tournamentOfMatch(p, p.a, p.ca, p.sa, ma); got != tu {
			t.Errorf("round %d: a lost the attachment: %q", round, got)
		}
		if got := tournamentOfMatch(p, p.b, p.cb, p.sb, mb); got != tu {
			t.Errorf("round %d: b did not get the attachment: %q", round, got)
		}
	}
}

// The other library edits the match afterwards, without having heard of the
// tournament: its later record says nothing about one, and detaches nothing.
func scenarioLegacyAttachSurvivesLaterEdit(p *csPair) {
	t := p.t
	for round := range 4 {
		ma, mb, _, tu := heldBeforeTheAttach(p, "e"+string(rune('a'+round)))
		tick()
		if err := p.b.Matches().UpdateComment(p.cb, p.sb, mb, "edited on b"); err != nil {
			t.Fatal(err)
		}
		for range 2 {
			p.back()
			p.send()
		}
		if got := tournamentOfMatch(p, p.a, p.ca, p.sa, ma); got != tu {
			t.Errorf("round %d: b's later edit detached the match on a: %q", round, got)
		}
		if got := tournamentOfMatch(p, p.b, p.cb, p.sb, mb); got != tu {
			t.Errorf("round %d: b did not get the attachment: %q", round, got)
		}
		for _, m := range []struct {
			s     storage.Storage
			c     context.Context
			scope string
			id    int64
		}{{p.a, p.ca, p.sa, ma}, {p.b, p.cb, p.sb, mb}} {
			if got, err := m.s.Matches().Get(m.c, m.scope, m.id); err != nil || got.Comment != "edited on b" {
				t.Errorf("round %d: the edit did not reach both: %+v, %v", round, got, err)
			}
		}
	}
}

// A detachment made since is a gesture like any other: the older attachment
// does not come back, not even by the file that carried it.
func scenarioExplicitDetachStands(p *csPair) {
	t := p.t
	ma, mb, _, _ := heldBeforeTheAttach(p, "detach")
	full, err := changeset.Export(p.ca, p.a, p.sa, storage.Cursor{}, 1_000_000)
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		p.send()
		p.back()
	}
	tick()
	if err := p.b.Tournaments().RemoveMatch(p.cb, p.sb, mb); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		p.back()
		p.send()
	}
	if got := tournamentOfMatch(p, p.a, p.ca, p.sa, ma); got != "" {
		t.Errorf("the detachment did not reach a: %q", got)
	}
	if _, err := changeset.Import(p.cb, p.b, p.sb, full); err != nil {
		t.Fatal(err)
	}
	if got := tournamentOfMatch(p, p.b, p.cb, p.sb, mb); got != "" {
		t.Errorf("an old file undid the detachment on b: %q", got)
	}
}
