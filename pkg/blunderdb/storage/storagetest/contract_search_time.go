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

// testSearchDecisionTimeWithMoveError pins that `E` and `tm` judge the SAME
// play: a position played slowly with a small error in one match and fast with
// a large one in another is not "slow and wrong", although it is both slow and
// wrong across its plays.
func testSearchDecisionTimeWithMoveError(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ms := func(v int64) *int64 { return &v }
	mkGame := func(n int) int64 {
		m := domain.Match{Player1Name: "me", Player2Name: "them", MatchLength: 7,
			MatchDate: time.Date(2025, 6, n, 0, 0, 0, 0, time.UTC)}
		matchID, err := s.Matches().Save(ctx, "", &m)
		if err != nil {
			t.Fatalf("Save match %d: %v", n, err)
		}
		gameID, err := s.Matches().CreateGame(ctx, "", &domain.Game{MatchID: matchID, GameNumber: 1, Winner: 1, PointsWon: 1})
		if err != nil {
			t.Fatalf("CreateGame %d: %v", n, err)
		}
		return gameID
	}
	gameA, gameB := mkGame(1), mkGame(2)
	pos := statsDecisionPos(t, 0)
	id, err := s.Positions().Save(ctx, "", &pos)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}
	play := func(gameID int64, checkerMove string, d int64) {
		mv := domain.Move{GameID: gameID, MoveNumber: 1, MoveType: "checker", PositionID: id,
			Player: 1, CheckerMove: checkerMove, DecisionMS: ms(d)}
		if _, err := s.Matches().CreateMove(ctx, "", &mv); err != nil {
			t.Fatalf("CreateMove: %v", err)
		}
	}
	play(gameA, "13/11 24/23", 40000) // 50 mp, slow
	play(gameB, "13/11 6/4", 2000)    // 200 mp, fast
	small, big := 0.05, 0.20
	a := domain.PositionAnalysis{
		AnalysisType: "CheckerMove",
		PlayedMoves:  []string{"13/11 24/23", "13/11 6/4"},
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
			{Move: "8/6 6/4", Equity: 0.50},
			{Move: "13/11 24/23", Equity: 0.45, EquityError: &small},
			{Move: "13/11 6/4", Equity: 0.30, EquityError: &big},
		}},
	}
	if err := s.Analyses().Save(ctx, "", id, &a); err != nil {
		t.Fatalf("Save analysis: %v", err)
	}
	for _, c := range []struct {
		time, err string
		want      int
	}{
		{"tm>30", "E>100", 0}, // slow, and wrong, but not the same play
		{"tm<5", "E>100", 1},  // fast and wrong
		{"tm>30", "E<100", 1}, // slow and small
		{"tm>30", "", 1},
		{"", "E>100", 1},
	} {
		got := searchIDs(t, s, domain.SearchFilters{DecisionTimeFilter: c.time, MoveErrorFilter: c.err})
		if len(got) != c.want {
			t.Errorf("%s %s: got %v, want %d position(s)", c.time, c.err, got, c.want)
		}
	}
}
