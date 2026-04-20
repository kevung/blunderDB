package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/kevung/gnubgparser"
)

// ============================================================================
// GnuBG / Jellyfish import functions (SGF, MAT, TXT formats)
// ============================================================================

// ImportGnuBGMatchFromText imports a match from clipboard/string content in MAT/TXT format
// using the gnubgparser library. Only MAT/TXT format is supported (no SGF).
func (d *Database) ImportGnuBGMatchFromText(content string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	gnuMatch, err := gnubgparser.ParseMAT(strings.NewReader(content))
	if err != nil {
		return 0, fmt.Errorf("failed to parse match text: %w", err)
	}

	return d.importGnuBGMatchInternal(gnuMatch, "clipboard", false)
}

// ImportGnuBGMatch imports a match from a GnuBG file (SGF, MAT, or TXT format)
// using the gnubgparser library. SGF files include full analysis data,
// while MAT/TXT files contain only moves (no analysis).
func (d *Database) ImportGnuBGMatch(filePath string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Determine format from extension and parse accordingly
	ext := strings.ToLower(filepath.Ext(filePath))
	var gnuMatch *gnubgparser.Match
	var err error

	// isSGF indicates whether moves use absolute coordinates (Player 0's system)
	// SGF: moves are in absolute coords, MoveString is in letter format (needs conversion)
	// MAT/TXT: moves are in player-relative coords, MoveString is already human-readable
	isSGF := ext == ".sgf"

	switch ext {
	case ".sgf":
		gnuMatch, err = gnubgparser.ParseSGFFile(filePath)
	case ".mat", ".txt":
		gnuMatch, err = gnubgparser.ParseMATFile(filePath)
	default:
		return 0, fmt.Errorf("unsupported file format: %s", ext)
	}

	if err != nil {
		return 0, fmt.Errorf("failed to parse file: %w", err)
	}

	return d.importGnuBGMatchInternal(gnuMatch, filePath, isSGF)
}

// importGnuBGMatchInternal is the shared implementation for importing a parsed GnuBG match.
// isSGF indicates SGF format where moves use absolute coordinates.
func (d *Database) importGnuBGMatchInternal(gnuMatch *gnubgparser.Match, filePath string, isSGF bool) (int64, error) {
	// Parse match date
	var matchDate time.Time
	if gnuMatch.Metadata.Date != "" {
		for _, layout := range []string{
			"2006-01-02 15:04:05",
			"2006-01-02",
			"2006/01/02",
			"01/02/2006",
			"January 2, 2006",
			time.RFC3339,
		} {
			if t, parseErr := time.Parse(layout, gnuMatch.Metadata.Date); parseErr == nil {
				matchDate = t
				break
			}
		}
	}
	if matchDate.IsZero() {
		matchDate = time.Now()
	}

	// Compute match hash for duplicate detection
	matchHash := ComputeGnuBGMatchHash(gnuMatch)

	// Compute canonical hash (format-independent)
	canonicalHash := ComputeCanonicalMatchHashFromGnuBG(gnuMatch)

	// Check if this match already exists (same format)
	existingMatchID, err := d.checkMatchExistsLocked(matchHash)
	if err != nil {
		return 0, fmt.Errorf("failed to check for duplicate match: %w", err)
	}
	if existingMatchID > 0 {
		return 0, ErrDuplicateMatch
	}

	// Check for canonical duplicate (same match from different format)
	canonicalMatchID, err := d.checkCanonicalMatchExistsLocked(canonicalHash)
	if err != nil {
		return 0, fmt.Errorf("failed to check for canonical duplicate: %w", err)
	}
	isCanonicalDuplicate := canonicalMatchID > 0

	// Begin transaction for atomic import
	tx, err := d.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Insert match metadata or reuse existing canonical match
	var matchID int64
	if isCanonicalDuplicate {
		matchID = canonicalMatchID
		slog.Info("canonical duplicate detected, reusing match", "matchID", matchID)
	} else {
		result, err := tx.Exec(`
			INSERT INTO match (player1_name, player2_name, event, location, round,
			                   match_length, match_date, file_path, game_count, match_hash, canonical_hash)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, gnuMatch.Metadata.Player1, gnuMatch.Metadata.Player2,
			gnuMatch.Metadata.Event, gnuMatch.Metadata.Place, gnuMatch.Metadata.Round,
			gnuMatch.Metadata.MatchLength, matchDate, filePath, len(gnuMatch.Games), matchHash, canonicalHash)

		if err != nil {
			return 0, fmt.Errorf("failed to insert match: %w", err)
		}

		matchID, err = result.LastInsertId()
		if err != nil {
			return 0, fmt.Errorf("failed to get match ID: %w", err)
		}

		// Auto-link tournament from event metadata
		eventName := strings.TrimSpace(gnuMatch.Metadata.Event)
		if eventName != "" {
			var tournamentID int64
			err2 := tx.QueryRow(`SELECT id FROM tournament WHERE name = ?`, eventName).Scan(&tournamentID)
			if err2 != nil {
				res2, err3 := tx.Exec(`INSERT INTO tournament (name, date, location) VALUES (?, '', '')`, eventName)
				if err3 == nil {
					tournamentID, err = res2.LastInsertId()
					if err != nil {
						return 0, fmt.Errorf("failed to get last insert ID: %w", err)
					}
				}
			}
			if tournamentID > 0 {
				_, err = tx.Exec(`UPDATE match SET tournament_id = ? WHERE id = ?`, tournamentID, matchID)
				if err != nil {
					slog.Warn("failed to link match to tournament", "err", err)
				}
			}
		}
	}

	// Per-import deduplication cache keyed by Zobrist hash.
	cache := newImportCache()

	if isCanonicalDuplicate {
		// Canonical duplicate: import analysis to existing positions, create genuinely new ones
		for gameIdx, game := range gnuMatch.Games {
			game.GameNumber = gameIdx + 1
			currentBoard := initStandardGnuBGPosition()

			for i := range game.Moves {
				moveRec := &game.Moves[i]

				switch string(moveRec.Type) {
				case "setboard":
					if moveRec.Position != nil {
						currentBoard = *moveRec.Position
					}
					continue
				case "setdice", "setcube", "setcubepos":
					if moveRec.Type == "setcube" {
						currentBoard.CubeValue = moveRec.CubeValue
					}
					if moveRec.Type == "setcubepos" {
						currentBoard.CubeOwner = moveRec.CubeOwner
					}
					continue
				}

				if moveRec.Position == nil {
					posCopy := currentBoard
					moveRec.Position = &posCopy
				}

				switch moveRec.Type {
				case "move":
					if moveRec.Analysis != nil && len(moveRec.Analysis.Moves) > 0 {
						pos, err := d.createPositionFromGnuBG(moveRec.Position, &game, gnuMatch.Metadata.MatchLength)
						if err != nil {
							continue
						}
						pos.PlayerOnRoll = moveRec.Player
						pos.DecisionType = CheckerAction
						pos.Dice = [2]int{moveRec.Dice[0], moveRec.Dice[1]}

						posID, err := d.findOrCreatePositionForCanonicalDuplicate(tx, pos, cache)
						if err != nil {
							continue
						}

						var checkerMoveStr string
						if isSGF {
							checkerMoveStr = convertGnuBGMoveToString(moveRec.Move, moveRec.Player)
						} else {
							checkerMoveStr = moveRec.MoveString
							if checkerMoveStr == "" {
								checkerMoveStr = convertPlayerRelativeMoveToString(moveRec.Move)
							}
						}
						err = d.saveGnuBGCheckerAnalysisToPositionInTx(tx, posID, moveRec.Analysis, moveRec.Player, checkerMoveStr, isSGF)
						if err != nil {
							slog.Warn("failed to save analysis for canonical duplicate", "err", err)
						}
					}
					if moveRec.CubeAnalysis != nil {
						pos, err := d.createPositionFromGnuBG(moveRec.Position, &game, gnuMatch.Metadata.MatchLength)
						if err == nil {
							pos.PlayerOnRoll = moveRec.Player
							pos.DecisionType = CheckerAction
							pos.Dice = [2]int{moveRec.Dice[0], moveRec.Dice[1]}
							posID, err := d.findOrCreatePositionForCanonicalDuplicate(tx, pos, cache)
							if err == nil {
								// Convert MWC to EMG for match play (copy to avoid mutating original)
								cubeAnalysis := *moveRec.CubeAnalysis
								if gnuMatch.Metadata.MatchLength > 0 && moveRec.Position != nil {
									convertGnuBGCubeMWCToEMG(&cubeAnalysis, game.Score[0], game.Score[1], moveRec.Player, moveRec.Position.CubeValue, gnuMatch.Metadata.MatchLength)
								}
								_ = d.saveGnuBGCubeAnalysisForCheckerPositionInTx(tx, posID, &cubeAnalysis)
							}
						}
					}

				case "double":
					// Only import cube analysis for "double" entries (skip take/drop which are redundant)
					if moveRec.CubeAnalysis != nil {
						pos, err := d.createPositionFromGnuBG(moveRec.Position, &game, gnuMatch.Metadata.MatchLength)
						if err != nil {
							continue
						}
						pos.PlayerOnRoll = moveRec.Player
						pos.DecisionType = CubeAction
						pos.Dice = [2]int{0, 0}

						posID, err := d.findOrCreatePositionForCanonicalDuplicate(tx, pos, cache)
						if err != nil {
							continue
						}

						// Determine played action by looking at the opponent's response
						cubeAction := "Double/Pass" // default
						for j := i + 1; j < len(game.Moves); j++ {
							nextType := string(game.Moves[j].Type)
							if nextType == "take" {
								cubeAction = "Double/Take"
								break
							} else if nextType == "drop" {
								cubeAction = "Double/Pass"
								break
							} else if nextType != "setboard" && nextType != "setdice" && nextType != "setcube" && nextType != "setcubepos" {
								break
							}
						}
						// Convert MWC to EMG for match play (copy to avoid mutating original)
						cubeAnalysis := *moveRec.CubeAnalysis
						if gnuMatch.Metadata.MatchLength > 0 && moveRec.Position != nil {
							convertGnuBGCubeMWCToEMG(&cubeAnalysis, game.Score[0], game.Score[1], moveRec.Player, moveRec.Position.CubeValue, gnuMatch.Metadata.MatchLength)
						}
						err = d.saveGnuBGCubeAnalysisToPositionInTx(tx, posID, &cubeAnalysis, cubeAction)
						if err != nil {
							slog.Warn("failed to save cube analysis for canonical duplicate", "err", err)
						}
					}
				case "take", "drop":
					// Skip — cube analysis was already saved on the "double" entry
				}

				// Update board state
				switch string(moveRec.Type) {
				case "move":
					applyGnuBGCheckerMove(&currentBoard, moveRec, isSGF)
				case "take":
					if currentBoard.CubeValue == 0 {
						currentBoard.CubeValue = 2
					} else {
						currentBoard.CubeValue *= 2
					}
					currentBoard.CubeOwner = moveRec.Player
				}
			}
		}
	} else {
		// Normal import path
		for gameIdx, game := range gnuMatch.Games {
			game.GameNumber = gameIdx + 1
			gameID, err := d.importGnuBGGame(tx, matchID, &game)
			if err != nil {
				return 0, fmt.Errorf("failed to import game %d: %w", game.GameNumber, err)
			}

			currentBoard := initStandardGnuBGPosition()

			moveNumber := int32(0)
			for i := range game.Moves {
				// Cancellation check at the top of every move iteration.
				if atomic.LoadInt32(&d.importCancelled) != 0 {
					return 0, fmt.Errorf("import cancelled")
				}

				moveRec := &game.Moves[i]

				switch string(moveRec.Type) {
				case "setboard":
					if moveRec.Position != nil {
						currentBoard = *moveRec.Position
					}
					continue
				case "setdice":
					continue
				case "setcube":
					currentBoard.CubeValue = moveRec.CubeValue
					continue
				case "setcubepos":
					currentBoard.CubeOwner = moveRec.CubeOwner
					continue
				}

				if moveRec.Position == nil {
					posCopy := currentBoard
					moveRec.Position = &posCopy
				}

				// For "double" moves, determine the actual cube action
				// by looking ahead to the opponent's response (take/drop)
				cubeAction := ""
				if string(moveRec.Type) == "double" {
					cubeAction = "Double/Pass" // default if no response found
					for j := i + 1; j < len(game.Moves); j++ {
						nextType := string(game.Moves[j].Type)
						if nextType == "take" {
							cubeAction = "Double/Take"
							break
						} else if nextType == "drop" {
							cubeAction = "Double/Pass"
							break
						} else if nextType != "setboard" && nextType != "setdice" && nextType != "setcube" && nextType != "setcubepos" {
							break // unexpected move type, stop looking
						}
					}
				}

				err := d.importGnuBGMove(tx, gameID, moveNumber, moveRec, &game, gnuMatch.Metadata.MatchLength, cache, isSGF, cubeAction)
				if err != nil {
					slog.Warn("failed to import move", "move", moveNumber, "game", game.GameNumber, "err", err)
					moveNumber++
					continue
				}

				switch string(moveRec.Type) {
				case "move":
					applyGnuBGCheckerMove(&currentBoard, moveRec, isSGF)
				case "take":
					if currentBoard.CubeValue == 0 {
						currentBoard.CubeValue = 2
					} else {
						currentBoard.CubeValue *= 2
					}
					currentBoard.CubeOwner = moveRec.Player
				}

				// take/drop don't increment moveNumber (they're part of the double action)
				if string(moveRec.Type) != "take" && string(moveRec.Type) != "drop" {
					moveNumber++
				}
			}
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	slog.Info("imported GnuBG match", "matchID", matchID, "games", len(gnuMatch.Games), "file", filePath)
	return matchID, nil
}

// importGnuBGGame inserts a game record from gnubgparser data and returns its ID
func (d *Database) importGnuBGGame(tx *sql.Tx, matchID int64, game *gnubgparser.Game) (int64, error) {
	result, err := tx.Exec(`
		INSERT INTO game (match_id, game_number, initial_score_1, initial_score_2,
		                  winner, points_won, move_count)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, matchID, game.GameNumber, game.Score[0], game.Score[1],
		game.Winner, game.Points, len(game.Moves))

	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// importGnuBGMove imports a single move record from gnubgparser data
// isSGF indicates SGF format where moves use absolute coordinates (Player 0's system)
func (d *Database) importGnuBGMove(tx *sql.Tx, gameID int64, moveNumber int32, moveRec *gnubgparser.MoveRecord, game *gnubgparser.Game, matchLength int, cache *importCache, isSGF bool, cubeAction string) error {
	switch moveRec.Type {
	case "move":
		return d.importGnuBGCheckerMove(tx, gameID, moveNumber, moveRec, game, matchLength, cache, isSGF)
	case "double":
		// cubeAction is determined by the caller based on the opponent's response
		// ("Double/Take" or "Double/Pass")
		return d.importGnuBGCubeMove(tx, gameID, moveNumber, moveRec, game, matchLength, cache, cubeAction, isSGF)
	case "take", "drop":
		// Skip take/drop as separate entries — the "double" entry already captures
		// the full cube decision (like XG's single "Double/Pass" or "Double/Take")
		return nil
	case "resign":
		// Skip resign moves - they don't produce positions
		return nil
	default:
		// Skip unknown move types
		return nil
	}
}

// importGnuBGCheckerMove handles importing a checker move from gnubgparser
// isSGF indicates SGF format where moves use absolute coordinates and MoveString is in letter format
func (d *Database) importGnuBGCheckerMove(tx *sql.Tx, gameID int64, moveNumber int32, moveRec *gnubgparser.MoveRecord, game *gnubgparser.Game, matchLength int, cache *importCache, isSGF bool) error {
	player := moveRec.Player // 0 or 1, maps directly to blunderDB

	// Convert player to XG-style encoding for DB storage consistency
	// The DB player column uses XG encoding (1=Player1, -1=Player2) so that
	// GetMatchMovePositions can use convertXGPlayerToBlunderDB uniformly.
	dbPlayer := convertBlunderDBPlayerToXG(player)

	// Get move string
	var checkerMoveStr string
	if isSGF {
		// For SGF files, Move[8]int is in absolute coordinates (Player 0's system).
		// Always compute from Move array since MoveString is in letter format.
		checkerMoveStr = convertGnuBGMoveToString(moveRec.Move, moveRec.Player)
	} else {
		// For MAT/TXT files, MoveString is already in human-readable notation.
		checkerMoveStr = moveRec.MoveString
		if checkerMoveStr == "" {
			// Move[8]int is in player-relative coordinates for MAT/TXT
			checkerMoveStr = convertPlayerRelativeMoveToString(moveRec.Move)
		}
	}

	// If position is available, create and save it
	var positionID int64
	if moveRec.Position != nil {
		pos, err := d.createPositionFromGnuBG(moveRec.Position, game, matchLength)
		if err != nil {
			return fmt.Errorf("failed to create position: %w", err)
		}

		pos.PlayerOnRoll = player
		pos.DecisionType = CheckerAction
		pos.Dice = [2]int{moveRec.Dice[0], moveRec.Dice[1]}

		posID, err := d.savePositionInTxWithCache(tx, pos, cache)
		if err != nil {
			return fmt.Errorf("failed to save position: %w", err)
		}
		positionID = posID
	}

	// Save move record
	var moveResult sql.Result
	var err error
	if positionID > 0 {
		moveResult, err = tx.Exec(`
			INSERT INTO move (game_id, move_number, move_type, position_id, player,
			                  dice_1, dice_2, checker_move)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, gameID, moveNumber, "checker", positionID, dbPlayer,
			moveRec.Dice[0], moveRec.Dice[1], checkerMoveStr)
	} else {
		moveResult, err = tx.Exec(`
			INSERT INTO move (game_id, move_number, move_type, position_id, player,
			                  dice_1, dice_2, checker_move)
			VALUES (?, ?, ?, NULL, ?, ?, ?, ?)
		`, gameID, moveNumber, "checker", dbPlayer,
			moveRec.Dice[0], moveRec.Dice[1], checkerMoveStr)
	}
	if err != nil {
		return err
	}

	moveID, err := moveResult.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	// Save analysis if available (SGF files only)
	if moveRec.Analysis != nil && len(moveRec.Analysis.Moves) > 0 && positionID > 0 {
		// Save to move_analysis table
		for _, moveOpt := range moveRec.Analysis.Moves {
			err = d.saveGnuBGMoveAnalysisInTx(tx, moveID, &moveOpt)
			if err != nil {
				slog.Warn("failed to save checker analysis", "err", err)
			}
		}

		// Save to position analysis table (for UI compatibility)
		err = d.saveGnuBGCheckerAnalysisToPositionInTx(tx, positionID, moveRec.Analysis, moveRec.Player, checkerMoveStr, isSGF)
		if err != nil {
			slog.Warn("failed to save position analysis", "err", err)
		}
	}

	// Save cube analysis if available on a checker move position
	if moveRec.CubeAnalysis != nil && positionID > 0 {
		// Convert MWC to EMG for match play (copy to avoid mutating original)
		cubeAnalysis := *moveRec.CubeAnalysis
		if matchLength > 0 && moveRec.Position != nil {
			convertGnuBGCubeMWCToEMG(&cubeAnalysis, game.Score[0], game.Score[1], player, moveRec.Position.CubeValue, matchLength)
		}
		err = d.saveGnuBGCubeAnalysisForCheckerPositionInTx(tx, positionID, &cubeAnalysis)
		if err != nil {
			slog.Warn("failed to save cube analysis for checker position", "err", err)
		}
	}

	return nil
}

// importGnuBGCubeMove handles importing a cube move (double/take/drop) from gnubgparser
func (d *Database) importGnuBGCubeMove(tx *sql.Tx, gameID int64, moveNumber int32, moveRec *gnubgparser.MoveRecord, game *gnubgparser.Game, matchLength int, cache *importCache, cubeAction string, isSGF bool) error {
	player := moveRec.Player // 0 or 1

	// Convert player to XG-style encoding for DB storage consistency
	dbPlayer := convertBlunderDBPlayerToXG(player)

	var positionID int64
	if moveRec.Position != nil {
		pos, err := d.createPositionFromGnuBG(moveRec.Position, game, matchLength)
		if err != nil {
			return fmt.Errorf("failed to create position: %w", err)
		}

		pos.PlayerOnRoll = player
		pos.DecisionType = CubeAction
		pos.Dice = [2]int{0, 0} // No dice for cube decisions

		posID, err := d.savePositionInTxWithCache(tx, pos, cache)
		if err != nil {
			return fmt.Errorf("failed to save position: %w", err)
		}
		positionID = posID
	}

	// Save move record
	var moveResult sql.Result
	var err error
	if positionID > 0 {
		moveResult, err = tx.Exec(`
			INSERT INTO move (game_id, move_number, move_type, position_id, player,
			                  dice_1, dice_2, cube_action)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, gameID, moveNumber, "cube", positionID, dbPlayer, 0, 0, cubeAction)
	} else {
		moveResult, err = tx.Exec(`
			INSERT INTO move (game_id, move_number, move_type, position_id, player,
			                  dice_1, dice_2, cube_action)
			VALUES (?, ?, ?, NULL, ?, ?, ?, ?)
		`, gameID, moveNumber, "cube", dbPlayer, 0, 0, cubeAction)
	}
	if err != nil {
		return err
	}

	moveID, err := moveResult.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	// Save cube analysis if available (SGF files only)
	if moveRec.CubeAnalysis != nil && positionID > 0 {
		// Convert MWC to EMG for match play (copy to avoid mutating original)
		cubeAnalysis := *moveRec.CubeAnalysis
		if matchLength > 0 && moveRec.Position != nil {
			convertGnuBGCubeMWCToEMG(&cubeAnalysis, game.Score[0], game.Score[1], player, moveRec.Position.CubeValue, matchLength)
		}

		// Save to move_analysis table
		err = d.saveGnuBGCubeMoveAnalysisInTx(tx, moveID, &cubeAnalysis)
		if err != nil {
			slog.Warn("failed to save cube analysis", "err", err)
		}

		// Save to position analysis table (for UI compatibility)
		err = d.saveGnuBGCubeAnalysisToPositionInTx(tx, positionID, &cubeAnalysis, cubeAction)
		if err != nil {
			slog.Warn("failed to save position cube analysis", "err", err)
		}
	}

	return nil
}

// initStandardGnuBGPosition returns a gnubgparser.Position set to the standard
// backgammon starting position. Used when SGF files don't include explicit setboard events.
//
// In gnuBG's player-relative encoding (0=ace point, 23=24-point):
//   - Each player has: 2@pt23, 5@pt12, 3@pt7, 5@pt5 (15 checkers total)
func initStandardGnuBGPosition() gnubgparser.Position {
	var pos gnubgparser.Position
	pos.CubeValue = 1
	pos.CubeOwner = -1 // center

	// Standard starting position (same for both players from their own perspective)
	for p := 0; p < 2; p++ {
		pos.Board[p][23] = 2 // 24-point: 2 checkers
		pos.Board[p][12] = 5 // 13-point: 5 checkers
		pos.Board[p][7] = 3  // 8-point: 3 checkers
		pos.Board[p][5] = 5  // 6-point: 5 checkers
	}

	return pos
}

// applyGnuBGCheckerMove updates a gnubgparser board state after a checker move.
//
// When isAbsoluteCoords is true (SGF format):
//
//	Move[8]int uses absolute coordinates (Player 0's system: 0=1pt, 23=24pt, 24=bar, 25=off).
//	For Player 1, indices must be mirrored to reach the player-relative board.
//
// When isAbsoluteCoords is false (MAT/TXT format):
//
//	Move[8]int uses player-relative coordinates (from player's perspective:
//	0=ace/home, 23=24pt, 24=bar, -1=off). No mirroring needed.
//
// Board[p] always uses player-relative coords (0=ace/home, 23=far, 24=bar).
func applyGnuBGCheckerMove(board *gnubgparser.Position, moveRec *gnubgparser.MoveRecord, isAbsoluteCoords bool) {
	player := moveRec.Player
	opponent := 1 - player

	for i := 0; i < 8; i += 2 {
		from := moveRec.Move[i]
		to := moveRec.Move[i+1]
		if from == -1 {
			break
		}

		var fromBoard, toBoard, opponentBoard int
		var isBearOff bool

		if isAbsoluteCoords {
			// SGF: absolute coordinates — mirror for Player 1
			fromBoard = from
			if player == 1 && from != 24 {
				fromBoard = 23 - from
			}

			isBearOff = (to == 25)

			if !isBearOff {
				toBoard = to
				if player == 1 {
					toBoard = 23 - to
				}
				// Opponent's board index for the same physical point
				if player == 0 {
					opponentBoard = 23 - to
				} else {
					opponentBoard = to
				}
			}
		} else {
			// MAT/TXT: player-relative coordinates — no mirroring needed
			fromBoard = from // already in player's perspective

			isBearOff = (to == -1)

			if !isBearOff {
				toBoard = to // already in player's perspective
				// Opponent sees the mirror of this physical point
				opponentBoard = 23 - to
			}
		}

		// Remove checker from source point
		if fromBoard >= 0 && fromBoard <= 24 {
			board.Board[player][fromBoard]--
		}

		// If bearing off, checker leaves the board entirely
		if isBearOff {
			continue
		}

		// Check for hit at destination
		if opponentBoard >= 0 && opponentBoard <= 23 {
			if board.Board[opponent][opponentBoard] == 1 {
				// Hit: send opponent's checker to the bar
				board.Board[opponent][opponentBoard] = 0
				board.Board[opponent][24]++
			}
		}

		// Place checker at destination
		if toBoard >= 0 && toBoard <= 24 {
			board.Board[player][toBoard]++
		}
	}
}

// createPositionFromGnuBG converts a gnubgparser.Position to a blunderDB Position
//
// gnubgparser board encoding (player-relative):
//   - Board[player][0-23]: board points from player's perspective (0=ace/home, 23=far)
//   - Board[player][24]: checkers on bar
//
// blunderDB board encoding (absolute):
//   - Points[0]: Player 2's bar (White)
//   - Points[1-24]: board points (standard numbering)
//   - Points[25]: Player 1's bar (Black)
//   - Color 0 = Player 1 (Black, moves 24→1), Color 1 = Player 2 (White, moves 1→24)
//
// Mapping:
//   - Board[0][i] (player 0/Black): blunderDB point (i+1), Color 0
//   - Board[1][i] (player 1/White): blunderDB point (24-i), Color 1
//   - Board[0][24] (player 0 bar): blunderDB Points[25]
//   - Board[1][24] (player 1 bar): blunderDB Points[0]
func (d *Database) createPositionFromGnuBG(gnubgPos *gnubgparser.Position, game *gnubgparser.Game, matchLength int) (*Position, error) {
	// Calculate away scores
	// blunderDB stores scores as "points away from winning"
	awayScore1 := matchLength - game.Score[0]
	awayScore2 := matchLength - game.Score[1]

	// Handle unlimited/money match (matchLength == 0)
	if matchLength == 0 {
		awayScore1 = -1
		awayScore2 = -1
	}

	// Convert cube value from actual (1,2,4,8...) to exponent (0,1,2,3...)
	cubeValue := 0
	if gnubgPos.CubeValue > 0 {
		for v := gnubgPos.CubeValue; v > 1; v >>= 1 {
			cubeValue++
		}
	}

	// Cube owner: gnubgparser uses -1=center, 0=player0, 1=player1 (same as blunderDB)
	cubeOwner := gnubgPos.CubeOwner

	pos := &Position{
		PlayerOnRoll: gnubgPos.OnRoll, // Will be overridden from move context
		DecisionType: CheckerAction,   // Will be overridden from move context
		Score:        [2]int{awayScore1, awayScore2},
		Cube: Cube{
			Value: cubeValue,
			Owner: cubeOwner,
		},
		Dice: [2]int{0, 0}, // Will be set from move data
	}

	// Initialize all points as empty
	for i := 0; i < 26; i++ {
		pos.Board.Points[i] = Point{Checkers: 0, Color: -1}
	}

	// Place Player 0's (Black) checkers
	for pt := 0; pt < 25; pt++ {
		count := gnubgPos.Board[0][pt]
		if count > 0 {
			if pt == 24 {
				// Bar: Player 0's bar → blunderDB index 25
				pos.Board.Points[25] = Point{Checkers: count, Color: 0}
			} else {
				// Board point: gnubg pt → blunderDB pt+1
				pos.Board.Points[pt+1] = Point{Checkers: count, Color: 0}
			}
		}
	}

	// Place Player 1's (White) checkers
	for pt := 0; pt < 25; pt++ {
		count := gnubgPos.Board[1][pt]
		if count > 0 {
			if pt == 24 {
				// Bar: Player 1's bar → blunderDB index 0
				pos.Board.Points[0] = Point{Checkers: count, Color: 1}
			} else {
				// Board point: gnubg pt (from player 1's view) → blunderDB (24-pt)
				pos.Board.Points[24-pt] = Point{Checkers: count, Color: 1}
			}
		}
	}

	// Calculate bearoff (15 checkers total per player minus those on the board)
	player1Total := 0
	player2Total := 0
	for i := 0; i < 26; i++ {
		if pos.Board.Points[i].Color == 0 {
			player1Total += pos.Board.Points[i].Checkers
		} else if pos.Board.Points[i].Color == 1 {
			player2Total += pos.Board.Points[i].Checkers
		}
	}
	pos.Board.Bearoff = [2]int{15 - player1Total, 15 - player2Total}

	return pos, nil
}

// convertGnuBGMoveToString converts a gnubgparser Move[8]int to standard notation.
// This function handles moves in ABSOLUTE coordinates (SGF format).
// Move encoding: 0-23 = board points (absolute), 24 = bar, 25 = off, -1 = unused
// For player 0: gnubg point i → standard point (i+1)
// For player 1: gnubg point i → standard point (24-i)
func convertGnuBGMoveToString(move [8]int, player int) string {
	formatPoint := func(pt int, p int) string {
		if pt == 24 {
			return "bar"
		}
		if pt == 25 {
			return "off"
		}
		if p == 0 {
			return fmt.Sprintf("%d", pt+1) // Player 0: 0→1, 23→24
		}
		return fmt.Sprintf("%d", 24-pt) // Player 1: 0→24, 23→1
	}

	return formatGnuBGMoveItems(move, player, formatPoint)
}

// convertPlayerRelativeMoveToString converts a player-relative Move[8]int to standard notation.
// This function handles moves in PLAYER-RELATIVE coordinates (MAT/TXT format).
// Move encoding: 0-23 = board points (from player's perspective), 24 = bar, -1 = off
// For both players: point i → standard point (i+1)
func convertPlayerRelativeMoveToString(move [8]int) string {
	formatPoint := func(pt int, _ int) string {
		if pt == 24 {
			return "bar"
		}
		if pt == -1 {
			return "off"
		}
		return fmt.Sprintf("%d", pt+1) // Player-relative: 0→1, 23→24
	}

	return formatGnuBGMoveItems(move, 0, formatPoint)
}

// formatGnuBGMoveItems is a helper that formats move items using a point formatter.
func formatGnuBGMoveItems(move [8]int, player int, formatPoint func(int, int) string) string {

	type moveItem struct {
		from string
		to   string
	}

	var items []moveItem
	for i := 0; i < 8; i += 2 {
		from := move[i]
		to := move[i+1]
		if from == -1 {
			break
		}
		items = append(items, moveItem{
			from: formatPoint(from, player),
			to:   formatPoint(to, player),
		})
	}

	if len(items) == 0 {
		return "Cannot Move"
	}

	// Sort by 'from' point descending (standard notation)
	sort.Slice(items, func(i, j int) bool {
		// Parse to int for comparison
		fi, _ := strconv.Atoi(items[i].from)
		fj, _ := strconv.Atoi(items[j].from)
		if items[i].from == "bar" {
			fi = 25
		}
		if items[j].from == "bar" {
			fj = 25
		}
		return fi > fj
	})

	// Group identical moves with multiplier
	var moves []string
	for i := 0; i < len(items); {
		item := items[i]
		count := 1
		for j := i + 1; j < len(items); j++ {
			if items[j].from == item.from && items[j].to == item.to {
				count++
			} else {
				break
			}
		}
		if count > 1 {
			moves = append(moves, fmt.Sprintf("%s/%s(%d)", item.from, item.to, count))
		} else {
			moves = append(moves, fmt.Sprintf("%s/%s", item.from, item.to))
		}
		i += count
	}

	return strings.Join(moves, " ")
}

// saveGnuBGMoveAnalysisInTx saves a gnubgparser MoveOption to the move_analysis table
func (d *Database) saveGnuBGMoveAnalysisInTx(tx *sql.Tx, moveID int64, moveOpt *gnubgparser.MoveOption) error {
	// gnubgparser rates are 0-1 fractions; convert to integer × 100 of percentage
	player1WinRate := int64(math.Round(float64(moveOpt.Player1WinRate) * 100.0 * 100))
	player2WinRate := int64(math.Round(float64(moveOpt.Player2WinRate) * 100.0 * 100))

	_, err := tx.Exec(`
		INSERT INTO move_analysis (move_id, analysis_type, depth, equity, equity_error,
		                           win_rate, gammon_rate, backgammon_rate,
		                           opponent_win_rate, opponent_gammon_rate, opponent_backgammon_rate)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, moveID, "checker", translateGnuBGAnalysisDepth(moveOpt.AnalysisDepth),
		int64(math.Round(moveOpt.Equity*1000)), int64(0),
		player1WinRate, int64(math.Round(float64(moveOpt.Player1GammonRate)*100.0*100)), int64(math.Round(float64(moveOpt.Player1BackgammonRate)*100.0*100)),
		player2WinRate, int64(math.Round(float64(moveOpt.Player2GammonRate)*100.0*100)), int64(math.Round(float64(moveOpt.Player2BackgammonRate)*100.0*100)))

	return err
}

// saveGnuBGCubeMoveAnalysisInTx saves gnubgparser CubeAnalysis to the move_analysis table
func (d *Database) saveGnuBGCubeMoveAnalysisInTx(tx *sql.Tx, moveID int64, analysis *gnubgparser.CubeAnalysis) error {
	player1WinRate := int64(math.Round(float64(analysis.Player1WinRate) * 100.0 * 100))
	player2WinRate := int64(math.Round(float64(analysis.Player2WinRate) * 100.0 * 100))

	_, err := tx.Exec(`
		INSERT INTO move_analysis (move_id, analysis_type, depth, equity, equity_error,
		                           win_rate, gammon_rate, backgammon_rate,
		                           opponent_win_rate, opponent_gammon_rate, opponent_backgammon_rate)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, moveID, "cube", translateGnuBGAnalysisDepth(analysis.AnalysisDepth),
		int64(math.Round(analysis.CubefulNoDouble*1000)), int64(0),
		player1WinRate, int64(math.Round(float64(analysis.Player1GammonRate)*100.0*100)), int64(math.Round(float64(analysis.Player1BackgammonRate)*100.0*100)),
		player2WinRate, int64(math.Round(float64(analysis.Player2GammonRate)*100.0*100)), int64(math.Round(float64(analysis.Player2BackgammonRate)*100.0*100)))

	return err
}

// saveGnuBGCheckerAnalysisToPositionInTx converts gnubgparser MoveAnalysis to PositionAnalysis and saves it
// isSGF indicates SGF format where Move[8]int is in absolute coordinates and MoveString is in letter format
func (d *Database) saveGnuBGCheckerAnalysisToPositionInTx(tx *sql.Tx, positionID int64, analysis *gnubgparser.MoveAnalysis, player int, playedMoveStr string, isSGF bool) error {
	if analysis == nil || len(analysis.Moves) == 0 {
		return nil
	}

	posAnalysis := PositionAnalysis{
		PositionID:            int(positionID),
		AnalysisType:          "CheckerMove",
		AnalysisEngineVersion: "GNU Backgammon",
		CreationDate:          time.Now(),
		LastModifiedDate:      time.Now(),
	}

	// Build checker moves list
	checkerMoves := make([]CheckerMove, 0, len(analysis.Moves))
	for i, moveOpt := range analysis.Moves {
		var moveStr string
		if isSGF {
			// For SGF, always convert from Move[8]int (absolute coords) to numeric notation
			moveStr = convertGnuBGMoveToString(moveOpt.Move, player)
		} else {
			// For MAT/TXT, use the existing human-readable MoveString
			moveStr = moveOpt.MoveString
			if moveStr == "" {
				// Move[8]int is in player-relative coordinates for MAT/TXT
				moveStr = convertPlayerRelativeMoveToString(moveOpt.Move)
			}
		}

		var equityError *float64
		if i > 0 {
			diff := analysis.Moves[0].Equity - moveOpt.Equity
			equityError = &diff
		}

		checkerMove := CheckerMove{
			Index:                    i,
			AnalysisDepth:            translateGnuBGAnalysisDepth(moveOpt.AnalysisDepth),
			AnalysisEngine:           "GNUbg",
			Move:                     moveStr,
			Equity:                   moveOpt.Equity,
			EquityError:              equityError,
			PlayerWinChance:          float64(moveOpt.Player1WinRate) * 100.0,
			PlayerGammonChance:       float64(moveOpt.Player1GammonRate) * 100.0,
			PlayerBackgammonChance:   float64(moveOpt.Player1BackgammonRate) * 100.0,
			OpponentWinChance:        float64(moveOpt.Player2WinRate) * 100.0,
			OpponentGammonChance:     float64(moveOpt.Player2GammonRate) * 100.0,
			OpponentBackgammonChance: float64(moveOpt.Player2BackgammonRate) * 100.0,
		}
		checkerMoves = append(checkerMoves, checkerMove)
	}

	posAnalysis.CheckerAnalysis = &CheckerAnalysis{
		Moves: checkerMoves,
	}

	// Set played move
	if playedMoveStr != "" {
		posAnalysis.PlayedMoves = []string{playedMoveStr}
	}

	return d.saveAnalysisInTx(tx, positionID, posAnalysis)
}

// saveGnuBGCubeAnalysisToPositionInTx converts gnubgparser CubeAnalysis to PositionAnalysis and saves it
func (d *Database) saveGnuBGCubeAnalysisToPositionInTx(tx *sql.Tx, positionID int64, analysis *gnubgparser.CubeAnalysis, playedCubeAction string) error {
	if analysis == nil {
		return nil
	}

	posAnalysis := PositionAnalysis{
		PositionID:            int(positionID),
		AnalysisType:          "DoublingCube",
		AnalysisEngineVersion: "GNU Backgammon",
		CreationDate:          time.Now(),
		LastModifiedDate:      time.Now(),
	}

	cubefulNoDouble := analysis.CubefulNoDouble
	cubefulDoubleTake := analysis.CubefulDoubleTake
	cubefulDoublePass := analysis.CubefulDoublePass

	// Calculate best equity considering opponent's optimal response
	effectiveDoubleEquity := cubefulDoubleTake
	if cubefulDoublePass < cubefulDoubleTake {
		effectiveDoubleEquity = cubefulDoublePass
	}

	bestEquity := cubefulNoDouble
	bestAction := "No Double"
	if effectiveDoubleEquity > cubefulNoDouble {
		bestEquity = effectiveDoubleEquity
		if cubefulDoubleTake <= cubefulDoublePass {
			bestAction = "Double, Take"
		} else {
			bestAction = "Double, Pass"
		}
	}

	cubeAnalysis := DoublingCubeAnalysis{
		AnalysisDepth:             translateGnuBGAnalysisDepth(analysis.AnalysisDepth),
		AnalysisEngine:            "GNUbg",
		PlayerWinChances:          float64(analysis.Player1WinRate) * 100.0,
		PlayerGammonChances:       float64(analysis.Player1GammonRate) * 100.0,
		PlayerBackgammonChances:   float64(analysis.Player1BackgammonRate) * 100.0,
		OpponentWinChances:        float64(analysis.Player2WinRate) * 100.0,
		OpponentGammonChances:     float64(analysis.Player2GammonRate) * 100.0,
		OpponentBackgammonChances: float64(analysis.Player2BackgammonRate) * 100.0,
		CubelessNoDoubleEquity:    analysis.CubelessEquity,
		CubelessDoubleEquity:      analysis.CubelessEquity,
		CubefulNoDoubleEquity:     cubefulNoDouble,
		CubefulNoDoubleError:      cubefulNoDouble - bestEquity,
		CubefulDoubleTakeEquity:   cubefulDoubleTake,
		CubefulDoubleTakeError:    cubefulDoubleTake - bestEquity,
		CubefulDoublePassEquity:   cubefulDoublePass,
		CubefulDoublePassError:    cubefulDoublePass - bestEquity,
		BestCubeAction:            bestAction,
		WrongPassPercentage:       float64(analysis.WrongPassTakePercent) * 100.0,
		WrongTakePercentage:       0.0,
	}

	posAnalysis.DoublingCubeAnalysis = &cubeAnalysis

	// Set played cube action
	if playedCubeAction != "" {
		posAnalysis.PlayedCubeActions = []string{playedCubeAction}
	}

	return d.saveAnalysisInTx(tx, positionID, posAnalysis)
}

// saveGnuBGCubeAnalysisForCheckerPositionInTx saves cube analysis to a checker position
// This allows displaying cube info when pressing 'd' on a checker decision
func (d *Database) saveGnuBGCubeAnalysisForCheckerPositionInTx(tx *sql.Tx, positionID int64, analysis *gnubgparser.CubeAnalysis) error {
	if analysis == nil {
		return nil
	}

	// Build a PositionAnalysis with just the cube analysis and let saveAnalysisInTx handle merging
	posAnalysis := PositionAnalysis{
		PositionID:            int(positionID),
		AnalysisType:          "CheckerMove",
		AnalysisEngineVersion: "GNU Backgammon",
		CreationDate:          time.Now(),
		LastModifiedDate:      time.Now(),
	}

	cubefulNoDouble := analysis.CubefulNoDouble
	cubefulDoubleTake := analysis.CubefulDoubleTake
	cubefulDoublePass := analysis.CubefulDoublePass

	effectiveDoubleEquity := cubefulDoubleTake
	if cubefulDoublePass < cubefulDoubleTake {
		effectiveDoubleEquity = cubefulDoublePass
	}

	bestEquity := cubefulNoDouble
	bestAction := "No Double"
	if effectiveDoubleEquity > cubefulNoDouble {
		bestEquity = effectiveDoubleEquity
		if cubefulDoubleTake <= cubefulDoublePass {
			bestAction = "Double, Take"
		} else {
			bestAction = "Double, Pass"
		}
	}

	cubeAnalysis := DoublingCubeAnalysis{
		AnalysisDepth:             translateGnuBGAnalysisDepth(analysis.AnalysisDepth),
		AnalysisEngine:            "GNUbg",
		PlayerWinChances:          float64(analysis.Player1WinRate) * 100.0,
		PlayerGammonChances:       float64(analysis.Player1GammonRate) * 100.0,
		PlayerBackgammonChances:   float64(analysis.Player1BackgammonRate) * 100.0,
		OpponentWinChances:        float64(analysis.Player2WinRate) * 100.0,
		OpponentGammonChances:     float64(analysis.Player2GammonRate) * 100.0,
		OpponentBackgammonChances: float64(analysis.Player2BackgammonRate) * 100.0,
		CubelessNoDoubleEquity:    analysis.CubelessEquity,
		CubelessDoubleEquity:      analysis.CubelessEquity,
		CubefulNoDoubleEquity:     cubefulNoDouble,
		CubefulNoDoubleError:      cubefulNoDouble - bestEquity,
		CubefulDoubleTakeEquity:   cubefulDoubleTake,
		CubefulDoubleTakeError:    cubefulDoubleTake - bestEquity,
		CubefulDoublePassEquity:   cubefulDoublePass,
		CubefulDoublePassError:    cubefulDoublePass - bestEquity,
		BestCubeAction:            bestAction,
		WrongPassPercentage:       float64(analysis.WrongPassTakePercent) * 100.0,
		WrongTakePercentage:       0.0,
	}

	posAnalysis.DoublingCubeAnalysis = &cubeAnalysis

	return d.saveAnalysisInTx(tx, positionID, posAnalysis)
}

// translateGnuBGAnalysisDepth converts gnuBG analysis depth to a human-readable string
// gnuBG uses ply levels: 0=0-ply (contact/race evaluation), 1=1-ply, 2=2-ply, etc.
func translateGnuBGAnalysisDepth(depth int) string {
	if depth >= 0 {
		return fmt.Sprintf("%d-ply", depth)
	}
	return fmt.Sprintf("%d", depth)
}

// ComputeGnuBGMatchHash generates a unique hash for a gnubgparser match
// Used to detect duplicate imports
func ComputeGnuBGMatchHash(match *gnubgparser.Match) string {
	var hashBuilder strings.Builder

	// Include metadata (normalized)
	p1 := strings.TrimSpace(strings.ToLower(match.Metadata.Player1))
	p2 := strings.TrimSpace(strings.ToLower(match.Metadata.Player2))
	hashBuilder.WriteString(fmt.Sprintf("meta:%s|%s|%d|", p1, p2, match.Metadata.MatchLength))

	// Include full game transcription
	for gameIdx, game := range match.Games {
		hashBuilder.WriteString(fmt.Sprintf("g%d:%d,%d,%d,%d|",
			gameIdx, game.Score[0], game.Score[1], game.Winner, game.Points))

		// Include all moves in the game
		for moveIdx, moveRec := range game.Moves {
			hashBuilder.WriteString(fmt.Sprintf("m%d:%s,", moveIdx, string(moveRec.Type)))

			if moveRec.Type == "move" {
				hashBuilder.WriteString(fmt.Sprintf("d%d%d,p%s|",
					moveRec.Dice[0], moveRec.Dice[1], moveRec.MoveString))
			} else if moveRec.Type == "double" || moveRec.Type == "take" || moveRec.Type == "drop" {
				hashBuilder.WriteString(fmt.Sprintf("c%s|", string(moveRec.Type)))
			}
		}
	}

	hash := sha256.Sum256([]byte(hashBuilder.String()))
	return hex.EncodeToString(hash[:])
}
