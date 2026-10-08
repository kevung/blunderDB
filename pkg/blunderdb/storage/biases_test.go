package storage

import (
	"math"
	"testing"
)

// TestSignedBias_IntervalAndVerdict pins the estimator of ADR-0078: the mean
// of the signs, its normal interval, and no direction below the minimum.
func TestSignedBias_IntervalAndVerdict(t *testing.T) {
	var b SignedBias
	for i := 0; i < 40; i++ {
		switch {
		case i < 10:
			b.add(1, 100)
		case i < 12:
			b.add(-1, 50)
		default:
			b.add(0, 0)
		}
	}
	b.measure()
	want := 8.0 / 40
	half := StudyPlanZ * math.Sqrt((12.0/40-want*want)/40)
	if math.Abs(b.Bias-want) > 1e-12 || math.Abs(b.Low-(want-half)) > 1e-12 {
		t.Errorf("Bias %g Low %g; want %g, %g", b.Bias, b.Low, want, want-half)
	}
	if b.Verdict != BiasTooMuch || b.PlusMP != 1000 || b.MinusMP != 100 {
		t.Errorf("got %+v", b)
	}
	var small SignedBias
	for i := 0; i < BiasMinDecisions-1; i++ {
		small.add(1, 10)
	}
	small.measure()
	if small.Verdict != BiasInsufficient {
		t.Errorf("Verdict %q below the minimum, want insufficient", small.Verdict)
	}
}

// TestBuildDirectionalBiases_Axes: a take or a pass lands on the take/pass
// axis, an offer on the doubling axis and in its score cell, and an unread
// play is counted apart.
func TestBuildDirectionalBiases_Axes(t *testing.T) {
	got := BuildDirectionalBiases([]BiasCubeRow{
		{Best: "Double, Pass", Played: "Take", ErrorMP: 200},
		{Best: "Double, Take", Played: "Pass", ErrorMP: 100},
		{Best: "No Double", Played: "Double", MoverAway: 3, OpponentAway: 5, ErrorMP: 80},
		{Best: "Double, Take", Played: "No Double", MoverAway: 3, OpponentAway: 5, ErrorMP: 60},
		{Best: "Double, Take", Played: "Double", MoverAway: 2, OpponentAway: 2},
	}, []BiasCheckerRow{{Sign: 1, ErrorMP: 70}, {}, {Unread: true}})
	if got.TakePass.Decisions != 2 || got.TakePass.Plus != 1 || got.TakePass.Minus != 1 || got.TakePass.PlusMP != 200 {
		t.Errorf("TakePass %+v", got.TakePass)
	}
	if got.Doubles.Decisions != 3 || got.Doubles.Plus != 1 || got.Doubles.Minus != 1 {
		t.Errorf("Doubles %+v", got.Doubles)
	}
	if len(got.DoublesByScore) != 2 || got.DoublesByScore[0].MoverAway != 2 || got.DoublesByScore[1].Decisions != 2 {
		t.Errorf("DoublesByScore %+v", got.DoublesByScore)
	}
	if got.Blots.Decisions != 2 || got.Blots.Plus != 1 || got.BlotsUnread != 1 {
		t.Errorf("Blots %+v unread %d", got.Blots, got.BlotsUnread)
	}
}
