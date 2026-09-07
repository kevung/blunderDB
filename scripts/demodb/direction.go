package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
)

// A directed tournament in the demonstration database (issue #397, ux.md §4.2).
//
// The entry cost is a first-rank constraint of this whole piece of work, and nothing reduces it
// like SEEING A ROOM RUNNING before directing one's own. So the demo carries a Swiss of
// thirty-two, on the edge of switching to the bracket: matches played, matches running on
// tables, proposals waiting, and standings already worth reading.
//
// Everything here is generated, never hand-made, and the names are fictional — the
// demonstration database names nobody real (issue #162, and `demo_test.go` greps for the people
// the fixtures name).
//
// ## The one thing that ages
//
// A running match's elapsed time is counted against the reader's clock, so a demo opened months
// after it was built shows tables that have been playing for months. The alternative was to
// leave no match running at all, which would take away one of the four things this tournament
// exists to show. The file is rebuilt at every schema bump, and the trade is deliberate.

const (
	demoDirectionName     = "Open de Saint-Ludovic 2026"
	demoDirectionDate     = "2026-05-16"
	demoDirectionLocation = "Saint-Ludovic"
)

// demoEntrants are the thirty-two fictional players of the demonstration tournament: a name, a
// club, and an entry rating. Nobody real, and nobody named in the fixtures.
var demoEntrants = []tournoi.Player{
	{Name: "Ada Fairweather", Club: "Saint-Ludovic", Rating: 8.5},
	{Name: "Sol Marchetti", Club: "Saint-Ludovic", Rating: 9.2},
	{Name: "Iris Okonkwo", Club: "Val-Perrin", Rating: 7.8},
	{Name: "Bram Vesterholt", Club: "Val-Perrin", Rating: 10.4},
	{Name: "Noor Delacroix", Club: "Saint-Ludovic", Rating: 6.9},
	{Name: "Tomás Belmonte", Club: "Rive-Haute", Rating: 11.7},
	{Name: "Élise Wenzel", Club: "Rive-Haute", Rating: 5.4},
	{Name: "Kai Lindqvist", Club: "Val-Perrin", Rating: 12.1},
	{Name: "Mira Sandoval", Club: "Saint-Ludovic", Rating: 8.0},
	{Name: "Otto Brennecke", Club: "Rive-Haute", Rating: 9.9},
	{Name: "Livia Contarini", Club: "Val-Perrin", Rating: 7.1},
	{Name: "Anders Holmgren", Club: "Rive-Haute", Rating: 13.3},
	{Name: "Zoé Ferrandis", Club: "Saint-Ludovic", Rating: 6.2},
	{Name: "Rui Nakagawa", Club: "Val-Perrin", Rating: 10.8},
	{Name: "Hedda Silvermane", Club: "Rive-Haute", Rating: 8.8},
	{Name: "Casimir Duvalier", Club: "Saint-Ludovic", Rating: 14.0},
	{Name: "Nadia Berkoutova", Club: "Val-Perrin", Rating: 7.5},
	{Name: "Émile Vantongeren", Club: "Rive-Haute", Rating: 9.6},
	{Name: "Selma Aaltonen", Club: "Saint-Ludovic", Rating: 11.2},
	{Name: "Gaspard Roussillon", Club: "Val-Perrin", Rating: 6.6},
	{Name: "Wren Ashcombe", Club: "Rive-Haute", Rating: 12.8},
	{Name: "Ilya Zoranić", Club: "Saint-Ludovic", Rating: 8.3},
	{Name: "Beatriz Alcántara", Club: "Val-Perrin", Rating: 10.1},
	{Name: "Théo Marchandeau", Club: "Rive-Haute", Rating: 5.9},
	{Name: "Yuki Tamblyn", Club: "Saint-Ludovic", Rating: 13.7},
	{Name: "Roos van Herpen", Club: "Val-Perrin", Rating: 9.0},
	{Name: "Amara Sissoko", Club: "Rive-Haute", Rating: 7.3},
	{Name: "Lorcan Whitbourne", Club: "Saint-Ludovic", Rating: 11.9},
	{Name: "Petra Kalnina", Club: "Val-Perrin", Rating: 6.4},
	{Name: "Milo Achterberg", Club: "Rive-Haute", Rating: 10.6},
	{Name: "Solange Ferreira", Club: "Saint-Ludovic", Rating: 8.7},
	{Name: "Viggo Ranheim", Club: "Val-Perrin", Rating: 12.4},
}

// demoDirectionConfig is a club Sunday: a Swiss with two lives that switches to a bracket of
// sixteen, matches to seven points, twelve tables.
const demoDirectionConfig = `{
	"name": "Open de Saint-Ludovic 2026",
	"min_per_point": 8,
	"tables": {"count": 12},
	"prizes": {"entry_fee": 20, "retention": {"percent": 10},
	           "sections": {"all": {"percents": [50, 30, 20]}}},
	"phases": [
		{"kind": "swiss_lives", "length": 7, "lives": 2, "mode": "continuous", "target": 16},
		{"kind": "lives_bracket", "length": 9, "final_length": 11}
	]
}`

// buildDirection directs the demonstration tournament up to the edge of the switch.
//
// `seed` is the engine's draw seed, so the same build gives the same pairings; `finished` is how
// many matches are played out, and what is left running is what the tables show.
func buildDemoDirection(d *database.Database, now time.Time, seed int64) error {
	_ = now // le journal porte l'heure de construction ; voir la note d'en-tête.
	tournamentID, err := buildDirection(d, seed)
	if err != nil {
		return err
	}
	return verifyDirection(d, tournamentID)
}

func buildDirection(d *database.Database, seed int64) (int64, error) {
	tournamentID, err := d.CreateTournament(demoDirectionName, demoDirectionDate, demoDirectionLocation)
	if err != nil {
		return 0, fmt.Errorf("creating the demo tournament: %w", err)
	}
	if err := d.UpdateTournamentComment(tournamentID,
		"Tournoi dirigé de démonstration : suisse à deux vies de trente-deux joueurs, "+
			"au bord de la bascule vers le tableau. Ouvrez-le pour voir une salle qui tourne."); err != nil {
		return 0, fmt.Errorf("commenting the demo tournament: %w", err)
	}
	if err := d.CreateDirection(tournamentID, demoDirectionConfig, seed); err != nil {
		return 0, fmt.Errorf("directing the demo tournament: %w", err)
	}
	players, err := entrantsJSON()
	if err != nil {
		return 0, err
	}
	if err := d.EnterParticipants(tournamentID, players); err != nil {
		return 0, fmt.Errorf("entering the demo participants: %w", err)
	}
	if err := playDemoRounds(d, tournamentID, seed); err != nil {
		return 0, err
	}
	return tournamentID, nil
}

// entrantsJSON renders the entrants the way the panel's "take last time's entrants" does.
func entrantsJSON() (string, error) {
	blob, err := json.Marshal(demoEntrants)
	if err != nil {
		return "", fmt.Errorf("the demo entrants: %w", err)
	}
	return string(blob), nil
}

// playDemoRounds plays three rounds and leaves the fourth half-played: some tables busy, some
// proposals waiting, and a standings table that already says something.
//
// The winner of each match is drawn from a seeded generator rather than picked by rating: a demo
// where the favourite always wins shows a Swiss that never mixes, and the lives table is the
// part of this screen worth looking at.
func playDemoRounds(d *database.Database, tournamentID int64, seed int64) error {
	rng := rand.New(rand.NewSource(seed))
	const rounds = 3
	for r := 0; r < rounds; r++ {
		if err := launchDemoRound(d, tournamentID); err != nil {
			return err
		}
		v, err := d.GetDirection(tournamentID)
		if err != nil {
			return err
		}
		for _, m := range v.Running {
			winner := m.A
			if rng.Intn(2) == 1 {
				winner = m.B
			}
			// A score is free, and a director who types one types the loser's: seven to
			// three, seven to five. Never the winner's, which is the match length.
			loser := 1 + rng.Intn(m.Length-1)
			if _, err := d.EnterResult(tournamentID, string(m.ID), string(winner), m.Length, loser, ""); err != nil {
				return fmt.Errorf("entering a demo result: %w", err)
			}
		}
	}
	// The fourth round: launched, and left running. This is what the tables show.
	return launchDemoRound(d, tournamentID)
}

// launchDemoRound confirms what the engine proposes. The draw comes before the matches, so the
// gesture may have to be repeated — exactly as it is in the panel.
func launchDemoRound(d *database.Database, tournamentID int64) error {
	for i := 0; i < 4; i++ {
		v, err := d.ConfirmAllProposals(tournamentID)
		if err != nil {
			return fmt.Errorf("launching a demo round: %w", err)
		}
		if len(v.Running) > 0 {
			return nil
		}
	}
	return fmt.Errorf("the demo tournament launched no match")
}

// verifyDirection replays the demonstration Direction and refuses a warning: a demo that opens
// on a complaint teaches the wrong thing.
func verifyDirection(d *database.Database, tournamentID int64) error {
	v, err := d.GetDirection(tournamentID)
	if err != nil {
		return err
	}
	if len(v.Warnings) > 0 {
		return fmt.Errorf("the demo direction replays with %d warning(s): %+v", len(v.Warnings), v.Warnings)
	}
	if _, err := direction.Open(context.Background(), d.DirectionStore(), tournamentID); err != nil {
		return fmt.Errorf("the demo direction does not reopen: %w", err)
	}
	return nil
}
