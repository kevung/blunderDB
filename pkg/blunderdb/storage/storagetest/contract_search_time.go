package storagetest

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testSearchDecisionTimeFilter pins the `tm` filter: bounds in seconds, either
// duration of the Move counting, an unknown duration matching nothing, and the
// filter combining with the error filter.
func testSearchDecisionTimeFilter(t *testing.T, s storage.Storage) {
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
	type fixture struct {
		decision, cube *int64
	}
	fixtures := []fixture{
		{ms(2000), nil},       // 0: 2 s
		{ms(45000), nil},      // 1: 45 s
		{nil, nil},            // 2: unknown
		{ms(1000), ms(60000)}, // 3: quick move, long cube decision
	}
	ids := make([]int64, len(fixtures))
	for i, fx := range fixtures {
		pos := statsDecisionPos(t, i)
		id, err := s.Positions().Save(ctx, "", &pos)
		if err != nil {
			t.Fatalf("Save position %d: %v", i, err)
		}
		ids[i] = id
		mv := domain.Move{GameID: gameID, MoveNumber: int32(i + 1), MoveType: "checker", PositionID: id,
			Player: 1, CheckerMove: "13/11 6/4", DecisionMS: fx.decision, CubeDecisionMS: fx.cube}
		if _, err := s.Matches().CreateMove(ctx, "", &mv); err != nil {
			t.Fatalf("CreateMove %d: %v", i, err)
		}
	}
	want := func(filter string, idx ...int) {
		t.Helper()
		var w []int64
		for _, i := range idx {
			w = append(w, ids[i])
		}
		got := searchIDs(t, s, domain.SearchFilters{DecisionTimeFilter: filter})
		sort.Slice(got, func(a, b int) bool { return got[a] < got[b] })
		sort.Slice(w, func(a, b int) bool { return w[a] < w[b] })
		if len(got) != len(w) {
			t.Errorf("%s: got %v, want %v", filter, got, w)
			return
		}
		for i := range got {
			if got[i] != w[i] {
				t.Errorf("%s: got %v, want %v", filter, got, w)
				return
			}
		}
	}
	want("tm>30", 1, 3)
	want("tm<5", 0, 3)
	want("tm1.5,3", 0)
	want("tm>600")
}
