package training

import (
	"math"
	"testing"
	"time"

	"github.com/kevung/blunderdb/internal/cputime"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
)

// budgetPerQuestion is the stated cost of one question — walk AND truth —
// in units of the reference evaluation (ADR-0041 rule 5: « the per-question
// cost against a stated threshold »): one judge evaluation of the opening at
// the canonical depth, measured on the same machine in the same round. A
// question is one such truth plus a 0-ply walk, about 0.6 reference for the
// pool and 1.7 for the board; four leaves more than twice that, so crossing it means a
// change of approach (deeper truth, searcher rebuilt per question, walk no
// longer 0-ply).
//
// A ratio, not milliseconds: a machine that is slower, hyperthreaded or busy
// with other packages slows the reference as much as the question, where an
// absolute threshold would measure the machine.
//
// Serial on purpose: sixteen workers cost the same, and a background question
// must not take every core from a running batch analysis.
const budgetPerQuestion = 4.0

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
	refPos := opening()
	// ratio is the cost of a question over the reference measured just before
	// it, the lowest of the rounds.
	ratio := make([]float64, len(cases))
	per := make([]time.Duration, len(cases))
	distinct := make([]map[domain.Board]bool, len(cases))
	for i, c := range cases {
		generator.evaluation(c.req, rng, stopped) // warm the searchers in
		per[i] = time.Duration(1<<63 - 1)
		ratio[i] = math.Inf(1)
		distinct[i] = make(map[domain.Board]bool, draws)
	}
	for r := 0; r < rounds; r++ {
		for ci, c := range cases {
			const refRuns = 2
			refClock := cputime.Start()
			for i := 0; i < refRuns; i++ {
				if _, err := gammonnet.EvaluatePositionWith(generator.judge, refPos, gammonnet.DefaultPly, gammonnet.DefaultPruneK, 0); err != nil {
					t.Fatalf("reference evaluation: %v", err)
				}
			}
			ref := refClock.Elapsed() / refRuns
			clock := cputime.Start()
			unit = clock.Unit()
			for i := 0; i < draws; i++ {
				q := generator.evaluation(c.req, rng, stopped)
				if !q.Generated {
					t.Fatalf("%s draw %d refused: %q", c.name, i, q.Refusal)
				}
				distinct[ci][q.Position.Board] = true
			}
			round := clock.Elapsed() / draws
			per[ci] = min(per[ci], round)
			if ref > 0 {
				ratio[ci] = min(ratio[ci], float64(round)/float64(ref))
			}
		}
	}
	for ci, c := range cases {
		t.Logf("%s: %v of %s per question, %.2f reference evaluations, fastest of %d rounds of %d draws (budget %.1f)", c.name, per[ci], unit, ratio[ci], rounds, draws, budgetPerQuestion)

		// A generator that returned one constant position would be fast and
		// pass a timing assertion on its own. It does not pass this one.
		if len(distinct[ci]) < draws/2 {
			t.Fatalf("%s: only %d distinct positions in %d draws: the generator is not generating", c.name, len(distinct[ci]), draws*rounds)
		}
		if ratio[ci] > budgetPerQuestion {
			t.Errorf("%s: a question costs %.2f reference evaluations, over the stated budget of %.1f", c.name, ratio[ci], budgetPerQuestion)
		}
	}
}
