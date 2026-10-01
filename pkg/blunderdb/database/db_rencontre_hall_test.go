package database

import (
	"testing"
)

// hallSetup is a room of ten tables: A seats its sixteen players on the first tables, B pairs
// two of the same persons by hand on table 9.
func hallSetup(t *testing.T) (d *Database, rID, a, b int64, bMatch string) {
	t.Helper()
	d = newTestDB(t)
	_, a = directedTournament(t, d, 16)
	_, b = directedTournament(t, d, 16)
	r, err := d.CreateRencontre("Festival", "", "", 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, tID := range []int64{a, b} {
		if _, err := d.AttachToRencontre(tID, r.ID); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := d.ConfirmAllProposals(a); err != nil {
		t.Fatal(err)
	}
	vb, err := d.StartMatchManually(b, "a", "b", 0, 9)
	if err != nil || len(vb.Running) != 1 {
		t.Fatalf("manual pairing in B: %v", err)
	}
	return d, r.ID, a, b, string(vb.Running[0].ID)
}

func TestRencontreTableGridMergesTheRoom(t *testing.T) {
	d, rID, a, b, bMatch := hallSetup(t)
	h, err := d.RencontreTableGrid(rID)
	if err != nil {
		t.Fatal(err)
	}
	if len(h.Events) != 2 || h.Events[0].TournamentID != a || h.Events[1].TournamentID != b || h.Events[1].Index != 1 {
		t.Fatalf("events = %+v", h.Events)
	}
	if len(h.Events[0].Names) != 16 {
		t.Errorf("A's names: %d, want 16", len(h.Events[0].Names))
	}
	if len(h.Cells) != 10 {
		t.Fatalf("the hall has %d cells, want one per table (10)", len(h.Cells))
	}
	aMatches := 0
	for i, c := range h.Cells {
		if c.Table != i+1 {
			t.Errorf("cell %d is table %d", i, c.Table)
		}
		if c.Elsewhere != "" {
			t.Errorf("table %d says elsewhere %q: in the hall no table is elsewhere", c.Table, c.Elsewhere)
		}
		switch {
		case c.Table == 9:
			if c.TournamentID != b || c.MatchID != bMatch || c.EventIndex != 1 {
				t.Errorf("table 9 = %+v, want B's manual match", c)
			}
		case c.MatchID != "":
			aMatches++
			if c.TournamentID != a || c.EventIndex != 0 || c.Event == "" {
				t.Errorf("table %d = %+v, want an A match", c.Table, c)
			}
		default:
			if c.TournamentID != 0 || c.EventIndex != -1 || !c.Free {
				t.Errorf("idle table %d = %+v, want free and of no event", c.Table, c)
			}
		}
	}
	if aMatches != 8 {
		t.Errorf("A runs %d matches in the hall, want 8", aMatches)
	}
}

// Dropping B's match on a table where A plays swaps the two: one table change in each log,
// written together, and the room never holds two matches on one table.
func TestMoveMatchSwapsWithASisterEvent(t *testing.T) {
	d, rID, a, b, bMatch := hallSetup(t)
	h, err := d.RencontreTableGrid(rID)
	if err != nil {
		t.Fatal(err)
	}
	var target int
	var aMatch string
	for _, c := range h.Cells {
		if c.TournamentID == a && c.MatchID != "" {
			target, aMatch = c.Table, c.MatchID
			break
		}
	}
	beforeA, beforeB := eventCount(t, d, a), eventCount(t, d, b)
	if _, err := d.MoveMatchToTable(b, bMatch, target); err != nil {
		t.Fatalf("a swap with a sister event is refused: %v", err)
	}
	if eventCount(t, d, a) != beforeA+1 || eventCount(t, d, b) != beforeB+1 {
		t.Errorf("events written: A %d→%d, B %d→%d; want one each", beforeA, eventCount(t, d, a), beforeB, eventCount(t, d, b))
	}
	h, err = d.RencontreTableGrid(rID)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range h.Cells {
		if c.Shared {
			t.Errorf("table %d holds two matches after a swap", c.Table)
		}
		switch c.Table {
		case target:
			if c.TournamentID != b || c.MatchID != bMatch {
				t.Errorf("table %d = %s of %d, want B's %s", target, c.MatchID, c.TournamentID, bMatch)
			}
		case 9:
			if c.TournamentID != a || c.MatchID != aMatch {
				t.Errorf("table 9 = %s of %d, want A's %s", c.MatchID, c.TournamentID, aMatch)
			}
		}
	}
	// The same gesture puts them back.
	if _, err := d.MoveMatchToTable(b, bMatch, 9); err != nil {
		t.Fatal(err)
	}
	if h, _ = d.RencontreTableGrid(rID); h.Cells[8].TournamentID != b || h.Cells[target-1].MatchID != aMatch {
		t.Errorf("swapping back: table 9 = %+v, table %d = %+v", h.Cells[8], target, h.Cells[target-1])
	}
}

// Moving onto a free table of the room touches no sister log.
func TestMoveMatchToAFreeTableInTheRoom(t *testing.T) {
	d, rID, a, b, bMatch := hallSetup(t)
	beforeA := eventCount(t, d, a)
	if _, err := d.MoveMatchToTable(b, bMatch, 10); err != nil {
		t.Fatal(err)
	}
	if eventCount(t, d, a) != beforeA {
		t.Error("a move to a free table wrote in the sister log")
	}
	h, err := d.RencontreTableGrid(rID)
	if err != nil {
		t.Fatal(err)
	}
	if h.Cells[9].MatchID != bMatch || !h.Cells[8].Free {
		t.Errorf("table 9 = %+v, table 10 = %+v", h.Cells[8], h.Cells[9])
	}
}
