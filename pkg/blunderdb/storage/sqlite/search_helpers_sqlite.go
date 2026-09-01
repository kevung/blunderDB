package sqlite

import (
	"context"
	"database/sql"
	"math"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/searchfilter"
)

// This file holds the search-filter predicates that need database access,
// ported from the Database wrapper's db_search.go and db_filter_match.go and
// re-expressed against execer (package database imports storage/sqlite, so
// the reverse import is not possible). The pure parsers and in-memory
// predicates live in storage/searchfilter, shared with the PostgreSQL backend.

// getMatchIDsForTournament returns all match IDs belonging to a tournament.
func getMatchIDsForTournament(ctx context.Context, db execer, tournamentID int64) ([]int64, error) {
	rows, err := db.QueryContext(ctx, `SELECT id FROM match WHERE tournament_id = ?`, tournamentID)
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

// loadCommentText returns the concatenated comment text of a position. A
// position may have several comment entries (see AddComment); all of them are
// joined so the "Search Text" filter can match against any one of them.
func loadCommentText(ctx context.Context, db execer, positionID int64) (string, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT text FROM comment WHERE position_id = ? AND text != '' ORDER BY id ASC`,
		positionID)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var parts []string
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err != nil {
			return "", err
		}
		parts = append(parts, text)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	return strings.Join(parts, "\n\n"), nil
}

// getPlayer1MovesForPosition returns player-1's checker moves and cube actions
// recorded in the move table for a position.
func getPlayer1MovesForPosition(ctx context.Context, db execer, positionID int64) ([]string, []string) {
	rows, err := db.QueryContext(ctx,
		`SELECT checker_move, cube_action FROM move WHERE position_id = ? AND player = 1`, positionID)
	if err != nil {
		return nil, nil
	}
	defer rows.Close()
	checkerMoves := make(map[string]bool)
	cubeActions := make(map[string]bool)
	for rows.Next() {
		var cm, ca sql.NullString
		if err := rows.Scan(&cm, &ca); err != nil {
			continue
		}
		if cm.Valid && cm.String != "" {
			checkerMoves[engine.NormalizeMove(cm.String)] = true
		}
		if ca.Valid && ca.String != "" {
			cubeActions[ca.String] = true
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
	keywords := searchfilter.ParseSearchTextKeywords(searchText)
	if len(keywords) == 0 {
		return false
	}
	comment, err := loadCommentText(ctx, db, p.ID)
	if err != nil {
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

// isPlayer1TakePassCubeAction reports whether player-1's recorded cube action
// for a position was a take or pass.
func isPlayer1TakePassCubeAction(ctx context.Context, db execer, p *domain.Position) bool {
	_, player1CubeActions := getPlayer1MovesForPosition(ctx, db, p.ID)
	for _, action := range player1CubeActions {
		if engine.IsResponseCubeAction(action) {
			return true
		}
	}
	return false
}

// matchesMoveErrorFilter filters positions by the equity error of player-1's
// played move (millipoints): E>x, E<x, Ex,y. analysis is the position's
// already-decoded analysis (the caller decoded it once from a.data because
// MoveErrorFilter is in needAnalysis — see search_sqlite.go); this predicate
// no longer re-queries and re-decompresses it per row.
func matchesMoveErrorFilter(ctx context.Context, db execer, p *domain.Position, analysis *domain.PositionAnalysis, filter string) bool {
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
		for _, played := range playedActions {
			if e, ok := engine.CubeActionError(analysis.DoublingCubeAnalysis, played); ok {
				moveError = math.Abs(e)
				found = true
				break
			}
		}
	}

	if !found {
		return false
	}

	return searchfilter.MatchesMoveError(math.Round(moveError*1000), filter)
}
