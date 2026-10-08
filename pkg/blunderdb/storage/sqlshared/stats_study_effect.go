package sqlshared

import (
	"context"
	"fmt"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// StudyEffect measures each studied family of the study plan before and after
// the day it was first studied (storage.BuildStudyEffect, ADR-0078).
func (s *StatsStore) StudyEffect(ctx context.Context, scope string, filter storage.StatsFilter) (*storage.StudyEffect, error) {
	rows, _, _, err := s.classifiedErrors(ctx, scope, filter, true)
	if err != nil {
		return nil, err
	}
	filter, err = s.withPlayerAliases(ctx, scope, filter)
	if err != nil {
		return nil, fmt.Errorf("StudyEffect aliases: %w", err)
	}
	whereSQL, args := s.buildStatsWhereClause(scope, filter)
	q, err := s.DB.Query(ctx,
		`SELECT COALESCE(p.game_type, 0), p.decision_type, `+s.DB.DateText("m.match_date")+`, COUNT(*) `+
			statsBaseJoin+whereSQL+` AND COALESCE(p.match_length, m.match_length, 0) > 0
			GROUP BY p.game_type, p.decision_type, m.match_date`, args...)
	if err != nil {
		return nil, errf(s.DB, "StudyEffect decisions", err)
	}
	var decisions []storage.StudyEffectDecisions
	err = func() error {
		defer q.Close()
		for q.Next() {
			var gameType, decisionType int
			var d storage.StudyEffectDecisions
			if err := q.Scan(&gameType, &decisionType, &d.Day, &d.Count); err != nil {
				return err
			}
			d.GameType = domain.GameType(gameType).String()
			d.Kind = "checker"
			if decisionType == 1 {
				d.Kind = "cube"
			}
			decisions = append(decisions, d)
		}
		return q.Err()
	}()
	if err != nil {
		return nil, errf(s.DB, "StudyEffect decisions", err)
	}
	studied, err := s.firstStudyDays(ctx, scope)
	if err != nil {
		return nil, err
	}
	return storage.BuildStudyEffect(rows, decisions, studied), nil
}

// firstStudyDays is the UTC day of each position's first study action: a
// study mark, an Anki review, a quiz answer (ADR-0078 rule 2).
func (s *StatsStore) firstStudyDays(ctx context.Context, scope string) (map[int64]string, error) {
	out := map[int64]string{}
	keep := func(id int64, t time.Time) {
		if t.IsZero() {
			return
		}
		day := t.UTC().Format("2006-01-02")
		if cur, ok := out[id]; !ok || day < cur {
			out[id] = day
		}
	}
	markTenant, markArgs := s.DB.TenantFilter("", scope)
	if err := s.scanStudyTimes(ctx, `SELECT position_id, marked_at FROM study_mark WHERE `+markTenant, markArgs,
		func(id int64, raw any) {
			if v, ok := raw.(int64); ok {
				keep(id, time.Unix(v, 0))
			}
		}); err != nil {
		return nil, errf(s.DB, "first study days (marks)", err)
	}
	asText := func(id int64, raw any) {
		if v, ok := raw.(string); ok {
			keep(id, parseStudyTime(v))
		}
	}
	reviewTenant, reviewArgs := s.DB.TenantFilter("", scope)
	if err := s.scanStudyTimes(ctx, `SELECT position_id, `+s.DB.TimestampText("MIN(reviewed_at)")+`
		FROM anki_review_log WHERE `+reviewTenant+` AND position_id IS NOT NULL GROUP BY position_id`,
		reviewArgs, asText); err != nil {
		return nil, errf(s.DB, "first study days (reviews)", err)
	}
	itemTenant, itemArgs := s.DB.TenantFilter("i", scope)
	sessionTenant, sessionArgs := s.DB.TenantFilter("ts", scope)
	if err := s.scanStudyTimes(ctx, `SELECT i.position_id, `+s.DB.TimestampText("MIN(ts.created_at)")+`
		FROM training_item i JOIN training_session ts ON ts.id = i.session_id
		WHERE `+itemTenant+` AND `+sessionTenant+` AND i.position_id IS NOT NULL GROUP BY i.position_id`,
		append(itemArgs, sessionArgs...), asText); err != nil {
		return nil, errf(s.DB, "first study days (quiz)", err)
	}
	return out, nil
}

// scanStudyTimes runs a (position_id, time) query and hands each row on; the
// time column is an integer for a mark and text for the others.
func (s *StatsStore) scanStudyTimes(ctx context.Context, query string, args []any, each func(int64, any)) error {
	rows, err := s.DB.Query(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var raw any
		if err := rows.Scan(&id, &raw); err != nil {
			return err
		}
		switch v := raw.(type) {
		case []byte:
			raw = string(v)
		case time.Time:
			raw = v.UTC().Format(time.RFC3339Nano)
		case int32:
			raw = int64(v)
		case int:
			raw = int64(v)
		}
		each(id, raw)
	}
	return rows.Err()
}

// parseStudyTime reads a stored timestamp in the spellings the two backends
// produce, its date alone when nothing finer parses.
func parseStudyTime(v string) time.Time {
	if t := parseTimeText(v); !t.IsZero() {
		return t
	}
	for _, layout := range []string{"2006-01-02 15:04:05", time.DateOnly} {
		if len(v) >= len(layout) {
			if t, err := time.Parse(layout, v[:len(layout)]); err == nil {
				return t
			}
		}
	}
	return time.Time{}
}

// DirectionalBiases tallies the signed biases of the filter's decisions
// (storage.BuildDirectionalBiases, ADR-0078).
func (s *StatsStore) DirectionalBiases(ctx context.Context, scope string, filter storage.StatsFilter) (*storage.DirectionalBiases, error) {
	filter, err := s.withPlayerAliases(ctx, scope, filter)
	if err != nil {
		return nil, fmt.Errorf("DirectionalBiases aliases: %w", err)
	}
	whereSQL, args := s.buildStatsWhereClause(scope, filter)

	cubeRows, err := s.DB.Query(ctx,
		`SELECT `+ActionLabelOrEmptyFor(s.DB, "a.best_cube_action")+`, `+ActionLabelOrEmptyFor(s.DB, "mv.cube_action")+`,
			COALESCE(p.score_1, 0), COALESCE(p.score_2, 0), COALESCE(a.cube_error, 0) `+
			statsBaseJoin+whereSQL+` AND p.decision_type = 1`, args...)
	if err != nil {
		return nil, errf(s.DB, "DirectionalBiases cube", err)
	}
	var cube []storage.BiasCubeRow
	err = func() error {
		defer cubeRows.Close()
		for cubeRows.Next() {
			var r storage.BiasCubeRow
			if err := cubeRows.Scan(&r.Best, &r.Played, &r.MoverAway, &r.OpponentAway, &r.ErrorMP); err != nil {
				return err
			}
			cube = append(cube, r)
		}
		return cubeRows.Err()
	}()
	if err != nil {
		return nil, errf(s.DB, "DirectionalBiases cube", err)
	}

	// The analysis blob is only decoded for a play that cost something: a
	// free play is a best play and reads 0 without a board.
	checkerRows, err := s.DB.Query(ctx,
		`SELECT p.state, COALESCE(p.player_on_roll, 0), COALESCE(p.dice_1, 0), COALESCE(p.dice_2, 0),
			COALESCE(a.best_move_equity_error, 0), COALESCE(mv.checker_move, ''),
			CASE WHEN a.best_move_equity_error > 0 THEN a.data END `+
			statsBaseJoin+whereSQL+` AND p.decision_type = 0`, args...)
	if err != nil {
		return nil, errf(s.DB, "DirectionalBiases checker", err)
	}
	var checker []storage.BiasCheckerRow
	err = func() error {
		defer checkerRows.Close()
		for checkerRows.Next() {
			var state, data []byte
			var onRoll, dice1, dice2 int
			var errMP int64
			var played string
			if err := checkerRows.Scan(&state, &onRoll, &dice1, &dice2, &errMP, &played, &data); err != nil {
				return err
			}
			pos, ok := positionOfState(string(state))
			if !ok || !engine.HasContact(&pos.Board) {
				continue
			}
			if errMP <= 0 {
				checker = append(checker, storage.BiasCheckerRow{})
				continue
			}
			pos.PlayerOnRoll = onRoll
			pos.Dice = [2]int{dice1, dice2}
			row := storage.BiasCheckerRow{ErrorMP: errMP, Unread: true}
			if ana, err := engine.DecodeAnalysisFromStorage(data); err == nil {
				if sign, ok := engine.BlotDeviation(&pos, &ana, played); ok {
					row.Sign, row.Unread = sign, false
				}
			}
			checker = append(checker, row)
		}
		return checkerRows.Err()
	}()
	if err != nil {
		return nil, errf(s.DB, "DirectionalBiases checker", err)
	}
	return storage.BuildDirectionalBiases(cube, checker), nil
}
