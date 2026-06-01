package ingest

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/xgparser/xgparser"
)

// This file holds the pure XG → domain mapping helpers. They are lifted
// verbatim (modulo dropping the *Database receiver and using domain types
// directly) from pkg/blunderdb/database/db_import_xg.go so the daemon can build
// a MatchGraph without the SQLite-only Database wrapper. The legacy code path
// is intentionally left untouched; the xg parity test gates this against it.

// parseMatchDate tries the date formats XG emits and returns the parsed time,
// falling back to time.Now() when the string is empty or unrecognised.
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

// convertXGPlayerToBlunderDB converts XG player encoding to blunderDB encoding.
// XG: 1 = Player 1, -1 = Player 2.  blunderDB: 0 = Player 1, 1 = Player 2.
func convertXGPlayerToBlunderDB(xgPlayer int32) int {
	if xgPlayer == 1 {
		return 0 // Player 1
	}
	return 1 // Player 2
}

// xgAbs returns the absolute value of an int8 as an int.
func xgAbs(x int8) int {
	if x < 0 {
		return int(-x)
	}
	return int(x)
}

// createPositionFromXG converts an xgparser.Position to a domain.Position.
// activePlayer indicates which XG player (-1 or 1) is on roll in this position.
func createPositionFromXG(xgPos xgparser.Position, game *xgparser.Game, matchLength int32, activePlayer int32) (*domain.Position, error) {
	activePlayerBlunderDB := convertXGPlayerToBlunderDB(activePlayer)
	opponentPlayerBlunderDB := 1 - activePlayerBlunderDB

	awayScore1 := int(matchLength) - int(game.InitialScore[0])
	awayScore2 := int(matchLength) - int(game.InitialScore[1])
	if matchLength == 0 {
		awayScore1 = -1
		awayScore2 = -1
	}

	// Map XG cube position (relative to active player) to blunderDB absolute.
	cubeOwner := -1
	if xgPos.CubePos == 1 {
		cubeOwner = activePlayerBlunderDB
	} else if xgPos.CubePos == -1 {
		cubeOwner = opponentPlayerBlunderDB
	}

	// Convert cube value from XG (1,2,4,8…) to blunderDB exponent (0,1,2,3…).
	cubeValue := 0
	if xgPos.Cube > 0 {
		for v := int(xgPos.Cube); v > 1; v >>= 1 {
			cubeValue++
		}
	}

	pos := &domain.Position{
		PlayerOnRoll: 0,
		DecisionType: domain.CheckerAction,
		Score:        [2]int{awayScore1, awayScore2},
		Cube: domain.Cube{
			Value: cubeValue,
			Owner: cubeOwner,
		},
		Dice: [2]int{0, 0},
	}

	// XG stores checkers from the active player's perspective; mirror to the
	// fixed blunderDB orientation (Player 1 bottom/black, Player 2 top/white).
	for i := 0; i < 26; i++ {
		checkerCount := xgPos.Checkers[i]

		targetIndex := i
		if activePlayerBlunderDB == 1 {
			if i >= 1 && i <= 24 {
				targetIndex = 25 - i
			} else if i == 0 {
				targetIndex = 25
			} else if i == 25 {
				targetIndex = 0
			}
		}

		if checkerCount == 0 {
			pos.Board.Points[targetIndex] = domain.Point{Checkers: 0, Color: -1}
			continue
		}

		var ownerColor int
		if checkerCount > 0 {
			ownerColor = activePlayerBlunderDB
		} else {
			ownerColor = opponentPlayerBlunderDB
		}

		pos.Board.Points[targetIndex] = domain.Point{
			Checkers: xgAbs(checkerCount),
			Color:    ownerColor,
		}
	}

	// Calculate bearoff (checkers borne off) from on-board totals.
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

// translateAnalysisDepth converts XG analysis depth codes to human-readable strings.
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

// convertCubeAction converts a cube action code to its string form.
func convertCubeAction(action int32) string {
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

// convertRawCubeAction converts raw Double/Take values to an action string.
func convertRawCubeAction(double, take int32) string {
	if double == 0 {
		return "No Double"
	} else if double == 1 {
		if take == 1 {
			return "Double/Take"
		}
		return "Double/Pass"
	}
	return fmt.Sprintf("Unknown(D=%d,T=%d)", double, take)
}

// inferMoveMultipliers expands a compact XG move array to include all
// repetitions, using the position difference to infer counts.
func inferMoveMultipliers(partialMove [8]int32, initialPos, finalPos *xgparser.Position) [8]int32 {
	if initialPos == nil || finalPos == nil {
		return partialMove
	}

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
		if from >= 1 && from <= 6 && to <= 0 && to != -2 {
			to = -2
		}
		moveCount[moveSpec{from, to}]++
		totalInputMoves++
	}

	if totalInputMoves == 0 {
		return partialMove
	}

	if totalInputMoves > 1 {
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
		if !allSame {
			return partialMove
		}
	}

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

	if len(uniqueMoves) == 1 && moveCount[uniqueMoves[0]] > 1 {
		return partialMove
	}

	netChange := make(map[int32]int32)
	for _, ms := range uniqueMoves {
		if ms.from == 25 {
			netChange[25] = int32(initialPos.Checkers[25]) - int32(finalPos.Checkers[25])
		} else if ms.from >= 1 && ms.from <= 24 {
			netChange[ms.from] = int32(initialPos.Checkers[ms.from]) - int32(finalPos.Checkers[ms.from])
		}
		if ms.to >= 1 && ms.to <= 24 {
			netChange[ms.to] = int32(initialPos.Checkers[ms.to]) - int32(finalPos.Checkers[ms.to])
		}
	}

	arriving := make(map[int32]int32)

	var expandedMove [8]int32
	for i := range expandedMove {
		expandedMove[i] = -1
	}
	moveIndex := 0

	for _, ms := range uniqueMoves {
		var count int32 = 1

		if ms.to == -2 {
			count = netChange[ms.from] + arriving[ms.from]
		} else if ms.to >= 1 && ms.to <= 24 {
			srcLoss := netChange[ms.from]
			srcReceive := arriving[ms.from]
			count = srcLoss - srcReceive
			if count <= 0 {
				count = 1
			}
			arriving[ms.to] += count
		}

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

// xgMoveWithHit represents a single move in XG format with hit information.
type xgMoveWithHit struct {
	from  int32
	to    int32
	isHit bool
}

// convertXGMoveToString converts an XG move array to readable notation.
func convertXGMoveToString(playedMove [8]int32) string {
	var fromPts []int32
	var toPts []int32
	for i := 0; i < 8; i += 2 {
		from := playedMove[i]
		to := playedMove[i+1]
		if from == -1 {
			break
		}
		if from >= 1 && from <= 6 && to <= 0 && to != -2 {
			to = -2
		}
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

	fromPts, toPts = mergeSlides(fromPts, toPts)

	type moveItem struct {
		from int32
		to   int32
	}
	items := make([]moveItem, len(fromPts))
	for i := range fromPts {
		items[i] = moveItem{from: fromPts[i], to: toPts[i]}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].from > items[j].from
	})

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
			moves = append(moves, fmt.Sprintf("%s/%s(%d)", formatPoint(item.from), formatPoint(item.to), count))
		} else {
			moves = append(moves, fmt.Sprintf("%s/%s", formatPoint(item.from), formatPoint(item.to)))
		}
		i += count
	}

	return strings.Join(moves, " ")
}

// convertXGMoveToStringWithHits converts an XG move array to readable notation
// with hit markers (*), using the initial position to detect blots.
func convertXGMoveToStringWithHits(playedMove [8]int32, initialPos *xgparser.Position) string {
	if initialPos == nil {
		return convertXGMoveToString(playedMove)
	}

	positionCopy := make([]int8, 26)
	copy(positionCopy, initialPos.Checkers[:])

	var items []xgMoveWithHit
	for i := 0; i < 8; i += 2 {
		from := playedMove[i]
		to := playedMove[i+1]
		if from == -1 {
			break
		}
		if from >= 1 && from <= 6 && to <= 0 && to != -2 {
			to = -2
		}
		if (from != 25 && from != -2 && (from < 1 || from > 24)) ||
			(to != 25 && to != -2 && (to < 1 || to > 24)) {
			break
		}

		isHit := false
		if to >= 1 && to <= 24 {
			if positionCopy[to] == -1 {
				isHit = true
				positionCopy[to] = 0
			}
		}

		if from >= 1 && from <= 24 {
			if positionCopy[from] > 0 {
				positionCopy[from]--
			}
		} else if from == 25 {
			if positionCopy[25] > 0 {
				positionCopy[25]--
			}
		}

		if to >= 1 && to <= 24 {
			positionCopy[to]++
		}

		items = append(items, xgMoveWithHit{from: from, to: to, isHit: isHit})
	}

	if len(items) == 0 {
		return "Cannot Move"
	}

	items = mergeSlidesWithHits(items)

	sort.Slice(items, func(i, j int) bool {
		return items[i].from > items[j].from
	})

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

	var moves []string
	for i := 0; i < len(items); {
		item := items[i]
		count := 1
		allHits := item.isHit
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

// mergeSlidesWithHits merges consecutive moves of the same checker, preserving
// hit info (a chain stops extending at a hit so the hit is shown).
func mergeSlidesWithHits(items []xgMoveWithHit) []xgMoveWithHit {
	if len(items) <= 1 {
		return items
	}

	result := make([]xgMoveWithHit, 0, len(items))
	used := make([]bool, len(items))

	for i := 0; i < len(items); i++ {
		if used[i] {
			continue
		}

		chainFrom := items[i].from
		chainTo := items[i].to
		chainHit := items[i].isHit
		used[i] = true

		if !chainHit {
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

// mergeSlides merges consecutive moves of the same checker (e.g. 14/12 12/8 →
// 14/8), but not through points used by multiple checkers.
func mergeSlides(fromPts, toPts []int32) ([]int32, []int32) {
	if len(fromPts) <= 1 {
		return fromPts, toPts
	}

	toCount := make(map[int32]int)
	for _, t := range toPts {
		toCount[t]++
	}
	fromCount := make(map[int32]int)
	for _, f := range fromPts {
		fromCount[f]++
	}

	resultFrom := make([]int32, 0, len(fromPts))
	resultTo := make([]int32, 0, len(toPts))
	used := make([]bool, len(fromPts))

	for i := 0; i < len(fromPts); i++ {
		if used[i] {
			continue
		}

		chainFrom := fromPts[i]
		chainTo := toPts[i]
		used[i] = true

		for changed := true; changed; {
			changed = false
			for j := 0; j < len(fromPts); j++ {
				if used[j] {
					continue
				}
				if fromPts[j] == chainTo {
					if toCount[chainTo] > 1 || fromCount[chainTo] > 1 {
						continue
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

// cubeAnalysisParams holds the format-independent cube analysis values needed
// to build a domain.DoublingCubeAnalysis.
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

// computeBestCubeAction determines the best cube action and equity from the
// three cubeful equities.
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

// buildDoublingCubeAnalysis creates a DoublingCubeAnalysis from the given params.
func buildDoublingCubeAnalysis(p cubeAnalysisParams) domain.DoublingCubeAnalysis {
	bestEquity, bestAction := computeBestCubeAction(p.CubefulNoDoubleEquity, p.CubefulDoubleTakeEquity, p.CubefulDoublePassEquity)
	return domain.DoublingCubeAnalysis{
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

// buildCheckerAnalysis converts XG checker analyses into a domain.PositionAnalysis
// holding the ranked checker moves and the played move. initialPosition (the
// position the move is played from) enables hit detection and multiplier
// inference; playedMove is the actually-played move (used as the source of
// truth for analysis[0], a workaround for incomplete multi-submove arrays).
func buildCheckerAnalysis(analyses []xgparser.CheckerAnalysis, initialPosition *xgparser.Position, playedMove *[8]int32) *domain.PositionAnalysis {
	if len(analyses) == 0 {
		return nil
	}

	posAnalysis := &domain.PositionAnalysis{
		AnalysisType:          "CheckerMove",
		AnalysisEngineVersion: "XG",
		CreationDate:          time.Now(),
		LastModifiedDate:      time.Now(),
	}

	checkerMoves := make([]domain.CheckerMove, 0, len(analyses))
	for i, analysis := range analyses {
		var move [8]int32

		if i == 0 && playedMove != nil {
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

		if initialPosition != nil {
			move = inferMoveMultipliers(move, initialPosition, &analysis.Position)
		}

		var moveStr string
		if initialPosition != nil {
			moveStr = convertXGMoveToStringWithHits(move, initialPosition)
		} else {
			moveStr = convertXGMoveToString(move)
		}

		var equityError *float64
		if i > 0 {
			diff := float64(analyses[0].Equity - analysis.Equity)
			equityError = &diff
		}

		checkerMoves = append(checkerMoves, domain.CheckerMove{
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
		})
	}

	posAnalysis.CheckerAnalysis = &domain.CheckerAnalysis{Moves: checkerMoves}

	if playedMove != nil {
		playedMoveStr := convertXGMoveToStringWithHits(*playedMove, initialPosition)
		if playedMoveStr != "" {
			posAnalysis.PlayedMoves = []string{playedMoveStr}
		}
	}

	return posAnalysis
}

// buildCubeAnalysisFromText converts an XG (text) CubeAnalysis into a
// DoublingCube domain.PositionAnalysis. playedCubeAction, when non-empty, is
// recorded so cube_error can be computed.
func buildCubeAnalysisFromText(analysis *xgparser.CubeAnalysis, playedCubeAction string) *domain.PositionAnalysis {
	if analysis == nil {
		return nil
	}

	posAnalysis := &domain.PositionAnalysis{
		AnalysisType:          "DoublingCube",
		AnalysisEngineVersion: "XG",
		CreationDate:          time.Now(),
		LastModifiedDate:      time.Now(),
	}
	if playedCubeAction != "" {
		posAnalysis.PlayedCubeActions = []string{playedCubeAction}
	}

	cube := buildDoublingCubeAnalysis(cubeAnalysisParams{
		Depth:                     translateAnalysisDepth(int(analysis.AnalysisDepth)),
		Engine:                    "XG",
		PlayerWinChances:          float64(analysis.Player1WinRate) * 100.0,
		PlayerGammonChances:       float64(analysis.Player1GammonRate) * 100.0,
		PlayerBackgammonChances:   float64(analysis.Player1BgRate) * 100.0,
		OpponentWinChances:        float64(1.0-analysis.Player1WinRate) * 100.0,
		OpponentGammonChances:     float64(analysis.Player2GammonRate) * 100.0,
		OpponentBackgammonChances: float64(analysis.Player2BgRate) * 100.0,
		CubelessNoDoubleEquity:    float64(analysis.CubelessNoDouble),
		CubelessDoubleEquity:      float64(analysis.CubelessDouble),
		CubefulNoDoubleEquity:     float64(analysis.CubefulNoDouble),
		CubefulDoubleTakeEquity:   float64(analysis.CubefulDoubleTake),
		CubefulDoublePassEquity:   float64(analysis.CubefulDoublePass),
		WrongPassPercentage:       float64(analysis.WrongPassTakePercent) * 100.0,
	})
	posAnalysis.DoublingCubeAnalysis = &cube
	return posAnalysis
}

// buildRawCubeForChecker builds a DoublingCube domain.PositionAnalysis from a
// raw EngineStructDoubleAction, to attach the cube decision to a checker
// position (the "press d on a checker move" case). It sets no played action.
func buildRawCubeForChecker(doubled *xgparser.EngineStructDoubleAction) *domain.PositionAnalysis {
	if doubled == nil {
		return nil
	}
	posAnalysis := &domain.PositionAnalysis{
		AnalysisType:          "CheckerMove",
		AnalysisEngineVersion: "XG",
		CreationDate:          time.Now(),
		LastModifiedDate:      time.Now(),
	}
	cube := buildDoublingCubeAnalysis(rawCubeParams(doubled))
	posAnalysis.DoublingCubeAnalysis = &cube
	return posAnalysis
}

// buildRawCubeForNoDouble builds a DoublingCube domain.PositionAnalysis from a
// raw EngineStructDoubleAction for a standalone "No Double" cube decision.
func buildRawCubeForNoDouble(doubled *xgparser.EngineStructDoubleAction) *domain.PositionAnalysis {
	if doubled == nil {
		return nil
	}
	posAnalysis := &domain.PositionAnalysis{
		AnalysisType:          "DoublingCube",
		AnalysisEngineVersion: "XG",
		PlayedCubeActions:     []string{"No Double"},
		CreationDate:          time.Now(),
		LastModifiedDate:      time.Now(),
	}
	cube := buildDoublingCubeAnalysis(rawCubeParams(doubled))
	posAnalysis.DoublingCubeAnalysis = &cube
	return posAnalysis
}

// rawCubeParams maps an EngineStructDoubleAction's Eval array to cubeAnalysisParams.
// XG Eval layout: [0]=opp bg, [1]=opp gammon, [2]=opp win, [4]=player gammon,
// [5]=player bg, [6]=cubeless equity.
func rawCubeParams(doubled *xgparser.EngineStructDoubleAction) cubeAnalysisParams {
	return cubeAnalysisParams{
		Depth:                     translateAnalysisDepth(int(doubled.Level)),
		Engine:                    "XG",
		PlayerWinChances:          (1.0 - float64(doubled.Eval[2])) * 100.0,
		PlayerGammonChances:       float64(doubled.Eval[4]) * 100.0,
		PlayerBackgammonChances:   float64(doubled.Eval[5]) * 100.0,
		OpponentWinChances:        float64(doubled.Eval[2]) * 100.0,
		OpponentGammonChances:     float64(doubled.Eval[1]) * 100.0,
		OpponentBackgammonChances: float64(doubled.Eval[0]) * 100.0,
		CubelessNoDoubleEquity:    float64(doubled.Eval[6]),
		CubelessDoubleEquity:      float64(doubled.Eval[6]),
		CubefulNoDoubleEquity:     float64(doubled.EquB),
		CubefulDoubleTakeEquity:   float64(doubled.EquDouble),
		CubefulDoublePassEquity:   float64(doubled.EquDrop),
	}
}
