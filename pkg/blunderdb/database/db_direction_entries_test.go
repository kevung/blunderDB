package database

import (
	"testing"
)

// preparedDirection directs a Tournament and leaves it in preparation, with no entry.
func preparedDirection(t *testing.T, d *Database) int64 {
	t.Helper()
	tID, err := d.CreateTournament("Open de Lyon", "2026-09-12", "Lyon")
	if err != nil {
		t.Fatal(err)
	}
	cfg := `{"name":"Open de Lyon","tables":{"count":8},"phases":[
		{"kind":"swiss_lives","length":7,"lives":2,"mode":"continuous","target":16},
		{"kind":"lives_bracket","length":9}]}`
	if err := d.CreateDirection(tID, cfg, 7); err != nil {
		t.Fatal(err)
	}
	return tID
}

// TestEnterOneByOne: twenty entries in a row, which is what a director types before a club
// tournament. Each one is written straight away and survives a reopen.
func TestEnterOneByOne(t *testing.T) {
	d := newTestDB(t)
	tID := preparedDirection(t, d)

	for i := 0; i < 20; i++ {
		name := "Joueur " + string(rune('A'+i))
		if _, err := d.AddParticipant(tID, name, "Lyon", float64(4+i%8)); err != nil {
			t.Fatalf("entry %d: %v", i, err)
		}
	}
	rows, err := d.Participants(tID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 20 {
		t.Fatalf("%d entries, want 20", len(rows))
	}
	if rows[0].Name != "Joueur A" || rows[19].Name != "Joueur T" {
		t.Errorf("entry order is not kept: %q … %q", rows[0].Name, rows[19].Name)
	}
	if rows[0].Club != "Lyon" || rows[0].Rating == 0 {
		t.Errorf("club and rating are not kept: %+v", rows[0])
	}
	for _, r := range rows {
		if r.State != "free" {
			t.Errorf("%s should be free before any match, not %q", r.Name, r.State)
		}
	}
	// They are in the log, so they survive a crash.
	v, err := d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	if v.EventCount != 21 { // created + twenty entries
		t.Errorf("%d events, want 21", v.EventCount)
	}
}

// TestEntryIdentifiersAreDistinct: two people who sign the same get distinct identifiers.
// blunderDB has no notion of a person behind a name, and merging them here would invent one.
func TestEntryIdentifiersAreDistinct(t *testing.T) {
	d := newTestDB(t)
	tID := preparedDirection(t, d)

	for i := 0; i < 3; i++ {
		if _, err := d.AddParticipant(tID, "Jean Dupont", "", 0); err != nil {
			t.Fatalf("entry %d: %v", i, err)
		}
	}
	rows, err := d.Participants(tID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("three people signing the same are three entries, got %d", len(rows))
	}
	seen := map[string]bool{}
	for _, r := range rows {
		if seen[r.ID] {
			t.Errorf("identifier %q used twice", r.ID)
		}
		seen[r.ID] = true
		if r.Name != "Jean Dupont" {
			t.Errorf("the name must stay what the director typed: %q", r.Name)
		}
	}
}

// TestCorrectingAnEntryKeepsItsIdentifier: correcting a name must not undo a Slot. The
// identifier is what a Slot points at.
func TestCorrectingAnEntryKeepsItsIdentifier(t *testing.T) {
	d := newTestDB(t)
	tID := preparedDirection(t, d)

	if _, err := d.AddParticipant(tID, "Kevin Ungr", "", 0); err != nil {
		t.Fatal(err)
	}
	rows, err := d.Participants(tID)
	if err != nil {
		t.Fatal(err)
	}
	id := rows[0].ID

	if _, err := d.UpdateParticipant(tID, id, "Kévin Unger", "Paris", 5.5); err != nil {
		t.Fatalf("correcting: %v", err)
	}
	rows, err = d.Participants(tID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("correcting must not create a second entry: %d", len(rows))
	}
	if rows[0].ID != id {
		t.Errorf("the identifier changed: %q then %q", id, rows[0].ID)
	}
	if rows[0].Name != "Kévin Unger" || rows[0].Club != "Paris" || rows[0].Rating != 5.5 {
		t.Errorf("the correction was not kept: %+v", rows[0])
	}
	if _, err := d.UpdateParticipant(tID, "personne", "X", "", 0); err == nil {
		t.Error("correcting an entry that does not exist must fail")
	}
}

// TestWithdrawImmediateAndDeferred: withdrawing now loses the running match by forfeit;
// withdrawing later lets it finish.
func TestWithdrawImmediateAndDeferred(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 24)
	m := runningMatch(t, d, tID)

	// Deferred: no longer paired, but the match goes on.
	if _, err := d.WithdrawParticipant(tID, string(m.A), true); err != nil {
		t.Fatal(err)
	}
	rows, err := d.Participants(tID)
	if err != nil {
		t.Fatal(err)
	}
	var leaving string
	for _, r := range rows {
		if r.ID == string(m.A) {
			leaving = r.State
		}
	}
	if leaving != "leaving" {
		t.Errorf("a deferred withdrawal shows as leaving, not %q", leaving)
	}
	v, err := d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	stillRunning := false
	for _, r := range v.Running {
		if r.ID == m.ID {
			stillRunning = true
		}
	}
	if !stillRunning {
		t.Error("a deferred withdrawal lets the running match finish")
	}

	// Immediate, on someone else: the running match is lost by forfeit.
	other := v.Running[len(v.Running)-1]
	if _, err := d.WithdrawParticipant(tID, string(other.A), false); err != nil {
		t.Fatal(err)
	}
	v, err = d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range v.Running {
		if r.ID == other.ID {
			t.Error("an immediate withdrawal ends the running match")
		}
	}
}

// TestEntrySuggestionsComeFromThePlayers: the entry field offers the Players of this database,
// so choosing one fixes the spelling the Matches carry.
func TestEntrySuggestionsComeFromThePlayers(t *testing.T) {
	d := newTestDBWithXG(t)
	sugg, err := d.EntrySuggestions()
	if err != nil {
		t.Fatal(err)
	}
	if len(sugg) == 0 {
		t.Fatal("a database with matches offers its players at entry time")
	}
	for _, s := range sugg {
		if s.Name == "" {
			t.Error("a suggestion without a name is useless")
		}
	}
}

// TestOpponentsAreNames: what a director checks before pairing two people by hand is who they
// have already met, by name.
func TestOpponentsAreNames(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 24)
	m := runningMatch(t, d, tID)
	if _, err := d.EnterResult(tID, string(m.ID), string(m.A), 7, 3, ""); err != nil {
		t.Fatal(err)
	}
	rows, err := d.Participants(tID)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if r.ID != string(m.A) {
			continue
		}
		if len(r.Opponents) == 0 {
			t.Fatal("having played, they have met someone")
		}
		if r.Opponents[0] == string(m.B) {
			t.Errorf("opponents are names, not identifiers: %q", r.Opponents[0])
		}
		if r.Wins != 1 {
			t.Errorf("%d wins, want 1", r.Wins)
		}
	}
}
