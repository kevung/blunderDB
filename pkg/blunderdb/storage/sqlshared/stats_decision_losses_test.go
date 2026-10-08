package sqlshared

import (
	"math"
	"reflect"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// The options a difficulty weighs are the deciding player's: the doubler's
// no double and double, the answerer's take and pass, a checker decision's
// candidates.
func TestDecisionCosts(t *testing.T) {
	cube := &domain.PositionAnalysis{DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{
		CubefulNoDoubleEquity: 0.5, CubefulNoDoubleError: -0.25,
		CubefulDoubleTakeEquity: 0.75, CubefulDoubleTakeError: 0,
		CubefulDoublePassEquity: 1.0, CubefulDoublePassError: 0.25,
	}}
	for _, c := range []struct {
		action string
		want   []float64
	}{
		{"No Double", []float64{0.25, 0}},
		{"Double", []float64{0.25, 0}},
		{"Take", []float64{0, 0.25}},
		{"Pass", []float64{0, 0.25}},
		{"", nil},
	} {
		if got := decisionCosts(cube, "cube", c.action); !reflect.DeepEqual(got, c.want) {
			t.Errorf("cube %q: got %v, want %v", c.action, got, c.want)
		}
	}

	e1, e2 := -0.03, 0.05
	checker := &domain.PositionAnalysis{CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
		{Move: "a"}, {Move: "b", EquityError: &e1}, {Move: "c"}, {Move: "d", EquityError: &e2},
	}}}
	if got := decisionCosts(checker, "checker", ""); !reflect.DeepEqual(got, []float64{0, 0.03, 0.05}) {
		t.Errorf("checker: got %v (a candidate without a cost is left out)", got)
	}
	if decisionCosts(nil, "checker", "") != nil || decisionCosts(cube, "checker", "") != nil || decisionCosts(checker, "cube", "Take") != nil {
		t.Error("a decision its analysis does not price has no costs")
	}

	// A position only a rollout analysed is priced from that rollout, as its
	// loss is (ColumnSource, ADR-0060).
	rolled := &domain.PositionAnalysis{Rollouts: []domain.RolloutAnalysis{{
		Kind:       domain.RolloutKindMoves,
		Candidates: []domain.RolloutCandidate{{Move: "a", Equity: 0.1}, {Move: "b", Equity: 0.04}},
	}}}
	if got := decisionCosts(rolled, "checker", ""); len(got) != 2 || got[0] != 0 || math.Abs(got[1]-0.06) > 1e-12 {
		t.Errorf("rollout only: got %v, want [0 0.06]", got)
	}
}
