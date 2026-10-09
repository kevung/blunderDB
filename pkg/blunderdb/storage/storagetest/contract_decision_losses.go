// Contract case for the per-decision MWC loss of a Match: every Move listed in
// Transcript order, a Move without a figure left unscored rather than at zero,
// and each player's losses adding up to the Match's badge.
// The table that runs it lives in contract.go.
package storagetest

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func testStatsMatchDecisionLosses(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7,
		MatchDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)}
	matchID, err := s.Matches().Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}
	var gameIDs [2]int64
	for i := range gameIDs {
		g := domain.Game{MatchID: matchID, GameNumber: int32(i + 1), Winner: 1, PointsWon: 1}
		if gameIDs[i], err = s.Matches().CreateGame(ctx, "", &g); err != nil {
			t.Fatalf("CreateGame: %v", err)
		}
	}

	// One Position per play: the statistics score a Position by the error
	// stored with its analysis, and the losses must add up to them.
	scored := func(slot int, err float64) int64 {
		pos := statsDecisionPos(t, slot)
		id, e := s.Positions().Save(ctx, "", &pos)
		if e != nil {
			t.Fatalf("Save position (slot %d): %v", slot, e)
		}
		ca := &domain.CheckerAnalysis{Moves: []domain.CheckerMove{{Move: "24/22 13/11", Equity: 0.5}}}
		played := "24/22 13/11"
		other := err
		if err > 0 {
			played = "13/11 24/23"
		} else {
			other = 0.1 // a second candidate keeps the decision unforced
		}
		ca.Moves = append(ca.Moves, domain.CheckerMove{Move: "13/11 24/23", Equity: 0.5 - other, EquityError: &other})
		if e := s.Analyses().Save(ctx, "", id, &domain.PositionAnalysis{AnalysisType: "CheckerMove", PlayedMoves: []string{played}, CheckerAnalysis: ca}); e != nil {
			t.Fatalf("Save analysis (slot %d): %v", slot, e)
		}
		return id
	}
	unanalysed := statsDecisionPos(t, 8)
	unanalysedID, err := s.Positions().Save(ctx, "", &unanalysed)
	if err != nil {
		t.Fatalf("Save unanalysed position: %v", err)
	}

	type play struct {
		game   int
		posID  int64
		player int32
		label  string
	}
	plays := []play{
		{0, scored(2, 0.060), 1, "13/11 24/23"},
		{0, scored(3, 0.150), -1, "13/11 24/23"},
		{0, unanalysedID, 1, "13/11 24/23"},
		{1, scored(4, 0.020), -1, "13/11 24/23"},
		{1, scored(5, 0.300), 1, "13/11 24/23"},
		{1, scored(6, 0), -1, "24/22 13/11"}, // the best play: a loss of zero, scored
	}
	for i, p := range plays {
		mv := domain.Move{GameID: gameIDs[p.game], MoveNumber: int32(i + 1), MoveType: "checker",
			PositionID: p.posID, Player: p.player, CheckerMove: p.label}
		if _, err := s.Matches().CreateMove(ctx, "", &mv); err != nil {
			t.Fatalf("CreateMove %d: %v", i, err)
		}
	}
	// Another match on the same library: its Moves must not leak in.
	statsFixtureMatch(t, s, 10, "Carol", "Dave")

	got, err := s.Stats().MatchDecisionLosses(ctx, "", matchID)
	if err != nil {
		t.Fatalf("MatchDecisionLosses: %v", err)
	}
	if len(got) != len(plays) {
		t.Fatalf("got %d decisions, want %d: %+v", len(got), len(plays), got)
	}
	for i, d := range got {
		if d.GameNumber != plays[i].game+1 || d.MoveNumber != i+1 || d.DecisionType != "checker" {
			t.Errorf("decision %d out of order or mistyped: %+v", i, d)
		}
		if wantPlayer := map[int32]int{1: 0, -1: 1}[plays[i].player]; d.Player != wantPlayer {
			t.Errorf("decision %d: player %d, want %d", i, d.Player, wantPlayer)
		}
	}
	if got[2].MWCLoss != nil {
		t.Errorf("a Move with no analysis is unscored, got %v", *got[2].MWCLoss)
	}
	if got[5].MWCLoss == nil || *got[5].MWCLoss != 0 {
		t.Errorf("the best play is scored at zero, got %v", got[5].MWCLoss)
	}
	if got[1].MWCLoss == nil || got[0].MWCLoss == nil || *got[1].MWCLoss <= *got[0].MWCLoss {
		t.Fatalf("a 150 mP error should cost more than a 60 mP one: %+v", got[:2])
	}

	// Difficulty (ADR-0076): each scored decision is binary, a gap Δ between
	// the two candidates, so the reference player loses Δ·π(worse) and a
	// worse play's difficulty is its loss times 1/(1+e^{Δ/τ}).
	gaps := []float64{0.060, 0.150, 0, 0.020, 0.300, 0.1}
	wantAvoidable := []bool{true, true, false, false, true, false}
	for i, d := range got {
		if d.Avoidable != wantAvoidable[i] {
			t.Errorf("decision %d: avoidable %v, want %v", i, d.Avoidable, wantAvoidable[i])
		}
		if i == 2 {
			if d.Difficulty != nil {
				t.Errorf("an unscored Move has no difficulty, got %v", *d.Difficulty)
			}
			continue
		}
		if d.Difficulty == nil || *d.Difficulty <= 0 {
			t.Errorf("decision %d: difficulty %v, want a positive figure", i, d.Difficulty)
			continue
		}
		share := 1 / (1 + math.Exp(gaps[i]/storage.DifficultyTemperature))
		if i != 5 && math.Abs(*d.Difficulty / *d.MWCLoss - share) > 1e-6 {
			t.Errorf("decision %d: difficulty/loss %v, want %v", i, *d.Difficulty / *d.MWCLoss, share)
		}
	}

	var sum [2]float64
	for _, d := range got {
		if d.MWCLoss != nil {
			sum[d.Player] += *d.MWCLoss
		}
	}
	badges, err := s.Stats().MatchBadges(ctx, "", []int64{matchID})
	if err != nil {
		t.Fatalf("MatchBadges: %v", err)
	}
	b := badges[matchID]
	if math.Abs(sum[0]-b.MWCLoss) > 1e-12 || math.Abs(sum[1]-b.MWCLoss2) > 1e-12 {
		t.Errorf("per-decision sums %v differ from the badge (%v, %v)", sum, b.MWCLoss, b.MWCLoss2)
	}
	if b.MWCLoss <= 0 || b.MWCLoss2 <= 0 {
		t.Errorf("badge should carry a loss for both players: %+v", b)
	}

	// The match review (ADR-0078) reads the same decisions: its PR and L7 are
	// the badge's, its errors to review rank the avoidable part of the loss,
	// and without durations every error is of unknown pace.
	review, err := s.Stats().MatchReview(ctx, "", matchID)
	if err != nil {
		t.Fatalf("MatchReview: %v", err)
	}
	for seat, want := range [2]struct {
		pr   float64
		mwc7 domain.MWC7
	}{{b.PR, b.MWC7}, {b.PR2, b.MWC7P2}} {
		p := review.Players[seat]
		if math.Abs(p.PR-want.pr) > 1e-12 || math.Abs(p.MWC7.Loss-want.mwc7.Loss) > 1e-12 || p.MWC7.HasInterval != want.mwc7.HasInterval {
			t.Errorf("seat %d: review PR %v L7 %+v, badge PR %v L7 %+v", seat, p.PR, p.MWC7, want.pr, want.mwc7)
		}
		if p.Luck.Available {
			t.Errorf("seat %d: an unfinished match without luck has no adjusted result: %+v", seat, p.Luck)
		}
	}
	p1 := review.Players[0]
	if len(p1.ToReview) != 2 || p1.ToReview[0].MoveNumber != 5 || p1.ToReview[1].MoveNumber != 1 {
		t.Errorf("player 1 should review moves 5 then 1: %+v", p1.ToReview)
	}
	if p1.Pace.Unknown != 2 || p1.Pace.Hasty+p1.Pace.Deliberate != 0 {
		t.Errorf("errors without a time are of unknown pace: %+v", p1.Pace)
	}
}

// testStatsMatchDecisionLossesCube runs the cube decisions end to end: a
// double, a pass and a take, each a 100 mP error with a single rival option.
// The answer sits on the position after the double, the cube turned to twice
// the doubler's and held by no one; it is priced at the cube the double was
// offered at, so a pass costs what the double cost at the same score. The
// difficulty weighs the deciding player's two options, as for a checker play.
func testStatsMatchDecisionLossesCube(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7,
		MatchDate: time.Date(2025, 6, 2, 0, 0, 0, 0, time.UTC)}
	matchID, err := s.Matches().Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}
	gameID, err := s.Matches().CreateGame(ctx, "", &domain.Game{MatchID: matchID, GameNumber: 1, Winner: 1, PointsWon: 2})
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	// A cube position has no dice, so the score tells the take's position
	// from the pass's.
	cubePos := func(away, cubeLog2, owner int, dca *domain.DoublingCubeAnalysis, played string) int64 {
		pos := statsDecisionPos(t, 0)
		pos.DecisionType = domain.CubeAction
		pos.Dice = [2]int{0, 0}
		pos.Score = [2]int{away, away}
		pos.Cube = domain.Cube{Value: cubeLog2, Owner: owner}
		id, err := s.Positions().Save(ctx, "", &pos)
		if err != nil {
			t.Fatalf("Save cube position (%d-away): %v", away, err)
		}
		if err := s.Analyses().Save(ctx, "", id, &domain.PositionAnalysis{AnalysisType: "DoublingCube",
			PlayedCubeActions: []string{played}, DoublingCubeAnalysis: dca}); err != nil {
			t.Fatalf("Save cube analysis (%d-away): %v", away, err)
		}
		return id
	}
	// No double is right by 100 mP; taking is right by 100 mP; then, on
	// another position, passing is right by 100 mP.
	double := cubePos(4, 0, domain.None, &domain.DoublingCubeAnalysis{
		CubefulNoDoubleEquity: 0.5, CubefulDoubleTakeEquity: 0.4, CubefulDoublePassEquity: 1.0,
		CubefulDoubleTakeError: 0.1, CubefulDoublePassError: 0.5}, "Double")
	pass := cubePos(4, 1, domain.None, &domain.DoublingCubeAnalysis{
		CubefulNoDoubleEquity: 0.5, CubefulDoubleTakeEquity: 0.9, CubefulDoublePassEquity: 1.0,
		CubefulDoubleTakeError: 0.4, CubefulDoublePassError: 0.5}, "Pass")
	take := cubePos(3, 1, domain.None, &domain.DoublingCubeAnalysis{
		CubefulNoDoubleEquity: 0.5, CubefulDoubleTakeEquity: 1.1, CubefulDoublePassEquity: 1.0,
		CubefulDoubleTakeError: 0.6, CubefulDoublePassError: 0.5}, "Take")
	moves := []struct {
		pos    int64
		player int32
		action string
	}{{double, 1, "Double"}, {pass, -1, "Pass"}, {take, -1, "Take"}}
	for i, mv := range moves {
		if _, err := s.Matches().CreateMove(ctx, "", &domain.Move{GameID: gameID, MoveNumber: int32(i + 1), MoveType: "cube",
			PositionID: mv.pos, Player: mv.player, CubeAction: mv.action}); err != nil {
			t.Fatalf("CreateMove %d: %v", i, err)
		}
	}

	got, err := s.Stats().MatchDecisionLosses(ctx, "", matchID)
	if err != nil {
		t.Fatalf("MatchDecisionLosses: %v", err)
	}
	if len(got) != len(moves) {
		t.Fatalf("got %d decisions, want %d: %+v", len(got), len(moves), got)
	}
	share := 1 / (1 + math.Exp(0.1/storage.DifficultyTemperature))
	for i, d := range got {
		if d.DecisionType != "cube" || d.Rolled || d.ErrorMP == nil || *d.ErrorMP != 100 {
			t.Errorf("decision %d (%s): %+v, want a cube decision of 100 mP", i, moves[i].action, d)
			continue
		}
		if d.MWCLoss == nil || *d.MWCLoss <= 0 || d.Difficulty == nil {
			t.Errorf("decision %d (%s): loss %v, difficulty %v", i, moves[i].action, d.MWCLoss, d.Difficulty)
			continue
		}
		if r := *d.Difficulty / *d.MWCLoss; math.Abs(r-share) > 1e-6 {
			t.Errorf("decision %d (%s): difficulty/loss %v, want %v", i, moves[i].action, r, share)
		}
	}
	if len(got) == 3 && got[0].MWCLoss != nil && got[1].MWCLoss != nil && math.Abs(*got[1].MWCLoss-*got[0].MWCLoss) > 1e-12 {
		t.Errorf("the pass costs %v, the double %v: an answer is priced at the cube offered, not the doubled one",
			*got[1].MWCLoss, *got[0].MWCLoss)
	}

	var sum [2]float64
	for _, d := range got {
		if d.MWCLoss != nil {
			sum[d.Player] += *d.MWCLoss
		}
	}
	badges, err := s.Stats().MatchBadges(ctx, "", []int64{matchID})
	if err != nil {
		t.Fatalf("MatchBadges: %v", err)
	}
	if b := badges[matchID]; math.Abs(sum[0]-b.MWCLoss) > 1e-12 || math.Abs(sum[1]-b.MWCLoss2) > 1e-12 {
		t.Errorf("per-decision sums %v differ from the badge (%v, %v)", sum, b.MWCLoss, b.MWCLoss2)
	}
	review, err := s.Stats().MatchReview(ctx, "", matchID)
	if err != nil {
		t.Fatalf("MatchReview: %v", err)
	}
	if d := review.Players[1].Difficulty; d.Decisions != 2 || math.Abs(d.Loss-sum[1]) > 1e-12 {
		t.Errorf("Bob's served difficulty summary %+v, want two decisions losing %v", d, sum[1])
	}
	if l := review.Players[0].Luck; l.Rolls != 0 {
		t.Errorf("a cube decision is not a roll: %+v", l)
	}
}
