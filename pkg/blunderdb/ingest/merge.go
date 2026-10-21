package ingest

import (
	"reflect"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// Analysis merge/normalisation: the incoming analysis is merged into the stored
// one (combining checker moves, keeping per-engine cube analyses, unioning
// played moves/cube actions) and move ordering re-normalised. AnalysisStore.Save
// only *replaces*, so WriteMatch loads and merges here first;
// mergeAnalysis(nil, incoming) is the insert-path normalisation. The XG parity
// test holds it byte-faithful.

// mergeCheckerMoves merges two sets of checker moves keyed by move string,
// preferring the higher-depth analysis on conflict, then ranks them deeper
// analysis first (engine.RankCheckerMovesDepthFirst).
func mergeCheckerMoves(existing, incoming []domain.CheckerMove) []domain.CheckerMove {
	result := engine.MergeCheckerMoves(existing, incoming)
	engine.RankCheckerMovesDepthFirst(result)
	return result
}

// sortCheckerMovesByEquity ranks an analysis' checker moves deeper analysis
// first (engine.RankCheckerMovesDepthFirst): the final normalisation of both
// the insert and update paths.
func sortCheckerMovesByEquity(a *domain.PositionAnalysis) {
	if a.CheckerAnalysis != nil {
		engine.RankCheckerMovesDepthFirst(a.CheckerAnalysis.Moves)
	}
}

// mergeAnalysis combines incoming into existing (which may be nil when no
// analysis is stored yet for the position) and returns the analysis to persist.
// AnalysisStore.Save then encodes and derives the scalar columns.
func mergeAnalysis(existing *domain.PositionAnalysis, incoming domain.PositionAnalysis) domain.PositionAnalysis {
	a := incoming
	a.LastModifiedDate = time.Now()

	if existing != nil {
		a.CreationDate = existing.CreationDate

		// Merge checker analysis.
		if existing.CheckerAnalysis != nil && a.CheckerAnalysis != nil {
			a.CheckerAnalysis = &domain.CheckerAnalysis{
				Moves: mergeCheckerMoves(existing.CheckerAnalysis.Moves, a.CheckerAnalysis.Moves),
			}
		} else if existing.CheckerAnalysis != nil && a.CheckerAnalysis == nil {
			a.CheckerAnalysis = existing.CheckerAnalysis
		}

		// Merge doubling cube analysis, keeping all engine analyses.
		if existing.DoublingCubeAnalysis != nil && a.DoublingCubeAnalysis != nil {
			existingEngine := existing.DoublingCubeAnalysis.AnalysisEngine
			incomingEngine := a.DoublingCubeAnalysis.AnalysisEngine

			if existingEngine != incomingEngine && existingEngine != "" && incomingEngine != "" {
				allCube := make([]domain.DoublingCubeAnalysis, 0)
				if len(existing.AllCubeAnalyses) > 0 {
					allCube = append(allCube, existing.AllCubeAnalyses...)
				} else {
					allCube = append(allCube, *existing.DoublingCubeAnalysis)
				}
				hasIncoming := false
				for _, ca := range allCube {
					if ca.AnalysisEngine == incomingEngine {
						hasIncoming = true
						break
					}
				}
				if !hasIncoming {
					allCube = append(allCube, *a.DoublingCubeAnalysis)
				}
				engine.SortCubeAnalysesByEngine(allCube)
				a.AllCubeAnalyses = allCube
			} else {
				if len(existing.AllCubeAnalyses) > 0 {
					a.AllCubeAnalyses = existing.AllCubeAnalyses
				}
			}
		} else if existing.DoublingCubeAnalysis != nil && a.DoublingCubeAnalysis == nil {
			a.DoublingCubeAnalysis = existing.DoublingCubeAnalysis
			a.AllCubeAnalyses = existing.AllCubeAnalyses
		}

		// Merge played moves.
		existingPlayedMoves := existing.PlayedMoves
		if existing.PlayedMove != "" && len(existingPlayedMoves) == 0 {
			existingPlayedMoves = []string{existing.PlayedMove}
		}
		incomingPlayedMoves := a.PlayedMoves
		if a.PlayedMove != "" && len(incomingPlayedMoves) == 0 {
			incomingPlayedMoves = []string{a.PlayedMove}
		}
		a.PlayedMoves = engine.UnionPlayed(existingPlayedMoves, incomingPlayedMoves)

		// Merge played cube actions.
		existingCubeActions := existing.PlayedCubeActions
		if existing.PlayedCubeAction != "" && len(existingCubeActions) == 0 {
			existingCubeActions = []string{existing.PlayedCubeAction}
		}
		incomingCubeActions := a.PlayedCubeActions
		if a.PlayedCubeAction != "" && len(incomingCubeActions) == 0 {
			incomingCubeActions = []string{a.PlayedCubeAction}
		}
		a.PlayedCubeActions = engine.UnionPlayed(existingCubeActions, incomingCubeActions)

		// Rollouts are their own analyses: a caller that does not know them
		// (an import, the frontend saving what it edited) must not drop them.
		a.Rollouts = domain.MergeRollouts(existing.Rollouts, a.Rollouts)

		a.PlayedMove = ""
		a.PlayedCubeAction = ""

		sortCheckerMovesByEquity(&a)
		return a
	}

	// Insert path: no existing analysis.
	if a.CreationDate.IsZero() {
		a.CreationDate = time.Now()
	}
	if a.PlayedMove != "" && len(a.PlayedMoves) == 0 {
		a.PlayedMoves = []string{a.PlayedMove}
		a.PlayedMove = ""
	}
	if a.PlayedCubeAction != "" && len(a.PlayedCubeActions) == 0 {
		a.PlayedCubeActions = []string{a.PlayedCubeAction}
		a.PlayedCubeAction = ""
	}
	sortCheckerMovesByEquity(&a)
	return a
}

// deepenAnalysis folds incoming into existing under the rule of a re-imported
// match: an analysis already stored is replaced only by a strictly deeper one
// (domain.AnalysisDepthRank), and on a tie the stored one stays — the
// opposite tie-break of mergeCheckerMoves, which serves the cross-format
// enrichment where the incoming engine is another one. What existing lacks
// (a candidate move, a cube analysis, a rollout) is added. It returns
// existing itself, unchanged, when incoming brings nothing, so the caller can
// tell a deepened position from an untouched one by pointer.
func deepenAnalysis(existing *domain.PositionAnalysis, incoming domain.PositionAnalysis) *domain.PositionAnalysis {
	if existing == nil {
		a := mergeAnalysis(nil, incoming)
		return &a
	}
	changed := false
	a := *existing

	if incoming.CheckerAnalysis != nil && len(incoming.CheckerAnalysis.Moves) > 0 {
		var moves []domain.CheckerMove
		if existing.CheckerAnalysis != nil {
			moves = append(moves, existing.CheckerAnalysis.Moves...)
		}
		for _, m := range incoming.CheckerAnalysis.Moves {
			at := -1
			for i := range moves {
				if moves[i].Move == m.Move {
					at = i
					break
				}
			}
			switch {
			case at < 0:
				moves = append(moves, m)
				changed = true
			case domain.AnalysisDepthRank(m.AnalysisDepth) > domain.AnalysisDepthRank(moves[at].AnalysisDepth):
				moves[at] = m
				changed = true
			}
		}
		if changed {
			a.CheckerAnalysis = &domain.CheckerAnalysis{Moves: moves}
		}
	}

	if in := incoming.DoublingCubeAnalysis; in != nil {
		cur := existing.DoublingCubeAnalysis
		switch {
		case cur == nil:
			c := *in
			a.DoublingCubeAnalysis = &c
			changed = true
		case cur.AnalysisEngine == in.AnalysisEngine || cur.AnalysisEngine == "" || in.AnalysisEngine == "":
			if domain.AnalysisDepthRank(in.AnalysisDepth) > domain.AnalysisDepthRank(cur.AnalysisDepth) {
				c := *in
				a.DoublingCubeAnalysis = &c
				if len(existing.AllCubeAnalyses) > 0 {
					all := append([]domain.DoublingCubeAnalysis(nil), existing.AllCubeAnalyses...)
					for i := range all {
						if all[i].AnalysisEngine == cur.AnalysisEngine {
							all[i] = c
						}
					}
					a.AllCubeAnalyses = all
				}
				changed = true
			}
		default:
			all := existing.AllCubeAnalyses
			if len(all) == 0 {
				all = []domain.DoublingCubeAnalysis{*cur}
			}
			all = append([]domain.DoublingCubeAnalysis(nil), all...)
			at := -1
			for i := range all {
				if all[i].AnalysisEngine == in.AnalysisEngine {
					at = i
					break
				}
			}
			switch {
			case at < 0:
				all = append(all, *in)
				changed = true
			case domain.AnalysisDepthRank(in.AnalysisDepth) > domain.AnalysisDepthRank(all[at].AnalysisDepth):
				all[at] = *in
				changed = true
			}
			if changed {
				engine.SortCubeAnalysesByEngine(all)
				a.AllCubeAnalyses = all
			}
		}
	}

	if len(incoming.Rollouts) > 0 {
		// Existing is passed as MergeRollouts' incoming side so that it wins
		// a tie in games, as every stored analysis does under this rule; a
		// new signature or a longer series of a known one still comes in.
		if merged := domain.MergeRollouts(incoming.Rollouts, existing.Rollouts); !reflect.DeepEqual(merged, existing.Rollouts) {
			a.Rollouts = merged
			changed = true
		}
	}

	if !changed {
		return existing
	}
	a.LastModifiedDate = time.Now()
	sortCheckerMovesByEquity(&a)
	return &a
}
