package sqlshared

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// RecurringErrors classifies every error of the filter and groups them
// (storage.GroupRecurringErrors).
//
// The theme is computed on the fly rather than stored: it is derived, never
// edited, and a stored copy would need a schema step for every refinement of
// the rules. A checker error costs one analysis decode and two legal-move
// generations; each blob is decoded once per position however many plays
// reach it.
//
// The cost charged is the stats' own error column (statsErrExpr) over the
// stats' own counted decisions, so a group's PRCost and the filter's PR are
// one formula on one scale (ADR-0019).
func (s *StatsStore) RecurringErrors(ctx context.Context, scope string, filter storage.StatsFilter) (*storage.RecurringErrors, error) {
	settings, err := librarySettings(ctx, s.DB, scope)
	if err != nil {
		return nil, errf(s.DB, "RecurringErrors settings", err)
	}
	whereSQL, baseArgs := s.buildStatsWhereClause(scope, filter)

	var numDecisions int
	if err := s.DB.QueryRow(ctx, `SELECT COUNT(*) `+statsBaseJoin+whereSQL, baseArgs...).Scan(&numDecisions); err != nil {
		return nil, errf(s.DB, "RecurringErrors count", err)
	}

	rows, err := s.DB.Query(ctx,
		`SELECT p.id, p.state, COALESCE(p.player_on_roll, 0), COALESCE(p.dice_1, 0), COALESCE(p.dice_2, 0),
			COALESCE(p.game_type, 0), p.decision_type, (`+statsErrExpr+`),
			COALESCE(mv.checker_move, ''), COALESCE(mv.cube_action, ''), COALESCE(a.best_cube_action, ''),
			CASE WHEN p.decision_type = 0 THEN a.data END `+
			statsBaseJoin+whereSQL+` AND (`+statsErrExpr+`) >= ?`,
		append(append([]any{}, baseArgs...), settings.ErrorThresholdMP)...)
	if err != nil {
		return nil, errf(s.DB, "RecurringErrors query", err)
	}
	defer rows.Close()

	decoded := make(map[int64]*domain.PositionAnalysis)
	var classified []storage.RecurringErrorRow
	for rows.Next() {
		var id, errMP int64
		var state, checkerMove, cubeAction, bestCube string
		var onRoll, dice1, dice2, gameType, decisionType int
		var data []byte
		if err := rows.Scan(&id, &state, &onRoll, &dice1, &dice2, &gameType, &decisionType, &errMP,
			&checkerMove, &cubeAction, &bestCube, &data); err != nil {
			return nil, errf(s.DB, "RecurringErrors scan", err)
		}
		row := storage.RecurringErrorRow{
			PositionID: id,
			GameType:   domain.GameType(gameType).String(),
			ErrorMP:    errMP,
			Theme:      storage.RecurringThemeNone,
		}
		if decisionType == 1 {
			row.Kind = "cube"
			row.Theme = cubeTheme(bestCube, cubeAction)
		} else {
			row.Kind = "checker"
			ana, seen := decoded[id]
			if !seen {
				if a, err := engine.DecodeAnalysisFromStorage(data); err == nil {
					ana = &a
				}
				decoded[id] = ana // nil too: an undecodable blob is not retried
			}
			if pos, ok := positionOfState(state); ok && ana != nil {
				pos.ID = id
				pos.PlayerOnRoll = onRoll
				pos.Dice = [2]int{dice1, dice2}
				if theme := engine.ExplainChecker(&pos, ana, checkerMove).Theme; theme != "" {
					row.Theme = theme
				}
			}
		}
		classified = append(classified, row)
	}
	if err := rows.Err(); err != nil {
		return nil, errf(s.DB, "RecurringErrors rows", err)
	}
	return storage.GroupRecurringErrors(classified, numDecisions, settings.ErrorThresholdMP), nil
}

// cubeTheme is the direction of a cube error: one of the four error cells of
// the cube matrix. A "right" cell with a cost above the threshold means the
// two labels disagree with the stored error; it is not named.
func cubeTheme(best, played string) string {
	switch cell := storage.ClassifyCubeDirection(best, played); cell {
	case storage.CubeCellOfferMissed, storage.CubeCellOfferPremature,
		storage.CubeCellAnswerWrongPass, storage.CubeCellAnswerWrongTake:
		return cell
	}
	return storage.RecurringThemeNone
}
