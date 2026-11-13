package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	tournoi "github.com/kevung/nicomaque"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Confirming what the engine proposes, and doing what it did not: the engine
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
		return nil, direction.Refused(fmt.Errorf("direction: proposal: %w", err))
	}
	if err := d.confirmProposal(ctx, tournamentID, a); err != nil {
		return nil, err
	}
	return d.viewAfter(ctx, tournamentID)
}

func (d *Service) confirmProposal(ctx context.Context, tournamentID int64, a tournoi.Action) (err error) {
	d, release, shared, err := d.lockTables(ctx, tournamentID)
	if err != nil {
		return err
	}
	defer release(&err)
	dir, err := direction.Open(ctx, d.dirStore(), tournamentID)
	if err != nil {
		return err
	}
	if shared && a.Kind == tournoi.ActStartMatch {
		room, err := d.roomAround(ctx, tournamentID, dir)
		if err != nil {
			return err
		}
		if err := room.refuseSeat(dir, a); err != nil {
			return err
		}
	}
	if err := d.refuseTable(ctx, dir, tournamentID, a); err != nil {
		return err
	}
	return confirm(ctx, dir, a)
}

// refuseSeat refuses launching a match on a table a sister event plays on, or with a person
// who sits at a sister's match: a proposal shown before the sister wrote is stale.
func (r sisterRoom) refuseSeat(dir *direction.Direction, a tournoi.Action) error {
	if a.Kind != tournoi.ActStartMatch {
		return nil
	}
	if _, taken := r.tables[a.Table]; a.Table > 0 && taken {
		return direction.RefuseAs(direction.ReasonTableTaken, "direction: table %d is taken", a.Table)
	}
	// One person cannot sit at two matches: a proposal shown before its player sat down
	// next door, or held for that very reason, is not launched.
	for _, p := range []tournoi.PlayerID{a.A, a.B} {
		if seat, busy := r.players[p]; busy && p != "" {
			name := string(p)
			if pl := dir.State().Players[p]; pl != nil && pl.Name != "" {
				name = pl.Name
			}
			return direction.RefuseAs(direction.ReasonPlayerBusy, "direction: %s plays in %s, table %d", name, seat.Event, seat.Table)
		}
	}
	return nil
}

// refuseTable refuses launching a match on a table that does not exist, is outside the
// event's rooms, or is closed to it.
func (d *Service) refuseTable(ctx context.Context, dir *direction.Direction, tournamentID int64, a tournoi.Action) error {
	if a.Kind != tournoi.ActStartMatch || a.Table <= 0 {
		return nil
	}
	if err := d.tableBeyond(ctx, dir, tournamentID, a.Table); err != nil {
		return err
	}
	if err := d.refuseOutside(ctx, dir, tournamentID, a.Table); err != nil {
		return err
	}
	return d.refuseClosed(ctx, dir, tournamentID, a)
}

// ConfirmProposals records a list of proposals exactly as the director was shown them, in one
// gesture and at one instant, all or none: the matches launched together are one batch.
//
// The list travels back as the engine's own JSON, like ConfirmProposal's action. Nothing is
// written, and the refusal carries its direction.Reason, when one of them
//   - is held back by the engine — it waits for a table or a player — or is a wait
//     (ReasonProposalHeld): the director is shown those as not confirmable (Held), so a list
//     holding one is not what the panel offered, and launching it would seat a match nowhere;
//   - names a table or a person a sister event of the room took since (ReasonTableTaken,
//     ReasonPlayerBusy);
//   - is no longer proposed as listed, or is listed twice (ReasonProposalChanged).
//
// A tied repechage and a closing are confirmed when listed: listing them is the deliberate
// choice ConfirmAllProposals waits for. What the list brings about — the queue behind a
// repechage, the next phase's draw — is proposed in the view handed back and never confirmed
// here; when a repechage changes a proposal listed after it, the whole list is refused.
func (d *Service) ConfirmProposals(ctx context.Context, tournamentID int64, actionsJSON string) (*DirectionView, error) {
	var list []tournoi.Action
	if err := json.Unmarshal([]byte(actionsJSON), &list); err != nil {
		return nil, direction.Refused(fmt.Errorf("direction: proposals: %w", err))
	}
	if len(list) == 0 {
		return nil, direction.Refusef("direction: no proposal to confirm")
	}
	if err := d.confirmProposals(ctx, tournamentID, list); err != nil {
		return nil, err
	}
	return d.viewAfter(ctx, tournamentID)
}

func (d *Service) confirmProposals(ctx context.Context, tournamentID int64, list []tournoi.Action) (err error) {
	d, release, _, err := d.lockTables(ctx, tournamentID)
	if err != nil {
		return err
	}
	defer release(&err)
	dir, err := direction.Open(ctx, d.dirStore(), tournamentID)
	if err != nil {
		return err
	}
	// One instant for the list, and the proposals it is compared with read at that instant.
	now := time.Now()
	room, err := d.roomAround(ctx, tournamentID, dir)
	if err != nil {
		return err
	}
	for _, a := range list {
		if a.Kind == tournoi.ActWait || heldForTable(a) {
			return direction.RefuseAs(direction.ReasonProposalHeld, "direction: %s is held back, there is nothing to confirm", describe(a))
		}
		if err := room.refuseSeat(dir, a); err != nil {
			return err
		}
	}
	if err := stillProposed(room.propose(dir, now), list); err != nil {
		return err
	}
	for i, a := range list {
		if err := d.refuseTable(ctx, dir, tournamentID, a); err != nil {
			return err
		}
		if err := confirmAt(ctx, dir, a, now); err != nil {
			return err
		}
		if a.Kind == tournoi.ActRepechage && i+1 < len(list) {
			// The repechage changes who enters the next phase: what was listed behind it
			// must still be what the engine proposes over the new state.
			if err := stillProposed(room.propose(dir, now), list[i+1:]); err != nil {
				return err
			}
		}
	}
	return nil
}

// stillProposed refuses a list one action of which is not among proposals, each proposal
// answering for one action of the list.
func stillProposed(proposals, list []tournoi.Action) error {
	left := map[string]int{}
	for _, p := range proposals {
		left[proposalKey(p)]++
	}
	for _, a := range list {
		k := proposalKey(a)
		if left[k] == 0 {
			return direction.RefuseAs(direction.ReasonProposalChanged, "direction: %s is no longer proposed", describe(a))
		}
		left[k]--
	}
	return nil
}

// proposalKey is what makes two proposals the same one: everything the engine decided, less
// its warning, which follows the clock and changes nothing of what is recorded.
func proposalKey(a tournoi.Action) string {
	a.Warn = ""
	blob, _ := json.Marshal(a)
	return string(blob)
}

// describe names a proposal in a refusal.
func describe(a tournoi.Action) string {
	switch {
	case a.A != "" && a.B != "":
		return fmt.Sprintf("%s %s - %s", a.Kind, a.A, a.B)
	case a.A != "":
		return fmt.Sprintf("%s %s", a.Kind, a.A)
	}
	return string(a.Kind)
}

// ConfirmAllProposals records every proposal in one gesture, all or none: a refusal or a failure
// at any proposal leaves the log as it was, and the director confirms again what is still
// proposed.
func (d *Service) ConfirmAllProposals(ctx context.Context, tournamentID int64) (*DirectionView, error) {
	if err := d.confirmAllProposals(ctx, tournamentID); err != nil {
		return nil, err
	}
	return d.viewAfter(ctx, tournamentID)
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
	room, err := d.roomAround(ctx, tournamentID, dir)
	if err != nil {
		return err
	}
	proposals := room.propose(dir, now)
	// held: a repechage between tied candidates waits for the director's choice, and so does
	// the passage it would change — confirming it now would turn the place into a bye.
	held := false
	for i, again := 0, 0; i < len(proposals); i++ {
		a := proposals[i]
		if a.Kind == tournoi.ActWait {
			continue
		}
		if a.Kind == tournoi.ActRepechage {
			if a.Label.Players > 1 {
				held = true
				continue
			}
			if err := confirmAt(ctx, dir, a, now); err != nil {
				return err
			}
			// The repechage changes who enters the next phase: the rest of the queue is
			// proposed again over the new state.
			if again++; again > len(dir.State().Order) {
				return fmt.Errorf("direction: repechage proposed again without end")
			}
			proposals, i = room.propose(dir, now), -1
			continue
		}
		if held && (a.Kind == tournoi.ActNextPhase || a.Kind == tournoi.ActDraw) {
			continue
		}
		if a.Kind == tournoi.ActFinish {
			// Closing is not launching. It freezes the final standings, so it deserves a
			// deliberate click of its own rather than riding along with a queue of matches
			// (found by the standings test, which closed the tournament without meaning to).
			continue
		}
		if heldForTable(a) {
			// A proposal with no table stays in the queue: launching it here would put two
			// matches on one table, or none, without the director ever choosing. In rounds
			// mode the round stays open until all its players are engaged, so the rest of it
			// is proposed again as tables free up.
			continue
		}
		if err := confirmAt(ctx, dir, a, now); err != nil {
			return err
		}
	}
	return nil
}

// Held tells a proposal the director cannot confirm yet: the engine holds it back until a
// table or a player frees up, and says so in the proposal's reason. The panel shows it apart,
// and neither ConfirmAllProposals nor ConfirmProposals launches it.
func Held(a tournoi.Action) bool { return heldForTable(a) }

// heldForTable tells a match the engine proposes without a table to play on: every table taken,
// or a player unavailable or at a match of a sister event. It stays in the queue until its
// table, or its player, frees up.
func heldForTable(a tournoi.Action) bool {
	return a.Reason == tournoi.ReasonWaitingTable || a.Reason == tournoi.ReasonPlayerUnavailable ||
		a.Reason == tournoi.ReasonPlayerBusy
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
func (d *Service) StartMatchManually(ctx context.Context, tournamentID int64, a, b string, length, table int) (*DirectionView, error) {
	return d.StartMatchInRound(ctx, tournamentID, 0, a, b, length, table)
}

// StartMatchInRound is StartMatchManually naming the round, which a manual phase needs: there
// the director places every match in a round (the last one, or the next to open it). round 0
// is the latest round of a manual phase; any other phase refuses a round, its own pairing
// numbers them.
func (d *Service) StartMatchInRound(ctx context.Context, tournamentID int64, round int, a, b string, length, table int) (*DirectionView, error) {
	if err := d.startMatchManually(ctx, tournamentID, round, a, b, length, table); err != nil {
		return nil, err
	}
	return d.viewAfter(ctx, tournamentID)
}

func (d *Service) startMatchManually(ctx context.Context, tournamentID int64, round int, a, b string, length, table int) (err error) {
	d, release, shared, err := d.lockTables(ctx, tournamentID)
	if err != nil {
		return err
	}
	defer release(&err)
	// The sisters' tables are read in the gesture's transaction, under the room's lock and the
	// database's guard: no sister seats a match on one of them before this match is written.
	sisters := map[int]string{}
	if shared {
		room, err := d.roomAround(ctx, tournamentID, nil)
		if err != nil {
			return err
		}
		sisters = room.tables
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
			return direction.Refusef("direction: the tournament has not started")
		}
		if length <= 0 {
			length = st.Phases[st.Current].Length
		}
		if err := d.tableBeyond(ctx, dir, tournamentID, table); err != nil {
			return err
		}
		if err := d.refuseOutside(ctx, dir, tournamentID, table); err != nil {
			return err
		}
		if table <= 0 {
			// No number typed: the first free table, as a proposal would get — one no sister
			// event of the room plays on, inside the event's rooms, neither reserved nor kept
			// for someone. With none left the match still starts, under the grid's "no table"
			// cell.
			skip, err := d.closedTables(ctx, dir, tournamentID)
			if err != nil {
				return err
			}
			for n, ev := range sisters {
				skip[n] = ev
			}
			table = firstFreeTable(st, "", st.Current, skip)
		} else if _, next := sisters[table]; next || occupant(st, table, "") != nil {
			return direction.RefuseAs(direction.ReasonTableTaken, "direction: table %d is taken", table)
		}
		ph := st.Phases[st.Current]
		if ph.Cfg.Kind == tournoi.KindManual {
			if round == 0 {
				round = max(ph.Round, 1)
			}
			ev := st.ManualMatchEvent(round, tournoi.PlayerID(a), tournoi.PlayerID(b), time.Now())
			ev.Length, ev.Table = length, table
			return dir.Apply(ctx, ev)
		}
		if round != 0 {
			return direction.Refusef("direction: only a manual phase takes a round, phase %d pairs its own", st.Current+1)
		}
		return confirm(ctx, dir, tournoi.Action{
			Kind: tournoi.ActStartMatch, Phase: st.Current,
			A: tournoi.PlayerID(a), B: tournoi.PlayerID(b),
			Length: length, Table: table,
		})
	})
	if err != nil {
		return err
	}
	return nil
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
