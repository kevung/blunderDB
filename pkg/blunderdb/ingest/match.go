package ingest

import (
	"context"
	"errors"

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
// position's analysis. Analyses holds the analysis fragments to apply to the
// position, in order: each is merged into whatever is already stored for the
// (deduplicated) position, mirroring the sequence of saveAnalysisInTx calls the
// legacy importer makes (e.g. a checker move with a preceding cube decision
// contributes a checker fragment then a cube fragment). It is empty for a move
// with no engine analysis. Comments are the free-text notes attached to the
// position in the source file (one per entry).
type MoveGraph struct {
	Move     domain.Move
	Position *domain.Position
	Analyses []*domain.PositionAnalysis
	Comments []string
}

// WriteResult summarises a WriteMatch call.
type WriteResult struct {
	MatchID        int64
	Skipped        bool // true when an exact same-format duplicate was found (nothing written)
	Enriched       bool // true when a cross-format (canonical) duplicate was enriched in place
	SavedPositions int
}

// WriteMatch persists a MatchGraph through tx. It is the single Storage-based
// sink shared by every import format. The whole graph is written inside the
// caller-provided transaction, so an import of several matches is atomic and
// cancellable as a unit.
//
// Duplicate detection has two levels, mirroring the legacy importers:
//   - Exact same-format duplicate (MatchHash already present): nothing is
//     written, WriteResult.Skipped is true.
//   - Cross-format duplicate (CanonicalHash present from another format, e.g.
//     the same match imported from XG and then GnuBG): the match/game/move rows
//     are NOT recreated, but the graph's positions and analyses are still
//     written — deduplicating by Zobrist they land on the existing positions,
//     and mergeAnalysis combines this format's analysis with what is already
//     stored (cross-engine cube analyses, extra checker moves). Enriched is
//     true.
//
// Positions dedup independently by Zobrist hash inside PositionStore.Save.
func WriteMatch(ctx context.Context, tx storage.Tx, scope string, g *MatchGraph, prog func(Progress)) (WriteResult, error) {
	var res WriteResult

	// Exact same-format duplicate → skip entirely.
	if g.Match.MatchHash != "" {
		id, found, err := tx.Matches().FindByHash(ctx, scope, g.Match.MatchHash, "")
		if err != nil {
			return res, err
		}
		if found {
			return WriteResult{MatchID: id, Skipped: true}, nil
		}
	}

	// Cross-format (canonical) duplicate → enrich the existing match's positions.
	enrich := false
	var matchID int64
	if g.Match.CanonicalHash != "" {
		id, found, err := tx.Matches().FindByHash(ctx, scope, "", g.Match.CanonicalHash)
		if err != nil {
			return res, err
		}
		if found {
			enrich = true
			matchID = id
		}
	}

	if !enrich {
		id, err := tx.Matches().Save(ctx, scope, &g.Match)
		if err != nil {
			return res, err
		}
		matchID = id
	}
	res.MatchID = matchID
	res.Enriched = enrich

	counter := Progress{Matches: 1}
	for gi := range g.Games {
		gg := &g.Games[gi]
		var gameID int64
		if !enrich {
			gg.Game.MatchID = matchID
			id, err := tx.Matches().CreateGame(ctx, scope, &gg.Game)
			if err != nil {
				return res, err
			}
			gameID = id
		}
		counter.Games++

		for mi := range gg.Moves {
			mg := &gg.Moves[mi]
			if mg.Position != nil {
				posID, err := savePositionWithAnalyses(ctx, tx, scope, mg.Position, mg.Analyses, mg.Comments)
				if err != nil {
					return res, err
				}
				mg.Move.PositionID = posID
				res.SavedPositions++
				counter.Positions++
			}
			if !enrich {
				mg.Move.GameID = gameID
				if _, err := tx.Matches().CreateMove(ctx, scope, &mg.Move); err != nil {
					return res, err
				}
			}
			if prog != nil {
				prog(counter)
			}
		}
	}
	return res, nil
}

// savePositionWithAnalyses saves pos (deduplicated by Zobrist) and applies each
// analysis fragment in order via load-merge-save, then adds the comments. It is
// shared by WriteMatch (per move) and the single-position importers.
//
// AnalysisStore.Save replaces, so each fragment is merged into whatever is
// already stored for the position before saving — reproducing the legacy
// sequence of saveAnalysisInTx calls, including its round-then-recompute of
// equity errors across successive merges onto one position.
func savePositionWithAnalyses(ctx context.Context, tx storage.Tx, scope string, pos *domain.Position, analyses []*domain.PositionAnalysis, comments []string) (int64, error) {
	posID, err := tx.Positions().Save(ctx, scope, pos)
	if err != nil {
		return 0, err
	}
	for _, frag := range analyses {
		if frag == nil {
			continue
		}
		var existing *domain.PositionAnalysis
		switch cur, err := tx.Analyses().Load(ctx, scope, posID); {
		case err == nil:
			existing = cur
		case errors.Is(err, storage.ErrNotFound):
			// no analysis yet
		default:
			return posID, err
		}
		merged := mergeAnalysis(existing, *frag)
		if err := tx.Analyses().Save(ctx, scope, posID, &merged); err != nil {
			return posID, err
		}
	}
	for _, c := range comments {
		if _, err := tx.Comments().Add(ctx, scope, posID, c); err != nil {
			return posID, err
		}
	}
	return posID, nil
}
