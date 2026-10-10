package duel

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// TestCombinedSeedPinned: the stated derivation of the combined seed, pinned —
// a change to it changes the dice of every Duel that has one — and what it
// depends on: each contribution, their order, and where one ends.
func TestCombinedSeedPinned(t *testing.T) {
	seed, _ := newSeed(fixedRand(7))
	got, err := CombinedSeed(seed, [2]string{"alice", "bob"})
	if err != nil {
		t.Fatal(err)
	}
	// Computed outside Go, from the formula as dice.go states it.
	if want := "2eecb5d3a779f3f30683153b0ee83099a72c2052b1fcd1365ab5ec41478f796d"; got != want {
		t.Errorf("CombinedSeed(seed 0x07…, alice, bob) = %s; the derivation changed", got)
	}
	for _, other := range [][2]string{{"bob", "alice"}, {"alic", "ebob"}, {"alice", ""}, {"", "alicebob"}} {
		if o, _ := CombinedSeed(seed, other); o == got {
			t.Errorf("contributions %q give the same seed as %q", other, [2]string{"alice", "bob"})
		}
	}
	if _, err := CombinedSeed("not hex", [2]string{"a", "b"}); err == nil {
		t.Error("a seed that is not hexadecimal was accepted")
	}
}

// TestCombinedSeedDuel: a Duel with a combined seed rolls nothing, awaits no
// Decision and runs no clock before each external Side has contributed once;
// the Match's origin then lets anyone recompute every roll.
func TestCombinedSeedDuel(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	now := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	svc := New(st, Options{Sides: testSides, Rand: fixedRand(5), Now: func() time.Time { return now }})
	cad := Cadence{Name: "rapid-3+12", Reserve: 180, Delay: 12}
	s, err := svc.Create(ctx, "", Settings{MatchLength: 5, Cadence: &cad, CombinedSeed: true,
		Sides: [2]SideSpec{external("A"), external("B")}})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	published := s.Fingerprint
	if len(s.Actions) != 0 || s.Awaiting != nil || !slices.Equal(s.AwaitingContribution, []int{0, 1}) {
		t.Fatalf("before the contributions: actions %v, awaiting %+v, contributions owed %v", s.Actions, s.Awaiting, s.AwaitingContribution)
	}
	reserve := s.Clock.Reserve
	now = now.Add(time.Hour)
	if s, err = svc.Flag(ctx, "", s.ID); err != nil || s.Clock.Reserve != reserve || s.Clock.OverTime != 0 {
		t.Fatalf("the clock ran before the contributions: %+v, %v", s.Clock, err)
	}
	_, err = svc.Play(ctx, "", s.ID, s.Revision, Play{Side: 0, Kind: PlayRoll})
	if k := refusalKind(t, err); k != RefusedContributionPending {
		t.Fatalf("a play before the contributions: %v", err)
	}

	if s, err = svc.Contribute(ctx, "", s.ID, s.Revision, 0, "alice's ‖ dice"); err != nil {
		t.Fatalf("Contribute 1: %v", err)
	}
	if len(s.Actions) != 0 || s.Awaiting != nil || !slices.Equal(s.AwaitingContribution, []int{1}) {
		t.Fatalf("after one contribution: actions %v, awaiting %+v", s.Actions, s.Awaiting)
	}
	for _, c := range []struct {
		side  int
		value string
	}{{0, "again"}, {1, ""}, {1, strings.Repeat("x", MaxContribution+1)}, {2, "x"}} {
		if _, err := svc.Contribute(ctx, "", s.ID, s.Revision, c.side, c.value); refusalKind(t, err) != RefusedContribution {
			t.Errorf("contribution %d %q: %v", c.side, c.value, err)
		}
	}
	if s, err = svc.Contribute(ctx, "", s.ID, s.Revision, 1, "bob"); err != nil {
		t.Fatalf("Contribute 2: %v", err)
	}
	if s.Awaiting == nil || len(s.AwaitingContribution) != 0 || s.Contributions != [2]string{"alice's ‖ dice", "bob"} {
		t.Fatalf("after the contributions: awaiting %+v, %v", s.Awaiting, s.Contributions)
	}
	if _, err := svc.Contribute(ctx, "", s.ID, s.Revision, 1, "late"); refusalKind(t, err) != RefusedContribution {
		t.Errorf("a contribution after the first roll: %v", err)
	}
	for range 4 {
		if s, err = svc.Play(ctx, "", s.ID, s.Revision, answer(*s.Awaiting)); err != nil {
			t.Fatalf("Play: %v", err)
		}
	}
	kept, err := svc.Forfeit(ctx, "", s.ID, s.Revision, 0)
	if err != nil || kept.Ended == nil {
		t.Fatalf("Forfeit: %+v, %v", kept, err)
	}

	// Recomputed from the origin alone, as outside blunderDB.
	o, err := ReadOrigin(ctx, st, "", kept.Ended.MatchID)
	if err != nil || o == nil {
		t.Fatalf("ReadOrigin: %+v, %v", o, err)
	}
	if fp, _ := Fingerprint(o.DiceSeed); fp != published {
		t.Errorf("the revealed seed does not give the published fingerprint")
	}
	rollSeed, _ := CombinedSeed(o.DiceSeed, [2]string{o.Contributions[0], o.Contributions[1]})
	if !slices.Equal(o.Contributions, []string{"alice's ‖ dice", "bob"}) || o.RollSeed != rollSeed {
		t.Fatalf("origin: contributions %q, roll seed %q", o.Contributions, o.RollSeed)
	}
	rank, opening, checked := 0, true, 0
	for _, a := range kept.Actions {
		if a.Kind != transcript.KindChecker {
			continue
		}
		d, _ := Roll(rollSeed, rank)
		rank++
		for opening && d[0] == d[1] {
			d, _ = Roll(rollSeed, rank)
			rank++
		}
		opening = false
		if !sameRoll(d, a.Dice) {
			t.Fatalf("roll %d recomputed %v, played %v", checked, d, a.Dice)
		}
		checked++
	}
	if checked < 2 {
		t.Fatalf("only %d rolls checked", checked)
	}
}

func sameRoll(a, b [2]int) bool { return a == b || a == [2]int{b[1], b[0]} }

// TestOrdinaryDuelRollsFromItsSeed: a Duel without a combined seed rolls
// Roll(DiceSeed, rank) at every rank, the opening's doubles drawn again —
// what a reader recomputes from the revealed seed alone.
func TestOrdinaryDuelRollsFromItsSeed(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	svc := newService(t, st, 11)
	s, err := svc.Create(ctx, "", Settings{MatchLength: 5, Sides: [2]SideSpec{external("A"), external("B")}})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	for range 12 {
		if s, err = svc.Play(ctx, "", s.ID, s.Revision, answer(*s.Awaiting)); err != nil {
			t.Fatalf("Play: %v", err)
		}
	}
	kept, err := svc.Forfeit(ctx, "", s.ID, s.Revision, 0)
	if err != nil || kept.Ended == nil {
		t.Fatalf("Forfeit: %+v, %v", kept, err)
	}
	o, err := ReadOrigin(ctx, st, "", kept.Ended.MatchID)
	if err != nil || o == nil {
		t.Fatalf("ReadOrigin: %+v, %v", o, err)
	}
	if len(o.Contributions) != 0 || o.RollSeed != "" {
		t.Fatalf("origin: contributions %q, roll seed %q; the dice seed alone", o.Contributions, o.RollSeed)
	}
	rank, opening, checked := 0, true, 0
	for _, a := range kept.Actions {
		if a.Kind != transcript.KindChecker {
			continue
		}
		d, _ := Roll(o.DiceSeed, rank)
		rank++
		for opening && d[0] == d[1] {
			d, _ = Roll(o.DiceSeed, rank)
			rank++
		}
		opening = false
		if !sameRoll(d, a.Dice) {
			t.Fatalf("roll %d recomputed %v at rank %d, played %v", checked, d, rank-1, a.Dice)
		}
		checked++
	}
	if checked < 4 {
		t.Fatalf("only %d rolls checked", checked)
	}
}

// TestCombinedSeedWithABot: only the external Side contributes; a combined
// seed between two delegated Sides is refused, and a Duel without one refuses
// every contribution and keeps its origin without any.
func TestCombinedSeedWithABot(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	svc := newService(t, st, 5)
	s, err := svc.Create(ctx, "", Settings{MatchLength: 3, CombinedSeed: true,
		Sides: [2]SideSpec{{Kind: sideFirst}, external("B")}})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !slices.Equal(s.AwaitingContribution, []int{1}) {
		t.Fatalf("contributions owed %v, want player 2's only", s.AwaitingContribution)
	}
	if _, err := svc.Contribute(ctx, "", s.ID, s.Revision, 0, "x"); refusalKind(t, err) != RefusedContribution {
		t.Errorf("a delegated Side's contribution: %v", err)
	}
	if s, err = svc.Contribute(ctx, "", s.ID, s.Revision, 1, "b"); err != nil || len(s.Actions) == 0 && s.Awaiting == nil {
		t.Fatalf("Contribute: %+v, %v", s, err)
	}

	if _, err := svc.Create(ctx, "", Settings{MatchLength: 3, CombinedSeed: true,
		Sides: [2]SideSpec{{Kind: sideFirst}, {Kind: sideFirst}}}); !errors.Is(err, storage.ErrInvalid) {
		t.Errorf("a combined seed between two delegated Sides: %v", err)
	}

	plain, err := svc.Create(ctx, "", Settings{MatchLength: 3, Sides: [2]SideSpec{external("A"), external("B")}})
	if err != nil || plain.AwaitingContribution != nil || plain.Awaiting == nil {
		t.Fatalf("Create without a combined seed: %+v, %v", plain, err)
	}
	if _, err := svc.Contribute(ctx, "", plain.ID, plain.Revision, 0, "x"); refusalKind(t, err) != RefusedContribution {
		t.Errorf("a contribution to a Duel without a combined seed: %v", err)
	}
	kept, err := svc.Forfeit(ctx, "", plain.ID, plain.Revision, 0)
	if err != nil || kept.Ended == nil {
		t.Fatalf("Forfeit: %+v, %v", kept, err)
	}
	if o, err := ReadOrigin(ctx, st, "", kept.Ended.MatchID); err != nil || o.Contributions != nil || o.RollSeed != "" {
		t.Errorf("origin without a combined seed: %+v, %v", o, err)
	}
}
