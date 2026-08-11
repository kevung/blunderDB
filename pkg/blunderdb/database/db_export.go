package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/issuance"
)

// ExportDatabase creates a new database file containing the current selection of positions
// with their analysis and comments
func (d *Database) ExportDatabase(opts ExportOptions) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// skipped counts every row silently dropped below (a scan/insert failure
	// that only aborts that one row, not the whole export — see the
	// individual slog.Warn calls for what and why). Not threaded through the
	// return value: ExportDatabase already has 40+ early-return error paths
	// and three external callers (CLI); a final aggregate warning is a much
	// smaller, equally discoverable surface for something that is, by
	// definition, not fatal to the export.
	skipped := 0

	// Check that the current database is open
	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	// Resolve the selection from identifiers when the caller sent those rather than whole
	// positions. Reading them here costs one query on a database that is already open, and
	// spares the caller from serialising the entire set across the bridge.
	if len(opts.Positions) == 0 && len(opts.PositionIDs) > 0 {
		loaded, err := d.positionsByIDsLocked(opts.PositionIDs)
		if err != nil {
			return fmt.Errorf("cannot read the positions to export: %w", err)
		}
		opts.Positions = loaded
	}

	// Seal the watermark before writing anything: it is the producer's statement about this
	// file, and a failure to sign must not leave a half-made export behind.
	watermarkDocument, err := sealWatermark(strings.TrimSpace(opts.Watermark), strings.TrimSpace(opts.WatermarkNote))
	if err != nil {
		return fmt.Errorf("cannot sign the watermark: %w", err)
	}

	// A password-protected export is built as an ordinary database first, then wrapped. The
	// intermediate file is removed whether or not wrapping succeeds — it is the whole
	// database, unprotected, sitting next to the protected copy.
	finalPath := opts.ExportPath
	if opts.Password != "" {
		// A protected export is named .dbx even when the caller asked for .db: the GUI's
		// save dialog forces a .db name before the user has chosen a password, and a file
		// whose name says "database" but whose contents are encrypted confuses every other
		// tool. ExportedPath reports where it actually landed.
		finalPath = issuance.ProtectedPath(finalPath)
		opts.ExportPath = finalPath + ".plain"
		defer func() {
			if _, statErr := os.Stat(opts.ExportPath); statErr == nil {
				if rmErr := os.Remove(opts.ExportPath); rmErr != nil {
					slog.Warn("removing the intermediate export", "path", opts.ExportPath, "err", rmErr)
				}
			}
		}()
	}

	// Create the export database on the current live schema (storage/sqlite.Bootstrap)
	// instead of a hand-rolled subset — see db_export_position.go. This is what makes
	// zobrist_hash, every scalar column and the search indexes actually exist in the
	// exported file (fiche-04): the old two-column "id, state" position table left them
	// NULL forever, since nothing ever wrote them.
	exportDB, err := newExportDB(opts.ExportPath)
	if err != nil {
		return err
	}
	defer exportDB.Close()

	// Copy metadata by ALLOW-LIST, never by exclusion. An exported file is handed to
	// someone else, and the source database may hold an Issue register listing every other
	// recipient of a course along with the passwords of every distribution. A deny-list
	// would leak whatever document is added six months from now; an allow-list will not.
	// See pkg/blunderdb/issuance and ADR-0007.
	for key, value := range issuance.Carried(opts.Metadata) {
		_, err = exportDB.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES (?, ?)`, key, value)
		if err != nil {
			slog.Warn("inserting metadata in export database", "key", key, "err", err)
		}
	}

	// The watermark, when the producer asked for one. It is written verbatim: the signature
	// is over these exact bytes.
	if watermarkDocument != "" {
		if _, err = exportDB.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES (?, ?)`,
			issuance.KeyWatermark, watermarkDocument); err != nil {
			return fmt.Errorf("cannot write the watermark into the exported file: %w", err)
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

	// Export all positions with their analysis and comments.
	//
	// The loop below used to ask the source database three questions about every single
	// position — its analysis, its played moves, its comment. On a real database that is
	// ~264 000 statements for 88 000 positions, and it dominated the export: 8.3 s of the
	// 14.8 s measured, more than half. The positions are therefore walked in batches, and
	// each batch's analyses, moves and comments are read in one statement each. The body of
	// the loop is unchanged apart from reading those three from the prefetched maps.
	//
	// Batching rather than prefetching everything keeps the memory bounded: analyses are
	// compressed blobs, and a whole database's worth would be hundreds of megabytes.
	idMapping := make(map[int64]int64) // map old position ID to new position ID

	// Prepare the statements the loop runs for every position. Passing SQL text to
	// tx.Exec makes database/sql prepare, execute and close a statement each time: 88 000
	// positions meant parsing the same statements 88 000 times, and it showed up as
	// the largest cost left on the sequential path once the analyses were decoded in
	// parallel.
	insertPosition, err := tx.Prepare(exportPositionInsertSQL)
	if err != nil {
		return fmt.Errorf("cannot prepare the position insert: %w", err)
	}
	defer insertPosition.Close()
	lookupPosition, err := tx.Prepare(exportPositionLookupSQL)
	if err != nil {
		return fmt.Errorf("cannot prepare the position lookup: %w", err)
	}
	defer lookupPosition.Close()
	insertAnalysis, err := tx.Prepare(`INSERT INTO analysis (position_id, data) VALUES (?, ?)`)
	if err != nil {
		return fmt.Errorf("cannot prepare the analysis insert: %w", err)
	}
	defer insertAnalysis.Close()
	insertComment, err := tx.Prepare(`INSERT INTO comment (position_id, text) VALUES (?, ?)`)
	if err != nil {
		return fmt.Errorf("cannot prepare the comment insert: %w", err)
	}
	defer insertComment.Close()

	const prefetchBatch = 1000
	var analysisByPosition map[int64][]byte
	var decodedByPosition map[int64]*PositionAnalysis
	var movesByPosition map[int64][]exportedMove
	var commentByPosition map[int64]string

	for positionIndex, position := range opts.Positions {
		if positionIndex%prefetchBatch == 0 {
			end := min(positionIndex+prefetchBatch, len(opts.Positions))
			batch := make([]int64, 0, end-positionIndex)
			for _, p := range opts.Positions[positionIndex:end] {
				batch = append(batch, p.ID)
			}
			var prefetchErr error
			if opts.IncludeAnalysis {
				if analysisByPosition, prefetchErr = d.analysisForPositions(batch); prefetchErr != nil {
					return fmt.Errorf("cannot read analyses to export: %w", prefetchErr)
				}
				decodedByPosition = decodeAnalysesConcurrently(analysisByPosition)
				if opts.IncludePlayedMoves {
					if movesByPosition, prefetchErr = d.movesForPositions(batch); prefetchErr != nil {
						return fmt.Errorf("cannot read played moves to export: %w", prefetchErr)
					}
				}
			}
			if opts.IncludeComments {
				if commentByPosition, prefetchErr = d.commentsForPositions(batch); prefetchErr != nil {
					return fmt.Errorf("cannot read comments to export: %w", prefetchErr)
				}
			}
		}

		oldPositionID := position.ID

		// Insert the position into the export database in the same compact form, with
		// the same zobrist_hash and scalar columns, that a live SavePosition would
		// produce (see insertExportPosition in db_export_position.go) — carrying
		// provenance so the export round-trips (ADR-0001).
		newPositionID, err := insertExportPosition(insertPosition, lookupPosition, position)
		if err != nil {
			slog.Warn("inserting position into export database", "positionID", oldPositionID, "err", err)
			skipped++
			continue
		}

		// Store the ID mapping
		idMapping[oldPositionID] = newPositionID

		// Export analysis if it exists and if includeAnalysis is true
		if opts.IncludeAnalysis {
			decoded, hasAnalysis := decodedByPosition[oldPositionID]
			if hasAnalysis {
				if decoded != nil {
					analysis := *decoded
					analysis.PositionID = int(newPositionID)

					// Handle played moves
					if opts.IncludePlayedMoves {
						// Load played moves from the move table and merge with existing
						{
							playedMoves := movesByPosition[oldPositionID]

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
							for _, played := range playedMoves {
								{
									checkerMove := played.checkerMove
									cubeAction := played.cubeAction
									if checkerMove.Valid && checkerMove.String != "" {
										existingMoves[normalizeMove(checkerMove.String)] = true
									}
									if cubeAction.Valid && cubeAction.String != "" {
										existingCubeActions[cubeAction.String] = true
									}
								}
							}

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

					if _, insertErr := insertAnalysis.Exec(newPositionID, string(updatedAnalysisJSON)); insertErr != nil {
						slog.Warn("inserting analysis for position", "newID", newPositionID, "oldID", oldPositionID, "err", insertErr)
					}
				}
			}
		}

		// Export comment if it exists and if includeComments is true
		if opts.IncludeComments {
			comment := commentByPosition[oldPositionID]
			if comment != "" {
				if _, insertErr := insertComment.Exec(newPositionID, comment); insertErr != nil {
					slog.Warn("inserting comment for position", "newID", newPositionID, "oldID", oldPositionID, "err", insertErr)
				}
			}
		}
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		return err
	}

	// Everything below writes tens of thousands of rows one statement at a time — matches,
	// games, moves, move analyses, collections, tournaments. Left in autocommit each of them
	// is its own transaction, and that phase alone accounted for 4.6 s of a 9.1 s export.
	// exportDB is pinned to a single connection (see ConfigurePool above), so a plain
	// BEGIN/COMMIT groups the whole tail without touching the hundreds of call sites.
	if _, err = exportDB.Exec(`BEGIN`); err != nil {
		return fmt.Errorf("cannot start the export's second phase: %w", err)
	}
	tailCommitted := false
	defer func() {
		if !tailCommitted {
			_, _ = exportDB.Exec(`ROLLBACK`)
		}
	}()

	// Export filter library if includeFilterLibrary is true
	if opts.IncludeFilterLibrary {
		rows, err := d.db.Query(`SELECT name, command, COALESCE(edit_position, '') FROM filter_library`)
		if err != nil {
			slog.Warn("querying filter library for export", "err", err)
			skipped++
		} else {
			defer rows.Close()

			for rows.Next() {
				var name, command, editPosition string
				if err := rows.Scan(&name, &command, &editPosition); err != nil {
					slog.Warn("scanning filter library entry", "err", err)
					skipped++
					continue
				}
				if _, err := exportDB.Exec(`INSERT INTO filter_library (name, command, edit_position) VALUES (?, ?, ?)`, name, command, editPosition); err != nil {
					slog.Warn("inserting filter library entry", "name", name, "err", err)
					skipped++
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
		if err != nil {
			slog.Warn("querying matches for export", "err", err)
			skipped++
		} else {
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
					skipped++
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
					skipped++
					continue
				}

				newMatchID, err := result.LastInsertId()
				if err != nil {
					slog.Warn("getting new match ID", "err", err)
					skipped++
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
					skipped++
					continue
				}

				for gameRows.Next() {
					var oldGameID int64
					var gameNumber, score1, score2, winner, pointsWon int32
					var moveCountVal int

					err := gameRows.Scan(&oldGameID, &gameNumber, &score1, &score2, &winner, &pointsWon, &moveCountVal)
					if err != nil {
						slog.Warn("scanning game", "err", err)
						skipped++
						continue
					}

					result, err := exportDB.Exec(`
						INSERT INTO game (match_id, game_number, initial_score_1, initial_score_2, winner, points_won, move_count)
						VALUES (?, ?, ?, ?, ?, ?, ?)
					`, newMatchID, gameNumber, score1, score2, winner, pointsWon, moveCountVal)
					if err != nil {
						slog.Warn("inserting game", "err", err)
						skipped++
						continue
					}

					newGameID, err := result.LastInsertId()
					if err != nil {
						slog.Warn("getting new game ID", "err", err)
						skipped++
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
					skipped++
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
						skipped++
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
						// Preparing these two statements once was measured and made no
						// difference (7.08 s against 7.16 s, inside the noise): the driver
						// already caches them. The transaction around this whole phase is
						// what mattered.
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
						skipped++
						continue
					}

					newMoveID, err := result.LastInsertId()
					if err != nil {
						slog.Warn("getting new move ID", "err", err)
						skipped++
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
					slog.Warn("querying move analysis for move", "moveID", oldMoveID, "err", err)
					skipped++
					continue
				}

				for analysisRows.Next() {
					var analysisType, depth string
					var equity, equityError, winRate, gammonRate, backgammonRate float64
					var oppWinRate, oppGammonRate, oppBackgammonRate float64

					err := analysisRows.Scan(&analysisType, &depth, &equity, &equityError, &winRate, &gammonRate, &backgammonRate,
						&oppWinRate, &oppGammonRate, &oppBackgammonRate)
					if err != nil {
						slog.Warn("scanning move analysis", "moveID", oldMoveID, "err", err)
						skipped++
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
						skipped++
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
				skipped++
				continue
			}

			result, err := exportDB.Exec(`INSERT INTO collection (name, description, sort_order, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
				name, description, sortOrder, createdAt, updatedAt)
			if err != nil {
				slog.Warn("inserting collection", "collectionID", collectionID, "err", err)
				skipped++
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
				skipped++
				continue
			}
			for cpRows.Next() {
				var oldPosID int64
				var cpSortOrder int
				var addedAt string
				if err := cpRows.Scan(&oldPosID, &cpSortOrder, &addedAt); err != nil {
					slog.Warn("scanning collection_position", "collectionID", collectionID, "err", err)
					skipped++
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
				skipped++
				continue
			}

			result, err := exportDB.Exec(`INSERT INTO tournament (name, date, location, sort_order, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
				name, date, location, sortOrder, createdAt, updatedAt)
			if err != nil {
				slog.Warn("inserting tournament", "tournamentID", tournamentID, "err", err)
				skipped++
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
			if mterr != nil {
				slog.Warn("querying match/tournament links", "err", mterr)
				skipped++
			} else {
				for matchTournamentRows.Next() {
					var oldMatchID int64
					var oldTournamentID int64
					if err := matchTournamentRows.Scan(&oldMatchID, &oldTournamentID); err != nil {
						slog.Warn("scanning match/tournament link", "err", err)
						skipped++
						continue
					}
					newMatchID, matchExported := matchIDMapping[oldMatchID]
					newTournamentID, tournamentExported := tournamentIDMapping[oldTournamentID]
					if matchExported && tournamentExported {
						_, _ = exportDB.Exec(`UPDATE match SET tournament_id = ? WHERE id = ?`, newTournamentID, newMatchID)
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

	if _, err = exportDB.Exec(`COMMIT`); err != nil {
		return fmt.Errorf("cannot finish the export's second phase: %w", err)
	}
	tailCommitted = true

	slog.Info("exported positions", "count", len(opts.Positions), "path", opts.ExportPath)

	// Wrap the finished database in its encrypted container. exportDB still holds the file
	// open, so close it first: the last pages may not have been flushed, and on Windows a
	// file with an open handle cannot be read wholesale.
	if opts.Password != "" {
		if err := exportDB.Close(); err != nil {
			return fmt.Errorf("cannot finalise the exported database: %w", err)
		}
		env, err := issuance.DecodeEnvelope(watermarkDocument)
		if err != nil {
			return err
		}
		if err := issuance.WrapContainer(opts.ExportPath, finalPath, env, opts.Password); err != nil {
			return err
		}
		slog.Info("protected the exported database", "path", finalPath)
	}
	if skipped > 0 {
		slog.Warn("export completed with some rows skipped", "skipped", skipped, "path", finalPath)
	}
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

// positionsByIDsLocked reads the given positions from the open database, in the order the
// caller listed them, so an export is byte-identical whichever way its selection arrived.
// Unknown identifiers are skipped rather than failing the export: a position deleted between
// the moment the user chose it and the moment they confirmed is not a reason to lose the
// rest.
//
// The caller already holds d.mu, hence the suffix.
func (d *Database) positionsByIDsLocked(ids []int64) ([]Position, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	// One statement, chunked: SQLite's default parameter limit is 32766, and a real
	// selection runs to tens of thousands of positions.
	const chunk = 900
	byID := make(map[int64]Position, len(ids))
	for start := 0; start < len(ids); start += chunk {
		end := min(start+chunk, len(ids))
		batch := ids[start:end]
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(batch)), ",")
		args := make([]any, len(batch))
		for i, id := range batch {
			args[i] = id
		}
		rows, err := d.db.Query(`SELECT `+positionSelectCols+` FROM position WHERE id IN (`+placeholders+`)`, args...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			position, err := scanPositionRow(rows)
			if err != nil {
				rows.Close()
				return nil, err
			}
			byID[position.ID] = position
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()
	}

	out := make([]Position, 0, len(ids))
	for _, id := range ids {
		if position, ok := byID[id]; ok {
			out = append(out, position)
		}
	}
	return out, nil
}

// exportedMove is one row of the `move` table as the export needs it: the played checker
// move and cube action recorded against a position.
type exportedMove struct {
	checkerMove sql.NullString
	cubeAction  sql.NullString
}

// analysisForPositions reads the stored analysis blob of every position in the batch.
func (d *Database) analysisForPositions(ids []int64) (map[int64][]byte, error) {
	out := make(map[int64][]byte, len(ids))
	err := d.forEachInBatch(ids, `SELECT position_id, data FROM analysis WHERE position_id IN `, func(rows *sql.Rows) error {
		var id int64
		var data []byte
		if err := rows.Scan(&id, &data); err != nil {
			return err
		}
		out[id] = data
		return nil
	})
	return out, err
}

// movesForPositions reads the played moves of every position in the batch, keeping the
// database's own order within a position.
func (d *Database) movesForPositions(ids []int64) (map[int64][]exportedMove, error) {
	out := make(map[int64][]exportedMove, len(ids))
	err := d.forEachInBatch(ids, `SELECT position_id, checker_move, cube_action FROM move WHERE position_id IN `, func(rows *sql.Rows) error {
		var id int64
		var move exportedMove
		if err := rows.Scan(&id, &move.checkerMove, &move.cubeAction); err != nil {
			return err
		}
		out[id] = append(out[id], move)
		return nil
	})
	return out, err
}

// commentsForPositions reads the comment of every position in the batch. A position with
// several comments keeps the first one, which is what the per-position query it replaces
// did.
func (d *Database) commentsForPositions(ids []int64) (map[int64]string, error) {
	out := make(map[int64]string, len(ids))
	err := d.forEachInBatch(ids, `SELECT position_id, text FROM comment WHERE position_id IN `, func(rows *sql.Rows) error {
		var id int64
		var text string
		if err := rows.Scan(&id, &text); err != nil {
			return err
		}
		if _, seen := out[id]; !seen {
			out[id] = text
		}
		return nil
	})
	return out, err
}

// forEachInBatch runs prefix + an IN list over the identifiers, splitting them so the
// statement stays under SQLite's parameter limit, and hands every row to scan.
func (d *Database) forEachInBatch(ids []int64, prefix string, scan func(*sql.Rows) error) error {
	const chunk = 900
	for start := 0; start < len(ids); start += chunk {
		end := min(start+chunk, len(ids))
		batch := ids[start:end]
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(batch)), ",")
		args := make([]any, len(batch))
		for i, id := range batch {
			args[i] = id
		}
		rows, err := d.db.Query(prefix+`(`+placeholders+`)`, args...)
		if err != nil {
			return err
		}
		for rows.Next() {
			if err := scan(rows); err != nil {
				rows.Close()
				return err
			}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()
	}
	return nil
}

// decodeAnalysesConcurrently decodes a batch of stored analyses in parallel.
//
// Decoding is decompression followed by a JSON unmarshal — pure computation, independent
// from one position to the next, and 38% of the export's time once the per-position queries
// were gone. Spreading a batch across the machine's cores is the whole optimisation; the
// writes stay sequential inside their single transaction.
//
// An analysis that fails to decode maps to nil, which the caller treats as "no analysis to
// export" — the same outcome the sequential version produced when the unmarshal failed.
func decodeAnalysesConcurrently(raw map[int64][]byte) map[int64]*PositionAnalysis {
	out := make(map[int64]*PositionAnalysis, len(raw))
	if len(raw) == 0 {
		return out
	}

	type job struct {
		id   int64
		data []byte
	}
	jobs := make([]job, 0, len(raw))
	for id, data := range raw {
		jobs = append(jobs, job{id: id, data: data})
	}

	workers := min(runtime.NumCPU(), len(jobs))
	decoded := make([]*PositionAnalysis, len(jobs))
	var wg sync.WaitGroup
	var next atomic.Int64
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				i := int(next.Add(1)) - 1
				if i >= len(jobs) {
					return
				}
				analysis, err := decodeAnalysisFromStorage(jobs[i].data)
				if err != nil {
					slog.Warn("decoding analysis for export", "positionID", jobs[i].id, "err", err)
					continue
				}
				decoded[i] = &analysis
			}
		}()
	}
	wg.Wait()

	for i, j := range jobs {
		out[j.id] = decoded[i]
	}
	return out
}
