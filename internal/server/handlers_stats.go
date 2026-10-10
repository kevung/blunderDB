package server

import (
	"context"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/report"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type statsComputeReq struct {
	Filter storage.StatsFilter `json:"filter"`
}

type statsHeadToHeadReq struct {
	PlayerA string              `json:"player_a"`
	PlayerB string              `json:"player_b"`
	Filter  storage.StatsFilter `json:"filter"`
}

type statsWindowReq struct {
	Filter storage.StatsFilter `json:"filter"`
	Months int                 `json:"months"`
}

type statsRankingReq struct {
	Filter       storage.StatsFilter `json:"filter"`
	MinDecisions int                 `json:"min_decisions"`
}

type statsReportReq struct {
	Filter storage.StatsFilter `json:"filter"`
	// Language is the report's language ("en" when empty or unknown).
	Language string `json:"language"`
}

type statsReportResp struct {
	HTML string `json:"html"`
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
		// Le rapport HTML autonome du filtre, du même générateur que la GUI et
		// `stats report`.
		{http.MethodPost, "/v1/stats.report", rpc(func(ctx context.Context, scope string, req statsReportReq) (statsReportResp, error) {
			html, err := report.Build(ctx, s.opts.Storage, scope, req.Filter, req.Language, nil)
			return statsReportResp{HTML: html}, err
		})},
		// Les erreurs du filtre groupées par plan de jeu et par thème, la
		// plus coûteuse d'abord ; chaque groupe porte ses positions.
		{http.MethodPost, "/v1/stats.recurringErrors", rpc(func(ctx context.Context, scope string, req statsComputeReq) (*storage.RecurringErrors, error) {
			return ss().RecurringErrors(ctx, scope, req.Filter)
		})},
		// Le plan d'étude du filtre (ADR-0077) : les familles d'erreurs classées
		// par MWC récupérable, à part celles qui manquent de preuve.
		{http.MethodPost, "/v1/stats.studyPlan", rpc(func(ctx context.Context, scope string, req statsComputeReq) (*storage.StudyPlan, error) {
			return ss().StudyPlan(ctx, scope, req.Filter)
		})},
		// Avant/après l'étude de chaque famille étudiée (ADR-0079).
		{http.MethodPost, "/v1/stats.studyEffect", rpc(func(ctx context.Context, scope string, req statsComputeReq) (*storage.StudyEffect, error) {
			return ss().StudyEffect(ctx, scope, req.Filter)
		})},
		// Les biais signés : prises/refus, doubles par score, blots (ADR-0079).
		{http.MethodPost, "/v1/stats.biases", rpc(func(ctx context.Context, scope string, req statsComputeReq) (*storage.DirectionalBiases, error) {
			return ss().DirectionalBiases(ctx, scope, req.Filter)
		})},
		// Les positions des familles du plan, tirées pour un quiz si Size > 0.
		{http.MethodPost, "/v1/stats.studyPlanIds", rpc(func(ctx context.Context, scope string, req studyIDsReq) (idsResp, error) {
			ids, err := storage.StudyPlanIDs(ctx, s.opts.Storage, scope, req.Filter, req.Rank, req.Size)
			return idsResp{PositionIDs: ids}, err
		})},
		// La file d'étude des familles du plan, l'excès le plus grand d'abord.
		{http.MethodPost, "/v1/stats.studyPlanQueue", rpc(func(ctx context.Context, scope string, req studyIDsReq) ([]domain.StudyQueueEntry, error) {
			plan, err := ss().StudyPlan(ctx, scope, req.Filter)
			if err != nil {
				return nil, err
			}
			return plan.QueueEntries(req.Rank), nil
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
		// Les positions de chaque ligne des ventilations : le chiffre cliquable
		// est la longueur de la liste que charge la sélection "breakdown".
		{http.MethodPost, "/v1/stats.breakdownPositionCounts", rpc(func(ctx context.Context, scope string, req statsComputeReq) (storage.BreakdownPositionCounts, error) {
			return ss().BreakdownPositionCounts(ctx, scope, req.Filter)
		})},
		{http.MethodPost, "/v1/stats.positionIdsByTournament", rpc(func(ctx context.Context, scope string, req tournamentIDReq) (idsResp, error) {
			ids, err := ss().PositionIDsByTournament(ctx, scope, req.TournamentID)
			return idsResp{PositionIDs: ids}, err
		})},
		{http.MethodPost, "/v1/stats.positionIdsByMatch", rpc(func(ctx context.Context, scope string, req matchIDReq) (idsResp, error) {
			ids, err := ss().PositionIDsByMatch(ctx, scope, req.MatchID)
			return idsResp{PositionIDs: ids}, err
		})},
		{http.MethodPost, "/v1/stats.analysisEngines", rpc(func(ctx context.Context, scope string, _ struct{}) ([]string, error) {
			return ss().AnalysisEngines(ctx, scope)
		})},
		{http.MethodPost, "/v1/stats.playerNames", rpc(func(ctx context.Context, scope string, _ struct{}) ([]storage.PlayerFrequency, error) {
			return ss().PlayerNames(ctx, scope)
		})},
		{http.MethodPost, "/v1/stats.playerTable", rpc(func(ctx context.Context, scope string, req statsComputeReq) ([]storage.PlayerRow, error) {
			return ss().PlayerTable(ctx, scope, req.Filter)
		})},
		// Deux joueurs l'un contre l'autre : matchs communs, PR de chacun, bilan.
		// Les positions décidées par les deux joueurs que l'un a bien jouées et pas l'autre.
		{http.MethodPost, "/v1/stats.playerContrast", rpc(func(ctx context.Context, scope string, req statsHeadToHeadReq) (*storage.PlayerContrast, error) {
			return ss().PlayerContrast(ctx, scope, req.PlayerA, req.PlayerB, req.Filter)
		})},
		{http.MethodPost, "/v1/stats.headToHead", rpc(func(ctx context.Context, scope string, req statsHeadToHeadReq) (*storage.HeadToHead, error) {
			return ss().HeadToHead(ctx, scope, req.PlayerA, req.PlayerB, req.Filter)
		})},
		// Le PR sur une fenêtre calendaire glissante de Months mois.
		{http.MethodPost, "/v1/stats.prByWindow", rpc(func(ctx context.Context, scope string, req statsWindowReq) ([]storage.WindowStats, error) {
			return ss().PRByWindow(ctx, scope, req.Filter, req.Months)
		})},
		// Le classement par PR des joueurs d'au moins MinDecisions décisions comptées.
		{http.MethodPost, "/v1/stats.ranking", rpc(func(ctx context.Context, scope string, req statsRankingReq) ([]storage.RankedPlayer, error) {
			rows, err := ss().PlayerTable(ctx, scope, req.Filter)
			if err != nil {
				return nil, err
			}
			return storage.RankPlayers(rows, req.MinDecisions), nil
		})},
		{http.MethodPost, "/v1/stats.matchDetail", rpc(func(ctx context.Context, scope string, req matchIDReq) (*storage.MatchDetailStats, error) {
			return ss().MatchDetail(ctx, scope, req.MatchID)
		})},
		{http.MethodPost, "/v1/stats.matchMoveGrades", rpc(func(ctx context.Context, scope string, req matchIDReq) ([]storage.MoveGrade, error) {
			return ss().MatchMoveGrades(ctx, scope, req.MatchID)
		})},
		{http.MethodPost, "/v1/stats.matchDecisionLosses", rpc(func(ctx context.Context, scope string, req matchIDReq) ([]storage.DecisionLoss, error) {
			return ss().MatchDecisionLosses(ctx, scope, req.MatchID)
		})},
		{http.MethodPost, "/v1/stats.matchReview", rpc(func(ctx context.Context, scope string, req matchIDReq) (storage.MatchReview, error) {
			return ss().MatchReview(ctx, scope, req.MatchID)
		})},
		{http.MethodPost, "/v1/stats.tournamentReview", rpc(func(ctx context.Context, scope string, req tournamentReviewReq) (storage.TournamentReview, error) {
			return ss().TournamentReview(ctx, scope, req.TournamentID, req.Player)
		})},
		{http.MethodPost, "/v1/stats.matchTimeSummary", rpc(func(ctx context.Context, scope string, req matchIDReq) (storage.MatchTimeSummary, error) {
			return ss().MatchTimeSummary(ctx, scope, req.MatchID)
		})},
		{http.MethodPost, "/v1/stats.timeErrors", rpc(func(ctx context.Context, scope string, _ struct{}) ([]storage.TimeErrorRow, error) {
			return ss().TimeErrors(ctx, scope)
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

// tournamentReviewReq names the tournament and the player of a tournament
// review; an empty player is the one in the most of its matches.
type tournamentReviewReq struct {
	TournamentID int64  `json:"tournamentId"`
	Player       string `json:"player"`
}
