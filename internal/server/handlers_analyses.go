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
			if req.Analysis == nil {
				return errMissing("analysis")
			}
			return as().Save(ctx, scope, req.PositionID, req.Analysis)
		})},
		{http.MethodPost, "/v1/analyses.load", rpc(func(ctx context.Context, scope string, req positionIDReq) (*domain.PositionAnalysis, error) {
			return as().Load(ctx, scope, req.PositionID)
		})},
		// Analyses of the listed positions, in id order; an id without one is
		// skipped, as positions.loadByIds does.
		{http.MethodPost, "/v1/analyses.loadByIds", rpc(func(ctx context.Context, scope string, req idsReq) ([]domain.PositionAnalysis, error) {
			byID, err := as().LoadMany(ctx, scope, req.IDs)
			if err != nil {
				return nil, err
			}
			out := make([]domain.PositionAnalysis, 0, len(req.IDs))
			for _, id := range req.IDs {
				if a := byID[id]; a != nil {
					cp := *a
					cp.PositionID = int(id)
					out = append(out, cp)
				}
			}
			return out, nil
		})},
		{http.MethodPost, "/v1/analyses.delete", rpcVoid(func(ctx context.Context, scope string, req positionIDReq) error {
			return as().Delete(ctx, scope, req.PositionID)
		})},
		// Réparation des colonnes dénormalisées, explicite et jamais à
		// l'ouverture ; le compte rendu dit combien de lignes ont RÉELLEMENT
		// changé.
		{http.MethodPost, "/v1/analyses.repair", rpc(func(ctx context.Context, scope string, _ struct{}) (repairResp, error) {
			n, err := as().RepairDenormalisedColumns(ctx, scope)
			return repairResp{Repaired: n}, err
		})},
	}
}

// repairResp reports how many analyses had at least one denormalised column
// rewritten.
type repairResp struct {
	Repaired int `json:"repaired"`
}
