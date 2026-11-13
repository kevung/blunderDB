package sqlshared

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/reach"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/searchfilter"
)

// SearchStore implements storage.SearchStore. position is a domain table:
// the query is confined to the scope's tenant through Dialect.TenantFilter.
type SearchStore struct{ DB Execer }

var _ storage.SearchStore = (*SearchStore)(nil)

// Find streams the positions matching f within the scope's tenant, a port of
// the Database wrapper's LoadPositionsByFiltersCore: cheap predicates in SQL,
// the rest in Go on the narrowed set.
//
// opts.Limit/Offset bound the results: without a Go-side predicate they go
// into the SQL query; with one, the candidates are scanned a chunk at a time
// until the window is full (see scan).
func (s *SearchStore) Find(ctx context.Context, scope string, f domain.SearchFilters, opts storage.ListOpts) iter.Seq2[*domain.Position, error] {
	return func(yield func(*domain.Position, error) bool) {
		chunks, err := s.find(ctx, scope, f, opts)
		if err != nil {
			yield(nil, err)
			return
		}
		for _, positions := range chunks {
			for i := range positions {
				if !yield(&positions[i], nil) {
					return
				}
			}
		}
	}
}

// searchWhereClause is what buildWhere hands find: the WHERE clause text and
// its bound arguments, plus the state later phases need that buildWhere
// already computed while reading f.
type searchWhereClause struct {
	scope         string
	where         string
	args          []any
	needAnalysis  bool
	useSQLFilters bool
	bitboardTight bool
	// moveErrorInGo is set by a move-error search: the rows player 1 played
	// more than one way escape the SQL column (position.multi_played lets them
	// through) and are scored in Go.
	moveErrorInGo bool
	// effInclude is f.Filter with the points shared with ExcludeFilter
	// cleared, so "Except" wins over "At least" on those points.
	effInclude domain.Position
	// likeTarget is the position a ranked query ranks against, loaded once
	// here because the class it imposes on the WHERE clause is read off it —
	// its kind of decision, its regime, and the matches it was met in.
	likeTarget *domain.Position
	// structure is set by masks on the checkers: predicates no statistics describe.
	structure bool
	// after is the cursor a page resumes from, nil for the first page.
	after *searchCursor
	// keys, for a page read by cursor, receives the sort key of every id the window
	// reads, from the statement that reads the id: the next cursor anchors on the
	// key the row had when it was ordered, not on a second look at a row that may
	// have gone meanwhile.
	keys map[int64]any
}

// appendRowFlagClauses adds the filters on properties of the row, not of the
// board: provenance (individually imported) and the source-tool study mark.
// Mirroring a position cannot change them, so they stay in SQL even in mirror
// search, where every board filter falls back to the Go phase.
func (s *SearchStore) appendRowFlagClauses(f domain.SearchFilters, where *strings.Builder) {
	if f.IndividuallyImportedFilter {
		where.WriteString(" AND " + s.DB.Bool("p.individually_imported", true))
	}
	if f.FlaggedFilter {
		where.WriteString(" AND " + s.DB.Bool("p.flagged", true))
	}
}

// appendRateClauses adds the win, gammon and backgammon rate filters of both players.
func (s *SearchStore) appendRateClauses(ctx context.Context, scope string, f domain.SearchFilters, where *strings.Builder, argsp *[]any) error {
	args := *argsp
	defer func() { *argsp = args }()
	// Win/gammon rate as `p.id IN (SELECT position_id FROM analysis …)`,
	// not a clause on the LEFT JOIN: the join form drives the scan through
	// idx_analysis_win_gammon in rate order and forces a TEMP B-TREE sort
	// for the ORDER BY p.id. The IN-subquery keeps the rowid-order scan, the
	// tenant predicate leading the rate indexes on PostgreSQL.
	var winGammonWhere strings.Builder
	var winGammonArgs []any
	wMin, wMax, wHasMin, wHasMax := searchfilter.ParseFloatFilterExpr(f.WinRateFilter, "w")
	searchfilter.AppendIntRangeSQL("player1_win_rate", int(math.Round(wMin*100)), int(math.Round(wMax*100)), wHasMin, wHasMax, &winGammonWhere, &winGammonArgs)
	gMin, gMax, gHasMin, gHasMax := searchfilter.ParseFloatFilterExpr(f.GammonRateFilter, "g")
	searchfilter.AppendIntRangeSQL("player1_gammon_rate", int(math.Round(gMin*100)), int(math.Round(gMax*100)), gHasMin, gHasMax, &winGammonWhere, &winGammonArgs)
	if winGammonWhere.Len() > 0 {
		aTenant, aArgs := s.DB.TenantFilter("", scope)
		sub := "SELECT position_id FROM analysis WHERE " + aTenant + winGammonWhere.String()
		subArgs := append(append([]any{}, aArgs...), winGammonArgs...)
		if err := s.narrowToIDs(ctx, sub, subArgs, where, &args, s.commonAnalysisClause(aTenant+winGammonWhere.String(), subArgs, where, &args)); err != nil {
			return err
		}
	}
	bMin, bMax, bHasMin, bHasMax := searchfilter.ParseFloatFilterExpr(f.BackgammonRateFilter, "b")
	searchfilter.AppendIntRangeSQL("a.player1_backgammon_rate", int(math.Round(bMin*100)), int(math.Round(bMax*100)), bHasMin, bHasMax, where, &args)
	// The player-2 rates take the same IN-subquery: through the LEFT JOIN the
	// scan went by rate order, then looked up and sorted every candidate.
	var winGammon2Where strings.Builder
	var winGammon2Args []any
	WMin, WMax, WHasMin, WHasMax := searchfilter.ParseFloatFilterExpr(f.Player2WinRateFilter, "W")
	searchfilter.AppendIntRangeSQL("player2_win_rate", int(math.Round(WMin*100)), int(math.Round(WMax*100)), WHasMin, WHasMax, &winGammon2Where, &winGammon2Args)
	GMin, GMax, GHasMin, GHasMax := searchfilter.ParseFloatFilterExpr(f.Player2GammonRateFilter, "G")
	searchfilter.AppendIntRangeSQL("player2_gammon_rate", int(math.Round(GMin*100)), int(math.Round(GMax*100)), GHasMin, GHasMax, &winGammon2Where, &winGammon2Args)
	if winGammon2Where.Len() > 0 {
		aTenant, aArgs := s.DB.TenantFilter("", scope)
		sub := "SELECT position_id FROM analysis WHERE " + aTenant + winGammon2Where.String()
		subArgs := append(append([]any{}, aArgs...), winGammon2Args...)
		if err := s.narrowToIDs(ctx, sub, subArgs, where, &args, s.commonAnalysisClause(aTenant+winGammon2Where.String(), subArgs, where, &args)); err != nil {
			return err
		}
	}
	BMin, BMax, BHasMin, BHasMax := searchfilter.ParseFloatFilterExpr(f.Player2BackgammonRateFilter, "B")
	searchfilter.AppendIntRangeSQL("a.player2_backgammon_rate", int(math.Round(BMin*100)), int(math.Round(BMax*100)), BHasMin, BHasMax, where, &args)
	return nil
}

// commonAnalysisClause writes a common condition on the analysis. SQLite materialises an IN
// subquery whole, a table read per match, where a check per position of the walk stops at the
// window; PostgreSQL hashes it from its index (nil: the plain subquery).
func (s *SearchStore) commonAnalysisClause(cond string, condArgs []any, where *strings.Builder, args *[]any) func() {
	if s.DB.Name() != "sqlite" {
		return nil
	}
	return func() {
		where.WriteString(" AND EXISTS (SELECT 1 FROM analysis WHERE position_id = p.id AND " + cond + ")")
		*args = append(*args, condArgs...)
	}
}

// readIDList reads a column of ids as the text of a literal list, up to limit if positive.
// Written as literals, parsed integers only: no placeholder limit of the database applies.
func (s *SearchStore) readIDList(ctx context.Context, query string, queryArgs []any, limit int) ([]string, error) {
	if limit > 0 {
		limitClause, limitArgs := s.DB.LimitOffset(limit, 0)
		query += limitClause
		queryArgs = append(append([]any{}, queryArgs...), limitArgs...)
	}
	rows, err := s.DB.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, errf(s.DB, "search id list", err)
	}
	defer rows.Close()
	ids := make([]string, 0, 64)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, errf(s.DB, "search id list scan", err)
		}
		ids = append(ids, strconv.FormatInt(id, 10))
	}
	if err := rows.Err(); err != nil {
		return nil, errf(s.DB, "search id list rows", err)
	}
	return ids, nil
}

// placedCheckers counts the checkers a board places for each side, bar included.
func placedCheckers(b domain.Board) (black, white int) {
	for _, p := range b.Points {
		switch {
		case p.Checkers <= 0 || p.Color < 0:
		case p.Color == domain.Black:
			black += p.Checkers
		default:
			white += p.Checkers
		}
	}
	return black, white
}

// blunderFloor is the predicate of idx_analysis_blunder, in thousandths of a point.
// blunderListLimit: a list costs about a millisecond of planning per thousand ids.
const (
	blunderFloor     = 100
	blunderListLimit = 50000
)

// idLists gathers the literal lists of one search: two lists in one statement make the
// planner scan the table against both, seconds where their intersection, read here,
// is a few rows by primary key.
type idLists struct {
	seen bool
	ids  []string
}

type idListsKey struct{}

func (l *idLists) intersect(ids []string) {
	if !l.seen {
		l.seen, l.ids = true, ids
		return
	}
	in := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		in[id] = struct{}{}
	}
	kept := l.ids[:0]
	for _, id := range l.ids {
		if _, ok := in[id]; ok {
			kept = append(kept, id)
		}
	}
	l.ids = kept
}

func (l *idLists) write(s *SearchStore, where *strings.Builder, args *[]any) {
	switch {
	case !l.seen:
	case len(l.ids) == 0:
		where.WriteString(" AND 0=1")
	default:
		cond, condArgs := s.DB.IDsIn("p.id", l.ids)
		where.WriteString(" AND " + cond)
		*args = append(*args, condArgs...)
	}
}

// occupancyProbeLimit: a whole occupancy shared by more positions is no use from its index,
// each row a heap read before the sort, a millisecond when cold, where the walk in id order
// reads the masks from an index.
const occupancyProbeLimit = 500

// rareOccupancy tells whether an occupancy is worth asking its index.
func (s *SearchStore) rareOccupancy(ctx context.Context, scope, col string, v int64) (bool, error) {
	own, ownArgs := s.DB.TenantFilter("", scope)
	found, err := s.readIDList(ctx, "SELECT 1 FROM position WHERE "+own+" AND "+col+" = ?", append(append([]any{}, ownArgs...), v), occupancyProbeLimit+1)
	return len(found) <= occupancyProbeLimit, err
}

// narrowLimit is how many matches a filter on another table may name by literal list.
// Atomic so a test can move it (SetNarrowLimit) while other searches run.
var narrowLimit atomic.Int64

func init() { narrowLimit.Store(5000) }

// SetNarrowLimit sets narrowLimit and returns a func restoring the previous one; below
// zero, no condition is read first. For tests only: both ways of writing a condition
// are exercised on every backend.
func SetNarrowLimit(n int) (restore func()) {
	old := narrowLimit.Swap(int64(n))
	return func() { narrowLimit.Store(old) }
}

// narrowToIDs writes the condition "p.id comes out of sub". A rare condition is the
// planner's trap: its estimate at the tail of a histogram is off by hundreds, so under
// a LIMIT it walks the tenant in id order hoping to meet the matches soon. Up to
// narrowLimit matches are therefore read first, from the index of the condition, and
// written as a list. Beyond it the matches are common, the walk does meet them soon,
// and common writes the clause (nil: sub as a subquery). A condition is read twice in
// two statements; a position arriving between them only shifts the moment of its
// appearance.
func (s *SearchStore) narrowToIDs(ctx context.Context, sub string, subArgs []any, where *strings.Builder, args *[]any, common func()) error {
	return s.narrowWithin(ctx, int(narrowLimit.Load()), sub, subArgs, where, args, common)
}

func (s *SearchStore) narrowWithin(ctx context.Context, limit int, sub string, subArgs []any, where *strings.Builder, args *[]any, common func()) error {
	if narrowLimit.Load() < 0 {
		limit = -1
	}
	writeCommon := func() {
		if common != nil {
			common()
			return
		}
		where.WriteString(" AND p.id IN (" + sub + ")")
		*args = append(*args, subArgs...)
	}
	if limit < 0 {
		writeCommon()
		return nil
	}
	ids, err := s.readIDList(ctx, `SELECT * FROM (`+sub+`) AS probe`, subArgs, limit+1)
	if err != nil {
		return err
	}
	switch {
	case len(ids) > limit:
		writeCommon()
	case len(ids) == 0:
		where.WriteString(" AND 0=1")
	default:
		if lists, ok := ctx.Value(idListsKey{}).(*idLists); ok {
			lists.intersect(ids)
			return nil
		}
		cond, condArgs := s.DB.IDsIn("p.id", ids)
		where.WriteString(" AND " + cond)
		*args = append(*args, condArgs...)
	}
	return nil
}

// appendMoveErrorClause adds the move-error filter and reports whether Go
// must settle it afterwards.
func (s *SearchStore) appendMoveErrorClause(ctx context.Context, scope string, f domain.SearchFilters, where *strings.Builder, argsp *[]any) (bool, error) {
	// The denormalised error column scores ONE play (the first of
	// PlayedMoves, see AnalysisStore.Save). A position played several ways
	// is let through by its multi_played indicator and settled in Go by
	// matchesMoveErrorFilter on the largest error: the column can only
	// under-state it. The indicator is kept by triggers on move, so the
	// clause reads two columns of the row and no move at all.
	if f.MoveErrorFilter == "" {
		return false, nil
	}
	eMin, eMax, eHasMin, eHasMax := searchfilter.ParseFloatFilterExpr(f.MoveErrorFilter, "E")
	eqMin := int(math.Round(eMin))
	eqMax := int(math.Round(eMax))
	var cond string
	var bound []any
	switch {
	case eHasMin && eHasMax:
		cond, bound = " BETWEEN ? AND ?", []any{eqMin, eqMax}
	case eHasMin:
		cond, bound = " >= ?", []any{eqMin}
	case eHasMax:
		cond, bound = " <= ?", []any{eqMax}
	default:
		return false, nil
	}
	// Rare, the error is read from its two indexes, positionErrExpr split in its two
	// branches. Common, the walk in id order meets its rows soon, on the joined analysis.
	tenant, tenantArgs := s.DB.TenantFilter("an", scope)
	own, ownArgs := s.DB.TenantFilter("", scope)
	recheck := func() {
		where.WriteString(" AND (" + positionErrExpr + cond + " OR " + s.DB.Bool("p.multi_played", true) + ")")
		*argsp = append(*argsp, bound...)
	}
	// From a blunder up, idx_analysis_blunder holds every candidate in id order, whatever
	// the decision of the position: the threshold is written out for the planner to see
	// that the partial index applies.
	if eHasMin && eqMin >= blunderFloor {
		n := strconv.Itoa(eqMin)
		sub := "SELECT position_id FROM analysis WHERE " + own + " AND (best_move_equity_error >= " + n + " OR cube_error >= " + n + ")" +
			" UNION ALL SELECT id FROM position WHERE " + own + " AND " + s.DB.Bool("multi_played", true)
		// As a subquery the planner, estimating two hundred rows out of the union, sorts
		// them all, a heap read each; a list it walks in id order and stops at the window.
		// Too long for a list, the walk tests each id against the hashed index, without
		// reading a row the filter drops; the few multi-played positions go by list.
		multi, err := s.readIDList(ctx, "SELECT id FROM position WHERE "+own+" AND "+s.DB.Bool("multi_played", true), ownArgs, blunderListLimit)
		if err != nil {
			return true, err
		}
		common := func() {
			where.WriteString(" AND (p.id IN (SELECT position_id FROM analysis WHERE " + own + " AND (best_move_equity_error >= " + n + " OR cube_error >= " + n + "))")
			*argsp = append(*argsp, ownArgs...)
			if len(multi) > 0 {
				cond, condArgs := s.DB.IDsIn("p.id", multi)
				where.WriteString(" OR " + cond)
				*argsp = append(*argsp, condArgs...)
			}
			where.WriteString(")")
		}
		if err := s.narrowWithin(ctx, blunderListLimit, sub, append(append([]any{}, ownArgs...), ownArgs...), where, argsp, common); err != nil {
			return true, err
		}
		recheck()
		return true, nil
	}
	branch := func(decision, col string) string {
		return `SELECT an.position_id FROM analysis an JOIN position q ON q.id = an.position_id` + andSameTenant(s.DB, "q", "an") + ` WHERE ` + tenant + ` AND ` + decision + ` AND an.` + col + cond
	}
	// The probe keeps narrowLimit+1 ids of the union; each branch stops there by itself.
	// A LIMIT above the union alone leaves the planner free to hash-join a branch,
	// reading every candidate of the tenant before keeping a few thousand of them.
	stop := " LIMIT " + strconv.FormatInt(max(narrowLimit.Load(), 0)+1, 10) + ") AS "
	sub := "SELECT * FROM (" + branch("q.decision_type = 1", "cube_error") + stop + "cube UNION ALL " +
		"SELECT * FROM (" + branch("COALESCE(q.decision_type, 0) <> 1", "best_move_equity_error") + stop + "checker UNION ALL " +
		"SELECT * FROM (SELECT id FROM position WHERE " + own + " AND " + s.DB.Bool("multi_played", true) + stop + "multi"
	subArgs := append(append(append(append(append([]any{}, tenantArgs...), bound...), tenantArgs...), bound...), ownArgs...)
	err := s.narrowToIDs(ctx, sub, subArgs, where, argsp, recheck)
	return true, err
}

// buildWhere is whereClause for every reader but Rank. Each of them refuses
// `like` rather than ignoring it: the relation is decided by Rank alone, and
// a reader that skipped the token would select the whole library.
func (s *SearchStore) buildWhere(ctx context.Context, scope string, f domain.SearchFilters) (searchWhereClause, error) {
	if f.LikeFilter {
		return searchWhereClause{}, storage.ErrLikeUnranked
	}
	return s.whereClause(ctx, scope, f)
}

// whereClause translates f into the WHERE clause of the search query: cheap
// predicates that can be pushed to SQL become clause text and bound
// arguments; the rest are left to applyGoFilters (matchesGoFilters below),
// which is what needAnalysis/useSQLFilters/bitboardTight/moveErrorInGo/
// effInclude in the returned searchWhereClause are for.
func (s *SearchStore) whereClause(ctx context.Context, scope string, f domain.SearchFilters) (searchWhereClause, error) {
	lists := &idLists{}
	ctx = context.WithValue(ctx, idListsKey{}, lists)
	useSQLFilters := !f.MirrorFilter
	moveErrorInGo := false

	// Decode the compressed analysis blob per row only when a Go-side filter
	// reads it: move pattern, mirror re-checks, date, equity, margin. The rate, cube
	// and move-error filters run on denormalised SQL columns. MoveErrorFilter
	// is deliberately NOT a trigger: its Go re-check runs on the mirror path
	// (already covered) or on the multi-played positions, loaded one by one
	// after the scan; triggering here decodes every row for nothing.
	needAnalysis := f.MovePatternFilter != "" || f.MirrorFilter ||
		f.DateFilter != "" || f.EquityFilter != "" || f.MarginFilter != ""

	// On points shared with the exclusion structure, "Except" wins over "At least":
	// clear those points from the include filter so the two are not contradictory.
	effInclude := domain.EffectiveIncludeFilter(f.Filter, f.ExcludeFilter)

	// The tenant predicate comes first, and its arguments first, so the
	// placeholders line up once the PostgreSQL adapter has rebound them.
	var where strings.Builder
	tenant, args := s.DB.TenantFilter("p", scope)
	where.WriteString(tenant)

	s.appendRowFlagClauses(f, &where)

	// Comment presence, origin, tags and phase are row properties too, so they
	// stay in SQL even in mirror search (and in SQL for cost: a presence filter
	// is often the only thing narrowing the scan).
	s.appendClosedListClauses(scope, f, &where, &args)
	appendAwayClauses(f, &where, &args)
	if err := s.appendCommentAuthorClause(ctx, scope, f, &where, &args); err != nil {
		return searchWhereClause{}, err
	}

	if err := s.appendIdentityClauses(ctx, scope, f, &where, &args); err != nil {
		return searchWhereClause{}, err
	}
	var prMissing *prOfMissing
	if f.PlayerPRFilter != "" {
		var err error
		if prMissing, err = s.prOfMissingMatches(ctx, scope, f.PlayerPRFilter); err != nil {
			return searchWhereClause{}, err
		}
	}
	// An aliased name stands for the whole person: every spelling the alias
	// table gives them is searched.
	var aliases storage.AliasMap
	if f.PlayerFilter != "" || f.OpponentFilter != "" {
		var err error
		if aliases, err = aliasMap(ctx, s.DB, scope, storage.AliasPlayer); err != nil {
			return searchWhereClause{}, err
		}
	}
	if err := s.appendCorpusClauses(ctx, scope, f, prMissing, aliases, &where, &args); err != nil {
		return searchWhereClause{}, err
	}

	// A ranked query reads its target here, once: the stored position it
	// names, else the drawn board (ADR-0112), read as a POSITION, not a
	// pattern — checkers not placed count as borne off. Nothing narrows the
	// candidates: the relation holds across the whole library, whatever the
	// decision, the score or the match.
	var likeTarget *domain.Position
	if f.LikeFilter {
		if f.LikeTargetID > 0 {
			loaded, err := LoadTargetPosition(ctx, s.DB, scope, f.LikeTargetID)
			if err != nil {
				return searchWhereClause{}, err
			}
			likeTarget = loaded
		} else {
			board := f.LikeTargetBoard
			likeTarget = &board
		}
	}

	var bitboardTight, structure bool
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

		// How many times the position was MET: the move rows that reach it. A
		// correlated subquery rather than a denormalised column that every
		// import would have to keep true; idx_move_position answers it.
		if f.EncounterFilter != "" {
			nMin, nMax, nHasMin, nHasMax := searchfilter.ParseIntFilterExpr(f.EncounterFilter, "n")
			countSQL, countArgs := s.encounterCountSQL(scope, f, aliases)
			// The placeholders of the count come before the bounds' in the text.
			var rng strings.Builder
			var rngArgs []any
			searchfilter.AppendIntRangeSQL(countSQL, nMin, nMax, nHasMin, nHasMax, &rng, &rngArgs)
			if rng.Len() > 0 {
				where.WriteString(rng.String())
				args = append(args, countArgs...)
				args = append(args, rngArgs...)
			}
		}

		if err := s.appendRateClauses(ctx, scope, f, &where, &args); err != nil {
			return searchWhereClause{}, err
		}
		moveError, err := s.appendMoveErrorClause(ctx, scope, f, &where, &args)
		if err != nil {
			return searchWhereClause{}, err
		}
		moveErrorInGo = moveError

		if f.DecisionTimeFilter != "" {
			if cond, condArgs := decisionTimeSQL(s.DB, scope, f.DecisionTimeFilter); cond != "" {
				where.WriteString(" AND " + cond)
				args = append(args, condArgs...)
			}
		}

		if searchfilter.HasBoardFilter(effInclude.Board) {
			occ1Req, pt1Req, occ2Req, pt2Req, tight := engine.CheckerStructureMasks(effInclude)
			bitboardTight, structure = tight, true
			where.WriteString(" AND (p.occupancy_1 & ?) = ? AND (p.point_mask_1 & ?) = ?")
			where.WriteString(" AND (p.occupancy_2 & ?) = ? AND (p.point_mask_2 & ?) = ?")
			args = append(args,
				int64(occ1Req), int64(occ1Req), int64(pt1Req), int64(pt1Req),
				int64(occ2Req), int64(occ2Req), int64(pt2Req), int64(pt2Req))
			// A side whose fifteen checkers are all placed has nowhere else to be: its
			// occupancy is implied whole, and an index answers what masks cannot.
			black, white := placedCheckers(effInclude.Board)
			pip1, pip2 := engine.PipCounts(effInclude.Board)
			for _, side := range []struct {
				placed int
				col    string
				occ    int64
				pipCol string
				pip    int
			}{{black, "occupancy_1", int64(occ1Req), "p.pip_1", pip1}, {white, "occupancy_2", int64(occ2Req), "p.pip_2", pip2}} {
				if side.placed != domain.CheckersPerPlayer {
					continue
				}
				// Masks cannot count beyond two checkers: the pip count, from the same index
				// as the masks, spares the Go phase nearly all its candidates.
				where.WriteString(" AND (" + side.pipCol + " = ? OR " + side.pipCol + " IS NULL)")
				args = append(args, side.pip)
				rare := black == white // both sides whole: the pair is the index's own key
				if !rare {
					var err error
					if rare, err = s.rareOccupancy(ctx, scope, side.col, side.occ); err != nil {
						return searchWhereClause{}, err
					}
				}
				if rare {
					where.WriteString(" AND p." + side.col + " = ?")
					args = append(args, side.occ)
				}
			}
		}

		// Exclusion structure ("Sauf"): drop positions that contain ANY of the
		// excluded elements (OR semantics across points). Keep a position only when
		// none of its points match an excluded element. Template points with >2
		// checkers are not representable as bitmasks and are left to the Go-side
		// check (Position.ContainsAnyCheckerOf) below.
		if searchfilter.HasBoardFilter(f.ExcludeFilter.Board) {
			eSingle1, eMade1, eSingle2, eMade2 := engine.ExclusionMasks(f.ExcludeFilter)
			structure = true
			where.WriteString(" AND (p.occupancy_1 & ?) = 0 AND (p.point_mask_1 & ?) = 0")
			where.WriteString(" AND (p.occupancy_2 & ?) = 0 AND (p.point_mask_2 & ?) = 0")
			args = append(args,
				int64(eSingle1), int64(eMade1), int64(eSingle2), int64(eMade2))
		}
	}

	// A text or tag filter can only match a position that carries comment
	// text, and that is a few thousand rows however large the library: say so
	// in SQL, so the scan reads those positions instead of every one and
	// decodes none of the others. The Go predicate still decides the match.
	if f.SearchText != "" || len(domain.ParseTagFilter(f.TagFilter)) > 0 {
		cTenant, cArgs := s.DB.TenantFilter("", scope)
		where.WriteString(" AND p.id IN (SELECT position_id FROM comment WHERE " + cTenant + " AND text != '')")
		args = append(args, cArgs...)
	}

	lists.write(s, &where, &args)
	return searchWhereClause{
		scope:         scope,
		where:         where.String(),
		args:          args,
		needAnalysis:  needAnalysis,
		useSQLFilters: useSQLFilters,
		bitboardTight: bitboardTight,
		structure:     structure,
		moveErrorInGo: moveErrorInGo,
		effInclude:    effInclude,
		likeTarget:    likeTarget,
	}, nil
}

// Rank answers a query carrying the `like` token: the candidates Find would
// return that stand within LikeMaxDistance plays of the target, nearest
// first, each with its distance (ADR-0112).
//
// The distance is decided in Go, necessarily: it is a search over legal
// plays, and several filters are decided in applyGoFilters after the scan
// anyway, so the ranking must see all the survivors.
func (s *SearchStore) Rank(ctx context.Context, scope string, f domain.SearchFilters, opts storage.ListOpts) ([]storage.SimilarPosition, error) {
	if !f.LikeFilter {
		return nil, fmt.Errorf("Rank needs a query carrying the `like` token; use Find for an unordered search")
	}
	order := f.LikeMaxDistance
	if order == 0 {
		order = reach.DefaultOrder
	}
	if err := reach.CheckOrder(order); err != nil {
		return nil, fmt.Errorf("%w: %w", storage.ErrInvalid, err)
	}
	wc, err := s.whereClause(ctx, scope, f)
	if err != nil {
		return nil, err
	}
	target := reach.NewTarget(reach.Of(wc.likeTarget), order)
	var out []storage.SimilarPosition
	// The candidates are read chunk by chunk and only the neighbours kept:
	// a library holds far more positions than stand a few plays away.
	err = s.scan(ctx, f, wc, storage.ListOpts{}, func(chunk []domain.Position) bool {
		for i := range chunk {
			if wc.likeTarget.ID != 0 && chunk[i].ID == wc.likeTarget.ID {
				continue
			}
			if d, ok := target.Distance(reach.Of(&chunk[i])); ok {
				out = append(out, storage.SimilarPosition{Position: chunk[i], Distance: d})
			}
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	// Ties break on the id, so the same library answers the same question the
	// same way twice — a ranking whose order wobbles between two runs cannot
	// be paged, and cannot be compared against a recorded result.
	sort.Slice(out, func(i, j int) bool {
		if out[i].Distance != out[j].Distance {
			return out[i].Distance < out[j].Distance
		}
		return out[i].Position.ID < out[j].Position.ID
	})

	if opts.Offset > 0 {
		if opts.Offset >= len(out) {
			return nil, nil
		}
		out = out[opts.Offset:]
	}
	if opts.Limit > 0 && len(out) > opts.Limit {
		out = out[:opts.Limit]
	}
	return out, nil
}

// find returns the survivors as the scan read them, chunk after chunk, so
// that no position is copied into one slice holding them all.
func (s *SearchStore) find(ctx context.Context, scope string, f domain.SearchFilters, opts storage.ListOpts) ([][]domain.Position, error) {
	wc, err := s.buildWhere(ctx, scope, f)
	if err != nil {
		return nil, err
	}
	var chunks [][]domain.Position
	err = s.scan(ctx, f, wc, opts, func(positions []domain.Position) bool {
		chunks = append(chunks, positions)
		return true
	})
	return chunks, err
}

// findWith is find once the WHERE clause is built: the scan, and the Go-side
// predicates that finish it, the survivors in one slice: a cursor page reads
// its own bounded window through it.
func (s *SearchStore) findWith(ctx context.Context, f domain.SearchFilters, wc searchWhereClause, opts storage.ListOpts) ([]domain.Position, error) {
	var chunks [][]domain.Position
	n := 0
	err := s.scan(ctx, f, wc, opts, func(positions []domain.Position) bool {
		chunks = append(chunks, positions)
		n += len(positions)
		return true
	})
	if err != nil || len(chunks) == 1 {
		return firstChunk(chunks), err
	}
	all := make([]domain.Position, 0, n)
	for _, c := range chunks {
		all = append(all, c...)
	}
	return all, nil
}

func firstChunk(chunks [][]domain.Position) []domain.Position {
	if len(chunks) == 0 {
		return nil
	}
	return chunks[0]
}

// searchChunk is how many SQL candidates one round of a Go-filtered scan
// holds in memory at most: the scan's footprint is bounded by it, not by the
// size of the library. The first round reads searchFirstChunk and each next
// one doubles, so a first page whose rows mostly survive does not decode
// thousands of blobs to fill a hundred lines. Atomic so a test can shrink it
// (SetSearchChunk) while other searches run.
var searchChunk atomic.Int64

func init() { searchChunk.Store(4096) }

// SetSearchChunk sets the largest chunk of a Go-filtered scan and returns a
// func restoring the previous one. For tests only: it lets a handful of
// positions span several chunks, so the resumption (after the last id, or by
// OFFSET) is exercised on every backend.
func SetSearchChunk(n int) (restore func()) {
	old := searchChunk.Swap(int64(n))
	return func() { searchChunk.Store(old) }
}

const searchFirstChunk = 256

// goPhase reports whether a predicate is left for applyGoFilters to decide
// on some row, i.e. whether the SQL match can differ from the result. When it
// cannot, LIMIT/OFFSET and COUNT go to SQL as they are, and a search for ids
// never reconstructs a position. It mirrors matchesGoFilters' guards; erring
// on true only costs speed.
func (wc searchWhereClause) goPhase(f domain.SearchFilters) bool {
	return !wc.useSQLFilters ||
		(wc.bitboardTight && searchfilter.HasBoardFilter(wc.effInclude.Board)) ||
		searchfilter.HasBoardFilter(f.ExcludeFilter.Board) ||
		wc.moveErrorInGo ||
		f.Player1CheckerInZoneFilter != "" || f.Player2CheckerInZoneFilter != "" ||
		f.Player1OutfieldBlotFilter != "" || f.Player2OutfieldBlotFilter != "" ||
		f.Player1JanBlotFilter != "" || f.Player2JanBlotFilter != "" ||
		f.SearchText != "" || f.TagFilter != "" ||
		f.DateFilter != "" || f.EquityFilter != "" || f.MarginFilter != "" ||
		f.MovePatternFilter != ""
}

// keysetOrder reports whether the sort is the id alone, so that a chunked
// scan can resume after the last id seen instead of skipping an OFFSET that
// grows with every chunk.
func keysetOrder(sort string) bool {
	return domain.SearchOrderByClause(sort) == "p.id"
}

// scan hands visit the survivors of the search, in order, within opts, one
// chunk at a time: the window is counted on the survivors, not on the SQL
// candidates, so a page is never short because the Go phase rejected rows of
// it. visit never sees an empty chunk, and owns each one it is handed; its
// returning false stops the scan.
//
// Without a Go phase, LIMIT/OFFSET go to SQL and the first page comes back
// as soon as SQL has it. With one, the candidates are read searchChunk at a
// time and the scan stops once the window is full.
func (s *SearchStore) scan(ctx context.Context, f domain.SearchFilters, wc searchWhereClause, opts storage.ListOpts, visit func([]domain.Position) bool) error {
	if !wc.goPhase(f) {
		positions, _, _, err := s.scanChunk(ctx, f, wc, "", nil, opts.Limit, opts.Offset)
		if err != nil || len(positions) == 0 {
			return err
		}
		visit(positions)
		return nil
	}

	keyset := keysetOrder(f.Sort)
	maxChunk := int(searchChunk.Load())
	chunk := min(searchFirstChunk, maxChunk)
	// Each candidate read is a heap read of the position and of its analysis: a page reads
	// little more than itself, and the chunks double when the Go phase rejects.
	if opts.Limit > 0 {
		chunk = min(chunk, opts.Offset+opts.Limit+1)
	}
	skip, taken := opts.Offset, 0
	var after int64
	resumed := false
	sqlOffset := 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		extra, extraArgs, offset := "", []any(nil), sqlOffset
		if keyset {
			offset = 0
			if resumed {
				extra, extraArgs = afterID, []any{after}
			}
		}
		positions, n, lastID, err := s.scanChunk(ctx, f, wc, extra, extraArgs, chunk, offset)
		if err != nil {
			return err
		}
		dropped := min(skip, len(positions))
		positions, skip = positions[dropped:], skip-dropped
		if opts.Limit > 0 && len(positions) > opts.Limit-taken {
			positions = positions[:opts.Limit-taken]
		}
		if len(positions) > 0 {
			if !visit(positions) {
				return nil
			}
			taken += len(positions)
			if opts.Limit > 0 && taken >= opts.Limit {
				return nil
			}
		}
		if n < chunk {
			return nil
		}
		after, resumed = lastID, true
		sqlOffset += n
		chunk = min(chunk*2, maxChunk)
	}
}

// windowIDs cuts a window on ids alone, position rows and analyses being read
// only for the rows kept, so that neither OFFSET nor the neighbouring tenants' rows
// in the analysis index are paid for.
//
// A sort key is read in two phases, because "key DESC NULLS LAST, p.id" has no index
// to follow: an index runs its NULLs at one end only, and a key on the analysis sits
// on the nullable side of the join. The rows with a key go first, the only ones an
// index can order, from the analysis when the key is there (a row with a key has an
// analysis, so the inner join loses none); those without follow in id order.
func (s *SearchStore) windowIDs(ctx context.Context, f domain.SearchFilters, wc searchWhereClause, extra string, extraArgs []any, limit, offset int) ([]int64, error) {
	key, desc, _ := sortKey(f.Sort)
	filter := wc.where + extra
	args := append(append([]any{}, wc.args...), extraArgs...)
	joins := ""
	if readsAnalysis.MatchString(filter) || strings.HasPrefix(key, "a.") {
		joins = " LEFT JOIN analysis a ON a.position_id = p.id" + andSameTenant(s.DB, "a", "p")
	}
	plain := "position p" + joins
	if key == "" {
		// Where the window starts: the walk of a probe starts there too.
		start, startArgs := extra, append([]any{}, extraArgs...)
		if wc.after != nil {
			cond, condArgs := wc.after.cond()
			filter, args = filter+cond, append(args, condArgs...)
			start, startArgs = start+cond, append(startArgs, condArgs...)
		}
		order := "p.id"
		if offset == 0 && strings.Count(start, afterID)*len(afterID) == len(start) {
			rows := int(idWalkProbe.Load())
			// Masks have no index to fall back on: the walk is given more rows.
			if !wc.structure {
				rows /= columnProbeShare
			}
			ids, tried, err := s.probeIDWalk(ctx, wc.scope, joins, start, startArgs, filter, args, limit, rows)
			if err != nil || len(ids) == limit {
				return ids, err
			}
			if tried && s.fewRowsExpected(ctx, plain, filter, args) {
				order = sortedIDOrder
			}
		}
		return s.readIDs(ctx, plain, filter, order, "", args, limit, offset, nil)
	}
	dir := " ASC"
	if desc {
		dir = " DESC"
	}
	keyed, keyedFilter, keyedArgs := plain, filter+" AND "+key+" IS NOT NULL", args
	if strings.HasPrefix(key, "a.") {
		tenant, tenantArgs := s.DB.TenantFilter("a", wc.scope)
		keyed = "analysis a JOIN position p ON p.id = a.position_id" + andSameTenant(s.DB, "p", "a")
		keyedFilter += " AND " + tenant
		keyedArgs = append(append([]any{}, args...), tenantArgs...)
	}
	keyedOrder := key + dir + ", p.id"
	// A cube decision plays no move: its played-move error is 0 throughout, so the order
	// is the id's. The error index would make it walk every checker error of the tenant first.
	if key == "a.best_move_equity_error" && f.DecisionTypeFilter && f.Filter.DecisionType == domain.CubeAction {
		keyedOrder = "p.id"
	}
	tailFilter, tailArgs := filter+" AND "+key+" IS NULL", args
	var ids []int64
	skipKeyed := false
	if wc.after != nil {
		var cond string
		var condArgs []any
		cond, condArgs, skipKeyed = wc.after.keyedCond()
		keyedFilter, keyedArgs = keyedFilter+cond, append(append([]any{}, keyedArgs...), condArgs...)
		cond, condArgs = wc.after.tailCond()
		tailFilter, tailArgs = tailFilter+cond, append(append([]any{}, tailArgs...), condArgs...)
	}
	if !skipKeyed {
		var err error
		ids, err = s.readIDs(ctx, keyed, keyedFilter, keyedOrder, key, keyedArgs, limit, offset, wc.keys)
		if err != nil || len(ids) == limit {
			return ids, err
		}
	}
	rest := 0
	if skipKeyed {
		rest = offset
	} else if len(ids) == 0 && offset > 0 {
		var n int
		if err := s.DB.QueryRow(ctx, `SELECT COUNT(*) FROM `+keyed+` WHERE `+keyedFilter, keyedArgs...).Scan(&n); err != nil {
			return nil, errf(s.DB, "search keyed count", err)
		}
		rest = max(offset-n, 0)
	}
	tail, err := s.readIDs(ctx, plain, tailFilter, "p.id", "", tailArgs, limit-len(ids), rest, nil)
	if err != nil {
		return nil, err
	}
	if wc.keys != nil {
		for _, id := range tail {
			wc.keys[id] = nil
		}
	}
	return append(ids, tail...), nil
}

// afterID resumes a walk in id order.
const afterID = " AND p.id > ?"

// idWalkProber is implemented by an Execer whose planner chooses between the
// walk in id order and a pass over the tenant on an estimate of the rows a
// filter keeps: an estimate it cannot make for masks, and makes wrong for
// columns that go together.
type idWalkProber interface {
	ProbesIDWalk() bool
	// EstimateRows is how many rows the planner expects of a query.
	EstimateRows(ctx context.Context, query string, args ...any) (float64, error)
}

// columnProbeShare is the part of idWalkProbe a filter without masks walks:
// when it keeps few rows an index finds them, and a long walk is time lost.
const columnProbeShare = 4

// sortedIDOrder is the id order, written so that no index gives it: the rows
// are found by the filter's own indexes, then sorted.
const sortedIDOrder = "p.id + 0"

// sortedReadMax is how many rows a read by the filter's indexes may expect
// and still be asked to sort them: about a second of fetching, at worst.
// Atomic so a test can move it (SetSortedReadMax) while other searches run.
var sortedReadMax atomic.Int64

func init() { sortedReadMax.Store(500000) }

// SetSortedReadMax sets the rows a sorted read may expect, -1 for never
// sorting, and returns a function that restores the previous value.
func SetSortedReadMax(n int) (restore func()) {
	old := sortedReadMax.Swap(int64(n))
	return func() { sortedReadMax.Store(old) }
}

// fewRowsExpected tells, after a short probe, whether the window is better read
// by the filter's indexes and sorted than walked in id order. The probe showed
// the walk does not fill the window where it starts; the planner, left to
// choose, still bets on it when two columns it believes independent are rarely
// true together, and walks the whole tenant. Its estimate of the rows kept,
// too high in that case, is what bounds the other read: under sortedReadMax
// the walk is forbidden, above it the planner chooses as before — a filter
// that keeps much of the tenant further on in the ids is cheaper walked than
// fetched whole.
func (s *SearchStore) fewRowsExpected(ctx context.Context, from, filter string, args []any) bool {
	p, ok := s.DB.(idWalkProber)
	if !ok {
		return false
	}
	rows, err := p.EstimateRows(ctx, `SELECT p.id FROM `+from+` WHERE `+filter, args...)
	return err == nil && rows <= float64(sortedReadMax.Load())
}

// idWalkProbe is how many rows of the tenant a probe walks, in id order.
// Atomic so a test can move it (SetIDWalkProbe) while other searches run.
var idWalkProbe atomic.Int64

func init() { idWalkProbe.Store(20000) }

// SetIDWalkProbe sets how many rows a probe walks, 0 for no probe, and returns
// a function that restores the previous value.
func SetIDWalkProbe(n int) (restore func()) {
	old := idWalkProbe.Swap(int64(n))
	return func() { idWalkProbe.Store(old) }
}

// probeIDWalk reads a window in id order from the next rows rows of the
// tenant, and says whether it tried. A full window is the answer: the rows
// are walked in the order asked, and the filter met limit of them. A short one
// says nothing of the rows further on, and is thrown away: what it cost is the
// rows walked, not a pass over the tenant.
//
// The walk is written, not left to the planner: a filter it believes keeps one
// row (masks) sends it to read and sort the whole tenant for a page that the
// first hundred rows fill.
func (s *SearchStore) probeIDWalk(ctx context.Context, scope, joins, start string, startArgs []any, filter string, args []any, limit, rows int) (ids []int64, tried bool, err error) {
	p, ok := s.DB.(idWalkProber)
	// A subquery of the filter would run here and again after a short window,
	// a list of ids be sent twice; and a position named is reached by its key.
	if !ok || !p.ProbesIDWalk() || limit <= 0 || limit >= rows || strings.Contains(filter, "SELECT") || strings.Contains(filter, "p.id IN (") {
		return nil, false, nil
	}
	tenant, walkArgs := s.DB.TenantFilter("p", scope)
	from := "(SELECT * FROM position p WHERE " + tenant + start + " ORDER BY p.id LIMIT " + strconv.Itoa(rows) + ") p" + joins
	walkArgs = append(append(walkArgs, startArgs...), args...)
	ids, err = s.readIDs(ctx, from, filter, "p.id", "", walkArgs, limit, 0, nil)
	return ids, err == nil, err
}

// readIDs reads the ids of a window; with keys, it also reads keyCol beside each id.
func (s *SearchStore) readIDs(ctx context.Context, from, filter, order, keyCol string, args []any, limit, offset int, keys map[int64]any) ([]int64, error) {
	limitClause, limitArgs := s.DB.LimitOffset(limit, offset)
	columns := "p.id"
	if keys != nil {
		columns += ", " + keyCol
	}
	rows, err := s.DB.Query(ctx, `SELECT `+columns+` FROM `+from+` WHERE `+filter+` ORDER BY `+order+limitClause,
		append(append([]any{}, args...), limitArgs...)...)
	if err != nil {
		return nil, errf(s.DB, "search ids query", err)
	}
	defer rows.Close()
	ids := make([]int64, 0, limit)
	for rows.Next() {
		var id int64
		var key any
		dest := []any{&id}
		if keys != nil {
			dest = append(dest, &key)
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, errf(s.DB, "search ids scan", err)
		}
		if keys != nil {
			keys[id] = key
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, errf(s.DB, "search ids rows", err)
	}
	return ids, nil
}

// readsAnalysis spots a reference to the joined analysis in a clause. A subquery
// with its own alias a only costs the shortcut, never a wrong answer.
var readsAnalysis = regexp.MustCompile(`\ba\.`)

// scanChunk runs one bounded query of the scan — the WHERE clause plus
// extra, ordered, within limit/offset — and passes its rows through the Go
// phase. It also returns how many rows SQL gave and the id of the last one,
// which is where a keyset scan resumes.
func (s *SearchStore) scanChunk(ctx context.Context, f domain.SearchFilters, wc searchWhereClause, extra string, extraArgs []any, limit, offset int) ([]domain.Position, int, int64, error) {
	// a.data, the compressed analysis blob (~600 bytes/row), is fetched only
	// when wc.needAnalysis says a Go-side filter reads it.
	analysisDataCol := "NULL"
	if wc.needAnalysis {
		analysisDataCol = "a.data"
	}

	limitClause, limitArgs := s.DB.LimitOffset(limit, offset)

	order := domain.SearchOrderByClause(f.Sort)
	columns := `p.id, p.state,
		p.decision_type, p.player_on_roll, p.dice_1, p.dice_2,
		p.cube_value, p.cube_owner, p.score_1, p.score_2,
		p.has_jacoby, p.has_beaver, p.max_cube, p.is_cube_response,
		p.individually_imported, p.flagged,
		a.id, ` + analysisDataCol + ` AS data`
	if limit > 0 && limit <= maxWindowInline {
		ids, err := s.windowIDs(ctx, f, wc, extra, extraArgs, limit, offset)
		if err != nil || len(ids) == 0 {
			return nil, 0, 0, err
		}
		tenant, tenantArgs := s.DB.TenantFilter("p", wc.scope)
		rows, err := s.DB.Query(ctx, `SELECT `+columns+` FROM position p
	LEFT JOIN analysis a ON a.position_id = p.id`+andSameTenant(s.DB, "a", "p")+`
	WHERE `+tenant+` AND p.id IN (`+Placeholders(len(ids))+`) ORDER BY `+order, append(tenantArgs, int64Args(ids)...)...)
		if err != nil {
			return nil, 0, 0, errf(s.DB, "search query", err)
		}
		defer rows.Close()
		return s.finishChunk(ctx, f, wc, rows, limit)
	}

	if wc.after != nil {
		cond, condArgs := wc.after.cond()
		extra, extraArgs = extra+cond, append(append([]any{}, extraArgs...), condArgs...)
	}
	query := `SELECT ` + columns + `
	FROM position p
	LEFT JOIN analysis a ON a.position_id = p.id` + andSameTenant(s.DB, "a", "p") + `
	WHERE ` + wc.where + extra + ` ORDER BY ` + order + limitClause

	args := append(append(append([]any{}, wc.args...), extraArgs...), limitArgs...)
	rows, err := s.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, 0, errf(s.DB, "search query", err)
	}
	defer rows.Close()
	return s.finishChunk(ctx, f, wc, rows, limit)
}

// maxWindowInline bounds the ids a window fetches by a literal list: above it, the
// statement would pass the placeholder limit of both databases.
const maxWindowInline = 30000

// finishChunk reads the rows of a chunk and passes them through the Go phase.
func (s *SearchStore) finishChunk(ctx context.Context, f domain.SearchFilters, wc searchWhereClause, rows Rows, limit int) ([]domain.Position, int, int64, error) {
	positions, scanned, err := s.scanRows(rows, wc.needAnalysis, limit)
	if err != nil {
		return nil, 0, 0, err
	}
	n := len(positions)
	var lastID int64
	if n > 0 {
		lastID = positions[n-1].ID
	}

	positions, err = s.applyGoFilters(ctx, f, wc, positions, scanned)
	return positions, n, lastID, err
}

// FindIDs returns the ids of the survivors of the search, within opts. With
// no Go phase it is a projection on p.id: no row is decoded, no position
// reconstructed.
func (s *SearchStore) FindIDs(ctx context.Context, scope string, f domain.SearchFilters, opts storage.ListOpts) ([]int64, error) {
	wc, err := s.buildWhere(ctx, scope, f)
	if err != nil {
		return nil, err
	}
	return s.findIDsWith(ctx, f, wc, opts)
}

// findIDsWith is FindIDs once the WHERE clause is built.
func (s *SearchStore) findIDsWith(ctx context.Context, f domain.SearchFilters, wc searchWhereClause, opts storage.ListOpts) ([]int64, error) {
	if wc.goPhase(f) {
		var ids []int64
		err := s.scan(ctx, f, wc, opts, func(positions []domain.Position) bool {
			for i := range positions {
				ids = append(ids, positions[i].ID)
			}
			return true
		})
		return ids, err
	}

	if opts.Limit > 0 {
		return s.windowIDs(ctx, f, wc, "", nil, opts.Limit, opts.Offset)
	}
	if wc.after != nil {
		return nil, fmt.Errorf("%w: a page read by cursor needs a limit", storage.ErrInvalid)
	}
	limitClause, limitArgs := s.DB.LimitOffset(opts.Limit, opts.Offset)
	query := `SELECT p.id FROM position p
	LEFT JOIN analysis a ON a.position_id = p.id` + andSameTenant(s.DB, "a", "p") + `
	WHERE ` + wc.where + ` ORDER BY ` + domain.SearchOrderByClause(f.Sort) + limitClause
	rows, err := s.DB.Query(ctx, query, append(append([]any{}, wc.args...), limitArgs...)...)
	if err != nil {
		return nil, errf(s.DB, "search ids query", err)
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, errf(s.DB, "search ids scan", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, errf(s.DB, "search ids rows", err)
	}
	return ids, nil
}

// Count returns how many positions the search finds: one COUNT in SQL when
// no Go phase is left, else a chunked scan that holds one chunk at a time.
func (s *SearchStore) Count(ctx context.Context, scope string, f domain.SearchFilters) (int, error) {
	wc, err := s.buildWhere(ctx, scope, f)
	if err != nil {
		return 0, err
	}
	if wc.goPhase(f) {
		n := 0
		err := s.scan(ctx, f, wc, storage.ListOpts{}, func(positions []domain.Position) bool {
			n += len(positions)
			return true
		})
		return n, err
	}
	return s.countWhere(ctx, wc, "", nil)
}

// countRows reads the rows of a search one by one up to limit; on failure n is what it reached.
func (s *SearchStore) countRows(ctx context.Context, wc searchWhereClause, limit int) (n int, err error) {
	limitClause, limitArgs := s.DB.LimitOffset(limit, 0)
	rows, err := s.DB.Query(ctx, `SELECT 1 FROM position p
	LEFT JOIN analysis a ON a.position_id = p.id`+andSameTenant(s.DB, "a", "p")+`
	WHERE `+wc.where+limitClause, append(append([]any{}, wc.args...), limitArgs...)...)
	if err != nil {
		return 0, errf(s.DB, "search bounded count", err)
	}
	defer rows.Close()
	for rows.Next() {
		n++
	}
	if err := rows.Err(); err != nil {
		return n, errf(s.DB, "search bounded count", err)
	}
	return n, nil
}

// CountUpTo counts at most max survivors, one row past it to tell a result of
// exactly max from a longer one. The Go phase stops on the same condition.
//
// A caller bounds the count in time with ctx: rows are counted as they arrive, and a
// count cut short reports what it reached as capped, "at least n". Cut before its first
// row, it fails with the context's error.
func (s *SearchStore) CountUpTo(ctx context.Context, scope string, f domain.SearchFilters, max int) (int, bool, error) {
	if max <= 0 {
		return 0, false, fmt.Errorf("search count: max %d: %w", max, storage.ErrInvalid)
	}
	wc, err := s.buildWhere(ctx, scope, f)
	if err != nil {
		return 0, false, err
	}
	n := 0
	count := func(wc searchWhereClause) error {
		left := max + 1 - n
		if !wc.goPhase(f) {
			m, err := s.countRows(ctx, wc, left)
			n += m
			return err
		}
		return s.scan(ctx, f, wc, storage.ListOpts{Limit: left}, func(positions []domain.Position) bool {
			n += len(positions)
			return true
		})
	}
	// A move error alone calls for Go only on the rows played more than one way: the others
	// are counted from their columns, without reading an analysis.
	bySQL := wc
	bySQL.moveErrorInGo = false
	if wc.moveErrorInGo && !bySQL.goPhase(f) {
		bySQL.where = wc.where + " AND " + s.DB.Bool("p.multi_played", false)
		if err = count(bySQL); err == nil && n <= max {
			wc.where += " AND " + s.DB.Bool("p.multi_played", true)
			err = count(wc)
		}
	} else {
		err = count(wc)
	}
	if err != nil {
		if ctx.Err() != nil && n > 0 {
			return n, true, nil
		}
		return 0, false, err
	}
	if n > max {
		return max, true, nil
	}
	return n, false, nil
}

func (s *SearchStore) countWhere(ctx context.Context, wc searchWhereClause, extra string, extraArgs []any) (int, error) {
	query := `SELECT COUNT(*) FROM position p
	LEFT JOIN analysis a ON a.position_id = p.id` + andSameTenant(s.DB, "a", "p") + `
	WHERE ` + wc.where + extra
	var n int
	if err := s.DB.QueryRow(ctx, query, append(append([]any{}, wc.args...), extraArgs...)...).Scan(&n); err != nil {
		return 0, errf(s.DB, "search count", err)
	}
	return n, nil
}

// IndexOf returns the rank of id among the survivors of the search, in their
// order; found is false when the search does not find id. With no Go phase it
// is COUNTs: the rows sorted before id, compared on the sort key then the id.
// Otherwise the scan walks to id.
func (s *SearchStore) IndexOf(ctx context.Context, scope string, f domain.SearchFilters, id int64) (int, bool, error) {
	wc, err := s.buildWhere(ctx, scope, f)
	if err != nil {
		return 0, false, err
	}
	if !wc.goPhase(f) {
		return s.indexOfInSQL(ctx, wc, f.Sort, id)
	}
	index, found := 0, false
	err = s.scan(ctx, f, wc, storage.ListOpts{}, func(positions []domain.Position) bool {
		for i := range positions {
			if positions[i].ID == id {
				found = true
				return false
			}
			index++
		}
		return true
	})
	if err != nil || !found {
		return 0, false, err
	}
	return index, true, nil
}

// sortKey returns the column a search sort orders on before its p.id
// tiebreak, and whether it runs descending, NULLs last; "" for the id order.
// It is read off domain.SearchOrderByClause so the rank cannot drift from the
// ORDER BY.
func sortKey(order string) (col string, desc, ok bool) {
	clause := domain.SearchOrderByClause(order)
	if clause == "p.id" {
		return "", false, true
	}
	if col, ok = strings.CutSuffix(clause, " DESC NULLS LAST, p.id"); ok {
		return col, true, true
	}
	col, ok = strings.CutSuffix(clause, " ASC NULLS LAST, p.id")
	return col, false, ok
}

// indexOfInSQL ranks id without reading a row: one query for id's sort key
// (none means the search does not find it), one COUNT of the rows ordered
// before it under "key DESC NULLS LAST, p.id".
func (s *SearchStore) indexOfInSQL(ctx context.Context, wc searchWhereClause, order string, id int64) (int, bool, error) {
	col, desc, ok := sortKey(order)
	if !ok {
		return 0, false, fmt.Errorf("search: no rank for sort %q", order)
	}
	if col == "" {
		hit, err := s.countWhere(ctx, wc, " AND p.id = ?", []any{id})
		if err != nil || hit == 0 {
			return 0, false, err
		}
		before, err := s.countWhere(ctx, wc, " AND p.id < ?", []any{id})
		return before, err == nil, err
	}

	query := `SELECT ` + col + ` FROM position p
	LEFT JOIN analysis a ON a.position_id = p.id` + andSameTenant(s.DB, "a", "p") + `
	WHERE ` + wc.where + ` AND p.id = ?`
	var key any
	switch err := s.DB.QueryRow(ctx, query, append(append([]any{}, wc.args...), id)...).Scan(&key); {
	case errors.Is(err, ErrNoRows):
		return 0, false, nil
	case err != nil:
		return 0, false, errf(s.DB, "search rank key", err)
	}
	// NULLs sort last: before a NULL key come every keyed row and the NULL
	// rows of a smaller id; before a key k, the keys ahead of it (larger when
	// descending, smaller when ascending) and k's smaller ids.
	ahead := " < ?"
	if desc {
		ahead = " > ?"
	}
	extra, extraArgs := " AND ("+col+" IS NOT NULL OR p.id < ?)", []any{id}
	if key != nil {
		extra, extraArgs = " AND ("+col+ahead+" OR ("+col+" = ? AND p.id < ?))", []any{key, key, id}
	}
	before, err := s.countWhere(ctx, wc, extra, extraArgs)
	return before, err == nil, err
}

// scannedRow is one row of buildWhere's query, decoded into the shape
// applyGoFilters works with. is_cube_response is read from its own column
// rather than the position blob, so it has to travel with the row to the
// filter phase.
type scannedRow struct {
	ana            *domain.PositionAnalysis
	isCubeResponse bool
}

// scanRows drains rows into a []scannedRow before any Go-side filtering
// starts, decoding each row's compressed analysis blob when needAnalysis
// says a later filter will read it.
//
// The positions come back in a slice of their own, parallel to the rows,
// sized to capHint (the LIMIT of the query, 0 when unbounded): a position is
// several hundred bytes, and applyGoFilters then keeps the survivors in that
// same slice instead of copying them into another one.
func (s *SearchStore) scanRows(rows Rows, needAnalysis bool, capHint int) ([]domain.Position, []scannedRow, error) {
	// Drain the cursor before filtering: the Go-side predicates open queries
	// of their own, and a cursor holds its pooled connection until exhausted.
	// On ":memory:" (one connection, sqlite.ConfigurePool) that deadlocks at
	// once; on a real pool, under enough concurrent searches. The buffer is
	// one chunk of the scan, never the whole library.
	positions := make([]domain.Position, 0, capHint)
	scanned := make([]scannedRow, 0, capHint)

	for rows.Next() {
		// Nullable columns scan into pointers, which both drivers leave nil on
		// NULL; the flags are INTEGER 0/1 on SQLite and BOOLEAN on PostgreSQL,
		// and both drivers convert either into a *bool.
		var posID int64
		var posState []byte // position.state is a BLOB (ADR-0071)
		var pDT, pPOR, pD1, pD2, pCV, pCO, pS1, pS2 *int64
		var pHJ, pHB, pICR, pII, pFlag *bool
		var pMC *int64
		var anaID *int64
		var anaData []byte

		if err := rows.Scan(
			&posID, &posState,
			&pDT, &pPOR, &pD1, &pD2, &pCV, &pCO, &pS1, &pS2, &pHJ, &pHB, &pMC, &pICR,
			&pII, &pFlag,
			&anaID, &anaData,
		); err != nil {
			return nil, nil, errf(s.DB, "search scan", err)
		}

		position := engine.ReconstructPosition(posID, string(posState),
			derefInt(pDT), derefInt(pPOR), derefInt(pD1), derefInt(pD2),
			derefInt(pCV), derefInt(pCO), derefInt(pS1), derefInt(pS2),
			boolToInt(pHJ), boolToInt(pHB))
		// Row properties rather than board identity, applied on top of the
		// reconstructed position as PositionStore.Load does (ADR-0001, ADR-0006).
		position.IndividuallyImported = pII != nil && *pII
		position.Flagged = pFlag != nil && *pFlag
		position.MaxCube = derefInt(pMC)

		var ana *domain.PositionAnalysis
		if needAnalysis && anaID != nil && len(anaData) > 0 {
			// a.data is stored compressed: decode as AnalysisStore.Load does, a
			// bare json.Unmarshal fails silently.
			if a, decErr := engine.DecodeAnalysisFromStorage(anaData); decErr == nil {
				ana = &a
			}
		}

		if len(positions) == cap(positions) {
			positions, scanned = growScanned(positions, scanned)
		}
		positions = append(positions, position)
		scanned = append(scanned, scannedRow{ana: ana, isCubeResponse: pICR != nil && *pICR})
	}
	if err := rows.Err(); err != nil {
		return nil, nil, errf(s.DB, "search rows", err)
	}
	// Hand the connection back before the predicates start querying.
	if err := rows.Close(); err != nil {
		return nil, nil, errf(s.DB, "search rows close", err)
	}

	return positions, scanned, nil
}

// growScanned doubles the room of an unbounded scan. append grows a large
// slice by a quarter at a time, which for a scan of the whole library
// allocates and copies about five times what it ends up holding; doubling
// keeps that to twice.
func growScanned(positions []domain.Position, scanned []scannedRow) ([]domain.Position, []scannedRow) {
	n := max(2*cap(positions), 256)
	p := make([]domain.Position, len(positions), n)
	copy(p, positions)
	r := make([]scannedRow, len(scanned), n)
	copy(r, scanned)
	return p, r
}

// applyGoFilters runs the Go-side predicates buildWhere could not push to
// SQL against each scanned row (and, for MirrorFilter, its mirror image
// too), preloading per-family batch queries first. The survivors are kept in
// positions itself, in order, and returned as its prefix.
func (s *SearchStore) applyGoFilters(ctx context.Context, f domain.SearchFilters, wc searchWhereClause, positions []domain.Position, scanned []scannedRow) ([]domain.Position, error) {
	var err error
	// Preload comment texts and player-1 plays in one batched query per
	// family instead of one per candidate, only when the filter is active and
	// after the cursor is drained.
	var commentTexts map[int64]string
	var player1MovesByID map[int64]player1Moves
	if f.SearchText != "" || f.TagFilter != "" || f.MoveErrorFilter != "" {
		ids := make([]int64, len(positions))
		for i := range positions {
			ids[i] = positions[i].ID
		}
		if f.SearchText != "" || f.TagFilter != "" {
			commentTexts, err = loadCommentTexts(ctx, s.DB, ids)
			if err != nil {
				return nil, errf(s.DB, "search preload comments", err)
			}
		}
		if f.MoveErrorFilter != "" {
			// With a time filter beside it, the error is that of the plays the
			// time keeps, not of any play the position was ever given.
			player1MovesByID, err = loadPlayer1MovesWhere(ctx, s.DB, ids, decisionTimeBounds(f.DecisionTimeFilter, "move"))
			if err != nil {
				return nil, errf(s.DB, "search preload player-1 moves", err)
			}
		}
	}

	kept := 0
	// Built once for the whole scan: the six checks only depend on f, not on
	// the row being tested.
	rates := rateFilterChecks(f)
	wantedTags := domain.ParseTagFilter(f.TagFilter)

	for i, row := range scanned {
		position, ana := positions[i], row.ana

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
				if !matchesAwayFilters(pos, f) {
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
			} else if f.MoveErrorFilter != "" && (player1MovesByID[pos.ID].plays > 1 || f.DecisionTimeFilter != "") {
				// A multi-played position is scored by its largest error, and one
				// beside a time filter by that of the plays the time keeps; its
				// blob may not have been fetched with the scan, so load it now.
				if ana == nil {
					ana = loadAnalysis(ctx, s.DB, wc.scope, pos.ID)
				}
				if !matchesMoveErrorFilterPreloaded(ana, player1MovesByID[pos.ID], f.MoveErrorFilter) {
					return false, nil
				}
			}

			if !matchesZoneAndBlotFilters(pos, f) {
				return false, nil
			}
			if !matchesCommentFilters(commentTexts[pos.ID], f.SearchText, wantedTags) {
				return false, nil
			}
			if f.DateFilter != "" && !searchfilter.MatchesDateFilter(ana, f.DateFilter) {
				return false, nil
			}
			if f.EquityFilter != "" && !searchfilter.AnalysisMatchesEquityFilter(f.EquityFilter, ana) {
				return false, nil
			}
			if !searchfilter.AnalysisMatchesMarginFilter(f.MarginFilter, ana, row.isCubeResponse) {
				return false, nil
			}
			return true, nil
		}

		addPosition := func(pos domain.Position) {
			if f.MoveErrorFilter != "" && pos.DecisionType == domain.CubeAction &&
				isPlayer1TakePassCubeActionPreloaded(player1MovesByID[pos.ID]) {
				pos = pos.Mirror()
			}
			// kept <= i: the row overwritten was already read into position.
			positions[kept] = pos
			kept++
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

	return positions[:kept], nil
}

// rateFilterCheck is one of the six win/gammon/backgammon-rate search
// filters: the filter field, the token named in a parse error, and the two
// PositionAnalysis fields (cube, else first checker move) holding the rate.
type rateFilterCheck struct {
	filter  string
	token   string
	extract func(*domain.PositionAnalysis) (float64, bool)
}

// rateFilterChecks builds the six checks for f; an empty filter field passes
// every row in matchesRateFilters.
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
// to read fails it too.
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

// appendIdentityClauses narrows the search to positions NAMED by something
// outside the board: the matches or tournaments they were met in, an explicit
// list of position ids. (The players, the match's own properties and the
// verdict's provenance are appendCorpusClauses.) They are one
// family — each turns a closed list of ids into a `p.id IN (…)` clause, and an
// empty list means "nothing matches" rather than "no narrowing", which is why
// each writes `0=1` instead of falling through.
//
// Provenance stays in SQL even in mirror search, like the row properties
// buildWhere writes just before calling this: mirroring a board cannot change
// which match a position was met in.
func (s *SearchStore) appendIdentityClauses(ctx context.Context, scope string, f domain.SearchFilters, where *strings.Builder, args *[]any) error {
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
					// A query failure must not silently read as "this
					// tournament has no positions".
					matchIDs, err := getMatchIDsForTournament(ctx, s.DB, tID)
					if err != nil {
						return err
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
				*args = append(*args, id)
			}
		} else {
			where.WriteString(" AND 0=1")
		}
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
				*args = append(*args, id)
			}
		} else {
			where.WriteString(" AND 0=1")
		}
	}

	// Written as literals, parsed integers only: a hidden list has no reason to stop
	// at the placeholder limit of the database.
	var excluded []string
	for _, idStr := range strings.Split(f.ExcludePositionIDs, ",") {
		if id, err := strconv.ParseInt(strings.TrimSpace(idStr), 10, 64); err == nil {
			excluded = append(excluded, strconv.FormatInt(id, 10))
		}
	}
	if len(excluded) > 0 {
		where.WriteString(" AND p.id NOT IN (" + strings.Join(excluded, ",") + ")")
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
				*args = append(*args, id)
			}
		} else {
			where.WriteString(" AND 0=1")
		}
	}

	return nil
}

// appendClosedListClauses adds the filters whose value comes from a short
// closed vocabulary rather than being a number or a free string: whether a
// comment is there at all (`co`/`xco`), where it came from (`co:user`)
// and the position's derived phase (`ph:race`, ADR-0035). They share one
// shape: a ";"-separated list against a fixed vocabulary, unknown values
// dropped, one IN. (Also keeps buildWhere under .golangci.yml's statement
// ceiling.)
// appendCommentAuthorClause keeps positions carrying a comment signed by the
// `au"…"` name: a third EXISTS, independent of presence and origin. The whole
// name, any case — Unicode case, which neither backend's LOWER gives alike
// (SQLite folds ASCII only), so the names are resolved in Go first and SQL
// sees an exact IN list. A "%" or "_" typed in a name is a character, never a
// wildcard.
func (s *SearchStore) appendCommentAuthorClause(ctx context.Context, scope string, f domain.SearchFilters, where *strings.Builder, args *[]any) error {
	name := strings.TrimSpace(searchfilter.QuotedName(f.CommentAuthorFilter, "au"))
	if name == "" {
		return nil
	}
	cTenant, cArgs := s.DB.TenantFilter("c", scope)
	rows, err := s.DB.Query(ctx, `SELECT DISTINCT c.author FROM comment c WHERE `+cTenant+
		` AND COALESCE(c.author, '') <> ''`, cArgs...)
	if err != nil {
		return errf(s.DB, "comment authors", err)
	}
	var authors []any
	for rows.Next() {
		var a string
		if err := rows.Scan(&a); err != nil {
			rows.Close()
			return errf(s.DB, "comment authors", err)
		}
		if strings.EqualFold(a, name) {
			authors = append(authors, a)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return errf(s.DB, "comment authors", err)
	}
	if len(authors) == 0 {
		where.WriteString(" AND 1 = 0")
		return nil
	}
	where.WriteString(" AND EXISTS (SELECT 1 FROM comment c WHERE " + cTenant +
		" AND c.position_id = p.id AND COALESCE(c.text, '') <> '' AND c.author IN (" +
		Placeholders(len(authors)) + "))")
	*args = append(*args, cArgs...)
	*args = append(*args, authors...)
	return nil
}

func (s *SearchStore) appendClosedListClauses(scope string, f domain.SearchFilters, where *strings.Builder, args *[]any) {
	// Comment presence: `co` (has one) / `xco` (has none). Asking for both is
	// contradictory rather than ambiguous; "none" wins and the search comes
	// back empty, which is the honest answer. COALESCE is deliberate:
	// `c.text <> ''` is NULL on a NULL text, which EXISTS would drop and NOT
	// EXISTS keep; empty text counts as no comment (CONTEXT.md). The subquery
	// carries the tenant predicate: idx_comment_position is keyed on it.
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

	// Tags. This clause is a NARROWING, not the answer: a LIKE cannot tell
	// #prime from #priming. It exists so a tag search does not preload every comment
	// in the database; the exact, delimited test runs in Go on the survivors
	// (domain.MatchesAllTags). One EXISTS per tag, because every named tag
	// must be present — see domain.SearchFilters.TagFilter for why that is
	// AND where the two lists around it are OR.
	for _, tag := range domain.ParseTagFilter(f.TagFilter) {
		cTenant, cArgs := s.DB.TenantFilter("c", scope)
		where.WriteString(" AND EXISTS (SELECT 1 FROM comment c WHERE " + cTenant +
			" AND c.position_id = p.id AND COALESCE(c.text, '') " + s.DB.ILike() + " ?)")
		*args = append(*args, cArgs...)
		*args = append(*args, "%"+tag+"%")
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

	// Derived plan of play, read the same way and for the same
	// reason: one indexed column, never reclassified at query time.
	if types := domain.SplitFilterList(f.GameTypeFilter); len(types) > 0 {
		var codes []any
		for _, name := range types {
			if gt, ok := domain.ParseGameType(name); ok {
				codes = append(codes, int(gt))
			}
		}
		if len(codes) > 0 {
			where.WriteString(" AND p.game_type IN (" + Placeholders(len(codes)) + ")")
			*args = append(*args, codes...)
		}
	}
}
