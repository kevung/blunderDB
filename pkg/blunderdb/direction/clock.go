package direction

import (
	"sort"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
	"github.com/PileOfCells/backgammon-tournoi/sim"
)

// What the clock strip reads besides the engine's counters (issue #456, D8.3).
//
// Three numbers a director asks for and the engine does not state as such:
//
//   - the PLAYING TIME: the time during which at least one match was running — the union of
//     the matches' [start, end) intervals, a running match counting up to now. A night, a lunch
//     with every table empty, a Monday between two rounds of a club championship: nothing is
//     being played, and nothing is counted. On a single evening it is the elapsed time less the
//     gaps between rounds; over five days it stops counting the nights as play (S5 showed
//     « 48 h 45 » for two and a half days).
//   - the DAY: the calendar day of play, counted from the day of the first launched match,
//     in the host's time zone. Day 1 is the first.
//   - the ESTIMATED END: the engine's own forecast (sim.Forecast) — it replays the log and plays
//     the rest of the tournament K times at the configured pace — its median remaining time,
//     laid out after now and pushed past every declared break. A night that is not declared as
//     a break counts as play: the engine knows the breaks, not the hall's opening hours.

// PlayingTime is the union of the matches' running intervals up to now.
func PlayingTime(st *tournoi.State, now time.Time) time.Duration {
	if st == nil {
		return 0
	}
	type span struct{ a, b time.Time }
	var spans []span
	for _, id := range st.MatchOrder {
		m := st.Matches[id]
		if m == nil || m.Start.IsZero() || m.Status == tournoi.Cancelled {
			continue
		}
		end := m.End
		if m.Status == tournoi.Running || end.IsZero() || end.After(now) {
			end = now
		}
		if end.After(m.Start) {
			spans = append(spans, span{m.Start, end})
		}
	}
	sort.Slice(spans, func(i, j int) bool { return spans[i].a.Before(spans[j].a) })
	var total time.Duration
	var cur span
	for i, s := range spans {
		if i == 0 {
			cur = s
			continue
		}
		if !s.a.After(cur.b) {
			if s.b.After(cur.b) {
				cur.b = s.b
			}
			continue
		}
		total += cur.b.Sub(cur.a)
		cur = s
	}
	if len(spans) > 0 {
		total += cur.b.Sub(cur.a)
	}
	return total
}

// DayOfPlay is the calendar day of now counted from start's (1 = the same day), in now's
// location; 0 when nothing has started.
func DayOfPlay(start, now time.Time) int {
	if start.IsZero() {
		return 0
	}
	s := start.In(now.Location())
	a := time.Date(s.Year(), s.Month(), s.Day(), 12, 0, 0, 0, time.UTC)
	b := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, time.UTC)
	return int(b.Sub(a).Hours()/24) + 1
}

// AfterBreaks lays `play` out from now and returns when it ends, skipping every break: a
// break already under way delays the start, and a break met on the way adds its length.
func AfterBreaks(now time.Time, play time.Duration, breaks []tournoi.TimeRange) time.Time {
	bs := append([]tournoi.TimeRange(nil), breaks...)
	sort.Slice(bs, func(i, j int) bool { return bs[i].Start.Before(bs[j].Start) })
	t := now
	left := play
	for _, b := range bs {
		if !b.End.After(t) {
			continue
		}
		if b.Start.After(t) {
			if run := b.Start.Sub(t); run >= left {
				break
			} else {
				left -= run
			}
		}
		t = b.End
	}
	return t.Add(left)
}

// forecastRuns is the number of simulated ends. The median of 15 is stable to a few minutes
// on the scenarios of the simulation (S1, S5), and 15 replays of a 355-event log cost what
// one refresh of the strip can afford.
const forecastRuns = 15

// EstimatedEnd is the median forecast end of the tournament, pushed past the declared breaks;
// ok is false when the engine cannot forecast (a finished tournament, a stuck simulation).
func (d *Direction) EstimatedEnd(now time.Time) (time.Time, bool) {
	if d.st == nil || d.st.Finished {
		return time.Time{}, false
	}
	runs, err := sim.Forecast(d.journal, now, forecastRuns, 0, d.rec.TournamentID)
	if err != nil || len(runs) == 0 {
		return time.Time{}, false
	}
	median := runs[len(runs)/2]
	return AfterBreaks(now, time.Duration(median*float64(time.Minute)), d.st.Config.Breaks), true
}
