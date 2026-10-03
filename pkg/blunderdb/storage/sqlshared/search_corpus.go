package sqlshared

import (
	"math"
	"strconv"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/searchfilter"
)

// likeEscape is the ESCAPE clause every name comparison carries: the name is
// literal and only `*` is a wildcard (searchfilter.NameLikePattern).
const likeEscape = ` ESCAPE '\'`

// appendCorpusClauses writes the filters that name something about the match a
// position was met in, or about the verdict stored for it: the match's length
// and date, who played it and at what seat, its tournament and round, the
// player's PR, the engine and depth of the analysis, the kind of cube decision.
// They are properties of stored rows, not of the board, so — like the identity
// clauses — they stay in SQL even in mirror search.
func (s *SearchStore) appendCorpusClauses(scope string, f domain.SearchFilters, where *strings.Builder, args *[]any) {
	s.appendMatchLevelClause(scope, f, where, args)

	if f.MatchDateFilter != "" {
		from, until, ok := searchfilter.ParseMatchDate(f.MatchDateFilter)
		if !ok {
			where.WriteString(" AND 0=1")
		}
		// Half-open on the day after the last bound, so a time of day on the
		// last day still counts and both dialects compare the same text/date.
		if from != "" {
			where.WriteString(" AND p.match_date >= " + s.DB.TimestampArg())
			*args = append(*args, from)
		}
		if until != "" {
			where.WriteString(" AND p.match_date < " + s.DB.TimestampArg())
			*args = append(*args, until)
		}
	}

	s.appendProvenanceClause(scope, f, where, args)

	// A cube response filter names a kind of cube decision by itself. When the
	// board's own cube decision type is on, buildWhere has already written it.
	if f.CubeResponseFilter != "" && !(f.DecisionTypeFilter && f.Filter.DecisionType == domain.CubeAction && !f.MirrorFilter) {
		where.WriteString(" AND p.decision_type = ? AND " + s.DB.Bool("p.is_cube_response", f.CubeResponseFilter == "takepass"))
		*args = append(*args, domain.CubeAction)
	}
}

// appendMatchLevelClause writes ONE subquery over the moves that reach a
// position, so every condition about the match — the two players and the seat
// of the one who decided, the tournament, the round, the PR — holds of the same
// match and the same move. Separate subqueries would let `pl"A" op"B"` be
// satisfied by A's match against C and a match of B against D.
func (s *SearchStore) appendMatchLevelClause(scope string, f domain.SearchFilters, where *strings.Builder, args *[]any) {
	player, seatOnly := searchfilter.PlayerSpec(f.PlayerFilter)
	opponent := ""
	if f.OpponentFilter != "" {
		opponent = searchfilter.QuotedName(f.OpponentFilter, "op")
	}
	tournament := ""
	if f.TournamentNameFilter != "" {
		tournament = searchfilter.QuotedName(f.TournamentNameFilter, "tn")
	}
	rounds := domain.SplitFilterList(f.RoundFilter)
	if player == "" && opponent == "" && tournament == "" && len(rounds) == 0 && f.PlayerPRFilter == "" && f.MatchLengthFilter == "" {
		return
	}

	like := s.DB.ILike()
	var cond strings.Builder
	var condArgs []any
	name := func(col string) string { return col + " " + like + " ?" + likeEscape }
	pat := searchfilter.NameLikePattern

	// Who sat where. a is the named player, b the opponent; the seat of the
	// player who took the decision is move.player (1 = player1, -1 = player2).
	switch {
	case player != "" && opponent != "":
		seat1 := "(" + name("mt.player1_name") + " AND " + name("mt.player2_name") + ")"
		seat2 := "(" + name("mt.player2_name") + " AND " + name("mt.player1_name") + ")"
		if seatOnly {
			seat1 = "(mv.player = 1 AND " + seat1 + ")"
			seat2 = "(mv.player = -1 AND " + seat2 + ")"
		}
		cond.WriteString(" AND (" + seat1 + " OR " + seat2 + ")")
		condArgs = append(condArgs, pat(player), pat(opponent), pat(player), pat(opponent))
	case player != "" && seatOnly:
		cond.WriteString(" AND ((mv.player = 1 AND " + name("mt.player1_name") + ") OR (mv.player = -1 AND " + name("mt.player2_name") + "))")
		condArgs = append(condArgs, pat(player), pat(player))
	case player != "" || opponent != "":
		who := player
		if who == "" {
			who = opponent
		}
		cond.WriteString(" AND (" + name("mt.player1_name") + " OR " + name("mt.player2_name") + ")")
		condArgs = append(condArgs, pat(who), pat(who))
	}

	if tournament != "" {
		tTenant, tArgs := s.DB.TenantFilter("t", scope)
		cond.WriteString(" AND mt.tournament_id IN (SELECT t.id FROM tournament t WHERE " + tTenant + " AND " + name("t.name") + ")")
		condArgs = append(condArgs, tArgs...)
		condArgs = append(condArgs, pat(tournament))
	}

	if len(rounds) > 0 {
		parts := make([]string, len(rounds))
		for i, r := range rounds {
			parts[i] = name("mt.round")
			condArgs = append(condArgs, pat(r))
		}
		cond.WriteString(" AND (" + strings.Join(parts, " OR ") + ")")
	}

	if f.MatchLengthFilter != "" {
		// `ml:7` reads like the other counts once its colon is dropped. The
		// length is the match's: position.match_length is never written.
		expr := strings.Replace(f.MatchLengthFilter, "ml:", "ml", 1)
		lo, hi, hasLo, hasHi := searchfilter.ParseIntFilterExpr(expr, "ml")
		if !hasLo && !hasHi {
			cond.WriteString(" AND 0=1")
		}
		searchfilter.AppendIntRangeSQL("mt.match_length", lo, hi, hasLo, hasHi, &cond, &condArgs)
	}

	if f.PlayerPRFilter != "" {
		lo, hi, hasLo, hasHi := searchfilter.ParseFloatFilterExpr(f.PlayerPRFilter, "pr")
		if !hasLo && !hasHi {
			cond.WriteString(" AND 0=1")
		} else {
			// PR is stored per match and per seat; the seat is the one of the
			// move's player. A NULL PR fails every comparison, so it drops out.
			msTenant, msArgs := s.DB.TenantFilter("ms", scope)
			cond.WriteString(" AND EXISTS (SELECT 1 FROM match_stats ms WHERE " + msTenant +
				" AND ms.match_id = mt.id AND ms.seat = (CASE WHEN mv.player = 1 THEN 1 ELSE 2 END)")
			condArgs = append(condArgs, msArgs...)
			switch {
			case hasLo && hasHi && lo == hi:
				cond.WriteString(" AND ms.pr = ?")
				condArgs = append(condArgs, lo)
			case hasLo && hasHi:
				cond.WriteString(" AND ms.pr BETWEEN ? AND ?")
				condArgs = append(condArgs, math.Min(lo, hi), math.Max(lo, hi))
			case hasLo:
				cond.WriteString(" AND ms.pr >= ?")
				condArgs = append(condArgs, lo)
			default:
				cond.WriteString(" AND ms.pr <= ?")
				condArgs = append(condArgs, hi)
			}
			cond.WriteString(")")
		}
	}

	mtTenant, mtArgs := s.DB.TenantFilter("mt", scope)
	where.WriteString(" AND p.id IN (SELECT mv.position_id FROM move mv" +
		" JOIN game g ON mv.game_id = g.id" +
		" JOIN match mt ON g.match_id = mt.id" +
		" WHERE " + mtTenant + cond.String() + ")")
	*args = append(*args, mtArgs...)
	*args = append(*args, condArgs...)
}

// appendProvenanceClause writes the `ad:` filter on the analysis row's
// engine and depth columns. The values are engines (a prefix of the engine
// label, so `gammonnet` finds "gammonNet v1.2.1") and depths (the rank of
// domain.AnalysisDepthRank: `3ply` is 3, `3ply+` at least 3, `book` 100,
// `rollout` at least 101). One engine and one depth must hold, each among its
// own alternatives.
func (s *SearchStore) appendProvenanceClause(scope string, f domain.SearchFilters, where *strings.Builder, args *[]any) {
	values := domain.SplitFilterList(f.AnalysisProvenanceFilter)
	if len(values) == 0 {
		return
	}
	var engines, depths []string
	var engineArgs, depthArgs []any
	for _, v := range values {
		v = strings.ToLower(v)
		switch {
		case v == "book":
			depths = append(depths, "a.analysis_depth = ?")
			depthArgs = append(depthArgs, 100)
		case v == "rollout":
			depths = append(depths, "a.analysis_depth >= ?")
			depthArgs = append(depthArgs, 101)
		case strings.HasSuffix(v, "ply") || strings.HasSuffix(v, "ply+"):
			n, err := strconv.Atoi(strings.TrimSuffix(strings.TrimSuffix(v, "+"), "ply"))
			if err != nil {
				depths = append(depths, "0=1")
				continue
			}
			op := "="
			if strings.HasSuffix(v, "+") {
				op = ">="
			}
			depths = append(depths, "a.analysis_depth "+op+" ?")
			depthArgs = append(depthArgs, n)
		default:
			engines = append(engines, "a.analysis_engine "+s.DB.ILike()+" ?"+likeEscape)
			engineArgs = append(engineArgs, searchfilter.NameLikePattern(v)+"%")
		}
	}
	aTenant, aArgs := s.DB.TenantFilter("a", scope)
	where.WriteString(" AND EXISTS (SELECT 1 FROM analysis a WHERE " + aTenant + " AND a.position_id = p.id")
	*args = append(*args, aArgs...)
	if len(engines) > 0 {
		where.WriteString(" AND (" + strings.Join(engines, " OR ") + ")")
		*args = append(*args, engineArgs...)
	}
	if len(depths) > 0 {
		where.WriteString(" AND (" + strings.Join(depths, " OR ") + ")")
		*args = append(*args, depthArgs...)
	}
	where.WriteString(")")
}
