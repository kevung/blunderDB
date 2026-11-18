package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/direction/service"
)

const simulatedSwiss = `{"name":"Open","tables":{"count":8},"phases":[{"kind":"swiss_lives","length":7,"lives":2,"mode":"continuous"}]}`

func simulateReq(players string) service.SimulateRequest {
	return service.SimulateRequest{Config: json.RawMessage(simulatedSwiss), Players: json.RawMessage(players),
		Start: "2026-10-01T09:00:00Z", Runs: 20, Seed: 42}
}

// TestSimulate_SameSeedSameAnswer: a simulation is a function of its request, and its figures
// hold together.
func TestSimulate_SameSeedSameAnswer(t *testing.T) {
	a, err := service.Simulate(context.Background(), simulateReq("32"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := service.Simulate(context.Background(), simulateReq("32"))
	if err != nil || !reflect.DeepEqual(a, b) {
		t.Fatalf("the same request answered %+v then %+v (%v)", a, b, err)
	}
	other := simulateReq("32")
	other.Seed = 43
	if c, err := service.Simulate(context.Background(), other); err != nil || reflect.DeepEqual(a, c) {
		t.Errorf("another seed answered the same %+v (%v)", c, err)
	}

	if a.Runs != 20 || a.Matches < 31 || a.Warnings == nil {
		t.Errorf("simulation = %+v; want 20 runs, at least 31 matches for 32 players with two lives, a list of warnings", a)
	}
	if !(a.MinutesP10 > 0 && a.MinutesP10 <= a.MinutesP50 && a.MinutesP50 <= a.MinutesP90) {
		t.Errorf("minutes = %v, %v, %v; want positive and ordered", a.MinutesP10, a.MinutesP50, a.MinutesP90)
	}
	if a.PeakTables != 16 || !slices.Contains(a.Warnings, service.SimulateWarnTablesExceeded) {
		t.Errorf("peakTables = %d, warnings %v; want the 16 matches of a first round of 32, and the warning that 8 tables are fewer", a.PeakTables, a.Warnings)
	}
	if a.MatchesPerPlayerMin < 2 || a.MatchesPerPlayerMax < a.MatchesPerPlayerMin {
		t.Errorf("matches per player = %d to %d; want at least the two a player with two lives plays", a.MatchesPerPlayerMin, a.MatchesPerPlayerMax)
	}
	start, _ := time.Parse(time.RFC3339, "2026-10-01T09:00:00Z")
	want := start.Add(time.Duration(a.MinutesP50 * float64(time.Minute))).Format(time.RFC3339)
	if a.EstimatedEnd != want {
		t.Errorf("estimatedEnd = %q; want the start plus the median, %q", a.EstimatedEnd, want)
	}
}

// TestSimulate_TablesAndField: a room large enough raises no warning; a list of entries is
// played as given, and no start means no estimated end.
func TestSimulate_TablesAndField(t *testing.T) {
	wide := simulateReq("32")
	wide.Tables = 16
	if w, err := service.Simulate(context.Background(), wide); err != nil || w.PeakTables != 16 || len(w.Warnings) != 0 {
		t.Errorf("on sixteen tables: %+v (%v); want a peak of 16 and no warning", w, err)
	}

	list := simulateReq(`[{"id":"a","name":"A","rating":4},{"id":"b","name":"B"},{"id":"c","name":"C","rating":8},{"id":"d","name":"D"}]`)
	list.Start = ""
	l, err := service.Simulate(context.Background(), list)
	if err != nil || l.EstimatedEnd != "" || l.Matches < 3 {
		t.Errorf("a list of four = %+v (%v); want it played, with no estimated end", l, err)
	}
}

// TestSimulate_Bounds: what would hold the processor or the memory is refused, never clamped,
// and each refusal names its own bound.
func TestSimulate_Bounds(t *testing.T) {
	config := func(phase string) func(*service.SimulateRequest) {
		return func(r *service.SimulateRequest) {
			r.Config = json.RawMessage(`{"name":"x","phases":[` + phase + `]}`)
		}
	}
	players := func(raw string) func(*service.SimulateRequest) {
		return func(r *service.SimulateRequest) { r.Players = json.RawMessage(raw) }
	}
	for name, c := range map[string]struct {
		change func(*service.SimulateRequest)
		want   string
	}{
		"too many runs":      {func(r *service.SimulateRequest) { r.Runs = service.SimulateMaxRuns + 1 }, "1 to 200 runs, not 201"},
		"negative runs":      {func(r *service.SimulateRequest) { r.Runs = -1 }, "1 to 200 runs, not -1"},
		"too many players":   {players("513"), "2 to 512 players, not 513"},
		"one player":         {players("1"), "2 to 512 players, not 1"},
		"negative players":   {players("-3"), "2 to 512 players, not -3"},
		"a list too long":    {players("[" + strings.Repeat(`{"id":"a"},`, service.SimulateMaxPlayers) + `{"id":"b"}]`), "2 to 512 players, not 513"},
		"a list too heavy":   {players(`[{"id":"a"},{"id":"` + strings.Repeat("b", 1<<20) + `"}]`), "players is larger than"},
		"no players":         {func(r *service.SimulateRequest) { r.Players = nil }, "needs players"},
		"players a string":   {players(`"many"`), "neither a number nor a list"},
		"players a fraction": {players("9e18"), "neither a number nor a list"},
		"twice the same id":  {players(`[{"id":"a"},{"id":"a"}]`), "an id of its own"},
		"negative tables":    {func(r *service.SimulateRequest) { r.Tables = -1 }, "0 to 4096 tables, not -1"},
		"too many tables":    {func(r *service.SimulateRequest) { r.Tables = 1000000000 }, "0 to 4096 tables, not 1000000000"},
		"a start no time":    {func(r *service.SimulateRequest) { r.Start = "tomorrow" }, "RFC 3339"},
		"no config":          {func(r *service.SimulateRequest) { r.Config = nil }, "empty configuration"},
		"an unknown format":  {config(`{"kind":"nope","length":5}`), ""},
		"a length too long":  {config(`{"kind":"bracket","length":1000000000}`), "length is 0 to 99, not 1000000000"},
		"a final too long":   {config(`{"kind":"bracket","length":5,"final_length":100}`), "final_length is 0 to 99, not 100"},
		"a round too long":   {config(`{"kind":"bracket","length":5,"lengths":[7,1000]}`), "lengths is 0 to 99, not 1000"},
		"too many lives":     {config(`{"kind":"swiss_lives","length":5,"lives":1000000000}`), "lives is 0 to 100, not 1000000000"},
		"too many phases":    {config(strings.TrimSuffix(strings.Repeat(`{"kind":"bracket","length":5},`, 17), ",")), "at most 16 phases, not 17"},
		"a config too heavy": {func(r *service.SimulateRequest) {
			r.Config = json.RawMessage(`{"name":"` + strings.Repeat("x", 1<<20) + `","phases":[{"kind":"bracket","length":5}]}`)
		}, "config is larger than"},
	} {
		req := simulateReq("16")
		c.change(&req)
		got, err := service.Simulate(context.Background(), req)
		if !errors.Is(err, direction.ErrRefused) {
			t.Errorf("%s: answered %+v (%v); want a refusal", name, got, err)
		} else if !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: refused with %q; want its own bound, %q", name, err, c.want)
		}
	}
}

// TestSimulate_HugeFieldAllocatesNothing: the number of players comes from the request, so it
// is refused before anything is sized by it.
func TestSimulate_HugeFieldAllocatesNothing(t *testing.T) {
	for _, n := range []string{"3000000000", "1000000000000", "9000000000000000000"} {
		req := service.SimulateRequest{Config: json.RawMessage(`{"name":"x","phases":[{"kind":"bracket","length":5}]}`), Players: json.RawMessage(n)}
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		_, err := service.Simulate(context.Background(), req)
		runtime.ReadMemStats(&after)
		if !errors.Is(err, direction.ErrRefused) || !strings.Contains(err.Error(), "2 to 512 players, not "+n) {
			t.Errorf("%s players: %v; want the refusal of the bound", n, err)
		}
		if grew := after.TotalAlloc - before.TotalAlloc; grew > 1<<20 {
			t.Errorf("%s players: refused after allocating %d bytes; want nothing sized by the request", n, grew)
		}
	}
}

// TestSimulate_TimeBudget: past its computing time a simulation stops, but never before its
// first run, which the budget does not decide: with no time at all a request is answered on
// that one run, under the warning time_budget. A caller that left gets its own cancellation
// back.
func TestSimulate_TimeBudget(t *testing.T) {
	was := service.SimulateBudget
	t.Cleanup(func() { service.SimulateBudget = was })
	service.SimulateBudget = time.Nanosecond
	req := simulateReq("128")
	req.Runs = service.SimulateMaxRuns
	got, err := service.Simulate(context.Background(), req)
	if err != nil || got.Runs != 1 || !slices.Contains(got.Warnings, service.SimulateWarnTimeBudget) {
		t.Errorf("with no time: %+v (%v); want the first run, under time_budget", got, err)
	}
	service.SimulateBudget = time.Minute
	req.Runs = 20
	if got, err := service.Simulate(context.Background(), req); err != nil || got.Runs != 20 {
		t.Errorf("the same request with its time: %+v (%v); want its 20 runs", got, err)
	}
	gone, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.Simulate(gone, simulateReq("256")); !errors.Is(err, context.Canceled) {
		t.Errorf("a caller that left: %v; want its cancellation, not a verdict on the request", err)
	}
}

// unhurried gives the simulations of a test the time a slow machine needs: what it checks
// is the answer, not the budget.
func unhurried(t *testing.T) {
	t.Helper()
	was := service.SimulateBudget
	t.Cleanup(func() { service.SimulateBudget = was })
	service.SimulateBudget = time.Minute
}

// TestSimulate_AcceptsWhatCreationAccepts: directions.create bounds neither the lives nor the
// field; a Swiss of 20 lives and a large bracket are simulated, and the largest field is
// read, its bracket answered as too heavy rather than refused as malformed.
func TestSimulate_AcceptsWhatCreationAccepts(t *testing.T) {
	unhurried(t)
	req := simulateReq("8")
	req.Config = json.RawMessage(strings.Replace(simulatedSwiss, `"lives":2`, `"lives":20`, 1))
	req.Runs = 1
	if got, err := service.Simulate(context.Background(), req); err != nil || got.Runs != 1 {
		t.Errorf("a Swiss of 20 lives: %+v (%v); want one run", got, err)
	}
	req = service.SimulateRequest{Config: json.RawMessage(`{"name":"x","phases":[{"kind":"bracket","length":5}]}`), Players: json.RawMessage("256"), Runs: 1}
	if got, err := service.Simulate(context.Background(), req); err != nil || got.Matches != 255 {
		t.Errorf("a bracket of 256: %+v (%v); want its 255 matches", got, err)
	}
	req.Players = json.RawMessage("512")
	if got, err := service.Simulate(context.Background(), req); !errors.Is(err, service.ErrSimulationTooHeavy) || errors.Is(err, direction.ErrRefused) {
		t.Errorf("a bracket of 512: %+v (%v); want too heavy, not a refusal of its shape", got, err)
	}
}

// TestSimulate_NothingToSay: a format that plays no match with this field, and a tournament
// whose end has no date, are refused rather than answered with figures that say nothing.
func TestSimulate_NothingToSay(t *testing.T) {
	unhurried(t)
	req := simulateReq("2")
	req.Config = json.RawMessage(`{"name":"x","phases":[{"kind":"gsl","length":7,"target":512}]}`)
	if got, err := service.Simulate(context.Background(), req); !errors.Is(err, direction.ErrRefused) || !strings.Contains(err.Error(), "plays no match with 2 players") {
		t.Errorf("two players in a GSL aiming at 512: %+v (%v); want the refusal of a format that plays no match", got, err)
	}
	req = simulateReq("16")
	req.Start = "9999-12-31T23:30:00Z"
	if got, err := service.Simulate(context.Background(), req); !errors.Is(err, direction.ErrRefused) || !strings.Contains(err.Error(), "leaves no date") {
		t.Errorf("a start at the end of the calendar: %+v (%v); want a refusal, not a date RFC 3339 cannot write", got, err)
	}
	req = simulateReq("8")
	req.Config = json.RawMessage(`{"name":"x","min_per_point":60,"phases":[{"kind":"swiss_lives","length":99,"lives":50,"mode":"continuous"}]}`)
	req.Runs = 1
	if got, err := service.Simulate(context.Background(), req); !errors.Is(err, direction.ErrRefused) || !strings.Contains(err.Error(), "more than a year") {
		t.Errorf("a tournament of years: %+v (%v); want a refusal", got, err)
	}
}

// TestSimulate_PaceOfTheConfiguration: the lengths are those of the pace the configuration
// announces, as the clock of a live tournament reads it — not of the engine's default.
func TestSimulate_PaceOfTheConfiguration(t *testing.T) {
	unhurried(t)
	p50 := func(pace string) float64 {
		req := service.SimulateRequest{Config: json.RawMessage(`{"name":"x","min_per_point":` + pace + `,"phases":[{"kind":"bracket","length":7}]}`),
			Players: json.RawMessage("16"), Runs: 20, Seed: 7}
		got, err := service.Simulate(context.Background(), req)
		if err != nil {
			t.Fatalf("min_per_point %s: %v", pace, err)
		}
		return got.MinutesP50
	}
	quick, plain, slow := p50("4"), p50("8"), p50("16")
	if !(quick > 0 && math.Abs(plain/quick-2) < 0.01 && math.Abs(slow/quick-4) < 0.01) {
		t.Errorf("p50 = %.0f, %.0f, %.0f minutes at 4, 8, 16 minutes a point; want it to follow the pace", quick, plain, slow)
	}
	req := service.SimulateRequest{Config: json.RawMessage(`{"name":"x","phases":[{"kind":"bracket","length":7}]}`), Players: json.RawMessage("16"), Runs: 20, Seed: 7}
	if got, err := service.Simulate(context.Background(), req); err != nil || got.MinutesP50 != plain {
		t.Errorf("no min_per_point: %+v (%v); want the engine's 8 minutes a point, %.0f minutes", got, err, plain)
	}
}

// TestSimulate_BreaksMoveTheEnd: with a start, the declared breaks push the lengths and the
// estimated end, as they push the forecast of a live tournament; with no start a break, which
// is a date, is met by nothing.
func TestSimulate_BreaksMoveTheEnd(t *testing.T) {
	unhurried(t)
	const bracket = `"phases":[{"kind":"bracket","length":7}]}`
	lunch := `{"name":"x","breaks":[{"start":"2026-10-01T09:30:00Z","end":"2026-10-01T11:00:00Z"}],` + bracket
	run := func(config, start string) *service.Simulation {
		got, err := service.Simulate(context.Background(), service.SimulateRequest{Config: json.RawMessage(config),
			Players: json.RawMessage("16"), Runs: 20, Seed: 7, Start: start})
		if err != nil {
			t.Fatalf("%s from %q: %v", config, start, err)
		}
		return got
	}
	plain := run(`{"name":"x",`+bracket, "2026-10-01T09:00:00Z")
	paused := run(lunch, "2026-10-01T09:00:00Z")
	later := func(paused, plain float64) bool { return math.Abs(paused-plain-90) < 1e-6 }
	if plain.MinutesP10 <= 30 || !later(paused.MinutesP10, plain.MinutesP10) || !later(paused.MinutesP50, plain.MinutesP50) || !later(paused.MinutesP90, plain.MinutesP90) {
		t.Errorf("with a break of 90 minutes the lengths are %v, %v, %v; without, %v, %v, %v: want each 90 minutes longer",
			paused.MinutesP10, paused.MinutesP50, paused.MinutesP90, plain.MinutesP10, plain.MinutesP50, plain.MinutesP90)
	}
	start, _ := time.Parse(time.RFC3339, "2026-10-01T09:00:00Z")
	if want := start.Add(time.Duration(paused.MinutesP50 * float64(time.Minute))).Format(time.RFC3339); paused.EstimatedEnd != want || paused.EstimatedEnd == plain.EstimatedEnd {
		t.Errorf("estimatedEnd = %q with the break, %q without; want %q", paused.EstimatedEnd, plain.EstimatedEnd, want)
	}
	if got := run(lunch, ""); got.MinutesP50 != plain.MinutesP50 {
		t.Errorf("with no start the break changed the median: %v, not %v", got.MinutesP50, plain.MinutesP50)
	}
}

// TestSimulate_BudgetIsTwoSeconds: the computing time of one simulation is what the daemon's
// gate and its hosts count on.
func TestSimulate_BudgetIsTwoSeconds(t *testing.T) {
	if service.SimulateBudget != 2*time.Second {
		t.Errorf("SimulateBudget = %v; want 2s", service.SimulateBudget)
	}
}

// TestSimulate_WorkWithinTheBounds: at the budget of production, the heaviest requests the
// bounds admit are answered, or refused as too heavy at their first run, and hold little
// memory; the requests whose single draw would outgrow both are refused before anything is
// played. The pools are those the engine draws: as many as the size asks, the field dealt
// evenly between them.
func TestSimulate_WorkWithinTheBounds(t *testing.T) {
	if testing.Short() {
		t.Skip("plays the budget of production several times")
	}
	rr := func(size int, more string) string {
		return fmt.Sprintf(`{"name":"x","phases":[{"kind":"round_robin","length":1,"group_size":%d,"qualifiers":1%s}]}`, size, more)
	}
	const refusal = "matches a phase"
	for name, tc := range map[string]struct {
		config, players string
		refused         bool
	}{
		"one pool of 91":                   {rr(91, ""), "91", false},
		"100 players in pools of 91":       {rr(91, ""), "100", false},
		"100 players in pools of 92":       {rr(92, ""), "100", false},
		"pools of 17 for the full field":   {rr(17, ""), "512", false},
		"91 players in one group":          {`{"name":"x","phases":[{"kind":"round_robin","length":1,"groups":1}]}`, "91", false},
		"a Swiss of 100 lives by rounds":   {`{"name":"x","phases":[{"kind":"swiss_lives","lives":100,"length":1,"mode":"rounds","pairing":"wins"}]}`, "512", false},
		"a Swiss of 100 lives by batches":  {`{"name":"x","phases":[{"kind":"swiss_lives","lives":100,"length":1,"batch_minutes":1}]}`, "512", false},
		"a double elimination":             {`{"name":"x","phases":[{"kind":"bracket","length":1,"consolation":true,"last_chance":true,"reconciliation":true,"recharge":true}]}`, "512", false},
		"one pool of 92":                   {rr(92, ""), "92", true},
		"one pool of the full field":       {rr(512, ""), "512", true},
		"one group of the full field":      {`{"name":"x","phases":[{"kind":"round_robin","length":1,"groups":1}]}`, "512", true},
		"92 players in one group":          {`{"name":"x","phases":[{"kind":"round_robin","length":1,"groups":1}]}`, "92", true},
		"pools of 18 for the full field":   {rr(18, ""), "512", true},
		"pools of 92 entered later by 100": {`{"name":"x","phases":[{"kind":"round_robin","length":1,"group_size":2,"qualifiers":1},{"kind":"round_robin","length":1,"group_size":92,"qualifiers":1,"entry":"all"}]}`, "100", true},
		"a large pool entered later": {`{"name":"x","phases":[{"kind":"round_robin","length":1,"group_size":2,"qualifiers":1},` +
			`{"kind":"round_robin","length":1,"group_size":512,"qualifiers":1,"entry":"all"}]}`, "512", true},
	} {
		req := service.SimulateRequest{Config: json.RawMessage(tc.config), Players: json.RawMessage(tc.players), Runs: 1}
		runtime.GC()
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		base, peak := m.HeapAlloc, m.HeapAlloc
		stop, sampled := make(chan struct{}), make(chan struct{})
		go func() {
			defer close(sampled)
			tick := time.NewTicker(2 * time.Millisecond)
			defer tick.Stop()
			for {
				select {
				case <-stop:
					return
				case <-tick.C:
					var m runtime.MemStats
					runtime.ReadMemStats(&m)
					peak = max(peak, m.HeapAlloc)
				}
			}
		}()
		began := time.Now()
		_, err := service.Simulate(context.Background(), req)
		elapsed := time.Since(began)
		close(stop)
		<-sampled
		grew := (peak - base) >> 20
		t.Logf("%s: %v, %d MiB of heap (%v)", name, elapsed.Round(time.Millisecond), grew, err)
		if tc.refused {
			if !errors.Is(err, direction.ErrRefused) || !strings.Contains(err.Error(), refusal) || elapsed > 100*time.Millisecond {
				t.Errorf("%s: %v after %v and %d MiB of heap; want the refusal of the phase's matches, at once", name, err, elapsed, grew)
			}
			continue
		}
		if err != nil && !errors.Is(err, service.ErrSimulationTooHeavy) {
			t.Errorf("%s: %v; want an answer, or too heavy", name, err)
		}
		if err != nil && strings.Contains(err.Error(), refusal) {
			t.Errorf("%s: %v; the pools the engine draws are within the bound", name, err)
		}
		if over := elapsed - service.SimulateBudget; over > 250*time.Millisecond && !raceEnabled {
			t.Errorf("%s: answered %v past the budget; want a run cut within a quarter of a second", name, over)
		}
		if grew > 64 {
			t.Errorf("%s: held %d MiB of heap; want 64 at most", name, grew)
		}
	}
}

// TestSimulate_UsefulOrTooHeavy: whatever the time, a format is answered on every run asked
// for, or refused as too heavy by its work — the same answer on every machine, which the
// time budget does not decide.
func TestSimulate_UsefulOrTooHeavy(t *testing.T) {
	if testing.Short() {
		t.Skip("plays some three hundred runs")
	}
	unhurried(t)
	for name, tc := range map[string]struct {
		config, players string
		heavy           bool
	}{
		"a club of 40 in pools of 4":  {`{"name":"x","phases":[{"kind":"round_robin","length":5,"group_size":4,"qualifiers":2},{"kind":"bracket","length":7}]}`, "40", false},
		"one pool of 25":              {`{"name":"x","phases":[{"kind":"round_robin","length":5,"groups":1}]}`, "25", false},
		"a bracket of 256":            {`{"name":"x","phases":[{"kind":"bracket","length":7}]}`, "256", false},
		"a Swiss of 2 lives for 128":  {simulatedSwiss, "128", false},
		"a club of 40 in one pool":    {`{"name":"x","phases":[{"kind":"round_robin","length":5,"groups":1}]}`, "40", true},
		"200 players in pools of 4":   {`{"name":"x","phases":[{"kind":"round_robin","length":5,"group_size":4,"qualifiers":2},{"kind":"bracket","length":7}]}`, "200", false},
		"one pool of 30":              {`{"name":"x","phases":[{"kind":"round_robin","length":5,"groups":1}]}`, "30", false},
		"a double elimination of 128": {`{"name":"x","phases":[{"kind":"bracket","length":1,"consolation":true,"last_chance":true,"reconciliation":true,"recharge":true}]}`, "128", false},
		"a double elimination of 256": {`{"name":"x","phases":[{"kind":"bracket","length":1,"consolation":true,"last_chance":true,"reconciliation":true,"recharge":true}]}`, "256", true},
		"a bracket of 512":            {`{"name":"x","phases":[{"kind":"bracket","length":7}]}`, "512", true},
		"a Swiss of 7 rounds for 128": {`{"name":"x","phases":[{"kind":"swiss_rounds","length":5,"rounds":7}]}`, "128", false},
		"a Swiss of 2 lives for 512":  {simulatedSwiss, "512", true},
		"a Swiss of 100 lives for 64": {`{"name":"x","phases":[{"kind":"swiss_lives","lives":100,"length":1,"mode":"rounds","pairing":"wins"}]}`, "64", true},
		"a double elimination of 512": {`{"name":"x","phases":[{"kind":"bracket","length":1,"consolation":true,"last_chance":true,"reconciliation":true,"recharge":true}]}`, "512", true},
	} {
		req := service.SimulateRequest{Config: json.RawMessage(tc.config), Players: json.RawMessage(tc.players), Seed: 3}
		got, err := service.Simulate(context.Background(), req)
		if !tc.heavy {
			if err != nil || got.Runs != service.SimulateDefaultRuns || slices.Contains(got.Warnings, service.SimulateWarnTimeBudget) {
				t.Errorf("%s: %+v (%v); want its %d runs", name, got, err, service.SimulateDefaultRuns)
			}
			continue
		}
		if !errors.Is(err, service.ErrSimulationTooHeavy) || errors.Is(err, direction.ErrRefused) || !strings.Contains(err.Error(), "fewer players") {
			t.Errorf("%s: %+v (%v); want too heavy, saying what to reduce", name, got, err)
			continue
		}
		if _, again := service.Simulate(context.Background(), req); again == nil || again.Error() != err.Error() {
			t.Errorf("%s: %v, then %v; want the same answer", name, err, again)
		}
	}
}
