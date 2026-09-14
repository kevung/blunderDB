package training

import (
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// budgetPerQuestion is the stated cost of one question — walk AND truth — on
// the reference machine (ADR-0041 rule 5: « the per-question cost against a
// stated threshold »). Measured on 2026-09-14 on sixteen cores under a load
// of 9.5: 65 ms from the pool (a race), 96 ms from the board (contact), for
// a serial judge at the canonical depth. The budget leaves three times that:
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
