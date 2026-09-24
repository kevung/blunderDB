// Contract case for the grades the Match panel's Transcript colours its rows
// with (#287): each Move scored by its own play, graded at the library's
// thresholds (ADR-0046).
// The table that runs it lives in contract.go.
package storagetest

import (
	"context"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func testStatsMatchMoveGrades(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7,
		MatchDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)}
	matchID, err := s.Matches().Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}
	gameID, err := s.Matches().CreateGame(ctx, "", &domain.Game{MatchID: matchID, GameNumber: 1, Winner: 1, PointsWon: 1})
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}

	// One checker Position reached five times — the same Position played
	// several ways within one match, which the denormalised column (first
	// play only) cannot grade.
	checker := statsDecisionPos(t, 2)
	checkerID, err := s.Positions().Save(ctx, "", &checker)
	if err != nil {
		t.Fatalf("Save checker position: %v", err)
	}
	e60, e150, e20 := 0.060, 0.150, 0.020
	if err := s.Analyses().Save(ctx, "", checkerID, &domain.PositionAnalysis{
		AnalysisType: "CheckerMove",
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
			{Move: "24/22 13/11", Equity: 0.500},
			{Move: "13/11 24/23", Equity: 0.440, EquityError: &e60},
			{Move: "8/6 6/4", Equity: 0.350, EquityError: &e150},
			{Move: "6/2", Equity: 0.480, EquityError: &e20},
		}},
	}); err != nil {
		t.Fatalf("Save checker analysis: %v", err)
	}

	cube := statsDecisionPos(t, 3)
	cube.DecisionType = domain.CubeAction
	cube.Dice = [2]int{0, 0}
	cubeID, err := s.Positions().Save(ctx, "", &cube)
	if err != nil {
		t.Fatalf("Save cube position: %v", err)
	}
	if err := s.Analyses().Save(ctx, "", cubeID, &domain.PositionAnalysis{
		AnalysisType: "DoublingCube",
		DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{
			CubefulNoDoubleEquity: 0.500, CubefulNoDoubleError: 0.300,
			CubefulDoubleTakeEquity: 0.800, CubefulDoubleTakeError: 0,
			CubefulDoublePassEquity: 1.000, CubefulDoublePassError: 0.200,
		},
	}); err != nil {
		t.Fatalf("Save cube analysis: %v", err)
	}

	type play struct {
		posID           int64
		player          int32
		moveType, label string
	}
	plays := []play{
		{checkerID, 1, "checker", "24/22 13/11"}, // best: 0
		{checkerID, -1, "checker", "8/6 6/4"},    // 150
		{checkerID, 1, "checker", "13/11 24/23"}, // 60
		{checkerID, -1, "checker", "6/2"},        // 20
		{checkerID, 1, "checker", "bar/20"},      // not a candidate: unscored
		{cubeID, 1, "cube", "No Double"},         // 300
		{cubeID, 1, "cube", "Double"},            // 0
		{cubeID, -1, "cube", "Pass"},             // 200 (passing a take)
	}
	ids := make([]int64, len(plays))
	for i, p := range plays {
		mv := domain.Move{GameID: gameID, MoveNumber: int32(i + 1), MoveType: p.moveType, PositionID: p.posID, Player: p.player}
		if p.moveType == "cube" {
			mv.CubeAction = p.label
		} else {
			mv.CheckerMove = p.label
		}
		if ids[i], err = s.Matches().CreateMove(ctx, "", &mv); err != nil {
			t.Fatalf("CreateMove %q: %v", p.label, err)
		}
	}

	// Another match on the same library: its Moves must not leak in.
	statsFixtureMatch(t, s, 5, "Carol", "Dave")

	check := func(when string, want []storage.MoveGrade) {
		t.Helper()
		got, err := s.Stats().MatchMoveGrades(ctx, "", matchID)
		if err != nil {
			t.Fatalf("%s: MatchMoveGrades: %v", when, err)
		}
		if len(got) != len(want) {
			t.Fatalf("%s: got %d grades %+v, want %d %+v", when, len(got), got, len(want), want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("%s: grade %d: got %+v, want %+v", when, i, got[i], want[i])
			}
		}
	}

	// Default thresholds, 50 and 100.
	check("defaults", []storage.MoveGrade{
		{MoveID: ids[0], ErrorMP: 0},
		{MoveID: ids[1], ErrorMP: 150, Grade: storage.MoveGradeBlunder},
		{MoveID: ids[2], ErrorMP: 60, Grade: storage.MoveGradeError},
		{MoveID: ids[3], ErrorMP: 20},
		{MoveID: ids[5], ErrorMP: 300, Grade: storage.MoveGradeBlunder},
		{MoveID: ids[6], ErrorMP: 0},
		{MoveID: ids[7], ErrorMP: 200, Grade: storage.MoveGradeBlunder},
	})

	// The library's own thresholds move the grades — and the comparison is
	// inclusive: 20 at an error threshold of 20 is an Error.
	if err := s.LibrarySettings().Save(ctx, "", storage.LibrarySettings{ErrorThresholdMP: 20, BlunderThresholdMP: 160}); err != nil {
		t.Fatalf("Save settings: %v", err)
	}
	check("thresholds 20/160", []storage.MoveGrade{
		{MoveID: ids[0], ErrorMP: 0},
		{MoveID: ids[1], ErrorMP: 150, Grade: storage.MoveGradeError},
		{MoveID: ids[2], ErrorMP: 60, Grade: storage.MoveGradeError},
		{MoveID: ids[3], ErrorMP: 20, Grade: storage.MoveGradeError},
		{MoveID: ids[5], ErrorMP: 300, Grade: storage.MoveGradeBlunder},
		{MoveID: ids[6], ErrorMP: 0},
		{MoveID: ids[7], ErrorMP: 200, Grade: storage.MoveGradeBlunder},
	})

	// A match without analysed Moves grades nothing, and says so without error.
	empty := domain.Match{Player1Name: "Nobody", Player2Name: "Nowhere"}
	emptyID, err := s.Matches().Save(ctx, "", &empty)
	if err != nil {
		t.Fatalf("Save empty match: %v", err)
	}
	if got, err := s.Stats().MatchMoveGrades(ctx, "", emptyID); err != nil || len(got) != 0 {
		t.Fatalf("empty match: got %+v, %v; want nothing", got, err)
	}
}
