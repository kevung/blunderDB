package storagetest

import (
	"context"
	"slices"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testSearchFilterByAway pins the `aw`, `AW`, `cr` and `pcr` filters on both
// backends: `aw` bounds the distance to victory of the player (player 1, on
// roll as stored), `AW` the opponent's, the Crawford sentinels 0 and 1 being
// one point each and money play never matching; `cr` and `pcr` tell the
// Crawford game from the games after it. Mirror search swaps `aw` and `AW`.
func testSearchFilterByAway(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ids := map[string]int64{}
	for name, score := range map[string][2]int{
		"money":   {domain.Unlimited, domain.Unlimited},
		"post-2":  {domain.PostCrawford, 2},
		"craw-3":  {domain.Crawford, 3},
		"2-2":     {2, 2},
		"2-3":     {2, 3},
		"3-9":     {3, 9},
		"craw-1":  {domain.Crawford, domain.Crawford},
		"post-cr": {domain.PostCrawford, domain.Crawford},
		"4-4":     {4, 4},
		"11-post": {11, domain.PostCrawford},
	} {
		p := domain.InitializePosition()
		p.DecisionType = domain.CheckerAction
		p.Score = score
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save %s: %v", name, err)
		}
		ids[name] = id
	}

	cases := []struct {
		f    domain.SearchFilters
		want []string
	}{
		{domain.SearchFilters{AwayFilter: "aw<1"}, []string{"post-2", "craw-3", "craw-1", "post-cr"}},
		{domain.SearchFilters{AwayFilter: "aw:1"}, []string{"post-2", "craw-3", "craw-1", "post-cr"}},
		{domain.SearchFilters{AwayFilter: "aw:2,3"}, []string{"2-2", "2-3", "3-9"}},
		{domain.SearchFilters{AwayFilter: "aw>4"}, []string{"4-4", "11-post"}},
		{domain.SearchFilters{AwayFilter: "aw:0"}, nil},
		{domain.SearchFilters{OpponentAwayFilter: "AW:1"}, []string{"craw-1", "post-cr", "11-post"}},
		{domain.SearchFilters{OpponentAwayFilter: "AW>3"}, []string{"craw-3", "2-3", "3-9", "4-4"}},
		{domain.SearchFilters{AwayFilter: "aw:2", OpponentAwayFilter: "AW:3"}, []string{"2-3"}},
		{domain.SearchFilters{AwayFilter: "aw:4", OpponentAwayFilter: "AW:4"}, []string{"4-4"}},
		{domain.SearchFilters{CrawfordFilter: domain.CrawfordGame}, []string{"craw-3", "craw-1", "post-cr"}},
		{domain.SearchFilters{CrawfordFilter: domain.CrawfordAfter}, []string{"post-2", "post-cr", "11-post"}},
		{domain.SearchFilters{CrawfordFilter: domain.CrawfordNone}, nil},
		{domain.SearchFilters{AwayFilter: "aw:1", CrawfordFilter: domain.CrawfordAfter}, []string{"post-2", "post-cr"}},
		{domain.SearchFilters{OpponentAwayFilter: "AW:1", CrawfordFilter: domain.CrawfordAfter}, []string{"post-cr", "11-post"}},
		// Mirror search reads the position or its mirror, so `aw` and `AW`
		// swap: 3-away against 2-away is met by "2-3" seen from the other side.
		{domain.SearchFilters{AwayFilter: "aw:3", OpponentAwayFilter: "AW:2", MirrorFilter: true}, []string{"2-3"}},
		{domain.SearchFilters{AwayFilter: "aw>4", MirrorFilter: true}, []string{"3-9", "4-4", "11-post"}},
		{domain.SearchFilters{CrawfordFilter: domain.CrawfordGame, MirrorFilter: true}, []string{"craw-3", "craw-1", "post-cr"}},
	}
	for _, c := range cases {
		var want []int64
		for _, n := range c.want {
			want = append(want, ids[n])
		}
		got := searchIDs(t, s, c.f)
		slices.Sort(got)
		slices.Sort(want)
		if !slices.Equal(got, want) {
			t.Errorf("aw %q AW %q cr %q M %v: got %v, want %v (ids %v)",
				c.f.AwayFilter, c.f.OpponentAwayFilter, c.f.CrawfordFilter, c.f.MirrorFilter, got, want, ids)
		}
	}
}

// testSearchAwayBeyondMatchLength pins the shortcut of an away bound past the
// longest match admitted: in a 7-point match no side stands 8 away, and at 7
// away both sides stand at its first decision.
func testSearchAwayBeyondMatchLength(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	matchID, err := s.Matches().Save(ctx, "", &domain.Match{Player1Name: "A", Player2Name: "B", MatchLength: 7})
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}
	gameID, err := s.Matches().CreateGame(ctx, "", &domain.Game{MatchID: matchID, GameNumber: 1, Winner: 1, PointsWon: 1})
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	p := domain.InitializePosition()
	p.DecisionType = domain.CheckerAction
	p.Score = [2]int{7, 7}
	id, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}
	mv := domain.Move{GameID: gameID, MoveNumber: 1, MoveType: "checker", PositionID: id, Player: 1, CheckerMove: "13/11 6/4"}
	if _, err := s.Matches().CreateMove(ctx, "", &mv); err != nil {
		t.Fatalf("CreateMove: %v", err)
	}
	for _, c := range []struct {
		f    domain.SearchFilters
		want []int64
	}{
		{domain.SearchFilters{MatchLengthFilter: "ml:7", AwayFilter: "aw:8"}, nil},
		{domain.SearchFilters{MatchLengthFilter: "ml:7", AwayFilter: "aw:7"}, []int64{id}},
		{domain.SearchFilters{MatchLengthFilter: "ml:7", OpponentAwayFilter: "AW:8"}, nil},
		{domain.SearchFilters{MatchLengthFilter: "ml:7", OpponentAwayFilter: "AW:7"}, []int64{id}},
	} {
		if got := searchIDs(t, s, c.f); !slices.Equal(got, c.want) {
			t.Errorf("ml:7 aw %q AW %q: got %v, want %v", c.f.AwayFilter, c.f.OpponentAwayFilter, got, c.want)
		}
	}
}
