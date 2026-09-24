package database

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tournoi "github.com/PileOfCells/backgammon-tournoi"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
)

func journalOf(t *testing.T, d *Database, tID int64) tournoi.Journal {
	t.Helper()
	dir, err := direction.Open(context.Background(), d.DirectionStore(), tID)
	if err != nil {
		t.Fatal(err)
	}
	return dir.Journal()
}

// S4, the Friday before the third Monday (#451, D7.2;
// tasks/nicomaque/simulation-2026-09/rapport/S4.md O6): two rounds played, the third proposed
// and not launched. Its sheet prints, dated by the director, and NOTHING is written to the log:
// launching the round to print it would have stamped it Friday.
func s4Friday(t *testing.T) (*Database, int64) {
	t.Helper()
	d := newTestDB(t)
	tID := directedAt(t, d, 25, `{"name":"Championnat du lundi","min_per_point":8,"tables":{"count":14},
		"phases":[{"kind":"swiss_lives","length":7,"lives":6,"mode":"rounds"}]}`)
	start := localAt(2026, 10, 5, 20, 0)
	// The hall closes at 21:00: the second round starts before, the third would start after.
	playUntil(t, d, tID, start, localAt(2026, 10, 5, 23, 0), hall{open: 20, close: 21}, 8)
	return d, tID
}

func proposedStarts(t *testing.T, d *Database, tID int64) []tournoi.Action {
	t.Helper()
	v, err := d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	var out []tournoi.Action
	for _, a := range v.Proposals {
		if a.Kind == tournoi.ActStartMatch {
			out = append(out, a)
		}
	}
	return out
}

func TestUpcomingSheetPrintsTheProposedRoundAndWritesNothing(t *testing.T) {
	d, tID := s4Friday(t)
	rounds, err := d.DirectionRounds(tID)
	if err != nil {
		t.Fatal(err)
	}
	if rounds != 2 {
		t.Fatalf("the scenario must stop after two rounds, got %d", rounds)
	}
	starts := proposedStarts(t, d, tID)
	if len(starts) < 10 {
		t.Fatalf("the third round must be proposed: %d start proposals", len(starts))
	}
	events := len(journalOf(t, d, tID))

	html, err := d.DirectionUpcomingSheetHTML(tID, "lundi 12/10, <20 h>")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "lundi 12/10, &lt;20 h&gt;") {
		t.Error("the sheet carries the date the director typed, escaped")
	}
	if !strings.Contains(html, " 3</h2>") {
		t.Errorf("the sheet is the third round's: %s", html[:min(len(html), 400)])
	}
	for _, a := range starts {
		for _, id := range []tournoi.PlayerID{a.A, a.B} {
			name := "Joueur " + strings.TrimPrefix(string(id), "p")
			if !strings.Contains(html, name) {
				t.Errorf("%s is proposed (%+v) and missing from the sheet:\n%s", name, a, html)
			}
		}
	}
	if !strings.Contains(html, "window.print()") {
		t.Error("the sheet opens the print dialog, like the sheet of a launched round")
	}

	path, err := d.WriteDirectionUpcomingSheet(tID, "lundi 12/10, 20 h")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) == "appariements.html" {
		t.Error("the announced sheet must not overwrite the sheet of the round being played")
	}
	defer os.Remove(path)
	if b, err := os.ReadFile(path); err != nil || !strings.Contains(string(b), "lundi 12/10, 20 h") {
		t.Errorf("written sheet: %v", err)
	}

	if got := len(journalOf(t, d, tID)); got != events {
		t.Errorf("printing an announced round wrote %d events", got-events)
	}
	if after, _ := d.DirectionRounds(tID); after != 2 {
		t.Errorf("nothing launched: still two rounds, got %d", after)
	}
	if again := proposedStarts(t, d, tID); len(again) != len(starts) {
		t.Errorf("the queue is untouched: %d proposals, then %d", len(starts), len(again))
	}
}

func TestUpcomingSheetWithNothingProposed(t *testing.T) {
	d := newTestDB(t)
	tID := directedAt(t, d, 4, `{"name":"Court","tables":{"count":4},"phases":[{"kind":"bracket","length":3}]}`)
	// Draw, then launch everything: nothing is left to announce.
	for i := 0; i < 3 && len(proposedStarts(t, d, tID)) > 0 || i == 0; i++ {
		if _, err := d.ConfirmAllProposals(tID); err != nil {
			t.Fatal(err)
		}
	}
	if got := proposedStarts(t, d, tID); len(got) != 0 {
		t.Fatalf("the semi-finals are running, nothing should be proposed: %v", got)
	}
	if _, err := d.DirectionUpcomingSheetHTML(tID, "demain"); err == nil {
		t.Error("with no proposed match there is no sheet to announce")
	}
}
