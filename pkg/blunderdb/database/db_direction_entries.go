package database

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
)

// The entries of a Direction (ADR-0047 §4, issue #369).
//
// A Participant is an entry in ONE Direction: a name, a club, an entry rating. It is not a
// Player and not a person. The only link to a Player is a name the director chose to spell the
// same, so that the Matches filling this Participant's Slots carry that Player's literal name.
// Choosing an existing Player at entry time fixes the spelling and pre-fills the rating from
// their PR; nothing is inferred afterwards, and there is no reusable "person" record — which is
// the identity CONTEXT.md has always refused.

// ParticipantRow is one line of the players view: the entry, plus what the replayed state knows
// about them. Everything derived here is recomputed at every call.
type ParticipantRow struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Club   string  `json:"club,omitempty"`
	Rating float64 `json:"rating,omitempty"`

	Wins   int `json:"wins"`
	Losses int `json:"losses"`
	Lives  int `json:"lives"`
	Byes   int `json:"byes"`
	// Opponents are the names already met, which is what a director checks before pairing
	// two people by hand.
	Opponents []string `json:"opponents,omitempty"`
	// State is a CODE — playing, free, withdrawn, leaving — rendered by the frontend.
	State string `json:"state"`
	Table int    `json:"table,omitempty"`
}

// EntrySuggestion is a Player of this database offered at entry time: the literal name their
// Matches carry, and the PR that pre-fills the entry rating.
type EntrySuggestion struct {
	Name    string  `json:"name"`
	PR      float64 `json:"pr"`
	Matches int     `json:"matches"`
}

// EntrySuggestions returns the Players of this database, most-played first, for the entry
// field's autocompletion. Choosing one is a decision of the director's, never an inference: it
// fixes the spelling so the Matches line up later.
func (d *Database) EntrySuggestions() ([]EntrySuggestion, error) {
	rows, err := d.GetPlayerTable(StatsFilter{})
	if err != nil {
		return nil, err
	}
	out := make([]EntrySuggestion, 0, len(rows))
	for _, r := range rows {
		out = append(out, EntrySuggestion{Name: r.Name, PR: r.PR, Matches: r.Matches})
	}
	return out, nil
}

// Participants lists the entries of a Direction with their current standing.
func (d *Database) Participants(tournamentID int64) ([]ParticipantRow, error) {
	dir, err := direction.Open(context.Background(), d.DirectionStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	st := dir.State()
	if st == nil {
		return nil, nil
	}
	busy := map[tournoi.PlayerID]*tournoi.Match{}
	for _, m := range st.Running() {
		busy[m.A], busy[m.B] = m, m
	}
	ph := st.Phases[st.Current]
	out := make([]ParticipantRow, 0, len(st.Order))
	for _, id := range st.Order {
		p := st.Players[id]
		if p == nil {
			continue
		}
		row := ParticipantRow{
			ID: string(id), Name: p.Name, Club: p.Club, Rating: p.Rating,
			Wins: ph.Wins[id], Losses: ph.Losses[id], Byes: ph.Byes[id],
			Lives: ph.Lives[id] - ph.Losses[id],
		}
		if row.Lives < 0 {
			row.Lives = 0
		}
		for _, o := range ph.Opponents[id] {
			row.Opponents = append(row.Opponents, playerNameIn(st, o))
		}
		switch {
		case st.Withdrawn[id]:
			row.State = "withdrawn"
		case st.WithdrawAfter[id]:
			// Leaving: no longer paired, but finishing the match they are at.
			row.State = "leaving"
		case busy[id] != nil:
			row.State = "playing"
			row.Table = busy[id].Table
		default:
			row.State = "free"
		}
		out = append(out, row)
	}
	return out, nil
}

// AddParticipant enters one player. It is an event, in preparation as afterwards: a late
// arrival is an entry like any other, and the engine decides where they come in.
func (d *Database) AddParticipant(tournamentID int64, name, club string, rating float64) (*DirectionView, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("direction: an entry needs a name")
	}
	ctx := context.Background()
	dir, err := direction.Open(ctx, d.DirectionStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	st := dir.State()
	if st == nil {
		return nil, direction.ErrNoDirection
	}
	taken := map[string]bool{}
	for id := range st.Players {
		taken[string(id)] = true
	}
	p := tournoi.Player{ID: tournoi.PlayerID(participantID(name, taken)), Name: name, Club: club, Rating: rating}
	if err := dir.Apply(ctx, tournoi.PlayerAddedEvent(p, time.Now())); err != nil {
		return nil, err
	}
	return d.GetDirection(tournamentID)
}

// UpdateParticipant corrects an entry — a mistyped name, a missing club, a wrong rating —
// WITHOUT changing its identifier. That matters: the identifier is what a Slot points at, so
// correcting a name must not undo a Match already attached to it.
func (d *Database) UpdateParticipant(tournamentID int64, id, name, club string, rating float64) (*DirectionView, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("direction: an entry needs a name")
	}
	ctx := context.Background()
	dir, err := direction.Open(ctx, d.DirectionStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	st := dir.State()
	if st == nil {
		return nil, direction.ErrNoDirection
	}
	if _, ok := st.Players[tournoi.PlayerID(id)]; !ok {
		return nil, fmt.Errorf("direction: no entry %q", id)
	}
	// Re-adding under the same identifier is how the engine records a correction: the entry
	// keeps its place, its matches and its Slots.
	p := tournoi.Player{ID: tournoi.PlayerID(id), Name: name, Club: club, Rating: rating}
	if err := dir.Apply(ctx, tournoi.PlayerAddedEvent(p, time.Now())); err != nil {
		return nil, err
	}
	return d.GetDirection(tournamentID)
}

// WithdrawParticipant removes a player. Immediately, their running matches are lost by forfeit;
// deferred, they are no longer paired but the match they are at goes to its end — the case of
// someone who has a train at six.
func (d *Database) WithdrawParticipant(tournamentID int64, id string, afterCurrent bool) (*DirectionView, error) {
	ctx := context.Background()
	dir, err := direction.Open(ctx, d.DirectionStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	ev := tournoi.PlayerWithdrawnEvent(tournoi.PlayerID(id), time.Now())
	if afterCurrent {
		ev = tournoi.PlayerWithdrawnAfterCurrentEvent(tournoi.PlayerID(id), time.Now())
	}
	if err := dir.Apply(ctx, ev); err != nil {
		return nil, err
	}
	return d.GetDirection(tournamentID)
}

// participantID derives a stable identifier from a name. Two people who sign the same get a
// visible numeric suffix rather than being merged: blunderDB has no notion of a person behind
// a name, and inventing one here would be exactly the identity CONTEXT.md refuses.
func participantID(name string, taken map[string]bool) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		case b.Len() > 0 && b.String()[b.Len()-1] != '-':
			b.WriteRune('-')
		}
	}
	base := strings.Trim(b.String(), "-")
	if base == "" {
		base = "joueur"
	}
	if !taken[base] {
		return base
	}
	for n := 2; ; n++ {
		candidate := fmt.Sprintf("%s-%d", base, n)
		if !taken[candidate] {
			return candidate
		}
	}
}
