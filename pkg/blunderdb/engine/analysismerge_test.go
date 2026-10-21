package engine

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// Two candidates of equal equity from the same engine are ordered by
// notation, whatever order they arrive in: a merge must be deterministic for
// every writer, or re-saving a stored analysis would reorder its moves.
func TestRankCheckerMovesBreaksEquityTiesByNotation(t *testing.T) {
	for _, in := range [][]domain.CheckerMove{
		{{Move: "24/18 13/11", Equity: 0.1}, {Move: "13/7 13/11", Equity: 0.1}, {Move: "8/2 6/2", Equity: 0.3}},
		{{Move: "13/7 13/11", Equity: 0.1}, {Move: "8/2 6/2", Equity: 0.3}, {Move: "24/18 13/11", Equity: 0.1}},
	} {
		moves := MergeCheckerMoves(nil, in)
		RankCheckerMoves(moves)
		want := []string{"8/2 6/2", "13/7 13/11", "24/18 13/11"}
		for i, w := range want {
			if moves[i].Move != w || moves[i].Index != i {
				t.Fatalf("rank %d = %q (index %d), want %q; full %v", i, moves[i].Move, moves[i].Index, w, moves)
			}
		}
		if moves[0].EquityError != nil || moves[2].EquityError == nil || *moves[2].EquityError != moves[0].Equity-moves[2].Equity {
			t.Errorf("equity errors not measured from the best: %v", moves)
		}
	}
}

// At equal equity XG ranks before GNUbg, before the notation is looked at.
func TestRankCheckerMovesPrefersXGOnATie(t *testing.T) {
	moves := []domain.CheckerMove{
		{Move: "a", Equity: 0.1, AnalysisEngine: "GNUbg"},
		{Move: "b", Equity: 0.1, AnalysisEngine: "XG"},
	}
	RankCheckerMoves(moves)
	if moves[0].Move != "b" {
		t.Errorf("first = %q, want the XG entry", moves[0].Move)
	}
}

// The depth-first rank puts a deeper analysis first even at a lower equity.
func TestRankCheckerMovesDepthFirst(t *testing.T) {
	moves := []domain.CheckerMove{
		{Move: "a", Equity: 0.5, AnalysisDepth: "1-ply"},
		{Move: "b", Equity: 0.1, AnalysisDepth: "4-ply"},
	}
	RankCheckerMovesDepthFirst(moves)
	if moves[0].Move != "b" {
		t.Errorf("first = %q, want the 4-ply entry", moves[0].Move)
	}
}

// On a conflict the incoming move wins at an equal or deeper rank only.
func TestMergeCheckerMovesKeepsTheDeeperEntry(t *testing.T) {
	existing := []domain.CheckerMove{{Move: "a", Equity: 0.2, AnalysisDepth: "4-ply"}}
	got := MergeCheckerMoves(existing, []domain.CheckerMove{{Move: "a", Equity: 0.9, AnalysisDepth: "2-ply"}})
	if len(got) != 1 || got[0].Equity != 0.2 {
		t.Errorf("shallower incoming replaced the stored entry: %v", got)
	}
	got = MergeCheckerMoves(existing, []domain.CheckerMove{{Move: "a", Equity: 0.9, AnalysisDepth: "4-ply"}})
	if len(got) != 1 || got[0].Equity != 0.9 {
		t.Errorf("incoming at equal depth did not replace the stored entry: %v", got)
	}
}
