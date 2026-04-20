package main

import (
	"fmt"
	"math"
	"strings"
)

// statsErrExpr is the SQL CASE expression that selects the correct error column
// based on position decision type. Shared with db_search.go.
const statsErrExpr = "CASE WHEN p.decision_type = 1 THEN a.cube_error ELSE a.best_move_equity_error END"

// blunderThresholdMP is the error threshold (in stored millipoints units) above
// which a decision is counted as a blunder. 100 ≈ 0.1 EMG.
const blunderThresholdMP = 100

// StatsFilter defines the filtering criteria for ComputeStats.
type StatsFilter struct {
	PlayerName    string
	TournamentIDs []int64
	DateFrom      string // ISO "YYYY-MM-DD"
	DateTo        string // ISO "YYYY-MM-DD"
	DecisionType  int    // -1=all, 0=checker, 1=cube
	MatchLength   []int
}

// StatsTotals holds high-level counts for a stats result.
type StatsTotals struct {
	NumPositions   int
	NumMatches     int
	NumTournaments int
	NumDecisions   int
}

// TournamentStats holds aggregated stats for a single tournament.
type TournamentStats struct {
	ID           int64
	Name         string
	Date         string
	PR           float64
	MWC          float64
	NumDecisions int
}

// MatchStats holds aggregated stats for a single match.
type MatchStats struct {
	ID           int64
	Date         string
	PlayerName   string
	PR           float64
	MWC          float64
	NumDecisions int
}

// CubeActionStats holds aggregated stats grouped by cube action.
type CubeActionStats struct {
	Action       string
	PR           float64
	MWC          float64
	NumDecisions int
	BlunderCount int
}

// ErrorBucket groups decisions by magnitude of error.
type ErrorBucket struct {
	MinMP int
	MaxMP int
	Count int
}

// BlunderEntry identifies a single bad decision.
type BlunderEntry struct {
	PositionID   int64
	MatchID      int64
	TournamentID int64
	ErrorMP      int64
	MWCLoss      float64
	Description  string
}

// StatsResult contains all computed statistics for the given filter.
type StatsResult struct {
	Totals              StatsTotals
	PRGlobal            float64
	PRChecker           float64
	PRCube              float64
	PRRolling           map[int]float64 // keyed by N: 5,10,50,100,250,500,1000
	MWCGlobal           float64         // sum of MWC losses across all match-play decisions
	MWCChecker          float64         // MWC loss from checker play errors
	MWCCube             float64         // MWC loss from cube action errors
	MWCRolling          map[int]float64 // rolling MWC loss over N most-recent decisions (same keys as PRRolling)
	MWCAvailable        bool            // true if at least one match-play decision contributed
	PerTournament       []TournamentStats
	PerMatch            []MatchStats
	CubeActionBreakdown []CubeActionStats
	ErrorHistogram      []ErrorBucket
	TopBlunders         []BlunderEntry
}

// pr computes the Performance Rating from a sum of errors (millipoints stored
// units) and the number of decisions. Formula: 500 × sumErrMP / 1000 / nDecisions.
func pr(sumErrMP int64, nDecisions int) float64 {
	if nDecisions == 0 {
		return 0
	}
	return 500 * float64(sumErrMP) / 1000 / float64(nDecisions)
}

// statsBaseJoin is the FROM + JOIN fragment shared by all stats queries.
const statsBaseJoin = `FROM position p
JOIN analysis a ON a.position_id = p.id
JOIN move mv ON mv.position_id = p.id
JOIN game g ON g.id = mv.game_id
JOIN match m ON m.id = g.match_id
LEFT JOIN tournament t ON t.id = m.tournament_id`

// buildStatsWhereClause constructs a WHERE clause and argument slice for the
// given filter. The returned whereSQL starts with " WHERE " (or is empty when no
// filter is active). It is private and shared with fiche 03.
func buildStatsWhereClause(filter StatsFilter) (whereSQL string, args []any) {
	var clauses []string

	if filter.PlayerName != "" {
		clauses = append(clauses, "((m.player1_name = ? AND mv.player = 0) OR (m.player2_name = ? AND mv.player = 1))")
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

	// Exclude positions with NULL analysis (no error data).
	clauses = append(clauses, "a.position_id IS NOT NULL")
	clauses = append(clauses, "("+statsErrExpr+") IS NOT NULL")

	whereSQL = " WHERE " + strings.Join(clauses, " AND ")
	return whereSQL, args
}

// ComputeStats aggregates performance metrics for the given filter.
func (d *Database) ComputeStats(filter StatsFilter) (*StatsResult, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	whereSQL, baseArgs := buildStatsWhereClause(filter)

	result := &StatsResult{
		PRRolling: make(map[int]float64),
	}

	// ── 1. Totals ────────────────────────────────────────────────────────────
	row := d.db.QueryRow(
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
	rows, err := d.db.Query(
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

	// ── 3. PR per tournament ──────────────────────────────────────────────────
	rows, err = d.db.Query(
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
			var ts TournamentStats
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
	rows, err = d.db.Query(
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
			var ms MatchStats
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
		rows, err = d.db.Query(
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
				var cs CubeActionStats
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

	rows, err = d.db.Query(histogramSQL, baseArgs...)
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
			result.ErrorHistogram = append(result.ErrorHistogram, ErrorBucket{
				MinMP: bucketMin,
				MaxMP: bucketMaxMap[bucketMin],
				Count: cnt,
			})
		}
	}()

	// ── 7. Top blunders ───────────────────────────────────────────────────────
	rows, err = d.db.Query(
		`SELECT p.id, m.id, COALESCE(m.tournament_id, 0), (`+statsErrExpr+`) as emg `+
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
			var be BlunderEntry
			if err2 := rows.Scan(&be.PositionID, &be.MatchID, &be.TournamentID, &be.ErrorMP); err2 != nil {
				return
			}
			result.TopBlunders = append(result.TopBlunders, be)
		}
	}()

	// ── 8. Rolling PR ─────────────────────────────────────────────────────────
	rollingNs := []int{5, 10, 50, 100, 250, 500, 1000}
	maxN := rollingNs[len(rollingNs)-1]

	recentRows, err := d.db.Query(
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
	// Stream per-row data (err, position context) in most-recent-first order and
	// aggregate MWC losses in Go. One supplementary SQL pass; O(n_decisions).
	{
		mwcPassSQL := `SELECT ` + statsErrExpr + ` as err,` +
			` COALESCE(p.score_1, 0), COALESCE(p.score_2, 0), mv.player,` +
			` COALESCE(p.cube_value, 1), COALESCE(p.match_length, m.match_length, 0),` +
			` COALESCE(m.tournament_id, 0), m.id,` +
			` COALESCE(a.best_cube_action, ''), p.decision_type, p.id ` +
			statsBaseJoin + whereSQL +
			` ORDER BY m.match_date DESC, mv.move_number DESC`

		mwcRows, mwcErr := d.db.Query(mwcPassSQL, baseArgs...)
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
				var score0, score1, fMove, cubeValue, matchLength int
				var tournamentID, matchID int64
				var cubeAction string
				var dt int
				var posID int64
				if err2 := mwcRows.Scan(&errMP, &score0, &score1, &fMove, &cubeValue, &matchLength,
					&tournamentID, &matchID, &cubeAction, &dt, &posID); err2 != nil {
					return
				}

				rowIdx++
				mwcLoss := ConvertEMGLossToMWCLoss(int(errMP), score0, score1, fMove, cubeValue, matchLength)

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

				// Rolling thresholds count all decisions (including money-game), mirroring PRRolling.
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

// ── Drill-down ────────────────────────────────────────────────────────────────

// SelectionSpec describes what the user selected in the stats panel (a click on
// a bar, point, or row). The frontend passes this to GetPositionIDsByStatsSelection
// to obtain the matching position IDs for navigation.
type SelectionSpec struct {
	Kind         string // "all", "checker", "cube", "cube_action",
	// "error_bucket", "tournament", "match",
	// "last_n", "position", "top_blunders"
	CubeAction   string // "NoDouble" | "DoubleTake" | "DoublePass" | "TooGood"
	BucketMinMP  int    // inclusive
	BucketMaxMP  int    // exclusive; -1 = +∞
	TournamentID int64
	MatchID      int64
	LastN        int
	PositionID   int64
	OnlyWithError bool // for "cube_action", "checker", "cube" → error > 0
}

// buildSelectionWhereClause produces the extra WHERE fragment and optional
// ORDER BY / LIMIT fragment for a given SelectionSpec.
//
// whereAdd starts with " AND " or is empty (never contains "WHERE").
// orderLimit is the ORDER BY … LIMIT … suffix or empty.
// args contains the combined parameters for whereAdd followed by orderLimit.
func buildSelectionWhereClause(sel SelectionSpec) (whereAdd string, orderLimit string, args []any) {
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
		orderLimit = "ORDER BY (" + statsErrExpr + ") DESC LIMIT 10"
	// "all" → no extra clauses
	}
	return whereAdd, orderLimit, args
}

// scanPositionIDs scans a rows result set returning a single int64 column into
// a slice. It closes rows and propagates any iteration error.
func scanPositionIDs(rows interface {
	Next() bool
	Scan(...any) error
	Close() error
	Err() error
}) ([]int64, error) {
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

// GetPositionIDsByStatsSelection resolves a user selection made in the Stats
// panel into a deduplicated list of position IDs. The StatsFilter is always
// applied so that the IDs correspond exactly to what is displayed in the panel
// (invariant: "ce qu'on clique = ce qu'on voit").
func (d *Database) GetPositionIDsByStatsSelection(filter StatsFilter, sel SelectionSpec) ([]int64, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	whereSQL, baseArgs := buildStatsWhereClause(filter)
	whereAdd, orderLimit, selArgs := buildSelectionWhereClause(sel)

	query := "SELECT DISTINCT p.id " + statsBaseJoin + whereSQL + whereAdd
	if orderLimit != "" {
		query += " " + orderLimit
	}

	allArgs := append(append([]any{}, baseArgs...), selArgs...)
	rows, err := d.db.Query(query, allArgs...)
	if err != nil {
		return nil, fmt.Errorf("GetPositionIDsByStatsSelection (%s): %w", sel.Kind, err)
	}
	return scanPositionIDs(rows)
}

// GetPositionIDsByTournament returns all position IDs belonging to the given
// tournament, regardless of any stats filter. Used when the user explicitly
// reopens a tournament (Open tournament action).
func (d *Database) GetPositionIDsByTournament(tournamentID int64) ([]int64, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	query := "SELECT DISTINCT p.id " + statsBaseJoin +
		" WHERE m.tournament_id = ? AND a.position_id IS NOT NULL AND (" + statsErrExpr + ") IS NOT NULL"
	rows, err := d.db.Query(query, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("GetPositionIDsByTournament: %w", err)
	}
	return scanPositionIDs(rows)
}

// GetPositionIDsByMatch returns all position IDs belonging to the given match,
// regardless of any stats filter. Used when the user explicitly reopens a match.
func (d *Database) GetPositionIDsByMatch(matchID int64) ([]int64, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	query := "SELECT DISTINCT p.id " + statsBaseJoin +
		" WHERE m.id = ? AND a.position_id IS NOT NULL AND (" + statsErrExpr + ") IS NOT NULL"
	rows, err := d.db.Query(query, matchID)
	if err != nil {
		return nil, fmt.Errorf("GetPositionIDsByMatch: %w", err)
	}
	return scanPositionIDs(rows)
}
