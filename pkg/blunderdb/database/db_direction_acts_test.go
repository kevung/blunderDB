package database

import (
	"encoding/json"
	"strings"
	"testing"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
)

// startedDirection creates a Tournament, directs it and starts it with n entrants.
func startedDirection(t *testing.T, d *Database, n int) int64 {
	t.Helper()
	tID, err := d.CreateTournament("Open de Lyon", "2026-09-12", "Lyon")
	if err != nil {
		t.Fatal(err)
	}
	cfg := `{"name":"Open de Lyon","tables":{"count":8},"phases":[
		{"kind":"swiss_lives","length":7,"lives":2,"mode":"continuous","target":16},
		{"kind":"lives_bracket","length":9}]}`
	if err := d.CreateDirection(tID, cfg); err != nil {
		t.Fatal(err)
	}
	var players []string
	for i := 0; i < n; i++ {
		id := string(rune('a'+i%26)) + string(rune('a'+i/26))
		players = append(players, `{"id":"`+id+`","name":"`+id+`"}`)
	}
	if err := d.StartDirection(tID, 7, "["+strings.Join(players, ",")+"]"); err != nil {
		t.Fatal(err)
	}
	return tID
}

func proposalJSON(t *testing.T, a tournoi.Action) string {
	t.Helper()
	b, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestConfirmProposal_OneGesture: confirming a proposal is one call, and the view that comes
// back is already replayed — the panel never has to ask twice.
func TestConfirmProposal_OneGesture(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 24)

	v, err := d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	var first tournoi.Action
	for _, a := range v.Proposals {
		if a.Kind == tournoi.ActStartMatch {
			first = a
			break
		}
	}
	if first.Kind == "" {
		t.Fatal("no match proposed")
	}
	before := v.EventCount

	after, err := d.ConfirmProposal(tID, proposalJSON(t, first))
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if after.EventCount != before+1 {
		t.Errorf("%d events after confirming, want %d", after.EventCount, before+1)
	}
	if len(after.Running) != 1 {
		t.Errorf("%d matches running, want 1", len(after.Running))
	}
	if after.Running[0].A != first.A || after.Running[0].B != first.B {
		t.Errorf("the match launched is not the one proposed: %s-%s", after.Running[0].A, after.Running[0].B)
	}
}

// TestConfirmAll_WholeQueue: the whole queue goes in one gesture, whatever its length. That is
// the two-click budget the UX document holds this to.
func TestConfirmAll_WholeQueue(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 24)

	v, err := d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	want := 0
	for _, a := range v.Proposals {
		if a.Kind != tournoi.ActWait {
			want++
		}
	}
	if want < 2 {
		t.Fatalf("expected several proposals to confirm, got %d", want)
	}

	after, err := d.ConfirmAllProposals(tID)
	if err != nil {
		t.Fatalf("confirm all: %v", err)
	}
	if len(after.Running) != want {
		t.Errorf("%d matches running after confirming everything, want %d", len(after.Running), want)
	}
	// Every table handed out is distinct: two matches never share one.
	seen := map[int]bool{}
	for _, m := range after.Running {
		if m.Table == 0 {
			continue
		}
		if seen[m.Table] {
			t.Errorf("table %d handed to two matches", m.Table)
		}
		seen[m.Table] = true
	}
}

// TestIgnoredProposalComesBack: a proposal the director does not confirm is offered again,
// identical. Ignoring is not an event — it is simply not acting.
func TestIgnoredProposalComesBack(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 24)

	first, err := d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	second, err := d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Proposals) != len(second.Proposals) {
		t.Fatalf("the queue changed without a decision: %d then %d", len(first.Proposals), len(second.Proposals))
	}
	for i := range first.Proposals {
		if first.Proposals[i].A != second.Proposals[i].A || first.Proposals[i].B != second.Proposals[i].B {
			t.Errorf("proposal %d changed without a decision", i)
		}
	}
	if first.EventCount != second.EventCount {
		t.Error("looking at the queue must write nothing")
	}
}

// TestManualPairingAcceptedAndFlagged: the director pairs two free players the engine did not
// propose. It is accepted; only the impossible is refused.
func TestManualPairingAcceptedAndFlagged(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 24)

	free, err := d.FreeParticipants(tID)
	if err != nil {
		t.Fatal(err)
	}
	if len(free) < 4 {
		t.Fatalf("expected free players, got %d", len(free))
	}
	// The first and the LAST of the list: not a pairing the engine would make.
	a, b := free[0].ID, free[len(free)-1].ID
	v, err := d.StartMatchManually(tID, string(a), string(b), 5, 0)
	if err != nil {
		t.Fatalf("a manual pairing between two free players must be accepted: %v", err)
	}
	found := false
	for _, m := range v.Running {
		if (m.A == a && m.B == b) || (m.A == b && m.B == a) {
			found = true
			if m.Length != 5 {
				t.Errorf("the chosen length was not kept: %d", m.Length)
			}
		}
	}
	if !found {
		t.Error("the manual match is not running")
	}

	// The impossible is still refused: a player already at a table.
	if _, err := d.StartMatchManually(tID, string(a), string(free[1].ID), 7, 0); err == nil {
		t.Error("pairing a player who is already playing must be refused")
	}
	// And an unknown player.
	if _, err := d.StartMatchManually(tID, "personne", string(free[1].ID), 7, 0); err == nil {
		t.Error("pairing an unknown player must be refused")
	}
}

// TestFreeParticipantsShrinksAsMatchesStart: the waiting queue is derived, never stored.
func TestFreeParticipantsShrinksAsMatchesStart(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 24)

	before, err := d.FreeParticipants(tID)
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != 24 {
		t.Fatalf("%d free before any match, want 24", len(before))
	}
	v, err := d.ConfirmAllProposals(tID)
	if err != nil {
		t.Fatal(err)
	}
	after, err := d.FreeParticipants(tID)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != 24-2*len(v.Running) {
		t.Errorf("%d free after %d matches started, want %d", len(after), len(v.Running), 24-2*len(v.Running))
	}
}
