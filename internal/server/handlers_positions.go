package server

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
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

// xgpReq carries a base64-encoded .xgp single-position file.
type xgpReq struct {
	Data string `json:"data"`
}

// xgpResp is the parsed position plus its analysis (if the file carried one).
type xgpResp struct {
	Position *domain.Position         `json:"position"`
	Analysis *domain.PositionAnalysis `json:"analysis,omitempty"`
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
		// Parse a single-position XG file (.xgp) into a Position + optional analysis
		// (pure; no storage). The file bytes arrive base64-encoded. Whole matches
		// (.xg) are a separate import, not this single-position path.
		{http.MethodPost, "/v1/positions.fromXGP", rpc(func(ctx context.Context, scope string, req xgpReq) (xgpResp, error) {
			raw, err := base64.StdEncoding.DecodeString(req.Data)
			if err != nil {
				return xgpResp{}, fmt.Errorf("%w: bad base64 data", storage.ErrInvalid)
			}
			f, err := os.CreateTemp("", "blunderdb-*.xgp")
			if err != nil {
				return xgpResp{}, err
			}
			defer os.Remove(f.Name())
			if _, err := f.Write(raw); err != nil {
				f.Close()
				return xgpResp{}, err
			}
			f.Close()

			graphs, err := ingest.MapXGPPosition(f.Name())
			if err != nil {
				return xgpResp{}, fmt.Errorf("%w: %v", storage.ErrInvalid, err)
			}
			if len(graphs) == 0 || graphs[0].Position == nil {
				return xgpResp{}, fmt.Errorf("%w: no position in file", storage.ErrInvalid)
			}
			resp := xgpResp{Position: graphs[0].Position}
			if len(graphs[0].Analyses) > 0 {
				resp.Analysis = graphs[0].Analyses[0]
			}
			return resp, nil
		})},
		{http.MethodPost, "/v1/positions.delete", rpcVoid(func(ctx context.Context, scope string, req idReq) error {
			return ps().Delete(ctx, scope, req.ID)
		})},
		{http.MethodPost, "/v1/positions.list", rpcStream(func(ctx context.Context, scope string, req listReq) iterPositions {
			return ps().List(ctx, scope, storage.ListOpts{Limit: req.Limit, Offset: req.Offset})
		})},
	}
}
