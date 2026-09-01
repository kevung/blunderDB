// Contract cases for matches: lookup by hash, listing, copy-on-write swaps, the
// match/game/move cascade and per-move luck.
// The table that runs them lives in contract.go.
package storagetest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func testMatchFindByHash(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ms := s.Matches()

	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7,
		MatchHash: "h1", CanonicalHash: "c1"}
	id, err := ms.Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}

	if got, found, err := ms.FindByHash(ctx, "", "h1", ""); err != nil || !found || got != id {
		t.Fatalf("FindByHash(match_hash): got=%d found=%v err=%v, want id=%d found", got, found, err, id)
	}
	if got, found, err := ms.FindByHash(ctx, "", "", "c1"); err != nil || !found || got != id {
		t.Fatalf("FindByHash(canonical): got=%d found=%v err=%v, want id=%d found", got, found, err, id)
	}
	if _, found, err := ms.FindByHash(ctx, "", "nope", "nope"); err != nil || found {
		t.Fatalf("FindByHash(absent): found=%v err=%v, want not found", found, err)
	}

	// Two hash-less matches must both store (NULL canonical_hash, not '',
	// so the UNIQUE index does not reject the second).
	for i := range 2 {
		hm := domain.Match{Player1Name: "X", Player2Name: "Y", MatchLength: 3}
		if _, err := ms.Save(ctx, "", &hm); err != nil {
			t.Fatalf("Save hash-less match %d: %v", i, err)
		}
	}
}

// testMatchListFilterSortPaginate pins the filter/sort/pagination contract of
// MatchStore.List across backends. Data: three matches with distinct dates,
// lengths and players; Alice plays two of them, one of which is in a tournament.
func testMatchListFilterSortPaginate(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ms := s.Matches()

	date := func(iso string) time.Time {
		tm, err := time.Parse("2006-01-02", iso)
		if err != nil {
			t.Fatalf("parse date %q: %v", iso, err)
		}
		return tm
	}
	// Save oldest→newest so ids and dates disagree with default (date-desc) order.
	old := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7, MatchDate: date("2023-06-15")}
	mid := domain.Match{Player1Name: "Carol", Player2Name: "Dave", MatchLength: 3, MatchDate: date("2024-06-15")}
	// A late-in-the-day timestamp guards the inclusive DateTo boundary.
	recent := domain.Match{Player1Name: "Eve", Player2Name: "Alice", MatchLength: 11, MatchDate: date("2025-06-15").Add(23 * time.Hour)}
	oldID, err := ms.Save(ctx, "", &old)
	if err != nil {
		t.Fatalf("Save old: %v", err)
	}
	midID, err := ms.Save(ctx, "", &mid)
	if err != nil {
		t.Fatalf("Save mid: %v", err)
	}
	recentID, err := ms.Save(ctx, "", &recent)
	if err != nil {
		t.Fatalf("Save recent: %v", err)
	}

	tID, err := s.Tournaments().Create(ctx, "", "Cup", "2024-01-01", "Paris")
	if err != nil {
		t.Fatalf("Create tournament: %v", err)
	}
	if err := s.Tournaments().AddMatch(ctx, "", tID, midID); err != nil {
		t.Fatalf("AddMatch: %v", err)
	}

	ids := func(opts storage.MatchListOpts) []int64 {
		t.Helper()
		var out []int64
		for m, err := range ms.List(ctx, "", opts) {
			if err != nil {
				t.Fatalf("List(%+v): %v", opts, err)
			}
			out = append(out, m.ID)
		}
		return out
	}
	eq := func(name string, got, want []int64) {
		t.Helper()
		if len(got) != len(want) {
			t.Errorf("%s: got %v, want %v", name, got, want)
			return
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("%s: got %v, want %v", name, got, want)
				return
			}
		}
	}

	// Default: every match, most recent first.
	eq("default order", ids(storage.MatchListOpts{}), []int64{recentID, midID, oldID})
	// Sort keys.
	eq("date_asc", ids(storage.MatchListOpts{Sort: "date_asc"}), []int64{oldID, midID, recentID})
	eq("length_desc", ids(storage.MatchListOpts{Sort: "length_desc"}), []int64{recentID, oldID, midID})
	eq("length_asc", ids(storage.MatchListOpts{Sort: "length_asc"}), []int64{midID, oldID, recentID})
	// Player filter: Alice is player 1 of `old` and player 2 of `recent`.
	eq("player Alice", ids(storage.MatchListOpts{PlayerName: "Alice"}), []int64{recentID, oldID})
	// Date filters, inclusive on the day (recent is at 23:00 on the DateTo day).
	eq("from 2024-01-01", ids(storage.MatchListOpts{DateFrom: "2024-01-01"}), []int64{recentID, midID})
	eq("to 2024-12-31", ids(storage.MatchListOpts{DateTo: "2024-12-31"}), []int64{midID, oldID})
	eq("year 2025", ids(storage.MatchListOpts{DateFrom: "2025-01-01", DateTo: "2025-12-31"}), []int64{recentID})
	// Match length.
	eq("length 3 or 7", ids(storage.MatchListOpts{MatchLength: []int{3, 7}, Sort: "date_asc"}), []int64{oldID, midID})
	// Tournament.
	eq("tournament", ids(storage.MatchListOpts{TournamentIDs: []int64{tID}}), []int64{midID})
	// Pagination over the default order.
	eq("limit 2", ids(storage.MatchListOpts{Limit: 2}), []int64{recentID, midID})
	eq("limit 2 offset 1", ids(storage.MatchListOpts{Limit: 2, Offset: 1}), []int64{midID, oldID})
	eq("offset 2", ids(storage.MatchListOpts{Offset: 2}), []int64{oldID})
	// Combined: Alice's matches, oldest first, first one only.
	eq("combined", ids(storage.MatchListOpts{PlayerName: "Alice", Sort: "date_asc", Limit: 1}), []int64{oldID})
}

// testMatchSwapCopyOnWrite pins #107: swapping a match's players must not mutate
// a position shared with another match, and the swapped position must keep a
// consistent Zobrist hash (score/cube are hashed). Two matches reference the same
// position; after swapping match A, match B's position is untouched and A's is a
// correctly-hashed swapped copy.
func testMatchSwapCopyOnWrite(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	// A shared position with an asymmetric score so the swap is observable.
	p := checkerPos()
	p.Score = [2]int{3, 1}
	sharedID, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}

	makeMatch := func(p1, p2 string, posID int64) int64 {
		m := domain.Match{Player1Name: p1, Player2Name: p2, MatchLength: 7}
		mid, err := s.Matches().Save(ctx, "", &m)
		if err != nil {
			t.Fatalf("Save match: %v", err)
		}
		g := domain.Game{MatchID: mid, GameNumber: 1, InitialScore: [2]int32{3, 1}, Winner: 1, PointsWon: 1}
		gid, err := s.Matches().CreateGame(ctx, "", &g)
		if err != nil {
			t.Fatalf("CreateGame: %v", err)
		}
		mv := domain.Move{GameID: gid, MoveNumber: 1, MoveType: "checker", PositionID: posID, Player: 1, Dice: [2]int32{3, 1}, CheckerMove: "8/5 6/5"}
		if _, err := s.Matches().CreateMove(ctx, "", &mv); err != nil {
			t.Fatalf("CreateMove: %v", err)
		}
		return mid
	}
	matchA := makeMatch("Alice", "Bob", sharedID)
	matchB := makeMatch("Carol", "Dave", sharedID)

	if err := s.Matches().SwapPlayers(ctx, "", matchA); err != nil {
		t.Fatalf("SwapPlayers: %v", err)
	}

	// The position id each match's move now points at.
	posOf := func(matchID int64) (int64, domain.Position) {
		t.Helper()
		var pid int64
		for mv, err := range s.Matches().MovesByMatch(ctx, "", matchID) {
			if err != nil {
				t.Fatalf("MovesByMatch: %v", err)
			}
			pid = mv.PositionID
		}
		pos, err := s.Positions().Load(ctx, "", pid)
		if err != nil {
			t.Fatalf("Load position %d: %v", pid, err)
		}
		return pid, *pos
	}

	// Match B (not swapped) must keep the original shared position, unchanged.
	bID, bPos := posOf(matchB)
	if bID != sharedID {
		t.Errorf("match B position moved from %d to %d (should be untouched)", sharedID, bID)
	}
	if bPos.Score != [2]int{3, 1} {
		t.Errorf("shared position corrupted by swap: score = %v, want [3 1]", bPos.Score)
	}

	// Match A now points at a swapped copy with the mirrored score.
	aID, aPos := posOf(matchA)
	if aID == sharedID {
		t.Errorf("swap mutated the shared position in place (A still on %d)", sharedID)
	}
	if aPos.Score != [2]int{1, 3} {
		t.Errorf("swapped position score = %v, want [1 3]", aPos.Score)
	}

	// Zobrist consistency: re-saving A's swapped content must dedup back to the
	// same row (a stale hash would create a new row, returning a different id).
	reSaved, err := s.Positions().Save(ctx, "", &aPos)
	if err != nil {
		t.Fatalf("re-save swapped position: %v", err)
	}
	if reSaved != aID {
		t.Errorf("swapped position has a stale Zobrist: re-save returned %d, want %d", reSaved, aID)
	}

	// The swap's orphan cleanup mirrors DeleteCascade's (see positionIsHeldSQL):
	// a position that was only this match's, and that nothing else holds once
	// the move repoints to the swapped copy, must be purged under its old id.
	orphan := checkerPos()
	orphan.Score = [2]int{2, 0}
	orphanID, err := s.Positions().Save(ctx, "", &orphan)
	if err != nil {
		t.Fatalf("Save orphan position: %v", err)
	}
	matchC := makeMatch("Eve", "Frank", orphanID)
	if err := s.Matches().SwapPlayers(ctx, "", matchC); err != nil {
		t.Fatalf("SwapPlayers (orphan case): %v", err)
	}
	if _, err := s.Positions().Load(ctx, "", orphanID); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("swap did not purge the orphaned old position: got %v, want ErrNotFound", err)
	}

	// A retained position (flagged, ADR-0006) must survive the swap under its
	// old id even though no move points at it any more: positionIsHeldSQL keeps
	// it alive the same way it would across a match deletion.
	retained := checkerPos()
	retained.Score = [2]int{4, 0}
	retained.Flagged = true
	retainedID, err := s.Positions().Save(ctx, "", &retained)
	if err != nil {
		t.Fatalf("Save retained position: %v", err)
	}
	matchD := makeMatch("Gina", "Hank", retainedID)
	if err := s.Matches().SwapPlayers(ctx, "", matchD); err != nil {
		t.Fatalf("SwapPlayers (retained case): %v", err)
	}
	if _, err := s.Positions().Load(ctx, "", retainedID); err != nil {
		t.Errorf("swap purged a flagged position that should have been retained: %v", err)
	}
}

func testMatchCreateGameMove(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7}
	matchID, err := s.Matches().Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}
	if matchID == 0 || m.ID != matchID {
		t.Fatalf("Save match id: id=%d m.ID=%d", matchID, m.ID)
	}

	g := domain.Game{MatchID: matchID, GameNumber: 1, Winner: 1, PointsWon: 2}
	gameID, err := s.Matches().CreateGame(ctx, "", &g)
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}

	p := checkerPos()
	posID, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}
	mv := domain.Move{
		GameID: gameID, MoveNumber: 1, MoveType: "checker", PositionID: posID,
		Player: 1, Dice: [2]int32{3, 1}, CheckerMove: "8/5 6/5",
	}
	moveID, err := s.Matches().CreateMove(ctx, "", &mv)
	if err != nil {
		t.Fatalf("CreateMove: %v", err)
	}
	if moveID == 0 {
		t.Fatal("CreateMove returned id 0")
	}

	got, err := s.Matches().Get(ctx, "", matchID)
	if err != nil {
		t.Fatalf("Get match: %v", err)
	}
	if got.Player1Name != "Alice" || got.MatchLength != 7 {
		t.Errorf("Get match: %+v", got)
	}

	var games []domain.Game
	for g, err := range s.Matches().Games(ctx, "", matchID) {
		if err != nil {
			t.Fatalf("Games: %v", err)
		}
		games = append(games, *g)
	}
	if len(games) != 1 || games[0].Winner != 1 || games[0].PointsWon != 2 {
		t.Fatalf("Games: %+v", games)
	}

	var moves []domain.Move
	for mv, err := range s.Matches().Moves(ctx, "", gameID) {
		if err != nil {
			t.Fatalf("Moves: %v", err)
		}
		moves = append(moves, *mv)
	}
	if len(moves) != 1 || moves[0].CheckerMove != "8/5 6/5" || moves[0].PositionID != posID {
		t.Fatalf("Moves: %+v", moves)
	}

	var mps []domain.MatchMovePosition
	for mp, err := range s.Matches().MovePositions(ctx, "", matchID) {
		if err != nil {
			t.Fatalf("MovePositions: %v", err)
		}
		mps = append(mps, *mp)
	}
	if len(mps) != 1 {
		t.Fatalf("MovePositions count: got %d, want 1", len(mps))
	}
	// Move stored with XG player 1 maps to blunderDB player 0.
	if mps[0].PlayerOnRoll != 0 {
		t.Errorf("MovePositions PlayerOnRoll: got %d, want 0", mps[0].PlayerOnRoll)
	}
	if mps[0].Player1Name != "Alice" || mps[0].CheckerMove != "8/5 6/5" {
		t.Errorf("MovePositions context: %+v", mps[0])
	}
}

// testMoveLuckRoundTrip pins the three states move.luck_mp has to keep apart on
// both backends: a lucky roll, an unlucky one, and a roll whose luck is
// unknown. The unknown case is the one worth a test — a backend that reads a
// NULL column back as 0 would silently turn "we don't know" into "the dice were
// exactly fair", which is what every luck average must not average over.
// Negative values matter too: luck is signed, and a column or scan that lost
// the sign would still look plausible on a lucky roll.
func testMoveLuckRoundTrip(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	matchID, err := s.Matches().Save(ctx, "",
		&domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7})
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}
	gameID, err := s.Matches().CreateGame(ctx, "", &domain.Game{MatchID: matchID, GameNumber: 1})
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}

	lucky, unlucky := int32(214), int32(-329)
	cases := []struct {
		name string
		want *int32
	}{
		{"lucky", &lucky},
		{"unlucky", &unlucky},
		{"unknown", nil},
	}
	for i, c := range cases {
		mv := domain.Move{GameID: gameID, MoveNumber: int32(i), MoveType: "checker",
			Player: 1, Dice: [2]int32{3, 1}, CheckerMove: "8/5 6/5", LuckMP: c.want}
		if _, err := s.Matches().CreateMove(ctx, "", &mv); err != nil {
			t.Fatalf("CreateMove(%s): %v", c.name, err)
		}
	}

	var got []*int32
	for mv, err := range s.Matches().Moves(ctx, "", gameID) {
		if err != nil {
			t.Fatalf("Moves: %v", err)
		}
		got = append(got, mv.LuckMP)
	}
	if len(got) != len(cases) {
		t.Fatalf("Moves: got %d moves, want %d", len(got), len(cases))
	}
	for i, c := range cases {
		switch {
		case c.want == nil && got[i] != nil:
			t.Errorf("%s roll: got luck %d, want unknown (NULL)", c.name, *got[i])
		case c.want != nil && got[i] == nil:
			t.Errorf("%s roll: got unknown, want luck %d", c.name, *c.want)
		case c.want != nil && *got[i] != *c.want:
			t.Errorf("%s roll: got luck %d, want %d", c.name, *got[i], *c.want)
		}
	}

	// MovesByMatch reads the same column through a different query; it must
	// agree with Moves rather than quietly drop the value.
	var byMatch []*int32
	for mv, err := range s.Matches().MovesByMatch(ctx, "", matchID) {
		if err != nil {
			t.Fatalf("MovesByMatch: %v", err)
		}
		byMatch = append(byMatch, mv.LuckMP)
	}
	if len(byMatch) != len(got) {
		t.Fatalf("MovesByMatch: got %d moves, want %d", len(byMatch), len(got))
	}
	for i := range got {
		if (byMatch[i] == nil) != (got[i] == nil) ||
			(got[i] != nil && *byMatch[i] != *got[i]) {
			t.Errorf("move %d: MovesByMatch and Moves disagree on luck", i)
		}
	}
}
