package storagetest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// statsCheckerDecision saves one checker decision whose played move costs
// errEq and gives up gammonSwing percentage points of gammon chances against
// the best move.
func statsCheckerDecision(t *testing.T, s storage.Storage, gameID int64, slot int,
	player int32, played string, errEq, gammonSwing float64) int64 {
	t.Helper()
	ctx := context.Background()
	pos := statsDecisionPos(t, slot)
	posID, err := s.Positions().Save(ctx, "", &pos)
	if err != nil {
		t.Fatalf("Save position (slot %d): %v", slot, err)
	}
	mv := domain.Move{GameID: gameID, MoveNumber: int32(slot), MoveType: "checker",
		PositionID: posID, Player: player, CheckerMove: played}
	if _, err := s.Matches().CreateMove(ctx, "", &mv); err != nil {
		t.Fatalf("CreateMove (slot %d): %v", slot, err)
	}
	zero := 0.0
	a := domain.PositionAnalysis{
		PlayedMoves: []string{played},
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
			{Move: "8/5 6/5", Equity: 0.50, EquityError: &zero, PlayerWinChance: 55, PlayerGammonChance: 20},
			{Move: played, Equity: 0.50 - errEq, EquityError: &errEq, PlayerWinChance: 55, PlayerGammonChance: 20 - gammonSwing},
		}},
	}
	if err := s.Analyses().Save(ctx, "", posID, &a); err != nil {
		t.Fatalf("Save analysis (slot %d): %v", slot, err)
	}
	return posID
}

// testStatsRecurringErrors pins the grouping of errors by plan of play and
// theme: a checker theme comes from the explanation rules, a cube theme from
// the direction of the cube error, an error no rule names lands in "none",
// and a play below the Error threshold joins no group while still counting
// in the PR denominator.
func testStatsRecurringErrors(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	if _, err := s.Stats().DateRange(ctx, ""); errors.Is(err, storage.ErrInternal) {
		t.Skip("Stats not implemented on this backend")
	}
	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7,
		MatchDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)}
	matchID, err := s.Matches().Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}
	gameID, err := s.Matches().CreateGame(ctx, "", &domain.Game{MatchID: matchID, GameNumber: 1})
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}

	gammonID := statsCheckerDecision(t, s, gameID, 0, 1, "24/18 13/11", 0.200, 8)        // gammon, Alice
	noneID := statsCheckerDecision(t, s, gameID, 1, -1, "24/14", 0.080, 0)               // unreadable move: no theme, Bob
	statsCheckerDecision(t, s, gameID, 2, 1, "13/9", 0.010, 0)                           // below the threshold
	statsCubeDecision(t, s, gameID, 3, 1, "No Double", "Double, Take", 0.40, 0.55, 1.00) // missed, Alice
	statsCubeDecision(t, s, gameID, 4, -1, "Take", "Double, Pass", 0.90, 1.30, 1.00)     // wrong take, Bob

	res, err := s.Stats().RecurringErrors(ctx, "", storage.StatsFilter{DecisionType: -1})
	if err != nil {
		t.Fatalf("RecurringErrors: %v", err)
	}
	if res.NumDecisions != 5 {
		t.Errorf("NumDecisions = %d, want 5 (the play below the threshold still counts)", res.NumDecisions)
	}
	type key struct{ kind, theme string }
	got := map[key]storage.RecurringErrorGroup{}
	for _, g := range res.Groups {
		got[key{g.Kind, g.Theme}] = g
	}
	want := []key{{"checker", "gammon"},
		{"cube", storage.CubeCellOfferMissed}, {"cube", storage.CubeCellAnswerWrongTake}}
	for _, k := range want {
		if _, ok := got[k]; !ok {
			t.Errorf("no %s/%s group in %+v", k.kind, k.theme, res.Groups)
		}
	}
	if len(res.Groups) != len(want) {
		t.Errorf("got %d groups, want %d: %+v", len(res.Groups), len(want), res.Groups)
	}
	g := got[key{"checker", "gammon"}]
	if g.Count != 1 || g.SumErrorMP != 200 || len(g.PositionIDs) != 1 || g.PositionIDs[0] != gammonID {
		t.Errorf("gammon group = %+v, want one decision of 200 mp at position %d", g, gammonID)
	}
	if want := 500 * 200.0 / 1000 / 5; g.PRCost != want {
		t.Errorf("gammon PRCost = %v, want %v (the PR formula over every counted decision)", g.PRCost, want)
	}
	// The errors no rule names stay out of the ranking, listed apart.
	if len(res.Unthemed) != 1 {
		t.Fatalf("Unthemed = %+v, want one group", res.Unthemed)
	}
	if n := res.Unthemed[0]; n.Theme != storage.RecurringThemeNone || n.Kind != "" ||
		n.SumErrorMP != 80 || len(n.PositionIDs) != 1 || n.PositionIDs[0] != noneID {
		t.Errorf("unthemed group = %+v, want one 80 mp error at position %d", n, noneID)
	}
	for i := 1; i < len(res.Groups); i++ {
		if res.Groups[i].SumErrorMP > res.Groups[i-1].SumErrorMP {
			t.Errorf("groups not ranked by summed cost: %+v", res.Groups)
		}
	}

	alice, err := s.Stats().RecurringErrors(ctx, "", storage.StatsFilter{DecisionType: -1, PlayerName: "Alice"})
	if err != nil {
		t.Fatalf("RecurringErrors(Alice): %v", err)
	}
	if len(alice.Unthemed) != 0 {
		t.Errorf("Alice has Bob's unthemed error: %+v", alice.Unthemed)
	}
	for _, g := range alice.Groups {
		if g.Theme == storage.CubeCellAnswerWrongTake {
			t.Errorf("Alice's errors include Bob's group %+v", g)
		}
	}
	if len(alice.Groups) != 2 {
		t.Errorf("Alice has %d groups, want 2 (gammon, missed double): %+v", len(alice.Groups), alice.Groups)
	}
}
