// Contract case for MatchesByPosition: which matches have a move on a position.
package storagetest

import (
	"context"
	"slices"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func testMatchesByPosition(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ms := s.Matches()

	shared := checkerPos()
	sharedID, err := s.Positions().Save(ctx, "", &shared)
	if err != nil {
		t.Fatal(err)
	}
	alone := cubePos()
	aloneID, err := s.Positions().Save(ctx, "", &alone)
	if err != nil {
		t.Fatal(err)
	}

	var matchIDs []int64
	for i := range 3 {
		mid, err := ms.Save(ctx, "", &domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 5})
		if err != nil {
			t.Fatal(err)
		}
		matchIDs = append(matchIDs, mid)
		if i == 2 {
			continue // a match with no move on the position
		}
		gid, err := ms.CreateGame(ctx, "", &domain.Game{MatchID: mid, GameNumber: 1})
		if err != nil {
			t.Fatal(err)
		}
		// Two moves on the same position in one match: it is listed once.
		for n := range 2 {
			if _, err := ms.CreateMove(ctx, "", &domain.Move{GameID: gid, MoveNumber: int32(n + 1), MoveType: "checker", PositionID: sharedID, Player: 1}); err != nil {
				t.Fatal(err)
			}
		}
	}

	got, err := ms.MatchesByPosition(ctx, "", sharedID)
	if err != nil || !slices.Equal(got, matchIDs[:2]) {
		t.Errorf("MatchesByPosition(shared) = %v, %v; want %v", got, err, matchIDs[:2])
	}
	for name, id := range map[string]int64{"unreferenced": aloneID, "unknown": 999999} {
		got, err := ms.MatchesByPosition(ctx, "", id)
		if err != nil || got == nil || len(got) != 0 {
			t.Errorf("MatchesByPosition(%s) = %#v, %v; want an empty, non-nil slice", name, got, err)
		}
	}

	// Once the match is gone, the position is free again.
	if err := ms.DeleteCascade(ctx, "", matchIDs[0]); err != nil {
		t.Fatal(err)
	}
	if got, err := ms.MatchesByPosition(ctx, "", sharedID); err != nil || !slices.Equal(got, matchIDs[1:2]) {
		t.Errorf("after delete MatchesByPosition = %v, %v; want %v", got, err, matchIDs[1:2])
	}
}
