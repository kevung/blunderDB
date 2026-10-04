package gammonnet

import (
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// TestSupersedeEntriesKeepsWhatIsNotOurs: gammonNet's earlier entries go
// whatever their depth; the other engines' entries, the played moves, the
// creation date and the rollouts stay.
func TestSupersedeEntriesKeepsWhatIsNotOurs(t *testing.T) {
	created := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	xgCube := domain.DoublingCubeAnalysis{AnalysisEngine: "XG", BestCubeAction: "No Double"}
	existing := &domain.PositionAnalysis{
		CreationDate: created,
		XGID:         "XGID=-b----E-C---eE---c-e----B-:0:0:1:52:0:0:3:0:10",
		Player1:      "Alice",
		Player2:      "Bob",
		PlayedMoves:  []string{"13/11 6/4"},
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
			{Move: "8/6 6/4", Equity: 0.5, AnalysisEngine: EngineVersion, AnalysisDepth: "2-ply"},
			{Move: "24/22 13/11", Equity: 0.4, AnalysisEngine: "XG", AnalysisDepth: "XG Roller++"},
		}},
		DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{AnalysisEngine: EngineVersion},
		AllCubeAnalyses:      []domain.DoublingCubeAnalysis{{AnalysisEngine: EngineVersion}, xgCube},
		Rollouts:             []domain.RolloutAnalysis{{Signature: "s"}},
	}
	verdict := domain.PositionAnalysis{
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
			{Move: "13/11 6/4", Equity: 0.45, AnalysisEngine: EngineVersion, AnalysisDepth: "0-ply"},
		}},
	}

	got := SupersedeEntries(existing, verdict)

	if m := got.CheckerAnalysis.Moves; len(m) != 2 || m[0].Move != "13/11 6/4" || m[1].Move != "24/22 13/11" ||
		m[1].EquityError == nil || *m[1].EquityError < 0.049 || *m[1].EquityError > 0.051 {
		t.Errorf("checker moves = %+v, want the 0-ply verdict ranked above XG's", m)
	}
	if got.DoublingCubeAnalysis == nil || got.DoublingCubeAnalysis.AnalysisEngine != "XG" || got.AllCubeAnalyses != nil {
		t.Errorf("cube = %+v / %+v, want XG's alone", got.DoublingCubeAnalysis, got.AllCubeAnalyses)
	}
	if len(got.PlayedMoves) != 1 || got.PlayedMoves[0] != "13/11 6/4" {
		t.Errorf("PlayedMoves = %v", got.PlayedMoves)
	}
	if got.XGID != existing.XGID || got.Player1 != "Alice" || got.Player2 != "Bob" {
		t.Errorf("XGID/players = %q %q %q, want the imported ones", got.XGID, got.Player1, got.Player2)
	}
	if again := SupersedeEntries(existing, domain.PositionAnalysis{XGID: "XGID=other", Player1: "Carol"}); again.XGID != "XGID=other" || again.Player1 != "Carol" || again.Player2 != "Bob" {
		t.Errorf("a verdict's own XGID/players must win: %q %q %q", again.XGID, again.Player1, again.Player2)
	}
	if !got.CreationDate.Equal(created) || len(got.Rollouts) != 1 {
		t.Errorf("CreationDate %v, rollouts %d", got.CreationDate, len(got.Rollouts))
	}
	if len(existing.CheckerAnalysis.Moves) != 2 || existing.CheckerAnalysis.Moves[0].AnalysisEngine != EngineVersion {
		t.Errorf("existing was modified: %+v", existing.CheckerAnalysis.Moves)
	}
}
