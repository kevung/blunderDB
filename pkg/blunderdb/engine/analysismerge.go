package engine

import (
	"slices"
	"sort"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// The helpers every writer of an analysis merges with — the wrapper's
// SaveAnalysis, an import (package ingest), a gammonNet verdict
// (gammonnet.SupersedeEntries). One copy, so that a position merged by any
// of them is ordered the same way, and the order is total: re-merging what
// is stored reproduces it byte for byte, which AnalysisStore.Merge relies on
// to skip a rewrite.

// EnginePriority orders engines for display and ties: XG, then GNUbg, then
// the others.
func EnginePriority(eng string) int {
	switch strings.ToLower(eng) {
	case "xg":
		return 0
	case "gnubg":
		return 1
	default:
		return 2
	}
}

// SortCubeAnalysesByEngine sorts cube analyses XG first, then GNUbg, then the
// others, keeping the order within an engine.
func SortCubeAnalysesByEngine(analyses []domain.DoublingCubeAnalysis) {
	sort.SliceStable(analyses, func(i, j int) bool {
		return EnginePriority(analyses[i].AnalysisEngine) < EnginePriority(analyses[j].AnalysisEngine)
	})
}

// checkerMoveOutranksByEquity is the equity order of candidate moves: the
// higher equity first, XG first on a tie, then by notation, so that two
// moves never compare equal.
func checkerMoveOutranksByEquity(a, b domain.CheckerMove) bool {
	if a.Equity != b.Equity {
		return a.Equity > b.Equity
	}
	if pa, pb := EnginePriority(a.AnalysisEngine), EnginePriority(b.AnalysisEngine); pa != pb {
		return pa < pb
	}
	return a.Move < b.Move
}

// checkerMoveOutranksDepthFirst orders candidate moves as XG ranks them: the
// deeper analysis first, then checkerMoveOutranksByEquity. Depth comes before
// equity because a shallow evaluation's equity is not comparable to a deeper
// one's — a 2-ply move scoring above the 4-ply best is not better, only less
// examined.
func checkerMoveOutranksDepthFirst(a, b domain.CheckerMove) bool {
	if da, db := domain.AnalysisDepthRank(a.AnalysisDepth), domain.AnalysisDepthRank(b.AnalysisDepth); da != db {
		return da > db
	}
	return checkerMoveOutranksByEquity(a, b)
}

// RankCheckerMoves sorts moves by equity (checkerMoveOutranksByEquity) and
// recomputes each one's index and error to the best.
func RankCheckerMoves(moves []domain.CheckerMove) {
	rankCheckerMoves(moves, checkerMoveOutranksByEquity)
}

// RankCheckerMovesDepthFirst is RankCheckerMoves with the deeper analysis
// ranked first (checkerMoveOutranksDepthFirst): the order of an imported
// match, whose first candidate is the one every error is measured from.
func RankCheckerMovesDepthFirst(moves []domain.CheckerMove) {
	rankCheckerMoves(moves, checkerMoveOutranksDepthFirst)
}

func rankCheckerMoves(moves []domain.CheckerMove, less func(a, b domain.CheckerMove) bool) {
	if len(moves) == 0 {
		return
	}
	sort.Slice(moves, func(i, j int) bool { return less(moves[i], moves[j]) })
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

// MergeCheckerMoves unions two sets of candidate moves keyed by notation; on
// a conflict the incoming entry wins at an equal or deeper depth rank. The
// labels are free text ("2-ply", "XG Roller++"), so their rank is compared,
// never the strings. The result is in first-seen order, unranked: the caller
// ranks it (RankCheckerMoves or RankCheckerMovesDepthFirst).
func MergeCheckerMoves(existing, incoming []domain.CheckerMove) []domain.CheckerMove {
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

// MergeCubeAnalyses folds an incoming cube analysis into what a position
// already carries, keeping one entry per engine (ADR-0013).
//
// The primary is the incoming one when present, else the existing one: a
// caller silent about the cube must not blank it. The set is deduplicated by
// engine, incoming first (a newer XG replaces XG's entry), and sorted so the
// display does not depend on import order. A single engine needs no set: an
// array of one is noise in every blob that carries it, and AllCubeAnalyses
// is read as "several" everywhere.
func MergeCubeAnalyses(existing *domain.DoublingCubeAnalysis, existingAll []domain.DoublingCubeAnalysis,
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
	SortCubeAnalysesByEngine(all)
	primary := incoming
	if primary == nil {
		primary = existing
	}
	if len(all) < 2 {
		return primary, nil
	}
	return primary, all
}

// UnionPlayed unions played checker moves in their normal notation
// (NormalizeMove), sorted; an empty entry is not a play. Cube actions go
// through UnionCubeActions: NormalizeMove would turn "No Double" into
// "Double No".
func UnionPlayed(existing, incoming []string) []string {
	set := make(map[string]bool, len(existing)+len(incoming))
	for _, m := range existing {
		if m != "" {
			set[NormalizeMove(m)] = true
		}
	}
	for _, m := range incoming {
		if m != "" {
			set[NormalizeMove(m)] = true
		}
	}
	out := make([]string, 0, len(set))
	for m := range set {
		out = append(out, m)
	}
	sort.Strings(out)
	return out
}

// UnionCubeActions unions played cube actions, sorted; an empty entry is not
// a play. The doubler's position is written two ways: the move table says
// "Double", an importer's analysis says "Double/Take" or "Double/Pass", the
// offer with its answer. A bare double says nothing the combined label does
// not, so it is dropped beside one: a reader would otherwise list one double
// twice. Spellings of one no-double, take or pass ("No Double", "Double No")
// are one action too: the first met, the stored one, stays.
func UnionCubeActions(existing, incoming []string) []string {
	set := make(map[string]bool, len(existing)+len(incoming))
	seen := make(map[string]bool, 3)
	combined := false
	for _, list := range [2][]string{existing, incoming} {
		for _, a := range list {
			if a == "" {
				continue
			}
			if c := CanonicalCubeAction(a); c == CubeNoDouble || c == CubeTake || c == CubePass {
				if seen[c] && !set[a] {
					continue
				}
				seen[c] = true
			}
			set[a] = true
			if isCombinedDouble(a) {
				combined = true
			}
		}
	}
	out := make([]string, 0, len(set))
	for a := range set {
		if combined && isBareDouble(a) {
			continue
		}
		out = append(out, a)
	}
	sort.Strings(out)
	return out
}

// isCombinedDouble reports whether a names an offer with its answer.
func isCombinedDouble(a string) bool {
	if CanonicalCubeAction(a) != CubeDouble {
		return false
	}
	s := normaliseCubeLabel(a)
	return strings.Contains(s, "take") || strings.Contains(s, "pass") ||
		strings.Contains(s, "drop") || strings.Contains(s, "beaver")
}

// isBareDouble reports whether a names an offer and nothing more.
func isBareDouble(a string) bool {
	return CanonicalCubeAction(a) == CubeDouble && !isCombinedDouble(a)
}

// MergeImportedAnalysis is domain.MergeImportedAnalysis with the plays of both
// sides: whichever analysis an import keeps, what the source library saw
// played at the position joins what the target saw, as a save unions them.
// changed also reports a play the target did not have.
func MergeImportedAnalysis(existing, imported *domain.PositionAnalysis) (*domain.PositionAnalysis, bool) {
	merged, changed := domain.MergeImportedAnalysis(existing, imported)
	if existing == nil || imported == nil {
		return merged, changed
	}
	moves := UnionPlayed(legacyPlayed(existing.PlayedMoves, existing.PlayedMove),
		legacyPlayed(imported.PlayedMoves, imported.PlayedMove))
	actions := UnionCubeActions(legacyPlayed(existing.PlayedCubeActions, existing.PlayedCubeAction),
		legacyPlayed(imported.PlayedCubeActions, imported.PlayedCubeAction))
	movesDiffer := !slices.Equal(UnionPlayed(legacyPlayed(merged.PlayedMoves, merged.PlayedMove), nil), moves)
	actionsDiffer := !slices.Equal(UnionCubeActions(legacyPlayed(merged.PlayedCubeActions, merged.PlayedCubeAction), nil), actions)
	if !movesDiffer && !actionsDiffer {
		return merged, changed
	}
	out := *merged
	if movesDiffer {
		out.PlayedMoves, out.PlayedMove = moves, ""
	}
	if actionsDiffer {
		out.PlayedCubeActions, out.PlayedCubeAction = actions, ""
	}
	return &out, true
}

// legacyPlayed is played, or the deprecated single field when played is empty.
func legacyPlayed(played []string, single string) []string {
	if single != "" && len(played) == 0 {
		return []string{single}
	}
	return played
}
