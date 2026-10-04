package storagetest

import (
	"context"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/rollout"
	"github.com/kevung/blunderdb/pkg/blunderdb/rollouts"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testGammonNetVerdictKeepsTheRow: a gammonNet verdict written the way the
// serve daemon's stale sweep writes it replaces gammonNet's earlier entries
// and nothing else — the moves and cube actions played there, the creation
// date and the rollouts stay, and so does the denormalised error column the
// move-error search reads. It is the rule the CLI and the GUI write by
// (gammonnet.SupersedeEntries).
func testGammonNetVerdictKeepsTheRow(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	p := statsDecisionPos(t, 0)
	id, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}

	verdict := func(depth string, played, best float64) *domain.PositionAnalysis {
		return &domain.PositionAnalysis{
			AnalysisType: "CheckerMove", AnalysisEngineVersion: gammonnet.EngineVersion,
			CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
				{Move: "8/6 6/4", Equity: best, AnalysisEngine: gammonnet.EngineVersion, AnalysisDepth: depth},
				{Move: "13/11 6/4", Equity: played, AnalysisEngine: gammonnet.EngineVersion, AnalysisDepth: depth},
			}},
		}
	}
	created := time.Date(2025, 3, 4, 5, 6, 7, 0, time.UTC)
	first := verdict("1-ply", 0.30, 0.50)
	first.PlayedMoves = []string{"13/11 6/4"}
	first.PlayedCubeActions = []string{"Double"}
	first.CreationDate = created
	diff := 0.20
	first.CheckerAnalysis.Moves[1].Index, first.CheckerAnalysis.Moves[1].EquityError = 1, &diff
	if err := s.Analyses().Save(ctx, "", id, first); err != nil {
		t.Fatalf("Save analysis: %v", err)
	}
	if err := rollouts.Store(ctx, s, "", id, movesRollout(rollout.Fast(), "8/6 6/4", 0.6)); err != nil {
		t.Fatalf("Store rollout: %v", err)
	}
	if ids := searchIDs(t, s, domain.SearchFilters{MoveErrorFilter: "E>100"}); len(ids) != 1 || ids[0] != id {
		t.Fatalf("before the sweep, E>100 found %v, want [%d]", ids, id)
	}

	// The sweep's verdict carries no played move: the engine knows none.
	if err := rollouts.SaveValuedAnalysis(ctx, s, "", id, verdict("0-ply", 0.25, 0.45), 0); err != nil {
		t.Fatalf("SaveValuedAnalysis: %v", err)
	}
	got, err := s.Analyses().Load(ctx, "", id)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got.PlayedMoves) != 1 || got.PlayedMoves[0] != "13/11 6/4" {
		t.Errorf("PlayedMoves = %v, want [13/11 6/4]", got.PlayedMoves)
	}
	if len(got.PlayedCubeActions) != 1 || got.PlayedCubeActions[0] != "Double" {
		t.Errorf("PlayedCubeActions = %v, want [Double]", got.PlayedCubeActions)
	}
	if !got.CreationDate.Equal(created) {
		t.Errorf("CreationDate = %v, want %v", got.CreationDate, created)
	}
	if len(got.Rollouts) != 1 {
		t.Errorf("rollouts: got %d, want 1", len(got.Rollouts))
	}
	if m := got.CheckerAnalysis.Moves; len(m) != 2 || m[0].AnalysisDepth != "0-ply" || m[1].AnalysisDepth != "0-ply" {
		t.Errorf("gammonNet's earlier entries not replaced: %+v", m)
	}
	if ids := searchIDs(t, s, domain.SearchFilters{MoveErrorFilter: "E>100"}); len(ids) != 1 || ids[0] != id {
		t.Errorf("after the sweep, E>100 found %v, want [%d]: the error column lost the played move", ids, id)
	}
}
