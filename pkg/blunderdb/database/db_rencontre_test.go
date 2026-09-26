package database

import (
	"encoding/json"
	"slices"
	"testing"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
)

func eventCount(t *testing.T, d *Database, tID int64) int {
	t.Helper()
	v, err := d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	return v.EventCount
}

// A table out of service is declared ONCE for the room and lands, as one configuration change,
// in the log of every member Direction (ADR-0056 §2).
func TestRencontreTableOutOfServiceDeclaredOnce(t *testing.T) {
	d := newTestDB(t)
	_, a := directedTournament(t, d, 16)
	_, b := directedTournament(t, d, 12)

	r, err := d.CreateRencontre("Festival", "2026-10-03", "2026-10-04", 14)
	if err != nil {
		t.Fatal(err)
	}
	// Attaching shows first what changes: the event's 8 tables become the room's 14.
	p, err := d.PreviewAttachToRencontre(a, r.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(p.Changes, func(c ConfigChange) bool { return c.Code == "tableCount" && c.To == "14" }) {
		t.Fatalf("preview of the attach = %+v, want the table count going to 14", p.Changes)
	}
	for _, tID := range []int64{a, b} {
		if _, err := d.AttachToRencontre(tID, r.ID); err != nil {
			t.Fatalf("attach %d: %v", tID, err)
		}
		if v, _ := d.GetDirection(tID); v.Config.Tables.Count != 14 || v.RencontreID != r.ID {
			t.Fatalf("attached tournament %d: %d tables, rencontre %d", tID, v.Config.Tables.Count, v.RencontreID)
		}
	}

	beforeA, beforeB := eventCount(t, d, a), eventCount(t, d, b)
	view, err := d.SetRencontreTableOutOfService(r.ID, 3, true)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(view.Room.Unavailable, []int{3}) {
		t.Errorf("room after the gesture: %+v", view.Room)
	}
	for _, c := range []struct {
		id     int64
		before int
	}{{a, beforeA}, {b, beforeB}} {
		v, _ := d.GetDirection(c.id)
		if v.EventCount != c.before+1 {
			t.Errorf("tournament %d: %d event(s) written, want exactly one", c.id, v.EventCount-c.before)
		}
		if !slices.Equal(v.Config.Tables.Unavailable, []int{3}) {
			t.Errorf("tournament %d: out of service %v, want [3]", c.id, v.Config.Tables.Unavailable)
		}
		for _, pr := range v.Proposals {
			if pr.Table == 3 {
				t.Errorf("tournament %d proposes table 3, out of service", c.id)
			}
		}
	}
	// The same gesture again changes nothing and writes nothing.
	if _, err := d.SetRencontreTableOutOfService(r.ID, 3, true); err != nil {
		t.Fatal(err)
	}
	if n := eventCount(t, d, a); n != beforeA+1 {
		t.Errorf("repeating the gesture wrote %d more event(s)", n-beforeA-1)
	}
}

// A sister event's running matches are tables this one must not propose (ADR-0056 §2).
func TestRencontreProposalsAvoidTheSistersTables(t *testing.T) {
	d := newTestDB(t)
	_, a := directedTournament(t, d, 16)
	_, b := directedTournament(t, d, 16)
	r, err := d.CreateRencontre("Festival", "", "", 8)
	if err != nil {
		t.Fatal(err)
	}
	for _, tID := range []int64{a, b} {
		if _, err := d.AttachToRencontre(tID, r.ID); err != nil {
			t.Fatal(err)
		}
	}
	va, err := d.ConfirmAllProposals(a)
	if err != nil {
		t.Fatal(err)
	}
	var taken []int
	for _, m := range va.Running {
		taken = append(taken, m.Table)
	}
	if len(taken) == 0 {
		t.Fatal("the first event launched nothing")
	}
	vb, err := d.GetDirection(b)
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(taken)
	if !slices.Equal(vb.BusyTables, taken) {
		t.Errorf("busy tables seen from B = %v, want A's %v", vb.BusyTables, taken)
	}
	vb, err = d.ConfirmAllProposals(b)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range vb.Running {
		if slices.Contains(taken, m.Table) {
			t.Errorf("B launched %s on table %d, where A plays", m.ID, m.Table)
		}
	}

	// Detached, B sees an empty room again and keeps its tables.
	if err := d.DetachFromRencontre(b); err != nil {
		t.Fatal(err)
	}
	if vb, _ = d.GetDirection(b); len(vb.BusyTables) != 0 || vb.RencontreID != 0 || vb.Config.Tables.Count != 8 {
		t.Errorf("detached: busy %v, rencontre %d, %d tables", vb.BusyTables, vb.RencontreID, vb.Config.Tables.Count)
	}
}

// A player at a match of a sister event is not proposed in this one, and the panel says where
// they play; paired by hand anyway, the match is accepted and the seat shown on its table.
func TestRencontreBusyPlayersAreNotProposed(t *testing.T) {
	d := newTestDB(t)
	_, a := directedTournament(t, d, 16)
	_, b := directedTournament(t, d, 16) // the same sixteen names: the same persons
	r, err := d.CreateRencontre("Festival", "", "", 20)
	if err != nil {
		t.Fatal(err)
	}
	for _, tID := range []int64{a, b} {
		if _, err := d.AttachToRencontre(tID, r.ID); err != nil {
			t.Fatal(err)
		}
	}
	va, err := d.ConfirmAllProposals(a)
	if err != nil {
		t.Fatal(err)
	}
	seatOf := map[string]int{}
	for _, m := range va.Running {
		seatOf[string(m.A)], seatOf[string(m.B)] = m.Table, m.Table
	}
	if len(seatOf) != 16 {
		t.Fatalf("A seats %d players, want all 16", len(seatOf))
	}
	vb, err := d.GetDirection(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(vb.Elsewhere) != 16 {
		t.Fatalf("B sees %d players busy next door, want 16", len(vb.Elsewhere))
	}
	for id, seat := range vb.Elsewhere {
		if seat.Event != "Open de Lyon" || seat.Table != seatOf[id] {
			t.Errorf("%s: seen at %+v, plays at table %d", id, seat, seatOf[id])
		}
	}
	for _, p := range vb.Proposals {
		if p.Kind == tournoi.ActStartMatch && p.Table > 0 {
			t.Errorf("B proposes %s-%s at table %d while both play in A", p.A, p.B, p.Table)
		}
	}
	if vb, err = d.ConfirmAllProposals(b); err != nil || len(vb.Running) != 0 {
		t.Fatalf("B launched %d match(es) of busy players (%v)", len(vb.Running), err)
	}

	// The director pairs two of them by hand: accepted, and the grid says where they also sit.
	if _, err := d.StartMatchManually(b, "a", "b", 0, 0); err != nil {
		t.Fatalf("a manual pairing of busy players is refused: %v", err)
	}
	cells, err := d.TableGrid(b)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range cells {
		if c.MatchID == "" {
			continue
		}
		found = true
		if c.AElsewhere == nil || c.BElsewhere == nil || c.AElsewhere.Table != seatOf[c.A] {
			t.Errorf("manual match %s: seats %v / %v, want A's tables", c.MatchID, c.AElsewhere, c.BElsewhere)
		}
	}
	if !found {
		t.Error("the manual match is on no cell of the grid")
	}
}

// Deleting a Rencontre goes through the trash and detaches its events without touching them.
func TestRencontreTrashDetaches(t *testing.T) {
	d := newTestDB(t)
	_, a := directedTournament(t, d, 8)
	r, err := d.CreateRencontre("Festival", "", "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.AttachToRencontre(a, r.ID); err != nil {
		t.Fatal(err)
	}
	before := eventCount(t, d, a)
	if _, err := d.TrashRencontre(r.ID); err != nil {
		t.Fatal(err)
	}
	v, err := d.GetDirection(a)
	if err != nil {
		t.Fatalf("the event must survive its Rencontre: %v", err)
	}
	if v.RencontreID != 0 || v.EventCount != before || v.Config.Tables.Count != 10 {
		t.Errorf("after the trash: rencontre %d, %d events (had %d), %d tables", v.RencontreID, v.EventCount, before, v.Config.Tables.Count)
	}
	if n, _ := d.CountTrash(); n != 1 {
		t.Errorf("trash holds %d entries, want the Rencontre", n)
	}
}

// Restoring a Rencontre from the trash attaches its events again on the room's tables, as
// attaching does: detached, an event may have changed its own.
func TestRencontreRestoreRealignsTables(t *testing.T) {
	d := newTestDB(t)
	_, a := directedTournament(t, d, 8)
	r, err := d.CreateRencontre("Festival", "", "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.AttachToRencontre(a, r.ID); err != nil {
		t.Fatal(err)
	}
	trashID, err := d.TrashRencontre(r.ID)
	if err != nil {
		t.Fatal(err)
	}
	v, err := d.GetDirection(a)
	if err != nil {
		t.Fatal(err)
	}
	cfg := v.Config
	cfg.Tables.Count = 6
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.SetDirectionConfig(a, string(raw)); err != nil {
		t.Fatal(err)
	}
	before := eventCount(t, d, a)

	rid, err := d.RestoreFromTrash(trashID)
	if err != nil {
		t.Fatal(err)
	}
	if v, err = d.GetDirection(a); err != nil {
		t.Fatal(err)
	}
	if v.RencontreID != rid || v.Config.Tables.Count != 10 {
		t.Errorf("restored: rencontre %d (want %d), %d tables (want the room's 10)", v.RencontreID, rid, v.Config.Tables.Count)
	}
	if got := eventCount(t, d, a); got != before+1 {
		t.Errorf("the alignment wrote %d event(s), want one configuration change", got-before)
	}
}

// Changing a room setting from one member's settings is the room's gesture: the preview names the
// sister events, and saving writes it in their logs too.
func TestRencontreSettingsChangeReachesTheRoom(t *testing.T) {
	d := newTestDB(t)
	_, a := directedTournament(t, d, 8)
	_, b := directedTournament(t, d, 8)
	r, _ := d.CreateRencontre("Festival", "", "", 10)
	for _, tID := range []int64{a, b} {
		if _, err := d.AttachToRencontre(tID, r.ID); err != nil {
			t.Fatal(err)
		}
	}
	v, _ := d.GetDirection(a)
	cfg := v.Config
	cfg.Tables.Unavailable = []int{7}
	blob, _ := json.Marshal(cfg)
	p, err := d.PreviewDirectionConfig(a, string(blob))
	if err != nil {
		t.Fatal(err)
	}
	if len(p.AlsoFor) != 1 {
		t.Errorf("preview names %v, want the sister event", p.AlsoFor)
	}
	if err := d.SetDirectionConfig(a, string(blob)); err != nil {
		t.Fatal(err)
	}
	if vb, _ := d.GetDirection(b); !slices.Equal(vb.Config.Tables.Unavailable, []int{7}) {
		t.Errorf("sister event: out of service %v, want [7]", vb.Config.Tables.Unavailable)
	}
	cells, err := d.TableGrid(b)
	if err != nil || !cells[6].Unavailable {
		t.Errorf("sister grid: table 7 = %+v (%v)", cells[6], err)
	}
}
