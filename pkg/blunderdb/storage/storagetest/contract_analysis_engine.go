// Contract case for AnalysisStore.WithEngine. The table that runs it lives in
// contract.go.
package storagetest

import (
	"context"
	"fmt"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testAnalysisWithEngineSkipsOtherEngines: over a corpus where one analysis in
// a hundred is gammonNet's, WithEngine yields exactly those, by ascending
// position id, and every other row is dismissed on the analysis_engine column
// without a decode: the number of records handed over (each one decoded) is
// the number of rows of the engine asked for, not the table size.
func testAnalysisWithEngineSkipsOtherEngines(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	total := 10000
	if testing.Short() {
		total = 1000
	}
	const prefix = "gammonNet "
	want := map[int64]string{}
	var order []int64
	for n := range total {
		p := distinctPos(n)
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save position %d: %v", n, err)
		}
		label := "XG 2.0"
		if n%100 == 0 {
			label = fmt.Sprintf("gammonNet v1.%d.0", n%7)
			want[id] = label
			order = append(order, id)
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

	var got []int64
	for rec, err := range s.Analyses().WithEngine(ctx, "", prefix) {
		if err != nil {
			t.Fatalf("WithEngine: %v", err)
		}
		got = append(got, rec.PositionID)
		label := rec.Analysis.CheckerAnalysis.Moves[0].AnalysisEngine
		if label != want[rec.PositionID] {
			t.Errorf("position %d: engine %q, want %q", rec.PositionID, label, want[rec.PositionID])
		}
	}
	if len(got) != len(order) {
		t.Fatalf("WithEngine yielded %d analyses (decodes), want %d out of %d stored", len(got), len(order), total)
	}
	for i := range got {
		if got[i] != order[i] {
			t.Fatalf("record %d is position %d, want %d: not ascending by position id", i, got[i], order[i])
		}
	}

	// A caller that stops early stops the pull.
	n := 0
	for _, err := range s.Analyses().WithEngine(ctx, "", prefix) {
		if err != nil {
			t.Fatalf("WithEngine: %v", err)
		}
		if n++; n == 3 {
			break
		}
	}
	if n != 3 {
		t.Errorf("early break after %d records, want 3", n)
	}
}

// distinctPos returns a position unique to n < 16384: n written in base 4 as
// the checker counts of seven otherwise empty points.
func distinctPos(n int) domain.Position {
	p := domain.InitializePosition()
	p.DecisionType = domain.CheckerAction
	for i := range 7 {
		p.Board.Points[9+i] = domain.Point{Checkers: n % 4, Color: domain.White}
		n /= 4
	}
	return p
}
