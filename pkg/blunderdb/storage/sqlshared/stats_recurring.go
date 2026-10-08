package sqlshared

import (
	"context"
	"fmt"
	"math"

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
	rows, numDecisions, thresholdMP, err := s.classifiedErrors(ctx, scope, filter, false)
	if err != nil {
		return nil, err
	}
	plain := make([]storage.RecurringErrorRow, len(rows))
	for i, r := range rows {
		plain[i] = r.RecurringErrorRow
	}
	return storage.GroupRecurringErrors(plain, numDecisions, thresholdMP), nil
}

// StudyPlan prices the same classified errors in MWC and difficulty, the way
// MatchDecisionLosses prices a match's decisions, and ranks their families
// (storage.BuildStudyPlan, ADR-0077).
func (s *StatsStore) StudyPlan(ctx context.Context, scope string, filter storage.StatsFilter) (*storage.StudyPlan, error) {
	rows, numDecisions, thresholdMP, err := s.classifiedErrors(ctx, scope, filter, true)
	if err != nil {
		return nil, err
	}
	return storage.BuildStudyPlan(rows, numDecisions, thresholdMP), nil
}

// classifiedErrors lists every error of the filter with its theme and, when
// priced, its MWC loss and difficulty. It returns the filter's counted
// decisions and the library's Error threshold alongside.
func (s *StatsStore) classifiedErrors(ctx context.Context, scope string, filter storage.StatsFilter, priced bool) ([]storage.StudyPlanRow, int, int, error) {
	filter, err := s.withPlayerAliases(ctx, scope, filter)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("classifiedErrors aliases: %w", err)
	}
	settings, err := librarySettings(ctx, s.DB, scope)
	if err != nil {
		return nil, 0, 0, errf(s.DB, "classifiedErrors settings", err)
	}
	whereSQL, baseArgs := s.buildStatsWhereClause(scope, filter)

	var numDecisions int
	if err := s.DB.QueryRow(ctx, `SELECT COUNT(*) `+statsBaseJoin+whereSQL, baseArgs...).Scan(&numDecisions); err != nil {
		return nil, 0, 0, errf(s.DB, "classifiedErrors count", err)
	}

	// A cube blob is only decoded to price a cube error: the recurring view
	// never reads it.
	dataExpr := "CASE WHEN p.decision_type = 0 THEN a.data END"
	if priced {
		dataExpr = "a.data"
	}
	rows, err := s.DB.Query(ctx,
		`SELECT p.id, p.state, COALESCE(p.player_on_roll, 0), COALESCE(p.dice_1, 0), COALESCE(p.dice_2, 0),
			COALESCE(p.game_type, 0), p.decision_type, (`+statsErrExpr+`),
			COALESCE(mv.checker_move, ''), `+ActionLabelOrEmptyFor(s.DB, "mv.cube_action")+`, `+ActionLabelOrEmptyFor(s.DB, "a.best_cube_action")+`,
			`+dataExpr+`, m.id, COALESCE(m.player1_name, ''), COALESCE(m.player2_name, ''),
			COALESCE(mv.player, 0), COALESCE(p.score_1, 0), COALESCE(p.score_2, 0), `+cubeMultiplierExpr+`,
			COALESCE(p.match_length, m.match_length, 0), COALESCE(m.match_length, 0), `+s.DB.DateText("m.match_date")+` `+
			statsBaseJoin+whereSQL+` AND (`+statsErrExpr+`) >= ?`,
		append(append([]any{}, baseArgs...), settings.ErrorThresholdMP)...)
	if err != nil {
		return nil, 0, 0, errf(s.DB, "classifiedErrors query", err)
	}
	defer rows.Close()

	decoded := make(map[int64]*domain.PositionAnalysis)
	analysisOf := func(id int64, data []byte) *domain.PositionAnalysis {
		ana, seen := decoded[id]
		if !seen {
			if a, err := engine.DecodeAnalysisFromStorage(data); err == nil {
				ana = &a
			}
			decoded[id] = ana // nil too: an undecodable blob is not retried
		}
		return ana
	}
	var classified []storage.StudyPlanRow
	for rows.Next() {
		var id, errMP, matchID int64
		var checkerMove, cubeAction, bestCube, p1, p2 string
		var state []byte
		var onRoll, dice1, dice2, gameType, decisionType int
		var rawPlayer, away0, away1, cubeValue, matchLength, labelLength int
		var data []byte
		var day string
		if err := rows.Scan(&id, &state, &onRoll, &dice1, &dice2, &gameType, &decisionType, &errMP,
			&checkerMove, &cubeAction, &bestCube, &data, &matchID, &p1, &p2,
			&rawPlayer, &away0, &away1, &cubeValue, &matchLength, &labelLength, &day); err != nil {
			return nil, 0, 0, errf(s.DB, "classifiedErrors scan", err)
		}
		row := storage.StudyPlanRow{RecurringErrorRow: storage.RecurringErrorRow{
			PositionID: id,
			GameType:   domain.GameType(gameType).String(),
			ErrorMP:    errMP,
			Theme:      storage.RecurringThemeNone,
		}, MatchID: matchID}
		if decisionType == 1 {
			row.Kind = "cube"
			row.Theme = cubeTheme(bestCube, cubeAction)
		} else {
			row.Kind = "checker"
			ana := analysisOf(id, data)
			if pos, ok := positionOfState(string(state)); ok && ana != nil {
				pos.ID = id
				pos.PlayerOnRoll = onRoll
				pos.Dice = [2]int{dice1, dice2}
				if theme := engine.ExplainChecker(&pos, ana, checkerMove).Theme; theme != "" {
					row.Theme = theme
				}
			}
		}
		if priced {
			row.Label = matchLabel(p1, p2, labelLength)
			row.Day = day
			if loss := decisionMWCLoss(errMP, away0, away1, rawPlayer, cubeValue, matchLength); !math.IsNaN(loss) {
				row.Loss = &loss
				if costs := decisionCosts(analysisOf(id, data), row.Kind, cubeAction); costs != nil {
					// As MatchDecisionLosses: the conversion is linear in equity.
					perUnit := decisionMWCLoss(1000, away0, away1, rawPlayer, cubeValue, matchLength)
					diff := perUnit * storage.ReferenceExpectedLoss(costs, storage.DifficultyTemperature)
					row.Difficulty = &diff
					row.Avoidable = storage.IsAvoidable(loss, diff, true)
				}
			}
		}
		classified = append(classified, row)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, 0, errf(s.DB, "classifiedErrors rows", err)
	}
	return classified, numDecisions, settings.ErrorThresholdMP, nil
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
