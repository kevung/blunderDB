package engine

import (
	"math"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

func TestDecisionMargin(t *testing.T) {
	checker := func(eqs ...float64) *domain.PositionAnalysis {
		ca := &domain.CheckerAnalysis{}
		for _, e := range eqs {
			ca.Moves = append(ca.Moves, domain.CheckerMove{Equity: e})
		}
		return &domain.PositionAnalysis{AnalysisType: "CheckerMove", CheckerAnalysis: ca}
	}
	cube := &domain.PositionAnalysis{AnalysisType: "DoublingCube", DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{
		CubefulNoDoubleEquity: 0.600, CubefulDoubleTakeEquity: 0.640, CubefulDoublePassEquity: 1.000,
	}}
	tooGood := &domain.PositionAnalysis{AnalysisType: "DoublingCube", DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{
		CubefulNoDoubleEquity: 1.100, CubefulDoubleTakeEquity: 1.400, CubefulDoublePassEquity: 1.000,
	}}
	cases := []struct {
		name     string
		ana      *domain.PositionAnalysis
		response bool
		want     float64
		ok       bool
	}{
		{"two best, sorted", checker(0.120, 0.100, -0.300), false, 0.020, true},
		{"two best, unsorted", checker(-0.300, 0.100, 0.120), false, 0.020, true},
		{"tied best", checker(0.050, 0.050), false, 0, true},
		{"single candidate", checker(0.4), false, 0, false},
		{"no analysis", nil, false, 0, false},
		{"double: no double against double/take", cube, false, 0.040, true},
		{"take against pass", cube, true, 0.360, true},
		{"too good: no double against double/pass", tooGood, false, 0.100, true},
		{"cube analysis absent", &domain.PositionAnalysis{AnalysisType: "DoublingCube"}, false, 0, false},
	}
	for _, c := range cases {
		got, ok := DecisionMargin(c.ana, c.response)
		if ok != c.ok || math.Abs(got-c.want) > 1e-9 {
			t.Errorf("%s: got (%v, %v), want (%v, %v)", c.name, got, ok, c.want, c.ok)
		}
	}
}
