package sqlshared

import (
	"context"
	"fmt"
	"slices"
	"strconv"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The breakdowns count decisions; a drill-down loads positions, and one
// position reached by several decisions (an opening roll, met in every game)
// is loaded once. A row's clickable figures are therefore its positions, read
// by breakdownPositionSets — the same reading the drill-down returns, so the
// figure shown and the list loaded are one computation.

// positionSets are a breakdown row's positions: all of them, and those where
// a decision of the selection is a blunder.
type positionSets struct{ all, blunders map[int64]bool }

func (p *positionSets) add(id int64, blunder bool) {
	p.all[id] = true
	if blunder {
		p.blunders[id] = true
	}
}

// breakdownKeyName is the key a row is listed under, as the breakdown shows
// it: the token of a phase or plan of play, and for a score the cell
// computePerScore folds the pair into.
func breakdownKeyName(dim string, k breakdownKey) string {
	switch dim {
	case storage.BreakdownPhase:
		return domain.GamePhase(k.k1).String()
	case storage.BreakdownGameType:
		return domain.GameType(k.k1).String()
	case storage.BreakdownScore:
		k = scoreKey(k)
		if k.nulls == 0 && k.k1 < 0 && k.k2 < 0 {
			return "money"
		}
		return strconv.Itoa(k.k1) + "-" + strconv.Itoa(k.k2)
	}
	return ""
}

// breakdownPositionSets reads the positions of every row of dimension dim.
// filter must already carry the player aliases (withPlayerAliases).
func (s *StatsStore) breakdownPositionSets(ctx context.Context, scope string, filter storage.StatsFilter, dim string) (map[string]*positionSets, error) {
	settings, err := librarySettings(ctx, s.DB, scope)
	if err != nil {
		return nil, fmt.Errorf("breakdown positions settings: %w", err)
	}
	whereSQL, args := s.buildStatsWhereClause(scope, filter)
	blunder := `CASE WHEN ` + statsErrExpr + ` >= ? THEN 1 ELSE 0 END`
	args = append([]any{settings.BlunderThresholdMP}, args...)

	out := map[string]*positionSets{}
	row := func(key string) *positionSets {
		p := out[key]
		if p == nil {
			p = &positionSets{all: map[int64]bool{}, blunders: map[int64]bool{}}
			out[key] = p
		}
		return p
	}

	if dim == storage.BreakdownTag {
		// The decisions of commented positions, as computePerTag reads them.
		commented, cArgs := s.DB.TenantFilter("c", scope)
		byPosition := map[int64]bool{}
		err := scanEach(ctx, s.DB,
			`SELECT DISTINCT p.id, `+blunder+` `+statsBaseJoin+whereSQL+
				` AND p.id IN (SELECT c.position_id FROM comment c WHERE `+commented+` AND c.text != '')`,
			append(args, cArgs...), func(r Rows) error {
				var id int64
				var b int
				if err := r.Scan(&id, &b); err != nil {
					return err
				}
				byPosition[id] = byPosition[id] || b == 1
				return nil
			})
		if err != nil {
			return nil, fmt.Errorf("breakdown positions (tag): %w", err)
		}
		if len(byPosition) == 0 {
			return out, nil
		}
		tags, err := s.tagsOfPositions(ctx, scope, byPosition)
		if err != nil {
			return nil, err
		}
		for id, b := range byPosition {
			for _, tag := range tags[id] {
				row(tag).add(id, b)
			}
		}
		return out, nil
	}

	var cols []string
	switch dim {
	case storage.BreakdownPhase:
		cols = []string{"p.game_phase"}
	case storage.BreakdownGameType:
		cols = []string{"p.game_type"}
	case storage.BreakdownScore:
		cols = []string{"p.score_1", "p.score_2"}
	default:
		return nil, fmt.Errorf("breakdown positions: unknown dimension %q", dim)
	}
	// NULL is read as 0 and flagged, as breakdownRowsBy reads it.
	shown, nulls := "", "0"
	for i, c := range cols {
		shown += `COALESCE(` + c + `, 0), `
		nulls += fmt.Sprintf(` + CASE WHEN %s IS NULL THEN %d ELSE 0 END`, c, 1<<i)
	}
	err = scanEach(ctx, s.DB,
		`SELECT DISTINCT p.id, `+shown+nulls+`, `+blunder+` `+statsBaseJoin+whereSQL, args,
		func(r Rows) error {
			var id int64
			var k breakdownKey
			var b int
			dest := []any{&id, &k.k1}
			if len(cols) > 1 {
				dest = append(dest, &k.k2)
			}
			if err := r.Scan(append(dest, &k.nulls, &b)...); err != nil {
				return err
			}
			row(breakdownKeyName(dim, k)).add(id, b == 1)
			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("breakdown positions (%s): %w", dim, err)
	}
	return out, nil
}

// breakdownSelectionIDs is PositionIDsBySelection for Kind "breakdown", in
// ascending id order.
func (s *StatsStore) breakdownSelectionIDs(ctx context.Context, scope string, filter storage.StatsFilter, sel storage.SelectionSpec) ([]int64, error) {
	sets, err := s.breakdownPositionSets(ctx, scope, filter, sel.Breakdown)
	if err != nil {
		return nil, err
	}
	p := sets[sel.BreakdownKey]
	if p == nil {
		return nil, nil
	}
	from := p.all
	if sel.OnlyBlunders {
		from = p.blunders
	}
	ids := make([]int64, 0, len(from))
	for id := range from {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids, nil
}

// BreakdownPositionCounts counts the positions behind every breakdown row.
func (s *StatsStore) BreakdownPositionCounts(ctx context.Context, scope string, filter storage.StatsFilter) (storage.BreakdownPositionCounts, error) {
	filter, err := s.withPlayerAliases(ctx, scope, filter)
	if err != nil {
		return nil, fmt.Errorf("BreakdownPositionCounts aliases: %w", err)
	}
	out := storage.BreakdownPositionCounts{}
	for _, dim := range []string{storage.BreakdownPhase, storage.BreakdownGameType, storage.BreakdownTag, storage.BreakdownScore} {
		sets, err := s.breakdownPositionSets(ctx, scope, filter, dim)
		if err != nil {
			return nil, err
		}
		counts := make(map[string]storage.BreakdownPositionCount, len(sets))
		for key, p := range sets {
			counts[key] = storage.BreakdownPositionCount{Positions: len(p.all), Blunders: len(p.blunders)}
		}
		out[dim] = counts
	}
	return out, nil
}
