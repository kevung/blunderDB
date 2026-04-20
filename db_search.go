package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// parseIntFilterExpr parses prefixed integer filter strings (e.g. "p>5", "p<20", "p3,10").
// Returns min, max (inclusive bounds) and flags indicating which bound is set.
func parseIntFilterExpr(filter, prefix string) (min, max int, hasMin, hasMax bool) {
	if !strings.HasPrefix(filter, prefix) {
		return
	}
	rest := filter[len(prefix):]
	if strings.HasPrefix(rest, ">") {
		v, err := strconv.Atoi(strings.TrimSpace(rest[1:]))
		if err != nil {
			return
		}
		return v, 0, true, false
	}
	if strings.HasPrefix(rest, "<") {
		v, err := strconv.Atoi(strings.TrimSpace(rest[1:]))
		if err != nil {
			return
		}
		return 0, v, false, true
	}
	parts := strings.SplitN(rest, ",", 2)
	if len(parts) == 1 {
		v, err := strconv.Atoi(strings.TrimSpace(rest))
		if err != nil {
			return
		}
		return v, v, true, true
	}
	v1, e1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	v2, e2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if e1 != nil || e2 != nil {
		return
	}
	if v1 > v2 {
		v1, v2 = v2, v1
	}
	return v1, v2, true, true
}

// parseFloatFilterExpr is the float64 variant of parseIntFilterExpr.
func parseFloatFilterExpr(filter, prefix string) (min, max float64, hasMin, hasMax bool) {
	if !strings.HasPrefix(filter, prefix) {
		return
	}
	rest := filter[len(prefix):]
	if strings.HasPrefix(rest, ">") {
		v, err := strconv.ParseFloat(strings.TrimSpace(rest[1:]), 64)
		if err != nil {
			return
		}
		return v, 0, true, false
	}
	if strings.HasPrefix(rest, "<") {
		v, err := strconv.ParseFloat(strings.TrimSpace(rest[1:]), 64)
		if err != nil {
			return
		}
		return 0, v, false, true
	}
	parts := strings.SplitN(rest, ",", 2)
	if len(parts) == 1 {
		v, err := strconv.ParseFloat(strings.TrimSpace(rest), 64)
		if err != nil {
			return
		}
		return v, v, true, true
	}
	v1, e1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	v2, e2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if e1 != nil || e2 != nil {
		return
	}
	if v1 > v2 {
		v1, v2 = v2, v1
	}
	return v1, v2, true, true
}

// appendIntRangeSQL appends "AND column op ?" to where and the bound(s) to args.
func appendIntRangeSQL(column string, min, max int, hasMin, hasMax bool, where *strings.Builder, args *[]any) {
	if !hasMin && !hasMax {
		return
	}
	if hasMin && hasMax {
		if min == max {
			where.WriteString(" AND " + column + " = ?")
			*args = append(*args, min)
		} else {
			where.WriteString(" AND " + column + " BETWEEN ? AND ?")
			*args = append(*args, min, max)
		}
	} else if hasMin {
		where.WriteString(" AND " + column + " >= ?")
		*args = append(*args, min)
	} else {
		where.WriteString(" AND " + column + " <= ?")
		*args = append(*args, max)
	}
}

// hasBoardFilter returns true if at least one point in b has non-empty checkers.
func hasBoardFilter(b Board) bool {
	for _, p := range b.Points {
		if p.Checkers > 0 && p.Color >= 0 {
			return true
		}
	}
	return false
}

// analysisMatchesFloatFilter checks value against a prefixed float filter string.
// Returns true when filter is empty or the (rounded) value satisfies the expression.
func analysisMatchesFloatFilter(filter, prefix string, value float64) bool {
	if filter == "" {
		return true
	}
	mn, mx, hasMin, hasMax := parseFloatFilterExpr(filter, prefix)
	if !hasMin && !hasMax {
		return true
	}
	value = roundToHundredthPercent(value)
	if hasMin && value < mn {
		return false
	}
	if hasMax && value > mx {
		return false
	}
	return true
}

// analysisMatchesEquityFilter checks the best-move equity of ana against the "e"-prefixed filter.
func analysisMatchesEquityFilter(filter string, ana *PositionAnalysis) bool {
	if filter == "" {
		return true
	}
	if ana == nil {
		return false
	}
	var equity float64
	if ana.AnalysisType == "DoublingCube" && ana.DoublingCubeAnalysis != nil {
		equity = ana.DoublingCubeAnalysis.CubefulNoDoubleEquity
	} else if ana.AnalysisType == "CheckerMove" && ana.CheckerAnalysis != nil && len(ana.CheckerAnalysis.Moves) > 0 {
		equity = ana.CheckerAnalysis.Moves[0].Equity
	} else {
		return false
	}
	equity = roundToMillipoint(equity)
	mn, mx, hasMin, hasMax := parseFloatFilterExpr(filter, "e")
	if !hasMin && !hasMax {
		return true
	}
	// Filter values are in millipoints; stored equity is in equity points.
	if hasMin && equity < mn/1000.0 {
		return false
	}
	if hasMax && equity > mx/1000.0 {
		return false
	}
	return true
}

// analysisMatchesMovePattern checks a move-pattern filter against pre-fetched analysis,
// avoiding a LoadAnalysis round-trip.
func analysisMatchesMovePattern(filter string, ana *PositionAnalysis) bool {
	if filter == "" {
		return true
	}
	if ana == nil {
		return false
	}
	movePatternMatch := strings.Trim(filter, `m"'`)
	movePatterns := strings.Split(strings.ToLower(movePatternMatch), ";")
	if ana.AnalysisType == "CheckerMove" && ana.CheckerAnalysis != nil && len(ana.CheckerAnalysis.Moves) > 0 {
		move := strings.ToLower(ana.CheckerAnalysis.Moves[0].Move)
		for _, pattern := range movePatterns {
			if strings.Contains(move, pattern) {
				return true
			}
		}
	} else if ana.AnalysisType == "DoublingCube" && ana.DoublingCubeAnalysis != nil {
		for _, pattern := range movePatterns {
			switch pattern {
			case "nd":
				if ana.DoublingCubeAnalysis.CubefulNoDoubleError == 0 {
					return true
				}
			case "dt":
				if ana.DoublingCubeAnalysis.CubefulDoubleTakeError == 0 {
					return true
				}
			case "dp":
				if ana.DoublingCubeAnalysis.CubefulDoublePassError == 0 {
					return true
				}
			}
		}
	}
	return false
}

// loadPositionsByFiltersCore is the internal implementation behind LoadPositionsByFilters.
// It returns matching positions and a map[positionID]*PositionAnalysis built from the LEFT
// JOIN, so callers can inspect analysis data without extra LoadAnalysis round-trips.
//
// When mirrorFilter is false, every cheap predicate is pushed to SQL (one LEFT JOIN query).
// When mirrorFilter is true, orientation-specific SQL clauses are disabled and all checks
// fall back to Go-side evaluation on the already-narrowed result set.
func (d *Database) loadPositionsByFiltersCore(
	f SearchFilters,
) ([]Position, map[int64]*PositionAnalysis, error) {
	// Push orientation-specific predicates to SQL only when f.MirrorFilter is off.
	// Mirror search needs both orientations checked; disabling SQL-side orientation
	// clauses and evaluating them in Go is correct (if slower) for that rare path.
	useSQLFilters := !f.MirrorFilter

	// -----------------------------------------------------------------------
	// 1. Build the SQL WHERE clause
	// -----------------------------------------------------------------------
	var where strings.Builder
	var args []any
	where.WriteString("1=1")

	// --- match / tournament filters (orientation-neutral: always pushed to SQL) ---
	if f.MatchIDsFilter != "" || f.TournamentIDsFilter != "" {
		var allMatchIDs []int64
		if f.MatchIDsFilter != "" {
			if ids, err := parseFilterIDList(f.MatchIDsFilter); err == nil {
				allMatchIDs = append(allMatchIDs, ids...)
			}
		}
		if f.TournamentIDsFilter != "" {
			if tIDs, err := parseFilterIDList(f.TournamentIDsFilter); err == nil {
				for _, tID := range tIDs {
					if matchIDs, err := d.getMatchIDsForTournament(tID); err == nil {
						allMatchIDs = append(allMatchIDs, matchIDs...)
					}
				}
			}
		}
		if len(allMatchIDs) > 0 {
			placeholders := strings.Repeat("?,", len(allMatchIDs))
			placeholders = placeholders[:len(placeholders)-1]
			where.WriteString(
				" AND p.id IN (SELECT m.position_id FROM move m" +
					" WHERE m.game_id IN (SELECT id FROM game WHERE match_id IN (" + placeholders + ")))")
			for _, id := range allMatchIDs {
				args = append(args, id)
			}
		} else {
			where.WriteString(" AND 0=1") // no matching IDs: force empty result
		}
	}

	// --- restrict to specific position IDs (e.g. "search in current results") ---
	if f.RestrictToPositionIDs != "" {
		var ids []int64
		for _, idStr := range strings.Split(f.RestrictToPositionIDs, ",") {
			if id, err := strconv.ParseInt(strings.TrimSpace(idStr), 10, 64); err == nil {
				ids = append(ids, id)
			}
		}
		if len(ids) > 0 {
			placeholders := strings.Repeat("?,", len(ids))
			placeholders = placeholders[:len(placeholders)-1]
			where.WriteString(" AND p.id IN (" + placeholders + ")")
			for _, id := range ids {
				args = append(args, id)
			}
		} else {
			where.WriteString(" AND 0=1")
		}
	}

	// --- orientation-specific SQL filters (disabled when f.MirrorFilter=true) ---
	var bitboardTight bool
	if useSQLFilters {
		if f.DecisionTypeFilter {
			where.WriteString(" AND p.decision_type = ? AND p.player_on_roll = ?")
			args = append(args, f.Filter.DecisionType, f.Filter.PlayerOnRoll)
		}
		if f.DiceRollFilter {
			d1, d2 := f.Filter.Dice[0], f.Filter.Dice[1]
			if d1 == d2 {
				where.WriteString(" AND p.dice_1 = ? AND p.dice_2 = ? AND p.player_on_roll = ? AND p.decision_type = ?")
				args = append(args, d1, d2, f.Filter.PlayerOnRoll, f.Filter.DecisionType)
			} else {
				where.WriteString(" AND ((p.dice_1 = ? AND p.dice_2 = ?) OR (p.dice_1 = ? AND p.dice_2 = ?)) AND p.player_on_roll = ? AND p.decision_type = ?")
				args = append(args, d1, d2, d2, d1, f.Filter.PlayerOnRoll, f.Filter.DecisionType)
			}
		}
		if f.IncludeCube {
			if f.Filter.Cube.Value == 0 {
				where.WriteString(" AND p.cube_value IS NULL")
			} else {
				where.WriteString(" AND p.cube_value = ? AND p.cube_owner = ?")
				args = append(args, f.Filter.Cube.Value, f.Filter.Cube.Owner)
			}
		}
		if f.IncludeScore {
			where.WriteString(" AND p.score_1 = ? AND p.score_2 = ?")
			args = append(args, f.Filter.Score[0], f.Filter.Score[1])
		}
		if f.NoContactFilter {
			where.WriteString(" AND p.no_contact = 1")
		}

		// Integer range filters on position scalar columns.
		pMin, pMax, pHasMin, pHasMax := parseIntFilterExpr(f.PipCountFilter, "p")
		appendIntRangeSQL("p.pip_diff", pMin, pMax, pHasMin, pHasMax, &where, &args)
		PMin, PMax, PHasMin, PHasMax := parseIntFilterExpr(f.Player1AbsolutePipCountFilter, "P")
		appendIntRangeSQL("p.pip_1", PMin, PMax, PHasMin, PHasMax, &where, &args)
		oMin, oMax, oHasMin, oHasMax := parseIntFilterExpr(f.Player1CheckerOffFilter, "o")
		appendIntRangeSQL("p.off_1", oMin, oMax, oHasMin, oHasMax, &where, &args)
		OMin, OMax, OHasMin, OHasMax := parseIntFilterExpr(f.Player2CheckerOffFilter, "O")
		appendIntRangeSQL("p.off_2", OMin, OMax, OHasMin, OHasMax, &where, &args)
		kMin, kMax, kHasMin, kHasMax := parseIntFilterExpr(f.Player1BackCheckerFilter, "k")
		appendIntRangeSQL("p.back_checkers_1", kMin, kMax, kHasMin, kHasMax, &where, &args)
		KMin, KMax, KHasMin, KHasMax := parseIntFilterExpr(f.Player2BackCheckerFilter, "K")
		appendIntRangeSQL("p.back_checkers_2", KMin, KMax, KHasMin, KHasMax, &where, &args)

		// Integer range filters on analysis scalar columns (LEFT JOIN; NULL rows are excluded
		// by the BETWEEN/>=/<= comparisons, which is correct: no analysis → no rate).
		// Rates are stored as integer × 100 (hundredths of percent); scale user input accordingly.
		wMin, wMax, wHasMin, wHasMax := parseFloatFilterExpr(f.WinRateFilter, "w")
		appendIntRangeSQL("a.player1_win_rate", int(math.Round(wMin*100)), int(math.Round(wMax*100)), wHasMin, wHasMax, &where, &args)
		gMin, gMax, gHasMin, gHasMax := parseFloatFilterExpr(f.GammonRateFilter, "g")
		appendIntRangeSQL("a.player1_gammon_rate", int(math.Round(gMin*100)), int(math.Round(gMax*100)), gHasMin, gHasMax, &where, &args)
		bMin, bMax, bHasMin, bHasMax := parseFloatFilterExpr(f.BackgammonRateFilter, "b")
		appendIntRangeSQL("a.player1_backgammon_rate", int(math.Round(bMin*100)), int(math.Round(bMax*100)), bHasMin, bHasMax, &where, &args)
		WMin, WMax, WHasMin, WHasMax := parseFloatFilterExpr(f.Player2WinRateFilter, "W")
		appendIntRangeSQL("a.player2_win_rate", int(math.Round(WMin*100)), int(math.Round(WMax*100)), WHasMin, WHasMax, &where, &args)
		GMin, GMax, GHasMin, GHasMax := parseFloatFilterExpr(f.Player2GammonRateFilter, "G")
		appendIntRangeSQL("a.player2_gammon_rate", int(math.Round(GMin*100)), int(math.Round(GMax*100)), GHasMin, GHasMax, &where, &args)
		BMin, BMax, BHasMin, BHasMax := parseFloatFilterExpr(f.Player2BackgammonRateFilter, "B")
		appendIntRangeSQL("a.player2_backgammon_rate", int(math.Round(BMin*100)), int(math.Round(BMax*100)), BHasMin, BHasMax, &where, &args)

		// Move-error f.Filter: "E"-prefixed values are millipoints; columns are now integer millipoints.
		// Both best_move_equity_error and cube_error are stored as non-negative absolute values.
		// Use the appropriate column based on decision type to avoid false positives
		// (e.g. cube positions always have best_move_equity_error=0, which would match range 0-N).
		if f.MoveErrorFilter != "" {
			eMin, eMax, eHasMin, eHasMax := parseFloatFilterExpr(f.MoveErrorFilter, "E")
			eqMin := int(math.Round(eMin))
			eqMax := int(math.Round(eMax))
			// Pick the error column that matches the position's decision type:
			// decision_type=0 (checker) → best_move_equity_error
			// decision_type=1 (cube)    → cube_error
			// When no decision_type f.Filter is active, check both with CASE.
			errExpr := "CASE WHEN p.decision_type = 1 THEN a.cube_error ELSE a.best_move_equity_error END"
			if eHasMin && eHasMax {
				where.WriteString(" AND " + errExpr + " BETWEEN ? AND ?")
				args = append(args, eqMin, eqMax)
			} else if eHasMin {
				where.WriteString(" AND " + errExpr + " >= ?")
				args = append(args, eqMin)
			} else if eHasMax {
				where.WriteString(" AND " + errExpr + " <= ?")
				args = append(args, eqMax)
			}
		}

		// Bitboard pre-f.Filter for checker-structure patterns.
		if hasBoardFilter(f.Filter.Board) {
			occ1Req, pt1Req, occ2Req, pt2Req, tight := CheckerStructureMasks(f.Filter)
			bitboardTight = tight
			where.WriteString(" AND (p.occupancy_1 & ?) = ? AND (p.point_mask_1 & ?) = ?")
			where.WriteString(" AND (p.occupancy_2 & ?) = ? AND (p.point_mask_2 & ?) = ?")
			args = append(args,
				int64(occ1Req), int64(occ1Req), int64(pt1Req), int64(pt1Req),
				int64(occ2Req), int64(occ2Req), int64(pt2Req), int64(pt2Req))
		}
	}

	// -----------------------------------------------------------------------
	// 2. Execute the single LEFT JOIN query
	// -----------------------------------------------------------------------
	query := `SELECT p.id, p.state,
		p.decision_type, p.player_on_roll, p.dice_1, p.dice_2,
		p.cube_value, p.cube_owner, p.score_1, p.score_2,
		p.has_jacoby, p.has_beaver,
		a.id, a.data,
		a.cube_error, a.best_move_equity_error,
		a.player1_win_rate, a.player1_gammon_rate, a.player1_backgammon_rate,
		a.player2_win_rate, a.player2_gammon_rate, a.player2_backgammon_rate,
		a.best_cube_action
	FROM position p
	LEFT JOIN analysis a ON a.position_id = p.id
	WHERE ` + where.String() + ` ORDER BY p.id`

	d.mu.RLock()
	rows, err := d.db.Query(query, args...)
	d.mu.RUnlock()
	if err != nil {
		return nil, nil, fmt.Errorf("search query: %w", err)
	}
	defer rows.Close()

	// -----------------------------------------------------------------------
	// 3. Scan rows and apply Go-side filters on the shrunk result set
	// -----------------------------------------------------------------------
	var positions []Position
	analysisMap := make(map[int64]*PositionAnalysis)

	for rows.Next() {
		var posID int64
		var posJSON string
		var pDT, pPOR, pD1, pD2, pCV, pCO, pS1, pS2, pHJ, pHB sql.NullInt64
		var anaID sql.NullInt64
		var anaJSON sql.NullString
		var cubeError, moveError sql.NullFloat64
		var p1Win, p1Gammon, p1BG, p2Win, p2Gammon, p2BG sql.NullFloat64
		var bestCubeAction sql.NullString

		if err := rows.Scan(
			&posID, &posJSON,
			&pDT, &pPOR, &pD1, &pD2, &pCV, &pCO, &pS1, &pS2, &pHJ, &pHB,
			&anaID, &anaJSON,
			&cubeError, &moveError,
			&p1Win, &p1Gammon, &p1BG,
			&p2Win, &p2Gammon, &p2BG,
			&bestCubeAction,
		); err != nil {
			return nil, nil, err
		}

		position := reconstructPosition(posID, posJSON,
			int(pDT.Int64), int(pPOR.Int64), int(pD1.Int64), int(pD2.Int64),
			int(pCV.Int64), int(pCO.Int64), int(pS1.Int64), int(pS2.Int64),
			int(pHJ.Int64), int(pHB.Int64))

		// Parse analysis from the JOIN — one unmarshal per row, no extra DB round-trip.
		var ana *PositionAnalysis
		if anaID.Valid && anaJSON.Valid && anaJSON.String != "" {
			var a PositionAnalysis
			if jsonErr := json.Unmarshal([]byte(anaJSON.String), &a); jsonErr == nil {
				ana = &a
				analysisMap[posID] = ana
			}
		}

		// matchesGoFilters evaluates the predicates that cannot be (or were not)
		// pushed to SQL: tight checker-structure, orientation-specific filters in
		// mirror mode, and always-Go-side filters (zone, blot, f.SearchText, equity).
		matchesGoFilters := func(pos Position) bool {
			// Checker structure: re-check when bitboard pre-f.Filter couldn't fully
			// discriminate (tight=true: exact count > 2) or when SQL was disabled.
			if hasBoardFilter(f.Filter.Board) {
				if !useSQLFilters || bitboardTight {
					if !pos.MatchesCheckerPosition(f.Filter) {
						return false
					}
				}
			}

			// Orientation-specific filters evaluated in Go when f.MirrorFilter=true.
			if !useSQLFilters {
				if !pos.MatchesCheckerPosition(f.Filter) {
					return false
				}
				if f.IncludeCube && !pos.MatchesCubePosition(f.Filter) {
					return false
				}
				if f.IncludeScore && !pos.MatchesScorePosition(f.Filter) {
					return false
				}
				if f.DecisionTypeFilter && !pos.MatchesDecisionType(f.Filter) {
					return false
				}
				if f.DiceRollFilter && !pos.MatchesDiceRoll(f.Filter) {
					return false
				}
				if f.NoContactFilter && !pos.MatchesNoContact() {
					return false
				}
				if f.PipCountFilter != "" && !pos.MatchesPipCountFilter(f.PipCountFilter) {
					return false
				}
				if f.Player1AbsolutePipCountFilter != "" && !pos.MatchesPlayer1AbsolutePipCount(f.Player1AbsolutePipCountFilter) {
					return false
				}
				if f.Player1CheckerOffFilter != "" && !pos.MatchesPlayer1CheckerOff(f.Player1CheckerOffFilter) {
					return false
				}
				if f.Player2CheckerOffFilter != "" && !pos.MatchesPlayer2CheckerOff(f.Player2CheckerOffFilter) {
					return false
				}
				if f.Player1BackCheckerFilter != "" && !pos.MatchesPlayer1BackChecker(f.Player1BackCheckerFilter) {
					return false
				}
				if f.Player2BackCheckerFilter != "" && !pos.MatchesPlayer2BackChecker(f.Player2BackCheckerFilter) {
					return false
				}
				// Analysis-rate filters in mirror mode: use pre-fetched analysis.
				if f.WinRateFilter != "" {
					if ana == nil {
						return false
					}
					var wr float64
					if ana.DoublingCubeAnalysis != nil {
						wr = ana.DoublingCubeAnalysis.PlayerWinChances
					} else if ana.CheckerAnalysis != nil && len(ana.CheckerAnalysis.Moves) > 0 {
						wr = ana.CheckerAnalysis.Moves[0].PlayerWinChance
					} else {
						return false
					}
					if !analysisMatchesFloatFilter(f.WinRateFilter, "w", wr) {
						return false
					}
				}
				if f.GammonRateFilter != "" {
					if ana == nil {
						return false
					}
					var gr float64
					if ana.DoublingCubeAnalysis != nil {
						gr = ana.DoublingCubeAnalysis.PlayerGammonChances
					} else if ana.CheckerAnalysis != nil && len(ana.CheckerAnalysis.Moves) > 0 {
						gr = ana.CheckerAnalysis.Moves[0].PlayerGammonChance
					} else {
						return false
					}
					if !analysisMatchesFloatFilter(f.GammonRateFilter, "g", gr) {
						return false
					}
				}
				if f.BackgammonRateFilter != "" {
					if ana == nil {
						return false
					}
					var bgr float64
					if ana.DoublingCubeAnalysis != nil {
						bgr = ana.DoublingCubeAnalysis.PlayerBackgammonChances
					} else if ana.CheckerAnalysis != nil && len(ana.CheckerAnalysis.Moves) > 0 {
						bgr = ana.CheckerAnalysis.Moves[0].PlayerBackgammonChance
					} else {
						return false
					}
					if !analysisMatchesFloatFilter(f.BackgammonRateFilter, "b", bgr) {
						return false
					}
				}
				if f.Player2WinRateFilter != "" {
					if ana == nil {
						return false
					}
					var wr float64
					if ana.DoublingCubeAnalysis != nil {
						wr = ana.DoublingCubeAnalysis.OpponentWinChances
					} else if ana.CheckerAnalysis != nil && len(ana.CheckerAnalysis.Moves) > 0 {
						wr = ana.CheckerAnalysis.Moves[0].OpponentWinChance
					} else {
						return false
					}
					if !analysisMatchesFloatFilter(f.Player2WinRateFilter, "W", wr) {
						return false
					}
				}
				if f.Player2GammonRateFilter != "" {
					if ana == nil {
						return false
					}
					var gr float64
					if ana.DoublingCubeAnalysis != nil {
						gr = ana.DoublingCubeAnalysis.OpponentGammonChances
					} else if ana.CheckerAnalysis != nil && len(ana.CheckerAnalysis.Moves) > 0 {
						gr = ana.CheckerAnalysis.Moves[0].OpponentGammonChance
					} else {
						return false
					}
					if !analysisMatchesFloatFilter(f.Player2GammonRateFilter, "G", gr) {
						return false
					}
				}
				if f.Player2BackgammonRateFilter != "" {
					if ana == nil {
						return false
					}
					var bgr float64
					if ana.DoublingCubeAnalysis != nil {
						bgr = ana.DoublingCubeAnalysis.OpponentBackgammonChances
					} else if ana.CheckerAnalysis != nil && len(ana.CheckerAnalysis.Moves) > 0 {
						bgr = ana.CheckerAnalysis.Moves[0].OpponentBackgammonChance
					} else {
						return false
					}
					if !analysisMatchesFloatFilter(f.Player2BackgammonRateFilter, "B", bgr) {
						return false
					}
				}
				// Move-error in mirror mode: fall back to the existing method.
				if f.MoveErrorFilter != "" && !pos.MatchesMoveErrorFilter(f.MoveErrorFilter, d) {
					return false
				}
			}

			// Always Go-side: zone, blot, f.SearchText, f.DateFilter, f.EquityFilter.
			if f.Player1CheckerInZoneFilter != "" && !pos.MatchesPlayer1CheckerInZone(f.Player1CheckerInZoneFilter) {
				return false
			}
			if f.Player2CheckerInZoneFilter != "" && !pos.MatchesPlayer2CheckerInZone(f.Player2CheckerInZoneFilter) {
				return false
			}
			if f.Player1OutfieldBlotFilter != "" && !pos.MatchesPlayer1OutfieldBlot(f.Player1OutfieldBlotFilter) {
				return false
			}
			if f.Player2OutfieldBlotFilter != "" && !pos.MatchesPlayer2OutfieldBlot(f.Player2OutfieldBlotFilter) {
				return false
			}
			if f.Player1JanBlotFilter != "" && !pos.MatchesPlayer1JanBlot(f.Player1JanBlotFilter) {
				return false
			}
			if f.Player2JanBlotFilter != "" && !pos.MatchesPlayer2JanBlot(f.Player2JanBlotFilter) {
				return false
			}
			if f.SearchText != "" && !pos.MatchesSearchText(f.SearchText, d) {
				return false
			}
			if f.DateFilter != "" && !pos.MatchesDateFilter(f.DateFilter, d) {
				return false
			}
			if f.EquityFilter != "" && !analysisMatchesEquityFilter(f.EquityFilter, ana) {
				return false
			}
			return true
		}

		// addPosition mirrors take/pass cube positions so player1 appears at the bottom.
		addPosition := func(pos Position) {
			if f.MoveErrorFilter != "" && pos.DecisionType == CubeAction && pos.IsPlayer1TakePassCubeAction(d) {
				pos = pos.Mirror()
			}
			positions = append(positions, pos)
		}

		if matchesGoFilters(position) {
			if analysisMatchesMovePattern(f.MovePatternFilter, ana) {
				addPosition(position)
			}
		} else if f.MirrorFilter {
			mirrored := position.Mirror()
			if matchesGoFilters(mirrored) {
				if analysisMatchesMovePattern(f.MovePatternFilter, ana) {
					addPosition(mirrored)
				}
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return positions, analysisMap, nil
}

// LoadPositionsByFilters returns positions matching the supplied filters.
// This is the public Wails-bound method that accepts a single SearchFilters struct.
// Internally it delegates to loadPositionsByFiltersCore and discards the analysis map.
func (d *Database) LoadPositionsByFilters(f SearchFilters) ([]Position, error) {
	positions, _, err := d.loadPositionsByFiltersCore(f)
	return positions, err
}

// parseFilterIDList parses a match/tournament ID filter string.
// Supports: "5" (single), "2,7" (range from 2 to 7), or multiple IDs passed
// as a pre-joined comma-separated list from the frontend.
func parseFilterIDList(s string) ([]int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	parts := strings.Split(s, ",")
	if len(parts) == 2 {
		// Could be a range (e.g., "2,7" means IDs 2 through 7)
		start, err1 := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
		end, err2 := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
		if err1 == nil && err2 == nil && end > start {
			var ids []int64
			for i := start; i <= end; i++ {
				ids = append(ids, i)
			}
			return ids, nil
		}
	}
	// Otherwise treat as explicit list of IDs separated by ";"
	parts = strings.Split(s, ";")
	var ids []int64
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := strconv.ParseInt(p, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid ID %q: %v", p, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// getPositionIDsForMatch returns all position IDs linked to a given match.
func (d *Database) getPositionIDsForMatch(matchID int64) ([]int64, error) {
	d.mu.RLock()
	rows, err := d.db.Query(`
		SELECT DISTINCT mv.position_id
		FROM move mv
		INNER JOIN game g ON mv.game_id = g.id
		WHERE g.match_id = ?
	`, matchID)
	d.mu.RUnlock()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// getMatchIDsForTournament returns all match IDs belonging to a tournament.
func (d *Database) getMatchIDsForTournament(tournamentID int64) ([]int64, error) {
	d.mu.RLock()
	rows, err := d.db.Query(`SELECT id FROM match WHERE tournament_id = ?`, tournamentID)
	d.mu.RUnlock()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
