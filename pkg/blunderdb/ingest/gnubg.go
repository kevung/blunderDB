package ingest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/gnubgparser"
)

// MapGnuBG parses a GnuBG file (.sgf with analysis, or .mat/.txt moves-only)
// into a backend-independent MatchGraph. It re-implements the mapping half of
// database.ImportGnuBGMatch without touching SQLite. See MapXG for the model.
//
// Like MapXG it always produces the full graph; cross-format canonical-duplicate
// enrichment is deferred (WriteMatch skips a whole match on a hash hit).
func MapGnuBG(path string) (*MatchGraph, error) {
	ext := strings.ToLower(filepath.Ext(path))
	isSGF := ext == ".sgf"

	var match *gnubgparser.Match
	var err error
	switch ext {
	case ".sgf":
		match, err = gnubgparser.ParseSGFFile(path)
	case ".mat", ".txt":
		match, err = gnubgparser.ParseMATFile(path)
	default:
		return nil, fmt.Errorf("ingest: unsupported gnubg file format: %s", ext)
	}
	if err != nil {
		return nil, fmt.Errorf("ingest: parse gnubg file: %w", err)
	}

	graph := &MatchGraph{
		Match: domain.Match{
			Player1Name:   match.Metadata.Player1,
			Player2Name:   match.Metadata.Player2,
			Event:         match.Metadata.Event,
			Location:      match.Metadata.Place,
			Round:         match.Metadata.Round,
			MatchLength:   int32(match.Metadata.MatchLength),
			MatchDate:     parseMatchDate(match.Metadata.Date),
			FilePath:      path,
			GameCount:     len(match.Games),
			MatchHash:     computeGnuBGMatchHash(match),
			CanonicalHash: computeCanonicalMatchHashFromGnuBG(match),
		},
	}

	for gameIdx := range match.Games {
		game := match.Games[gameIdx]
		game.GameNumber = gameIdx + 1
		graph.Games = append(graph.Games, GameGraph{
			Game: domain.Game{
				GameNumber:   int32(game.GameNumber),
				InitialScore: [2]int32{int32(game.Score[0]), int32(game.Score[1])},
				Winner:       int32(game.Winner),
				PointsWon:    int32(game.Points),
				MoveCount:    len(game.Moves),
			},
			Moves: mapGnuBGGameMoves(&game, match.Metadata.MatchLength, isSGF),
		})
	}

	return graph, nil
}

// mapGnuBGGameMoves replays one game's move records, reconstructing the board
// state (setboard/setcube/setcubepos/setdice events plus checker moves) and
// emitting a MoveGraph per checker move and per cube decision. It mirrors the
// normal-import loop in database.importGnuBGMatchInternal.
func mapGnuBGGameMoves(game *gnubgparser.Game, matchLength int, isSGF bool) []MoveGraph {
	var out []MoveGraph
	currentBoard := initStandardGnuBGPosition()
	moveNumber := int32(0)

	for i := range game.Moves {
		moveRec := &game.Moves[i]

		switch string(moveRec.Type) {
		case "setboard":
			if moveRec.Position != nil {
				currentBoard = *moveRec.Position
			}
			continue
		case "setdice":
			continue
		case "setcube":
			currentBoard.CubeValue = moveRec.CubeValue
			continue
		case "setcubepos":
			currentBoard.CubeOwner = moveRec.CubeOwner
			continue
		}

		// Snapshot the current board for this move if the parser didn't supply one.
		posPtr := moveRec.Position
		if posPtr == nil {
			c := currentBoard
			posPtr = &c
		}

		switch moveRec.Type {
		case "move":
			out = append(out, mapGnuBGCheckerMove(moveNumber, moveRec, posPtr, game, matchLength, isSGF))
		case "double":
			out = append(out, mapGnuBGCubeMove(moveNumber, moveRec, posPtr, game, matchLength, i))
		}

		// Advance the board.
		switch string(moveRec.Type) {
		case "move":
			applyGnuBGCheckerMove(&currentBoard, moveRec, isSGF)
		case "take":
			if currentBoard.CubeValue == 0 {
				currentBoard.CubeValue = 2
			} else {
				currentBoard.CubeValue *= 2
			}
			currentBoard.CubeOwner = moveRec.Player
		}

		if string(moveRec.Type) != "take" && string(moveRec.Type) != "drop" {
			moveNumber++
		}
	}

	return out
}

// mapGnuBGCheckerMove builds the MoveGraph for a checker decision, attaching the
// checker analysis and (if present) the cube-for-checker analysis as ordered
// fragments — matching the legacy two-save sequence.
func mapGnuBGCheckerMove(moveNumber int32, moveRec *gnubgparser.MoveRecord, posPtr *gnubgparser.Position, game *gnubgparser.Game, matchLength int, isSGF bool) MoveGraph {
	player := moveRec.Player
	checkerMoveStr := gnuBGCheckerMoveStr(moveRec.Move, moveRec.MoveString, player, isSGF)

	pos, _ := createPositionFromGnuBG(posPtr, game, matchLength)
	pos.PlayerOnRoll = player
	pos.DecisionType = domain.CheckerAction
	pos.Dice = [2]int{moveRec.Dice[0], moveRec.Dice[1]}

	var analyses []*domain.PositionAnalysis
	if moveRec.Analysis != nil && len(moveRec.Analysis.Moves) > 0 {
		if a := buildGnuBGCheckerAnalysis(moveRec.Analysis, player, checkerMoveStr, isSGF); a != nil {
			analyses = append(analyses, a)
		}
	}
	if moveRec.CubeAnalysis != nil {
		cubeAnalysis := *moveRec.CubeAnalysis
		if matchLength > 0 {
			convertGnuBGCubeMWCToEMG(&cubeAnalysis, game.Score[0], game.Score[1], player, posPtr.CubeValue, matchLength)
		}
		if a := buildGnuBGCubeForChecker(&cubeAnalysis); a != nil {
			analyses = append(analyses, a)
		}
	}

	return MoveGraph{
		Move: domain.Move{
			MoveNumber:  moveNumber,
			MoveType:    "checker",
			Player:      blunderDBPlayerToXG(player),
			Dice:        [2]int32{int32(moveRec.Dice[0]), int32(moveRec.Dice[1])},
			CheckerMove: checkerMoveStr,
		},
		Position: pos,
		Analyses: analyses,
	}
}

// mapGnuBGCubeMove builds the MoveGraph for a "double" decision. The played
// action (Double/Take vs Double/Pass) is read from the opponent's response.
func mapGnuBGCubeMove(moveNumber int32, moveRec *gnubgparser.MoveRecord, posPtr *gnubgparser.Position, game *gnubgparser.Game, matchLength, idx int) MoveGraph {
	player := moveRec.Player
	cubeAction := gnuBGDoubleResponse(game, idx)

	pos, _ := createPositionFromGnuBG(posPtr, game, matchLength)
	pos.PlayerOnRoll = player
	pos.DecisionType = domain.CubeAction
	pos.Dice = [2]int{0, 0}

	var analyses []*domain.PositionAnalysis
	if moveRec.CubeAnalysis != nil {
		cubeAnalysis := *moveRec.CubeAnalysis
		if matchLength > 0 {
			convertGnuBGCubeMWCToEMG(&cubeAnalysis, game.Score[0], game.Score[1], player, posPtr.CubeValue, matchLength)
		}
		if a := buildGnuBGCubeAnalysis(&cubeAnalysis, cubeAction); a != nil {
			analyses = append(analyses, a)
		}
	}

	return MoveGraph{
		Move: domain.Move{
			MoveNumber: moveNumber,
			MoveType:   "cube",
			Player:     blunderDBPlayerToXG(player),
			Dice:       [2]int32{0, 0},
			CubeAction: cubeAction,
		},
		Position: pos,
		Analyses: analyses,
	}
}

// gnuBGDoubleResponse looks ahead from a "double" record at index idx to find
// the opponent's take/drop response, returning "Double/Take" or "Double/Pass".
func gnuBGDoubleResponse(game *gnubgparser.Game, idx int) string {
	cubeAction := "Double/Pass" // default
	for j := idx + 1; j < len(game.Moves); j++ {
		switch string(game.Moves[j].Type) {
		case "take":
			return "Double/Take"
		case "drop":
			return "Double/Pass"
		case "setboard", "setdice", "setcube", "setcubepos":
			continue
		default:
			return cubeAction
		}
	}
	return cubeAction
}

// computeGnuBGMatchHash is the format-specific content hash. Copied from
// database.ComputeGnuBGMatchHash; TestGnuBGHashParity keeps them in lock-step.
func computeGnuBGMatchHash(match *gnubgparser.Match) string {
	var b strings.Builder
	p1 := strings.TrimSpace(strings.ToLower(match.Metadata.Player1))
	p2 := strings.TrimSpace(strings.ToLower(match.Metadata.Player2))
	b.WriteString(fmt.Sprintf("meta:%s|%s|%d|", p1, p2, match.Metadata.MatchLength))

	for gameIdx, game := range match.Games {
		b.WriteString(fmt.Sprintf("g%d:%d,%d,%d,%d|",
			gameIdx, game.Score[0], game.Score[1], game.Winner, game.Points))
		for moveIdx, moveRec := range game.Moves {
			b.WriteString(fmt.Sprintf("m%d:%s,", moveIdx, string(moveRec.Type)))
			if moveRec.Type == "move" {
				b.WriteString(fmt.Sprintf("d%d%d,p%s|",
					moveRec.Dice[0], moveRec.Dice[1], moveRec.MoveString))
			} else if moveRec.Type == "double" || moveRec.Type == "take" || moveRec.Type == "drop" {
				b.WriteString(fmt.Sprintf("c%s|", string(moveRec.Type)))
			}
		}
	}

	hash := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(hash[:])
}

// computeCanonicalMatchHashFromGnuBG is the format-independent hash, identical
// to computeCanonicalMatchHashFromXG for the same match (cross-format dedup).
// Copied from database.ComputeCanonicalMatchHashFromGnuBG.
func computeCanonicalMatchHashFromGnuBG(match *gnubgparser.Match) string {
	var b strings.Builder
	p1 := strings.TrimSpace(strings.ToLower(match.Metadata.Player1))
	p2 := strings.TrimSpace(strings.ToLower(match.Metadata.Player2))
	if p1 > p2 {
		p1, p2 = p2, p1
	}
	b.WriteString(fmt.Sprintf("canonical2:%s|%s|%d|%d|", p1, p2, match.Metadata.MatchLength, len(match.Games)))

	for gameIdx, game := range match.Games {
		b.WriteString(fmt.Sprintf("g%d|", gameIdx))
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
				b.WriteString(fmt.Sprintf("d%d%d|", d1, d2))
				diceCount++
			}
		}
	}

	hash := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(hash[:])
}

// GnuBGImporter implements Importer for GnuBG .sgf/.mat/.txt files.
type GnuBGImporter struct{ S storage.Storage }

func (im GnuBGImporter) Import(ctx context.Context, scope string, src Source, prog func(Progress)) (Summary, error) {
	if src.Path == "" {
		return Summary{}, fmt.Errorf("ingest: gnubg import requires a file path")
	}

	graph, err := MapGnuBG(src.Path)
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
