package server

import (
	"context"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type filterSaveReq struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

type filterUpdateReq struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Command string `json:"command"`
}

type editPositionSaveReq struct {
	FilterName   string `json:"filterName"`
	EditPosition string `json:"editPosition"`
}

type filterNameReq struct {
	FilterName string `json:"filterName"`
}

func (s *Server) filterRoutes() []route {
	fs := func() storage.FilterStore { return s.opts.Storage.Filters() }
	return []route{
		{http.MethodPost, "/v1/filters.save", rpc(func(ctx context.Context, scope string, req filterSaveReq) (idResp, error) {
			id, err := fs().Save(ctx, scope, req.Name, req.Command)
			return idResp{ID: id}, err
		})},
		{http.MethodPost, "/v1/filters.update", rpcVoid(func(ctx context.Context, scope string, req filterUpdateReq) error {
			return fs().Update(ctx, scope, req.ID, req.Name, req.Command)
		})},
		{http.MethodPost, "/v1/filters.delete", rpcVoid(func(ctx context.Context, scope string, req idReq) error {
			return fs().Delete(ctx, scope, req.ID)
		})},
		{http.MethodPost, "/v1/filters.list", rpcStream(func(ctx context.Context, scope string, _ struct{}) iterFilters {
			return fs().List(ctx, scope)
		})},
		{http.MethodPost, "/v1/filters.saveEditPosition", rpcVoid(func(ctx context.Context, scope string, req editPositionSaveReq) error {
			return fs().SaveEditPosition(ctx, scope, req.FilterName, req.EditPosition)
		})},
		{http.MethodPost, "/v1/filters.loadEditPosition", rpc(func(ctx context.Context, scope string, req filterNameReq) (textResp, error) {
			ep, err := fs().LoadEditPosition(ctx, scope, req.FilterName)
			return textResp{Text: ep}, err
		})},
	}
}
