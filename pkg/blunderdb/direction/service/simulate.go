package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	// SimulateMaxPlayers is the largest field a simulation reads. directions.create bounds no
	// field; the simulation does, on purpose, so that nothing is sized by the request: what
	// a field within it may cost is simulateMaxRunCost's to bound, for it depends on the
	// format too.
	SimulateMaxPlayers  = 512
	SimulateMaxRuns     = 200
	SimulateDefaultRuns = 30
	SimulateMaxTables   = 4096
	SimulateMaxPhases   = 16
	// SimulateMaxLength bounds every match length of the format, in points.
	SimulateMaxLength = 99
	// directions.create does not bound the lives, and the engine admits any: this is well
	// past a real tournament, and under it SimulateBudget is what protects the processor (a
	// run is cut between two of its events).
	SimulateMaxLives = 100
	// SimulateMaxPhaseMatches bounds the matches one phase draws at once. A run is cut
	// between two events only, and the draw of a round robin is one event that creates every
	// match of every pool: its time and its memory grow with the square of the pool, out of
	// the reach of SimulateBudget. At this bound a draw holds a few megabytes for a few tens
	// of milliseconds, several times what a run plays within the budget.
	SimulateMaxPhaseMatches = 4096
	// The raw sizes of the two free-form members: nothing is decoded past them, so no slice
	// is ever sized by what a request says.
	simulateMaxConfigBytes  = 64 << 10
	simulateMaxPlayersBytes = 256 << 10
	// simulateMaxSteps bounds one simulated tournament, whatever its format.
	simulateMaxSteps = 50000
	// simulateMaxRunCost bounds the cost of the first simulated tournament, run with the
	// request's own seed, as eventCost and simulateSearchStepCost count it: the engine copies
	// about eventWork at every event, so a run costs the sum, which grows with the square of
	// its matches, and its pairing search adds its steps. Counted,
	// not timed, and on that one run only, the bound gives a request the same verdict on
	// every machine whatever the budget lets the other runs reach. At it a run takes about
	// 0.3 s on an ordinary processor: under it SimulateBudget plays about SimulateUsefulRuns
	// runs or more; past it, a handful or none, and the request is refused at its first run
	// instead (ErrSimulationTooHeavy). The runs of other seeds vary by some 15 %: they are
	// held under twice the bound, a run past it counted as failed, for its memory only.
	simulateMaxRunCost = 30_000_000
	// SimulateUsefulRuns is the fewest runs whose 10th and 90th centiles say something.
	SimulateUsefulRuns = 10
	// simulateSearchStepCost weighs one step of the engine's pairing search, a pair looked
	// at, kept or not, in the units of simulateMaxRunCost: 5 to 10 ns measured. The engine
	// counts the steps (tournoi.SearchBudget), so a search that never finds a round without
	// rematch — a club holding more than half the field with avoid_clubs, a Swiss by rounds
	// as long as its field — ends the run on its count as the events do, not on a clock.
	simulateSearchStepCost = 1
)

// SimulateBudget is the computing time the runs after the first may take: with the two
// places of the daemon's gate taken, the most a host computes for callers who are only
// asking. The first run is not cut by it — simulateMaxRunCost holds it to a fraction of the
// budget —, so every request admitted stands on one run at least, whatever the machine; past
// the budget the runs already played are answered, under the warning SimulateWarnTimeBudget.
var SimulateBudget = 2 * time.Second

// ErrSimulationTooHeavy: the request is well formed and within the bounds, but the first of
// its tournaments, its events and its pairing search together, costs more than
// simulateMaxRunCost, so the budget would stand an answer on a handful of runs. The verdict
// is counted, not timed: the same on every machine.
var ErrSimulationTooHeavy = fmt.Errorf("direction: this format is too heavy to simulate %d times within %v: one of its tournaments outgrows what a simulation computes; ask for fewer players, smaller pools, fewer lives, fewer rounds or a lighter format (fewer runs do not help)",
	SimulateUsefulRuns, SimulateBudget)

// errRunTooHeavy stops a run past its cost; Simulate answers it as ErrSimulationTooHeavy at
// the first run, as a failed run at another.
var errRunTooHeavy = errors.New("direction: run past its cost")

// simulateObserve, nil outside the tests, is called on every event of a simulated run, in the
// goroutine that plays it: a test weighs there what the run holds while nothing else allocates,
// which no sampler running beside it can do on every machine.
var simulateObserve func()

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
)

// SimulateRequest describes the tournament to simulate.
type SimulateRequest struct {
	// Config is the engine's configuration, as directions.create takes it.
	Config json.RawMessage `json:"config"`
	// Players is either a number — a field of that many players of ordinary, spread levels —
	// or a list of entries ({"id","name","rating"}); a rating of 0 takes the mean of the others.
	Players json.RawMessage `json:"players"`
	// Tables, when positive, replaces the configuration's number of tables: the room the
	// simulation plays in, a match waiting for a free table.
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
	// tournament's length, from the first match launched to the last result, at the
	// configuration's min_per_point. With a Start the configuration's breaks are included: a
	// break is a date, and a tournament with no start meets none.
	MinutesP10 float64 `json:"minutesP10"`
	MinutesP50 float64 `json:"minutesP50"`
	MinutesP90 float64 `json:"minutesP90"`
	// EstimatedEnd is Start plus MinutesP50 (RFC 3339), so past the breaks; empty without a Start.
	EstimatedEnd string `json:"estimatedEnd,omitempty"`
	// PeakTables is the most matches ever running at once: the tables the format wants so that
	// no match waits for one, never more than the room has.
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
	if err := boundSimulatedWork(cfg, len(players)); err != nil {
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
		// The first run decides the verdict: it answers to its count only, never to a clock, so
		// that the verdict is the same on every machine.
		limit := parent
		if i > 0 {
			if ctx.Err() != nil {
				warnings[SimulateWarnTimeBudget] = true
				break
			}
			limit = ctx
		}
		peak, cost, most := 0, 0, simulateMaxRunCost
		if i > 0 {
			most *= 2
		}
		// The engine counts its pairing search and stops it past the limit, inside a proposal
		// the hook never sees the end of; the hook adds the steps already searched to the count
		// and leaves the search what the events have not spent. A limit of 0 would be none.
		search := &tournoi.SearchBudget{Limit: int64(most / simulateSearchStepCost)}
		r := sim.Run(cfg, players, sim.Options{Seed: req.Seed + int64(i), Start: start, MaxSteps: simulateMaxSteps,
			MinPerPoint: cfg.MinPerPoint, Search: search,
			Hook: func(s *tournoi.State, _ tournoi.Event) error {
				if simulateObserve != nil {
					simulateObserve()
				}
				peak = max(peak, len(s.Running()))
				if cost += eventCost(cfg, s); cost+int(search.Used)*simulateSearchStepCost > most {
					return errRunTooHeavy
				}
				search.Limit = max(int64((most-cost)/simulateSearchStepCost), 1)
				return limit.Err()
			}})
		// The caller left: nobody reads a verdict on the request.
		if err := parent.Err(); err != nil {
			return nil, err
		}
		if i == 0 {
			if errors.Is(r.Err, errRunTooHeavy) || errors.Is(r.Err, tournoi.ErrSearchBudget) {
				return nil, ErrSimulationTooHeavy
			}
		}
		if ctx.Err() != nil && errors.Is(r.Err, ctx.Err()) {
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
		if firstErr != nil {
			return nil, direction.Refusef("direction: this format cannot be simulated: %v", firstErr)
		}
		return nil, direction.Refusef("direction: this format cannot be simulated: no run reached its end")
	}
	if !start.IsZero() && len(cfg.Breaks) > 0 {
		// The engine's simulation plays through the breaks; the clock of a live tournament
		// lays its forecast past them, and so does this one.
		for i, m := range minutes {
			minutes[i] = direction.AfterBreaks(start, time.Duration(m*float64(time.Minute)), cfg.Breaks).Sub(start).Minutes()
		}
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
	out.Warnings = make([]string, 0, len(warnings))
	for w := range warnings {
		out.Warnings = append(out.Warnings, w)
	}
	slices.Sort(out.Warnings)
	return out, nil
}

// boundSimulatedWork refuses a format whose phase would draw more matches at once than a
// simulation plays: the pools of a round robin, every pair of each. Every other format draws
// at most one match for two players.
func boundSimulatedWork(cfg tournoi.Config, players int) error {
	for i, p := range cfg.Phases {
		if p.Kind != tournoi.KindRoundRobin {
			continue
		}
		// The first phase draws the field; a later one is entered by no more players than the
		// field, and a few entrants may sit in larger pools than the whole field does.
		least := players
		if i > 0 {
			least = 2
		}
		for n := least; n <= players; n++ {
			if size, drawn := poolDraw(p, n); drawn > SimulateMaxPhaseMatches {
				return direction.Refusef("direction: in phase %d of a simulation, %d players draw pools of up to %d, so %d matches; a simulation plays at most %d matches a phase: ask for smaller pools or fewer players",
					i+1, n, size, drawn, SimulateMaxPhaseMatches)
			}
		}
	}
	return nil
}

// poolDraw is the largest pool and the matches the engine draws for n entrants of a round
// robin: it deals them in turn into its pools, which so differ by one player at most.
func poolDraw(p tournoi.PhaseConfig, n int) (size, drawn int) {
	var pools int
	switch {
	case p.Groups > 0:
		pools = max(min(p.Groups, n/2), 1)
	case p.GroupSize > 0:
		pools = max((n+p.GroupSize-1)/p.GroupSize, 1)
	default:
		pools = max((n+3)/4, 1)
	}
	size = (n + pools - 1) / pools
	big := n - (size-1)*pools // the pools of size players; the others have one fewer
	return size, big*size*(size-1)/2 + (pools-big)*(size-1)*(size-2)/2
}

// eventWork is about what the engine goes through at one event of s: its players, its
// matches and the planned matches of the phase in play — a pool plans every pair at its
// draw. A Swiss plans nothing and pairs as it goes: the engine counts the steps of a Swiss
// by rounds' search itself (simulateSearchStepCost), but a continuous Swiss with lives pairs
// every group of its entrants anew at every event, greedily and uncounted, and a share of
// their square stands for that.
func eventWork(s *tournoi.State) int {
	n := len(s.Players) + len(s.MatchOrder)
	if s.Current < 0 || s.Current >= len(s.Phases) {
		return n
	}
	ph := s.Phases[s.Current]
	for _, sec := range ph.Sections {
		n += len(sec.Matches)
	}
	if continuousLives(ph.Cfg) {
		n += len(ph.Entrants) * len(ph.Entrants) / 64
	}
	return n
}

// continuousLives: a Swiss with lives that pairs as players come free, not by rounds.
func continuousLives(p tournoi.PhaseConfig) bool {
	return p.Kind == tournoi.KindSwissLives && p.Mode != "rounds"
}

// eventCost weighs eventWork by the phase in play, as a run's cost in tens of nanoseconds
// of an ordinary processor: the engine's work per unit differs by format, from a bracket's
// sections to a continuous Swiss pairing at every event. The weights are measured, the
// search's steps apart, from 32 to 200 players. Before its first phase a tournament is
// weighed as that phase.
func eventCost(cfg tournoi.Config, s *tournoi.State) int {
	var p tournoi.PhaseConfig
	if s.Current >= 0 && s.Current < len(s.Phases) {
		p = s.Phases[s.Current].Cfg
	} else if len(cfg.Phases) > 0 {
		p = cfg.Phases[0]
	}
	var weight int
	switch {
	case p.Kind == tournoi.KindBracket, p.Kind == tournoi.KindLivesBracket, p.Kind == tournoi.KindGSL:
		weight = 22
	case continuousLives(p):
		weight = 54
	case p.Kind == tournoi.KindRoundRobin:
		weight = 40
	default: // swiss_rounds, a Swiss with lives by rounds, manual, and an unknown kind
		weight = 15
	}
	return weight * eventWork(s)
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
			Groups        int   `json:"groups"`
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
			{"group_size", p.GroupSize, SimulateMaxPlayers}, {"groups", p.Groups, SimulateMaxPlayers},
			{"qualifiers", p.Qualifiers, SimulateMaxPlayers},
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
