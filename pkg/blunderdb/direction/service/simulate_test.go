package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
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

// TestSimulate_Bounds: what would hold the processor is refused, never clamped.
func TestSimulate_Bounds(t *testing.T) {
	for name, change := range map[string]func(*service.SimulateRequest){
		"too many runs":     func(r *service.SimulateRequest) { r.Runs = service.SimulateMaxRuns + 1 },
		"negative runs":     func(r *service.SimulateRequest) { r.Runs = -1 },
		"too many players":  func(r *service.SimulateRequest) { r.Players = json.RawMessage("513") },
		"one player":        func(r *service.SimulateRequest) { r.Players = json.RawMessage("1") },
		"no players":        func(r *service.SimulateRequest) { r.Players = nil },
		"players a string":  func(r *service.SimulateRequest) { r.Players = json.RawMessage(`"many"`) },
		"twice the same id": func(r *service.SimulateRequest) { r.Players = json.RawMessage(`[{"id":"a"},{"id":"a"}]`) },
		"negative tables":   func(r *service.SimulateRequest) { r.Tables = -1 },
		"a start no time":   func(r *service.SimulateRequest) { r.Start = "tomorrow" },
		"no config":         func(r *service.SimulateRequest) { r.Config = nil },
		"an unknown format": func(r *service.SimulateRequest) {
			r.Config = json.RawMessage(`{"phases":[{"kind":"nope","length":5}]}`)
		},
	} {
		req := simulateReq("16")
		change(&req)
		if got, err := service.Simulate(context.Background(), req); err == nil {
			t.Errorf("%s: answered %+v; want a refusal", name, got)
		}
	}
}

// TestSimulate_TimeBudget: past its computing time a simulation stops; with no run ended it
// is a refusal, and a cancelled request stops as well.
func TestSimulate_TimeBudget(t *testing.T) {
	was := service.SimulateBudget
	t.Cleanup(func() { service.SimulateBudget = was })
	service.SimulateBudget = time.Nanosecond
	req := simulateReq("256")
	req.Runs = service.SimulateMaxRuns
	began := time.Now()
	got, err := service.Simulate(context.Background(), req)
	if err == nil || time.Since(began) > 5*time.Second {
		t.Errorf("with no time: %+v (%v) after %v; want a prompt refusal", got, err, time.Since(began))
	}
	if !errors.Is(err, direction.ErrRefused) {
		t.Errorf("out of time is %v; want a refusal, the caller's to act on", err)
	}
}
