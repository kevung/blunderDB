package sqlshared_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// A Go-filtered scan read a chunk at a time must answer what one unbounded
// scan answers, on both ways of resuming: after the last id (id order) and
// by OFFSET (any other order), and across a window straddling two chunks.
func TestSearchScanAcrossChunks(t *testing.T) {
	ctx := context.Background()
	s, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	defer s.Close()
	var kept []int64
	for n := 1; n <= 23; n++ {
		p := domain.InitializePosition()
		p.DecisionType = domain.CheckerAction
		p.Score = [2]int{n, 0}
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save %d: %v", n, err)
		}
		if n%3 != 0 {
			if _, err := s.Comments().Add(ctx, "", id, "keep"); err != nil {
				t.Fatalf("comment: %v", err)
			}
			kept = append(kept, id)
		}
	}
	defer sqlshared.SetSearchChunk(4)()

	for _, sort := range []string{"", "error"} {
		f := domain.SearchFilters{SearchText: "keep", Sort: sort}
		all, err := s.Search().FindIDs(ctx, "", f, storage.ListOpts{})
		if err != nil || !reflect.DeepEqual(all, kept) {
			t.Fatalf("sort %q: FindIDs = %v, %v; want %v", sort, all, err, kept)
		}
		page, err := s.Search().FindIDs(ctx, "", f, storage.ListOpts{Offset: 5, Limit: 6})
		if err != nil || !reflect.DeepEqual(page, kept[5:11]) {
			t.Errorf("sort %q: window [5,11) = %v, %v; want %v", sort, page, err, kept[5:11])
		}
		if n, err := s.Search().Count(ctx, "", f); err != nil || n != len(kept) {
			t.Errorf("sort %q: Count = %d, %v; want %d", sort, n, err, len(kept))
		}
		if at, ok, err := s.Search().IndexOf(ctx, "", f, kept[13]); err != nil || !ok || at != 13 {
			t.Errorf("sort %q: IndexOf = %d, %v, %v; want 13", sort, at, ok, err)
		}
	}
}
