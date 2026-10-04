package rollouts

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// TestSaveValuedAnalysisSupersedesAnEarlierDepth: the storage path a daemon's
// stale sweep writes through replaces gammonNet's previous verdict, so a
// verdict at a shallower depth leaves nothing stale at that depth — the same
// outcome Database.AnalyzeStaleGammonNet reaches for the CLI and the GUI.
func TestSaveValuedAnalysisSupersedesAnEarlierDepth(t *testing.T) {
	ctx := context.Background()
	st, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	var pos domain.Position
	pos.Board.Points[8] = domain.Point{Checkers: 15, Color: domain.White}
	pos.Board.Points[17] = domain.Point{Checkers: 15, Color: domain.Black}
	pos.Dice = [2]int{6, 5}
	pos.Score = [2]int{-1, -1}
	id, err := st.Positions().Save(ctx, "", &pos)
	if err != nil {
		t.Fatalf("Positions().Save: %v", err)
	}

	verdict := func(depth string, moves ...string) *domain.PositionAnalysis {
		a := &domain.PositionAnalysis{PositionID: int(id), AnalysisType: "CheckerMove", CheckerAnalysis: &domain.CheckerAnalysis{}}
		for i, m := range moves {
			a.CheckerAnalysis.Moves = append(a.CheckerAnalysis.Moves, domain.CheckerMove{
				Index: i, Move: m, Equity: -float64(i) / 10, AnalysisEngine: gammonnet.EngineVersion, AnalysisDepth: depth,
			})
		}
		return a
	}
	if err := SaveValuedAnalysis(ctx, st, "", id, verdict("1-ply", "17/11 17/12", "17/11 11/6"), 0); err != nil {
		t.Fatalf("SaveValuedAnalysis (1-ply): %v", err)
	}
	if err := SaveValuedAnalysis(ctx, st, "", id, verdict("0-ply", "17/11 11/6"), 0); err != nil {
		t.Fatalf("SaveValuedAnalysis (0-ply): %v", err)
	}
	got, err := st.Analyses().Load(ctx, "", id)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if gammonnet.IsStaleAnalysis(got, "0-ply") {
		t.Errorf("analysis still stale at 0-ply after a 0-ply verdict: %+v", got.CheckerAnalysis)
	}
}
