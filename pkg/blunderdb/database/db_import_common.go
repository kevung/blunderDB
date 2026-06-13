package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"
	"time"

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

// importCache is a per-import Zobrist-hash → position-ID lookup table.
// It prevents redundant SQL round-trips for positions that appear more than once
// inside a single match (common in bearoff analyses and canonical-duplicate imports).
type importCache struct {
	m map[uint64]int64 // zobristHash → positionID
}

func newImportCache() *importCache {
	return &importCache{m: make(map[uint64]int64)}
}

// savePositionInTxWithCache saves a position within a transaction using a Zobrist-hash
// based cache for deduplication.  Positions are normalized before storage
// (player_on_roll = 0) to prevent storing duplicates.
//
// Algorithm:
//  1. Local cache hit  → return immediately (no SQL).
//  2. INSERT OR IGNORE → no-ops silently if zobrist_hash already exists (UNIQUE index).
//  3. SELECT id+state  → fetches the id whether we just inserted or it already existed.
//  4. Dedup hit        → verify the stored board matches the incoming position to detect
//     the ~2⁻⁴⁴ Zobrist-collision corner case; log a warning if they differ.
//  5. New row          → UPDATE state to embed the assigned ID (backwards-compat readers).
func (d *Database) savePositionInTxWithCache(tx *sql.Tx, position *Position, cache *importCache) (int64, error) {
	// Normalize: always store from the perspective of player on roll (player_on_roll == 0).
	normalizedPosition := position.NormalizeForStorage()
	normalizedPosition.ID = 0

	cols := populatePositionColumns(&normalizedPosition)
	hash := cols.ZobristHash

	// 1. Local cache hit — no SQL needed.
	if id, ok := cache.m[hash]; ok {
		return id, nil
	}

	compactState := encodeBoardCompact(normalizedPosition.Board)

	noContactInt := 0
	if cols.NoContact {
		noContactInt = 1
	}

	// 2. INSERT OR IGNORE — idempotent thanks to the UNIQUE index on zobrist_hash.
	res, err := tx.Exec(`
		INSERT OR IGNORE INTO position (
			zobrist_hash, decision_type, player_on_roll, dice_1, dice_2,
			cube_value, cube_owner, score_1, score_2,
			has_jacoby, has_beaver,
			pip_1, pip_2, pip_diff, off_1, off_2,
			back_checkers_1, back_checkers_2, no_contact,
			occupancy_1, occupancy_2, point_mask_1, point_mask_2,
			state
		) VALUES (?,?,?,?,?, ?,?,?,?,  ?,?,  ?,?,?,?,?,  ?,?,?,  ?,?,?,?,  ?)`,
		int64(hash), cols.DecisionType, normalizedPosition.PlayerOnRoll, cols.Dice1, cols.Dice2,
		cols.CubeValue, cols.CubeOwner, cols.Score1, cols.Score2,
		cols.HasJacoby, cols.HasBeaver,
		cols.Pip1, cols.Pip2, cols.PipDiff, cols.Off1, cols.Off2,
		cols.BackCheckers1, cols.BackCheckers2, noContactInt,
		int64(cols.Occupancy1), int64(cols.Occupancy2), int64(cols.PointMask1), int64(cols.PointMask2),
		compactState)
	if err != nil {
		return 0, err
	}
	rowsAffected, _ := res.RowsAffected()

	// 3. Fetch the id (state not needed — board is compared via compact string).
	var positionID int64
	var storedState string
	err = tx.QueryRow(`SELECT id, state FROM position WHERE zobrist_hash = ?`, int64(hash)).Scan(&positionID, &storedState)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch position by zobrist_hash: %w", err)
	}

	if rowsAffected == 0 {
		// 4. Dedup hit: structural collision guard.
		// Compare compact board strings — if they differ we have a genuine Zobrist
		// collision (probability ≈ 2⁻⁴⁴). For legacy JSON state, decode and
		// re-encode to compact for a fair comparison.
		storedBoard := storedState
		if !isCompactState(storedState) {
			var storedPos Position
			if jsonErr := json.Unmarshal([]byte(storedState), &storedPos); jsonErr == nil {
				storedBoard = encodeBoardCompact(storedPos.Board)
			}
		}
		if storedBoard != compactState {
			fmt.Fprintf(os.Stderr,
				"WARNING: Zobrist collision for hash %d — stored board differs from incoming; returning existing id %d\n",
				hash, positionID)
		}
	}

	cache.m[hash] = positionID
	return positionID, nil
}

// findOrCreatePositionForCanonicalDuplicate looks up or creates a position during
// canonical-duplicate imports (same match re-imported from a different format).
// Delegates to savePositionInTxWithCache which uses the Zobrist hash index and the
// per-import cache for efficient deduplication.
func (d *Database) findOrCreatePositionForCanonicalDuplicate(tx *sql.Tx, position *Position, cache *importCache) (int64, error) {
	return d.savePositionInTxWithCache(tx, position, cache)
}

// savePositionInTx saves a position within a transaction checking for duplicates via
// the Zobrist-hash unique index (no external cache — one INSERT+SELECT roundtrip each call).
func (d *Database) savePositionInTx(tx *sql.Tx, position *Position) (int64, error) {
	cache := newImportCache()
	return d.savePositionInTxWithCache(tx, position, cache)
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

// saveAnalysisInTx saves a PositionAnalysis within a transaction, merging with existing analysis if present
func (d *Database) saveAnalysisInTx(tx *sql.Tx, positionID int64, analysis PositionAnalysis) error {
	// Ensure the positionID is set in the analysis
	analysis.PositionID = int(positionID)

	// Update last modified date
	analysis.LastModifiedDate = time.Now()

	// Check if an analysis already exists for the given position ID
	var existingID int64
	var existingAnalysisData []byte
	err := tx.QueryRow(`SELECT id, data FROM analysis WHERE position_id = ?`, positionID).Scan(&existingID, &existingAnalysisData)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if existingID > 0 {
		// Parse existing analysis
		existingAnalysis, err := decodeAnalysisFromStorage(existingAnalysisData)
		if err != nil {
			return err
		}

		// Preserve the existing creation date
		analysis.CreationDate = existingAnalysis.CreationDate

		// Merge checker analysis if both exist
		if existingAnalysis.CheckerAnalysis != nil && analysis.CheckerAnalysis != nil {
			analysis.CheckerAnalysis.Moves = mergeCheckerMoves(
				existingAnalysis.CheckerAnalysis.Moves,
				analysis.CheckerAnalysis.Moves,
			)
		} else if existingAnalysis.CheckerAnalysis != nil && analysis.CheckerAnalysis == nil {
			// Keep existing checker analysis if new one is nil
			analysis.CheckerAnalysis = existingAnalysis.CheckerAnalysis
		}

		// Merge doubling cube analysis - keep all engine analyses in AllCubeAnalyses
		if existingAnalysis.DoublingCubeAnalysis != nil && analysis.DoublingCubeAnalysis != nil {
			// Both have cube analysis - check if they're from different engines
			existingEngine := existingAnalysis.DoublingCubeAnalysis.AnalysisEngine
			incomingEngine := analysis.DoublingCubeAnalysis.AnalysisEngine

			if existingEngine != incomingEngine && existingEngine != "" && incomingEngine != "" {
				// Different engines: build AllCubeAnalyses array with both
				allCube := make([]DoublingCubeAnalysis, 0)
				// Add existing engine's cube analyses
				if len(existingAnalysis.AllCubeAnalyses) > 0 {
					allCube = append(allCube, existingAnalysis.AllCubeAnalyses...)
				} else {
					allCube = append(allCube, *existingAnalysis.DoublingCubeAnalysis)
				}
				// Add incoming if not already present for this engine
				hasIncoming := false
				for _, ca := range allCube {
					if ca.AnalysisEngine == incomingEngine {
						hasIncoming = true
						break
					}
				}
				if !hasIncoming {
					allCube = append(allCube, *analysis.DoublingCubeAnalysis)
				}
				sortCubeAnalysesByEngine(allCube)
				analysis.AllCubeAnalyses = allCube
				// Primary DoublingCubeAnalysis stays as the incoming one
			} else {
				// Same engine or empty engine - keep incoming, preserve AllCubeAnalyses
				if len(existingAnalysis.AllCubeAnalyses) > 0 {
					analysis.AllCubeAnalyses = existingAnalysis.AllCubeAnalyses
				}
			}
		} else if existingAnalysis.DoublingCubeAnalysis != nil && analysis.DoublingCubeAnalysis == nil {
			analysis.DoublingCubeAnalysis = existingAnalysis.DoublingCubeAnalysis
			analysis.AllCubeAnalyses = existingAnalysis.AllCubeAnalyses
		}

		// Merge played moves (support both old single field and new array field)
		existingPlayedMoves := existingAnalysis.PlayedMoves
		if existingAnalysis.PlayedMove != "" && len(existingPlayedMoves) == 0 {
			existingPlayedMoves = []string{existingAnalysis.PlayedMove}
		}
		incomingPlayedMoves := analysis.PlayedMoves
		if analysis.PlayedMove != "" && len(incomingPlayedMoves) == 0 {
			incomingPlayedMoves = []string{analysis.PlayedMove}
		}
		analysis.PlayedMoves = mergePlayedMoves(existingPlayedMoves, incomingPlayedMoves)

		// Merge played cube actions
		existingCubeActions := existingAnalysis.PlayedCubeActions
		if existingAnalysis.PlayedCubeAction != "" && len(existingCubeActions) == 0 {
			existingCubeActions = []string{existingAnalysis.PlayedCubeAction}
		}
		incomingCubeActions := analysis.PlayedCubeActions
		if analysis.PlayedCubeAction != "" && len(incomingCubeActions) == 0 {
			incomingCubeActions = []string{analysis.PlayedCubeAction}
		}
		analysis.PlayedCubeActions = mergePlayedMoves(existingCubeActions, incomingCubeActions)

		// Clear deprecated single fields after merging
		analysis.PlayedMove = ""
		analysis.PlayedCubeAction = ""

		// Sort checker moves by equity after merging
		if analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 0 {
			sort.Slice(analysis.CheckerAnalysis.Moves, func(i, j int) bool {
				return analysis.CheckerAnalysis.Moves[i].Equity > analysis.CheckerAnalysis.Moves[j].Equity
			})
			// Recalculate indices and equity errors
			bestEquity := analysis.CheckerAnalysis.Moves[0].Equity
			for i := range analysis.CheckerAnalysis.Moves {
				analysis.CheckerAnalysis.Moves[i].Index = i
				if i == 0 {
					analysis.CheckerAnalysis.Moves[i].EquityError = nil
				} else {
					diff := bestEquity - analysis.CheckerAnalysis.Moves[i].Equity
					analysis.CheckerAnalysis.Moves[i].EquityError = &diff
				}
			}
		}

		// Update the existing analysis
		roundAnalysisForStorage(&analysis)
		analysisData, err := encodeAnalysisForStorage(&analysis)
		if err != nil {
			return err
		}
		playedMove := ""
		if len(analysis.PlayedMoves) > 0 {
			playedMove = analysis.PlayedMoves[0]
		}
		playedCubeAction := ""
		if len(analysis.PlayedCubeActions) > 0 {
			playedCubeAction = analysis.PlayedCubeActions[0]
		}
		aCols := populateAnalysisColumns(&analysis, playedMove, playedCubeAction)
		_, err = tx.Exec(`UPDATE analysis SET
			data=?, best_cube_action=?, cube_error=?,
			best_move_equity_error=?,
			player1_win_rate=?, player1_gammon_rate=?, player1_backgammon_rate=?,
			player2_win_rate=?, player2_gammon_rate=?, player2_backgammon_rate=?,
			is_forced=?, is_close_cube=?
			WHERE id=?`,
			analysisData,
			aCols.BestCubeAction, aCols.CubeError,
			aCols.BestMoveEquityError,
			aCols.Player1WinRate, aCols.Player1GammonRate, aCols.Player1BackgammonRate,
			aCols.Player2WinRate, aCols.Player2GammonRate, aCols.Player2BackgammonRate,
			aCols.IsForced, aCols.IsCloseCube,
			existingID)
		if err != nil {
			return err
		}
	} else {
		// Set creation date if not already set
		if analysis.CreationDate.IsZero() {
			analysis.CreationDate = time.Now()
		}

		// Convert single played move to array if needed
		if analysis.PlayedMove != "" && len(analysis.PlayedMoves) == 0 {
			analysis.PlayedMoves = []string{analysis.PlayedMove}
			analysis.PlayedMove = ""
		}
		if analysis.PlayedCubeAction != "" && len(analysis.PlayedCubeActions) == 0 {
			analysis.PlayedCubeActions = []string{analysis.PlayedCubeAction}
			analysis.PlayedCubeAction = ""
		}

		// Sort checker moves by equity
		if analysis.CheckerAnalysis != nil && len(analysis.CheckerAnalysis.Moves) > 0 {
			sort.Slice(analysis.CheckerAnalysis.Moves, func(i, j int) bool {
				return analysis.CheckerAnalysis.Moves[i].Equity > analysis.CheckerAnalysis.Moves[j].Equity
			})
			// Recalculate indices and equity errors
			bestEquity := analysis.CheckerAnalysis.Moves[0].Equity
			for i := range analysis.CheckerAnalysis.Moves {
				analysis.CheckerAnalysis.Moves[i].Index = i
				if i == 0 {
					analysis.CheckerAnalysis.Moves[i].EquityError = nil
				} else {
					diff := bestEquity - analysis.CheckerAnalysis.Moves[i].Equity
					analysis.CheckerAnalysis.Moves[i].EquityError = &diff
				}
			}
		}

		// Insert a new analysis
		roundAnalysisForStorage(&analysis)
		analysisData, err := encodeAnalysisForStorage(&analysis)
		if err != nil {
			return err
		}
		playedMove := ""
		if len(analysis.PlayedMoves) > 0 {
			playedMove = analysis.PlayedMoves[0]
		}
		playedCubeAction := ""
		if len(analysis.PlayedCubeActions) > 0 {
			playedCubeAction = analysis.PlayedCubeActions[0]
		}
		aCols := populateAnalysisColumns(&analysis, playedMove, playedCubeAction)
		_, err = tx.Exec(`INSERT INTO analysis (
			position_id, data,
			best_cube_action, cube_error, best_move_equity_error,
			player1_win_rate, player1_gammon_rate, player1_backgammon_rate,
			player2_win_rate, player2_gammon_rate, player2_backgammon_rate,
			is_forced, is_close_cube
		) VALUES (?,?, ?,?,?, ?,?,?, ?,?,?, ?,?)`,
			positionID, analysisData,
			aCols.BestCubeAction, aCols.CubeError, aCols.BestMoveEquityError,
			aCols.Player1WinRate, aCols.Player1GammonRate, aCols.Player1BackgammonRate,
			aCols.Player2WinRate, aCols.Player2GammonRate, aCols.Player2BackgammonRate,
			aCols.IsForced, aCols.IsCloseCube)
		if err != nil {
			return err
		}
	}

	// Flag the position as a take/pass cube response if any recorded played cube
	// action is a response (Take/Pass/Drop) rather than a doubling decision. This
	// lets search distinguish "double/no-double" from "take/pass" cube decisions.
	// OR semantics across matches for a deduped position: only ever set to 1,
	// never reset to 0.
	for _, action := range analysis.PlayedCubeActions {
		if engine.IsResponseCubeAction(action) {
			if _, err := tx.Exec(`UPDATE position SET is_cube_response = 1 WHERE id = ?`, positionID); err != nil {
				return err
			}
			break
		}
	}

	return nil
}

// ErrDuplicateMatch is returned when attempting to import a match that already exists
var ErrDuplicateMatch = fmt.Errorf("duplicate match: this match has already been imported")

// parseMatchDate tries multiple date formats and returns the parsed time.
// Returns time.Now() if dateStr is empty or no format matches.
func parseMatchDate(dateStr string) time.Time {
	if dateStr == "" {
		return time.Now()
	}
	for _, layout := range []string{
		"Jan 2, 2006",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"2006/01/02",
		"01/02/2006",
		"January 2, 2006",
		time.RFC3339,
	} {
		if t, err := time.Parse(layout, dateStr); err == nil {
			return t
		}
	}
	return time.Now()
}

// checkDuplicateMatchLocked checks both format-specific and canonical hash
// for duplicate matches. Must be called with d.mu held.
// Returns (canonicalMatchID, isCanonicalDuplicate, error).
// Returns ErrDuplicateMatch if the exact same-format hash already exists.
func (d *Database) checkDuplicateMatchLocked(matchHash, canonicalHash string) (int64, bool, error) {
	existingMatchID, err := d.checkMatchExistsLocked(matchHash)
	if err != nil {
		return 0, false, fmt.Errorf("failed to check for duplicate match: %w", err)
	}
	if existingMatchID > 0 {
		return 0, false, ErrDuplicateMatch
	}

	canonicalMatchID, err := d.checkCanonicalMatchExistsLocked(canonicalHash)
	if err != nil {
		return 0, false, fmt.Errorf("failed to check for canonical duplicate: %w", err)
	}
	return canonicalMatchID, canonicalMatchID > 0, nil
}

// moveAnalysisRow holds the pre-extracted values for a move_analysis INSERT.
// Each importer converts its format-specific data into this struct, then
// calls insertMoveAnalysisRow for the actual INSERT.
type moveAnalysisRow struct {
	MoveID                 int64
	AnalysisType           string // "checker" or "cube"
	Depth                  string
	Equity                 int64
	WinRate                int64
	GammonRate             int64
	BackgammonRate         int64
	OpponentWinRate        int64
	OpponentGammonRate     int64
	OpponentBackgammonRate int64
}

// insertMoveAnalysisRow inserts a single row into the move_analysis table.
func insertMoveAnalysisRow(tx *sql.Tx, row moveAnalysisRow) error {
	_, err := tx.Exec(`
		INSERT INTO move_analysis (move_id, analysis_type, depth, equity, equity_error,
		                           win_rate, gammon_rate, backgammon_rate,
		                           opponent_win_rate, opponent_gammon_rate, opponent_backgammon_rate)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, row.MoveID, row.AnalysisType, row.Depth,
		row.Equity, int64(0),
		row.WinRate, row.GammonRate, row.BackgammonRate,
		row.OpponentWinRate, row.OpponentGammonRate, row.OpponentBackgammonRate)
	return err
}

// computeBestCubeAction determines the best cube action and equity from the three
// cubeful equities. Returns (bestEquity, bestAction).
func computeBestCubeAction(cubefulNoDouble, cubefulDoubleTake, cubefulDoublePass float64) (float64, string) {
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
	return bestEquity, bestAction
}

// autoLinkTournament finds or creates a tournament by name and links it to the given match.
// Does nothing if eventName is empty.
func autoLinkTournament(tx *sql.Tx, matchID int64, eventName string) {
	eventName = strings.TrimSpace(eventName)
	if eventName == "" {
		return
	}
	var tournamentID int64
	err := tx.QueryRow(`SELECT id FROM tournament WHERE name = ?`, eventName).Scan(&tournamentID)
	if err != nil {
		// Tournament doesn't exist yet — create it
		res, err2 := tx.Exec(`INSERT INTO tournament (name, date, location) VALUES (?, '', '')`, eventName)
		if err2 == nil {
			tournamentID, _ = res.LastInsertId()
		}
	}
	if tournamentID > 0 {
		if _, err := tx.Exec(`UPDATE match SET tournament_id = ? WHERE id = ?`, tournamentID, matchID); err != nil {
			slog.Warn("failed to link match to tournament", "err", err)
		}
	}
}

// cubeAnalysisParams holds the format-independent cube analysis values needed to build
// a DoublingCubeAnalysis struct. Each importer extracts these from its format-specific types.
type cubeAnalysisParams struct {
	Depth                     string
	Engine                    string
	PlayerWinChances          float64
	PlayerGammonChances       float64
	PlayerBackgammonChances   float64
	OpponentWinChances        float64
	OpponentGammonChances     float64
	OpponentBackgammonChances float64
	CubelessNoDoubleEquity    float64
	CubelessDoubleEquity      float64
	CubefulNoDoubleEquity     float64
	CubefulDoubleTakeEquity   float64
	CubefulDoublePassEquity   float64
	WrongPassPercentage       float64
	WrongTakePercentage       float64
}

// buildDoublingCubeAnalysis creates a DoublingCubeAnalysis from the given params,
// computing best action and error deltas automatically.
func buildDoublingCubeAnalysis(p cubeAnalysisParams) DoublingCubeAnalysis {
	bestEquity, bestAction := computeBestCubeAction(p.CubefulNoDoubleEquity, p.CubefulDoubleTakeEquity, p.CubefulDoublePassEquity)
	return DoublingCubeAnalysis{
		AnalysisDepth:             p.Depth,
		AnalysisEngine:            p.Engine,
		PlayerWinChances:          p.PlayerWinChances,
		PlayerGammonChances:       p.PlayerGammonChances,
		PlayerBackgammonChances:   p.PlayerBackgammonChances,
		OpponentWinChances:        p.OpponentWinChances,
		OpponentGammonChances:     p.OpponentGammonChances,
		OpponentBackgammonChances: p.OpponentBackgammonChances,
		CubelessNoDoubleEquity:    p.CubelessNoDoubleEquity,
		CubelessDoubleEquity:      p.CubelessDoubleEquity,
		CubefulNoDoubleEquity:     p.CubefulNoDoubleEquity,
		CubefulNoDoubleError:      p.CubefulNoDoubleEquity - bestEquity,
		CubefulDoubleTakeEquity:   p.CubefulDoubleTakeEquity,
		CubefulDoubleTakeError:    p.CubefulDoubleTakeEquity - bestEquity,
		CubefulDoublePassEquity:   p.CubefulDoublePassEquity,
		CubefulDoublePassError:    p.CubefulDoublePassEquity - bestEquity,
		BestCubeAction:            bestAction,
		WrongPassPercentage:       p.WrongPassPercentage,
		WrongTakePercentage:       p.WrongTakePercentage,
	}
}

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

// CheckMatchExistsLocked is the same as CheckMatchExists but doesn't acquire the lock
// Use this when you already hold the database lock
func (d *Database) checkMatchExistsLocked(matchHash string) (int64, error) {
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

// checkCanonicalMatchExistsLocked checks if a match with the given canonical hash already exists
// Returns the existing match ID if found, 0 otherwise
func (d *Database) checkCanonicalMatchExistsLocked(canonicalHash string) (int64, error) {
	var existingID int64
	err := d.db.QueryRow(`SELECT id FROM match WHERE canonical_hash = ?`, canonicalHash).Scan(&existingID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("error checking for canonical match: %w", err)
	}
	return existingID, nil
}
