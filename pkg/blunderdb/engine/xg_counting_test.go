package engine

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

func candidates(equities ...float64) *domain.CheckerAnalysis {
	ca := &domain.CheckerAnalysis{}
	for i, e := range equities {
		ca.Moves = append(ca.Moves, domain.CheckerMove{Index: i, Move: string(rune('a' + i)), Equity: e})
	}
	return ca
}

// TestIsForcedChecker pins XG's rule: one legal play, or every legal play
// analysed at one equity; anything else is a decision.
func TestIsForcedChecker(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		ca    *domain.CheckerAnalysis
		legal int
		want  bool
	}{
		{"single legal play", candidates(0.1, 0.05), 1, true},
		{"dance", nil, 0, true},
		{"all legal plays tie", candidates(0.123, 0.123, 0.123), 3, true},
		{"all legal plays, one apart", candidates(0.124, 0.123), 2, false},
		{"tie but a legal play unanalysed", candidates(0.2, 0.2), 3, false},
		{"more candidates than legal plays, all tied", candidates(0.2, 0.2, 0.2), 2, true},
		{"unknown legal count, lone candidate", candidates(0.3), LegalPlaysUnknown, true},
		{"unknown legal count, two tied candidates", candidates(0.3, 0.3), LegalPlaysUnknown, false},
		{"no candidates, several legal plays", nil, 4, false},
	}
	for _, c := range cases {
		if got := IsForcedChecker(c.ca, c.legal); got != c.want {
			t.Errorf("%s: IsForcedChecker = %v, want %v", c.name, got, c.want)
		}
	}
}

// TestComputeIsCloseCube pins which cube decisions count: every offer and
// answer, and a no-double within 0.200 of min(D/T, D/P) with a positive
// equity — a too-good position near cashing included.
func TestComputeIsCloseCube(t *testing.T) {
	t.Parallel()
	dca := func(nd, dt, dp float64) *domain.DoublingCubeAnalysis {
		return &domain.DoublingCubeAnalysis{CubefulNoDoubleEquity: nd, CubefulDoubleTakeEquity: dt, CubefulDoublePassEquity: dp}
	}
	cases := []struct {
		name   string
		dca    *domain.DoublingCubeAnalysis
		played string
		want   int64
	}{
		{"double offered, far from doubling", dca(0.1, -0.5, 1), "Double", 1},
		{"take", nil, "Take", 1},
		{"pass", nil, "Pass", 1},
		{"missed double", dca(0.3, 0.9, 1), "No Double", 1},
		{"no double just inside", dca(0.799, 0.6, 1), "No Double", 1},
		{"no double too far", dca(0.81, 0.6, 1), "No Double", 0},
		{"too good, within 0.2 of cashing", dca(1.15, 1.4, 1), "No Double", 1},
		{"too good, far past cashing", dca(1.25, 1.6, 1), "No Double", 0},
		{"no double, negative equity", dca(-0.05, 0.1, 1), "No Double", 0},
		{"no cube analysis", nil, "No Double", 0},
	}
	for _, c := range cases {
		if got := ComputeIsCloseCube(c.dca, c.played); got != c.want {
			t.Errorf("%s: ComputeIsCloseCube = %d, want %d", c.name, got, c.want)
		}
	}
}

// TestPlayedCandidate matches a play spelt two ways. In a list that writes its
// hits, a missing "*" means no hit; in one that never does, a match naming
// two candidates is refused.
func TestPlayedCandidate(t *testing.T) {
	t.Parallel()
	opening := domain.InitializePosition()
	opening.Dice = [2]int{5, 1}
	moves := []domain.CheckerMove{{Move: "13/8 6/5", AnalysisEngine: "XG"}, {Move: "24/23 13/8", AnalysisEngine: "XG"}, {Move: "13/7", AnalysisEngine: "XG"}}
	if m := PlayedCandidate(&opening, moves, "13/8 8/7"); m == nil || m.Move != "13/7" {
		t.Errorf("chained hops: got %v, want 13/7", m)
	}
	if m := PlayedCandidate(&opening, moves, "6/5 13/8"); m == nil || m.Move != "13/8 6/5" {
		t.Errorf("parts reordered: got %v", m)
	}
	if m := PlayedCandidate(&opening, moves, "6/off 5/off"); m != nil {
		t.Errorf("unknown play: got %v, want none", m)
	}
	if m := PlayedCandidate(nil, moves, "13/8 8/7"); m == nil || m.Move != "13/7" {
		t.Errorf("without a position, the one candidate of its shape: got %v", m)
	}
	sameShape := []domain.CheckerMove{{Move: "6/5 5/1", AnalysisEngine: "GNUbg"}, {Move: "6/2 2/1", AnalysisEngine: "GNUbg"}}
	if m := PlayedCandidate(nil, sameShape, "6/1"); m != nil {
		t.Errorf("without a position, two candidates of one shape: got %v, want none", m)
	}
	if m := PlayedCandidate(nil, moves, "6/5 13/8"); m == nil || m.Move != "13/8 6/5" {
		t.Errorf("without a position, exact spelling: got %v", m)
	}

	// 4-1, a blot on 2, point 1 empty: "6/2 2/1" goes through 2, so it hits,
	// whatever its source wrote.
	blot2 := position41(2)
	gn := []domain.CheckerMove{{Move: "6/5 5/1", AnalysisEngine: "gammonNet v1.7.0"}, {Move: "6/2* 2/1", AnalysisEngine: "gammonNet v1.7.0"}}
	if m := PlayedCandidate(&blot2, gn, "6/2 2/1"); m == nil || m.Move != "6/2* 2/1" {
		t.Errorf("an unmarked played move follows its steps: got %v, want 6/2* 2/1", m)
	}
	if m := PlayedCandidate(&blot2, gn, "6/1"); m != nil {
		t.Errorf("6/1 may go through 5 or hit on 2: got %v, want none", m)
	}
	if m := PlayedCandidate(&blot2, gn, "6/5/1"); m == nil || m.Move != "6/5 5/1" {
		t.Errorf("through 5: got %v", m)
	}
	gnu := []domain.CheckerMove{{Move: "6/2 2/1", AnalysisEngine: "GNUbg"}}
	if m := PlayedCandidate(&blot2, gnu, "6/5* 5/1"); m != nil {
		t.Errorf("6/5* 5/1 hits on an empty point: got %v, want none", m)
	}
	blots52 := position41(5, 2)
	gnu = []domain.CheckerMove{{Move: "6/2 2/1", AnalysisEngine: "GNUbg"}, {Move: "6/5 5/1", AnalysisEngine: "GNUbg"}}
	if m := PlayedCandidate(&blots52, gnu, "6/5* 5/1"); m == nil || m.Move != "6/5 5/1" {
		t.Errorf("a hit on 5 is gnubg's 6/5 5/1, not its 6/2 2/1: got %v", m)
	}

	xg := []domain.CheckerMove{{Move: "6/5* 5/1*", AnalysisEngine: "XG"}, {Move: "6/2 2/1*", AnalysisEngine: "XG"}}
	two := prisesEn41()
	if m := PlayedCandidate(&two, xg, "6/2/1*"); m == nil || m.Move != "6/2 2/1*" {
		t.Errorf("one hit is not two: got %v, want 6/2 2/1*", m)
	}
	if m := PlayedCandidate(&two, xg, "6/2/1"); m == nil || m.Move != "6/2 2/1*" {
		t.Errorf("6/2/1 lands on the blot on 1: got %v, want 6/2 2/1*", m)
	}

	// 6-2, a blot on 7: gnubg's "13/7 13/11" hits without saying so.
	hit7 := domain.InitializePosition()
	hit7.Dice = [2]int{6, 2}
	hit7.Board.Points[7] = domain.Point{Checkers: 1, Color: domain.White}
	gn = []domain.CheckerMove{{Move: "24/18 13/11", AnalysisEngine: "gammonNet v1.7.0"}, {Move: "13/7* 13/11", AnalysisEngine: "gammonNet v1.7.0"}}
	if m := PlayedCandidate(&hit7, gn, "13/7 13/11"); m == nil || m.Move != "13/7* 13/11" {
		t.Errorf("an unmarked hit: got %v, want 13/7* 13/11", m)
	}
	merged := []domain.CheckerMove{{Move: "13/7* 13/11", AnalysisEngine: "XG"}, {Move: "13/7 13/11", AnalysisEngine: "GNUbg"}}
	if m := PlayedCandidate(&hit7, merged, "13/11 13/7"); m == nil || m.AnalysisEngine != "GNUbg" {
		t.Errorf("the engine written as the played move: got %v", m)
	}
	if m := PlayedCandidate(&hit7, merged, "13/11 13/7*"); m == nil || m.AnalysisEngine != "XG" {
		t.Errorf("same board, first listed: got %v", m)
	}
}

// TestPopulateAnalysisColumns_UnscoredMove: a played move no candidate names
// has an unknown cost, written NULL, never 0.
func TestPopulateAnalysisColumns_UnscoredMove(t *testing.T) {
	t.Parallel()
	e := 0.05
	a := &domain.PositionAnalysis{CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
		{Move: "13/9", Equity: 0.1}, {Move: "24/20", Equity: 0.05, EquityError: &e},
	}}}
	pos := domain.InitializePosition()
	pos.Dice = [2]int{4, 2}
	if c := PopulateAnalysisColumns(a, "6/2", "", &pos); !c.BestMoveUnscored || c.BestMoveErrorArg() != nil {
		t.Errorf("unknown played move: %+v", c)
	}
	if c := PopulateAnalysisColumns(a, "24/20", "", &pos); c.BestMoveUnscored || c.BestMoveErrorArg() != int64(50) {
		t.Errorf("scored played move: %+v", c)
	}
}
