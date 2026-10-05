package storagetest

import (
	"context"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testStatsMatchTimeSummary pins the per-player sum of decision times: an
// unknown duration is counted apart and never as zero, a cube decision taken
// before a roll joins the play after it in one turn of the clock, and the
// overrun is the part of a turn the reserve could not pay.
func testStatsMatchTimeSummary(t *testing.T, s storage.Storage) {
	ctx := context.Background()
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
	ms := func(v int64) *int64 { return &v }
	moves := []domain.Move{
		{Player: 1, MoveType: "checker", CheckerMove: "13/11", DecisionMS: ms(5000)},
		{Player: 2, MoveType: "cube", CubeAction: "Double", DecisionMS: ms(3000)},
		{Player: 1, MoveType: "cube", CubeAction: "Take", DecisionMS: ms(1000)},
		{Player: 2, MoveType: "checker", CheckerMove: "8/5", DecisionMS: ms(4000)},
		{Player: 1, MoveType: "checker", CheckerMove: "6/4", DecisionMS: ms(9000), CubeDecisionMS: ms(1000)},
		{Player: 1, MoveType: "checker", CheckerMove: "24/22"},
	}
	for i := range moves {
		pos := statsDecisionPos(t, i)
		id, err := s.Positions().Save(ctx, "", &pos)
		if err != nil {
			t.Fatalf("Save position %d: %v", i, err)
		}
		moves[i].GameID, moves[i].MoveNumber, moves[i].PositionID = gameID, int32(i+1), id
		if _, err := s.Matches().CreateMove(ctx, "", &moves[i]); err != nil {
			t.Fatalf("CreateMove %d: %v", i, err)
		}
	}

	got, err := s.Stats().MatchTimeSummary(ctx, "", matchID)
	if err != nil {
		t.Fatalf("MatchTimeSummary: %v", err)
	}
	if got.HasCadence {
		t.Errorf("a Match not played here has no Cadence")
	}
	p1, p2 := got.Players[0], got.Players[1]
	if p1.TotalMS != 16000 || p1.CheckerCount != 2 || p1.CheckerTotalMS != 14000 || p1.CubeCount != 2 || p1.CubeTotalMS != 2000 || p1.Unknown != 1 {
		t.Errorf("player 1: %+v", p1)
	}
	if p2.TotalMS != 7000 || p2.CubeCount != 1 || p2.CheckerCount != 1 || p2.Unknown != 0 {
		t.Errorf("player 2: %+v", p2)
	}

	origin := storage.MatchOrigin{MatchID: matchID, DiceSeed: "s", Cadence: `{"reserve":10,"delay":2}`}
	if err := s.Duels().SetOrigin(ctx, "", &origin); err != nil {
		t.Fatalf("SetOrigin: %v", err)
	}
	got, err = s.Stats().MatchTimeSummary(ctx, "", matchID)
	if err != nil {
		t.Fatalf("MatchTimeSummary under a Cadence: %v", err)
	}
	p1, p2 = got.Players[0], got.Players[1]
	// Player 1: turns of 5 s (3 s charged), 1 s (none), 10 s (8 s charged
	// against the 7 s left), unknown: one turn over by 1 s.
	if !got.HasCadence || p1.OverrunTurns != 1 || p1.OverrunMS != 1000 {
		t.Errorf("player 1 overrun: %+v", got)
	}
	// Player 2: a double of 3 s and the play after it, 4 s, are one turn of
	// 7 s, 5 s charged, within the reserve.
	if p2.OverrunTurns != 0 || p2.OverrunMS != 0 {
		t.Errorf("player 2 overrun: %+v", p2)
	}
}
