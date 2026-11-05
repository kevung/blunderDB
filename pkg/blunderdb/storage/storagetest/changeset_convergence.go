package storagetest

import (
	"context"
	"encoding/json"
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
// tournament and a match its sender attached to it.
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
