package sqlshared

import (
	"context"
	"database/sql"
	"errors"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// MatchTimeSummary adds up the decision times of a Match per player. A Move
// with a NULL duration is counted as unknown and enters no total. Moves carry
// the XG side code: 1 is player 1, -1 player 2. The overrun is the fact the
// Duel recorded in the Match's origin, never recomputed here.
func (s *StatsStore) MatchTimeSummary(ctx context.Context, scope string, matchID int64) (storage.MatchTimeSummary, error) {
	var out storage.MatchTimeSummary
	tenant, args := s.DB.TenantFilter("mv", scope)
	rows, err := s.DB.Query(ctx,
		`SELECT mv.player, `+ActionLabelOrEmptyFor(s.DB, "mv.move_type")+`,
			mv.decision_ms, mv.cube_decision_ms
		 FROM move mv JOIN game g ON g.id = mv.game_id
		 WHERE `+tenant+` AND g.match_id = ?`,
		append(args, matchID)...)
	if err != nil {
		return out, errf(s.DB, "MatchTimeSummary query", err)
	}
	defer rows.Close()
	for rows.Next() {
		var player int
		var moveType string
		var d, c sql.NullInt64
		if err := rows.Scan(&player, &moveType, &d, &c); err != nil {
			return out, errf(s.DB, "MatchTimeSummary scan", err)
		}
		var ps *storage.PlayerTimeSummary
		switch player {
		case 1:
			ps = &out.Players[0]
		case -1:
			ps = &out.Players[1]
		default:
			continue
		}
		known := false
		add := func(v *int64, cube bool) {
			if v == nil {
				return
			}
			known = true
			ps.TotalMS += *v
			if cube {
				ps.CubeCount++
				ps.CubeTotalMS += *v
			} else {
				ps.CheckerCount++
				ps.CheckerTotalMS += *v
			}
		}
		if moveType == "cube" {
			add(NullableMS(d), true)
		} else {
			add(NullableMS(d), false)
			add(NullableMS(c), true)
		}
		if !known {
			ps.Unknown++
		}
	}
	if err := rows.Err(); err != nil {
		return out, errf(s.DB, "MatchTimeSummary rows", err)
	}

	otenant, oargs := s.DB.TenantFilter("mo", scope)
	var cadence string
	var overTime int
	err = s.DB.QueryRow(ctx,
		`SELECT mo.cadence, mo.over_time FROM match_origin mo WHERE `+otenant+` AND mo.match_id = ?`,
		append(oargs, matchID)...).Scan(&cadence, &overTime)
	if err != nil && !errors.Is(err, ErrNoRows) {
		return out, errf(s.DB, "MatchTimeSummary origin", err)
	}
	if err == nil && cadence != "" {
		out.HasCadence = true
		if overTime == 1 || overTime == 2 {
			out.Players[overTime-1].OverTime = true
		}
	}
	return out, nil
}
