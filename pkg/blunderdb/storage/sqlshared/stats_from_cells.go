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
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"
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
// among the per-decision figures), same result. Each kind of cell is read
// once, the phase cells raw: every figure at the grain of a match or of the
// whole selection is a sum of them, so one read serves the totals, the
// worst-error bound, the MWC sums and the per-phase rows. A read per figure
// went over the same cells again each time, and their cost is the rows
// read, not the arithmetic.
func (s *StatsStore) computeFromCells(ctx context.Context, q statsQuery, result *storage.StatsResult) error {
	where, args := s.cellsWhere(q.scope, q.filter)
	phase, err := s.readPhaseCells(ctx, where, args)
	if err != nil {
		return err
	}
	cube, err := s.readCubeCells(ctx, where, args)
	if err != nil {
		return err
	}
	for _, pass := range []func(context.Context, statsQuery, string, []any, *storage.StatsResult) error{
		func(ctx context.Context, q statsQuery, where string, args []any, r *storage.StatsResult) error {
			return s.totalsFromCells(ctx, phase, where, args, r)
		},
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
		func(_ context.Context, _ statsQuery, _ string, _ []any, r *storage.StatsResult) error {
			cubeFromCells(cube, r)
			return nil
		},
		s.histogramFromCells,
		func(ctx context.Context, q statsQuery, _ string, _ []any, r *storage.StatsResult) error {
			return s.topBlundersFromCells(ctx, q, phase, r)
		},
		s.rollingFromCells,
		func(_ context.Context, _ statsQuery, _ string, _ []any, r *storage.StatsResult) error {
			mwcFromCells(phase, cube, r)
			return nil
		},
		func(ctx context.Context, q statsQuery, where string, args []any, r *storage.StatsResult) error {
			return s.breakdownsFromCells(ctx, phase, where, args, r)
		},
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

// phaseCell is one phase cell of the selection, with its match's tournament
// (0 for none).
type phaseCell struct {
	match, tournament                 int64
	decisionType, k1                  int
	decisions, errorMP, maxErrorMP    int64
	blunders, mwcDecisions, positions int64
	mwcLoss                           float64
}

// readPhaseCells reads the selection's phase cells, in key order: the read
// sorts nothing.
func (s *StatsStore) readPhaseCells(ctx context.Context, where string, args []any) ([]phaseCell, error) {
	var out []phaseCell
	err := scanEach(ctx, s.DB,
		`SELECT c.match_id, COALESCE(m.tournament_id, 0), c.decision_type, c.k1, c.decisions, c.error_mp, c.max_error_mp,
			c.blunders, c.mwc_decisions, c.positions, c.mwc_loss`+cellsJoin+cellKind(where, cellPhase),
		args, func(r Rows) error {
			var c phaseCell
			if err := r.Scan(&c.match, &c.tournament, &c.decisionType, &c.k1, &c.decisions, &c.errorMP, &c.maxErrorMP,
				&c.blunders, &c.mwcDecisions, &c.positions, &c.mwcLoss); err != nil {
				return err
			}
			out = append(out, c)
			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("phase cells: %w", err)
	}
	return out, nil
}

// cubeCell is the selection's cube cells of one best and played action.
type cubeCell struct {
	k1                           int
	best, played                 string
	errorMP, decisions, blunders int64
	mwcLoss                      float64
}

// readCubeCells sums the selection's cube cells by best and played action,
// in the order of the best action's code.
func (s *StatsStore) readCubeCells(ctx context.Context, where string, args []any) ([]cubeCell, error) {
	d := s.DB
	var out []cubeCell
	err := scanEach(ctx, d,
		`SELECT c.k1, `+ActionLabelOrEmptyFor(d, "c.k1")+`, `+ActionLabelOrEmptyFor(d, "c.k2")+`, `+d.Bigint(`SUM(c.error_mp)`)+`, `+
			d.Bigint(`SUM(c.decisions)`)+`, `+d.Bigint(`SUM(c.blunders)`)+`, SUM(c.mwc_loss)`+cellsJoin+cellKind(where, cellCube)+
			` GROUP BY c.k1, c.k2, 2, 3 ORDER BY c.k1, c.k2`,
		args, func(r Rows) error {
			var c cubeCell
			if err := r.Scan(&c.k1, &c.best, &c.played, &c.errorMP, &c.decisions, &c.blunders, &c.mwcLoss); err != nil {
				return err
			}
			out = append(out, c)
			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("cube cells: %w", err)
	}
	return out, nil
}

func (s *StatsStore) totalsFromCells(ctx context.Context, phase []phaseCell, where string, args []any, result *storage.StatsResult) error {
	var decisions, private int64
	matches, tournaments := map[int64]bool{}, map[int64]bool{}
	for _, c := range phase {
		decisions += c.decisions
		private += c.positions
		matches[c.match] = true
		if c.tournament != 0 {
			tournaments[c.tournament] = true
		}
	}
	result.Totals.NumDecisions = int(decisions)
	result.Totals.NumMatches = len(matches)
	result.Totals.NumTournaments = len(tournaments)
	// A private position is in one seat's phase cell and nowhere else; a
	// shared one has a row for each seat reaching it (sharePositions).
	var shared int
	if err := s.DB.QueryRow(ctx, `SELECT COUNT(DISTINCT c.position_id)`+positionsJoin+where, args...).Scan(&shared); err != nil {
		return fmt.Errorf("totals positions (cells): %w", err)
	}
	result.Totals.NumPositions = int(private) + shared
	return nil
}

// cubeFromCells is computeCubeActionBreakdown and computeCubeDirections.
func cubeFromCells(cube []cubeCell, result *storage.StatsResult) {
	var cells []storage.CubeDirectionRow
	var sumErr []int64
	for i, c := range cube {
		cells = append(cells, storage.CubeDirectionRow{Best: c.best, Played: c.played, Count: int(c.decisions), ErrorMP: c.errorMP})
		if i == 0 || cube[i-1].k1 != c.k1 {
			result.CubeActionBreakdown = append(result.CubeActionBreakdown, storage.CubeActionStats{Action: c.best})
			sumErr = append(sumErr, 0)
		}
		last := len(result.CubeActionBreakdown) - 1
		cs := &result.CubeActionBreakdown[last]
		cs.NumDecisions += int(c.decisions)
		cs.BlunderCount += int(c.blunders)
		sumErr[last] += c.errorMP
	}
	for i := range result.CubeActionBreakdown {
		cs := &result.CubeActionBreakdown[i]
		cs.PR = pr(sumErr[i], cs.NumDecisions)
	}
	result.CubeDirections = storage.TallyCubeDirections(cells)
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
func topBlunderMatches(phase []phaseCell) []int64 {
	var bound int64
	if len(phase) >= 10 {
		largest := make([]int64, len(phase))
		for i, c := range phase {
			largest[i] = c.maxErrorMP
		}
		slices.SortFunc(largest, func(a, b int64) int { return cmp.Compare(b, a) })
		bound = largest[9]
	}
	var ids []int64
	seen := map[int64]bool{}
	for _, c := range phase {
		if (len(phase) < 10 || c.maxErrorMP >= bound) && !seen[c.match] {
			seen[c.match] = true
			ids = append(ids, c.match)
		}
	}
	return ids
}

// topBlundersFromCells is computeTopBlunders over the matches that can hold
// the result, and their MWC losses from computeMWCPass over the same.
func (s *StatsStore) topBlundersFromCells(ctx context.Context, q statsQuery, phase []phaseCell, result *storage.StatsResult) error {
	rq := restrictTo(q, topBlunderMatches(phase))
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
// per-match and per-cube-action rows, as computeMWCPass does. A match's
// losses of one decision type count only when the table values one of its
// decisions, as the direct pass skips a decision it cannot value.
func mwcFromCells(phase []phaseCell, cube []cubeCell, result *storage.StatsResult) {
	type key struct {
		match int64
		dt    int
	}
	type group struct {
		tournament int64
		loss       float64
		n          int64
	}
	groups := map[key]*group{}
	var keys []key
	for _, c := range phase {
		k := key{c.match, c.decisionType}
		g := groups[k]
		if g == nil {
			g = &group{tournament: c.tournament}
			groups[k] = g
			keys = append(keys, k)
		}
		g.loss += c.mwcLoss
		g.n += c.mwcDecisions
	}
	slices.SortFunc(keys, func(a, b key) int {
		return cmp.Or(cmp.Compare(a.match, b.match), cmp.Compare(a.dt, b.dt))
	})
	byMatch := map[int64]float64{}
	byTournament := map[int64]float64{}
	for _, k := range keys {
		g := groups[k]
		if g.n == 0 {
			continue
		}
		result.MWCAvailable = true
		result.MWCGlobal += g.loss
		if k.dt == 0 {
			result.MWCChecker += g.loss
		} else {
			result.MWCCube += g.loss
		}
		byMatch[k.match] += g.loss
		if g.tournament != 0 {
			byTournament[g.tournament] += g.loss
		}
	}
	byAction := map[string]float64{}
	for _, c := range cube {
		byAction[c.best] += c.mwcLoss
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
}

// breakdownsFromCells is computePerPhase, computePerGameType and computePerScore.
func (s *StatsStore) breakdownsFromCells(ctx context.Context, phase []phaseCell, where string, args []any, result *storage.StatsResult) error {
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
	byPhase := map[int]*tally{}
	for _, c := range phase {
		t := byPhase[c.k1]
		if t == nil {
			t = &tally{}
			byPhase[c.k1] = t
		}
		t.sumErr += c.errorMP
		t.n += c.decisions
		t.blunders += c.blunders
	}
	// Grouped by the stored value, NULL (cellNull) apart from 0, and shown
	// with NULL read as 0, as the direct pass's COALESCE does.
	for _, k := range slices.Sorted(maps.Keys(byPhase)) {
		t := byPhase[k]
		shown := k
		if k == cellNull {
			shown = 0
		}
		result.PerPhase = append(result.PerPhase, storage.PhaseStats{
			Phase: domain.GamePhase(shown).String(), PR: pr(t.sumErr, int(t.n)), NumDecisions: int(t.n), BlunderCount: int(t.blunders),
		})
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
