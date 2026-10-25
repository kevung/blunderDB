package storage

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// MergeSaver is the capability of a store whose backend folds a saved
// analysis into the stored one itself, atomically — the remote backend, whose
// daemon does it in one call (/v1/analyses.merge) where a Merge would take
// two.
type MergeSaver interface {
	SaveMerged(ctx context.Context, scope string, positionID int64, analysis domain.PositionAnalysis) error
}

// SaveMergedAnalysis folds analysis into what positionID already carries
// (MergeSavedAnalysis) and writes the result: the save a user's analysis goes
// through, from the GUI, the CLI or the daemon alike. A MergeSaver folds it
// on its side; otherwise the fold runs inside Merge, whose read and write
// share one transaction. Only a store with neither reads, then saves.
func SaveMergedAnalysis(ctx context.Context, store AnalysisStore, scope string, positionID int64, analysis domain.PositionAnalysis) error {
	if ms, ok := store.(MergeSaver); ok {
		return ms.SaveMerged(ctx, scope, positionID, analysis)
	}
	analysis.PositionID = int(positionID)
	analysis.LastModifiedDate = time.Now()
	_, err := store.Merge(ctx, scope, positionID, nil, func(existing *domain.PositionAnalysis) *domain.PositionAnalysis {
		merged := MergeSavedAnalysis(existing, analysis)
		return &merged
	})
	if !errors.Is(err, ErrUnsupported) {
		return err
	}
	var existing *domain.PositionAnalysis
	stored, err := store.Load(ctx, scope, positionID)
	switch {
	case err == nil:
		existing = stored
	case !errors.Is(err, ErrNotFound):
		return err
	}
	merged := MergeSavedAnalysis(existing, analysis)
	return store.Save(ctx, scope, positionID, &merged)
}

// WithPlayedHeader is the response header a daemon sets on analyses.load when
// it understands withPlayed: one that does not would answer the analysis
// without the plays, and nothing in the body tells the two apart.
const WithPlayedHeader = "X-Blunderdb-With-Played"

// PlayedLoader is the capability of a store whose backend joins the played
// moves to an analysis itself — the remote backend, whose daemon reads the
// move table it does not serve (/v1/analyses.load with withPlayed).
type PlayedLoader interface {
	LoadWithPlayed(ctx context.Context, scope string, positionID int64) (*domain.PositionAnalysis, error)
}

// LoadAnalysisWithPlayed is the analysis of positionID as a reader sees it:
// the stored one, its played actions supplemented by what the matches played
// at the position (MergePlayedActions), its cube analyses XG first. It answers
// ErrNotFound when the position has none. A PlayedLoader joins on its side.
func LoadAnalysisWithPlayed(ctx context.Context, s Stores, scope string, positionID int64) (*domain.PositionAnalysis, error) {
	if pl, ok := s.Analyses().(PlayedLoader); ok {
		return pl.LoadWithPlayed(ctx, scope, positionID)
	}
	loaded, err := s.Analyses().Load(ctx, scope, positionID)
	if err != nil {
		return nil, err
	}
	analysis := *loaded
	byPosition, err := s.Matches().MovesByPositions(ctx, scope, []int64{positionID})
	if err != nil {
		return nil, err
	}
	var checkerMoves, cubeActions []string
	for _, mv := range byPosition[positionID] {
		checkerMoves = append(checkerMoves, mv.CheckerMove)
		cubeActions = append(cubeActions, mv.CubeAction)
	}
	MergePlayedActions(&analysis, checkerMoves, cubeActions)
	if len(analysis.AllCubeAnalyses) > 0 {
		engine.SortCubeAnalysesByEngine(analysis.AllCubeAnalyses)
	}
	return &analysis, nil
}

// MergeSavedAnalysis folds analysis into existing, the analysis the position
// already carries (nil when none): the candidate moves and cube analyses of
// both, keyed by notation and by engine, the union of what was played, the
// rollouts of both, the creation date of the stored one. The moves are ranked
// by equity (engine.RankCheckerMoves).
func MergeSavedAnalysis(existing *domain.PositionAnalysis, analysis domain.PositionAnalysis) domain.PositionAnalysis {
	if existing == nil {
		if analysis.CreationDate.IsZero() {
			analysis.CreationDate = time.Now()
		}
		if analysis.PlayedMove != "" && len(analysis.PlayedMoves) == 0 {
			analysis.PlayedMoves = []string{analysis.PlayedMove}
			analysis.PlayedMove = ""
		}
		if analysis.PlayedCubeAction != "" && len(analysis.PlayedCubeActions) == 0 {
			analysis.PlayedCubeActions = []string{analysis.PlayedCubeAction}
			analysis.PlayedCubeAction = ""
		}
		if analysis.CheckerAnalysis != nil {
			engine.RankCheckerMoves(analysis.CheckerAnalysis.Moves)
		}
		return analysis
	}

	analysis.CreationDate = existing.CreationDate

	if existing.CheckerAnalysis != nil && analysis.CheckerAnalysis != nil {
		analysis.CheckerAnalysis.Moves = engine.MergeCheckerMoves(
			existing.CheckerAnalysis.Moves, analysis.CheckerAnalysis.Moves)
	} else if existing.CheckerAnalysis != nil {
		analysis.CheckerAnalysis = existing.CheckerAnalysis
	}

	// Every engine's cube analysis is kept (ADR-0013). DoublingCubeAnalysis
	// stays the PRIMARY one the scalar columns read; AllCubeAnalyses carries
	// the whole set.
	analysis.DoublingCubeAnalysis, analysis.AllCubeAnalyses = engine.MergeCubeAnalyses(
		existing.DoublingCubeAnalysis, existing.AllCubeAnalyses,
		analysis.DoublingCubeAnalysis, analysis.AllCubeAnalyses)

	// The old single fields still count as a play when the array is empty.
	analysis.PlayedMoves = engine.UnionPlayed(
		playedOrSingle(existing.PlayedMoves, existing.PlayedMove),
		playedOrSingle(analysis.PlayedMoves, analysis.PlayedMove))
	analysis.PlayedCubeActions = engine.UnionCubeActions(
		playedOrSingle(existing.PlayedCubeActions, existing.PlayedCubeAction),
		playedOrSingle(analysis.PlayedCubeActions, analysis.PlayedCubeAction))
	analysis.PlayedMove = ""
	analysis.PlayedCubeAction = ""

	// Rollouts are their own analyses: a caller that does not know them
	// (an import, the frontend saving what it edited) must not drop them.
	analysis.Rollouts = domain.MergeRollouts(existing.Rollouts, analysis.Rollouts)

	if analysis.CheckerAnalysis != nil {
		engine.RankCheckerMoves(analysis.CheckerAnalysis.Moves)
	}
	return analysis
}

// playedOrSingle is played, or the legacy single field when played is empty.
func playedOrSingle(played []string, single string) []string {
	if single != "" && len(played) == 0 {
		return []string{single}
	}
	return played
}

// MergePlayedActions folds the moves recorded against a position into its
// analysis's played actions: the union of what the analysis stored and what
// the matches played there, normalised, sorted, the legacy single fields set
// to the first. An empty entry is not a play.
func MergePlayedActions(analysis *domain.PositionAnalysis, checkerMoves, cubeActions []string) {
	existingMoves := make(map[string]bool)
	for _, m := range analysis.PlayedMoves {
		existingMoves[engine.NormalizeMove(m)] = true
	}
	// Backward compatibility: include old single PlayedMove field
	if analysis.PlayedMove != "" {
		existingMoves[engine.NormalizeMove(analysis.PlayedMove)] = true
	}
	for _, m := range checkerMoves {
		if m != "" {
			existingMoves[engine.NormalizeMove(m)] = true
		}
	}
	analysis.PlayedMoves = make([]string, 0, len(existingMoves))
	for m := range existingMoves {
		analysis.PlayedMoves = append(analysis.PlayedMoves, m)
	}
	sort.Strings(analysis.PlayedMoves)

	// Backward compatibility: include old single PlayedCubeAction field
	analysis.PlayedCubeActions = engine.UnionCubeActions(
		append(append([]string(nil), analysis.PlayedCubeActions...), analysis.PlayedCubeAction), cubeActions)

	// The old single fields, read by older frontends, hold the first.
	if len(analysis.PlayedMoves) > 0 {
		analysis.PlayedMove = analysis.PlayedMoves[0]
	}
	if len(analysis.PlayedCubeActions) > 0 {
		analysis.PlayedCubeAction = analysis.PlayedCubeActions[0]
	}
}
