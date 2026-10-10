package storagetest

import (
	"context"
	"slices"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testStatsAnalysisEngines: the options of the Stats engine filter are the
// distinct engine labels of the stored analyses, sorted, none when the
// database holds no analysis.
func testStatsAnalysisEngines(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	got, err := s.Stats().AnalysisEngines(ctx, "")
	if err != nil || len(got) != 0 {
		t.Fatalf("empty database: %v, %v; want none", got, err)
	}
	for n, label := range []string{"XG", "GNUbg", "XG", "gammonNet v1.6.0"} {
		p := distinctPos(n)
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save position %d: %v", n, err)
		}
		a := &domain.PositionAnalysis{
			AnalysisType: "CheckerMove",
			CheckerAnalysis: &domain.CheckerAnalysis{
				Moves: []domain.CheckerMove{{Index: 0, Move: "13/11 24/23", Equity: 0.1, AnalysisEngine: label, AnalysisDepth: "2-ply"}},
			},
		}
		if err := s.Analyses().Save(ctx, "", id, a); err != nil {
			t.Fatalf("Save analysis %d: %v", n, err)
		}
	}
	got, err = s.Stats().AnalysisEngines(ctx, "")
	if err != nil {
		t.Fatalf("AnalysisEngines: %v", err)
	}
	if want := []string{"GNUbg", "XG", "gammonNet v1.6.0"}; !slices.Equal(got, want) {
		t.Errorf("AnalysisEngines = %v, want %v", got, want)
	}
}
