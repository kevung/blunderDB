package service_test

import (
	"context"
	"encoding/json"
	"errors"
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
		"too many players":   {players("1025"), "2 to 1024 players, not 1025"},
		"one player":         {players("1"), "2 to 1024 players, not 1"},
		"negative players":   {players("-3"), "2 to 1024 players, not -3"},
		"a list too long":    {players("[" + strings.Repeat(`{"id":"a"},`, service.SimulateMaxPlayers) + `{"id":"b"}]`), "2 to 1024 players, not 1025"},
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
		if !errors.Is(err, direction.ErrRefused) || !strings.Contains(err.Error(), "2 to 1024 players, not "+n) {
			t.Errorf("%s players: %v; want the refusal of the bound", n, err)
		}
		if grew := after.TotalAlloc - before.TotalAlloc; grew > 1<<20 {
			t.Errorf("%s players: refused after allocating %d bytes; want nothing sized by the request", n, grew)
		}
	}
}

// TestSimulate_TimeBudget: past its computing time a simulation stops. With no run ended the
// answer is ErrSimulationTooHeavy, which is no refusal of the request: the same request is
// answered with more time. A caller that left gets its own cancellation back.
func TestSimulate_TimeBudget(t *testing.T) {
	was := service.SimulateBudget
	t.Cleanup(func() { service.SimulateBudget = was })
	service.SimulateBudget = time.Nanosecond
	req := simulateReq("256")
	req.Runs = service.SimulateMaxRuns
	began := time.Now()
	got, err := service.Simulate(context.Background(), req)
	if !errors.Is(err, service.ErrSimulationTooHeavy) || errors.Is(err, direction.ErrRefused) || time.Since(began) > 5*time.Second {
		t.Errorf("with no time: %+v (%v) after %v; want too heavy, promptly, and no refusal", got, err, time.Since(began))
	}
	if err == nil || !strings.Contains(err.Error(), "fewer players") {
		t.Errorf("too heavy says %q; want what to reduce", err)
	}
	service.SimulateBudget = time.Minute
	req = simulateReq("16")
	if _, err := service.Simulate(context.Background(), req); err != nil {
		t.Errorf("the same kind of request with its time: %v", err)
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
// field; a Swiss of 20 lives and a field of 600 are simulated.
func TestSimulate_AcceptsWhatCreationAccepts(t *testing.T) {
	unhurried(t)
	req := simulateReq("8")
	req.Config = json.RawMessage(strings.Replace(simulatedSwiss, `"lives":2`, `"lives":20`, 1))
	req.Runs = 1
	if got, err := service.Simulate(context.Background(), req); err != nil || got.Runs != 1 {
		t.Errorf("a Swiss of 20 lives: %+v (%v); want one run", got, err)
	}
	req = service.SimulateRequest{Config: json.RawMessage(`{"name":"x","phases":[{"kind":"bracket","length":5}]}`), Players: json.RawMessage("600"), Runs: 1}
	if got, err := service.Simulate(context.Background(), req); err != nil || got.Matches != 599 {
		t.Errorf("a bracket of 600: %+v (%v); want its 599 matches", got, err)
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
	req.Config = json.RawMessage(`{"name":"x","min_per_point":60,"phases":[{"kind":"swiss_lives","length":99,"lives":100,"mode":"continuous"}]}`)
	req.Runs = 1
	if got, err := service.Simulate(context.Background(), req); !errors.Is(err, direction.ErrRefused) || !strings.Contains(err.Error(), "more than a year") {
		t.Errorf("a tournament of years: %+v (%v); want a refusal", got, err)
	}
}
