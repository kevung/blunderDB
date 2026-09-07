package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
)

// Confirming what the engine proposes, and doing what it did not (ADR-0047 §5).
//
// The posture is the one ADR-0044 took for a transcription, moved from a match to a room: the
// engine proposes, the director decides, the Direction records, and nothing is refused but the
// impossible. So every call here writes an event and comes back with the replayed view; none of
// them second-guesses the director.

// ConfirmProposal records one proposal the director confirmed. The proposal travels back as the
// engine's own JSON, so the frontend confirms exactly what it was shown rather than describing
// it again in its own words — a description that could drift from the engine's.
func (d *Database) ConfirmProposal(tournamentID int64, actionJSON string) (*DirectionView, error) {
	var a tournoi.Action
	if err := json.Unmarshal([]byte(actionJSON), &a); err != nil {
		return nil, fmt.Errorf("direction: proposal: %w", err)
	}
	ctx := context.Background()
	dir, err := direction.Open(ctx, d.DirectionStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	if err := confirm(ctx, dir, a); err != nil {
		return nil, err
	}
	return d.GetDirection(tournamentID)
}

// ConfirmAllProposals records every proposal in one gesture. It is two clicks for the director
// whatever the count, which is the budget the UX document holds this to.
//
// It stops at the first refusal and returns it with what has already been recorded: a partial
// round is a real state the director can act on, whereas rolling back would throw away
// decisions that were valid. Nothing is silently skipped.
func (d *Database) ConfirmAllProposals(tournamentID int64) (*DirectionView, error) {
	ctx := context.Background()
	dir, err := direction.Open(ctx, d.DirectionStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	for _, a := range dir.Propose() {
		if a.Kind == tournoi.ActWait {
			continue
		}
		if err := confirm(ctx, dir, a); err != nil {
			return nil, err
		}
	}
	return d.GetDirection(tournamentID)
}

// confirm turns one action into its event and records it.
func confirm(ctx context.Context, dir *direction.Direction, a tournoi.Action) error {
	ev, err := dir.EventFor(a, time.Now())
	if err != nil {
		return err
	}
	return dir.Apply(ctx, ev)
}

// StartMatchManually launches a match the engine did not propose: two free Participants, the
// length and the table the director chose.
//
// This is the escape hatch that makes the whole panel usable by a real director — the one who
// knows a player has a train to catch. A pairing that does not match the graph is ACCEPTED and
// leaves a standing warning; only the impossible is refused, and the engine says which
// (unknown player, already playing, against themselves).
func (d *Database) StartMatchManually(tournamentID int64, a, b string, length, table int) (*DirectionView, error) {
	ctx := context.Background()
	dir, err := direction.Open(ctx, d.DirectionStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	st := dir.State()
	if st == nil {
		return nil, fmt.Errorf("direction: the tournament has not started")
	}
	if length <= 0 {
		length = st.Phases[st.Current].Length
	}
	act := tournoi.Action{
		Kind: tournoi.ActStartMatch, Phase: st.Current,
		A: tournoi.PlayerID(a), B: tournoi.PlayerID(b),
		Length: length, Table: table,
	}
	if err := confirm(ctx, dir, act); err != nil {
		return nil, err
	}
	return d.GetDirection(tournamentID)
}

// FreeParticipants names the Participants of the current phase who are not playing: who the
// director can pair by hand, and the waiting queue the panel shows.
func (d *Database) FreeParticipants(tournamentID int64) ([]tournoi.Player, error) {
	dir, err := direction.Open(context.Background(), d.DirectionStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	st := dir.State()
	if st == nil {
		return nil, nil
	}
	busy := map[tournoi.PlayerID]bool{}
	for _, m := range st.Running() {
		busy[m.A], busy[m.B] = true, true
	}
	var out []tournoi.Player
	for _, id := range st.Order {
		p := st.Players[id]
		if p == nil || busy[id] || st.Withdrawn[id] {
			continue
		}
		out = append(out, *p)
	}
	return out, nil
}
