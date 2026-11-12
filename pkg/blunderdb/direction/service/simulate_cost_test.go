package service

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"
	"time"

	tournoi "github.com/kevung/nicomaque"
	"github.com/kevung/nicomaque/sim"
)

// TestSimulate_VerdictOfTheFirstRun: the runs of a format vary with their seed, and near
// simulateMaxRunCost some cross it while others do not. The verdict is that of the first run
// alone, so the same request is admitted or refused whatever the budget lets the others
// reach: here a field whose third run crosses the bound is answered, at any budget, and one
// whose first run crosses it is refused, at any budget and for a single run.
func TestSimulate_VerdictOfTheFirstRun(t *testing.T) {
	if testing.Short() {
		t.Skip("plays a few dozen runs near the bound")
	}
	const config = `{"name":"x","phases":[{"kind":"round_robin","length":5,"group_size":4,"qualifiers":2},{"kind":"bracket","length":7}]}`
	cfg, err := parseDirectionConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	costs := func(seed int64, runs int) []int {
		players, err := simulatedField(json.RawMessage("228"), seed)
		if err != nil {
			t.Fatal(err)
		}
		out := make([]int, runs)
		for i := range out {
			sim.Run(cfg, players, sim.Options{Seed: seed + int64(i), MaxSteps: simulateMaxSteps, MinPerPoint: cfg.MinPerPoint,
				Hook: func(s *tournoi.State, _ tournoi.Event) error { out[i] += eventCost(cfg, s); return nil }})
		}
		return out
	}
	const admitted, refused = 18, 0
	if c := costs(admitted, 3); !(c[0] <= simulateMaxRunCost && c[2] > simulateMaxRunCost && slices.Max(c) <= 2*simulateMaxRunCost) {
		t.Fatalf("seed %d costs %v; the test wants a first run under %d and the third over it", admitted, c, simulateMaxRunCost)
	}
	if c := costs(refused, 1); c[0] <= simulateMaxRunCost {
		t.Fatalf("seed %d costs %v; the test wants a first run over %d", refused, c, simulateMaxRunCost)
	}
	was := SimulateBudget
	t.Cleanup(func() { SimulateBudget = was })
	for _, budget := range []time.Duration{time.Nanosecond, 2 * time.Second, time.Minute} {
		SimulateBudget = budget
		req := SimulateRequest{Config: json.RawMessage(config), Players: json.RawMessage("228"), Seed: admitted, Runs: 6}
		got, err := Simulate(context.Background(), req)
		if err != nil || got.Runs < 1 {
			t.Errorf("budget %v, seed %d: %+v (%v); want an answer", budget, admitted, got, err)
		}
		if budget == time.Minute && (got == nil || got.Runs != 6) {
			t.Errorf("budget %v, seed %d: %+v; want its 6 runs", budget, admitted, got)
		}
		for _, runs := range []int{1, 30} {
			req := SimulateRequest{Config: json.RawMessage(config), Players: json.RawMessage("228"), Seed: refused, Runs: runs}
			if got, err := Simulate(context.Background(), req); !errors.Is(err, ErrSimulationTooHeavy) {
				t.Errorf("budget %v, seed %d, %d runs: %+v (%v); want too heavy", budget, refused, runs, got, err)
			}
		}
	}
}
