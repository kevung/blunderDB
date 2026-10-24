package storagetest

import (
	"context"
	"slices"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testSearchFilterByAway pins the `aw` filter on both backends: it bounds the
// distance to victory of BOTH players, so the Crawford sentinels 0 and 1 are
// one point each, and money play never matches.
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
		"post-11": {domain.PostCrawford, 11},
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
		tok  string
		want []string
	}{
		{"aw<1", []string{"craw-1", "post-cr"}},
		{"aw:1", []string{"craw-1", "post-cr"}},
		{"aw<2", []string{"post-2", "2-2", "craw-1", "post-cr"}},
		{"aw<3", []string{"post-2", "craw-3", "2-2", "2-3", "craw-1", "post-cr"}},
		{"aw:2,3", []string{"2-2", "2-3"}},
		{"aw:4", []string{"4-4"}},
		{"aw>3", []string{"3-9", "4-4"}},
		{"aw>4", []string{"4-4"}},
		{"aw>1", []string{"post-2", "craw-3", "2-2", "2-3", "3-9", "craw-1", "post-cr", "4-4", "post-11"}},
		{"aw:0", nil},
	}
	for _, c := range cases {
		var want []int64
		for _, n := range c.want {
			want = append(want, ids[n])
		}
		got := searchIDs(t, s, domain.SearchFilters{AwayFilter: c.tok})
		slices.Sort(got)
		slices.Sort(want)
		if !slices.Equal(got, want) {
			t.Errorf("%s: got %v, want %v (ids %v)", c.tok, got, want, ids)
		}
	}

	// Mirror search runs its board filters in Go; the away bound, the same for
	// both players, holds there too.
	got := searchIDs(t, s, domain.SearchFilters{AwayFilter: "aw:1", MirrorFilter: true})
	want := []int64{ids["craw-1"], ids["post-cr"]}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("aw:1 mirror: got %v, want %v", got, want)
	}
}
