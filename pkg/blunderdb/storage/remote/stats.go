package remote

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type statsFilterReq struct {
	Filter storage.StatsFilter `json:"filter"`
}

type positionIDsResp struct {
	PositionIDs []int64 `json:"positionIds"`
}

func (s statsStore) DateRange(ctx context.Context, scope string) (storage.StatsDateRange, error) {
	var r storage.StatsDateRange
	err := s.c.call(ctx, scope, "stats.dateRange", nil, &r)
	return r, err
}

func (s statsStore) Compute(ctx context.Context, scope string, filter storage.StatsFilter) (*storage.StatsResult, error) {
	var r *storage.StatsResult
	if err := s.c.call(ctx, scope, "stats.compute", statsFilterReq{filter}, &r); err != nil {
		return nil, err
	}
	return r, nil
}

// PerGameType is composed over stats.compute: the daemon serves the whole
// result, and a route of its own would only save the transfer.
func (s statsStore) PerGameType(ctx context.Context, scope string, filter storage.StatsFilter) ([]storage.GameTypeStats, error) {
	r, err := s.Compute(ctx, scope, filter)
	if err != nil {
		return nil, err
	}
	return r.PerGameType, nil
}

func (s statsStore) PositionIDsBySelection(ctx context.Context, scope string, filter storage.StatsFilter, sel storage.SelectionSpec) ([]int64, error) {
	var r positionIDsResp
	err := s.c.call(ctx, scope, "stats.positionIdsBySelection", struct {
		Filter    storage.StatsFilter   `json:"filter"`
		Selection storage.SelectionSpec `json:"selection"`
	}{filter, sel}, &r)
	return r.PositionIDs, err
}

func (s statsStore) BreakdownPositionCounts(ctx context.Context, scope string, filter storage.StatsFilter) (storage.BreakdownPositionCounts, error) {
	var r storage.BreakdownPositionCounts
	err := s.c.call(ctx, scope, "stats.breakdownPositionCounts", statsFilterReq{filter}, &r)
	return r, err
}

func (s statsStore) PositionIDsByTournament(ctx context.Context, scope string, tournamentID int64) ([]int64, error) {
	var r positionIDsResp
	err := s.c.call(ctx, scope, "stats.positionIdsByTournament", struct {
		TournamentID int64 `json:"tournamentId"`
	}{tournamentID}, &r)
	return r.PositionIDs, err
}

func (s statsStore) PositionIDsByMatch(ctx context.Context, scope string, matchID int64) ([]int64, error) {
	var r positionIDsResp
	err := s.c.call(ctx, scope, "stats.positionIdsByMatch", matchIDReq{matchID}, &r)
	return r.PositionIDs, err
}

func (s statsStore) AnalysisEngines(ctx context.Context, scope string) ([]string, error) {
	var r []string
	err := s.c.call(ctx, scope, "stats.analysisEngines", nil, &r)
	return r, err
}

func (s statsStore) PlayerNames(ctx context.Context, scope string) ([]storage.PlayerFrequency, error) {
	var r []storage.PlayerFrequency
	err := s.c.call(ctx, scope, "stats.playerNames", nil, &r)
	return r, err
}

func (s statsStore) PlayerTable(ctx context.Context, scope string, filter storage.StatsFilter) ([]storage.PlayerRow, error) {
	var r []storage.PlayerRow
	err := s.c.call(ctx, scope, "stats.playerTable", statsFilterReq{filter}, &r)
	return r, err
}

func (s statsStore) MatchDetail(ctx context.Context, scope string, matchID int64) (*storage.MatchDetailStats, error) {
	var r *storage.MatchDetailStats
	if err := s.c.call(ctx, scope, "stats.matchDetail", matchIDReq{matchID}, &r); err != nil {
		return nil, err
	}
	return r, nil
}

func (s statsStore) MatchMoveGrades(ctx context.Context, scope string, matchID int64) ([]storage.MoveGrade, error) {
	var r []storage.MoveGrade
	err := s.c.call(ctx, scope, "stats.matchMoveGrades", matchIDReq{matchID}, &r)
	return r, err
}

func (s statsStore) MatchDecisionLosses(ctx context.Context, scope string, matchID int64) ([]storage.DecisionLoss, error) {
	var r []storage.DecisionLoss
	err := s.c.call(ctx, scope, "stats.matchDecisionLosses", matchIDReq{matchID}, &r)
	return r, err
}

func (s statsStore) MatchReview(ctx context.Context, scope string, matchID int64) (storage.MatchReview, error) {
	var r storage.MatchReview
	err := s.c.call(ctx, scope, "stats.matchReview", matchIDReq{matchID}, &r)
	return r, err
}

func (s statsStore) TournamentReview(ctx context.Context, scope string, tournamentID int64, player string) (storage.TournamentReview, error) {
	var r storage.TournamentReview
	err := s.c.call(ctx, scope, "stats.tournamentReview", struct {
		TournamentID int64  `json:"tournamentId"`
		Player       string `json:"player"`
	}{tournamentID, player}, &r)
	return r, err
}

func (s statsStore) MatchTimeSummary(ctx context.Context, scope string, matchID int64) (storage.MatchTimeSummary, error) {
	var r storage.MatchTimeSummary
	err := s.c.call(ctx, scope, "stats.matchTimeSummary", matchIDReq{matchID}, &r)
	return r, err
}

func (s statsStore) TimeErrors(ctx context.Context, scope string) ([]storage.TimeErrorRow, error) {
	var r []storage.TimeErrorRow
	err := s.c.call(ctx, scope, "stats.timeErrors", nil, &r)
	return r, err
}

func (s statsStore) MatchBadges(ctx context.Context, scope string, matchIDs []int64) (map[int64]storage.MatchBadge, error) {
	var r struct {
		Badges map[int64]storage.MatchBadge `json:"badges"`
	}
	err := s.c.call(ctx, scope, "stats.matchBadges", struct {
		MatchIDs []int64 `json:"matchIds"`
	}{matchIDs}, &r)
	return r.Badges, err
}

func (s statsStore) TournamentBadges(ctx context.Context, scope string) (map[int64]storage.TournamentBadge, error) {
	var r struct {
		Badges map[int64]storage.TournamentBadge `json:"badges"`
	}
	err := s.c.call(ctx, scope, "stats.tournamentBadges", nil, &r)
	return r.Badges, err
}

func (s statsStore) RecurringErrors(ctx context.Context, scope string, filter storage.StatsFilter) (*storage.RecurringErrors, error) {
	var r *storage.RecurringErrors
	if err := s.c.call(ctx, scope, "stats.recurringErrors", statsFilterReq{filter}, &r); err != nil {
		return nil, err
	}
	return r, nil
}

func (s statsStore) StudyPlan(ctx context.Context, scope string, filter storage.StatsFilter) (*storage.StudyPlan, error) {
	var r *storage.StudyPlan
	if err := s.c.call(ctx, scope, "stats.studyPlan", statsFilterReq{filter}, &r); err != nil {
		return nil, err
	}
	return r, nil
}

func (s statsStore) SuggestReferences(ctx context.Context, scope string, req storage.ReferenceRequest) (*storage.ReferenceSuggestions, error) {
	var r *storage.ReferenceSuggestions
	if err := s.c.call(ctx, scope, "collections.suggest", req, &r); err != nil {
		return nil, err
	}
	return r, nil
}

func (s statsStore) StudyEffect(ctx context.Context, scope string, filter storage.StatsFilter) (*storage.StudyEffect, error) {
	var r *storage.StudyEffect
	if err := s.c.call(ctx, scope, "stats.studyEffect", statsFilterReq{filter}, &r); err != nil {
		return nil, err
	}
	return r, nil
}

func (s statsStore) DirectionalBiases(ctx context.Context, scope string, filter storage.StatsFilter) (*storage.DirectionalBiases, error) {
	var r *storage.DirectionalBiases
	if err := s.c.call(ctx, scope, "stats.biases", statsFilterReq{filter}, &r); err != nil {
		return nil, err
	}
	return r, nil
}

type headToHeadReq struct {
	PlayerA string              `json:"player_a"`
	PlayerB string              `json:"player_b"`
	Filter  storage.StatsFilter `json:"filter"`
}

func (s statsStore) HeadToHead(ctx context.Context, scope, playerA, playerB string, filter storage.StatsFilter) (*storage.HeadToHead, error) {
	var r *storage.HeadToHead
	if err := s.c.call(ctx, scope, "stats.headToHead", headToHeadReq{playerA, playerB, filter}, &r); err != nil {
		return nil, err
	}
	return r, nil
}

func (s statsStore) PlayerContrast(ctx context.Context, scope, playerA, playerB string, filter storage.StatsFilter) (*storage.PlayerContrast, error) {
	var r *storage.PlayerContrast
	if err := s.c.call(ctx, scope, "stats.playerContrast", headToHeadReq{playerA, playerB, filter}, &r); err != nil {
		return nil, err
	}
	return r, nil
}

func (s statsStore) PRByWindow(ctx context.Context, scope string, filter storage.StatsFilter, months int) ([]storage.WindowStats, error) {
	var r []storage.WindowStats
	err := s.c.call(ctx, scope, "stats.prByWindow", struct {
		Filter storage.StatsFilter `json:"filter"`
		Months int                 `json:"months"`
	}{filter, months}, &r)
	return r, err
}

func (s statsStore) MatchIDs(ctx context.Context, scope string, filter storage.StatsFilter) ([]int64, error) {
	var r struct {
		MatchIDs []int64 `json:"match_ids"`
	}
	err := s.c.call(ctx, scope, "stats.matchIds", statsFilterReq{filter}, &r)
	return r.MatchIDs, err
}

func (s statsStore) TournamentIDs(ctx context.Context, scope string, filter storage.StatsFilter) ([]int64, error) {
	var r struct {
		TournamentIDs []int64 `json:"tournament_ids"`
	}
	err := s.c.call(ctx, scope, "stats.tournamentIds", statsFilterReq{filter}, &r)
	return r.TournamentIDs, err
}

// RebuildMatchStats runs stats.rebuildMatchStats, which reports no progress
// on the way: progress is called once, when it ends.
func (s statsStore) RebuildMatchStats(ctx context.Context, scope string, progress func(done, total int)) (int, error) {
	var r struct {
		Matches int `json:"matches"`
	}
	if err := s.c.call(ctx, scope, "stats.rebuildMatchStats", nil, &r); err != nil {
		return 0, err
	}
	if progress != nil {
		progress(r.Matches, r.Matches)
	}
	return r.Matches, nil
}
