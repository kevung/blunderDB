package sqlshared

// stats_from_cells.go — Compute over match_stats and its cells
// (match_stats_cells.go) instead of over every decision row. Every predicate
// a filter without provenance carries is at the grain of a cell — match,
// seat, decision type, table of the analysis — so a sum of cells is the sum
// of the decisions the direct passes would select. The three figures that
// depend on individual decisions (the worst errors, the most recent
// decisions, their MWC) run the direct passes over the few matches that can
// hold them, which the cells name.

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// cellsJoin reads match_stats_cell; positionsJoin match_stats_position.
// Both alias their table c, so cellsWhere serves either.
const (
	cellsJoin     = ` FROM match_stats_cell c JOIN match m ON m.id = c.match_id`
	positionsJoin = ` FROM match_stats_position c JOIN match m ON m.id = c.match_id`
)

// cellsWhere renders filter against cellsJoin or positionsJoin at the grain
// of the direct passes: the named players' seats, the decision type, and the
// MET predicate on the table of each cell's analyses (metComparableDecision).
func (s *StatsStore) cellsWhere(scope string, filter storage.StatsFilter) (string, []any) {
	d := s.DB
	clauses, args := s.matchClauses(scope, filter)
	tenant, targs := d.TenantFilter("c", scope)
	clauses = append(clauses, tenant)
	args = append(args, targs...)
	if names := storage.PlayerNameSet(filter); len(names) > 0 {
		ph := Placeholders(len(names))
		clauses = append(clauses, "((c.seat = 1 AND m.player1_name IN ("+ph+")) OR (c.seat = 2 AND m.player2_name IN ("+ph+")))")
		for range 2 {
			for _, n := range names {
				args = append(args, n)
			}
		}
	}
	if filter.DecisionType >= 0 {
		clauses = append(clauses, "c.decision_type = ?")
		args = append(args, filter.DecisionType)
	}
	cur, curArgs := currentMETExpr(d, scope)
	clauses = append(clauses, "("+matchIsMoney+" OR NULLIF(c.met_id, 0) IS NOT DISTINCT FROM "+cur+")")
	args = append(args, curArgs...)
	return " WHERE " + strings.Join(clauses, " AND "), args
}

// cellKind narrows a cellsWhere clause to one kind.
func cellKind(where string, kind int) string {
	return fmt.Sprintf("%s AND c.kind = %d", where, kind)
}

// unNull renders a cell dimension with cellNull read back as 0, the
// COALESCE the direct passes report a NULL column with.
func unNull(col string) string {
	return fmt.Sprintf("CASE WHEN %s = %d THEN 0 ELSE %s END", col, cellNull, col)
}

// computeFromCells is Compute's run over match_stats and its cells, inside
// the read transaction: same passes, same order (the MWC back-fill last
// among the per-decision figures), same result.
func (s *StatsStore) computeFromCells(ctx context.Context, q statsQuery, result *storage.StatsResult) error {
	where, args := s.cellsWhere(q.scope, q.filter)
	for _, pass := range []func(context.Context, statsQuery, string, []any, *storage.StatsResult) error{
		s.totalsFromCells,
		func(ctx context.Context, q statsQuery, _ string, _ []any, r *storage.StatsResult) error {
			for _, p := range []func(context.Context, statsQuery, *storage.StatsResult) error{
				s.prByDecisionTypeFromTable, s.snowieGlobalFromTable, s.perTournamentFromTable, s.perMatchFromTable,
			} {
				if err := p(ctx, q, r); err != nil {
					return err
				}
			}
			return nil
		},
		s.cubeFromCells,
		s.histogramFromCells,
		s.topBlundersFromCells,
		s.rollingFromCells,
		s.mwcFromCells,
		s.breakdownsFromCells,
		func(ctx context.Context, q statsQuery, _ string, _ []any, r *storage.StatsResult) error {
			return s.computePerTag(ctx, q, r)
		},
	} {
		if err := pass(ctx, q, where, args, result); err != nil {
			return err
		}
	}
	return nil
}

func (s *StatsStore) totalsFromCells(ctx context.Context, _ statsQuery, where string, args []any, result *storage.StatsResult) error {
	d := s.DB
	var decisions, private int64
	if err := d.QueryRow(ctx,
		`SELECT `+d.Bigint(`COALESCE(SUM(c.decisions), 0)`)+`, COUNT(DISTINCT c.match_id), COUNT(DISTINCT m.tournament_id), `+
			d.Bigint(`COALESCE(SUM(c.positions), 0)`)+cellsJoin+cellKind(where, cellPhase), args...,
	).Scan(&decisions, &result.Totals.NumMatches, &result.Totals.NumTournaments, &private); err != nil {
		return fmt.Errorf("totals (cells): %w", err)
	}
	result.Totals.NumDecisions = int(decisions)
	// A private position is in one seat's phase cell and nowhere else; a
	// shared one has a row for each seat reaching it (sharePositions).
	var shared int
	if err := d.QueryRow(ctx, `SELECT COUNT(DISTINCT c.position_id)`+positionsJoin+where, args...).Scan(&shared); err != nil {
		return fmt.Errorf("totals positions (cells): %w", err)
	}
	result.Totals.NumPositions = int(private) + shared
	return nil
}

// cubeFromCells is computeCubeActionBreakdown and computeCubeDirections.
func (s *StatsStore) cubeFromCells(ctx context.Context, _ statsQuery, where string, args []any, result *storage.StatsResult) error {
	d := s.DB
	cube := cellKind(where, cellCube)
	if err := scanEach(ctx, d,
		`SELECT `+ActionLabelOrEmptyFor(d, "c.k1")+`, `+d.Bigint(`SUM(c.error_mp)`)+`, `+d.Bigint(`SUM(c.decisions)`)+`, `+
			d.Bigint(`SUM(c.blunders)`)+cellsJoin+cube+` GROUP BY c.k1, 1 ORDER BY c.k1`,
		args, func(r Rows) error {
			var cs storage.CubeActionStats
			var sumErr, n, blunders int64
			if err := r.Scan(&cs.Action, &sumErr, &n, &blunders); err != nil {
				return err
			}
			cs.NumDecisions, cs.BlunderCount = int(n), int(blunders)
			cs.PR = pr(sumErr, cs.NumDecisions)
			result.CubeActionBreakdown = append(result.CubeActionBreakdown, cs)
			return nil
		}); err != nil {
		return fmt.Errorf("cube action breakdown (cells): %w", err)
	}
	var cells []storage.CubeDirectionRow
	if err := scanEach(ctx, d,
		`SELECT `+ActionLabelOrEmptyFor(d, "c.k1")+`, `+ActionLabelOrEmptyFor(d, "c.k2")+`, `+d.Bigint(`SUM(c.decisions)`)+`, `+
			d.Bigint(`SUM(c.error_mp)`)+cellsJoin+cube+` GROUP BY c.k1, c.k2, 1, 2`,
		args, func(r Rows) error {
			var c storage.CubeDirectionRow
			var n int64
			if err := r.Scan(&c.Best, &c.Played, &n, &c.ErrorMP); err != nil {
				return err
			}
			c.Count = int(n)
			cells = append(cells, c)
			return nil
		}); err != nil {
		return fmt.Errorf("cube directions (cells): %w", err)
	}
	result.CubeDirections = storage.TallyCubeDirections(cells)
	return nil
}

func (s *StatsStore) histogramFromCells(ctx context.Context, _ statsQuery, where string, args []any, result *storage.StatsResult) error {
	d := s.DB
	bucketMax := map[int]int{0: 5, 5: 10, 10: 25, 25: 50, 50: 100, 100: -1}
	return scanEach(ctx, d,
		`SELECT c.k1, `+d.Bigint(`SUM(c.decisions)`)+cellsJoin+cellKind(where, cellBucket)+` GROUP BY c.k1 ORDER BY c.k1`,
		args, func(r Rows) error {
			var bucket, n int64
			if err := r.Scan(&bucket, &n); err != nil {
				return fmt.Errorf("error histogram (cells): %w", err)
			}
			result.ErrorHistogram = append(result.ErrorHistogram, storage.ErrorBucket{
				MinMP: int(bucket), MaxMP: bucketMax[int(bucket)], Count: int(n),
			})
			return nil
		})
}

// restrictTo narrows the direct passes' selection to matchIDs.
func restrictTo(q statsQuery, matchIDs []int64) statsQuery {
	r := q
	r.baseArgs = append([]any{}, q.baseArgs...)
	if len(matchIDs) == 0 {
		r.whereSQL += " AND 1 = 0"
		return r
	}
	r.whereSQL += " AND m.id IN (" + Placeholders(len(matchIDs)) + ")"
	r.baseArgs = append(r.baseArgs, int64Args(matchIDs)...)
	return r
}

// topBlunderMatches names the matches that can hold the ten worst errors:
// every cell's largest error is one decision of the selection, so the tenth
// largest of them, L, bounds the tenth worst error from below, and every
// decision at L or above lies in a cell whose largest error is at L or above.
func (s *StatsStore) topBlunderMatches(ctx context.Context, where string, args []any) ([]int64, error) {
	d := s.DB
	phase := cellKind(where, cellPhase)
	var bound int64
	err := d.QueryRow(ctx, `SELECT c.max_error_mp`+cellsJoin+phase+` ORDER BY c.max_error_mp DESC LIMIT 1 OFFSET 9`, args...).Scan(&bound)
	switch {
	case err == nil:
		phase += " AND c.max_error_mp >= ?"
		args = append(append([]any{}, args...), bound)
	case !errors.Is(err, ErrNoRows):
		return nil, fmt.Errorf("top blunders bound (cells): %w", err)
	}
	var ids []int64
	err = scanEach(ctx, d, `SELECT DISTINCT c.match_id`+cellsJoin+phase, args, func(r Rows) error {
		var id int64
		if err := r.Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("top blunder matches (cells): %w", err)
	}
	return ids, nil
}

// topBlundersFromCells is computeTopBlunders over the matches that can hold
// the result, and their MWC losses from computeMWCPass over the same.
func (s *StatsStore) topBlundersFromCells(ctx context.Context, q statsQuery, where string, args []any, result *storage.StatsResult) error {
	ids, err := s.topBlunderMatches(ctx, where, args)
	if err != nil {
		return err
	}
	rq := restrictTo(q, ids)
	if err := s.computeTopBlunders(ctx, rq, result); err != nil {
		return err
	}
	scratch := &storage.StatsResult{TopBlunders: result.TopBlunders}
	if err := s.computeMWCPass(ctx, rq, scratch); err != nil {
		return err
	}
	result.TopBlunders = scratch.TopBlunders
	return nil
}

// rollingWindow is the largest window computeRollingPR and the MWC pass read.
const rollingWindow = 1000

// recentMatches names the matches that hold the rollingWindow most recent
// decisions: the matches in the direct passes' order (match_date DESC) until
// they total rollingWindow decisions, then every further match of the same
// date, since the direct order interleaves a date's matches by move number.
func (s *StatsStore) recentMatches(ctx context.Context, where string, args []any) ([]int64, error) {
	d := s.DB
	var ids []int64
	var total int64
	var boundary *string
	stop := fmt.Errorf("enough")
	err := scanEach(ctx, d,
		`SELECT c.match_id, `+d.DateText("m.match_date")+`, `+d.Bigint(`SUM(c.decisions)`)+cellsJoin+cellKind(where, cellPhase)+
			` GROUP BY c.match_id, m.match_date ORDER BY m.match_date DESC`,
		args, func(r Rows) error {
			var id, n int64
			var date string
			if err := r.Scan(&id, &date, &n); err != nil {
				return err
			}
			if boundary != nil && date != *boundary {
				return stop
			}
			ids = append(ids, id)
			total += n
			if boundary == nil && total >= rollingWindow {
				boundary = &date
			}
			return nil
		})
	if err != nil && err != stop { //nolint:errorlint // the sentinel is ours and never wrapped
		return nil, fmt.Errorf("recent matches (cells): %w", err)
	}
	return ids, nil
}

// rollingFromCells is computeRollingPR and the MWC pass's rolling sums over
// the matches holding the most recent decisions.
func (s *StatsStore) rollingFromCells(ctx context.Context, q statsQuery, where string, args []any, result *storage.StatsResult) error {
	ids, err := s.recentMatches(ctx, where, args)
	if err != nil {
		return err
	}
	rq := restrictTo(q, ids)
	if err := s.computeRollingPR(ctx, rq, result); err != nil {
		return err
	}
	scratch := &storage.StatsResult{}
	if err := s.computeMWCPass(ctx, rq, scratch); err != nil {
		return err
	}
	result.MWCRolling = scratch.MWCRolling
	return nil
}

// mwcFromCells fills the MWC sums and back-fills the per-tournament,
// per-match and per-cube-action rows, as computeMWCPass does.
func (s *StatsStore) mwcFromCells(ctx context.Context, _ statsQuery, where string, args []any, result *storage.StatsResult) error {
	d := s.DB
	byMatch := map[int64]float64{}
	byTournament := map[int64]float64{}
	if err := scanEach(ctx, d,
		`SELECT c.match_id, COALESCE(m.tournament_id, 0), c.decision_type, SUM(c.mwc_loss), `+d.Bigint(`SUM(c.mwc_decisions)`)+
			cellsJoin+cellKind(where, cellPhase)+` GROUP BY c.match_id, m.tournament_id, c.decision_type ORDER BY c.match_id, c.decision_type`,
		args, func(r Rows) error {
			var match, tournament, n int64
			var dt int
			var loss float64
			if err := r.Scan(&match, &tournament, &dt, &loss, &n); err != nil {
				return err
			}
			if n == 0 {
				return nil
			}
			result.MWCAvailable = true
			result.MWCGlobal += loss
			if dt == 0 {
				result.MWCChecker += loss
			} else {
				result.MWCCube += loss
			}
			byMatch[match] += loss
			if tournament != 0 {
				byTournament[tournament] += loss
			}
			return nil
		}); err != nil {
		return fmt.Errorf("MWC (cells): %w", err)
	}
	byAction := map[string]float64{}
	if err := scanEach(ctx, d,
		`SELECT `+ActionLabelOrEmptyFor(d, "c.k1")+`, SUM(c.mwc_loss)`+cellsJoin+cellKind(where, cellCube)+` GROUP BY 1`,
		args, func(r Rows) error {
			var label string
			var loss float64
			if err := r.Scan(&label, &loss); err != nil {
				return err
			}
			byAction[label] += loss
			return nil
		}); err != nil {
		return fmt.Errorf("MWC per cube action (cells): %w", err)
	}
	for i, ts := range result.PerTournament {
		result.PerTournament[i].MWC = byTournament[ts.ID]
	}
	for i, ms := range result.PerMatch {
		result.PerMatch[i].MWC = byMatch[ms.ID]
	}
	for i, cs := range result.CubeActionBreakdown {
		result.CubeActionBreakdown[i].MWC = byAction[cs.Action]
	}
	if result.MWCRolling == nil {
		result.MWCRolling = map[int]float64{}
	}
	return nil
}

// breakdownsFromCells is computePerPhase, computePerGameType and computePerScore.
func (s *StatsStore) breakdownsFromCells(ctx context.Context, _ statsQuery, where string, args []any, result *storage.StatsResult) error {
	d := s.DB
	sums := d.Bigint(`SUM(c.error_mp)`) + `, ` + d.Bigint(`SUM(c.decisions)`) + `, ` + d.Bigint(`SUM(c.blunders)`)
	type tally struct{ sumErr, n, blunders int64 }
	read := func(kind int, cols, group string, fn func(k1, k2 int, t tally)) error {
		return scanEach(ctx, d, `SELECT `+cols+`, `+sums+cellsJoin+cellKind(where, kind)+` GROUP BY `+group+` ORDER BY `+group,
			args, func(r Rows) error {
				var k1, k2 int64
				var t tally
				dest := []any{&k1}
				if kind == cellScore {
					dest = append(dest, &k2)
				}
				if err := r.Scan(append(dest, &t.sumErr, &t.n, &t.blunders)...); err != nil {
					return err
				}
				fn(int(k1), int(k2), t)
				return nil
			})
	}
	if err := read(cellPhase, unNull("c.k1"), "c.k1", func(k, _ int, t tally) {
		result.PerPhase = append(result.PerPhase, storage.PhaseStats{
			Phase: domain.GamePhase(k).String(), PR: pr(t.sumErr, int(t.n)), NumDecisions: int(t.n), BlunderCount: int(t.blunders),
		})
	}); err != nil {
		return fmt.Errorf("per-phase (cells): %w", err)
	}
	if err := read(cellGameType, unNull("c.k1"), "c.k1", func(k, _ int, t tally) {
		result.PerGameType = append(result.PerGameType, storage.GameTypeStats{
			GameType: domain.GameType(k).String(), PR: pr(t.sumErr, int(t.n)), NumDecisions: int(t.n), BlunderCount: int(t.blunders),
		})
	}); err != nil {
		return fmt.Errorf("per-game-type (cells): %w", err)
	}
	if err := read(cellScore, unNull("c.k1")+`, `+unNull("c.k2"), "c.k1, c.k2", func(k1, k2 int, t tally) {
		result.PerScore = append(result.PerScore, storage.ScoreCellStats{
			MoverAway: k1, OpponentAway: k2, PR: pr(t.sumErr, int(t.n)), NumDecisions: int(t.n), BlunderCount: int(t.blunders),
		})
	}); err != nil {
		return fmt.Errorf("per-score (cells): %w", err)
	}
	// The direct pass sorts the matrix in Go; the cells' order is already
	// that one save for NULL scores, read as 0 and kept apart.
	sortScoreCells(result.PerScore)
	return nil
}
