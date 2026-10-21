// Contract case for the batched game insert.
package storagetest

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testCreateGamesEqualsCreateGame pins CreateGames against CreateGame one by
// one: ids in call order across two matches, the same rows read back, and the
// match_stats rows of each match invalidated — a stale row would hide the
// decisions of the new games from every PR panel.
func testCreateGamesEqualsCreateGame(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ms := s.Matches()
	matchA, posA := statsFixtureMatch(t, s, 0, "Alice", "Bob")
	matchB, posB := statsFixtureMatch(t, s, 2, "Carol", "Dave")
	if _, err := s.Stats().MatchStats(ctx, "", nil); err != nil {
		t.Fatalf("MatchStats: %v", err)
	}

	if err := ms.CreateGames(ctx, "", nil); err != nil {
		t.Fatalf("CreateGames(nil): %v", err)
	}
	batch := []*domain.Game{
		{MatchID: matchA, GameNumber: 2, InitialScore: [2]int32{1, 0}, Winner: 1, PointsWon: 2, MoveCount: 1},
		{MatchID: matchB, GameNumber: 2, InitialScore: [2]int32{0, 1}, Winner: -1, PointsWon: 1, MoveCount: 1},
		{MatchID: matchA, GameNumber: 3, InitialScore: [2]int32{3, 0}, Winner: -1, PointsWon: 4},
	}
	if err := ms.CreateGames(ctx, "", batch); err != nil {
		t.Fatalf("CreateGames: %v", err)
	}
	for i, g := range batch {
		if g.ID <= 0 || (i > 0 && g.ID <= batch[i-1].ID) {
			t.Fatalf("CreateGames ids %d, %d, %d: want increasing positive ids", batch[0].ID, batch[1].ID, batch[2].ID)
		}
	}
	want := map[int64]*domain.Game{}
	for _, g := range batch {
		want[g.ID] = g
	}
	for _, matchID := range []int64{matchA, matchB} {
		for g, err := range ms.Games(ctx, "", matchID) {
			if err != nil {
				t.Fatalf("Games(%d): %v", matchID, err)
			}
			w, ok := want[g.ID]
			if !ok {
				continue
			}
			if *g != *w {
				t.Errorf("game %d reads back %+v, CreateGames stored %+v", g.ID, *g, *w)
			}
			delete(want, g.ID)
		}
	}
	if len(want) != 0 {
		t.Errorf("%d games from CreateGames not listed by Games", len(want))
	}

	for i, pos := range []int64{posA[0], posB[0]} {
		if _, err := ms.CreateMove(ctx, "", &domain.Move{GameID: batch[i].ID, MoveNumber: 1, MoveType: "checker",
			PositionID: pos, Player: 1, CheckerMove: "13/11 24/23"}); err != nil {
			t.Fatalf("CreateMove: %v", err)
		}
	}
	rows, err := s.Stats().MatchStats(ctx, "", nil)
	if err != nil {
		t.Fatalf("MatchStats: %v", err)
	}
	got := map[int64]int{}
	for _, r := range rows {
		got[r.MatchID] += r.Decisions
	}
	for _, matchID := range []int64{matchA, matchB} {
		detail, err := s.Stats().MatchDetail(ctx, "", matchID)
		if err != nil {
			t.Fatalf("MatchDetail(%d): %v", matchID, err)
		}
		if direct := detail.Player1.TotalDecisions + detail.Player2.TotalDecisions; got[matchID] != direct {
			t.Errorf("match %d: match_stats counts %d decisions, direct %d: CreateGames left a stale row", matchID, got[matchID], direct)
		}
	}
}
