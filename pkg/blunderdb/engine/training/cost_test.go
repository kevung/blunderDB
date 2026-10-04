package training

import (
	"testing"
	"time"

	"github.com/kevung/blunderdb/internal/cputime"
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
	// makes of it. `go test ./...` runs other packages beside this one, and on
	// a runner with few cores a heavy neighbour takes a share of the processor
	// for the whole measure: a wall clock then charges every round with the
	// wait for a core. The cost is therefore the processor time the process
	// received (cputime), which a wait does not grow, and the fastest of
	// several interleaved rounds, which sheds what remains of the noise
	// (caches, a sibling hyperthread). A real regression (deeper truth,
	// searcher rebuilt per question, a fan-out to every core) costs more
	// processor time in every round and still crosses the budget.
	const draws, rounds = 20, 5
	unit := ""
	per := make([]time.Duration, len(cases))
	distinct := make([]map[domain.Board]bool, len(cases))
	for i, c := range cases {
		generator.evaluation(c.req, rng, stopped) // warm the searchers in
		per[i] = time.Duration(1<<63 - 1)
		distinct[i] = make(map[domain.Board]bool, draws)
	}
	for r := 0; r < rounds; r++ {
		for ci, c := range cases {
			clock := cputime.Start()
			unit = clock.Unit()
			for i := 0; i < draws; i++ {
				q := generator.evaluation(c.req, rng, stopped)
				if !q.Generated {
					t.Fatalf("%s draw %d refused: %q", c.name, i, q.Refusal)
				}
				distinct[ci][q.Position.Board] = true
			}
			if round := clock.Elapsed() / draws; round < per[ci] {
				per[ci] = round
			}
		}
	}
	for ci, c := range cases {
		t.Logf("%s: %v of %s per question, fastest of %d rounds of %d draws (budget %v)", c.name, per[ci], unit, rounds, draws, budgetPerQuestion)

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
