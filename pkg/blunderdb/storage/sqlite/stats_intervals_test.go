package sqlite

import (
	"context"
	"math"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The statistics resample matches (ADR-0078): the global PR's interval
// counts the demo's matches and holds the PR; a per-match row of a selection
// without a player gathers two seats, so its L7 has no interval; and L7
// splits into checker and cube parts that add up to it.
func TestStatsIntervalsResampleMatches(t *testing.T) {
	ctx := context.Background()
	w, err := Open(ctx, demoCopy(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	got, err := w.Stats().Compute(ctx, "", storage.StatsFilter{DecisionType: -1})
	if err != nil {
		t.Fatal(err)
	}
	iv := got.PRInterval
	if !iv.Available || iv.Units != got.Totals.NumMatches || iv.Low > got.PRGlobal || iv.High < got.PRGlobal {
		t.Fatalf("PR %.3f, interval %+v over %d matches", got.PRGlobal, iv, got.Totals.NumMatches)
	}
	for _, ms := range got.PerMatch {
		if ms.MWC7.HasInterval {
			t.Errorf("match %d: two seats pooled into an interval %+v", ms.ID, ms.MWC7)
		}
	}
	if sum := got.MWC7Checker.Loss + got.MWC7Cube.Loss; math.Abs(sum-got.MWC7.Loss) > 1e-9 {
		t.Errorf("checker %.5f + cube %.5f != L7 %.5f", got.MWC7Checker.Loss, got.MWC7Cube.Loss, got.MWC7.Loss)
	}
	for _, c := range got.PerPhase {
		if c.PRInterval.Available && (c.PRInterval.Low > c.PR || c.PRInterval.High < c.PR) {
			t.Errorf("phase %s: PR %.3f outside %+v", c.Phase, c.PR, c.PRInterval)
		}
	}
}
