package storagetest

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

func saveWindowPositions(ctx context.Context, t *testing.T, s storage.Storage, n int) []int64 {
	t.Helper()
	ids := make([]int64, 0, n)
	for i := 0; i < n; i++ {
		p := domain.InitializePosition()
		p.Score = [2]int{3 + i, 5}
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save %d: %v", i, err)
		}
		ids = append(ids, id)
	}
	return ids
}

// CountUpTo answers Count while the result fits under max, and says so when it does not: the caller
// shows "more than max" rather than a wrong number. A Go-only filter must count the same way.
func testSearchCountUpTo(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	saveWindowPositions(ctx, t, s, 12)
	var onEmptyPoint domain.Position
	onEmptyPoint.Board.Points[3] = domain.Point{Checkers: 1, Color: domain.Black}
	for _, tc := range []struct {
		name   string
		f      domain.SearchFilters
		max    int
		n      int
		capped bool
	}{
		{"under", domain.SearchFilters{}, 50, 12, false},
		{"exactly", domain.SearchFilters{}, 12, 12, false},
		{"over", domain.SearchFilters{}, 5, 5, true},
		{"go phase, over", domain.SearchFilters{ExcludeFilter: onEmptyPoint}, 5, 5, true},
		{"none", domain.SearchFilters{PositionIDsFilter: "999999999;999999998"}, 5, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n, capped, err := s.Search().CountUpTo(ctx, "", tc.f, tc.max)
			if err != nil || n != tc.n || capped != tc.capped {
				t.Fatalf("CountUpTo = %d, %v, %v; want %d, %v", n, capped, err, tc.n, tc.capped)
			}
			full, err := s.Search().Count(ctx, "", tc.f)
			if err != nil {
				t.Fatal(err)
			}
			if !tc.capped && full != n {
				t.Fatalf("CountUpTo %d differs from Count %d", n, full)
			}
		})
	}
}

// ExcludePositionIDs takes rows out of the result BEFORE the window, so a page stays full and the
// count stays true, whatever the sort.
func testSearchExcludePositionIDs(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ids := saveWindowPositions(ctx, t, s, 10)
	f := domain.SearchFilters{ExcludePositionIDs: fmt.Sprintf("%d,%d", ids[0], ids[4])}

	n, err := s.Search().Count(ctx, "", f)
	if err != nil || n != 8 {
		t.Fatalf("Count = %d, %v; want 8", n, err)
	}
	got, err := s.Search().FindIDs(ctx, "", f, storage.ListOpts{Limit: 4})
	if err != nil {
		t.Fatal(err)
	}
	want := []int64{ids[1], ids[2], ids[3], ids[5]}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("FindIDs = %v; want %v", got, want)
	}
	// A malformed id must not silently widen the result, nor narrow it to nothing.
	for _, bad := range []string{",", strings.Repeat(" ", 3)} {
		n, err := s.Search().Count(ctx, "", domain.SearchFilters{ExcludePositionIDs: bad})
		if err != nil || n != 10 {
			t.Fatalf("blank list %q: Count = %d, %v; want 10", bad, n, err)
		}
	}
}

// The window is a cut of the unbounded result, whatever the plan behind it: pages read one after the
// other give the same rows in the same order, for a filter or a sort that reads the analysis and for
// one that does not.
func testSearchWindowsCutTheFullResult(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ids := saveWindowPositions(ctx, t, s, 20)
	for i, id := range ids {
		if i%4 == 3 {
			continue // no analysis: its sort key is NULL and goes last
		}
		errv := float64(i%5) / 10
		a := &domain.PositionAnalysis{AnalysisType: "CheckerMove", PlayedMove: "13/7", CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
			{Index: 0, Move: "24/18", Equity: 0.1, PlayerWinChance: float64(50 + i)},
			{Index: 1, Move: "13/7", Equity: 0.1 - errv, EquityError: &errv, PlayerWinChance: float64(50 + i)},
		}}}
		if err := s.Analyses().Save(ctx, "", id, a); err != nil {
			t.Fatalf("Analyses().Save: %v", err)
		}
	}
	for _, sort := range []string{"", "error", "winrate", "close", "pip_asc", "pip_desc", "date_asc", "date_desc"} {
		for _, name := range []string{"all", "by analysis"} {
			f := domain.SearchFilters{Sort: sort}
			if name == "by analysis" {
				f.MoveErrorFilter = "E>50"
			}
			var whole []int64
			for p, err := range s.Search().Find(ctx, "", f, storage.ListOpts{}) {
				if err != nil {
					t.Fatal(err)
				}
				whole = append(whole, p.ID)
			}
			var pages []int64
			for off := 0; ; off += 7 {
				n := 0
				for p, err := range s.Search().Find(ctx, "", f, storage.ListOpts{Limit: 7, Offset: off}) {
					if err != nil {
						t.Fatal(err)
					}
					pages = append(pages, p.ID)
					n++
				}
				if n < 7 {
					break
				}
			}
			if fmt.Sprint(whole) != fmt.Sprint(pages) {
				t.Fatalf("sort %q, %s: pages %v, whole %v", sort, name, pages, whole)
			}
		}
	}
}

// A condition on another table is written either as the list of its matches or as a
// subquery, by how many there are: a test written against few rows also holds of the
// form a large library gets.
func asCommonCondition(test func(*testing.T, storage.Storage)) func(*testing.T, storage.Storage) {
	return func(t *testing.T, s storage.Storage) {
		defer sqlshared.SetNarrowLimit(-1)()
		test(t, s)
	}
}

// asHeavyCondition: a filter on the match, checked position by position.
func asHeavyCondition(test func(*testing.T, storage.Storage)) func(*testing.T, storage.Storage) {
	return func(t *testing.T, s storage.Storage) {
		defer sqlshared.SetNarrowLimit(0)()
		test(t, s)
	}
}

// A cube decision plays no move, so the error sort of cube decisions reads as the id order, the unanalysed
// ones after. A row that keeps a played-move error on a cube decision would break the shortcut: the
// writer must not produce one.
func testSearchErrorSortOfCubeDecisions(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	var ids []int64
	for i := 0; i < 5; i++ {
		p := domain.InitializePosition()
		p.Score = [2]int{3 + i, 5}
		p.DecisionType = domain.CubeAction
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save: %v", err)
		}
		ids = append(ids, id)
	}
	for _, id := range ids[1:] {
		a := domain.PositionAnalysis{AnalysisType: "DoublingCube", PlayedCubeAction: "No double",
			DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{AnalysisDepth: "2-ply", BestCubeAction: "Double, take", CubefulNoDoubleError: -0.2}}
		if err := s.Analyses().Save(ctx, "", id, &a); err != nil {
			t.Fatalf("Save analysis: %v", err)
		}
	}
	checker := domain.InitializePosition()
	checker.Score = [2]int{20, 21}
	if _, err := s.Positions().Save(ctx, "", &checker); err != nil {
		t.Fatalf("Save: %v", err)
	}
	f := domain.SearchFilters{DecisionTypeFilter: true, Sort: "error"}
	f.Filter.DecisionType = domain.CubeAction
	f.Filter.PlayerOnRoll = domain.Black
	want := append(append([]int64{}, ids[1:]...), ids[0])
	got, err := s.Search().FindIDs(ctx, "", f, storage.ListOpts{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("error sort of cube decisions = %v, want %v", got, want)
	}
	page, err := s.Search().FindIDs(ctx, "", f, storage.ListOpts{Limit: 2, Offset: 3})
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(page) != fmt.Sprint(want[3:]) {
		t.Fatalf("window across the unanalysed = %v, want %v", page, want[3:])
	}
}
