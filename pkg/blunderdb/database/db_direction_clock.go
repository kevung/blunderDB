package database

import (
	"context"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
)

// The clock of a Direction (ADR-0047 §6, issue #377).
//
// A director wonders all evening whether they will finish before midnight, and watches the
// tables that are dragging. The engine can answer both; what was missing was somewhere to read
// it.
//
// The threshold for "slow" is 1.5× the expected duration, which is the default the UX document
// fixes — a default, not a law of nature.
const slowFactor = 1.5

// ClockView is the clock strip: one line of text, not a dashboard.
type ClockView struct {
	// ElapsedSeconds counts from the first match launched, not from the Direction's creation:
	// a tournament prepared the evening before has not been running since then.
	ElapsedSeconds  int     `json:"elapsedSeconds"`
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
	dir, err := direction.Open(context.Background(), d.DirectionStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	st := dir.State()
	if st == nil {
		return nil, nil
	}
	now := time.Now()

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
