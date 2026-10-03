package server

import (
	"context"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type statsComputeReq struct {
	Filter storage.StatsFilter `json:"filter"`
}

type statsTrainingReq struct {
	Filter storage.StatsFilter `json:"filter"`
	// Window is "week" (default) or "month".
	Window string `json:"window"`
}

type studyIDsReq struct {
	Filter storage.StatsFilter `json:"filter"`
	// Rank is 0 for the worst groups, n>0 for the n-th alone.
	Rank int `json:"rank"`
	// Size draws at most that many positions; 0 returns them all.
	Size int `json:"size"`
}

type statsSelectionReq struct {
	Filter    storage.StatsFilter   `json:"filter"`
	Selection storage.SelectionSpec `json:"selection"`
}

type idsResp struct {
	PositionIDs []int64 `json:"positionIds"`
}

// matchBadgesReq scopes a badge computation to the given match ids. An empty
// list falls back to the whole-database scan.
type matchBadgesReq struct {
	MatchIDs []int64 `json:"matchIds"`
}

// matchBadgesResp keys each badge by its match id (JSON object keys are the
// stringified ids).
type matchBadgesResp struct {
	Badges map[int64]storage.MatchBadge `json:"badges"`
}

// tournamentBadgesResp keys each badge by its tournament id (JSON object keys
// are the stringified ids).
type tournamentBadgesResp struct {
	Badges map[int64]storage.TournamentBadge `json:"badges"`
}

func (s *Server) statsRoutes() []route {
	ss := func() storage.StatsStore { return s.opts.Storage.Stats() }
	return []route{
		{http.MethodPost, "/v1/stats.dateRange", rpc(func(ctx context.Context, scope string, _ struct{}) (storage.StatsDateRange, error) {
			return ss().DateRange(ctx, scope)
		})},
		// The per-match statistics recomputed from scratch (repair --stats).
		{http.MethodPost, "/v1/stats.rebuildMatchStats", rpc(func(ctx context.Context, scope string, _ struct{}) (rebuildMatchStatsResp, error) {
			n, err := ss().RebuildMatchStats(ctx, scope, nil)
			return rebuildMatchStatsResp{Matches: n}, err
		})},
		{http.MethodPost, "/v1/stats.compute", rpc(func(ctx context.Context, scope string, req statsComputeReq) (*storage.StatsResult, error) {
			return ss().Compute(ctx, scope, req.Filter)
		})},
		// Les erreurs du filtre groupées par plan de jeu et par thème, la
		// plus coûteuse d'abord ; chaque groupe porte ses positions.
		{http.MethodPost, "/v1/stats.recurringErrors", rpc(func(ctx context.Context, scope string, req statsComputeReq) (*storage.RecurringErrors, error) {
			return ss().RecurringErrors(ctx, scope, req.Filter)
		})},
		// Le PR du quiz Décision et la rétention Anki par fenêtre calendaire,
		// sur le même calendrier que le PR des matchs réels.
		{http.MethodPost, "/v1/stats.training", rpc(func(ctx context.Context, scope string, req statsTrainingReq) (*storage.TrainingStats, error) {
			if req.Window == "" {
				req.Window = storage.TrainingWindowWeek
			}
			return storage.ComputeTrainingStats(ctx, s.opts.Storage, scope, req.Filter, req.Window)
		})},
		// Les positions des erreurs récurrentes du filtre, tirées pour un quiz si Size > 0.
		{http.MethodPost, "/v1/stats.studyIds", rpc(func(ctx context.Context, scope string, req studyIDsReq) (idsResp, error) {
			ids, err := storage.StudyIDs(ctx, s.opts.Storage, scope, req.Filter, req.Rank, req.Size)
			return idsResp{PositionIDs: ids}, err
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
		{http.MethodPost, "/v1/stats.playerTable", rpc(func(ctx context.Context, scope string, req statsComputeReq) ([]storage.PlayerRow, error) {
			return ss().PlayerTable(ctx, scope, req.Filter)
		})},
		{http.MethodPost, "/v1/stats.matchDetail", rpc(func(ctx context.Context, scope string, req matchIDReq) (*storage.MatchDetailStats, error) {
			return ss().MatchDetail(ctx, scope, req.MatchID)
		})},
		{http.MethodPost, "/v1/stats.matchMoveGrades", rpc(func(ctx context.Context, scope string, req matchIDReq) ([]storage.MoveGrade, error) {
			return ss().MatchMoveGrades(ctx, scope, req.MatchID)
		})},
		{http.MethodPost, "/v1/stats.matchBadges", rpc(func(ctx context.Context, scope string, req matchBadgesReq) (matchBadgesResp, error) {
			badges, err := ss().MatchBadges(ctx, scope, req.MatchIDs)
			return matchBadgesResp{Badges: badges}, err
		})},
		{http.MethodPost, "/v1/stats.tournamentBadges", rpc(func(ctx context.Context, scope string, _ struct{}) (tournamentBadgesResp, error) {
			badges, err := ss().TournamentBadges(ctx, scope)
			return tournamentBadgesResp{Badges: badges}, err
		})},
	}
}

// rebuildMatchStatsResp reports how many matches had their statistics recomputed.
type rebuildMatchStatsResp struct {
	Matches int `json:"matches"`
}
