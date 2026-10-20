package remote

import (
	"context"
	"iter"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type searchFindReq struct {
	Filters domain.SearchFilters `json:"filters"`
	Limit   int                  `json:"limit"`
	Offset  int                  `json:"offset"`
}

func (s searchStore) Find(ctx context.Context, scope string, f domain.SearchFilters, opts storage.ListOpts) iter.Seq2[*domain.Position, error] {
	return stream[*domain.Position](ctx, s.c, scope, "search.find", searchFindReq{f, opts.Limit, opts.Offset})
}

func (s searchStore) FindIDs(ctx context.Context, scope string, f domain.SearchFilters, opts storage.ListOpts) ([]int64, error) {
	var ids []int64
	err := s.c.call(ctx, scope, "search.ids", searchFindReq{f, opts.Limit, opts.Offset}, &ids)
	return ids, err
}

func (s searchStore) Count(ctx context.Context, scope string, f domain.SearchFilters) (int, error) {
	var n int
	err := s.c.call(ctx, scope, "search.count", searchFindReq{Filters: f}, &n)
	return n, err
}

// IndexOf reads search.indexOf, which answers -1 for a position the search
// does not reach.
func (s searchStore) IndexOf(ctx context.Context, scope string, f domain.SearchFilters, id int64) (int, bool, error) {
	var i int
	if err := s.c.call(ctx, scope, "search.indexOf", struct {
		Filters domain.SearchFilters `json:"filters"`
		ID      int64                `json:"id"`
	}{f, id}, &i); err != nil {
		return 0, false, err
	}
	if i < 0 {
		return 0, false, nil
	}
	return i, true, nil
}
