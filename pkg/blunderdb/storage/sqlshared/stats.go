package sqlshared

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// StatsStore implements storage.StatsStore, written once for both backends
// with the dialect differences woven in:
//   - tenant scoping: every query is confined to the scope's tenant through
//     Dialect.TenantFilter on the position root (joins are by global primary
//     key, so scoping the root confines the whole join graph) — a no-op
//     predicate on SQLite, whose schema has no tenant column;
//   - types: is_forced / is_close_cube are INTEGER 0/1 on SQLite and BOOLEAN
//     on PostgreSQL (Dialect.Bool), match_date is TEXT vs TIMESTAMPTZ
//     (Dialect.TimestampArg / DateText);
//   - SUM() over BIGINT yields NUMERIC in PostgreSQL, so the running totals
//     go through Dialect.Bigint before scanning into int64.
//
// DateRange is not implemented here: its predicate on match_date (a text
// sentinel on SQLite, a plain IS NOT NULL on PostgreSQL) has no shared form,
// so each backend keeps its own on top of this store.
type StatsStore struct{ DB Execer }

// statsErrExpr selects the error column that applies to a decision; it is
// shared with the search store's move-error filter.
const statsErrExpr = "CASE WHEN p.decision_type = 1 THEN a.cube_error ELSE a.best_move_equity_error END"

// The Error and Blunder thresholds are the library's settings
// (storage.LibrarySettings, ADR-0046, defaults 50 and 100), read once per
// entry point and carried down. Comparisons are inclusive: a cost of exactly
// the threshold is on the wrong side of the line.

// countedExpr renders the SQL predicate selecting the decisions that count
// toward PR and decision tallies, as XG counts them:
//   - Checker: unforced plays (a.is_forced is false, engine.IsForcedChecker)
//     whose cost is known — a played move none of the candidates names has a
//     NULL error and is left out, as XG leaves out a play it never analysed.
//   - Cube: every double offered, take and pass (the move's own cube action,
//     not the analysis's: a deduplicated position may have been doubled in
//     one match and not in another), and a no-double flagged close
//     (a.is_close_cube, engine.ComputeIsCloseCube).
//
// A function of the dialect because the two flags are INTEGER 0/1 on SQLite,
// BOOLEAN on PostgreSQL.
func countedExpr(d Dialect) string {
	return "((p.decision_type = 0 AND " + d.Bool("a.is_forced", false) + " AND a.best_move_equity_error IS NOT NULL) OR (p.decision_type = 1 AND (" + ActionNotInSQL("mv.cube_action", "", "No Double", "NoDouble") + " OR " + d.Bool("a.is_close_cube", true) + ")))"
}

// UnscoredPlaySQL is true for a checker position whose analysis cannot score
// the play made there: a NULL error, the played move naming no candidate
// (engine.AnalysisColumns.BestMoveUnscored). Such a play is left out of every
// count, as an unanalysed one is left out of the analysed counts; pos is the
// position's alias.
func UnscoredPlaySQL(pos string) string {
	return "EXISTS (SELECT 1 FROM analysis ua WHERE ua.position_id = " + pos + ".id AND ua.best_move_equity_error IS NULL)"
}

// cubeMultiplierExpr is the cube value (1, 2, 4, …) from its stored log2
// exponent. The CAST keeps the shift an integer operation on PostgreSQL, where
// cube_value is BIGINT and `1 << bigint` is not defined; SQLite accepts it as
// written.
const cubeMultiplierExpr = "(1 << CAST(COALESCE(p.cube_value, 0) AS INTEGER))"

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

// playerFilterClause renders the player filter two ways. With seatAware, a row
// is kept only when one of the named players IS the one who took the decision —
// what every per-player figure needs. Without it, a row is kept as soon as one
// of them played in the match, whichever side moved: the Snowie denominator
// counts both players' moves (see Compute).
//
// Both forms bind the names twice, so the arguments do not depend on the form.
func playerFilterClause(names []string, seatAware bool) (clause string, args []any) {
	ph := strings.TrimSuffix(strings.Repeat("?,", len(names)), ",")
	if seatAware {
		clause = "((m.player1_name IN (" + ph + ") AND mv.player = 1) OR (m.player2_name IN (" + ph + ") AND mv.player = -1))"
	} else {
		clause = "(m.player1_name IN (" + ph + ") OR m.player2_name IN (" + ph + "))"
	}
	for range 2 {
		for _, n := range names {
			args = append(args, n)
		}
	}
	return clause, args
}

// buildBaseWhereClause constructs the base WHERE clause for the given filter,
// scoped to the tenant, without the countedExpr predicate. The tenant
// predicate is the first clause and its arguments the first args, so the '?'
// placeholders line up once rebound.
func (s *StatsStore) buildBaseWhereClause(scope string, filter storage.StatsFilter) (whereSQL string, args []any) {
	return s.buildBaseWhereClauseSeat(scope, filter, true)
}

// buildBaseWhereClauseSeat is buildBaseWhereClause with control over how the
// player filter is rendered (see playerFilterClause).
func (s *StatsStore) buildBaseWhereClauseSeat(scope string, filter storage.StatsFilter, seatAware bool) (whereSQL string, args []any) {
	tenant, args := s.DB.TenantFilter("p", scope)
	clauses := []string{tenant}

	if names := storage.PlayerNameSet(filter); len(names) > 0 {
		clause, pArgs := playerFilterClause(names, seatAware)
		clauses = append(clauses, clause)
		args = append(args, pArgs...)
	}

	if len(filter.TournamentIDs) > 0 {
		placeholders := strings.Repeat("?,", len(filter.TournamentIDs))
		placeholders = placeholders[:len(placeholders)-1]
		clauses = append(clauses, "m.tournament_id IN ("+placeholders+")")
		for _, id := range filter.TournamentIDs {
			args = append(args, id)
		}
	}

	clauses, args = appendDateClauses(s.DB, clauses, args, filter)

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

	if filter.AnalysisEngine != "" {
		clauses = append(clauses, "a.analysis_engine = ?")
		args = append(args, filter.AnalysisEngine)
	}
	if filter.MinAnalysisDepth > 0 {
		clauses = append(clauses, "COALESCE(a.analysis_depth, 0) >= ?")
		args = append(args, filter.MinAnalysisDepth)
	}

	metClause, metArgs := metComparableDecision(s.DB, scope)
	clauses = append(clauses, metClause)
	args = append(args, metArgs...)

	clauses = append(clauses, "a.position_id IS NOT NULL")
	clauses = append(clauses, "("+statsErrExpr+") IS NOT NULL")

	whereSQL = " WHERE " + strings.Join(clauses, " AND ")
	return whereSQL, args
}

// appendDateClauses adds the filter's match_date bounds. The bound strings are
// compared as text on SQLite and cast to timestamptz on PostgreSQL
// (Dialect.TimestampArg).
func appendDateClauses(d Dialect, clauses []string, args []any, filter storage.StatsFilter) ([]string, []any) {
	ts := d.TimestampArg()
	if filter.DateFrom != "" && filter.DateTo != "" {
		clauses = append(clauses, "m.match_date BETWEEN "+ts+" AND "+ts)
		args = append(args, filter.DateFrom, filter.DateTo)
	} else if filter.DateFrom != "" {
		clauses = append(clauses, "m.match_date >= "+ts)
		args = append(args, filter.DateFrom)
	} else if filter.DateTo != "" {
		clauses = append(clauses, "m.match_date <= "+ts)
		args = append(args, filter.DateTo)
	}
	return clauses, args
}

// buildStatsWhereClause wraps buildBaseWhereClause and appends the
// countedExpr predicate (XG/gnuBG semantics).
func (s *StatsStore) buildStatsWhereClause(scope string, filter storage.StatsFilter) (whereSQL string, args []any) {
	whereSQL, args = s.buildBaseWhereClause(scope, filter)
	whereSQL += " AND " + countedExpr(s.DB)
	return whereSQL, args
}

// buildSelectionWhereClause produces the extra WHERE fragment and optional
// ORDER BY / LIMIT fragment for a given SelectionSpec.
func buildSelectionWhereClause(d Dialect, sel storage.SelectionSpec) (whereAdd string, orderLimit string, args []any) {
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
		whereAdd = " AND p.decision_type = 1 AND " + ActionLabelFor(d, "a.best_cube_action") + " = ?"
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

// scanCubeDirectionIDs keeps the ids whose (ruling, action) pair falls in cell.
// An empty cell name keeps nothing: a drill-down that names no cell is a caller
// bug, and returning "everything" would look like a working feature.
func scanCubeDirectionIDs(rows Rows, cell string) ([]int64, error) {
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		var best, played string
		if err := rows.Scan(&id, &best, &played); err != nil {
			return nil, fmt.Errorf("scan cube direction row: %w", err)
		}
		if cell != "" && storage.ClassifyCubeDirection(best, played) == cell {
			ids = append(ids, id)
		}
	}
	return ids, rows.Err()
}

func scanPositionIDs(rows Rows) ([]int64, error) {
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

// PositionIDsBySelection resolves a user selection made in the Stats panel into
// a deduplicated list of position IDs, scoped to the tenant. The StatsFilter is
// always applied so the IDs correspond exactly to what is displayed in the
// panel.
func (s *StatsStore) PositionIDsBySelection(ctx context.Context, scope string, filter storage.StatsFilter, sel storage.SelectionSpec) ([]int64, error) {
	filter, err := s.withPlayerAliases(ctx, scope, filter)
	if err != nil {
		return nil, fmt.Errorf("PositionIDsBySelection aliases: %w", err)
	}
	whereSQL, baseArgs := s.buildStatsWhereClause(scope, filter)

	// A cube-direction cell is decided by reading two free-form labels, stated
	// once in Go (storage.ClassifyCubeDirection), so it is filtered here.
	if sel.Kind == "cube_direction" {
		rows, err := s.DB.Query(ctx,
			"SELECT DISTINCT p.id, "+ActionLabelOrEmptyFor(s.DB, "a.best_cube_action")+", "+ActionLabelOrEmptyFor(s.DB, "mv.cube_action")+" "+
				statsBaseJoin+whereSQL+" AND p.decision_type = 1", baseArgs...)
		if err != nil {
			return nil, fmt.Errorf("PositionIDsBySelection (cube_direction): %w", err)
		}
		return scanCubeDirectionIDs(rows, sel.CubeCell)
	}

	whereAdd, orderLimit, selArgs := buildSelectionWhereClause(s.DB, sel)

	query := "SELECT DISTINCT p.id " + statsBaseJoin + whereSQL + whereAdd
	if orderLimit != "" {
		query += " " + orderLimit
	}

	allArgs := append(append([]any{}, baseArgs...), selArgs...)
	rows, err := s.DB.Query(ctx, query, allArgs...)
	if err != nil {
		return nil, fmt.Errorf("PositionIDsBySelection (%s): %w", sel.Kind, err)
	}
	return scanPositionIDs(rows)
}

// PositionIDsByTournament returns all position IDs belonging to the given
// tournament for the tenant, regardless of any stats filter.
func (s *StatsStore) PositionIDsByTournament(ctx context.Context, scope string, tournamentID int64) ([]int64, error) {
	tenant, args := s.DB.TenantFilter("p", scope)
	query := "SELECT DISTINCT p.id " + statsBaseJoin +
		" WHERE " + tenant + " AND m.tournament_id = ? AND a.position_id IS NOT NULL AND (" + statsErrExpr + ") IS NOT NULL"
	rows, err := s.DB.Query(ctx, query, append(args, tournamentID)...)
	if err != nil {
		return nil, fmt.Errorf("PositionIDsByTournament: %w", err)
	}
	return scanPositionIDs(rows)
}

// PositionIDsByMatch returns all position IDs belonging to the given match for
// the tenant, regardless of any stats filter.
func (s *StatsStore) PositionIDsByMatch(ctx context.Context, scope string, matchID int64) ([]int64, error) {
	tenant, args := s.DB.TenantFilter("p", scope)
	query := "SELECT DISTINCT p.id " + statsBaseJoin +
		" WHERE " + tenant + " AND m.id = ? AND a.position_id IS NOT NULL AND (" + statsErrExpr + ") IS NOT NULL"
	rows, err := s.DB.Query(ctx, query, append(args, matchID)...)
	if err != nil {
		return nil, fmt.Errorf("PositionIDsByMatch: %w", err)
	}
	return scanPositionIDs(rows)
}

// PlayerNames returns all player names found in the tenant's matches, ranked
// by the total number of matches (player1 + player2 appearances) descending;
// ties break alphabetically.
func (s *StatsStore) PlayerNames(ctx context.Context, scope string) ([]storage.PlayerFrequency, error) {
	tenant, targs := s.DB.TenantFilter("", scope)
	rows, err := s.DB.Query(ctx, `
		SELECT name, COUNT(*) AS cnt
		FROM (
			SELECT player1_name AS name FROM match WHERE `+tenant+` AND player1_name != ''
			UNION ALL
			SELECT player2_name AS name FROM match WHERE `+tenant+` AND player2_name != ''
		) AS names
		GROUP BY name
		ORDER BY cnt DESC, name ASC
	`, append(append([]any{}, targs...), targs...)...)
	if err != nil {
		return nil, fmt.Errorf("PlayerNames: %w", err)
	}
	defer rows.Close()

	var raw []storage.PlayerFrequency
	for rows.Next() {
		var pf storage.PlayerFrequency
		if err := rows.Scan(&pf.Name, &pf.Count); err != nil {
			return nil, fmt.Errorf("PlayerNames scan: %w", err)
		}
		raw = append(raw, pf)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("PlayerNames rows: %w", err)
	}
	rows.Close()
	aliases, err := aliasMap(ctx, s.DB, scope, storage.AliasPlayer)
	if err != nil {
		return nil, fmt.Errorf("PlayerNames aliases: %w", err)
	}
	return mergePlayerFrequencies(raw, aliases), nil
}

// mergePlayerFrequencies folds every alias into its canonical name and
// re-ranks: count descending, then name.
func mergePlayerFrequencies(raw []storage.PlayerFrequency, aliases storage.AliasMap) []storage.PlayerFrequency {
	if len(aliases) == 0 {
		return raw
	}
	idx := map[string]int{}
	var out []storage.PlayerFrequency
	for _, pf := range raw {
		name := aliases.Canonical(pf.Name)
		if i, ok := idx[name]; ok {
			out[i].Count += pf.Count
			continue
		}
		idx[name] = len(out)
		out = append(out, storage.PlayerFrequency{Name: name, Count: pf.Count})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// withPlayerAliases widens the filter's player to every spelling the alias
// table gives the same person, so a stats request names the person and not
// one of their spellings.
func (s *StatsStore) withPlayerAliases(ctx context.Context, scope string, filter storage.StatsFilter) (storage.StatsFilter, error) {
	names := storage.PlayerNameSet(filter)
	if len(names) == 0 {
		return filter, nil
	}
	aliases, err := aliasMap(ctx, s.DB, scope, storage.AliasPlayer)
	if err != nil || len(aliases) == 0 {
		return filter, err
	}
	group := aliases.Group(names...)
	filter.PlayerName = group[0]
	filter.PlayerAliases = group[1:]
	return filter, nil
}

// MatchDetail computes per-player statistics for the given match, scoped to
// the tenant.
func (s *StatsStore) MatchDetail(ctx context.Context, scope string, matchID int64) (*storage.MatchDetailStats, error) {
	settings, err := librarySettings(ctx, s.DB, scope)
	if err != nil {
		return nil, fmt.Errorf("MatchDetail settings: %w", err)
	}
	tenant, targs := s.DB.TenantFilter("p", scope)
	query := `SELECT mv.player, p.decision_type, ` + ActionLabelOrEmptyFor(s.DB, "mv.cube_action") + `,
		(` + statsErrExpr + `) as err_mp,
		COALESCE(p.score_1, 0), COALESCE(p.score_2, 0),
		` + cubeMultiplierExpr + `,
		COALESCE(p.match_length, m.match_length, 0) ` +
		statsBaseJoin +
		` WHERE ` + tenant + ` AND m.id = ? AND a.position_id IS NOT NULL AND (` + statsErrExpr + `) IS NOT NULL AND ` + countedExpr(s.DB)

	rows, err := s.DB.Query(ctx, query, append(append([]any{}, targs...), matchID)...)
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
			return nil, fmt.Errorf("MatchDetail scan: %w", err)
		}

		fMove := 0
		if rawPlayer == -1 {
			fMove = 1
		}
		// The Crawford sentinel is decoded first (domain.PointsAway): a stored
		// 0 is one point away, post-Crawford, and reading it as a distance
		// says "has already won".
		currentScore0 := matchLength - domain.PointsAway(awayScore0)
		currentScore1 := matchLength - domain.PointsAway(awayScore1)
		mwcLoss := engine.ConvertEMGLossToMWCLoss(int(errMP), currentScore0, currentScore1, fMove, cubeValue, matchLength)
		if math.IsNaN(mwcLoss) {
			mwcLoss = 0
		}

		isError := errMP >= int64(settings.ErrorThresholdMP)
		isBlunder := errMP >= int64(settings.BlunderThresholdMP)
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
	// Second query without countedExpr: sum all equity errors per player, count
	// checker positions (decision_type=0, forced included) for both players.
	// Denominator = anTotalMoves[P1] + anTotalMoves[P2] (gnuBG formatgs.c).
	var snowieP1SumErr, snowieP2SumErr int64
	var snowieP1Checker, snowieP2Checker int
	{
		snowieRows, snowieErr := s.DB.Query(ctx,
			`SELECT mv.player, p.decision_type, (`+statsErrExpr+`) as err_mp `+
				statsBaseJoin+
				` WHERE `+tenant+` AND m.id = ? AND a.position_id IS NOT NULL AND (`+statsErrExpr+`) IS NOT NULL`,
			append(append([]any{}, targs...), matchID)...)
		if snowieErr != nil {
			return nil, fmt.Errorf("MatchDetail snowie query: %w", snowieErr)
		}
		var scanErr error
		func() {
			defer snowieRows.Close()
			for snowieRows.Next() {
				var rawPlayer, decisionType int
				var errMP int64
				if err2 := snowieRows.Scan(&rawPlayer, &decisionType, &errMP); err2 != nil {
					scanErr = err2
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
		if scanErr != nil {
			return nil, fmt.Errorf("MatchDetail snowie scan: %w", scanErr)
		}
		if err := snowieRows.Err(); err != nil {
			return nil, fmt.Errorf("MatchDetail snowie rows: %w", err)
		}
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

// MatchBadges computes the per-player PR and total MWC loss for every match
// in the tenant, keyed by match id. It is the list-row projection of
// MatchDetail; both share statsBaseJoin + countedExpr so a match's badge PR
// equals its detail PR.
func (s *StatsStore) MatchBadges(ctx context.Context, scope string, matchIDs []int64) (map[int64]storage.MatchBadge, error) {
	tenant, args := s.DB.TenantFilter("p", scope)
	query := `SELECT g.match_id, ` + statsErrExpr + ` as err_mp,
		COALESCE(p.score_1, 0), COALESCE(p.score_2, 0), mv.player,
		` + cubeMultiplierExpr + `, COALESCE(p.match_length, m.match_length, 0) ` +
		statsBaseJoin +
		` WHERE ` + tenant + ` AND a.position_id IS NOT NULL AND (` + statsErrExpr + `) IS NOT NULL AND ` + countedExpr(s.DB)
	if len(matchIDs) > 0 {
		ph := strings.TrimSuffix(strings.Repeat("?,", len(matchIDs)), ",")
		query += ` AND g.match_id IN (` + ph + `)`
		for _, id := range matchIDs {
			args = append(args, id)
		}
	}

	rows, err := s.DB.Query(ctx, query, args...)
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
			return nil, fmt.Errorf("MatchBadges scan: %w", err)
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
		// p.score_1/score_2 are away scores; ConvertEMGLossToMWCLoss wants
		// current scores, and the Crawford sentinel is decoded on the way.
		mwcLoss := engine.ConvertEMGLossToMWCLoss(int(errMP),
			matchLength-domain.PointsAway(awayScore0), matchLength-domain.PointsAway(awayScore1),
			fMove, cubeValue, matchLength)
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

// TournamentBadges computes each tournament's reference-player PR and total MWC
// loss for the tenant, keyed by tournament id. See storage.TournamentBadge for
// why the badge is the reference player's own PR rather than a both-players
// pool.
func (s *StatsStore) TournamentBadges(ctx context.Context, scope string) (map[int64]storage.TournamentBadge, error) {
	tenant, targs := s.DB.TenantFilter("p", scope)
	query := `SELECT m.tournament_id, ` + statsErrExpr + ` as err_mp,
		COALESCE(p.score_1, 0), COALESCE(p.score_2, 0), mv.player,
		` + cubeMultiplierExpr + `, COALESCE(p.match_length, m.match_length, 0),
		COALESCE(CASE WHEN mv.player = 1 THEN m.player1_name ELSE m.player2_name END, ''),
		g.match_id ` +
		statsBaseJoin +
		` WHERE ` + tenant + ` AND a.position_id IS NOT NULL AND (` + statsErrExpr + `) IS NOT NULL
		AND m.tournament_id IS NOT NULL AND ` + countedExpr(s.DB)

	rows, err := s.DB.Query(ctx, query, targs...)
	if err != nil {
		return nil, fmt.Errorf("TournamentBadges query: %w", err)
	}
	defer rows.Close()

	// Accumulate per (tournament, player). The badge reports the reference
	// player's own PR, not a both-players pool — see storage.TournamentBadge.
	acc := make(map[int64]map[string]*storage.TournamentPlayerAcc)
	for rows.Next() {
		var tournamentID, errMP, matchID int64
		var awayScore0, awayScore1, rawPlayer, cubeValue, matchLength int
		var moverName string
		if err := rows.Scan(&tournamentID, &errMP, &awayScore0, &awayScore1, &rawPlayer, &cubeValue, &matchLength, &moverName, &matchID); err != nil {
			return nil, fmt.Errorf("TournamentBadges scan: %w", err)
		}
		byPlayer := acc[tournamentID]
		if byPlayer == nil {
			byPlayer = make(map[string]*storage.TournamentPlayerAcc)
			acc[tournamentID] = byPlayer
		}
		a := byPlayer[moverName]
		if a == nil {
			a = &storage.TournamentPlayerAcc{Matches: make(map[int64]struct{})}
			byPlayer[moverName] = a
		}
		fMove := 0
		if rawPlayer == -1 {
			fMove = 1
		}
		a.SumErr += errMP
		a.Cnt++
		a.Matches[matchID] = struct{}{}
		if mwcLoss := engine.ConvertEMGLossToMWCLoss(int(errMP),
			matchLength-domain.PointsAway(awayScore0), matchLength-domain.PointsAway(awayScore1),
			fMove, cubeValue, matchLength); !math.IsNaN(mwcLoss) {
			a.MWC += mwcLoss
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make(map[int64]storage.TournamentBadge, len(acc))
	for tournamentID, byPlayer := range acc {
		out[tournamentID] = storage.PickReferencePlayer(byPlayer)
	}
	return out, nil
}

// playerTableFilter strips the player selection and decision type, which the
// players table ignores by design (see storage.StatsStore.PlayerTable).
func playerTableFilter(filter storage.StatsFilter) storage.StatsFilter {
	f := filter
	f.PlayerName = ""
	f.PlayerAliases = nil
	f.DecisionType = -1
	return f
}

// moverNameExpr names the player who took the decision, whichever seat they sat
// in. It is the players table's GROUP BY key.
const moverNameExpr = "CASE WHEN mv.player = 1 THEN m.player1_name ELSE m.player2_name END"

// buildMatchWhereClause renders the filter's match-level predicates (dates,
// tournaments, match length) against the match table alone, scoped to the
// tenant, for the queries that count matches and games rather than decisions.
func (s *StatsStore) buildMatchWhereClause(scope string, filter storage.StatsFilter) (whereSQL string, args []any) {
	clauses, args := s.matchClauses(scope, filter)
	metClause, metArgs := metComparableMatch(s.DB, scope)
	clauses = append(clauses, metClause)
	args = append(args, metArgs...)
	return " WHERE " + strings.Join(clauses, " AND "), args
}

// matchClauses are the match-level predicates of filter on alias m — tenant,
// tournaments, dates, lengths — without the MET predicate, which each reader
// states at its own grain.
func (s *StatsStore) matchClauses(scope string, filter storage.StatsFilter) (clauses []string, args []any) {
	tenant, targs := s.DB.TenantFilter("m", scope)
	clauses, args = []string{tenant}, targs

	if len(filter.TournamentIDs) > 0 {
		ph := strings.TrimSuffix(strings.Repeat("?,", len(filter.TournamentIDs)), ",")
		clauses = append(clauses, "m.tournament_id IN ("+ph+")")
		for _, id := range filter.TournamentIDs {
			args = append(args, id)
		}
	}

	clauses, args = appendDateClauses(s.DB, clauses, args, filter)

	if len(filter.MatchLength) > 0 {
		ph := strings.TrimSuffix(strings.Repeat("?,", len(filter.MatchLength)), ",")
		clauses = append(clauses, "m.match_length IN ("+ph+")")
		for _, ml := range filter.MatchLength {
			args = append(args, ml)
		}
	}

	return clauses, args
}

// PlayerTable computes one row per player over the matches the filter retains,
// scoped to the tenant. See the StatsStore contract for what the filter honours
// and how rows are ordered; the arithmetic that turns these queries into rows
// lives in storage.BuildPlayerRows.
func (s *StatsStore) PlayerTable(ctx context.Context, scope string, filter storage.StatsFilter) ([]storage.PlayerRow, error) {
	return inReadSnapshot(ctx, s, func(s *StatsStore) ([]storage.PlayerRow, error) {
		return s.playerTable(ctx, scope, filter)
	})
}

func (s *StatsStore) playerTable(ctx context.Context, scope string, filter storage.StatsFilter) ([]storage.PlayerRow, error) {
	d := s.DB
	settings, err := librarySettings(ctx, s.DB, scope)
	if err != nil {
		return nil, fmt.Errorf("PlayerTable settings: %w", err)
	}
	f := playerTableFilter(filter)
	matchWhere, matchArgs := s.buildMatchWhereClause(scope, f)

	var decisions []storage.PlayerDecisionStat
	var snowieErr map[string]int64
	var luck map[string]storage.PlayerLuckAcc
	// The Snowie denominator per match: every checker decision with a
	// position, both seats, but an unscored play (UnscoredPlaySQL).
	checkerMoves := `COALESCE((SELECT COUNT(*) FROM move mv2
		                  JOIN position p2 ON p2.id = mv2.position_id
		                  JOIN game g2 ON g2.id = mv2.game_id
		                  WHERE g2.match_id = m.id AND p2.decision_type = 0 AND NOT ` + UnscoredPlaySQL("p2") + `), 0)`
	// A reader that cannot write takes the direct path, as Compute does,
	// unless the table already holds every match.
	readable := false
	if fromMatchStats(f) {
		if readable, err = s.matchStatsReadable(ctx, scope); err != nil {
			return nil, err
		}
	}
	if readable {
		decisions, snowieErr, luck, err = s.playerSumsFromTable(ctx, scope, f)
		checkerMoves = `COALESCE((SELECT SUM(ms.checker_moves) FROM match_stats ms WHERE ms.match_id = m.id), 0)`
	} else {
		decisions, snowieErr, luck, err = s.playerSumsDirect(ctx, scope, f, settings)
	}
	if err != nil {
		return nil, err
	}

	// ── Matches: participation, outcome, and the Snowie denominator ───────────
	var matches []storage.MatchOutcomeRow
	rows, err := s.DB.Query(ctx,
		`SELECT COALESCE(m.player1_name,''), COALESCE(m.player2_name,''), COALESCE(m.match_length,0),
		        `+d.Bigint(`COALESCE((SELECT SUM(g.points_won) FROM game g
		                  WHERE g.match_id = m.id AND g.winner = 1), 0)`)+`,
		        `+d.Bigint(`COALESCE((SELECT SUM(g.points_won) FROM game g
		                  WHERE g.match_id = m.id AND g.winner = -1), 0)`)+`,
		        `+checkerMoves+`
		 FROM match m`+matchWhere,
		matchArgs...)
	if err != nil {
		return nil, fmt.Errorf("PlayerTable matches: %w", err)
	}
	var matchesScanErr error
	func() {
		defer rows.Close()
		for rows.Next() {
			var m storage.MatchOutcomeRow
			if err := rows.Scan(&m.Player1, &m.Player2, &m.MatchLength,
				&m.Points1, &m.Points2, &m.CheckerMoves); err != nil {
				matchesScanErr = err
				return
			}
			matches = append(matches, m)
		}
	}()
	if matchesScanErr != nil {
		return nil, fmt.Errorf("PlayerTable matches scan: %w", matchesScanErr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("PlayerTable matches rows: %w", err)
	}

	aliases, err := aliasMap(ctx, s.DB, scope, storage.AliasPlayer)
	if err != nil {
		return nil, fmt.Errorf("PlayerTable aliases: %w", err)
	}
	decisions, matches, snowieErr, luck = canonicalPlayerInputs(aliases, decisions, matches, snowieErr, luck)
	return storage.BuildPlayerRows(decisions, matches, snowieErr, luck), nil
}

// playerSumsDirect aggregates the players table's per-player sums from the
// decisions themselves: the path of a filter match_stats cannot apply.
func (s *StatsStore) playerSumsDirect(ctx context.Context, scope string, f storage.StatsFilter, settings storage.LibrarySettings) (decisions []storage.PlayerDecisionStat, snowieErr map[string]int64, luck map[string]storage.PlayerLuckAcc, err error) {
	d := s.DB
	statsWhere, statsArgs := s.buildStatsWhereClause(scope, f)
	baseWhere, baseArgs := s.buildBaseWhereClause(scope, f)
	matchWhere, matchArgs := s.buildMatchWhereClause(scope, f)

	// ── Counted decisions, per player and decision type ───────────────────────
	rows, err := s.DB.Query(ctx,
		`SELECT `+moverNameExpr+` AS pname, p.decision_type,`+
			` `+d.Bigint(`COALESCE(SUM(`+statsErrExpr+`),0)`)+`, COUNT(*),`+
			` `+d.Bigint(`COALESCE(SUM(CASE WHEN (`+statsErrExpr+`) >= ? THEN 1 ELSE 0 END),0)`)+`,`+
			` `+d.Bigint(`COALESCE(SUM(CASE WHEN (`+statsErrExpr+`) >= ? THEN 1 ELSE 0 END),0)`)+` `+
			statsBaseJoin+statsWhere+
			` GROUP BY pname, p.decision_type`,
		append([]any{settings.ErrorThresholdMP, settings.BlunderThresholdMP}, statsArgs...)...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("PlayerTable decisions: %w", err)
	}
	var scanErr error
	func() {
		defer rows.Close()
		for rows.Next() {
			var d storage.PlayerDecisionStat
			if err := rows.Scan(&d.Name, &d.DecisionType, &d.SumErrMP, &d.Count, &d.Errors, &d.Blunders); err != nil {
				scanErr = err
				return
			}
			decisions = append(decisions, d)
		}
	}()
	if scanErr != nil {
		return nil, nil, nil, fmt.Errorf("PlayerTable decisions scan: %w", scanErr)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, nil, fmt.Errorf("PlayerTable decisions rows: %w", err)
	}

	// ── Snowie numerator: every error, counted or not ─────────────────────────
	snowieErr = map[string]int64{}
	rows, err = s.DB.Query(ctx,
		`SELECT `+moverNameExpr+` AS pname, `+d.Bigint(`COALESCE(SUM(`+statsErrExpr+`),0)`)+` `+
			statsBaseJoin+baseWhere+` GROUP BY pname`,
		baseArgs...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("PlayerTable snowie: %w", err)
	}
	var snowieScanErr error
	func() {
		defer rows.Close()
		for rows.Next() {
			var name string
			var sum int64
			if err := rows.Scan(&name, &sum); err != nil {
				snowieScanErr = err
				return
			}
			snowieErr[name] = sum
		}
	}()
	if snowieScanErr != nil {
		return nil, nil, nil, fmt.Errorf("PlayerTable snowie scan: %w", snowieScanErr)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, nil, fmt.Errorf("PlayerTable snowie rows: %w", err)
	}

	// ── Luck, over the rolls that carry it ────────────────────────────────────
	// COUNT(mv.luck_mp) skips NULLs, so the denominator is the number of rolls
	// actually measured — never the number played (ADR-0010).
	luck = map[string]storage.PlayerLuckAcc{}
	rows, err = s.DB.Query(ctx,
		`SELECT `+moverNameExpr+` AS pname, `+d.Bigint(`COALESCE(SUM(mv.luck_mp),0)`)+`, COUNT(mv.luck_mp)
		 FROM move mv
		 JOIN game g ON g.id = mv.game_id
		 JOIN match m ON m.id = g.match_id`+matchWhere+
			` AND mv.luck_mp IS NOT NULL GROUP BY pname`,
		matchArgs...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("PlayerTable luck: %w", err)
	}
	var luckScanErr error
	func() {
		defer rows.Close()
		for rows.Next() {
			var name string
			var acc storage.PlayerLuckAcc
			if err := rows.Scan(&name, &acc.SumMP, &acc.Rolls); err != nil {
				luckScanErr = err
				return
			}
			luck[name] = acc
		}
	}()
	if luckScanErr != nil {
		return nil, nil, nil, fmt.Errorf("PlayerTable luck scan: %w", luckScanErr)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, nil, fmt.Errorf("PlayerTable luck rows: %w", err)
	}

	return decisions, snowieErr, luck, nil
}

// canonicalPlayerInputs renames every alias to its canonical name in the
// players table's raw sums, so one person signing two ways is one row. The
// sums merge before any rate is taken: a PR is never averaged.
func canonicalPlayerInputs(aliases storage.AliasMap, decisions []storage.PlayerDecisionStat, matches []storage.MatchOutcomeRow,
	snowieErr map[string]int64, luck map[string]storage.PlayerLuckAcc,
) ([]storage.PlayerDecisionStat, []storage.MatchOutcomeRow, map[string]int64, map[string]storage.PlayerLuckAcc) {
	if len(aliases) == 0 {
		return decisions, matches, snowieErr, luck
	}
	type key struct {
		name string
		dt   int
	}
	idx := map[key]int{}
	var dec []storage.PlayerDecisionStat
	for _, d := range decisions {
		d.Name = aliases.Canonical(d.Name)
		k := key{d.Name, int(d.DecisionType)}
		if i, ok := idx[k]; ok {
			dec[i].SumErrMP += d.SumErrMP
			dec[i].Count += d.Count
			dec[i].Errors += d.Errors
			dec[i].Blunders += d.Blunders
			continue
		}
		idx[k] = len(dec)
		dec = append(dec, d)
	}
	for i := range matches {
		matches[i].Player1 = aliases.Canonical(matches[i].Player1)
		matches[i].Player2 = aliases.Canonical(matches[i].Player2)
	}
	sn := make(map[string]int64, len(snowieErr))
	for n, v := range snowieErr {
		sn[aliases.Canonical(n)] += v
	}
	lk := make(map[string]storage.PlayerLuckAcc, len(luck))
	for n, v := range luck {
		c := aliases.Canonical(n)
		acc := lk[c]
		acc.SumMP += v.SumMP
		acc.Rolls += v.Rolls
		lk[c] = acc
	}
	return dec, matches, sn, lk
}
