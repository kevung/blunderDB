package sqlshared

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"slices"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// playerMWC7 pools each player's (match, seat) MWC losses into an L7 for the
// players table (ADR-0075, ADR-0078): the decisions the PR beside it counts,
// converted as the statistics convert them, one unit per match the player
// sat in. Names are canonical (aliases folded), as the table's rows are.
func (s *StatsStore) playerMWC7(ctx context.Context, scope string, f storage.StatsFilter, aliases storage.AliasMap) (map[string]domain.MWC7, error) {
	statsWhere, statsArgs := s.buildStatsWhereClause(scope, f)
	type unit struct {
		match  int64
		name   string
		length int
	}
	losses := map[unit]float64{}
	err := scanEach(ctx, s.DB,
		`SELECT `+moverNameExpr+`, m.id, COALESCE(`+statsErrExpr+`, 0), COALESCE(p.score_1, 0), COALESCE(p.score_2, 0), mv.player,
			`+cubeMultiplierExpr+`, COALESCE(p.match_length, m.match_length, 0), COALESCE(m.match_length, 0) `+
			statsBaseJoin+statsWhere,
		statsArgs, func(r Rows) error {
			var name string
			var match, errMP int64
			var away0, away1, rawPlayer, cube, length, matchN int
			if err := r.Scan(&name, &match, &errMP, &away0, &away1, &rawPlayer, &cube, &length, &matchN); err != nil {
				return err
			}
			if name == "" || !domain.MWC7Defined(matchN) {
				return nil
			}
			if loss := decisionMWCLoss(errMP, away0, away1, rawPlayer, cube, length); !math.IsNaN(loss) {
				losses[unit{match, aliases.Canonical(name), matchN}] += loss
			}
			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("PlayerTable MWC7: %w", err)
	}
	keys := make([]unit, 0, len(losses))
	for k := range losses {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, func(a, b unit) int { return cmp.Or(cmp.Compare(a.name, b.name), cmp.Compare(a.match, b.match)) })
	pools := map[string]*domain.MWC7Pool{}
	for _, k := range keys {
		p := pools[k.name]
		if p == nil {
			p = &domain.MWC7Pool{}
			pools[k.name] = p
		}
		p.AddLoss(losses[k], k.length)
	}
	out := make(map[string]domain.MWC7, len(pools))
	for name, p := range pools {
		out[name] = p.Result()
	}
	return out, nil
}
