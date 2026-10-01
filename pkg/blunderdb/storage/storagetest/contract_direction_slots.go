package storagetest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// newTournamentMatch stores a 5-point Match of the Tournament with the given
// games.
func newTournamentMatch(t *testing.T, s storage.Storage, tid int64, p1, p2 string, games ...domain.Game) int64 {
	return newTournamentMatchOf(t, s, tid, p1, p2, 5, games...)
}

func newTournamentMatchOf(t *testing.T, s storage.Storage, tid int64, p1, p2 string, length int32, games ...domain.Game) int64 {
	t.Helper()
	ctx := context.Background()
	m := domain.Match{Player1Name: p1, Player2Name: p2, MatchLength: length,
		MatchDate: time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC)}
	id, err := s.Matches().Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}
	if err := s.Tournaments().AddMatch(ctx, "", tid, id); err != nil {
		t.Fatalf("AddMatch: %v", err)
	}
	for i, g := range games {
		g.MatchID, g.GameNumber = id, int32(i+1)
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
	// Game.Winner is gnubg's encoding here: 1 is player 2.
	scored := newTournamentMatch(t, s, tid, "Anna", "Bruno", domain.Game{InitialScore: [2]int32{3, 2}, Winner: 1, PointsWon: 2})
	bare := newTournamentMatch(t, s, tid, "Carl", "Dora")

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
		!f.HasScore || f.Score1 != 3 || f.Score2 != 4 {
		t.Errorf("FilledSlots[0] = %+v, want M1 Anna-Bruno 3-4", f)
	}
	free, err := ds.UnattachedMatches(ctx, "", tid)
	if err != nil || len(free) != 1 || free[0].MatchID != bare || free[0].Player1 != "Carl" || free[0].Date != "2026-10-03" {
		t.Fatalf("UnattachedMatches = %+v, %v; want Carl-Dora dated 2026-10-03", free, err)
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
	mid := newTournamentMatch(t, s, tid, "Anna", "Bruno")
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

// testDirectionSlotScore: the final score of a filled Slot credits every game
// to its winner and never passes the length. Game.Winner has two encodings in
// the library — gnubg's 0/1 (-1 unfinished) for a transcribed, .mat or .sgf
// Match, XG's -1/1 (0 unfinished) for an .xg one — and with points won both
// read 1 as player 2 and anything else as player 1.
func testDirectionSlotScore(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ds := s.Directions()
	tid := newDirectedTournament(t, s, "Principal")
	g := func(s1, s2, winner, points int32) domain.Game {
		return domain.Game{InitialScore: [2]int32{s1, s2}, Winner: winner, PointsWon: points}
	}
	cases := []struct {
		name   string
		games  []domain.Game
		s1, s2 int
	}{
		{"gnubg, gammon at 5-5 to 7", []domain.Game{g(0, 0, 0, 5), g(5, 0, 1, 5), g(5, 5, 1, 4)}, 5, 7},
		{"gnubg, player 1 wins last", []domain.Game{g(0, 0, 1, 2), g(0, 2, 0, 6)}, 6, 2},
		{"gnubg, last game unfinished", []domain.Game{g(0, 0, 0, 2), g(2, 0, -1, 0)}, 2, 0},
		{"XG, player 1 wins last", []domain.Game{g(0, 0, 1, 2), g(0, 2, -1, 9)}, 7, 2},
		{"XG, last game unfinished", []domain.Game{g(0, 0, -1, 3), g(3, 0, 0, 0)}, 3, 0},
	}
	want := map[string][2]int{}
	for i, c := range cases {
		mid := newTournamentMatchOf(t, s, tid, "Anna", "Bruno", 7, c.games...)
		slot := "M" + string(rune('1'+i))
		if err := ds.AttachSlot(ctx, "", tid, slot, mid); err != nil {
			t.Fatalf("AttachSlot: %v", err)
		}
		want[slot] = [2]int{c.s1, c.s2}
	}
	filled, err := ds.FilledSlots(ctx, "", tid)
	if err != nil || len(filled) != len(cases) {
		t.Fatalf("FilledSlots = %+v, %v", filled, err)
	}
	for i, f := range filled {
		w := want[f.SlotID]
		if !f.HasScore || f.Score1 != w[0] || f.Score2 != w[1] {
			t.Errorf("%s: score %d-%d (has %v), want %d-%d", cases[i].name, f.Score1, f.Score2, f.HasScore, w[0], w[1])
		}
	}
}

// testDirectionDeleteDropsPairs: deleting a Direction removes its pairs, so a
// Direction created again on the same Tournament starts with none.
func testDirectionDeleteDropsPairs(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ds := s.Directions()
	tid := newDirectedTournament(t, s, "Doubles")
	if err := ds.SetPair(ctx, "", tid, "P1", []storage.PairMember{{Name: "Anna"}, {Name: "Bruno"}}); err != nil {
		t.Fatalf("SetPair: %v", err)
	}
	if err := ds.Delete(ctx, "", tid); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	rec := direction.Record{TournamentID: tid, FormatVersion: 1, EngineVersion: direction.EngineVersion,
		State: direction.StateDraft, Config: `{}`}
	if err := ds.Create(ctx, "", rec); err != nil {
		t.Fatalf("Create again: %v", err)
	}
	if got, err := ds.Pairs(ctx, "", tid); err != nil || len(got) != 0 {
		t.Errorf("Pairs after Delete and Create = %+v, %v; want none", got, err)
	}
}
