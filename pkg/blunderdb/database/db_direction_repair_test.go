package database

import (
	"strings"
	"testing"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
)

// Repairing a bracket knocked out of tune by a correction (issue #389).
//
// Correcting a bracket result long after the fact leaves the director in front of a wrong tree:
// the matches below it were played by the wrong people. The engine PROPOSES the repair — it
// never applies it. What is held here is that rule and its consequence: confirming nothing
// changes nothing, and confirming everything clears the warning.

// bracketDirection prepares a straight knock-out of n players.
func bracketDirection(t *testing.T, d *Database, n int) int64 {
	t.Helper()
	tID, err := d.CreateTournament("Open de Lyon", "2026-09-12", "Lyon")
	if err != nil {
		t.Fatal(err)
	}
	cfg := `{"name":"Open de Lyon","tables":{"count":16},"phases":[{"kind":"bracket","length":5}]}`
	if err := d.CreateDirection(tID, cfg, 7); err != nil {
		t.Fatal(err)
	}
	var players []string
	for i := 0; i < n; i++ {
		id := string(rune('a'+i%26)) + string(rune('a'+i/26))
		players = append(players, `{"id":"`+id+`","name":"Joueur `+id+`"}`)
	}
	if err := d.EnterParticipants(tID, "["+strings.Join(players, ",")+"]"); err != nil {
		t.Fatal(err)
	}
	return tID
}

// playOneRound launches everything launchable and enters a result for every running match. The
// lower identifier wins, so the run is deterministic.
func playOneRound(t *testing.T, d *Database, tID int64) int {
	t.Helper()
	// The draw comes before the matches, so "launch all" may have to be asked twice.
	var v *DirectionView
	for i := 0; i < 4; i++ {
		var err error
		if v, err = d.ConfirmAllProposals(tID); err != nil {
			t.Fatal(err)
		}
		if len(v.Running) > 0 {
			break
		}
	}
	played := 0
	for _, m := range v.Running {
		w := m.A
		if string(m.B) < string(m.A) {
			w = m.B
		}
		if _, err := d.EnterResult(tID, string(m.ID), string(w), m.Length, 0, ""); err != nil {
			t.Fatal(err)
		}
		played++
	}
	return played
}

// repairProposals are the cancellations the engine offers to put the tree back in tune.
// flipOldestResult corrects the oldest finished match in favour of the OTHER player, read from
// the log. Correcting in favour of whoever already won would change nothing, and the test would
// prove nothing.
func flipOldestResult(t *testing.T, d *Database, tID int64) TableCell {
	t.Helper()
	finished, err := d.FinishedMatches(tID, 40)
	if err != nil {
		t.Fatal(err)
	}
	if len(finished) == 0 {
		t.Fatal("no finished match to correct")
	}
	m := finished[len(finished)-1]
	entries, err := d.History(tID, "", m.MatchID)
	if err != nil {
		t.Fatal(err)
	}
	won := ""
	for _, e := range entries {
		if e.Winner != "" {
			won = e.Winner
		}
	}
	other := m.A
	if won == m.A {
		other = m.B
	}
	if _, err := d.CorrectResult(tID, m.MatchID, other, 0, 0, ""); err != nil {
		t.Fatal(err)
	}
	return m
}

func repairProposals(v *DirectionView) []tournoi.Action {
	var out []tournoi.Action
	for _, a := range v.Proposals {
		if a.Kind == tournoi.ActCancelMatch {
			out = append(out, a)
		}
	}
	return out
}

func TestDirectionRepair_ProposedAfterACorrection(t *testing.T) {
	d := newTestDB(t)
	tID := bracketDirection(t, d, 8)
	// Quarter-finals, then semi-finals: the semis are now built on the quarters' results.
	if n := playOneRound(t, d, tID); n != 4 {
		t.Fatalf("%d quarter-finals played, expected 4", n)
	}
	if n := playOneRound(t, d, tID); n != 2 {
		t.Fatalf("%d semi-finals played, expected 2", n)
	}

	before, err := d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	if len(repairProposals(before)) != 0 {
		t.Fatal("a tournament run normally proposes no repair")
	}

	// The oldest finished match is a quarter-final. Its result was wrong.
	flipOldestResult(t, d, tID)
	v, err := d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	if len(v.Warnings) == 0 {
		t.Fatal("correcting a quarter-final under a played semi-final raised no warning")
	}
	repairs := repairProposals(v)
	if len(repairs) == 0 {
		t.Fatal("no repair proposed for a tree knocked out of tune")
	}
	// A repair carries what it is about: the match, the players who did play it, and where.
	r := repairs[0]
	if r.Match == "" || r.A == "" || r.B == "" {
		t.Errorf("a repair proposal without its facts: %+v", r)
	}
}

// Confirming nothing changes nothing: no cancellation is applied on the director's behalf. They
// may well prefer to leave the bracket as it was played and note it by hand.
func TestDirectionRepair_ConfirmingNothingChangesNothing(t *testing.T) {
	d := newTestDB(t)
	tID := bracketDirection(t, d, 8)
	playOneRound(t, d, tID)
	playOneRound(t, d, tID)

	flipOldestResult(t, d, tID)
	first, err := d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	// Two refreshes and nothing else: the same warnings, the same repairs, nothing applied.
	second, err := d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Warnings) != len(first.Warnings) {
		t.Errorf("the warnings moved on their own: %d then %d", len(first.Warnings), len(second.Warnings))
	}
	if len(repairProposals(second)) != len(repairProposals(first)) {
		t.Error("the repair proposals moved on their own")
	}
	if second.EventCount != first.EventCount {
		t.Errorf("an event was written without the director confirming anything: %d then %d",
			first.EventCount, second.EventCount)
	}
}

// Confirming the whole repair makes the warning disappear.
func TestDirectionRepair_ConfirmingItAllClearsTheWarning(t *testing.T) {
	d := newTestDB(t)
	tID := bracketDirection(t, d, 8)
	playOneRound(t, d, tID)
	playOneRound(t, d, tID)

	flipOldestResult(t, d, tID)
	v, err := d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	if len(v.Warnings) == 0 || len(repairProposals(v)) == 0 {
		t.Fatal("the case is not the one being tested: no warning, or no repair proposed")
	}

	// The director confirms the cancellations, one after another, as the queue offers them.
	for step := 0; step < 10; step++ {
		v, err = d.GetDirection(tID)
		if err != nil {
			t.Fatal(err)
		}
		repairs := repairProposals(v)
		if len(repairs) == 0 {
			break
		}
		if _, err := d.ConfirmProposal(tID, proposalJSON(t, repairs[0])); err != nil {
			t.Fatalf("confirming a repair: %v", err)
		}
	}
	v, err = d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	if len(repairProposals(v)) != 0 {
		t.Fatal("repairs remain after confirming them all")
	}
	for _, w := range v.Warnings {
		if w.Code == "bracket_wrong_players" {
			t.Errorf("the warning survives its repair: %+v", w)
		}
	}
	// And the queue offers the right match: the places freed by the cancellations come back as
	// ordinary proposals, which is why they arrive at the NEXT call rather than being
	// manufactured by the repair.
	restart := 0
	for _, a := range v.Proposals {
		if a.Kind == tournoi.ActStartMatch {
			restart++
		}
	}
	if restart == 0 {
		t.Error("nothing is proposed to replay in place of the cancelled matches")
	}
}
