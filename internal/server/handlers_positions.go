package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type positionReq struct {
	Position *domain.Position `json:"position"`
}

type existsReq struct {
	Zobrist uint64 `json:"zobrist"`
}

type xgidReq struct {
	XGID string `json:"xgid"`
}

type existsResp struct {
	ID    int64 `json:"id"`
	Found bool  `json:"found"`
}

type listReq struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

func (s *Server) positionRoutes() []route {
	ps := func() storage.PositionStore { return s.opts.Storage.Positions() }
	return []route{
		{http.MethodPost, "/v1/positions.save", rpc(func(ctx context.Context, scope string, req positionReq) (idResp, error) {
			id, err := ps().Save(ctx, scope, req.Position)
			return idResp{ID: id}, err
		})},
		{http.MethodPost, "/v1/positions.update", rpcVoid(func(ctx context.Context, scope string, req positionReq) error {
			return ps().Update(ctx, scope, req.Position)
		})},
		{http.MethodPost, "/v1/positions.load", rpc(func(ctx context.Context, scope string, req idReq) (*domain.Position, error) {
			return ps().Load(ctx, scope, req.ID)
		})},
		{http.MethodPost, "/v1/positions.exists", rpc(func(ctx context.Context, scope string, req existsReq) (existsResp, error) {
			id, found, err := ps().Exists(ctx, scope, req.Zobrist)
			return existsResp{ID: id, Found: found}, err
		})},
		// Decode an XGID string into a Position (pure; no storage). Generic and
		// useful to the Desktop too (paste an XGID). Invalid input → 4xx.
		{http.MethodPost, "/v1/positions.fromXGID", rpc(func(ctx context.Context, scope string, req xgidReq) (*domain.Position, error) {
			pos, err := domain.DecodeXGID(req.XGID)
			if err != nil {
				return nil, fmt.Errorf("%w: %v", storage.ErrInvalid, err)
			}
			return &pos, nil
		})},
		{http.MethodPost, "/v1/positions.delete", rpcVoid(func(ctx context.Context, scope string, req idReq) error {
			return ps().Delete(ctx, scope, req.ID)
		})},
		{http.MethodPost, "/v1/positions.list", rpcStream(func(ctx context.Context, scope string, req listReq) iterPositions {
			return ps().List(ctx, scope, storage.ListOpts{Limit: req.Limit, Offset: req.Offset})
		})},
	}
}
