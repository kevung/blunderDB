package database

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kevung/bgfparser"
	"github.com/kevung/xgparser/xgparser"
)

// ============================================================================
// BGBlitz BGF import functions
// ============================================================================

// ImportBGFMatch imports a match from a BGBlitz BGF file using the bgfparser library.
// BGF files contain full match data including moves, analysis, and cube decisions.
func (d *Database) ImportBGFMatch(filePath string) (int64, error) {
	ctx, done := d.beginCancellableImport()
	defer done()

	d.mu.Lock()
	defer d.mu.Unlock()

	// Parse the BGF file
	bgfMatch, err := bgfparser.ParseBGF(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to parse BGF file: %w", err)
	}

	if bgfMatch.Data == nil {
		return 0, fmt.Errorf("BGF file contains no match data")
	}

	data := bgfMatch.Data

	// Extract match metadata
	nameGreen := bgfGetString(data, "nameGreen")
	nameRed := bgfGetString(data, "nameRed")
	matchLen := bgfGetInt(data, "matchlen")
	event := bgfGetString(data, "event")
	location := bgfGetString(data, "location")
	round := bgfGetString(data, "round")
	dateStr := bgfGetString(data, "date")

	// Parse match date
	matchDate := parseMatchDate(dateStr)

	// Extract games
	gamesData, ok := data["games"].([]interface{})
	if !ok || len(gamesData) == 0 {
		return 0, fmt.Errorf("BGF file contains no games")
	}

	// Compute match hash for duplicate detection
	matchHash := ComputeBGFMatchHash(bgfMatch)

	// Compute canonical hash (format-independent) for cross-format duplicate detection
	canonicalHash := ComputeCanonicalMatchHashFromBGF(bgfMatch)

	// Check for duplicate (same format) or canonical duplicate (cross-format)
	canonicalMatchID, isCanonicalDuplicate, err := d.checkDuplicateMatchLocked(matchHash, canonicalHash)
	if err != nil {
		return 0, err
	}

	// Begin transaction for atomic import
	tx, err := d.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	// Unconditional rollback: a no-op once tx.Commit() succeeds, but guarantees
	// the transaction is released on any early return — including ctx
	// cancellation, whose error propagates through a shadowed `err` and so
	// would not trip a conditional `if err != nil` rollback.
	defer tx.Rollback()

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
		`, nameGreen, nameRed, event, location, round,
			matchLen, matchDate, filePath, len(gamesData), matchHash, canonicalHash)
		if err != nil {
			return 0, fmt.Errorf("failed to insert match: %w", err)
		}

		matchID, err = result.LastInsertId()
		if err != nil {
			return 0, fmt.Errorf("failed to get match ID: %w", err)
		}

		autoLinkTournament(tx, matchID, event)
	}

	// Per-import deduplication cache keyed by Zobrist hash.
	cache := newImportCache()

	// Process each game
	for gameIdx, gameRaw := range gamesData {
		gameData, ok := gameRaw.(map[string]interface{})
		if !ok {
			continue
		}

		// Extract game metadata
		scoreGreen := bgfGetInt(gameData, "scoreGreen")
		scoreRed := bgfGetInt(gameData, "scoreRed")
		isCrawford := bgfGetBool(gameData, "isCrawford")
		wonPoints := bgfGetInt(gameData, "wonPoints")

		// Get moves
		movesData, ok := gameData["moves"].([]interface{})
		if !ok {
			continue
		}

		// Determine game winner from wonPoints and final positions
		// Winner is determined by who won the game
		winner := int32(0) // Will be computed from match final scores

		if !isCanonicalDuplicate {
			// Insert game record
			gameResult, err := tx.Exec(`
				INSERT INTO game (match_id, game_number, initial_score_1, initial_score_2,
				                  winner, points_won, move_count)
				VALUES (?, ?, ?, ?, ?, ?, ?)
			`, matchID, gameIdx+1, scoreGreen, scoreRed,
				winner, wonPoints, len(movesData))
			if err != nil {
				return 0, fmt.Errorf("failed to insert game %d: %w", gameIdx+1, err)
			}

			gameID, err := gameResult.LastInsertId()
			if err != nil {
				return 0, fmt.Errorf("failed to get game ID: %w", err)
			}

			// Process moves - build board state as we go
			boardState := bgfInitBoardFromGame(gameData)
			cubeValue := 1
			cubeOwner := -1 // center

			moveNumber := int32(0)
			pendingCubeDouble := false // tracks if previous move was a cube double encoded as amove
			for moveIdx, moveRaw := range movesData {
				// Cancellation check at the top of every move iteration.
				if err := ctx.Err(); err != nil {
					return 0, err
				}

				moveData, ok := moveRaw.(map[string]interface{})
				if !ok {
					continue
				}

				mtype := bgfGetString(moveData, "type")
				player := bgfGetInt(moveData, "player") // -1 = Green, 1 = Red

				switch mtype {
				case "amove":
					// Check if this is a cube action encoded as amove
					// (BGBlitz uses amove with from=[-1,-1,-1,-1] and green=7 for cube actions)
					fromArr := bgfGetIntArray(moveData, "from")
					if fromArr[0] == -1 {
						if pendingCubeDouble {
							// This is the response to a pending cube double (take/pass)
							pendingCubeDouble = false
							equity := bgfGetMap(moveData, "equity")
							if equity != nil {
								cd := bgfGetMap(equity, "cubeDecision")
								if cd != nil && bgfGetBool(cd, "hasAccepted") {
									// Take - update cube state
									cubeValue *= 2
									if player == -1 {
										cubeOwner = 0 // Green takes
									} else {
										cubeOwner = 1 // Red takes
									}
								}
								// Pass: no cube state update needed (game ends)
							}
							// Don't increment moveNumber (response is part of the double action)
							continue
						}

						// This is a cube double
						pendingCubeDouble = true
						cubeAction := "Double/Take" // default
						// Look ahead for the response
						for j := moveIdx + 1; j < len(movesData); j++ {
							nextMove, ok := movesData[j].(map[string]interface{})
							if !ok {
								continue
							}
							nextFrom := bgfGetIntArray(nextMove, "from")
							if nextFrom[0] == -1 {
								eq := bgfGetMap(nextMove, "equity")
								if eq != nil {
									cd := bgfGetMap(eq, "cubeDecision")
									if cd != nil && !bgfGetBool(cd, "hasAccepted") {
										cubeAction = "Double/Pass"
									}
								}
								break
							}
							break
						}

						err := d.importBGFCubeMove(tx, gameID, moveNumber, moveData, gameData, matchLen, boardState, cubeValue, cubeOwner, isCrawford, cache, cubeAction)
						if err != nil {
							slog.Warn("failed to import BGF cube move", "game", gameIdx+1, "err", err)
						}
						moveNumber++
						continue
					}

					// Normal checker move
					err := d.importBGFCheckerMove(tx, gameID, moveNumber, moveData, gameData, matchLen, boardState, cubeValue, cubeOwner, isCrawford, cache)
					if err != nil {
						slog.Warn("failed to import BGF move", "move", moveIdx, "game", gameIdx+1, "err", err)
					}
					// Update board state - skip when green=7 (unplayable die marker)
					// BGBlitz uses green=7 to indicate that one die couldn't be played.
					// The from/to data in these moves represents analysis recommendations,
					// not actual game moves, so applying them corrupts the board state.
					greenDie := bgfGetInt(moveData, "green")
					if greenDie != 7 {
						bgfApplyCheckerMove(&boardState, moveData, player)
					}
					moveNumber++

				case "adouble":
					// Cube double - find the response (take/pass)
					cubeAction := "Double/Pass"
					for j := moveIdx + 1; j < len(movesData); j++ {
						nextMove, ok := movesData[j].(map[string]interface{})
						if !ok {
							continue
						}
						nextType := bgfGetString(nextMove, "type")
						if nextType == "atake" {
							cubeAction = "Double/Take"
							break
						} else if nextType == "apass" {
							cubeAction = "Double/Pass"
							break
						} else if nextType != "amove" {
							break
						}
					}

					err := d.importBGFCubeMove(tx, gameID, moveNumber, moveData, gameData, matchLen, boardState, cubeValue, cubeOwner, isCrawford, cache, cubeAction)
					if err != nil {
						slog.Warn("failed to import BGF cube move", "game", gameIdx+1, "err", err)
					}
					moveNumber++

				case "atake":
					// Take - update cube state
					if cubeValue == 1 {
						cubeValue = 2
					} else {
						cubeValue *= 2
					}
					// The taker becomes the cube owner
					if player == -1 {
						cubeOwner = 0 // Green
					} else {
						cubeOwner = 1 // Red
					}
					// Don't increment moveNumber (part of the double action)

				case "apass":
					// Pass/Drop - game ends, don't need to update state
					// Don't increment moveNumber

				default:
					// Skip unknown move types
					continue
				}
			}
		} else {
			// Canonical duplicate: only import analysis to existing positions
			boardState := bgfInitBoardFromGame(gameData)
			cubeValue := 1
			cubeOwner := -1
			pendingCubeDouble2 := false

			for moveIdx, moveRaw := range movesData {
				moveData, ok := moveRaw.(map[string]interface{})
				if !ok {
					continue
				}

				mtype := bgfGetString(moveData, "type")
				player := bgfGetInt(moveData, "player")

				switch mtype {
				case "amove":
					// Check if this is a cube action encoded as amove
					fromArr := bgfGetIntArray(moveData, "from")
					if fromArr[0] == -1 {
						if pendingCubeDouble2 {
							pendingCubeDouble2 = false
							equity := bgfGetMap(moveData, "equity")
							if equity != nil {
								cd := bgfGetMap(equity, "cubeDecision")
								if cd != nil && bgfGetBool(cd, "hasAccepted") {
									cubeValue *= 2
									if player == -1 {
										cubeOwner = 0
									} else {
										cubeOwner = 1
									}
								}
							}
							continue
						}
						pendingCubeDouble2 = true
						cubeAction := "Double/Take"
						for j := moveIdx + 1; j < len(movesData); j++ {
							nextMove, ok := movesData[j].(map[string]interface{})
							if !ok {
								continue
							}
							nextFrom := bgfGetIntArray(nextMove, "from")
							if nextFrom[0] == -1 {
								eq := bgfGetMap(nextMove, "equity")
								if eq != nil {
									cd := bgfGetMap(eq, "cubeDecision")
									if cd != nil && !bgfGetBool(cd, "hasAccepted") {
										cubeAction = "Double/Pass"
									}
								}
								break
							}
							break
						}
						d.importBGFCubeAnalysisOnly(tx, moveData, gameData, matchLen, boardState, cubeValue, cubeOwner, isCrawford, cache, cubeAction)
						continue
					}

					d.importBGFCheckerAnalysisOnly(tx, moveData, gameData, matchLen, boardState, cubeValue, cubeOwner, isCrawford, cache)
					// Skip board update for green=7 moves (unplayable die - analysis data only)
					greenDie := bgfGetInt(moveData, "green")
					if greenDie != 7 {
						bgfApplyCheckerMove(&boardState, moveData, player)
					}

				case "adouble":
					cubeAction := "Double/Pass"
					for j := moveIdx + 1; j < len(movesData); j++ {
						nextMove, ok := movesData[j].(map[string]interface{})
						if !ok {
							continue
						}
						nextType := bgfGetString(nextMove, "type")
						if nextType == "atake" {
							cubeAction = "Double/Take"
							break
						} else if nextType == "apass" {
							cubeAction = "Double/Pass"
							break
						}
					}
					d.importBGFCubeAnalysisOnly(tx, moveData, gameData, matchLen, boardState, cubeValue, cubeOwner, isCrawford, cache, cubeAction)

				case "atake":
					if cubeValue == 1 {
						cubeValue = 2
					} else {
						cubeValue *= 2
					}
					if player == -1 {
						cubeOwner = 0
					} else {
						cubeOwner = 1
					}
				}
			}
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	slog.Info("imported BGF match", "matchID", matchID, "games", len(gamesData), "file", filePath)
	return matchID, nil
}

// importBGFCheckerMove imports a single checker move from a BGF file
func (d *Database) importBGFCheckerMove(tx *sql.Tx, gameID int64, moveNumber int32, moveData map[string]interface{}, gameData map[string]interface{}, matchLen int, boardState [28]int, cubeValue int, cubeOwner int, isCrawford bool, cache *importCache) error {
	player := bgfGetInt(moveData, "player") // -1 = Green, 1 = Red

	// Convert BGF player to blunderDB player encoding
	// BGF: -1 = Green (first player), 1 = Red (second player)
	// blunderDB: 0 = Player 1 (Green/Black), 1 = Player 2 (Red/White)
	blunderDBPlayer := bgfPlayerToBlunderDB(player)

	// Get dice
	dieGreen := bgfGetInt(moveData, "green")
	dieRed := bgfGetInt(moveData, "red")
	die1 := dieGreen
	die2 := dieRed

	// Handle impossible dice values (green=7 appears in BGBlitz for cube actions)
	// Cube actions with from[0]==-1 are now handled in the main loop, so this
	// should only fire for edge cases. Try to infer dice from the from/to arrays.
	if die1 > 6 || die2 > 6 || die1 < 1 || die2 < 1 {
		fromArr := bgfGetIntArray(moveData, "from")
		toArr := bgfGetIntArray(moveData, "to")
		if fromArr[0] == -1 {
			// No checker move at all - skip
			return nil
		}
		// Infer dice from the move sub-moves
		var diceUsed []int
		for j := 0; j < 4; j++ {
			if fromArr[j] == -1 {
				break
			}
			f := fromArr[j]
			t := toArr[j]
			if f == 25 {
				// From bar: die = destination point
				diceUsed = append(diceUsed, t)
			} else if t == 0 {
				// Bear off: die >= distance from point to off
				diceUsed = append(diceUsed, f)
			} else {
				diff := f - t
				if diff < 0 {
					diff = -diff
				}
				diceUsed = append(diceUsed, diff)
			}
		}
		if len(diceUsed) >= 2 {
			die1 = diceUsed[0]
			die2 = diceUsed[1]
		} else if len(diceUsed) == 1 {
			// Single sub-move: one die was used, the other couldn't be played
			if die1 >= 1 && die1 <= 6 {
				die2 = diceUsed[0]
			} else if die2 >= 1 && die2 <= 6 {
				die1 = diceUsed[0]
			} else {
				die1 = diceUsed[0]
				die2 = diceUsed[0] // best guess
			}
		}
	}

	// Create board position from current state
	pos := d.createPositionFromBGF(boardState, gameData, matchLen, cubeValue, cubeOwner, isCrawford)
	pos.PlayerOnRoll = blunderDBPlayer
	pos.DecisionType = CheckerAction
	pos.Dice = [2]int{die1, die2}

	// Save position
	posID, err := d.savePositionInTxWithCache(tx, pos, cache)
	if err != nil {
		return fmt.Errorf("failed to save position: %w", err)
	}

	// Convert move to string notation
	checkerMoveStr := bgfConvertMoveToString(moveData, player)

	// Convert player to XG-style encoding for DB storage consistency
	dbPlayer := convertBlunderDBPlayerToXG(blunderDBPlayer)

	// Save move record
	moveResult, err := tx.Exec(`
		INSERT INTO move (game_id, move_number, move_type, position_id, player,
		                  dice_1, dice_2, checker_move)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, gameID, moveNumber, "checker", posID, dbPlayer, die1, die2, checkerMoveStr)
	if err != nil {
		return err
	}

	moveID, err := moveResult.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	// Save analysis if available
	moveAnalysis, ok := moveData["moveAnalysis"].([]interface{})
	if ok && len(moveAnalysis) > 0 {
		// Save to move_analysis table (first/played move)
		for _, maRaw := range moveAnalysis {
			maData, ok := maRaw.(map[string]interface{})
			if !ok {
				continue
			}
			if bgfGetBool(maData, "played") {
				err = d.saveBGFMoveAnalysisInTx(tx, moveID, maData)
				if err != nil {
					slog.Warn("failed to save BGF move analysis", "err", err)
				}
				break // Only save the played move to move_analysis
			}
		}

		// Save to position analysis table (all moves for UI compatibility)
		err = d.saveBGFCheckerAnalysisToPositionInTx(tx, posID, moveAnalysis, blunderDBPlayer, checkerMoveStr)
		if err != nil {
			slog.Warn("failed to save BGF position analysis", "err", err)
		}
	}

	// Save cube analysis from the equity field if present on a checker move
	equity := bgfGetMap(moveData, "equity")
	if equity != nil {
		cubeDecision := bgfGetMap(equity, "cubeDecision")
		if cubeDecision != nil {
			stateOnMove := bgfGetString(cubeDecision, "stateOnMove")
			if stateOnMove != "" {
				err = d.saveBGFCubeAnalysisForCheckerPositionInTx(tx, posID, equity, cubeDecision)
				if err != nil {
					slog.Warn("failed to save cube analysis for checker position", "err", err)
				}
			}
		}
	}

	return nil
}

// importBGFCubeMove imports a cube double/take/pass move from a BGF file
func (d *Database) importBGFCubeMove(tx *sql.Tx, gameID int64, moveNumber int32, moveData map[string]interface{}, gameData map[string]interface{}, matchLen int, boardState [28]int, cubeValue int, cubeOwner int, isCrawford bool, cache *importCache, cubeAction string) error {
	player := bgfGetInt(moveData, "player")
	blunderDBPlayer := bgfPlayerToBlunderDB(player)

	// Create position from current board state
	pos := d.createPositionFromBGF(boardState, gameData, matchLen, cubeValue, cubeOwner, isCrawford)
	pos.PlayerOnRoll = blunderDBPlayer
	pos.DecisionType = CubeAction
	pos.Dice = [2]int{0, 0}

	// Save position
	posID, err := d.savePositionInTxWithCache(tx, pos, cache)
	if err != nil {
		return fmt.Errorf("failed to save position: %w", err)
	}

	// Convert player to XG-style encoding for DB storage consistency
	dbPlayer := convertBlunderDBPlayerToXG(blunderDBPlayer)

	// Save move record
	moveResult, err := tx.Exec(`
		INSERT INTO move (game_id, move_number, move_type, position_id, player,
		                  dice_1, dice_2, cube_action)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, gameID, moveNumber, "cube", posID, dbPlayer, 0, 0, cubeAction)
	if err != nil {
		return err
	}

	moveID, err := moveResult.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	// Save cube analysis from equity field
	equity := bgfGetMap(moveData, "equity")
	if equity != nil {
		cubeDecision := bgfGetMap(equity, "cubeDecision")
		if cubeDecision != nil {
			err = d.saveBGFCubeMoveAnalysisInTx(tx, moveID, equity, cubeDecision)
			if err != nil {
				slog.Warn("failed to save BGF cube analysis", "err", err)
			}

			err = d.saveBGFCubeAnalysisToPositionInTx(tx, posID, equity, cubeDecision, cubeAction)
			if err != nil {
				slog.Warn("failed to save BGF position cube analysis", "err", err)
			}
		}
	}

	return nil
}

// importBGFCheckerAnalysisOnly imports only analysis for a canonical duplicate
func (d *Database) importBGFCheckerAnalysisOnly(tx *sql.Tx, moveData map[string]interface{}, gameData map[string]interface{}, matchLen int, boardState [28]int, cubeValue int, cubeOwner int, isCrawford bool, cache *importCache) {
	player := bgfGetInt(moveData, "player")
	blunderDBPlayer := bgfPlayerToBlunderDB(player)

	dieGreen := bgfGetInt(moveData, "green")
	dieRed := bgfGetInt(moveData, "red")

	pos := d.createPositionFromBGF(boardState, gameData, matchLen, cubeValue, cubeOwner, isCrawford)
	pos.PlayerOnRoll = blunderDBPlayer
	pos.DecisionType = CheckerAction
	pos.Dice = [2]int{dieGreen, dieRed}

	posID, err := d.findOrCreatePositionForCanonicalDuplicate(tx, pos, cache)
	if err != nil {
		return
	}

	moveAnalysis, ok := moveData["moveAnalysis"].([]interface{})
	if ok && len(moveAnalysis) > 0 {
		checkerMoveStr := bgfConvertMoveToString(moveData, player)
		_ = d.saveBGFCheckerAnalysisToPositionInTx(tx, posID, moveAnalysis, blunderDBPlayer, checkerMoveStr)
	}

	equity := bgfGetMap(moveData, "equity")
	if equity != nil {
		cubeDecision := bgfGetMap(equity, "cubeDecision")
		if cubeDecision != nil && bgfGetString(cubeDecision, "stateOnMove") != "" {
			_ = d.saveBGFCubeAnalysisForCheckerPositionInTx(tx, posID, equity, cubeDecision)
		}
	}
}

// importBGFCubeAnalysisOnly imports only cube analysis for a canonical duplicate
func (d *Database) importBGFCubeAnalysisOnly(tx *sql.Tx, moveData map[string]interface{}, gameData map[string]interface{}, matchLen int, boardState [28]int, cubeValue int, cubeOwner int, isCrawford bool, cache *importCache, cubeAction string) {
	player := bgfGetInt(moveData, "player")
	blunderDBPlayer := bgfPlayerToBlunderDB(player)

	pos := d.createPositionFromBGF(boardState, gameData, matchLen, cubeValue, cubeOwner, isCrawford)
	pos.PlayerOnRoll = blunderDBPlayer
	pos.DecisionType = CubeAction
	pos.Dice = [2]int{0, 0}

	posID, err := d.findOrCreatePositionForCanonicalDuplicate(tx, pos, cache)
	if err != nil {
		return
	}

	equity := bgfGetMap(moveData, "equity")
	if equity != nil {
		cubeDecision := bgfGetMap(equity, "cubeDecision")
		if cubeDecision != nil {
			_ = d.saveBGFCubeAnalysisToPositionInTx(tx, posID, equity, cubeDecision, cubeAction)
		}
	}
}

// createPositionFromBGF creates a blunderDB Position from BGF board state
func (d *Database) createPositionFromBGF(boardState [28]int, gameData map[string]interface{}, matchLen int, cubeValue int, cubeOwner int, isCrawford bool) *Position {
	scoreGreen := bgfGetInt(gameData, "scoreGreen")
	scoreRed := bgfGetInt(gameData, "scoreRed")

	// Calculate away scores (points away from winning)
	awayScore1 := matchLen - scoreGreen
	awayScore2 := matchLen - scoreRed

	if matchLen == 0 {
		awayScore1 = -1
		awayScore2 = -1
	}

	// Convert cube value to exponent for blunderDB (2^n representation)
	cubeExponent := 0
	if cubeValue > 0 {
		for v := cubeValue; v > 1; v >>= 1 {
			cubeExponent++
		}
	}

	pos := &Position{
		PlayerOnRoll: 0,
		DecisionType: CheckerAction,
		Score:        [2]int{awayScore1, awayScore2},
		Cube: Cube{
			Value: cubeExponent,
			Owner: cubeOwner,
		},
		Dice: [2]int{0, 0},
	}

	// TODO: Determine Jacoby/Beaver from match settings for money games (matchLen == 0).

	// Initialize all points as empty
	for i := 0; i < 26; i++ {
		pos.Board.Points[i] = Point{Checkers: 0, Color: -1}
	}

	// Convert BGF board encoding to blunderDB:
	// BGF board is indexed from Green's far side (24-point) to near side (1-point):
	//   BGF index 0 = Green's 24-point, BGF index 23 = Green's 1-point
	//   So: BGF index i corresponds to board point (24-i)
	// BGF index 24: Green's bar, index 25: Red's bar
	// BGF index 26: Green's borne off, index 27: Red's borne off
	// Positive = Green checkers, Negative = Red checkers
	//
	// blunderDB:
	// - Color 0 = Player 1 (Green) moves 24→1 (same as BGF Green)
	// - Color 1 = Player 2 (Red) moves 1→24
	// - Index 0 = Player 2's bar (Red/White), Index 25 = Player 1's bar (Green/Black)
	// - Index 1-24 = Points 1-24

	// Map board points (BGF index i → blunderDB point 24-i)
	for i := 0; i < 24; i++ {
		count := boardState[i]
		blunderDBPoint := 24 - i // BGF index 0 = point 24, index 23 = point 1
		if count > 0 {
			// Green checkers (positive)
			pos.Board.Points[blunderDBPoint] = Point{Checkers: count, Color: 0} // Color 0 = Green/Player 1
		} else if count < 0 {
			// Red checkers (negative)
			pos.Board.Points[blunderDBPoint] = Point{Checkers: -count, Color: 1} // Color 1 = Red/Player 2
		}
	}

	// Map bar: BGF index 24 = Green's bar → blunderDB index 25
	if boardState[24] > 0 {
		pos.Board.Points[25] = Point{Checkers: boardState[24], Color: 0}
	}
	// BGF index 25 = Red's bar → blunderDB index 0
	if boardState[25] < 0 {
		pos.Board.Points[0] = Point{Checkers: -boardState[25], Color: 1}
	} else if boardState[25] > 0 {
		// Red bar is stored as positive in some encodings
		pos.Board.Points[0] = Point{Checkers: boardState[25], Color: 1}
	}

	// Calculate bearoff
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

	return pos
}

// bgfInitBoardFromGame extracts the initial board position from a BGF game
func bgfInitBoardFromGame(gameData map[string]interface{}) [28]int {
	var board [28]int

	initial, ok := gameData["initial"].(map[string]interface{})
	if !ok {
		// Return standard starting position
		board = [28]int{2, 0, 0, 0, 0, -5, 0, -3, 0, 0, 0, 5, -5, 0, 0, 0, 3, 0, 5, 0, 0, 0, 0, -2, 0, 0, 0, 0}
		return board
	}

	points, ok := initial["points"].([]interface{})
	if !ok || len(points) < 28 {
		board = [28]int{2, 0, 0, 0, 0, -5, 0, -3, 0, 0, 0, 5, -5, 0, 0, 0, 3, 0, 5, 0, 0, 0, 0, -2, 0, 0, 0, 0}
		return board
	}

	for i := 0; i < 28 && i < len(points); i++ {
		board[i] = bgfToInt(points[i])
	}

	return board
}

// bgfApplyCheckerMove updates the board state after a BGF checker move.
// BGF from/to use 1-based point numbering from the active player's perspective:
//   - Points 1-24 = board positions (1 = player's 1-point)
//   - 25 = bar (from only)
//   - 0 = bear off (to only)
//
// Board state uses 0-based Green's perspective: indices 0-23 = points 1-24,
// 24 = Green's bar, 25 = Red's bar, 26 = Green off, 27 = Red off.
func bgfApplyCheckerMove(boardState *[28]int, moveData map[string]interface{}, player int) {
	fromArr := bgfGetIntArray(moveData, "from")
	toArr := bgfGetIntArray(moveData, "to")

	for i := 0; i < 4; i++ {
		from := fromArr[i]
		to := toArr[i]
		if from == -1 {
			break // No more submoves
		}

		// from/to come from an untrusted .bgf file. Compute the board indices
		// and skip the whole sub-move if either falls outside the 28-point
		// board, so a malformed file can't panic. Legal BGF values (0..25)
		// always map in range, so valid imports are unaffected.
		if player == -1 {
			// Green moves: Green's point N maps to board index (24-N)
			// BGF board: index 0 = Green's 24-point, index 23 = Green's 1-point
			// Green moves in decreasing direction (24→1→off)
			fromIdx := 24 // Green's bar
			if from != 25 {
				fromIdx = 24 - from // Green's point N → board index (24-N)
			}
			toIdx := 26 // bear-off slot used when to == 0
			if to != 0 {
				toIdx = 24 - to // Green's point N → board index (24-N)
			}
			if fromIdx < 0 || fromIdx >= 28 || toIdx < 0 || toIdx >= 28 {
				continue
			}

			// Remove checker from source
			boardState[fromIdx]--

			if to == 0 {
				// Bear off
				boardState[26]++
			} else {
				// Check for hit
				if boardState[toIdx] < 0 {
					// Hit Red checker - move it to Red's bar
					boardState[25] += boardState[toIdx] // boardState[toIdx] is negative, so this decrements
					boardState[toIdx] = 0
				}
				boardState[toIdx]++
			}
		} else {
			// Red moves: Red's point N maps to board index (N-1)
			// Red's point N = Green's point (25-N) = board index 24-(25-N) = N-1
			// Red moves in increasing direction (from Green's perspective)
			fromIdx := 25 // Red's bar
			if from != 25 {
				fromIdx = from - 1 // Red's point N → board index (N-1)
			}
			toIdx := 27 // bear-off slot used when to == 0
			if to != 0 {
				toIdx = to - 1 // Red's point N → board index (N-1)
			}
			if fromIdx < 0 || fromIdx >= 28 || toIdx < 0 || toIdx >= 28 {
				continue
			}

			// Remove checker from source (Red checkers are negative)
			boardState[fromIdx]++

			if to == 0 {
				// Bear off
				boardState[27]--
			} else {
				// Check for hit
				if boardState[toIdx] > 0 {
					// Hit Green checker - move it to Green's bar
					boardState[24] += boardState[toIdx]
					boardState[toIdx] = 0
				}
				boardState[toIdx]--
			}
		}
	}
}

// bgfConvertMoveToString converts BGF move from/to arrays to standard notation.
// BGF from/to are 1-based from the active player's perspective (25=bar, 0=off).
func bgfConvertMoveToString(moveData map[string]interface{}, player int) string {
	fromArr := bgfGetIntArray(moveData, "from")
	toArr := bgfGetIntArray(moveData, "to")

	if fromArr[0] == -1 {
		return "" // No move (fanned/dance)
	}

	type submove struct {
		from int
		to   int
	}

	moves := make([]submove, 0, 4)
	for i := 0; i < 4; i++ {
		if fromArr[i] == -1 {
			break
		}
		// from/to are already 1-based from player perspective
		from := fromArr[i]
		to := toArr[i]

		// from=25 means bar, keep as 25
		// to=0 means bear off, keep as 0

		moves = append(moves, submove{from, to})
	}

	if len(moves) == 0 {
		return ""
	}

	// Sort by source point descending
	sort.Slice(moves, func(i, j int) bool {
		return moves[i].from > moves[j].from
	})

	// Group identical moves
	var parts []string
	i := 0
	for i < len(moves) {
		count := 1
		for i+count < len(moves) && moves[i+count].from == moves[i].from && moves[i+count].to == moves[i].to {
			count++
		}

		fromStr := fmt.Sprintf("%d", moves[i].from)
		if moves[i].from == 25 {
			fromStr = "bar"
		}

		toStr := fmt.Sprintf("%d", moves[i].to)
		if moves[i].to == 0 {
			toStr = "off"
		}

		if count > 1 {
			parts = append(parts, fmt.Sprintf("%s/%s(%d)", fromStr, toStr, count))
		} else {
			parts = append(parts, fmt.Sprintf("%s/%s", fromStr, toStr))
		}

		i += count
	}

	return strings.Join(parts, " ")
}

// saveBGFMoveAnalysisInTx saves BGF move analysis to the move_analysis table
func (d *Database) saveBGFMoveAnalysisInTx(tx *sql.Tx, moveID int64, maData map[string]interface{}) error {
	eq := bgfGetMap(maData, "eq")
	if eq == nil {
		return nil
	}

	ply := bgfGetInt(maData, "ply")

	equity := bgfGetFloat(eq, "emg")
	if !bgfGetBool(eq, "hasEMG") {
		equity = bgfGetFloat(eq, "equity")
	}

	return insertMoveAnalysisRow(tx, moveAnalysisRow{
		MoveID:                 moveID,
		AnalysisType:           "checker",
		Depth:                  translateBGFAnalysisDepth(ply),
		Equity:                 int64(math.Round(equity * 1000)),
		WinRate:                int64(math.Round(bgfGetFloat(eq, "myWins") * 100.0 * 100)),
		GammonRate:             int64(math.Round(bgfGetFloat(eq, "myGammon") * 100.0 * 100)),
		BackgammonRate:         int64(math.Round(bgfGetFloat(eq, "myBackGammon") * 100.0 * 100)),
		OpponentWinRate:        int64(math.Round(bgfGetFloat(eq, "oppWins") * 100.0 * 100)),
		OpponentGammonRate:     int64(math.Round(bgfGetFloat(eq, "oppGammon") * 100.0 * 100)),
		OpponentBackgammonRate: int64(math.Round(bgfGetFloat(eq, "oppBackGammon") * 100.0 * 100)),
	})
}

// saveBGFCheckerAnalysisToPositionInTx saves BGF checker analysis to position analysis table
func (d *Database) saveBGFCheckerAnalysisToPositionInTx(tx *sql.Tx, positionID int64, moveAnalysis []interface{}, blunderDBPlayer int, playedMoveStr string) error {
	if len(moveAnalysis) == 0 {
		return nil
	}

	posAnalysis := PositionAnalysis{
		PositionID:            int(positionID),
		AnalysisType:          "CheckerMove",
		AnalysisEngineVersion: "BGBlitz",
		CreationDate:          time.Now(),
		LastModifiedDate:      time.Now(),
	}

	checkerMoves := make([]CheckerMove, 0, len(moveAnalysis))
	var bestEquity float64

	for i, maRaw := range moveAnalysis {
		maData, ok := maRaw.(map[string]interface{})
		if !ok {
			continue
		}

		eq := bgfGetMap(maData, "eq")
		if eq == nil {
			continue
		}

		ply := bgfGetInt(maData, "ply")
		played := bgfGetBool(maData, "played")

		equity := bgfGetFloat(eq, "emg")
		if !bgfGetBool(eq, "hasEMG") {
			equity = bgfGetFloat(eq, "equity")
		}

		if i == 0 {
			bestEquity = equity
		}

		// Convert move from analysis
		moveStr := ""
		moveInfo := bgfGetMap(maData, "move")
		if moveInfo != nil {
			moveStr = bgfConvertAnalysisMoveToString(moveInfo)
		}

		var equityError *float64
		if i > 0 {
			diff := bestEquity - equity
			equityError = &diff
		}

		checkerMove := CheckerMove{
			Index:                    i,
			AnalysisDepth:            translateBGFAnalysisDepth(ply),
			AnalysisEngine:           "BGBlitz",
			Move:                     moveStr,
			Equity:                   equity,
			EquityError:              equityError,
			PlayerWinChance:          bgfGetFloat(eq, "myWins") * 100.0,
			PlayerGammonChance:       bgfGetFloat(eq, "myGammon") * 100.0,
			PlayerBackgammonChance:   bgfGetFloat(eq, "myBackGammon") * 100.0,
			OpponentWinChance:        bgfGetFloat(eq, "oppWins") * 100.0,
			OpponentGammonChance:     bgfGetFloat(eq, "oppGammon") * 100.0,
			OpponentBackgammonChance: bgfGetFloat(eq, "oppBackGammon") * 100.0,
		}
		checkerMoves = append(checkerMoves, checkerMove)

		_ = played
	}

	posAnalysis.CheckerAnalysis = &CheckerAnalysis{
		Moves: checkerMoves,
	}

	if playedMoveStr != "" {
		posAnalysis.PlayedMoves = []string{playedMoveStr}
	}

	return d.saveAnalysisInTx(tx, positionID, posAnalysis)
}

// saveBGFCubeMoveAnalysisInTx saves BGF cube analysis to move_analysis table
func (d *Database) saveBGFCubeMoveAnalysisInTx(tx *sql.Tx, moveID int64, equity map[string]interface{}, cubeDecision map[string]interface{}) error {
	ply := 0 // Cube analysis doesn't have a ply field in BGF, use the move level
	if pVal, ok := equity["ply"]; ok {
		ply = bgfToInt(pVal)
	}

	return insertMoveAnalysisRow(tx, moveAnalysisRow{
		MoveID:                 moveID,
		AnalysisType:           "cube",
		Depth:                  translateBGFAnalysisDepth(ply),
		Equity:                 int64(math.Round(bgfGetFloat(cubeDecision, "eqNoDouble") * 1000)),
		WinRate:                int64(math.Round(bgfGetFloat(equity, "myWins") * 100.0 * 100)),
		GammonRate:             int64(math.Round(bgfGetFloat(equity, "myGammon") * 100.0 * 100)),
		BackgammonRate:         int64(math.Round(bgfGetFloat(equity, "myBackGammon") * 100.0 * 100)),
		OpponentWinRate:        int64(math.Round(bgfGetFloat(equity, "oppWins") * 100.0 * 100)),
		OpponentGammonRate:     int64(math.Round(bgfGetFloat(equity, "oppGammon") * 100.0 * 100)),
		OpponentBackgammonRate: int64(math.Round(bgfGetFloat(equity, "oppBackGammon") * 100.0 * 100)),
	})
}

// saveBGFCubeAnalysisToPositionInTx saves BGF cube analysis to position analysis table
func (d *Database) saveBGFCubeAnalysisToPositionInTx(tx *sql.Tx, positionID int64, equity map[string]interface{}, cubeDecision map[string]interface{}, playedCubeAction string) error {
	posAnalysis := PositionAnalysis{
		PositionID:            int(positionID),
		AnalysisType:          "DoublingCube",
		AnalysisEngineVersion: "BGBlitz",
		CreationDate:          time.Now(),
		LastModifiedDate:      time.Now(),
	}

	cubefulNoDouble := bgfGetFloat(cubeDecision, "eqNoDouble")
	cubefulDoubleTake := bgfGetFloat(cubeDecision, "eqDoubleTake")
	cubefulDoublePass := bgfGetFloat(cubeDecision, "eqDoublePass")
	cubelessEquity := bgfGetFloat(cubeDecision, "eqCubeLess")
	_ = bgfGetFloat(cubeDecision, "eqCubeFul") // cubefulEquity available but not directly used

	cubeAnalysis := buildDoublingCubeAnalysis(cubeAnalysisParams{
		Depth:                     "2-ply",
		Engine:                    "BGBlitz",
		PlayerWinChances:          bgfGetFloat(equity, "myWins") * 100.0,
		PlayerGammonChances:       bgfGetFloat(equity, "myGammon") * 100.0,
		PlayerBackgammonChances:   bgfGetFloat(equity, "myBackGammon") * 100.0,
		OpponentWinChances:        bgfGetFloat(equity, "oppWins") * 100.0,
		OpponentGammonChances:     bgfGetFloat(equity, "oppGammon") * 100.0,
		OpponentBackgammonChances: bgfGetFloat(equity, "oppBackGammon") * 100.0,
		CubelessNoDoubleEquity:    cubelessEquity,
		CubelessDoubleEquity:      cubelessEquity,
		CubefulNoDoubleEquity:     cubefulNoDouble,
		CubefulDoubleTakeEquity:   cubefulDoubleTake,
		CubefulDoublePassEquity:   cubefulDoublePass,
	})

	// Override best action from BGF stateOnMove/stateOther when available
	stateOnMove := bgfGetString(cubeDecision, "stateOnMove")
	stateOther := bgfGetString(cubeDecision, "stateOther")
	if stateOnMove == "DOUBLE" || stateOnMove == "REDOUBLE" {
		if stateOther == "ACCEPT" {
			cubeAnalysis.BestCubeAction = "Double, Take"
		} else if stateOther == "REJECT" {
			cubeAnalysis.BestCubeAction = "Double, Pass"
		}
	} else if stateOnMove == "TO_GOOD" || stateOnMove == "NO_DOUBLE" {
		cubeAnalysis.BestCubeAction = "No Double"
	}

	posAnalysis.DoublingCubeAnalysis = &cubeAnalysis

	if playedCubeAction != "" {
		posAnalysis.PlayedCubeActions = []string{playedCubeAction}
	}

	return d.saveAnalysisInTx(tx, positionID, posAnalysis)
}

// saveBGFCubeAnalysisForCheckerPositionInTx saves cube analysis from equity field to a checker position
func (d *Database) saveBGFCubeAnalysisForCheckerPositionInTx(tx *sql.Tx, positionID int64, equity map[string]interface{}, cubeDecision map[string]interface{}) error {
	// Build a PositionAnalysis with just the cube analysis and let saveAnalysisInTx handle merging
	posAnalysis := PositionAnalysis{
		PositionID:            int(positionID),
		AnalysisType:          "CheckerMove",
		AnalysisEngineVersion: "BGBlitz",
		CreationDate:          time.Now(),
		LastModifiedDate:      time.Now(),
	}

	cubeAnalysis := buildDoublingCubeAnalysis(cubeAnalysisParams{
		Depth:                     "2-ply",
		Engine:                    "BGBlitz",
		PlayerWinChances:          bgfGetFloat(equity, "myWins") * 100.0,
		PlayerGammonChances:       bgfGetFloat(equity, "myGammon") * 100.0,
		PlayerBackgammonChances:   bgfGetFloat(equity, "myBackGammon") * 100.0,
		OpponentWinChances:        bgfGetFloat(equity, "oppWins") * 100.0,
		OpponentGammonChances:     bgfGetFloat(equity, "oppGammon") * 100.0,
		OpponentBackgammonChances: bgfGetFloat(equity, "oppBackGammon") * 100.0,
		CubelessNoDoubleEquity:    bgfGetFloat(cubeDecision, "eqCubeLess"),
		CubelessDoubleEquity:      bgfGetFloat(cubeDecision, "eqCubeLess"),
		CubefulNoDoubleEquity:     bgfGetFloat(cubeDecision, "eqNoDouble"),
		CubefulDoubleTakeEquity:   bgfGetFloat(cubeDecision, "eqDoubleTake"),
		CubefulDoublePassEquity:   bgfGetFloat(cubeDecision, "eqDoublePass"),
	})

	posAnalysis.DoublingCubeAnalysis = &cubeAnalysis

	return d.saveAnalysisInTx(tx, positionID, posAnalysis)
}

// bgfConvertAnalysisMoveToString converts a BGF move from analysis entry to string notation
// bgfConvertAnalysisMoveToString converts a BGF move from analysis entry to string notation.
// BGF from/to are 1-based from the active player's perspective (25=bar, 0=off).
func bgfConvertAnalysisMoveToString(moveInfo map[string]interface{}) string {
	fromArr := bgfGetIntArray(moveInfo, "from")
	toArr := bgfGetIntArray(moveInfo, "to")

	if fromArr[0] == -1 {
		return ""
	}

	type submove struct {
		from int
		to   int
	}

	moves := make([]submove, 0, 4)
	for i := 0; i < 4; i++ {
		if fromArr[i] == -1 {
			break
		}
		// from/to are already 1-based from player perspective
		from := fromArr[i]
		to := toArr[i]

		moves = append(moves, submove{from, to})
	}

	if len(moves) == 0 {
		return ""
	}

	sort.Slice(moves, func(i, j int) bool {
		return moves[i].from > moves[j].from
	})

	var parts []string
	i := 0
	for i < len(moves) {
		count := 1
		for i+count < len(moves) && moves[i+count].from == moves[i].from && moves[i+count].to == moves[i].to {
			count++
		}

		fromStr := fmt.Sprintf("%d", moves[i].from)
		if moves[i].from == 25 {
			fromStr = "bar"
		}
		toStr := fmt.Sprintf("%d", moves[i].to)
		if moves[i].to == 0 {
			toStr = "off"
		}

		if count > 1 {
			parts = append(parts, fmt.Sprintf("%s/%s(%d)", fromStr, toStr, count))
		} else {
			parts = append(parts, fmt.Sprintf("%s/%s", fromStr, toStr))
		}
		i += count
	}

	return strings.Join(parts, " ")
}

// translateBGFAnalysisDepth converts BGF ply level to human-readable string
func translateBGFAnalysisDepth(ply int) string {
	if ply > 0 {
		return fmt.Sprintf("%d-ply", ply)
	}
	return "0-ply"
}

// ComputeBGFMatchHash generates a unique hash for a BGF match for duplicate detection
func ComputeBGFMatchHash(match *bgfparser.Match) string {
	var hashBuilder strings.Builder

	data := match.Data
	p1 := strings.TrimSpace(strings.ToLower(bgfGetString(data, "nameGreen")))
	p2 := strings.TrimSpace(strings.ToLower(bgfGetString(data, "nameRed")))
	matchLen := bgfGetInt(data, "matchlen")
	hashBuilder.WriteString(fmt.Sprintf("bgf:%s|%s|%d|", p1, p2, matchLen))

	gamesData, _ := data["games"].([]interface{})
	for gameIdx, gameRaw := range gamesData {
		g, ok := gameRaw.(map[string]interface{})
		if !ok {
			continue
		}
		hashBuilder.WriteString(fmt.Sprintf("g%d:%d,%d,%d|",
			gameIdx, bgfGetInt(g, "scoreGreen"), bgfGetInt(g, "scoreRed"), bgfGetInt(g, "wonPoints")))

		movesData, _ := g["moves"].([]interface{})
		for moveIdx, moveRaw := range movesData {
			m, ok := moveRaw.(map[string]interface{})
			if !ok {
				continue
			}
			mtype := bgfGetString(m, "type")
			hashBuilder.WriteString(fmt.Sprintf("m%d:%s,", moveIdx, mtype))
			if mtype == "amove" {
				d1 := bgfGetInt(m, "green")
				d2 := bgfGetInt(m, "red")
				hashBuilder.WriteString(fmt.Sprintf("d%d%d|", d1, d2))
			} else if mtype == "adouble" || mtype == "atake" || mtype == "apass" {
				hashBuilder.WriteString(fmt.Sprintf("c%s|", mtype))
			}
		}
	}

	hash := sha256.Sum256([]byte(hashBuilder.String()))
	return hex.EncodeToString(hash[:])
}

// ComputeCanonicalMatchHashFromBGF computes a format-independent match hash from BGF data.
// Must produce the same hash as ComputeCanonicalMatchHashFromXG for the same match.
// Uses only the first N dice per game for cross-format compatibility.
func ComputeCanonicalMatchHashFromBGF(match *bgfparser.Match) string {
	var hashBuilder strings.Builder

	data := match.Data
	p1 := strings.TrimSpace(strings.ToLower(bgfGetString(data, "nameGreen")))
	p2 := strings.TrimSpace(strings.ToLower(bgfGetString(data, "nameRed")))
	matchLen := bgfGetInt(data, "matchlen")

	if p1 > p2 {
		p1, p2 = p2, p1
	}

	gamesData, _ := data["games"].([]interface{})
	hashBuilder.WriteString(fmt.Sprintf("canonical2:%s|%s|%d|%d|", p1, p2, matchLen, len(gamesData)))

	for gameIdx, gameRaw := range gamesData {
		g, ok := gameRaw.(map[string]interface{})
		if !ok {
			continue
		}
		hashBuilder.WriteString(fmt.Sprintf("g%d|", gameIdx))

		diceCount := 0
		movesData, _ := g["moves"].([]interface{})
		for _, moveRaw := range movesData {
			if diceCount >= maxCanonicalDicePerGame {
				break
			}
			m, ok := moveRaw.(map[string]interface{})
			if !ok {
				continue
			}
			mtype := bgfGetString(m, "type")
			if mtype == "amove" {
				// Skip cube actions encoded as amove (from[0] == -1)
				fromArr := bgfGetIntArray(m, "from")
				if len(fromArr) > 0 && fromArr[0] == -1 {
					continue
				}
				d1 := bgfGetInt(m, "green")
				d2 := bgfGetInt(m, "red")
				if d1 > d2 {
					d1, d2 = d2, d1
				}
				hashBuilder.WriteString(fmt.Sprintf("d%d%d|", d1, d2))
				diceCount++
			}
		}
	}

	hash := sha256.Sum256([]byte(hashBuilder.String()))
	return hex.EncodeToString(hash[:])
}

// ImportBGFPosition imports a single BGBlitz position from a TXT file
func (d *Database) ImportBGFPosition(filePath string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	pos, err := bgfparser.ParseTXT(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to parse BGBlitz position file: %w", err)
	}

	return d.saveBGFPositionWithAnalysis(pos)
}

// ImportBGFPositionFromText imports a BGBlitz position from text content (clipboard/string)
func (d *Database) ImportBGFPositionFromText(content string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	pos, err := bgfparser.ParseTXTFromReader(strings.NewReader(content))
	if err != nil {
		return 0, fmt.Errorf("failed to parse BGBlitz position text: %w", err)
	}

	return d.saveBGFPositionWithAnalysis(pos)
}

// saveBGFPositionWithAnalysis converts a bgfparser.Position to blunderDB Position and saves it
func (d *Database) saveBGFPositionWithAnalysis(bgfPos *bgfparser.Position) (int64, error) {
	// Convert bgfparser.Position to blunderDB Position
	pos := d.convertBGFTextPosition(bgfPos)

	// Save position to database (inline, since caller already holds the mutex)
	normalizedPosition := pos.NormalizeForStorage()
	compactState := encodeBoardCompact(normalizedPosition.Board)

	cols := populatePositionColumns(pos)
	noContactInt := 0
	if cols.NoContact {
		noContactInt = 1
	}

	tx, err := d.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.Exec(`
		INSERT INTO position (
			zobrist_hash, decision_type, player_on_roll, dice_1, dice_2,
			cube_value, cube_owner, score_1, score_2,
			has_jacoby, has_beaver,
			pip_1, pip_2, pip_diff, off_1, off_2,
			back_checkers_1, back_checkers_2, no_contact,
			occupancy_1, occupancy_2, point_mask_1, point_mask_2,
			state
		) VALUES (?,?,?,?,?, ?,?,?,?,  ?,?,  ?,?,?,?,?,  ?,?,?,  ?,?,?,?,  ?)`,
		int64(cols.ZobristHash), cols.DecisionType, normalizedPosition.PlayerOnRoll, cols.Dice1, cols.Dice2,
		cols.CubeValue, cols.CubeOwner, cols.Score1, cols.Score2,
		cols.HasJacoby, cols.HasBeaver,
		cols.Pip1, cols.Pip2, cols.PipDiff, cols.Off1, cols.Off2,
		cols.BackCheckers1, cols.BackCheckers2, noContactInt,
		int64(cols.Occupancy1), int64(cols.Occupancy2), int64(cols.PointMask1), int64(cols.PointMask2),
		compactState)
	if err != nil {
		return 0, fmt.Errorf("failed to insert position: %w", err)
	}

	positionID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get position ID: %w", err)
	}

	// Save checker evaluation analysis if available
	if len(bgfPos.Evaluations) > 0 {
		posAnalysis := PositionAnalysis{
			PositionID:            int(positionID),
			XGID:                  bgfPos.XGID,
			Player1:               bgfPos.PlayerX,
			Player2:               bgfPos.PlayerO,
			AnalysisType:          "CheckerMove",
			AnalysisEngineVersion: "BGBlitz",
			CreationDate:          time.Now(),
			LastModifiedDate:      time.Now(),
		}

		checkerMoves := make([]CheckerMove, 0, len(bgfPos.Evaluations))
		for i, eval := range bgfPos.Evaluations {
			var equityError *float64
			if i > 0 {
				diff := bgfPos.Evaluations[0].Equity - eval.Equity
				equityError = &diff
			}

			checkerMove := CheckerMove{
				Index:                    i,
				AnalysisDepth:            "2-ply", // BGBlitz TXT files don't specify ply, default to 2-ply
				AnalysisEngine:           "BGBlitz",
				Move:                     eval.Move,
				Equity:                   eval.Equity,
				EquityError:              equityError,
				PlayerWinChance:          eval.Win * 100.0,
				PlayerGammonChance:       eval.WinG * 100.0,
				PlayerBackgammonChance:   eval.WinBG * 100.0,
				OpponentWinChance:        (1.0 - eval.Win) * 100.0,
				OpponentGammonChance:     eval.LoseG * 100.0,
				OpponentBackgammonChance: eval.LoseBG * 100.0,
			}
			checkerMoves = append(checkerMoves, checkerMove)
		}

		posAnalysis.CheckerAnalysis = &CheckerAnalysis{
			Moves: checkerMoves,
		}

		roundAnalysisForStorage(&posAnalysis)
		analysisData, err := encodeAnalysisForStorage(&posAnalysis)
		if err != nil {
			return 0, fmt.Errorf("failed to marshal checker analysis: %w", err)
		}
		_, err = tx.Exec(`INSERT INTO analysis (position_id, data) VALUES (?, ?)`, positionID, analysisData)
		if err != nil {
			return 0, fmt.Errorf("failed to save checker analysis for BGBlitz position: %w", err)
		}
	}

	// Save cube decision analysis if available
	if len(bgfPos.CubeDecisions) > 0 {
		posAnalysis := PositionAnalysis{
			PositionID:            int(positionID),
			XGID:                  bgfPos.XGID,
			Player1:               bgfPos.PlayerX,
			Player2:               bgfPos.PlayerO,
			AnalysisType:          "DoublingCube",
			AnalysisEngineVersion: "BGBlitz",
			CreationDate:          time.Now(),
			LastModifiedDate:      time.Now(),
		}

		// Find the best action using multilingual classifier
		var noDouble, doubleTake, doublePass *bgfparser.CubeDecision
		for i := range bgfPos.CubeDecisions {
			cd := &bgfPos.CubeDecisions[i]
			switch classifyBGFCubeAction(cd.Action) {
			case "nodbl":
				noDouble = cd
			case "take":
				doubleTake = cd
			case "pass":
				doublePass = cd
			}
		}

		cubefulNoDouble := 0.0
		cubefulDoubleTake := 0.0
		cubefulDoublePass := 1.0 // Default pass equity

		if noDouble != nil {
			cubefulNoDouble = noDouble.EMG
		}
		if doubleTake != nil {
			cubefulDoubleTake = doubleTake.EMG
		}
		if doublePass != nil {
			cubefulDoublePass = doublePass.EMG
		}

		bestEquity, bestAction := computeBestCubeAction(cubefulNoDouble, cubefulDoubleTake, cubefulDoublePass)

		cubeAnalysis := DoublingCubeAnalysis{
			AnalysisDepth:           "2-ply",
			AnalysisEngine:          "BGBlitz",
			CubelessNoDoubleEquity:  bgfPos.CubelessEquity,
			CubelessDoubleEquity:    bgfPos.CubelessEquity,
			CubefulNoDoubleEquity:   cubefulNoDouble,
			CubefulNoDoubleError:    cubefulNoDouble - bestEquity,
			CubefulDoubleTakeEquity: cubefulDoubleTake,
			CubefulDoubleTakeError:  cubefulDoubleTake - bestEquity,
			CubefulDoublePassEquity: cubefulDoublePass,
			CubefulDoublePassError:  cubefulDoublePass - bestEquity,
			BestCubeAction:          bestAction,
		}

		posAnalysis.DoublingCubeAnalysis = &cubeAnalysis

		roundAnalysisForStorage(&posAnalysis)
		analysisData, err := encodeAnalysisForStorage(&posAnalysis)
		if err != nil {
			return 0, fmt.Errorf("failed to marshal cube analysis: %w", err)
		}
		_, err = tx.Exec(`INSERT INTO analysis (position_id, data) VALUES (?, ?)`, positionID, analysisData)
		if err != nil {
			return 0, fmt.Errorf("failed to save cube analysis for BGBlitz position: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit position with analysis: %w", err)
	}
	return positionID, nil
}

// ImportXGPPosition imports an XG position file (.xgp) as a standalone position with analysis.
// XGP files use the same binary format as .xg match files but contain a single position.
func (d *Database) ImportXGPPosition(filePath string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Parse the xgp file using the standard XG parser
	match, err := xgparser.ParseXGFromFile(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to parse XGP file: %w", err)
	}

	if len(match.Games) == 0 || len(match.Games[0].Moves) == 0 {
		return 0, fmt.Errorf("XGP file contains no position data")
	}

	game := &match.Games[0]
	move := &game.Moves[0]

	// Determine position type and create blunderDB position
	var pos *Position

	if move.MoveType == "checker" && move.CheckerMove != nil {
		pos, err = d.createPositionFromXG(move.CheckerMove.Position, game, match.Metadata.MatchLength, 0, move.CheckerMove.ActivePlayer)
		if err != nil {
			return 0, fmt.Errorf("failed to create position from XGP: %w", err)
		}
		pos.PlayerOnRoll = convertXGPlayerToBlunderDB(move.CheckerMove.ActivePlayer)
		pos.DecisionType = CheckerAction
		pos.Dice = [2]int{int(move.CheckerMove.Dice[0]), int(move.CheckerMove.Dice[1])}
	} else if move.MoveType == "cube" && move.CubeMove != nil {
		pos, err = d.createPositionFromXG(move.CubeMove.Position, game, match.Metadata.MatchLength, 0, move.CubeMove.ActivePlayer)
		if err != nil {
			return 0, fmt.Errorf("failed to create position from XGP: %w", err)
		}
		pos.PlayerOnRoll = convertXGPlayerToBlunderDB(move.CubeMove.ActivePlayer)
		pos.DecisionType = CubeAction
		pos.Dice = [2]int{0, 0}
	} else {
		return 0, fmt.Errorf("XGP file contains unsupported move type: %s", move.MoveType)
	}

	// Save position to database using the proper column-populating path.
	tx, err := d.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	positionID, err := d.savePositionInTx(tx, pos)
	if err != nil {
		return 0, fmt.Errorf("failed to save position: %w", err)
	}

	// Save analysis
	if move.MoveType == "checker" && move.CheckerMove != nil && len(move.CheckerMove.Analysis) > 0 {
		err = d.saveCheckerAnalysisToPositionInTx(tx, positionID, move.CheckerMove.Analysis,
			&move.CheckerMove.Position, move.CheckerMove.ActivePlayer, &move.CheckerMove.PlayedMove)
		if err != nil {
			slog.Warn("failed to save checker analysis for XGP position", "err", err)
		}
	} else if move.MoveType == "cube" && move.CubeMove != nil && move.CubeMove.Analysis != nil {
		err = d.saveCubeAnalysisToPositionInTx(tx, positionID, move.CubeMove.Analysis, d.convertCubeAction(move.CubeMove.CubeAction))
		if err != nil {
			slog.Warn("failed to save cube analysis for XGP position", "err", err)
		}
	}

	// If there's also a second move (e.g., checker move following a cube decision),
	// save that analysis too on the same position
	if len(game.Moves) > 1 {
		secondMove := &game.Moves[1]
		if secondMove.MoveType == "checker" && secondMove.CheckerMove != nil && len(secondMove.CheckerMove.Analysis) > 0 {
			// Create the checker position to store as a separate position
			checkerPos, err := d.createPositionFromXG(secondMove.CheckerMove.Position, game, match.Metadata.MatchLength, 1, secondMove.CheckerMove.ActivePlayer)
			if err == nil {
				checkerPos.PlayerOnRoll = convertXGPlayerToBlunderDB(secondMove.CheckerMove.ActivePlayer)
				checkerPos.DecisionType = CheckerAction
				checkerPos.Dice = [2]int{int(secondMove.CheckerMove.Dice[0]), int(secondMove.CheckerMove.Dice[1])}

				checkerPosID, err := d.savePositionInTx(tx, checkerPos)
				if err == nil {
					_ = d.saveCheckerAnalysisToPositionInTx(tx, checkerPosID, secondMove.CheckerMove.Analysis,
						&secondMove.CheckerMove.Position, secondMove.CheckerMove.ActivePlayer, &secondMove.CheckerMove.PlayedMove)
				}
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit XGP position: %w", err)
	}

	slog.Info("imported XGP position", "positionID", positionID, "file", filePath)
	return positionID, nil
}

// convertBGFTextPosition converts a bgfparser.Position from TXT format to blunderDB Position
func (d *Database) convertBGFTextPosition(bgfPos *bgfparser.Position) *Position {
	pos := &Position{
		PlayerOnRoll: 0,
		DecisionType: CheckerAction,
	}

	// Convert board from bgfparser encoding to blunderDB
	// bgfparser Board[26]:
	//   Index 0: (unused or bar-like)
	//   Index 1-24: Points 1-24 (positive=X/Green, negative=O/Red)
	//   Index 25: (unused or bar-like)
	// blunderDB:
	//   Index 0: Player 2's bar (Red/White)
	//   Index 1-24: Points 1-24
	//   Index 25: Player 1's bar (Green/Black)

	for i := 0; i < 26; i++ {
		pos.Board.Points[i] = Point{Checkers: 0, Color: -1}
	}

	// Map points 1-24
	for i := 1; i <= 24; i++ {
		count := bgfPos.Board[i]
		if count > 0 {
			pos.Board.Points[i] = Point{Checkers: count, Color: 0} // Green = Color 0
		} else if count < 0 {
			pos.Board.Points[i] = Point{Checkers: -count, Color: 1} // Red = Color 1
		}
	}

	// Map bars from OnBar map
	if bgfPos.OnBar != nil {
		if xBar, ok := bgfPos.OnBar["X"]; ok && xBar > 0 {
			pos.Board.Points[25] = Point{Checkers: xBar, Color: 0} // Green bar
		}
		if oBar, ok := bgfPos.OnBar["O"]; ok && oBar > 0 {
			pos.Board.Points[0] = Point{Checkers: oBar, Color: 1} // Red bar
		}
	}

	// Calculate bearoff
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

	// Set player on roll
	if bgfPos.OnRoll == "O" {
		pos.PlayerOnRoll = 1
	} else {
		pos.PlayerOnRoll = 0
	}

	// Set dice
	pos.Dice = [2]int{bgfPos.Dice[0], bgfPos.Dice[1]}

	// Set cube
	cubeExponent := 0
	if bgfPos.CubeValue > 0 {
		for v := bgfPos.CubeValue; v > 1; v >>= 1 {
			cubeExponent++
		}
	}
	pos.Cube.Value = cubeExponent

	switch bgfPos.CubeOwner {
	case "X":
		pos.Cube.Owner = 0 // Green owns
	case "O":
		pos.Cube.Owner = 1 // Red owns
	default:
		pos.Cube.Owner = -1 // Center
	}

	// Set scores (away scores)
	if bgfPos.MatchLength > 0 {
		pos.Score = [2]int{bgfPos.MatchLength - bgfPos.ScoreX, bgfPos.MatchLength - bgfPos.ScoreO}
	} else {
		pos.Score = [2]int{-1, -1} // Unlimited
	}

	// Decision type based on available analysis
	if len(bgfPos.CubeDecisions) > 0 && len(bgfPos.Evaluations) == 0 {
		pos.DecisionType = CubeAction
		pos.Dice = [2]int{0, 0}
	}

	return pos
}

// ============================================================================
// BGF helper functions for extracting typed values from map[string]interface{}
// ============================================================================

// bgfPlayerToBlunderDB converts BGF player encoding to blunderDB encoding
// BGF: -1 = Green (first player), 1 = Red (second player)
// blunderDB: 0 = Player 1 (Green/Black), 1 = Player 2 (Red/White)
func bgfPlayerToBlunderDB(bgfPlayer int) int {
	if bgfPlayer == -1 {
		return 0 // Green = Player 1
	}
	return 1 // Red = Player 2
}

func bgfGetString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// classifyBGFCubeAction classifies a cube decision action string into one of
// "nodbl" (No Double / No Redouble), "take" (Double/Take, Redouble/Take),
// or "pass" (Double/Pass, Redouble/Pass).
// Handles multilingual action strings from BGBlitz text export (EN, FR, DE, JP, etc.).
func classifyBGFCubeAction(action string) string {
	action = strings.ToLower(strings.TrimSpace(action))

	// No Double / No Redouble patterns
	noDoublePatterns := []string{
		"no double", "no redouble", // English
		"pas de double", "pas de redouble", // French
		"kein doppel", "kein redoppel", // German
		"\u30c0\u30d6\u30eb\u305b\u305a", // Japanese: ダブルせず
	}
	for _, p := range noDoublePatterns {
		if strings.Contains(action, p) {
			return "nodbl"
		}
	}

	// Double/Take patterns (check before pass since "take" is more specific)
	takePatterns := []string{
		"take", "accept", // English
		"prendre", "accepter", // French
		"annehmen",           // German
		"\u53d7\u3051\u308b", // Japanese: 受ける
	}
	for _, p := range takePatterns {
		if strings.Contains(action, p) {
			return "take"
		}
	}

	// Double/Pass patterns
	passPatterns := []string{
		"pass", "reject", "decline", // English
		"refuser",            // French
		"ablehnen",           // German
		"\u964d\u308a\u308b", // Japanese: 降りる
	}
	for _, p := range passPatterns {
		if strings.Contains(action, p) {
			return "pass"
		}
	}

	// Fallback: if the action contains a separator ("/"), it's likely take or pass.
	// If no separator, it's likely no double.
	if !strings.Contains(action, "/") {
		return "nodbl"
	}

	return "unknown"
}

func bgfGetInt(m map[string]interface{}, key string) int {
	if v, ok := m[key]; ok {
		return bgfToInt(v)
	}
	return 0
}

func bgfGetFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok {
		return bgfToFloat(v)
	}
	return 0.0
}

func bgfGetBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func bgfGetMap(m map[string]interface{}, key string) map[string]interface{} {
	if v, ok := m[key]; ok {
		if sub, ok := v.(map[string]interface{}); ok {
			return sub
		}
	}
	return nil
}

func bgfGetIntArray(m map[string]interface{}, key string) [4]int {
	var result [4]int
	for i := range result {
		result[i] = -1
	}
	if v, ok := m[key]; ok {
		if arr, ok := v.([]interface{}); ok {
			for i := 0; i < 4 && i < len(arr); i++ {
				result[i] = bgfToInt(arr[i])
			}
		}
	}
	return result
}

func bgfToInt(v interface{}) int {
	switch val := v.(type) {
	case float64:
		return int(val)
	case int:
		return val
	case int64:
		return int(val)
	case string:
		n, _ := strconv.Atoi(val)
		return n
	}
	return 0
}

func bgfToFloat(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	}
	return 0.0
}
