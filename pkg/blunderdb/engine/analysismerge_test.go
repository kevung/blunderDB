package engine

import (
	"slices"
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

// The move table's "Double" and an importer's "Double/Take" are one double;
// "No Double" keeps its words.
func TestUnionCubeActionsFoldsTheBareDouble(t *testing.T) {
	got := UnionCubeActions([]string{"Double/Take", "No Double"}, []string{"Double", ""})
	want := []string{"Double/Take", "No Double"}
	if !slices.Equal(got, want) {
		t.Fatalf("UnionCubeActions = %v, want %v", got, want)
	}
	if got := UnionCubeActions([]string{"Double"}, []string{"Take"}); !slices.Equal(got, []string{"Double", "Take"}) {
		t.Fatalf("a bare double with no combined label is kept: %v", got)
	}
}

// "Double No" is XG's spelling of "No Double": the stored one stays, alone.
func TestUnionCubeActionsFoldsSpellingsOfOneAction(t *testing.T) {
	got := UnionCubeActions([]string{"Double No"}, []string{"No Double", "Take"})
	want := []string{"Double No", "Take"}
	if !slices.Equal(got, want) {
		t.Fatalf("UnionCubeActions = %v, want %v", got, want)
	}
	if got := UnionCubeActions([]string{"No Double"}, []string{"No Double"}); !slices.Equal(got, []string{"No Double"}) {
		t.Fatalf("identical entries must stay one: %v", got)
	}
}

// An import keeps the target's analysis, and still brings the plays the
// source saw.
func TestMergeImportedAnalysisUnionsThePlays(t *testing.T) {
	existing := &domain.PositionAnalysis{
		AnalysisType:    "CheckerMove",
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{{Move: "13/11 24/23"}}},
		PlayedMoves:     []string{"13/11 24/23"},
	}
	imported := &domain.PositionAnalysis{
		AnalysisType:    "CheckerMove",
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{{Move: "8/6 6/4"}}},
		PlayedMoves:     []string{"6/4 8/6"},
	}
	got, changed := MergeImportedAnalysis(existing, imported)
	if !changed || !slices.Equal(got.PlayedMoves, []string{"13/11 24/23", "6/4 8/6"}) {
		t.Fatalf("changed=%v PlayedMoves=%v", changed, got.PlayedMoves)
	}
	if got.CheckerAnalysis.Moves[0].Move != "13/11 24/23" || len(got.CheckerAnalysis.Moves) != 1 {
		t.Fatalf("the target's analysis must stay: %+v", got.CheckerAnalysis)
	}
	if len(existing.PlayedMoves) != 1 {
		t.Fatalf("existing was mutated: %v", existing.PlayedMoves)
	}
	if _, changed := MergeImportedAnalysis(got, &domain.PositionAnalysis{PlayedMoves: []string{"8/6 6/4"}}); changed {
		t.Fatal("a play the target already holds is no change")
	}
}
