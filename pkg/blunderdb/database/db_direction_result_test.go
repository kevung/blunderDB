package database

import (
	"context"
	"strings"
	"testing"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
)

// runningMatch starts one match and returns it.
func runningMatch(t *testing.T, d *Database, tID int64) *tournoi.Match {
	t.Helper()
	v, err := d.ConfirmAllProposals(tID)
	if err != nil {
		t.Fatal(err)
	}
	if len(v.Running) == 0 {
		t.Fatal("no match running")
	}
	return v.Running[0]
}

// TestResultWinnerOnly: the winner is the only thing required. A director often writes "Alice
// wins" and nothing else, and that is an ordinary result — not a half-filled form.
func TestResultWinnerOnly(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 24)
	m := runningMatch(t, d, tID)

	v, err := d.EnterResult(tID, string(m.ID), string(m.A), 0, 0, "")
	if err != nil {
		t.Fatalf("a result with no score must be accepted: %v", err)
	}
	for _, r := range v.Running {
		if r.ID == m.ID {
			t.Error("the match should be finished")
		}
	}
	if len(v.Warnings) != 0 {
		t.Errorf("a result with no score is not an inconsistency: %v", v.Warnings)
	}
}

// TestResultBeyondLengthIsAcceptedAndFlagged: during a tournament it is the director's word
// that stands. An impossible score is recorded, and shown as a warning.
func TestResultBeyondLengthIsAcceptedAndFlagged(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 24)
	m := runningMatch(t, d, tID)

	v, err := d.EnterResult(tID, string(m.ID), string(m.A), 99, 4, "")
	if err != nil {
		t.Fatalf("an impossible score must be recorded, not refused: %v", err)
	}
	// The match is recorded whatever the score: that much is this side's contract.
	finished := true
	for _, r := range v.Running {
		if r.ID == m.ID {
			finished = false
		}
	}
	if !finished {
		t.Error("the match should be finished")
	}
	found := false
	for _, w := range v.Warnings {
		if w.Code == tournoi.WarnScoreOverLength && w.Match == m.ID {
			found = true
		}
	}
	if !found {
		t.Errorf("a score beyond the length must be flagged as soon as it is entered: %v", v.Warnings)
	}
}

// TestResultNoteTravels: the remark — "ran out of time" — is rare and irreplaceable when it
// serves. It rides on the event and survives a replay.
func TestResultNoteTravels(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 24)
	m := runningMatch(t, d, tID)

	if _, err := d.EnterResult(tID, string(m.ID), string(m.A), 7, 3, "tombé au temps"); err != nil {
		t.Fatal(err)
	}
	events, err := d.DirectionStore().LoadEvents(context.Background(), tID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range events {
		if strings.Contains(string(e.Payload), "tombé au temps") {
			found = true
		}
	}
	if !found {
		t.Error("the remark must be in the log")
	}
}

// TestForfeitKeepsThePlayer: a forfeit for THIS match does not withdraw the player from the
// tournament — they go on to follow a loser's path.
func TestForfeitKeepsThePlayer(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 24)
	m := runningMatch(t, d, tID)
	absent := m.B

	v, err := d.EnterForfeit(tID, string(m.ID), string(m.A), "")
	if err != nil {
		t.Fatal(err)
	}
	stillThere := false
	for _, p := range v.Players {
		if p.ID == absent {
			stillThere = true
		}
	}
	if !stillThere {
		t.Error("a match forfeit does not remove the player from the tournament")
	}
}

// TestTableGrid: one cell per table, with what the director reads from two metres away.
func TestTableGrid(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 24)

	grid, err := d.TableGrid(tID)
	if err != nil {
		t.Fatal(err)
	}
	if len(grid) != 8 {
		t.Fatalf("%d cells, want 8 (the declared table count)", len(grid))
	}
	for _, c := range grid {
		if !c.Free {
			t.Errorf("table %d should be free before any match", c.Table)
		}
	}

	v, err := d.ConfirmAllProposals(tID)
	if err != nil {
		t.Fatal(err)
	}
	grid, err = d.TableGrid(tID)
	if err != nil {
		t.Fatal(err)
	}
	occupied := 0
	for _, c := range grid {
		if c.MatchID == "" {
			continue
		}
		occupied++
		if c.AName == "" || c.BName == "" {
			t.Errorf("table %d shows identifiers instead of names: %q %q", c.Table, c.AName, c.BName)
		}
		if c.Free || c.Unavailable || c.Reserved {
			t.Errorf("table %d is both busy and free", c.Table)
		}
	}
	if occupied != len(v.Running) {
		t.Errorf("%d occupied cells, %d matches running: every launched match must have a table", occupied, len(v.Running))
	}
	if occupied == 0 {
		t.Error("confirming everything should have filled tables")
	}
}

// TestTableGridShowsUnavailableAndReserved: a broken table and a reserved one are named as
// such, not shown as free.
func TestTableGridShowsUnavailableAndReserved(t *testing.T) {
	d := newTestDB(t)
	tID, err := d.CreateTournament("Salle contrainte", "2026-09-12", "")
	if err != nil {
		t.Fatal(err)
	}
	cfg := `{"name":"Salle contrainte","tables":{"count":4,"unavailable":[2],
		"reserved":[{"table":1,"section":"main","all_phases":true}]},
		"phases":[{"kind":"swiss_lives","length":7,"lives":2}]}`
	if err := d.CreateDirection(tID, cfg); err != nil {
		t.Fatal(err)
	}
	var players []string
	for i := 0; i < 8; i++ {
		id := string(rune('a' + i))
		players = append(players, `{"id":"`+id+`","name":"`+id+`"}`)
	}
	if err := d.StartDirection(tID, 3, "["+strings.Join(players, ",")+"]"); err != nil {
		t.Fatal(err)
	}
	grid, err := d.TableGrid(tID)
	if err != nil {
		t.Fatal(err)
	}
	byTable := map[int]TableCell{}
	for _, c := range grid {
		byTable[c.Table] = c
	}
	if !byTable[2].Unavailable {
		t.Errorf("table 2 is out of service, the grid says %+v", byTable[2])
	}
	if !byTable[1].Reserved {
		t.Errorf("table 1 is reserved to the main draw, the grid says %+v", byTable[1])
	}
	if !byTable[3].Free {
		t.Errorf("table 3 is reserved to nothing and should be free: %+v", byTable[3])
	}
}

// TestMoveAndCancel: a running match moves table, and one launched by mistake is cancelled.
func TestMoveAndCancel(t *testing.T) {
	d := newTestDB(t)
	// A roomy hall: sixteen tables for twelve matches, so there is somewhere to move to.
	tID, err := d.CreateTournament("Grande salle", "2026-09-12", "")
	if err != nil {
		t.Fatal(err)
	}
	cfg := `{"name":"Grande salle","tables":{"count":16},"phases":[
		{"kind":"swiss_lives","length":7,"lives":2,"mode":"continuous","target":16},
		{"kind":"lives_bracket","length":9}]}`
	if err := d.CreateDirection(tID, cfg); err != nil {
		t.Fatal(err)
	}
	var players []string
	for i := 0; i < 24; i++ {
		id := string(rune('a'+i%26)) + string(rune('a'+i/26))
		players = append(players, `{"id":"`+id+`","name":"`+id+`"}`)
	}
	if err := d.StartDirection(tID, 7, "["+strings.Join(players, ",")+"]"); err != nil {
		t.Fatal(err)
	}
	m := runningMatch(t, d, tID)

	// Move it to a table the grid says is free, which is what a director does.
	grid, err2 := d.TableGrid(tID)
	if err2 != nil {
		t.Fatal(err2)
	}
	target := 0
	for _, c := range grid {
		if c.Free {
			target = c.Table
			break
		}
	}
	if target == 0 {
		t.Fatal("no free table to move to")
	}
	if _, err := d.MoveMatchToTable(tID, string(m.ID), target); err != nil {
		t.Fatal(err)
	}
	if grid, err2 = d.TableGrid(tID); err2 != nil {
		t.Fatal(err2)
	}
	moved := false
	for _, c := range grid {
		if c.Table == target && c.MatchID == string(m.ID) {
			moved = true
		}
	}
	if !moved {
		t.Errorf("the match should be at table %d", target)
	}
	if _, err := d.MoveMatchToTable(tID, string(m.ID), 0); err == nil {
		t.Error("table 0 is not a table")
	}

	v, err := d.CancelMatch(tID, string(m.ID))
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range v.Running {
		if r.ID == m.ID {
			t.Error("a cancelled match is no longer running")
		}
	}
}
