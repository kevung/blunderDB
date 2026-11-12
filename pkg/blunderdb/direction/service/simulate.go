package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"slices"
	"time"

	tournoi "github.com/kevung/nicomaque"
	"github.com/kevung/nicomaque/rules"
	"github.com/kevung/nicomaque/sim"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
)

// The simulation of a tournament that does not exist yet: the director asks how long a format
// would last before entering anyone. It plays the engine's own simulation (sim.Run) and reads
// nothing from the database, so it needs neither a Tournament nor a Direction.

// The bounds of one simulation. They are refusals, not clamps: a caller who asked for a
// thousand runs must not read the answer of two hundred as theirs.
const (
	SimulateMaxPlayers  = 1024
	SimulateMaxRuns     = 200
	SimulateDefaultRuns = 30
	SimulateMaxTables   = 4096
	SimulateMaxPhases   = 16
	// SimulateMaxLength bounds every match length of the format, in points.
	SimulateMaxLength = 99
	// directions.create bounds neither the field nor the lives, and the engine admits any:
	// these two are well past a real tournament, and under them SimulateBudget is what
	// protects the processor (a run is cut between two of its events).
	SimulateMaxLives = 100
	// The raw sizes of the two free-form members: nothing is decoded past them, so no slice
	// is ever sized by what a request says.
	simulateMaxConfigBytes  = 64 << 10
	simulateMaxPlayersBytes = 256 << 10
	// simulateMaxSteps bounds one simulated tournament, whatever its format.
	simulateMaxSteps = 50000
)

// SimulateBudget is the computing time one simulation may take. Past it the runs already
// played are answered, under the warning SimulateWarnTimeBudget; with none played the answer
// is ErrSimulationTooHeavy.
var SimulateBudget = 2 * time.Second

// ErrSimulationTooHeavy: the request is well formed and within the bounds, but not one run
// ended within SimulateBudget. It is no refusal of the request's shape: the same request may
// be answered on a quieter machine, and a smaller field is answered here.
var ErrSimulationTooHeavy = errors.New("direction: the simulation ran out of time before one run ended: ask for fewer players or a shorter format (fewer runs help an answer cut short by time_budget, not this one)")

// simulateMaxMinutes bounds the length a simulated tournament may have and still be
// answered: past a year the figure says a break or a length is mistyped, and the end would
// leave the dates RFC 3339 writes.
const simulateMaxMinutes = 366 * 24 * 60

// The warnings of a simulation, as codes. The engine's own warning codes (tournoi.WarningCode)
// join them when a simulated tournament ends with one.
const (
	// SimulateWarnTimeBudget: the computing time ran out; Runs says how many runs the answer
	// stands on, and the same request may stand on another number next time.
	SimulateWarnTimeBudget = "time_budget"
	// SimulateWarnRunFailed: the engine could not play at least one run to its end; the
	// figures stand on the others.
	SimulateWarnRunFailed = "run_failed"
	// SimulateWarnTablesExceeded: the engine's simulation launches every match as soon as it
	// is paired, and at its peak this format wants more tables than the room has (PeakTables);
	// the lengths are those of a room that large, so a floor for this one.
	SimulateWarnTablesExceeded = "tables_exceeded"
)

// SimulateRequest describes the tournament to simulate.
type SimulateRequest struct {
	// Config is the engine's configuration, as directions.create takes it.
	Config json.RawMessage `json:"config"`
	// Players is either a number — a field of that many players of ordinary, spread levels —
	// or a list of entries ({"id","name","rating"}); a rating of 0 takes the mean of the others.
	Players json.RawMessage `json:"players"`
	// Tables, when positive, replaces the configuration's number of tables: the room PeakTables
	// is compared with.
	Tables int `json:"tables"`
	// Start (RFC 3339) is when the first match would be launched; EstimatedEnd needs it.
	Start string `json:"start"`
	// Runs is the number of simulated tournaments, SimulateDefaultRuns when 0.
	Runs int `json:"runs"`
	// Seed makes the answer reproducible: same request, same answer.
	Seed int64 `json:"seed"`
}

// Simulation is what the simulated tournaments say of the format.
type Simulation struct {
	// Runs is the number of tournaments the figures stand on.
	Runs int `json:"runs"`
	// Matches is the median number of matches of a tournament.
	Matches int `json:"matches"`
	// MinutesP10, MinutesP50 and MinutesP90 are the 10th, 50th and 90th centiles of the
	// tournament's length, from the first match launched to the last result, breaks included.
	MinutesP10 float64 `json:"minutesP10"`
	MinutesP50 float64 `json:"minutesP50"`
	MinutesP90 float64 `json:"minutesP90"`
	// EstimatedEnd is Start plus MinutesP50 (RFC 3339), empty without a Start.
	EstimatedEnd string `json:"estimatedEnd,omitempty"`
	// PeakTables is the most matches ever running at once: the tables the format wants so that
	// no match waits for one. It can exceed the room's (SimulateWarnTablesExceeded).
	PeakTables int `json:"peakTables"`
	// MatchesPerPlayerMin and MatchesPerPlayerMax are the fewest and the most matches one
	// player played, over every run.
	MatchesPerPlayerMin int `json:"matchesPerPlayerMin"`
	MatchesPerPlayerMax int `json:"matchesPerPlayerMax"`
	// Warnings are codes, sorted, each once.
	Warnings []string `json:"warnings"`
}

// Simulate plays req.Runs tournaments of the format and summarises them. It is deterministic
// for a given request unless the time budget cuts it short, which the answer says.
func Simulate(ctx context.Context, req SimulateRequest) (*Simulation, error) {
	if len(req.Config) > simulateMaxConfigBytes {
		return nil, direction.Refusef("direction: config is larger than %d bytes", simulateMaxConfigBytes)
	}
	if len(req.Players) > simulateMaxPlayersBytes {
		return nil, direction.Refusef("direction: players is larger than %d bytes", simulateMaxPlayersBytes)
	}
	if req.Tables < 0 || req.Tables > SimulateMaxTables {
		return nil, direction.Refusef("direction: a simulation has 0 to %d tables, not %d", SimulateMaxTables, req.Tables)
	}
	if err := boundSimulatedConfig(req.Config); err != nil {
		return nil, err
	}
	cfg, err := parseDirectionConfig(string(req.Config))
	if err != nil {
		return nil, err
	}
	if req.Tables > 0 {
		cfg.Tables.Count = req.Tables
	}
	runs := req.Runs
	if runs == 0 {
		runs = SimulateDefaultRuns
	}
	if runs < 0 || runs > SimulateMaxRuns {
		return nil, direction.Refusef("direction: a simulation plays 1 to %d runs, not %d", SimulateMaxRuns, runs)
	}
	var start time.Time
	if req.Start != "" {
		if start, err = time.Parse(time.RFC3339, req.Start); err != nil {
			return nil, direction.Refusef("direction: start %q is not an RFC 3339 time", req.Start)
		}
	}
	players, err := simulatedField(req.Players, req.Seed)
	if err != nil {
		return nil, err
	}

	parent := ctx
	ctx, cancel := context.WithTimeout(ctx, SimulateBudget)
	defer cancel()
	var (
		minutes  []float64
		matches  []int
		warnings = map[string]bool{}
		out      = &Simulation{MatchesPerPlayerMin: -1}
		firstErr error
	)
	for i := range runs {
		peak := 0
		r := sim.Run(cfg, players, sim.Options{Seed: req.Seed + int64(i), Start: start, MaxSteps: simulateMaxSteps,
			Hook: func(s *tournoi.State, _ tournoi.Event) error {
				peak = max(peak, len(s.Running()))
				return ctx.Err()
			}})
		if ctx.Err() != nil {
			warnings[SimulateWarnTimeBudget] = true
			break
		}
		if r.Err != nil || r.State == nil || !r.State.Finished {
			if firstErr == nil {
				firstErr = r.Err
			}
			warnings[SimulateWarnRunFailed] = true
			continue
		}
		minutes = append(minutes, r.Minutes)
		matches = append(matches, r.NMatches)
		out.PeakTables = max(out.PeakTables, peak)
		played := map[tournoi.PlayerID]int{}
		for _, m := range r.State.Matches {
			for _, p := range []tournoi.PlayerID{m.A, m.B} {
				if p != tournoi.BYE && p != "" {
					played[p]++
				}
			}
		}
		for _, p := range players {
			n := played[p.ID]
			if out.MatchesPerPlayerMin < 0 || n < out.MatchesPerPlayerMin {
				out.MatchesPerPlayerMin = n
			}
			out.MatchesPerPlayerMax = max(out.MatchesPerPlayerMax, n)
		}
		for _, w := range r.State.Warnings {
			warnings[string(w.Code)] = true
		}
	}
	if len(minutes) == 0 {
		// The caller left: nobody reads a verdict on the request.
		if err := parent.Err(); err != nil {
			return nil, err
		}
		if ctx.Err() != nil && firstErr == nil {
			return nil, ErrSimulationTooHeavy
		}
		if firstErr != nil {
			return nil, direction.Refusef("direction: this format cannot be simulated: %v", firstErr)
		}
		return nil, direction.Refusef("direction: this format cannot be simulated: no run reached its end")
	}
	slices.Sort(minutes)
	slices.Sort(matches)
	out.Runs = len(minutes)
	out.Matches = matches[len(matches)/2]
	out.MinutesP10, out.MinutesP50, out.MinutesP90 = centile(minutes, 0.10), centile(minutes, 0.50), centile(minutes, 0.90)
	if matches[len(matches)-1] == 0 {
		return nil, direction.Refusef("direction: this format plays no match with %d players: there is nothing to simulate", len(players))
	}
	if out.MinutesP90 > simulateMaxMinutes {
		return nil, direction.Refusef("direction: the simulated tournament lasts more than a year (%.0f minutes): check its breaks and its lengths", out.MinutesP90)
	}
	if !start.IsZero() {
		end := start.Add(time.Duration(out.MinutesP50 * float64(time.Minute)))
		if end.Year() > 9999 {
			return nil, direction.Refusef("direction: start %q leaves no date for the end of the tournament", req.Start)
		}
		out.EstimatedEnd = end.Format(time.RFC3339)
	}
	if cfg.Tables.Count > 0 && out.PeakTables > cfg.Tables.Count {
		warnings[SimulateWarnTablesExceeded] = true
	}
	out.Warnings = make([]string, 0, len(warnings))
	for w := range warnings {
		out.Warnings = append(out.Warnings, w)
	}
	slices.Sort(out.Warnings)
	return out, nil
}

// centile is the nearest-rank p-th centile of sorted, which is not empty.
func centile[T any](sorted []T, p float64) T {
	return sorted[int(p*float64(len(sorted)-1)+0.5)]
}

// simulatedField is the field a simulation plays with: the entries given, or as many players
// as the number given, their levels drawn from seed.
func simulatedField(raw json.RawMessage, seed int64) ([]tournoi.Player, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return nil, direction.Refusef("direction: a simulation needs players, a number or a list")
	}
	var players []tournoi.Player
	if raw[0] == '[' {
		if err := json.Unmarshal(raw, &players); err != nil {
			return nil, direction.Refusef("direction: players is not a list of entries: %v", err)
		}
		if len(players) < 2 || len(players) > SimulateMaxPlayers {
			return nil, direction.Refusef("direction: a simulation plays 2 to %d players, not %d", SimulateMaxPlayers, len(players))
		}
		seen := map[tournoi.PlayerID]bool{}
		ratings := make([]float64, len(players))
		for i, p := range players {
			if p.ID == "" || p.ID == tournoi.BYE || seen[p.ID] {
				return nil, direction.Refusef("direction: entry %d of players needs an id of its own, not %q", i+1, p.ID)
			}
			seen[p.ID] = true
			ratings[i] = p.Rating
		}
		mean := rules.MeanRating(ratings)
		for i := range players {
			if players[i].Rating <= 0 {
				players[i].Rating = mean
			}
		}
	} else {
		var n int
		if err := json.Unmarshal(raw, &n); err != nil {
			return nil, direction.Refusef("direction: players is neither a number nor a list")
		}
		if n < 2 || n > SimulateMaxPlayers {
			return nil, direction.Refusef("direction: a simulation plays 2 to %d players, not %d", SimulateMaxPlayers, n)
		}
		// The engine's own test field: levels around 6, between 2 and 10.
		return sim.Champ(n, 6, 2, 2, 10, rand.New(rand.NewSource(seed))), nil
	}
	return players, nil
}

// boundSimulatedConfig refuses a configuration whose sizes would hold the processor or the
// memory, before the engine reads it: the engine validates a format, not its cost.
func boundSimulatedConfig(raw json.RawMessage) error {
	var c struct {
		Phases []struct {
			Lives         int   `json:"lives"`
			Length        int   `json:"length"`
			FinalLength   int   `json:"final_length"`
			Lengths       []int `json:"lengths"`
			LengthLate    int   `json:"length_late"`
			LateThreshold int   `json:"late_threshold"`
			BatchMinutes  int   `json:"batch_minutes"`
			Target        int   `json:"target"`
			GroupSize     int   `json:"group_size"`
			Qualifiers    int   `json:"qualifiers"`
		} `json:"phases"`
		MinPerPoint float64 `json:"min_per_point"`
	}
	// A configuration that does not decode is the engine's to refuse, in its own words.
	if json.Unmarshal(raw, &c) != nil {
		return nil
	}
	if len(c.Phases) > SimulateMaxPhases {
		return direction.Refusef("direction: a simulation plays at most %d phases, not %d", SimulateMaxPhases, len(c.Phases))
	}
	if c.MinPerPoint < 0 || c.MinPerPoint > 60 {
		return direction.Refusef("direction: in a simulation min_per_point is 0 to 60, not %v", c.MinPerPoint)
	}
	for i, p := range c.Phases {
		for _, b := range []struct {
			name   string
			v, max int
		}{
			{"length", p.Length, SimulateMaxLength}, {"final_length", p.FinalLength, SimulateMaxLength},
			{"length_late", p.LengthLate, SimulateMaxLength}, {"lives", p.Lives, SimulateMaxLives},
			{"late_threshold", p.LateThreshold, SimulateMaxPlayers}, {"target", p.Target, SimulateMaxPlayers},
			{"group_size", p.GroupSize, SimulateMaxPlayers}, {"qualifiers", p.Qualifiers, SimulateMaxPlayers},
			{"batch_minutes", p.BatchMinutes, 24 * 60},
		} {
			if b.v < 0 || b.v > b.max {
				return direction.Refusef("direction: in phase %d of a simulation %s is 0 to %d, not %d", i+1, b.name, b.max, b.v)
			}
		}
		for _, l := range p.Lengths {
			if l < 0 || l > SimulateMaxLength {
				return direction.Refusef("direction: in phase %d of a simulation lengths is 0 to %d, not %d", i+1, SimulateMaxLength, l)
			}
		}
	}
	return nil
}
