package sqlshared

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// cadenceClock is the part of a Duel's Cadence (duel.Cadence, as match_origin
// records it in JSON) that decides an overrun: the reserve and the delay. The
// storage layer cannot import the duel package, so the reading is mirrored
// here: a reserve is Reserve seconds, or ReservePerPoint seconds times half
// the points still to play at the score the Match started from, and a turn of
// the clock draws on it only for what exceeds Delay.
type cadenceClock struct {
	Reserve         int `json:"reserve"`
	ReservePerPoint int `json:"reservePerPoint"`
	Delay           int `json:"delay"`
}

func (c cadenceClock) reserveMS(matchLength int, score [2]int) int64 {
	if c.ReservePerPoint == 0 {
		return int64(c.Reserve) * 1000
	}
	left := int64(2*matchLength - score[0] - score[1])
	return int64(c.ReservePerPoint) * 1000 * max(left, 0) / 2
}

// MatchTimeSummary adds up the decision times of a Match per player. A Move
// with a NULL duration is counted as unknown and enters no total. The clock's
// turn is rebuilt as the Arbiter keeps it: a double is the start of a turn
// that ends with the player's next play, a take or a pass is a turn of its
// own, and a checker Move carries the cube decision taken before its roll.
func (s *StatsStore) MatchTimeSummary(ctx context.Context, scope string, matchID int64) (storage.MatchTimeSummary, error) {
	var out storage.MatchTimeSummary
	tenant, args := s.DB.TenantFilter("mv", scope)
	rows, err := s.DB.Query(ctx,
		`SELECT mv.player, `+ActionLabelOrEmptyFor(s.DB, "mv.move_type")+`, `+ActionLabelOrEmptyFor(s.DB, "mv.cube_action")+`,
			mv.decision_ms, mv.cube_decision_ms
		 FROM move mv JOIN game g ON g.id = mv.game_id
		 WHERE `+tenant+` AND g.match_id = ?
		 ORDER BY g.game_number, mv.move_number, mv.id`,
		append(args, matchID)...)
	if err != nil {
		return out, errf(s.DB, "MatchTimeSummary query", err)
	}
	type play struct {
		player           int
		moveType, action string
		decision, cube   *int64
	}
	var plays []play
	for rows.Next() {
		var p play
		var d, c sql.NullInt64
		if err := rows.Scan(&p.player, &p.moveType, &p.action, &d, &c); err != nil {
			rows.Close()
			return out, errf(s.DB, "MatchTimeSummary scan", err)
		}
		p.decision, p.cube = NullableMS(d), NullableMS(c)
		plays = append(plays, p)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, errf(s.DB, "MatchTimeSummary rows", err)
	}

	clock, reserve, err := s.matchClock(ctx, scope, matchID)
	if err != nil {
		return out, err
	}
	out.HasCadence = clock != nil
	var left [2]int64
	left[0], left[1] = reserve, reserve
	var turn [2]int64 // what the running turn of each player has used
	for _, p := range plays {
		if p.player != 1 && p.player != 2 {
			continue
		}
		i := p.player - 1
		ps := &out.Players[i]
		var ms int64
		known := false
		add := func(v *int64, cube bool) {
			if v == nil {
				return
			}
			known = true
			ms += *v
			ps.TotalMS += *v
			if cube {
				ps.CubeCount++
				ps.CubeTotalMS += *v
			} else {
				ps.CheckerCount++
				ps.CheckerTotalMS += *v
			}
		}
		isCube := p.moveType == "cube"
		if isCube {
			add(p.decision, true)
		} else {
			add(p.decision, false)
			add(p.cube, true)
		}
		if !known {
			ps.Unknown++
		}
		turn[i] += ms
		if isCube && !engine.IsResponseCubeAction(p.action) {
			continue // a double: the turn goes on with the play after the roll
		}
		if clock != nil {
			charge := max(turn[i]-int64(clock.Delay)*1000, 0)
			if charge > left[i] {
				ps.OverrunTurns++
				ps.OverrunMS += charge - left[i]
				left[i] = 0
			} else {
				left[i] -= charge
			}
		}
		turn[i] = 0
	}
	return out, nil
}

// matchClock reads the Cadence a Match was played under and the reserve it
// gave each player; nil when the Match was not played here, or under none.
func (s *StatsStore) matchClock(ctx context.Context, scope string, matchID int64) (*cadenceClock, int64, error) {
	tenant, args := s.DB.TenantFilter("mo", scope)
	var raw string
	var length sql.NullInt64
	err := s.DB.QueryRow(ctx,
		`SELECT mo.cadence, m.match_length FROM match_origin mo JOIN match m ON m.id = mo.match_id
		 WHERE `+tenant+` AND mo.match_id = ?`, append(args, matchID)...).Scan(&raw, &length)
	if errors.Is(err, ErrNoRows) || (err == nil && raw == "") {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, errf(s.DB, "MatchTimeSummary origin", err)
	}
	var c cadenceClock
	if json.Unmarshal([]byte(raw), &c) != nil || (c.Reserve == 0 && c.ReservePerPoint == 0) {
		return nil, 0, nil
	}
	var score [2]int
	gtenant, gargs := s.DB.TenantFilter("g", scope)
	var s1, s2 sql.NullInt64
	err = s.DB.QueryRow(ctx,
		`SELECT g.initial_score_1, g.initial_score_2 FROM game g
		 WHERE `+gtenant+` AND g.match_id = ? ORDER BY g.game_number LIMIT 1`, append(gargs, matchID)...).Scan(&s1, &s2)
	if err != nil && !errors.Is(err, ErrNoRows) {
		return nil, 0, errf(s.DB, "MatchTimeSummary start score", err)
	}
	score = [2]int{int(s1.Int64), int(s2.Int64)}
	return &c, c.reserveMS(int(length.Int64), score), nil
}
