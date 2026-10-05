package storagetest

import (
	"context"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testStatsTimeErrors pins the duration bands of the time/error table for the
// players of a Match written the way a Duel writes it: each player under its
// own name, an unknown duration in no band, a cube decision added to the play
// after it, and an unscored decision counted but never averaged as zero.
func testStatsTimeErrors(t *testing.T, s storage.Storage) {
	ms := func(v int64) *int64 { return &v }
	timedMatch(t, s, [][2]*int64{
		{ms(3000), ms(3000)}, // player 1: 6 s, band 1
		{ms(40000), nil},     // player 2: band 3
		{nil, nil},           // player 1: unknown, no band
		{ms(1000), nil},      // player 2: band 0
	})
	got, err := s.Stats().TimeErrors(context.Background(), "")
	if err != nil {
		t.Fatalf("TimeErrors: %v", err)
	}
	want := []storage.TimeErrorRow{
		{Player: "me", Bucket: 1, Decisions: 1},
		{Player: "them", Bucket: 0, Decisions: 1},
		{Player: "them", Bucket: 3, Decisions: 1},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

// testStatsTimeErrorsScoredByAnalysis pins that an analysis written after the
// Match, as the end of a Duel writes it, scores the table at once: no separate
// pass over move.error_mp stands between the analysis and the mean error.
func testStatsTimeErrorsScoredByAnalysis(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ms := func(v int64) *int64 { return &v }
	m := domain.Match{Player1Name: "me", Player2Name: "them", MatchLength: 7,
		MatchDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)}
	matchID, err := s.Matches().Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}
	gameID, err := s.Matches().CreateGame(ctx, "", &domain.Game{MatchID: matchID, GameNumber: 1, Winner: 1, PointsWon: 1})
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	pos := statsDecisionPos(t, 0)
	id, err := s.Positions().Save(ctx, "", &pos)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}
	for i, p := range []struct {
		player int32
		play   string
		d      int64
	}{{1, "13/11 24/23", 2000}, {-1, "13/11 6/4", 2000}} {
		mv := domain.Move{GameID: gameID, MoveNumber: int32(i + 1), MoveType: "checker", PositionID: id,
			Player: p.player, CheckerMove: p.play, DecisionMS: ms(p.d)}
		if _, err := s.Matches().CreateMove(ctx, "", &mv); err != nil {
			t.Fatalf("CreateMove: %v", err)
		}
	}
	small, big := 0.05, 0.20
	if err := s.Analyses().Save(ctx, "", id, &domain.PositionAnalysis{
		AnalysisType: "CheckerMove",
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
			{Move: "8/6 6/4", Equity: 0.50},
			{Move: "13/11 24/23", Equity: 0.45, EquityError: &small},
			{Move: "13/11 6/4", Equity: 0.30, EquityError: &big},
		}},
	}); err != nil {
		t.Fatalf("Save analysis: %v", err)
	}
	got, err := s.Stats().TimeErrors(ctx, "")
	if err != nil {
		t.Fatalf("TimeErrors: %v", err)
	}
	want := []storage.TimeErrorRow{
		{Player: "me", Bucket: 0, Decisions: 1, Scored: 1, MeanErrorMP: 50},
		{Player: "them", Bucket: 0, Decisions: 1, Scored: 1, MeanErrorMP: 200, Blunders: 1},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}
