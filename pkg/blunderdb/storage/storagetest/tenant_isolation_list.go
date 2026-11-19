package storagetest

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// checkMatchResultIsolation: a tenant's matches list their own result, and a
// swap in one tenant leaves the other's result alone.
func checkMatchResultIsolation(t *testing.T, ctx context.Context, s storage.Storage, a, b string) {
	ms := s.Matches()
	ma := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 5, FinalScore: &[2]int32{5, 2}, Winner: domain.WinnerPlayer1}
	idA, err := ms.Save(ctx, a, &ma)
	if err != nil {
		t.Fatal(err)
	}
	mb := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 5, FinalScore: &[2]int32{1, 5}, Winner: domain.WinnerPlayer2}
	idB, err := ms.Save(ctx, b, &mb)
	if err != nil {
		t.Fatal(err)
	}
	if err := ms.SwapPlayers(ctx, a, idA); err != nil {
		t.Fatal(err)
	}
	for scope, want := range map[string]struct {
		id     int64
		final  [2]int32
		winner int32
	}{a: {idA, [2]int32{2, 5}, domain.WinnerPlayer2}, b: {idB, [2]int32{1, 5}, domain.WinnerPlayer2}} {
		n := 0
		for m, err := range ms.List(ctx, scope, storage.MatchListOpts{}) {
			if err != nil {
				t.Fatal(err)
			}
			n++
			if m.ID != want.id || m.FinalScore == nil || *m.FinalScore != want.final || m.Winner != want.winner {
				t.Errorf("tenant %s lists %d %v winner %d; want %d %v winner %d", scope, m.ID, m.FinalScore, m.Winner, want.id, want.final, want.winner)
			}
		}
		if n != 1 {
			t.Errorf("tenant %s lists %d matches, want 1", scope, n)
		}
	}
}

// checkListFieldsIsolation: the list fields of a page answer only the
// tenant's positions, and their collections only its own.
func checkListFieldsIsolation(t *testing.T, ctx context.Context, s storage.Storage, a, b string) {
	prov, ok := s.(storage.ListFieldsProvider)
	if !ok {
		t.Skip("this backend answers no list fields")
	}
	p := checkerPos()
	id, err := s.Positions().Save(ctx, a, &p)
	if err != nil {
		t.Fatal(err)
	}
	c, err := s.Collections().Create(ctx, a, "Kept", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Collections().AddPosition(ctx, a, c, id); err != nil {
		t.Fatal(err)
	}
	if got, err := prov.PositionListFields().ListFields(ctx, b, []int64{id}); err != nil || len(got) != 0 {
		t.Errorf("tenant %s reads the list fields of %s's position: %+v, %v", b, a, got, err)
	}
	got, err := prov.PositionListFields().ListFields(ctx, a, []int64{id})
	if err != nil || len(got) != 1 || len(got[0].Collections) != 1 || got[0].Collections[0].ID != c || got[0].Analysed {
		t.Errorf("tenant %s list fields = %+v, %v; want its position, unanalysed, in its collection", a, got, err)
	}
}
