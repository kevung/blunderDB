package training

import (
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
)

// budgetPerQuestion is the stated cost of one question — walk AND truth — on
// the reference machine (ADR-0041 rule 5: « the per-question cost against a
// stated threshold »). Measured on 2026-09-14 on sixteen cores under a load
// of 9.5, on the AVX2 kernel: 65 ms from the pool (a race), 96 ms from the
// board (contact), for a serial judge at the canonical depth. The kernel is
// part of the measurement, not a detail of it — see the skip below. The
// budget leaves three times that:
// crossing it is a change of approach — a deeper truth, a searcher rebuilt per
// question, a walk that stopped being 0-ply — not a busy afternoon.
//
// Serial on purpose, and measured so: the same judge on sixteen workers cost
// the same (64 ms, 109 ms). A question is prepared in the background while
// the user thinks; a search that took every core for nothing would stack on
// whatever else is running, a batch analysis first.
const budgetPerQuestion = 300 * time.Millisecond

func TestTheCostOfAQuestionStaysUnderItsBudget(t *testing.T) {
	if testing.Short() || raceEnabled {
		t.Skip("timing measurement")
	}
	// The budget prices the vectorised kernel. Without one the pure-Go twin
	// evaluates the same network an order of magnitude slower, so the figure
	// would measure the missing kernel rather than the cost of a question:
	// 2.6 s and 4.8 s on macos-latest, which is arm64 and has no NEON kernel
	// yet (#151, and the comment in kernel_noasm.go). Stand aside rather than
	// state a second budget nobody measured — windows-latest and every x86
	// machine still hold the assertion.
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
