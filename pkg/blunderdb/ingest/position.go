package ingest

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/xgparser/xgparser"
)

// PositionGraph is a single stored position with its analysis fragments — the
// unit of a single-position import (XGP, and later BGF text). Unlike a
// MatchGraph it has no game/move context.
type PositionGraph struct {
	Position *domain.Position
	Analyses []*domain.PositionAnalysis
}

// MapXGPPosition parses an eXtreme Gammon .xgp single-position file into one or
// two PositionGraphs (the decision position, plus the following checker
// position when the file also carries it), reusing the XG match mappers. It
// re-implements the mapping half of database.ImportXGPPosition.
func MapXGPPosition(path string) ([]PositionGraph, error) {
	match, err := xgparser.ParseXGFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("ingest: parse xgp file: %w", err)
	}
	if len(match.Games) == 0 || len(match.Games[0].Moves) == 0 {
		return nil, fmt.Errorf("ingest: xgp file contains no position data")
	}

	game := &match.Games[0]
	move := &game.Moves[0]
	matchLen := match.Metadata.MatchLength

	var out []PositionGraph

	switch {
	case move.MoveType == "checker" && move.CheckerMove != nil:
		cm := move.CheckerMove
		pos, err := createPositionFromXG(cm.Position, game, matchLen, cm.ActivePlayer)
		if err != nil {
			return nil, fmt.Errorf("ingest: create xgp position: %w", err)
		}
		pos.PlayerOnRoll = convertXGPlayerToBlunderDB(cm.ActivePlayer)
		pos.DecisionType = domain.CheckerAction
		pos.Dice = [2]int{int(cm.Dice[0]), int(cm.Dice[1])}
		var an []*domain.PositionAnalysis
		if len(cm.Analysis) > 0 {
			if a := buildCheckerAnalysis(cm.Analysis, &cm.Position, &cm.PlayedMove); a != nil {
				an = append(an, a)
			}
		}
		out = append(out, PositionGraph{Position: pos, Analyses: an})

	case move.MoveType == "cube" && move.CubeMove != nil:
		cube := move.CubeMove
		pos, err := createPositionFromXG(cube.Position, game, matchLen, cube.ActivePlayer)
		if err != nil {
			return nil, fmt.Errorf("ingest: create xgp position: %w", err)
		}
		pos.PlayerOnRoll = convertXGPlayerToBlunderDB(cube.ActivePlayer)
		pos.DecisionType = domain.CubeAction
		pos.Dice = [2]int{0, 0}
		var an []*domain.PositionAnalysis
		if cube.Analysis != nil {
			if a := buildCubeAnalysisFromText(cube.Analysis, convertCubeAction(cube.CubeAction)); a != nil {
				an = append(an, a)
			}
		}
		out = append(out, PositionGraph{Position: pos, Analyses: an})

	default:
		return nil, fmt.Errorf("ingest: xgp file contains unsupported move type: %s", move.MoveType)
	}

	// A following checker move (e.g. after a cube decision) is stored as its
	// own position, mirroring database.ImportXGPPosition.
	if len(game.Moves) > 1 {
		sm := &game.Moves[1]
		if sm.MoveType == "checker" && sm.CheckerMove != nil && len(sm.CheckerMove.Analysis) > 0 {
			cm := sm.CheckerMove
			if checkerPos, err := createPositionFromXG(cm.Position, game, matchLen, cm.ActivePlayer); err == nil {
				checkerPos.PlayerOnRoll = convertXGPlayerToBlunderDB(cm.ActivePlayer)
				checkerPos.DecisionType = domain.CheckerAction
				checkerPos.Dice = [2]int{int(cm.Dice[0]), int(cm.Dice[1])}
				var an []*domain.PositionAnalysis
				if a := buildCheckerAnalysis(cm.Analysis, &cm.Position, &cm.PlayedMove); a != nil {
					an = append(an, a)
				}
				out = append(out, PositionGraph{Position: checkerPos, Analyses: an})
			}
		}
	}

	return out, nil
}

// PositionImporter implements Importer for single-position files. It dispatches
// on the upload's extension: .xgp uses the XG parser. (BGBlitz .txt position
// import is deferred until a fixture exists to gate it.)
type PositionImporter struct{ S storage.Storage }

func (im PositionImporter) Import(ctx context.Context, scope string, src Source, prog func(Progress)) (Summary, error) {
	if src.Path == "" {
		return Summary{}, fmt.Errorf("ingest: position import requires a file path")
	}

	var graphs []PositionGraph
	var err error
	switch strings.ToLower(filepath.Ext(src.Path)) {
	case ".xgp":
		graphs, err = MapXGPPosition(src.Path)
	default:
		return Summary{}, fmt.Errorf("ingest: unsupported position file format: %s", filepath.Ext(src.Path))
	}
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

	var sum Summary
	for i := range graphs {
		g := &graphs[i]
		if g.Position == nil {
			continue
		}
		if _, err := savePositionWithAnalyses(ctx, tx, scope, g.Position, g.Analyses, nil); err != nil {
			return sum, err
		}
		sum.SavedPositions++
		if prog != nil {
			prog(Progress{Positions: sum.SavedPositions})
		}
	}

	if err := ctx.Err(); err != nil {
		return sum, err
	}
	if err := tx.Commit(); err != nil {
		return sum, err
	}
	committed = true
	return sum, nil
}
