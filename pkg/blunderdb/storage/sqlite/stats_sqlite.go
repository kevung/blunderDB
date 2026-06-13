package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// statsStore implements storage.StatsStore over SQLite. The SQL is the same
// aggregate logic that historically lived on the Database wrapper
// (database/db_stats.go); the bodies are ported with the receiver's *sql.DB
// swapped for the store's execer and the legacy DTOs swapped for the storage.*
// equivalents. scope is ignored — the SQLite schema has no tenant column
// (per-tenant isolation is a PostgreSQL concern).
type statsStore struct{ db execer }

var _ storage.StatsStore = (*statsStore)(nil)

// statsErrExpr is defined in search_sqlite.go (shared) and reused here:
//   CASE WHEN p.decision_type = 1 THEN a.cube_error ELSE a.best_move_equity_error END

// blunderThresholdMP is the error threshold (in stored millipoints units) above
// which a decision is counted as a blunder. 100 ≈ 0.1 EMG.
const blunderThresholdMP = 100

// statsCountedExpr is the SQL predicate selecting the decisions that count
// toward PR and decision tallies (XG semantics):
//   - Checker: unforced positions only (a.is_forced = 0).
//   - Cube: a position is counted when either (a) it is a "close" decision per
//     gnuBG isCloseCubedecision (a.is_close_cube = 1) with the exclusion below,
//     OR (b) the player took an active cube action other than NoDouble (Double,
//     Take, Pass). The OR ensures premature doublings — wrong doubles with a
//     large equity gap that are technically "not close" — are still counted.
//
// Exclusion for close NoDoubles: XG's stored EMG equities at extreme match
// scores (mover's away ≤ 2 with centered cube) are amplified by a factor of
// ~3–4× beyond the normal [-1, 1] range, so gnuBG's 0.16 threshold falsely
// marks these positions as "close". We exclude correctly-played (cube_error=0)
// NoDouble positions where the cube is still centered (cube_value=0) and the
// mover needs ≤ 2 points to win, mirroring XG's actual decision counting.
const statsCountedExpr = "((p.decision_type = 0 AND a.is_forced = 0) OR (p.decision_type = 1 AND (COALESCE(mv.cube_action, '') NOT IN ('', 'No Double', 'NoDouble') OR (a.is_close_cube = 1 AND NOT (COALESCE(a.cube_error, 0) = 0 AND COALESCE(p.cube_value, 0) = 0 AND CASE WHEN mv.player = 1 THEN COALESCE(p.score_1, 99) ELSE COALESCE(p.score_2, 99) END <= 2)))))"

// statsBaseJoin is the FROM + JOIN fragment shared by all stats queries.
const statsBaseJoin = `FROM position p
JOIN analysis a ON a.position_id = p.id
JOIN move mv ON mv.position_id = p.id
JOIN game g ON g.id = mv.game_id
JOIN match m ON m.id = g.match_id
LEFT JOIN tournament t ON t.id = m.tournament_id`

// pr computes the Performance Rating from a sum of errors (millipoints stored
// units) and the number of decisions. Formula: 500 × sumErrMP / 1000 / nDecisions.
func pr(sumErrMP int64, nDecisions int) float64 {
	if nDecisions == 0 {
		return 0
	}
	return 500 * float64(sumErrMP) / 1000 / float64(nDecisions)
}

// snowieER computes the Snowie Error Rate from a sum of errors (millipoints
// stored units) and the total checker move count for both players combined.
func snowieER(sumErrMP int64, nMovesBoth int) float64 {
	if nMovesBoth == 0 {
		return 0
	}
	return 500 * float64(sumErrMP) / 1000 / float64(nMovesBoth)
}

// buildBaseWhereClause constructs the base WHERE clause for the given filter
// without the statsCountedExpr predicate.
func buildBaseWhereClause(filter storage.StatsFilter) (whereSQL string, args []any) {
	var clauses []string

	if filter.PlayerName != "" {
		clauses = append(clauses, "((m.player1_name = ? AND mv.player = 1) OR (m.player2_name = ? AND mv.player = -1))")
		args = append(args, filter.PlayerName, filter.PlayerName)
	}

	if len(filter.TournamentIDs) > 0 {
		placeholders := strings.Repeat("?,", len(filter.TournamentIDs))
		placeholders = placeholders[:len(placeholders)-1]
		clauses = append(clauses, "m.tournament_id IN ("+placeholders+")")
		for _, id := range filter.TournamentIDs {
			args = append(args, id)
		}
	}

	if filter.DateFrom != "" && filter.DateTo != "" {
		clauses = append(clauses, "m.match_date BETWEEN ? AND ?")
		args = append(args, filter.DateFrom, filter.DateTo)
	} else if filter.DateFrom != "" {
		clauses = append(clauses, "m.match_date >= ?")
		args = append(args, filter.DateFrom)
	} else if filter.DateTo != "" {
		clauses = append(clauses, "m.match_date <= ?")
		args = append(args, filter.DateTo)
	}

	if filter.DecisionType >= 0 {
		clauses = append(clauses, "p.decision_type = ?")
		args = append(args, filter.DecisionType)
	}

	if len(filter.MatchLength) > 0 {
		placeholders := strings.Repeat("?,", len(filter.MatchLength))
		placeholders = placeholders[:len(placeholders)-1]
		clauses = append(clauses, "m.match_length IN ("+placeholders+")")
		for _, ml := range filter.MatchLength {
			args = append(args, ml)
		}
	}

	clauses = append(clauses, "a.position_id IS NOT NULL")
	clauses = append(clauses, "("+statsErrExpr+") IS NOT NULL")

	whereSQL = " WHERE " + strings.Join(clauses, " AND ")
	return whereSQL, args
}

// buildStatsWhereClause wraps buildBaseWhereClause and appends the
// statsCountedExpr predicate (XG/gnuBG semantics).
func buildStatsWhereClause(filter storage.StatsFilter) (whereSQL string, args []any) {
	whereSQL, args = buildBaseWhereClause(filter)
	whereSQL += " AND " + statsCountedExpr
	return whereSQL, args
}

// buildSelectionWhereClause produces the extra WHERE fragment and optional
// ORDER BY / LIMIT fragment for a given SelectionSpec.
func buildSelectionWhereClause(sel storage.SelectionSpec) (whereAdd string, orderLimit string, args []any) {
	switch sel.Kind {
	case "checker":
		whereAdd = " AND p.decision_type = 0"
		if sel.OnlyWithError {
			whereAdd += " AND (" + statsErrExpr + ") > 0"
		}
	case "cube":
		whereAdd = " AND p.decision_type = 1"
		if sel.OnlyWithError {
			whereAdd += " AND (" + statsErrExpr + ") > 0"
		}
	case "cube_action":
		whereAdd = " AND p.decision_type = 1 AND a.best_cube_action = ?"
		args = append(args, sel.CubeAction)
		if sel.OnlyWithError {
			whereAdd += " AND (" + statsErrExpr + ") > 0"
		}
	case "error_bucket":
		whereAdd = " AND (" + statsErrExpr + ") >= ?"
		args = append(args, sel.BucketMinMP)
		if sel.BucketMaxMP != -1 {
			whereAdd += " AND (" + statsErrExpr + ") < ?"
			args = append(args, sel.BucketMaxMP)
		}
	case "tournament":
		whereAdd = " AND m.tournament_id = ?"
		args = append(args, sel.TournamentID)
	case "match":
		whereAdd = " AND m.id = ?"
		args = append(args, sel.MatchID)
	case "last_n":
		orderLimit = "ORDER BY m.match_date DESC, mv.move_number DESC LIMIT ?"
		args = append(args, sel.LastN)
	case "position":
		whereAdd = " AND p.id = ?"
		args = append(args, sel.PositionID)
	case "top_blunders":
		limit := 10
		if sel.LastN > 0 {
			limit = sel.LastN
		}
		orderLimit = "ORDER BY (" + statsErrExpr + ") DESC LIMIT ?"
		args = append(args, limit)
		// "all" → no extra clauses
	}
	return whereAdd, orderLimit, args
}

// scanPositionIDs scans a single-int64-column result set into a slice, closing
// rows and propagating any iteration error.
func scanPositionIDs(rows *sql.Rows) ([]int64, error) {
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan position id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// DateRange returns the minimum and maximum match dates present in the
// database. Both fields are empty when no matches with a date exist.
func (s *statsStore) DateRange(ctx context.Context, scope string) (storage.StatsDateRange, error) {
	var min, max string
	if err := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(MIN(SUBSTR(match_date,1,10)),''), COALESCE(MAX(SUBSTR(match_date,1,10)),'')
		 FROM match
		 WHERE match_date IS NOT NULL AND match_date != '' AND match_date != '0001-01-01T00:00:00Z'`,
	).Scan(&min, &max); err != nil {
		return storage.StatsDateRange{}, fmt.Errorf("sqlite: stats date range: %w", err)
	}
	return storage.StatsDateRange{DateFrom: min, DateTo: max}, nil
}

// Compute aggregates performance metrics for the given filter.
func (s *statsStore) Compute(ctx context.Context, scope string, filter storage.StatsFilter) (*storage.StatsResult, error) {
	whereSQL, baseArgs := buildStatsWhereClause(filter)

	result := &storage.StatsResult{
		PRRolling: make(map[int]float64),
	}

	// ── 1. Totals ────────────────────────────────────────────────────────────
	row := s.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT p.id), COUNT(DISTINCT m.id), COUNT(DISTINCT m.tournament_id), COUNT(*) `+
			statsBaseJoin+whereSQL,
		baseArgs...,
	)
	if err := row.Scan(
		&result.Totals.NumPositions,
		&result.Totals.NumMatches,
		&result.Totals.NumTournaments,
		&result.Totals.NumDecisions,
	); err != nil {
		return nil, fmt.Errorf("totals query: %w", err)
	}

	// ── 2. PR global + per decision_type ─────────────────────────────────────
	rows, err := s.db.QueryContext(ctx,
		`SELECT p.decision_type, SUM(`+statsErrExpr+`), COUNT(*) `+
			statsBaseJoin+whereSQL+
			` GROUP BY p.decision_type`,
		baseArgs...,
	)
	if err != nil {
		return nil, fmt.Errorf("PR by decision_type query: %w", err)
	}
	var totalErrSum int64
	var totalErrCount int
	func() {
		defer rows.Close()
		for rows.Next() {
			var dt int
			var sumErr int64
			var cnt int
			if err2 := rows.Scan(&dt, &sumErr, &cnt); err2 != nil {
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
	result.PRGlobal = pr(totalErrSum, totalErrCount)

	// ── Snowie ER (global) ────────────────────────────────────────────────────
	{
		snowieFilter := filter
		snowieFilter.DecisionType = -1 // count all decision types
		snowieWhere, snowieArgs := buildBaseWhereClause(snowieFilter)
		var snowieSumErr int64
		var snowieCheckerCnt int
		_ = s.db.QueryRowContext(ctx,
			`SELECT COALESCE(SUM(`+statsErrExpr+`),0), `+
				`COALESCE(SUM(CASE WHEN p.decision_type=0 THEN 1 ELSE 0 END),0) `+
				statsBaseJoin+snowieWhere,
			snowieArgs...,
		).Scan(&snowieSumErr, &snowieCheckerCnt)
		result.SnowieGlobal = snowieER(snowieSumErr, snowieCheckerCnt)
	}

	// ── 3. PR per tournament ──────────────────────────────────────────────────
	rows, err = s.db.QueryContext(ctx,
		`SELECT m.tournament_id, COALESCE(t.name,''), COALESCE(t.date,''), SUM(`+statsErrExpr+`), COUNT(*) `+
			statsBaseJoin+whereSQL+
			` AND m.tournament_id IS NOT NULL`+
			` GROUP BY m.tournament_id ORDER BY t.date, t.created_at`,
		baseArgs...,
	)
	if err != nil {
		return nil, fmt.Errorf("PR per tournament query: %w", err)
	}
	func() {
		defer rows.Close()
		for rows.Next() {
			var ts storage.TournamentStats
			var sumErr int64
			var cnt int
			if err2 := rows.Scan(&ts.ID, &ts.Name, &ts.Date, &sumErr, &cnt); err2 != nil {
				return
			}
			ts.NumDecisions = cnt
			ts.PR = pr(sumErr, cnt)
			result.PerTournament = append(result.PerTournament, ts)
		}
	}()

	// ── 4. PR per match ───────────────────────────────────────────────────────
	rows, err = s.db.QueryContext(ctx,
		`SELECT m.id, COALESCE(m.match_date,''), SUM(`+statsErrExpr+`), COUNT(*) `+
			statsBaseJoin+whereSQL+
			` GROUP BY m.id ORDER BY m.match_date`,
		baseArgs...,
	)
	if err != nil {
		return nil, fmt.Errorf("PR per match query: %w", err)
	}
	func() {
		defer rows.Close()
		for rows.Next() {
			var ms storage.MatchStats
			var sumErr int64
			var cnt int
			if err2 := rows.Scan(&ms.ID, &ms.Date, &sumErr, &cnt); err2 != nil {
				return
			}
			ms.NumDecisions = cnt
			ms.PR = pr(sumErr, cnt)
			result.PerMatch = append(result.PerMatch, ms)
		}
	}()

	// ── 5. Cube-action breakdown ──────────────────────────────────────────────
	{
		cubeWhere := whereSQL + " AND p.decision_type = 1"
		rows, err = s.db.QueryContext(ctx,
			`SELECT COALESCE(a.best_cube_action,''), SUM(a.cube_error), COUNT(*),`+
				` SUM(CASE WHEN a.cube_error > ? THEN 1 ELSE 0 END) `+
				statsBaseJoin+cubeWhere+
				` GROUP BY a.best_cube_action`,
			append([]any{blunderThresholdMP}, baseArgs...)...,
		)
		if err != nil {
			return nil, fmt.Errorf("cube action breakdown query: %w", err)
		}
		func() {
			defer rows.Close()
			for rows.Next() {
				var cs storage.CubeActionStats
				var sumErr int64
				if err2 := rows.Scan(&cs.Action, &sumErr, &cs.NumDecisions, &cs.BlunderCount); err2 != nil {
					return
				}
				cs.PR = pr(sumErr, cs.NumDecisions)
				result.CubeActionBreakdown = append(result.CubeActionBreakdown, cs)
			}
		}()
	}

	// ── 6. Error histogram ────────────────────────────────────────────────────
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
		statsBaseJoin + whereSQL +
		` GROUP BY bucket ORDER BY bucket`

	rows, err = s.db.QueryContext(ctx, histogramSQL, baseArgs...)
	if err != nil {
		return nil, fmt.Errorf("error histogram query: %w", err)
	}
	bucketMaxMap := map[int]int{0: 5, 5: 10, 10: 25, 25: 50, 50: 100, 100: -1}
	func() {
		defer rows.Close()
		for rows.Next() {
			var bucketMin, cnt int
			if err2 := rows.Scan(&bucketMin, &cnt); err2 != nil {
				return
			}
			result.ErrorHistogram = append(result.ErrorHistogram, storage.ErrorBucket{
				MinMP: bucketMin,
				MaxMP: bucketMaxMap[bucketMin],
				Count: cnt,
			})
		}
	}()

	// ── 7. Top blunders ───────────────────────────────────────────────────────
	rows, err = s.db.QueryContext(ctx,
		`SELECT p.id, m.id, COALESCE(m.tournament_id, 0), (`+statsErrExpr+`) as emg,`+
			` p.decision_type,`+
			` COALESCE(m.match_date, '') as match_date,`+
			` COALESCE(m.player1_name, '') || ' vs ' || COALESCE(m.player2_name, '') as player_names `+
			statsBaseJoin+whereSQL+
			` ORDER BY emg DESC LIMIT 10`,
		baseArgs...,
	)
	if err != nil {
		return nil, fmt.Errorf("top blunders query: %w", err)
	}
	func() {
		defer rows.Close()
		for rows.Next() {
			var be storage.BlunderEntry
			if err2 := rows.Scan(&be.PositionID, &be.MatchID, &be.TournamentID, &be.ErrorMP,
				&be.DecisionType, &be.MatchDate, &be.PlayerNames); err2 != nil {
				return
			}
			result.TopBlunders = append(result.TopBlunders, be)
		}
	}()

	// ── 8. Rolling PR ─────────────────────────────────────────────────────────
	rollingNs := []int{5, 10, 50, 100, 250, 500, 1000}
	maxN := rollingNs[len(rollingNs)-1]

	recentRows, err := s.db.QueryContext(ctx,
		`SELECT (`+statsErrExpr+`) as err `+
			statsBaseJoin+whereSQL+
			` ORDER BY m.match_date DESC, mv.move_number DESC LIMIT ?`,
		append(baseArgs, maxN)...,
	)
	if err != nil {
		return nil, fmt.Errorf("rolling PR query: %w", err)
	}
	var recentErrors []int64
	func() {
		defer recentRows.Close()
		for recentRows.Next() {
			var e int64
			if err2 := recentRows.Scan(&e); err2 != nil {
				return
			}
			recentErrors = append(recentErrors, e)
		}
	}()

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

	// ── MWC pass ──────────────────────────────────────────────────────────────
	// Stream per-row data in most-recent-first order and aggregate MWC losses in
	// Go. One supplementary SQL pass; O(n_decisions).
	{
		mwcPassSQL := `SELECT ` + statsErrExpr + ` as err,` +
			` COALESCE(p.score_1, 0), COALESCE(p.score_2, 0), mv.player,` +
			` (1 << COALESCE(p.cube_value, 0)), COALESCE(p.match_length, m.match_length, 0),` +
			` COALESCE(m.tournament_id, 0), m.id,` +
			` COALESCE(a.best_cube_action, ''), p.decision_type, p.id ` +
			statsBaseJoin + whereSQL +
			` ORDER BY m.match_date DESC, mv.move_number DESC`

		mwcRows, mwcErr := s.db.QueryContext(ctx, mwcPassSQL, baseArgs...)
		if mwcErr != nil {
			return nil, fmt.Errorf("MWC pass query: %w", mwcErr)
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

	return result, nil
}

// PositionIDsBySelection resolves a user selection made in the Stats panel into
// a deduplicated list of position IDs. The StatsFilter is always applied so the
// IDs correspond exactly to what is displayed in the panel.
func (s *statsStore) PositionIDsBySelection(ctx context.Context, scope string, filter storage.StatsFilter, sel storage.SelectionSpec) ([]int64, error) {
	whereSQL, baseArgs := buildStatsWhereClause(filter)
	whereAdd, orderLimit, selArgs := buildSelectionWhereClause(sel)

	query := "SELECT DISTINCT p.id " + statsBaseJoin + whereSQL + whereAdd
	if orderLimit != "" {
		query += " " + orderLimit
	}

	allArgs := append(append([]any{}, baseArgs...), selArgs...)
	rows, err := s.db.QueryContext(ctx, query, allArgs...)
	if err != nil {
		return nil, fmt.Errorf("PositionIDsBySelection (%s): %w", sel.Kind, err)
	}
	return scanPositionIDs(rows)
}

// PositionIDsByTournament returns all position IDs belonging to the given
// tournament, regardless of any stats filter.
func (s *statsStore) PositionIDsByTournament(ctx context.Context, scope string, tournamentID int64) ([]int64, error) {
	query := "SELECT DISTINCT p.id " + statsBaseJoin +
		" WHERE m.tournament_id = ? AND a.position_id IS NOT NULL AND (" + statsErrExpr + ") IS NOT NULL"
	rows, err := s.db.QueryContext(ctx, query, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("PositionIDsByTournament: %w", err)
	}
	return scanPositionIDs(rows)
}

// PositionIDsByMatch returns all position IDs belonging to the given match,
// regardless of any stats filter.
func (s *statsStore) PositionIDsByMatch(ctx context.Context, scope string, matchID int64) ([]int64, error) {
	query := "SELECT DISTINCT p.id " + statsBaseJoin +
		" WHERE m.id = ? AND a.position_id IS NOT NULL AND (" + statsErrExpr + ") IS NOT NULL"
	rows, err := s.db.QueryContext(ctx, query, matchID)
	if err != nil {
		return nil, fmt.Errorf("PositionIDsByMatch: %w", err)
	}
	return scanPositionIDs(rows)
}

// PlayerNames returns all player names found in the match table, ranked by the
// total number of matches (player1 + player2 appearances) descending; ties
// break alphabetically.
func (s *statsStore) PlayerNames(ctx context.Context, scope string) ([]storage.PlayerFrequency, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT name, COUNT(*) AS cnt
		FROM (
			SELECT player1_name AS name FROM match WHERE player1_name != ''
			UNION ALL
			SELECT player2_name AS name FROM match WHERE player2_name != ''
		)
		GROUP BY name
		ORDER BY cnt DESC, name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("PlayerNames: %w", err)
	}
	defer rows.Close()

	var result []storage.PlayerFrequency
	for rows.Next() {
		var pf storage.PlayerFrequency
		if err := rows.Scan(&pf.Name, &pf.Count); err != nil {
			return nil, fmt.Errorf("PlayerNames scan: %w", err)
		}
		result = append(result, pf)
	}
	return result, rows.Err()
}

// MatchDetail computes per-player statistics for the given match.
func (s *statsStore) MatchDetail(ctx context.Context, scope string, matchID int64) (*storage.MatchDetailStats, error) {
	query := `SELECT mv.player, p.decision_type, COALESCE(mv.cube_action,''),
		(` + statsErrExpr + `) as err_mp,
		COALESCE(p.score_1, 0), COALESCE(p.score_2, 0),
		(1 << COALESCE(p.cube_value, 0)),
		COALESCE(p.match_length, m.match_length, 0) ` +
		statsBaseJoin +
		` WHERE m.id = ? AND a.position_id IS NOT NULL AND (` + statsErrExpr + `) IS NOT NULL AND ` + statsCountedExpr

	rows, err := s.db.QueryContext(ctx, query, matchID)
	if err != nil {
		return nil, fmt.Errorf("MatchDetail query: %w", err)
	}
	defer rows.Close()

	type playerAcc struct {
		totalSumErr   int64
		totalCnt      int
		totalErrors   int
		totalBlunders int
		totalMWC      float64

		checkerSumErr   int64
		checkerCnt      int
		checkerErrors   int
		checkerBlunders int
		checkerMWC      float64

		doubleSumErr   int64
		doubleCnt      int
		doubleErrors   int
		doubleBlunders int
		doubleMWC      float64

		takeSumErr   int64
		takeCnt      int
		takeErrors   int
		takeBlunders int
		takeMWC      float64
	}

	var p1, p2 playerAcc

	for rows.Next() {
		var rawPlayer, decisionType int
		var cubeAction string
		var errMP int64
		var awayScore0, awayScore1, cubeValue, matchLength int
		if err := rows.Scan(&rawPlayer, &decisionType, &cubeAction, &errMP,
			&awayScore0, &awayScore1, &cubeValue, &matchLength); err != nil {
			continue
		}

		fMove := 0
		if rawPlayer == -1 {
			fMove = 1
		}
		currentScore0 := matchLength - awayScore0
		currentScore1 := matchLength - awayScore1
		mwcLoss := engine.ConvertEMGLossToMWCLoss(int(errMP), currentScore0, currentScore1, fMove, cubeValue, matchLength)
		if math.IsNaN(mwcLoss) {
			mwcLoss = 0
		}

		isError := errMP > 0
		isBlunder := errMP >= blunderThresholdMP
		isTake := cubeAction == "Take" || cubeAction == "Pass"

		acc := &p1
		if rawPlayer == -1 {
			acc = &p2
		}

		acc.totalSumErr += errMP
		acc.totalCnt++
		acc.totalMWC += mwcLoss
		if isError {
			acc.totalErrors++
		}
		if isBlunder {
			acc.totalBlunders++
		}

		if decisionType == 0 {
			acc.checkerSumErr += errMP
			acc.checkerCnt++
			acc.checkerMWC += mwcLoss
			if isError {
				acc.checkerErrors++
			}
			if isBlunder {
				acc.checkerBlunders++
			}
		} else {
			if isTake {
				acc.takeSumErr += errMP
				acc.takeCnt++
				acc.takeMWC += mwcLoss
				if isError {
					acc.takeErrors++
				}
				if isBlunder {
					acc.takeBlunders++
				}
			} else {
				acc.doubleSumErr += errMP
				acc.doubleCnt++
				acc.doubleMWC += mwcLoss
				if isError {
					acc.doubleErrors++
				}
				if isBlunder {
					acc.doubleBlunders++
				}
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("MatchDetail scan: %w", err)
	}

	// ── Snowie ER pass ────────────────────────────────────────────────────────
	// Second query without statsCountedExpr: sum all equity errors per player,
	// count checker positions (decision_type=0, forced included) for both
	// players. Denominator = anTotalMoves[P1] + anTotalMoves[P2].
	var snowieP1SumErr, snowieP2SumErr int64
	var snowieP1Checker, snowieP2Checker int
	{
		snowieRows, snowieErr := s.db.QueryContext(ctx,
			`SELECT mv.player, p.decision_type, (`+statsErrExpr+`) as err_mp `+
				statsBaseJoin+
				` WHERE m.id = ? AND a.position_id IS NOT NULL AND (`+statsErrExpr+`) IS NOT NULL`,
			matchID)
		if snowieErr != nil {
			return nil, fmt.Errorf("MatchDetail snowie query: %w", snowieErr)
		}
		func() {
			defer snowieRows.Close()
			for snowieRows.Next() {
				var rawPlayer, decisionType int
				var errMP int64
				if err2 := snowieRows.Scan(&rawPlayer, &decisionType, &errMP); err2 != nil {
					return
				}
				if rawPlayer == 1 {
					snowieP1SumErr += errMP
					if decisionType == 0 {
						snowieP1Checker++
					}
				} else {
					snowieP2SumErr += errMP
					if decisionType == 0 {
						snowieP2Checker++
					}
				}
			}
		}()
	}
	snowieDenom := snowieP1Checker + snowieP2Checker

	buildStats := func(a *playerAcc) storage.MatchPlayerDetailStats {
		cubeCnt := a.doubleCnt + a.takeCnt
		cubeSumErr := a.doubleSumErr + a.takeSumErr
		return storage.MatchPlayerDetailStats{
			TotalDecisions:   a.totalCnt,
			TotalErrors:      a.totalErrors,
			TotalBlunders:    a.totalBlunders,
			TotalEquityError: float64(a.totalSumErr) / 1000,
			PR:               pr(a.totalSumErr, a.totalCnt),
			MWCLoss:          a.totalMWC,

			CheckerDecisions:   a.checkerCnt,
			CheckerErrors:      a.checkerErrors,
			CheckerBlunders:    a.checkerBlunders,
			CheckerEquityError: float64(a.checkerSumErr) / 1000,
			PRChecker:          pr(a.checkerSumErr, a.checkerCnt),
			CheckerMWCLoss:     a.checkerMWC,

			DoubleDecisions:   a.doubleCnt,
			DoubleErrors:      a.doubleErrors,
			DoubleBlunders:    a.doubleBlunders,
			DoubleEquityError: float64(a.doubleSumErr) / 1000,
			DoubleMWCLoss:     a.doubleMWC,

			TakeDecisions:   a.takeCnt,
			TakeErrors:      a.takeErrors,
			TakeBlunders:    a.takeBlunders,
			TakeEquityError: float64(a.takeSumErr) / 1000,
			TakeMWCLoss:     a.takeMWC,

			PRCube:      pr(cubeSumErr, cubeCnt),
			CubeMWCLoss: a.doubleMWC + a.takeMWC,
		}
	}

	stats := &storage.MatchDetailStats{
		MatchID: matchID,
		Player1: buildStats(&p1),
		Player2: buildStats(&p2),
	}
	stats.Player1.SnowieER = snowieER(snowieP1SumErr, snowieDenom)
	stats.Player2.SnowieER = snowieER(snowieP2SumErr, snowieDenom)
	return stats, nil
}

// MatchBadges computes the per-player PR and total MWC loss for every match,
// keyed by match id. It is the list-row projection of MatchDetail; both share
// statsBaseJoin + statsCountedExpr so a match's badge PR equals its detail PR.
func (s *statsStore) MatchBadges(ctx context.Context, scope string) (map[int64]storage.MatchBadge, error) {
	query := `SELECT g.match_id, ` + statsErrExpr + ` as err_mp,
		COALESCE(p.score_1, 0), COALESCE(p.score_2, 0), mv.player,
		(1 << COALESCE(p.cube_value, 0)), COALESCE(p.match_length, m.match_length, 0) ` +
		statsBaseJoin +
		` WHERE a.position_id IS NOT NULL AND (` + statsErrExpr + `) IS NOT NULL AND ` + statsCountedExpr

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("MatchBadges query: %w", err)
	}
	defer rows.Close()

	type playerAcc struct {
		sumErr int64
		cnt    int
		mwc    float64
	}
	type matchAcc struct{ p1, p2 playerAcc }
	acc := make(map[int64]*matchAcc)
	for rows.Next() {
		var matchID, errMP int64
		var awayScore0, awayScore1, rawPlayer, cubeValue, matchLength int
		if err := rows.Scan(&matchID, &errMP, &awayScore0, &awayScore1, &rawPlayer, &cubeValue, &matchLength); err != nil {
			continue
		}
		a := acc[matchID]
		if a == nil {
			a = &matchAcc{}
			acc[matchID] = a
		}
		fMove := 0
		if rawPlayer == -1 {
			fMove = 1
		}
		// p.score_1/score_2 are away scores; ConvertEMGLossToMWCLoss wants current scores.
		mwcLoss := engine.ConvertEMGLossToMWCLoss(int(errMP), matchLength-awayScore0, matchLength-awayScore1, fMove, cubeValue, matchLength)
		pa := &a.p1
		if rawPlayer != 1 { // player2 on roll (rawPlayer == -1)
			pa = &a.p2
		}
		pa.sumErr += errMP
		pa.cnt++
		if !math.IsNaN(mwcLoss) {
			pa.mwc += mwcLoss
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make(map[int64]storage.MatchBadge, len(acc))
	for matchID, a := range acc {
		out[matchID] = storage.MatchBadge{
			PR:       pr(a.p1.sumErr, a.p1.cnt),
			MWCLoss:  a.p1.mwc,
			PR2:      pr(a.p2.sumErr, a.p2.cnt),
			MWCLoss2: a.p2.mwc,
		}
	}
	return out, nil
}

// TournamentBadges computes the aggregate PR and total MWC loss for every
// tournament (all matches pooled, both players), keyed by tournament id.
func (s *statsStore) TournamentBadges(ctx context.Context, scope string) (map[int64]storage.TournamentBadge, error) {
	query := `SELECT m.tournament_id, ` + statsErrExpr + ` as err_mp,
		COALESCE(p.score_1, 0), COALESCE(p.score_2, 0), mv.player,
		(1 << COALESCE(p.cube_value, 0)), COALESCE(p.match_length, m.match_length, 0) ` +
		statsBaseJoin +
		` WHERE a.position_id IS NOT NULL AND (` + statsErrExpr + `) IS NOT NULL
		AND m.tournament_id IS NOT NULL AND ` + statsCountedExpr

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("TournamentBadges query: %w", err)
	}
	defer rows.Close()

	type tournAcc struct {
		sumErr int64
		cnt    int
		mwc    float64
	}
	acc := make(map[int64]*tournAcc)
	for rows.Next() {
		var tournamentID, errMP int64
		var awayScore0, awayScore1, rawPlayer, cubeValue, matchLength int
		if err := rows.Scan(&tournamentID, &errMP, &awayScore0, &awayScore1, &rawPlayer, &cubeValue, &matchLength); err != nil {
			continue
		}
		a := acc[tournamentID]
		if a == nil {
			a = &tournAcc{}
			acc[tournamentID] = a
		}
		fMove := 0
		if rawPlayer == -1 {
			fMove = 1
		}
		a.sumErr += errMP
		a.cnt++
		if mwcLoss := engine.ConvertEMGLossToMWCLoss(int(errMP), matchLength-awayScore0, matchLength-awayScore1, fMove, cubeValue, matchLength); !math.IsNaN(mwcLoss) {
			a.mwc += mwcLoss
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make(map[int64]storage.TournamentBadge, len(acc))
	for tournamentID, a := range acc {
		out[tournamentID] = storage.TournamentBadge{PR: pr(a.sumErr, a.cnt), MWCLoss: a.mwc}
	}
	return out, nil
}
