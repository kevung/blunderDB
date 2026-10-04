package sqlshared

import (
	"context"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
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
func (s *SearchStore) appendCorpusClauses(scope string, f domain.SearchFilters, prMissing *prOfMissing, aliases storage.AliasMap, where *strings.Builder, args *[]any) {
	s.appendMatchLevelClause(scope, f, prMissing, aliases, where, args)

	if f.MatchDateFilter != "" {
		from, until, ok := searchfilter.ParseMatchDate(f.MatchDateFilter)
		if !ok {
			where.WriteString(" AND 0=1")
		}
		// Half-open on the day after the last bound, so a time of day on the
		// last day still counts. A day bound is midnight UTC (ADR-0070).
		for _, b := range []struct{ day, op string }{{from, " >= "}, {until, " < "}} {
			if b.day == "" {
				continue
			}
			t, err := time.Parse(time.DateOnly, b.day)
			if err != nil {
				where.WriteString(" AND 0=1")
				continue
			}
			ph, arg := s.DB.InstantArg(t)
			where.WriteString(" AND p.match_date" + b.op + ph)
			*args = append(*args, arg)
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
func (s *SearchStore) appendMatchLevelClause(scope string, f domain.SearchFilters, prMissing *prOfMissing, aliases storage.AliasMap, where *strings.Builder, args *[]any) {
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

	// person writes "col matches one of who's spellings": an aliased player is
	// every name the alias table gives them.
	person := func(col, who string) (string, []any) {
		group := aliases.Group(who)
		if len(group) == 0 {
			group = []string{who}
		}
		parts := make([]string, len(group))
		vals := make([]any, len(group))
		for i, n := range group {
			parts[i] = name(col)
			vals[i] = pat(n)
		}
		return "(" + strings.Join(parts, " OR ") + ")", vals
	}

	// Who sat where. a is the named player, b the opponent; the seat of the
	// player who took the decision is move.player (1 = player1, -1 = player2).
	switch {
	case player != "" && opponent != "":
		p1, a1 := person("mt.player1_name", player)
		o2, b2 := person("mt.player2_name", opponent)
		p2, a2 := person("mt.player2_name", player)
		o1, b1 := person("mt.player1_name", opponent)
		seat1 := "(" + p1 + " AND " + o2 + ")"
		seat2 := "(" + p2 + " AND " + o1 + ")"
		if seatOnly {
			seat1 = "(mv.player = 1 AND " + seat1 + ")"
			seat2 = "(mv.player = -1 AND " + seat2 + ")"
		}
		cond.WriteString(" AND (" + seat1 + " OR " + seat2 + ")")
		condArgs = append(condArgs, a1...)
		condArgs = append(condArgs, b2...)
		condArgs = append(condArgs, a2...)
		condArgs = append(condArgs, b1...)
	case player != "" && seatOnly:
		p1, a1 := person("mt.player1_name", player)
		p2, a2 := person("mt.player2_name", player)
		cond.WriteString(" AND ((mv.player = 1 AND " + p1 + ") OR (mv.player = -1 AND " + p2 + "))")
		condArgs = append(condArgs, a1...)
		condArgs = append(condArgs, a2...)
	case player != "" || opponent != "":
		who := player
		if who == "" {
			who = opponent
		}
		p1, a1 := person("mt.player1_name", who)
		p2, a2 := person("mt.player2_name", who)
		cond.WriteString(" AND (" + p1 + " OR " + p2 + ")")
		condArgs = append(condArgs, a1...)
		condArgs = append(condArgs, a2...)
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
			cond.WriteString(" AND (EXISTS (SELECT 1 FROM match_stats ms WHERE " + msTenant +
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
			// Matches the table lacks, measured on the fly by a reader that
			// could not fill it (prOfMissingMatches).
			if prMissing != nil {
				for _, side := range []struct {
					seat string
					ids  []int64
				}{{"mv.player = 1", prMissing.seat1}, {"mv.player <> 1", prMissing.seat2}} {
					if len(side.ids) > 0 {
						cond.WriteString(" OR (" + side.seat + " AND mt.id IN (" + Placeholders(len(side.ids)) + "))")
						condArgs = append(condArgs, int64Args(side.ids)...)
					}
				}
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

// prOfMissing lists, per seat, the matches absent from match_stats whose PR
// satisfies the `pr` filter.
type prOfMissing struct{ seat1, seat2 []int64 }

// prOfMissingMatches makes match_stats complete before the `pr` filter reads
// it. A connection that cannot write measures the missing matches instead,
// from their decisions, and returns those that pass; nil means the table
// answers alone.
func (s *SearchStore) prOfMissingMatches(ctx context.Context, scope, filter string) (*prOfMissing, error) {
	st := &StatsStore{DB: s.DB}
	if !RefusesWrites(ctx, s.DB) {
		_, err := st.FillMatchStats(ctx, scope, nil)
		return nil, err
	}
	ids, err := st.missingMatchStats(ctx, scope)
	if err != nil || len(ids) == 0 {
		return nil, err
	}
	lo, hi, hasLo, hasHi := searchfilter.ParseFloatFilterExpr(filter, "pr")
	if !hasLo && !hasHi {
		return nil, nil
	}
	settings, err := librarySettings(ctx, s.DB, scope)
	if err != nil {
		return nil, err
	}
	rows, err := st.computeMatchStatsRows(ctx, s.DB, scope, ids, settings)
	if err != nil {
		return nil, err
	}
	out := &prOfMissing{}
	for _, r := range rows {
		// The table stores NULL for a seat without decisions, which no
		// comparison accepts.
		if r.Decisions == 0 {
			continue
		}
		var ok bool
		switch {
		case hasLo && hasHi && lo == hi:
			ok = r.PR == lo
		case hasLo && hasHi:
			ok = r.PR >= math.Min(lo, hi) && r.PR <= math.Max(lo, hi)
		case hasLo:
			ok = r.PR >= lo
		default:
			ok = r.PR <= hi
		}
		if !ok {
			continue
		}
		if r.Seat == 1 {
			out.seat1 = append(out.seat1, r.MatchID)
		} else {
			out.seat2 = append(out.seat2, r.MatchID)
		}
	}
	return out, nil
}
