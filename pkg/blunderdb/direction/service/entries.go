package service

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	tournoi "github.com/PileOfCells/backgammon-tournoi"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The entries of a Direction (tasks/nicomaque/fonctionnel.md §4). A Participant is an entry in ONE Direction, not a
// Player nor a person (CONTEXT.md). The only link to a Player is a name spelled the same;
// choosing an existing Player at entry fixes the spelling and pre-fills the rating from their
// PR, and nothing is inferred afterwards.

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
	// State is a CODE — playing, free, withdrawn, leaving, absent — rendered by the frontend.
	State string `json:"state"`
	Table int    `json:"table,omitempty"`
	// AbsentUntil (RFC3339) and AbsentRound say WHEN an "absent" row returns — one of the two,
	// never both (the engine refuses declaring both at once).
	AbsentUntil string `json:"absentUntil,omitempty"`
	AbsentRound int    `json:"absentRound,omitempty"`
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
func (d *Service) EntrySuggestions(ctx context.Context) ([]EntrySuggestion, error) {
	rows, err := d.st.Stats().PlayerTable(ctx, d.scope, storage.StatsFilter{})
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
func (d *Service) Participants(ctx context.Context, tournamentID int64) ([]ParticipantRow, error) {
	dir, err := direction.Open(ctx, d.dirStore(), tournamentID)
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
	now := time.Now()
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
		// isAbsent replays the engine's own absent() test (unexported): a time deadline
		// compares to the wall clock, a round deadline to the phase's own round counter. Only
		// used to LABEL the row — the engine decides pairing on its own, from the journal.
		absence, declared := st.Unavailable[id]
		isAbsent := declared
		if declared {
			switch {
			case absence.Round > 0:
				isAbsent = absence.Phase == st.Current && ph.Round < absence.Round
			case !absence.Until.IsZero():
				isAbsent = now.Before(absence.Until)
			}
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
		case isAbsent:
			row.State = "absent"
			if absence.Round > 0 {
				row.AbsentRound = absence.Round
			} else {
				row.AbsentUntil = absence.Until.Format(time.RFC3339)
			}
		default:
			row.State = "free"
		}
		out = append(out, row)
	}
	return out, nil
}

// AddParticipant enters one player. It is an event, in preparation as afterwards: a late
// arrival is an entry like any other, and the engine decides where they come in.
func (d *Service) AddParticipant(ctx context.Context, tournamentID int64, name, club string, rating float64) (*DirectionView, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("direction: an entry needs a name")
	}
	dir, err := direction.Open(ctx, d.dirStore(), tournamentID)
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
	return d.GetDirection(ctx, tournamentID)
}

// UpdateParticipant corrects an entry — a mistyped name, a missing club, a wrong rating —
// WITHOUT changing its identifier. That matters: the identifier is what a Slot points at, so
// correcting a name must not undo a Match already attached to it.
func (d *Service) UpdateParticipant(ctx context.Context, tournamentID int64, id, name, club string, rating float64) (*DirectionView, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("direction: an entry needs a name")
	}
	dir, err := direction.Open(ctx, d.dirStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	st := dir.State()
	if st == nil {
		return nil, direction.ErrNoDirection
	}
	cur, ok := st.Players[tournoi.PlayerID(id)]
	if !ok {
		return nil, fmt.Errorf("direction: no entry %q", id)
	}
	// The whole record travels, so what this form does not edit keeps its value; a correction
	// leaves the entry's place, its matches, its Slots and a withdrawal untouched.
	p := *cur
	p.Name, p.Club, p.Rating = name, club, rating
	if err := dir.Apply(ctx, tournoi.PlayerUpdatedEvent(p, time.Now())); err != nil {
		return nil, err
	}
	return d.GetDirection(ctx, tournamentID)
}

// ReinstateParticipant brings a withdrawn player back into the tournament: they are
// paired again, with the results and lives they had when they left; the matches their
// withdrawal lost by forfeit stay lost. It is refused for someone who has not withdrawn, so a
// mistaken click cannot re-enter a player twice.
func (d *Service) ReinstateParticipant(ctx context.Context, tournamentID int64, id string) (*DirectionView, error) {
	dir, err := direction.Open(ctx, d.dirStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	st := dir.State()
	if st == nil {
		return nil, direction.ErrNoDirection
	}
	p, ok := st.Players[tournoi.PlayerID(id)]
	if !ok {
		return nil, fmt.Errorf("direction: no entry %q", id)
	}
	if !st.Withdrawn[p.ID] {
		return nil, fmt.Errorf("direction: %q has not withdrawn", id)
	}
	// Re-entering under the same identifier is the engine's only way back: it clears the
	// withdrawal and keeps everything else.
	if err := dir.Apply(ctx, tournoi.PlayerAddedEvent(*p, time.Now())); err != nil {
		return nil, err
	}
	return d.GetDirection(ctx, tournamentID)
}

// WithdrawParticipant removes a player. Immediately, their running matches are lost by forfeit;
// deferred, they are no longer paired but the match they are at goes to its end — the case of
// someone who has a train at six.
func (d *Service) WithdrawParticipant(ctx context.Context, tournamentID int64, id string, afterCurrent bool) (*DirectionView, error) {
	dir, err := direction.Open(ctx, d.dirStore(), tournamentID)
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
	return d.GetDirection(ctx, tournamentID)
}

// MakeParticipantAbsent declares a player unavailable for a time (D7.1): they keep their rank,
// their lives and everywhere they stand, but the engine stops pairing them — a suisse waits for
// them by name, a bracket holds their match without a table — until the deadline. Exactly one
// of until/round is given: until (RFC3339, empty = none) for a return at an hour, round (0 =
// none) for a return at a round of the current swiss-by-rounds phase. Giving both, or neither,
// or a round outside a rounds phase, is refused by the engine itself.
func (d *Service) MakeParticipantAbsent(ctx context.Context, tournamentID int64, id, until string, round int) (*DirectionView, error) {
	dir, err := direction.Open(ctx, d.dirStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	var ev tournoi.Event
	if round > 0 {
		ev = tournoi.PlayerUnavailableUntilRoundEvent(tournoi.PlayerID(id), round, time.Now())
	} else {
		var t time.Time
		if until != "" {
			t, err = time.Parse(time.RFC3339, until)
			if err != nil {
				return nil, fmt.Errorf("direction: absence: heure de retour invalide: %w", err)
			}
		}
		if t.IsZero() {
			return nil, fmt.Errorf("direction: absence: une heure ou une ronde de retour est requise")
		}
		ev = tournoi.PlayerUnavailableEvent(tournoi.PlayerID(id), t, time.Now())
	}
	if err := dir.Apply(ctx, ev); err != nil {
		return nil, err
	}
	return d.GetDirection(ctx, tournamentID)
}

// MakeParticipantAvailable lifts an absence, whatever its deadline, and returns the player to
// pairing right away — the one-click way back the queue's own countdown does automatically.
func (d *Service) MakeParticipantAvailable(ctx context.Context, tournamentID int64, id string) (*DirectionView, error) {
	dir, err := direction.Open(ctx, d.dirStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	if err := dir.Apply(ctx, tournoi.PlayerAvailableEvent(tournoi.PlayerID(id), time.Now())); err != nil {
		return nil, err
	}
	return d.GetDirection(ctx, tournamentID)
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
