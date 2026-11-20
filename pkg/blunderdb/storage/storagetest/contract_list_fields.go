// Contract case for the list fields of a page of positions: the played error
// or its absence, the date, the collections, in the order asked.
package storagetest

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func testPositionListFields(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	prov, ok := s.(storage.ListFieldsProvider)
	if !ok {
		t.Fatal("this backend answers no list fields")
	}
	lf := prov.PositionListFields()

	_, played := statsFixtureMatch(t, s, 0, "Alice", "Bob")
	loose := statsDecisionPos(t, 5)
	looseID, err := s.Positions().Save(ctx, "", &loose)
	if err != nil {
		t.Fatal(err)
	}
	cols := s.Collections()
	second, err := cols.Create(ctx, "", "Second", "")
	if err != nil {
		t.Fatal(err)
	}
	first, err := cols.Create(ctx, "", "First", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := cols.Reorder(ctx, "", []int64{first, second}); err != nil {
		t.Fatal(err)
	}
	for _, c := range []int64{second, first} {
		if err := cols.AddPosition(ctx, "", c, played[0]); err != nil {
			t.Fatal(err)
		}
	}

	got, err := lf.ListFields(ctx, "", []int64{looseID, 987654, played[0], played[1], looseID})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 || got[0].PositionID != looseID || got[1].PositionID != played[0] ||
		got[2].PositionID != played[1] || got[3].PositionID != looseID {
		t.Fatalf("ListFields = %+v: want the caller's order, the missing id skipped, the repeat answered", got)
	}

	a := got[1]
	if !a.Analysed || a.PlayedErrorMP == nil || *a.PlayedErrorMP != 50 {
		t.Errorf("played position: analysed %v error %v, want analysed with 50 mp", a.Analysed, a.PlayedErrorMP)
	}
	if a.Date != "2025-06-01" || !a.MatchDated {
		t.Errorf("played position dated %q (match %v), want its match's day 2025-06-01", a.Date, a.MatchDated)
	}
	if len(a.Collections) != 2 || a.Collections[0].ID != first || a.Collections[0].Name != "First" || a.Collections[1].ID != second {
		t.Errorf("collections = %+v, want First then Second, in the library's order", a.Collections)
	}
	if b := got[2]; b.Collections == nil || len(b.Collections) != 0 {
		t.Errorf("a position in no collection lists %#v, want an empty list", b.Collections)
	}

	l := got[0]
	if l.Analysed || l.PlayedErrorMP != nil {
		t.Errorf("unanalysed position: analysed %v error %v, want no analysis and no error", l.Analysed, l.PlayedErrorMP)
	}
	if l.MatchDated || !regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`).MatchString(l.Date) {
		t.Errorf("loose position dated %q (match %v), want the day it was written", l.Date, l.MatchDated)
	} else if d, _ := time.Parse(time.DateOnly, l.Date); time.Since(d) > 48*time.Hour || time.Until(d) > 48*time.Hour {
		t.Errorf("loose position dated %s, want about today", l.Date)
	}

	if empty, err := lf.ListFields(ctx, "", nil); err != nil || len(empty) != 0 {
		t.Errorf("ListFields(no id) = %+v, %v; want none", empty, err)
	}
}
