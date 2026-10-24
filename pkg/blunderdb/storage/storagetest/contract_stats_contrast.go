package storagetest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testStatsPlayerContrast pins the side-by-side list: a position both players
// decided and answered on opposite sides of the Error threshold, each scored on
// their own play, the widest gap first; a position only one of them met, or
// both played well or both badly, is not in it.
func testStatsPlayerContrast(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	if _, err := s.Stats().DateRange(ctx, ""); errors.Is(err, storage.ErrInternal) {
		t.Skip("Stats not implemented on this backend")
	}
	mkGame := func(p1, p2 string, day int) int64 {
		m := domain.Match{Player1Name: p1, Player2Name: p2, MatchLength: 7,
			MatchDate: time.Date(2025, 6, day, 0, 0, 0, 0, time.UTC)}
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
	// Alice sat first against Carol, Bob second against Dave: seats differ.
	gameAlice, gameBob := mkGame("Alice", "Carol", 1), mkGame("Dave", "Bob", 2)

	// decide records one position played twice, once per player, each play
	// costing its own error; the analysis lists both plays.
	n := int32(0)
	decide := func(slot int, playA, playB string, errA, errB float64) int64 {
		pos := statsDecisionPos(t, slot)
		posID, err := s.Positions().Save(ctx, "", &pos)
		if err != nil {
			t.Fatalf("Save position: %v", err)
		}
		for _, mv := range []domain.Move{
			{GameID: gameAlice, PositionID: posID, Player: 1, CheckerMove: playA},
			{GameID: gameBob, PositionID: posID, Player: -1, CheckerMove: playB},
		} {
			if mv.CheckerMove == "" {
				continue
			}
			n++
			mv.MoveNumber, mv.MoveType = n, "checker"
			if _, err := s.Matches().CreateMove(ctx, "", &mv); err != nil {
				t.Fatalf("CreateMove: %v", err)
			}
		}
		zero := 0.0
		moves := []domain.CheckerMove{{Move: "8/5 6/5", Equity: 0.50, EquityError: &zero}}
		for _, pm := range []struct {
			mv string
			e  float64
		}{{playA, errA}, {playB, errB}} {
			if pm.mv != "" {
				e := pm.e
				moves = append(moves, domain.CheckerMove{Move: pm.mv, Equity: 0.50 - e, EquityError: &e})
			}
		}
		a := domain.PositionAnalysis{PlayedMoves: []string{playA}, CheckerAnalysis: &domain.CheckerAnalysis{Moves: moves}}
		if err := s.Analyses().Save(ctx, "", posID, &a); err != nil {
			t.Fatalf("Save analysis: %v", err)
		}
		return posID
	}
	aliceWell := decide(0, "13/11 24/23", "13/11 6/4", 0.010, 0.120) // gap 110
	bobWell := decide(1, "24/18 13/11", "13/9 24/22", 0.300, 0.020)  // gap 280
	decide(2, "13/9 24/22", "24/18 6/4", 0.200, 0.150)               // both badly
	decide(3, "13/11 6/4", "24/22 13/11", 0.005, 0.020)              // both well
	decide(4, "24/20 13/11", "", 0.400, 0)                           // Alice only

	got, err := s.Stats().PlayerContrast(ctx, "", "Alice", "Bob", storage.StatsFilter{DecisionType: -1})
	if err != nil {
		t.Fatalf("PlayerContrast: %v", err)
	}
	if got.UnscoredMoves != 0 {
		t.Errorf("UnscoredMoves = %d, want 0: the analysis prices every play", got.UnscoredMoves)
	}
	if got.CommonPositions != 4 {
		t.Errorf("CommonPositions = %d, want 4", got.CommonPositions)
	}
	if len(got.Positions) != 2 || got.Positions[0].PositionID != bobWell || got.Positions[1].PositionID != aliceWell {
		t.Fatalf("Positions = %+v, want [%d %d] (widest gap first)", got.Positions, bobWell, aliceWell)
	}
	if p := got.Positions[0]; p.WellPlayed != "b" || p.ErrorMPA != 300 || p.ErrorMPB != 20 {
		t.Errorf("bobWell = %+v, want b well, 300/20 mp", p)
	}
	if p := got.Positions[1]; p.WellPlayed != "a" || p.TimesA != 1 || p.TimesB != 1 {
		t.Errorf("aliceWell = %+v, want a well, once each", p)
	}

	// The order of the players flips the sides, not the set.
	back, err := s.Stats().PlayerContrast(ctx, "", "Bob", "Alice", storage.StatsFilter{DecisionType: -1})
	if err != nil {
		t.Fatalf("PlayerContrast reversed: %v", err)
	}
	if len(back.Positions) != 2 || back.Positions[0].WellPlayed != "a" || back.Positions[0].PositionID != bobWell {
		t.Errorf("reversed = %+v, want bobWell first with a well", back.Positions)
	}

	if _, err := s.Stats().PlayerContrast(ctx, "", "Alice", "Alice", storage.StatsFilter{DecisionType: -1}); !errors.Is(err, storage.ErrInvalid) {
		t.Errorf("same player: err = %v, want ErrInvalid", err)
	}
}

// testStatsPlayerContrastUnpricedCube pins that a cube decision whose played
// action prices nothing — no action recorded, or a label no importer could map
// — leaves its player unscored there: the per-move column reads it 0, and
// taken at its word that 0 would pair a priced no double with a perfect play
// and invent a common position.
func testStatsPlayerContrastUnpricedCube(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	if _, err := s.Stats().DateRange(ctx, ""); errors.Is(err, storage.ErrInternal) {
		t.Skip("Stats not implemented on this backend")
	}
	if err := s.LibrarySettings().Save(ctx, "", storage.LibrarySettings{ErrorThresholdMP: 20, BlunderThresholdMP: 40}); err != nil {
		t.Fatalf("Save settings: %v", err)
	}
	mkGame := func(p1, p2 string, day int) int64 {
		m := domain.Match{Player1Name: p1, Player2Name: p2, MatchLength: 7,
			MatchDate: time.Date(2025, 6, day, 0, 0, 0, 0, time.UTC)}
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
	gameAlice, gameBob := mkGame("Alice", "Carol", 1), mkGame("Dave", "Bob", 2)

	// Each position: Alice keeps the cube where Double, Take is right
	// (150 mp), Bob's cube move carries the unpriced action.
	n := int32(0)
	for slot, bob := range []struct {
		action string
		played []string
	}{
		{"", []string{"No Double"}},
		{"Unknown(7)", []string{"No Double", "Unknown(7)"}},
	} {
		pos := statsDecisionPos(t, slot)
		pos.DecisionType = domain.CubeAction
		posID, err := s.Positions().Save(ctx, "", &pos)
		if err != nil {
			t.Fatalf("Save cube position (slot %d): %v", slot, err)
		}
		for _, mv := range []domain.Move{
			{GameID: gameAlice, PositionID: posID, Player: 1, CubeAction: "No Double"},
			{GameID: gameBob, PositionID: posID, Player: -1, CubeAction: bob.action},
		} {
			n++
			mv.MoveNumber, mv.MoveType = n, "cube"
			if _, err := s.Matches().CreateMove(ctx, "", &mv); err != nil {
				t.Fatalf("CreateMove (slot %d): %v", slot, err)
			}
		}
		a := domain.PositionAnalysis{
			PlayedCubeActions: bob.played,
			DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{
				BestCubeAction:          "Double, Take",
				CubefulNoDoubleEquity:   0.40,
				CubefulDoubleTakeEquity: 0.55,
				CubefulDoublePassEquity: 1.00,
				CubefulNoDoubleError:    -0.150,
			},
		}
		if err := s.Analyses().Save(ctx, "", posID, &a); err != nil {
			t.Fatalf("Save cube analysis (slot %d): %v", slot, err)
		}
	}

	got, err := s.Stats().PlayerContrast(ctx, "", "Alice", "Bob", storage.StatsFilter{DecisionType: -1})
	if err != nil {
		t.Fatalf("PlayerContrast: %v", err)
	}
	if got.CommonPositions != 0 || len(got.Positions) != 0 {
		t.Errorf("CommonPositions = %d, Positions = %+v, want none: Bob's unpriced cube was scored as a perfect play",
			got.CommonPositions, got.Positions)
	}
	if got.UnscoredMoves != 2 {
		t.Errorf("UnscoredMoves = %d, want 2: Bob's two unpriced cube decisions", got.UnscoredMoves)
	}
	// Alice's no double stays priced: alone, it is hers, not a hole.
	solo, err := s.Stats().PlayerContrast(ctx, "", "Alice", "Carol", storage.StatsFilter{DecisionType: -1})
	if err != nil {
		t.Fatalf("PlayerContrast Alice/Carol: %v", err)
	}
	if solo.UnscoredMoves != 0 {
		t.Errorf("Alice/Carol UnscoredMoves = %d, want 0: a no double is priced", solo.UnscoredMoves)
	}
}
