package gammonnet

import (
	"sort"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// SupersedeEntries is what a gammonNet verdict writes over existing, the
// analysis a position already carries (nil when none). It is the one rule
// every writer of a verdict applies — Database.saveAnalysisLocked for the
// CLI and the GUI, rollouts.SaveValuedAnalysis for the serve daemon — so a
// re-analysis leaves the same row whichever mode ran it.
//
// The verdict replaces gammonNet's earlier entries whatever their depth or
// version: merged by "the deeper entry wins", a sweep at a shallower depth
// would keep the old moves and find the position stale again on every pass.
// Everything else the row holds is not gammonNet's to drop and is kept: the
// other engines' entries (ADR-0013), the rollouts (ADR-0060), the moves and
// cube actions played there — they feed the denormalised error columns the
// search reads — and the creation date.
func SupersedeEntries(existing *domain.PositionAnalysis, verdict domain.PositionAnalysis) domain.PositionAnalysis {
	out := verdict
	if out.PlayedMove != "" && len(out.PlayedMoves) == 0 {
		out.PlayedMoves = []string{out.PlayedMove}
	}
	if out.PlayedCubeAction != "" && len(out.PlayedCubeActions) == 0 {
		out.PlayedCubeActions = []string{out.PlayedCubeAction}
	}
	out.PlayedMove, out.PlayedCubeAction = "", ""
	if existing == nil {
		rankCheckerMoves(out.CheckerAnalysis)
		return out
	}

	kept := withoutOurEntries(existing)
	if !kept.CreationDate.IsZero() {
		out.CreationDate = kept.CreationDate
	}

	switch {
	case kept.CheckerAnalysis != nil && out.CheckerAnalysis != nil:
		out.CheckerAnalysis = &domain.CheckerAnalysis{
			Moves: mergeCheckerMoves(kept.CheckerAnalysis.Moves, out.CheckerAnalysis.Moves),
		}
	case out.CheckerAnalysis == nil:
		out.CheckerAnalysis = kept.CheckerAnalysis
	}
	rankCheckerMoves(out.CheckerAnalysis)

	out.DoublingCubeAnalysis, out.AllCubeAnalyses = mergeCubeAnalyses(
		kept.DoublingCubeAnalysis, kept.AllCubeAnalyses,
		out.DoublingCubeAnalysis, out.AllCubeAnalyses)

	played := kept.PlayedMoves
	if kept.PlayedMove != "" && len(played) == 0 {
		played = []string{kept.PlayedMove}
	}
	out.PlayedMoves = unionPlayed(played, out.PlayedMoves)
	actions := kept.PlayedCubeActions
	if kept.PlayedCubeAction != "" && len(actions) == 0 {
		actions = []string{kept.PlayedCubeAction}
	}
	out.PlayedCubeActions = unionPlayed(actions, out.PlayedCubeActions)

	out.Rollouts = domain.MergeRollouts(kept.Rollouts, out.Rollouts)
	return out
}

func isOurs(eng string) bool { return strings.HasPrefix(eng, EngineLabelPrefix) }

// withoutOurEntries is a copy of a holding no checker move nor cube entry
// gammonNet wrote, at any version; a never shares its slices with it. A
// primary cube analysis that was gammonNet's falls back to another engine's
// entry, or to none.
func withoutOurEntries(a *domain.PositionAnalysis) domain.PositionAnalysis {
	out := *a
	if a.CheckerAnalysis != nil {
		var moves []domain.CheckerMove
		for _, m := range a.CheckerAnalysis.Moves {
			if !isOurs(m.AnalysisEngine) {
				moves = append(moves, m)
			}
		}
		out.CheckerAnalysis = nil
		if len(moves) > 0 {
			out.CheckerAnalysis = &domain.CheckerAnalysis{Moves: moves}
		}
	}
	var others []domain.DoublingCubeAnalysis
	for _, ca := range a.AllCubeAnalyses {
		if !isOurs(ca.AnalysisEngine) {
			others = append(others, ca)
		}
	}
	out.AllCubeAnalyses = others
	if a.DoublingCubeAnalysis != nil && isOurs(a.DoublingCubeAnalysis.AnalysisEngine) {
		out.DoublingCubeAnalysis = nil
		if len(others) > 0 {
			first := others[0]
			out.DoublingCubeAnalysis = &first
		}
	}
	return out
}

// enginePriority orders engines for display and ties: XG, then GNUbg, then
// the others.
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

// mergeCheckerMoves keys moves by notation; on a conflict the entry at the
// deeper (or equal) depth rank wins. The labels are free text, so their rank
// is compared, never the strings.
func mergeCheckerMoves(existing, incoming []domain.CheckerMove) []domain.CheckerMove {
	byMove := make(map[string]domain.CheckerMove, len(existing)+len(incoming))
	order := make([]string, 0, len(existing)+len(incoming))
	put := func(m domain.CheckerMove) {
		if _, ok := byMove[m.Move]; !ok {
			order = append(order, m.Move)
		}
		byMove[m.Move] = m
	}
	for _, m := range existing {
		put(m)
	}
	for _, m := range incoming {
		if prev, ok := byMove[m.Move]; ok &&
			domain.AnalysisDepthRank(m.AnalysisDepth) < domain.AnalysisDepthRank(prev.AnalysisDepth) {
			continue
		}
		put(m)
	}
	out := make([]domain.CheckerMove, 0, len(order))
	for _, k := range order {
		out = append(out, byMove[k])
	}
	return out
}

// rankCheckerMoves sorts the moves by equity, XG first on a tie, and
// recomputes each one's index and error to the best.
func rankCheckerMoves(c *domain.CheckerAnalysis) {
	if c == nil || len(c.Moves) == 0 {
		return
	}
	moves := c.Moves
	sort.SliceStable(moves, func(i, j int) bool {
		if moves[i].Equity != moves[j].Equity {
			return moves[i].Equity > moves[j].Equity
		}
		return enginePriority(moves[i].AnalysisEngine) < enginePriority(moves[j].AnalysisEngine)
	})
	best := moves[0].Equity
	for i := range moves {
		moves[i].Index = i
		moves[i].EquityError = nil
		if i > 0 {
			diff := best - moves[i].Equity
			moves[i].EquityError = &diff
		}
	}
}

// mergeCubeAnalyses keeps one cube entry per engine, incoming first so it
// wins its engine's slot. The primary is the incoming one when present, else
// the existing one; a single engine needs no set.
func mergeCubeAnalyses(existing *domain.DoublingCubeAnalysis, existingAll []domain.DoublingCubeAnalysis,
	incoming *domain.DoublingCubeAnalysis, incomingAll []domain.DoublingCubeAnalysis) (*domain.DoublingCubeAnalysis, []domain.DoublingCubeAnalysis) {
	var candidates []domain.DoublingCubeAnalysis
	candidates = append(candidates, incomingAll...)
	if incoming != nil {
		candidates = append(candidates, *incoming)
	}
	candidates = append(candidates, existingAll...)
	if existing != nil {
		candidates = append(candidates, *existing)
	}
	if len(candidates) == 0 {
		return nil, nil
	}
	seen := map[string]bool{}
	var all []domain.DoublingCubeAnalysis
	for _, ca := range candidates {
		if seen[ca.AnalysisEngine] {
			continue
		}
		seen[ca.AnalysisEngine] = true
		all = append(all, ca)
	}
	sort.SliceStable(all, func(i, j int) bool {
		return enginePriority(all[i].AnalysisEngine) < enginePriority(all[j].AnalysisEngine)
	})
	primary := incoming
	if primary == nil {
		primary = existing
	}
	if len(all) < 2 {
		return primary, nil
	}
	return primary, all
}

// unionPlayed unions played moves or cube actions in their normal notation,
// sorted.
func unionPlayed(existing, incoming []string) []string {
	set := make(map[string]bool, len(existing)+len(incoming))
	for _, m := range existing {
		if m != "" {
			set[engine.NormalizeMove(m)] = true
		}
	}
	for _, m := range incoming {
		if m != "" {
			set[engine.NormalizeMove(m)] = true
		}
	}
	out := make([]string, 0, len(set))
	for m := range set {
		out = append(out, m)
	}
	sort.Strings(out)
	return out
}
