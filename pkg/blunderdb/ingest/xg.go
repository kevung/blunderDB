package ingest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/xgparser/xgparser"
)

// MapXG parses an eXtreme Gammon .xg file at path into a backend-independent
// MatchGraph (positions + analyses + comments + move list). It re-implements
// the mapping half of database.ImportXGMatch — the half that turns parsed XG
// records into domain objects — without touching SQLite. WriteMatch persists
// the result through any Storage backend.
//
// Note: MapXG always produces the full match graph. Cross-format de-duplication
// (importing the analysis of an XG match whose canonical hash already exists
// from a GnuBG/MAT import into the existing positions) is NOT reproduced here;
// WriteMatch skips a whole match on a hash hit. That legacy enrichment path is
// a follow-up; for a fresh import the two paths are field-for-field identical,
// which the parity test enforces.
func MapXG(path string) (*MatchGraph, error) {
	imp := xgparser.NewImport(path)
	segments, err := imp.GetFileSegments()
	if err != nil {
		return nil, fmt.Errorf("ingest: xg get file segments: %w", err)
	}

	match, err := xgparser.ParseXG(segments)
	if err != nil {
		return nil, fmt.Errorf("ingest: parse xg: %w", err)
	}

	rawCubeInfo := parseRawCubeInfo(segments)

	graph := &MatchGraph{
		Match: domain.Match{
			Player1Name:   match.Metadata.Player1Name,
			Player2Name:   match.Metadata.Player2Name,
			Event:         match.Metadata.Event,
			Location:      match.Metadata.Location,
			Round:         match.Metadata.Round,
			MatchLength:   match.Metadata.MatchLength,
			MatchDate:     parseMatchDate(match.Metadata.DateTime),
			FilePath:      path,
			GameCount:     len(match.Games),
			MatchHash:     computeMatchHash(match),
			CanonicalHash: computeCanonicalMatchHashFromXG(match),
		},
	}

	for gameIdx := range match.Games {
		game := match.Games[gameIdx]
		gg := GameGraph{
			Game: domain.Game{
				GameNumber:   game.GameNumber,
				InitialScore: game.InitialScore,
				Winner:       game.Winner,
				PointsWon:    game.PointsWon,
				MoveCount:    len(game.Moves),
			},
			Moves: mapGameMoves(gameIdx, &game, match.Metadata.MatchLength, rawCubeInfo),
		}
		graph.Games = append(graph.Games, gg)
	}

	return graph, nil
}

// rawCubeAction holds raw cube action data extracted from XG game-file segments.
type rawCubeAction struct {
	Double  int32
	Take    int32
	ActiveP int32
	CubeB   int32
	Doubled *xgparser.EngineStructDoubleAction
}

// parseRawCubeInfo replays the XG game-file segments to recover complete cube
// action records, keyed by "game_cubeIdx" (1-based game number).
func parseRawCubeInfo(segments []*xgparser.Segment) map[string]*rawCubeAction {
	rawCubeInfo := make(map[string]*rawCubeAction)
	for _, seg := range segments {
		if seg.Type != xgparser.SegmentXGGameFile {
			continue
		}
		records, _ := xgparser.ParseGameFile(seg.Data, -1)
		gameNum := int32(0)
		cubeIdx := 0
		for _, rec := range records {
			switch r := rec.(type) {
			case *xgparser.HeaderGameEntry:
				gameNum = r.GameNumber
				cubeIdx = 0
			case *xgparser.CubeEntry:
				if r.Double != -2 { // Skip initial positions
					key := fmt.Sprintf("%d_%d", gameNum, cubeIdx)
					rawCubeInfo[key] = &rawCubeAction{
						Double:  r.Double,
						Take:    r.Take,
						ActiveP: r.ActiveP,
						CubeB:   r.CubeB,
						Doubled: r.Doubled,
					}
					cubeIdx++
				}
			}
		}
	}
	return rawCubeInfo
}

// mapGameMoves replicates database.importXGGamesAndMoves' per-move loop:
// associating raw cube records with moves, carrying a skipped "No Double"
// comment forward to the next checker move, and flattening each XG move into
// the one or two MoveGraphs it produces.
func mapGameMoves(gameIdx int, game *xgparser.Game, matchLength int32, rawCubeInfo map[string]*rawCubeAction) []MoveGraph {
	var out []MoveGraph

	cubeIdx := 0
	var lastCubeAnalysis *rawCubeAction
	var pendingCubeComment string

	for moveIdx := range game.Moves {
		move := game.Moves[moveIdx] // local copy; we may rewrite its comment

		var rawCube *rawCubeAction
		if move.MoveType == "cube" {
			key := fmt.Sprintf("%d_%d", gameIdx+1, cubeIdx)
			if rc, ok := rawCubeInfo[key]; ok {
				rawCube = rc
				lastCubeAnalysis = rc
			}
			cubeIdx++

			isNoDouble := false
			if rawCube != nil {
				isNoDouble = (rawCube.Double != 1)
			} else if move.CubeMove != nil {
				isNoDouble = (move.CubeMove.CubeAction == 0)
			}
			if isNoDouble && move.Comment != "" {
				pendingCubeComment = move.Comment
			}
		} else if move.MoveType == "checker" {
			rawCube = lastCubeAnalysis
			lastCubeAnalysis = nil

			if pendingCubeComment != "" {
				if move.Comment == "" {
					move.Comment = pendingCubeComment
				} else {
					move.Comment = pendingCubeComment + "\n" + move.Comment
				}
				pendingCubeComment = ""
			}
		}

		out = append(out, mapMove(int32(moveIdx), &move, game, matchLength, rawCube)...)
	}

	return out
}

// mapMove maps a single XG move into MoveGraphs, mirroring
// database.importMoveWithCacheAndRawCube. A checker move yields one MoveGraph; a
// Double/Take or Double/Pass yields two; a skipped/irrelevant cube decision
// yields none.
func mapMove(moveNumber int32, move *xgparser.Move, game *xgparser.Game, matchLength int32, rawCube *rawCubeAction) []MoveGraph {
	switch {
	case move.MoveType == "checker" && move.CheckerMove != nil:
		return mapCheckerMove(moveNumber, move, game, matchLength, rawCube)
	case move.MoveType == "cube" && move.CubeMove != nil:
		return mapCubeMove(moveNumber, move, game, matchLength, rawCube)
	default:
		return nil
	}
}

// mapCheckerMove builds the MoveGraph for a checker decision, combining the
// checker analysis with the preceding cube decision (attached to the checker
// position so it can be inspected) exactly as the legacy two saveAnalysis calls
// would merge.
func mapCheckerMove(moveNumber int32, move *xgparser.Move, game *xgparser.Game, matchLength int32, rawCube *rawCubeAction) []MoveGraph {
	cm := move.CheckerMove
	pos, err := createPositionFromXG(cm.Position, game, matchLength, cm.ActivePlayer)
	if err != nil {
		return nil
	}
	pos.PlayerOnRoll = convertXGPlayerToBlunderDB(cm.ActivePlayer)
	pos.DecisionType = domain.CheckerAction
	pos.Dice = [2]int{int(cm.Dice[0]), int(cm.Dice[1])}

	// The legacy importer saves the checker analysis first, then (if a cube
	// decision preceded this move) merges the cube analysis onto the same
	// position. Emitting them as ordered fragments reproduces that — including
	// the round-then-recompute of equity errors on the second merge.
	var analyses []*domain.PositionAnalysis
	if len(cm.Analysis) > 0 {
		if a := buildCheckerAnalysis(cm.Analysis, &cm.Position, &cm.PlayedMove); a != nil {
			analyses = append(analyses, a)
		}
	}
	if rawCube != nil && rawCube.Doubled != nil {
		if a := buildRawCubeForChecker(rawCube.Doubled); a != nil {
			analyses = append(analyses, a)
		}
	}

	mg := MoveGraph{
		Move: domain.Move{
			MoveNumber:  moveNumber,
			MoveType:    "checker",
			Player:      cm.ActivePlayer,
			Dice:        cm.Dice,
			CheckerMove: convertXGMoveToStringWithHits(cm.PlayedMove, &cm.Position),
		},
		Position: pos,
		Analyses: analyses,
	}
	if move.Comment != "" {
		mg.Comments = []string{move.Comment}
	}
	return []MoveGraph{mg}
}

// mapCubeMove builds the MoveGraph(s) for a cube decision: an explicit
// Double/Take (two positions), a Double/Pass (doubling + pass positions), a
// single cube action, or a counted "No Double" decision.
func mapCubeMove(moveNumber int32, move *xgparser.Move, game *xgparser.Game, matchLength int32, rawCube *rawCubeAction) []MoveGraph {
	cube := move.CubeMove

	isExplicitCubeAction := false
	if rawCube != nil {
		isExplicitCubeAction = (rawCube.Double == 1)
	} else {
		isExplicitCubeAction = (cube.CubeAction != 0)
	}

	if !isExplicitCubeAction {
		return mapNoDoubleMove(moveNumber, move, game, matchLength, rawCube)
	}

	if rawCube != nil && rawCube.Double == 1 && rawCube.Take == 1 {
		return mapDoubleTakeMove(moveNumber, move, game, matchLength)
	}
	return mapSingleCubeMove(moveNumber, move, game, matchLength, rawCube)
}

// mapNoDoubleMove handles an implicit "No Double" decision that XG still counts
// as a cube decision (only when the player actually holds the cube and analysis
// is available).
func mapNoDoubleMove(moveNumber int32, move *xgparser.Move, game *xgparser.Game, matchLength int32, rawCube *rawCubeAction) []MoveGraph {
	cube := move.CubeMove

	hasRawAnalysis := rawCube != nil && rawCube.Doubled != nil
	hasTextAnalysis := cube.Analysis != nil
	if !hasRawAnalysis && !hasTextAnalysis {
		return nil
	}

	// Only count when the active player can actually double.
	if rawCube != nil {
		canDouble := rawCube.CubeB == 0 ||
			(rawCube.ActiveP > 0 && rawCube.CubeB > 0) ||
			(rawCube.ActiveP < 0 && rawCube.CubeB < 0)
		if !canDouble {
			return nil
		}
	} else {
		c := cube.Position.Cube
		ap := cube.ActivePlayer
		canDouble := c == 0 || (ap > 0 && c > 0) || (ap < 0 && c < 0)
		if !canDouble {
			return nil
		}
	}

	pos, err := createPositionFromXG(cube.Position, game, matchLength, cube.ActivePlayer)
	if err != nil {
		return nil
	}
	pos.PlayerOnRoll = convertXGPlayerToBlunderDB(cube.ActivePlayer)
	pos.DecisionType = domain.CubeAction
	pos.Dice = [2]int{0, 0}

	var analysis *domain.PositionAnalysis
	if hasRawAnalysis {
		analysis = buildRawCubeForNoDouble(rawCube.Doubled)
	} else {
		analysis = buildCubeAnalysisFromText(cube.Analysis, "No Double")
	}

	return []MoveGraph{{
		Move: domain.Move{
			MoveNumber: moveNumber,
			MoveType:   "cube",
			Player:     cube.ActivePlayer,
			Dice:       [2]int32{0, 0},
			CubeAction: "No Double",
		},
		Position: pos,
		Analyses: frag(analysis),
	}}
}

// mapDoubleTakeMove handles a Double that was Taken: a doubling-decision
// position and the opponent's take-decision position.
func mapDoubleTakeMove(moveNumber int32, move *xgparser.Move, game *xgparser.Game, matchLength int32) []MoveGraph {
	cube := move.CubeMove

	pos1, err := createPositionFromXG(cube.Position, game, matchLength, cube.ActivePlayer)
	if err != nil {
		return nil
	}
	pos1.PlayerOnRoll = convertXGPlayerToBlunderDB(cube.ActivePlayer)
	pos1.DecisionType = domain.CubeAction
	pos1.Dice = [2]int{0, 0}

	mg1 := MoveGraph{
		Move: domain.Move{
			MoveNumber: moveNumber,
			MoveType:   "cube",
			Player:     cube.ActivePlayer,
			Dice:       [2]int32{0, 0},
			CubeAction: "Double",
		},
		Position: pos1,
		Analyses: frag(buildCubeAnalysisFromText(cube.Analysis, "Double/Take")),
	}

	// Position 2: take decision (opponent, cube still centred at doubled value).
	pos2 := *pos1
	pos2.Cube.Value++
	opponentPlayer := 1 - pos1.PlayerOnRoll
	pos2.PlayerOnRoll = opponentPlayer
	pos2.Cube.Owner = -1

	mg2 := MoveGraph{
		Move: domain.Move{
			MoveNumber: moveNumber,
			MoveType:   "cube",
			Player:     blunderDBPlayerToXG(opponentPlayer),
			Dice:       [2]int32{0, 0},
			CubeAction: "Take",
		},
		Position: &pos2,
		Analyses: frag(buildCubeAnalysisFromText(cube.Analysis, "Take")),
	}
	if move.Comment != "" {
		mg2.Comments = []string{move.Comment}
	}

	return []MoveGraph{mg1, mg2}
}

// mapSingleCubeMove handles a single explicit cube action (e.g. Double/Pass).
// For a Double/Pass it also emits the passer's take/pass decision position.
func mapSingleCubeMove(moveNumber int32, move *xgparser.Move, game *xgparser.Game, matchLength int32, rawCube *rawCubeAction) []MoveGraph {
	cube := move.CubeMove

	pos, err := createPositionFromXG(cube.Position, game, matchLength, cube.ActivePlayer)
	if err != nil {
		return nil
	}
	pos.PlayerOnRoll = convertXGPlayerToBlunderDB(cube.ActivePlayer)
	pos.DecisionType = domain.CubeAction
	pos.Dice = [2]int{0, 0}

	var cubeActionStr string
	if rawCube != nil {
		cubeActionStr = convertRawCubeAction(rawCube.Double, rawCube.Take)
	} else {
		cubeActionStr = convertCubeAction(cube.CubeAction)
	}

	mg := MoveGraph{
		Move: domain.Move{
			MoveNumber: moveNumber,
			MoveType:   "cube",
			Player:     cube.ActivePlayer,
			Dice:       [2]int32{0, 0},
			CubeAction: cubeActionStr,
		},
		Position: pos,
		Analyses: frag(buildCubeAnalysisFromText(cube.Analysis, cubeActionStr)),
	}
	if move.Comment != "" {
		mg.Comments = []string{move.Comment}
	}
	out := []MoveGraph{mg}

	// Double/Pass: emit the passer's decision position too.
	if rawCube != nil && rawCube.Take == 0 {
		pos2 := *pos
		pos2.Cube.Value++
		opponentPlayer := 1 - pos.PlayerOnRoll
		pos2.PlayerOnRoll = opponentPlayer
		pos2.Cube.Owner = -1

		out = append(out, MoveGraph{
			Move: domain.Move{
				MoveNumber: moveNumber,
				MoveType:   "cube",
				Player:     blunderDBPlayerToXG(opponentPlayer),
				Dice:       [2]int32{0, 0},
				CubeAction: "Pass",
			},
			Position: &pos2,
			Analyses: frag(buildCubeAnalysisFromText(cube.Analysis, "Pass")),
		})
	}

	return out
}

// frag wraps a single (possibly nil) analysis fragment as the Analyses slice.
func frag(a *domain.PositionAnalysis) []*domain.PositionAnalysis {
	if a == nil {
		return nil
	}
	return []*domain.PositionAnalysis{a}
}

// blunderDBPlayerToXG converts a blunderDB player index (0/1) to XG encoding (1/-1).
func blunderDBPlayerToXG(blunderDBPlayer int) int32 {
	if blunderDBPlayer == 0 {
		return 1
	}
	return -1
}

// computeMatchHash is the format-specific content hash used for exact
// duplicate detection. Copied from database.ComputeMatchHash; the
// TestXGHashParity test keeps the two in lock-step.
func computeMatchHash(match *xgparser.Match) string {
	var b strings.Builder
	p1 := strings.TrimSpace(strings.ToLower(match.Metadata.Player1Name))
	p2 := strings.TrimSpace(strings.ToLower(match.Metadata.Player2Name))
	b.WriteString(fmt.Sprintf("meta:%s|%s|%d|", p1, p2, match.Metadata.MatchLength))

	for gameIdx, game := range match.Games {
		b.WriteString(fmt.Sprintf("g%d:%d,%d,%d,%d|",
			gameIdx, game.InitialScore[0], game.InitialScore[1], game.Winner, game.PointsWon))
		for moveIdx, move := range game.Moves {
			b.WriteString(fmt.Sprintf("m%d:%s,", moveIdx, move.MoveType))
			if move.CheckerMove != nil {
				b.WriteString(fmt.Sprintf("d%d%d,p%v|",
					move.CheckerMove.Dice[0], move.CheckerMove.Dice[1], move.CheckerMove.PlayedMove))
			}
			if move.CubeMove != nil {
				b.WriteString(fmt.Sprintf("c%d|", move.CubeMove.CubeAction))
			}
		}
	}

	hash := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(hash[:])
}

// maxCanonicalDicePerGame bounds the dice included in the canonical hash so it
// is identical across export formats. Mirrors database.maxCanonicalDicePerGame.
const maxCanonicalDicePerGame = 10

// computeCanonicalMatchHashFromXG is the format-independent hash for
// cross-format duplicate detection. Copied from
// database.ComputeCanonicalMatchHashFromXG; TestXGHashParity guards equality.
func computeCanonicalMatchHashFromXG(match *xgparser.Match) string {
	var b strings.Builder
	p1 := strings.TrimSpace(strings.ToLower(match.Metadata.Player1Name))
	p2 := strings.TrimSpace(strings.ToLower(match.Metadata.Player2Name))
	if p1 > p2 {
		p1, p2 = p2, p1
	}
	b.WriteString(fmt.Sprintf("canonical2:%s|%s|%d|%d|", p1, p2, match.Metadata.MatchLength, len(match.Games)))

	for gameIdx, game := range match.Games {
		b.WriteString(fmt.Sprintf("g%d|", gameIdx))
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
				b.WriteString(fmt.Sprintf("d%d%d|", d1, d2))
				diceCount++
			}
		}
	}

	hash := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(hash[:])
}

// XGImporter implements Importer for eXtreme Gammon .xg files. It needs a
// seekable file, so it requires Source.Path (the daemon spools the upload).
type XGImporter struct{ S storage.Storage }

func (im XGImporter) Import(ctx context.Context, scope string, src Source, prog func(Progress)) (Summary, error) {
	if src.Path == "" {
		return Summary{}, fmt.Errorf("ingest: xg import requires a file path")
	}

	graph, err := MapXG(src.Path)
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

	sum := Summary{
		SavedPositions: res.SavedPositions,
		Matches:        1,
		MatchID:        res.MatchID,
	}
	if res.Skipped {
		sum.SkippedDuplicates = 1
		sum.SavedPositions = 0
	}
	return sum, nil
}
