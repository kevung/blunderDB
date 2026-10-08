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
	rows, numDecisions, thresholdMP, err := s.classifiedErrors(ctx, scope, filter, classifyOptions{})
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
	rows, numDecisions, thresholdMP, err := s.classifiedErrors(ctx, scope, filter, classifyOptions{priced: true})
	if err != nil {
		return nil, err
	}
	plan := make([]storage.StudyPlanRow, len(rows))
	for i, r := range rows {
		plan[i] = r.StudyPlanRow
	}
	return storage.BuildStudyPlan(plan, numDecisions, thresholdMP), nil
}

// SuggestReferences reads the same priced errors with their boards and the
// state of their lesson, and the positions something already deals with, and
// proposes the reference positions of ADR-0080 (storage.SuggestReferences).
func (s *StatsStore) SuggestReferences(ctx context.Context, scope string, req storage.ReferenceRequest) (*storage.ReferenceSuggestions, error) {
	rows, numDecisions, thresholdMP, err := s.classifiedErrors(ctx, scope, req.Filter,
		classifyOptions{priced: true, reference: true, matchIDs: req.MatchIDs})
	if err != nil {
		return nil, err
	}
	handled, err := s.handledPositions(ctx, scope)
	if err != nil {
		return nil, err
	}
	return storage.SuggestReferences(rows, handled, numDecisions, thresholdMP, req.Size), nil
}

// handledPositions are the positions the study queue's backlog leaves out:
// unhandledSQL negated, word for word.
func (s *StatsStore) handledPositions(ctx context.Context, scope string) (map[int64]bool, error) {
	tenant, targs := s.DB.TenantFilter("p", scope)
	rows, err := s.DB.Query(ctx, `SELECT p.id FROM position p WHERE `+tenant+` AND NOT (1 = 1`+unhandledSQL+`)`, targs...)
	if err != nil {
		return nil, errf(s.DB, "handledPositions query", err)
	}
	defer rows.Close()
	handled := map[int64]bool{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, errf(s.DB, "handledPositions scan", err)
		}
		handled[id] = true
	}
	if err := rows.Err(); err != nil {
		return nil, errf(s.DB, "handledPositions rows", err)
	}
	return handled, nil
}

// classifyOptions say how much classifiedErrors reads: priced adds the MWC
// loss and the difficulty, reference the board and the lesson's state;
// matchIDs narrows the filter to these matches.
type classifyOptions struct {
	priced, reference bool
	matchIDs          []int64
}

// classifiedErrors lists every error of the filter with its theme and, when
// priced, its MWC loss and difficulty. It returns the filter's counted
// decisions and the library's Error threshold alongside.
func (s *StatsStore) classifiedErrors(ctx context.Context, scope string, filter storage.StatsFilter, opts classifyOptions) ([]storage.ReferenceRow, int, int, error) {
	priced := opts.priced
	filter, err := s.withPlayerAliases(ctx, scope, filter)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("classifiedErrors aliases: %w", err)
	}
	settings, err := librarySettings(ctx, s.DB, scope)
	if err != nil {
		return nil, 0, 0, errf(s.DB, "classifiedErrors settings", err)
	}
	whereSQL, baseArgs := s.buildStatsWhereClause(scope, filter)
	if len(opts.matchIDs) > 0 {
		whereSQL += " AND m.id IN (" + Placeholders(len(opts.matchIDs)) + ")"
		for _, id := range opts.matchIDs {
			baseArgs = append(baseArgs, id)
		}
	}

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
	var classified []storage.ReferenceRow
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
		row := storage.ReferenceRow{StudyPlanRow: storage.StudyPlanRow{RecurringErrorRow: storage.RecurringErrorRow{
			PositionID: id,
			GameType:   domain.GameType(gameType).String(),
			ErrorMP:    errMP,
			Theme:      storage.RecurringThemeNone,
		}, MatchID: matchID}}
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
					if opts.reference {
						s.readReference(&row, string(state), onRoll, away0, away1, analysisOf(id, data), cubeAction, costs)
					}
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

// readReference fills what a reference proposal reads beyond the plan: the
// board seen from the side on roll, the score of each side and the state of
// the position's lesson.
func (s *StatsStore) readReference(row *storage.ReferenceRow, state string, onRoll, away0, away1 int,
	ana *domain.PositionAnalysis, cubeAction string, costs []float64) {
	if pos, ok := positionOfState(state); ok {
		pos.PlayerOnRoll = onRoll
		row.Vector = engine.BuildSimilarityVector(&pos)
	}
	away := [2]int{domain.PointsAway(away0), domain.PointsAway(away1)}
	if onRoll == 1 {
		away[0], away[1] = away[1], away[0]
	}
	row.AwayOnRoll, row.AwayOpponent = away[0], away[1]
	row.GapMP, row.Unstable, row.RolledOut = storage.ReferenceLesson(ana, row.Kind, cubeAction, costs)
}
