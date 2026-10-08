package storagetest

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testStatsRollingOrderAcrossGames: move numbers restart in every game, so
// "the most recent N decisions" must be ordered by game before move number.
// Game 1 holds four costly decisions, game 2 four flawless ones; the five
// latest decisions are all of game 2 plus the last of game 1, which is a
// single 0.2 error: PR 500*0.2/5 = 20. Ordered by move number alone the
// window would take two of game 1's.
func testStatsRollingOrderAcrossGames(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	m := domain.Match{Player1Name: "Ann", Player2Name: "Abe", MatchLength: 7,
		MatchDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)}
	matchID, err := s.Matches().Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}
	slot := 0
	for gameNumber := 1; gameNumber <= 2; gameNumber++ {
		gameID, err := s.Matches().CreateGame(ctx, "", &domain.Game{MatchID: matchID, GameNumber: int32(gameNumber), Winner: 1, PointsWon: 1})
		if err != nil {
			t.Fatalf("CreateGame: %v", err)
		}
		lost := 0.0
		if gameNumber == 1 {
			lost = 0.2
		}
		for moveNumber := 1; moveNumber <= 4; moveNumber++ {
			pos := statsDecisionPos(t, slot)
			slot++
			posID, err := s.Positions().Save(ctx, "", &pos)
			if err != nil {
				t.Fatalf("Save position: %v", err)
			}
			mv := domain.Move{GameID: gameID, MoveNumber: int32(moveNumber), MoveType: "checker",
				PositionID: posID, Player: 1, CheckerMove: "13/11 24/23"}
			if _, err := s.Matches().CreateMove(ctx, "", &mv); err != nil {
				t.Fatalf("CreateMove: %v", err)
			}
			a := domain.PositionAnalysis{
				PlayedMoves: []string{"13/11 24/23"},
				CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
					{Move: "8/6 6/4", Equity: 0.50},
					{Move: "13/11 24/23", Equity: 0.50 - lost, EquityError: &lost},
				}},
			}
			if err := s.Analyses().Save(ctx, "", posID, &a); err != nil {
				t.Fatalf("Save analysis: %v", err)
			}
		}
	}
	res, err := s.Stats().Compute(ctx, "", storage.StatsFilter{DecisionType: -1})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if got := res.PRRolling[5]; math.Abs(got-20) > 1e-6 {
		t.Errorf("rolling PR over the last 5 decisions = %v, want 20", got)
	}
}
