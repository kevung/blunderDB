package ingest

import (
	"sort"
	"strings"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// This file ports the analysis merge/normalisation semantics of the legacy
// database.saveAnalysisInTx into the backend-agnostic ingest path. The legacy
// helper loads the existing analysis for a position, merges the incoming one
// into it (combining checker moves, keeping per-engine cube analyses, unioning
// played moves/cube actions) and re-normalises move ordering before storage.
//
// AnalysisStore.Save only *replaces*, so WriteMatch performs the load+merge
// here before calling Save. mergeAnalysis(nil, incoming) reproduces the
// insert-path normalisation; mergeAnalysis(existing, incoming) reproduces the
// update-path merge. Keeping this byte-faithful to the legacy code is what the
// XG parity test verifies.

// enginePriority returns a sort priority for analysis engines (XG first).
func enginePriority(eng string) int {
	switch strings.ToLower(eng) {
	case "xg":
		return 0
	case "gnubg":
		return 1
	default:
		return 2
	}
}

// sortCubeAnalysesByEngine sorts cube analyses so XG comes first, then GnuBG.
func sortCubeAnalysesByEngine(analyses []domain.DoublingCubeAnalysis) {
	sort.SliceStable(analyses, func(i, j int) bool {
		return enginePriority(analyses[i].AnalysisEngine) < enginePriority(analyses[j].AnalysisEngine)
	})
}

// mergeCheckerMoves merges two sets of checker moves keyed by move string,
// preferring the higher-depth analysis on conflict, then re-ranks by equity
// (XG preferred as tiebreaker) and recomputes per-move equity errors.
func mergeCheckerMoves(existing, incoming []domain.CheckerMove) []domain.CheckerMove {
	moveMap := make(map[string]domain.CheckerMove)
	for _, m := range existing {
		moveMap[m.Move] = m
	}
	for _, m := range incoming {
		if existingMove, exists := moveMap[m.Move]; exists {
			if m.AnalysisDepth >= existingMove.AnalysisDepth {
				moveMap[m.Move] = m
			}
		} else {
			moveMap[m.Move] = m
		}
	}

	result := make([]domain.CheckerMove, 0, len(moveMap))
	for _, m := range moveMap {
		result = append(result, m)
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Equity != result[j].Equity {
			return result[i].Equity > result[j].Equity
		}
		return enginePriority(result[i].AnalysisEngine) < enginePriority(result[j].AnalysisEngine)
	})

	if len(result) > 0 {
		bestEquity := result[0].Equity
		for i := range result {
			result[i].Index = i
			if i == 0 {
				result[i].EquityError = nil
			} else {
				diff := bestEquity - result[i].Equity
				result[i].EquityError = &diff
			}
		}
	}

	return result
}

// mergePlayedMoves unions played moves/cube actions, normalising and sorting.
func mergePlayedMoves(existing, incoming []string) []string {
	moveSet := make(map[string]bool)
	for _, m := range existing {
		if m != "" {
			moveSet[engine.NormalizeMove(m)] = true
		}
	}
	for _, m := range incoming {
		if m != "" {
			moveSet[engine.NormalizeMove(m)] = true
		}
	}
	result := make([]string, 0, len(moveSet))
	for m := range moveSet {
		result = append(result, m)
	}
	sort.Strings(result)
	return result
}

// sortCheckerMovesByEquity sorts an analysis' checker moves by equity descending
// and recomputes indices and equity errors (the final normalisation legacy
// applies in both the insert and update paths).
func sortCheckerMovesByEquity(a *domain.PositionAnalysis) {
	if a.CheckerAnalysis == nil || len(a.CheckerAnalysis.Moves) == 0 {
		return
	}
	moves := a.CheckerAnalysis.Moves
	sort.Slice(moves, func(i, j int) bool {
		return moves[i].Equity > moves[j].Equity
	})
	bestEquity := moves[0].Equity
	for i := range moves {
		moves[i].Index = i
		if i == 0 {
			moves[i].EquityError = nil
		} else {
			diff := bestEquity - moves[i].Equity
			moves[i].EquityError = &diff
		}
	}
}

// mergeAnalysis combines incoming into existing (which may be nil when no
// analysis is stored yet for the position) and returns the analysis to persist.
// It mirrors database.saveAnalysisInTx exactly; AnalysisStore.Save then encodes
// and derives the scalar columns.
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
				sortCubeAnalysesByEngine(allCube)
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
		a.PlayedMoves = mergePlayedMoves(existingPlayedMoves, incomingPlayedMoves)

		// Merge played cube actions.
		existingCubeActions := existing.PlayedCubeActions
		if existing.PlayedCubeAction != "" && len(existingCubeActions) == 0 {
			existingCubeActions = []string{existing.PlayedCubeAction}
		}
		incomingCubeActions := a.PlayedCubeActions
		if a.PlayedCubeAction != "" && len(incomingCubeActions) == 0 {
			incomingCubeActions = []string{a.PlayedCubeAction}
		}
		a.PlayedCubeActions = mergePlayedMoves(existingCubeActions, incomingCubeActions)

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
