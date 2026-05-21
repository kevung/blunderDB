package postgres

import (
	"context"
	"errors"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// This file holds the search-filter helpers, ported from the SQLite backend.
// The pure parsers are identical across backends; the predicates that need
// database access are re-expressed against the pgx execer. The query builders
// emit '?' placeholders and the assembled query is rebound to '$N' by rebind.

// rebind rewrites a query built with positional '?' placeholders into the
// PostgreSQL '$1, $2, …' form. The search query contains no string literals,
// so a straight sequential substitution is correct.
func rebind(query string) string {
	var b strings.Builder
	n := 0
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			n++
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(n))
			continue
		}
		b.WriteByte(query[i])
	}
	return b.String()
}

// parseIntFilterExpr parses prefixed integer filter strings (e.g. "p>5").
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

// appendIntRangeSQL appends "AND column op ?" to where and the bound(s) to
// args. The '?' placeholders are rebound to '$N' once the query is assembled.
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
func hasBoardFilter(b domain.Board) bool {
	for _, p := range b.Points {
		if p.Checkers > 0 && p.Color >= 0 {
			return true
		}
	}
	return false
}

// analysisMatchesFloatFilter checks value against a prefixed float filter string.
func analysisMatchesFloatFilter(filter, prefix string, value float64) bool {
	if filter == "" {
		return true
	}
	mn, mx, hasMin, hasMax := parseFloatFilterExpr(filter, prefix)
	if !hasMin && !hasMax {
		return true
	}
	value = engine.RoundToHundredthPercent(value)
	if hasMin && value < mn {
		return false
	}
	if hasMax && value > mx {
		return false
	}
	return true
}

// analysisMatchesEquityFilter checks the best-move equity of ana against the
// "e"-prefixed filter.
func analysisMatchesEquityFilter(filter string, ana *domain.PositionAnalysis) bool {
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
	equity = engine.RoundToMillipoint(equity)
	mn, mx, hasMin, hasMax := parseFloatFilterExpr(filter, "e")
	if !hasMin && !hasMax {
		return true
	}
	if hasMin && equity < mn/1000.0 {
		return false
	}
	if hasMax && equity > mx/1000.0 {
		return false
	}
	return true
}

// analysisMatchesMovePattern checks a move-pattern filter against pre-fetched
// analysis.
func analysisMatchesMovePattern(filter string, ana *domain.PositionAnalysis) bool {
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

// parseFilterIDList parses a match/tournament ID filter string.
func parseFilterIDList(s string) ([]int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	parts := strings.Split(s, ",")
	if len(parts) == 2 {
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
	parts = strings.Split(s, ";")
	var ids []int64
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := strconv.ParseInt(p, 10, 64)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// getMatchIDsForTournament returns all match IDs belonging to a tournament.
func getMatchIDsForTournament(ctx context.Context, db execer, tournamentID int64) ([]int64, error) {
	rows, err := db.Query(ctx, `SELECT id FROM match WHERE tournament_id = $1`, tournamentID)
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

// loadAnalysisJSON loads and decodes the stored analysis for a position,
// returning nil when none is present or on any error.
func loadAnalysisJSON(ctx context.Context, db execer, positionID int64) *domain.PositionAnalysis {
	var data []byte
	if err := db.QueryRow(ctx,
		`SELECT data FROM analysis WHERE position_id = $1`, positionID).Scan(&data); err != nil {
		return nil
	}
	a, err := engine.DecodeAnalysisFromStorage(data)
	if err != nil {
		return nil
	}
	return &a
}

// loadCommentText returns the concatenated comment text of a position.
func loadCommentText(ctx context.Context, db execer, positionID int64) (string, error) {
	var text string
	err := db.QueryRow(ctx, `SELECT text FROM comment WHERE position_id = $1`, positionID).Scan(&text)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return text, nil
}

// getPlayer1MovesForPosition returns player-1's checker moves and cube actions
// recorded in the move table for a position.
func getPlayer1MovesForPosition(ctx context.Context, db execer, positionID int64) ([]string, []string) {
	rows, err := db.Query(ctx,
		`SELECT checker_move, cube_action FROM move WHERE position_id = $1 AND player = 1`, positionID)
	if err != nil {
		return nil, nil
	}
	defer rows.Close()
	checkerMoves := make(map[string]bool)
	cubeActions := make(map[string]bool)
	for rows.Next() {
		var cm, ca *string
		if err := rows.Scan(&cm, &ca); err != nil {
			continue
		}
		if cm != nil && *cm != "" {
			checkerMoves[engine.NormalizeMove(*cm)] = true
		}
		if ca != nil && *ca != "" {
			cubeActions[*ca] = true
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil
	}
	var checkerMovesList, cubeActionsList []string
	for m := range checkerMoves {
		checkerMovesList = append(checkerMovesList, m)
	}
	for a := range cubeActions {
		cubeActionsList = append(cubeActionsList, a)
	}
	return checkerMovesList, cubeActionsList
}

// matchesSearchText reports whether a position's comment matches a "t"-filter.
func matchesSearchText(ctx context.Context, db execer, p *domain.Position, searchText string) bool {
	comment, err := loadCommentText(ctx, db, p.ID)
	if err != nil {
		return false
	}
	searchTextMatch := strings.Trim(searchText, ` t"'`)
	searchTextArray := strings.Split(strings.ToLower(searchTextMatch), ";")
	comment = strings.ToLower(comment)
	for _, text := range searchTextArray {
		if strings.Contains(comment, text) {
			return true
		}
	}
	return false
}

// isPlayer1TakePassCubeAction reports whether player-1's recorded cube action
// for a position was a take or pass.
func isPlayer1TakePassCubeAction(ctx context.Context, db execer, p *domain.Position) bool {
	_, player1CubeActions := getPlayer1MovesForPosition(ctx, db, p.ID)
	for _, action := range player1CubeActions {
		actionLower := strings.ToLower(action)
		if strings.Contains(actionLower, "take") || actionLower == "dt" ||
			strings.Contains(actionLower, "pass") || strings.Contains(actionLower, "drop") || actionLower == "dp" {
			return true
		}
	}
	return false
}

// matchesMoveErrorFilter filters positions by the equity error of player-1's
// played move (millipoints): E>x, E<x, Ex,y.
func matchesMoveErrorFilter(ctx context.Context, db execer, p *domain.Position, filter string) bool {
	analysis := loadAnalysisJSON(ctx, db, p.ID)
	if analysis == nil {
		return false
	}
	player1CheckerMoves, player1CubeActions := getPlayer1MovesForPosition(ctx, db, p.ID)

	var moveError float64
	found := false

	if analysis.AnalysisType == "CheckerMove" && analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 0 {
		playedMoves := player1CheckerMoves
		if len(playedMoves) == 0 {
			return false
		}
		for _, played := range playedMoves {
			for i, m := range analysis.CheckerAnalysis.Moves {
				if strings.EqualFold(engine.NormalizeMove(m.Move), engine.NormalizeMove(played)) {
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

// matchesDateFilter filters positions by the analysis creation date:
// T>d, T<d, Td1,d2.
func matchesDateFilter(ctx context.Context, db execer, p *domain.Position, filter string) bool {
	analysis := loadAnalysisJSON(ctx, db, p.ID)
	if analysis == nil {
		return false
	}
	creationDate := analysis.CreationDate

	if strings.HasPrefix(filter, "T>") {
		date, err := time.ParseInLocation("2006/01/02", filter[2:], creationDate.Location())
		if err != nil {
			return false
		}
		return creationDate.After(date) || creationDate.Equal(date)
	} else if strings.HasPrefix(filter, "T<") {
		date, err := time.ParseInLocation("2006/01/02", filter[2:], creationDate.Location())
		if err != nil {
			return false
		}
		date = date.Add(24 * time.Hour).Add(-1 * time.Second)
		return creationDate.Before(date)
	} else if strings.HasPrefix(filter, "T") {
		dateRange := strings.Split(filter[1:], ",")
		if len(dateRange) != 2 {
			return false
		}
		startDate, err1 := time.ParseInLocation("2006/01/02", dateRange[0], creationDate.Location())
		endDate, err2 := time.ParseInLocation("2006/01/02", dateRange[1], creationDate.Location())
		if err1 != nil || err2 != nil {
			return false
		}
		if startDate.After(endDate) {
			startDate, endDate = endDate, startDate
		}
		endDate = endDate.Add(24 * time.Hour).Add(-1 * time.Second)
		return (creationDate.After(startDate) || creationDate.Equal(startDate)) &&
			(creationDate.Before(endDate) || creationDate.Equal(endDate))
	}
	return false
}
