// Contract case for the played actions a Performance Rating is a sum of.
// The table that runs it lives in contract.go.
package storagetest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testAnalysisTakesPlayedActionsFromTheMatch pins issue #268: a match imported
// WITHOUT an analysis, then analysed here, must get a Performance Rating.
//
// The failure this replaces was silent and looked like success. gammonNet is
// handed a position, and a position does not remember what anybody did with
// it, so an analysis computed here carries no played move. best_move_equity_error
// — the column PR is a sum of — was therefore left at zero on every decision,
// and the whole match scored a flawless 0.00. Nothing errored; the number was
// simply the number for "nobody ever said which move was played".
//
// The `move` table is the other half, written at import whether or not any
// analysis came with the file. The store now reads it when the analysis is
// silent (engine.PlayedActionsFor), and this case is the contract both
// backends answer to: same fixture, same PR.
func testAnalysisTakesPlayedActionsFromTheMatch(t *testing.T, s storage.Storage) {
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

	// The played move is the SECOND-best candidate, so the decision has a real
	// error to lose. A fixture where the best move was played would pass with
	// the defect still in place.
	const played, best = "13/11 24/23", "8/6 6/5"
	loss := 0.080

	pos := statsDecisionPos(t, 0)
	pos.DecisionType = domain.CheckerAction
	posID, err := s.Positions().Save(ctx, "", &pos)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}
	mv := domain.Move{GameID: gameID, MoveNumber: 1, MoveType: "checker",
		PositionID: posID, Player: 1, CheckerMove: played}
	if _, err := s.Matches().CreateMove(ctx, "", &mv); err != nil {
		t.Fatalf("CreateMove: %v", err)
	}

	// The analysis an engine computes here: ranked candidates, and NOTHING
	// about what was played.
	a := domain.PositionAnalysis{
		AnalysisType: "CheckerMove",
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
			{Index: 0, Move: best, Equity: 0.400},
			{Index: 1, Move: played, Equity: 0.320, EquityError: &loss},
		}},
	}
	if err := s.Analyses().Save(ctx, "", posID, &a); err != nil {
		if errors.Is(err, storage.ErrInternal) {
			t.Skip("Analyses not implemented on this backend")
		}
		t.Fatalf("Save analysis: %v", err)
	}

	// One unforced decision losing 0.080: PR = 500 × 80 / 1000 / 1 = 40.
	res, err := s.Stats().Compute(ctx, "", storage.StatsFilter{DecisionType: -1})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if res.Totals.NumDecisions != 1 {
		t.Fatalf("checker decisions counted: got %d, want 1", res.Totals.NumDecisions)
	}
	if res.PRChecker != 40 {
		t.Errorf("PRChecker = %v, want 40 — the played move's error never reached the column", res.PRChecker)
	}

	// And the repair reaches the same answer from the same two halves: a
	// database analysed before this rule existed is fixed without re-running
	// the engine over it. Nothing is wrong here, so nothing is rewritten.
	n, err := s.Analyses().RepairDenormalisedColumns(ctx, "")
	if err != nil {
		t.Fatalf("Repair: %v", err)
	}
	if n != 0 {
		t.Errorf("repair of a sound database touched %d row(s), want 0", n)
	}
}
