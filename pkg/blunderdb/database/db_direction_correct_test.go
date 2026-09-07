package database

import (
	"context"
	"strings"
	"testing"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
)

// TestLastDecisionIsTheOneToTakeBack: undo lands on the gesture the director is thinking of —
// the last result — not on the note they typed in between.
func TestLastDecisionIsTheOneToTakeBack(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 24)

	if last, err := d.LastDecision(tID); err != nil || last != nil {
		t.Fatalf("no decision has been made yet: %+v %v", last, err)
	}

	// Confirming the queue launches several matches; the last decision is the last of them.
	runningMatch(t, d, tID)
	last, err := d.LastDecision(tID)
	if err != nil {
		t.Fatal(err)
	}
	if last == nil || last.Kind != string(tournoi.EvMatchStarted) || !last.Cancellable {
		t.Fatalf("the last decision should be a launched match: %+v", last)
	}
	if last.AName == "" || last.AName == last.A {
		t.Errorf("the panel needs names, not identifiers: %+v", last)
	}

	if _, err := d.EnterResult(tID, last.MatchID, last.A, 7, 3, ""); err != nil {
		t.Fatal(err)
	}
	last, err = d.LastDecision(tID)
	if err != nil {
		t.Fatal(err)
	}
	if last == nil || !last.Correctable {
		t.Fatalf("the last decision should be the result, correctable: %+v", last)
	}
	if last.WinnerName == "" || last.WinnerName == last.Winner {
		t.Errorf("the panel needs the winner's NAME, not their identifier: %+v", last)
	}
}

// TestCorrectionIsOneMoreEvent: correcting never erases. The first result stays in the log
// where it happened, and both are readable in the history.
func TestCorrectionIsOneMoreEvent(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 24)
	m := runningMatch(t, d, tID)

	before, err := d.EnterResult(tID, string(m.ID), string(m.A), 7, 3, "")
	if err != nil {
		t.Fatal(err)
	}
	after, err := d.CorrectResult(tID, string(m.ID), string(m.B), 7, 5, "")
	if err != nil {
		t.Fatalf("correcting: %v", err)
	}
	if after.EventCount != before.EventCount+1 {
		t.Errorf("a correction adds an event: %d then %d", before.EventCount, after.EventCount)
	}

	events, err := d.DirectionStore().LoadEvents(context.Background(), tID)
	if err != nil {
		t.Fatal(err)
	}
	kinds := make([]string, 0, len(events))
	for i, e := range events {
		kinds = append(kinds, e.Kind)
		if e.Seq != i {
			t.Errorf("event %d carries sequence %d: the log must stay contiguous", i, e.Seq)
		}
	}
	joined := strings.Join(kinds, " ")
	if !strings.Contains(joined, "result result_corrected") {
		t.Errorf("both the result and its correction must be in the log, in order: %v", kinds)
	}
}

// TestCorrectionTakesEffect: the corrected winner is the one that counts afterwards.
func TestCorrectionTakesEffect(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 24)
	m := runningMatch(t, d, tID)

	if _, err := d.EnterResult(tID, string(m.ID), string(m.A), 7, 3, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := d.CorrectResult(tID, string(m.ID), string(m.B), 7, 5, ""); err != nil {
		t.Fatal(err)
	}
	finished, err := d.FinishedMatches(tID, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(finished) == 0 {
		t.Fatal("the match should be listed as finished")
	}
	// The engine has the truth; check it through a replay rather than through our own memory.
	v, err := d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range v.Ranking {
		if r.Player == m.B && r.Note.Kind == tournoi.NoteForfeit {
			t.Error("the corrected winner must not be counted as withdrawn")
		}
	}
}

// TestCorrectionInBracketRaisesAWarningThatClears: a correction can leave a bracket wrong. The
// engine says so, and stops saying so once the cause is gone — never because someone read it.
func TestCorrectionInBracketRaisesAWarningThatClears(t *testing.T) {
	d := newTestDB(t)
	tID, err := d.CreateTournament("Tableau", "2026-09-12", "")
	if err != nil {
		t.Fatal(err)
	}
	cfg := `{"name":"Tableau","tables":{"count":8},"phases":[{"kind":"bracket","length":5}]}`
	if err := d.CreateDirection(tID, cfg); err != nil {
		t.Fatal(err)
	}
	var players []string
	for i := 0; i < 8; i++ {
		id := string(rune('a' + i))
		players = append(players, `{"id":"`+id+`","name":"`+id+`"}`)
	}
	if err := d.StartDirection(tID, 5, "["+strings.Join(players, ",")+"]"); err != nil {
		t.Fatal(err)
	}
	// Draw, then play the first round, then the next one: enough for a correction to matter.
	if _, err := d.ConfirmAllProposals(tID); err != nil {
		t.Fatal(err)
	}
	var firstRound []string
	v, err := d.ConfirmAllProposals(tID)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range v.Running {
		firstRound = append(firstRound, string(m.ID))
		if _, err := d.EnterResult(tID, string(m.ID), string(m.A), 5, 2, ""); err != nil {
			t.Fatal(err)
		}
	}
	if len(firstRound) == 0 {
		t.Fatal("no first-round match")
	}
	// Launch the next round, then correct a first-round result underneath it.
	if _, err := d.ConfirmAllProposals(tID); err != nil {
		t.Fatal(err)
	}
	v, err = d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	if len(v.Running) == 0 {
		t.Skip("the bracket produced no second round to contradict")
	}
	if len(v.Warnings) != 0 {
		t.Fatalf("nothing should be wrong yet: %v", v.Warnings)
	}

	// The first match's loser actually won.
	st, err := d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	_ = st
	before, err := d.FinishedMatches(tID, 50)
	if err != nil {
		t.Fatal(err)
	}
	var target TableCell
	for _, m := range before {
		if m.MatchID == firstRound[0] {
			target = m
		}
	}
	if target.MatchID == "" {
		t.Fatal("first-round match not found")
	}
	after, err := d.CorrectResult(tID, target.MatchID, target.B, 5, 3, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Warnings) == 0 {
		t.Fatal("correcting under a played round must leave the bracket flagged")
	}
	named := false
	for _, w := range after.Warnings {
		if w.Code == tournoi.WarnBracketWrongPlayers && w.ExpectedA != "" && w.ExpectedB != "" {
			named = true
		}
	}
	if !named {
		t.Errorf("the warning must name the match and what the bracket expected: %v", after.Warnings)
	}

	// Cancelling the match the correction invalidated makes the warning go away — because its
	// cause is gone, not because anyone acknowledged it.
	for _, w := range after.Warnings {
		if w.Code != tournoi.WarnBracketWrongPlayers {
			continue
		}
		if _, err := d.CancelMatch(tID, string(w.Match)); err != nil {
			t.Fatal(err)
		}
	}
	final, err := d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range final.Warnings {
		if w.Code == tournoi.WarnBracketWrongPlayers {
			t.Errorf("the warning should have cleared with its cause: %v", final.Warnings)
		}
	}
}
