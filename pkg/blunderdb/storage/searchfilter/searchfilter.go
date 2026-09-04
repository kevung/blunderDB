// Package searchfilter holds the pure, backend-agnostic half of the position
// search: the parsers for the filter strings the command line and the search
// panel produce ("p>5", "e-100,100", "T2024/01/01,2024/12/31", "1,3;5"…) and
// the in-memory predicates that check a decoded analysis against them.
//
// Nothing in this package touches a database. The two storage backends
// (storage/sqlite, storage/postgres) call it from their Search implementation;
// the SQL they assemble, and the predicates that need a query (comment text,
// player-1 moves), stay in the backend. Keeping the pure part here means the
// filter semantics are written once and cannot drift between backends.
//
// AppendIntRangeSQL emits '?' placeholders. The SQLite backend binds them as
// they are; the PostgreSQL backend rebinds the assembled query to '$N' (see
// rebind in storage/postgres), so the output is dialect-neutral by
// construction.
package searchfilter

import (
	"strconv"
	"strings"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// ParseIntFilterExpr parses prefixed integer filter strings (e.g. "p>5").
func ParseIntFilterExpr(filter, prefix string) (min, max int, hasMin, hasMax bool) {
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

// ParseFloatFilterExpr is the float64 variant of ParseIntFilterExpr.
func ParseFloatFilterExpr(filter, prefix string) (min, max float64, hasMin, hasMax bool) {
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

// AppendIntRangeSQL appends "AND column op ?" to where and the bound(s) to args.
func AppendIntRangeSQL(column string, min, max int, hasMin, hasMax bool, where *strings.Builder, args *[]any) {
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

// HasBoardFilter returns true if at least one point in b has non-empty checkers.
func HasBoardFilter(b domain.Board) bool {
	for _, p := range b.Points {
		if p.Checkers > 0 && p.Color >= 0 {
			return true
		}
	}
	return false
}

// AnalysisMatchesFloatFilter checks value against a prefixed float filter string.
func AnalysisMatchesFloatFilter(filter, prefix string, value float64) bool {
	if filter == "" {
		return true
	}
	mn, mx, hasMin, hasMax := ParseFloatFilterExpr(filter, prefix)
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

// AnalysisMatchesEquityFilter checks the best-move equity of ana against the "e"-prefixed filter.
func AnalysisMatchesEquityFilter(filter string, ana *domain.PositionAnalysis) bool {
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
	mn, mx, hasMin, hasMax := ParseFloatFilterExpr(filter, "e")
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

// PlayerName unwraps the player filter the frontend sends. The command bar and
// the search panel both build the token `pl"Name"` and hand it over whole, the
// same way they do for t"…" (comment text) and m"…" (move pattern) — both of
// which are unwrapped where they are read (ParseSearchTextKeywords,
// AnalysisMatchesMovePattern). The player filter was not, so it reached the SQL
// comparison with its wrapper still on and matched no player at all: searching
// by player from the GUI silently returned nothing (B.18, #186).
//
// A bare name is returned unchanged, so the CLI and the server — which pass the
// name on its own — are unaffected.
func PlayerName(filter string) string {
	s := strings.TrimSpace(filter)
	if len(s) >= 4 && strings.HasPrefix(s, "pl") && (s[2] == '"' || s[2] == '\'') && s[len(s)-1] == s[2] {
		return s[3 : len(s)-1]
	}
	return s
}

// AnalysisMatchesMovePattern checks a move-pattern filter against pre-fetched analysis.
func AnalysisMatchesMovePattern(filter string, ana *domain.PositionAnalysis) bool {
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

// ParseFilterIDList parses a match/tournament ID filter string. It accepts a
// two-value comma range ("2,7" -> 2..7 inclusive), a semicolon-separated
// explicit list ("2;5;9"), a comma-separated explicit list ("2,5,9"), or any
// mix of comma and semicolon separators.
func ParseFilterIDList(s string) ([]int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	commaParts := strings.Split(s, ",")
	if len(commaParts) == 2 {
		start, err1 := strconv.ParseInt(strings.TrimSpace(commaParts[0]), 10, 64)
		end, err2 := strconv.ParseInt(strings.TrimSpace(commaParts[1]), 10, 64)
		if err1 == nil && err2 == nil && end > start {
			var ids []int64
			for i := start; i <= end; i++ {
				ids = append(ids, i)
			}
			return ids, nil
		}
	}
	// Not a two-value range: treat every comma-separated part as its own
	// semicolon-list parse, so "1,3,5", "1;3;5", and mixes of both work.
	var ids []int64
	for _, commaPart := range commaParts {
		for _, p := range strings.Split(commaPart, ";") {
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
	}
	return ids, nil
}

// ParseSearchTextKeywords extracts the lowercased, trimmed, non-empty keywords
// from a t"tag1;tag2;..." search filter. It strips the frontend's t"..."
// wrapper, splits on ';', trims whitespace around each tag, and drops empty
// tags (so a stray trailing ';' or surrounding spaces no longer match every
// comment or fail to match a valid tag).
func ParseSearchTextKeywords(searchText string) []string {
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

// isPlayer1TakePassCubeAction reports whether player-1's recorded cube action
// MatchesMoveError checks the equity error of a played move, already
// expressed in millipoints and rounded, against an "E"-prefixed filter:
// E>x, E<x, Ex,y (both bounds inclusive, in either order). The backends
// compute the millipoint error from the recorded move and the decoded
// analysis, then delegate the comparison here.
func MatchesMoveError(moveErrorMillipoints float64, filter string) bool {
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

// MatchesDateFilter filters positions by the analysis creation date: T>d, T<d,
// Td1,d2. analysis is the position's already-decoded analysis (the backends
// list DateFilter in needAnalysis), so this predicate needs no query or
// decompression of its own; it previously ran one of each per candidate row.
func MatchesDateFilter(analysis *domain.PositionAnalysis, filter string) bool {
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
