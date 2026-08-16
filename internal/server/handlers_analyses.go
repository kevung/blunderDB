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
		// Réparation des colonnes dénormalisées. Explicite et jamais automatique :
		// réécrire les colonnes d'analyse de tout un tenant à la simple ouverture
		// d'une base n'est pas quelque chose qu'un outil fait dans le dos de son
		// utilisateur. Le compte rendu dit combien de lignes ont RÉELLEMENT changé,
		// ce qui distingue « quelque chose n'allait pas » de « ça a tourné ».
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
