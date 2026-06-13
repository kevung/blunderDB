package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/gnubgparser"
	"github.com/kevung/xgparser/xgparser"
)

// writeImportedMatch persists a mapped MatchGraph through the storage backend,
// shared by the format-specific Import* methods that delegate to the ingest
// pipeline. It preserves the GUI/CLI duplicate contract: an exact same-format
// re-import returns ErrDuplicateMatch (the ingest layer reports it as a silent
// skip), while a cross-format canonical duplicate is enriched in place and
// returns the existing match id without error. Callers must hold d.mu.
func (d *Database) writeImportedMatch(ctx context.Context, graph *ingest.MatchGraph) (int64, error) {
	tx, err := d.store.BeginTx(ctx)
	if err != nil {
		return 0, err
	}
	res, err := ingest.WriteMatch(ctx, tx, "", graph, nil)
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	if res.Skipped {
		_ = tx.Rollback()
		return 0, ErrDuplicateMatch
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return res.MatchID, nil
}

// writeImportedPosition persists the positions of a single-position import
// (XGP / BGF text) through the storage backend and returns the id of the first
// stored position. Positions dedup by Zobrist hash, so re-importing the same
// position returns its existing id. Callers must hold d.mu.
func (d *Database) writeImportedPosition(graphs []ingest.PositionGraph) (int64, error) {
	if len(graphs) == 0 {
		return 0, fmt.Errorf("no position to import")
	}
	ctx := context.Background()
	tx, err := d.store.BeginTx(ctx)
	if err != nil {
		return 0, err
	}
	var firstID int64
	for i := range graphs {
		id, err := ingest.WritePosition(ctx, tx, "", &graphs[i])
		if err != nil {
			_ = tx.Rollback()
			return 0, err
		}
		if i == 0 {
			firstID = id
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return firstID, nil
}

// enginePriority returns a sort priority for analysis engines.
// XG gets priority 0 (first), GNUbg gets 1, unknown/empty gets 2.
func enginePriority(engine string) int {
	switch strings.ToLower(engine) {
	case "xg":
		return 0
	case "gnubg":
		return 1
	default:
		return 2
	}
}

// sortCubeAnalysesByEngine sorts cube analyses so XG comes first, then GNUbg, then others.
func sortCubeAnalysesByEngine(analyses []DoublingCubeAnalysis) {
	sort.SliceStable(analyses, func(i, j int) bool {
		return enginePriority(analyses[i].AnalysisEngine) < enginePriority(analyses[j].AnalysisEngine)
	})
}

// mergeCheckerMoves merges two sets of checker moves, avoiding duplicates
// Moves are considered duplicates if they have the same move string
// Returns moves sorted by equity (highest first), with XG engine preferred as tiebreaker
func mergeCheckerMoves(existing, incoming []CheckerMove) []CheckerMove {
	// Use a map to track unique moves by their move string
	moveMap := make(map[string]CheckerMove)

	// Add existing moves to the map
	for _, m := range existing {
		moveMap[m.Move] = m
	}

	// Add incoming moves, prefer incoming if there's a conflict (newer analysis)
	for _, m := range incoming {
		if existingMove, exists := moveMap[m.Move]; exists {
			// If the incoming move has the same depth or higher quality analysis, use it
			// Otherwise keep the existing one
			if m.AnalysisDepth >= existingMove.AnalysisDepth {
				moveMap[m.Move] = m
			}
		} else {
			moveMap[m.Move] = m
		}
	}

	// Convert map to slice
	result := make([]CheckerMove, 0, len(moveMap))
	for _, m := range moveMap {
		result = append(result, m)
	}

	// Sort by equity (highest first), with XG engine preferred as tiebreaker
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Equity != result[j].Equity {
			return result[i].Equity > result[j].Equity
		}
		return enginePriority(result[i].AnalysisEngine) < enginePriority(result[j].AnalysisEngine)
	})

	// Recalculate equity errors relative to the best move
	if len(result) > 0 {
		bestEquity := result[0].Equity
		for i := range result {
			result[i].Index = i
			if i == 0 {
				result[i].EquityError = nil
			} else {
				diff := bestEquity - result[i].Equity
				result[i].EquityError = &diff
			}
		}
	}

	return result
}

// mergePlayedMoves merges played moves/cube actions, avoiding duplicates
func mergePlayedMoves(existing, incoming []string) []string {
	// Use a map to track unique moves
	moveSet := make(map[string]bool)

	for _, m := range existing {
		if m != "" {
			moveSet[normalizeMove(m)] = true
		}
	}

	for _, m := range incoming {
		if m != "" {
			moveSet[normalizeMove(m)] = true
		}
	}

	// Convert map to slice
	result := make([]string, 0, len(moveSet))
	for m := range moveSet {
		result = append(result, m)
	}

	sort.Strings(result)
	return result
}

// normalizeMove is re-exported from package engine (see analysiscodec.go).
var normalizeMove = engine.NormalizeMove

// ErrDuplicateMatch is returned when attempting to import a match that already exists
var ErrDuplicateMatch = fmt.Errorf("duplicate match: this match has already been imported")

// ComputeMatchHash generates a unique hash for a match based on full match transcription
// This is used to detect duplicate imports - includes all moves and decisions
func ComputeMatchHash(match *xgparser.Match) string {
	var hashBuilder strings.Builder

	// Include metadata (normalized)
	p1 := strings.TrimSpace(strings.ToLower(match.Metadata.Player1Name))
	p2 := strings.TrimSpace(strings.ToLower(match.Metadata.Player2Name))
	hashBuilder.WriteString(fmt.Sprintf("meta:%s|%s|%d|", p1, p2, match.Metadata.MatchLength))

	// Include full game transcription
	for gameIdx, game := range match.Games {
		hashBuilder.WriteString(fmt.Sprintf("g%d:%d,%d,%d,%d|",
			gameIdx, game.InitialScore[0], game.InitialScore[1], game.Winner, game.PointsWon))

		// Include all moves in the game
		for moveIdx, move := range game.Moves {
			hashBuilder.WriteString(fmt.Sprintf("m%d:%s,", moveIdx, move.MoveType))

			if move.CheckerMove != nil {
				// Include dice and played move
				hashBuilder.WriteString(fmt.Sprintf("d%d%d,p%v|",
					move.CheckerMove.Dice[0], move.CheckerMove.Dice[1],
					move.CheckerMove.PlayedMove))
			}

			if move.CubeMove != nil {
				// Include cube action details
				hashBuilder.WriteString(fmt.Sprintf("c%d|", move.CubeMove.CubeAction))
			}
		}
	}

	// Compute SHA256 hash of the full transcription
	hash := sha256.Sum256([]byte(hashBuilder.String()))
	return hex.EncodeToString(hash[:])
}

// computeMatchHashFromStoredData computes a hash for existing matches in the database
// This is used during migration when we don't have access to the original XG file
func computeMatchHashFromStoredData(db *sql.DB, matchID int64, p1Name, p2Name string, matchLength int32) string {
	var hashBuilder strings.Builder

	// Include metadata (normalized)
	p1 := strings.TrimSpace(strings.ToLower(p1Name))
	p2 := strings.TrimSpace(strings.ToLower(p2Name))
	hashBuilder.WriteString(fmt.Sprintf("meta:%s|%s|%d|", p1, p2, matchLength))

	// Query all games for this match
	gameRows, err := db.Query(`
		SELECT id, game_number, initial_score_1, initial_score_2, winner, points_won 
		FROM game WHERE match_id = ? ORDER BY game_number`, matchID)
	if err != nil {
		// Fallback to simple hash
		hash := sha256.Sum256([]byte(hashBuilder.String()))
		return hex.EncodeToString(hash[:])
	}
	defer gameRows.Close()

	for gameRows.Next() {
		var gameID int64
		var gameNum, initScore1, initScore2, winner, pointsWon int32
		if err := gameRows.Scan(&gameID, &gameNum, &initScore1, &initScore2, &winner, &pointsWon); err != nil {
			continue
		}

		hashBuilder.WriteString(fmt.Sprintf("g%d:%d,%d,%d,%d|", gameNum, initScore1, initScore2, winner, pointsWon))

		// Query all moves for this game
		moveRows, err := db.Query(`
			SELECT move_number, move_type, dice_1, dice_2, checker_move, cube_action 
			FROM move WHERE game_id = ? ORDER BY move_number`, gameID)
		if err != nil {
			continue
		}

		for moveRows.Next() {
			var moveNum int32
			var moveType string
			var dice1, dice2 int32
			var checkerMove, cubeAction sql.NullString
			if err := moveRows.Scan(&moveNum, &moveType, &dice1, &dice2, &checkerMove, &cubeAction); err != nil {
				continue
			}

			hashBuilder.WriteString(fmt.Sprintf("m%d:%s,", moveNum, moveType))
			if moveType == "checker" && checkerMove.Valid {
				hashBuilder.WriteString(fmt.Sprintf("d%d%d,p%s|", dice1, dice2, checkerMove.String))
			}
			if moveType == "cube" && cubeAction.Valid {
				hashBuilder.WriteString(fmt.Sprintf("c%s|", cubeAction.String))
			}
		}
		if err := moveRows.Err(); err != nil {
			return ""
		}
		moveRows.Close()
	}
	if err := gameRows.Err(); err != nil {
		return ""
	}

	// Compute SHA256 hash
	hash := sha256.Sum256([]byte(hashBuilder.String()))
	return hex.EncodeToString(hash[:])
}

// CheckMatchExists checks if a match with the given hash already exists in the database
// Returns the existing match ID if found, 0 otherwise
func (d *Database) CheckMatchExists(matchHash string) (int64, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var existingID int64
	err := d.db.QueryRow(`SELECT id FROM match WHERE match_hash = ?`, matchHash).Scan(&existingID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("error checking for duplicate match: %w", err)
	}
	return existingID, nil
}

// maxCanonicalDicePerGame limits how many dice per game are included in the canonical hash.
// Different file formats (XG, SGF, MAT) handle end-of-game dice differently:
// XG records game-ending rolls that SGF/MAT may omit, and MAT can diverge
// in later moves due to parser edge cases. Using only the first N dice avoids
// these differences while providing strong match identification.
// 10 dice per game * 21 possible outcomes * 7+ games = astronomically low collision probability.
const maxCanonicalDicePerGame = 10

// ComputeCanonicalMatchHashFromXG computes a format-independent match hash from XG data.
// This hash uses only the first N dice per game (physical events identical across all export
// formats) plus normalized player names, match length, and game count, making it identical
// whether the match was imported from XG, GnuBG (SGF/MAT), or BGBlitz (BGF) format.
func ComputeCanonicalMatchHashFromXG(match *xgparser.Match) string {
	var hashBuilder strings.Builder

	// Normalized player names (sorted alphabetically for consistency)
	p1 := strings.TrimSpace(strings.ToLower(match.Metadata.Player1Name))
	p2 := strings.TrimSpace(strings.ToLower(match.Metadata.Player2Name))
	if p1 > p2 {
		p1, p2 = p2, p1
	}
	hashBuilder.WriteString(fmt.Sprintf("canonical2:%s|%s|%d|%d|", p1, p2, match.Metadata.MatchLength, len(match.Games)))

	for gameIdx, game := range match.Games {
		hashBuilder.WriteString(fmt.Sprintf("g%d|", gameIdx))
		diceCount := 0
		for _, move := range game.Moves {
			if diceCount >= maxCanonicalDicePerGame {
				break
			}
			if move.MoveType == "checker" && move.CheckerMove != nil {
				d1 := move.CheckerMove.Dice[0]
				d2 := move.CheckerMove.Dice[1]
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

// ComputeCanonicalMatchHashFromGnuBG computes a format-independent match hash from GnuBG data.
// Must produce the same hash as ComputeCanonicalMatchHashFromXG for the same match.
// Uses only the first N dice per game for cross-format compatibility.
func ComputeCanonicalMatchHashFromGnuBG(match *gnubgparser.Match) string {
	var hashBuilder strings.Builder

	p1 := strings.TrimSpace(strings.ToLower(match.Metadata.Player1))
	p2 := strings.TrimSpace(strings.ToLower(match.Metadata.Player2))
	if p1 > p2 {
		p1, p2 = p2, p1
	}
	hashBuilder.WriteString(fmt.Sprintf("canonical2:%s|%s|%d|%d|", p1, p2, match.Metadata.MatchLength, len(match.Games)))

	for gameIdx, game := range match.Games {
		hashBuilder.WriteString(fmt.Sprintf("g%d|", gameIdx))
		diceCount := 0
		for _, moveRec := range game.Moves {
			if diceCount >= maxCanonicalDicePerGame {
				break
			}
			if moveRec.Type == "move" {
				d1 := moveRec.Dice[0]
				d2 := moveRec.Dice[1]
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
