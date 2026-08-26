package database

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
)

// ========== Tournament Functions ==========

// CreateTournament creates a new tournament
func (d *Database) CreateTournament(name string, date string, location string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return 0, fmt.Errorf("no database is currently open")
	}

	// Get the max sort_order
	var maxOrder int
	err := d.db.QueryRow(`SELECT COALESCE(MAX(sort_order), -1) FROM tournament`).Scan(&maxOrder)
	if err != nil {
		maxOrder = -1
	}

	result, err := d.db.Exec(`
		INSERT INTO tournament (name, date, location, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, datetime('now'), datetime('now'))
	`, name, date, location, maxOrder+1)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return id, nil
}

// GetAllTournaments returns all tournaments with their match counts
func (d *Database) GetAllTournaments() ([]Tournament, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	rows, err := d.db.Query(`
		SELECT 
			t.id,
			t.name,
			COALESCE(t.date, ''),
			COALESCE(t.location, ''),
			t.sort_order,
			t.created_at,
			t.updated_at,
			COUNT(m.id) as match_count,
			COALESCE(t.comment, '')
		FROM tournament t
		LEFT JOIN match m ON t.id = m.tournament_id
		GROUP BY t.id
		ORDER BY t.date DESC, t.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tournaments []Tournament
	for rows.Next() {
		var t Tournament
		err := rows.Scan(&t.ID, &t.Name, &t.Date, &t.Location, &t.SortOrder, &t.CreatedAt, &t.UpdatedAt, &t.MatchCount, &t.Comment)
		if err != nil {
			slog.Warn("scanning tournament", "err", err)
			continue
		}
		tournaments = append(tournaments, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := d.applyTournamentBadges(tournaments); err != nil {
		// Non-fatal
		_ = err
	}

	return tournaments, nil
}

// UpdateTournament updates a tournament's details
func (d *Database) UpdateTournament(id int64, name string, date string, location string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	_, err := d.db.Exec(`
		UPDATE tournament SET name = ?, date = ?, location = ?, updated_at = datetime('now')
		WHERE id = ?
	`, name, date, location, id)
	if err != nil {
		return err
	}

	return nil
}

// DeleteTournament deletes a tournament (matches are unlinked, not deleted)
func (d *Database) DeleteTournament(id int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Unlink matches from this tournament
	_, err = tx.Exec(`UPDATE match SET tournament_id = NULL WHERE tournament_id = ?`, id)
	if err != nil {
		return err
	}

	// Delete the tournament
	_, err = tx.Exec(`DELETE FROM tournament WHERE id = ?`, id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// AddMatchToTournament adds a match to a tournament
func (d *Database) AddMatchToTournament(tournamentID int64, matchID int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get the max sort order for this tournament
	var maxOrder int
	err = tx.QueryRow(`SELECT COALESCE(MAX(tournament_sort_order), -1) FROM match WHERE tournament_id = ?`, tournamentID).Scan(&maxOrder)
	if err != nil {
		maxOrder = -1
	}

	_, err = tx.Exec(`UPDATE match SET tournament_id = ?, tournament_sort_order = ? WHERE id = ?`, tournamentID, maxOrder+1, matchID)
	if err != nil {
		return err
	}

	// Update tournament's updated_at
	_, err = tx.Exec(`UPDATE tournament SET updated_at = datetime('now') WHERE id = ?`, tournamentID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// RemoveMatchFromTournament removes a match from a tournament
func (d *Database) RemoveMatchFromTournament(matchID int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	_, err := d.db.Exec(`UPDATE match SET tournament_id = NULL, tournament_sort_order = 0 WHERE id = ?`, matchID)
	return err
}

// UpdateMatchComment updates the comment of a match
func (d *Database) UpdateMatchComment(matchID int64, comment string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	_, err := d.db.Exec(`UPDATE match SET comment = ? WHERE id = ?`, comment, matchID)
	return err
}

// UpdateTournamentComment updates the comment of a tournament
func (d *Database) UpdateTournamentComment(tournamentID int64, comment string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	_, err := d.db.Exec(`UPDATE tournament SET comment = ?, updated_at = datetime('now') WHERE id = ?`, comment, tournamentID)
	return err
}

// ReorderTournamentMatches sets the sort order for matches in a tournament.
// matchIDs should be in the desired order.
func (d *Database) ReorderTournamentMatches(tournamentID int64, matchIDs []int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for i, matchID := range matchIDs {
		_, err := tx.Exec(`UPDATE match SET tournament_sort_order = ? WHERE id = ? AND tournament_id = ?`, i, matchID, tournamentID)
		if err != nil {
			return err
		}
	}

	_, err = tx.Exec(`UPDATE tournament SET updated_at = datetime('now') WHERE id = ?`, tournamentID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// SetMatchTournamentByName assigns a match to a tournament by name.
// If tournamentName is empty, the match is unlinked from any tournament.
// If no tournament with that name exists, one is created.
func (d *Database) SetMatchTournamentByName(matchID int64, tournamentName string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	name := strings.TrimSpace(tournamentName)
	if name == "" {
		_, err := d.db.Exec(`UPDATE match SET tournament_id = NULL WHERE id = ?`, matchID)
		return err
	}

	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Look for existing tournament with that name
	var tournamentID int64
	err = tx.QueryRow(`SELECT id FROM tournament WHERE name = ?`, name).Scan(&tournamentID)
	if err != nil {
		// Create new tournament
		res, err2 := tx.Exec(`INSERT INTO tournament (name, date, location) VALUES (?, '', '')`, name)
		if err2 != nil {
			return err2
		}
		tournamentID, err2 = res.LastInsertId()
		if err2 != nil {
			return err2
		}
	}

	_, err = tx.Exec(`UPDATE match SET tournament_id = ? WHERE id = ?`, tournamentID, matchID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`UPDATE tournament SET updated_at = datetime('now') WHERE id = ?`, tournamentID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// GetTournamentMatches returns all matches in a tournament
func (d *Database) GetTournamentMatches(tournamentID int64) ([]Match, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	rows, err := d.db.Query(`
		SELECT 
			id, player1_name, player2_name, event, location, round, 
			match_length, match_date, import_date, file_path, game_count, tournament_id,
			COALESCE(last_visited_position, -1) as last_visited_position,
			COALESCE(comment, '') as comment,
			COALESCE(tournament_sort_order, 0) as tournament_sort_order
		FROM match 
		WHERE tournament_id = ?
		ORDER BY tournament_sort_order ASC, match_date DESC
	`, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []Match
	for rows.Next() {
		var m Match
		var tournamentID sql.NullInt64
		err := rows.Scan(&m.ID, &m.Player1Name, &m.Player2Name, &m.Event, &m.Location, &m.Round,
			&m.MatchLength, &m.MatchDate, &m.ImportDate, &m.FilePath, &m.GameCount, &tournamentID, &m.LastVisitedPosition,
			&m.Comment, &m.TournamentSortOrder)
		if err != nil {
			// tournamentID here would be the loop's own (still zero-value,
			// unscanned) sql.NullInt64, not the tournament this query is
			// scoped to (the function parameter of the same name) — the log
			// line only names the query, not that value, to avoid printing a
			// misleading id.
			slog.Warn("scanning match for GetTournamentMatches", "err", err)
			continue
		}
		if tournamentID.Valid {
			tid := tournamentID.Int64
			m.TournamentID = &tid
		}
		matches = append(matches, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := d.applyMatchBadges(matches); err != nil {
		// Non-fatal: log but continue without stats
		_ = err
	}

	return matches, nil
}

// GetMatchTournament returns the tournament a match belongs to (if any)
func (d *Database) GetMatchTournament(matchID int64) (*Tournament, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	var tournamentID sql.NullInt64
	err := d.db.QueryRow(`SELECT tournament_id FROM match WHERE id = ?`, matchID).Scan(&tournamentID)
	if err != nil {
		return nil, err
	}

	if !tournamentID.Valid {
		return nil, nil // Match is not in any tournament
	}

	var t Tournament
	err = d.db.QueryRow(`
		SELECT id, name, COALESCE(date, ''), COALESCE(location, ''), sort_order, created_at, updated_at
		FROM tournament WHERE id = ?
	`, tournamentID.Int64).Scan(&t.ID, &t.Name, &t.Date, &t.Location, &t.SortOrder, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &t, nil
}

// ExportTournaments exports specific tournaments and their matches to a
// database file. watermark and watermarkNote mirror ExportDatabase's
// Watermark/WatermarkNote: an empty watermark means the export carries none.
// See db_export_position.go for the schema and position-writing code shared
// with the other two export paths.
func (d *Database) ExportTournaments(exportPath string, tournamentIDs []int64, metadata map[string]string, includeAnalysis bool, includeComments bool, watermark string, watermarkNote string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	// Collect all match IDs from selected tournaments
	var matchIDs []int64
	for _, tournamentID := range tournamentIDs {
		rows, err := d.db.Query(`SELECT id FROM match WHERE tournament_id = ?`, tournamentID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var matchID int64
			if err := rows.Scan(&matchID); err != nil {
				slog.Warn("scanning match id for tournament export", "tournamentID", tournamentID, "err", err)
				continue
			}
			matchIDs = append(matchIDs, matchID)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		rows.Close()
	}

	// Collect all unique position IDs from matches
	positionIDsMap := make(map[int64]bool)
	for _, matchID := range matchIDs {
		rows, err := d.db.Query(`
			SELECT DISTINCT m.position_id
			FROM move m
			JOIN game g ON m.game_id = g.id
			WHERE g.match_id = ? AND m.position_id IS NOT NULL
		`, matchID)
		if err != nil {
			slog.Warn("querying positions for tournament export", "matchID", matchID, "err", err)
			continue
		}
		for rows.Next() {
			var posID int64
			if err := rows.Scan(&posID); err != nil {
				slog.Warn("scanning position id for tournament export", "matchID", matchID, "err", err)
				continue
			}
			positionIDsMap[posID] = true
		}
		if err := rows.Err(); err != nil {
			return err
		}
		rows.Close()
	}

	// Convert map to slice
	var positionIDs []int64
	for id := range positionIDsMap {
		positionIDs = append(positionIDs, id)
	}

	exportDB, err := newExportDB(exportPath)
	if err != nil {
		return err
	}
	defer exportDB.Close()

	if err := writeExportMetadata(exportDB, metadata, watermark, watermarkNote); err != nil {
		return err
	}

	// Read every position and (if requested) its analysis/comment in one batched
	// statement each, instead of the N+1 per-position SELECTs this used to run —
	// the same helpers ExportDatabase uses (db_export.go).
	positions, err := d.positionsByIDsLocked(positionIDs)
	if err != nil {
		return fmt.Errorf("cannot read the positions to export: %w", err)
	}
	var analysisByPosition map[int64][]byte
	if includeAnalysis {
		if analysisByPosition, err = d.analysisForPositions(positionIDs); err != nil {
			return fmt.Errorf("cannot read analyses to export: %w", err)
		}
	}
	var commentByPosition map[int64]string
	if includeComments {
		if commentByPosition, err = d.commentsForPositions(positionIDs); err != nil {
			return fmt.Errorf("cannot read comments to export: %w", err)
		}
	}

	// The whole write phase — positions, tournaments, matches, games, moves,
	// move analyses — runs inside one transaction instead of autocommitting each
	// statement, exactly like ExportDatabase's second phase (db_export.go):
	// exportDB is pinned to a single connection (newExportDB), so a plain
	// BEGIN/COMMIT groups every exportDB.Exec below without touching each call
	// site.
	if _, err = exportDB.Exec(`BEGIN`); err != nil {
		return fmt.Errorf("cannot start the export transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = exportDB.Exec(`ROLLBACK`)
		}
	}()

	insertPosition, err := exportDB.Prepare(exportPositionInsertSQL)
	if err != nil {
		return fmt.Errorf("cannot prepare the position insert: %w", err)
	}
	defer insertPosition.Close()
	lookupPosition, err := exportDB.Prepare(exportPositionLookupSQL)
	if err != nil {
		return fmt.Errorf("cannot prepare the position lookup: %w", err)
	}
	defer lookupPosition.Close()

	// Export positions
	oldToNewID := make(map[int64]int64, len(positions))
	for _, pos := range positions {
		posID := pos.ID
		newID, insErr := insertExportPosition(insertPosition, lookupPosition, pos)
		if insErr != nil {
			slog.Warn("inserting position into tournament export database", "positionID", posID, "err", insErr)
			continue
		}
		oldToNewID[posID] = newID

		// Export analysis if requested
		if includeAnalysis {
			if data, ok := analysisByPosition[posID]; ok {
				// Decompress for export compatibility
				jsonData, decErr := decompressAnalysisData(data)
				if decErr != nil {
					slog.Warn("decompressing analysis for tournament export", "positionID", posID, "err", decErr)
				} else {
					_, _ = exportDB.Exec(`INSERT INTO analysis (position_id, data) VALUES (?, ?)`, newID, string(jsonData))
				}
			}
		}

		// Export comments if requested
		if includeComments {
			if text := commentByPosition[posID]; text != "" {
				_, _ = exportDB.Exec(`INSERT INTO comment (position_id, text) VALUES (?, ?)`, newID, text)
			}
		}
	}

	// Export tournaments
	tournamentIDMapping := make(map[int64]int64)
	for _, tournamentID := range tournamentIDs {
		var name, date, location string
		var sortOrder int
		var createdAt, updatedAt string
		err := d.db.QueryRow(`SELECT name, COALESCE(date, ''), COALESCE(location, ''), sort_order, created_at, updated_at FROM tournament WHERE id = ?`, tournamentID).
			Scan(&name, &date, &location, &sortOrder, &createdAt, &updatedAt)
		if err != nil {
			slog.Warn("reading tournament for export", "tournamentID", tournamentID, "err", err)
			continue
		}

		result, err := exportDB.Exec(`INSERT INTO tournament (name, date, location, sort_order, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
			name, date, location, sortOrder, createdAt, updatedAt)
		if err != nil {
			slog.Warn("inserting tournament into export database", "tournamentID", tournamentID, "err", err)
			continue
		}
		newTournamentID, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get last insert ID: %w", err)
		}
		tournamentIDMapping[tournamentID] = newTournamentID
	}

	// Export matches and their games/moves
	matchIDMapping := make(map[int64]int64)
	for _, matchID := range matchIDs {
		var player1Name, player2Name, event, location, round, filePath string
		var matchLength, gameCount int
		var matchDate, importDate string
		var matchHash sql.NullString
		var srcTournamentID sql.NullInt64

		err := d.db.QueryRow(`
			SELECT player1_name, player2_name, event, location, round, match_length, 
			       match_date, import_date, file_path, game_count, match_hash, tournament_id 
			FROM match WHERE id = ?`, matchID).
			Scan(&player1Name, &player2Name, &event, &location, &round, &matchLength,
				&matchDate, &importDate, &filePath, &gameCount, &matchHash, &srcTournamentID)
		if err != nil {
			slog.Warn("reading match for tournament export", "matchID", matchID, "err", err)
			continue
		}

		var newTournamentID sql.NullInt64
		if srcTournamentID.Valid {
			if newID, ok := tournamentIDMapping[srcTournamentID.Int64]; ok {
				newTournamentID = sql.NullInt64{Int64: newID, Valid: true}
			}
		}

		result, err := exportDB.Exec(`
			INSERT INTO match (player1_name, player2_name, event, location, round, match_length, 
			                   match_date, import_date, file_path, game_count, match_hash, tournament_id)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			player1Name, player2Name, event, location, round, matchLength,
			matchDate, importDate, filePath, gameCount, matchHash, newTournamentID)
		if err != nil {
			slog.Warn("inserting match into tournament export database", "matchID", matchID, "err", err)
			continue
		}
		newMatchID, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get last insert ID: %w", err)
		}
		matchIDMapping[matchID] = newMatchID

		// Export games
		gameRows, err := d.db.Query(`SELECT id, game_number, initial_score_1, initial_score_2, winner, points_won, move_count FROM game WHERE match_id = ?`, matchID)
		if err != nil {
			slog.Warn("querying games for tournament export", "matchID", matchID, "err", err)
			continue
		}

		gameIDMapping := make(map[int64]int64)
		for gameRows.Next() {
			var gameID int64
			var gameNumber, initialScore1, initialScore2, winner, pointsWon, moveCount int
			if err := gameRows.Scan(&gameID, &gameNumber, &initialScore1, &initialScore2, &winner, &pointsWon, &moveCount); err != nil {
				slog.Warn("scanning game for tournament export", "matchID", matchID, "err", err)
				continue
			}

			result, err := exportDB.Exec(`INSERT INTO game (match_id, game_number, initial_score_1, initial_score_2, winner, points_won, move_count) VALUES (?, ?, ?, ?, ?, ?, ?)`,
				newMatchID, gameNumber, initialScore1, initialScore2, winner, pointsWon, moveCount)
			if err != nil {
				slog.Warn("inserting game for tournament export", "matchID", matchID, "err", err)
				continue
			}
			newGameID, err := result.LastInsertId()
			if err != nil {
				return fmt.Errorf("failed to get last insert ID: %w", err)
			}
			gameIDMapping[gameID] = newGameID
		}
		if err := gameRows.Err(); err != nil {
			return err
		}
		gameRows.Close()

		// Export moves for each game
		for oldGameID, newGameID := range gameIDMapping {
			moveRows, err := d.db.Query(`SELECT id, move_number, move_type, position_id, player, dice_1, dice_2, checker_move, cube_action, luck_mp FROM move WHERE game_id = ?`, oldGameID)
			if err != nil {
				slog.Warn("querying moves for tournament export", "gameID", oldGameID, "err", err)
				continue
			}

			for moveRows.Next() {
				var moveID int64
				var moveNumber, player, dice1, dice2 int
				var moveType, checkerMove, cubeAction string
				var oldPosID sql.NullInt64
				var luckMP sql.NullInt32
				if err := moveRows.Scan(&moveID, &moveNumber, &moveType, &oldPosID, &player, &dice1, &dice2, &checkerMove, &cubeAction, &luckMP); err != nil {
					slog.Warn("scanning move for tournament export", "gameID", oldGameID, "err", err)
					continue
				}

				var newPosID sql.NullInt64
				if oldPosID.Valid {
					if newID, ok := oldToNewID[oldPosID.Int64]; ok {
						newPosID = sql.NullInt64{Int64: newID, Valid: true}
					}
				}

				// NULL travels as NULL: an unknown luck must not land in the
				// exported database as a neutral roll.
				var luckValue any
				if luckMP.Valid {
					luckValue = luckMP.Int32
				}

				result, err := exportDB.Exec(`INSERT INTO move (game_id, move_number, move_type, position_id, player, dice_1, dice_2, checker_move, cube_action, luck_mp) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					newGameID, moveNumber, moveType, newPosID, player, dice1, dice2, checkerMove, cubeAction, luckValue)
				if err != nil {
					slog.Warn("inserting move for tournament export", "moveID", moveID, "err", err)
					continue
				}
				newMoveID, err := result.LastInsertId()
				if err != nil {
					return fmt.Errorf("failed to get last insert ID: %w", err)
				}

				// Export move_analysis
				analysisRows, err := d.db.Query(`SELECT analysis_type, depth, equity, equity_error, win_rate, gammon_rate, backgammon_rate, opponent_win_rate, opponent_gammon_rate, opponent_backgammon_rate FROM move_analysis WHERE move_id = ?`, moveID)
				if err != nil {
					slog.Warn("querying move analysis for tournament export", "moveID", moveID, "err", err)
					continue
				}
				for analysisRows.Next() {
					var analysisType, depth string
					var equity, equityError, winRate, gammonRate, bgRate, oppWinRate, oppGammonRate, oppBgRate float64
					if err := analysisRows.Scan(&analysisType, &depth, &equity, &equityError, &winRate, &gammonRate, &bgRate, &oppWinRate, &oppGammonRate, &oppBgRate); err != nil {
						slog.Warn("scanning move analysis for tournament export", "moveID", moveID, "err", err)
						continue
					}
					_, _ = exportDB.Exec(`INSERT INTO move_analysis (move_id, analysis_type, depth, equity, equity_error, win_rate, gammon_rate, backgammon_rate, opponent_win_rate, opponent_gammon_rate, opponent_backgammon_rate) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
						newMoveID, analysisType, depth, equity, equityError, winRate, gammonRate, bgRate, oppWinRate, oppGammonRate, oppBgRate)
				}
				if err := analysisRows.Err(); err != nil {
					return err
				}
				analysisRows.Close()
			}
			if err := moveRows.Err(); err != nil {
				return err
			}
			moveRows.Close()
		}
	}

	if _, err = exportDB.Exec(`COMMIT`); err != nil {
		return fmt.Errorf("cannot finish the export transaction: %w", err)
	}
	committed = true

	return nil
}
