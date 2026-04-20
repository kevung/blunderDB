package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"
	"time"
)

// ExportDatabase creates a new database file containing the current selection of positions
// with their analysis and comments
func (d *Database) ExportDatabase(opts ExportOptions) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check that the current database is open
	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	// Delete the export file if it already exists
	if _, err := os.Stat(opts.ExportPath); err == nil {
		// File exists, remove it
		if err := os.Remove(opts.ExportPath); err != nil {
			return fmt.Errorf("cannot remove existing export file: %v", err)
		}
		slog.Debug("removed existing export file", "path", opts.ExportPath)
	}

	// Create a new database for export
	exportDB, err := sql.Open("sqlite", opts.ExportPath)
	if err != nil {
		return err
	}
	defer exportDB.Close()

	// Create the schema for the export database
	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS position (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			state TEXT
		)
	`)
	if err != nil {
		return err
	}

	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS analysis (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			position_id INTEGER UNIQUE,
			data JSON,
			FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS comment (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			position_id INTEGER UNIQUE,
			text TEXT,
			FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS metadata (
			key TEXT PRIMARY KEY,
			value TEXT
		)
	`)
	if err != nil {
		return err
	}

	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS command_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			command TEXT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS filter_library (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			command TEXT,
			edit_position TEXT
		)
	`)
	if err != nil {
		return err
	}

	// Create search_history table (required for version >= 1.3.0)
	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS search_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			command TEXT,
			position TEXT,
			timestamp INTEGER
		)
	`)
	if err != nil {
		return err
	}

	// Create match-related tables (required for version >= 1.4.0)
	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS match (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			player1_name TEXT,
			player2_name TEXT,
			event TEXT,
			location TEXT,
			round TEXT,
			match_length INTEGER,
			match_date DATETIME,
			import_date DATETIME DEFAULT CURRENT_TIMESTAMP,
			file_path TEXT,
			game_count INTEGER DEFAULT 0,
			match_hash TEXT,
			tournament_id INTEGER REFERENCES tournament(id) ON DELETE SET NULL,
			last_visited_position INTEGER DEFAULT -1
		)
	`)
	if err != nil {
		return err
	}

	// Create tournament table
	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS tournament (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			date TEXT,
			location TEXT,
			sort_order INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	// Create collection tables (required for version >= 1.5.0)
	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS collection (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT,
			sort_order INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS collection_position (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			collection_id INTEGER NOT NULL,
			position_id INTEGER NOT NULL,
			sort_order INTEGER DEFAULT 0,
			added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(collection_id) REFERENCES collection(id) ON DELETE CASCADE,
			FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE,
			UNIQUE(collection_id, position_id)
		)
	`)
	if err != nil {
		return err
	}

	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS game (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			match_id INTEGER,
			game_number INTEGER,
			initial_score_1 INTEGER,
			initial_score_2 INTEGER,
			winner INTEGER,
			points_won INTEGER,
			move_count INTEGER DEFAULT 0,
			FOREIGN KEY(match_id) REFERENCES match(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS move (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			game_id INTEGER,
			move_number INTEGER,
			move_type TEXT,
			position_id INTEGER,
			player INTEGER,
			dice_1 INTEGER,
			dice_2 INTEGER,
			checker_move TEXT,
			cube_action TEXT,
			FOREIGN KEY(game_id) REFERENCES game(id) ON DELETE CASCADE,
			FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE SET NULL
		)
	`)
	if err != nil {
		return err
	}

	_, err = exportDB.Exec(`
		CREATE TABLE IF NOT EXISTS move_analysis (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			move_id INTEGER,
			analysis_type TEXT,
			depth TEXT,
			equity INTEGER,
			equity_error INTEGER,
			win_rate INTEGER,
			gammon_rate INTEGER,
			backgammon_rate INTEGER,
			opponent_win_rate INTEGER,
			opponent_gammon_rate INTEGER,
			opponent_backgammon_rate INTEGER,
			FOREIGN KEY(move_id) REFERENCES move(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	// Insert database version
	_, err = exportDB.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES ('database_version', ?)`, DatabaseVersion)
	if err != nil {
		return err
	}

	// Insert metadata (user, description, dateOfCreation)
	for key, value := range opts.Metadata {
		if value != "" {
			_, err = exportDB.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES (?, ?)`, key, value)
			if err != nil {
				slog.Warn("inserting metadata in export database", "key", key, "err", err)
			}
		}
	}

	// If dateOfCreation is not provided, set it to current date
	if opts.Metadata["dateOfCreation"] == "" {
		currentDate := time.Now().Format("2006-01-02")
		_, err = exportDB.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES ('dateOfCreation', ?)`, currentDate)
		if err != nil {
			slog.Warn("inserting default creation date in export database", "err", err)
		}
	}

	// Begin transaction for export
	tx, err := exportDB.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			slog.Warn("transaction rolled back due to error during export")
		}
	}()

	// Export all positions with their analysis and comments
	idMapping := make(map[int64]int64) // map old position ID to new position ID

	for _, position := range opts.Positions {
		oldPositionID := position.ID

		// Reset the ID for the new database
		position.ID = 0

		// Marshal the full position (export uses full JSON for backward compatibility)
		positionJSON := fullPositionJSON(position)

		// Insert the position into the export database
		result, err := tx.Exec(`INSERT INTO position (state) VALUES (?)`, positionJSON)
		if err != nil {
			slog.Warn("inserting position into export database", "positionID", oldPositionID, "err", err)
			continue
		}

		newPositionID, err := result.LastInsertId()
		if err != nil {
			slog.Warn("getting last insert ID for position", "positionID", oldPositionID, "err", err)
			continue
		}

		// Store the ID mapping
		idMapping[oldPositionID] = newPositionID

		// Export analysis if it exists and if includeAnalysis is true
		if opts.IncludeAnalysis {
			var analysisData []byte
			analysisErr := d.db.QueryRow(`SELECT data FROM analysis WHERE position_id = ?`, oldPositionID).Scan(&analysisData)
			if analysisErr == nil {
				// Update position_id in the analysis JSON
				analysis, unmarshalErr := decodeAnalysisFromStorage(analysisData)
				if unmarshalErr == nil {
					analysis.PositionID = int(newPositionID)

					// Handle played moves
					if opts.IncludePlayedMoves {
						// Load played moves from the move table and merge with existing
						moveRows, moveErr := d.db.Query(`
							SELECT checker_move, cube_action 
							FROM move 
							WHERE position_id = ?
						`, oldPositionID)

						if moveErr == nil {

							// Collect all moves from the database
							existingMoves := make(map[string]bool)
							existingCubeActions := make(map[string]bool)

							// Include existing PlayedMoves from analysis JSON
							for _, m := range analysis.PlayedMoves {
								if m != "" {
									existingMoves[normalizeMove(m)] = true
								}
							}
							if analysis.PlayedMove != "" {
								existingMoves[normalizeMove(analysis.PlayedMove)] = true
							}

							// Include existing PlayedCubeActions from analysis JSON
							for _, a := range analysis.PlayedCubeActions {
								if a != "" {
									existingCubeActions[a] = true
								}
							}
							if analysis.PlayedCubeAction != "" {
								existingCubeActions[analysis.PlayedCubeAction] = true
							}

							// Add moves from move table
							for moveRows.Next() {
								var checkerMove sql.NullString
								var cubeAction sql.NullString
								if scanErr := moveRows.Scan(&checkerMove, &cubeAction); scanErr == nil {
									if checkerMove.Valid && checkerMove.String != "" {
										existingMoves[normalizeMove(checkerMove.String)] = true
									}
									if cubeAction.Valid && cubeAction.String != "" {
										existingCubeActions[cubeAction.String] = true
									}
								}
							}
							if err := moveRows.Err(); err != nil {
								return err
							}
							moveRows.Close()

							// Convert to slices
							analysis.PlayedMoves = make([]string, 0, len(existingMoves))
							for m := range existingMoves {
								analysis.PlayedMoves = append(analysis.PlayedMoves, m)
							}
							sort.Strings(analysis.PlayedMoves)

							analysis.PlayedCubeActions = make([]string, 0, len(existingCubeActions))
							for a := range existingCubeActions {
								analysis.PlayedCubeActions = append(analysis.PlayedCubeActions, a)
							}
							sort.Strings(analysis.PlayedCubeActions)
						}
					} else {
						// Clear played move fields if IncludePlayedMoves is false
						analysis.PlayedMove = ""
						analysis.PlayedCubeAction = ""
						analysis.PlayedMoves = nil
						analysis.PlayedCubeActions = nil
					}

					updatedAnalysisJSON, err := json.Marshal(analysis)
					if err != nil {
						return fmt.Errorf("failed to marshal JSON: %w", err)
					}

					if _, insertErr := tx.Exec(`INSERT INTO analysis (position_id, data) VALUES (?, ?)`, newPositionID, string(updatedAnalysisJSON)); insertErr != nil {
						slog.Warn("inserting analysis for position", "newID", newPositionID, "oldID", oldPositionID, "err", insertErr)
					}
				}
			} else if analysisErr != sql.ErrNoRows {
				slog.Warn("querying analysis for position", "positionID", oldPositionID, "err", analysisErr)
			}
		}

		// Export comment if it exists and if includeComments is true
		if opts.IncludeComments {
			var comment string
			commentErr := d.db.QueryRow(`SELECT text FROM comment WHERE position_id = ?`, oldPositionID).Scan(&comment)
			if commentErr == nil && comment != "" {
				if _, insertErr := tx.Exec(`INSERT INTO comment (position_id, text) VALUES (?, ?)`, newPositionID, comment); insertErr != nil {
					slog.Warn("inserting comment for position", "newID", newPositionID, "oldID", oldPositionID, "err", insertErr)
				}
			} else if commentErr != nil && commentErr != sql.ErrNoRows {
				slog.Warn("querying comment for position", "positionID", oldPositionID, "err", commentErr)
			}
		}
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		return err
	}

	// Export filter library if includeFilterLibrary is true
	if opts.IncludeFilterLibrary {
		rows, err := d.db.Query(`SELECT name, command, COALESCE(edit_position, '') FROM filter_library`)
		if err == nil {
			defer rows.Close()

			for rows.Next() {
				var name, command, editPosition string
				err := rows.Scan(&name, &command, &editPosition)
				if err == nil {
					_, err = exportDB.Exec(`INSERT INTO filter_library (name, command, edit_position) VALUES (?, ?, ?)`, name, command, editPosition)
					if err != nil {
						slog.Warn("inserting filter library entry", "name", name, "err", err)
					}
				}
			}
			if err := rows.Err(); err != nil {
				return err
			}
		}
	}

	// Export matches if includeMatches is true
	matchIDMapping := make(map[int64]int64) // old match ID -> new match ID (accessible for tournament linking)
	if opts.IncludeMatches {
		matchCount := 0
		gameCount := 0
		moveCount := 0
		moveAnalysisCount := 0

		// Get matches - filter by matchIDs if provided, otherwise get all
		var matchRows *sql.Rows
		if len(opts.MatchIDs) > 0 {
			// Build IN clause for specific match IDs
			placeholders := make([]string, len(opts.MatchIDs))
			args := make([]interface{}, len(opts.MatchIDs))
			for i, id := range opts.MatchIDs {
				placeholders[i] = "?"
				args[i] = id
			}
			query := fmt.Sprintf(`
				SELECT id, player1_name, player2_name, event, location, round,
				       match_length, match_date, import_date, file_path, game_count, match_hash, tournament_id
				FROM match
				WHERE id IN (%s)
			`, strings.Join(placeholders, ","))
			matchRows, err = d.db.Query(query, args...)
		} else {
			matchRows, err = d.db.Query(`
				SELECT id, player1_name, player2_name, event, location, round,
				       match_length, match_date, import_date, file_path, game_count, match_hash, tournament_id
				FROM match
			`)
		}
		if err == nil {
			defer matchRows.Close()

			for matchRows.Next() {
				var oldMatchID int64
				var player1Name, player2Name, event, location, round, filePath string
				var matchLength int32
				var matchDate, importDate time.Time
				var gameCountVal int
				var matchHash sql.NullString
				var tournamentID sql.NullInt64

				err := matchRows.Scan(&oldMatchID, &player1Name, &player2Name, &event, &location, &round,
					&matchLength, &matchDate, &importDate, &filePath, &gameCountVal, &matchHash, &tournamentID)
				if err != nil {
					slog.Warn("scanning match", "err", err)
					continue
				}

				// Insert match into export database
				var result sql.Result
				if matchHash.Valid {
					result, err = exportDB.Exec(`
						INSERT INTO match (player1_name, player2_name, event, location, round,
						                   match_length, match_date, import_date, file_path, game_count, match_hash)
						VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
					`, player1Name, player2Name, event, location, round,
						matchLength, matchDate, importDate, filePath, gameCountVal, matchHash.String)
				} else {
					result, err = exportDB.Exec(`
						INSERT INTO match (player1_name, player2_name, event, location, round,
						                   match_length, match_date, import_date, file_path, game_count)
						VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
					`, player1Name, player2Name, event, location, round,
						matchLength, matchDate, importDate, filePath, gameCountVal)
				}
				if err != nil {
					slog.Warn("inserting match", "err", err)
					continue
				}

				newMatchID, err := result.LastInsertId()
				if err != nil {
					slog.Warn("getting new match ID", "err", err)
					continue
				}
				matchIDMapping[oldMatchID] = newMatchID
				matchCount++
			}
			if err := matchRows.Err(); err != nil {
				return err
			}

			// Export games for each match
			gameIDMapping := make(map[int64]int64) // old game ID -> new game ID

			for oldMatchID, newMatchID := range matchIDMapping {
				gameRows, err := d.db.Query(`
					SELECT id, game_number, initial_score_1, initial_score_2, winner, points_won, move_count
					FROM game
					WHERE match_id = ?
				`, oldMatchID)
				if err != nil {
					slog.Warn("querying games for match", "matchID", oldMatchID, "err", err)
					continue
				}

				for gameRows.Next() {
					var oldGameID int64
					var gameNumber, score1, score2, winner, pointsWon int32
					var moveCountVal int

					err := gameRows.Scan(&oldGameID, &gameNumber, &score1, &score2, &winner, &pointsWon, &moveCountVal)
					if err != nil {
						slog.Warn("scanning game", "err", err)
						continue
					}

					result, err := exportDB.Exec(`
						INSERT INTO game (match_id, game_number, initial_score_1, initial_score_2, winner, points_won, move_count)
						VALUES (?, ?, ?, ?, ?, ?, ?)
					`, newMatchID, gameNumber, score1, score2, winner, pointsWon, moveCountVal)
					if err != nil {
						slog.Warn("inserting game", "err", err)
						continue
					}

					newGameID, err := result.LastInsertId()
					if err != nil {
						slog.Warn("getting new game ID", "err", err)
						continue
					}
					gameIDMapping[oldGameID] = newGameID
					gameCount++
				}
				if err := gameRows.Err(); err != nil {
					return err
				}
				gameRows.Close()
			}

			// Export moves for each game
			moveIDMapping := make(map[int64]int64) // old move ID -> new move ID

			for oldGameID, newGameID := range gameIDMapping {
				moveRows, err := d.db.Query(`
					SELECT id, move_number, move_type, position_id, player, dice_1, dice_2, checker_move, cube_action
					FROM move
					WHERE game_id = ?
				`, oldGameID)
				if err != nil {
					slog.Warn("querying moves for game", "gameID", oldGameID, "err", err)
					continue
				}

				for moveRows.Next() {
					var oldMoveID, positionID int64
					var moveNumber, player, dice1, dice2 int32
					var moveType string
					var checkerMove, cubeAction sql.NullString

					err := moveRows.Scan(&oldMoveID, &moveNumber, &moveType, &positionID, &player, &dice1, &dice2, &checkerMove, &cubeAction)
					if err != nil {
						slog.Warn("scanning move", "err", err)
						continue
					}

					// Map the position ID to the new database
					newPositionID, posExists := idMapping[positionID]
					if !posExists {
						// Position might not have been exported (not in the selection)
						// Still export the move but with null position_id
						newPositionID = 0
					}

					var result sql.Result
					if newPositionID > 0 {
						result, err = exportDB.Exec(`
							INSERT INTO move (game_id, move_number, move_type, position_id, player, dice_1, dice_2, checker_move, cube_action)
							VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
						`, newGameID, moveNumber, moveType, newPositionID, player, dice1, dice2,
							checkerMove.String, cubeAction.String)
					} else {
						result, err = exportDB.Exec(`
							INSERT INTO move (game_id, move_number, move_type, position_id, player, dice_1, dice_2, checker_move, cube_action)
							VALUES (?, ?, ?, NULL, ?, ?, ?, ?, ?)
						`, newGameID, moveNumber, moveType, player, dice1, dice2,
							checkerMove.String, cubeAction.String)
					}
					if err != nil {
						slog.Warn("inserting move", "err", err)
						continue
					}

					newMoveID, err := result.LastInsertId()
					if err != nil {
						slog.Warn("getting new move ID", "err", err)
						continue
					}
					moveIDMapping[oldMoveID] = newMoveID
					moveCount++
				}
				if err := moveRows.Err(); err != nil {
					return err
				}
				moveRows.Close()
			}

			// Export move analysis for each move
			for oldMoveID, newMoveID := range moveIDMapping {
				analysisRows, err := d.db.Query(`
					SELECT analysis_type, depth, equity, equity_error, win_rate, gammon_rate, backgammon_rate,
					       opponent_win_rate, opponent_gammon_rate, opponent_backgammon_rate
					FROM move_analysis
					WHERE move_id = ?
				`, oldMoveID)
				if err != nil {
					continue
				}

				for analysisRows.Next() {
					var analysisType, depth string
					var equity, equityError, winRate, gammonRate, backgammonRate float64
					var oppWinRate, oppGammonRate, oppBackgammonRate float64

					err := analysisRows.Scan(&analysisType, &depth, &equity, &equityError, &winRate, &gammonRate, &backgammonRate,
						&oppWinRate, &oppGammonRate, &oppBackgammonRate)
					if err != nil {
						continue
					}

					_, err = exportDB.Exec(`
						INSERT INTO move_analysis (move_id, analysis_type, depth, equity, equity_error, win_rate, gammon_rate, backgammon_rate,
						                           opponent_win_rate, opponent_gammon_rate, opponent_backgammon_rate)
						VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
					`, newMoveID, analysisType, depth, equity, equityError, winRate, gammonRate, backgammonRate,
						oppWinRate, oppGammonRate, oppBackgammonRate)
					if err != nil {
						slog.Warn("inserting move analysis", "err", err)
						continue
					}
					moveAnalysisCount++
				}
				if err := analysisRows.Err(); err != nil {
					return err
				}
				analysisRows.Close()
			}
		}

		slog.Info("exported matches", "matches", matchCount, "games", gameCount, "moves", moveCount, "moveAnalyses", moveAnalysisCount)
	}

	// Export collections if requested
	if opts.IncludeCollections && len(opts.CollectionIDs) > 0 {
		collectionCount := 0
		collectionPosCount := 0

		for _, collectionID := range opts.CollectionIDs {
			var name, description string
			var sortOrder int
			var createdAt, updatedAt string
			err := d.db.QueryRow(`SELECT name, COALESCE(description, ''), sort_order, COALESCE(strftime('%Y-%m-%d %H:%M:%S', created_at), ''), COALESCE(strftime('%Y-%m-%d %H:%M:%S', updated_at), '') FROM collection WHERE id = ?`, collectionID).
				Scan(&name, &description, &sortOrder, &createdAt, &updatedAt)
			if err != nil {
				slog.Warn("reading collection", "collectionID", collectionID, "err", err)
				continue
			}

			result, err := exportDB.Exec(`INSERT INTO collection (name, description, sort_order, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
				name, description, sortOrder, createdAt, updatedAt)
			if err != nil {
				slog.Warn("inserting collection", "collectionID", collectionID, "err", err)
				continue
			}
			newCollectionID, err := result.LastInsertId()
			if err != nil {
				return fmt.Errorf("failed to get last insert ID: %w", err)
			}
			collectionCount++

			// Export collection_position mappings
			cpRows, err := d.db.Query(`SELECT position_id, sort_order, added_at FROM collection_position WHERE collection_id = ?`, collectionID)
			if err != nil {
				slog.Warn("querying collection_position", "collectionID", collectionID, "err", err)
				continue
			}
			for cpRows.Next() {
				var oldPosID int64
				var cpSortOrder int
				var addedAt string
				if err := cpRows.Scan(&oldPosID, &cpSortOrder, &addedAt); err != nil {
					continue
				}
				if newPosID, ok := idMapping[oldPosID]; ok {
					_, _ = exportDB.Exec(`INSERT INTO collection_position (collection_id, position_id, sort_order, added_at) VALUES (?, ?, ?, ?)`,
						newCollectionID, newPosID, cpSortOrder, addedAt)
					collectionPosCount++
				}
			}
			if err := cpRows.Err(); err != nil {
				return err
			}
			cpRows.Close()
		}

		slog.Info("exported collections", "collections", collectionCount, "positionMappings", collectionPosCount)
	}

	// Export tournaments if requested
	if len(opts.TournamentIDs) > 0 {
		tournamentCount := 0
		tournamentIDMapping := make(map[int64]int64)

		for _, tournamentID := range opts.TournamentIDs {
			var name string
			var date, location sql.NullString
			var sortOrder int
			var createdAt, updatedAt string
			err := d.db.QueryRow(`SELECT name, date, location, sort_order, created_at, updated_at FROM tournament WHERE id = ?`, tournamentID).
				Scan(&name, &date, &location, &sortOrder, &createdAt, &updatedAt)
			if err != nil {
				slog.Warn("reading tournament", "tournamentID", tournamentID, "err", err)
				continue
			}

			result, err := exportDB.Exec(`INSERT INTO tournament (name, date, location, sort_order, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
				name, date, location, sortOrder, createdAt, updatedAt)
			if err != nil {
				slog.Warn("inserting tournament", "tournamentID", tournamentID, "err", err)
				continue
			}
			newTournamentID, err := result.LastInsertId()
			if err != nil {
				return fmt.Errorf("failed to get last insert ID: %w", err)
			}
			tournamentIDMapping[tournamentID] = newTournamentID
			tournamentCount++
		}

		// Update tournament_id on exported matches that belong to exported tournaments
		if opts.IncludeMatches && len(matchIDMapping) > 0 {
			matchTournamentRows, mterr := d.db.Query(`SELECT id, tournament_id FROM match WHERE tournament_id IS NOT NULL`)
			if mterr == nil {
				for matchTournamentRows.Next() {
					var oldMatchID int64
					var oldTournamentID int64
					if err := matchTournamentRows.Scan(&oldMatchID, &oldTournamentID); err == nil {
						newMatchID, matchExported := matchIDMapping[oldMatchID]
						newTournamentID, tournamentExported := tournamentIDMapping[oldTournamentID]
						if matchExported && tournamentExported {
							_, _ = exportDB.Exec(`UPDATE match SET tournament_id = ? WHERE id = ?`, newTournamentID, newMatchID)
						}
					}
				}
				if err := matchTournamentRows.Err(); err != nil {
					return err
				}
				matchTournamentRows.Close()
			}
		}

		slog.Info("exported tournaments", "count", tournamentCount)
	}

	slog.Info("exported positions", "count", len(opts.Positions), "path", opts.ExportPath)
	return nil
}

// DeleteFile is a helper function to delete a file
func DeleteFile(filePath string) error {
	err := os.Remove(filePath)
	if err != nil {
		return err
	}
	return nil
}
