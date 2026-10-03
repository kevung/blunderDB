package sqlshared

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
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
// already computed while reading f.
type searchWhereClause struct {
	where         string
	args          []any
	needAnalysis  bool
	useSQLFilters bool
	bitboardTight bool
	// multiPlayed lists the positions player 1 played more than one way;
	// only filled by a plain move-error search, where those rows escape the
	// SQL column and are scored in Go.
	multiPlayed map[int64]bool
	// effInclude is f.Filter with the points shared with ExcludeFilter
	// cleared, so "Except" wins over "At least" on those points.
	effInclude domain.Position
	// likeTarget is the position a ranked query ranks against, loaded once
	// here because the class it imposes on the WHERE clause is read off it —
	// its kind of decision, its regime, and the matches it was met in.
	likeTarget *domain.Position
}

// buildWhere translates f into the WHERE clause of the search query: cheap
// predicates that can be pushed to SQL become clause text and bound
// arguments; the rest are left to applyGoFilters (matchesGoFilters below),
// which is what needAnalysis/useSQLFilters/bitboardTight/multiPlayed/
// effInclude in the returned searchWhereClause are for.
func (s *SearchStore) buildWhere(ctx context.Context, scope string, f domain.SearchFilters) (searchWhereClause, error) {
	useSQLFilters := !f.MirrorFilter
	var multiPlayed map[int64]bool

	// Decode the compressed analysis blob per row only when a Go-side filter
	// reads it: move pattern, mirror re-checks, date, equity. The rate, cube
	// and move-error filters run on denormalised SQL columns. MoveErrorFilter
	// is deliberately NOT a trigger: its Go re-check runs on the mirror path
	// (already covered) or on the few multiPlayed positions, loaded one by one
	// after the scan; triggering here decodes every row for nothing.
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

	// Comment presence, origin, tags and phase are row properties too, so they
	// stay in SQL even in mirror search (and in SQL for cost: a presence filter
	// is often the only thing narrowing the scan).
	s.appendClosedListClauses(scope, f, &where, &args)

	if err := s.appendIdentityClauses(ctx, scope, f, &where, &args); err != nil {
		return searchWhereClause{}, err
	}

	// A ranked query narrows to the target's equivalence class BEFORE anything
	// else: a neighbour is the same problem nearby, and the rest of the WHERE
	// clause then says which of those problems the user is asking about
	// (ADR-0043).
	var likeTarget *domain.Position
	if f.LikeFilter {
		// The target is the named stored position, else the drawn board
		// (ADR-0043 rule 3), read as a POSITION, not a pattern: checkers not
		// placed count as borne off.
		var t *domain.Position
		if f.LikeTargetID > 0 {
			loaded, err := LoadTargetPosition(ctx, s.DB, scope, f.LikeTargetID)
			if err != nil {
				return searchWhereClause{}, err
			}
			t = loaded
		} else {
			board := f.LikeTargetBoard
			t = &board
		}
		likeTarget = t
		class := storage.SimilarOptions{}
		if !f.LikeWidened {
			class = storage.ClassOf(t)
		}
		excluded, err := MatchesOfPosition(ctx, s.DB, scope, t.ID)
		if err != nil {
			return searchWhereClause{}, err
		}
		AppendClassSQL(class, excluded, &where, &args)
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

		// How many times the position was MET: the move rows that reach it. A
		// correlated subquery rather than a denormalised column that every
		// import would have to keep true; idx_move_position answers it.
		if f.EncounterFilter != "" {
			nMin, nMax, nHasMin, nHasMax := searchfilter.ParseIntFilterExpr(f.EncounterFilter, "n")
			searchfilter.AppendIntRangeSQL("(SELECT COUNT(*) FROM move mv WHERE mv.position_id = p.id)",
				nMin, nMax, nHasMin, nHasMax, &where, &args)
		}

		// Win/gammon rate as `p.id IN (SELECT position_id FROM analysis …)`,
		// not a clause on the LEFT JOIN: the join form drives the scan through
		// idx_analysis_win_gammon in rate order and forces a TEMP B-TREE sort
		// for the ORDER BY p.id. The IN-subquery keeps the rowid-order scan and
		// is answered from the covering index (position_id is its third column;
		// idx_analysis_win_gammon_covering on PostgreSQL, hence the tenant
		// predicate there).
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

		// The denormalised error column scores ONE play (the first of
		// PlayedMoves, see AnalysisStore.Save). A position played several ways
		// is let through and settled in Go by matchesMoveErrorFilter on the
		// largest error: the column can only under-state it. The set is listed
		// once before the scan; a correlated subquery here doubled query time.
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
		likeTarget:    likeTarget,
	}, nil
}

// Rank answers a query carrying the `like` token: the same candidates Find
// would return, ordered by how far each stands from the target, with that
// distance attached (ADR-0043).
//
// The ordering is done in Go, necessarily: several filters are decided in
// applyGoFilters after the scan, so the ranking must see all the survivors.
// ADR-0043 keeps that exhaustive scan over an approximate index.
func (s *SearchStore) Rank(ctx context.Context, scope string, f domain.SearchFilters, opts storage.ListOpts) ([]storage.SimilarPosition, error) {
	if !f.LikeFilter {
		return nil, fmt.Errorf("Rank needs a query carrying the `like` token; use Find for an unordered search")
	}
	wc, err := s.buildWhere(ctx, scope, f)
	if err != nil {
		return nil, err
	}
	// The candidates are gathered UNBOUNDED: opts bounds the ranking, not the
	// scan. Paging the scan would rank an arbitrary page and call it nearest.
	candidates, err := s.findWith(ctx, f, wc, storage.ListOpts{})
	if err != nil {
		return nil, err
	}

	wanted := engine.BuildSimilarityVector(wc.likeTarget)
	out := make([]storage.SimilarPosition, 0, len(candidates))
	for i := range candidates {
		if candidates[i].ID == wc.likeTarget.ID {
			continue
		}
		d := engine.SimilarityDistance(wanted, engine.BuildSimilarityVector(&candidates[i]))
		// The ceiling drops a neighbour outright rather than ranking it last:
		// a ranking that finds nothing close comes back empty, and says so,
		// instead of handing over the least distant of the unrelated.
		if f.LikeMaxDistance > 0 && d > f.LikeMaxDistance {
			continue
		}
		out = append(out, storage.SimilarPosition{Position: candidates[i], Distance: d})
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

func (s *SearchStore) find(ctx context.Context, scope string, f domain.SearchFilters, opts storage.ListOpts) ([]domain.Position, error) {
	wc, err := s.buildWhere(ctx, scope, f)
	if err != nil {
		return nil, err
	}
	return s.findWith(ctx, f, wc, opts)
}

// findWith is find once the WHERE clause is built: the scan, and the Go-side
// predicates that finish it. Rank shares it so a ranked query and a plain one
// select the same rows by the same rules, and differ only in what happens to
// the survivors afterwards.
func (s *SearchStore) findWith(ctx context.Context, f domain.SearchFilters, wc searchWhereClause, opts storage.ListOpts) ([]domain.Position, error) {
	var positions []domain.Position
	err := s.scan(ctx, f, wc, opts, func(pos domain.Position) bool {
		positions = append(positions, pos)
		return true
	})
	return positions, err
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
		(f.MoveErrorFilter != "" && len(wc.multiPlayed) > 0) ||
		f.Player1CheckerInZoneFilter != "" || f.Player2CheckerInZoneFilter != "" ||
		f.Player1OutfieldBlotFilter != "" || f.Player2OutfieldBlotFilter != "" ||
		f.Player1JanBlotFilter != "" || f.Player2JanBlotFilter != "" ||
		f.SearchText != "" || f.TagFilter != "" ||
		f.DateFilter != "" || f.EquityFilter != "" || f.MovePatternFilter != ""
}

// keysetOrder reports whether the sort is the id alone, so that a chunked
// scan can resume after the last id seen instead of skipping an OFFSET that
// grows with every chunk.
func keysetOrder(sort string) bool {
	return domain.SearchOrderByClause(sort) == "p.id"
}

// scan hands visit the survivors of the search, in order, within opts: the
// window is counted on the survivors, not on the SQL candidates, so a page
// is never short because the Go phase rejected rows of it. visit returning
// false stops the scan.
//
// Without a Go phase, LIMIT/OFFSET go to SQL and the first page comes back
// as soon as SQL has it. With one, the candidates are read searchChunk at a
// time and the scan stops once the window is full.
func (s *SearchStore) scan(ctx context.Context, f domain.SearchFilters, wc searchWhereClause, opts storage.ListOpts, visit func(domain.Position) bool) error {
	if !wc.goPhase(f) {
		positions, _, _, err := s.scanChunk(ctx, f, wc, "", nil, opts.Limit, opts.Offset)
		if err != nil {
			return err
		}
		for _, pos := range positions {
			if !visit(pos) {
				return nil
			}
		}
		return nil
	}

	keyset := keysetOrder(f.Sort)
	maxChunk := int(searchChunk.Load())
	chunk := min(searchFirstChunk, maxChunk)
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
				extra, extraArgs = " AND p.id > ?", []any{after}
			}
		}
		positions, n, lastID, err := s.scanChunk(ctx, f, wc, extra, extraArgs, chunk, offset)
		if err != nil {
			return err
		}
		for _, pos := range positions {
			if skip > 0 {
				skip--
				continue
			}
			if !visit(pos) {
				return nil
			}
			taken++
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

	query := `SELECT p.id, p.state,
		p.decision_type, p.player_on_roll, p.dice_1, p.dice_2,
		p.cube_value, p.cube_owner, p.score_1, p.score_2,
		p.has_jacoby, p.has_beaver, p.max_cube, p.is_cube_response,
		p.individually_imported, p.flagged,
		a.id, ` + analysisDataCol + ` AS data
	FROM position p
	LEFT JOIN analysis a ON a.position_id = p.id
	WHERE ` + wc.where + extra + ` ORDER BY ` + domain.SearchOrderByClause(f.Sort) + limitClause

	args := append(append(append([]any{}, wc.args...), extraArgs...), limitArgs...)
	rows, err := s.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, 0, errf(s.DB, "search query", err)
	}
	defer rows.Close()

	scanned, err := s.scanRows(rows, wc.needAnalysis)
	if err != nil {
		return nil, 0, 0, err
	}
	var lastID int64
	if len(scanned) > 0 {
		lastID = scanned[len(scanned)-1].pos.ID
	}

	positions, err := s.applyGoFilters(ctx, f, wc, scanned)
	return positions, len(scanned), lastID, err
}

// FindIDs returns the ids of the survivors of the search, within opts. With
// no Go phase it is a projection on p.id: no row is decoded, no position
// reconstructed.
func (s *SearchStore) FindIDs(ctx context.Context, scope string, f domain.SearchFilters, opts storage.ListOpts) ([]int64, error) {
	wc, err := s.buildWhere(ctx, scope, f)
	if err != nil {
		return nil, err
	}
	if wc.goPhase(f) {
		var ids []int64
		err := s.scan(ctx, f, wc, opts, func(pos domain.Position) bool {
			ids = append(ids, pos.ID)
			return true
		})
		return ids, err
	}

	limitClause, limitArgs := s.DB.LimitOffset(opts.Limit, opts.Offset)
	query := `SELECT p.id FROM position p
	LEFT JOIN analysis a ON a.position_id = p.id
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
		err := s.scan(ctx, f, wc, storage.ListOpts{}, func(domain.Position) bool {
			n++
			return true
		})
		return n, err
	}
	return s.countWhere(ctx, wc, "", nil)
}

func (s *SearchStore) countWhere(ctx context.Context, wc searchWhereClause, extra string, extraArgs []any) (int, error) {
	query := `SELECT COUNT(*) FROM position p
	LEFT JOIN analysis a ON a.position_id = p.id
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
	err = s.scan(ctx, f, wc, storage.ListOpts{}, func(pos domain.Position) bool {
		if pos.ID == id {
			found = true
			return false
		}
		index++
		return true
	})
	if err != nil || !found {
		return 0, false, err
	}
	return index, true, nil
}

// sortKey returns the column a search sort orders on before its p.id
// tiebreak, descending with NULLs last; "" for the id order. It is read off
// domain.SearchOrderByClause so the rank cannot drift from the ORDER BY.
func sortKey(order string) (col string, ok bool) {
	clause := domain.SearchOrderByClause(order)
	if clause == "p.id" {
		return "", true
	}
	return strings.CutSuffix(clause, " DESC NULLS LAST, p.id")
}

// indexOfInSQL ranks id without reading a row: one query for id's sort key
// (none means the search does not find it), one COUNT of the rows ordered
// before it under "key DESC NULLS LAST, p.id".
func (s *SearchStore) indexOfInSQL(ctx context.Context, wc searchWhereClause, order string, id int64) (int, bool, error) {
	col, ok := sortKey(order)
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
	LEFT JOIN analysis a ON a.position_id = p.id
	WHERE ` + wc.where + ` AND p.id = ?`
	var key any
	switch err := s.DB.QueryRow(ctx, query, append(append([]any{}, wc.args...), id)...).Scan(&key); {
	case errors.Is(err, ErrNoRows):
		return 0, false, nil
	case err != nil:
		return 0, false, errf(s.DB, "search rank key", err)
	}
	// NULLs sort last: before a NULL key come every keyed row and the NULL
	// rows of a smaller id; before a key k, the larger keys and k's smaller ids.
	extra, extraArgs := " AND ("+col+" IS NOT NULL OR p.id < ?)", []any{id}
	if key != nil {
		extra, extraArgs = " AND ("+col+" > ? OR ("+col+" = ? AND p.id < ?))", []any{key, key, id}
	}
	before, err := s.countWhere(ctx, wc, extra, extraArgs)
	return before, err == nil, err
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
	// Drain the cursor before filtering: the Go-side predicates open queries
	// of their own, and a cursor holds its pooled connection until exhausted.
	// On ":memory:" (one connection, sqlite.ConfigurePool) that deadlocks at
	// once; on a real pool, under enough concurrent searches. The buffer is
	// one chunk of the scan, never the whole library.
	var scanned []scannedRow

	for rows.Next() {
		// Nullable columns scan into pointers, which both drivers leave nil on
		// NULL; the flags are INTEGER 0/1 on SQLite and BOOLEAN on PostgreSQL,
		// and both drivers convert either into a *bool.
		var posID int64
		var posState string
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
			return nil, errf(s.DB, "search scan", err)
		}

		position := engine.ReconstructPosition(posID, posState,
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
// too), preloading per-family batch queries first.
func (s *SearchStore) applyGoFilters(ctx context.Context, f domain.SearchFilters, wc searchWhereClause, scanned []scannedRow) ([]domain.Position, error) {
	var err error
	// Preload comment texts and player-1 plays in one batched query per
	// family instead of one per candidate, only when the filter is active and
	// after the cursor is drained.
	var commentTexts map[int64]string
	var player1MovesByID map[int64]player1Moves
	if f.SearchText != "" || f.TagFilter != "" || f.MoveErrorFilter != "" {
		ids := make([]int64, len(scanned))
		for i, row := range scanned {
			ids[i] = row.pos.ID
		}
		if f.SearchText != "" || f.TagFilter != "" {
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
	wantedTags := domain.ParseTagFilter(f.TagFilter)

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
				// A multi-played position is scored by its largest error; its
				// blob may not have been fetched with the scan, so load it now.
				if ana == nil {
					ana = loadAnalysis(ctx, s.DB, pos.ID)
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
// outside the board: the matches or tournaments they were met in, the player
// who sat at one of the seats, an explicit list of position ids. They are one
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
		*args = append(*args, playerName, playerName)
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
