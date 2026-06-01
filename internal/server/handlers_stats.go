package server

import (
	"context"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type statsComputeReq struct {
	Filter storage.StatsFilter `json:"filter"`
}

type statsSelectionReq struct {
	Filter    storage.StatsFilter   `json:"filter"`
	Selection storage.SelectionSpec `json:"selection"`
}

type idsResp struct {
	PositionIDs []int64 `json:"positionIds"`
}

func (s *Server) statsRoutes() []route {
	ss := func() storage.StatsStore { return s.opts.Storage.Stats() }
	return []route{
		{http.MethodPost, "/v1/stats.dateRange", rpc(func(ctx context.Context, scope string, _ struct{}) (storage.StatsDateRange, error) {
			return ss().DateRange(ctx, scope)
		})},
		{http.MethodPost, "/v1/stats.compute", rpc(func(ctx context.Context, scope string, req statsComputeReq) (*storage.StatsResult, error) {
			return ss().Compute(ctx, scope, req.Filter)
		})},
		{http.MethodPost, "/v1/stats.positionIdsBySelection", rpc(func(ctx context.Context, scope string, req statsSelectionReq) (idsResp, error) {
			ids, err := ss().PositionIDsBySelection(ctx, scope, req.Filter, req.Selection)
			return idsResp{PositionIDs: ids}, err
		})},
		{http.MethodPost, "/v1/stats.positionIdsByTournament", rpc(func(ctx context.Context, scope string, req tournamentIDReq) (idsResp, error) {
			ids, err := ss().PositionIDsByTournament(ctx, scope, req.TournamentID)
			return idsResp{PositionIDs: ids}, err
		})},
		{http.MethodPost, "/v1/stats.positionIdsByMatch", rpc(func(ctx context.Context, scope string, req matchIDReq) (idsResp, error) {
			ids, err := ss().PositionIDsByMatch(ctx, scope, req.MatchID)
			return idsResp{PositionIDs: ids}, err
		})},
		{http.MethodPost, "/v1/stats.playerNames", rpc(func(ctx context.Context, scope string, _ struct{}) ([]storage.PlayerFrequency, error) {
			return ss().PlayerNames(ctx, scope)
		})},
		{http.MethodPost, "/v1/stats.matchDetail", rpc(func(ctx context.Context, scope string, req matchIDReq) (*storage.MatchDetailStats, error) {
			return ss().MatchDetail(ctx, scope, req.MatchID)
		})},
	}
}
