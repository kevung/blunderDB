// Contract case for the best play a match's moves carry beside their cost:
// what the stored analysis ranks first for each Move's decision, so that a
// review shows "better: <play>" without running an engine.
// The table that runs it lives in contract.go.
package storagetest

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func testMatchMovesBestPlay(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	savePos := func(slot int, cube bool) int64 {
		t.Helper()
		p := statsDecisionPos(t, slot)
		if cube {
			p.DecisionType = domain.CubeAction
			p.Dice = [2]int{0, 0}
			p.Score = [2]int{slot % 7, 4} // without dice, only the score tells cube positions apart
		}
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save position %d: %v", slot, err)
		}
		return id
	}
	saveAnalysis := func(posID int64, a domain.PositionAnalysis) {
		t.Helper()
		if err := s.Analyses().Save(ctx, "", posID, &a); err != nil {
			t.Fatalf("Save analysis of %d: %v", posID, err)
		}
	}

	e60 := 0.060
	checker := savePos(10, false)
	saveAnalysis(checker, domain.PositionAnalysis{AnalysisType: "CheckerMove",
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
			{Move: "24/22 13/11", Equity: 0.500},
			{Move: "13/11 24/23", Equity: 0.440, EquityError: &e60},
		}}})
	bare := savePos(11, false) // never analysed
	// The stored errors rule, as they score error_mp; the label agrees here.
	doublePass := savePos(12, true)
	saveAnalysis(doublePass, domain.PositionAnalysis{AnalysisType: "DoublingCube",
		DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{BestCubeAction: "Double, Pass",
			CubefulNoDoubleEquity: 0.700, CubefulDoubleTakeEquity: 1.300, CubefulDoublePassEquity: 1.000,
			CubefulNoDoubleError: -0.300, CubefulDoubleTakeError: 0.300}})
	// No label: the errors decide alone.
	doubleTake := savePos(13, true)
	saveAnalysis(doubleTake, domain.PositionAnalysis{AnalysisType: "DoublingCube",
		DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{
			CubefulNoDoubleEquity: 0.500, CubefulDoubleTakeEquity: 0.800, CubefulDoublePassEquity: 1.000,
			CubefulNoDoubleError: -0.300, CubefulDoublePassError: 0.200}})
	// Too good contains "double" yet rules against the double.
	tooGood := savePos(14, true)
	saveAnalysis(tooGood, domain.PositionAnalysis{AnalysisType: "DoublingCube",
		DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{BestCubeAction: "Too good to double, pass",
			CubefulNoDoubleEquity: 1.400, CubefulDoubleTakeEquity: 1.800, CubefulDoublePassEquity: 1.000,
			CubefulDoubleTakeError: 0.400, CubefulDoublePassError: -0.400}})
	// An imported label can contradict the equities it travels with (BGF, XG
	// text): doubling is worth 0.640 against 0.620, whatever the label says.
	mislabelled := savePos(16, true)
	saveAnalysis(mislabelled, domain.PositionAnalysis{AnalysisType: "DoublingCube",
		DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{BestCubeAction: "No Double",
			CubefulNoDoubleEquity: 0.620, CubefulDoubleTakeEquity: 0.640, CubefulDoublePassEquity: 1.000,
			CubefulNoDoubleError: -0.020, CubefulDoublePassError: 0.360}})
	// Both doubling choices cost the same: neither is better than the one played.
	even := savePos(17, true)
	saveAnalysis(even, domain.PositionAnalysis{AnalysisType: "DoublingCube",
		DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{BestCubeAction: "Double, Take",
			CubefulNoDoubleEquity: 0.640, CubefulDoubleTakeEquity: 0.640, CubefulDoublePassEquity: 1.000,
			CubefulDoublePassError: 0.360}})

	type play struct {
		posID           int64
		player          int32
		moveType, label string
		best, bestCube  string
	}
	plays := []play{
		{checker, 1, "checker", "24/22 13/11", "24/22 13/11", ""},
		{checker, -1, "checker", "13/11 24/23", "24/22 13/11", ""},
		{checker, 1, "checker", "bar/20", "24/22 13/11", ""}, // not a candidate: best still named
		{bare, 1, "checker", "8/5 6/5", "", ""},
		{doublePass, 1, "cube", "No Double", "", "Double"},
		{doublePass, -1, "cube", "Take", "", "Pass"},
		{doubleTake, -1, "cube", "Double", "", "Double"},
		{doubleTake, 1, "cube", "Pass", "", "Take"},
		{tooGood, 1, "cube", "Double", "", "No Double"},
		{tooGood, -1, "cube", "Take", "", "Pass"},
		{mislabelled, 1, "cube", "Double", "", "Double"},
		{mislabelled, 1, "cube", "No Double", "", "Double"},
		{mislabelled, -1, "cube", "Pass", "", "Take"},
		{even, 1, "cube", "No Double", "", "No Double"},
		{even, 1, "cube", "Double", "", "Double"},
		{doublePass, 1, "checker", "13/9", "", ""}, // a checker play on a cube-only analysis
	}

	newMatch := func(name string) (int64, int64) {
		t.Helper()
		m := domain.Match{Player1Name: name, Player2Name: "Bob", MatchLength: 7,
			MatchDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)}
		matchID, err := s.Matches().Save(ctx, "", &m)
		if err != nil {
			t.Fatalf("Save match: %v", err)
		}
		gameID, err := s.Matches().CreateGame(ctx, "", &domain.Game{MatchID: matchID, GameNumber: 1, Winner: 1, PointsWon: 1})
		if err != nil {
			t.Fatalf("CreateGame: %v", err)
		}
		return matchID, gameID
	}
	matchID, gameID := newMatch("Alice")
	ids := make([]int64, len(plays))
	for i, p := range plays {
		mv := domain.Move{GameID: gameID, MoveNumber: int32(i + 1), MoveType: p.moveType, PositionID: p.posID, Player: p.player}
		if p.moveType == "cube" {
			mv.CubeAction = p.label
		} else {
			mv.CheckerMove = p.label
		}
		var err error
		if ids[i], err = s.Matches().CreateMove(ctx, "", &mv); err != nil {
			t.Fatalf("CreateMove %q: %v", p.label, err)
		}
	}

	check := func(read string, it func(func(*domain.Move, error) bool)) {
		t.Helper()
		byID := map[int64]*domain.Move{}
		for mv, err := range it {
			if err != nil {
				t.Fatalf("%s: %v", read, err)
			}
			byID[mv.ID] = mv
		}
		for i, p := range plays {
			mv := byID[ids[i]]
			if mv == nil {
				t.Errorf("%s: move %d missing", read, i)
				continue
			}
			if mv.BestMove != p.best || mv.BestCubeAction != p.bestCube {
				t.Errorf("%s: move %d (%s): best = %q / %q, want %q / %q",
					read, i, p.label, mv.BestMove, mv.BestCubeAction, p.best, p.bestCube)
			}
		}
	}
	check("MovesByMatch", s.Matches().MovesByMatch(ctx, "", matchID))
	check("Moves", s.Matches().Moves(ctx, "", gameID))

	// The cost and the best action come from one analysis, both ways: the
	// best play, played, costs nothing; any other scored play costs something.
	for mv, err := range s.Matches().Moves(ctx, "", gameID) {
		if err != nil {
			t.Fatalf("Moves: %v", err)
		}
		played, best := mv.CheckerMove, mv.BestMove
		if mv.MoveType == "cube" {
			played, best = mv.CubeAction, mv.BestCubeAction
		}
		if best == "" || mv.ErrorMP == nil {
			continue
		}
		if same := strings.EqualFold(best, played); same != (*mv.ErrorMP == 0) {
			t.Errorf("move %d plays %q, best %q, yet ErrorMP = %d", mv.ID, played, best, *mv.ErrorMP)
		}
	}

	// The same Position reached from another match names the same best play.
	otherMatch, otherGame := newMatch("Carol")
	if _, err := s.Matches().CreateMove(ctx, "", &domain.Move{GameID: otherGame, MoveNumber: 1,
		MoveType: "checker", PositionID: checker, Player: -1, CheckerMove: "13/11 24/23"}); err != nil {
		t.Fatalf("CreateMove other match: %v", err)
	}
	for mv, err := range s.Matches().MovesByMatch(ctx, "", otherMatch) {
		if err != nil {
			t.Fatalf("MovesByMatch other: %v", err)
		}
		if mv.BestMove != "24/22 13/11" {
			t.Errorf("other match: BestMove = %q, want 24/22 13/11", mv.BestMove)
		}
	}

	// RecordedMoves reads what was played, nothing the analysis says.
	for mv, err := range s.Matches().RecordedMoves(ctx, "", gameID) {
		if err != nil {
			t.Fatalf("RecordedMoves: %v", err)
		}
		if mv.BestMove != "" || mv.BestCubeAction != "" {
			t.Errorf("RecordedMoves: move %d carries best %q / %q", mv.ID, mv.BestMove, mv.BestCubeAction)
		}
	}
}

// CheckBestPlayTenantIsolation proves the best play a tenant's moves carry
// comes from that tenant's rows only. Row ids are drawn from sequences shared
// by every tenant, so two tenants never hold the same Position id and an
// analysis cannot cross over by its position_id; what can leak is a whole
// move list reached by another tenant's match or game id. Tenant b therefore
// asks for tenant a's match and game and must read nothing, then reads its own
// moves with its own best play. Run under row-level security by the
// PostgreSQL backend.
func CheckBestPlayTenantIsolation(t *testing.T, s storage.Storage, a, b string) {
	t.Helper()
	ctxA := tenantContext(t, context.Background(), a)
	ctxB := tenantContext(t, context.Background(), b)

	setUp := func(ctx context.Context, scope, best string) (matchID, gameID int64) {
		t.Helper()
		p := statsDecisionPos(t, 15)
		posID, err := s.Positions().Save(ctx, scope, &p)
		if err != nil {
			t.Fatalf("%s: Save position: %v", scope, err)
		}
		e := 0.050
		if err := s.Analyses().Save(ctx, scope, posID, &domain.PositionAnalysis{AnalysisType: "CheckerMove",
			CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
				{Move: best, Equity: 0.300},
				{Move: "13/11 24/23", Equity: 0.250, EquityError: &e},
			}}}); err != nil {
			t.Fatalf("%s: Save analysis: %v", scope, err)
		}
		m := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 5,
			MatchDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)}
		if matchID, err = s.Matches().Save(ctx, scope, &m); err != nil {
			t.Fatalf("%s: Save match: %v", scope, err)
		}
		if gameID, err = s.Matches().CreateGame(ctx, scope, &domain.Game{MatchID: matchID, GameNumber: 1, Winner: 1, PointsWon: 1}); err != nil {
			t.Fatalf("%s: CreateGame: %v", scope, err)
		}
		if _, err := s.Matches().CreateMove(ctx, scope, &domain.Move{GameID: gameID, MoveNumber: 1,
			MoveType: "checker", PositionID: posID, Player: 1, CheckerMove: "13/11 24/23"}); err != nil {
			t.Fatalf("%s: CreateMove: %v", scope, err)
		}
		return matchID, gameID
	}
	read := func(scope, via string, it func(func(*domain.Move, error) bool)) []string {
		t.Helper()
		var bests []string
		for mv, err := range it {
			if err != nil {
				t.Fatalf("%s: %s: %v", scope, via, err)
			}
			bests = append(bests, mv.BestMove)
		}
		return bests
	}

	matchA, gameA := setUp(ctxA, a, "24/22 13/11")
	matchB, gameB := setUp(ctxB, b, "8/5 6/5")

	if got := read(b, "MovesByMatch", s.Matches().MovesByMatch(ctxB, b, matchA)); len(got) != 0 {
		t.Errorf("tenant %s reads tenant %s's match: best plays %q leaked", b, a, got)
	}
	if got := read(b, "Moves", s.Matches().Moves(ctxB, b, gameA)); len(got) != 0 {
		t.Errorf("tenant %s reads tenant %s's game: best plays %q leaked", b, a, got)
	}
	for _, c := range []struct {
		scope    string
		ctx      context.Context
		matchID  int64
		gameID   int64
		wantBest string
	}{{a, ctxA, matchA, gameA, "24/22 13/11"}, {b, ctxB, matchB, gameB, "8/5 6/5"}} {
		for via, it := range map[string]func(func(*domain.Move, error) bool){
			"MovesByMatch": s.Matches().MovesByMatch(c.ctx, c.scope, c.matchID),
			"Moves":        s.Matches().Moves(c.ctx, c.scope, c.gameID),
		} {
			if got := read(c.scope, via, it); len(got) != 1 || got[0] != c.wantBest {
				t.Errorf("tenant %s %s: best plays %q, want its own [%s]", c.scope, via, got, c.wantBest)
			}
		}
	}
}
