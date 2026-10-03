package ingest

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// analysedGraph is sampleGraph with one checker analysis and one cube
// analysis at depth on its single position.
func analysedGraph(depth string, equity float64) *MatchGraph {
	g := sampleGraph()
	mg := &g.Games[0].Moves[0]
	mg.Analyses = []*domain.PositionAnalysis{{
		AnalysisType: "CheckerMove",
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
			{Move: "8/5 6/5", AnalysisDepth: depth, AnalysisEngine: "XG", Equity: equity},
			{Move: "24/23 13/10", AnalysisDepth: depth, AnalysisEngine: "XG", Equity: equity - 0.1},
		}},
		DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{AnalysisDepth: depth, AnalysisEngine: "XG", CubefulNoDoubleEquity: equity},
	}}
	return g
}

func storedAnalysis(t *testing.T, s *sqlite.Storage) *domain.PositionAnalysis {
	t.Helper()
	ctx := context.Background()
	ids, err := s.Positions().ListIDs(ctx, "", storage.ListOpts{})
	if err != nil || len(ids) != 1 {
		t.Fatalf("positions = %v, %v; want one", ids, err)
	}
	a, err := s.Analyses().Load(ctx, "", ids[0])
	if err != nil {
		t.Fatal(err)
	}
	return a
}

// A deeper analysis of a match already stored replaces the stored one; a
// shallower one, or the same one again, leaves it — whatever the order.
func TestDuplicateMatchDeepensItsAnalyses(t *testing.T) {
	ctx := context.Background()
	s, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if res := writeGraph(t, s, analysedGraph("3-ply", 0.10)); res.Skipped {
		t.Fatal("first import skipped")
	}

	res := writeGraph(t, s, analysedGraph("XG Roller++", 0.20))
	if !res.Skipped || res.Deepened != 1 {
		t.Fatalf("deeper re-import: %+v, want Skipped and Deepened=1", res)
	}
	a := storedAnalysis(t, s)
	if got := a.CheckerAnalysis.Moves[0].AnalysisDepth; got != "XG Roller++" {
		t.Errorf("checker depth after deeper re-import = %q", got)
	}
	if got := a.DoublingCubeAnalysis.AnalysisDepth; got != "XG Roller++" {
		t.Errorf("cube depth after deeper re-import = %q", got)
	}

	for _, again := range []*MatchGraph{analysedGraph("2-ply", 0.30), analysedGraph("XG Roller++", 0.20)} {
		res = writeGraph(t, s, again)
		if !res.Skipped || res.Deepened != 0 {
			t.Fatalf("shallower or equal re-import: %+v, want Skipped and Deepened=0", res)
		}
	}
	a = storedAnalysis(t, s)
	if got := a.CheckerAnalysis.Moves[0]; got.AnalysisDepth != "XG Roller++" || got.Equity != 0.20 {
		t.Errorf("best move after shallower re-import = %+v", got)
	}

	counts, _ := s.Metadata().Counts(ctx, "")
	if counts.Matches != 1 || counts.Games != 1 || counts.Moves != 1 {
		t.Fatalf("counts = %+v, want one match, game and move", counts)
	}
}

// SkipDuplicates restores the plain skip: the deeper analysis is not taken.
func TestDuplicateMatchSkipDuplicatesKeepsStored(t *testing.T) {
	ctx := context.Background()
	s, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	writeGraph(t, s, analysedGraph("3-ply", 0.10))
	g := analysedGraph("XG Roller++", 0.20)
	g.SkipDuplicates = true
	if res := writeGraph(t, s, g); !res.Skipped || res.Deepened != 0 {
		t.Fatalf("skip-duplicates re-import: %+v", res)
	}
	if got := storedAnalysis(t, s).CheckerAnalysis.Moves[0].AnalysisDepth; got != "3-ply" {
		t.Errorf("depth = %q, want 3-ply kept", got)
	}
}

func TestDeepenAnalysisReturnsExistingWhenNothingDeeper(t *testing.T) {
	existing := analysedGraph("4-ply", 0.1).Games[0].Moves[0].Analyses[0]
	incoming := analysedGraph("4-ply", 0.5).Games[0].Moves[0].Analyses[0]
	if got := deepenAnalysis(existing, *incoming); got != existing {
		t.Fatal("an equal-depth analysis must leave the stored one")
	}
	incoming.CheckerAnalysis.Moves = append(incoming.CheckerAnalysis.Moves, domain.CheckerMove{Move: "13/9", AnalysisDepth: "1-ply", Equity: -1})
	got := deepenAnalysis(existing, *incoming)
	if got == existing || len(got.CheckerAnalysis.Moves) != 3 || got.CheckerAnalysis.Moves[0].Equity != 0.1 {
		t.Fatalf("a missing candidate is added, the stored ones kept: %+v", got.CheckerAnalysis)
	}
}
