package direction

import (
	"testing"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
)

func at(day, h, m int) time.Time { return time.Date(2026, 9, day, h, m, 0, 0, time.Local) }

func stateOf(ms ...*tournoi.Match) *tournoi.State {
	st := &tournoi.State{Matches: map[tournoi.MatchID]*tournoi.Match{}}
	for _, m := range ms {
		st.MatchOrder = append(st.MatchOrder, m.ID)
		st.Matches[m.ID] = m
	}
	return st
}

// The playing time is the union of the running intervals: two overlapping matches count once,
// and the night between Monday and Tuesday counts not at all (#456, S5 « 48 h 45 »).
func TestPlayingTimeSkipsTheNight(t *testing.T) {
	st := stateOf(
		&tournoi.Match{ID: "m1", Status: tournoi.Finished, Start: at(21, 9, 0), End: at(21, 10, 0)},
		&tournoi.Match{ID: "m2", Status: tournoi.Finished, Start: at(21, 9, 30), End: at(21, 11, 0)},
		&tournoi.Match{ID: "m3", Status: tournoi.Cancelled, Start: at(21, 20, 0)},
		&tournoi.Match{ID: "m4", Status: tournoi.Running, Start: at(22, 9, 0)},
	)
	got := PlayingTime(st, at(22, 9, 45))
	if want := 2*time.Hour + 45*time.Minute; got != want {
		t.Fatalf("playing time = %v, want %v (9:00-11:00 on Monday, 9:00-9:45 on Tuesday)", got, want)
	}
}

func TestPlayingTimeOfNothing(t *testing.T) {
	if got := PlayingTime(stateOf(), at(21, 9, 0)); got != 0 {
		t.Fatalf("nothing launched, %v played", got)
	}
	if got := PlayingTime(nil, at(21, 9, 0)); got != 0 {
		t.Fatalf("no state, %v played", got)
	}
}

func TestDayOfPlay(t *testing.T) {
	for _, c := range []struct {
		start, now time.Time
		want       int
	}{
		{time.Time{}, at(21, 9, 0), 0},
		{at(21, 9, 0), at(21, 23, 59), 1},
		{at(21, 23, 0), at(22, 0, 30), 2},
		{at(21, 9, 0), at(23, 9, 0), 3},
	} {
		if got := DayOfPlay(c.start, c.now); got != c.want {
			t.Errorf("DayOfPlay(%v, %v) = %d, want %d", c.start, c.now, got, c.want)
		}
	}
}

// The play still to come is laid out after now and pushed past every declared break.
func TestAfterBreaks(t *testing.T) {
	lunch := tournoi.TimeRange{Start: at(21, 12, 30), End: at(21, 14, 0)}
	night := tournoi.TimeRange{Start: at(21, 23, 0), End: at(22, 9, 0)}
	breaks := []tournoi.TimeRange{night, lunch}
	for _, c := range []struct {
		name string
		now  time.Time
		play time.Duration
		want time.Time
	}{
		{"before any break", at(21, 10, 0), time.Hour, at(21, 11, 0)},
		{"across the lunch", at(21, 12, 0), time.Hour, at(21, 14, 30)},
		{"during the lunch", at(21, 13, 0), time.Hour, at(21, 15, 0)},
		{"across the lunch and the night", at(21, 12, 0), 10 * time.Hour, at(22, 9, 30)},
		{"nothing left", at(21, 12, 0), 0, at(21, 12, 0)},
	} {
		if got := AfterBreaks(c.now, c.play, breaks); !got.Equal(c.want) {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}
