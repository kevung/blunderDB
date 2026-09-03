package server

import (
	"context"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// searchFindReq mirrors storage.ListOpts over the wire (see handlers_positions.go's
// positionListReq): an all-zero Limit/Offset keeps the unbounded scan every
// caller got before pagination was pushed to SQL (B.10, #178).
type searchFindReq struct {
	Filters domain.SearchFilters `json:"filters"`
	Limit   int                  `json:"limit"`
	Offset  int                  `json:"offset"`
}

func (s *Server) searchRoutes() []route {
	ss := func() storage.SearchStore { return s.opts.Storage.Search() }
	return []route{
		{http.MethodPost, "/v1/search.find", rpcStream(func(ctx context.Context, scope string, req searchFindReq) iterPositions {
			return ss().Find(ctx, scope, req.Filters, storage.ListOpts{Limit: req.Limit, Offset: req.Offset})
		})},
	}
}
