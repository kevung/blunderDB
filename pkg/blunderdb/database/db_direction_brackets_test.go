package database

import (
	"strings"
	"testing"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
)

// TestBracketsOfASwissPhase: a Swiss phase has no graph before its switch. What a director
// reads there is the lives board — who has how many lives left, and whom they have met.
func TestBracketsOfASwissPhase(t *testing.T) {
	d := newTestDB(t)
	tID := startedDirection(t, d, 24)

	phases, err := d.Brackets(tID)
	if err != nil {
		t.Fatal(err)
	}
	if len(phases) != 1 {
		t.Fatalf("%d phases, want 1 (the next one has not been entered)", len(phases))
	}
	p := phases[0]
	if p.Kind != tournoi.KindSwissLives || !p.Current {
		t.Fatalf("first phase: %+v", p)
	}
	if len(p.Sections) != 0 {
		t.Error("a Swiss phase has no graph")
	}
	if len(p.Lives) != 24 {
		t.Fatalf("%d lives rows, want 24", len(p.Lives))
	}
	for _, r := range p.Lives {
		if r.Name == "" || r.Name == r.ID {
			t.Errorf("the lives board shows names, not identifiers: %+v", r)
		}
		if r.Lives != 2 {
			t.Errorf("%s starts with 2 lives, has %d", r.Name, r.Lives)
		}
	}

	// Play a round: the loser drops a life, and both have met someone.
	m := runningMatch(t, d, tID)
	if _, err := d.EnterResult(tID, string(m.ID), string(m.A), 7, 3, ""); err != nil {
		t.Fatal(err)
	}
	phases, err = d.Brackets(tID)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range phases[0].Lives {
		if r.ID == string(m.B) {
			if r.Lives != 1 || r.Losses != 1 {
				t.Errorf("the loser drops a life: %+v", r)
			}
			if len(r.Opponents) != 1 {
				t.Errorf("they have met someone: %+v", r)
			}
		}
	}
}

// TestBracketsOfABracketPhase: a drawn bracket comes back as sections of matches, each on its
// display row, with the engine's label untranslated.
func TestBracketsOfABracketPhase(t *testing.T) {
	d := newTestDB(t)
	tID, err := d.CreateTournament("Tableau", "2026-09-12", "")
	if err != nil {
		t.Fatal(err)
	}
	cfg := `{"name":"Tableau","tables":{"count":8},"phases":[{"kind":"bracket","length":5,"consolation":true}]}`
	if err := d.CreateDirection(tID, cfg, 5); err != nil {
		t.Fatal(err)
	}
	var players []string
	for i := 0; i < 8; i++ {
		id := string(rune('a' + i))
		players = append(players, `{"id":"`+id+`","name":"Joueur `+id+`"}`)
	}
	if err := d.EnterParticipants(tID, "["+strings.Join(players, ",")+"]"); err != nil {
		t.Fatal(err)
	}

	// Before the draw there is no graph, and the view must say so rather than show nothing.
	phases, err := d.Brackets(tID)
	if err != nil {
		t.Fatal(err)
	}
	if phases[0].Drawn {
		t.Error("nothing is drawn yet")
	}

	if _, err := d.ConfirmAllProposals(tID); err != nil { // the draw
		t.Fatal(err)
	}
	phases, err = d.Brackets(tID)
	if err != nil {
		t.Fatal(err)
	}
	p := phases[0]
	if !p.Drawn {
		t.Fatal("the bracket is drawn")
	}
	if len(p.Sections) < 2 {
		t.Fatalf("a main draw and its consolation: %d section(s)", len(p.Sections))
	}
	var main BracketSection
	for _, s := range p.Sections {
		if s.Kind == "main" {
			main = s
		}
	}
	if main.Name == "" {
		t.Fatal("no main draw")
	}
	if main.Rounds != 3 { // eight players: quarters, semis, final
		t.Errorf("%d rounds, want 3", main.Rounds)
	}
	rows := map[int]int{}
	for _, m := range main.Matches {
		rows[m.Round]++
		if m.Label.Kind == "" {
			t.Errorf("every place carries the engine's label: %+v", m)
		}
	}
	if rows[0] != 4 || rows[1] != 2 || rows[2] != 1 {
		t.Errorf("rows should be 4/2/1, got %v", rows)
	}
	// The first round knows its players by name; the later rows do not yet.
	for _, m := range main.Matches {
		if m.Round != 0 {
			continue
		}
		if m.AName == "" || m.BName == "" {
			t.Errorf("first-round places are filled: %+v", m)
		}
	}
}

// TestBracketFlagsTheMatchTheEngineComplainsAbout: a warning must be visible ON the bracket,
// not only in a list far from it.
func TestBracketFlagsTheMatchTheEngineComplainsAbout(t *testing.T) {
	d := newTestDB(t)
	tID, err := d.CreateTournament("Tableau", "2026-09-12", "")
	if err != nil {
		t.Fatal(err)
	}
	cfg := `{"name":"Tableau","tables":{"count":8},"phases":[{"kind":"bracket","length":5}]}`
	if err := d.CreateDirection(tID, cfg, 5); err != nil {
		t.Fatal(err)
	}
	var players []string
	for i := 0; i < 8; i++ {
		id := string(rune('a' + i))
		players = append(players, `{"id":"`+id+`","name":"Joueur `+id+`"}`)
	}
	if err := d.EnterParticipants(tID, "["+strings.Join(players, ",")+"]"); err != nil {
		t.Fatal(err)
	}
	if _, err := d.ConfirmAllProposals(tID); err != nil { // draw
		t.Fatal(err)
	}
	v, err := d.ConfirmAllProposals(tID) // first round
	if err != nil {
		t.Fatal(err)
	}
	var first []string
	for _, m := range v.Running {
		first = append(first, string(m.ID))
		if _, err := d.EnterResult(tID, string(m.ID), string(m.A), 5, 2, ""); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := d.ConfirmAllProposals(tID); err != nil { // semis
		t.Fatal(err)
	}
	// Correcting underneath makes a semi-final wrong.
	played, err := d.FinishedMatches(tID, 50)
	if err != nil {
		t.Fatal(err)
	}
	var target TableCell
	for _, m := range played {
		if m.MatchID == first[0] {
			target = m
		}
	}
	if target.MatchID == "" {
		t.Skip("no first-round match to contradict")
	}
	after, err := d.CorrectResult(tID, target.MatchID, target.B, 5, 3, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Warnings) == 0 {
		t.Skip("the bracket did not become inconsistent")
	}
	phases, err := d.Brackets(tID)
	if err != nil {
		t.Fatal(err)
	}
	flagged := 0
	for _, p := range phases {
		for _, s := range p.Sections {
			for _, m := range s.Matches {
				if m.Flagged {
					flagged++
				}
			}
		}
	}
	if flagged == 0 {
		t.Error("the match the engine complains about must be flagged on the bracket itself")
	}
}
