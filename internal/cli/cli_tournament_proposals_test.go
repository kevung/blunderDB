package cli

import (
	"strings"
	"testing"
)

// `tournament proposals` and `tournament confirm` are the panel's queue without the panel: a
// repechage is shown, named, and confirmed like any proposal.
func TestCLI_TournamentConfirmsARepechage(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	tID, err := cli.db.CreateTournament("Open A", "2026-10-04", "Lyon")
	if err != nil {
		t.Fatal(err)
	}
	cfg := `{"name":"Open A","tables":{"count":4},"phases":[
		{"kind":"round_robin","length":5,"group_size":4,"qualifiers":2,"entry":"survivors"},
		{"kind":"bracket","length":5,"entry":"survivors"}]}`
	if err := cli.db.CreateDirection(tID, cfg, 7); err != nil {
		t.Fatal(err)
	}
	if err := cli.db.EnterParticipants(tID, `[{"id":"a","name":"Joueur a"},{"id":"b","name":"Joueur b"},`+
		`{"id":"c","name":"Joueur c"},{"id":"d","name":"Joueur d"}]`); err != nil {
		t.Fatal(err)
	}
	playRoundsCLI(t, cli, tID, 3)
	if _, err := cli.db.WithdrawParticipant(tID, "a", false); err != nil {
		t.Fatal(err)
	}

	out := captureStdout(t, func() {
		if err := cli.Run([]string{"tournament", "proposals", "--db", dbPath, "--id", itoa64(tID)}); err != nil {
			t.Fatalf("proposals: %v", err)
		}
	})
	if !strings.Contains(out, "1\trepechage\tRepêchage poule A\tJoueur c replaces Joueur a") {
		t.Fatalf("the queue does not offer the repechage first:\n%s", out)
	}

	out = captureStdout(t, func() {
		if err := cli.Run([]string{"tournament", "confirm", "--db", dbPath, "--id", itoa64(tID), "--n", "1"}); err != nil {
			t.Fatalf("confirm: %v", err)
		}
	})
	if strings.Contains(out, "repechage") || !strings.Contains(out, "next_phase") {
		t.Fatalf("after the repechage the queue reads:\n%s", out)
	}
	v, err := cli.db.ConfirmAllProposals(tID)
	if err != nil {
		t.Fatal(err)
	}
	if v.Phase != 1 {
		t.Fatalf("still at phase %d", v.Phase)
	}
	for _, r := range v.Ranking {
		if r.Player == "a" && r.Note.Kind != "withdrawn" {
			t.Errorf("the withdrawn reads %+v", r)
		}
	}
}
