package database

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

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
	// The view reads p.sections.length: no graph is an empty list, never null.
	if js, _ := json.Marshal(p); !strings.Contains(string(js), `"sections":[]`) {
		t.Errorf("a phase without a graph must send an empty list: %s", js)
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
	// Every later place names the two places whose winners meet there: the view draws its
	// lines from that, not from positions.
	keys := map[string]bool{}
	for _, m := range main.Matches {
		keys[m.Key] = true
	}
	for _, m := range main.Matches {
		if m.Round == 0 {
			continue
		}
		if len(m.Feeds) != 2 {
			t.Errorf("a place after the first round has two feeders: %+v", m)
		}
		for _, f := range m.Feeds {
			if !keys[f.Key] || f.Loser {
				t.Errorf("feeder %+v must be a winner of this draw", f)
			}
		}
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

// undrawnBrackets starts a Direction on cfg with n participants, without drawing.
func undrawnBrackets(t *testing.T, cfg string, n int) []BracketPhase {
	t.Helper()
	d := newTestDB(t)
	tID, err := d.CreateTournament("Squelette", "2026-09-12", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := d.CreateDirection(tID, cfg, 5); err != nil {
		t.Fatal(err)
	}
	var players []string
	for i := 0; i < n; i++ {
		id := "p" + strings.Repeat("x", i)
		players = append(players, `{"id":"`+id+`","name":"Joueur `+id+`"}`)
	}
	if err := d.EnterParticipants(tID, "["+strings.Join(players, ",")+"]"); err != nil {
		t.Fatal(err)
	}
	phases, err := d.Brackets(tID)
	if err != nil {
		t.Fatal(err)
	}
	return phases
}

func sectionsByKind(p BracketPhase) map[string]BracketSection {
	out := map[string]BracketSection{}
	for _, s := range p.Sections {
		out[s.Kind] = s
	}
	return out
}

// TestBracketSkeleton: an undrawn elimination phase shows the shape the engine will draw — with
// its consolation, last chance and grand final — and nobody in it.
func TestBracketSkeleton(t *testing.T) {
	cases := []struct {
		name    string
		cfg     string
		n       int
		matches map[string]int // section kind -> number of places
	}{
		{"simple", `{"name":"T","tables":{"count":8},"phases":[{"kind":"bracket","length":5}]}`, 8, map[string]int{"main": 7}},
		{"non-power-of-two", `{"name":"T","tables":{"count":8},"phases":[{"kind":"bracket","length":5}]}`, 6, map[string]int{"main": 7}},
		{"consolation", `{"name":"T","tables":{"count":8},"phases":[{"kind":"bracket","length":5,"consolation":true}]}`, 8, nil},
		{"last chance", `{"name":"T","tables":{"count":8},"phases":[{"kind":"bracket","length":5,"consolation":true,"last_chance":true}]}`, 16, nil},
		{"double elimination", `{"name":"T","tables":{"count":8},"phases":[{"kind":"bracket","length":5,"consolation":true,"reconciliation":true,"recharge":true}]}`, 8, map[string]int{"main": 7, "gf": 2}},
		{"lives bracket", `{"name":"T","tables":{"count":8},"phases":[{"kind":"lives_bracket","length":5}]}`, 8, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := undrawnBrackets(t, c.cfg, c.n)[0]
			if p.Drawn {
				t.Fatal("nothing is drawn yet")
			}
			if len(p.Sections) == 0 {
				t.Fatal("no skeleton")
			}
			for _, s := range p.Sections {
				for _, m := range s.Matches {
					if m.A != "" || m.B != "" || m.AName != "" || m.BName != "" || m.MatchID != "" || m.Done || m.Walkover || m.Winner != "" {
						t.Fatalf("a skeleton place holds nobody: %+v", m)
					}
				}
			}
			by := sectionsByKind(p)
			for kind, want := range c.matches {
				if got := len(by[kind].Matches); got != want {
					t.Errorf("%s: %d places, want %d", kind, got, want)
				}
			}
			if c.name == "consolation" && by["conso"].Name == "" {
				t.Error("the consolation is part of the skeleton")
			}
			if c.name == "last chance" && by["last"].Name == "" {
				t.Error("the last chance is part of the skeleton")
			}
		})
	}
}

// TestBracketSkeletonSize: the size comes from those who enter, not from everyone registered,
// and is not guessed when it cannot be known.
func TestBracketSkeletonSize(t *testing.T) {
	// A Swiss phase followed by a bracket: only the Swiss phase exists until it ends.
	phases := undrawnBrackets(t, `{"name":"T","tables":{"count":8},"phases":[{"kind":"swiss_lives","length":5,"lives":2},{"kind":"bracket","length":5,"entry":"top:8"}]}`, 24)
	if len(phases) != 1 || len(phases[0].Sections) != 0 {
		t.Fatalf("a Swiss phase has no skeleton: %+v", phases)
	}

	cfg := tournoi.PhaseConfig{Kind: tournoi.KindBracket, Length: 5, Entry: "top:8"}
	sim, _, err := tournoi.New(tournoi.Config{Name: "T", Phases: []tournoi.PhaseConfig{{Kind: tournoi.KindSwissLives, Length: 5, Lives: 2}, cfg}}, 1, time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 24; i++ {
		id := tournoi.PlayerID("q" + strings.Repeat("x", i))
		if err := sim.Apply(tournoi.Event{Version: tournoi.JournalVersion, Kind: tournoi.EvPlayerAdded, Player: &tournoi.Player{ID: id, Name: string(id)}}); err != nil {
			t.Fatal(err)
		}
	}
	secs, ok := bracketSkeleton(sim, &tournoi.PhaseState{Index: 1, Cfg: cfg})
	if !ok || len(secs) != 1 || len(secs[0].Matches) != 7 {
		t.Fatalf("top:8 of 24 is a bracket of 8 (7 places): ok=%v %+v", ok, secs)
	}
	cfg.Entry = "survivors"
	if _, ok := bracketSkeleton(sim, &tournoi.PhaseState{Index: 1, Cfg: cfg}); ok {
		t.Error("survivors of an unfinished Swiss phase: size unknown, no skeleton")
	}
	cfg.Kind, cfg.Entry = tournoi.KindLivesBracket, "all"
	if _, ok := bracketSkeleton(sim, &tournoi.PhaseState{Index: 1, Cfg: cfg}); ok {
		t.Error("a lives bracket after another phase: lives differ, no skeleton")
	}
}
