package main

import (
	"database/sql"
	"fmt"
	"math"
	"strings"
)

// StatsDateRange holds the earliest and latest match dates in the database.
type StatsDateRange struct {
	DateFrom string `json:"DateFrom"` // ISO "YYYY-MM-DD", empty if no matches
	DateTo   string `json:"DateTo"`   // ISO "YYYY-MM-DD", empty if no matches
}

// GetStatsDateRange returns the minimum and maximum match dates present in the
// database. Both fields are empty when no matches with a date exist.
func (d *Database) GetStatsDateRange() StatsDateRange {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil {
		return StatsDateRange{}
	}
	var min, max string
	_ = d.db.QueryRow(
		`SELECT COALESCE(MIN(SUBSTR(match_date,1,10)),''), COALESCE(MAX(SUBSTR(match_date,1,10)),'')
		 FROM match
		 WHERE match_date IS NOT NULL AND match_date != '' AND match_date != '0001-01-01T00:00:00Z'`,
	).Scan(&min, &max)
	return StatsDateRange{DateFrom: min, DateTo: max}
}

// statsErrExpr is the SQL CASE expression that selects the correct error column
// based on position decision type. Shared with db_search.go.
const statsErrExpr = "CASE WHEN p.decision_type = 1 THEN a.cube_error ELSE a.best_move_equity_error END"

// blunderThresholdMP is the error threshold (in stored millipoints units) above
// which a decision is counted as a blunder. 100 ≈ 0.1 EMG.
const blunderThresholdMP = 100

// statsCountedExpr is the SQL predicate selecting the decisions that count
// toward PR and decision tallies (XG + gnuBG semantics). Forced checker plays
// and non-close cube decisions are excluded.
const statsCountedExpr = "((p.decision_type = 0 AND a.is_forced = 0) OR (p.decision_type = 1 AND a.is_close_cube = 1))"

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
	NumPositions   int `json:"NumPositions"`
	NumMatches     int `json:"NumMatches"`
	NumTournaments int `json:"NumTournaments"`
	NumDecisions   int `json:"NumDecisions"`
}

// TournamentStats holds aggregated stats for a single tournament.
type TournamentStats struct {
	ID           int64   `json:"ID"`
	Name         string  `json:"Name"`
	Date         string  `json:"Date"`
	PR           float64 `json:"PR"`
	MWC          float64 `json:"MWC"`
	NumDecisions int     `json:"NumDecisions"`
}

// MatchStats holds aggregated stats for a single match.
type MatchStats struct {
	ID           int64   `json:"ID"`
	Date         string  `json:"Date"`
	PlayerName   string  `json:"PlayerName"`
	PR           float64 `json:"PR"`
	MWC          float64 `json:"MWC"`
	NumDecisions int     `json:"NumDecisions"`
}

// CubeActionStats holds aggregated stats grouped by cube action.
type CubeActionStats struct {
	Action       string  `json:"Action"`
	PR           float64 `json:"PR"`
	MWC          float64 `json:"MWC"`
	NumDecisions int     `json:"NumDecisions"`
	BlunderCount int     `json:"BlunderCount"`
}

// ErrorBucket groups decisions by magnitude of error.
type ErrorBucket struct {
	MinMP int `json:"MinMP"`
	MaxMP int `json:"MaxMP"`
	Count int `json:"Count"`
}

// BlunderEntry identifies a single bad decision.
type BlunderEntry struct {
	PositionID   int64   `json:"PositionID"`
	MatchID      int64   `json:"MatchID"`
	TournamentID int64   `json:"TournamentID"`
	ErrorMP      int64   `json:"ErrorMP"`
	MWCLoss      float64 `json:"MWCLoss"`
	Description  string  `json:"Description"`
	DecisionType int     `json:"DecisionType"` // 0=checker, 1=cube
	MatchDate    string  `json:"MatchDate"`    // ISO date string, may be empty
	PlayerNames  string  `json:"PlayerNames"`  // "Player1 vs Player2"
}

// StatsResult contains all computed statistics for the given filter.
type StatsResult struct {
	Totals              StatsTotals       `json:"Totals"`
	PRGlobal            float64           `json:"PRGlobal"`
	PRChecker           float64           `json:"PRChecker"`
	PRCube              float64           `json:"PRCube"`
	PRRolling           map[int]float64   `json:"PRRolling"`    // keyed by N: 5,10,50,100,250,500,1000
	MWCGlobal           float64           `json:"MWCGlobal"`    // sum of MWC losses across all match-play decisions
	MWCChecker          float64           `json:"MWCChecker"`   // MWC loss from checker play errors
	MWCCube             float64           `json:"MWCCube"`      // MWC loss from cube action errors
	MWCRolling          map[int]float64   `json:"MWCRolling"`   // rolling MWC loss over N most-recent decisions (same keys as PRRolling)
	MWCAvailable        bool              `json:"MWCAvailable"` // true if at least one match-play decision contributed
	PerTournament       []TournamentStats `json:"PerTournament"`
	PerMatch            []MatchStats      `json:"PerMatch"`
	CubeActionBreakdown []CubeActionStats `json:"CubeActionBreakdown"`
	ErrorHistogram      []ErrorBucket     `json:"ErrorHistogram"`
	TopBlunders         []BlunderEntry    `json:"TopBlunders"`
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

	// Exclude positions with NULL analysis (no error data).
	clauses = append(clauses, "a.position_id IS NOT NULL")
	clauses = append(clauses, "("+statsErrExpr+") IS NOT NULL")

	// Apply XG/gnuBG counting semantics: exclude forced checker plays and
	// non-close cube decisions from all PR and decision-count queries.
	clauses = append(clauses, statsCountedExpr)

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
			var be BlunderEntry
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
			` (1 << COALESCE(p.cube_value, 0)), COALESCE(p.match_length, m.match_length, 0),` +
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

				// Convert XG player encoding (1 or -1) to gnuBG fMove (0 or 1).
				// XG encodes player 0 (bottom) as 1 and player 1 (top) as -1.
				fMove := 0
				if rawPlayer == -1 {
					fMove = 1
				}
				// p.score_1/score_2 are away scores (points still needed); gnuBGGetME
				// expects current scores (games already won).
				currentScore0 := matchLength - awayScore0
				currentScore1 := matchLength - awayScore1

				mwcLoss := ConvertEMGLossToMWCLoss(int(errMP), currentScore0, currentScore1, fMove, cubeValue, matchLength)

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
	Kind string // "all", "checker", "cube", "cube_action",
	// "error_bucket", "tournament", "match",
	// "last_n", "position", "top_blunders"
	CubeAction    string // "NoDouble" | "DoubleTake" | "DoublePass" | "TooGood"
	BucketMinMP   int    // inclusive
	BucketMaxMP   int    // exclusive; -1 = +∞
	TournamentID  int64
	MatchID       int64
	LastN         int
	PositionID    int64
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

// PlayerFrequency pairs a player name with the number of matches in which they appear.
type PlayerFrequency struct {
	Name  string
	Count int
}

// GetAllPlayerNames returns all player names found in the match table, ranked by
// the total number of matches (player1 + player2 appearances) descending.
// Names that are equal in frequency are sorted alphabetically.
func (d *Database) GetAllPlayerNames() ([]PlayerFrequency, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	rows, err := d.db.Query(`
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
		return nil, fmt.Errorf("GetAllPlayerNames: %w", err)
	}
	defer rows.Close()

	var result []PlayerFrequency
	for rows.Next() {
		var pf PlayerFrequency
		if err := rows.Scan(&pf.Name, &pf.Count); err != nil {
			return nil, fmt.Errorf("GetAllPlayerNames scan: %w", err)
		}
		result = append(result, pf)
	}
	return result, rows.Err()
}

// matchPRMWCAcc accumulates error millipoints and MWC losses for one match.
type matchPRMWCAcc struct {
	sumErr int64
	cnt    int
	mwc    float64
}

// matchPlayerPRMWCAcc tracks per-player accumulators for a match.
type matchPlayerPRMWCAcc struct {
	p1 matchPRMWCAcc
	p2 matchPRMWCAcc
}

// populateMatchStats computes PR and total MWC loss for each match in the slice
// and sets the PR/PR2 and MWCLoss/MWCLoss2 fields in-place, split by player.
// Must be called while the caller holds at least the read lock on the database.
func populateMatchStats(db *sql.DB, matches []Match) error {
	if len(matches) == 0 {
		return nil
	}

	matchByID := make(map[int64]*Match, len(matches))
	for i := range matches {
		matchByID[matches[i].ID] = &matches[i]
	}

	query := `SELECT g.match_id, ` + statsErrExpr + ` as err_mp,
		COALESCE(p.score_1, 0), COALESCE(p.score_2, 0), mv.player,
		(1 << COALESCE(p.cube_value, 0)), COALESCE(p.match_length, m.match_length, 0) ` +
		statsBaseJoin +
		` WHERE a.position_id IS NOT NULL AND (` + statsErrExpr + `) IS NOT NULL AND ` + statsCountedExpr

	rows, err := db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	acc := make(map[int64]*matchPlayerPRMWCAcc)
	for rows.Next() {
		var matchID int64
		var errMP int64
		var awayScore0, awayScore1, rawPlayer, cubeValue, matchLength int
		if err := rows.Scan(&matchID, &errMP, &awayScore0, &awayScore1, &rawPlayer, &cubeValue, &matchLength); err != nil {
			continue
		}
		if _, ok := matchByID[matchID]; !ok {
			continue
		}
		a, ok := acc[matchID]
		if !ok {
			a = &matchPlayerPRMWCAcc{}
			acc[matchID] = a
		}
		fMove := 0
		if rawPlayer == -1 {
			fMove = 1
		}
		// p.score_1/score_2 are away scores; ConvertEMGLossToMWCLoss expects current scores.
		currentScore0 := matchLength - awayScore0
		currentScore1 := matchLength - awayScore1
		mwcLoss := ConvertEMGLossToMWCLoss(int(errMP), currentScore0, currentScore1, fMove, cubeValue, matchLength)
		if rawPlayer == 1 { // player1 on roll
			a.p1.sumErr += errMP
			a.p1.cnt++
			if !math.IsNaN(mwcLoss) {
				a.p1.mwc += mwcLoss
			}
		} else { // player2 on roll (rawPlayer == -1)
			a.p2.sumErr += errMP
			a.p2.cnt++
			if !math.IsNaN(mwcLoss) {
				a.p2.mwc += mwcLoss
			}
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for matchID, a := range acc {
		if m, ok := matchByID[matchID]; ok {
			m.PR = pr(a.p1.sumErr, a.p1.cnt)
			m.MWCLoss = a.p1.mwc
			m.PR2 = pr(a.p2.sumErr, a.p2.cnt)
			m.MWCLoss2 = a.p2.mwc
		}
	}
	return nil
}

// populateTournamentStats computes PR and total MWC loss for each tournament
// (aggregated across all matches in the tournament) and sets the fields in-place.
// Must be called while the caller holds at least the read lock on the database.
func populateTournamentStats(db *sql.DB, tournaments []Tournament) error {
	if len(tournaments) == 0 {
		return nil
	}

	tournByID := make(map[int64]*Tournament, len(tournaments))
	for i := range tournaments {
		tournByID[tournaments[i].ID] = &tournaments[i]
	}

	query := `SELECT m.tournament_id, ` + statsErrExpr + ` as err_mp,
		COALESCE(p.score_1, 0), COALESCE(p.score_2, 0), mv.player,
		(1 << COALESCE(p.cube_value, 0)), COALESCE(p.match_length, m.match_length, 0) ` +
		statsBaseJoin +
		` WHERE a.position_id IS NOT NULL AND (` + statsErrExpr + `) IS NOT NULL
		AND m.tournament_id IS NOT NULL AND ` + statsCountedExpr

	rows, err := db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	type tournAcc struct {
		sumErr int64
		cnt    int
		mwc    float64
	}
	acc := make(map[int64]*tournAcc)
	for rows.Next() {
		var tournamentID int64
		var errMP int64
		var awayScore0, awayScore1, rawPlayer, cubeValue, matchLength int
		if err := rows.Scan(&tournamentID, &errMP, &awayScore0, &awayScore1, &rawPlayer, &cubeValue, &matchLength); err != nil {
			continue
		}
		if _, ok := tournByID[tournamentID]; !ok {
			continue
		}
		a, ok := acc[tournamentID]
		if !ok {
			a = &tournAcc{}
			acc[tournamentID] = a
		}
		a.sumErr += errMP
		a.cnt++
		// Convert XG player encoding (1/-1) to gnuBG fMove (0/1).
		// Convert away scores to current scores for ConvertEMGLossToMWCLoss.
		fMove := 0
		if rawPlayer == -1 {
			fMove = 1
		}
		currentScore0 := matchLength - awayScore0
		currentScore1 := matchLength - awayScore1
		if mwcLoss := ConvertEMGLossToMWCLoss(int(errMP), currentScore0, currentScore1, fMove, cubeValue, matchLength); !math.IsNaN(mwcLoss) {
			a.mwc += mwcLoss
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for tournamentID, a := range acc {
		if t, ok := tournByID[tournamentID]; ok {
			t.PR = pr(a.sumErr, a.cnt)
			t.MWCLoss = a.mwc
		}
	}
	return nil
}

// MatchPlayerDetailStats holds per-player statistics for a single match.
type MatchPlayerDetailStats struct {
	// Overall
	TotalDecisions   int     `json:"total_decisions"`
	TotalErrors      int     `json:"total_errors"`
	TotalBlunders    int     `json:"total_blunders"`
	TotalEquityError float64 `json:"total_equity_error"` // sum of errors in EMG
	PR               float64 `json:"pr"`
	MWCLoss          float64 `json:"mwc_loss"`

	// Checker play
	CheckerDecisions   int     `json:"checker_decisions"`
	CheckerErrors      int     `json:"checker_errors"`
	CheckerBlunders    int     `json:"checker_blunders"`
	CheckerEquityError float64 `json:"checker_equity_error"` // in EMG
	PRChecker          float64 `json:"pr_checker"`
	CheckerMWCLoss     float64 `json:"checker_mwc_loss"`

	// Cube play: doubling decisions
	DoubleDecisions   int     `json:"double_decisions"`
	DoubleErrors      int     `json:"double_errors"`
	DoubleBlunders    int     `json:"double_blunders"`
	DoubleEquityError float64 `json:"double_equity_error"` // in EMG
	DoubleMWCLoss     float64 `json:"double_mwc_loss"`

	// Cube play: take/pass decisions
	TakeDecisions   int     `json:"take_decisions"`
	TakeErrors      int     `json:"take_errors"`
	TakeBlunders    int     `json:"take_blunders"`
	TakeEquityError float64 `json:"take_equity_error"` // in EMG
	TakeMWCLoss     float64 `json:"take_mwc_loss"`

	// Combined cube PR
	PRCube      float64 `json:"pr_cube"`
	CubeMWCLoss float64 `json:"cube_mwc_loss"`
}

// MatchDetailStats holds per-player statistics for a single match.
type MatchDetailStats struct {
	MatchID int64                  `json:"match_id"`
	Player1 MatchPlayerDetailStats `json:"player1"`
	Player2 MatchPlayerDetailStats `json:"player2"`
}

// GetMatchDetailStats computes per-player statistics for the given match.
func (d *Database) GetMatchDetailStats(matchID int64) (*MatchDetailStats, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	query := `SELECT mv.player, p.decision_type, COALESCE(mv.cube_action,''),
		(` + statsErrExpr + `) as err_mp,
		COALESCE(p.score_1, 0), COALESCE(p.score_2, 0),
		(1 << COALESCE(p.cube_value, 0)),
		COALESCE(p.match_length, m.match_length, 0) ` +
		statsBaseJoin +
		` WHERE m.id = ? AND a.position_id IS NOT NULL AND (` + statsErrExpr + `) IS NOT NULL AND ` + statsCountedExpr

	rows, err := d.db.Query(query, matchID)
	if err != nil {
		return nil, fmt.Errorf("GetMatchDetailStats query: %w", err)
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
		// p.score_1/score_2 are away scores; ConvertEMGLossToMWCLoss expects current scores.
		currentScore0 := matchLength - awayScore0
		currentScore1 := matchLength - awayScore1
		mwcLoss := ConvertEMGLossToMWCLoss(int(errMP), currentScore0, currentScore1, fMove, cubeValue, matchLength)
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
		return nil, fmt.Errorf("GetMatchDetailStats scan: %w", err)
	}

	buildStats := func(a *playerAcc) MatchPlayerDetailStats {
		cubeCnt := a.doubleCnt + a.takeCnt
		cubeSumErr := a.doubleSumErr + a.takeSumErr
		return MatchPlayerDetailStats{
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

	return &MatchDetailStats{
		MatchID: matchID,
		Player1: buildStats(&p1),
		Player2: buildStats(&p2),
	}, nil
}
