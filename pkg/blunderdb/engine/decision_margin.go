package engine

import (
	"math"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// DecisionMargin is how far the best choice of a decision stands from the
// next best, in the analysis' own equity: the difficulty of the position,
// whatever was played in it. A checker play compares its two best candidates.
// A cube decision is the doubler's (no double against the double, worth
// min(take, pass) since the opponent answers best) unless isResponse, then the
// responder's (take against pass). ok is false without an analysis of the
// decision, or with a single candidate: nothing to choose between.
func DecisionMargin(ana *domain.PositionAnalysis, isResponse bool) (float64, bool) {
	if ana == nil {
		return 0, false
	}
	switch {
	case ana.AnalysisType == "CheckerMove" && ana.CheckerAnalysis != nil:
		moves := ana.CheckerAnalysis.Moves
		if len(moves) < 2 {
			return 0, false
		}
		best, next := math.Inf(-1), math.Inf(-1)
		for _, m := range moves {
			switch {
			case m.Equity > best:
				best, next = m.Equity, best
			case m.Equity > next:
				next = m.Equity
			}
		}
		return best - next, true
	case ana.AnalysisType == "DoublingCube" && ana.DoublingCubeAnalysis != nil:
		d := ana.DoublingCubeAnalysis
		if isResponse {
			return math.Abs(d.CubefulDoubleTakeEquity - d.CubefulDoublePassEquity), true
		}
		double := math.Min(d.CubefulDoubleTakeEquity, d.CubefulDoublePassEquity)
		return math.Abs(d.CubefulNoDoubleEquity - double), true
	}
	return 0, false
}
