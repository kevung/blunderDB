package sqlshared

// stats_intervals.go — the PR intervals of the statistics (ADR-0077). An
// aggregate resamples its matches: the two seats of a match play the same
// positions and form one unit, and the games of a match share an opponent and
// a sitting, so neither is an independent sample. Both computation paths sum
// the same (key, match) units here, so they give the same interval.

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// prScale turns a ratio of millipoints per decision into a PR (see pr).
const prScale = 500.0 / 1000.0

// matchUnits sums errors and decisions per key and match.
type matchUnits[K comparable] map[K]map[int64]*[2]int64

func (u matchUnits[K]) add(k K, match, errMP, decisions int64) {
	byMatch := u[k]
	if byMatch == nil {
		byMatch = map[int64]*[2]int64{}
		u[k] = byMatch
	}
	t := byMatch[match]
	if t == nil {
		t = &[2]int64{}
		byMatch[match] = t
	}
	t[0] += errMP
	t[1] += decisions
}

// interval is k's PR interval, its matches pooled in id order so the sums
// are the same on every run.
func (u matchUnits[K]) interval(k K) domain.Interval {
	byMatch := u[k]
	var p domain.RatioPool
	for _, id := range slices.Sorted(maps.Keys(byMatch)) {
		t := byMatch[id]
		p.Add(float64(t[0]), float64(t[1]))
	}
	return p.Interval(prScale)
}

// computePRInterval is the direct pass of PRGlobal's interval.
func (s *StatsStore) computePRInterval(ctx context.Context, q statsQuery, result *storage.StatsResult) error {
	d := s.DB
	units := matchUnits[struct{}]{}
	if err := scanEach(ctx, d,
		`SELECT m.id, `+d.Bigint(`SUM(`+statsErrExpr+`)`)+`, COUNT(*) `+q.join+q.whereSQL+` GROUP BY m.id`,
		q.baseArgs, func(r Rows) error {
			var match, sumErr, n int64
			if err := r.Scan(&match, &sumErr, &n); err != nil {
				return err
			}
			units.add(struct{}{}, match, sumErr, n)
			return nil
		}); err != nil {
		return fmt.Errorf("PR interval: %w", err)
	}
	result.PRInterval = units.interval(struct{}{})
	return nil
}

// breakdownKey is one row of a direct breakdown: the dimension's values as
// shown (NULL read as 0) and which of them were NULL, since a NULL and a 0
// are two rows, as in the cells.
type breakdownKey struct{ k1, k2, nulls int }

// breakdownRow is a direct breakdown row folded over its matches.
type breakdownRow struct {
	breakdownKey
	sumErr, n, blunders int64
	interval            domain.Interval
}

// breakdownRows runs a breakdown over the position columns cols, grouped by
// match too, and folds the match rows back into one row per key, in key
// order, each with its interval over its matches.
func (s *StatsStore) breakdownRows(ctx context.Context, q statsQuery, cols ...string) ([]breakdownRow, error) {
	d := s.DB
	shown, nulls := "", "0"
	for i, c := range cols {
		shown += `COALESCE(` + c + `, 0), `
		nulls += fmt.Sprintf(` + CASE WHEN %s IS NULL THEN %d ELSE 0 END`, c, 1<<i)
	}
	group := ""
	for _, c := range cols {
		group += c + `, `
	}
	group += `m.id`
	var order []breakdownKey
	rows := map[breakdownKey]*breakdownRow{}
	units := matchUnits[breakdownKey]{}
	err := scanEach(ctx, d,
		`SELECT `+shown+nulls+`, m.id, `+d.Bigint(`SUM(`+statsErrExpr+`)`)+`, COUNT(*), `+
			d.Bigint(`SUM(CASE WHEN `+statsErrExpr+` >= ? THEN 1 ELSE 0 END)`)+` `+
			q.join+q.whereSQL+` GROUP BY `+group+` ORDER BY `+group,
		append([]any{q.settings.BlunderThresholdMP}, q.baseArgs...), func(r Rows) error {
			var k breakdownKey
			var match, sumErr, n, blunders int64
			dest := []any{&k.k1}
			if len(cols) > 1 {
				dest = append(dest, &k.k2)
			}
			if err := r.Scan(append(dest, &k.nulls, &match, &sumErr, &n, &blunders)...); err != nil {
				return err
			}
			row := rows[k]
			if row == nil {
				row = &breakdownRow{breakdownKey: k}
				rows[k] = row
				order = append(order, k)
			}
			row.sumErr += sumErr
			row.n += n
			row.blunders += blunders
			units.add(k, match, sumErr, n)
			return nil
		})
	if err != nil {
		return nil, err
	}
	out := make([]breakdownRow, 0, len(order))
	for _, k := range order {
		row := rows[k]
		row.interval = units.interval(k)
		out = append(out, *row)
	}
	return out, nil
}
