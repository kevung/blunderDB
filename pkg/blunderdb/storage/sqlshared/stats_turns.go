package sqlshared

import (
	"context"
	"database/sql"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// MatchTurns lists the Moves of a Match in match order with the durations they
// kept; a NULL duration is nil, never zero.
func (s *StatsStore) MatchTurns(ctx context.Context, scope string, matchID int64) (storage.MatchTurns, error) {
	var out storage.MatchTurns
	tenant, args := s.DB.TenantFilter("mv", scope)
	rows, err := s.DB.Query(ctx,
		`SELECT g.game_number, g.initial_score_1, g.initial_score_2, mv.player, `+ActionLabelOrEmptyFor(s.DB, "mv.move_type")+`,
			mv.decision_ms, mv.cube_decision_ms
		 FROM move mv JOIN game g ON g.id = mv.game_id
		 WHERE `+tenant+` AND g.match_id = ?
		 ORDER BY g.game_number ASC, mv.move_number ASC`,
		append(args, matchID)...)
	if err != nil {
		return out, errf(s.DB, "MatchTurns query", err)
	}
	defer rows.Close()
	first := true
	for rows.Next() {
		var game int
		var score [2]int32
		var player int
		var moveType string
		var d, c sql.NullInt64
		if err := rows.Scan(&game, &score[0], &score[1], &player, &moveType, &d, &c); err != nil {
			return out, errf(s.DB, "MatchTurns scan", err)
		}
		if first {
			out.Score, first = score, false
		}
		side := 0
		if player == -1 {
			side = 1
		}
		out.Turns = append(out.Turns, storage.MatchTurn{Player: side, Cube: moveType == "cube", CubeMS: NullableMS(c), PlayMS: NullableMS(d)})
	}
	if err := rows.Err(); err != nil {
		return out, errf(s.DB, "MatchTurns rows", err)
	}
	return out, nil
}
