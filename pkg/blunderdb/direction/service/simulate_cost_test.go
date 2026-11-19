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

// clubField is n players of whom club share one club, and the others a club each.
func clubField(n, club int) json.RawMessage {
	var b []string
	for i := range n {
		c := fmt.Sprintf("C%d", i)
		if i < club {
			c = "A"
		}
		b = append(b, fmt.Sprintf(`{"id":"p%d","club":%q}`, i, c))
	}
	return json.RawMessage("[" + strings.Join(b, ",") + "]")
}

// TestSimulate_SearchCountedNotTimed: a club holding more than half the field with avoid_clubs
// sends the engine's pairing search after a round it never finds. Its events alone stay under
// simulateMaxRunCost; the steps the engine counts carry it past, and the request is refused on
// that count at any budget, the time the machine takes deciding nothing.
func TestSimulate_SearchCountedNotTimed(t *testing.T) {
	const config = `{"name":"x","phases":[{"kind":"swiss_rounds","length":5,"rounds":9,"avoid_clubs":true}]}`
	players := clubField(100, 60)
	cfg, err := parseDirectionConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	field, err := simulatedField(players, 0)
	if err != nil {
		t.Fatal(err)
	}
	search, cost := &tournoi.SearchBudget{}, 0
	r := sim.Run(cfg, field, sim.Options{Seed: 0, MaxSteps: simulateMaxSteps, MinPerPoint: cfg.MinPerPoint, Search: search,
		Hook: func(s *tournoi.State, _ tournoi.Event) error { cost += eventCost(cfg, s); return nil }})
	if r.Err != nil {
		t.Fatal(r.Err)
	}
	if searched := int(search.Used) * simulateSearchStepCost; cost > simulateMaxRunCost || cost+searched <= simulateMaxRunCost {
		t.Fatalf("events %d, search %d; the test wants the events under %d and the search past it", cost, searched, simulateMaxRunCost)
	}
	saved := SimulateBudget
	t.Cleanup(func() { SimulateBudget = saved })
	for _, budget := range []time.Duration{time.Nanosecond, time.Hour} {
		SimulateBudget = budget
		req := SimulateRequest{Config: json.RawMessage(config), Players: players, Runs: 30}
		if got, err := Simulate(context.Background(), req); !errors.Is(err, ErrSimulationTooHeavy) {
			t.Errorf("budget %v: %+v (%v); want too heavy", budget, got, err)
		}
	}
}

// TestSimulate_SwissRoundsAnswerToTheCount: no bound on a Swiss by rounds' rounds stands
// before the count. A Swiss as long as a large field is refused on the steps its search
// spends; one whose search ends early, a small field's among them, is answered at any number
// of rounds.
func TestSimulate_SwissRoundsAnswerToTheCount(t *testing.T) {
	for _, tc := range []struct {
		players, rounds int
		refused         bool
	}{
		{36, 35, true}, {40, 99, true},
		{12, 99, false}, {37, 17, false}, {16, 11, false}, {13, 9, false}, {8, 20, false},
	} {
		config := fmt.Sprintf(`{"name":"x","phases":[{"kind":"swiss_rounds","length":5,"rounds":%d}]}`, tc.rounds)
		req := SimulateRequest{Config: json.RawMessage(config), Players: json.RawMessage(fmt.Sprint(tc.players)), Runs: 1}
		got, err := Simulate(context.Background(), req)
		if refused := errors.Is(err, ErrSimulationTooHeavy); refused != tc.refused || (!refused && err != nil) {
			t.Errorf("%d players, %d rounds: %+v (%v); want refused %v", tc.players, tc.rounds, got, err, tc.refused)
		}
	}
}

// TestSimulate_OrdinarySwissAdmitted: the Swiss of a club or an open, 32 to 200 players in 5
// to 11 rounds, with or without a fifth of the field sharing a club under avoid_clubs, is
// answered. A bound that refuses them bounds the wrong thing.
func TestSimulate_OrdinarySwissAdmitted(t *testing.T) {
	if testing.Short() {
		t.Skip("plays some forty first runs")
	}
	for _, n := range []int{32, 64, 100, 128, 200} {
		for _, rounds := range []int{5, 7, 9, 11} {
			for _, clubs := range []bool{false, true} {
				config := fmt.Sprintf(`{"name":"x","phases":[{"kind":"swiss_rounds","length":5,"rounds":%d,"avoid_clubs":%v}]}`, rounds, clubs)
				players := json.RawMessage(fmt.Sprint(n))
				if clubs {
					players = clubField(n, n/5)
				}
				req := SimulateRequest{Config: json.RawMessage(config), Players: players, Runs: 1}
				if got, err := Simulate(context.Background(), req); err != nil || got.Runs != 1 {
					t.Errorf("%d players, %d rounds, avoid_clubs %v: %+v (%v); want answered", n, rounds, clubs, got, err)
				}
			}
		}
	}
}
