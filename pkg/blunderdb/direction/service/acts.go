package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Confirming what the engine proposes, and doing what it did not (tasks/nicomaque/fonctionnel.md §5): the engine
// proposes, the director decides, the Direction records, and only the impossible is refused.
// Every call writes an event and returns the replayed view.

// ConfirmProposal records one proposal the director confirmed. The proposal travels back as the
// engine's own JSON, so the frontend confirms exactly what it was shown rather than describing
// it again in its own words — a description that could drift from the engine's.
//
// In a Rencontre the proposal may be stale: a sister event may have taken its table since it was
// shown. It is then refused, and proposed again on a table still free.
func (d *Service) ConfirmProposal(ctx context.Context, tournamentID int64, actionJSON string) (*DirectionView, error) {
	var a tournoi.Action
	if err := json.Unmarshal([]byte(actionJSON), &a); err != nil {
		return nil, fmt.Errorf("direction: proposal: %w", err)
	}
	if err := d.confirmProposal(ctx, tournamentID, a); err != nil {
		return nil, err
	}
	return d.GetDirection(ctx, tournamentID)
}

func (d *Service) confirmProposal(ctx context.Context, tournamentID int64, a tournoi.Action) (err error) {
	d, release, shared, err := d.lockTables(ctx, tournamentID)
	if err != nil {
		return err
	}
	defer release(&err)
	if shared && a.Kind == tournoi.ActStartMatch && a.Table > 0 {
		if _, taken := d.roomAround(ctx, tournamentID, nil).tables[a.Table]; taken {
			return fmt.Errorf("direction: table %d is taken", a.Table)
		}
	}
	dir, err := direction.Open(ctx, d.dirStore(), tournamentID)
	if err != nil {
		return err
	}
	return confirm(ctx, dir, a)
}

// ConfirmAllProposals records every proposal in one gesture. It stops at the first refusal and
// returns it with what was already recorded: a partial round is actionable, a rollback would
// discard valid decisions.
func (d *Service) ConfirmAllProposals(ctx context.Context, tournamentID int64) (*DirectionView, error) {
	if err := d.confirmAllProposals(ctx, tournamentID); err != nil {
		return nil, err
	}
	return d.GetDirection(ctx, tournamentID)
}

func (d *Service) confirmAllProposals(ctx context.Context, tournamentID int64) (err error) {
	// The room's lock in a Rencontre: the sisters' tables are read once, and none of them may
	// take one of the tables handed out here before the batch is written.
	d, release, _, err := d.lockTables(ctx, tournamentID)
	if err != nil {
		return err
	}
	defer release(&err)
	dir, err := direction.Open(ctx, d.dirStore(), tournamentID)
	if err != nil {
		return err
	}
	// One instant for the whole batch: what is launched together is one round, and the engine
	// reads that from the start times.
	now := time.Now()
	// Proposed at the SAME instant the events will carry, or the queue and its confirmation
	// disagree about a micro-round's deadline.
	// The sister events of the Rencontre, if any, hold tables the engine must not give out.
	for _, a := range dir.ProposeWith(now, d.outside(ctx, tournamentID, dir)) {
		if a.Kind == tournoi.ActWait {
			continue
		}
		if a.Kind == tournoi.ActFinish {
			// Closing is not launching. It freezes the final standings, so it deserves a
			// deliberate click of its own rather than riding along with a queue of matches
			// (found by the standings test, which closed the tournament without meaning to).
			continue
		}
		if a.Reason == tournoi.ReasonWaitingTable || a.Reason == tournoi.ReasonPlayerUnavailable ||
			a.Reason == tournoi.ReasonPlayerBusy {
			// A proposal with no table stays in the queue: launching it here would put two
			// matches on one table, or none, without the director ever choosing. In rounds
			// mode the round stays open until all its players are engaged, so the rest of it
			// is proposed again as tables free up (tasks/nicomaque/fonctionnel.md §3.2).
			continue
		}
		if err := confirmAt(ctx, dir, a, now); err != nil {
			return err
		}
	}
	return nil
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
// (unknown player, already playing, against themselves) — and a table another match is
// played on, in this event or a sister of its room, which would hide that match.
func (d *Service) StartMatchManually(ctx context.Context, tournamentID int64, a, b string, length, table int) (_ *DirectionView, err error) {
	d, release, shared, err := d.lockTables(ctx, tournamentID)
	if err != nil {
		return nil, err
	}
	defer release(&err)
	// The sisters' tables are read before the transaction opens — on a single-connection
	// database a read beside it would wait for it forever — and the room's lock keeps them as
	// read until the match is written.
	sisters := map[int]string{}
	if shared {
		sisters = d.roomAround(ctx, tournamentID, nil).tables
	}
	// The occupancy check and the write share one transaction: checked outside it, two
	// directors could both see the table free and both start a match on it.
	err = d.directionTx(ctx, func(ctx context.Context, _ storage.Tx, store direction.Store) error {
		dir, err := direction.Open(ctx, store, tournamentID)
		if err != nil {
			return err
		}
		st := dir.State()
		if st == nil {
			return fmt.Errorf("direction: the tournament has not started")
		}
		if length <= 0 {
			length = st.Phases[st.Current].Length
		}
		if table <= 0 {
			// No number typed: the first free table, as a proposal would get — one no sister
			// event of the room plays on either. With none left the match still starts, under
			// the grid's "no table" cell.
			table = firstFreeTable(st, "", st.Current, sisters)
		} else if _, next := sisters[table]; next || occupant(st, table, "") != nil {
			return fmt.Errorf("direction: table %d is taken", table)
		}
		return confirm(ctx, dir, tournoi.Action{
			Kind: tournoi.ActStartMatch, Phase: st.Current,
			A: tournoi.PlayerID(a), B: tournoi.PlayerID(b),
			Length: length, Table: table,
		})
	})
	if err != nil {
		return nil, err
	}
	return d.GetDirection(ctx, tournamentID)
}

// firstFreeTable is the table the engine would give a proposal of this section and phase: the
// smallest one that is not in use, not out of service and not reserved for something else; 0
// when there is none. sisters are the tables the room's other events play on.
//
// It restates Nicomaque's unexported assignTables (engine.go) through AvailableFor, and must
// follow it if it changes.
func firstFreeTable(st *tournoi.State, section string, phase int, sisters map[int]string) int {
	used := map[int]bool{}
	for t := range sisters {
		used[t] = true
	}
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
func (d *Service) FreeParticipants(ctx context.Context, tournamentID int64) ([]tournoi.Player, error) {
	dir, err := direction.Open(ctx, d.dirStore(), tournamentID)
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
