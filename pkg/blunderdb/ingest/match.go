package ingest

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// MatchGraph is the backend-independent, fully-parsed representation of a match
// ready to be written through Storage. Format parsers (XG, GnuBG, BGF, …) map
// their output into a MatchGraph; WriteMatch persists it.
type MatchGraph struct {
	Match domain.Match
	Games []GameGraph
}

// GameGraph is one game with its ordered moves.
type GameGraph struct {
	Game  domain.Game
	Moves []MoveGraph
}

// MoveGraph is one move plus the position it was played from and that
// position's analysis (either may be nil, e.g. a cube move with no checker
// position, or a move without engine analysis).
type MoveGraph struct {
	Move     domain.Move
	Position *domain.Position
	Analysis *domain.PositionAnalysis
}

// WriteResult summarises a WriteMatch call.
type WriteResult struct {
	MatchID        int64
	Skipped        bool // true when the match was a duplicate (not written)
	SavedPositions int
}

// WriteMatch persists a MatchGraph through tx. It is the single Storage-based
// sink shared by every import format. The whole graph is written inside the
// caller-provided transaction, so an import of several matches is atomic and
// cancellable as a unit.
//
// Duplicate detection: if the match's MatchHash/CanonicalHash already exists,
// the match is not re-written and WriteResult.Skipped is true (with the
// existing match id). Positions dedup independently by Zobrist hash inside
// PositionStore.Save.
func WriteMatch(ctx context.Context, tx storage.Tx, scope string, g *MatchGraph, prog func(Progress)) (WriteResult, error) {
	var res WriteResult

	if g.Match.MatchHash != "" || g.Match.CanonicalHash != "" {
		id, found, err := tx.Matches().FindByHash(ctx, scope, g.Match.MatchHash, g.Match.CanonicalHash)
		if err != nil {
			return res, err
		}
		if found {
			return WriteResult{MatchID: id, Skipped: true}, nil
		}
	}

	matchID, err := tx.Matches().Save(ctx, scope, &g.Match)
	if err != nil {
		return res, err
	}
	res.MatchID = matchID

	counter := Progress{Matches: 1}
	for gi := range g.Games {
		gg := &g.Games[gi]
		gg.Game.MatchID = matchID
		gameID, err := tx.Matches().CreateGame(ctx, scope, &gg.Game)
		if err != nil {
			return res, err
		}
		counter.Games++

		for mi := range gg.Moves {
			mg := &gg.Moves[mi]
			if mg.Position != nil {
				posID, err := tx.Positions().Save(ctx, scope, mg.Position)
				if err != nil {
					return res, err
				}
				mg.Move.PositionID = posID
				res.SavedPositions++
				counter.Positions++
				if mg.Analysis != nil {
					if err := tx.Analyses().Save(ctx, scope, posID, mg.Analysis); err != nil {
						return res, err
					}
				}
			}
			mg.Move.GameID = gameID
			if _, err := tx.Matches().CreateMove(ctx, scope, &mg.Move); err != nil {
				return res, err
			}
			if prog != nil {
				prog(counter)
			}
		}
	}
	return res, nil
}
