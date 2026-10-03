package database

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/rollout"
)

// tinySettings keeps the engine's part under a second.
func tinySettings() rollout.Settings {
	s := rollout.Fast()
	s.MaxGames, s.MinGames, s.Truncation, s.Candidates = 36, 36, 2, 2
	return s
}

// A stored rollout sits beside the imported analysis; saving that analysis
// again — an import, the GUI's own save — keeps it; the batch skips the
// position on the next run.
func TestRolloutPosition_StoresBesideAndSurvivesSaveAnalysis(t *testing.T) {
	d := newTestDB(t)
	p := domain.InitializePosition()
	id, err := d.SavePosition(&p)
	if err != nil {
		t.Fatal(err)
	}
	imported := PositionAnalysis{AnalysisType: "CheckerMove", CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
		{Move: "8/5 6/5", Equity: 0.1, AnalysisEngine: "XG", AnalysisDepth: "XG Roller++"},
	}}}
	if err := d.SaveAnalysis(id, imported); err != nil {
		t.Fatal(err)
	}

	s := tinySettings()
	res, err := d.RolloutPosition(context.Background(), id, s, nil, true, nil)
	if err != nil || res == nil {
		t.Fatalf("RolloutPosition: %v", err)
	}
	if err := d.SaveAnalysis(id, imported); err != nil {
		t.Fatal(err)
	}
	a, err := d.LoadAnalysis(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Rollouts) != 1 || a.Rollouts[0].Signature != s.Signature() {
		t.Fatalf("rollouts after a re-save: %+v", a.Rollouts)
	}
	for _, m := range a.CheckerAnalysis.Moves {
		if m.AnalysisEngine != "XG" {
			t.Errorf("a rollout entry entered the imported analysis: %+v", m)
		}
	}

	todo, err := d.PositionsToRollout(context.Background(), domain.SearchFilters{}, s)
	if err != nil || len(todo) != 0 {
		t.Errorf("PositionsToRollout after the rollout: %d, %v", len(todo), err)
	}
	sum, err := d.RolloutFiltered(context.Background(), domain.SearchFilters{}, s, nil)
	if err != nil || sum.Total != 0 {
		t.Errorf("RolloutFiltered reran a done position: %+v, %v", sum, err)
	}
}
