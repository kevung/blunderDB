package direction

import (
	"slices"
	"time"

	tournoi "github.com/kevung/nicomaque"
	"github.com/kevung/nicomaque/rules"
	"github.com/kevung/nicomaque/sim"
)

// What the clock strip reads besides the engine's counters.
//
// Three numbers a director asks for; their arithmetic is the shared tournament rules':
//
//   - the PLAYING TIME: the time during which at least one match was running — the union of
//     the matches' [start, end) intervals, a running match counting up to now. Gaps with every
//     table empty (a night, a lunch, a week between club rounds) are not counted.
//   - the DAY: the calendar day of play, counted from the day of the first launched match,
//     in the host's time zone. Day 1 is the first.
//   - the ESTIMATED END: the median remaining time of the engine's forecast (sim.Forecast), laid
//     out after now and pushed past every declared break. An undeclared night counts as play.

// PlayingTime is the union of the matches' running intervals up to now.
func PlayingTime(st *tournoi.State, now time.Time) time.Duration {
	if st == nil {
		return 0
	}
	return rules.PlayingTime(st.RulesMatches(), now)
}

// DayOfPlay is the calendar day of now counted from start's (1 = the same day), in now's
// location; 0 when nothing has started.
func DayOfPlay(start, now time.Time) int { return rules.DayOfPlay(start, now) }

// AfterBreaks lays `play` out from now and returns when it ends, skipping every break: a
// break already under way delays the start, and a break met on the way adds its length.
func AfterBreaks(now time.Time, play time.Duration, breaks []tournoi.TimeRange) time.Time {
	cfg := tournoi.Config{Breaks: breaks}
	return rules.AfterBreaks(now, play, cfg.BreakIntervals())
}

// forecastRuns is the number of simulated ends. The median of 15 is stable to a few minutes
// on the scenarios of the simulation (S1, S5), and 15 replays of a 355-event log cost what
// one refresh of the strip can afford.
const forecastRuns = 15

// EstimatedEnd is the median forecast end of the tournament, pushed past the declared breaks;
// ok is false when the engine cannot forecast (a finished tournament, a stuck simulation).
func (d *Direction) EstimatedEnd(now time.Time) (time.Time, bool) {
	f, ok := d.EndForecast(now)
	return f.End, ok
}

// EndForecast is the forecast end of a tournament with its uncertainty: the median of the
// simulated ends, and the 10th and 90th centiles around it (nearest rank).
type EndForecast struct {
	Early, End, Late time.Time
}

// EndForecast is EstimatedEnd with the early and late ends, each pushed past the declared
// breaks.
func (d *Direction) EndForecast(now time.Time) (EndForecast, bool) {
	if d.st == nil || d.st.Finished {
		return EndForecast{}, false
	}
	runs, err := sim.Forecast(d.journal, now, forecastRuns, 0, d.rec.TournamentID)
	if err != nil || len(runs) == 0 {
		return EndForecast{}, false
	}
	breaks := d.st.Config.BreakIntervals()
	remaining := sim.Durations(runs)
	end, ok := rules.EstimatedEnd(now, remaining, breaks)
	if !ok {
		return EndForecast{}, false
	}
	slices.Sort(remaining)
	at := func(p float64) time.Time {
		return rules.AfterBreaks(now, remaining[int(p*float64(len(remaining)-1)+0.5)], breaks)
	}
	return EndForecast{Early: at(0.10), End: end, Late: at(0.90)}, true
}
