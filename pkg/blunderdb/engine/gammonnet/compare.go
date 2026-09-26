package gammonnet

import (
	"errors"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// What gammonNet is worth on a user's own library: where it disagrees with
// the imported analyses, and what the disagreement costs.
//
// This file holds the comparison (what counts as the same answer, pricing,
// aggregation); each storage mode gathers its own ids and feeds CompareOne.
// Nothing here writes: ADR-0013 protects an imported analysis
// unconditionally, so the comparison can run on any library.

// AnalysisComparison is what one comparison sweep found.
//
// Rates are over the decisions actually compared: a position with no stored
// move, or that gammonNet declines, is counted (Refused) and left out.
type AnalysisComparison struct {
	// Compared is the number of decisions both engines answered.
	Compared int `json:"compared"`
	// Refused is how many positions gammonNet declined (a match score beyond
	// its match equity table, a cube state it refuses). Not a failure.
	Refused int `json:"refused"`
	// Failed is how many could not be loaded or evaluated for another reason.
	Failed int `json:"failed"`

	// SameBest counts the decisions where both engines name the same best
	// move, or the same cube action. It is the headline: how often the two
	// would have you play the same thing.
	SameBest int `json:"sameBest"`
	// Checker/Cube split the same count by kind of decision, because the two
	// have nothing to do with each other and a single rate hides which is
	// which.
	CheckerCompared int `json:"checkerCompared"`
	CheckerSameBest int `json:"checkerSameBest"`
	CubeCompared    int `json:"cubeCompared"`
	CubeSameBest    int `json:"cubeSameBest"`

	// CostQuantiles are the distribution of what following gammonNet would
	// have cost ACCORDING TO THE STORED ANALYSIS, in its units (ADR-0019):
	// zero when they agree. Priced in one direction only, since pricing a
	// disagreement twice invites reading the smaller number.
	CostMean float64 `json:"costMean"`
	CostP50  float64 `json:"costP50"`
	CostP95  float64 `json:"costP95"`
	CostMax  float64 `json:"costMax"`
	// OverThreshold counts the decisions whose cost exceeds
	// ComparisonBlunderThreshold — the disagreements worth looking at.
	OverThreshold int `json:"overThreshold"`

	// ByPhase splits the disagreement rate by the position's derived phase
	// (ADR-0035), which is what answers "concentrated where". Keyed by the
	// phase's stable token.
	ByPhase map[string]ComparisonBucket `json:"byPhase"`

	// Worst lists the most expensive disagreements, worst first.
	Worst []ComparisonDisagreement `json:"worst,omitempty"`
}

// ComparisonBucket is one slice of a comparison.
type ComparisonBucket struct {
	Compared int     `json:"compared"`
	SameBest int     `json:"sameBest"`
	CostMean float64 `json:"costMean"`
}

// ComparisonDisagreement is one decision the two engines answered differently.
type ComparisonDisagreement struct {
	PositionID int64   `json:"positionId"`
	Kind       string  `json:"kind"` // "checker" or "cube"
	Phase      string  `json:"phase"`
	Stored     string  `json:"stored"`    // the imported engine's answer
	GammonNet  string  `json:"gammonNet"` // gammonNet's answer
	Engine     string  `json:"engine"`    // who wrote the stored analysis
	Cost       float64 `json:"cost"`      // priced on the stored analysis's scale
}

// ComparisonBlunderThreshold is the equity above which a disagreement is worth
// a user's attention — the same 0.05 integration_gate_test.go blocks on.
const ComparisonBlunderThreshold = 0.05

// MaxComparisonDisagreements is how many disagreements the report names. Ten:
// enough to see whether they share a shape, few enough to read.
const MaxComparisonDisagreements = 10

// CompareOne evaluates one position and compares gammonNet's answer with the
// stored one. It never writes.
func CompareOne(pos *domain.Position, stored *domain.PositionAnalysis, id int64, searcher *Searcher, ply, pruneK, candidates int) ComparisonSample {
	if pos == nil || stored == nil {
		return ComparisonSample{outcome: GNFailed}
	}
	result, err := EvaluatePositionWith(searcher, *pos, ply, pruneK, candidates)
	if err != nil {
		if errors.Is(err, ErrNotEvaluable) {
			return ComparisonSample{outcome: GNRefused}
		}
		slog.Warn("gammonnet comparison: evaluating a position failed", "position_id", id, "error", err)
		return ComparisonSample{outcome: GNFailed}
	}

	dis := ComparisonDisagreement{
		PositionID: id,
		Phase:      engine.ClassifyGamePhase(pos).String(),
	}

	switch {
	case len(result.Moves) > 0 && stored.CheckerAnalysis != nil && len(stored.CheckerAnalysis.Moves) > 0:
		return compareCheckerAnswer(stored, result.Moves[0].Move, dis.Phase, id)

	case result.Cube != nil && stored.DoublingCubeAnalysis != nil:
		dis.Kind = "cube"
		dis.Engine = stored.DoublingCubeAnalysis.AnalysisEngine
		dis.GammonNet = result.Cube.BestCubeAction
		dis.Stored = stored.DoublingCubeAnalysis.BestCubeAction
		same := cubeActionKind(dis.GammonNet) == cubeActionKind(dis.Stored)
		if !same {
			dis.Cost = cubeDisagreementCost(stored.DoublingCubeAnalysis, dis.GammonNet)
		}
		return ComparisonSample{dis: dis, same: same, cube: true}
	}
	// One side answered a kind of decision the other did not: nothing to
	// compare, and not a failure of either.
	return ComparisonSample{outcome: GNRefused}
}

// compareCheckerAnswer compares one checker decision: the stored best move
// against the move gammonNet prefers, both read through engine.CanonicalMove
// so a difference of dialect is never reported as a difference of opinion.
func compareCheckerAnswer(stored *domain.PositionAnalysis, ours, phase string, id int64) ComparisonSample {
	moves := stored.CheckerAnalysis.Moves
	dis := ComparisonDisagreement{
		PositionID: id,
		Kind:       "checker",
		Phase:      phase,
		Engine:     moves[0].AnalysisEngine,
		Stored:     moves[0].Move,
		GammonNet:  ours,
	}
	same := engine.CanonicalMove(ours) == engine.CanonicalMove(dis.Stored)
	if !same {
		dis.Cost = checkerDisagreementCost(moves, ours)
	}
	return ComparisonSample{dis: dis, same: same}
}

// checkerDisagreementCost prices gammonNet's preferred move on the STORED
// analysis's scale: the stored best move's equity minus the stored equity of
// the move gammonNet prefers.
//
// A move the stored analysis does not list cannot be priced and costs 0, a
// known limit; candidacy is integration_gate_test.go's criterion 3.
func checkerDisagreementCost(moves []domain.CheckerMove, want string) float64 {
	if len(moves) == 0 {
		return 0
	}
	best := moves[0].Equity
	target := engine.CanonicalMove(want)
	for _, m := range moves {
		if engine.CanonicalMove(m.Move) == target {
			return math.Max(0, best-m.Equity)
		}
	}
	return 0
}

// cubeDisagreementCost prices gammonNet's cube action on the stored analysis's
// scale, from the three cubeful equities the stored analysis carries.
func cubeDisagreementCost(a *domain.DoublingCubeAnalysis, want string) float64 {
	// Engines word their actions differently: match on the substance.
	equity := func(action string) (float64, bool) {
		switch cubeActionKind(action) {
		case cubeNoDouble:
			return a.CubefulNoDoubleEquity, true
		case cubeDoubleTake:
			return a.CubefulDoubleTakeEquity, true
		case cubeDoublePass:
			return a.CubefulDoublePassEquity, true
		}
		return 0, false
	}
	got, okGot := equity(want)
	best, okBest := equity(a.BestCubeAction)
	if !okGot || !okBest {
		return 0
	}
	return math.Max(0, best-got)
}

// String renders a comparison as the block the CLI prints.
func (c AnalysisComparison) String() string {
	if c.Compared == 0 {
		return "No decision could be compared."
	}
	pct := func(n, d int) string {
		if d == 0 {
			return "—"
		}
		return fmt.Sprintf("%.1f%% (%d/%d)", 100*float64(n)/float64(d), n, d)
	}
	s := fmt.Sprintf("compared: %d decision(s)  (refused %d, failed %d)\n", c.Compared, c.Refused, c.Failed)
	s += fmt.Sprintf("same best answer: %s\n", pct(c.SameBest, c.Compared))
	if c.CheckerCompared > 0 {
		s += fmt.Sprintf("  checker play:   %s\n", pct(c.CheckerSameBest, c.CheckerCompared))
	}
	if c.CubeCompared > 0 {
		s += fmt.Sprintf("  cube decision:  %s\n", pct(c.CubeSameBest, c.CubeCompared))
	}
	s += "cost of following gammonNet, on the imported analysis's own scale:\n"
	s += fmt.Sprintf("  mean %.4f  median %.4f  p95 %.4f  max %.4f\n", c.CostMean, c.CostP50, c.CostP95, c.CostMax)
	s += fmt.Sprintf("  above %.2f: %s\n", ComparisonBlunderThreshold, pct(c.OverThreshold, c.Compared))
	if len(c.ByPhase) > 0 {
		s += "by phase:\n"
		phases := make([]string, 0, len(c.ByPhase))
		for p := range c.ByPhase {
			phases = append(phases, p)
		}
		sort.Strings(phases)
		for _, p := range phases {
			b := c.ByPhase[p]
			s += fmt.Sprintf("  %-12s %s  mean cost %.4f\n", p, pct(b.SameBest, b.Compared), b.CostMean)
		}
	}
	if len(c.Worst) > 0 {
		s += "worst disagreements:\n"
		for i, d := range c.Worst {
			s += fmt.Sprintf("  %2d. position %d (%s, %s)  %s said %q, gammonNet says %q  cost %.4f\n",
				i+1, d.PositionID, d.Kind, d.Phase, d.Engine, d.Stored, d.GammonNet, d.Cost)
		}
	}
	return s
}

// cubeActionKind folds an engine's cube-action wording ("No Double",
// "No double", "Double, take"…) onto the three substantive answers. "Too
// good" folds into "no double", as integration_gate_test.go buckets it.
func cubeActionKind(action string) int {
	a := strings.ToLower(strings.TrimSpace(action))
	switch {
	case strings.Contains(a, "too good"), strings.HasPrefix(a, "no double"),
		strings.HasPrefix(a, "no redouble"), strings.HasPrefix(a, "nodouble"):
		return cubeNoDouble
	case strings.Contains(a, "pass"), strings.Contains(a, "drop"):
		return cubeDoublePass
	case strings.Contains(a, "take"), strings.Contains(a, "beaver"):
		return cubeDoubleTake
	}
	return cubeUnknown
}

const (
	cubeUnknown = iota
	cubeNoDouble
	cubeDoubleTake
	cubeDoublePass
)

// ComparisonSample is one compared decision, produced by CompareOne and
// folded by Aggregate; its fields are this file's affair.
type ComparisonSample struct {
	outcome int
	dis     ComparisonDisagreement
	same    bool
	cube    bool
}

// Outcome codes for a comparison sample, mirroring the analysis batch's
// split: a legitimately declined position is REFUSED, not failed.
const (
	// GNCompared: both engines answered, and the sample carries the comparison.
	GNCompared = iota
	// GNRefused: nothing to compare — the engine declined the position, or the
	// two answered different kinds of decision.
	GNRefused
	// GNFailed: the position could not be loaded or evaluated.
	GNFailed
)

// Aggregate folds a stream of samples into one comparison, so both storage
// modes count the same way.
func Aggregate(samples []ComparisonSample) AnalysisComparison {
	cmp := AnalysisComparison{ByPhase: map[string]ComparisonBucket{}}
	var costs []float64
	phaseCost := map[string]float64{}
	for _, res := range samples {
		switch res.outcome {
		case GNRefused:
			cmp.Refused++
			continue
		case GNFailed:
			cmp.Failed++
			continue
		}
		cmp.Compared++
		if res.cube {
			cmp.CubeCompared++
		} else {
			cmp.CheckerCompared++
		}
		if res.same {
			cmp.SameBest++
			if res.cube {
				cmp.CubeSameBest++
			} else {
				cmp.CheckerSameBest++
			}
		}
		costs = append(costs, res.dis.Cost)
		if res.dis.Cost > ComparisonBlunderThreshold {
			cmp.OverThreshold++
		}
		b := cmp.ByPhase[res.dis.Phase]
		b.Compared++
		if res.same {
			b.SameBest++
		}
		cmp.ByPhase[res.dis.Phase] = b
		phaseCost[res.dis.Phase] += res.dis.Cost
		if !res.same {
			cmp.Worst = append(cmp.Worst, res.dis)
		}
	}

	for phase, b := range cmp.ByPhase {
		if b.Compared > 0 {
			b.CostMean = phaseCost[phase] / float64(b.Compared)
			cmp.ByPhase[phase] = b
		}
	}
	if len(costs) > 0 {
		sort.Float64s(costs)
		var sum float64
		for _, c := range costs {
			sum += c
		}
		cmp.CostMean = sum / float64(len(costs))
		cmp.CostP50 = costs[len(costs)*50/100]
		cmp.CostP95 = costs[min(len(costs)*95/100, len(costs)-1)]
		cmp.CostMax = costs[len(costs)-1]
	}
	sort.Slice(cmp.Worst, func(i, j int) bool {
		if cmp.Worst[i].Cost != cmp.Worst[j].Cost {
			return cmp.Worst[i].Cost > cmp.Worst[j].Cost
		}
		return cmp.Worst[i].PositionID < cmp.Worst[j].PositionID
	})
	if len(cmp.Worst) > MaxComparisonDisagreements {
		cmp.Worst = cmp.Worst[:MaxComparisonDisagreements]
	}
	return cmp
}
