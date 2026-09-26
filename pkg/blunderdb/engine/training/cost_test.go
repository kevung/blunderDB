package training

import (
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
)

// budgetPerQuestion is the stated cost of one question — walk AND truth — on
// the reference machine (ADR-0041 rule 5: « the per-question cost against a
// stated threshold »). A serial judge at the canonical depth on the AVX2
// kernel costs about 65 ms (pool) and 96 ms (board) under load; the budget
// leaves three times that, so crossing it means a change of approach (deeper
// truth, searcher rebuilt per question, walk no longer 0-ply).
//
// Serial on purpose: sixteen workers cost the same, and a background question
// must not take every core from a running batch analysis.
const budgetPerQuestion = 300 * time.Millisecond

func TestTheCostOfAQuestionStaysUnderItsBudget(t *testing.T) {
	if testing.Short() || raceEnabled {
		t.Skip("timing measurement")
	}
	// The budget prices the vectorised kernel; the pure-Go fallback (arm64,
	// see kernel_noasm.go) is an order of magnitude slower, so stand aside
	// rather than state a second, unmeasured budget.

	if kernel := gammonnet.KernelName(); kernel == "go" {
		t.Skipf("kernel %q is the pure-Go fallback: the budget is stated for a vectorised one (#151)", kernel)
	}
	rng := seededRNG()
	// A clock that never moves: the walk's deadline never falls, so the
	// fallback cannot hide a slow walk behind pool shapes at zero plies.
	frozen := time.Now()
	stopped := func() time.Time { return frozen }
	seed := opening()

	for _, c := range []struct {
		name string
		req  EvaluationRequest
	}{
		{"pool (race)", EvaluationRequest{Source: SourcePool}},
		{"board (contact)", EvaluationRequest{Source: SourceBoard, Seed: &seed}},
	} {
		generator.evaluation(c.req, rng, stopped) // warm the searchers in
		const draws = 20
		distinct := make(map[domain.Board]bool, draws)
		start := time.Now()
		for i := 0; i < draws; i++ {
			q := generator.evaluation(c.req, rng, stopped)
			if !q.Generated {
				t.Fatalf("%s draw %d refused: %q", c.name, i, q.Refusal)
			}
			distinct[q.Position.Board] = true
		}
		per := time.Since(start) / draws
		t.Logf("%s: %v per question over %d draws (budget %v)", c.name, per, draws, budgetPerQuestion)

		// A generator that returned one constant position would be fast and
		// pass a timing assertion on its own. It does not pass this one.
		if len(distinct) < draws/2 {
			t.Fatalf("%s: only %d distinct positions in %d draws: the generator is not generating", c.name, len(distinct), draws)
		}
		if per > budgetPerQuestion {
			t.Errorf("%s: a question costs %v, over the stated budget of %v", c.name, per, budgetPerQuestion)
		}
	}
}
