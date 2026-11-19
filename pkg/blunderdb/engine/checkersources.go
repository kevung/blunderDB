package engine

import (
	"slices"
	"sort"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// A checker source is one engine at one depth (domain.CheckerSourceAnalysis).
// The merged view of a position (CheckerAnalysis, MergeCheckerMoves) keeps the
// deepest evaluation of each move and drops the others; the sources keep each
// one, so a reader can show what one engine alone said. The merged view is
// never rebuilt from the sources: it is computed as it always was, and the
// sources ride beside it.

// CheckerSourcesOf reads a list of candidate moves that carries no sources as
// sources. A list from one engine is one source, holding the moves as they
// came even when the engine evaluated them at several depths (XG rolls out
// its best candidates and leaves the others shallower), labelled as the
// provenance columns label it (analysis_engine, analysis_depth): the engine
// and the depth of the deepest move. A list mixing engines is a merged view
// whose indices and errors were measured from another engine's best move;
// what each engine alone said is lost, so it gives no source at all.
func CheckerSourcesOf(moves []domain.CheckerMove) []domain.CheckerSourceAnalysis {
	if len(moves) == 0 {
		return nil
	}
	src := domain.CheckerSourceAnalysis{AnalysisEngine: moves[0].AnalysisEngine, AnalysisDepth: moves[0].AnalysisDepth}
	for _, m := range moves[1:] {
		if m.AnalysisEngine != src.AnalysisEngine {
			return nil
		}
		if domain.AnalysisDepthRank(m.AnalysisDepth) > domain.AnalysisDepthRank(src.AnalysisDepth) {
			src.AnalysisDepth = m.AnalysisDepth
		}
	}
	src.Moves = moves
	return []domain.CheckerSourceAnalysis{src}
}

// CheckerSources is every source's evaluation of a's candidate moves: the
// stored set when several sources evaluated them, else the merged view as its
// single source (CheckerSourcesOf). Nil when a has no checker analysis, or
// when its merged view mixes engines and carries no stored set.
func CheckerSources(a *domain.PositionAnalysis) []domain.CheckerSourceAnalysis {
	switch {
	case a == nil:
		return nil
	case len(a.AllCheckerAnalyses) > 0:
		return a.AllCheckerAnalyses
	case a.CheckerAnalysis == nil || len(a.CheckerAnalysis.Moves) == 0:
		return nil
	}
	return CheckerSourcesOf(a.CheckerAnalysis.Moves)
}

// MergeCheckerSources folds an incoming evaluation of the candidate moves into
// the sources a position already carries, one entry per engine and depth. The
// incoming entry replaces its own source's, so re-importing a source rewrites
// it with itself. existingMoves is the stored merged view, read as the
// sources when existingAll is nil; incomingMoves likewise when incomingAll is.
//
// An incoming evaluation already in the merged view (a reader saving back what
// it loaded) is no new source: the existing ones stay. As with
// MergeCubeAnalyses, a single source needs no set, and the answer is nil. The
// moves are copies rounded for storage, so a result compares equal to what a
// store gives back.
func MergeCheckerSources(existingMoves []domain.CheckerMove, existingAll []domain.CheckerSourceAnalysis,
	incomingMoves []domain.CheckerMove, incomingAll []domain.CheckerSourceAnalysis) []domain.CheckerSourceAnalysis {
	existing := existingAll
	if existing == nil {
		existing = CheckerSourcesOf(existingMoves)
	}
	incoming := incomingAll
	if incoming == nil && !evaluatedIn(existingMoves, incomingMoves) {
		incoming = CheckerSourcesOf(incomingMoves)
	}
	seen := map[[2]string]bool{}
	var all []domain.CheckerSourceAnalysis
	for _, list := range [2][]domain.CheckerSourceAnalysis{incoming, existing} {
		for _, s := range list {
			key := [2]string{s.AnalysisEngine, s.AnalysisDepth}
			if seen[key] || len(s.Moves) == 0 {
				continue
			}
			seen[key] = true
			s.Moves = slices.Clone(s.Moves)
			roundCheckerMoves(s.Moves)
			all = append(all, s)
		}
	}
	if len(all) < 2 {
		return nil
	}
	SortCheckerSources(all)
	return all
}

// SortCheckerSources orders sources as the cube analyses are (EnginePriority),
// the deeper first within an engine, then by label, so that the order is
// total and a re-merge reproduces the stored bytes.
func SortCheckerSources(sources []domain.CheckerSourceAnalysis) {
	sort.SliceStable(sources, func(i, j int) bool {
		a, b := sources[i], sources[j]
		if pa, pb := EnginePriority(a.AnalysisEngine), EnginePriority(b.AnalysisEngine); pa != pb {
			return pa < pb
		}
		if a.AnalysisEngine != b.AnalysisEngine {
			return a.AnalysisEngine < b.AnalysisEngine
		}
		if ra, rb := domain.AnalysisDepthRank(a.AnalysisDepth), domain.AnalysisDepthRank(b.AnalysisDepth); ra != rb {
			return ra > rb
		}
		return a.AnalysisDepth < b.AnalysisDepth
	})
}

// evaluatedIn reports whether every incoming move is already in merged with
// the same engine, depth and equity at storage precision. An empty incoming
// list is: it evaluates nothing.
func evaluatedIn(merged, incoming []domain.CheckerMove) bool {
	for _, m := range incoming {
		if !slices.ContainsFunc(merged, func(s domain.CheckerMove) bool {
			return s.Move == m.Move && s.AnalysisEngine == m.AnalysisEngine && s.AnalysisDepth == m.AnalysisDepth &&
				RoundToMillipoint(s.Equity) == RoundToMillipoint(m.Equity)
		}) {
			return false
		}
	}
	return true
}

// MergeAnalysisCheckerSources is MergeCheckerSources over two analyses, read
// before the merged view replaces incoming's moves; existing is nil when the
// position has none.
func MergeAnalysisCheckerSources(existing, incoming *domain.PositionAnalysis) []domain.CheckerSourceAnalysis {
	var existingMoves, incomingMoves []domain.CheckerMove
	var existingAll []domain.CheckerSourceAnalysis
	if existing != nil {
		existingAll = existing.AllCheckerAnalyses
		if existing.CheckerAnalysis != nil {
			existingMoves = existing.CheckerAnalysis.Moves
		}
	}
	if incoming.CheckerAnalysis != nil {
		incomingMoves = incoming.CheckerAnalysis.Moves
	}
	return MergeCheckerSources(existingMoves, existingAll, incomingMoves, incoming.AllCheckerAnalyses)
}
