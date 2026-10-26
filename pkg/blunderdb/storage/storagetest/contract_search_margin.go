package storagetest

import (
	"context"
	"slices"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testSearchFilterByMargin pins the `mg` filter on both backends: the margin
// between the best choice and the next, read off the stored analysis — two
// best candidates for a checker play, no double against the double for the
// doubler, take against pass for the responder — and nothing for a position
// without analysis or with a single candidate. Mirroring changes no margin.
func testSearchFilterByMargin(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ids := map[string]int64{}
	save := func(name string, n int, decision int, a *domain.PositionAnalysis) {
		p := domain.InitializePosition()
		p.DecisionType = decision
		p.Score = [2]int{n, 11}
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save %s: %v", name, err)
		}
		if a != nil {
			if err := s.Analyses().Save(ctx, "", id, a); err != nil {
				t.Fatalf("Save analysis %s: %v", name, err)
			}
		}
		ids[name] = id
	}
	checker := func(eqs ...float64) *domain.PositionAnalysis {
		ca := &domain.CheckerAnalysis{}
		for i, e := range eqs {
			ca.Moves = append(ca.Moves, domain.CheckerMove{Index: i, Move: "13/11 24/23", Equity: e})
		}
		return &domain.PositionAnalysis{AnalysisType: "CheckerMove", CheckerAnalysis: ca}
	}
	cube := func(played string) *domain.PositionAnalysis {
		return &domain.PositionAnalysis{
			AnalysisType: "DoublingCube",
			DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{
				CubefulNoDoubleEquity: 0.600, CubefulDoubleTakeEquity: 0.640, CubefulDoublePassEquity: 1.000,
				BestCubeAction: "Double, Take",
			},
			PlayedCubeActions: []string{played},
		}
	}
	save("close", 2, domain.CheckerAction, checker(0.120, 0.100, -0.300))
	save("obvious", 3, domain.CheckerAction, checker(0.300, 0.050))
	save("single", 4, domain.CheckerAction, checker(0.400))
	save("bare", 5, domain.CheckerAction, nil)
	save("double", 6, domain.CubeAction, cube("Double"))
	save("take", 7, domain.CubeAction, cube("Take"))

	cases := []struct {
		f    domain.SearchFilters
		want []string
	}{
		{domain.SearchFilters{MarginFilter: "mg<20"}, []string{"close"}},
		{domain.SearchFilters{MarginFilter: "mg<40"}, []string{"close", "double"}},
		{domain.SearchFilters{MarginFilter: "mg>100"}, []string{"obvious", "take"}},
		{domain.SearchFilters{MarginFilter: "mg20,250"}, []string{"close", "obvious", "double"}},
		{domain.SearchFilters{MarginFilter: "mg>0"}, []string{"close", "obvious", "double", "take"}},
		{domain.SearchFilters{MarginFilter: "mg<40", MirrorFilter: true}, []string{"close", "double"}},
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
			t.Errorf("%s M %v: got %v, want %v (ids %v)", c.f.MarginFilter, c.f.MirrorFilter, got, want, ids)
		}
	}
}
