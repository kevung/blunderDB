// Contract case: a filter past what a column can hold finds nothing. Both
// backends answer the same, whatever width each gives the column.
package storagetest

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func testSearchFilterPastTheColumn(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	saveWindowPositions(ctx, t, s, 6)
	wideCube, wideScore, wideDice := domain.InitializePosition(), domain.InitializePosition(), domain.InitializePosition()
	wideCube.Cube.Value = 3000000
	wideScore.Score = [2]int{3000000, 3000000}
	wideDice.Dice = [2]int{40000, 40000}
	const past16, past32 = "70000", "3000000000"
	for _, tc := range []struct {
		name string
		f    domain.SearchFilters
		n    int
	}{
		{"pip diff above", domain.SearchFilters{PipCountFilter: "p>" + past16}, 0},
		{"pip diff above 32 bits", domain.SearchFilters{PipCountFilter: "p>" + past32}, 0},
		{"pip diff below", domain.SearchFilters{PipCountFilter: "p<" + past16}, 6},
		{"pip diff below minus", domain.SearchFilters{PipCountFilter: "p>-" + past16}, 6},
		{"own pips above", domain.SearchFilters{Player1AbsolutePipCountFilter: "P>" + past16}, 0},
		{"off above", domain.SearchFilters{Player1CheckerOffFilter: "o>" + past16}, 0},
		{"their off above", domain.SearchFilters{Player2CheckerOffFilter: "O>" + past16}, 0},
		{"back above", domain.SearchFilters{Player1BackCheckerFilter: "k>" + past16}, 0},
		{"their back above", domain.SearchFilters{Player2BackCheckerFilter: "K>" + past16}, 0},
		{"win above", domain.SearchFilters{WinRateFilter: "w>" + past16}, 0},
		{"gammon above", domain.SearchFilters{GammonRateFilter: "g>" + past16}, 0},
		{"backgammon above", domain.SearchFilters{BackgammonRateFilter: "b>" + past16}, 0},
		{"their win above", domain.SearchFilters{Player2WinRateFilter: "W>" + past16}, 0},
		{"their gammon above", domain.SearchFilters{Player2GammonRateFilter: "G>" + past16}, 0},
		{"their backgammon above", domain.SearchFilters{Player2BackgammonRateFilter: "B>" + past16}, 0},
		{"move error above", domain.SearchFilters{MoveErrorFilter: "E>" + past32}, 0},
		{"equity above", domain.SearchFilters{EquityFilter: "e>" + past16}, 0},
		{"away above", domain.SearchFilters{AwayFilter: "aw>" + past16}, 0},
		{"away exact", domain.SearchFilters{OpponentAwayFilter: "AW:" + past16}, 0},
		{"cube", domain.SearchFilters{Filter: wideCube, IncludeCube: true}, 0},
		{"score", domain.SearchFilters{Filter: wideScore, IncludeScore: true}, 0},
		{"dice", domain.SearchFilters{Filter: wideDice}, -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n, err := s.Search().Count(ctx, "", tc.f)
			if err != nil {
				t.Fatalf("Count: %v", err)
			}
			if tc.n >= 0 && n != tc.n {
				t.Errorf("Count = %d, want %d", n, tc.n)
			}
			ids, err := s.Search().FindIDs(ctx, "", tc.f, storage.ListOpts{Limit: 24})
			if err != nil {
				t.Fatalf("FindIDs: %v", err)
			}
			if len(ids) != n {
				t.Errorf("FindIDs = %d rows, Count = %d", len(ids), n)
			}
		})
	}
}
