package storagetest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// newTournamentMatch stores a Match of the Tournament with one game won by
// player 1 for points, from the given scores.
func newTournamentMatch(t *testing.T, s storage.Storage, tid int64, p1, p2 string, score [2]int32, points int32) int64 {
	t.Helper()
	ctx := context.Background()
	m := domain.Match{Player1Name: p1, Player2Name: p2, MatchLength: 5,
		MatchDate: time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC)}
	id, err := s.Matches().Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}
	if err := s.Tournaments().AddMatch(ctx, "", tid, id); err != nil {
		t.Fatalf("AddMatch: %v", err)
	}
	if points > 0 {
		g := domain.Game{MatchID: id, GameNumber: 1, InitialScore: score, Winner: 1, PointsWon: points}
		if _, err := s.Matches().CreateGame(ctx, "", &g); err != nil {
			t.Fatalf("CreateGame: %v", err)
		}
	}
	return id
}

// testDirectionSlots: a Match fills one Slot, reads back with the score its
// games give, leaves the unattached list, and is released by DetachSlot.
func testDirectionSlots(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ds := s.Directions()
	tid := newDirectedTournament(t, s, "Principal")
	scored := newTournamentMatch(t, s, tid, "Anna", "Bruno", [2]int32{3, 2}, 2)
	bare := newTournamentMatch(t, s, tid, "Carl", "Dora", [2]int32{}, 0)

	if err := ds.AttachSlot(ctx, "", tid, "M1", 999999); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("AttachSlot of a missing Match = %v, want ErrNotFound", err)
	}
	if err := ds.AttachSlot(ctx, "", tid, "M1", scored); err != nil {
		t.Fatalf("AttachSlot: %v", err)
	}
	if err := ds.AttachSlot(ctx, "", tid, "M1", bare); err == nil {
		t.Fatal("a second Match in an occupied Slot was accepted")
	}
	filled, err := ds.FilledSlots(ctx, "", tid)
	if err != nil || len(filled) != 1 {
		t.Fatalf("FilledSlots = %+v, %v; want one", filled, err)
	}
	f := filled[0]
	if f.SlotID != "M1" || f.MatchID != scored || f.Player1 != "Anna" || f.Player2 != "Bruno" || f.Length != 5 ||
		!f.HasScore || f.Score1 != 5 || f.Score2 != 2 {
		t.Errorf("FilledSlots[0] = %+v, want M1 Anna-Bruno 5-2", f)
	}
	free, err := ds.UnattachedMatches(ctx, "", tid)
	if err != nil || len(free) != 1 || free[0].MatchID != bare || free[0].Player1 != "Carl" || free[0].Date == "" {
		t.Fatalf("UnattachedMatches = %+v, %v; want Carl-Dora with a date", free, err)
	}
	if gotT, slot, err := ds.SlotOf(ctx, "", scored); err != nil || gotT != tid || slot != "M1" {
		t.Errorf("SlotOf(filled) = %d, %q, %v", gotT, slot, err)
	}
	if _, slot, err := ds.SlotOf(ctx, "", bare); err != nil || slot != "" {
		t.Errorf("SlotOf(unfilled) = %q, %v; want none, no error", slot, err)
	}
	if _, _, err := ds.SlotOf(ctx, "", 999999); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("SlotOf(missing) = %v, want ErrNotFound", err)
	}
	if err := ds.DetachSlot(ctx, "", tid, "M1"); err != nil {
		t.Fatalf("DetachSlot: %v", err)
	}
	if filled, _ := ds.FilledSlots(ctx, "", tid); len(filled) != 0 {
		t.Errorf("FilledSlots after DetachSlot = %+v", filled)
	}
	if m, err := s.Matches().Get(ctx, "", scored); err != nil || m == nil {
		t.Errorf("Match after DetachSlot: %v", err)
	}
}

// testDirectionDeleteFreesSlots: deleting a Direction empties its Slots and
// never deletes the Match that filled one.
func testDirectionDeleteFreesSlots(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ds := s.Directions()
	tid := newDirectedTournament(t, s, "Principal")
	mid := newTournamentMatch(t, s, tid, "Anna", "Bruno", [2]int32{}, 0)
	if err := ds.AttachSlot(ctx, "", tid, "M1", mid); err != nil {
		t.Fatalf("AttachSlot: %v", err)
	}
	if err := ds.Delete(ctx, "", tid); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, slot, err := ds.SlotOf(ctx, "", mid); err != nil || slot != "" {
		t.Errorf("SlotOf after Delete = %q, %v; want the Slot freed", slot, err)
	}
	if m, err := s.Matches().Get(ctx, "", mid); err != nil || m == nil {
		t.Fatalf("Match after Delete: %v", err)
	}
	owner, err := s.Tournaments().TournamentOf(ctx, "", mid)
	if err != nil || owner == nil || owner.ID != tid {
		t.Errorf("TournamentOf after Delete = %+v, %v; want the Match kept in %d", owner, err, tid)
	}
}

// testDirectionPairs: SetPair replaces a Participant's persons in seat
// order, and leaves the other pairs alone.
func testDirectionPairs(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ds := s.Directions()
	tid := newDirectedTournament(t, s, "Doubles")
	if got, err := ds.Pairs(ctx, "", tid); err != nil || len(got) != 0 {
		t.Fatalf("Pairs of a singles event = %+v, %v", got, err)
	}
	if err := ds.SetPair(ctx, "", tid, "P1", []storage.PairMember{{Name: "Anna", Club: "Lyon", Rating: 1600}, {Name: "Bruno"}}); err != nil {
		t.Fatalf("SetPair: %v", err)
	}
	if err := ds.SetPair(ctx, "", tid, "P2", []storage.PairMember{{Name: "Carl"}, {Name: "Dora"}}); err != nil {
		t.Fatalf("SetPair: %v", err)
	}
	if err := ds.SetPair(ctx, "", tid, "P1", []storage.PairMember{{Name: "Émile"}, {Name: "Anna", Club: "Lyon", Rating: 1600}}); err != nil {
		t.Fatalf("SetPair again: %v", err)
	}
	got, err := ds.Pairs(ctx, "", tid)
	if err != nil {
		t.Fatalf("Pairs: %v", err)
	}
	if p := got["P1"]; len(p) != 2 || p[0].Name != "Émile" || p[1] != (storage.PairMember{Name: "Anna", Club: "Lyon", Rating: 1600}) {
		t.Errorf("P1 = %+v, want Émile then Anna", p)
	}
	if p := got["P2"]; len(p) != 2 || p[0].Name != "Carl" || p[1].Name != "Dora" {
		t.Errorf("P2 = %+v, want Carl then Dora", p)
	}
}
