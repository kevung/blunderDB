package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
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

// TestSimulate_SwissRoundsRefusedBeforeAnyRun: a Swiss by rounds whose rounds approach its
// field spends its time in the engine's pairing search, which eventCost does not see: the
// request is refused before anything is played when the field enters it first, and as the
// phase opens, at the first run, when an earlier phase picks its entrants.
func TestSimulate_SwissRoundsRefusedBeforeAnyRun(t *testing.T) {
	swiss := func(rounds int) string {
		return fmt.Sprintf(`{"name":"x","phases":[{"kind":"swiss_rounds","length":5,"rounds":%d}]}`, rounds)
	}
	for _, tc := range []struct {
		players, rounds int
		refused         bool
	}{
		{36, 35, true}, {34, 36, true}, {40, 99, true}, {37, 17, true}, {64, 17, true}, {13, 9, true}, {16, 11, true},
		{36, 16, false}, {64, 16, false}, {13, 8, false}, {16, 10, false}, {12, 99, false}, {8, 20, false},
	} {
		cfg, err := parseDirectionConfig(swiss(tc.rounds))
		if err != nil {
			t.Fatal(err)
		}
		err = boundSimulatedWork(cfg, tc.players)
		if tc.refused != errors.Is(err, ErrSimulationTooHeavy) {
			t.Errorf("%d players, %d rounds: %v; want refused %v before any run", tc.players, tc.rounds, err, tc.refused)
		}
		if err != nil && !strings.Contains(err.Error(), "rounds at most") {
			t.Errorf("%d players, %d rounds: %v; want the rounds this field allows", tc.players, tc.rounds, err)
		}
	}
	// 72 players in pools of 4, two qualifying: 36 enter the Swiss, whose 20 rounds are too many.
	const later = `{"name":"x","phases":[{"kind":"round_robin","length":1,"group_size":4,"qualifiers":2},{"kind":"swiss_rounds","length":1,"rounds":20}]}`
	cfg, err := parseDirectionConfig(later)
	if err != nil {
		t.Fatal(err)
	}
	if err := boundSimulatedWork(cfg, 72); err != nil {
		t.Fatalf("a later Swiss is not refused before its entrants are known: %v", err)
	}
	for _, runs := range []int{1, 30} {
		req := SimulateRequest{Config: json.RawMessage(later), Players: json.RawMessage("72"), Runs: runs}
		if got, err := Simulate(context.Background(), req); !errors.Is(err, ErrSimulationTooHeavy) || !strings.Contains(err.Error(), "36 players") {
			t.Errorf("%d runs: %+v (%v); want too heavy as the Swiss of 36 opens", runs, got, err)
		}
	}
}

// TestSimulate_SwissSearchWithinItsCount: the count is what a run copies; the engine's pairing
// search is apart, and runs out its 200 000 steps once for every rematch a round of a Swiss by
// rounds has to allow, at every event of that round (measured: about 1.1 ms an event and a
// rematch, the time of 200 000 steps). Within swissRoundsAt, no such search may outweigh the
// count; past it, the same measure catches it. The steps are counted from the rematches
// played, not timed.
func TestSimulate_SwissSearchWithinItsCount(t *testing.T) {
	if testing.Short() {
		t.Skip("plays a few dozen Swiss tournaments")
	}
	// searchShare is the searched steps over the count: a step about 5 ns, a unit of the count 10.
	searchShare := func(players, rounds int, seed int64) float64 {
		cfg, err := parseDirectionConfig(fmt.Sprintf(`{"name":"x","phases":[{"kind":"swiss_rounds","length":5,"rounds":%d}]}`, rounds))
		if err != nil {
			t.Fatal(err)
		}
		field, err := simulatedField(json.RawMessage(fmt.Sprint(players)), seed)
		if err != nil {
			t.Fatal(err)
		}
		met := map[[2]tournoi.PlayerID]bool{}
		rematches := map[int]int{}
		seen, cost, steps := 0, 0, 0
		r := sim.Run(cfg, field, sim.Options{Seed: seed, MaxSteps: simulateMaxSteps, MinPerPoint: cfg.MinPerPoint,
			Hook: func(s *tournoi.State, _ tournoi.Event) error {
				cost += eventCost(cfg, s)
				for ; seen < len(s.MatchOrder); seen++ {
					m := s.Matches[s.MatchOrder[seen]]
					pair := [2]tournoi.PlayerID{min(m.A, m.B), max(m.A, m.B)}
					if met[pair] {
						rematches[m.Round]++
					}
					met[pair] = true
				}
				steps += 200_000 * rematches[s.Phases[s.Current].Round]
				return nil
			}})
		if r.Err != nil {
			t.Fatalf("%d players, %d rounds: %v", players, rounds, r.Err)
		}
		return float64(steps) / 2 / float64(cost)
	}
	for _, players := range []int{13, 16, 25, 32, 37, 48} {
		rounds := swissRoundsAt(players)
		for seed := int64(1); seed <= 4; seed++ {
			if share := searchShare(players, rounds, seed); share > 1 {
				t.Errorf("%d players, %d rounds, seed %d: the search weighs %.1f times the count; want it under", players, rounds, seed, share)
			}
		}
	}
	// Past the bound the measure sees the search: 16 players at 16 rounds must replay someone.
	if share := searchShare(16, 16, 1); share <= 1 {
		t.Errorf("16 players, 16 rounds: the search weighs %.1f times the count; the measure should see it past the bound", share)
	}
}
