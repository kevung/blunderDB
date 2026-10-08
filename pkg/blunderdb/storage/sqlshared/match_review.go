package sqlshared

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// MatchReview reads a match's decisions as MatchDecisionLosses lists them,
// and the score it started and ended at, and hands them to
// storage.BuildMatchReview (ADR-0077). The winning chances at the start come
// from the table the losses and the luck are converted with.
func (s *StatsStore) MatchReview(ctx context.Context, scope string, matchID int64) (storage.MatchReview, error) {
	decisions, err := s.MatchDecisionLosses(ctx, scope, matchID)
	if err != nil {
		return storage.MatchReview{}, err
	}
	tenant, args := s.DB.TenantFilter("m", scope)
	var length, start1, start2, end1, end2 int
	err = scanEach(ctx, s.DB,
		`SELECT COALESCE(m.match_length, 0),
		        COALESCE((SELECT g.initial_score_1 FROM game g WHERE g.match_id = m.id ORDER BY g.game_number LIMIT 1), 0),
		        COALESCE((SELECT g.initial_score_2 FROM game g WHERE g.match_id = m.id ORDER BY g.game_number LIMIT 1), 0),
		        COALESCE((SELECT MAX(g.initial_score_1 + CASE WHEN g.points_won > 0 AND g.winner = 1 THEN g.points_won ELSE 0 END) FROM game g WHERE g.match_id = m.id), 0),
		        COALESCE((SELECT MAX(g.initial_score_2 + CASE WHEN g.points_won > 0 AND g.winner = -1 THEN g.points_won ELSE 0 END) FROM game g WHERE g.match_id = m.id), 0)
		 FROM match m WHERE `+tenant+` AND m.id = ?`,
		append(args, matchID), func(r Rows) error {
			return r.Scan(&length, &start1, &start2, &end1, &end2)
		})
	if err != nil {
		return storage.MatchReview{}, errf(s.DB, "MatchReview match", err)
	}
	startMWC := 0.5
	if length > 0 && length <= 64 && (start1 != 0 || start2 != 0) && start1 < length && start2 < length {
		startMWC = engine.GnuBGGetME(start1, start2, length, 0, 0, 0, false)
	}
	outcome := storage.MatchOutcome(int32(length), int32(end1), int32(end2))
	if length <= 0 {
		outcome = 0
	}
	return storage.BuildMatchReview(matchID, decisions, length, startMWC, outcome), nil
}
