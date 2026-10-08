package ingest

import (
	"reflect"
	"sort"
	"strings"
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

// checkerMoveRanksFirst orders candidate moves as XG ranks them: the deeper
// analysis first, then by equity, XG first on a tie, then by notation. Depth
// comes before equity because a shallow evaluation's equity is not comparable
// to a deeper one's — a 2-ply move scoring above the 4-ply best is not better,
// only less examined — and the first candidate is the one every error is
// measured from. The order is total: a merge is then deterministic, so
// re-merging what is stored reproduces it byte for byte and
// AnalysisStore.Merge can skip the write.
func checkerMoveRanksFirst(a, b domain.CheckerMove) bool {
	if da, db := domain.AnalysisDepthRank(a.AnalysisDepth), domain.AnalysisDepthRank(b.AnalysisDepth); da != db {
		return da > db
	}
	if a.Equity != b.Equity {
		return a.Equity > b.Equity
	}
	if pa, pb := enginePriority(a.AnalysisEngine), enginePriority(b.AnalysisEngine); pa != pb {
		return pa < pb
	}
	return a.Move < b.Move
}

// mergeCheckerMoves merges two sets of checker moves keyed by move string,
// preferring the higher-depth analysis on conflict, then re-ranks them
// (checkerMoveRanksFirst) and recomputes per-move equity errors.
func mergeCheckerMoves(existing, incoming []domain.CheckerMove) []domain.CheckerMove {
	moveMap := make(map[string]domain.CheckerMove)
	for _, m := range existing {
		moveMap[m.Move] = m
	}
	for _, m := range incoming {
		if existingMove, exists := moveMap[m.Move]; exists {
			if domain.AnalysisDepthRank(m.AnalysisDepth) >= domain.AnalysisDepthRank(existingMove.AnalysisDepth) {
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

	sort.Slice(result, func(i, j int) bool { return checkerMoveRanksFirst(result[i], result[j]) })

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

// sortCheckerMovesByEquity sorts an analysis' checker moves by depth, then
// equity (checkerMoveRanksFirst), and recomputes indices and equity errors (the final normalisation of both the
// insert and update paths).
func sortCheckerMovesByEquity(a *domain.PositionAnalysis) {
	if a.CheckerAnalysis == nil || len(a.CheckerAnalysis.Moves) == 0 {
		return
	}
	moves := a.CheckerAnalysis.Moves
	sort.Slice(moves, func(i, j int) bool { return checkerMoveRanksFirst(moves[i], moves[j]) })
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
				sortCubeAnalysesByEngine(all)
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
