package database

import (
	"strings"
	"testing"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
)

// TestHistoryIsReadable: every decision is there, in order, with names rather than identifiers.
func TestHistoryIsReadable(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 24)
	m := runningMatch(t, d, tID)
	if _, err := d.EnterResult(tID, string(m.ID), string(m.A), 7, 3, "tombé au temps"); err != nil {
		t.Fatal(err)
	}

	h, err := d.History(tID, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(h) < 26 { // created + 24 entries + launches + one result
		t.Fatalf("%d entries, want the whole log", len(h))
	}
	for i, e := range h {
		if e.Seq != i {
			t.Fatalf("entry %d carries sequence %d: the history is the log, in order", i, e.Seq)
		}
		if e.Time == "" {
			t.Errorf("every decision is timestamped: %+v", e)
		}
	}
	// The result carries the remark and the names.
	found := false
	for _, e := range h {
		if e.Kind != string(tournoi.EvResult) {
			continue
		}
		found = true
		if e.Text != "tombé au temps" {
			t.Errorf("the remark must be readable in the history: %q", e.Text)
		}
		if e.WinnerName == "" || e.WinnerName == e.Winner {
			t.Errorf("the history shows names: %+v", e)
		}
		if !e.Correctable {
			t.Error("a result can be corrected from the history")
		}
	}
	if !found {
		t.Error("the result is in the history")
	}
}

// TestHistoryFiltersByPlayer: how a director answers "what happened to Hugo?" without reading
// the whole log.
func TestHistoryFiltersByPlayer(t *testing.T) {
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
	var name string
	for _, r := range rows {
		if r.ID == string(m.A) {
			name = r.Name
		}
	}
	if name == "" {
		t.Fatal("player not found")
	}

	h, err := d.History(tID, name, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(h) == 0 {
		t.Fatal("their entry, their match and its result all name them")
	}
	for _, e := range h {
		hit := strings.Contains(e.PlayerName, name) || strings.Contains(e.AName, name) ||
			strings.Contains(e.BName, name) || strings.Contains(e.WinnerName, name)
		if !hit {
			t.Errorf("entry %d does not name %s: %+v", e.Seq, name, e)
		}
	}

	// And by match.
	byMatch, err := d.History(tID, "", string(m.ID))
	if err != nil {
		t.Fatal(err)
	}
	if len(byMatch) < 2 { // launched, then its result
		t.Fatalf("%d entries for the match, want its launch and its result", len(byMatch))
	}
	for _, e := range byMatch {
		if e.MatchID != string(m.ID) {
			t.Errorf("filtered on a match, got %q", e.MatchID)
		}
	}
}

// TestNoteIsTheDirectorsOwnWords: the one place in the log where free text goes, and it is
// refused by nothing — including after the close, which is exactly when one gets written.
func TestNoteIsTheDirectorsOwnWords(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 24)

	if _, err := d.AddDirectionNote(tID, "Table 3 déplacée, bruit de la rue"); err != nil {
		t.Fatal(err)
	}
	h, err := d.History(tID, "", "")
	if err != nil {
		t.Fatal(err)
	}
	last := h[len(h)-1]
	if last.Kind != string(tournoi.EvNote) || !strings.Contains(last.Text, "bruit") {
		t.Fatalf("the note is the last entry, with the director's words: %+v", last)
	}

	playToTheEnd(t, d, tID)
	if _, err := d.CloseDirection(tID); err != nil {
		t.Fatal(err)
	}
	if _, err := d.AddDirectionNote(tID, "Prix remis le lendemain"); err != nil {
		t.Errorf("a note about a closed tournament is exactly when one is written: %v", err)
	}
}

// TestSinceLastGesture: what changed while the director was somewhere else. A director who also
// plays comes back between two of their own matches and needs that first, not the whole log.
func TestSinceLastGesture(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 24)

	v, err := d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	mark := v.EventCount - 1 // the last sequence number they saw

	m := runningMatch(t, d, tID)
	if _, err := d.EnterResult(tID, string(m.ID), string(m.A), 7, 3, ""); err != nil {
		t.Fatal(err)
	}

	since, err := d.SinceLastGesture(tID, mark)
	if err != nil {
		t.Fatal(err)
	}
	if len(since) == 0 {
		t.Fatal("matches were launched and one result entered")
	}
	for _, e := range since {
		if e.Seq <= mark {
			t.Errorf("entry %d is not after the mark %d", e.Seq, mark)
		}
	}
	// Nothing new means nothing to read.
	v, err = d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	none, err := d.SinceLastGesture(tID, v.EventCount-1)
	if err != nil {
		t.Fatal(err)
	}
	if len(none) != 0 {
		t.Errorf("%d entries after the last one", len(none))
	}
}
