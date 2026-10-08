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

	// Difficulty (ADR-0075): each scored decision is binary, a gap Δ between
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
}
