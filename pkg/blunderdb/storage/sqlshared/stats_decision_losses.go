package sqlshared

import (
	"context"
	"math"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// MatchDecisionLosses lists every Move of a Match with the winning chances its
// play cost. The error, the counted predicate and the conversion are the ones
// MatchBadges and the statistics cells use (statsErrExpr, countedExpr,
// decisionMWCLoss), so a player's losses add up to the badge's MWC loss. A
// Move outside the counted set, or one the conversion cannot price, keeps a
// nil loss.
func (s *StatsStore) MatchDecisionLosses(ctx context.Context, scope string, matchID int64) ([]storage.DecisionLoss, error) {
	tenant, args := s.DB.TenantFilter("mv", scope)
	rows, err := s.DB.Query(ctx,
		`SELECT mv.id, g.game_number, mv.move_number, COALESCE(mv.player, 0), `+ActionLabelOrEmptyFor(s.DB, "mv.move_type")+`,
			CASE WHEN a.position_id IS NOT NULL AND (`+statsErrExpr+`) IS NOT NULL AND `+countedExpr(s.DB)+` THEN 1 ELSE 0 END,
			COALESCE(`+statsErrExpr+`, 0), COALESCE(p.score_1, 0), COALESCE(p.score_2, 0),
			`+cubeMultiplierExpr+`, COALESCE(p.match_length, m.match_length, 0)
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

	var out []storage.DecisionLoss
	for rows.Next() {
		var d storage.DecisionLoss
		var rawPlayer, counted, away0, away1, cubeValue, matchLength int
		var moveType string
		var errMP int64
		if err := rows.Scan(&d.MoveID, &d.GameNumber, &d.MoveNumber, &rawPlayer, &moveType, &counted,
			&errMP, &away0, &away1, &cubeValue, &matchLength); err != nil {
			return nil, errf(s.DB, "MatchDecisionLosses scan", err)
		}
		d.DecisionType = "checker"
		if moveType == "cube" {
			d.DecisionType = "cube"
		}
		if rawPlayer == -1 {
			d.Player = 1
		}
		if counted == 1 {
			if loss := decisionMWCLoss(errMP, away0, away1, rawPlayer, cubeValue, matchLength); !math.IsNaN(loss) {
				d.MWCLoss = &loss
			}
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, errf(s.DB, "MatchDecisionLosses rows", err)
	}
	return out, nil
}
