package server

import (
	"context"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type analysisSaveReq struct {
	PositionID int64                    `json:"positionId"`
	Analysis   *domain.PositionAnalysis `json:"analysis"`
}

type positionIDReq struct {
	PositionID int64 `json:"positionId"`
}

func (s *Server) analysisRoutes() []route {
	as := func() storage.AnalysisStore { return s.opts.Storage.Analyses() }
	return []route{
		{http.MethodPost, "/v1/analyses.save", rpcVoid(func(ctx context.Context, scope string, req analysisSaveReq) error {
			return as().Save(ctx, scope, req.PositionID, req.Analysis)
		})},
		{http.MethodPost, "/v1/analyses.load", rpc(func(ctx context.Context, scope string, req positionIDReq) (*domain.PositionAnalysis, error) {
			return as().Load(ctx, scope, req.PositionID)
		})},
		{http.MethodPost, "/v1/analyses.delete", rpcVoid(func(ctx context.Context, scope string, req positionIDReq) error {
			return as().Delete(ctx, scope, req.PositionID)
		})},
	}
}
