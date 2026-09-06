package sqlshared

import (
	"context"
	"iter"
	"math"
	"strconv"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/searchfilter"
)

// SearchStore implements storage.SearchStore. position is a domain table:
// the query is confined to the scope's tenant through Dialect.TenantFilter.
type SearchStore struct{ DB Execer }

var _ storage.SearchStore = (*SearchStore)(nil)

// Find streams the positions matching f. It is a faithful port of the
// Database wrapper's LoadPositionsByFiltersCore: the cheap predicates are
// pushed to SQL, the rest are evaluated in Go on the narrowed result set.
// Results are restricted to the scope's tenant.
//
// opts.Limit/Offset are pushed into the SQL query itself (LIMIT/OFFSET on the
// ORDER BY that already picks the result order), bounding what the SQL scan
// returns before any of it reaches Go. A zero ListOpts keeps today's
// behaviour: no limit, from the start. Because the Go-side predicates below
// (mirror search, the checker-structure/date/equity/move-pattern filters)
// still run AFTER the SQL scan, on the page it returned, they can reject a
// page's candidates the same way they already reject any SQL-matched row —
// the guarantee is "at most opts.Limit SQL-matched candidates were
// considered", not "exactly opts.Limit results returned"; a caller paging
// through a search that also uses one of those filters may see short pages.
func (s *SearchStore) Find(ctx context.Context, scope string, f domain.SearchFilters, opts storage.ListOpts) iter.Seq2[*domain.Position, error] {
	return func(yield func(*domain.Position, error) bool) {
		positions, err := s.find(ctx, scope, f, opts)
		if err != nil {
			yield(nil, err)
			return
		}
		for i := range positions {
			if !yield(&positions[i], nil) {
				return
			}
		}
	}
}

// searchWhereClause is what buildWhere hands find: the WHERE clause text and
// its bound arguments, plus the state later phases need that buildWhere
// already had to compute while reading f (B.15, #183 — find used to carry
// all of this, and the query execution, the row scan and the Go-side filter
// pass, as one 730-line function).
type searchWhereClause struct {
	where         string
	args          []any
	needAnalysis  bool
	useSQLFilters bool
	bitboardTight bool
	// multiPlayed lists the positions player 1 played more than one way;
	// only filled by a plain move-error search, where those rows escape the
	// SQL column and are scored in Go (#167).
	multiPlayed map[int64]bool
	// effInclude is f.Filter with the points shared with ExcludeFilter
	// cleared, so "Except" wins over "At least" on those points.
	effInclude domain.Position
}

// buildWhere translates f into the WHERE clause of the search query: cheap
// predicates that can be pushed to SQL become clause text and bound
// arguments; the rest are left to applyGoFilters (matchesGoFilters below),
// which is what needAnalysis/useSQLFilters/bitboardTight/multiPlayed/
// effInclude in the returned searchWhereClause are for.
func (s *SearchStore) buildWhere(ctx context.Context, scope string, f domain.SearchFilters) (searchWhereClause, error) {
	useSQLFilters := !f.MirrorFilter
	// multiPlayed lists the positions player 1 played more than one way;
	// only filled by a plain move-error search, where those rows escape the
	// SQL column and are scored in Go (#167).
	var multiPlayed map[int64]bool

	// The decoded analysis is consumed by the move-pattern filter, the Go-side
	// analysis re-checks of mirror search, and the date/equity filters below —
	// every other analysis filter (win/gammon/backgammon rate, cube error,
	// move error) runs on the denormalised SQL columns instead. So decode the
	// (zlib-compressed) blob per row only when one of those paths needs it — a
	// search using none of them skips the decompress+unmarshal of every row.
	//
	// MoveErrorFilter is deliberately NOT one of the triggers: it is pushed to
	// SQL like the rate filters (statsErrExpr in the WHERE builder below), and
	// its Go-side re-check (matchesMoveErrorFilter) runs on the mirror path
	// (`!useSQLFilters`, already covered by the `|| f.MirrorFilter` term
	// below) and, on the plain path, only for the handful of positions player
	// 1 played more than once (multiPlayed below), whose analysis is loaded
	// one by one after the scan. Adding it here used to force a bulk a.data
	// decode on every plain MoveErrorFilter search even though nothing read
	// the result: on the tournois fixture that turned
	// BenchmarkSearch_ErrorAboveTenth's ~2 200 SQL-matched rows into ~2 200
	// needless decodes, ~80ms → ~200ms.
	//
	// DateFilter has no SQL pushdown at all (unlike MoveErrorFilter) and used
	// to decode independently, once per candidate row, inside
	// searchfilter.MatchesDateFilter (a second query plus a second decompression on top of
	// this one whenever both ran). Folding it into needAnalysis makes this the
	// only decode.
	needAnalysis := f.MovePatternFilter != "" || f.MirrorFilter ||
		f.DateFilter != "" || f.EquityFilter != ""

	// On points shared with the exclusion structure, "Except" wins over "At least":
	// clear those points from the include filter so the two are not contradictory.
	effInclude := domain.EffectiveIncludeFilter(f.Filter, f.ExcludeFilter)

	// The tenant predicate comes first, and its arguments first, so the
	// placeholders line up once the PostgreSQL adapter has rebound them.
	var where strings.Builder
	tenant, args := s.DB.TenantFilter("p", scope)
	where.WriteString(tenant)

	// Provenance is a property of the row, not of the board, so mirroring a
	// position cannot change it: this one filter stays in SQL even in mirror
	// search, where every board filter falls back to the Go phase.
	if f.IndividuallyImportedFilter {
		where.WriteString(" AND " + s.DB.Bool("p.individually_imported", true))
	}

	// The source-tool study mark is likewise a property of the row, so it too
	// stays in SQL even in mirror search.
	if f.FlaggedFilter {
		where.WriteString(" AND " + s.DB.Bool("p.flagged", true))
	}

	// Whether a position carries a comment is likewise a property of the row and
	// not of the board, so this too stays in SQL even in mirror search. Keeping
	// it here rather than in the Go phase also matters for cost: the Go-side
	// SearchText check runs one query per candidate position, which is fine for
	// a rarely-used content filter but not for a presence filter that is
	// routinely the only thing narrowing the scan.
	//
	// COALESCE is deliberate: comment.text is nullable, and a bare
	// `c.text <> ''` evaluates to NULL — not false — on a NULL row, which would
	// silently drop it from EXISTS and keep it in NOT EXISTS. Empty text counts
	// as no comment either way (see CONTEXT.md).
	//
	// On PostgreSQL the subquery carries tenant_id as well as position_id: it
	// is what idx_comment_position is keyed on, and RLS aside, a scope must
	// never read across tenants.
	s.appendClosedListClauses(scope, f, &where, &args)

	if f.MatchIDsFilter != "" || f.TournamentIDsFilter != "" {
		var allMatchIDs []int64
		if f.MatchIDsFilter != "" {
			if ids, err := searchfilter.ParseFilterIDList(f.MatchIDsFilter); err == nil {
				allMatchIDs = append(allMatchIDs, ids...)
			}
		}
		if f.TournamentIDsFilter != "" {
			if tIDs, err := searchfilter.ParseFilterIDList(f.TournamentIDsFilter); err == nil {
				for _, tID := range tIDs {
					// A query failure here (a locked database, a dropped
					// connection) must not silently narrow the tournament
					// filter to "no matches" — that reads as "this
					// tournament has no positions", not as the outage it
					// is (B.6, #174).
					matchIDs, err := getMatchIDsForTournament(ctx, s.DB, tID)
					if err != nil {
						return searchWhereClause{}, err
					}
					allMatchIDs = append(allMatchIDs, matchIDs...)
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
			where.WriteString(" AND 0=1")
		}
	}

	// Player filter: keep positions that occur in any match where the named
	// player sat at either seat. A case-insensitive LIKE with no wildcards
	// (Dialect.ILike: SQLite's LIKE is already case-insensitive for ASCII,
	// PostgreSQL needs ILIKE) gives exact matching for ASCII names, mirroring
	// the match-id subquery shape.
	// The frontend sends the token whole (`pl"Name"`); the CLI and the server
	// send a bare name. searchfilter.PlayerName accepts both.
	if playerName := searchfilter.PlayerName(f.PlayerFilter); playerName != "" {
		like := s.DB.ILike()
		where.WriteString(
			" AND p.id IN (SELECT mv.position_id FROM move mv" +
				" JOIN game g ON mv.game_id = g.id" +
				" JOIN match mt ON g.match_id = mt.id" +
				" WHERE mt.player1_name " + like + " ? OR mt.player2_name " + like + " ?)")
		args = append(args, playerName, playerName)
	}

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

	// User-facing position-id filter (command-line token `id`). Uses the same
	// list/range semantics as the match/tournament filters (e.g. "2,7" is the
	// range 2..7; ";"-joined values are an explicit list).
	if f.PositionIDsFilter != "" {
		ids, err := searchfilter.ParseFilterIDList(f.PositionIDsFilter)
		if err == nil && len(ids) > 0 {
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

	var bitboardTight bool
	if useSQLFilters {
		if f.DecisionTypeFilter {
			where.WriteString(" AND p.decision_type = ? AND p.player_on_roll = ?")
			args = append(args, f.Filter.DecisionType, f.Filter.PlayerOnRoll)
			// Cube sub-type: distinguish double/no-double from take/pass responses.
			if f.Filter.DecisionType == domain.CubeAction {
				switch f.CubeResponseFilter {
				case "double":
					where.WriteString(" AND " + s.DB.Bool("p.is_cube_response", false))
				case "takepass":
					where.WriteString(" AND " + s.DB.Bool("p.is_cube_response", true))
				}
			}
		}
		if f.DiceRollFilter {
			if f.DiceRollMode == "first" {
				where.WriteString(" AND (p.dice_1 = ? OR p.dice_2 = ?) AND p.player_on_roll = ? AND p.decision_type = ?")
				args = append(args, f.Filter.Dice[0], f.Filter.Dice[0], f.Filter.PlayerOnRoll, f.Filter.DecisionType)
			} else {
				d1, d2 := f.Filter.Dice[0], f.Filter.Dice[1]
				if d1 == d2 {
					where.WriteString(" AND p.dice_1 = ? AND p.dice_2 = ? AND p.player_on_roll = ? AND p.decision_type = ?")
					args = append(args, d1, d2, f.Filter.PlayerOnRoll, f.Filter.DecisionType)
				} else {
					where.WriteString(" AND ((p.dice_1 = ? AND p.dice_2 = ?) OR (p.dice_1 = ? AND p.dice_2 = ?)) AND p.player_on_roll = ? AND p.decision_type = ?")
					args = append(args, d1, d2, d2, d1, f.Filter.PlayerOnRoll, f.Filter.DecisionType)
				}
			}
		}
		// Except-dice (xD65): exclude positions rolled with any of the listed rolls,
		// each in either order. Unscoped by on-roll/decision-type — a roll is a roll
		// whoever holds it; cube decisions (dice 0-0) never match, so they survive.
		for _, pair := range domain.ParseExceptDice(f.ExceptDiceFilter) {
			where.WriteString(" AND NOT ((p.dice_1 = ? AND p.dice_2 = ?) OR (p.dice_1 = ? AND p.dice_2 = ?))")
			args = append(args, pair[0], pair[1], pair[1], pair[0])
		}
		if f.IncludeCube {
			if f.Filter.Cube.Value == 0 {
				where.WriteString(" AND p.cube_value IS NULL")
			} else if f.DecisionTypeFilter && f.CubeResponseFilter == "takepass" {
				// A take/pass offered cube is always centered (owner -1); the board
				// can't build a centered value>1 cube, so match the centered owner.
				where.WriteString(" AND p.cube_value = ? AND p.cube_owner = -1")
				args = append(args, f.Filter.Cube.Value)
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
			where.WriteString(" AND " + s.DB.Bool("p.no_contact", true))
		}

		pMin, pMax, pHasMin, pHasMax := searchfilter.ParseIntFilterExpr(f.PipCountFilter, "p")
		searchfilter.AppendIntRangeSQL("p.pip_diff", pMin, pMax, pHasMin, pHasMax, &where, &args)
		PMin, PMax, PHasMin, PHasMax := searchfilter.ParseIntFilterExpr(f.Player1AbsolutePipCountFilter, "P")
		searchfilter.AppendIntRangeSQL("p.pip_1", PMin, PMax, PHasMin, PHasMax, &where, &args)
		oMin, oMax, oHasMin, oHasMax := searchfilter.ParseIntFilterExpr(f.Player1CheckerOffFilter, "o")
		searchfilter.AppendIntRangeSQL("p.off_1", oMin, oMax, oHasMin, oHasMax, &where, &args)
		OMin, OMax, OHasMin, OHasMax := searchfilter.ParseIntFilterExpr(f.Player2CheckerOffFilter, "O")
		searchfilter.AppendIntRangeSQL("p.off_2", OMin, OMax, OHasMin, OHasMax, &where, &args)
		kMin, kMax, kHasMin, kHasMax := searchfilter.ParseIntFilterExpr(f.Player1BackCheckerFilter, "k")
		searchfilter.AppendIntRangeSQL("p.back_checkers_1", kMin, kMax, kHasMin, kHasMax, &where, &args)
		KMin, KMax, KHasMin, KHasMax := searchfilter.ParseIntFilterExpr(f.Player2BackCheckerFilter, "K")
		searchfilter.AppendIntRangeSQL("p.back_checkers_2", KMin, KMax, KHasMin, KHasMax, &where, &args)

		// Win/gammon rate: pushed as `p.id IN (SELECT position_id FROM analysis
		// WHERE …)` rather than a plain `AND a.player1_win_rate/gammon_rate …`
		// clause on the outer LEFT JOIN. With the LEFT JOIN form the planner's
		// only efficient path is idx_analysis_win_gammon(win_rate, gammon_rate),
		// which returns rows ordered by rate, not by p.id — the ORDER BY at the
		// end of this query then needs a full TEMP B-TREE sort. Feeding p.id
		// through an IN-subquery instead lets SQLite keep scanning `position` in
		// its natural (already p.id-ordered) rowid order and test membership per
		// row, so the sort disappears entirely; idx_analysis_win_gammon now
		// carries position_id as a third column (schema_sqlite.go) so the
		// subquery is answered from the index alone, no analysis-table lookup.
		// See FOLLOWUPS.md #4 and fiche-05 T3 for the verified EXPLAIN QUERY PLAN.
		// PostgreSQL's idx_analysis_win_gammon_covering plays the same role,
		// which is why the subquery also carries the tenant predicate there.
		var winGammonWhere strings.Builder
		var winGammonArgs []any
		wMin, wMax, wHasMin, wHasMax := searchfilter.ParseFloatFilterExpr(f.WinRateFilter, "w")
		searchfilter.AppendIntRangeSQL("player1_win_rate", int(math.Round(wMin*100)), int(math.Round(wMax*100)), wHasMin, wHasMax, &winGammonWhere, &winGammonArgs)
		gMin, gMax, gHasMin, gHasMax := searchfilter.ParseFloatFilterExpr(f.GammonRateFilter, "g")
		searchfilter.AppendIntRangeSQL("player1_gammon_rate", int(math.Round(gMin*100)), int(math.Round(gMax*100)), gHasMin, gHasMax, &winGammonWhere, &winGammonArgs)
		if winGammonWhere.Len() > 0 {
			aTenant, aArgs := s.DB.TenantFilter("", scope)
			where.WriteString(" AND p.id IN (SELECT position_id FROM analysis WHERE " + aTenant + winGammonWhere.String() + ")")
			args = append(args, aArgs...)
			args = append(args, winGammonArgs...)
		}
		bMin, bMax, bHasMin, bHasMax := searchfilter.ParseFloatFilterExpr(f.BackgammonRateFilter, "b")
		searchfilter.AppendIntRangeSQL("a.player1_backgammon_rate", int(math.Round(bMin*100)), int(math.Round(bMax*100)), bHasMin, bHasMax, &where, &args)
		WMin, WMax, WHasMin, WHasMax := searchfilter.ParseFloatFilterExpr(f.Player2WinRateFilter, "W")
		searchfilter.AppendIntRangeSQL("a.player2_win_rate", int(math.Round(WMin*100)), int(math.Round(WMax*100)), WHasMin, WHasMax, &where, &args)
		GMin, GMax, GHasMin, GHasMax := searchfilter.ParseFloatFilterExpr(f.Player2GammonRateFilter, "G")
		searchfilter.AppendIntRangeSQL("a.player2_gammon_rate", int(math.Round(GMin*100)), int(math.Round(GMax*100)), GHasMin, GHasMax, &where, &args)
		BMin, BMax, BHasMin, BHasMax := searchfilter.ParseFloatFilterExpr(f.Player2BackgammonRateFilter, "B")
		searchfilter.AppendIntRangeSQL("a.player2_backgammon_rate", int(math.Round(BMin*100)), int(math.Round(BMax*100)), BHasMin, BHasMax, &where, &args)

		// The denormalised error column scores ONE play (the first of the
		// analysis' PlayedMoves, see AnalysisStore.Save) — exact for a position
		// player 1 played once, and that is the whole table but for a few
		// openings. A position played several ways is let through regardless
		// and settled in Go by matchesMoveErrorFilter, which takes the largest
		// error among the plays (#167): the column, being one of them, can
		// only under-state it, so "E>x" would silently drop the position and
		// "E<x" would keep it. The set is listed once, before the scan
		// (multiPlayedPlayer1Positions): a correlated subquery here ran on
		// every row the column rejected and doubled the query's time.
		if f.MoveErrorFilter != "" {
			var err error
			if multiPlayed, err = multiPlayedPlayer1Positions(ctx, s.DB, scope); err != nil {
				return searchWhereClause{}, err
			}
			eMin, eMax, eHasMin, eHasMax := searchfilter.ParseFloatFilterExpr(f.MoveErrorFilter, "E")
			eqMin := int(math.Round(eMin))
			eqMax := int(math.Round(eMax))
			var cond string
			if eHasMin && eHasMax {
				cond = statsErrExpr + " BETWEEN ? AND ?"
				args = append(args, eqMin, eqMax)
			} else if eHasMin {
				cond = statsErrExpr + " >= ?"
				args = append(args, eqMin)
			} else if eHasMax {
				cond = statsErrExpr + " <= ?"
				args = append(args, eqMax)
			}
			if cond != "" {
				if len(multiPlayed) > 0 {
					placeholders := strings.Repeat("?,", len(multiPlayed))
					cond = "(" + cond + " OR p.id IN (" + placeholders[:len(placeholders)-1] + "))"
					for id := range multiPlayed {
						args = append(args, id)
					}
				}
				where.WriteString(" AND " + cond)
			}
		}

		if searchfilter.HasBoardFilter(effInclude.Board) {
			occ1Req, pt1Req, occ2Req, pt2Req, tight := engine.CheckerStructureMasks(effInclude)
			bitboardTight = tight
			where.WriteString(" AND (p.occupancy_1 & ?) = ? AND (p.point_mask_1 & ?) = ?")
			where.WriteString(" AND (p.occupancy_2 & ?) = ? AND (p.point_mask_2 & ?) = ?")
			args = append(args,
				int64(occ1Req), int64(occ1Req), int64(pt1Req), int64(pt1Req),
				int64(occ2Req), int64(occ2Req), int64(pt2Req), int64(pt2Req))
		}

		// Exclusion structure ("Sauf"): drop positions that contain ANY of the
		// excluded elements (OR semantics across points). Keep a position only when
		// none of its points match an excluded element. Template points with >2
		// checkers are not representable as bitmasks and are left to the Go-side
		// check (Position.ContainsAnyCheckerOf) below.
		if searchfilter.HasBoardFilter(f.ExcludeFilter.Board) {
			eSingle1, eMade1, eSingle2, eMade2 := engine.ExclusionMasks(f.ExcludeFilter)
			where.WriteString(" AND (p.occupancy_1 & ?) = 0 AND (p.point_mask_1 & ?) = 0")
			where.WriteString(" AND (p.occupancy_2 & ?) = 0 AND (p.point_mask_2 & ?) = 0")
			args = append(args,
				int64(eSingle1), int64(eMade1), int64(eSingle2), int64(eMade2))
		}
	}

	return searchWhereClause{
		where:         where.String(),
		args:          args,
		needAnalysis:  needAnalysis,
		useSQLFilters: useSQLFilters,
		bitboardTight: bitboardTight,
		multiPlayed:   multiPlayed,
		effInclude:    effInclude,
	}, nil
}

func (s *SearchStore) find(ctx context.Context, scope string, f domain.SearchFilters, opts storage.ListOpts) ([]domain.Position, error) {
	wc, err := s.buildWhere(ctx, scope, f)
	if err != nil {
		return nil, err
	}

	// a.data is the compressed analysis blob (~600 bytes/row on the tournois
	// fixture) and is the only column here wc.needAnalysis gates: every other
	// selected analysis column is a cheap denormalised scalar used by the SQL
	// WHERE clause itself. A search that needs none of the Go-side
	// analysis-dependent filters (move pattern, mirror, date, move-error,
	// equity — see wc.needAnalysis above) has no use for the blob, so skip
	// fetching and transporting it: NULL is 1 byte on the wire instead of ~600,
	// for every row, sorted or not.
	analysisDataCol := "NULL"
	if wc.needAnalysis {
		analysisDataCol = "a.data"
	}

	limitClause, limitArgs := s.DB.LimitOffset(opts.Limit, opts.Offset)

	query := `SELECT p.id, p.state,
		p.decision_type, p.player_on_roll, p.dice_1, p.dice_2,
		p.cube_value, p.cube_owner, p.score_1, p.score_2,
		p.has_jacoby, p.has_beaver, p.is_cube_response,
		p.individually_imported, p.flagged,
		a.id, ` + analysisDataCol + ` AS data
	FROM position p
	LEFT JOIN analysis a ON a.position_id = p.id
	WHERE ` + wc.where + ` ORDER BY ` + domain.SearchOrderByClause(f.Sort) + limitClause

	rows, err := s.DB.Query(ctx, query, append(wc.args, limitArgs...)...)
	if err != nil {
		return nil, errf(s.DB, "search query", err)
	}
	defer rows.Close()

	scanned, err := s.scanRows(rows, wc.needAnalysis)
	if err != nil {
		return nil, err
	}

	return s.applyGoFilters(ctx, f, wc, scanned)
}

// scannedRow is one row of buildWhere's query, decoded into the shape
// applyGoFilters works with. is_cube_response is read from its own column
// rather than the position blob, so it has to travel with the row to the
// filter phase.
type scannedRow struct {
	pos            domain.Position
	ana            *domain.PositionAnalysis
	isCubeResponse bool
}

// scanRows drains rows into a []scannedRow before any Go-side filtering
// starts, decoding each row's compressed analysis blob when needAnalysis
// says a later filter will read it.
func (s *SearchStore) scanRows(rows Rows, needAnalysis bool) ([]scannedRow, error) {
	// Drain the cursor before filtering. A cursor holds a pooled connection
	// until it is exhausted, and the Go-side predicates below open queries of
	// their own (comment text, creation date, played-move error, take/pass cube
	// action). Running them inside the scan loop therefore needs a second
	// connection for the whole duration of the scan — which an ":memory:"
	// SQLite database can never provide, being pinned to exactly one
	// connection (sqlite.ConfigurePool): the nested query waits for a
	// connection only the cursor can release, and the cursor only advances
	// once the nested query answers. A file or PostgreSQL pool merely
	// postpones the same shape: once enough concurrent searches each hold a
	// cursor, every connection is a cursor waiting for a connection that will
	// never come.
	//
	// Buffering costs nothing here: find already materialises its whole result
	// set, so these rows were going to be held in memory regardless.
	var scanned []scannedRow

	for rows.Next() {
		// Nullable columns scan into pointers, which both drivers leave nil on
		// NULL; the flags are INTEGER 0/1 on SQLite and BOOLEAN on PostgreSQL,
		// and both drivers convert either into a *bool.
		var posID int64
		var posState string
		var pDT, pPOR, pD1, pD2, pCV, pCO, pS1, pS2 *int64
		var pHJ, pHB, pICR, pII, pFlag *bool
		var anaID *int64
		var anaData []byte

		if err := rows.Scan(
			&posID, &posState,
			&pDT, &pPOR, &pD1, &pD2, &pCV, &pCO, &pS1, &pS2, &pHJ, &pHB, &pICR,
			&pII, &pFlag,
			&anaID, &anaData,
		); err != nil {
			return nil, errf(s.DB, "search scan", err)
		}

		position := engine.ReconstructPosition(posID, posState,
			derefInt(pDT), derefInt(pPOR), derefInt(pD1), derefInt(pD2),
			derefInt(pCV), derefInt(pCO), derefInt(pS1), derefInt(pS2),
			boolToInt(pHJ), boolToInt(pHB))
		// Row properties rather than board identity, so they are applied on top
		// of the reconstructed position (ADR-0001, docs/adr/0006). Without this
		// a searched position always came back unmarked, unlike the same
		// position read through PositionStore.Load.
		position.IndividuallyImported = pII != nil && *pII
		position.Flagged = pFlag != nil && *pFlag

		var ana *domain.PositionAnalysis
		if needAnalysis && anaID != nil && len(anaData) > 0 {
			// a.data is stored compressed (engine.EncodeAnalysisForStorage;
			// see AnalysisStore.Save), so it must go through the same decoder as
			// AnalysisStore.Load. A bare json.Unmarshal of the compressed bytes
			// silently fails (first byte is the zstd/zlib header, never '{'), leaving
			// ana nil on every row — which broke every analysis-dependent Go-side
			// filter (move pattern, the win/gammon/equity fallbacks used by
			// mirror search).
			if a, decErr := engine.DecodeAnalysisFromStorage(anaData); decErr == nil {
				ana = &a
			}
		}

		scanned = append(scanned, scannedRow{pos: position, ana: ana, isCubeResponse: pICR != nil && *pICR})
	}
	if err := rows.Err(); err != nil {
		return nil, errf(s.DB, "search rows", err)
	}
	// Hand the connection back before the predicates start querying.
	if err := rows.Close(); err != nil {
		return nil, errf(s.DB, "search rows close", err)
	}

	return scanned, nil
}

// applyGoFilters runs the Go-side predicates buildWhere could not push to
// SQL against each scanned row (and, for MirrorFilter, its mirror image
// too), preloading per-family batch queries first — exactly what find used
// to do inline, now split out so buildWhere/scanRows/applyGoFilters can each
// be read (and in buildWhere's case, tested) on their own (B.15, #183).
func (s *SearchStore) applyGoFilters(ctx context.Context, f domain.SearchFilters, wc searchWhereClause, scanned []scannedRow) ([]domain.Position, error) {
	var err error
	// Preload, in one batched query per family, what the per-row predicates
	// below used to fetch one row at a time: a SearchText filter checked every
	// SQL-matched candidate's comment with its own query (loadCommentText),
	// and a MoveErrorFilter (plus the take/pass mirror check in addPosition)
	// checked every candidate's recorded plays with another — 2 000 SQL-matched
	// rows meant 2 000-4 000 extra round trips (B.10, #178). Both preloads are
	// gated on the filter actually being active, and both run only once the
	// cursor above is drained and its connection is free.
	var commentTexts map[int64]string
	var player1MovesByID map[int64]player1Moves
	if f.SearchText != "" || f.MoveErrorFilter != "" {
		ids := make([]int64, len(scanned))
		for i, row := range scanned {
			ids[i] = row.pos.ID
		}
		if f.SearchText != "" {
			commentTexts, err = loadCommentTexts(ctx, s.DB, ids)
			if err != nil {
				return nil, errf(s.DB, "search preload comments", err)
			}
		}
		if f.MoveErrorFilter != "" {
			player1MovesByID, err = loadPlayer1Moves(ctx, s.DB, ids)
			if err != nil {
				return nil, errf(s.DB, "search preload player-1 moves", err)
			}
		}
	}

	var positions []domain.Position
	// Built once for the whole scan: the six checks only depend on f, not on
	// the row being tested.
	rates := rateFilterChecks(f)

	for _, row := range scanned {
		position, ana := row.pos, row.ana

		matchesGoFilters := func(pos domain.Position) (bool, error) {
			if searchfilter.HasBoardFilter(wc.effInclude.Board) {
				if !wc.useSQLFilters || wc.bitboardTight {
					if !pos.MatchesCheckerPosition(wc.effInclude) {
						return false, nil
					}
				}
			}

			// Exclusion structure: reject positions that contain ANY excluded element
			// (authoritative; also covers template counts >2 the SQL mask skips).
			if searchfilter.HasBoardFilter(f.ExcludeFilter.Board) {
				if pos.ContainsAnyCheckerOf(f.ExcludeFilter) {
					return false, nil
				}
			}

			if !wc.useSQLFilters {
				if !pos.MatchesCheckerPosition(wc.effInclude) {
					return false, nil
				}
				if f.IncludeCube && !pos.MatchesCubePosition(f.Filter) {
					return false, nil
				}
				if f.IncludeScore && !pos.MatchesScorePosition(f.Filter) {
					return false, nil
				}
				if f.DecisionTypeFilter && !pos.MatchesDecisionType(f.Filter) {
					return false, nil
				}
				// Cube sub-type (take/pass vs double/no-double) lives in the
				// is_cube_response column, scanned separately above.
				if f.DecisionTypeFilter && f.Filter.DecisionType == domain.CubeAction {
					isResp := row.isCubeResponse
					if f.CubeResponseFilter == "double" && isResp {
						return false, nil
					}
					if f.CubeResponseFilter == "takepass" && !isResp {
						return false, nil
					}
				}
				if f.DiceRollFilter && !pos.MatchesDiceRollMode(f.Filter, f.DiceRollMode) {
					return false, nil
				}
				if f.ExceptDiceFilter != "" && !pos.MatchesExceptDice(domain.ParseExceptDice(f.ExceptDiceFilter)) {
					return false, nil
				}
				if f.NoContactFilter && !pos.MatchesNoContact() {
					return false, nil
				}
				if f.PipCountFilter != "" && !pos.MatchesPipCountFilter(f.PipCountFilter) {
					return false, nil
				}
				if f.Player1AbsolutePipCountFilter != "" && !pos.MatchesPlayer1AbsolutePipCount(f.Player1AbsolutePipCountFilter) {
					return false, nil
				}
				if f.Player1CheckerOffFilter != "" && !pos.MatchesPlayer1CheckerOff(f.Player1CheckerOffFilter) {
					return false, nil
				}
				if f.Player2CheckerOffFilter != "" && !pos.MatchesPlayer2CheckerOff(f.Player2CheckerOffFilter) {
					return false, nil
				}
				if f.Player1BackCheckerFilter != "" && !pos.MatchesPlayer1BackChecker(f.Player1BackCheckerFilter) {
					return false, nil
				}
				if f.Player2BackCheckerFilter != "" && !pos.MatchesPlayer2BackChecker(f.Player2BackCheckerFilter) {
					return false, nil
				}
				if !matchesRateFilters(rates, ana) {
					return false, nil
				}
				if f.MoveErrorFilter != "" {
					if !matchesMoveErrorFilterPreloaded(ana, player1MovesByID[pos.ID], f.MoveErrorFilter) {
						return false, nil
					}
				}
			} else if f.MoveErrorFilter != "" && wc.multiPlayed[pos.ID] {
				// The SQL column scored one play; a multi-played position is
				// scored here by its largest error (#167). Its blob was not
				// fetched with the scan (wc.needAnalysis is false on this path
				// unless another filter wanted it), so load it now — the set
				// is a handful of rows, and the cursor is already drained.
				if ana == nil {
					ana = loadAnalysis(ctx, s.DB, pos.ID)
				}
				if !matchesMoveErrorFilterPreloaded(ana, player1MovesByID[pos.ID], f.MoveErrorFilter) {
					return false, nil
				}
			}

			if f.Player1CheckerInZoneFilter != "" && !pos.MatchesPlayer1CheckerInZone(f.Player1CheckerInZoneFilter) {
				return false, nil
			}
			if f.Player2CheckerInZoneFilter != "" && !pos.MatchesPlayer2CheckerInZone(f.Player2CheckerInZoneFilter) {
				return false, nil
			}
			if f.Player1OutfieldBlotFilter != "" && !pos.MatchesPlayer1OutfieldBlot(f.Player1OutfieldBlotFilter) {
				return false, nil
			}
			if f.Player2OutfieldBlotFilter != "" && !pos.MatchesPlayer2OutfieldBlot(f.Player2OutfieldBlotFilter) {
				return false, nil
			}
			if f.Player1JanBlotFilter != "" && !pos.MatchesPlayer1JanBlot(f.Player1JanBlotFilter) {
				return false, nil
			}
			if f.Player2JanBlotFilter != "" && !pos.MatchesPlayer2JanBlot(f.Player2JanBlotFilter) {
				return false, nil
			}
			if f.SearchText != "" && !matchesSearchTextPreloaded(commentTexts[pos.ID], f.SearchText) {
				return false, nil
			}
			if f.DateFilter != "" && !searchfilter.MatchesDateFilter(ana, f.DateFilter) {
				return false, nil
			}
			if f.EquityFilter != "" && !searchfilter.AnalysisMatchesEquityFilter(f.EquityFilter, ana) {
				return false, nil
			}
			return true, nil
		}

		addPosition := func(pos domain.Position) {
			if f.MoveErrorFilter != "" && pos.DecisionType == domain.CubeAction &&
				isPlayer1TakePassCubeActionPreloaded(player1MovesByID[pos.ID]) {
				pos = pos.Mirror()
			}
			positions = append(positions, pos)
		}

		ok, err := matchesGoFilters(position)
		if err != nil {
			return nil, err
		}
		if ok {
			if searchfilter.AnalysisMatchesMovePattern(f.MovePatternFilter, ana) {
				addPosition(position)
			}
		} else if f.MirrorFilter {
			mirrored := position.Mirror()
			ok2, err := matchesGoFilters(mirrored)
			if err != nil {
				return nil, err
			}
			if ok2 {
				if searchfilter.AnalysisMatchesMovePattern(f.MovePatternFilter, ana) {
					addPosition(mirrored)
				}
			}
		}
	}

	return positions, nil

}

// rateFilterCheck is one of the six win/gammon/backgammon-rate search
// filters, folded into a table (B.15, #183): find used to carry each as its
// own ~15-line copy — parse the filter, read the rate from the cube
// analysis or, failing that, the first checker move, compare — differing
// only in which domain.SearchFilters field it read, the token
// AnalysisMatchesFloatFilter names in a parse error, and which two
// PositionAnalysis fields hold the rate.
type rateFilterCheck struct {
	filter  string
	token   string
	extract func(*domain.PositionAnalysis) (float64, bool)
}

// rateFilterChecks builds the six checks active for f. All six are always
// present; an empty filter field simply passes every row in
// matchesRateFilters, exactly as an absent `if f.XRateFilter != ""` used to.
func rateFilterChecks(f domain.SearchFilters) [6]rateFilterCheck {
	return [6]rateFilterCheck{
		{f.WinRateFilter, "w", func(ana *domain.PositionAnalysis) (float64, bool) {
			if ana.DoublingCubeAnalysis != nil {
				return ana.DoublingCubeAnalysis.PlayerWinChances, true
			}
			if ana.CheckerAnalysis != nil && len(ana.CheckerAnalysis.Moves) > 0 {
				return ana.CheckerAnalysis.Moves[0].PlayerWinChance, true
			}
			return 0, false
		}},
		{f.GammonRateFilter, "g", func(ana *domain.PositionAnalysis) (float64, bool) {
			if ana.DoublingCubeAnalysis != nil {
				return ana.DoublingCubeAnalysis.PlayerGammonChances, true
			}
			if ana.CheckerAnalysis != nil && len(ana.CheckerAnalysis.Moves) > 0 {
				return ana.CheckerAnalysis.Moves[0].PlayerGammonChance, true
			}
			return 0, false
		}},
		{f.BackgammonRateFilter, "b", func(ana *domain.PositionAnalysis) (float64, bool) {
			if ana.DoublingCubeAnalysis != nil {
				return ana.DoublingCubeAnalysis.PlayerBackgammonChances, true
			}
			if ana.CheckerAnalysis != nil && len(ana.CheckerAnalysis.Moves) > 0 {
				return ana.CheckerAnalysis.Moves[0].PlayerBackgammonChance, true
			}
			return 0, false
		}},
		{f.Player2WinRateFilter, "W", func(ana *domain.PositionAnalysis) (float64, bool) {
			if ana.DoublingCubeAnalysis != nil {
				return ana.DoublingCubeAnalysis.OpponentWinChances, true
			}
			if ana.CheckerAnalysis != nil && len(ana.CheckerAnalysis.Moves) > 0 {
				return ana.CheckerAnalysis.Moves[0].OpponentWinChance, true
			}
			return 0, false
		}},
		{f.Player2GammonRateFilter, "G", func(ana *domain.PositionAnalysis) (float64, bool) {
			if ana.DoublingCubeAnalysis != nil {
				return ana.DoublingCubeAnalysis.OpponentGammonChances, true
			}
			if ana.CheckerAnalysis != nil && len(ana.CheckerAnalysis.Moves) > 0 {
				return ana.CheckerAnalysis.Moves[0].OpponentGammonChance, true
			}
			return 0, false
		}},
		{f.Player2BackgammonRateFilter, "B", func(ana *domain.PositionAnalysis) (float64, bool) {
			if ana.DoublingCubeAnalysis != nil {
				return ana.DoublingCubeAnalysis.OpponentBackgammonChances, true
			}
			if ana.CheckerAnalysis != nil && len(ana.CheckerAnalysis.Moves) > 0 {
				return ana.CheckerAnalysis.Moves[0].OpponentBackgammonChance, true
			}
			return 0, false
		}},
	}
}

// matchesRateFilters reports whether ana satisfies every active check (an
// empty filter string is inactive and always passes); a nil ana fails any
// active check, and an analysis with neither a cube nor a checker-move rate
// to read fails it too — both match the six original blocks' behaviour.
func matchesRateFilters(checks [6]rateFilterCheck, ana *domain.PositionAnalysis) bool {
	for _, c := range checks {
		if c.filter == "" {
			continue
		}
		if ana == nil {
			return false
		}
		v, ok := c.extract(ana)
		if !ok {
			return false
		}
		if !searchfilter.AnalysisMatchesFloatFilter(c.filter, c.token, v) {
			return false
		}
	}
	return true
}

func derefInt(p *int64) int {
	if p == nil {
		return 0
	}
	return int(*p)
}

func boolToInt(p *bool) int {
	if p != nil && *p {
		return 1
	}
	return 0
}

// appendClosedListClauses adds the filters whose value comes from a short
// closed vocabulary rather than being a number or a free string: whether a
// comment is there at all (`co`/`xco`), where it came from (`co:user`, #263)
// and the position's derived phase (`ph:race`, #264, ADR-0035).
//
// A method of its own because buildWhere sits at .golangci.yml's statement
// ceiling and every filter added to it pushes it over — and because these
// share a shape nothing else in the query has: read a ";"-separated list
// against a fixed vocabulary, drop what it does not recognise, emit an IN.
func (s *SearchStore) appendClosedListClauses(scope string, f domain.SearchFilters, where *strings.Builder, args *[]any) {
	// Comment presence: `co` (has one) / `xco` (has none). Asking for both is
	// contradictory rather than ambiguous; "none" wins and the search comes
	// back empty, which is the honest answer.
	if f.CommentFilter == "has" || f.CommentFilter == "none" {
		cTenant, cArgs := s.DB.TenantFilter("c", scope)
		not := ""
		if f.CommentFilter == "none" {
			not = "NOT "
		}
		where.WriteString(" AND " + not + "EXISTS (SELECT 1 FROM comment c" +
			" WHERE " + cTenant + " AND c.position_id = p.id AND COALESCE(c.text, '') <> '')")
		*args = append(*args, cArgs...)
	}

	// Comment provenance. A separate EXISTS from the presence filter above:
	// the two are independent AND clauses, so `xco co:user` yields nothing
	// rather than being rejected as contradictory — the same treatment
	// `xco t"blot"` already gets.
	if origins := domain.SplitFilterList(f.CommentOriginFilter); len(origins) > 0 {
		cTenant, cArgs := s.DB.TenantFilter("c", scope)
		where.WriteString(" AND EXISTS (SELECT 1 FROM comment c WHERE " + cTenant +
			" AND c.position_id = p.id AND COALESCE(c.text, '') <> '' AND c.origin IN (" +
			Placeholders(len(origins)) + "))")
		*args = append(*args, cArgs...)
		for _, o := range origins {
			*args = append(*args, string(domain.ParseCommentOrigin(o)))
		}
	}

	// Derived phase (ADR-0035): one indexed column, never reclassified at
	// query time. An unrecognised name is dropped rather than refused here —
	// the CLI and the command bar are where a typo is named; a filter whose
	// every value is unknown simply does not narrow.
	if phases := domain.SplitFilterList(f.GamePhaseFilter); len(phases) > 0 {
		var codes []any
		for _, name := range phases {
			if ph, ok := domain.ParseGamePhase(name); ok {
				codes = append(codes, int(ph))
			}
		}
		if len(codes) > 0 {
			where.WriteString(" AND p.game_phase IN (" + Placeholders(len(codes)) + ")")
			*args = append(*args, codes...)
		}
	}
}
