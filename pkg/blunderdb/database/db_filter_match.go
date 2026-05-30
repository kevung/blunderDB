package database

import (
	"database/sql"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"time"
)

// The functions below evaluate position filters that need database access
// (analysis, comments, played moves). Pure board-only predicates live as
// methods on domain.Position in the domain package.

func MatchesSearchText(p *Position, searchText string, d *Database) bool {
	keywords := parseSearchTextKeywords(searchText)
	if len(keywords) == 0 {
		return false
	}
	comment, err := d.LoadComment(p.ID)
	if err != nil {
		slog.Warn("loading comment for position", "positionID", p.ID, "err", err)
		return false
	}
	comment = strings.ToLower(comment)
	for _, kw := range keywords {
		if strings.Contains(comment, kw) {
			return true
		}
	}
	return false
}

// parseSearchTextKeywords extracts the lowercased, trimmed, non-empty keywords
// from a t"tag1;tag2;..." search filter. It strips the frontend's t"..."
// wrapper, splits on ';', trims whitespace around each tag, and drops empty
// tags (so a stray trailing ';' or surrounding spaces no longer match every
// comment or fail to match a valid tag).
func parseSearchTextKeywords(searchText string) []string {
	s := strings.TrimSpace(searchText)
	// Strip the t"..." wrapper: a leading 't' immediately followed by a
	// quote, then the surrounding quotes.
	if len(s) >= 2 && s[0] == 't' && (s[1] == '"' || s[1] == '\'') {
		s = s[1:]
	}
	s = strings.ToLower(strings.Trim(s, `"'`))
	var keywords []string
	for _, kw := range strings.Split(s, ";") {
		if kw = strings.TrimSpace(kw); kw != "" {
			keywords = append(keywords, kw)
		}
	}
	return keywords
}

func MatchesPlayer2BackgammonRate(p *Position, filter string, d *Database) bool {
	analysis, err := d.LoadAnalysis(p.ID)
	if err != nil || analysis == nil {
		return false
	}

	var backgammonRate float64
	if analysis.AnalysisType == "DoublingCube" && analysis.DoublingCubeAnalysis != nil {
		backgammonRate = analysis.DoublingCubeAnalysis.OpponentBackgammonChances
	} else if analysis.AnalysisType == "CheckerMove" && analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 0 {
		backgammonRate = analysis.CheckerAnalysis.Moves[0].OpponentBackgammonChance
	} else {
		return false
	}

	backgammonRate = roundToHundredthPercent(backgammonRate)

	if strings.HasPrefix(filter, "B>") && !strings.HasPrefix(filter, "BO>") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			slog.Warn("parsing filter value", "value", filter[2:])
			return false
		}
		return backgammonRate >= value
	} else if strings.HasPrefix(filter, "B<") && !strings.HasPrefix(filter, "BO<") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			slog.Warn("parsing filter value", "value", filter[2:])
			return false
		}
		return backgammonRate <= value
	} else if strings.HasPrefix(filter, "B") && !strings.HasPrefix(filter, "BO") {
		values := strings.Split(filter[1:], ",")
		if len(values) != 2 {
			slog.Warn("parsing filter values", "value", filter[1:])
			return false
		}
		value1, err1 := strconv.ParseFloat(values[0], 64)
		value2, err2 := strconv.ParseFloat(values[1], 64)
		if err1 != nil || err2 != nil {
			slog.Warn("parsing filter values", "v1", values[0], "v2", values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return backgammonRate >= minValue && backgammonRate <= maxValue
	}
	return false
}

func MatchesPlayer2GammonRate(p *Position, filter string, d *Database) bool {
	analysis, err := d.LoadAnalysis(p.ID)
	if err != nil || analysis == nil {
		return false
	}

	var gammonRate float64
	if analysis.AnalysisType == "DoublingCube" && analysis.DoublingCubeAnalysis != nil {
		gammonRate = analysis.DoublingCubeAnalysis.OpponentGammonChances
	} else if analysis.AnalysisType == "CheckerMove" && analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 0 {
		gammonRate = analysis.CheckerAnalysis.Moves[0].OpponentGammonChance
	} else {
		return false
	}

	gammonRate = roundToHundredthPercent(gammonRate)

	if strings.HasPrefix(filter, "G>") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			slog.Warn("parsing filter value", "value", filter[2:])
			return false
		}
		return gammonRate >= value
	} else if strings.HasPrefix(filter, "G<") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			slog.Warn("parsing filter value", "value", filter[2:])
			return false
		}
		return gammonRate <= value
	} else if strings.HasPrefix(filter, "G") {
		values := strings.Split(filter[1:], ",")
		if len(values) != 2 {
			slog.Warn("parsing filter values", "value", filter[1:])
			return false
		}
		value1, err1 := strconv.ParseFloat(values[0], 64)
		value2, err2 := strconv.ParseFloat(values[1], 64)
		if err1 != nil || err2 != nil {
			slog.Warn("parsing filter values", "v1", values[0], "v2", values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return gammonRate >= minValue && gammonRate <= maxValue
	}
	return false
}

func MatchesWinRate(p *Position, filter string, d *Database) bool {
	analysis, err := d.LoadAnalysis(p.ID)
	if err != nil || analysis == nil {
		return false
	}

	var winRate float64
	if analysis.AnalysisType == "DoublingCube" && analysis.DoublingCubeAnalysis != nil {
		winRate = analysis.DoublingCubeAnalysis.PlayerWinChances
	} else if analysis.AnalysisType == "CheckerMove" && analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 0 {
		winRate = analysis.CheckerAnalysis.Moves[0].PlayerWinChance
	} else {
		return false
	}

	winRate = roundToHundredthPercent(winRate)

	if strings.HasPrefix(filter, "w>") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			slog.Warn("parsing filter value", "value", filter[2:])
			return false
		}
		return winRate >= value
	} else if strings.HasPrefix(filter, "w<") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			slog.Warn("parsing filter value", "value", filter[2:])
			return false
		}
		return winRate <= value
	} else if strings.HasPrefix(filter, "w") {
		values := strings.Split(filter[1:], ",")
		if len(values) != 2 {
			slog.Warn("parsing filter values", "value", filter[1:])
			return false
		}
		value1, err1 := strconv.ParseFloat(values[0], 64)
		value2, err2 := strconv.ParseFloat(values[1], 64)
		if err1 != nil || err2 != nil {
			slog.Warn("parsing filter values", "v1", values[0], "v2", values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return winRate >= minValue && winRate <= maxValue
	}
	return false
}

func MatchesPlayer2WinRate(p *Position, filter string, d *Database) bool {
	analysis, err := d.LoadAnalysis(p.ID)
	if err != nil || analysis == nil {
		return false
	}

	var winRate float64
	if analysis.AnalysisType == "DoublingCube" && analysis.DoublingCubeAnalysis != nil {
		winRate = analysis.DoublingCubeAnalysis.OpponentWinChances
	} else if analysis.AnalysisType == "CheckerMove" && analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 0 {
		winRate = analysis.CheckerAnalysis.Moves[0].OpponentWinChance
	} else {
		return false
	}

	winRate = roundToHundredthPercent(winRate)

	if strings.HasPrefix(filter, "W>") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			slog.Warn("parsing filter value", "value", filter[2:])
			return false
		}
		return winRate >= value
	} else if strings.HasPrefix(filter, "W<") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			slog.Warn("parsing filter value", "value", filter[2:])
			return false
		}
		return winRate <= value
	} else if strings.HasPrefix(filter, "W") {
		values := strings.Split(filter[1:], ",")
		if len(values) != 2 {
			slog.Warn("parsing filter values", "value", filter[1:])
			return false
		}
		value1, err1 := strconv.ParseFloat(values[0], 64)
		value2, err2 := strconv.ParseFloat(values[1], 64)
		if err1 != nil || err2 != nil {
			slog.Warn("parsing filter values", "v1", values[0], "v2", values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return winRate >= minValue && winRate <= maxValue
	}
	return false
}

func MatchesGammonRate(p *Position, filter string, d *Database) bool {
	analysis, err := d.LoadAnalysis(p.ID)
	if err != nil || analysis == nil {
		return false
	}

	var gammonRate float64
	if analysis.AnalysisType == "DoublingCube" && analysis.DoublingCubeAnalysis != nil {
		gammonRate = analysis.DoublingCubeAnalysis.PlayerGammonChances
	} else if analysis.AnalysisType == "CheckerMove" && analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 0 {
		gammonRate = analysis.CheckerAnalysis.Moves[0].PlayerGammonChance
	} else {
		return false
	}

	gammonRate = roundToHundredthPercent(gammonRate)

	if strings.HasPrefix(filter, "g>") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			slog.Warn("parsing filter value", "value", filter[2:])
			return false
		}
		return gammonRate >= value
	} else if strings.HasPrefix(filter, "g<") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			slog.Warn("parsing filter value", "value", filter[2:])
			return false
		}
		return gammonRate <= value
	} else if strings.HasPrefix(filter, "g") {
		values := strings.Split(filter[1:], ",")
		if len(values) != 2 {
			slog.Warn("parsing filter values", "value", filter[1:])
			return false
		}
		value1, err1 := strconv.ParseFloat(values[0], 64)
		value2, err2 := strconv.ParseFloat(values[1], 64)
		if err1 != nil || err2 != nil {
			slog.Warn("parsing filter values", "v1", values[0], "v2", values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return gammonRate >= minValue && gammonRate <= maxValue
	}
	return false
}

func MatchesBackgammonRate(p *Position, filter string, d *Database) bool {
	analysis, err := d.LoadAnalysis(p.ID)
	if err != nil || analysis == nil {
		return false
	}

	var backgammonRate float64
	if analysis.AnalysisType == "DoublingCube" && analysis.DoublingCubeAnalysis != nil {
		backgammonRate = analysis.DoublingCubeAnalysis.PlayerBackgammonChances
	} else if analysis.AnalysisType == "CheckerMove" && analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 0 {
		backgammonRate = analysis.CheckerAnalysis.Moves[0].PlayerBackgammonChance
	} else {
		return false
	}

	backgammonRate = roundToHundredthPercent(backgammonRate)

	if strings.HasPrefix(filter, "b>") && !strings.HasPrefix(filter, "bo>") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			slog.Warn("parsing filter value", "value", filter[2:])
			return false
		}
		return backgammonRate >= value
	} else if strings.HasPrefix(filter, "b<") && !strings.HasPrefix(filter, "bo<") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			slog.Warn("parsing filter value", "value", filter[2:])
			return false
		}
		return backgammonRate <= value
	} else if strings.HasPrefix(filter, "b") && !strings.HasPrefix(filter, "bo") {
		values := strings.Split(filter[1:], ",")
		if len(values) != 2 {
			slog.Warn("parsing filter values", "value", filter[1:])
			return false
		}
		value1, err1 := strconv.ParseFloat(values[0], 64)
		value2, err2 := strconv.ParseFloat(values[1], 64)
		if err1 != nil || err2 != nil {
			slog.Warn("parsing filter values", "v1", values[0], "v2", values[1])
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return backgammonRate >= minValue && backgammonRate <= maxValue
	}
	return false
}

func MatchesEquityFilter(p *Position, filter string, d *Database) bool {
	analysis, err := d.LoadAnalysis(p.ID)
	if err != nil || analysis == nil {
		return false
	}

	var equity float64
	if analysis.AnalysisType == "DoublingCube" && analysis.DoublingCubeAnalysis != nil {
		equity = analysis.DoublingCubeAnalysis.CubefulNoDoubleEquity
	} else if analysis.AnalysisType == "CheckerMove" && analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 0 {
		equity = analysis.CheckerAnalysis.Moves[0].Equity
	} else {
		return false
	}

	equity = roundToMillipoint(equity)

	if strings.HasPrefix(filter, "e>") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			slog.Warn("parsing filter value", "value", filter[2:])
			return false
		}
		value /= 1000 // Convert millipoints to points
		return equity >= value
	} else if strings.HasPrefix(filter, "e<") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			slog.Warn("parsing filter value", "value", filter[2:])
			return false
		}
		value /= 1000 // Convert millipoints to points
		return equity <= value
	} else if strings.HasPrefix(filter, "e") {
		values := strings.Split(filter[1:], ",")
		if len(values) != 2 {
			slog.Warn("parsing filter values", "value", filter[1:])
			return false
		}
		value1, err1 := strconv.ParseFloat(values[0], 64)
		value2, err2 := strconv.ParseFloat(values[1], 64)
		if err1 != nil || err2 != nil {
			slog.Warn("parsing filter values", "v1", values[0], "v2", values[1])
			return false
		}
		value1 /= 1000 // Convert millipoints to points
		value2 /= 1000 // Convert millipoints to points
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return equity >= minValue && equity <= maxValue
	}
	return false
}

// getPlayer1MovesForPosition returns checker moves and cube actions played by player1 for a position.
// Player1 is identified by player=1 in XG encoding in the move table.
func (d *Database) getPlayer1MovesForPosition(positionID int64) ([]string, []string) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.db.Query(`SELECT checker_move, cube_action FROM move WHERE position_id = ? AND player = 1`, positionID)
	if err != nil {
		return nil, nil
	}
	defer rows.Close()

	checkerMoves := make(map[string]bool)
	cubeActions := make(map[string]bool)
	for rows.Next() {
		var cm sql.NullString
		var ca sql.NullString
		if err := rows.Scan(&cm, &ca); err != nil {
			continue
		}
		if cm.Valid && cm.String != "" {
			checkerMoves[normalizeMove(cm.String)] = true
		}
		if ca.Valid && ca.String != "" {
			cubeActions[ca.String] = true
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil
	}

	var checkerMovesList []string
	for m := range checkerMoves {
		checkerMovesList = append(checkerMovesList, m)
	}
	var cubeActionsList []string
	for a := range cubeActions {
		cubeActionsList = append(cubeActionsList, a)
	}
	return checkerMovesList, cubeActionsList
}

// IsPlayer1TakePassCubeAction returns true if player1's cube action for this position
// was a take or pass (as opposed to double or no-double).
// This is used to determine board orientation: take/pass positions should be shown
// from the taker's perspective (mirrored) so player1 appears at the bottom.
func IsPlayer1TakePassCubeAction(p *Position, d *Database) bool {
	_, player1CubeActions := d.getPlayer1MovesForPosition(p.ID)
	for _, action := range player1CubeActions {
		actionLower := strings.ToLower(action)
		if strings.Contains(actionLower, "take") || actionLower == "dt" ||
			strings.Contains(actionLower, "pass") || strings.Contains(actionLower, "drop") || actionLower == "dp" {
			return true
		}
	}
	return false
}

// MatchesMoveErrorFilter filters positions by the equity error of the played move (in millipoints).
// By default, only considers errors made by player1 (player1 in match context).
// Supports E>x, E<x, Ex,y syntax.
func MatchesMoveErrorFilter(p *Position, filter string, d *Database) bool {
	analysis, err := d.LoadAnalysis(p.ID)
	if err != nil || analysis == nil {
		return false
	}

	// Get only player1's moves for this position from the move table
	player1CheckerMoves, player1CubeActions := d.getPlayer1MovesForPosition(p.ID)

	var moveError float64
	found := false

	if analysis.AnalysisType == "CheckerMove" && analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 0 {
		// Use only player1's moves (from match context)
		playedMoves := player1CheckerMoves
		if len(playedMoves) == 0 {
			return false
		}
		// Find the played move in the analysis moves and get its error
		for _, played := range playedMoves {
			for i, m := range analysis.CheckerAnalysis.Moves {
				if strings.EqualFold(normalizeMove(m.Move), normalizeMove(played)) {
					if i == 0 {
						moveError = 0
					} else if m.EquityError != nil {
						moveError = math.Abs(*m.EquityError)
					}
					found = true
					break
				}
			}
			if found {
				break
			}
		}
	} else if analysis.AnalysisType == "DoublingCube" && analysis.DoublingCubeAnalysis != nil {
		// Use only player1's cube actions (from match context)
		playedActions := player1CubeActions
		if len(playedActions) == 0 {
			return false
		}
		bestAction := strings.ToLower(analysis.DoublingCubeAnalysis.BestCubeAction)
		for _, played := range playedActions {
			playedLower := strings.ToLower(played)
			if playedLower == bestAction {
				moveError = 0
				found = true
			} else {
				switch {
				case strings.Contains(playedLower, "no double") || playedLower == "nd":
					moveError = math.Abs(analysis.DoublingCubeAnalysis.CubefulNoDoubleError)
					found = true
				case strings.Contains(playedLower, "take") || playedLower == "dt":
					moveError = math.Abs(analysis.DoublingCubeAnalysis.CubefulDoubleTakeError)
					found = true
				case strings.Contains(playedLower, "pass") || strings.Contains(playedLower, "drop") || playedLower == "dp":
					moveError = math.Abs(analysis.DoublingCubeAnalysis.CubefulDoublePassError)
					found = true
				}
			}
			if found {
				break
			}
		}
	}

	if !found {
		return false
	}

	// Convert move error from equity points to millipoints and round to nearest millipoint
	moveErrorMillipoints := math.Round(moveError * 1000)

	if strings.HasPrefix(filter, "E>") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			return false
		}
		return moveErrorMillipoints >= value
	} else if strings.HasPrefix(filter, "E<") {
		value, err := strconv.ParseFloat(filter[2:], 64)
		if err != nil {
			return false
		}
		return moveErrorMillipoints <= value
	} else if strings.HasPrefix(filter, "E") {
		values := strings.Split(filter[1:], ",")
		if len(values) != 2 {
			return false
		}
		value1, err1 := strconv.ParseFloat(values[0], 64)
		value2, err2 := strconv.ParseFloat(values[1], 64)
		if err1 != nil || err2 != nil {
			return false
		}
		minValue := value1
		maxValue := value2
		if value1 > value2 {
			minValue = value2
			maxValue = value1
		}
		return moveErrorMillipoints >= minValue && moveErrorMillipoints <= maxValue
	}
	return false
}

func MatchesMovePattern(p *Position, filter string, d *Database) bool {
	analysis, err := d.LoadAnalysis(p.ID)
	if err != nil || analysis == nil {
		return false
	}

	// Extract the move pattern from the raw string
	movePatternMatch := strings.Trim(filter, `m"'`)
	movePatterns := strings.Split(strings.ToLower(movePatternMatch), ";")

	if analysis.AnalysisType == "CheckerMove" && analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 0 {
		move := strings.ToLower(analysis.CheckerAnalysis.Moves[0].Move)
		for _, pattern := range movePatterns {
			if strings.Contains(move, pattern) {
				return true
			}
		}
	} else if analysis.AnalysisType == "DoublingCube" && analysis.DoublingCubeAnalysis != nil {
		for _, pattern := range movePatterns {
			switch pattern {
			case "nd":
				if analysis.DoublingCubeAnalysis.CubefulNoDoubleError == 0 {
					return true
				}
			case "dt":
				if analysis.DoublingCubeAnalysis.CubefulDoubleTakeError == 0 {
					return true
				}
			case "dp":
				if analysis.DoublingCubeAnalysis.CubefulDoublePassError == 0 {
					return true
				}
			}
		}
	}
	return false
}

func MatchesDateFilter(p *Position, filter string, d *Database) bool {
	analysis, err := d.LoadAnalysis(p.ID)
	if err != nil || analysis == nil {
		return false
	}

	creationDate := analysis.CreationDate

	if strings.HasPrefix(filter, "T>") {
		dateStr := filter[2:]
		date, err := time.ParseInLocation("2006/01/02", dateStr, creationDate.Location())
		if err != nil {
			slog.Warn("parsing date filter value", "value", dateStr)
			return false
		}
		match := creationDate.After(date) || creationDate.Equal(date)
		return match
	} else if strings.HasPrefix(filter, "T<") {
		dateStr := filter[2:]
		date, err := time.ParseInLocation("2006/01/02", dateStr, creationDate.Location())
		if err != nil {
			slog.Warn("parsing date filter value", "value", dateStr)
			return false
		}
		date = date.Add(24 * time.Hour).Add(-1 * time.Second) // Include the entire day
		match := creationDate.Before(date)
		return match
	} else if strings.HasPrefix(filter, "T") {
		dateRange := strings.Split(filter[1:], ",")
		if len(dateRange) != 2 {
			slog.Warn("parsing date range filter values", "value", filter[1:])
			return false
		}
		startDate, err1 := time.ParseInLocation("2006/01/02", dateRange[0], creationDate.Location())
		endDate, err2 := time.ParseInLocation("2006/01/02", dateRange[1], creationDate.Location())
		if err1 != nil || err2 != nil {
			slog.Warn("parsing date range filter values", "v1", dateRange[0], "v2", dateRange[1])
			return false
		}
		if startDate.After(endDate) {
			startDate, endDate = endDate, startDate // Swap to ensure correct order
		}
		endDate = endDate.Add(24 * time.Hour).Add(-1 * time.Second) // Include the entire day
		match := (creationDate.After(startDate) || creationDate.Equal(startDate)) && (creationDate.Before(endDate) || creationDate.Equal(endDate))
		return match
	}
	return false
}
