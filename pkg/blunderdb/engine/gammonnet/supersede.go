package gammonnet

import (
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
// search reads — the creation date, and the XGID and player names an import
// recorded when the verdict carries none: the XGID is what the Crawford
// repair reads the match state from.
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
	if out.XGID == "" {
		out.XGID = kept.XGID
	}
	if out.Player1 == "" {
		out.Player1 = kept.Player1
	}
	if out.Player2 == "" {
		out.Player2 = kept.Player2
	}

	switch {
	case kept.CheckerAnalysis != nil && out.CheckerAnalysis != nil:
		out.CheckerAnalysis = &domain.CheckerAnalysis{
			Moves: engine.MergeCheckerMoves(kept.CheckerAnalysis.Moves, out.CheckerAnalysis.Moves),
		}
	case out.CheckerAnalysis == nil:
		out.CheckerAnalysis = kept.CheckerAnalysis
	}
	rankCheckerMoves(out.CheckerAnalysis)

	out.DoublingCubeAnalysis, out.AllCubeAnalyses = engine.MergeCubeAnalyses(
		kept.DoublingCubeAnalysis, kept.AllCubeAnalyses,
		out.DoublingCubeAnalysis, out.AllCubeAnalyses)

	played := kept.PlayedMoves
	if kept.PlayedMove != "" && len(played) == 0 {
		played = []string{kept.PlayedMove}
	}
	out.PlayedMoves = engine.UnionPlayed(played, out.PlayedMoves)
	actions := kept.PlayedCubeActions
	if kept.PlayedCubeAction != "" && len(actions) == 0 {
		actions = []string{kept.PlayedCubeAction}
	}
	out.PlayedCubeActions = engine.UnionPlayed(actions, out.PlayedCubeActions)

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

// rankCheckerMoves ranks c's moves by equity (engine.RankCheckerMoves).
func rankCheckerMoves(c *domain.CheckerAnalysis) {
	if c != nil {
		engine.RankCheckerMoves(c.Moves)
	}
}
