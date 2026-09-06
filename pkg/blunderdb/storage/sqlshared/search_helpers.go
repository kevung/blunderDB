package sqlshared

import (
	"context"
	"math"
	"sort"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/searchfilter"
)

// This file holds the search-filter predicates that need database access,
// ported from the Database wrapper's db_search.go and db_filter_match.go and
// re-expressed against Execer (package database imports storage/sqlite, so
// the reverse import is not possible). The pure parsers and in-memory
// predicates live in storage/searchfilter.
//
// Every predicate below that touches the database returns an error alongside
// its bool: a locked database or a dropped connection must surface as a
// search failure, never as a silent "does not match" that empties or shrinks
// the result set without saying why (B.6, #174).

// getMatchIDsForTournament returns all match IDs belonging to a tournament.
// A scan failure is propagated rather than skipped: silently dropping one
// row here used to shrink a tournament filter to fewer matches than the
// tournament actually has, indistinguishable from "the tournament really
// only has that many".
func getMatchIDsForTournament(ctx context.Context, db Execer, tournamentID int64) ([]int64, error) {
	rows, err := db.Query(ctx, `SELECT id FROM match WHERE tournament_id = ?`, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// forEachIDBatch runs `prefix (?,?,…) suffix` over ids in chunks that stay
// under SQLite's bound-variable limit (32766 by default — a real search
// narrows to tens of thousands of candidates), handing every row to scan. It
// restates storage/sqlite's forEachIn against the dialect-agnostic Execer, so
// the batched preloads below run unchanged on both backends.
func forEachIDBatch(ctx context.Context, db Execer, ids []int64, prefix, suffix string, scan func(Rows) error) error {
	const chunk = 900
	for start := 0; start < len(ids); start += chunk {
		end := start + chunk
		if end > len(ids) {
			end = len(ids)
		}
		batch := ids[start:end]
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(batch)), ",")
		args := make([]any, len(batch))
		for i, id := range batch {
			args[i] = id
		}
		if err := func() error {
			rows, err := db.Query(ctx, prefix+"("+placeholders+")"+suffix, args...)
			if err != nil {
				return err
			}
			defer rows.Close()
			for rows.Next() {
				if err := scan(rows); err != nil {
					return err
				}
			}
			return rows.Err()
		}(); err != nil {
			return err
		}
	}
	return nil
}

// loadCommentTexts returns the concatenated comment text of every id in
// positionIDs, keyed by position id — the batched form of what used to be a
// per-position query (loadCommentText), run once per SQL-matched candidate
// of a SearchText filter (B.10, #178). A position with no comment text is
// simply absent from the map. A position may have several comment entries
// (see AddComment); all of them are joined, in id order, so the "Search
// Text" filter can match against any one of them.
func loadCommentTexts(ctx context.Context, db Execer, positionIDs []int64) (map[int64]string, error) {
	parts := make(map[int64][]string)
	err := forEachIDBatch(ctx, db, positionIDs,
		`SELECT position_id, text FROM comment WHERE position_id IN `,
		` AND text != '' ORDER BY position_id ASC, id ASC`,
		func(rows Rows) error {
			var id int64
			var text string
			if err := rows.Scan(&id, &text); err != nil {
				return err
			}
			parts[id] = append(parts[id], text)
			return nil
		})
	if err != nil {
		return nil, err
	}
	out := make(map[int64]string, len(parts))
	for id, ps := range parts {
		out[id] = strings.Join(ps, "\n\n")
	}
	return out, nil
}

// player1Moves is one position's distinct player-1 plays: the batched
// counterpart of getPlayer1MovesForPosition's two return values.
type player1Moves struct {
	checkerMoves []string
	cubeActions  []string
}

// loadPlayer1Moves returns, for every id in positionIDs, player-1's distinct
// checker moves and cube actions recorded in the move table — the batched
// form of what used to be a per-position query (getPlayer1MovesForPosition),
// run once per candidate needing the move-error filter or the take/pass
// mirror check (B.10, #178). A position deduplicated across matches
// (CONTEXT.md, Deduplication) can carry several plays; each list is sorted so
// the predicates built on top do not inherit a run-to-run lottery from map
// iteration (#167).
func loadPlayer1Moves(ctx context.Context, db Execer, positionIDs []int64) (map[int64]player1Moves, error) {
	checkerSets := make(map[int64]map[string]bool)
	cubeSets := make(map[int64]map[string]bool)
	err := forEachIDBatch(ctx, db, positionIDs,
		`SELECT position_id, checker_move, cube_action FROM move WHERE player = 1 AND position_id IN `,
		``,
		func(rows Rows) error {
			var id int64
			var cm, ca *string
			if err := rows.Scan(&id, &cm, &ca); err != nil {
				return err
			}
			if cm != nil && *cm != "" {
				if checkerSets[id] == nil {
					checkerSets[id] = make(map[string]bool)
				}
				checkerSets[id][engine.NormalizeMove(*cm)] = true
			}
			if ca != nil && *ca != "" {
				if cubeSets[id] == nil {
					cubeSets[id] = make(map[string]bool)
				}
				cubeSets[id][*ca] = true
			}
			return nil
		})
	if err != nil {
		return nil, err
	}
	out := make(map[int64]player1Moves, len(checkerSets)+len(cubeSets))
	for id, set := range checkerSets {
		m := out[id]
		m.checkerMoves = sortedKeys(set)
		out[id] = m
	}
	for id, set := range cubeSets {
		m := out[id]
		m.cubeActions = sortedKeys(set)
		out[id] = m
	}
	return out, nil
}

func sortedKeys(set map[string]bool) []string {
	if len(set) == 0 {
		return nil
	}
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// multiPlayedPlayer1Positions lists the positions on which player 1 recorded
// more than one distinct play (checker move or cube action) in the move
// table — positions deduplicated across matches and played differently each
// time. On those the denormalised error column, which scores a single play,
// cannot answer a move-error filter; SearchStore.find lets them through to
// matchesMoveErrorFilter instead (#167). The set is small by nature (openings
// and early replies: 14 positions on the 156-match tournois fixture), which
// is why it is listed once here rather than tested per row in the search
// query. The self-join walks idx_move_position once per player-1 move and
// costs ~15 ms on that fixture's 58 000 moves, against ~25 ms for the
// equivalent GROUP BY … HAVING COUNT(DISTINCT …), whose distinct count needs a
// temporary B-tree. A move written two ways ("13/11 24/23" and "24/23 13/11")
// counts as two plays here; the Go re-check normalises them, so the cost is
// one needless decode, never a wrong answer.
func multiPlayedPlayer1Positions(ctx context.Context, db Execer, scope string) (map[int64]bool, error) {
	tenant, args := db.TenantFilter("m1", scope)
	rows, err := db.Query(ctx,
		`SELECT DISTINCT m1.position_id FROM move m1
		 JOIN move m2 ON m2.position_id = m1.position_id AND m2.id > m1.id AND m2.player = 1
		 WHERE `+tenant+` AND m1.player = 1 AND m1.position_id IS NOT NULL
		   AND (COALESCE(m1.checker_move, '') <> COALESCE(m2.checker_move, '')
		     OR COALESCE(m1.cube_action, '') <> COALESCE(m2.cube_action, ''))`,
		args...)
	if err != nil {
		return nil, errf(db, "multi-played positions", err)
	}
	defer rows.Close()
	ids := make(map[int64]bool)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, errf(db, "multi-played positions scan", err)
		}
		ids[id] = true
	}
	if err := rows.Err(); err != nil {
		return nil, errf(db, "multi-played positions rows", err)
	}
	return ids, nil
}

// loadAnalysis returns a position's decoded analysis, or nil when it has none
// or the blob does not decode — the same answer the search scan gives a row
// it could not decode.
func loadAnalysis(ctx context.Context, db Execer, positionID int64) *domain.PositionAnalysis {
	var data []byte
	if err := db.QueryRow(ctx, `SELECT data FROM analysis WHERE position_id = ?`, positionID).Scan(&data); err != nil {
		return nil
	}
	a, err := engine.DecodeAnalysisFromStorage(data)
	if err != nil {
		return nil
	}
	return &a
}

// matchesSearchTextPreloaded reports whether comment (the position's
// concatenated comment text, from loadCommentTexts) matches a "t"-filter.
func matchesSearchTextPreloaded(comment, searchText string) bool {
	keywords := searchfilter.ParseSearchTextKeywords(searchText)
	if len(keywords) == 0 {
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

// isPlayer1TakePassCubeActionPreloaded reports whether player-1's recorded
// cube action for a position (moves, from loadPlayer1Moves) was a take or
// pass.
func isPlayer1TakePassCubeActionPreloaded(moves player1Moves) bool {
	for _, action := range moves.cubeActions {
		if engine.IsResponseCubeAction(action) {
			return true
		}
	}
	return false
}

// matchesMoveErrorFilterPreloaded filters positions by the equity error of
// player-1's played move (millipoints): E>x, E<x, Ex,y. analysis is the
// position's already-decoded analysis (the caller decoded it once from
// a.data — see search.go) and moves its preloaded plays (loadPlayer1Moves);
// neither is fetched here, so this predicate makes no query of its own.
//
// A position played more than once by player 1 (deduplicated across matches)
// has several recorded plays, possibly with different errors. The filter
// scores the position by the LARGEST of them: "E>100" asks "did I ever
// blunder here?", and the answer must not depend on which play the move
// table happens to yield first — before #167 it did, and the same search
// returned a different set from one run to the next. The decision is stated
// in doc/source/cmd_mode.rst next to the E filter; SearchStore.find routes
// the multi-played positions of a plain (non-mirror) search here too.
func matchesMoveErrorFilterPreloaded(analysis *domain.PositionAnalysis, moves player1Moves, filter string) bool {
	if analysis == nil {
		return false
	}
	moveError, found := player1MaxMoveError(analysis, moves.checkerMoves, moves.cubeActions)
	if !found {
		return false
	}
	return searchfilter.MatchesMoveError(math.Round(moveError*1000), filter)
}

// player1MaxMoveError returns the largest absolute equity error among the
// plays player 1 recorded on a position, as scored by its analysis: a checker
// move is looked up among the analysed candidates (the best move costs 0), a
// cube action goes through engine.CubeActionError. A play the analysis does
// not score (a move absent from the candidate list, an unknown cube label) is
// skipped; found is false when no play was scored at all.
func player1MaxMoveError(analysis *domain.PositionAnalysis, checkerMoves, cubeActions []string) (maxError float64, found bool) {
	switch {
	case analysis.AnalysisType == "CheckerMove" && analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 0:
		for _, played := range checkerMoves {
			normPlayed := engine.NormalizeMove(played)
			for i, m := range analysis.CheckerAnalysis.Moves {
				if !strings.EqualFold(engine.NormalizeMove(m.Move), normPlayed) {
					continue
				}
				var e float64
				if i > 0 && m.EquityError != nil {
					e = math.Abs(*m.EquityError)
				}
				maxError = math.Max(maxError, e)
				found = true
				break
			}
		}
	case analysis.AnalysisType == "DoublingCube" && analysis.DoublingCubeAnalysis != nil:
		for _, played := range cubeActions {
			if e, ok := engine.CubeActionError(analysis.DoublingCubeAnalysis, played); ok {
				maxError = math.Max(maxError, math.Abs(e))
				found = true
			}
		}
	}
	return maxError, found
}

// matchesZoneAndBlotFilters is the group of board-shape predicates that need
// nothing but the position and the filters: checkers in a zone, and blots in
// the outfield or in the home board, for each player.
//
// Extracted from applyGoFilters' predicate because it is a group — six
// filters, one subject, no dependency on the scan's preloaded data — and
// because the predicate it came from sits at the linter's complexity ceiling
// (gocognit 200) and every filter added to it pushes it over.
func matchesZoneAndBlotFilters(pos domain.Position, f domain.SearchFilters) bool {
	switch {
	case f.Player1CheckerInZoneFilter != "" && !pos.MatchesPlayer1CheckerInZone(f.Player1CheckerInZoneFilter):
		return false
	case f.Player2CheckerInZoneFilter != "" && !pos.MatchesPlayer2CheckerInZone(f.Player2CheckerInZoneFilter):
		return false
	case f.Player1OutfieldBlotFilter != "" && !pos.MatchesPlayer1OutfieldBlot(f.Player1OutfieldBlotFilter):
		return false
	case f.Player2OutfieldBlotFilter != "" && !pos.MatchesPlayer2OutfieldBlot(f.Player2OutfieldBlotFilter):
		return false
	case f.Player1JanBlotFilter != "" && !pos.MatchesPlayer1JanBlot(f.Player1JanBlotFilter):
		return false
	case f.Player2JanBlotFilter != "" && !pos.MatchesPlayer2JanBlot(f.Player2JanBlotFilter):
		return false
	}
	return true
}

// matchesCommentFilters is the pair of predicates read off a position's
// comment text: the free-text search and the tag search.
//
// One text, two rules, and the difference is the point (#265). SearchText is a
// SUBSTRING search — `t"#prime"` matches "#priming" — while the tag filter
// extracts the text's tags and compares them whole, so `#prime` does not.
// wantedTags is already normalised (domain.ParseTagFilter) by the caller,
// once for the whole scan.
func matchesCommentFilters(comment, searchText string, wantedTags []string) bool {
	if searchText != "" && !matchesSearchTextPreloaded(comment, searchText) {
		return false
	}
	return domain.MatchesAllTags(comment, wantedTags)
}
