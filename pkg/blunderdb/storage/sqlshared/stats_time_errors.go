package sqlshared

import (
	"context"
	"database/sql"
	"sort"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// TimeErrors implements storage.StatsStore.TimeErrors. The time of a decision
// is its checker play and the cube decision before it together; the error is
// scored from the Position's current analysis, as the search's E filter does,
// the blunder line the library's.
func (s *StatsStore) TimeErrors(ctx context.Context, scope string) ([]storage.TimeErrorRow, error) {
	settings, err := librarySettings(ctx, s.DB, scope)
	if err != nil {
		return nil, errf(s.DB, "TimeErrors settings", err)
	}
	tenant, args := s.DB.TenantFilter("mv", scope)
	// Scored here rather than read from move.error_mp: no analysis path writes
	// that column, and only timed (Duel) plays are decoded, so the cost stays
	// with the few rows the table reads and nothing is added to the import.
	rows, err := s.DB.Query(ctx,
		`SELECT COALESCE(m.player1_name, ''), COALESCE(m.player2_name, ''), mv.player,
			mv.decision_ms, mv.cube_decision_ms, `+scoredMoveCols(s.DB)+`
		 FROM move mv JOIN game g ON g.id = mv.game_id JOIN match m ON m.id = g.match_id
		 LEFT JOIN analysis a ON a.position_id = mv.position_id
		 WHERE `+tenant+` AND (mv.decision_ms IS NOT NULL OR mv.cube_decision_ms IS NOT NULL)`,
		args...)
	if err != nil {
		return nil, errf(s.DB, "TimeErrors query", err)
	}
	defer rows.Close()
	type key struct {
		player string
		bucket int
	}
	type acc struct {
		decisions, scored, blunders int
		errSum                      int64
	}
	cells := map[key]*acc{}
	scorer := PlayScorer{}
	for rows.Next() {
		var p1, p2 string
		var player int
		var d, c sql.NullInt64
		var mv domain.Move
		var data []byte
		if err := rows.Scan(&p1, &p2, &player, &d, &c,
			&mv.ID, &mv.PositionID, &mv.MoveType, &mv.CheckerMove, &mv.CubeAction, &data); err != nil {
			return nil, errf(s.DB, "TimeErrors scan", err)
		}
		name := p1
		switch player {
		case 1:
		case -1:
			name = p2
		default:
			continue
		}
		ms := d.Int64 + c.Int64
		bucket := len(storage.TimeBucketBounds)
		for i, bound := range storage.TimeBucketBounds {
			if ms < bound {
				bucket = i
				break
			}
		}
		a := cells[key{name, bucket}]
		if a == nil {
			a = &acc{}
			cells[key{name, bucket}] = a
		}
		a.decisions++
		scorer.Score(&mv, data)
		if e := mv.ErrorMP; e != nil {
			a.scored++
			a.errSum += int64(*e)
			if int(*e) >= settings.BlunderThresholdMP {
				a.blunders++
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, errf(s.DB, "TimeErrors rows", err)
	}
	out := make([]storage.TimeErrorRow, 0, len(cells))
	for k, a := range cells {
		row := storage.TimeErrorRow{Player: k.player, Bucket: k.bucket, Decisions: a.decisions, Scored: a.scored, Blunders: a.blunders}
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
