package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
)

// Confirming what the engine proposes, and doing what it did not (tasks/nicomaque/fonctionnel.md §5): the engine
// proposes, the director decides, the Direction records, and only the impossible is refused.
// Every call writes an event and returns the replayed view.

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

// ConfirmAllProposals records every proposal in one gesture. It stops at the first refusal and
// returns it with what was already recorded: a partial round is actionable, a rollback would
// discard valid decisions.
func (d *Database) ConfirmAllProposals(tournamentID int64) (*DirectionView, error) {
	ctx := context.Background()
	dir, err := direction.Open(ctx, d.DirectionStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	// One instant for the whole batch: what is launched together is one round, and the engine
	// reads that from the start times.
	now := time.Now()
	// Proposed at the SAME instant the events will carry, or the queue and its confirmation
	// disagree about a micro-round's deadline.
	for _, a := range dir.ProposeAt(now) {
		if a.Kind == tournoi.ActWait {
			continue
		}
		if a.Kind == tournoi.ActFinish {
			// Closing is not launching. It freezes the final standings, so it deserves a
			// deliberate click of its own rather than riding along with a queue of matches
			// (found by the standings test, which closed the tournament without meaning to).
			continue
		}
		if a.Reason == tournoi.ReasonWaitingTable {
			// A proposal with no free table stays in the queue: launching it here would put
			// two matches on one table, or none, without the director ever choosing. They
			// launch it themselves with a table they picked (tasks/nicomaque/fonctionnel.md §3.2).
			continue
		}
		if err := confirmAt(ctx, dir, a, now); err != nil {
			return nil, err
		}
	}
	return d.GetDirection(tournamentID)
}

// confirm turns one action into its event and records it.
func confirm(ctx context.Context, dir *direction.Direction, a tournoi.Action) error {
	return confirmAt(ctx, dir, a, time.Now())
}

// confirmAt is confirm with the instant given rather than taken.
//
// A BATCH of matches must carry ONE instant: the engine groups matches into rounds by start
// time, so successive time.Now() calls would make one round per match.
func confirmAt(ctx context.Context, dir *direction.Direction, a tournoi.Action, now time.Time) error {
	ev, err := dir.EventFor(a, now)
	if err != nil {
		return err
	}
	return dir.Apply(ctx, ev)
}

// StartMatchManually launches a match the engine did not propose: two free Participants, the
// length and the table the director chose.
//
// A pairing off the graph is ACCEPTED with a standing warning; only the impossible is refused
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
	if table <= 0 {
		// No number typed: the first free table, as a proposal would get. With none left the
		// match still starts, under the grid's "no table" cell.
		table = firstFreeTable(st, "", st.Current)
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

// firstFreeTable is the table the engine would give a proposal of this section and phase: the
// smallest one that is not in use, not out of service and not reserved for something else; 0
// when there is none.
//
// It restates Nicomaque's unexported assignTables (engine.go) through AvailableFor, and must
// follow it if it changes.
func firstFreeTable(st *tournoi.State, section string, phase int) int {
	used := map[int]bool{}
	for _, m := range st.Running() {
		if m.Table > 0 {
			used[m.Table] = true
		}
	}
	tables := st.Config.Tables
	for t := 1; tables.Count == 0 || t <= tables.Count; t++ {
		if tables.Count == 0 && t > len(used)+len(tables.Unavailable)+len(tables.Reserved)+1 {
			break // an unlimited room: nothing to find beyond this
		}
		if !used[t] && tables.AvailableFor(t, section, phase) {
			return t
		}
	}
	return 0
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
