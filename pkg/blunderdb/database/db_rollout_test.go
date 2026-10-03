package database

import (
	"context"
	"reflect"
	"strings"
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
	res, err := RolloutPosition(context.Background(), d, id, s, nil, true, nil)
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

	todo, err := PositionsToRollout(context.Background(), d, domain.SearchFilters{}, s)
	if err != nil || len(todo) != 0 {
		t.Errorf("PositionsToRollout after the rollout: %d, %v", len(todo), err)
	}
	sum, err := RolloutFiltered(context.Background(), d, domain.SearchFilters{}, s, nil)
	if err != nil || sum.Total != 0 {
		t.Errorf("RolloutFiltered reran a done position: %+v, %v", sum, err)
	}
}

// Wails binds every exported method of *Database to the webview: no rollout
// method may take a context.Context (nothing there can supply one) nor a
// Result the caller made up.
func TestDatabaseBindsNoUncallableRolloutMethod(t *testing.T) {
	ctxType := reflect.TypeOf((*context.Context)(nil)).Elem()
	typ := reflect.TypeOf(&Database{})
	for i := 0; i < typ.NumMethod(); i++ {
		m := typ.Method(i)
		if !strings.Contains(m.Name, "Rollout") {
			continue
		}
		for j := 1; j < m.Type.NumIn(); j++ {
			if m.Type.In(j) == ctxType || m.Type.In(j) == reflect.TypeOf((*rollout.Result)(nil)) {
				t.Errorf("(*Database).%s is bound to the webview but takes %s", m.Name, m.Type.In(j))
			}
		}
	}
}
