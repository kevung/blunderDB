package sqlshared

import (
	"context"
	"database/sql"
	"sort"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// TimeErrors implements storage.StatsStore.TimeErrors. The time of a decision
// is its checker play and the cube decision before it together; the error is
// scored from the Position's current analysis, as the search's E filter does,
// the blunder line the library's.
func (s *StatsStore) TimeErrors(ctx context.Context, scope string, matchIDs []int64) ([]storage.TimeErrorRow, error) {
	settings, err := librarySettings(ctx, s.DB, scope)
	if err != nil {
		return nil, errf(s.DB, "TimeErrors settings", err)
	}
	type key struct {
		player string
		bucket int
	}
	type acc struct {
		decisions, scored, blunders int
		errSum                      int64
		positions                   map[int64]bool
	}
	cells := map[key]*acc{}
	scorer := PlayScorer{}
	err = s.walkTimedDecisions(ctx, scope, matchIDs, func(name string, bucket int, mv *domain.Move, data []byte) {
		a := cells[key{name, bucket}]
		if a == nil {
			a = &acc{positions: map[int64]bool{}}
			cells[key{name, bucket}] = a
		}
		a.decisions++
		if mv.PositionID != 0 {
			a.positions[mv.PositionID] = true
		}
		scorer.Score(mv, data)
		if e := mv.ErrorMP; e != nil {
			a.scored++
			a.errSum += int64(*e)
			if int(*e) >= settings.BlunderThresholdMP {
				a.blunders++
			}
		}
	})
	if err != nil {
		return nil, err
	}
	out := make([]storage.TimeErrorRow, 0, len(cells))
	for k, a := range cells {
		row := storage.TimeErrorRow{Player: k.player, Bucket: k.bucket, Decisions: a.decisions, Scored: a.scored, Blunders: a.blunders, Positions: len(a.positions)}
		if a.scored > 0 {
			row.MeanErrorMP = float64(a.errSum) / float64(a.scored)
		}
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Player != out[j].Player {
			return out[i].Player < out[j].Player
		}
		return out[i].Bucket < out[j].Bucket
	})
	return out, nil
}

// walkTimedDecisions hands each timed decision on with its player's name, its
// duration band and its analysis blob. It is the one reading of the timed
// decisions, shared by TimeErrors and the "time_band" selection, so a list
// holds exactly what its count counts. A non-empty matchIDs keeps the
// decisions of those Matches alone.
func (s *StatsStore) walkTimedDecisions(ctx context.Context, scope string, matchIDs []int64, visit func(name string, bucket int, mv *domain.Move, data []byte)) error {
	tenant, args := s.DB.TenantFilter("mv", scope)
	// Scored by the caller rather than read from move.error_mp: no analysis
	// path writes that column, and only timed (Duel) plays are decoded, so the
	// cost stays with the few rows the table reads and nothing is added to the
	// import.
	query := `SELECT COALESCE(m.player1_name, ''), COALESCE(m.player2_name, ''), mv.player,
			mv.decision_ms, mv.cube_decision_ms, ` + scoredMoveCols(s.DB) + `
		 FROM move mv JOIN game g ON g.id = mv.game_id JOIN match m ON m.id = g.match_id
		 LEFT JOIN analysis a ON a.position_id = mv.position_id
		 WHERE ` + tenant + ` AND (mv.decision_ms IS NOT NULL OR mv.cube_decision_ms IS NOT NULL)`
	if len(matchIDs) > 0 {
		query += ` AND m.id IN (` + strings.TrimSuffix(strings.Repeat("?,", len(matchIDs)), ",") + `)`
		for _, id := range matchIDs {
			args = append(args, id)
		}
	}
	rows, err := s.DB.Query(ctx, query, args...)
	if err != nil {
		return errf(s.DB, "TimeErrors query", err)
	}
	defer rows.Close()
	for rows.Next() {
		var p1, p2 string
		var player int
		var d, c sql.NullInt64
		var mv domain.Move
		var data []byte
		if err := rows.Scan(&p1, &p2, &player, &d, &c,
			&mv.ID, &mv.PositionID, &mv.MoveType, &mv.CheckerMove, &mv.CubeAction, &data); err != nil {
			return errf(s.DB, "TimeErrors scan", err)
		}
		name := p1
		switch player {
		case 1:
		case -1:
			name = p2
		default:
			continue
		}
		visit(name, timeBucketOf(d.Int64+c.Int64), &mv, data)
	}
	if err := rows.Err(); err != nil {
		return errf(s.DB, "TimeErrors rows", err)
	}
	return nil
}

// timeBucketOf is the duration band of a decision that took ms milliseconds:
// the index of the first storage.TimeBucketBounds it stays under, or the band
// past them all.
func timeBucketOf(ms int64) int {
	for i, bound := range storage.TimeBucketBounds {
		if ms < bound {
			return i
		}
	}
	return len(storage.TimeBucketBounds)
}

// timeBandIDs lists the distinct positions of player's timed decisions in one
// duration band, in the order TimeErrors counts them as Positions.
func (s *StatsStore) timeBandIDs(ctx context.Context, scope, player string, bucket int, matchIDs []int64) ([]int64, error) {
	seen := map[int64]bool{}
	ids := []int64{}
	err := s.walkTimedDecisions(ctx, scope, matchIDs, func(name string, b int, mv *domain.Move, _ []byte) {
		if name != player || b != bucket || mv.PositionID == 0 || seen[mv.PositionID] {
			return
		}
		seen[mv.PositionID] = true
		ids = append(ids, mv.PositionID)
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids, nil
}
