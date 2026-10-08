// Contract case for the error of a decision as played in its match: a
// position reached by two matches with two different plays charges each
// match its own play, checker and cube alike, in every statistic. The table
// that runs it lives in contract.go.
package storagetest

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func testStatsPlayedDecisionPerMatch(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	checker := statsDecisionPos(t, 2)
	checkerID, err := s.Positions().Save(ctx, "", &checker)
	if err != nil {
		t.Fatalf("Save checker position: %v", err)
	}
	cube := statsDecisionPos(t, 3)
	cube.DecisionType = domain.CubeAction
	cube.Dice = [2]int{0, 0}
	cubeID, err := s.Positions().Save(ctx, "", &cube)
	if err != nil {
		t.Fatalf("Save cube position: %v", err)
	}

	newMatch := func(p1, p2 string, day int) int64 {
		m := domain.Match{Player1Name: p1, Player2Name: p2, MatchLength: 7,
			MatchDate: time.Date(2025, 6, day, 0, 0, 0, 0, time.UTC)}
		id, err := s.Matches().Save(ctx, "", &m)
		if err != nil {
			t.Fatalf("Save match: %v", err)
		}
		g := domain.Game{MatchID: id, GameNumber: 1, Winner: 1, PointsWon: 1}
		gameID, err := s.Matches().CreateGame(ctx, "", &g)
		if err != nil {
			t.Fatalf("CreateGame: %v", err)
		}
		return gameID<<32 | id
	}
	play := func(packed int64, n int32, cubeAction, checkerMove string) {
		gameID := packed >> 32
		cubeMv := domain.Move{GameID: gameID, MoveNumber: n, MoveType: "cube", PositionID: cubeID, Player: 1, CubeAction: cubeAction}
		if _, err := s.Matches().CreateMove(ctx, "", &cubeMv); err != nil {
			t.Fatalf("CreateMove cube: %v", err)
		}
		checkerMv := domain.Move{GameID: gameID, MoveNumber: n + 1, MoveType: "checker", PositionID: checkerID, Player: 1, CheckerMove: checkerMove}
		if _, err := s.Matches().CreateMove(ctx, "", &checkerMv); err != nil {
			t.Fatalf("CreateMove checker: %v", err)
		}
	}

	// Match A plays the best of both decisions; it is recorded before the
	// analyses exist, so its moves are scored when they are written.
	a := newMatch("Alice", "Bob", 1)
	play(a, 1, "Double", "24/22 13/11")

	// The analyses name match A's plays, as a file imported from it would.
	e150 := 0.150
	if err := s.Analyses().Save(ctx, "", checkerID, &domain.PositionAnalysis{
		AnalysisType: "CheckerMove", PlayedMoves: []string{"24/22 13/11"},
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
			{Move: "24/22 13/11", Equity: 0.500},
			{Move: "13/11 24/23", Equity: 0.350, EquityError: &e150},
		}},
	}); err != nil {
		t.Fatalf("Save checker analysis: %v", err)
	}
	if err := s.Analyses().Save(ctx, "", cubeID, &domain.PositionAnalysis{
		AnalysisType: "DoublingCube", PlayedCubeActions: []string{"Double"},
		DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{
			CubefulNoDoubleEquity: 0.500, CubefulNoDoubleError: 0.300,
			CubefulDoubleTakeEquity: 0.800, CubefulDoubleTakeError: 0,
			CubefulDoublePassEquity: 1.000, CubefulDoublePassError: 0.200,
		},
	}); err != nil {
		t.Fatalf("Save cube analysis: %v", err)
	}

	// Match B reaches the same two positions and plays both wrong: a close
	// no-double (300 mP) and the 150 mP checker play.
	b := newMatch("Carol", "Dave", 2)
	play(b, 1, "No Double", "13/11 24/23")
	matchA, matchB := a&(1<<32-1), b&(1<<32-1)

	const wantPRB = 500 * 0.450 / 2 // 450 mP over two decisions
	badges, err := s.Stats().MatchBadges(ctx, "", []int64{matchA, matchB})
	if err != nil {
		t.Fatalf("MatchBadges: %v", err)
	}
	if got := badges[matchA].PR; got != 0 {
		t.Errorf("match A played the best of both: PR %v, want 0", got)
	}
	if got := badges[matchB].PR; math.Abs(got-wantPRB) > 1e-9 {
		t.Errorf("match B: PR %v, want %v (its own plays, not match A's)", got, wantPRB)
	}
	if badges[matchA].MWCLoss != 0 || badges[matchB].MWCLoss <= 0 {
		t.Errorf("MWC loss: A %v (want 0), B %v (want > 0)", badges[matchA].MWCLoss, badges[matchB].MWCLoss)
	}

	for _, m := range []struct {
		id      int64
		pr, mwc float64
	}{{matchA, 0, 0}, {matchB, wantPRB, badges[matchB].MWCLoss}} {
		d, err := s.Stats().MatchDetail(ctx, "", m.id)
		if err != nil {
			t.Fatalf("MatchDetail %d: %v", m.id, err)
		}
		if d.Player1.TotalDecisions != 2 || math.Abs(d.Player1.PR-m.pr) > 1e-9 || math.Abs(d.Player1.MWCLoss-m.mwc) > 1e-12 {
			t.Errorf("MatchDetail %d: %d decisions, PR %v, MWC %v; want 2, %v, %v",
				m.id, d.Player1.TotalDecisions, d.Player1.PR, d.Player1.MWCLoss, m.pr, m.mwc)
		}
		losses, err := s.Stats().MatchDecisionLosses(ctx, "", m.id)
		if err != nil {
			t.Fatalf("MatchDecisionLosses %d: %v", m.id, err)
		}
		var sum float64
		for _, l := range losses {
			if l.MWCLoss != nil {
				sum += *l.MWCLoss
			}
		}
		if math.Abs(sum-m.mwc) > 1e-12 {
			t.Errorf("match %d: per-decision losses add up to %v, the badge says %v", m.id, sum, m.mwc)
		}
	}

	// The library statistics, through match_stats and its cells.
	if _, err := s.Stats().FillMatchStats(ctx, "", nil); err != nil {
		t.Fatalf("FillMatchStats: %v", err)
	}
	for _, c := range []struct {
		player string
		pr     float64
	}{{"Alice", 0}, {"Carol", wantPRB}} {
		res, err := s.Stats().Compute(ctx, "", storage.StatsFilter{PlayerName: c.player, DecisionType: -1})
		if err != nil {
			t.Fatalf("Compute %s: %v", c.player, err)
		}
		if res.Totals.NumDecisions != 2 || math.Abs(res.PRGlobal-c.pr) > 1e-9 {
			t.Errorf("Compute %s: %d decisions, PR %v; want 2, %v", c.player, res.Totals.NumDecisions, res.PRGlobal, c.pr)
		}
	}
}
