package database

import (
	"context"
	"database/sql"
	"errors"
	"sort"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The analysis-encoding helpers (zlib compression, scalar-column derivation,
// float rounding) live in package engine so the SQLite Storage backend can
// share them. These aliases keep this package compiling against the
// unqualified names. See pkg/blunderdb/engine/analysiscodec.go.
var (
	compressAnalysisData      = engine.CompressAnalysisData
	decompressAnalysisData    = engine.DecompressAnalysisData
	recompressAnalysisData    = engine.RecompressAnalysisData
	encodeAnalysisForStorage  = engine.EncodeAnalysisForStorage
	decodeAnalysisFromStorage = engine.DecodeAnalysisFromStorage
	computeIsCloseCube        = engine.ComputeIsCloseCube
	populateAnalysisColumns   = engine.PopulateAnalysisColumns
	roundToMillipoint         = engine.RoundToMillipoint
	roundToHundredthPercent   = engine.RoundToHundredthPercent
	roundAnalysisForStorage   = engine.RoundAnalysisForStorage
)

func (d *Database) SaveAnalysis(positionID int64, analysis PositionAnalysis) error {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	// Ensure the positionID is set in the analysis
	analysis.PositionID = int(positionID)

	// Update last modified date
	analysis.LastModifiedDate = time.Now()

	// Check if an analysis already exists for the given position ID
	var existingID int64
	var existingAnalysisData []byte
	err := d.db.QueryRow(`SELECT id, data FROM analysis WHERE position_id = ?`, positionID).Scan(&existingID, &existingAnalysisData)
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

		// Merge doubling cube analysis - keep existing if new is nil
		if existingAnalysis.DoublingCubeAnalysis != nil && analysis.DoublingCubeAnalysis == nil {
			analysis.DoublingCubeAnalysis = existingAnalysis.DoublingCubeAnalysis
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

	}

	// The store rounds, encodes and derives the scalar columns, then
	// inserts-or-updates the row keyed by position_id.
	return d.store.Analyses().Save(context.Background(), "", positionID, &analysis)
}

func (d *Database) LoadAnalysis(positionID int64) (*PositionAnalysis, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	loaded, err := d.store.Analyses().Load(context.Background(), "", positionID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			// Preserve the pre-delegation contract: callers expect sql.ErrNoRows.
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	analysis := *loaded

	// Load ALL played moves/cube actions from move table for this position
	// This supplements the PlayedMoves/PlayedCubeActions arrays stored in analysis
	rows, err := d.db.Query(`
		SELECT checker_move, cube_action
		FROM move
		WHERE position_id = ?
	`, positionID)

	if err == nil {
		defer rows.Close()

		// Collect all moves from the database
		dbCheckerMoves := make(map[string]bool)
		dbCubeActions := make(map[string]bool)

		for rows.Next() {
			var checkerMove sql.NullString
			var cubeAction sql.NullString
			if err := rows.Scan(&checkerMove, &cubeAction); err == nil {
				if checkerMove.Valid && checkerMove.String != "" {
					dbCheckerMoves[normalizeMove(checkerMove.String)] = true
				}
				if cubeAction.Valid && cubeAction.String != "" {
					dbCubeActions[cubeAction.String] = true
				}
			}
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}

		// Merge with existing PlayedMoves array
		existingMoves := make(map[string]bool)
		for _, m := range analysis.PlayedMoves {
			existingMoves[normalizeMove(m)] = true
		}
		// Backward compatibility: include old single PlayedMove field
		if analysis.PlayedMove != "" {
			existingMoves[normalizeMove(analysis.PlayedMove)] = true
		}

		// Combine all moves
		for m := range dbCheckerMoves {
			existingMoves[m] = true
		}

		// Convert to slice
		analysis.PlayedMoves = make([]string, 0, len(existingMoves))
		for m := range existingMoves {
			analysis.PlayedMoves = append(analysis.PlayedMoves, m)
		}
		sort.Strings(analysis.PlayedMoves)

		// Do the same for cube actions
		existingCubeActions := make(map[string]bool)
		for _, a := range analysis.PlayedCubeActions {
			existingCubeActions[a] = true
		}
		// Backward compatibility: include old single PlayedCubeAction field
		if analysis.PlayedCubeAction != "" {
			existingCubeActions[analysis.PlayedCubeAction] = true
		}

		for a := range dbCubeActions {
			existingCubeActions[a] = true
		}

		analysis.PlayedCubeActions = make([]string, 0, len(existingCubeActions))
		for a := range existingCubeActions {
			analysis.PlayedCubeActions = append(analysis.PlayedCubeActions, a)
		}
		sort.Strings(analysis.PlayedCubeActions)

		// For backward compatibility, also set the old single fields if there's exactly one
		if len(analysis.PlayedMoves) == 1 {
			analysis.PlayedMove = analysis.PlayedMoves[0]
		} else if len(analysis.PlayedMoves) > 0 {
			// Set to first one for backward compatibility with old frontend
			analysis.PlayedMove = analysis.PlayedMoves[0]
		}
		if len(analysis.PlayedCubeActions) == 1 {
			analysis.PlayedCubeAction = analysis.PlayedCubeActions[0]
		} else if len(analysis.PlayedCubeActions) > 0 {
			analysis.PlayedCubeAction = analysis.PlayedCubeActions[0]
		}
	}

	// Sort AllCubeAnalyses so XG comes first (for existing data in DB)
	if len(analysis.AllCubeAnalyses) > 0 {
		sortCubeAnalysesByEngine(analysis.AllCubeAnalyses)
	}

	return &analysis, nil
}

func (d *Database) DeleteAnalysis(positionID int64) error {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	_, err := d.db.Exec(`DELETE FROM analysis WHERE position_id = ?`, positionID)
	if err != nil {
		return err
	}
	return nil
}
