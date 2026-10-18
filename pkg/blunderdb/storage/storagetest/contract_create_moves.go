// Contract case for the batched move insert.
package storagetest

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testCreateMovesEqualsCreateMove pins CreateMoves against CreateMove one by
// one on the same moves: ids in call order, a move with no position (ID 0)
// among them, a position reached twice, and the per-seat decision errors that
// move.decision_error_mp feeds. Both paths score each move by the analysis its
// position holds when the call is made, so the analyses are saved first.
func testCreateMovesEqualsCreateMove(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	errors := []float64{0.05, 0.12, 0.30, 0}
	var pos [4]int64
	for slot, e := range errors {
		p := statsDecisionPos(t, slot)
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save position %d: %v", slot, err)
		}
		pos[slot] = id
		a := domain.PositionAnalysis{
			PlayedMoves: []string{"13/11 24/23"},
			CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
				{Move: "8/6 6/4", Equity: 0.50},
				{Move: "13/11 24/23", Equity: 0.50 - e, EquityError: &e},
			}},
		}
		if err := s.Analyses().Save(ctx, "", id, &a); err != nil {
			t.Fatalf("Save analysis %d: %v", slot, err)
		}
	}
	// position 0 twice, a move with no position in the middle.
	plan := []struct {
		pos    int64
		player int32
	}{{pos[0], 1}, {pos[1], -1}, {0, 1}, {pos[2], -1}, {pos[0], 1}, {pos[3], -1}}

	newGame := func() (matchID, gameID int64) {
		m := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7,
			MatchDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)}
		matchID, err := s.Matches().Save(ctx, "", &m)
		if err != nil {
			t.Fatalf("Save match: %v", err)
		}
		g := domain.Game{MatchID: matchID, GameNumber: 1, Winner: 1, PointsWon: 1}
		gameID, err = s.Matches().CreateGame(ctx, "", &g)
		if err != nil {
			t.Fatalf("CreateGame: %v", err)
		}
		return matchID, gameID
	}
	moves := func(gameID int64) []*domain.Move {
		var mvs []*domain.Move
		for i, p := range plan {
			mvs = append(mvs, &domain.Move{GameID: gameID, MoveNumber: int32(i + 1), MoveType: "checker",
				PositionID: p.pos, Player: p.player, CheckerMove: "13/11 24/23"})
		}
		return mvs
	}

	singleMatch, singleGame := newGame()
	for _, mv := range moves(singleGame) {
		if _, err := s.Matches().CreateMove(ctx, "", mv); err != nil {
			t.Fatalf("CreateMove: %v", err)
		}
	}
	batchMatch, batchGame := newGame()
	batch := moves(batchGame)
	if err := s.Matches().CreateMoves(ctx, "", batch); err != nil {
		t.Fatalf("CreateMoves: %v", err)
	}

	for i, mv := range batch {
		if mv.ID == 0 || (i > 0 && mv.ID <= batch[i-1].ID) {
			t.Fatalf("move ids %v: want set and strictly increasing in call order", moveIDs(batch))
		}
	}
	var gotOrder []int64
	for mv, err := range s.Matches().Moves(ctx, "", batchGame) {
		if err != nil {
			t.Fatalf("Moves: %v", err)
		}
		gotOrder = append(gotOrder, mv.ID)
	}
	if !reflect.DeepEqual(gotOrder, moveIDs(batch)) {
		t.Errorf("Moves lists %v, CreateMoves assigned %v", gotOrder, moveIDs(batch))
	}

	rows := func(matchID int64) []storage.MatchStatsRow {
		got, err := s.Stats().MatchStats(ctx, "", []int64{matchID})
		if err != nil {
			t.Fatalf("MatchStats: %v", err)
		}
		for i := range got {
			got[i].MatchID = 0
		}
		return got
	}
	want, got := rows(singleMatch), rows(batchMatch)
	if !reflect.DeepEqual(want, got) {
		t.Errorf("match_stats rows differ:\n one by one %+v\n batched    %+v", want, got)
	}
	var total int64
	for _, r := range got {
		total += r.ErrorMP
	}
	if total != 520 {
		t.Errorf("batched error total = %d mp, want 520 (50+120+300+50): the moves were not scored", total)
	}
}

func moveIDs(mvs []*domain.Move) []int64 {
	ids := make([]int64, len(mvs))
	for i, mv := range mvs {
		ids[i] = mv.ID
	}
	return ids
}
