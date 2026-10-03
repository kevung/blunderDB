package rollout

import (
	"context"
	"math"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
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

// gaugePairs come from testdata/test.xg (game, move): five of the thirty
// XG Roller++ decisions with a gap under 0.06 measured for ADR-0060.
var gaugePairs = []xgPair{
	{"XGID=-a-B-aD-C---dD---bbeB-----:0:0:-1:41:0:0:0:7:0", "24/20 8/7", "24/20 7/6", -0.243, -0.273},      // g0 m10
	{"XGID=--cB-BBBBA---B-b-bAbc-cA--:1:-1:-1:41:1:0:0:7:0", "10/6 3/2*", "10/6 8/7*", -0.656, -0.693},     // g1 m34
	{"XGID=aCCCCAa-------------cbbbd-:1:-1:1:63:1:0:0:7:0", "5/off 3/off", "5/off 4/1", 1.262, 1.235},      // g1 m77
	{"XGID=-CCDaB-----a--aab--bcbBbA-:1:1:-1:32:1:0:0:5:0", "21/19 14/11", "11/8 10/8", 0.981, 0.957},      // g4 m129
	{"XGID=-DEC-------AA------bbd-baA:1:1:1:41:1:0:0:5:0", "bar/24* 12/8", "bar/24* 11/7", -0.797, -0.823}, // g4 m164
}

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
// truncated rollouts. On the thirty pairs of ADR-0060 the gap between two
// plays differs from XG's by 0.0066 on average (0.032 at worst) where
// gammonNet 2-ply differs by 0.0146; the bound below leaves room for that
// worst case, not for a change of scale or of sign.
func TestGaugeAgainstXGRollouts(t *testing.T) {
	heavy(t)
	for _, p := range gaugePairs {
		ro := rollPair(t, p)
		gap := ro[p.best].Equity - ro[p.second].Equity
		xgGap := p.xgBest - p.xgSecond
		if math.Abs(gap-xgGap) > 0.035 {
			t.Errorf("%s: rollout gap %+.4f, XG Roller++ gap %+.4f", p.xgid, gap, xgGap)
		}
		if gap <= 0 {
			t.Errorf("%s: rollout prefers %s, XG prefers %s by %.3f", p.xgid, p.second, p.best, xgGap)
		}
		if d := math.Abs(ro[p.best].Equity - p.xgBest); d > 0.06 {
			t.Errorf("%s: rollout equity %+.4f, XG %+.4f", p.xgid, ro[p.best].Equity, p.xgBest)
		}
	}
}

// TestRolloutOverturnsTheSearch: gammonNet 2-ply prefers 11/8 10/8 by 0.06;
// XG Roller++ prefers 21/19 14/11 by 0.024. The rollout must side with XG,
// and say so with a JSD that separates the two.
func TestRolloutOverturnsTheSearch(t *testing.T) {
	heavy(t)
	p := gaugePairs[3]
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
