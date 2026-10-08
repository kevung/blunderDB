package sqlshared

import (
	"context"
	"math"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// MatchDecisionLosses lists every Move of a Match with the winning chances its
// play cost. The error, the counted predicate and the conversion are the ones
// MatchBadges and the statistics cells use (statsErrExpr, countedExpr,
// decisionMWCLoss), so a player's losses add up to the badge's MWC loss. A
// Move outside the counted set, or one the conversion cannot price, keeps a
// nil loss.
//
// A priced Move also carries its difficulty (ADR-0076): the reference
// player's expected loss over the options its analysis lists, converted by the
// same function, and the avoidable mark that compares the two. Each blob is
// decoded once per Position.
func (s *StatsStore) MatchDecisionLosses(ctx context.Context, scope string, matchID int64) ([]storage.DecisionLoss, error) {
	settings, err := librarySettings(ctx, s.DB, scope)
	if err != nil {
		return nil, errf(s.DB, "MatchDecisionLosses settings", err)
	}
	tenant, args := s.DB.TenantFilter("mv", scope)
	rows, err := s.DB.Query(ctx,
		`SELECT mv.id, g.game_number, mv.move_number, COALESCE(mv.player, 0), `+ActionLabelOrEmptyFor(s.DB, "mv.move_type")+`,
			`+ActionLabelOrEmptyFor(s.DB, "mv.cube_action")+`, COALESCE(mv.position_id, 0), a.data,
			CASE WHEN a.position_id IS NOT NULL AND (`+statsErrExpr+`) IS NOT NULL AND `+countedExpr(s.DB)+` THEN 1 ELSE 0 END,
			COALESCE(`+statsErrExpr+`, 0), COALESCE(p.score_1, 0), COALESCE(p.score_2, 0),
			`+cubeMultiplierExpr+`, COALESCE(p.match_length, m.match_length, 0), mv.luck_mp, mv.decision_ms,
			CASE WHEN p.id IS NULL THEN 0 ELSE 1 END
		 FROM move mv
		 JOIN game g ON g.id = mv.game_id
		 JOIN match m ON m.id = g.match_id
		 LEFT JOIN position p ON p.id = mv.position_id
		 LEFT JOIN analysis a ON a.position_id = p.id
		 WHERE `+tenant+` AND g.match_id = ?
		 ORDER BY g.game_number, mv.move_number, mv.id`,
		append(args, matchID)...)
	if err != nil {
		return nil, errf(s.DB, "MatchDecisionLosses query", err)
	}
	defer rows.Close()

	decoded := make(map[int64]*domain.PositionAnalysis)
	var out []storage.DecisionLoss
	for rows.Next() {
		var d storage.DecisionLoss
		var rawPlayer, counted, away0, away1, cubeValue, matchLength int
		var moveType, cubeAction string
		var positionID, errMP int64
		var data []byte
		var luckMP, durationMS *int64
		var hasPosition int
		if err := rows.Scan(&d.MoveID, &d.GameNumber, &d.MoveNumber, &rawPlayer, &moveType, &cubeAction, &positionID, &data, &counted,
			&errMP, &away0, &away1, &cubeValue, &matchLength, &luckMP, &durationMS, &hasPosition); err != nil {
			return nil, errf(s.DB, "MatchDecisionLosses scan", err)
		}
		d.DecisionType = "checker"
		if moveType == "cube" {
			d.DecisionType = "cube"
		}
		if rawPlayer == -1 {
			d.Player = 1
		}
		d.DurationMS = durationMS
		if luckMP != nil && hasPosition == 1 && d.DecisionType == "checker" {
			if luck := decisionMWCLoss(*luckMP, away0, away1, rawPlayer, cubeValue, matchLength); !math.IsNaN(luck) {
				d.Luck = &luck
			}
		}
		if counted == 1 {
			e := errMP
			d.ErrorMP = &e
			d.Error = errMP >= int64(settings.ErrorThresholdMP)
			if loss := decisionMWCLoss(errMP, away0, away1, rawPlayer, cubeValue, matchLength); !math.IsNaN(loss) {
				d.MWCLoss = &loss
				analysis, seen := decoded[positionID]
				if !seen {
					if a, err := engine.DecodeAnalysisFromStorage(data); err == nil {
						analysis = &a
					}
					decoded[positionID] = analysis // nil too: an undecodable blob is not retried
				}
				if costs := decisionCosts(analysis, d.DecisionType, cubeAction); costs != nil {
					// The conversion is linear in equity: its price of one unit
					// turns the reference loss into the unit of the play's loss.
					perUnit := decisionMWCLoss(1000, away0, away1, rawPlayer, cubeValue, matchLength)
					diff := perUnit * storage.ReferenceExpectedLoss(costs, storage.DifficultyTemperature)
					d.Difficulty = &diff
					d.Avoidable = storage.IsAvoidable(loss, diff, errMP >= int64(settings.ErrorThresholdMP))
				}
			}
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, errf(s.DB, "MatchDecisionLosses rows", err)
	}
	return out, nil
}

// decisionCosts lists what each option of a decision costs, in equity, as the
// play's own error is scored (ADR-0076): a checker decision's candidates, the
// best at 0; a doubling decision's no double and double (against the best
// answer, engine.CubeActionError); an answer's take and pass. nil when the
// analysis gives fewer than the decision's options.
func decisionCosts(analysis *domain.PositionAnalysis, decisionType, cubeAction string) []float64 {
	if analysis == nil {
		return nil
	}
	if decisionType == "cube" {
		dca := analysis.DoublingCubeAnalysis
		if dca == nil {
			return nil
		}
		switch engine.CanonicalCubeAction(cubeAction) {
		case engine.CubeNoDouble, engine.CubeDouble:
			nd, _ := engine.CubeActionError(dca, engine.CubeNoDouble)
			dbl, _ := engine.CubeActionError(dca, engine.CubeDouble)
			return []float64{math.Abs(nd), math.Abs(dbl)}
		case engine.CubeTake, engine.CubePass:
			take, _ := engine.CubeActionError(dca, engine.CubeTake)
			pass, _ := engine.CubeActionError(dca, engine.CubePass)
			return []float64{math.Abs(take), math.Abs(pass)}
		}
		return nil
	}
	ca := analysis.CheckerAnalysis
	if ca == nil || len(ca.Moves) == 0 {
		return nil
	}
	costs := []float64{0}
	for _, m := range ca.Moves[1:] {
		if m.EquityError != nil {
			costs = append(costs, math.Abs(*m.EquityError))
		}
	}
	return costs
}
