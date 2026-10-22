package database

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
)

// The comparison sweep looks at a position when ANY entry of its analysis was
// written by another engine: one gammonNet entry beside an XG one does not make
// the analysis gammonNet's own.
func TestPositionsWithForeignAnalysis(t *testing.T) {
	t.Parallel()
	d := newTestDB(t)
	ours := gammonnet.EngineLabelPrefix + "v1.0.0"
	checker := func(engines ...string) domain.PositionAnalysis {
		a := domain.PositionAnalysis{AnalysisType: "CheckerMove", CheckerAnalysis: &domain.CheckerAnalysis{}}
		for i, e := range engines {
			a.CheckerAnalysis.Moves = append(a.CheckerAnalysis.Moves, domain.CheckerMove{
				Index: i, Move: []string{"13/11 24/23", "24/21"}[i], Equity: 0.1 - float64(i)/10,
				AnalysisEngine: e, AnalysisDepth: "2-ply"})
		}
		return a
	}
	cases := []struct {
		analysis *domain.PositionAnalysis
		foreign  bool
	}{
		{ptr(checker("XG 2.0")), true},
		{ptr(checker(ours)), false},
		{ptr(checker(ours, "XG 2.0")), true},
		{nil, false},
	}
	var want []int64
	for i, c := range cases {
		p := domain.InitializePosition()
		p.DecisionType = domain.CheckerAction
		p.Board.Points[9] = domain.Point{Checkers: i + 1, Color: domain.White}
		id, err := d.SavePosition(&p)
		if err != nil {
			t.Fatalf("SavePosition %d: %v", i, err)
		}
		if c.analysis != nil {
			if err := d.store.Analyses().Save(context.Background(), "", id, c.analysis); err != nil {
				t.Fatalf("Save analysis %d: %v", i, err)
			}
		}
		if c.foreign {
			want = append(want, id)
		}
	}

	got, err := d.positionIDsWithForeignAnalysis()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("positionIDsWithForeignAnalysis = %v, want %v", got, want)
	}
	if n, err := d.CountPositionsWithForeignAnalysis(); err != nil || n != len(want) {
		t.Errorf("CountPositionsWithForeignAnalysis = %d, %v; want %d", n, err, len(want))
	}
}
