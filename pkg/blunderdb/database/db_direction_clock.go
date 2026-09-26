package database

import (
	"context"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
)

// The clock of a Direction (tasks/nicomaque/fonctionnel.md §6). A match is "slow" past 1.5× its expected duration,
// the UX document's default.
const slowFactor = 1.5

// ClockView is the clock strip: one line of text, not a dashboard.
type ClockView struct {
	// ElapsedSeconds counts from the first match launched, not from the Direction's creation:
	// a tournament prepared the evening before has not been running since then.
	ElapsedSeconds int `json:"elapsedSeconds"`
	// PlayingSeconds is the time during which at least one match was running: the
	// nights of a tournament over several days are not play. direction.PlayingTime.
	PlayingSeconds int `json:"playingSeconds"`
	// Day is the calendar day of play, 1 on the day of the first launched match, 0 before.
	Day int `json:"day"`
	// EstimatedEnd is the engine's forecast end (RFC 3339), empty when there is none.
	EstimatedEnd string `json:"estimatedEnd,omitempty"`
	// Finished: the tournament is closed, and the strip has nothing left to say.
	Finished        bool    `json:"finished"`
	Played          int     `json:"played"`
	Running         int     `json:"running"`
	MinutesPerPoint float64 `json:"minutesPerPoint"`
	PlannedPerPoint float64 `json:"plannedPerPoint"`
	SlowMatches     int     `json:"slowMatches"`
	// NextBreak is the next declared break, empty when none is coming.
	NextBreak string `json:"nextBreak,omitempty"`
	Warnings  int    `json:"warnings"`
}

// Clock returns what the strip shows, at the host's clock.
func (d *Database) Clock(tournamentID int64) (*ClockView, error) {
	return d.clockAt(tournamentID, time.Now())
}

// forecastTTL is how long an estimated end stands while nothing is written: the forecast
// replays the log fifteen times, and the strip refreshes every minute.
const forecastTTL = 5 * time.Minute

// forecastMemo keeps the last forecast of each Direction, keyed by the length of its log: a
// new event recomputes it, and so does forecastTTL without one (a match running late moves
// the end). It is a cache of a DERIVED value, never read back as state.
type forecastMemo struct {
	events int
	at     time.Time
	end    time.Time
	ok     bool
}

func (d *Database) estimatedEnd(tournamentID int64, dir *direction.Direction, now time.Time) (time.Time, bool) {
	n := len(dir.Journal())
	d.forecastMu.Lock()
	m, hit := d.forecasts[tournamentID]
	d.forecastMu.Unlock()
	if hit && m.events == n && !now.Before(m.at) && now.Sub(m.at) < forecastTTL {
		return m.end, m.ok
	}
	end, ok := dir.EstimatedEnd(now)
	d.forecastMu.Lock()
	if d.forecasts == nil {
		d.forecasts = map[int64]forecastMemo{}
	}
	d.forecasts[tournamentID] = forecastMemo{events: n, at: now, end: end, ok: ok}
	d.forecastMu.Unlock()
	return end, ok
}

// clockAt is Clock at a given instant, which is what lets a test read the strip of a
// five-day tournament on its Wednesday morning.
func (d *Database) clockAt(tournamentID int64, now time.Time) (*ClockView, error) {
	dir, err := direction.Open(context.Background(), d.DirectionStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	st := dir.State()
	if st == nil {
		return nil, nil
	}
	if st.Finished {
		// A closed tournament has no pace, no next break and no end to forecast: the strip
		// said « pause à HH:MM » after the prize-giving.
		return &ClockView{Finished: true, Warnings: len(st.Warnings)}, nil
	}

	// The first launched match is where the tournament's own clock starts.
	var start time.Time
	for _, id := range st.MatchOrder {
		if m := st.Matches[id]; m != nil && !m.Start.IsZero() {
			start = m.Start
			break
		}
	}
	v := &ClockView{
		PlannedPerPoint: st.Config.MinPerPoint,
		Warnings:        len(st.Warnings),
	}
	if !start.IsZero() {
		v.ElapsedSeconds = int(now.Sub(start).Seconds())
		v.PlayingSeconds = int(direction.PlayingTime(st, now).Seconds())
		v.Day = direction.DayOfPlay(start, now)
	}
	if end, ok := d.estimatedEnd(tournamentID, dir, now); ok {
		v.EstimatedEnd = end.Format(time.RFC3339)
	}
	c := st.ClockAt(now, start)
	v.Played, v.Running = c.Played, c.Running
	if c.AvgPerPt > 0 {
		v.MinutesPerPoint = c.AvgPerPt.Minutes()
	}
	v.SlowMatches = len(st.SlowMatches(now, slowFactor))
	for _, b := range st.Config.Breaks {
		if b.Start.After(now) {
			v.NextBreak = b.Start.Format(time.RFC3339)
			break
		}
	}
	return v, nil
}
