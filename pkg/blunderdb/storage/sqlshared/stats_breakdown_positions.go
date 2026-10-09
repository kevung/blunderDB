package sqlshared

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

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
//
// A non-empty only restricts the reading to that row: the key is pushed into
// the WHERE as a superset of the matching raw values and then checked as the
// breakdown names it, so a drill-down reads one row and not the dimension.
func (s *StatsStore) breakdownPositionSets(ctx context.Context, scope string, filter storage.StatsFilter, dim, only string) (map[string]*positionSets, error) {
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
	whereSQL += breakdownKeyWhere(dim, only)
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
			name := breakdownKeyName(dim, k)
			if only != "" && name != only {
				return nil
			}
			row(name).add(id, b == 1)
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
	sets, err := s.breakdownPositionSets(ctx, scope, filter, sel.Breakdown, sel.BreakdownKey)
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
	for _, dim := range []string{storage.BreakdownPhase, storage.BreakdownGameType, storage.BreakdownScore} {
		counts, err := s.breakdownCountsSQL(ctx, scope, filter, dim)
		if err != nil {
			return nil, err
		}
		out[dim] = counts
	}
	// A tag is not a column of the position: its rows come from the comments,
	// which only the commented positions carry.
	sets, err := s.breakdownPositionSets(ctx, scope, filter, storage.BreakdownTag, "")
	if err != nil {
		return nil, err
	}
	tags := make(map[string]storage.BreakdownPositionCount, len(sets))
	for key, p := range sets {
		tags[key] = storage.BreakdownPositionCount{Positions: len(p.all), Blunders: len(p.blunders)}
	}
	out[storage.BreakdownTag] = tags
	return out, nil
}

// breakdownDimensionColumns are the position columns a dimension is keyed on,
// with the SQL that reads them as breakdownRowsBy does: a NULL is read as 0 and
// flagged.
func breakdownDimensionColumns(dim string) (cols []string, shown, nulls string, err error) {
	switch dim {
	case storage.BreakdownPhase:
		cols = []string{"p.game_phase"}
	case storage.BreakdownGameType:
		cols = []string{"p.game_type"}
	case storage.BreakdownScore:
		cols = []string{"p.score_1", "p.score_2"}
	default:
		return nil, "", "", fmt.Errorf("breakdown positions: unknown dimension %q", dim)
	}
	nulls = "0"
	for i, c := range cols {
		if i > 0 {
			shown += ", "
		}
		shown += `COALESCE(` + c + `, 0)`
		nulls += fmt.Sprintf(` + CASE WHEN %s IS NULL THEN %d ELSE 0 END`, c, 1<<i)
	}
	return cols, shown, nulls, nil
}

// breakdownCountsSQL counts a column dimension's positions in one grouped pass,
// without materialising any id. A position has one value of each column, so it
// falls in one group: the groups' distinct counts add up exactly when
// computePerScore folds several raw pairs into one cell.
func (s *StatsStore) breakdownCountsSQL(ctx context.Context, scope string, filter storage.StatsFilter, dim string) (map[string]storage.BreakdownPositionCount, error) {
	settings, err := librarySettings(ctx, s.DB, scope)
	if err != nil {
		return nil, fmt.Errorf("breakdown positions settings: %w", err)
	}
	cols, shown, nulls, err := breakdownDimensionColumns(dim)
	if err != nil {
		return nil, err
	}
	whereSQL, args := s.buildStatsWhereClause(scope, filter)
	args = append([]any{settings.BlunderThresholdMP}, args...)
	out := map[string]storage.BreakdownPositionCount{}
	err = scanEach(ctx, s.DB,
		`SELECT `+shown+`, `+nulls+`, COUNT(DISTINCT p.id),
		        COUNT(DISTINCT CASE WHEN `+statsErrExpr+` >= ? THEN p.id END) `+
			statsBaseJoin+whereSQL+` GROUP BY `+shown+`, `+nulls, args,
		func(r Rows) error {
			var k breakdownKey
			var c storage.BreakdownPositionCount
			dest := []any{&k.k1}
			if len(cols) > 1 {
				dest = append(dest, &k.k2)
			}
			if err := r.Scan(append(dest, &k.nulls, &c.Positions, &c.Blunders)...); err != nil {
				return err
			}
			name := breakdownKeyName(dim, k)
			sum := out[name]
			sum.Positions += c.Positions
			sum.Blunders += c.Blunders
			out[name] = sum
			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("breakdown position counts (%s): %w", dim, err)
	}
	return out, nil
}

// breakdownKeyWhere is the SQL restricting a column dimension to the raw
// values a row key can come from; empty for no key or a dimension without
// columns. It may keep more than the row (the caller checks the name) but
// never less.
func breakdownKeyWhere(dim, key string) string {
	if key == "" {
		return ""
	}
	ints := func(vs []int) string {
		parts := make([]string, len(vs))
		for i, v := range vs {
			parts[i] = strconv.Itoa(v)
		}
		return strings.Join(parts, ",")
	}
	switch dim {
	case storage.BreakdownPhase:
		var vs []int
		for ph := range domain.GamePhaseNames {
			if ph.String() == key {
				vs = append(vs, int(ph))
			}
		}
		if len(vs) == 0 {
			return " AND 1 = 0"
		}
		return " AND COALESCE(p.game_phase, 0) IN (" + ints(vs) + ")"
	case storage.BreakdownGameType:
		var vs []int
		for gt := range domain.GameTypeNames {
			if gt.String() == key {
				vs = append(vs, int(gt))
			}
		}
		if len(vs) == 0 {
			return " AND 1 = 0"
		}
		return " AND COALESCE(p.game_type, 0) IN (" + ints(vs) + ")"
	case storage.BreakdownScore:
		if key == "money" {
			return " AND p.score_1 < 0 AND p.score_2 < 0"
		}
		a, b, ok := strings.Cut(key[1:], "-")
		if !ok {
			return " AND 1 = 0"
		}
		x, errA := strconv.Atoi(key[:1] + a)
		y, errB := strconv.Atoi(b)
		if errA != nil || errB != nil {
			return " AND 1 = 0"
		}
		return " AND COALESCE(p.score_1, 0) IN (" + ints(awayRaw(x)) + ") AND COALESCE(p.score_2, 0) IN (" + ints(awayRaw(y)) + ")"
	}
	return ""
}

// awayRaw are the stored values a displayed away score can come from.
func awayRaw(x int) []int {
	if domain.PointsAway(domain.PostCrawford) == x {
		return []int{x, domain.PostCrawford}
	}
	return []int{x}
}
