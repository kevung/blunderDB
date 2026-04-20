package main

import (
	"database/sql"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/kevung/xgparser/xgparser"
)

// Match import and management functions

// Import XG match file using xgparser library
// This function uses raw segment parsing to capture complete cube action information
func (d *Database) ImportXGMatch(filePath string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Parse the XG file using raw segments for complete data
	imp := xgparser.NewImport(filePath)
	segments, err := imp.GetFileSegments()
	if err != nil {
		return 0, fmt.Errorf("failed to get file segments: %w", err)
	}

	// Also parse lightweight structure for metadata
	match, err := xgparser.ParseXG(segments)
	if err != nil {
		fmt.Printf("Error parsing XG file: %v\n", err)
		return 0, fmt.Errorf("failed to parse XG file: %w", err)
	}

	// Parse raw records for complete cube information
	rawCubeInfo := make(map[string]*RawCubeAction) // key: "game_cubeIdx"
	for _, seg := range segments {
		if seg.Type == xgparser.SegmentXGGameFile {
			records, _ := xgparser.ParseGameFile(seg.Data, -1)
			gameNum := int32(0)
			cubeIdx := 0
			for _, rec := range records {
				switch r := rec.(type) {
				case *xgparser.HeaderGameEntry:
					gameNum = r.GameNumber
					cubeIdx = 0
				case *xgparser.CubeEntry:
					if r.Double != -2 { // Skip initial positions
						key := fmt.Sprintf("%d_%d", gameNum, cubeIdx)
						rawCubeInfo[key] = &RawCubeAction{
							Double:   r.Double,
							Take:     r.Take,
							ActiveP:  r.ActiveP,
							CubeB:    r.CubeB,
							Position: r.Position,
							Doubled:  r.Doubled,
						}
						cubeIdx++
					}
				}
			}
		}
	}

	// Parse match date
	var matchDate time.Time
	if match.Metadata.DateTime != "" {
		// Try to parse various date formats
		for _, layout := range []string{
			"2006-01-02 15:04:05",
			"2006-01-02",
			time.RFC3339,
		} {
			if t, err := time.Parse(layout, match.Metadata.DateTime); err == nil {
				matchDate = t
				break
			}
		}
	}
	if matchDate.IsZero() {
		matchDate = time.Now()
	}

	// Compute match hash for duplicate detection (includes full match transcription)
	matchHash := ComputeMatchHash(match)

	// Compute canonical hash (format-independent) for cross-format duplicate detection
	canonicalHash := ComputeCanonicalMatchHashFromXG(match)

	// Check if this exact match already exists (same format)
	existingMatchID, err := d.checkMatchExistsLocked(matchHash)
	if err != nil {
		return 0, fmt.Errorf("failed to check for duplicate match: %w", err)
	}
	if existingMatchID > 0 {
		return 0, ErrDuplicateMatch
	}

	// Check if same match was imported from a different format (canonical duplicate)
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

	// Insert match metadata (including match_hash and canonical_hash)
	// If canonical duplicate, don't create new match - reuse existing match ID
	var matchID int64
	if isCanonicalDuplicate {
		matchID = canonicalMatchID
		fmt.Printf("Canonical duplicate detected - reusing match ID %d, importing new analysis only\n", matchID)
	} else {
		result, err := tx.Exec(`
			INSERT INTO match (player1_name, player2_name, event, location, round, 
			                   match_length, match_date, file_path, game_count, match_hash, canonical_hash)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, match.Metadata.Player1Name, match.Metadata.Player2Name,
			match.Metadata.Event, match.Metadata.Location, match.Metadata.Round,
			match.Metadata.MatchLength, matchDate, filePath, len(match.Games), matchHash, canonicalHash)

		if err != nil {
			return 0, fmt.Errorf("failed to insert match: %w", err)
		}

		matchID, err = result.LastInsertId()
		if err != nil {
			return 0, fmt.Errorf("failed to get match ID: %w", err)
		}

		// Auto-link tournament from event metadata
		eventName := strings.TrimSpace(match.Metadata.Event)
		if eventName != "" {
			var tournamentID int64
			err2 := tx.QueryRow(`SELECT id FROM tournament WHERE name = ?`, eventName).Scan(&tournamentID)
			if err2 != nil {
				// Tournament doesn't exist yet — create it
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
					fmt.Printf("Warning: failed to link match to tournament: %v\n", err)
				}
			}
		}
	}

	// Build a per-import deduplication cache keyed by Zobrist hash.
	// The preload of all existing positions is no longer needed — the UNIQUE index
	// on position.zobrist_hash handles cross-import dedup via INSERT OR IGNORE.
	cache := newImportCache()

	if isCanonicalDuplicate {
		// Canonical duplicate: import analysis to existing positions, create genuinely new ones
		for _, game := range match.Games {
			var pendingCubeComment string // Comment from a skipped "No Double" cube decision
			for i, move := range game.Moves {
				// Detect "No Double" cube decisions that will be skipped
				if move.MoveType == "cube" && move.Comment != "" {
					isSkipped := move.CubeMove == nil || move.CubeMove.Analysis == nil
					if isSkipped {
						pendingCubeComment = move.Comment
					}
				}

				if move.MoveType == "checker" && move.CheckerMove != nil {
					// Carry forward comment from a skipped "No Double" cube decision
					if pendingCubeComment != "" {
						if move.Comment == "" {
							game.Moves[i].Comment = pendingCubeComment
							move.Comment = pendingCubeComment
						} else {
							combined := pendingCubeComment + "\n" + move.Comment
							game.Moves[i].Comment = combined
							move.Comment = combined
						}
						pendingCubeComment = ""
					}

					pos, err := d.createPositionFromXG(move.CheckerMove.Position, &game, int32(match.Metadata.MatchLength), 0, move.CheckerMove.ActivePlayer)
					if err != nil {
						continue
					}
					pos.PlayerOnRoll = convertXGPlayerToBlunderDB(move.CheckerMove.ActivePlayer)
					pos.DecisionType = CheckerAction
					pos.Dice = [2]int{int(move.CheckerMove.Dice[0]), int(move.CheckerMove.Dice[1])}

					posID, err := d.findOrCreatePositionForCanonicalDuplicate(tx, pos, cache)
					if err != nil {
						continue
					}

					// Save checker analysis to position
					if len(move.CheckerMove.Analysis) > 0 {
						err = d.saveCheckerAnalysisToPositionInTx(tx, posID, move.CheckerMove.Analysis,
							&move.CheckerMove.Position, move.CheckerMove.ActivePlayer, &move.CheckerMove.PlayedMove)
						if err != nil {
							fmt.Printf("Warning: failed to save analysis for canonical duplicate: %v\n", err)
						}
					}

					// Save comment from XG file if present
					if move.Comment != "" {
						_, err = tx.Exec(`INSERT INTO comment (position_id, text) VALUES (?, ?)`, posID, move.Comment)
						if err != nil {
							fmt.Printf("Warning: failed to save XG comment for canonical duplicate: %v\n", err)
						}
					}
				} else if move.MoveType == "cube" && move.CubeMove != nil && move.CubeMove.Analysis != nil {
					pos, err := d.createPositionFromXG(move.CubeMove.Position, &game, int32(match.Metadata.MatchLength), 0, move.CubeMove.ActivePlayer)
					if err != nil {
						continue
					}
					pos.PlayerOnRoll = convertXGPlayerToBlunderDB(move.CubeMove.ActivePlayer)
					pos.DecisionType = CubeAction
					pos.Dice = [2]int{0, 0}

					posID, err := d.findOrCreatePositionForCanonicalDuplicate(tx, pos, cache)
					if err != nil {
						continue
					}

					err = d.saveCubeAnalysisToPositionInTx(tx, posID, move.CubeMove.Analysis, d.convertCubeAction(move.CubeMove.CubeAction))
					if err != nil {
						fmt.Printf("Warning: failed to save cube analysis for canonical duplicate: %v\n", err)
					}

					// Save comment from XG file if present
					if move.Comment != "" {
						_, err = tx.Exec(`INSERT INTO comment (position_id, text) VALUES (?, ?)`, posID, move.Comment)
						if err != nil {
							fmt.Printf("Warning: failed to save XG comment for canonical duplicate: %v\n", err)
						}
					}
				}
			}
		}
	} else {
		// Normal import: create game/move records and import everything
		if err := d.importXGGamesAndMoves(tx, matchID, match, rawCubeInfo, cache); err != nil {
			return 0, err
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	fmt.Printf("Successfully imported match %d with %d games from %s\n", matchID, len(match.Games), filePath)
	return matchID, nil
}

// RawCubeAction stores raw cube action data from XG file
type RawCubeAction struct {
	Double   int32
	Take     int32
	ActiveP  int32
	CubeB    int32
	Position [26]int8
	Doubled  *xgparser.EngineStructDoubleAction // Full cube analysis data
}

// importXGGamesAndMoves imports all games, moves, and analysis from an XG match
func (d *Database) importXGGamesAndMoves(tx *sql.Tx, matchID int64, match *xgparser.Match, rawCubeInfo map[string]*RawCubeAction, cache *importCache) error {
	for gameIdx, game := range match.Games {
		gameID, err := d.importGame(tx, matchID, &game)
		if err != nil {
			return fmt.Errorf("failed to import game %d: %w", game.GameNumber, err)
		}

		cubeIdx := 0
		var lastCubeAnalysis *RawCubeAction
		var pendingCubeComment string // Comment from a skipped "No Double" cube decision

		for moveIdx, move := range game.Moves {
			// Cancellation check at the top of every move iteration.
			if atomic.LoadInt32(&d.importCancelled) != 0 {
				return fmt.Errorf("import cancelled")
			}

			var rawCube *RawCubeAction

			if move.MoveType == "cube" {
				key := fmt.Sprintf("%d_%d", gameIdx+1, cubeIdx)
				if rc, ok := rawCubeInfo[key]; ok {
					rawCube = rc
					lastCubeAnalysis = rc
				}
				cubeIdx++

				// Check if this is a "No Double" that will be skipped by importMoveWithCacheAndRawCube.
				// If it has a comment, carry it forward to the next checker move.
				isNoDouble := false
				if rawCube != nil {
					isNoDouble = (rawCube.Double != 1)
				} else if move.CubeMove != nil {
					isNoDouble = (move.CubeMove.CubeAction == 0)
				}
				if isNoDouble && move.Comment != "" {
					pendingCubeComment = move.Comment
				}
			} else if move.MoveType == "checker" {
				rawCube = lastCubeAnalysis
				lastCubeAnalysis = nil

				// Carry forward comment from a skipped "No Double" cube decision
				if pendingCubeComment != "" {
					if move.Comment == "" {
						move.Comment = pendingCubeComment
					} else {
						move.Comment = pendingCubeComment + "\n" + move.Comment
					}
					pendingCubeComment = ""
				}
			}

			err := d.importMoveWithCacheAndRawCube(tx, gameID, int32(moveIdx), &move, &game, int32(match.Metadata.MatchLength), cache, rawCube)
			if err != nil {
				return fmt.Errorf("failed to import move %d in game %d: %w", moveIdx, game.GameNumber, err)
			}
		}
	}
	return nil
}

// importGame inserts a game and returns its ID
func (d *Database) importGame(tx *sql.Tx, matchID int64, game *xgparser.Game) (int64, error) {
	result, err := tx.Exec(`
		INSERT INTO game (match_id, game_number, initial_score_1, initial_score_2,
		                  winner, points_won, move_count)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, matchID, game.GameNumber, game.InitialScore[0], game.InitialScore[1],
		game.Winner, game.PointsWon, len(game.Moves))

	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// importMoveWithCacheAndRawCube imports a move with raw cube data for complete action info
func (d *Database) importMoveWithCacheAndRawCube(tx *sql.Tx, gameID int64, moveNumber int32, move *xgparser.Move, game *xgparser.Game, matchLength int32, cache *importCache, rawCube *RawCubeAction) error {
	var positionID int64
	var player int32
	var dice [2]int32
	var checkerMoveStr string
	var cubeActionStr string

	if move.MoveType == "checker" && move.CheckerMove != nil {
		// Create position from checker move
		pos, err := d.createPositionFromXG(move.CheckerMove.Position, game, matchLength, moveNumber, move.CheckerMove.ActivePlayer)
		if err != nil {
			return fmt.Errorf("failed to create position: %w", err)
		}

		// Set position-specific attributes from move
		// Convert XG player encoding (-1, 1) to blunderDB encoding (0, 1)
		pos.PlayerOnRoll = convertXGPlayerToBlunderDB(move.CheckerMove.ActivePlayer)
		pos.DecisionType = CheckerAction
		pos.Dice = [2]int{int(move.CheckerMove.Dice[0]), int(move.CheckerMove.Dice[1])}

		// Save position with cache
		posID, err := d.savePositionInTxWithCache(tx, pos, cache)
		if err != nil {
			return fmt.Errorf("failed to save position: %w", err)
		}
		positionID = posID

		player = move.CheckerMove.ActivePlayer
		dice = move.CheckerMove.Dice

		// Convert move notation with hit detection for consistency with analysis display
		checkerMoveStr = d.convertXGMoveToStringWithHits(move.CheckerMove.PlayedMove, &move.CheckerMove.Position, move.CheckerMove.ActivePlayer)

		// Save move
		moveResult, err := tx.Exec(`
			INSERT INTO move (game_id, move_number, move_type, position_id, player,
			                  dice_1, dice_2, checker_move)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, gameID, moveNumber, "checker", positionID, player, dice[0], dice[1], checkerMoveStr)
		if err != nil {
			return err
		}

		moveID, err := moveResult.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get last insert ID: %w", err)
		}

		// Save analysis if available
		if len(move.CheckerMove.Analysis) > 0 {
			for _, analysis := range move.CheckerMove.Analysis {
				err = d.saveMoveAnalysisInTx(tx, moveID, &analysis)
				if err != nil {
					fmt.Printf("Warning: failed to save checker analysis: %v\n", err)
				}
			}

			err = d.saveCheckerAnalysisToPositionInTx(tx, positionID, move.CheckerMove.Analysis, &move.CheckerMove.Position, move.CheckerMove.ActivePlayer, &move.CheckerMove.PlayedMove)
			if err != nil {
				fmt.Printf("Warning: failed to save position analysis: %v\n", err)
			}
		}

		// Save cube analysis for this checker position (from the preceding CubeEntry)
		// This allows displaying cube info when pressing 'd' on a checker decision
		if rawCube != nil && rawCube.Doubled != nil {
			err = d.saveCubeAnalysisForCheckerPositionInTx(tx, positionID, rawCube)
			if err != nil {
				fmt.Printf("Warning: failed to save cube analysis for checker position: %v\n", err)
			}
		}

	} else if move.MoveType == "cube" && move.CubeMove != nil {
		// Handle explicit cube decisions
		// Only create positions for EXPLICIT cube actions (when a player actually doubles)
		// Skip implicit "No Double" decisions

		// Check if this is an explicit cube action
		isExplicitCubeAction := false
		if rawCube != nil {
			// Explicit action: Double was offered (Double == 1)
			isExplicitCubeAction = (rawCube.Double == 1)
		} else {
			// Fallback: check CubeAction field
			isExplicitCubeAction = (move.CubeMove.CubeAction != 0) // 0 = No Double
		}

		if !isExplicitCubeAction {
			// Skip implicit "No Double" decisions - don't create position
			return nil
		}

		if rawCube != nil && rawCube.Double == 1 && rawCube.Take == 1 {
			// DOUBLE/TAKE scenario: Create two positions

			// Position 1: Doubling decision (before the double)
			// The player on roll decides whether to double
			pos1, err := d.createPositionFromXG(move.CubeMove.Position, game, matchLength, moveNumber, move.CubeMove.ActivePlayer)
			if err != nil {
				return fmt.Errorf("failed to create doubling position: %w", err)
			}
			pos1.PlayerOnRoll = convertXGPlayerToBlunderDB(move.CubeMove.ActivePlayer)
			pos1.DecisionType = CubeAction
			pos1.Dice = [2]int{0, 0}

			// Save position 1 (doubling decision)
			posID1, err := d.savePositionInTxWithCache(tx, pos1, cache)
			if err != nil {
				return fmt.Errorf("failed to save doubling position: %w", err)
			}

			player = move.CubeMove.ActivePlayer

			// Save move 1: Double
			moveResult1, err := tx.Exec(`
				INSERT INTO move (game_id, move_number, move_type, position_id, player,
				                  dice_1, dice_2, cube_action)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			`, gameID, moveNumber, "cube", posID1, player, 0, 0, "Double")
			if err != nil {
				return err
			}
			moveID1, err := moveResult1.LastInsertId()
			if err != nil {
				return fmt.Errorf("failed to get last insert ID: %w", err)
			}

			// Save analysis to first position
			if move.CubeMove.Analysis != nil {
				err = d.saveCubeAnalysisInTx(tx, moveID1, move.CubeMove.Analysis)
				if err != nil {
					fmt.Printf("Warning: failed to save cube analysis: %v\n", err)
				}
				err = d.saveCubeAnalysisToPositionInTx(tx, posID1, move.CubeMove.Analysis, "Double/Take")
				if err != nil {
					fmt.Printf("Warning: failed to save position cube analysis: %v\n", err)
				}
			}

			// Position 2: Take/Pass decision (after the double)
			// The opponent decides whether to take or pass
			// Note: We show the position BEFORE the take decision is executed,
			// so the cube is still in the center at doubled value.
			// The cube ownership will be reflected in the NEXT checker move position.
			pos2 := *pos1                           // Clone the position
			pos2.Cube.Value++                       // Double the cube (increment exponent: 0→1, 1→2, etc.)
			opponentPlayer := 1 - pos1.PlayerOnRoll // Opponent player (blunderDB encoding)
			pos2.PlayerOnRoll = opponentPlayer      // Opponent decides whether to take
			pos2.Cube.Owner = -1                    // Cube still in center (decision not yet executed)

			// Save position 2 (take decision)
			posID2, err := d.savePositionInTxWithCache(tx, &pos2, cache)
			if err != nil {
				return fmt.Errorf("failed to save take position: %w", err)
			}
			positionID = posID2 // Use second position as the reference

			// Convert opponent from blunderDB (0,1) back to XG encoding (1,-1) for move table
			var opponentPlayerXG int32
			if opponentPlayer == 0 {
				opponentPlayerXG = 1
			} else {
				opponentPlayerXG = -1
			}

			// Save move 2: Take
			_, err = tx.Exec(`
				INSERT INTO move (game_id, move_number, move_type, position_id, player,
				                  dice_1, dice_2, cube_action)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			`, gameID, moveNumber, "cube", posID2, opponentPlayerXG, 0, 0, "Take")
			if err != nil {
				return err
			}

		} else {
			// Single cube action (Double without Take, or other actions)
			pos, err := d.createPositionFromXG(move.CubeMove.Position, game, matchLength, moveNumber, move.CubeMove.ActivePlayer)
			if err != nil {
				return fmt.Errorf("failed to create position: %w", err)
			}

			pos.PlayerOnRoll = convertXGPlayerToBlunderDB(move.CubeMove.ActivePlayer)
			pos.DecisionType = CubeAction
			pos.Dice = [2]int{0, 0}

			// Save position
			posID, err := d.savePositionInTxWithCache(tx, pos, cache)
			if err != nil {
				return fmt.Errorf("failed to save position: %w", err)
			}
			positionID = posID
			player = move.CubeMove.ActivePlayer

			// Determine cube action string
			if rawCube != nil {
				cubeActionStr = d.convertRawCubeAction(rawCube.Double, rawCube.Take)
			} else {
				cubeActionStr = d.convertCubeAction(move.CubeMove.CubeAction)
			}

			// Save move
			moveResult, err := tx.Exec(`
				INSERT INTO move (game_id, move_number, move_type, position_id, player,
				                  dice_1, dice_2, cube_action)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			`, gameID, moveNumber, "cube", positionID, player, 0, 0, cubeActionStr)
			if err != nil {
				return err
			}

			moveID, err := moveResult.LastInsertId()
			if err != nil {
				return fmt.Errorf("failed to get last insert ID: %w", err)
			}

			// Save cube analysis
			if move.CubeMove.Analysis != nil {
				err = d.saveCubeAnalysisInTx(tx, moveID, move.CubeMove.Analysis)
				if err != nil {
					fmt.Printf("Warning: failed to save cube analysis: %v\n", err)
				}

				err = d.saveCubeAnalysisToPositionInTx(tx, positionID, move.CubeMove.Analysis, cubeActionStr)
				if err != nil {
					fmt.Printf("Warning: failed to save position cube analysis: %v\n", err)
				}
			}
		}
	}

	// Save comment from XG file if present
	if move.Comment != "" && positionID > 0 {
		_, err := tx.Exec(`INSERT INTO comment (position_id, text) VALUES (?, ?)`, positionID, move.Comment)
		if err != nil {
			fmt.Printf("Warning: failed to save XG comment: %v\n", err)
		}
	}

	return nil
}

// convertRawCubeAction converts raw Double/Take values to action string
func (d *Database) convertRawCubeAction(double, take int32) string {
	// XG cube action encoding:
	// Double=0, Take=-1: No Double
	// Double=1, Take=1: Double, Take
	// Double=1, Take=-1: Double/Pass (opponent passed the double)
	// Double=-2: Initial position (should be filtered before)

	if double == 0 {
		return "No Double"
	} else if double == 1 {
		if take == 1 {
			return "Double/Take"
		} else {
			return "Double/Pass"
		}
	}
	return fmt.Sprintf("Unknown(D=%d,T=%d)", double, take)
}

// createPositionFromXG converts xgparser.Position to blunderDB Position
// activePlayer indicates which XG player (-1 or 1) is on roll in this position
func (d *Database) createPositionFromXG(xgPos xgparser.Position, game *xgparser.Game, matchLength int32, moveNum int32, activePlayer int32) (*Position, error) {
	// Convert XG player encoding to blunderDB encoding
	// XG: -1 = Player 1, 1 = Player 2
	// blunderDB: 0 = Player 1, 1 = Player 2
	activePlayerBlunderDB := convertXGPlayerToBlunderDB(activePlayer)
	opponentPlayerBlunderDB := 1 - activePlayerBlunderDB

	// Calculate away scores from current scores
	// In blunderDB, scores are "points away from winning"
	// game.InitialScore contains current scores (e.g., 2-3 in a 7-point match)
	// We need to convert to away scores (e.g., 5-away, 4-away)
	awayScore1 := int(matchLength) - int(game.InitialScore[0])
	awayScore2 := int(matchLength) - int(game.InitialScore[1])

	// Handle unlimited match (matchLength == 0)
	if matchLength == 0 {
		awayScore1 = -1
		awayScore2 = -1
	}

	// Map XG cube position to blunderDB format
	// XG CubePos is RELATIVE to active player:
	//   CubePos = 0: Center (no owner)
	//   CubePos = 1: Active player owns the cube
	//   CubePos = -1: Opponent owns the cube
	// blunderDB uses absolute encoding:
	//   -1 = center (no owner)
	//   0 = Player 1 owns (bottom, black)
	//   1 = Player 2 owns (top, white)
	cubeOwner := -1 // Default: center (no owner)
	if xgPos.CubePos == 1 {
		// Active player owns the cube
		cubeOwner = activePlayerBlunderDB
	} else if xgPos.CubePos == -1 {
		// Opponent owns the cube
		cubeOwner = opponentPlayerBlunderDB
	}
	// CubePos == 0 means center, cubeOwner stays -1

	// Convert cube value from XG (direct value 1,2,4,8...) to blunderDB (exponent 0,1,2,3...)
	// blunderDB displays 2^value, so we need log2(xgCube)
	cubeValue := 0
	if xgPos.Cube > 0 {
		// Calculate log2 of cube value
		for v := int(xgPos.Cube); v > 1; v >>= 1 {
			cubeValue++
		}
	}

	pos := &Position{
		PlayerOnRoll: 0, // Will be set from move context
		DecisionType: 0, // Checker decision by default
		Score:        [2]int{awayScore1, awayScore2},
		Cube: Cube{
			Value: cubeValue,
			Owner: cubeOwner,
		},
		Dice: [2]int{0, 0}, // Will be set from move data
	}

	// Convert checkers from XG format to blunderDB format
	// XG format: index 0-23 are points 1-24, index 24=bar, 25=opponent bar
	// Positive values = active player's checkers, negative = opponent's checkers
	//
	// In blunderDB:
	// - Color 0 = Player 1 (always at bottom, black, moves 24→1) - NEVER CHANGES
	// - Color 1 = Player 2 (always at top, white, moves 1→24) - NEVER CHANGES
	// - Points 1-24 with indices 1-24 in the array
	// - Index 0 = Player 2's bar (white), Index 25 = Player 1's bar (black)
	//
	// XG stores positions from the active player's perspective:
	// - Positive checkers = active player's checkers
	// - Negative checkers = opponent's checkers
	// - Point numbering is from active player's perspective
	//
	// Player mapping:
	// - XG Player 0 = blunderDB Player 1 (Color 0, bottom, black)
	// - XG Player 1 = blunderDB Player 2 (Color 1, top, white)
	//
	// Strategy:
	// 1. Determine which player owns the checkers based on sign AND activePlayer
	// 2. Assign colors based on OWNER, not on sign
	// 3. Mirror positions if activePlayer == 1 (since XG encodes from active player's view)
	for i := 0; i < 26; i++ {
		checkerCount := xgPos.Checkers[i]

		// Determine WHERE to place them (calculate targetIndex for ALL points, even empty)
		// XG uses 1-based indexing: index 1-24 = points 1-24, index 0 = opponent bar, index 25 = active bar
		// blunderDB also uses same: index 1-24 = points 1-24, index 0 = P2 bar, index 25 = P1 bar
		targetIndex := i
		if activePlayerBlunderDB == 1 {
			// Player 2's perspective, need to mirror to Player 1's perspective
			if i >= 1 && i <= 24 {
				// XG index i = Player 2's point i → Player 1's point (25 - i) → same index (25 - i)
				targetIndex = 25 - i
			} else if i == 0 {
				// Opponent's bar from Player 2's view = Player 1's bar
				targetIndex = 25
			} else if i == 25 {
				// Active player's bar (Player 2) → Player 2's bar
				targetIndex = 0
			}
		}
		// Player 1's perspective: direct mapping (XG index = blunderDB index)
		// XG index 1-24 = Player 1's points 1-24 → blunderDB index 1-24 (same)
		// XG index 0 = opponent bar (Player 2) → blunderDB index 0 (Player 2's bar)
		// XG index 25 = active bar (Player 1) → blunderDB index 25 (Player 1's bar)

		if checkerCount == 0 {
			pos.Board.Points[targetIndex] = Point{Checkers: 0, Color: -1}
			continue
		}

		// Determine WHO owns these checkers
		var ownerColor int // 0=Player1(black), 1=Player2(white)
		if checkerCount > 0 {
			// Positive = active player
			ownerColor = activePlayerBlunderDB
		} else {
			// Negative = opponent
			ownerColor = opponentPlayerBlunderDB
		}

		// Place checkers with FIXED color based on owner
		pos.Board.Points[targetIndex] = Point{
			Checkers: int(abs(checkerCount)),
			Color:    ownerColor,
		}
	}

	// Calculate bearoff (checkers borne off)
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

// Helper function for absolute value
func abs(x int8) int {
	if x < 0 {
		return int(-x)
	}
	return int(x)
}

// convertXGPlayerToBlunderDB converts XG player encoding to blunderDB encoding
// XG: -1 = Player 1, 1 = Player 2
// blunderDB: 0 = Player 1, 1 = Player 2
func convertXGPlayerToBlunderDB(xgPlayer int32) int {
	// XG uses 1 for first player, -1 for second player
	// blunderDB uses 0 for Player 1 (black, bottom), 1 for Player 2 (white, top)
	if xgPlayer == 1 {
		return 0 // Player 1
	}
	return 1 // Player 2
}

// convertBlunderDBPlayerToXG converts blunderDB player encoding (0/1) to XG encoding (1/-1)
// This is used when storing GnuBG-imported moves in the DB, so that GetMatchMovePositions
// can apply convertXGPlayerToBlunderDB uniformly for both XG and GnuBG imports.
func convertBlunderDBPlayerToXG(blunderDBPlayer int) int32 {
	if blunderDBPlayer == 0 {
		return 1 // Player 1
	}
	return -1 // Player 2
}

// saveMoveAnalysisInTx saves checker move analysis within a transaction
func (d *Database) saveMoveAnalysisInTx(tx *sql.Tx, moveID int64, analysis *xgparser.CheckerAnalysis) error {
	// Calculate win rates (player1 is player on roll) — stored as integer × 100
	player1WinRate := int64(math.Round(float64(analysis.Player1WinRate) * 100.0 * 100))
	player2WinRate := int64(math.Round(float64(1.0-analysis.Player1WinRate) * 100.0 * 100))

	_, err := tx.Exec(`
		INSERT INTO move_analysis (move_id, analysis_type, depth, equity, equity_error,
		                           win_rate, gammon_rate, backgammon_rate,
		                           opponent_win_rate, opponent_gammon_rate, opponent_backgammon_rate)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, moveID, "checker", translateAnalysisDepth(int(analysis.AnalysisDepth)),
		int64(math.Round(float64(analysis.Equity)*1000)), int64(0),
		player1WinRate, int64(math.Round(float64(analysis.Player1GammonRate)*100.0*100)), int64(math.Round(float64(analysis.Player1BgRate)*100.0*100)),
		player2WinRate, int64(math.Round(float64(analysis.Player2GammonRate)*100.0*100)), int64(math.Round(float64(analysis.Player2BgRate)*100.0*100)))

	return err
}

// saveCubeAnalysisInTx saves cube analysis within a transaction
func (d *Database) saveCubeAnalysisInTx(tx *sql.Tx, moveID int64, analysis *xgparser.CubeAnalysis) error {
	// Calculate win rates — stored as integer × 100
	player1WinRate := int64(math.Round(float64(analysis.Player1WinRate) * 100.0 * 100))
	player2WinRate := int64(math.Round(float64(1.0-analysis.Player1WinRate) * 100.0 * 100))

	_, err := tx.Exec(`
		INSERT INTO move_analysis (move_id, analysis_type, depth, equity, equity_error,
		                           win_rate, gammon_rate, backgammon_rate,
		                           opponent_win_rate, opponent_gammon_rate, opponent_backgammon_rate)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, moveID, "cube", translateAnalysisDepth(int(analysis.AnalysisDepth)),
		int64(math.Round(float64(analysis.CubefulNoDouble)*1000)), int64(0),
		player1WinRate, int64(math.Round(float64(analysis.Player1GammonRate)*100.0*100)), int64(math.Round(float64(analysis.Player1BgRate)*100.0*100)),
		player2WinRate, int64(math.Round(float64(analysis.Player2GammonRate)*100.0*100)), int64(math.Round(float64(analysis.Player2BgRate)*100.0*100)))

	return err
}

// saveCheckerAnalysisToPositionInTx converts XG checker analysis to PositionAnalysis and saves it
// playedMove is optional - if provided, it will be used as the source of truth for the first analysis
// (workaround for xgparser bug where analysis.Move may be incomplete for multi-submove bear-offs)
func (d *Database) saveCheckerAnalysisToPositionInTx(tx *sql.Tx, positionID int64, analyses []xgparser.CheckerAnalysis, initialPosition *xgparser.Position, activePlayer int32, playedMove *[8]int32) error {
	if len(analyses) == 0 {
		return nil
	}

	// Create PositionAnalysis structure
	posAnalysis := PositionAnalysis{
		PositionID:            int(positionID),
		AnalysisType:          "CheckerMove",
		AnalysisEngineVersion: "XG",
		CreationDate:          time.Now(),
		LastModifiedDate:      time.Now(),
	}

	// Build checker analysis with all moves
	checkerMoves := make([]CheckerMove, 0, len(analyses))
	for i, analysis := range analyses {
		// Convert move from [8]int8 to [8]int32 for convertXGMoveToString
		var move [8]int32

		// For the first analysis (i=0), use playedMove if available
		// This is a workaround for xgparser bug where analysis.Move may be incomplete
		// for multi-submove bear-offs or other complex moves
		if i == 0 && playedMove != nil {
			// Check if playedMove has more info than analysis.Move
			playedMoveCount := 0
			analysisMoveCount := 0
			for j := 0; j < 8; j += 2 {
				if (*playedMove)[j] != -1 {
					playedMoveCount++
				}
				if analysis.Move[j] != -1 {
					analysisMoveCount++
				}
			}
			// Use playedMove if it has more sub-moves
			if playedMoveCount > analysisMoveCount {
				for j := 0; j < 8; j++ {
					move[j] = (*playedMove)[j]
				}
			} else {
				for j := 0; j < 8; j++ {
					move[j] = int32(analysis.Move[j])
				}
			}
		} else {
			for j := 0; j < 8; j++ {
				move[j] = int32(analysis.Move[j])
			}
		}
		// Infer multipliers from position changes
		// XG stores moves compactly - e.g., 1/off(4) is stored as just 1/off once
		if initialPosition != nil {
			move = inferMoveMultipliers(move, initialPosition, &analysis.Position, activePlayer)
		}

		// Use move string with hit detection if initial position is available
		var moveStr string
		if initialPosition != nil {
			moveStr = d.convertXGMoveToStringWithHits(move, initialPosition, activePlayer)
		} else {
			moveStr = d.convertXGMoveToString(move, activePlayer)
		}

		var equityError *float64
		if i > 0 {
			diff := float64(analyses[0].Equity - analysis.Equity)
			equityError = &diff
		}

		checkerMove := CheckerMove{
			Index:                    i,
			AnalysisDepth:            translateAnalysisDepth(int(analysis.AnalysisDepth)),
			AnalysisEngine:           "XG",
			Move:                     moveStr,
			Equity:                   float64(analysis.Equity),
			EquityError:              equityError,
			PlayerWinChance:          float64(analysis.Player1WinRate) * 100.0,
			PlayerGammonChance:       float64(analysis.Player1GammonRate) * 100.0,
			PlayerBackgammonChance:   float64(analysis.Player1BgRate) * 100.0,
			OpponentWinChance:        float64(1.0-analysis.Player1WinRate) * 100.0,
			OpponentGammonChance:     float64(analysis.Player2GammonRate) * 100.0,
			OpponentBackgammonChance: float64(analysis.Player2BgRate) * 100.0,
		}
		checkerMoves = append(checkerMoves, checkerMove)
	}

	posAnalysis.CheckerAnalysis = &CheckerAnalysis{
		Moves: checkerMoves,
	}

	// Set played move on the analysis if available
	if playedMove != nil {
		playedMoveStr := d.convertXGMoveToStringWithHits(*playedMove, initialPosition, activePlayer)
		if playedMoveStr != "" {
			posAnalysis.PlayedMoves = []string{playedMoveStr}
		}
	}

	// Save to analysis table
	return d.saveAnalysisInTx(tx, positionID, posAnalysis)
}

// saveCubeAnalysisToPositionInTx converts XG cube analysis to PositionAnalysis and saves it.
// playedCubeAction is the cube action the player actually took (e.g. "No Double", "Double/Take",
// "Double/Pass", "Double"). When non-empty it is stored in PlayedCubeActions so that
// populateAnalysisColumns can compute cube_error.
func (d *Database) saveCubeAnalysisToPositionInTx(tx *sql.Tx, positionID int64, analysis *xgparser.CubeAnalysis, playedCubeAction string) error {
	if analysis == nil {
		return nil
	}

	// Create PositionAnalysis structure
	posAnalysis := PositionAnalysis{
		PositionID:            int(positionID),
		AnalysisType:          "DoublingCube",
		AnalysisEngineVersion: "XG",
		CreationDate:          time.Now(),
		LastModifiedDate:      time.Now(),
	}
	if playedCubeAction != "" {
		posAnalysis.PlayedCubeActions = []string{playedCubeAction}
	}

	cubefulNoDouble := float64(analysis.CubefulNoDouble)
	cubefulDoubleTake := float64(analysis.CubefulDoubleTake)
	cubefulDoublePass := float64(analysis.CubefulDoublePass)

	// Calculate best equity considering opponent's optimal response
	// If player doubles, opponent will choose the action that minimizes player's equity
	// So effective double equity = min(DoubleTake, DoublePass)
	effectiveDoubleEquity := cubefulDoubleTake
	if cubefulDoublePass < cubefulDoubleTake {
		effectiveDoubleEquity = cubefulDoublePass
	}

	// Player's best achievable equity = max(NoDouble, effectiveDoubleEquity)
	bestEquity := cubefulNoDouble
	bestAction := "No Double"
	if effectiveDoubleEquity > cubefulNoDouble {
		bestEquity = effectiveDoubleEquity
		// Best action is "Double, Take" or "Double, Pass" depending on opponent's response
		if cubefulDoubleTake <= cubefulDoublePass {
			bestAction = "Double, Take"
		} else {
			bestAction = "Double, Pass"
		}
	}

	// Build doubling cube analysis
	// Error is negative when this decision loses equity vs best (current - best)
	cubeAnalysis := DoublingCubeAnalysis{
		AnalysisDepth:             translateAnalysisDepth(int(analysis.AnalysisDepth)),
		AnalysisEngine:            "XG",
		PlayerWinChances:          float64(analysis.Player1WinRate) * 100.0,
		PlayerGammonChances:       float64(analysis.Player1GammonRate) * 100.0,
		PlayerBackgammonChances:   float64(analysis.Player1BgRate) * 100.0,
		OpponentWinChances:        float64(1.0-analysis.Player1WinRate) * 100.0,
		OpponentGammonChances:     float64(analysis.Player2GammonRate) * 100.0,
		OpponentBackgammonChances: float64(analysis.Player2BgRate) * 100.0,
		CubelessNoDoubleEquity:    float64(analysis.CubelessNoDouble),
		CubelessDoubleEquity:      float64(analysis.CubelessDouble),
		CubefulNoDoubleEquity:     cubefulNoDouble,
		CubefulNoDoubleError:      cubefulNoDouble - bestEquity,
		CubefulDoubleTakeEquity:   cubefulDoubleTake,
		CubefulDoubleTakeError:    cubefulDoubleTake - bestEquity,
		CubefulDoublePassEquity:   cubefulDoublePass,
		CubefulDoublePassError:    cubefulDoublePass - bestEquity,
		BestCubeAction:            bestAction,
		WrongPassPercentage:       float64(analysis.WrongPassTakePercent) * 100.0,
		WrongTakePercentage:       0.0, // XG provides WrongPassTakePercent which covers both
	}

	posAnalysis.DoublingCubeAnalysis = &cubeAnalysis

	// Save to analysis table
	return d.saveAnalysisInTx(tx, positionID, posAnalysis)
}

// saveCubeAnalysisForCheckerPositionInTx saves cube analysis from a RawCubeAction to a checker position
// This is used to attach the cube decision analysis to checker moves (from the preceding CubeEntry)
// It merges the cube info with existing checker analysis if present
func (d *Database) saveCubeAnalysisForCheckerPositionInTx(tx *sql.Tx, positionID int64, rawCube *RawCubeAction) error {
	if rawCube == nil || rawCube.Doubled == nil {
		return nil
	}

	doubled := rawCube.Doubled

	// Try to load existing analysis for this position
	var existingAnalysisData []byte
	var existingID int64
	err := tx.QueryRow(`SELECT id, data FROM analysis WHERE position_id = ?`, positionID).Scan(&existingID, &existingAnalysisData)

	var posAnalysis PositionAnalysis
	if err == nil && existingID > 0 {
		// Existing analysis found - merge cube info with it
		posAnalysis, err = decodeAnalysisFromStorage(existingAnalysisData)
		if err != nil {
			return err
		}
	} else {
		// No existing analysis - create new one
		posAnalysis = PositionAnalysis{
			PositionID:            int(positionID),
			AnalysisType:          "CheckerMove",
			AnalysisEngineVersion: "XG",
			CreationDate:          time.Now(),
		}
	}

	posAnalysis.LastModifiedDate = time.Now()

	// Extract data from EngineStructDoubleAction
	// XG Eval array mapping (from xgparser convertCubeEntry):
	// Eval[0] = opponent's backgammon rate
	// Eval[1] = opponent's gammon rate
	// Eval[2] = opponent's win rate (so player on roll's win rate = 1 - Eval[2])
	// Eval[4] = player on roll's gammon rate
	// Eval[5] = player on roll's backgammon rate
	// Eval[6] = cubeless equity
	cubefulNoDouble := float64(doubled.EquB)
	cubefulDoubleTake := float64(doubled.EquDouble)
	cubefulDoublePass := float64(doubled.EquDrop)

	// Win/gammon/bg rates from player on roll's perspective
	playerWin := (1.0 - float64(doubled.Eval[2])) * 100.0
	playerGammon := float64(doubled.Eval[4]) * 100.0
	playerBg := float64(doubled.Eval[5]) * 100.0
	opponentWin := float64(doubled.Eval[2]) * 100.0
	opponentGammon := float64(doubled.Eval[1]) * 100.0
	opponentBg := float64(doubled.Eval[0]) * 100.0

	// Calculate best equity considering opponent's optimal response
	// If player doubles, opponent will choose the action that minimizes player's equity
	// So effective double equity = min(DoubleTake, DoublePass)
	effectiveDoubleEquity := cubefulDoubleTake
	if cubefulDoublePass < cubefulDoubleTake {
		effectiveDoubleEquity = cubefulDoublePass
	}

	// Player's best achievable equity = max(NoDouble, effectiveDoubleEquity)
	bestEquity := cubefulNoDouble
	bestAction := "No Double"
	if effectiveDoubleEquity > cubefulNoDouble {
		bestEquity = effectiveDoubleEquity
		// Best action is "Double, Take" or "Double, Pass" depending on opponent's response
		if cubefulDoubleTake <= cubefulDoublePass {
			bestAction = "Double, Take"
		} else {
			bestAction = "Double, Pass"
		}
	}

	// Build doubling cube analysis
	// Error is negative when this decision loses equity vs best (current - best)
	cubeAnalysis := DoublingCubeAnalysis{
		AnalysisDepth:             translateAnalysisDepth(int(doubled.Level)),
		AnalysisEngine:            "XG",
		PlayerWinChances:          playerWin,
		PlayerGammonChances:       playerGammon,
		PlayerBackgammonChances:   playerBg,
		OpponentWinChances:        opponentWin,
		OpponentGammonChances:     opponentGammon,
		OpponentBackgammonChances: opponentBg,
		CubelessNoDoubleEquity:    float64(doubled.Eval[6]),
		CubelessDoubleEquity:      float64(doubled.Eval[6]), // Same as no double for cubeless
		CubefulNoDoubleEquity:     cubefulNoDouble,
		CubefulNoDoubleError:      cubefulNoDouble - bestEquity,
		CubefulDoubleTakeEquity:   cubefulDoubleTake,
		CubefulDoubleTakeError:    cubefulDoubleTake - bestEquity,
		CubefulDoublePassEquity:   cubefulDoublePass,
		CubefulDoublePassError:    cubefulDoublePass - bestEquity,
		BestCubeAction:            bestAction,
		WrongPassPercentage:       0.0, // Not available from EngineStructDoubleAction
		WrongTakePercentage:       0.0,
	}

	posAnalysis.DoublingCubeAnalysis = &cubeAnalysis

	// Save to analysis table (will update if exists, insert if not)
	return d.saveAnalysisInTx(tx, positionID, posAnalysis)
}

// inferMoveMultipliers analyzes a partial move array and the position difference
// to infer the correct number of repetitions for each move.
// XG sometimes stores only one instance of a move even when multiple checkers make the same move.
// This function expands the move array to include all repetitions.
// Returns the expanded move array with correct multipliers.
func inferMoveMultipliers(partialMove [8]int32, initialPos, finalPos *xgparser.Position, activePlayer int32) [8]int32 {
	if initialPos == nil || finalPos == nil {
		return partialMove
	}

	// First, count how many moves are explicitly in the input
	// and count occurrences of each unique (from,to) pair
	type moveSpec struct {
		from int32
		to   int32
	}
	moveCount := make(map[moveSpec]int32)
	totalInputMoves := 0

	for i := 0; i < 8; i += 2 {
		from := partialMove[i]
		to := partialMove[i+1]
		if from == -1 {
			break
		}
		// Handle implicit bear-off
		if from >= 1 && from <= 6 && to <= 0 && to != -2 {
			to = -2
		}
		moveCount[moveSpec{from, to}]++
		totalInputMoves++
	}

	if totalInputMoves == 0 {
		return partialMove
	}

	// If we already have multiple moves in input (not a compact representation),
	// just return the input as-is - no inference needed
	// XG uses compact representation only for doublets where same move repeats
	if totalInputMoves > 1 {
		// Check if all moves are the same (compact doublet notation)
		allSame := true
		var firstMove moveSpec
		first := true
		for ms := range moveCount {
			if first {
				firstMove = ms
				first = false
			} else if ms != firstMove {
				allSame = false
				break
			}
		}
		// If moves are different, no inference needed - return as-is
		if !allSame {
			return partialMove
		}
	}

	// At this point, either:
	// 1. We have a single move that might need expansion (e.g., [8,3,-1,-1,-1,-1,-1,-1] for 8/3(4))
	// 2. We have multiple identical moves (already explicit, no expansion needed)

	// Get the unique moves preserving order
	var uniqueMoves []moveSpec
	seen := make(map[moveSpec]bool)
	for i := 0; i < 8; i += 2 {
		from := partialMove[i]
		to := partialMove[i+1]
		if from == -1 {
			break
		}
		if from >= 1 && from <= 6 && to <= 0 && to != -2 {
			to = -2
		}
		ms := moveSpec{from, to}
		if !seen[ms] {
			seen[ms] = true
			uniqueMoves = append(uniqueMoves, ms)
		}
	}

	// If we have multiple identical moves in input, they're already explicit
	if len(uniqueMoves) == 1 && moveCount[uniqueMoves[0]] > 1 {
		return partialMove
	}

	// Calculate how many checkers left each source point
	// netChange[point] = initialCheckers - finalCheckers (positive = net loss of checkers)
	netChange := make(map[int32]int32)

	for _, ms := range uniqueMoves {
		if ms.from == 25 {
			netChange[25] = int32(initialPos.Checkers[25]) - int32(finalPos.Checkers[25])
		} else if ms.from >= 1 && ms.from <= 24 {
			netChange[ms.from] = int32(initialPos.Checkers[ms.from]) - int32(finalPos.Checkers[ms.from])
		}
		// Track destination changes too
		if ms.to >= 1 && ms.to <= 24 {
			netChange[ms.to] = int32(initialPos.Checkers[ms.to]) - int32(finalPos.Checkers[ms.to])
		}
	}

	// Build a flow model: how many checkers move from each source
	// Track arriving checkers for intermediate points
	arriving := make(map[int32]int32) // checkers arriving at each point

	var expandedMove [8]int32
	for i := range expandedMove {
		expandedMove[i] = -1
	}
	moveIndex := 0

	// Process moves in order
	for _, ms := range uniqueMoves {
		var count int32 = 1

		if ms.to == -2 {
			// Bear-off: count = net checkers that left this point + checkers arriving from other points
			count = netChange[ms.from] + arriving[ms.from]
		} else if ms.to >= 1 && ms.to <= 24 {
			// Move to board point
			// Count = how many checkers left the source AND didn't come back
			// For most cases, this is simply the net change of the source
			// But we need to also account for checker flow

			// Net change at source tells us how many checkers total left
			srcLoss := netChange[ms.from]

			// If source also receives checkers (from another move), we need less moves
			srcReceive := arriving[ms.from]

			// The move count is the source loss minus any checkers that arrived
			count = srcLoss - srcReceive
			if count <= 0 {
				count = 1
			}

			// After this move, these checkers arrive at the destination
			arriving[ms.to] += count
		}

		// Cap at 4 (maximum for doubles) or remaining slots
		maxMoves := int32(4)
		remainingSlots := int32((8 - moveIndex) / 2)
		if count > maxMoves {
			count = maxMoves
		}
		if count > remainingSlots {
			count = remainingSlots
		}
		if count < 1 {
			count = 1
		}

		for j := int32(0); j < count && moveIndex < 8; j++ {
			expandedMove[moveIndex] = ms.from
			expandedMove[moveIndex+1] = ms.to
			moveIndex += 2
		}
	}

	return expandedMove
}

// translateAnalysisDepth converts XG analysis depth codes to human-readable strings
// XG depth codes:
//   - 0-9 = (N+1)-ply search depth (XG stores ply-1 internally)
//   - 998-1000 = Book moves (different footnote references)
//   - 1001 = XG Roller (neural network)
//   - 1002 = XG Roller++ (extended neural network analysis)
func translateAnalysisDepth(depth int) string {
	switch {
	case depth >= 0 && depth <= 9:
		return fmt.Sprintf("%d-ply", depth+1)
	case depth >= 998 && depth <= 1000:
		return "Book"
	case depth == 1001:
		return "XG Roller"
	case depth == 1002:
		return "XG Roller++"
	default:
		return fmt.Sprintf("%d", depth)
	}
}

// convertXGMoveToString converts XG move format to readable string
// XG move format: [from, to, from, to, ...] where:
//   - 1-24 are board points (from active player's perspective)
//   - 25 is the bar
//   - -2 is bear off
//   - -1 is unused/end of move
//
// Moves in XG files and analysis are stored from the player on roll's perspective.
// They should be displayed as-is without mirroring, per standard backgammon notation.
func (d *Database) convertXGMoveToString(playedMove [8]int32, activePlayer int32) string {
	// Note: activePlayer is kept for API compatibility but moves don't need transformation
	// Moves are always from the roller's perspective (24 = furthest from home, 1 = closest)
	_ = activePlayer // unused but kept for signature compatibility

	// Parse raw moves into from/to pairs
	var fromPts []int32
	var toPts []int32
	for i := 0; i < 8; i += 2 {
		from := playedMove[i]
		to := playedMove[i+1]
		// Check for end of move marker (-1 when from is also -1)
		if from == -1 {
			break
		}
		// Handle implicit bear-off: XG sometimes encodes bear-off as to=-1 or to<=0
		// when the calculated destination (from - die) would be <= 0
		// This happens when bearing off from the home board (points 1-6)
		if from >= 1 && from <= 6 && to <= 0 && to != -2 {
			to = -2 // Convert to explicit bear-off
		}
		// Skip invalid point values (should be 1-24 for points, 25 for bar, -2 for bear off)
		if (from != 25 && from != -2 && (from < 1 || from > 24)) ||
			(to != 25 && to != -2 && (to < 1 || to > 24)) {
			break
		}
		fromPts = append(fromPts, from)
		toPts = append(toPts, to)
	}

	if len(fromPts) == 0 {
		return "Cannot Move"
	}

	// Try to merge consecutive checker slides (same checker moving through points)
	// E.g., 24/23 23/22 22/21 21/20 -> 24/20 when dice are all same
	fromPts, toPts = d.mergeSlides(fromPts, toPts)

	// Create sortable items
	type moveItem struct {
		from int32
		to   int32
	}
	items := make([]moveItem, len(fromPts))
	for i := range fromPts {
		items[i] = moveItem{from: fromPts[i], to: toPts[i]}
	}

	// Sort moves by 'from' point descending (standard backgammon notation)
	// bar (25) comes first, then higher points before lower points
	sort.Slice(items, func(i, j int) bool {
		return items[i].from > items[j].from
	})

	// Format each move as string
	formatPoint := func(p int32) string {
		if p == 25 {
			return "bar"
		} else if p == -2 {
			return "off"
		} else if p >= 1 && p <= 24 {
			return fmt.Sprintf("%d", p)
		}
		return fmt.Sprintf("?%d", p)
	}

	// Build move string, grouping identical moves with multiplier
	var moves []string
	for i := 0; i < len(items); {
		item := items[i]
		count := 1
		// Count consecutive identical moves
		for j := i + 1; j < len(items); j++ {
			if items[j].from == item.from && items[j].to == item.to {
				count++
			} else {
				break
			}
		}
		if count > 1 {
			moves = append(moves, fmt.Sprintf("%s/%s(%d)", formatPoint(item.from), formatPoint(item.to), count))
		} else {
			moves = append(moves, fmt.Sprintf("%s/%s", formatPoint(item.from), formatPoint(item.to)))
		}
		i += count
	}

	return strings.Join(moves, " ")
}

// convertXGMoveToStringWithHits converts XG move format to readable string with hit indicators (*)
// It uses the initial position to detect when a blot is hit
// Moves are displayed from the player on roll's perspective (standard notation).
// activePlayer: XG encoding: 1 = Player 1 (X), -1 = Player 2 (O)
func (d *Database) convertXGMoveToStringWithHits(playedMove [8]int32, initialPos *xgparser.Position, activePlayer int32) string {
	if initialPos == nil {
		return d.convertXGMoveToString(playedMove, activePlayer)
	}

	// Note: No mirroring needed - moves are always from the roller's perspective

	// Create a mutable copy of the position to track changes as we process moves
	// XG position format: Checkers[1-24] are points 1-24 (1-based indexing)
	// [0]=opponent's bar, [25]=player's bar
	// Positive values = player's checkers, negative = opponent's checkers
	positionCopy := make([]int8, 26)
	copy(positionCopy, initialPos.Checkers[:])

	// Parse raw moves into from/to pairs and track hits
	var items []xgMoveWithHit
	for i := 0; i < 8; i += 2 {
		from := playedMove[i]
		to := playedMove[i+1]
		// Check for end of move marker (-1 when from is also -1)
		if from == -1 {
			break
		}
		// Handle implicit bear-off: XG sometimes encodes bear-off as to=-1 or to<=0
		// when the calculated destination (from - die) would be <= 0
		// This happens when bearing off from the home board (points 1-6)
		if from >= 1 && from <= 6 && to <= 0 && to != -2 {
			to = -2 // Convert to explicit bear-off
		}
		// Skip invalid point values (should be 1-24 for points, 25 for bar, -2 for bear off)
		if (from != 25 && from != -2 && (from < 1 || from > 24)) ||
			(to != 25 && to != -2 && (to < 1 || to > 24)) {
			break
		}

		// Check if this move hits a blot
		// The destination point must have exactly one opponent checker (negative value in XG format)
		isHit := false
		if to >= 1 && to <= 24 {
			// Position.Checkers uses 1-based indexing: Checkers[1] = point 1, Checkers[24] = point 24
			if positionCopy[to] == -1 {
				isHit = true
				// Update position: opponent checker goes to bar
				positionCopy[to] = 0
			}
		}

		// Update position: move our checker
		if from >= 1 && from <= 24 {
			// Position.Checkers uses 1-based indexing
			if positionCopy[from] > 0 {
				positionCopy[from]--
			}
		} else if from == 25 {
			// From bar - player's bar is at index 25
			if positionCopy[25] > 0 {
				positionCopy[25]--
			}
		}

		if to >= 1 && to <= 24 {
			// Position.Checkers uses 1-based indexing
			positionCopy[to]++
		}

		// Store directly - no conversion needed since moves are already in roller's perspective
		items = append(items, xgMoveWithHit{from: from, to: to, isHit: isHit})
	}

	if len(items) == 0 {
		return "Cannot Move"
	}

	// Try to merge consecutive checker slides - but preserve hit info for the final move
	// For slides, only the last move in a chain can be a hit
	items = d.mergeSlidesWithHits(items)

	// Sort moves by 'from' point descending (standard backgammon notation)
	sort.Slice(items, func(i, j int) bool {
		return items[i].from > items[j].from
	})

	// Format each move as string
	formatPoint := func(p int32) string {
		if p == 25 {
			return "bar"
		} else if p == -2 {
			return "off"
		} else if p >= 1 && p <= 24 {
			return fmt.Sprintf("%d", p)
		}
		return fmt.Sprintf("?%d", p)
	}

	// Build move string, grouping identical moves with multiplier
	var moves []string
	for i := 0; i < len(items); {
		item := items[i]
		count := 1
		allHits := item.isHit
		// Count consecutive identical moves
		for j := i + 1; j < len(items); j++ {
			if items[j].from == item.from && items[j].to == item.to {
				count++
				allHits = allHits && items[j].isHit
			} else {
				break
			}
		}

		hitMarker := ""
		if item.isHit || allHits {
			hitMarker = "*"
		}

		if count > 1 {
			moves = append(moves, fmt.Sprintf("%s/%s%s(%d)", formatPoint(item.from), formatPoint(item.to), hitMarker, count))
		} else {
			moves = append(moves, fmt.Sprintf("%s/%s%s", formatPoint(item.from), formatPoint(item.to), hitMarker))
		}
		i += count
	}

	return strings.Join(moves, " ")
}

// mergeSlidesWithHits merges consecutive moves of the same checker, preserving hit info
// For example: 14/12 12/8 becomes 14/8, but only if 12 was just a waypoint (not hit)
// If there was a hit at the intermediate point, we keep both moves to show the hit
func (d *Database) mergeSlidesWithHits(items []xgMoveWithHit) []xgMoveWithHit {
	if len(items) <= 1 {
		return items
	}

	// Try to merge chains: if move[i].to == move[j].from and move[i] is not a hit,
	// they can be merged (the intermediate point was just a waypoint)
	result := make([]xgMoveWithHit, 0, len(items))
	used := make([]bool, len(items))

	for i := 0; i < len(items); i++ {
		if used[i] {
			continue
		}

		// Start a chain from this item
		chainFrom := items[i].from
		chainTo := items[i].to
		chainHit := items[i].isHit
		used[i] = true

		// Only extend if the current segment doesn't end with a hit
		// (we want to show hits, so don't merge past them)
		if !chainHit {
			// Extend chain forward: find items where from == chainTo
			for changed := true; changed; {
				changed = false
				for j := 0; j < len(items); j++ {
					if used[j] {
						continue
					}
					if items[j].from == chainTo {
						chainTo = items[j].to
						chainHit = items[j].isHit
						used[j] = true
						changed = true
						// If this new segment has a hit, stop extending
						if chainHit {
							break
						}
					}
				}
				if chainHit {
					break
				}
			}
		}

		result = append(result, xgMoveWithHit{from: chainFrom, to: chainTo, isHit: chainHit})
	}

	return result
}

// xgMoveWithHit represents a single move in XG format with hit information
type xgMoveWithHit struct {
	from  int32
	to    int32
	isHit bool
}

// mergeSlides merges consecutive moves of the same checker
// For example: 14/12 12/8 becomes 14/8
func (d *Database) mergeSlides(fromPts, toPts []int32) ([]int32, []int32) {
	if len(fromPts) <= 1 {
		return fromPts, toPts
	}

	// Count how many times each destination point is used
	// If multiple moves end at the same point, we should NOT merge through that point
	// because it means different checkers are moving, not the same checker sliding
	toCount := make(map[int32]int)
	for _, t := range toPts {
		toCount[t]++
	}

	// Also count how many times each point is a source
	fromCount := make(map[int32]int)
	for _, f := range fromPts {
		fromCount[f]++
	}

	// Try to merge chains: if move[i].to == move[j].from, they can be merged
	// BUT only if that intermediate point appears exactly once as a destination AND once as a source
	resultFrom := make([]int32, 0, len(fromPts))
	resultTo := make([]int32, 0, len(toPts))
	used := make([]bool, len(fromPts))

	for i := 0; i < len(fromPts); i++ {
		if used[i] {
			continue
		}

		// Start a chain from this item
		chainFrom := fromPts[i]
		chainTo := toPts[i]
		used[i] = true

		// Extend chain forward: find items where from == chainTo
		// Only merge if the intermediate point is not used by multiple checkers
		for changed := true; changed; {
			changed = false
			for j := 0; j < len(fromPts); j++ {
				if used[j] {
					continue
				}
				if fromPts[j] == chainTo {
					// Check if this is a valid merge (same checker moving)
					// Don't merge if chainTo is a destination for multiple moves
					// or if chainTo is a source for multiple moves
					// This indicates different checkers
					if toCount[chainTo] > 1 || fromCount[chainTo] > 1 {
						continue // Don't merge - different checkers
					}
					chainTo = toPts[j]
					used[j] = true
					changed = true
				}
			}
		}

		resultFrom = append(resultFrom, chainFrom)
		resultTo = append(resultTo, chainTo)
	}

	return resultFrom, resultTo
}

// convertCubeAction converts cube action code to string
func (d *Database) convertCubeAction(action int32) string {
	switch action {
	case 0:
		return "No Double"
	case 1:
		return "Double"
	case 2:
		return "Take"
	case 3:
		return "Pass"
	default:
		return fmt.Sprintf("Unknown(%d)", action)
	}
}
