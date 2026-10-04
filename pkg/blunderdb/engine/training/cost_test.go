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

	cases := []struct {
		name string
		req  EvaluationRequest
	}{
		{"pool (race)", EvaluationRequest{Source: SourcePool}},
		{"board (contact)", EvaluationRequest{Source: SourceBoard, Seed: &seed}},
	}
	// The budget bounds what a question costs, not what a loaded machine
	// makes of it: a wall-clock mean over one run swallows every scheduler
	// stall that lands inside it. The cost is therefore the fastest of
	// several rounds, the one least disturbed by the load. The rounds of the
	// cases are interleaved, so a slow stretch of the machine that outlasts
	// one round cannot cover all the rounds of one case: it costs each case
	// a round or two, not its whole measure. A real regression (deeper truth,
	// searcher rebuilt per question) slows every round and still crosses the
	// budget.
	const draws, rounds = 20, 5
	per := make([]time.Duration, len(cases))
	distinct := make([]map[domain.Board]bool, len(cases))
	for i, c := range cases {
		generator.evaluation(c.req, rng, stopped) // warm the searchers in
		per[i] = time.Duration(1<<63 - 1)
		distinct[i] = make(map[domain.Board]bool, draws)
	}
	for r := 0; r < rounds; r++ {
		for ci, c := range cases {
			start := time.Now()
			for i := 0; i < draws; i++ {
				q := generator.evaluation(c.req, rng, stopped)
				if !q.Generated {
					t.Fatalf("%s draw %d refused: %q", c.name, i, q.Refusal)
				}
				distinct[ci][q.Position.Board] = true
			}
			if round := time.Since(start) / draws; round < per[ci] {
				per[ci] = round
			}
		}
	}
	for ci, c := range cases {
		t.Logf("%s: %v per question, fastest of %d rounds of %d draws (budget %v)", c.name, per[ci], rounds, draws, budgetPerQuestion)

		// A generator that returned one constant position would be fast and
		// pass a timing assertion on its own. It does not pass this one.
		if len(distinct[ci]) < draws/2 {
			t.Fatalf("%s: only %d distinct positions in %d draws: the generator is not generating", c.name, len(distinct[ci]), draws*rounds)
		}
		if per[ci] > budgetPerQuestion {
			t.Errorf("%s: a question costs %v, over the stated budget of %v", c.name, per[ci], budgetPerQuestion)
		}
	}
}
