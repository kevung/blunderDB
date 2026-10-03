package rollout

import (
	"context"
	"math"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// xgPair is a checker decision of testdata/test.xg that XG analysed with
// XG Roller++ (360 truncated games with variance reduction), its two best
// plays and their equities as XG stored them.
type xgPair struct {
	xgid         string
	best, second string
	xgBest       float64
	xgSecond     float64
}

// gaugeSize is how many XG Roller++ decisions the gauge rolls out.
const gaugeSize = 30

// gaugePairs reads the first gaugeSize checker decisions of
// testdata/test.xg whose two best plays XG rolled out (XG Roller++) less
// than 0.06 apart, and whose notation blunderDB's legal plays share.
func gaugePairs(t *testing.T) []xgPair {
	t.Helper()
	graph, err := ingest.MapXG("../../../../testdata/test.xg")
	if err != nil {
		t.Fatal(err)
	}
	var pairs []xgPair
	for _, game := range graph.Games {
		for _, mv := range game.Moves {
			if mv.Position == nil {
				continue
			}
			var ca *domain.CheckerAnalysis
			for _, a := range mv.Analyses {
				if a.CheckerAnalysis != nil {
					ca = a.CheckerAnalysis
				}
			}
			if ca == nil || len(ca.Moves) < 2 {
				continue
			}
			m0, m1 := ca.Moves[0], ca.Moves[1]
			if m0.AnalysisDepth != "XG Roller++" || m1.AnalysisDepth != "XG Roller++" || m0.Equity-m1.Equity > 0.06 {
				continue
			}
			legal := map[string]bool{}
			for _, p := range domain.LegalMoves(mv.Position) {
				legal[p.Notation] = true
			}
			if !legal[m0.Move] || !legal[m1.Move] {
				continue
			}
			pairs = append(pairs, xgPair{"XGID=" + domain.EncodeXGID(mv.Position), m0.Move, m1.Move, m0.Equity, m1.Equity})
			if len(pairs) == gaugeSize {
				return pairs
			}
		}
	}
	t.Fatalf("testdata/test.xg holds %d gauge decisions, want %d", len(pairs), gaugeSize)
	return nil
}

// overturnPair (test.xg, game 4, move 129): gammonNet 2-ply prefers
// 11/8 10/8 by 0.06, XG Roller++ prefers 21/19 14/11 by 0.024.
var overturnPair = xgPair{"XGID=-CCDaB-----a--aab--bcbBbA-:1:1:-1:32:1:0:0:5:0", "21/19 14/11", "11/8 10/8", 0.981, 0.957}

func heavy(t *testing.T) {
	t.Helper()
	if testing.Short() || raceEnabled {
		t.Skip("full rollouts: skipped under -short and -race")
	}
}

func rollPair(t *testing.T, p xgPair) map[string]Candidate {
	t.Helper()
	r, err := Run(context.Background(), decode(t, p.xgid), Fast(), Options{Moves: []string{p.best, p.second}, NoBearoffTable: true})
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]Candidate{}
	for _, c := range r.Candidates {
		out[c.Move] = c
	}
	return out
}

// TestGaugeAgainstXGRollouts measures the Fast preset against XG's own
// truncated rollouts (ADR-0060). On the thirty decisions the gap between the
// two plays differs from XG's by 0.0066 on average where gammonNet 2-ply
// differs by 0.0146; the bound sits between the two. Wherever XG separates
// the plays by 0.01 or more, or the rollout separates them at JSD 3, the
// order must be XG's.
func TestGaugeAgainstXGRollouts(t *testing.T) {
	heavy(t)
	sum := 0.0
	pairs := gaugePairs(t)
	for _, p := range pairs {
		ro := rollPair(t, p)
		gap := ro[p.best].Equity - ro[p.second].Equity
		xgGap := p.xgBest - p.xgSecond
		sum += math.Abs(gap - xgGap)
		separated := ro[p.best].JSD >= 3 || ro[p.second].JSD >= 3
		if (xgGap >= 0.01 || separated) && gap <= 0 {
			t.Errorf("%s: rollout prefers %s (gap %+.4f), XG prefers %s by %.3f", p.xgid, p.second, gap, p.best, xgGap)
		}
	}
	if mean := sum / float64(len(pairs)); mean > 0.015 {
		t.Errorf("mean gap difference to XG Roller++ %.4f over %d decisions, want <= 0.015", mean, len(pairs))
	} else {
		t.Logf("mean gap difference to XG Roller++ %.4f over %d decisions", mean, len(pairs))
	}
}

// TestRolloutOverturnsTheSearch: gammonNet 2-ply prefers 11/8 10/8 by 0.06;
// XG Roller++ prefers 21/19 14/11 by 0.024. The rollout must side with XG,
// and say so with a JSD that separates the two.
func TestRolloutOverturnsTheSearch(t *testing.T) {
	heavy(t)
	p := overturnPair
	pos := decode(t, p.xgid)
	eval, err := gammonnet.EvaluatePosition(pos, 2, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	searched := map[string]float64{}
	for _, m := range eval.Moves {
		searched[m.Move] = m.Equity
	}
	if searched[p.second] <= searched[p.best] {
		t.Fatalf("2-ply already prefers %s (%+.4f vs %+.4f): the case no longer tests a reversal", p.best, searched[p.best], searched[p.second])
	}
	ro := rollPair(t, p)
	if ro[p.best].Equity <= ro[p.second].Equity || ro[p.second].JSD < 3 {
		t.Fatalf("rollout %s %+.4f, %s %+.4f (JSD %.2f): want %s ahead with JSD >= 3",
			p.best, ro[p.best].Equity, p.second, ro[p.second].Equity, ro[p.second].JSD, p.best)
	}
}
