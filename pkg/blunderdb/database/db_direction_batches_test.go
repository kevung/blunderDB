package database

import (
	"strings"
	"testing"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
)

// Micro-rounds and breaks (issue #388): the two rules of TIME the engine brings.
//
// Both are read against the WALL CLOCK and against nothing else — a micro-round's deadline falls
// while nobody writes anything, and a break's warning depends on when the match would end. So
// what these tests hold is not the rules (they are the engine's, and it tests them) but the
// thing blunderDB is responsible for: asking the engine at the right instant, and never letting
// a rule of time launch anything by itself.

// batchDirection prepares a tournament whose pairings go out in micro-rounds.
func batchDirection(t *testing.T, d *Database, n, minutes int) int64 {
	t.Helper()
	tID, err := d.CreateTournament("Open de Lyon", "2026-09-12", "Lyon")
	if err != nil {
		t.Fatal(err)
	}
	cfg := `{"name":"Open de Lyon","tables":{"count":16},"phases":[
		{"kind":"swiss_lives","length":7,"lives":2,"mode":"continuous","batch_minutes":` +
		itoa(minutes) + `}]}`
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

// With an interval set, the free players wait for the deadline and the queue says when.
func TestDirectionBatches_TheQueueCarriesItsDeadline(t *testing.T) {
	d := newTestDB(t)
	tID := batchDirection(t, d, 16, 20)
	// The first batch does not wait: with no match launched there is no previous batch.
	if _, err := d.ConfirmAllProposals(tID); err != nil {
		t.Fatal(err)
	}
	v, err := d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	if len(v.Running) == 0 {
		t.Fatal("the first batch should launch without waiting")
	}
	// Finish one match: the two players are free, and now they wait for the next batch.
	m := v.Running[0]
	if _, err := d.EnterResult(tID, string(m.ID), string(m.A), 0, 0, ""); err != nil {
		t.Fatal(err)
	}
	v, err = d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	var wait *tournoi.Action
	for i := range v.Proposals {
		if v.Proposals[i].Kind == tournoi.ActWait {
			wait = &v.Proposals[i]
		}
	}
	if wait == nil {
		t.Fatal("no wait proposal while the batch deadline has not come")
	}
	if wait.Reason != tournoi.ReasonWaitingBatch {
		t.Errorf("the queue waits for %q rather than for the batch", wait.Reason)
	}
	if wait.Until.IsZero() {
		t.Error("the wait carries no deadline, so the view can show no countdown")
	}
	if !wait.Until.After(time.Now()) {
		t.Errorf("the deadline %v is already past", wait.Until)
	}
}

// Without an interval, the behaviour is exactly lot 1's: the director asks for pairings when
// they want, and gets them.
func TestDirectionBatches_NoIntervalIsTheOldBehaviour(t *testing.T) {
	d := newTestDB(t)
	tID := batchDirection(t, d, 16, 0)
	if _, err := d.ConfirmAllProposals(tID); err != nil {
		t.Fatal(err)
	}
	v, err := d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	m := v.Running[0]
	if _, err := d.EnterResult(tID, string(m.ID), string(m.A), 0, 0, ""); err != nil {
		t.Fatal(err)
	}
	v, err = d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range v.Proposals {
		if a.Kind == tournoi.ActWait && a.Reason == tournoi.ReasonWaitingBatch {
			t.Fatal("a tournament with no interval should never wait for a batch")
		}
	}
}

// A proposal whose match would run into a break is MARKED and stays launchable. Nothing is
// blocked: the director knows things the engine does not.
func TestDirectionBatches_ABreakMarksAndBlocksNothing(t *testing.T) {
	d := newTestDB(t)
	tID := preparedDirection(t, d)
	// A break that starts in ten minutes, while a 7-point match is expected to take about an
	// hour: everything proposed now runs into it.
	start := time.Now().Add(10 * time.Minute).UTC().Format(time.RFC3339)
	end := time.Now().Add(90 * time.Minute).UTC().Format(time.RFC3339)
	cfg := `{"name":"Open de Lyon","tables":{"count":8},
		"breaks":[{"start":"` + start + `","end":"` + end + `"}],
		"phases":[{"kind":"swiss_lives","length":7,"lives":2,"mode":"continuous"}]}`
	if err := d.SetDirectionConfig(tID, cfg); err != nil {
		t.Fatal(err)
	}
	if err := d.EnterParticipants(tID, `[{"id":"aa","name":"Joueur aa"},{"id":"bb","name":"Joueur bb"}]`); err != nil {
		t.Fatal(err)
	}
	v, err := d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	marked := 0
	for _, a := range v.Proposals {
		if a.Kind == tournoi.ActStartMatch && a.Warn == tournoi.WarnEndsInBreak {
			marked++
		}
	}
	if marked == 0 {
		t.Fatal("no proposal is marked as running into the break")
	}
	// And it launches all the same: the warning is a remark, not a refusal.
	before, err := d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.ConfirmAllProposals(tID); err != nil {
		t.Fatalf("a marked proposal refused to launch: %v", err)
	}
	after, err := d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Running) <= len(before.Running) {
		t.Error("nothing was launched despite the warning being only a warning")
	}
}

// Nothing launches by itself. The deadline passing turns a wait into proposals, and there it
// stops: the director stays the only one who decides.
func TestDirectionBatches_TheDeadlineProposesAndLaunchesNothing(t *testing.T) {
	d := newTestDB(t)
	tID := batchDirection(t, d, 16, 20)
	if _, err := d.ConfirmAllProposals(tID); err != nil {
		t.Fatal(err)
	}
	v, err := d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	launched := len(v.Running)
	m := v.Running[0]
	if _, err := d.EnterResult(tID, string(m.ID), string(m.A), 0, 0, ""); err != nil {
		t.Fatal(err)
	}
	// Two refreshes, nothing else: the running matches are exactly those the director launched.
	for i := 0; i < 2; i++ {
		v, err = d.GetDirection(tID)
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(v.Running) != launched-1 {
		t.Errorf("%d matches running, expected the %d launched minus the one finished",
			len(v.Running), launched)
	}
}
