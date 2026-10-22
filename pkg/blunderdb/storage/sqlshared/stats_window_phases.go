package sqlshared

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// PRByWindowPhases — see storage.StatsStore.
func (s *StatsStore) PRByWindowPhases(ctx context.Context, scope string, filter storage.StatsFilter, months int) ([]storage.WindowStats, error) {
	return inReadSnapshot(ctx, s, func(s *StatsStore) ([]storage.WindowStats, error) {
		out, err := s.prByWindow(ctx, scope, filter, months)
		if err != nil || len(out) == 0 {
			return out, err
		}
		filter, err := s.withPlayerAliases(ctx, scope, filter)
		if err != nil {
			return nil, err
		}
		cells, err := s.phaseMonths(ctx, scope, filter)
		if err != nil {
			return nil, err
		}
		for i := range out {
			from, err1 := time.Parse("2006-01", out[i].From)
			to, err2 := time.Parse("2006-01", out[i].To)
			if err1 != nil || err2 != nil {
				return nil, fmt.Errorf("PR by window and phase: month %q..%q", out[i].From, out[i].To)
			}
			sums := map[int]phaseMonth{}
			for m := from; !m.After(to); m = m.AddDate(0, 1, 0) {
				for phase, c := range cells[m.Format("2006-01")] {
					w := sums[phase]
					w.sum += c.sum
					w.n += c.n
					w.matches += c.matches
					sums[phase] = w
				}
			}
			out[i].PerPhase = []storage.WindowPhaseStats{}
			for _, phase := range slices.Sorted(maps.Keys(sums)) {
				w := sums[phase]
				if w.n == 0 {
					continue
				}
				out[i].PerPhase = append(out[i].PerPhase, storage.WindowPhaseStats{
					Phase: domain.GamePhase(phase).String(), NumMatches: int(w.matches),
					NumDecisions: int(w.n), PR: pr(w.sum, int(w.n)),
				})
			}
		}
		return out, nil
	})
}

// phaseMonth is one phase of one calendar month: its error sum, its counted
// decisions and the matches that hold them.
type phaseMonth struct{ sum, n, matches int64 }

// phaseMonths reads the filter's counted decisions by calendar month and
// game phase, from the phase cells of match_stats when they are readable
// (the source of Compute's PerPhase), else — a read-only connection that
// finds them incomplete — from the decisions, as Compute's direct pass does.
// A position whose phase was never derived counts as phase 0, "unknown".
func (s *StatsStore) phaseMonths(ctx context.Context, scope string, filter storage.StatsFilter) (map[string]map[int]phaseMonth, error) {
	readable, err := s.matchStatsReadable(ctx, scope)
	if err != nil {
		return nil, err
	}
	d := s.DB
	month := `SUBSTR(` + d.DateText("m.match_date") + `, 1, 7)`
	var query string
	var args []any
	if readable {
		where, wargs := s.cellsWhere(scope, filter)
		phase := `CASE WHEN c.k1 = ` + fmt.Sprint(cellNull) + ` THEN 0 ELSE c.k1 END`
		query = `SELECT ` + month + ` AS mon, ` + phase + ` AS ph, ` + d.Bigint(`SUM(c.error_mp)`) + `, ` +
			d.Bigint(`SUM(c.decisions)`) + `, COUNT(DISTINCT CASE WHEN c.decisions > 0 THEN m.id END)` +
			cellsJoin + cellKind(where, cellPhase) + ` AND m.match_date IS NOT NULL GROUP BY 1, 2`
		args = wargs
	} else {
		where, wargs := s.buildStatsWhereClause(scope, filter)
		query = `SELECT ` + month + ` AS mon, COALESCE(p.game_phase, 0) AS ph, ` + d.Bigint(`SUM(`+statsErrExpr+`)`) + `, ` +
			`COUNT(*), COUNT(DISTINCT m.id) ` + statsBaseJoin + where + ` AND m.match_date IS NOT NULL GROUP BY 1, 2`
		args = wargs
	}
	out := map[string]map[int]phaseMonth{}
	err = scanEach(ctx, d, query, args, func(r Rows) error {
		var mon string
		var phase int
		var c phaseMonth
		if err := r.Scan(&mon, &phase, &c.sum, &c.n, &c.matches); err != nil {
			return err
		}
		if out[mon] == nil {
			out[mon] = map[int]phaseMonth{}
		}
		w := out[mon][phase]
		w.sum += c.sum
		w.n += c.n
		w.matches += c.matches
		out[mon][phase] = w
		return nil
	})
	if err != nil {
		return nil, errf(d, "PR by window and phase", err)
	}
	return out, nil
}
