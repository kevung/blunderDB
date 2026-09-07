package database

import (
	"testing"
)

// TestClockBeforeAnyMatch: nothing has been launched, so the tournament's own clock has not
// started. A tournament prepared the evening before has not been running since then.
func TestClockBeforeAnyMatch(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 24)

	c, err := d.Clock(tID)
	if err != nil {
		t.Fatal(err)
	}
	if c == nil {
		t.Fatal("a started Direction has a clock")
	}
	if c.ElapsedSeconds != 0 {
		t.Errorf("nothing launched, %d seconds elapsed", c.ElapsedSeconds)
	}
	if c.Played != 0 || c.Running != 0 {
		t.Errorf("no match played or running: %+v", c)
	}
	if c.PlannedPerPoint <= 0 {
		t.Errorf("the planned pace comes from the configuration: %v", c.PlannedPerPoint)
	}
}

// TestClockCountsMatches: launching and finishing matches moves the counters, and the observed
// pace appears once something has actually been played.
func TestClockCountsMatches(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 24)

	v, err := d.ConfirmAllProposals(tID)
	if err != nil {
		t.Fatal(err)
	}
	c, err := d.Clock(tID)
	if err != nil {
		t.Fatal(err)
	}
	if c.Running != len(v.Running) {
		t.Errorf("%d running on the clock, %d in the view", c.Running, len(v.Running))
	}
	if c.ElapsedSeconds < 0 {
		t.Errorf("elapsed cannot be negative: %d", c.ElapsedSeconds)
	}

	m := v.Running[0]
	if _, err := d.EnterResult(tID, string(m.ID), string(m.A), 7, 3, ""); err != nil {
		t.Fatal(err)
	}
	c, err = d.Clock(tID)
	if err != nil {
		t.Fatal(err)
	}
	if c.Played != 1 {
		t.Errorf("%d played, want 1", c.Played)
	}
	if c.Running != len(v.Running)-1 {
		t.Errorf("%d running after one result", c.Running)
	}
}

// TestClockCountsWarnings: the strip carries the count, so a director sees there is something
// to look at without a window interrupting them.
func TestClockCountsWarnings(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 24)
	m := runningMatch(t, d, tID)

	c, err := d.Clock(tID)
	if err != nil {
		t.Fatal(err)
	}
	if c.Warnings != 0 {
		t.Fatalf("nothing is wrong yet: %d", c.Warnings)
	}
	// A score beyond the length is recorded and flagged.
	if _, err := d.EnterResult(tID, string(m.ID), string(m.A), 99, 4, ""); err != nil {
		t.Fatal(err)
	}
	c, err = d.Clock(tID)
	if err != nil {
		t.Fatal(err)
	}
	if c.Warnings == 0 {
		t.Error("the strip must count what the engine complains about")
	}
}
