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

// analysisLoadReq is positionIDReq with the join of the played moves, off
// by default.
type analysisLoadReq struct {
	PositionID int64 `json:"positionId"`
	WithPlayed bool  `json:"withPlayed,omitempty"`
}

type positionIDReq struct {
	PositionID int64 `json:"positionId"`
}

func (s *Server) analysisRoutes() []route {
	as := func() storage.AnalysisStore { return s.opts.Storage.Analyses() }
	return []route{
		// Gauges of the analysis queue: positions with and without an
		// analysis, and the counters of the tenant's running or last sweep.
		{http.MethodPost, "/v1/analyses.queueStatus", rpc(s.analysisQueueStatus)},
		// API: Stores the analysis as given, replacing the one the position
		// carries: nothing of the stored analysis is kept. To add an analysis
		// to the stored one, call analyses.merge.
		{http.MethodPost, "/v1/analyses.save", rpcVoid(func(ctx context.Context, scope string, req analysisSaveReq) error {
			if req.Analysis == nil {
				return errMissing("analysis")
			}
			return as().Save(ctx, scope, req.PositionID, req.Analysis)
		})},
		// API: Folds the analysis into the one the position carries, in one
		// transaction, as the desktop and the CLI save it: the candidate moves
		// of both (keyed by notation), every engine's cube analysis, the union
		// of the plays, the rollouts of both, the stored creation date.
		{http.MethodPost, "/v1/analyses.merge", rpcVoid(func(ctx context.Context, scope string, req analysisSaveReq) error {
			if req.Analysis == nil {
				return errMissing("analysis")
			}
			return storage.SaveMergedAnalysis(ctx, as(), scope, req.PositionID, *req.Analysis)
		})},
		// API: The analysis as stored, 404 when the position has none. With
		// withPlayed, its played moves and cube actions are joined with those
		// the matches played at the position, normalised and sorted, and its
		// cube analyses come XG first, as the desktop and the CLI read it.
		{http.MethodPost, "/v1/analyses.load", rpc(func(ctx context.Context, scope string, req analysisLoadReq) (*domain.PositionAnalysis, error) {
			if req.WithPlayed {
				return storage.LoadAnalysisWithPlayed(ctx, s.opts.Storage, scope, req.PositionID)
			}
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
