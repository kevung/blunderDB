package sqlite_test

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// TestSearchSortByError exercises the search ORDER BY added for POS-B-T05: the
// "error" sort surfaces the biggest blunders first, and a position without an
// analysis is relegated to the end (NULLS LAST), never hidden.
func TestSearchSortByError(t *testing.T) {
	s := openTempDB(t)
	ctx := context.Background()
	fp := func(v float64) *float64 { return &v }

	// Save four distinct positions; three carry a checker analysis whose PLAYED
	// move has a known equity error, the fourth has none.
	save := func(i int, equityError *float64) int64 {
		p := benchPos(i)
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("save position %d: %v", i, err)
		}
		if equityError != nil {
			a := &domain.PositionAnalysis{
				PlayedMoves: []string{"24/18 13/11"},
				CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
					{Move: "13/11 24/18", Equity: 0.5},                       // best (different order, normalizes equal)
					{Move: "24/18 13/11", Equity: 0.5, EquityError: equityError},
				}},
			}
			if err := s.Analyses().Save(ctx, "", id, a); err != nil {
				t.Fatalf("save analysis %d: %v", i, err)
			}
		}
		return id
	}

	idBig := save(1, fp(0.30))   // worst blunder → first
	idMid := save(2, fp(0.20))   // middle
	idSmall := save(3, fp(0.10)) // smallest error
	idNone := save(4, nil)       // no analysis → last

	order := func(sort string) []int64 {
		var ids []int64
		for p, err := range s.Search().Find(ctx, "", domain.SearchFilters{Sort: sort}) {
			if err != nil {
				t.Fatalf("find(%q): %v", sort, err)
			}
			ids = append(ids, p.ID)
		}
		return ids
	}

	got := order("error")
	if len(got) != 4 {
		t.Fatalf("expected 4 results, got %d (%v)", len(got), got)
	}
	want := []int64{idBig, idMid, idSmall, idNone}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("error sort wrong: got %v want %v", got, want)
		}
	}

	// Default sort keeps the stable engine order (ascending id), no relegation.
	def := order("")
	if len(def) != 4 || def[0] != idBig {
		t.Fatalf("default sort should be by id ascending: %v", def)
	}
}
