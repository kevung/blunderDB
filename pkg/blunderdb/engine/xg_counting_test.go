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

// TestPlayedCandidate matches a play spelt two ways, and refuses a canonical
// match that names two candidates (the hit markers it drops told them apart).
func TestPlayedCandidate(t *testing.T) {
	t.Parallel()
	moves := []domain.CheckerMove{{Move: "13/6"}, {Move: "24/20 13/10"}, {Move: "8/4* 4/1"}, {Move: "8/4 4/1"}}
	if m := PlayedCandidate(moves, "13/9 9/6"); m == nil || m.Move != "13/6" {
		t.Errorf("chained hops: got %v, want 13/6", m)
	}
	if m := PlayedCandidate(moves, "13/10 24/20"); m == nil || m.Move != "24/20 13/10" {
		t.Errorf("parts reordered: got %v", m)
	}
	if m := PlayedCandidate(moves, "8/4* 4/1"); m == nil || m.Move != "8/4* 4/1" {
		t.Errorf("exact spelling with a hit: got %v", m)
	}
	if m := PlayedCandidate(moves, "8/1"); m != nil {
		t.Errorf("ambiguous canonical match: got %v, want none", m)
	}
	if m := PlayedCandidate(moves, "6/off 5/off"); m != nil {
		t.Errorf("unknown play: got %v, want none", m)
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
	if c := PopulateAnalysisColumns(a, "6/2", "", 5); !c.BestMoveUnscored || c.BestMoveErrorArg() != nil {
		t.Errorf("unknown played move: %+v", c)
	}
	if c := PopulateAnalysisColumns(a, "24/20", "", 5); c.BestMoveUnscored || c.BestMoveErrorArg() != int64(50) {
		t.Errorf("scored played move: %+v", c)
	}
}
