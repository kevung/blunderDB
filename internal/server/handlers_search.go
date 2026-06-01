package server

import (
	"context"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type searchFindReq struct {
	Filters domain.SearchFilters `json:"filters"`
}

func (s *Server) searchRoutes() []route {
	ss := func() storage.SearchStore { return s.opts.Storage.Search() }
	return []route{
		{http.MethodPost, "/v1/search.find", rpcStream(func(ctx context.Context, scope string, req searchFindReq) iterPositions {
			return ss().Find(ctx, scope, req.Filters)
		})},
	}
}
