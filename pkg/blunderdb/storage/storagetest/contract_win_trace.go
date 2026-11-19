package storagetest

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testStatsMatchWinTrace runs MatchWinTrace over a 7-point match and a money
// session: player 1's chances come from the best play's normalised equity,
// turned over for player 2, and move by the decision's loss; a decision
// without analysis has no value, and money has none at all.
func testStatsMatchWinTrace(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	save := func(slot int, score [2]int, errEq float64, analysed bool) int64 {
		pos := statsDecisionPos(t, slot)
		pos.Score = score
		id, err := s.Positions().Save(ctx, "", &pos)
		if err != nil {
			t.Fatalf("Save position (slot %d): %v", slot, err)
		}
		if !analysed {
			return id
		}
		ca := &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
			{Move: "24/22 13/11", Equity: 0.5},
			{Move: "13/11 24/23", Equity: 0.5 - errEq, EquityError: &errEq},
		}}
		if err := s.Analyses().Save(ctx, "", id, &domain.PositionAnalysis{AnalysisType: "CheckerMove",
			PlayedMoves: []string{"13/11 24/23"}, CheckerAnalysis: ca}); err != nil {
			t.Fatalf("Save analysis (slot %d): %v", slot, err)
		}
		return id
	}
	match := func(length int, posIDs []int64, players []int32) int64 {
		m := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: int32(length),
			MatchDate: time.Date(2025, 6, 2, 0, 0, 0, 0, time.UTC)}
		matchID, err := s.Matches().Save(ctx, "", &m)
		if err != nil {
			t.Fatalf("Save match: %v", err)
		}
		g := domain.Game{MatchID: matchID, GameNumber: 1, Winner: 1, PointsWon: 1}
		gameID, err := s.Matches().CreateGame(ctx, "", &g)
		if err != nil {
			t.Fatalf("CreateGame: %v", err)
		}
		for i, id := range posIDs {
			mv := domain.Move{GameID: gameID, MoveNumber: int32(i + 1), MoveType: "checker",
				PositionID: id, Player: players[i], CheckerMove: "13/11 24/23"}
			if _, err := s.Matches().CreateMove(ctx, "", &mv); err != nil {
				t.Fatalf("CreateMove %d: %v", i, err)
			}
		}
		return matchID
	}

	// 4-away each: current score 3-3 in a 7-point match, cube 1.
	matchID := match(7, []int64{
		save(2, [2]int{4, 4}, 0.060, true),
		save(3, [2]int{4, 4}, 0.150, true),
		save(4, [2]int{4, 4}, 0, false),
	}, []int32{1, -1, 1})
	got, err := s.Stats().MatchWinTrace(ctx, "", matchID)
	if err != nil {
		t.Fatalf("MatchWinTrace: %v", err)
	}
	if got.Money || len(got.Decisions) != 3 {
		t.Fatalf("got %+v, want 3 decisions of a match", got)
	}
	best := engine.EquityToMWC(0.5, 3, 3, 1, 7)
	for i, c := range []struct {
		player       int
		before, loss float64
	}{{0, best, engine.ConvertEMGLossToMWCLoss(60, 3, 3, 0, 1, 7)}, {1, 1 - best, -engine.ConvertEMGLossToMWCLoss(150, 3, 3, 0, 1, 7)}} {
		pt := got.Decisions[i]
		if pt.Player != c.player || !pt.Analysed || pt.MWCBefore == nil || pt.MWCAfter == nil {
			t.Fatalf("decision %d: %+v", i, pt)
		}
		if math.Abs(*pt.MWCBefore-c.before) > 1e-9 || math.Abs(*pt.MWCBefore-*pt.MWCAfter-c.loss) > 1e-9 {
			t.Errorf("decision %d: before %v after %v, want before %v moved by %v", i, *pt.MWCBefore, *pt.MWCAfter, c.before, c.loss)
		}
	}
	if pt := got.Decisions[2]; pt.Analysed || pt.MWCBefore != nil || pt.MWCAfter != nil {
		t.Errorf("unanalysed decision carries values: %+v", pt)
	}

	moneyID := match(0, []int64{save(5, [2]int{-1, -1}, 0.060, true)}, []int32{1})
	money, err := s.Stats().MatchWinTrace(ctx, "", moneyID)
	if err != nil {
		t.Fatalf("MatchWinTrace (money): %v", err)
	}
	if !money.Money || len(money.Decisions) != 1 {
		t.Fatalf("money: got %+v", money)
	}
	if pt := money.Decisions[0]; !pt.Analysed || pt.MWCBefore != nil || pt.MWCAfter != nil {
		t.Errorf("money decision: %+v, want analysed without chances", pt)
	}

	none, err := s.Stats().MatchWinTrace(ctx, "", 987654)
	if err != nil || none.Money || len(none.Decisions) != 0 {
		t.Errorf("unknown match: %+v, %v", none, err)
	}
}
