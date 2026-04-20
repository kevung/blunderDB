package main

import (
	"bytes"
	"compress/zlib"
	"database/sql"
	"encoding/json"
	"io"
	"math"
	"sort"
	"time"
)

// ---------------------------------------------------------------------------
// Analysis data compression helpers
// ---------------------------------------------------------------------------

// compressAnalysisData compresses raw JSON bytes using zlib (best compression).
func compressAnalysisData(jsonData []byte) ([]byte, error) {
	var buf bytes.Buffer
	w, err := zlib.NewWriterLevel(&buf, zlib.BestCompression)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(jsonData); err != nil {
		w.Close()
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// decompressAnalysisData auto-detects zlib-compressed data vs raw JSON.
// If the first byte is '{' the data is returned as-is (raw JSON).
// Otherwise it attempts zlib decompression.
func decompressAnalysisData(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}
	// Raw JSON always starts with '{'; zlib starts with 0x78.
	if data[0] == '{' {
		return data, nil
	}
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		// Not zlib — return as-is (might be legacy text).
		return data, nil
	}
	defer r.Close()
	return io.ReadAll(r)
}

// encodeAnalysisForStorage marshals a PositionAnalysis to JSON and compresses it.
func encodeAnalysisForStorage(a *PositionAnalysis) ([]byte, error) {
	jsonData, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	return compressAnalysisData(jsonData)
}

// decodeAnalysisFromStorage decompresses (if needed) and unmarshals analysis data.
func decodeAnalysisFromStorage(data []byte) (PositionAnalysis, error) {
	var a PositionAnalysis
	jsonData, err := decompressAnalysisData(data)
	if err != nil {
		return a, err
	}
	err = json.Unmarshal(jsonData, &a)
	return a, err
}

// recompressAnalysisData ensures data is in compressed format.
// If it's already compressed, returns as-is. If raw JSON, compresses it.
func recompressAnalysisData(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}
	if data[0] != '{' {
		// Already compressed (or at least not raw JSON)
		return data, nil
	}
	return compressAnalysisData(data)
}

// analysisColumns holds the derived scalar columns computed from a PositionAnalysis.
// These will be stored as indexed columns in the v2 schema (phase 02).
// This helper is intentionally dead code until phase 02 adds the columns.
//
// Win/gammon/backgammon rates follow the on-roll convention: "player1" is always
// the player on roll (PlayerOnRoll==0 after normalization), "player2" is the opponent.
type analysisColumns struct {
	BestCubeAction      string
	CubeError           int64 // equity loss × 1000 (millipoints); 0 if no played action
	BestMoveEquityError int64 // equity loss × 1000 (millipoints); 0 if no played move
	// Win/gammon/backgammon rates × 100 (hundredths of percent, on-roll perspective)
	Player1WinRate        int64
	Player1GammonRate     int64
	Player1BackgammonRate int64
	Player2WinRate        int64
	Player2GammonRate     int64
	Player2BackgammonRate int64
}

// populateAnalysisColumns computes scalar analysis columns from a PositionAnalysis.
// playedCubeAction and playedMove are the actions taken in this position (may be empty).
// Rates are stored as integer × 100 (hundredths of percent) and equities as × 1000 (millipoints).
func populateAnalysisColumns(a *PositionAnalysis, playedMove, playedCubeAction string) analysisColumns {
	var c analysisColumns
	if a == nil {
		return c
	}

	if dca := a.DoublingCubeAnalysis; dca != nil {
		c.BestCubeAction = dca.BestCubeAction

		// Win/gammon/backgammon rates — player1 = player on roll, scaled × 100.
		c.Player1WinRate = int64(math.Round(dca.PlayerWinChances * 100))
		c.Player1GammonRate = int64(math.Round(dca.PlayerGammonChances * 100))
		c.Player1BackgammonRate = int64(math.Round(dca.PlayerBackgammonChances * 100))
		c.Player2WinRate = int64(math.Round(dca.OpponentWinChances * 100))
		c.Player2GammonRate = int64(math.Round(dca.OpponentGammonChances * 100))
		c.Player2BackgammonRate = int64(math.Round(dca.OpponentBackgammonChances * 100))

		// Cube error: equity loss of the played cube action vs the best action, scaled × 1000.
		// Stored as absolute value (>= 0) so the sign convention matches best_move_equity_error.
		if playedCubeAction != "" {
			var raw float64
			switch playedCubeAction {
			case "NoDouble", "No Double":
				raw = dca.CubefulNoDoubleError
			case "Double", "Double/Take":
				raw = dca.CubefulDoubleTakeError
			case "Double/Pass":
				raw = dca.CubefulDoublePassError
			}
			c.CubeError = int64(math.Round(math.Abs(raw) * 1000))
		}
	} else if ca := a.CheckerAnalysis; ca != nil && len(ca.Moves) > 0 {
		// Fall back to checker analysis best move for win/gammon/backgammon rates, scaled × 100.
		best := ca.Moves[0]
		c.Player1WinRate = int64(math.Round(best.PlayerWinChance * 100))
		c.Player1GammonRate = int64(math.Round(best.PlayerGammonChance * 100))
		c.Player1BackgammonRate = int64(math.Round(best.PlayerBackgammonChance * 100))
		c.Player2WinRate = int64(math.Round(best.OpponentWinChance * 100))
		c.Player2GammonRate = int64(math.Round(best.OpponentGammonChance * 100))
		c.Player2BackgammonRate = int64(math.Round(best.OpponentBackgammonChance * 100))
	}

	// Best-move equity error: the equity error of the played checker move vs Moves[0], scaled × 1000.
	if playedMove != "" && a.CheckerAnalysis != nil {
		for _, m := range a.CheckerAnalysis.Moves {
			if m.Move == playedMove && m.EquityError != nil {
				c.BestMoveEquityError = int64(math.Round(*m.EquityError * 1000))
				break
			}
		}
	}

	return c
}

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
		_, err = d.db.Exec(`UPDATE analysis SET
			data=?, best_cube_action=?, cube_error=?,
			best_move_equity_error=?,
			player1_win_rate=?, player1_gammon_rate=?, player1_backgammon_rate=?,
			player2_win_rate=?, player2_gammon_rate=?, player2_backgammon_rate=?
			WHERE id=?`,
			analysisData,
			aCols.BestCubeAction, aCols.CubeError,
			aCols.BestMoveEquityError,
			aCols.Player1WinRate, aCols.Player1GammonRate, aCols.Player1BackgammonRate,
			aCols.Player2WinRate, aCols.Player2GammonRate, aCols.Player2BackgammonRate,
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
		_, err = d.db.Exec(`INSERT INTO analysis (
			position_id, data,
			best_cube_action, cube_error, best_move_equity_error,
			player1_win_rate, player1_gammon_rate, player1_backgammon_rate,
			player2_win_rate, player2_gammon_rate, player2_backgammon_rate
		) VALUES (?,?, ?,?,?, ?,?,?, ?,?,?)`,
			positionID, analysisData,
			aCols.BestCubeAction, aCols.CubeError, aCols.BestMoveEquityError,
			aCols.Player1WinRate, aCols.Player1GammonRate, aCols.Player1BackgammonRate,
			aCols.Player2WinRate, aCols.Player2GammonRate, aCols.Player2BackgammonRate)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *Database) LoadAnalysis(positionID int64) (*PositionAnalysis, error) {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	var analysisData []byte
	err := d.db.QueryRow(`SELECT data from analysis WHERE position_id = ?`, positionID).Scan(&analysisData)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}

	analysis, err := decodeAnalysisFromStorage(analysisData)
	if err != nil {
		return nil, err
	}

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

// roundToMillipoint rounds an equity value (in equity points) to the nearest millipoint (0.001).
func roundToMillipoint(v float64) float64 {
	return math.Round(v*1000) / 1000
}

// roundToHundredthPercent rounds a rate (in percent) to the nearest 0.01%.
func roundToHundredthPercent(v float64) float64 {
	return math.Round(v*100) / 100
}

// roundAnalysisForStorage rounds all float fields in a PositionAnalysis for compact JSON storage.
// Rates (win/gammon/backgammon chances) → 2 decimal places.
// Equities and errors → 3 decimal places (millipoint precision).
func roundAnalysisForStorage(a *PositionAnalysis) {
	if a == nil {
		return
	}
	roundDCA := func(dca *DoublingCubeAnalysis) {
		dca.PlayerWinChances = roundToHundredthPercent(dca.PlayerWinChances)
		dca.PlayerGammonChances = roundToHundredthPercent(dca.PlayerGammonChances)
		dca.PlayerBackgammonChances = roundToHundredthPercent(dca.PlayerBackgammonChances)
		dca.OpponentWinChances = roundToHundredthPercent(dca.OpponentWinChances)
		dca.OpponentGammonChances = roundToHundredthPercent(dca.OpponentGammonChances)
		dca.OpponentBackgammonChances = roundToHundredthPercent(dca.OpponentBackgammonChances)
		dca.CubelessNoDoubleEquity = roundToMillipoint(dca.CubelessNoDoubleEquity)
		dca.CubelessDoubleEquity = roundToMillipoint(dca.CubelessDoubleEquity)
		dca.CubefulNoDoubleEquity = roundToMillipoint(dca.CubefulNoDoubleEquity)
		dca.CubefulNoDoubleError = roundToMillipoint(dca.CubefulNoDoubleError)
		dca.CubefulDoubleTakeEquity = roundToMillipoint(dca.CubefulDoubleTakeEquity)
		dca.CubefulDoubleTakeError = roundToMillipoint(dca.CubefulDoubleTakeError)
		dca.CubefulDoublePassEquity = roundToMillipoint(dca.CubefulDoublePassEquity)
		dca.CubefulDoublePassError = roundToMillipoint(dca.CubefulDoublePassError)
		dca.WrongPassPercentage = roundToHundredthPercent(dca.WrongPassPercentage)
		dca.WrongTakePercentage = roundToHundredthPercent(dca.WrongTakePercentage)
	}
	if a.DoublingCubeAnalysis != nil {
		roundDCA(a.DoublingCubeAnalysis)
	}
	for i := range a.AllCubeAnalyses {
		roundDCA(&a.AllCubeAnalyses[i])
	}
	if ca := a.CheckerAnalysis; ca != nil {
		for i := range ca.Moves {
			m := &ca.Moves[i]
			m.Equity = roundToMillipoint(m.Equity)
			if m.EquityError != nil {
				rounded := roundToMillipoint(*m.EquityError)
				m.EquityError = &rounded
			}
			m.PlayerWinChance = roundToHundredthPercent(m.PlayerWinChance)
			m.PlayerGammonChance = roundToHundredthPercent(m.PlayerGammonChance)
			m.PlayerBackgammonChance = roundToHundredthPercent(m.PlayerBackgammonChance)
			m.OpponentWinChance = roundToHundredthPercent(m.OpponentWinChance)
			m.OpponentGammonChance = roundToHundredthPercent(m.OpponentGammonChance)
			m.OpponentBackgammonChance = roundToHundredthPercent(m.OpponentBackgammonChance)
		}
	}
}
