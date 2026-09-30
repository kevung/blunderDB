package database

import (
	"context"
	"strings"
	"testing"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
)

// hallWithTwoMatches opens a four-table hall and starts two matches by hand, on tables 1 and 2.
func hallWithTwoMatches(t *testing.T) (*Database, int64, string, string) {
	t.Helper()
	d := newTestDB(t)
	tID, err := d.CreateTournament("Salle", "2026-09-12", "")
	if err != nil {
		t.Fatal(err)
	}
	cfg := `{"name":"Salle","tables":{"count":4},"phases":[
		{"kind":"swiss_lives","length":7,"lives":2,"mode":"continuous","target":4}]}`
	if err := d.CreateDirection(tID, cfg, 7); err != nil {
		t.Fatal(err)
	}
	var players []string
	for _, id := range []string{"p0", "p1", "p2", "p3", "p4", "p5"} {
		players = append(players, `{"id":"`+id+`","name":"Joueur `+id+`"}`)
	}
	if err := d.EnterParticipants(tID, "["+strings.Join(players, ",")+"]"); err != nil {
		t.Fatal(err)
	}
	if _, err := d.StartMatchManually(tID, "p0", "p1", 0, 1); err != nil {
		t.Fatal(err)
	}
	v, err := d.StartMatchManually(tID, "p2", "p3", 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	var m1, m2 string
	for _, m := range v.Running {
		switch m.Table {
		case 1:
			m1 = string(m.ID)
		case 2:
			m2 = string(m.ID)
		}
	}
	if m1 == "" || m2 == "" {
		t.Fatalf("expected matches on tables 1 and 2: %+v", v.Running)
	}
	return d, tID, m1, m2
}

func tableOfMatch(t *testing.T, d *Database, tID int64, matchID string) int {
	t.Helper()
	v, err := d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range v.Running {
		if string(m.ID) == matchID {
			return m.Table
		}
	}
	t.Fatalf("match %s is not running", matchID)
	return 0
}

// Moving a match onto an occupied table swaps the two: no table ever holds two matches, and
// moving again puts everything back.
func TestMoveMatchToOccupiedTableSwaps(t *testing.T) {
	d, tID, m1, m2 := hallWithTwoMatches(t)
	if _, err := d.MoveMatchToTable(tID, m1, 2); err != nil {
		t.Fatal(err)
	}
	if got := tableOfMatch(t, d, tID, m1); got != 2 {
		t.Errorf("moved match: table %d, want 2", got)
	}
	if got := tableOfMatch(t, d, tID, m2); got != 1 {
		t.Errorf("displaced match: table %d, want 1 (swapped)", got)
	}
	for _, w := range mustWarnings(t, d, tID) {
		if w.Code == tournoi.WarnTableShared {
			t.Errorf("a swap leaves no shared table: %+v", w)
		}
	}
	// The swap is undone by the same gesture.
	if _, err := d.MoveMatchToTable(tID, m1, 1); err != nil {
		t.Fatal(err)
	}
	if tableOfMatch(t, d, tID, m1) != 1 || tableOfMatch(t, d, tID, m2) != 2 {
		t.Error("moving back must restore both tables")
	}
	// Moving a match to its own table is a no-op, not an error.
	if _, err := d.MoveMatchToTable(tID, m1, 1); err != nil {
		t.Errorf("moving to the same table: %v", err)
	}
}

// A match started by hand on an occupied table is refused: it would hide the other one.
func TestStartMatchManuallyRefusesOccupiedTable(t *testing.T) {
	d, tID, _, _ := hallWithTwoMatches(t)
	if _, err := d.StartMatchManually(tID, "p4", "p5", 0, 1); err == nil {
		t.Fatal("starting a match on an occupied table must be refused")
	}
	if _, err := d.StartMatchManually(tID, "p4", "p5", 0, 3); err != nil {
		t.Fatalf("a free table must be accepted: %v", err)
	}
}

// A shared table inherited from an older log (the engine accepts it with a warning) must not
// hide one of the two matches: both appear in the grid, flagged as a conflict.
func TestTableGridKeepsBothMatchesOfASharedTable(t *testing.T) {
	d, tID, m1, m2 := hallWithTwoMatches(t)
	ctx := context.Background()
	dir, err := direction.Open(ctx, d.DirectionStore(), tID)
	if err != nil {
		t.Fatal(err)
	}
	if err := dir.Apply(ctx, tournoi.TableChangedEvent(tournoi.MatchID(m2), 1, time.Now())); err != nil {
		t.Fatal(err)
	}
	cells, err := d.TableGrid(tID)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]TableCell{}
	for _, c := range cells {
		if c.MatchID != "" {
			seen[c.MatchID] = c
		}
	}
	for _, id := range []string{m1, m2} {
		c, ok := seen[id]
		if !ok {
			t.Fatalf("match %s vanished from the grid: %+v", id, cells)
		}
		if c.Table != 1 || !c.Shared {
			t.Errorf("match %s: table %d shared %v, want table 1 flagged shared", id, c.Table, c.Shared)
		}
	}
	// A swap resolves the conflict: moving one of the two to a free table.
	if _, err := d.MoveMatchToTable(tID, m2, 2); err != nil {
		t.Fatal(err)
	}
	if cells, err = d.TableGrid(tID); err != nil {
		t.Fatal(err)
	}
	for _, c := range cells {
		if c.Shared {
			t.Errorf("no conflict left, yet %+v is flagged", c)
		}
	}
}

func mustWarnings(t *testing.T, d *Database, tID int64) []tournoi.Warning {
	t.Helper()
	dir, err := direction.Open(context.Background(), d.DirectionStore(), tID)
	if err != nil {
		t.Fatal(err)
	}
	return dir.Warnings()
}
