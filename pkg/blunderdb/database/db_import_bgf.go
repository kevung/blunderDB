package database

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/kevung/bgfparser"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/xgparser/xgparser"
)

// ============================================================================
// BGBlitz BGF import functions
// ============================================================================

// ImportBGFMatch imports a match from a BGBlitz BGF file, delegating to the
// shared ingest pipeline (ingest.MapBGF -> ingest.WriteMatch) — the same path
// the headless server uses.
func (d *Database) ImportBGFMatch(filePath string) (int64, error) {
	ctx, done := d.beginCancellableImport()
	defer done()

	d.mu.Lock()
	defer d.mu.Unlock()

	graph, err := ingest.MapBGF(filePath)
	if err != nil {
		return 0, err
	}
	matchID, err := d.writeImportedMatch(ctx, graph)
	if err != nil {
		return 0, err
	}
	slog.Info("imported BGF match", "matchID", matchID, "file", filePath)
	return matchID, nil
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
