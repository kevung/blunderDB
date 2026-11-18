// Contract cases for the per-source checker evaluations
// (domain.PositionAnalysis.AllCheckerAnalyses). The table that runs them
// lives in contract.go; the tenant case in tenant_isolation.go.
package storagetest

import (
	"context"
	"reflect"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// sourceAnalysis is one engine's evaluation of checkerPos's candidates.
func sourceAnalysis(eng, depth string, equities ...float64) domain.PositionAnalysis {
	notations := []string{"13/11 24/23", "24/21", "13/10"}
	moves := make([]domain.CheckerMove, len(equities))
	for i, e := range equities {
		moves[i] = domain.CheckerMove{Move: notations[i], AnalysisEngine: eng, AnalysisDepth: depth, Equity: e}
	}
	engine.RankCheckerMovesDepthFirst(moves)
	return domain.PositionAnalysis{AnalysisType: "CheckerMove", CheckerAnalysis: &domain.CheckerAnalysis{Moves: moves}}
}

// mergeSaved folds incoming into what scope stores at posID, as
// analyses.merge, the desktop and the CLI save it, and reports a write.
func mergeSaved(t *testing.T, s storage.Storage, scope string, posID int64, incoming domain.PositionAnalysis) bool {
	t.Helper()
	wrote, err := s.Analyses().Merge(context.Background(), scope, posID, nil, func(existing *domain.PositionAnalysis) *domain.PositionAnalysis {
		m := storage.MergeSavedAnalysis(existing, incoming)
		return &m
	})
	if err != nil {
		t.Fatalf("Merge(%q): %v", scope, err)
	}
	return wrote
}

// testAnalysisKeepsCheckerSources: a second engine's evaluation of a position
// joins the first as its own source, Load and LoadMany give both, the merged
// view is the one the deepest-wins rule gives, and merging a source again
// writes nothing.
func testAnalysisKeepsCheckerSources(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	p := checkerPos()
	posID, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}
	xg := sourceAnalysis("XG", "XG Roller++", 0.30, 0.20, 0.10)
	gnu := sourceAnalysis("GNUbg", "2-ply", 0.25, 0.28)
	mergeSaved(t, s, "", posID, xg)
	mergeSaved(t, s, "", posID, gnu)

	got, err := s.Analyses().Load(ctx, "", posID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	srcs := got.AllCheckerAnalyses
	if len(srcs) != 2 || srcs[0].AnalysisEngine != "XG" || srcs[1].AnalysisEngine != "GNUbg" ||
		srcs[1].AnalysisDepth != "2-ply" || len(srcs[1].Moves) != 2 || srcs[1].Moves[0].Equity != 0.28 {
		t.Fatalf("sources = %+v, want XG then GNUbg with its own moves", srcs)
	}
	view := engine.MergeCheckerMoves(sourceAnalysis("XG", "XG Roller++", 0.30, 0.20, 0.10).CheckerAnalysis.Moves,
		sourceAnalysis("GNUbg", "2-ply", 0.25, 0.28).CheckerAnalysis.Moves)
	engine.RankCheckerMoves(view)
	want := domain.PositionAnalysis{CheckerAnalysis: &domain.CheckerAnalysis{Moves: view}}
	engine.RoundAnalysisForStorage(&want)
	if !reflect.DeepEqual(got.CheckerAnalysis.Moves, view) {
		t.Errorf("merged view = %+v\nwant %+v", got.CheckerAnalysis.Moves, view)
	}
	many, err := s.Analyses().LoadMany(ctx, "", []int64{posID})
	if err != nil {
		t.Fatalf("LoadMany: %v", err)
	}
	if a := many[posID]; a == nil || !reflect.DeepEqual(a.AllCheckerAnalyses, srcs) {
		t.Errorf("LoadMany sources = %+v, want Load's", a)
	}

	if mergeSaved(t, s, "", posID, sourceAnalysis("GNUbg", "2-ply", 0.25, 0.28)) {
		t.Error("merging the same source again wrote the analysis")
	}
	if mergeSaved(t, s, "", posID, *got) {
		t.Error("saving back the loaded analysis wrote it")
	}
}

// checkCheckerSourcesIsolation: one position under two tenants keeps each
// tenant's sources apart.
func checkCheckerSourcesIsolation(t *testing.T, ctx context.Context, s storage.Storage, a, b string) {
	pa, pb := checkerPos(), checkerPos()
	idA, err := s.Positions().Save(ctx, a, &pa)
	if err != nil {
		t.Fatalf("Save position(%s): %v", a, err)
	}
	idB, err := s.Positions().Save(ctx, b, &pb)
	if err != nil {
		t.Fatalf("Save position(%s): %v", b, err)
	}
	mergeSaved(t, s, a, idA, sourceAnalysis("XG", "XG Roller++", 0.30, 0.20))
	mergeSaved(t, s, a, idA, sourceAnalysis("GNUbg", "2-ply", 0.25, 0.28))
	mergeSaved(t, s, b, idB, sourceAnalysis("XG", "3-ply", 0.10, 0.05))

	gotA, err := s.Analyses().Load(ctx, a, idA)
	if err != nil || len(gotA.AllCheckerAnalyses) != 2 {
		t.Fatalf("Load(%s) sources = %+v, %v; want two", a, gotA, err)
	}
	gotB, err := s.Analyses().Load(ctx, b, idB)
	if err != nil {
		t.Fatalf("Load(%s): %v", b, err)
	}
	if gotB.AllCheckerAnalyses != nil || gotB.CheckerAnalysis.Moves[0].AnalysisDepth != "3-ply" {
		t.Errorf("Load(%s) = %+v, want its single 3-ply source", b, gotB)
	}
}
