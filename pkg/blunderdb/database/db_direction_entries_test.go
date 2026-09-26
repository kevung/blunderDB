package database

import (
	"context"
	"strings"
	"testing"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
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

// participantState reads a participant's state code from the Players view.
func participantState(t *testing.T, d *Database, tID int64, id string) string {
	t.Helper()
	rows, err := d.Participants(tID)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if r.ID == id {
			return r.State
		}
	}
	t.Fatalf("no participant %q", id)
	return ""
}

// TestUpdateWithdrawnKeepsWithdrawal: correcting a withdrawn player's club must not silently
// put them back in play, and it is ONE decision in the journal.
func TestUpdateWithdrawnKeepsWithdrawal(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 8)
	if _, err := d.WithdrawParticipant(tID, "aa", false); err != nil {
		t.Fatal(err)
	}
	before := len(journalOf(t, d, tID))
	if _, err := d.UpdateParticipant(tID, "aa", "Joueur aa", "Lyon", 5); err != nil {
		t.Fatal(err)
	}
	j := journalOf(t, d, tID)
	if len(j) != before+1 || j[len(j)-1].Kind != tournoi.EvPlayerUpdated {
		t.Fatalf("a correction writes one player_updated, got %d events ending in %q", len(j)-before, j[len(j)-1].Kind)
	}
	if got := participantState(t, d, tID, "aa"); got != "withdrawn" {
		t.Fatalf("a corrected entry must stay withdrawn, got %q", got)
	}
	rows, _ := d.Participants(tID)
	for _, r := range rows {
		if r.ID == "aa" && r.Club != "Lyon" {
			t.Errorf("the correction itself must take, got club %q", r.Club)
		}
	}
	v, err := d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range v.Proposals {
		if a.A == "aa" || a.B == "aa" {
			t.Errorf("a withdrawn player is proposed again after a correction: %+v", a)
		}
	}
}

// TestOldCorrectionPairStillReads: journals written before player_updated recorded a
// correction of a withdrawn entry as the entry added again, then withdrawn again at the same
// instant. Replayed, that pair still leaves the corrected, withdrawn entry.
func TestOldCorrectionPairStillReads(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 8)
	if _, err := d.WithdrawParticipant(tID, "aa", false); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	dir, err := direction.Open(ctx, d.DirectionStore(), tID)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	p := *dir.State().Players["aa"]
	p.Club = "Lyon"
	for _, ev := range []tournoi.Event{tournoi.PlayerAddedEvent(p, now), tournoi.PlayerWithdrawnEvent(p.ID, now)} {
		if err := dir.Apply(ctx, ev); err != nil {
			t.Fatal(err)
		}
	}
	if got := participantState(t, d, tID, "aa"); got != "withdrawn" {
		t.Fatalf("the old pair replays as a withdrawn entry, got %q", got)
	}
	rows, _ := d.Participants(tID)
	for _, r := range rows {
		if r.ID == "aa" && r.Club != "Lyon" {
			t.Errorf("the old pair's correction is lost, club %q", r.Club)
		}
	}
}

// TestReinstateParticipant: coming back is a named gesture, not a side effect.
func TestReinstateParticipant(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 8)
	if _, err := d.ReinstateParticipant(tID, "aa"); err == nil {
		t.Error("reinstating someone who never left must be refused")
	}
	if _, err := d.WithdrawParticipant(tID, "aa", false); err != nil {
		t.Fatal(err)
	}
	if _, err := d.ReinstateParticipant(tID, "aa"); err != nil {
		t.Fatal(err)
	}
	if got := participantState(t, d, tID, "aa"); got == "withdrawn" {
		t.Fatalf("a reinstated player is back in play, got %q", got)
	}
}

// TestMakeParticipantAbsentUntilTime: D7.1, an absence for a time keeps the standing it froze
// and the return is one call away.
func TestMakeParticipantAbsentUntilTime(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 8)
	var before ParticipantRow
	rowsBefore, err := d.Participants(tID)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rowsBefore {
		if r.ID == "aa" {
			before = r
		}
	}

	until := time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339)
	if _, err := d.MakeParticipantAbsent(tID, "aa", until, 0); err != nil {
		t.Fatal(err)
	}
	rows, err := d.Participants(tID)
	if err != nil {
		t.Fatal(err)
	}
	var after ParticipantRow
	for _, r := range rows {
		if r.ID == "aa" {
			after = r
		}
	}
	if after.State != "absent" {
		t.Fatalf("state = %q, want absent", after.State)
	}
	if after.AbsentUntil == "" {
		t.Error("an absence until an hour must carry AbsentUntil")
	}
	if after.Wins != before.Wins || after.Lives != before.Lives {
		t.Error("an absence must not change the standing it froze")
	}

	if _, err := d.MakeParticipantAvailable(tID, "aa"); err != nil {
		t.Fatal(err)
	}
	if got := participantState(t, d, tID, "aa"); got != "free" {
		t.Errorf("returning makes the row free again, got %q", got)
	}
}

// TestMakeParticipantAbsentNeedsADeadline: neither an hour nor a round is not an absence.
func TestMakeParticipantAbsentNeedsADeadline(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 8)
	if _, err := d.MakeParticipantAbsent(tID, "aa", "", 0); err == nil {
		t.Error("an absence needs either a time or a round")
	}
}

// TestMakeParticipantAbsentUntilRoundNeedsARoundsPhase: a round deadline only means something
// where rounds are counted.
func TestMakeParticipantAbsentUntilRoundNeedsARoundsPhase(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 8) // continuous mode, no rounds
	if _, err := d.MakeParticipantAbsent(tID, "aa", "", 3); err == nil {
		t.Error("a round deadline outside a rounds phase must be refused")
	}
}

// TestMakeParticipantAbsentUntilRound: in a swiss by rounds, "back at round N" is the other half
// of D7.1.
func TestMakeParticipantAbsentUntilRound(t *testing.T) {
	d := newTestDB(t)
	tID, err := d.CreateTournament("Open de Lyon", "2026-09-12", "Lyon")
	if err != nil {
		t.Fatal(err)
	}
	cfg := `{"name":"Open de Lyon","tables":{"count":8},"phases":[
		{"kind":"swiss_lives","length":7,"lives":2,"mode":"rounds","target":16}]}`
	if err := d.CreateDirection(tID, cfg, 7); err != nil {
		t.Fatal(err)
	}
	var players []string
	for i := 0; i < 8; i++ {
		id := string(rune('a'+i)) + "a"
		players = append(players, `{"id":"`+id+`","name":"Joueur `+id+`"}`)
	}
	if err := d.EnterParticipants(tID, "["+strings.Join(players, ",")+"]"); err != nil {
		t.Fatal(err)
	}

	if _, err := d.MakeParticipantAbsent(tID, "aa", "", 2); err != nil {
		t.Fatal(err)
	}
	if got := participantState(t, d, tID, "aa"); got != "absent" {
		t.Fatalf("state = %q, want absent", got)
	}
}
