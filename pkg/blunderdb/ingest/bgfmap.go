package ingest

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kevung/bgfparser"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// Pure BGBlitz (.bgf) → domain mappers, lifted from
// pkg/blunderdb/database/db_import_bgf.go with the *Database receiver dropped.
// BGF data arrives as untyped map[string]interface{} trees, so the bgfGet*
// accessors below mirror the legacy helpers verbatim. The legacy path is left
// untouched; the bgf parity test gates these against it.

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

// bgfInitBoardFromGame extracts the initial board position from a BGF game.
func bgfInitBoardFromGame(gameData map[string]interface{}) [28]int {
	var board [28]int
	std := [28]int{2, 0, 0, 0, 0, -5, 0, -3, 0, 0, 0, 5, -5, 0, 0, 0, 3, 0, 5, 0, 0, 0, 0, -2, 0, 0, 0, 0}

	initial, ok := gameData["initial"].(map[string]interface{})
	if !ok {
		return std
	}
	points, ok := initial["points"].([]interface{})
	if !ok || len(points) < 28 {
		return std
	}
	for i := 0; i < 28 && i < len(points); i++ {
		board[i] = bgfToInt(points[i])
	}
	return board
}

// bgfApplyCheckerMove updates the board state after a BGF checker move. BGF
// from/to are 1-based from the active player's perspective (25=bar, 0=off);
// board state is 0-based from Green's perspective.
func bgfApplyCheckerMove(boardState *[28]int, moveData map[string]interface{}, player int) {
	fromArr := bgfGetIntArray(moveData, "from")
	toArr := bgfGetIntArray(moveData, "to")

	for i := 0; i < 4; i++ {
		from := fromArr[i]
		to := toArr[i]
		if from == -1 {
			break
		}

		if player == -1 {
			var fromIdx int
			if from == 25 {
				fromIdx = 24
			} else {
				fromIdx = 24 - from
			}
			boardState[fromIdx]--
			if to == 0 {
				boardState[26]++
			} else {
				toIdx := 24 - to
				if boardState[toIdx] < 0 {
					boardState[25] += boardState[toIdx]
					boardState[toIdx] = 0
				}
				boardState[toIdx]++
			}
		} else {
			var fromIdx int
			if from == 25 {
				fromIdx = 25
			} else {
				fromIdx = from - 1
			}
			boardState[fromIdx]++
			if to == 0 {
				boardState[27]--
			} else {
				toIdx := to - 1
				if boardState[toIdx] > 0 {
					boardState[24] += boardState[toIdx]
					boardState[toIdx] = 0
				}
				boardState[toIdx]--
			}
		}
	}
}

// createPositionFromBGF builds a domain.Position from BGF board state.
func createPositionFromBGF(boardState [28]int, gameData map[string]interface{}, matchLen, cubeValue, cubeOwner int, isCrawford bool) *domain.Position {
	scoreGreen := bgfGetInt(gameData, "scoreGreen")
	scoreRed := bgfGetInt(gameData, "scoreRed")

	awayScore1 := matchLen - scoreGreen
	awayScore2 := matchLen - scoreRed
	if matchLen == 0 {
		awayScore1 = -1
		awayScore2 = -1
	}

	cubeExponent := 0
	if cubeValue > 0 {
		for v := cubeValue; v > 1; v >>= 1 {
			cubeExponent++
		}
	}

	pos := &domain.Position{
		PlayerOnRoll: 0,
		DecisionType: domain.CheckerAction,
		Score:        [2]int{awayScore1, awayScore2},
		Cube: domain.Cube{
			Value: cubeExponent,
			Owner: cubeOwner,
		},
		Dice: [2]int{0, 0},
	}

	for i := 0; i < 26; i++ {
		pos.Board.Points[i] = domain.Point{Checkers: 0, Color: -1}
	}

	// BGF index i → blunderDB point (24-i); positive=Green (color 0), negative=Red (color 1).
	for i := 0; i < 24; i++ {
		count := boardState[i]
		blunderDBPoint := 24 - i
		if count > 0 {
			pos.Board.Points[blunderDBPoint] = domain.Point{Checkers: count, Color: 0}
		} else if count < 0 {
			pos.Board.Points[blunderDBPoint] = domain.Point{Checkers: -count, Color: 1}
		}
	}

	// Green's bar (index 24) → blunderDB 25; Red's bar (index 25) → blunderDB 0.
	if boardState[24] > 0 {
		pos.Board.Points[25] = domain.Point{Checkers: boardState[24], Color: 0}
	}
	if boardState[25] < 0 {
		pos.Board.Points[0] = domain.Point{Checkers: -boardState[25], Color: 1}
	} else if boardState[25] > 0 {
		pos.Board.Points[0] = domain.Point{Checkers: boardState[25], Color: 1}
	}

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

	_ = isCrawford // reserved (matches legacy signature; Jacoby/Beaver TODO)
	return pos
}

// bgfFormatSubmoves renders a from/to array (1-based player perspective, 25=bar,
// 0=off) as standard notation, used for both played moves and analysis moves.
func bgfFormatSubmoves(fromArr, toArr [4]int) string {
	if fromArr[0] == -1 {
		return ""
	}
	type submove struct{ from, to int }
	moves := make([]submove, 0, 4)
	for i := 0; i < 4; i++ {
		if fromArr[i] == -1 {
			break
		}
		moves = append(moves, submove{fromArr[i], toArr[i]})
	}
	if len(moves) == 0 {
		return ""
	}
	sort.Slice(moves, func(i, j int) bool { return moves[i].from > moves[j].from })

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

// bgfConvertMoveToString converts a played BGF move to notation.
func bgfConvertMoveToString(moveData map[string]interface{}) string {
	return bgfFormatSubmoves(bgfGetIntArray(moveData, "from"), bgfGetIntArray(moveData, "to"))
}

// bgfConvertAnalysisMoveToString converts an analysis-entry move to notation.
func bgfConvertAnalysisMoveToString(moveInfo map[string]interface{}) string {
	return bgfFormatSubmoves(bgfGetIntArray(moveInfo, "from"), bgfGetIntArray(moveInfo, "to"))
}

// translateBGFAnalysisDepth converts a BGF ply level to a label.
func translateBGFAnalysisDepth(ply int) string {
	if ply > 0 {
		return fmt.Sprintf("%d-ply", ply)
	}
	return "0-ply"
}

// bgfEquityValue returns the EMG equity when present, else the raw equity.
func bgfEquityValue(eq map[string]interface{}) float64 {
	if bgfGetBool(eq, "hasEMG") {
		return bgfGetFloat(eq, "emg")
	}
	return bgfGetFloat(eq, "equity")
}

// buildBGFCheckerAnalysis converts a BGF moveAnalysis array into a checker
// PositionAnalysis fragment.
func buildBGFCheckerAnalysis(moveAnalysis []interface{}, playedMoveStr string) *domain.PositionAnalysis {
	if len(moveAnalysis) == 0 {
		return nil
	}

	posAnalysis := &domain.PositionAnalysis{
		AnalysisType:          "CheckerMove",
		AnalysisEngineVersion: "BGBlitz",
		CreationDate:          time.Now(),
		LastModifiedDate:      time.Now(),
	}

	checkerMoves := make([]domain.CheckerMove, 0, len(moveAnalysis))
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
		equity := bgfEquityValue(eq)
		if i == 0 {
			bestEquity = equity
		}

		moveStr := ""
		if moveInfo := bgfGetMap(maData, "move"); moveInfo != nil {
			moveStr = bgfConvertAnalysisMoveToString(moveInfo)
		}

		var equityError *float64
		if i > 0 {
			diff := bestEquity - equity
			equityError = &diff
		}

		checkerMoves = append(checkerMoves, domain.CheckerMove{
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
		})
	}

	posAnalysis.CheckerAnalysis = &domain.CheckerAnalysis{Moves: checkerMoves}
	if playedMoveStr != "" {
		posAnalysis.PlayedMoves = []string{playedMoveStr}
	}
	return posAnalysis
}

// bgfCubeParams maps a BGF equity + cubeDecision pair to cubeAnalysisParams.
func bgfCubeParams(equity, cubeDecision map[string]interface{}) cubeAnalysisParams {
	cubeless := bgfGetFloat(cubeDecision, "eqCubeLess")
	return cubeAnalysisParams{
		Depth:                     "2-ply",
		Engine:                    "BGBlitz",
		PlayerWinChances:          bgfGetFloat(equity, "myWins") * 100.0,
		PlayerGammonChances:       bgfGetFloat(equity, "myGammon") * 100.0,
		PlayerBackgammonChances:   bgfGetFloat(equity, "myBackGammon") * 100.0,
		OpponentWinChances:        bgfGetFloat(equity, "oppWins") * 100.0,
		OpponentGammonChances:     bgfGetFloat(equity, "oppGammon") * 100.0,
		OpponentBackgammonChances: bgfGetFloat(equity, "oppBackGammon") * 100.0,
		CubelessNoDoubleEquity:    cubeless,
		CubelessDoubleEquity:      cubeless,
		CubefulNoDoubleEquity:     bgfGetFloat(cubeDecision, "eqNoDouble"),
		CubefulDoubleTakeEquity:   bgfGetFloat(cubeDecision, "eqDoubleTake"),
		CubefulDoublePassEquity:   bgfGetFloat(cubeDecision, "eqDoublePass"),
	}
}

// buildBGFCubeAnalysis builds a DoublingCube fragment for a cube decision,
// overriding the best action from BGF's stateOnMove/stateOther when available.
func buildBGFCubeAnalysis(equity, cubeDecision map[string]interface{}, playedCubeAction string) *domain.PositionAnalysis {
	cube := buildDoublingCubeAnalysis(bgfCubeParams(equity, cubeDecision))

	stateOnMove := bgfGetString(cubeDecision, "stateOnMove")
	stateOther := bgfGetString(cubeDecision, "stateOther")
	if stateOnMove == "DOUBLE" || stateOnMove == "REDOUBLE" {
		if stateOther == "ACCEPT" {
			cube.BestCubeAction = "Double, Take"
		} else if stateOther == "REJECT" {
			cube.BestCubeAction = "Double, Pass"
		}
	} else if stateOnMove == "TO_GOOD" || stateOnMove == "NO_DOUBLE" {
		cube.BestCubeAction = "No Double"
	}

	posAnalysis := &domain.PositionAnalysis{
		AnalysisType:          "DoublingCube",
		AnalysisEngineVersion: "BGBlitz",
		DoublingCubeAnalysis:  &cube,
		CreationDate:          time.Now(),
		LastModifiedDate:      time.Now(),
	}
	if playedCubeAction != "" {
		posAnalysis.PlayedCubeActions = []string{playedCubeAction}
	}
	return posAnalysis
}

// buildBGFCubeForChecker builds a cube fragment to attach to a checker position
// (AnalysisType "CheckerMove"), with no best-action override or played action.
func buildBGFCubeForChecker(equity, cubeDecision map[string]interface{}) *domain.PositionAnalysis {
	cube := buildDoublingCubeAnalysis(bgfCubeParams(equity, cubeDecision))
	return &domain.PositionAnalysis{
		AnalysisType:          "CheckerMove",
		AnalysisEngineVersion: "BGBlitz",
		DoublingCubeAnalysis:  &cube,
		CreationDate:          time.Now(),
		LastModifiedDate:      time.Now(),
	}
}

// --- BGBlitz single-position text (.txt) ---

// convertBGFTextPosition converts a bgfparser.Position (parsed from a BGBlitz
// text export) into a domain.Position. Lifted from
// database.convertBGFTextPosition.
func convertBGFTextPosition(bgfPos *bgfparser.Position) *domain.Position {
	pos := &domain.Position{
		PlayerOnRoll: 0,
		DecisionType: domain.CheckerAction,
	}

	for i := 0; i < 26; i++ {
		pos.Board.Points[i] = domain.Point{Checkers: 0, Color: -1}
	}

	// Points 1-24: positive=X (Green, color 0), negative=O (Red, color 1).
	for i := 1; i <= 24; i++ {
		count := bgfPos.Board[i]
		if count > 0 {
			pos.Board.Points[i] = domain.Point{Checkers: count, Color: 0}
		} else if count < 0 {
			pos.Board.Points[i] = domain.Point{Checkers: -count, Color: 1}
		}
	}

	if bgfPos.OnBar != nil {
		if xBar, ok := bgfPos.OnBar["X"]; ok && xBar > 0 {
			pos.Board.Points[25] = domain.Point{Checkers: xBar, Color: 0}
		}
		if oBar, ok := bgfPos.OnBar["O"]; ok && oBar > 0 {
			pos.Board.Points[0] = domain.Point{Checkers: oBar, Color: 1}
		}
	}

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

	if bgfPos.OnRoll == "O" {
		pos.PlayerOnRoll = 1
	} else {
		pos.PlayerOnRoll = 0
	}

	pos.Dice = [2]int{bgfPos.Dice[0], bgfPos.Dice[1]}

	cubeExponent := 0
	if bgfPos.CubeValue > 0 {
		for v := bgfPos.CubeValue; v > 1; v >>= 1 {
			cubeExponent++
		}
	}
	pos.Cube.Value = cubeExponent
	switch bgfPos.CubeOwner {
	case "X":
		pos.Cube.Owner = 0
	case "O":
		pos.Cube.Owner = 1
	default:
		pos.Cube.Owner = -1
	}

	if bgfPos.MatchLength > 0 {
		pos.Score = [2]int{bgfPos.MatchLength - bgfPos.ScoreX, bgfPos.MatchLength - bgfPos.ScoreO}
	} else {
		pos.Score = [2]int{-1, -1}
	}

	if len(bgfPos.CubeDecisions) > 0 && len(bgfPos.Evaluations) == 0 {
		pos.DecisionType = domain.CubeAction
		pos.Dice = [2]int{0, 0}
	}

	return pos
}

// classifyBGFCubeAction classifies a (possibly multilingual) cube action string
// into "nodbl", "take", "pass" or "unknown". Lifted from database.
func classifyBGFCubeAction(action string) string {
	action = strings.ToLower(strings.TrimSpace(action))

	noDoublePatterns := []string{
		"no double", "no redouble",
		"pas de double", "pas de redouble",
		"kein doppel", "kein redoppel",
		"ダブルせず",
	}
	for _, p := range noDoublePatterns {
		if strings.Contains(action, p) {
			return "nodbl"
		}
	}

	takePatterns := []string{
		"take", "accept",
		"prendre", "accepter",
		"annehmen",
		"受ける",
	}
	for _, p := range takePatterns {
		if strings.Contains(action, p) {
			return "take"
		}
	}

	passPatterns := []string{
		"pass", "reject", "decline",
		"refuser",
		"ablehnen",
		"降りる",
	}
	for _, p := range passPatterns {
		if strings.Contains(action, p) {
			return "pass"
		}
	}

	if !strings.Contains(action, "/") {
		return "nodbl"
	}
	return "unknown"
}

// buildBGFTextCheckerAnalysis builds a checker PositionAnalysis fragment from a
// BGBlitz text position's evaluations.
func buildBGFTextCheckerAnalysis(bgfPos *bgfparser.Position) *domain.PositionAnalysis {
	if len(bgfPos.Evaluations) == 0 {
		return nil
	}
	posAnalysis := &domain.PositionAnalysis{
		XGID:                  bgfPos.XGID,
		Player1:               bgfPos.PlayerX,
		Player2:               bgfPos.PlayerO,
		AnalysisType:          "CheckerMove",
		AnalysisEngineVersion: "BGBlitz",
		CreationDate:          time.Now(),
		LastModifiedDate:      time.Now(),
	}
	moves := make([]domain.CheckerMove, 0, len(bgfPos.Evaluations))
	for i, eval := range bgfPos.Evaluations {
		var equityError *float64
		if i > 0 {
			diff := bgfPos.Evaluations[0].Equity - eval.Equity
			equityError = &diff
		}
		moves = append(moves, domain.CheckerMove{
			Index:                    i,
			AnalysisDepth:            "2-ply",
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
		})
	}
	posAnalysis.CheckerAnalysis = &domain.CheckerAnalysis{Moves: moves}
	return posAnalysis
}

// buildBGFTextCubeAnalysis builds a DoublingCube PositionAnalysis fragment from
// a BGBlitz text position's cube decisions.
func buildBGFTextCubeAnalysis(bgfPos *bgfparser.Position) *domain.PositionAnalysis {
	if len(bgfPos.CubeDecisions) == 0 {
		return nil
	}

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
	cubefulDoublePass := 1.0
	if noDouble != nil {
		cubefulNoDouble = noDouble.EMG
	}
	if doubleTake != nil {
		cubefulDoubleTake = doubleTake.EMG
	}
	if doublePass != nil {
		cubefulDoublePass = doublePass.EMG
	}

	cube := buildDoublingCubeAnalysis(cubeAnalysisParams{
		Depth:                   "2-ply",
		Engine:                  "BGBlitz",
		CubelessNoDoubleEquity:  bgfPos.CubelessEquity,
		CubelessDoubleEquity:    bgfPos.CubelessEquity,
		CubefulNoDoubleEquity:   cubefulNoDouble,
		CubefulDoubleTakeEquity: cubefulDoubleTake,
		CubefulDoublePassEquity: cubefulDoublePass,
	})

	return &domain.PositionAnalysis{
		XGID:                  bgfPos.XGID,
		Player1:               bgfPos.PlayerX,
		Player2:               bgfPos.PlayerO,
		AnalysisType:          "DoublingCube",
		AnalysisEngineVersion: "BGBlitz",
		DoublingCubeAnalysis:  &cube,
		CreationDate:          time.Now(),
		LastModifiedDate:      time.Now(),
	}
}
