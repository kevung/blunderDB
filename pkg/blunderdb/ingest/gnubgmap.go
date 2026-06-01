package ingest

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/gnubgparser"
)

// Pure GnuBG (SGF/MAT/TXT) → domain mappers, lifted from
// pkg/blunderdb/database/db_import_gnubg.go with the *Database receiver dropped.
// The legacy path is left untouched; the gnubg parity test gates these against
// it. See xgmap.go for the equivalent XG mappers and the rationale.

// initStandardGnuBGPosition returns a gnubgparser.Position set to the standard
// starting position, used when an SGF lacks an explicit setboard event.
func initStandardGnuBGPosition() gnubgparser.Position {
	var pos gnubgparser.Position
	pos.CubeValue = 1
	pos.CubeOwner = -1 // center
	for p := 0; p < 2; p++ {
		pos.Board[p][23] = 2
		pos.Board[p][12] = 5
		pos.Board[p][7] = 3
		pos.Board[p][5] = 5
	}
	return pos
}

// applyGnuBGCheckerMove updates a gnubgparser board state after a checker move.
// isAbsoluteCoords is true for SGF (absolute coords, mirrored for Player 1) and
// false for MAT/TXT (player-relative coords).
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
				if player == 0 {
					opponentBoard = 23 - to
				} else {
					opponentBoard = to
				}
			}
		} else {
			fromBoard = from
			isBearOff = (to == -1)
			if !isBearOff {
				toBoard = to
				opponentBoard = 23 - to
			}
		}

		if fromBoard >= 0 && fromBoard <= 24 {
			board.Board[player][fromBoard]--
		}
		if isBearOff {
			continue
		}
		if opponentBoard >= 0 && opponentBoard <= 23 {
			if board.Board[opponent][opponentBoard] == 1 {
				board.Board[opponent][opponentBoard] = 0
				board.Board[opponent][24]++
			}
		}
		if toBoard >= 0 && toBoard <= 24 {
			board.Board[player][toBoard]++
		}
	}
}

// createPositionFromGnuBG converts a gnubgparser.Position to a domain.Position.
func createPositionFromGnuBG(gnubgPos *gnubgparser.Position, game *gnubgparser.Game, matchLength int) (*domain.Position, error) {
	awayScore1 := matchLength - game.Score[0]
	awayScore2 := matchLength - game.Score[1]
	if matchLength == 0 {
		awayScore1 = -1
		awayScore2 = -1
	}

	cubeValue := 0
	if gnubgPos.CubeValue > 0 {
		for v := gnubgPos.CubeValue; v > 1; v >>= 1 {
			cubeValue++
		}
	}

	pos := &domain.Position{
		PlayerOnRoll: gnubgPos.OnRoll,
		DecisionType: domain.CheckerAction,
		Score:        [2]int{awayScore1, awayScore2},
		Cube: domain.Cube{
			Value: cubeValue,
			Owner: gnubgPos.CubeOwner,
		},
		Dice: [2]int{0, 0},
	}

	for i := 0; i < 26; i++ {
		pos.Board.Points[i] = domain.Point{Checkers: 0, Color: -1}
	}

	// Player 0 (Black): gnubg pt → blunderDB pt+1; bar → index 25.
	for pt := 0; pt < 25; pt++ {
		count := gnubgPos.Board[0][pt]
		if count > 0 {
			if pt == 24 {
				pos.Board.Points[25] = domain.Point{Checkers: count, Color: 0}
			} else {
				pos.Board.Points[pt+1] = domain.Point{Checkers: count, Color: 0}
			}
		}
	}

	// Player 1 (White): gnubg pt → blunderDB 24-pt; bar → index 0.
	for pt := 0; pt < 25; pt++ {
		count := gnubgPos.Board[1][pt]
		if count > 0 {
			if pt == 24 {
				pos.Board.Points[0] = domain.Point{Checkers: count, Color: 1}
			} else {
				pos.Board.Points[24-pt] = domain.Point{Checkers: count, Color: 1}
			}
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

	return pos, nil
}

// convertGnuBGMoveToString converts an absolute-coordinate (SGF) move to notation.
func convertGnuBGMoveToString(move [8]int, player int) string {
	formatPoint := func(pt int, p int) string {
		if pt == 24 {
			return "bar"
		}
		if pt == 25 {
			return "off"
		}
		if p == 0 {
			return fmt.Sprintf("%d", pt+1)
		}
		return fmt.Sprintf("%d", 24-pt)
	}
	return formatGnuBGMoveItems(move, player, formatPoint)
}

// convertPlayerRelativeMoveToString converts a player-relative (MAT/TXT) move.
func convertPlayerRelativeMoveToString(move [8]int) string {
	formatPoint := func(pt int, _ int) string {
		if pt == 24 {
			return "bar"
		}
		if pt == -1 {
			return "off"
		}
		return fmt.Sprintf("%d", pt+1)
	}
	return formatGnuBGMoveItems(move, 0, formatPoint)
}

// formatGnuBGMoveItems formats a move array using a point formatter, sorting by
// source point descending and grouping repeats with a multiplier.
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

	sort.Slice(items, func(i, j int) bool {
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

// translateGnuBGAnalysisDepth converts a gnuBG ply level to a label.
func translateGnuBGAnalysisDepth(depth int) string {
	if depth >= 0 {
		return fmt.Sprintf("%d-ply", depth)
	}
	return fmt.Sprintf("%d", depth)
}

// gnuBGCheckerMoveStr returns the played/option move string for a move array,
// honouring the SGF (absolute) vs MAT/TXT (relative) coordinate difference and
// preferring a pre-rendered MoveString when present (MAT/TXT).
func gnuBGCheckerMoveStr(move [8]int, moveString string, player int, isSGF bool) string {
	if isSGF {
		return convertGnuBGMoveToString(move, player)
	}
	if moveString != "" {
		return moveString
	}
	return convertPlayerRelativeMoveToString(move)
}

// buildGnuBGCheckerAnalysis converts a gnubgparser MoveAnalysis into a checker
// PositionAnalysis fragment.
func buildGnuBGCheckerAnalysis(analysis *gnubgparser.MoveAnalysis, player int, playedMoveStr string, isSGF bool) *domain.PositionAnalysis {
	if analysis == nil || len(analysis.Moves) == 0 {
		return nil
	}

	posAnalysis := &domain.PositionAnalysis{
		AnalysisType:          "CheckerMove",
		AnalysisEngineVersion: "GNU Backgammon",
		CreationDate:          time.Now(),
		LastModifiedDate:      time.Now(),
	}

	checkerMoves := make([]domain.CheckerMove, 0, len(analysis.Moves))
	for i, moveOpt := range analysis.Moves {
		moveStr := gnuBGCheckerMoveStr(moveOpt.Move, moveOpt.MoveString, player, isSGF)

		var equityError *float64
		if i > 0 {
			diff := analysis.Moves[0].Equity - moveOpt.Equity
			equityError = &diff
		}

		checkerMoves = append(checkerMoves, domain.CheckerMove{
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
		})
	}

	posAnalysis.CheckerAnalysis = &domain.CheckerAnalysis{Moves: checkerMoves}
	if playedMoveStr != "" {
		posAnalysis.PlayedMoves = []string{playedMoveStr}
	}
	return posAnalysis
}

// gnuBGCubeParams maps a gnubgparser.CubeAnalysis to cubeAnalysisParams.
func gnuBGCubeParams(analysis *gnubgparser.CubeAnalysis) cubeAnalysisParams {
	return cubeAnalysisParams{
		Depth:                     translateGnuBGAnalysisDepth(analysis.AnalysisDepth),
		Engine:                    "GNUbg",
		PlayerWinChances:          float64(analysis.Player1WinRate) * 100.0,
		PlayerGammonChances:       float64(analysis.Player1GammonRate) * 100.0,
		PlayerBackgammonChances:   float64(analysis.Player1BackgammonRate) * 100.0,
		OpponentWinChances:        float64(analysis.Player2WinRate) * 100.0,
		OpponentGammonChances:     float64(analysis.Player2GammonRate) * 100.0,
		OpponentBackgammonChances: float64(analysis.Player2BackgammonRate) * 100.0,
		CubelessNoDoubleEquity:    analysis.CubelessEquity,
		CubelessDoubleEquity:      analysis.CubelessEquity,
		CubefulNoDoubleEquity:     analysis.CubefulNoDouble,
		CubefulDoubleTakeEquity:   analysis.CubefulDoubleTake,
		CubefulDoublePassEquity:   analysis.CubefulDoublePass,
		WrongPassPercentage:       float64(analysis.WrongPassTakePercent) * 100.0,
	}
}

// buildGnuBGCubeAnalysis builds a DoublingCube PositionAnalysis fragment for a
// cube decision, recording the played cube action.
func buildGnuBGCubeAnalysis(analysis *gnubgparser.CubeAnalysis, playedCubeAction string) *domain.PositionAnalysis {
	if analysis == nil {
		return nil
	}
	cube := buildDoublingCubeAnalysis(gnuBGCubeParams(analysis))
	posAnalysis := &domain.PositionAnalysis{
		AnalysisType:          "DoublingCube",
		AnalysisEngineVersion: "GNU Backgammon",
		DoublingCubeAnalysis:  &cube,
		CreationDate:          time.Now(),
		LastModifiedDate:      time.Now(),
	}
	if playedCubeAction != "" {
		posAnalysis.PlayedCubeActions = []string{playedCubeAction}
	}
	return posAnalysis
}

// buildGnuBGCubeForChecker builds a cube fragment to attach to a checker
// position (AnalysisType "CheckerMove"), with no played cube action.
func buildGnuBGCubeForChecker(analysis *gnubgparser.CubeAnalysis) *domain.PositionAnalysis {
	if analysis == nil {
		return nil
	}
	cube := buildDoublingCubeAnalysis(gnuBGCubeParams(analysis))
	return &domain.PositionAnalysis{
		AnalysisType:          "CheckerMove",
		AnalysisEngineVersion: "GNU Backgammon",
		DoublingCubeAnalysis:  &cube,
		CreationDate:          time.Now(),
		LastModifiedDate:      time.Now(),
	}
}

// convertGnuBGCubeMWCToEMG converts a cube analysis' cubeful equities from Match
// Winning Chances to Equivalent Money Game equity, mirroring GNUbg's mwc2eq().
// Copied from database.convertGnuBGCubeMWCToEMG (depends only on engine + parser).
func convertGnuBGCubeMWCToEMG(analysis *gnubgparser.CubeAnalysis, score0, score1, fMove, cubeValue, matchLength int) {
	if matchLength <= 0 || analysis == nil {
		return
	}
	fCrawford := false

	mwcWin := float32(engine.GnuBGGetME(score0, score1, matchLength, fMove, cubeValue, fMove, fCrawford))
	mwcLose := float32(engine.GnuBGGetME(score0, score1, matchLength, fMove, cubeValue, 1-fMove, fCrawford))

	denom := mwcWin - mwcLose
	if denom < 1e-7 && denom > -1e-7 {
		analysis.CubefulDoublePass = 1.0
		return
	}
	sum := mwcWin + mwcLose

	ndMwc := float32(analysis.CubefulNoDouble)
	dtMwc := float32(analysis.CubefulDoubleTake)

	analysis.CubefulNoDouble = float64((2.0*ndMwc - sum) / denom)
	analysis.CubefulDoubleTake = float64((2.0*dtMwc - sum) / denom)
	analysis.CubefulDoublePass = 1.0

	effectiveDouble := analysis.CubefulDoubleTake
	if analysis.CubefulDoublePass < analysis.CubefulDoubleTake {
		effectiveDouble = analysis.CubefulDoublePass
	}
	if effectiveDouble > analysis.CubefulNoDouble {
		if analysis.CubefulDoubleTake <= analysis.CubefulDoublePass {
			analysis.BestAction = "Double, Take"
		} else {
			analysis.BestAction = "Double, Pass"
		}
	} else {
		analysis.BestAction = "No Double"
	}
}
