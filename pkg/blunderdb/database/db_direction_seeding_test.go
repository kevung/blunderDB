package database

import (
	"strings"
	"testing"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
)

// Seeding, optional and off by default (issue #394, fonctionnel.md §3.4).
//
// The engine's own study concludes "no protected seeds": that is the current culture of
// backgammon, where a full draw is what players expect, and the empty default is a DESIGN
// DECISION and not an oversight. Some organisers want them anyway. What is held here is that
// the option changes the draw when it is on, changes nothing when it is off, and replays to the
// same bracket either way.

// ratedBracket prepares a bracket of n players whose ratings are all different, so a seeded
// draw is fully determined by them.
func ratedBracket(t *testing.T, d *Database, n int, seeding string) int64 {
	t.Helper()
	tID, err := d.CreateTournament("Open de Lyon", "2026-09-12", "Lyon")
	if err != nil {
		t.Fatal(err)
	}
	seed := ""
	if seeding != "" {
		seed = `,"seeding":"` + seeding + `"`
	}
	cfg := `{"name":"Open de Lyon","tables":{"count":16},
		"phases":[{"kind":"bracket","length":5` + seed + `}]}`
	if err := d.CreateDirection(tID, cfg, 7); err != nil {
		t.Fatal(err)
	}
	var players []string
	for i := 0; i < n; i++ {
		id := string(rune('a' + i))
		// La cote décroît avec l'indice : « a » est le meilleur.
		rating := 20 - i
		players = append(players, `{"id":"`+id+`","name":"Joueur `+id+`","rating":`+itoa(rating)+`}`)
	}
	if err := d.EnterParticipants(tID, "["+strings.Join(players, ",")+"]"); err != nil {
		t.Fatal(err)
	}
	// Confirm the draw, and only the draw.
	v, err := d.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range v.Proposals {
		if a.Kind == tournoi.ActDraw {
			if _, err := d.ConfirmProposal(tID, proposalJSON(t, a)); err != nil {
				t.Fatal(err)
			}
			break
		}
	}
	return tID
}

// halfOf says which half of the bracket a place sits in, by its index among the first round's
// matches: the top half is the first half of the matches.
func halvesOfFirstRound(t *testing.T, d *Database, tID int64) map[string]int {
	t.Helper()
	phases, err := d.Brackets(tID)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]int{}
	for _, ph := range phases {
		for _, sec := range ph.Sections {
			var first []BracketMatch
			for _, m := range sec.Matches {
				if m.Round == 0 {
					first = append(first, m)
				}
			}
			for i, m := range first {
				half := 0
				if i >= len(first)/2 {
					half = 1
				}
				if m.A != "" {
					out[m.A] = half
				}
				if m.B != "" {
					out[m.B] = half
				}
			}
		}
	}
	return out
}

// Seeded, the best and the second are in opposite halves. That is the whole point of the
// classical placement, and the only thing an organiser who asks for seeds is asking for.
func TestSeeding_TheBestTwoAreInOppositeHalves(t *testing.T) {
	d := newTestDB(t)
	tID := ratedBracket(t, d, 16, tournoi.SeedingRating)

	halves := halvesOfFirstRound(t, d, tID)
	if len(halves) == 0 {
		t.Fatal("no first-round pairing to read")
	}
	if halves["a"] == halves["b"] {
		t.Errorf("the best two are both in half %d", halves["a"])
	}
	// And the best four are in four different quarters, which the same construction gives.
	quarters := map[int]int{}
	phases, err := d.Brackets(tID)
	if err != nil {
		t.Fatal(err)
	}
	for _, ph := range phases {
		for _, sec := range ph.Sections {
			var first []BracketMatch
			for _, m := range sec.Matches {
				if m.Round == 0 {
					first = append(first, m)
				}
			}
			for i, m := range first {
				q := i * 4 / len(first)
				for _, p := range []string{m.A, m.B} {
					if p == "a" || p == "b" || p == "c" || p == "d" {
						quarters[q]++
					}
				}
			}
		}
	}
	if len(quarters) != 4 {
		t.Errorf("the best four sit in %d quarters, expected 4: %v", len(quarters), quarters)
	}
}

// Off — the default — the draw is what it always was: the seed of the Direction decides it, and
// the ratings change nothing.
func TestSeeding_OffIsTheDrawOfBefore(t *testing.T) {
	d := newTestDB(t)
	plain := ratedBracket(t, d, 16, "")
	seeded := ratedBracket(t, d, 16, tournoi.SeedingRating)

	a := halvesOfFirstRound(t, d, plain)
	b := halvesOfFirstRound(t, d, seeded)
	if len(a) != len(b) {
		t.Fatalf("%d places drawn without seeds, %d with", len(a), len(b))
	}
	same := true
	for k, v := range a {
		if b[k] != v {
			same = false
		}
	}
	if same {
		t.Error("seeding changed nothing: with sixteen distinct ratings and the same seed, the two draws should differ")
	}
	// The draw with no seeding is reproducible from the Direction's seed alone.
	again := ratedBracket(t, d, 16, "")
	c := halvesOfFirstRound(t, d, again)
	for k, v := range a {
		if c[k] != v {
			t.Errorf("%s was in half %d and is now in half %d: the draw is not reproducible", k, v, c[k])
		}
	}
}

// Players with no rating are placed deterministically, and a replay gives the same bracket.
// A tournament where half the entrants have no rating is the ordinary case in a club.
func TestSeeding_UnratedPlayersAreDeterministic(t *testing.T) {
	d := newTestDB(t)
	build := func() int64 {
		tID, err := d.CreateTournament("Open de Lyon", "2026-09-12", "Lyon")
		if err != nil {
			t.Fatal(err)
		}
		cfg := `{"name":"Open de Lyon","tables":{"count":16},
			"phases":[{"kind":"bracket","length":5,"seeding":"rating"}]}`
		if err := d.CreateDirection(tID, cfg, 7); err != nil {
			t.Fatal(err)
		}
		var players []string
		for i := 0; i < 8; i++ {
			id := string(rune('a' + i))
			rating := "0"
			if i%2 == 0 {
				rating = itoa(20 - i)
			}
			players = append(players, `{"id":"`+id+`","name":"Joueur `+id+`","rating":`+rating+`}`)
		}
		if err := d.EnterParticipants(tID, "["+strings.Join(players, ",")+"]"); err != nil {
			t.Fatal(err)
		}
		v, err := d.GetDirection(tID)
		if err != nil {
			t.Fatal(err)
		}
		for _, a := range v.Proposals {
			if a.Kind == tournoi.ActDraw {
				if _, err := d.ConfirmProposal(tID, proposalJSON(t, a)); err != nil {
					t.Fatal(err)
				}
				break
			}
		}
		return tID
	}
	first := halvesOfFirstRound(t, d, build())
	second := halvesOfFirstRound(t, d, build())
	if len(first) == 0 {
		t.Fatal("no pairing drawn")
	}
	for k, v := range first {
		if second[k] != v {
			t.Errorf("%s was in half %d and is now in half %d: unrated players are not placed deterministically", k, v, second[k])
		}
	}
}
