package storagetest

import (
	"context"
	"slices"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testMatchListSortByPR: the match list orders by a PR over the whole
// library — seat 1's, seat 2's, the main player's wherever he sat (aliases
// included) and his opponent's. A match without that PR comes last in both
// directions; equal PRs fall back to the id in the sort's direction, so the
// order is total and a page is the slice of it a whole read gives.
func testMatchListSortByPR(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	slot := 0
	// errs[i] is seat i+1's equity error on its one decision; a negative
	// one leaves the seat without a decision.
	mk := func(p1, p2 string, errs [2]float64) int64 {
		t.Helper()
		m := domain.Match{Player1Name: p1, Player2Name: p2, MatchLength: 7}
		id, err := s.Matches().Save(ctx, "", &m)
		if err != nil {
			t.Fatalf("Save match: %v", err)
		}
		gameID, err := s.Matches().CreateGame(ctx, "", &domain.Game{MatchID: id, GameNumber: 1, Winner: 1, PointsWon: 1})
		if err != nil {
			t.Fatalf("CreateGame: %v", err)
		}
		for seat, e := range errs {
			if e < 0 {
				continue
			}
			pos := statsDecisionPos(t, slot)
			posID, err := s.Positions().Save(ctx, "", &pos)
			if err != nil {
				t.Fatalf("Save position: %v", err)
			}
			player := int32(1)
			if seat == 1 {
				player = -1
			}
			mv := domain.Move{GameID: gameID, MoveNumber: int32(slot), MoveType: "checker",
				PositionID: posID, Player: player, CheckerMove: "13/11 24/23"}
			if _, err := s.Matches().CreateMove(ctx, "", &mv); err != nil {
				t.Fatalf("CreateMove: %v", err)
			}
			a := domain.PositionAnalysis{
				PlayedMoves: []string{"13/11 24/23"},
				CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
					{Move: "8/6 6/4", Equity: 0.50},
					{Move: "13/11 24/23", Equity: 0.50 - e, EquityError: &e},
				}},
			}
			if err := s.Analyses().Save(ctx, "", posID, &a); err != nil {
				t.Fatalf("Save analysis: %v", err)
			}
			slot++
		}
		return id
	}

	a := mk("Kevin", "Bob", [2]float64{0.02, 0.08})
	b := mk("Carl", "K. Unger", [2]float64{0.04, 0.01})
	c := mk("Kevin", "Dan", [2]float64{0.02, 0.03})
	d := mk("Eve", "Fay", [2]float64{0.05, 0.06})
	e := mk("Kevin", "Gus", [2]float64{-1, -1})
	if err := s.Aliases().Set(ctx, "", storage.AliasPlayer, "K. Unger", "Kevin"); err != nil {
		t.Fatalf("alias: %v", err)
	}
	if err := s.LibrarySettings().SetMainPlayer(ctx, "", "Kevin"); err != nil {
		t.Fatalf("SetMainPlayer: %v", err)
	}

	ids := func(o storage.MatchListOpts) []int64 {
		t.Helper()
		var out []int64
		for m, err := range s.Matches().List(ctx, "", o) {
			if err != nil {
				t.Fatalf("List(%+v): %v", o, err)
			}
			out = append(out, m.ID)
		}
		return out
	}
	for _, tc := range []struct {
		sort string
		want []int64
	}{
		{"pr_main", []int64{b, a, c, d, e}},
		{"pr_main_desc", []int64{c, a, b, e, d}},
		{"pr_opponent", []int64{c, b, a, d, e}},
		{"pr_opponent_desc", []int64{a, b, c, e, d}},
		{"pr1", []int64{a, c, b, d, e}},
		{"pr1_desc", []int64{d, b, c, a, e}},
		{"pr2", []int64{b, c, d, a, e}},
		{"pr2_desc", []int64{a, d, c, b, e}},
	} {
		whole := ids(storage.MatchListOpts{Sort: tc.sort})
		if !slices.Equal(whole, tc.want) {
			t.Errorf("%s: %v; want %v", tc.sort, whole, tc.want)
			continue
		}
		for off := 0; off < len(whole); off += 2 {
			page := ids(storage.MatchListOpts{Sort: tc.sort, Limit: 2, Offset: off})
			if !slices.Equal(page, whole[off:min(off+2, len(whole))]) {
				t.Errorf("%s page at %d: %v; want %v", tc.sort, off, page, whole[off:min(off+2, len(whole))])
			}
		}
	}

	// The order's placeholders follow the filter's.
	if got, want := ids(storage.MatchListOpts{PlayerName: "Kevin", Sort: "pr_main", Limit: 3, Offset: 1}), []int64{a, c, e}; !slices.Equal(got, want) {
		t.Errorf("Kevin's matches by his PR, from the 2nd: %v; want %v", got, want)
	}

	// Without a main player no match has a main PR: the id alone orders.
	if err := s.LibrarySettings().SetMainPlayer(ctx, "", ""); err != nil {
		t.Fatalf("SetMainPlayer(\"\"): %v", err)
	}
	if got, want := ids(storage.MatchListOpts{Sort: "pr_main_desc"}), []int64{e, d, c, b, a}; !slices.Equal(got, want) {
		t.Errorf("pr_main_desc without a main player: %v; want %v", got, want)
	}
}
