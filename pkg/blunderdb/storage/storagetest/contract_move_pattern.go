package storagetest

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// RunMovePatternColumn fixes what a move-pattern search finds now that SQL
// narrows it by analysis.best_move: the analyses whose best checker play
// contains one of the patterns, case aside, and the cube analyses whose best
// action a pattern nd, dt or dp names; LIKE's own characters in a pattern are
// plain text. It holds with the condition read first as a list of ids and as
// a clause on the walk. A row whose column was never derived is still found,
// and rederive (the backend's pass over such rows) derives it. exec, the
// backend's own connection, clears the column as a row written before it.
func RunMovePatternColumn(t *testing.T, s storage.Storage, exec sqlshared.Execer, rederive func(context.Context) error) {
	ctx := context.Background()
	save := func(n int, a *domain.PositionAnalysis) int64 {
		t.Helper()
		p := distinctPos(n)
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save position %d: %v", n, err)
		}
		if a != nil {
			if err := s.Analyses().Save(ctx, "", id, a); err != nil {
				t.Fatalf("Save analysis %d: %v", n, err)
			}
		}
		return id
	}
	checker := func(best, second string) *domain.PositionAnalysis {
		return &domain.PositionAnalysis{AnalysisType: "CheckerMove", CheckerAnalysis: &domain.CheckerAnalysis{
			Moves: []domain.CheckerMove{{Index: 0, Move: best, Equity: 0.1}, {Index: 1, Move: second, Equity: 0}}}}
	}
	cube := func(nd, dt, dp float64) *domain.PositionAnalysis {
		return &domain.PositionAnalysis{AnalysisType: "DoublingCube", DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{
			BestCubeAction: "No Double", CubefulNoDoubleError: nd, CubefulDoubleTakeError: dt, CubefulDoublePassError: dp}}
	}
	a := save(0, checker("18/14 8/6", "24/20 13/11"))
	b := save(1, checker("18/14 13/11", "8/6 6/5"))
	c := save(2, checker("24/20 8/6", "13/11 6/5"))
	bar := save(3, checker("Bar/22 13/11", "24/21 6/5"))
	noDouble := save(4, cube(0, -0.1, -0.2))
	doubleTake := save(5, cube(-0.1, 0, -0.2))
	wild := save(6, checker("13/1% 6/5_", "8/7 6/5"))
	save(7, nil)

	cases := []struct {
		filter string
		want   []int64
	}{
		{`m"18/14 8/6"`, []int64{a}},
		{`m"8/6"`, []int64{a, c}},
		{`m"18/14;24/20"`, []int64{a, b, c}},
		{`m"BAR/22"`, []int64{bar}},
		{`m"nd"`, []int64{noDouble}},
		{`m"dt;8/6"`, []int64{a, c, doubleTake}},
		{`m"dp"`, nil},
		// The best play alone: 13/11 is a's second candidate.
		{`m"13/11"`, []int64{b, bar}},
		{`m"1_/1"`, nil},
		{`m"1%"`, []int64{wild}},
		{`m"6/5_"`, []int64{wild}},
		{`m"6/5\"`, nil},
		{`m""`, []int64{a, b, c, bar, wild}},
	}
	check := func(when string) {
		t.Helper()
		for _, limit := range []int{-1, 1, 5000} {
			restore := sqlshared.SetNarrowLimit(limit)
			for _, tc := range cases {
				got := searchIDs(t, s, domain.SearchFilters{MovePatternFilter: tc.filter})
				slices.Sort(got)
				want := slices.Clone(tc.want)
				slices.Sort(want)
				if fmt.Sprint(got) != fmt.Sprint(want) {
					t.Errorf("%s, narrow limit %d: %s found %v, want %v", when, limit, tc.filter, got, want)
				}
			}
			restore()
		}
	}
	check("derived at write")

	where, whereArgs := `position_id = ?`, []any{b}
	if col := exec.ScopeColumn(); col == "tenant_id" {
		where, whereArgs = col+` = ? AND `+where, []any{exec.ScopeArg(""), b}
	}
	stored := func() *string {
		var v *string
		if err := exec.QueryRow(ctx, `SELECT best_move FROM analysis WHERE `+where, whereArgs...).Scan(&v); err != nil {
			t.Fatalf("read best_move: %v", err)
		}
		return v
	}
	if v := stored(); v == nil || *v != "18/14 13/11" {
		t.Fatalf("best_move written %v, want 18/14 13/11", v)
	}
	if _, err := exec.Exec(ctx, `UPDATE analysis SET best_move = NULL WHERE `+where, whereArgs...); err != nil {
		t.Fatalf("clear best_move: %v", err)
	}
	check("not derived yet")
	if err := rederive(ctx); err != nil {
		t.Fatalf("rederive: %v", err)
	}
	if v := stored(); v == nil || *v != "18/14 13/11" {
		t.Errorf("best_move after the pass %v, want 18/14 13/11", v)
	}
	check("derived by the pass")

	// Saved again with the same best play, the row keeps its column.
	if err := s.Analyses().Save(ctx, "", b, checker("18/14 13/11", "8/6 6/5")); err != nil {
		t.Fatalf("Save analysis again: %v", err)
	}
	if v := stored(); v == nil || *v != "18/14 13/11" {
		t.Errorf("best_move after the same play saved again %v, want 18/14 13/11", v)
	}

	// A binary older than the column rewrites the blob alone, with a new
	// best play: the column must not keep naming the former one.
	blob, err := engine.EncodeAnalysisForStorage(checker("24/20 6/5", "18/14 13/11"))
	if err != nil {
		t.Fatalf("encode analysis: %v", err)
	}
	if _, err := exec.Exec(ctx, `UPDATE analysis SET data = ? WHERE `+where, append([]any{blob}, whereArgs...)...); err != nil {
		t.Fatalf("rewrite the blob alone: %v", err)
	}
	if v := stored(); v != nil {
		t.Errorf("best_move after the blob alone was rewritten %q, want NULL", *v)
	}
	for _, tc := range []struct {
		filter string
		want   []int64
	}{
		{`m"24/20 6/5"`, []int64{b}},
		{`m"18/14 13/11"`, nil},
		{`m"13/11"`, []int64{bar}},
	} {
		got := searchIDs(t, s, domain.SearchFilters{MovePatternFilter: tc.filter})
		slices.Sort(got)
		if fmt.Sprint(got) != fmt.Sprint(tc.want) {
			t.Errorf("after an older binary's write: %s found %v, want %v", tc.filter, got, tc.want)
		}
	}
}
