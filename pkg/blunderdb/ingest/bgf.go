package ingest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/kevung/bgfparser"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// MapBGF parses a BGBlitz .bgf file into a backend-independent MatchGraph,
// re-implementing the mapping half of database.ImportBGFMatch without SQLite.
// See MapXG for the model; like it, MapBGF always emits the full graph and
// defers cross-format canonical-duplicate enrichment to a follow-up.
func MapBGF(path string) (*MatchGraph, error) {
	match, err := bgfparser.ParseBGF(path)
	if err != nil {
		return nil, fmt.Errorf("ingest: parse bgf file: %w", err)
	}
	if match.Data == nil {
		return nil, fmt.Errorf("ingest: bgf file contains no match data")
	}
	data := match.Data

	matchLen := bgfGetInt(data, "matchlen")
	gamesData, ok := data["games"].([]interface{})
	if !ok || len(gamesData) == 0 {
		return nil, fmt.Errorf("ingest: bgf file contains no games")
	}

	graph := &MatchGraph{
		Match: domain.Match{
			Player1Name:   bgfGetString(data, "nameGreen"),
			Player2Name:   bgfGetString(data, "nameRed"),
			Event:         bgfGetString(data, "event"),
			Location:      bgfGetString(data, "location"),
			Round:         bgfGetString(data, "round"),
			MatchLength:   int32(matchLen),
			MatchDate:     parseMatchDate(bgfGetString(data, "date")),
			FilePath:      path,
			GameCount:     len(gamesData),
			MatchHash:     computeBGFMatchHash(match),
			CanonicalHash: computeCanonicalMatchHashFromBGF(match),
		},
	}

	for gameIdx, gameRaw := range gamesData {
		gameData, ok := gameRaw.(map[string]interface{})
		if !ok {
			continue
		}
		movesData, _ := gameData["moves"].([]interface{})
		graph.Games = append(graph.Games, GameGraph{
			Game: domain.Game{
				GameNumber:   int32(gameIdx + 1),
				InitialScore: [2]int32{int32(bgfGetInt(gameData, "scoreGreen")), int32(bgfGetInt(gameData, "scoreRed"))},
				Winner:       0,
				PointsWon:    int32(bgfGetInt(gameData, "wonPoints")),
				MoveCount:    len(movesData),
			},
			Moves: mapBGFGameMoves(gameData, movesData, matchLen),
		})
	}

	return graph, nil
}

// mapBGFGameMoves replays one game's move list, tracking the board and cube
// state and emitting a MoveGraph per checker move and per cube decision. It
// mirrors the normal-import loop in database.ImportBGFMatch, including the two
// ways BGBlitz encodes a cube double (a synthetic "amove" with from[0]==-1, and
// an explicit "adouble").
func mapBGFGameMoves(gameData map[string]interface{}, movesData []interface{}, matchLen int) []MoveGraph {
	var out []MoveGraph

	boardState := bgfInitBoardFromGame(gameData)
	cubeValue := 1
	cubeOwner := -1
	isCrawford := bgfGetBool(gameData, "isCrawford")
	pendingCubeDouble := false

	for moveIdx, moveRaw := range movesData {
		moveData, ok := moveRaw.(map[string]interface{})
		if !ok {
			continue
		}
		mtype := bgfGetString(moveData, "type")
		player := bgfGetInt(moveData, "player")

		switch mtype {
		case "amove":
			fromArr := bgfGetIntArray(moveData, "from")
			if fromArr[0] == -1 {
				// Cube action encoded as an amove.
				if pendingCubeDouble {
					pendingCubeDouble = false
					if equity := bgfGetMap(moveData, "equity"); equity != nil {
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
				pendingCubeDouble = true
				cubeAction := bgfAmoveDoubleResponse(movesData, moveIdx)
				out = append(out, mapBGFCubeMove(moveData, gameData, matchLen, boardState, cubeValue, cubeOwner, isCrawford, cubeAction))
				continue
			}

			// Normal checker move.
			out = append(out, mapBGFCheckerMove(moveData, gameData, matchLen, boardState, cubeValue, cubeOwner, isCrawford))
			// Skip board update for green=7 (unplayable die marker; analysis-only).
			if bgfGetInt(moveData, "green") != 7 {
				bgfApplyCheckerMove(&boardState, moveData, player)
			}

		case "adouble":
			cubeAction := bgfADoubleResponse(movesData, moveIdx)
			out = append(out, mapBGFCubeMove(moveData, gameData, matchLen, boardState, cubeValue, cubeOwner, isCrawford, cubeAction))

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

		case "apass":
			// Game ends; no state update.
		}
	}

	return out
}

// bgfAmoveDoubleResponse reads the take/pass response to an amove-encoded double.
func bgfAmoveDoubleResponse(movesData []interface{}, idx int) string {
	cubeAction := "Double/Take" // default
	for j := idx + 1; j < len(movesData); j++ {
		nextMove, ok := movesData[j].(map[string]interface{})
		if !ok {
			continue
		}
		nextFrom := bgfGetIntArray(nextMove, "from")
		if nextFrom[0] == -1 {
			if eq := bgfGetMap(nextMove, "equity"); eq != nil {
				cd := bgfGetMap(eq, "cubeDecision")
				if cd != nil && !bgfGetBool(cd, "hasAccepted") {
					cubeAction = "Double/Pass"
				}
			}
			break
		}
		break
	}
	return cubeAction
}

// bgfADoubleResponse reads the atake/apass response to an explicit adouble.
func bgfADoubleResponse(movesData []interface{}, idx int) string {
	cubeAction := "Double/Pass"
	for j := idx + 1; j < len(movesData); j++ {
		nextMove, ok := movesData[j].(map[string]interface{})
		if !ok {
			continue
		}
		switch bgfGetString(nextMove, "type") {
		case "atake":
			return "Double/Take"
		case "apass":
			return "Double/Pass"
		case "amove":
			continue
		default:
			return cubeAction
		}
	}
	return cubeAction
}

// bgfInferDice returns the two dice for a checker move, inferring them from the
// sub-moves when BGBlitz stores impossible die markers (e.g. green=7).
func bgfInferDice(moveData map[string]interface{}) (int, int) {
	die1 := bgfGetInt(moveData, "green")
	die2 := bgfGetInt(moveData, "red")

	if die1 > 6 || die2 > 6 || die1 < 1 || die2 < 1 {
		fromArr := bgfGetIntArray(moveData, "from")
		toArr := bgfGetIntArray(moveData, "to")
		if fromArr[0] == -1 {
			return die1, die2
		}
		var diceUsed []int
		for j := 0; j < 4; j++ {
			if fromArr[j] == -1 {
				break
			}
			f := fromArr[j]
			t := toArr[j]
			if f == 25 {
				diceUsed = append(diceUsed, t)
			} else if t == 0 {
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
			if die1 >= 1 && die1 <= 6 {
				die2 = diceUsed[0]
			} else if die2 >= 1 && die2 <= 6 {
				die1 = diceUsed[0]
			} else {
				die1 = diceUsed[0]
				die2 = diceUsed[0]
			}
		}
	}
	return die1, die2
}

// mapBGFCheckerMove builds the MoveGraph for a checker decision, with ordered
// [checker, cube-for-checker] analysis fragments matching the legacy two saves.
func mapBGFCheckerMove(moveData, gameData map[string]interface{}, matchLen int, boardState [28]int, cubeValue, cubeOwner int, isCrawford bool) MoveGraph {
	player := bgfGetInt(moveData, "player")
	blunderDBPlayer := bgfPlayerToBlunderDB(player)
	die1, die2 := bgfInferDice(moveData)

	pos := createPositionFromBGF(boardState, gameData, matchLen, cubeValue, cubeOwner, isCrawford)
	pos.PlayerOnRoll = blunderDBPlayer
	pos.DecisionType = domain.CheckerAction
	pos.Dice = [2]int{die1, die2}

	checkerMoveStr := bgfConvertMoveToString(moveData)

	var analyses []*domain.PositionAnalysis
	if moveAnalysis, ok := moveData["moveAnalysis"].([]interface{}); ok && len(moveAnalysis) > 0 {
		if a := buildBGFCheckerAnalysis(moveAnalysis, checkerMoveStr); a != nil {
			analyses = append(analyses, a)
		}
	}
	if equity := bgfGetMap(moveData, "equity"); equity != nil {
		if cd := bgfGetMap(equity, "cubeDecision"); cd != nil && bgfGetString(cd, "stateOnMove") != "" {
			analyses = append(analyses, buildBGFCubeForChecker(equity, cd))
		}
	}

	return MoveGraph{
		Move: domain.Move{
			MoveType:    "checker",
			Player:      convertBlunderDBPlayerToXG(blunderDBPlayer),
			Dice:        [2]int32{int32(die1), int32(die2)},
			CheckerMove: checkerMoveStr,
		},
		Position: pos,
		Analyses: analyses,
	}
}

// mapBGFCubeMove builds the MoveGraph for a cube decision.
func mapBGFCubeMove(moveData, gameData map[string]interface{}, matchLen int, boardState [28]int, cubeValue, cubeOwner int, isCrawford bool, cubeAction string) MoveGraph {
	player := bgfGetInt(moveData, "player")
	blunderDBPlayer := bgfPlayerToBlunderDB(player)

	pos := createPositionFromBGF(boardState, gameData, matchLen, cubeValue, cubeOwner, isCrawford)
	pos.PlayerOnRoll = blunderDBPlayer
	pos.DecisionType = domain.CubeAction
	pos.Dice = [2]int{0, 0}

	var analyses []*domain.PositionAnalysis
	if equity := bgfGetMap(moveData, "equity"); equity != nil {
		if cd := bgfGetMap(equity, "cubeDecision"); cd != nil {
			analyses = append(analyses, buildBGFCubeAnalysis(equity, cd, cubeAction))
		}
	}

	return MoveGraph{
		Move: domain.Move{
			MoveType:   "cube",
			Player:     convertBlunderDBPlayerToXG(blunderDBPlayer),
			Dice:       [2]int32{0, 0},
			CubeAction: cubeAction,
		},
		Position: pos,
		Analyses: analyses,
	}
}

// convertBlunderDBPlayerToXG converts a blunderDB player index (0/1) to XG
// encoding (1/-1) for the move table's player column (CLI/GUI read it back via
// the XG decoder uniformly). Same as blunderDBPlayerToXG; named to match the
// legacy call sites in the GnuBG/BGF mappers.
func convertBlunderDBPlayerToXG(p int) int32 { return blunderDBPlayerToXG(p) }

// computeBGFMatchHash is the format-specific content hash. Copied from
// database.ComputeBGFMatchHash; TestBGFHashParity keeps them in lock-step.
func computeBGFMatchHash(match *bgfparser.Match) string {
	var b strings.Builder
	data := match.Data
	p1 := strings.TrimSpace(strings.ToLower(bgfGetString(data, "nameGreen")))
	p2 := strings.TrimSpace(strings.ToLower(bgfGetString(data, "nameRed")))
	b.WriteString(fmt.Sprintf("bgf:%s|%s|%d|", p1, p2, bgfGetInt(data, "matchlen")))

	gamesData, _ := data["games"].([]interface{})
	for gameIdx, gameRaw := range gamesData {
		g, ok := gameRaw.(map[string]interface{})
		if !ok {
			continue
		}
		b.WriteString(fmt.Sprintf("g%d:%d,%d,%d|",
			gameIdx, bgfGetInt(g, "scoreGreen"), bgfGetInt(g, "scoreRed"), bgfGetInt(g, "wonPoints")))
		movesData, _ := g["moves"].([]interface{})
		for moveIdx, moveRaw := range movesData {
			m, ok := moveRaw.(map[string]interface{})
			if !ok {
				continue
			}
			mtype := bgfGetString(m, "type")
			b.WriteString(fmt.Sprintf("m%d:%s,", moveIdx, mtype))
			if mtype == "amove" {
				b.WriteString(fmt.Sprintf("d%d%d|", bgfGetInt(m, "green"), bgfGetInt(m, "red")))
			} else if mtype == "adouble" || mtype == "atake" || mtype == "apass" {
				b.WriteString(fmt.Sprintf("c%s|", mtype))
			}
		}
	}

	hash := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(hash[:])
}

// computeCanonicalMatchHashFromBGF is the format-independent hash, identical to
// computeCanonicalMatchHashFromXG for the same match. Copied from
// database.ComputeCanonicalMatchHashFromBGF.
func computeCanonicalMatchHashFromBGF(match *bgfparser.Match) string {
	var b strings.Builder
	data := match.Data
	p1 := strings.TrimSpace(strings.ToLower(bgfGetString(data, "nameGreen")))
	p2 := strings.TrimSpace(strings.ToLower(bgfGetString(data, "nameRed")))
	if p1 > p2 {
		p1, p2 = p2, p1
	}
	gamesData, _ := data["games"].([]interface{})
	b.WriteString(fmt.Sprintf("canonical2:%s|%s|%d|%d|", p1, p2, bgfGetInt(data, "matchlen"), len(gamesData)))

	for gameIdx, gameRaw := range gamesData {
		g, ok := gameRaw.(map[string]interface{})
		if !ok {
			continue
		}
		b.WriteString(fmt.Sprintf("g%d|", gameIdx))
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
			if bgfGetString(m, "type") == "amove" {
				fromArr := bgfGetIntArray(m, "from")
				if fromArr[0] == -1 {
					continue
				}
				d1 := bgfGetInt(m, "green")
				d2 := bgfGetInt(m, "red")
				if d1 > d2 {
					d1, d2 = d2, d1
				}
				b.WriteString(fmt.Sprintf("d%d%d|", d1, d2))
				diceCount++
			}
		}
	}

	hash := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(hash[:])
}

// BGFImporter implements Importer for BGBlitz .bgf files.
type BGFImporter struct{ S storage.Storage }

func (im BGFImporter) Import(ctx context.Context, scope string, src Source, prog func(Progress)) (Summary, error) {
	if src.Path == "" {
		return Summary{}, fmt.Errorf("ingest: bgf import requires a file path")
	}

	graph, err := MapBGF(src.Path)
	if err != nil {
		return Summary{}, err
	}

	tx, err := im.S.BeginTx(ctx)
	if err != nil {
		return Summary{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	res, err := WriteMatch(ctx, tx, scope, graph, prog)
	if err != nil {
		return Summary{}, err
	}
	if err := ctx.Err(); err != nil {
		return Summary{}, err
	}
	if err := tx.Commit(); err != nil {
		return Summary{}, err
	}
	committed = true

	sum := Summary{SavedPositions: res.SavedPositions, Matches: 1, MatchID: res.MatchID}
	if res.Skipped {
		sum.SkippedDuplicates = 1
		sum.SavedPositions = 0
	}
	return sum, nil
}
