package storagetest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// CheckChangeSetCardTwinBeforeTheJournal: b holds a reviewed card from before the
// journal, with no entry, and a holds a virgin draw of the same question. The
// reviews stay, on both sides, whichever uuid sorts first and whichever side
// sends first. pair gives two libraries that have never met, and what forgets
// the journal entries of b's cards.
func CheckChangeSetCardTwinBeforeTheJournal(t *testing.T, pair func(t *testing.T) (*ChangeSetPair, func())) {
	derived := sqlshared.DeriveObjectUUID("anki_card", "before the journal", 1)
	for _, c := range []struct {
		name         string
		drawAFirst   bool   // else b draws first
		derivedOf    string // the library whose card bears the derived uuid
		backFirst    bool   // b sends first
		keptIsTheOld bool   // the uuid b holds sorts first
	}{
		{name: "DerivedHeldVirginArrives", derivedOf: "b", keptIsTheOld: true},
		{name: "DerivedHeldSendsFirst", derivedOf: "b", backFirst: true, keptIsTheOld: true},
		{name: "OlderUUIDHeldVirginArrives", keptIsTheOld: true},
		{name: "NewerUUIDHeldVirginArrives", drawAFirst: true},
		{name: "VirginUnderDerivedArrives", drawAFirst: true, derivedOf: "a"},
	} {
		t.Run(c.name, func(t *testing.T) {
			p, forget := pair(t)
			p.t = t
			q, q2 := syncPos(830), syncPos(830)
			pa, err := p.a.Positions().Save(p.ca, p.sa, &q)
			if err != nil {
				t.Fatal(err)
			}
			pb, err := p.b.Positions().Save(p.cb, p.sb, &q2)
			if err != nil {
				t.Fatal(err)
			}
			deck, err := p.a.Anki().CreateDeck(p.ca, p.sa, "Old", "", domain.AnkiSourceSearch, 0, "")
			if err != nil {
				t.Fatal(err)
			}
			p.send()
			bdeck := famIDMust(p, p.b, p.cb, p.sb, "anki_deck", famUUID(t, p.a, p.ca, p.sa, "anki_deck", deck))
			draws := []func() error{
				func() error { return p.b.Anki().SyncWithPositions(p.cb, p.sb, bdeck, []int64{pb}) },
				func() error { return p.a.Anki().SyncWithPositions(p.ca, p.sa, deck, []int64{pa}) },
			}
			if c.drawAFirst {
				draws[0], draws[1] = draws[1], draws[0]
			}
			for _, draw := range draws {
				if err := draw(); err != nil {
					t.Fatal(err)
				}
				time.Sleep(20 * time.Millisecond)
			}
			ac, _ := cardOf(p, p.a, p.ca, p.sa, deck, pa)
			bc, _ := cardOf(p, p.b, p.cb, p.sb, bdeck, pb)
			review(p, p.b, p.cb, p.sb, bc, 3)
			name := func(s storage.Storage, ctx context.Context, scope string, id int64) {
				if err := s.(storage.ObjectUUIDProvider).ObjectUUIDs().SetObjectUUID(ctx, scope, "anki_card", id, derived); err != nil {
					t.Fatal(err)
				}
			}
			switch c.derivedOf {
			case "a":
				name(p.a, p.ca, p.sa, ac)
			case "b":
				name(p.b, p.cb, p.sb, bc)
			}
			forget()
			ua, ub := famUUID(t, p.a, p.ca, p.sa, "anki_card", ac), famUUID(t, p.b, p.cb, p.sb, "anki_card", bc)
			if (ub < ua) != c.keptIsTheOld {
				t.Fatalf("setup: the uuids sort otherwise than the case says: a %s, b %s", ua, ub)
			}
			if _, err := p.b.(storage.ChangeJournalProvider).ChangeJournal().Entry(p.cb, p.sb, ub); !errors.Is(err, storage.ErrNotFound) {
				t.Fatalf("setup: b's card has a journal entry: %v", err)
			}
			tick()
			if c.backFirst {
				p.back()
				p.send()
			} else {
				p.send()
				if cb, _ := ankiCard(p, p.b, p.cb, p.sb, bdeck, bc); cb.Reps != 3 {
					t.Errorf("the virgin draw wrote over the reviews of b: reps = %d, want 3", cb.Reps)
				}
				p.back()
			}
			p.send()
			ca, _ := ankiCard(p, p.a, p.ca, p.sa, deck, ac)
			cb, _ := ankiCard(p, p.b, p.cb, p.sb, bdeck, bc)
			if ca.Reps != 3 || cb.Reps != 3 {
				t.Errorf("reps a=%d b=%d, want 3 on both", ca.Reps, cb.Reps)
			}
			want := min(ua, ub)
			if ga, gb := famUUID(t, p.a, p.ca, p.sa, "anki_card", ac), famUUID(t, p.b, p.cb, p.sb, "anki_card", bc); ga != want || gb != want {
				t.Errorf("uuids a=%s b=%s, want the older %s on both", ga, gb, want)
			}
			if rep := p.replay(); rep.Applied != 0 {
				t.Errorf("a replay from a applied %+v", rep)
			}
			if rep := p.swapped().replay(); rep.Applied != 0 {
				t.Errorf("a replay from b applied %+v", rep)
			}
		})
	}
}
