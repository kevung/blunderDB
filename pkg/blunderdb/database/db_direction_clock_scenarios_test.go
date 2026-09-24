package database

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
)

// The clock strip on the two scenarios of the simulation that broke it (#456, D8.3;
// tasks/nicomaque/simulation-2026-09/rapport/S5.md, scenarios-go § 8): S1, one club evening
// that ends long after the hall closes, and S5, five days where the elapsed time counted the
// nights as play and no end was forecast anywhere.

// hall is the time of day the tables are open: matches are launched only inside it.
type hall struct{ open, close int }

func (h hall) inside(t time.Time) bool { return t.Hour() >= h.open && t.Hour() < h.close }

func (h hall) nextOpening(t time.Time) time.Time {
	o := time.Date(t.Year(), t.Month(), t.Day(), h.open, 0, 0, 0, t.Local().Location())
	if !t.Before(o) {
		o = o.AddDate(0, 0, 1)
	}
	return o
}

// directedAt creates a Direction with n entrants under cfg (JSON).
func directedAt(t *testing.T, d *Database, n int, cfg string) int64 {
	t.Helper()
	tID, err := d.CreateTournament("Scénario", "2026-10-05", "Lyon")
	if err != nil {
		t.Fatal(err)
	}
	if err := d.CreateDirection(tID, cfg, 11); err != nil {
		t.Fatal(err)
	}
	var ps []string
	for i := 0; i < n; i++ {
		ps = append(ps, fmt.Sprintf(`{"id":"p%02d","name":"Joueur %02d","rating":%d}`, i, i, 3+i%5))
	}
	if err := d.EnterParticipants(tID, "["+strings.Join(ps, ",")+"]"); err != nil {
		t.Fatal(err)
	}
	return tID
}

// playUntil plays the tournament from `from` to `until` with the host's discipline: every
// proposal is confirmed as soon as the hall is open, a match lasts minPerPoint per point, and
// A wins. Nothing is launched while the hall is closed, which is what a night is.
func playUntil(t *testing.T, d *Database, tID int64, from, until time.Time, h hall, minPerPoint float64) {
	t.Helper()
	ctx := context.Background()
	dir, err := direction.Open(ctx, d.DirectionStore(), tID)
	if err != nil {
		t.Fatal(err)
	}
	ends := map[tournoi.MatchID]time.Time{}
	now := from
	for steps := 0; steps < 5000 && now.Before(until) && !dir.State().Finished; steps++ {
		progressed := false
		if h.inside(now) {
			for _, a := range dir.ProposeAt(now) {
				if a.Kind == tournoi.ActWait || a.Kind == tournoi.ActFinish || a.Reason == tournoi.ReasonWaitingTable {
					continue
				}
				if err := confirmAt(ctx, dir, a, now); err != nil {
					t.Fatalf("%v at %v: %v", a.Kind, now, err)
				}
				progressed = true
				if a.Kind == tournoi.ActStartMatch {
					m := dir.State().Running()
					last := m[len(m)-1]
					ends[last.ID] = now.Add(time.Duration(minPerPoint * float64(last.Length) * float64(time.Minute)))
				} else {
					break
				}
			}
		}
		if progressed {
			continue
		}
		var next tournoi.MatchID
		for id, e := range ends {
			if next == "" || e.Before(ends[next]) || e.Equal(ends[next]) && id < next {
				next = id
			}
		}
		if next == "" {
			now = h.nextOpening(now)
			continue
		}
		if !ends[next].Before(until) {
			return
		}
		now = ends[next]
		m := dir.State().Matches[next]
		delete(ends, next)
		if err := dir.Apply(ctx, tournoi.ResultEvent(next, m.A, m.Length, 0, now)); err != nil {
			t.Fatal(err)
		}
		if !h.inside(now) {
			if len(ends) == 0 {
				now = h.nextOpening(now)
			}
		}
	}
}

func localAt(y int, mo time.Month, day, hh, mm int) time.Time {
	return time.Date(y, mo, day, hh, mm, 0, 0, time.Local)
}

// S1: twenty players, eight tables, one evening from 19:30. On a single day the strip keeps
// the elapsed time, and the forecast end is the engine's, laid after now.
func TestClockS1OneEvening(t *testing.T) {
	d := newTestDB(t)
	tID := directedAt(t, d, 20, `{"name":"Soir de club","min_per_point":6,"tables":{"count":8},"phases":[
		{"kind":"swiss_lives","length":5,"lives":2,"mode":"continuous","target":8},
		{"kind":"lives_bracket","length":7}]}`)
	start := localAt(2026, 10, 5, 19, 30)
	now := localAt(2026, 10, 5, 21, 0)
	playUntil(t, d, tID, start, now, hall{open: 8, close: 24}, 6)

	c, err := d.clockAt(tID, now)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("S1 at 21:00: %+v", *c)
	if c.Played == 0 {
		t.Fatal("the scenario played nothing")
	}
	if c.Day != 1 {
		t.Errorf("one evening is day 1, got %d", c.Day)
	}
	if c.ElapsedSeconds != int(now.Sub(start).Seconds()) {
		t.Errorf("elapsed = %d s, want %d s from the first match", c.ElapsedSeconds, int(now.Sub(start).Seconds()))
	}
	if c.PlayingSeconds <= 0 || c.PlayingSeconds > c.ElapsedSeconds {
		t.Errorf("playing time %d s must be within the elapsed %d s", c.PlayingSeconds, c.ElapsedSeconds)
	}
	end, err := time.Parse(time.RFC3339, c.EstimatedEnd)
	if err != nil {
		t.Fatalf("no estimated end on a running evening: %q (%v)", c.EstimatedEnd, err)
	}
	if !end.After(now) || end.Sub(now) > 12*time.Hour {
		t.Errorf("estimated end %v is not a plausible end of the evening after %v", end, now)
	}

	// The same instant gives the same forecast: it is memoised on the log, not redrawn.
	again, err := d.clockAt(tID, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if again.EstimatedEnd != c.EstimatedEnd {
		t.Errorf("a minute later with nothing written, the end moved: %s → %s", c.EstimatedEnd, again.EstimatedEnd)
	}
}

// S5: fifty-one players, five days, the hall open 9:00-23:00. On the Wednesday morning the
// strip read « 48 h 45 » — two nights counted as play. It now reads day 3 and the playing
// time, which leaves both nights out, and the declared night breaks push the forecast end.
func TestClockS5FiveDays(t *testing.T) {
	d := newTestDB(t)
	var breaks []string
	for day := 5; day < 10; day++ {
		breaks = append(breaks, fmt.Sprintf(`{"start":%q,"end":%q}`,
			localAt(2026, 10, day, 23, 0).Format(time.RFC3339), localAt(2026, 10, day+1, 9, 0).Format(time.RFC3339)))
	}
	tID := directedAt(t, d, 51, `{"name":"Cinq jours","min_per_point":8,"tables":{"count":6},
		"breaks":[`+strings.Join(breaks, ",")+`],
		"phases":[{"kind":"swiss_lives","length":9,"lives":3,"mode":"continuous"}]}`)
	start := localAt(2026, 10, 5, 9, 0)
	wednesday := localAt(2026, 10, 7, 9, 0)
	playUntil(t, d, tID, start, wednesday, hall{open: 9, close: 23}, 8)

	c, err := d.clockAt(tID, wednesday)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("S5 on Wednesday 9:00: %+v", *c)
	if c.Finished {
		t.Fatal("the scenario must still be running on Wednesday morning")
	}
	if c.Day != 3 {
		t.Errorf("Wednesday is day 3, got %d", c.Day)
	}
	if c.ElapsedSeconds != 48*3600 {
		t.Errorf("elapsed = %d s, want 48 h", c.ElapsedSeconds)
	}
	// Two days of fourteen open hours at most, whatever the pace: the nights are out.
	if c.PlayingSeconds <= 0 || c.PlayingSeconds > 2*15*3600 {
		t.Errorf("playing time = %.1f h, want at most two open days", float64(c.PlayingSeconds)/3600)
	}
	end, err := time.Parse(time.RFC3339, c.EstimatedEnd)
	if err != nil {
		t.Fatalf("no estimated end on Wednesday morning: %q (%v)", c.EstimatedEnd, err)
	}
	if !end.After(wednesday) {
		t.Errorf("estimated end %v before now", end)
	}
	if h := end.Hour(); h < 9 || h == 23 && end.Minute() > 0 {
		t.Errorf("estimated end %v falls inside a declared night", end)
	}
	if c.NextBreak == "" {
		t.Error("the Wednesday night is declared: it is the next break")
	}
}

// A closed tournament says nothing: no pace, no break, no end (#456).
func TestClockAfterClosing(t *testing.T) {
	d := newTestDB(t)
	tID := directedAt(t, d, 8, `{"name":"Court","min_per_point":2,"tables":{"count":4},
		"breaks":[{"start":"2099-01-01T12:00:00Z","end":"2099-01-01T13:00:00Z"}],
		"phases":[{"kind":"bracket","length":3}]}`)
	start := localAt(2026, 10, 5, 19, 0)
	playUntil(t, d, tID, start, start.Add(24*time.Hour), hall{open: 0, close: 24}, 2)
	if _, err := d.CloseDirection(tID); err != nil {
		t.Fatal(err)
	}
	c, err := d.clockAt(tID, start.Add(25*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if !c.Finished || c.NextBreak != "" || c.EstimatedEnd != "" || c.MinutesPerPoint != 0 {
		t.Errorf("a closed tournament's strip is empty: %+v", c)
	}
}
