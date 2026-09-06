package sqlshared

// stats_compute.go — StatsStore.Compute, one pass per section.
//
// Compute was a single 438-line function (the tallest in the tree, and the one
// that set .golangci.yml's funlen ceiling). It was never one computation: it is
// eleven SQL passes that each fill a different part of storage.StatsResult, laid
// end to end and separated by comment banners. Splitting it along those banners
// changes no SQL and no arithmetic — the passes run in the same order, against
// the same WHERE clause — and gives each one a name, a doc comment and its own
// local variables, where they used to share `rows`, `err` and `scanErr` across
// four hundred lines (B.15, #183).
//
// The order is not incidental: computeMWCPass reads the per-tournament,
// per-match, per-cube-action and top-blunder rows the passes before it
// collected, and back-fills their MWC. It must run last.

import (
	"context"
	"fmt"
	"math"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// statsQuery is what every pass needs and none of them changes: the tenant
// scope and the caller's filter, plus the WHERE clause and arguments derived
// from them once, in Compute.
type statsQuery struct {
	scope    string
	filter   storage.StatsFilter
	whereSQL string
	baseArgs []any
}

// Compute runs each statistics pass in turn and returns the assembled result.
// The passes are in stats_compute.go, one per section; see the file header.
func (s *StatsStore) Compute(ctx context.Context, scope string, filter storage.StatsFilter) (*storage.StatsResult, error) {
	whereSQL, baseArgs := s.buildStatsWhereClause(scope, filter)
	q := statsQuery{scope: scope, filter: filter, whereSQL: whereSQL, baseArgs: baseArgs}

	result := &storage.StatsResult{PRRolling: make(map[int]float64)}

	for _, pass := range []func(context.Context, statsQuery, *storage.StatsResult) error{
		s.computeTotals,
		s.computePRByDecisionType,
		s.computeSnowieGlobal,
		s.computePerTournament,
		s.computePerMatch,
		s.computeCubeActionBreakdown,
		s.computeCubeDirections,
		s.computeErrorHistogram,
		s.computeTopBlunders,
		s.computeRollingPR,
		s.computeMWCPass,
		// The three breakdowns of #266, last: each reuses the same counted
		// predicate and error column as the passes above, so none of them can
		// disagree with the global figures they slice.
		s.computePerPhase,
		s.computePerGameType,
		s.computePerScore,
		s.computePerTag,
	} {
		if err := pass(ctx, q, result); err != nil {
			return nil, err
		}
	}
	return result, nil
}

// computeTotals counts the rows the filter selects: positions, matches, tournaments and
// decisions.
func (s *StatsStore) computeTotals(ctx context.Context, q statsQuery, result *storage.StatsResult) error {
	row := s.DB.QueryRow(ctx,
		`SELECT COUNT(DISTINCT p.id), COUNT(DISTINCT m.id), COUNT(DISTINCT m.tournament_id), COUNT(*) `+
			statsBaseJoin+q.whereSQL,
		q.baseArgs...,
	)
	if err := row.Scan(
		&result.Totals.NumPositions,
		&result.Totals.NumMatches,
		&result.Totals.NumTournaments,
		&result.Totals.NumDecisions,
	); err != nil {
		return fmt.Errorf("totals query: %w", err)
	}
	return nil
}

// computePRByDecisionType fills PRChecker, PRCube and PRGlobal from one grouped pass.
func (s *StatsStore) computePRByDecisionType(ctx context.Context, q statsQuery, result *storage.StatsResult) error {
	d := s.DB
	rows, err := s.DB.Query(ctx,
		`SELECT p.decision_type, `+d.Bigint(`SUM(`+statsErrExpr+`)`)+`, COUNT(*) `+
			statsBaseJoin+q.whereSQL+
			` GROUP BY p.decision_type`,
		q.baseArgs...,
	)
	if err != nil {
		return fmt.Errorf("PR by decision_type query: %w", err)
	}
	var totalErrSum int64
	var totalErrCount int
	var scanErr error
	func() {
		defer rows.Close()
		for rows.Next() {
			var dt int
			var sumErr int64
			var cnt int
			if err2 := rows.Scan(&dt, &sumErr, &cnt); err2 != nil {
				scanErr = err2
				return
			}
			totalErrSum += sumErr
			totalErrCount += cnt
			switch dt {
			case 0:
				result.PRChecker = pr(sumErr, cnt)
			case 1:
				result.PRCube = pr(sumErr, cnt)
			}
		}
	}()
	if scanErr != nil {
		return fmt.Errorf("PR by decision_type scan: %w", scanErr)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("PR by decision_type rows: %w", err)
	}
	result.PRGlobal = pr(totalErrSum, totalErrCount)
	return nil
}

// computeSnowieGlobal fills SnowieGlobal.
//
// The Snowie rate divides ONE player's errors by BOTH players' checker moves
// (gnuBG formatgs.c:415-424, anTotalMoves[0] + anTotalMoves[1]), so numerator
// and denominator cannot share a WHERE clause: the player filter must narrow
// the errors to the decisions that player took, and must NOT narrow the move
// count to that player's own seat — it only restricts the matches counted.
// Reusing one clause for both halved the denominator and doubled every
// filtered Snowie ER; MatchDetail already got this right.
func (s *StatsStore) computeSnowieGlobal(ctx context.Context, q statsQuery, result *storage.StatsResult) error {
	d := s.DB
	// The Snowie rate divides ONE player's errors by BOTH players' checker
	// moves (gnuBG formatgs.c:415-424, anTotalMoves[0] + anTotalMoves[1]), so
	// numerator and denominator cannot share a WHERE clause: the player q.filter
	// must narrow the errors to the decisions that player took, and must NOT
	// narrow the move count to that player's own seat — it only restricts the
	// matches counted. Reusing one clause for both halved the denominator and
	// doubled every filtered Snowie ER; MatchDetail already got this right.
	{
		snowieFilter := q.filter
		snowieFilter.DecisionType = -1 // count all decision types
		numWhere, numArgs := s.buildBaseWhereClause(q.scope, snowieFilter)
		var snowieSumErr int64
		if err := s.DB.QueryRow(ctx,
			`SELECT `+d.Bigint(`COALESCE(SUM(`+statsErrExpr+`),0)`)+` `+statsBaseJoin+numWhere,
			numArgs...,
		).Scan(&snowieSumErr); err != nil {
			return fmt.Errorf("snowie ER (global) numerator: %w", err)
		}

		denWhere, denArgs := s.buildBaseWhereClauseSeat(q.scope, snowieFilter, false)
		var snowieCheckerCnt int
		if err := s.DB.QueryRow(ctx,
			`SELECT `+d.Bigint(`COALESCE(SUM(CASE WHEN p.decision_type=0 THEN 1 ELSE 0 END),0)`)+` `+
				statsBaseJoin+denWhere,
			denArgs...,
		).Scan(&snowieCheckerCnt); err != nil {
			return fmt.Errorf("snowie ER (global) denominator: %w", err)
		}

		result.SnowieGlobal = snowieER(snowieSumErr, snowieCheckerCnt)
	}
	return nil
}

// computePerTournament fills PerTournament.
//
// tournament.date is TEXT (a date string) in both schemas, unlike
// match.match_date — use it verbatim. The GROUP BY names every selected
// tournament column: PostgreSQL requires it, SQLite accepts it.
func (s *StatsStore) computePerTournament(ctx context.Context, q statsQuery, result *storage.StatsResult) error {
	d := s.DB
	var scanErr error
	// tournament.date is TEXT (a date string) in both schemas, unlike
	// match.match_date — use it verbatim. The GROUP BY names every selected
	// tournament column: PostgreSQL requires it, SQLite accepts it.
	rows, err := s.DB.Query(ctx,
		`SELECT m.tournament_id, COALESCE(t.name,''), COALESCE(t.date,''), `+d.Bigint(`SUM(`+statsErrExpr+`)`)+`, COUNT(*) `+
			statsBaseJoin+q.whereSQL+
			` AND m.tournament_id IS NOT NULL`+
			` GROUP BY m.tournament_id, t.name, t.date, t.created_at ORDER BY t.date, t.created_at`,
		q.baseArgs...,
	)
	if err != nil {
		return fmt.Errorf("PR per tournament query: %w", err)
	}
	func() {
		defer rows.Close()
		for rows.Next() {
			var ts storage.TournamentStats
			var sumErr int64
			var cnt int
			if err2 := rows.Scan(&ts.ID, &ts.Name, &ts.Date, &sumErr, &cnt); err2 != nil {
				scanErr = err2
				return
			}
			ts.NumDecisions = cnt
			ts.PR = pr(sumErr, cnt)
			result.PerTournament = append(result.PerTournament, ts)
		}
	}()
	if scanErr != nil {
		return fmt.Errorf("PR per tournament scan: %w", scanErr)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("PR per tournament rows: %w", err)
	}
	return nil
}

// computePerMatch fills PerMatch.
func (s *StatsStore) computePerMatch(ctx context.Context, q statsQuery, result *storage.StatsResult) error {
	d := s.DB
	var scanErr error
	rows, err := s.DB.Query(ctx,
		`SELECT m.id, `+d.DateText("m.match_date")+`, `+d.Bigint(`SUM(`+statsErrExpr+`)`)+`, COUNT(*) `+
			statsBaseJoin+q.whereSQL+
			` GROUP BY m.id, m.match_date ORDER BY m.match_date`,
		q.baseArgs...,
	)
	if err != nil {
		return fmt.Errorf("PR per match query: %w", err)
	}
	func() {
		defer rows.Close()
		for rows.Next() {
			var ms storage.MatchStats
			var sumErr int64
			var cnt int
			if err2 := rows.Scan(&ms.ID, &ms.Date, &sumErr, &cnt); err2 != nil {
				scanErr = err2
				return
			}
			ms.NumDecisions = cnt
			ms.PR = pr(sumErr, cnt)
			result.PerMatch = append(result.PerMatch, ms)
		}
	}()
	if scanErr != nil {
		return fmt.Errorf("PR per match scan: %w", scanErr)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("PR per match rows: %w", err)
	}
	return nil
}

// computeCubeActionBreakdown fills CubeActionBreakdown.
func (s *StatsStore) computeCubeActionBreakdown(ctx context.Context, q statsQuery, result *storage.StatsResult) error {
	d := s.DB
	cubeWhere := q.whereSQL + " AND p.decision_type = 1"
	rows, err := s.DB.Query(ctx,
		`SELECT COALESCE(a.best_cube_action,''), `+d.Bigint(`SUM(a.cube_error)`)+`, COUNT(*),`+
			` `+d.Bigint(`SUM(CASE WHEN a.cube_error >= ? THEN 1 ELSE 0 END)`)+` `+
			statsBaseJoin+cubeWhere+
			` GROUP BY a.best_cube_action`,
		append([]any{blunderThresholdMP}, q.baseArgs...)...,
	)
	if err != nil {
		return fmt.Errorf("cube action breakdown query: %w", err)
	}
	var scanErr error
	func() {
		defer rows.Close()
		for rows.Next() {
			var cs storage.CubeActionStats
			var sumErr int64
			if err2 := rows.Scan(&cs.Action, &sumErr, &cs.NumDecisions, &cs.BlunderCount); err2 != nil {
				scanErr = err2
				return
			}
			cs.PR = pr(sumErr, cs.NumDecisions)
			result.CubeActionBreakdown = append(result.CubeActionBreakdown, cs)
		}
	}()
	if scanErr != nil {
		return fmt.Errorf("cube action breakdown scan: %w", scanErr)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("cube action breakdown rows: %w", err)
	}
	return nil
}

// computeCubeDirections fills CubeDirections: the same scope as the breakdown above, crossed with
// the action actually played. The labels are interpreted in Go
// (storage.TallyCubeDirections), never in SQL: their spellings vary by
// importer and the recognition is stated in exactly one place — see
// kevung/blunderDB#115.
func (s *StatsStore) computeCubeDirections(ctx context.Context, q statsQuery, result *storage.StatsResult) error {
	d := s.DB
	// Same q.scope as 5, crossed with the action actually played. The labels are
	// interpreted in Go (storage.TallyCubeDirections), never in SQL: their
	// spellings vary by importer and the recognition is stated in exactly one
	// place — see kevung/blunderDB#115.
	{
		cubeWhere := q.whereSQL + " AND p.decision_type = 1"
		rows, err := s.DB.Query(ctx,
			`SELECT COALESCE(a.best_cube_action,''), COALESCE(mv.cube_action,''), COUNT(*),`+
				` `+d.Bigint(`COALESCE(SUM(a.cube_error),0)`)+` `+
				statsBaseJoin+cubeWhere+
				` GROUP BY a.best_cube_action, mv.cube_action`,
			q.baseArgs...,
		)
		if err != nil {
			return fmt.Errorf("cube direction query: %w", err)
		}
		var cells []storage.CubeDirectionRow
		var scanErr error
		func() {
			defer rows.Close()
			for rows.Next() {
				var c storage.CubeDirectionRow
				if err2 := rows.Scan(&c.Best, &c.Played, &c.Count, &c.ErrorMP); err2 != nil {
					scanErr = err2
					return
				}
				cells = append(cells, c)
			}
		}()
		if scanErr != nil {
			return fmt.Errorf("cube direction scan: %w", scanErr)
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("cube direction rows: %w", err)
		}
		result.CubeDirections = storage.TallyCubeDirections(cells)
	}
	return nil
}

// computeErrorHistogram fills ErrorHistogram.
func (s *StatsStore) computeErrorHistogram(ctx context.Context, q statsQuery, result *storage.StatsResult) error {
	var scanErr error
	histogramSQL := `SELECT
		CASE
			WHEN (` + statsErrExpr + `) < 5   THEN 0
			WHEN (` + statsErrExpr + `) < 10  THEN 5
			WHEN (` + statsErrExpr + `) < 25  THEN 10
			WHEN (` + statsErrExpr + `) < 50  THEN 25
			WHEN (` + statsErrExpr + `) < 100 THEN 50
			ELSE 100
		END as bucket,
		COUNT(*) ` +
		statsBaseJoin + q.whereSQL +
		` GROUP BY bucket ORDER BY bucket`

	rows, err := s.DB.Query(ctx, histogramSQL, q.baseArgs...)
	if err != nil {
		return fmt.Errorf("error histogram query: %w", err)
	}
	bucketMaxMap := map[int]int{0: 5, 5: 10, 10: 25, 25: 50, 50: 100, 100: -1}
	func() {
		defer rows.Close()
		for rows.Next() {
			var bucketMin, cnt int
			if err2 := rows.Scan(&bucketMin, &cnt); err2 != nil {
				scanErr = err2
				return
			}
			result.ErrorHistogram = append(result.ErrorHistogram, storage.ErrorBucket{
				MinMP: bucketMin,
				MaxMP: bucketMaxMap[bucketMin],
				Count: cnt,
			})
		}
	}()
	if scanErr != nil {
		return fmt.Errorf("error histogram scan: %w", scanErr)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error histogram rows: %w", err)
	}
	return nil
}

// computeTopBlunders fills TopBlunders.
func (s *StatsStore) computeTopBlunders(ctx context.Context, q statsQuery, result *storage.StatsResult) error {
	d := s.DB
	var scanErr error
	rows, err := s.DB.Query(ctx,
		`SELECT p.id, m.id, COALESCE(m.tournament_id, 0), (`+statsErrExpr+`) as emg,`+
			` p.decision_type,`+
			` `+d.DateText("m.match_date")+` as match_date,`+
			` COALESCE(m.player1_name, '') || ' vs ' || COALESCE(m.player2_name, '') as player_names `+
			statsBaseJoin+q.whereSQL+
			` ORDER BY emg DESC LIMIT 10`,
		q.baseArgs...,
	)
	if err != nil {
		return fmt.Errorf("top blunders query: %w", err)
	}
	func() {
		defer rows.Close()
		for rows.Next() {
			var be storage.BlunderEntry
			if err2 := rows.Scan(&be.PositionID, &be.MatchID, &be.TournamentID, &be.ErrorMP,
				&be.DecisionType, &be.MatchDate, &be.PlayerNames); err2 != nil {
				scanErr = err2
				return
			}
			result.TopBlunders = append(result.TopBlunders, be)
		}
	}()
	if scanErr != nil {
		return fmt.Errorf("top blunders scan: %w", scanErr)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("top blunders rows: %w", err)
	}
	return nil
}

// computeRollingPR fills PRRolling.
func (s *StatsStore) computeRollingPR(ctx context.Context, q statsQuery, result *storage.StatsResult) error {
	var scanErr error
	rollingNs := []int{5, 10, 50, 100, 250, 500, 1000}
	maxN := rollingNs[len(rollingNs)-1]

	recentRows, err := s.DB.Query(ctx,
		`SELECT (`+statsErrExpr+`) as err `+
			statsBaseJoin+q.whereSQL+
			` ORDER BY m.match_date DESC, mv.move_number DESC LIMIT ?`,
		append(q.baseArgs, maxN)...,
	)
	if err != nil {
		return fmt.Errorf("rolling PR query: %w", err)
	}
	var recentErrors []int64
	func() {
		defer recentRows.Close()
		for recentRows.Next() {
			var e int64
			if err2 := recentRows.Scan(&e); err2 != nil {
				scanErr = err2
				return
			}
			recentErrors = append(recentErrors, e)
		}
	}()
	if scanErr != nil {
		return fmt.Errorf("rolling PR scan: %w", scanErr)
	}
	if err := recentRows.Err(); err != nil {
		return fmt.Errorf("rolling PR rows: %w", err)
	}

	var cumSum int64
	for i, e := range recentErrors {
		cumSum += e
		n := i + 1
		for _, threshold := range rollingNs {
			if n == threshold {
				result.PRRolling[threshold] = pr(cumSum, n)
			}
		}
	}
	return nil
}

// computeMWCPass fills every MWC field, and back-fills the MWC of the per-tournament,
// per-match, per-cube-action and top-blunder rows the passes above collected.
//
// It streams per-row data in most-recent-first order and aggregates the MWC
// losses in Go: one supplementary SQL pass, O(n_decisions). It must therefore
// run last.
func (s *StatsStore) computeMWCPass(ctx context.Context, q statsQuery, result *storage.StatsResult) error {
	// Stream per-row data in most-recent-first order and aggregate MWC losses in
	// Go. One supplementary SQL pass; O(n_decisions).
	{
		mwcPassSQL := `SELECT ` + statsErrExpr + ` as err,` +
			` COALESCE(p.score_1, 0), COALESCE(p.score_2, 0), mv.player,` +
			` ` + cubeMultiplierExpr + `, COALESCE(p.match_length, m.match_length, 0),` +
			` COALESCE(m.tournament_id, 0), m.id,` +
			` COALESCE(a.best_cube_action, ''), p.decision_type, p.id ` +
			statsBaseJoin + q.whereSQL +
			` ORDER BY m.match_date DESC, mv.move_number DESC`

		mwcRows, mwcErr := s.DB.Query(ctx, mwcPassSQL, q.baseArgs...)
		if mwcErr != nil {
			return fmt.Errorf("MWC pass query: %w", mwcErr)
		}

		mwcByTournament := make(map[int64]float64)
		mwcByMatch := make(map[int64]float64)
		mwcByCubeAction := make(map[string]float64)
		blunderMWC := make(map[int64]float64)

		var mwcGlobal, mwcChecker, mwcCube float64
		var mwcAvailable bool
		var rowIdx int
		var mwcRollingCum float64
		mwcRollingThresholds := []int{5, 10, 50, 100, 250, 500, 1000}
		mwcRollingMap := make(map[int]float64)

		var scanErr error
		func() {
			defer mwcRows.Close()
			for mwcRows.Next() {
				var errMP int64
				var awayScore0, awayScore1, rawPlayer, cubeValue, matchLength int
				var tournamentID, matchID int64
				var cubeAction string
				var dt int
				var posID int64
				if err2 := mwcRows.Scan(&errMP, &awayScore0, &awayScore1, &rawPlayer, &cubeValue, &matchLength,
					&tournamentID, &matchID, &cubeAction, &dt, &posID); err2 != nil {
					scanErr = err2
					return
				}

				rowIdx++

				// XG encodes player 0 (bottom) as 1 and player 1 (top) as -1;
				// gnuBG fMove is 0 or 1.
				fMove := 0
				if rawPlayer == -1 {
					fMove = 1
				}
				// p.score_1/score_2 are away scores; ConvertEMGLossToMWCLoss
				// expects current scores (games already won).
				currentScore0 := matchLength - awayScore0
				currentScore1 := matchLength - awayScore1

				mwcLoss := engine.ConvertEMGLossToMWCLoss(int(errMP), currentScore0, currentScore1, fMove, cubeValue, matchLength)

				if !math.IsNaN(mwcLoss) {
					mwcAvailable = true
					mwcGlobal += mwcLoss
					if dt == 0 {
						mwcChecker += mwcLoss
					} else {
						mwcCube += mwcLoss
					}
					if tournamentID != 0 {
						mwcByTournament[tournamentID] += mwcLoss
					}
					mwcByMatch[matchID] += mwcLoss
					if dt == 1 {
						mwcByCubeAction[cubeAction] += mwcLoss
					}
					blunderMWC[posID] = mwcLoss
					mwcRollingCum += mwcLoss
				}

				for _, threshold := range mwcRollingThresholds {
					if rowIdx == threshold {
						mwcRollingMap[threshold] = mwcRollingCum
					}
				}
			}
		}()
		if scanErr != nil {
			return fmt.Errorf("MWC pass scan: %w", scanErr)
		}
		if err := mwcRows.Err(); err != nil {
			return fmt.Errorf("MWC pass rows: %w", err)
		}

		result.MWCGlobal = mwcGlobal
		result.MWCChecker = mwcChecker
		result.MWCCube = mwcCube
		result.MWCAvailable = mwcAvailable
		result.MWCRolling = mwcRollingMap

		for i, ts := range result.PerTournament {
			result.PerTournament[i].MWC = mwcByTournament[ts.ID]
		}
		for i, ms := range result.PerMatch {
			result.PerMatch[i].MWC = mwcByMatch[ms.ID]
		}
		for i, cs := range result.CubeActionBreakdown {
			result.CubeActionBreakdown[i].MWC = mwcByCubeAction[cs.Action]
		}
		for i, be := range result.TopBlunders {
			if loss, ok := blunderMWC[be.PositionID]; ok {
				result.TopBlunders[i].MWCLoss = loss
			}
		}
	}
	return nil
}
