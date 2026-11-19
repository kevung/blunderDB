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

// testStatsMatchWinTraceCube runs MatchWinTrace over cube decisions at an
// asymmetric score, where reading a response from the wrong side shows: a
// double and its take on the turned cube held by no one (the take stored from
// the responder's side, at the cube after the double), and a take the .xg
// fallback left on the doubler's own row (stored from the doubler's side, on
// the owned cube before the double, ADR-0083). Both read the doubler's
// equity at the cube before the double.
func testStatsMatchWinTraceCube(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	dca := func(nd float64) *domain.DoublingCubeAnalysis {
		return &domain.DoublingCubeAnalysis{CubefulNoDoubleEquity: nd, CubefulNoDoubleError: -0.2,
			CubefulDoubleTakeEquity: 0.6, CubefulDoublePassEquity: 1.0, CubefulDoublePassError: 0.4}
	}
	cubePos := func(score [2]int, cubeLog2, owner int, played string, nd float64) int64 {
		pos := statsDecisionPos(t, 0)
		pos.DecisionType = domain.CubeAction
		pos.Dice = [2]int{0, 0}
		pos.Score = score
		pos.Cube = domain.Cube{Value: cubeLog2, Owner: owner}
		id, err := s.Positions().Save(ctx, "", &pos)
		if err != nil {
			t.Fatalf("Save cube position %v: %v", score, err)
		}
		if err := s.Analyses().Save(ctx, "", id, &domain.PositionAnalysis{AnalysisType: "DoublingCube",
			PlayedCubeActions: []string{played}, DoublingCubeAnalysis: dca(nd)}); err != nil {
			t.Fatalf("Save cube analysis %v: %v", score, err)
		}
		return id
	}
	type row struct {
		pos    int64
		player int32
		action string
	}
	match := func(rows []row) storage.MatchWinTrace {
		m := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7,
			MatchDate: time.Date(2025, 6, 3, 0, 0, 0, 0, time.UTC)}
		matchID, err := s.Matches().Save(ctx, "", &m)
		if err != nil {
			t.Fatalf("Save match: %v", err)
		}
		gameID, err := s.Matches().CreateGame(ctx, "", &domain.Game{MatchID: matchID, GameNumber: 1, Winner: 1, PointsWon: 2})
		if err != nil {
			t.Fatalf("CreateGame: %v", err)
		}
		for i, r := range rows {
			if _, err := s.Matches().CreateMove(ctx, "", &domain.Move{GameID: gameID, MoveNumber: int32(i + 1), MoveType: "cube",
				PositionID: r.pos, Player: r.player, CubeAction: r.action}); err != nil {
				t.Fatalf("CreateMove %d: %v", i, err)
			}
		}
		trace, err := s.Stats().MatchWinTrace(ctx, "", matchID)
		if err != nil {
			t.Fatalf("MatchWinTrace: %v", err)
		}
		if len(trace.Decisions) != len(rows) {
			t.Fatalf("got %d points, want %d: %+v", len(trace.Decisions), len(rows), trace)
		}
		return trace
	}
	check := func(what string, pt storage.WinTracePoint, want float64) {
		t.Helper()
		if pt.MWCBefore == nil || pt.MWCAfter == nil {
			t.Fatalf("%s: no chances: %+v", what, pt)
		}
		if math.Abs(*pt.MWCBefore-want) > 1e-9 {
			t.Errorf("%s: player 1 before %v, want %v", what, *pt.MWCBefore, want)
		}
		if math.Abs(*pt.MWCAfter-*pt.MWCBefore) > 1e-9 {
			t.Errorf("%s: a right decision moves the chances from %v to %v", what, *pt.MWCBefore, *pt.MWCAfter)
		}
	}

	// Alice (player 1) is 5 away, Bob 3 away: current score 2-4. Alice
	// doubles on the centred cube, Bob takes on the turned one.
	trace := match([]row{
		{cubePos([2]int{5, 3}, 0, domain.None, "Double", 0.4), 1, "Double"},
		{cubePos([2]int{3, 5}, 1, domain.None, "Take", 0.4), -1, "Take"},
	})
	want := engine.EquityToMWC(0.6, 2, 4, 1, 7)
	check("double", trace.Decisions[0], want)
	check("take on the turned cube", trace.Decisions[1], want)

	// The .xg fallback: Alice's own row holds the take, on the cube she owns
	// at 2, stored from her side at the cube before the double.
	fallback := match([]row{{cubePos([2]int{5, 3}, 1, 0, "Take", 0.4), 1, "Take"}})
	check("take on the doubler's row", fallback.Decisions[0], engine.EquityToMWC(0.6, 2, 4, 2, 7))

	// Once the cube is offered, no double is no longer an option: a fallback
	// take reads the offered equity even where no double was worth more.
	tooGood := match([]row{{cubePos([2]int{5, 4}, 1, 0, "Take", 0.8), 1, "Take"}})
	check("take on the doubler's row, no double better", tooGood.Decisions[0], engine.EquityToMWC(0.6, 2, 3, 2, 7))
}
