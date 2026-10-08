package storagetest

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testStatsStudyEffect pins the before/after measure (ADR-0079) on both
// backends: a family is studied on the day of the first study mark on one of
// its positions, a match before that day and one after fill the two windows,
// and the change is the difference of their loss rates.
func testStatsStudyEffect(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	if _, err := s.Stats().DateRange(ctx, ""); errors.Is(err, storage.ErrInternal) {
		t.Skip("Stats not implemented on this backend")
	}
	game := func(date time.Time) int64 {
		m := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7, MatchDate: date}
		matchID, err := s.Matches().Save(ctx, "", &m)
		if err != nil {
			t.Fatalf("Save match: %v", err)
		}
		gameID, err := s.Matches().CreateGame(ctx, "", &domain.Game{MatchID: matchID, GameNumber: 1})
		if err != nil {
			t.Fatalf("CreateGame: %v", err)
		}
		return gameID
	}
	// The mark is dated now: the first match lies before it, the second after.
	before, after := game(time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)), game(time.Now().AddDate(1, 0, 0))
	var studiedID int64
	for slot := 0; slot < 6; slot++ {
		id := statsCheckerDecision(t, s, before, slot, 1, "24/18 13/11", 0.100+0.020*float64(slot), 8)
		if slot == 0 {
			studiedID = id
		}
	}
	for slot := 0; slot < 3; slot++ {
		statsCheckerDecision(t, s, after, slot, 1, "24/18 13/11", 0.100, 8)
	}
	if err := s.ImportBatches().SetStudied(ctx, "", studiedID, true); err != nil {
		t.Fatalf("SetStudied: %v", err)
	}

	got, err := s.Stats().StudyEffect(ctx, "", storage.StatsFilter{DecisionType: -1, PlayerName: "Alice"})
	if err != nil {
		t.Fatalf("StudyEffect: %v", err)
	}
	if len(got.Families) != 1 || got.Unstudied != 0 || got.MinDecisions != storage.StudyEffectMinDecisions {
		t.Fatalf("got %+v, want the gammon family alone, studied", got)
	}
	f := got.Families[0]
	if f.Theme != "gammon" || f.Studied != 1 || f.StudiedOn != time.Now().UTC().Format("2006-01-02") {
		t.Errorf("family %+v, want gammon studied today", f)
	}
	if f.Before.Errors != 6 || f.After.Errors != 3 || f.Before.Decisions != 6 || f.After.Decisions != 3 {
		t.Errorf("windows %+v / %+v, want 6 errors of 6 decisions, then 3 of 3", f.Before, f.After)
	}
	if f.Before.Loss <= 0 || math.Abs(f.Gain-(f.Before.Rate-f.After.Rate)) > 1e-12 || f.Low > f.Gain || f.High < f.Gain {
		t.Errorf("measure %+v", f)
	}
	if f.Verdict != storage.StudyEffectInsufficient {
		t.Errorf("Verdict %q with %d and %d decisions, want insufficient", f.Verdict, f.Before.Decisions, f.After.Decisions)
	}
}

// testStatsDirectionalBiases pins the signed biases (ADR-0079) on both
// backends: takes and passes against the bot's ruling, offers by score, and
// every checker decision with contact read or counted unread.
func testStatsDirectionalBiases(t *testing.T, s storage.Storage) {
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
	statsCheckerDecision(t, s, gameID, 0, 1, "24/18 13/11", 0.100, 0)
	statsCheckerDecision(t, s, gameID, 1, 1, "24/18 13/11", 0.050, 0)
	statsCubeDecision(t, s, gameID, 2, 1, "Take", "Double, Pass", 0.40, 1.20, 1.00)   // wrong take
	statsCubeDecision(t, s, gameID, 3, 1, "Pass", "Double, Take", 0.40, 0.60, 1.00)   // wrong pass
	statsCubeDecision(t, s, gameID, 4, 1, "Take", "Double, Take", 0.40, 0.60, 1.00)   // right take
	statsCubeDecision(t, s, gameID, 5, 1, "Double", "No Double", 0.40, 0.30, 1.00)    // premature
	statsCubeDecision(t, s, gameID, 6, 1, "Double", "Double, Take", 0.40, 0.60, 1.00) // right double
	statsCubeDecision(t, s, gameID, 7, -1, "Take", "Double, Pass", 0.40, 1.20, 1.00)  // Bob's

	got, err := s.Stats().DirectionalBiases(ctx, "", storage.StatsFilter{DecisionType: -1, PlayerName: "Alice"})
	if err != nil {
		t.Fatalf("DirectionalBiases: %v", err)
	}
	if tp := got.TakePass; tp.Decisions != 3 || tp.Plus != 1 || tp.Minus != 1 || tp.Verdict != storage.BiasInsufficient {
		t.Errorf("TakePass %+v, want 3 decisions, one wrong take, one wrong pass", tp)
	}
	if d := got.Doubles; d.Decisions != 2 || d.Plus != 1 || d.Minus != 0 {
		t.Errorf("Doubles %+v, want 2 offers, one premature", d)
	}
	if len(got.DoublesByScore) != 1 || got.DoublesByScore[0].MoverAway != 4 || got.DoublesByScore[0].Decisions != 2 {
		t.Errorf("DoublesByScore %+v, want the 4-away cell", got.DoublesByScore)
	}
	if got.Blots.Decisions+got.BlotsUnread != 2 {
		t.Errorf("Blots %+v unread %d, want the two checker decisions", got.Blots, got.BlotsUnread)
	}
}

// biasGame saves a one-game match for Alice and returns the game's id; a
// match length of 0 is money play.
func biasGame(t *testing.T, s storage.Storage, matchLength int32) int64 {
	t.Helper()
	ctx := context.Background()
	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: matchLength,
		MatchDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)}
	matchID, err := s.Matches().Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}
	gameID, err := s.Matches().CreateGame(ctx, "", &domain.Game{MatchID: matchID, GameNumber: 1})
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	return gameID
}

// testStatsDirectionalBiasesPerMatch pins that a bias reads each decision's
// error as played in its own match: two matches reach the same positions, the
// first plays right and the second errs, and the second's cost is counted —
// not the position's columns, which score the first match's play.
func testStatsDirectionalBiasesPerMatch(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	if _, err := s.Stats().DateRange(ctx, ""); errors.Is(err, storage.ErrInternal) {
		t.Skip("Stats not implemented on this backend")
	}
	first, second := biasGame(t, s, 7), biasGame(t, s, 7)

	cube := statsDecisionPos(t, 0)
	cube.DecisionType = domain.CubeAction
	cubeID, err := s.Positions().Save(ctx, "", &cube)
	if err != nil {
		t.Fatalf("Save cube position: %v", err)
	}
	checker := statsDecisionPos(t, 1)
	checker.Dice = [2]int{3, 1}
	checkerID, err := s.Positions().Save(ctx, "", &checker)
	if err != nil {
		t.Fatalf("Save checker position: %v", err)
	}
	for _, mv := range []domain.Move{
		{GameID: first, MoveNumber: 1, MoveType: "cube", PositionID: cubeID, Player: 1, CubeAction: "Pass"},
		{GameID: second, MoveNumber: 1, MoveType: "cube", PositionID: cubeID, Player: 1, CubeAction: "Take"},
		{GameID: first, MoveNumber: 2, MoveType: "checker", PositionID: checkerID, Player: 1, CheckerMove: "8/5 6/5"},
		{GameID: second, MoveNumber: 2, MoveType: "checker", PositionID: checkerID, Player: 1, CheckerMove: "24/23 13/10"},
	} {
		if _, err := s.Matches().CreateMove(ctx, "", &mv); err != nil {
			t.Fatalf("CreateMove: %v", err)
		}
	}
	zero, cost := 0.0, 0.150
	if err := s.Analyses().Save(ctx, "", cubeID, &domain.PositionAnalysis{
		PlayedCubeActions: []string{"Pass"},
		DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{BestCubeAction: "Double, Pass",
			CubefulNoDoubleEquity: 0.40, CubefulDoubleTakeEquity: 1.20, CubefulDoublePassEquity: 1.00,
			CubefulNoDoubleError: -0.100, CubefulDoubleTakeError: -0.100, CubefulDoublePassError: -0.100},
	}); err != nil {
		t.Fatalf("Save cube analysis: %v", err)
	}
	if err := s.Analyses().Save(ctx, "", checkerID, &domain.PositionAnalysis{
		PlayedMoves: []string{"8/5 6/5"},
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
			{Move: "8/5 6/5", Equity: 0.50, EquityError: &zero},
			{Move: "24/23 13/10", Equity: 0.35, EquityError: &cost},
		}},
	}); err != nil {
		t.Fatalf("Save checker analysis: %v", err)
	}

	got, err := s.Stats().DirectionalBiases(ctx, "", storage.StatsFilter{DecisionType: -1, PlayerName: "Alice"})
	if err != nil {
		t.Fatalf("DirectionalBiases: %v", err)
	}
	if tp := got.TakePass; tp.Decisions != 2 || tp.Plus != 1 || tp.PlusMP <= 0 {
		t.Errorf("TakePass %+v, want the second match's wrong take with its own cost", tp)
	}
	if b := got.Blots; b.Decisions != 2 || b.Plus != 1 || b.PlusMP != 150 || got.BlotsUnread != 0 {
		t.Errorf("Blots %+v unread %d, want the second match's bolder play at 150 mp", b, got.BlotsUnread)
	}
}

// testStatsDirectionalBiasesScores pins the score cells of the doubling bias:
// money play has its own cell, and a post-Crawford score reads as one away,
// never as the stored sentinels.
func testStatsDirectionalBiasesScores(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	if _, err := s.Stats().DateRange(ctx, ""); errors.Is(err, storage.ErrInternal) {
		t.Skip("Stats not implemented on this backend")
	}
	cubeAt := func(gameID int64, slot int, score [2]int) {
		t.Helper()
		pos := statsDecisionPos(t, slot)
		pos.DecisionType = domain.CubeAction
		pos.Score = score
		posID, err := s.Positions().Save(ctx, "", &pos)
		if err != nil {
			t.Fatalf("Save cube position: %v", err)
		}
		mv := domain.Move{GameID: gameID, MoveNumber: int32(slot), MoveType: "cube", PositionID: posID, Player: 1, CubeAction: "Double"}
		if _, err := s.Matches().CreateMove(ctx, "", &mv); err != nil {
			t.Fatalf("CreateMove: %v", err)
		}
		if err := s.Analyses().Save(ctx, "", posID, &domain.PositionAnalysis{
			PlayedCubeActions: []string{"Double"},
			DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{BestCubeAction: "No Double",
				CubefulNoDoubleEquity: 0.40, CubefulDoubleTakeEquity: 0.30, CubefulDoublePassEquity: 1.00},
		}); err != nil {
			t.Fatalf("Save cube analysis: %v", err)
		}
	}
	cubeAt(biasGame(t, s, 0), 0, [2]int{-1, -1})
	cubeAt(biasGame(t, s, 7), 1, [2]int{domain.PostCrawford, 3})

	got, err := s.Stats().DirectionalBiases(ctx, "", storage.StatsFilter{DecisionType: -1, PlayerName: "Alice"})
	if err != nil {
		t.Fatalf("DirectionalBiases: %v", err)
	}
	if len(got.DoublesByScore) != 2 {
		t.Fatalf("DoublesByScore %+v, want a money cell and a 1-away/3-away cell", got.DoublesByScore)
	}
	money, post := got.DoublesByScore[0], got.DoublesByScore[1]
	if !money.Money || money.MoverAway != 0 || money.OpponentAway != 0 || money.Decisions != 1 {
		t.Errorf("first cell %+v, want money play", money)
	}
	if post.Money || post.MoverAway != 1 || post.OpponentAway != 3 || post.Decisions != 1 {
		t.Errorf("second cell %+v, want 1-away/3-away", post)
	}
}
