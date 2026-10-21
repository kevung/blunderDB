package storagetest

import (
	"context"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// multiPlayedProber is the drift probe of a SQL backend: the positions whose
// multi_played indicator differs from a direct count of their moves. A
// backend without the indicator (storage/remote) skips the case.
type multiPlayedProber interface {
	MultiPlayedDrift(ctx context.Context, scope string) ([]int64, error)
}

// testMultiPlayedFollowsWrites holds position.multi_played to its definition
// across the writes that change who played what on a position: moves added
// one by one and in a batch, on positions shared by two matches, the players
// of a match swapped (and back), and the match deleted.
func testMultiPlayedFollowsWrites(t *testing.T, s storage.Storage) {
	probe, ok := s.Search().(multiPlayedProber)
	if !ok {
		t.Skip("backend keeps no multi_played indicator")
	}
	ctx := context.Background()
	ms := s.Matches()

	mkGame := func(n int) (matchID, gameID int64) {
		m := domain.Match{Player1Name: "me", Player2Name: "them", MatchLength: 7,
			MatchDate: time.Date(2025, 7, n, 0, 0, 0, 0, time.UTC)}
		matchID, err := ms.Save(ctx, "", &m)
		if err != nil {
			t.Fatalf("Save match %d: %v", n, err)
		}
		gameID, err = ms.CreateGame(ctx, "", &domain.Game{MatchID: matchID, GameNumber: 1, Winner: 1, PointsWon: 1})
		if err != nil {
			t.Fatalf("CreateGame %d: %v", n, err)
		}
		return matchID, gameID
	}
	_, game1 := mkGame(1)
	match2, game2 := mkGame(2)

	save := func(p domain.Position) int64 {
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save position: %v", err)
		}
		return id
	}
	move := func(gameID, posID int64, n, player int32, checkerMove, cubeAction string) *domain.Move {
		moveType := "checker"
		if cubeAction != "" {
			moveType = "cube"
		}
		return &domain.Move{GameID: gameID, MoveNumber: n, MoveType: moveType, PositionID: posID,
			Player: player, CheckerMove: checkerMove, CubeAction: cubeAction}
	}
	create := func(mv *domain.Move) {
		if _, err := ms.CreateMove(ctx, "", mv); err != nil {
			t.Fatalf("CreateMove: %v", err)
		}
	}
	noDrift := func(stage string) {
		t.Helper()
		drift, err := probe.MultiPlayedDrift(ctx, "")
		if err != nil {
			t.Fatalf("%s: MultiPlayedDrift: %v", stage, err)
		}
		if len(drift) > 0 {
			t.Errorf("%s: multi_played differs from the moves on positions %v", stage, drift)
		}
	}

	same := save(statsDecisionPos(t, 0))       // one play, twice
	twoWays := save(statsDecisionPos(t, 1))    // two plays, one per match
	byOpponent := save(statsDecisionPos(t, 2)) // two plays, by player 2
	once := save(statsDecisionPos(t, 3))
	cube := save(cubePos())

	create(move(game1, same, 1, 1, "13/11 24/23", ""))
	create(move(game1, twoWays, 2, 1, "13/11 24/23", ""))
	create(move(game1, once, 3, 1, "8/6 6/4", ""))
	create(move(game1, cube, 4, 1, "", "Double"))
	noDrift("first match")

	if err := ms.CreateMoves(ctx, "", []*domain.Move{
		move(game2, same, 1, 1, "13/11 24/23", ""),
		move(game2, twoWays, 2, 1, "13/11 6/4", ""),
		move(game2, byOpponent, 3, -1, "13/11 24/23", ""),
		move(game2, byOpponent, 4, -1, "13/11 6/4", ""),
		move(game2, cube, 5, 1, "", "No double"),
	}); err != nil {
		t.Fatalf("CreateMoves: %v", err)
	}
	noDrift("second match")

	// The swap hands the opponent's two plays to player 1 and takes the
	// second match's plays away from them.
	if err := ms.SwapPlayers(ctx, "", match2); err != nil {
		t.Fatalf("SwapPlayers: %v", err)
	}
	noDrift("players swapped")
	if err := ms.SwapPlayers(ctx, "", match2); err != nil {
		t.Fatalf("SwapPlayers back: %v", err)
	}
	noDrift("players swapped back")

	if err := ms.DeleteCascade(ctx, "", match2); err != nil {
		t.Fatalf("DeleteCascade: %v", err)
	}
	noDrift("second match deleted")
}
