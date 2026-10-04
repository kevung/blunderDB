package sqlshared

// stats_from_table.go — the passes of Compute and PlayerTable that read
// match_stats instead of re-aggregating move × analysis. Every filter they
// honour is a match-level predicate or a seat, which the table's rows carry;
// a filter on the decisions themselves (provenance) sends them back to the
// direct passes, so both paths always agree on what they return.

import (
	"context"
	"fmt"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// fromMatchStats reports whether every predicate of filter can be applied
// to match_stats rows.
func fromMatchStats(filter storage.StatsFilter) bool {
	return filter.AnalysisEngine == "" && filter.MinAnalysisDepth <= 0
}

// matchStatsWhere renders filter against `match_stats ms JOIN match m`. With
// seatAware the player filter keeps the rows of the named players' seats;
// without it, both seats of every match one of them played (the Snowie
// denominator).
func (s *StatsStore) matchStatsWhere(scope string, filter storage.StatsFilter, seatAware bool) (string, []any) {
	where, args := s.buildMatchWhereClause(scope, filter)
	names := storage.PlayerNameSet(filter)
	if len(names) == 0 {
		return where, args
	}
	ph := Placeholders(len(names))
	if seatAware {
		where += " AND ((ms.seat = 1 AND m.player1_name IN (" + ph + ")) OR (ms.seat = 2 AND m.player2_name IN (" + ph + ")))"
	} else {
		where += " AND (m.player1_name IN (" + ph + ") OR m.player2_name IN (" + ph + "))"
	}
	for range 2 {
		for _, n := range names {
			args = append(args, n)
		}
	}
	return where, args
}

const matchStatsJoin = ` FROM match_stats ms JOIN match m ON m.id = ms.match_id`

// matchStatsSource is the FROM clause of a read of match_stats: the table,
// filled first, or — on a connection that refuses writes and so cannot fill
// it — the same seat rows (both seats of every match, zero when nothing is
// counted) computed from the decisions within the query, so a read-only
// reader gets the figures a writer would and the table is never trusted
// while incomplete. Only the columns the readers here use are derived.
func (s *StatsStore) matchStatsSource(ctx context.Context, scope string) (string, []any, error) {
	if !RefusesWrites(ctx, s.DB) {
		if _, err := s.FillMatchStats(ctx, scope, nil); err != nil {
			return "", nil, err
		}
		return matchStatsJoin, nil, nil
	}
	d := s.DB
	pTenant, pArgs := d.TenantFilter("p", scope)
	mTenant, mArgs := d.TenantFilter("m0", scope)
	errE := `(` + statsErrExpr + `)`
	isCube := `p.decision_type = 1`
	from := ` FROM (SELECT m0.id AS match_id, s.seat AS seat,
		COALESCE(x.decisions, 0) AS decisions, COALESCE(x.error_mp, 0) AS error_mp,
		COALESCE(x.checker_decisions, 0) AS checker_decisions, COALESCE(x.checker_error_mp, 0) AS checker_error_mp,
		COALESCE(x.cube_decisions, 0) AS cube_decisions, COALESCE(x.cube_error_mp, 0) AS cube_error_mp
		FROM match m0 CROSS JOIN (SELECT 1 AS seat UNION ALL SELECT 2 AS seat) s
		LEFT JOIN (SELECT g.match_id AS match_id, ` + seatExpr + ` AS seat,
			COUNT(*) AS decisions, SUM(` + errE + `) AS error_mp,
			SUM(CASE WHEN ` + isCube + ` THEN 0 ELSE 1 END) AS checker_decisions,
			SUM(CASE WHEN ` + isCube + ` THEN 0 ELSE ` + errE + ` END) AS checker_error_mp,
			SUM(CASE WHEN ` + isCube + ` THEN 1 ELSE 0 END) AS cube_decisions,
			SUM(CASE WHEN ` + isCube + ` THEN ` + errE + ` ELSE 0 END) AS cube_error_mp
			` + statsBaseJoin + ` WHERE ` + pTenant + ` AND a.position_id IS NOT NULL AND ` + errE + ` IS NOT NULL AND ` + countedExpr(d) + `
			GROUP BY g.match_id, ` + seatExpr + `) x ON x.match_id = m0.id AND x.seat = s.seat
		WHERE ` + mTenant + `) ms JOIN match m ON m.id = ms.match_id`
	return from, append(append([]any{}, pArgs...), mArgs...), nil
}

// tableErrDecisions renders the error sum and decision count of the rows for
// the filter's decision type.
func tableErrDecisions(d Dialect, decisionType int) (sumErr, count string) {
	switch decisionType {
	case 0:
		return d.Bigint(`COALESCE(SUM(ms.checker_error_mp),0)`), d.Bigint(`COALESCE(SUM(ms.checker_decisions),0)`)
	case 1:
		return d.Bigint(`COALESCE(SUM(ms.cube_error_mp),0)`), d.Bigint(`COALESCE(SUM(ms.cube_decisions),0)`)
	}
	return d.Bigint(`COALESCE(SUM(ms.error_mp),0)`), d.Bigint(`COALESCE(SUM(ms.decisions),0)`)
}

// prByDecisionTypeFromTable is computePRByDecisionType over match_stats.
func (s *StatsStore) prByDecisionTypeFromTable(ctx context.Context, q statsQuery, result *storage.StatsResult) error {
	d := s.DB
	where, args := s.matchStatsWhere(q.scope, q.filter, true)
	var chkErr, chkN, cubeErr, cubeN int64
	if err := d.QueryRow(ctx,
		`SELECT `+d.Bigint(`COALESCE(SUM(ms.checker_error_mp),0)`)+`, `+d.Bigint(`COALESCE(SUM(ms.checker_decisions),0)`)+`, `+
			d.Bigint(`COALESCE(SUM(ms.cube_error_mp),0)`)+`, `+d.Bigint(`COALESCE(SUM(ms.cube_decisions),0)`)+
			matchStatsJoin+where, args...).Scan(&chkErr, &chkN, &cubeErr, &cubeN); err != nil {
		return fmt.Errorf("PR by decision_type (table): %w", err)
	}
	// A decision-type filter leaves the other type out entirely, as the
	// direct pass's WHERE does.
	switch q.filter.DecisionType {
	case 0:
		cubeErr, cubeN = 0, 0
	case 1:
		chkErr, chkN = 0, 0
	}
	result.PRChecker = pr(chkErr, int(chkN))
	result.PRCube = pr(cubeErr, int(cubeN))
	result.PRGlobal = pr(chkErr+cubeErr, int(chkN+cubeN))
	return nil
}

// snowieGlobalFromTable is computeSnowieGlobal over match_stats: the player's
// seats for the errors, both seats of his matches for the moves.
func (s *StatsStore) snowieGlobalFromTable(ctx context.Context, q statsQuery, result *storage.StatsResult) error {
	d := s.DB
	f := q.filter
	f.DecisionType = -1
	numWhere, numArgs := s.matchStatsWhere(q.scope, f, true)
	var num int64
	if err := d.QueryRow(ctx, `SELECT `+d.Bigint(`COALESCE(SUM(ms.snowie_error_mp),0)`)+matchStatsJoin+numWhere, numArgs...).Scan(&num); err != nil {
		return fmt.Errorf("snowie ER (table) numerator: %w", err)
	}
	denWhere, denArgs := s.matchStatsWhere(q.scope, f, false)
	var den int64
	if err := d.QueryRow(ctx, `SELECT `+d.Bigint(`COALESCE(SUM(ms.snowie_moves),0)`)+matchStatsJoin+denWhere, denArgs...).Scan(&den); err != nil {
		return fmt.Errorf("snowie ER (table) denominator: %w", err)
	}
	result.SnowieGlobal = snowieER(num, int(den))
	return nil
}

// perTournamentFromTable is computePerTournament over match_stats.
func (s *StatsStore) perTournamentFromTable(ctx context.Context, q statsQuery, result *storage.StatsResult) error {
	d := s.DB
	where, args := s.matchStatsWhere(q.scope, q.filter, true)
	sumErr, count := tableErrDecisions(d, q.filter.DecisionType)
	return scanEach(ctx, d,
		`SELECT m.tournament_id, COALESCE(t.name,''), COALESCE(t.date,''), `+sumErr+`, `+count+
			matchStatsJoin+` LEFT JOIN tournament t ON t.id = m.tournament_id`+where+
			` AND m.tournament_id IS NOT NULL`+
			` GROUP BY m.tournament_id, t.name, t.date, t.created_at HAVING `+count+` > 0 ORDER BY t.date, t.created_at`,
		args, func(r Rows) error {
			var ts storage.TournamentStats
			var sum, n int64
			if err := r.Scan(&ts.ID, &ts.Name, &ts.Date, &sum, &n); err != nil {
				return fmt.Errorf("PR per tournament (table): %w", err)
			}
			ts.NumDecisions = int(n)
			ts.PR = pr(sum, int(n))
			result.PerTournament = append(result.PerTournament, ts)
			return nil
		})
}

// perMatchFromTable is computePerMatch over match_stats.
func (s *StatsStore) perMatchFromTable(ctx context.Context, q statsQuery, result *storage.StatsResult) error {
	d := s.DB
	where, args := s.matchStatsWhere(q.scope, q.filter, true)
	sumErr, count := tableErrDecisions(d, q.filter.DecisionType)
	return scanEach(ctx, d,
		`SELECT m.id, `+d.DateText("m.match_date")+`, `+sumErr+`, `+count+matchStatsJoin+where+
			` GROUP BY m.id, m.match_date HAVING `+count+` > 0 ORDER BY m.match_date, m.id`,
		args, func(r Rows) error {
			var ms storage.MatchStats
			var sum, n int64
			if err := r.Scan(&ms.ID, &ms.Date, &sum, &n); err != nil {
				return fmt.Errorf("PR per match (table): %w", err)
			}
			ms.NumDecisions = int(n)
			ms.PR = pr(sum, int(n))
			result.PerMatch = append(result.PerMatch, ms)
			return nil
		})
}

// playerSumsFromTable is playerSumsDirect over match_stats. The errors and
// blunders of a player ride on his checker entry: BuildPlayerRows only
// totals them across decision types.
func (s *StatsStore) playerSumsFromTable(ctx context.Context, scope string, f storage.StatsFilter) (decisions []storage.PlayerDecisionStat, snowieErr map[string]int64, luck map[string]storage.PlayerLuckAcc, err error) {
	d := s.DB
	where, args := s.matchStatsWhere(scope, f, true)
	snowieErr = map[string]int64{}
	luck = map[string]storage.PlayerLuckAcc{}
	err = scanEach(ctx, d,
		`SELECT CASE WHEN ms.seat = 1 THEN m.player1_name ELSE m.player2_name END AS pname, `+
			d.Bigint(`SUM(ms.checker_error_mp)`)+`, `+d.Bigint(`SUM(ms.checker_decisions)`)+`, `+
			d.Bigint(`SUM(ms.cube_error_mp)`)+`, `+d.Bigint(`SUM(ms.cube_decisions)`)+`, `+
			d.Bigint(`SUM(ms.errors)`)+`, `+d.Bigint(`SUM(ms.blunders)`)+`, `+
			d.Bigint(`SUM(ms.snowie_error_mp)`)+`, `+d.Bigint(`SUM(ms.snowie_moves)`)+`, `+
			d.Bigint(`SUM(ms.luck_mp)`)+`, `+d.Bigint(`SUM(ms.luck_rolls)`)+
			matchStatsJoin+where+` GROUP BY pname`,
		args, func(r Rows) error {
			var name *string
			var chkErr, chkN, cubeErr, cubeN, errs, blunders, snowie, snowieMoves, luckMP, luckRolls int64
			if err := r.Scan(&name, &chkErr, &chkN, &cubeErr, &cubeN, &errs, &blunders, &snowie, &snowieMoves, &luckMP, &luckRolls); err != nil {
				return err
			}
			if name == nil {
				return nil
			}
			if chkN > 0 {
				decisions = append(decisions, storage.PlayerDecisionStat{Name: *name, DecisionType: 0, SumErrMP: chkErr, Count: int(chkN), Errors: int(errs), Blunders: int(blunders)})
			}
			if cubeN > 0 {
				dec := storage.PlayerDecisionStat{Name: *name, DecisionType: 1, SumErrMP: cubeErr, Count: int(cubeN)}
				if chkN == 0 {
					dec.Errors, dec.Blunders = int(errs), int(blunders)
				}
				decisions = append(decisions, dec)
			}
			// The direct path groups only the rows that exist: a player
			// without an analysed decision has no Snowie entry.
			if snowieMoves > 0 || snowie != 0 {
				snowieErr[*name] = snowie
			}
			if luckRolls > 0 {
				luck[*name] = storage.PlayerLuckAcc{SumMP: luckMP, Rolls: int(luckRolls)}
			}
			return nil
		})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("PlayerTable (table): %w", err)
	}
	return decisions, snowieErr, luck, nil
}

// HeadToHead — see storage.StatsStore.
func (s *StatsStore) HeadToHead(ctx context.Context, scope, playerA, playerB string, filter storage.StatsFilter) (*storage.HeadToHead, error) {
	if !fromMatchStats(filter) {
		return nil, fmt.Errorf("head to head with a provenance filter: %w", storage.ErrInvalid)
	}
	if playerA == "" || playerB == "" || playerA == playerB {
		return nil, fmt.Errorf("head to head needs two distinct players: %w", storage.ErrInvalid)
	}
	// Each side is a person: every spelling the alias table gives them.
	aliases, err := aliasMap(ctx, s.DB, scope, storage.AliasPlayer)
	if err != nil {
		return nil, fmt.Errorf("head to head aliases: %w", err)
	}
	if aliases.Canonical(playerA) == aliases.Canonical(playerB) {
		return nil, fmt.Errorf("head to head needs two distinct players: %w", storage.ErrInvalid)
	}
	groupA, groupB := aliases.Group(playerA), aliases.Group(playerB)
	isA := make(map[string]bool, len(groupA))
	for _, n := range groupA {
		isA[n] = true
	}
	from, fromArgs, err := s.matchStatsSource(ctx, scope)
	if err != nil {
		return nil, err
	}
	d := s.DB
	f := filter
	f.PlayerName, f.PlayerAliases = "", nil
	where, args := s.buildMatchWhereClause(scope, f)
	phA, phB := Placeholders(len(groupA)), Placeholders(len(groupB))
	where += " AND ((m.player1_name IN (" + phA + ") AND m.player2_name IN (" + phB + "))" +
		" OR (m.player1_name IN (" + phB + ") AND m.player2_name IN (" + phA + ")))"
	for _, g := range [][]string{groupA, groupB, groupB, groupA} {
		for _, n := range g {
			args = append(args, n)
		}
	}
	sumErr, count := tableErrDecisions(d, filter.DecisionType)
	out := &storage.HeadToHead{PlayerA: playerA, PlayerB: playerB, Matches: []storage.HeadToHeadMatch{}}
	byID := map[int64]*storage.HeadToHeadMatch{}
	var order []int64
	var sumA, sumB int64
	err = scanEach(ctx, d,
		`SELECT m.id, `+d.DateText("m.match_date")+`, COALESCE(m.match_length, 0), m.player1_name, ms.seat, `+sumErr+`, `+count+`,
		        `+d.Bigint(`COALESCE((SELECT SUM(g.points_won) FROM game g WHERE g.match_id = m.id AND g.winner = 1), 0)`)+`,
		        `+d.Bigint(`COALESCE((SELECT SUM(g.points_won) FROM game g WHERE g.match_id = m.id AND g.winner = -1), 0)`)+
			from+where+` GROUP BY m.id, m.match_date, m.match_length, m.player1_name, ms.seat ORDER BY m.match_date, m.id, ms.seat`,
		append(fromArgs, args...), func(r Rows) error {
			var id, sum, n, pts1, pts2 int64
			var date *string
			var length, seat int
			var p1 string
			if err := r.Scan(&id, &date, &length, &p1, &seat, &sum, &n, &pts1, &pts2); err != nil {
				return err
			}
			m, ok := byID[id]
			if !ok {
				m = &storage.HeadToHeadMatch{ID: id, MatchLength: length}
				if date != nil {
					m.Date = *date
				}
				outcome := storage.MatchOutcome(int32(length), int32(pts1), int32(pts2))
				if !isA[p1] {
					outcome = -outcome
				}
				m.Outcome = outcome
				byID[id] = m
				order = append(order, id)
			}
			// Seat 1 is A's when A sat as player 1.
			if (seat == 1) == isA[p1] {
				m.DecisionsA, m.PRA = int(n), pr(sum, int(n))
				sumA += sum
				out.DecisionsA += int(n)
			} else {
				m.DecisionsB, m.PRB = int(n), pr(sum, int(n))
				sumB += sum
				out.DecisionsB += int(n)
			}
			return nil
		})
	if err != nil {
		return nil, errf(d, "head to head", err)
	}
	for _, id := range order {
		m := byID[id]
		switch m.Outcome {
		case 1:
			out.WinsA++
		case -1:
			out.WinsB++
		}
		out.Matches = append(out.Matches, *m)
	}
	out.PRA, out.PRB = pr(sumA, out.DecisionsA), pr(sumB, out.DecisionsB)
	return out, nil
}

// PRByWindow — see storage.StatsStore.
func (s *StatsStore) PRByWindow(ctx context.Context, scope string, filter storage.StatsFilter, months int) ([]storage.WindowStats, error) {
	if !fromMatchStats(filter) {
		return nil, fmt.Errorf("PR by window with a provenance filter: %w", storage.ErrInvalid)
	}
	if months < 1 || months > 120 {
		return nil, fmt.Errorf("window of %d months: %w", months, storage.ErrInvalid)
	}
	filter, err := s.withPlayerAliases(ctx, scope, filter)
	if err != nil {
		return nil, err
	}
	from, fromArgs, err := s.matchStatsSource(ctx, scope)
	if err != nil {
		return nil, err
	}
	d := s.DB
	where, args := s.matchStatsWhere(scope, filter, true)
	sumErr, count := tableErrDecisions(d, filter.DecisionType)
	month := `SUBSTR(` + d.DateText("m.match_date") + `, 1, 7)`
	type bucket struct{ sum, n, matches int64 }
	buckets := map[string]bucket{}
	var first, last string
	err = scanEach(ctx, d,
		`SELECT `+month+` AS mon, `+sumErr+`, `+count+`, COUNT(DISTINCT CASE WHEN ms.decisions > 0 THEN m.id END)`+
			from+where+` AND m.match_date IS NOT NULL GROUP BY mon HAVING `+count+` > 0 ORDER BY mon`,
		append(fromArgs, args...), func(r Rows) error {
			var mon string
			var b bucket
			if err := r.Scan(&mon, &b.sum, &b.n, &b.matches); err != nil {
				return err
			}
			buckets[mon] = b
			if first == "" {
				first = mon
			}
			last = mon
			return nil
		})
	if err != nil {
		return nil, errf(d, "PR by window", err)
	}
	out := []storage.WindowStats{}
	if first == "" {
		return out, nil
	}
	start, err1 := time.Parse("2006-01", first)
	end, err2 := time.Parse("2006-01", last)
	if err1 != nil || err2 != nil {
		return nil, fmt.Errorf("PR by window: month %q..%q", first, last)
	}
	for cur := start; !cur.After(end); cur = cur.AddDate(0, 1, 0) {
		from := cur.AddDate(0, -(months - 1), 0)
		var w bucket
		for m := from; !m.After(cur); m = m.AddDate(0, 1, 0) {
			b := buckets[m.Format("2006-01")]
			w.sum += b.sum
			w.n += b.n
			w.matches += b.matches
		}
		out = append(out, storage.WindowStats{
			From: from.Format("2006-01"), To: cur.Format("2006-01"),
			NumMatches: int(w.matches), NumDecisions: int(w.n), PR: pr(w.sum, int(w.n)),
		})
	}
	return out, nil
}
