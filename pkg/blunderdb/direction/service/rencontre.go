package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/trash"
)

// A Rencontre gathers directed Tournaments playing in one room (ADR-0056). Its life: created,
// Tournaments attached (their tables aligned by a configuration change the director saw first)
// and detached, deleted through the trash. A gesture on the room is declared once and written as
// one configuration change in every member Direction, in a single transaction: each log still
// replays on its own, and none is left half-told.

// RencontreMember is one Tournament playing in the room.
type RencontreMember struct {
	TournamentID int64  `json:"tournamentId"`
	Name         string `json:"name"`
	State        string `json:"state"`
}

// RencontreView is a Rencontre as the panel shows it: the room as its members' logs state it
// now, and who plays in it.
type RencontreView struct {
	domain.Rencontre
	Room    direction.Room    `json:"room"`
	Members []RencontreMember `json:"members"`
}

// CreateRencontre opens a room with its number of tables.
func (d *Service) CreateRencontre(ctx context.Context, name, startsOn, endsOn string, tables int) (*RencontreView, error) {
	if name == "" {
		return nil, direction.Refusef("rencontre: a name is required")
	}
	if tables <= 0 {
		return nil, direction.Refusef("rencontre: the room needs at least one table")
	}
	id, err := d.st.Rencontres().Create(ctx, d.scope, domain.Rencontre{
		Name: name, StartsOn: startsOn, EndsOn: endsOn, Tables: tables,
	})
	if err != nil {
		return nil, err
	}
	d.publishRoom(ctx, id)
	return d.GetRencontre(ctx, id)
}

// ListRencontres returns every Rencontre, newest first.
func (d *Service) ListRencontres(ctx context.Context) ([]RencontreView, error) {
	rs, err := d.st.Rencontres().List(ctx, d.scope)
	if err != nil {
		return nil, err
	}
	out := make([]RencontreView, 0, len(rs))
	for _, r := range rs {
		v, err := d.rencontreView(ctx, d.st, d.dirStore(), r)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, nil
}

// GetRencontre returns one Rencontre with its room and members.
func (d *Service) GetRencontre(ctx context.Context, id int64) (*RencontreView, error) {
	r, err := d.st.Rencontres().Get(ctx, d.scope, id)
	if err != nil {
		return nil, err
	}
	return d.rencontreView(ctx, d.st, d.dirStore(), r)
}

// RencontreOf names the Rencontre a Tournament plays in, 0 when none.
func (d *Service) RencontreOf(ctx context.Context, tournamentID int64) (int64, error) {
	return d.st.Rencontres().Of(ctx, d.scope, tournamentID)
}

func (d *Service) rencontreView(ctx context.Context, stores storage.Stores, store direction.Store, r *domain.Rencontre) (*RencontreView, error) {
	v := &RencontreView{Rencontre: *r, Room: direction.Room{Tables: r.Tables}, Members: []RencontreMember{}}
	roomRead := false
	for _, tid := range r.TournamentIDs {
		m := RencontreMember{TournamentID: tid}
		if t, err := stores.Tournaments().Get(ctx, d.scope, tid); err == nil {
			m.Name = t.Name
		}
		if dir, err := direction.Open(ctx, store, tid); err == nil {
			m.State = string(dir.Record().State)
			// Every member sits in the same room; the first one with a log says what it is.
			if cfg, err := dir.Config(); err == nil && !roomRead {
				v.Room = direction.RoomOf(cfg)
				v.Room.Tables = r.Tables
				roomRead = true
			}
		}
		v.Members = append(v.Members, m)
	}
	return v, nil
}

// UpdateRencontre renames the room and moves its dates. A new number of tables is a gesture on
// the room: every member Direction records it.
func (d *Service) UpdateRencontre(ctx context.Context, id int64, name, startsOn, endsOn string, tables int) (*RencontreView, error) {
	if name == "" || tables <= 0 {
		return nil, direction.Refusef("rencontre: a name and at least one table are required")
	}
	err := d.lockedRoom(ctx, id, func(ctx context.Context, tx storage.Tx, store direction.Store, r *domain.Rencontre, room direction.Room) error {
		changed := r.Tables != tables
		r.Name, r.StartsOn, r.EndsOn, r.Tables = name, startsOn, endsOn, tables
		if err := tx.Rencontres().Update(ctx, d.scope, *r); err != nil {
			return err
		}
		if !changed {
			return nil
		}
		room.Tables = tables
		return applyRoom(ctx, store, r.TournamentIDs, room, false)
	})
	if err != nil {
		return nil, err
	}
	return d.afterRoomGesture(ctx, id)
}

// PreviewAttachToRencontre shows what attaching changes in the Tournament's configuration — its
// tables become the room's — without writing anything.
func (d *Service) PreviewAttachToRencontre(ctx context.Context, tournamentID, rencontreID int64) (*ConfigPreview, error) {
	next, err := d.alignedConfig(ctx, d.dirStore(), tournamentID, rencontreID)
	if err != nil {
		return nil, err
	}
	blob, err := json.Marshal(next)
	if err != nil {
		return nil, err
	}
	return d.PreviewDirectionConfig(ctx, tournamentID, string(blob))
}

// AttachToRencontre puts a directed Tournament in the room: its tables are aligned by a
// configuration change, and membership is recorded, in one transaction. Permitted at any time,
// the event under way included. A Tournament without a Direction has no room to join.
func (d *Service) AttachToRencontre(ctx context.Context, tournamentID, rencontreID int64) (*RencontreView, error) {
	if err := d.attach(ctx, tournamentID, rencontreID); err != nil {
		return nil, err
	}
	return d.afterRoomGesture(ctx, rencontreID)
}

// attach runs under the Tournament's lock and the room's: a gesture on the Tournament that read
// it unattached must not run alongside.
func (d *Service) attach(ctx context.Context, tournamentID, rencontreID int64) (err error) {
	d, release, err := d.lockMembership(ctx, tournamentID, rencontreID)
	if err != nil {
		return err
	}
	defer release(&err)
	return d.inRoom(ctx, rencontreID, func(ctx context.Context, tx storage.Tx, store direction.Store, r *domain.Rencontre, room direction.Room) error {
		if of, err := tx.Rencontres().Of(ctx, d.scope, tournamentID); err != nil {
			return err
		} else if of != 0 && of != rencontreID {
			return direction.Refusef("rencontre: tournament %d already plays in another Rencontre", tournamentID)
		}
		if err := alignTables(ctx, store, tournamentID, room); err != nil {
			return err
		}
		return tx.Rencontres().Attach(ctx, d.scope, tournamentID, rencontreID)
	})
}

// alignTables puts one Tournament on the room's tables by a configuration change, and writes
// nothing when it already sits on them.
func alignTables(ctx context.Context, store direction.Store, tournamentID int64, room direction.Room) error {
	dir, err := direction.Open(ctx, store, tournamentID)
	if err != nil {
		return err
	}
	cfg, err := dir.Config()
	if err != nil {
		return err
	}
	if next := direction.WithTables(cfg, room); !sameTables(cfg, next) {
		return dir.SetConfigAt(ctx, next, time.Now())
	}
	return nil
}

// Realign puts every member of a Rencontre back on the room's tables, as attaching
// them does. A restored Rencontre needs it: its events kept their own tables while detached.
func (d *Service) Realign(ctx context.Context, id int64) error {
	if err := d.realign(ctx, id); err != nil {
		return err
	}
	d.writeRencontrePages(context.WithoutCancel(ctx), id)
	return nil
}

func (d *Service) realign(ctx context.Context, id int64) (err error) {
	d, release, err := d.lockRoom(ctx, 0, id)
	if err != nil {
		return err
	}
	defer release(&err)
	return d.inRoom(ctx, id, func(ctx context.Context, _ storage.Tx, store direction.Store, r *domain.Rencontre, room direction.Room) error {
		for _, tid := range r.TournamentIDs {
			if err := alignTables(ctx, store, tid, room); err != nil {
				return fmt.Errorf("rencontre: tournament %d: %w", tid, err)
			}
		}
		return nil
	})
}

// DetachFromRencontre takes a Tournament out of its room. It keeps its log and its tables. Its
// own page and its former sisters' are rewritten: none of them shares the room any more.
func (d *Service) DetachFromRencontre(ctx context.Context, tournamentID int64) error {
	rid, err := d.detach(ctx, tournamentID)
	if err != nil {
		return err
	}
	ctx = context.WithoutCancel(ctx)
	d.writePages(ctx, tournamentID)
	if rid != 0 {
		d.writeRencontrePages(ctx, rid)
	}
	return nil
}

func (d *Service) detach(ctx context.Context, tournamentID int64) (rid int64, err error) {
	d, release, err := d.lockMembership(ctx, tournamentID, 0)
	if err != nil {
		return 0, err
	}
	defer release(&err)
	if rid, err = d.st.Rencontres().Of(ctx, d.scope, tournamentID); err != nil {
		return 0, err
	}
	return rid, d.st.Rencontres().Attach(ctx, d.scope, tournamentID, 0)
}

// TrashRencontre deletes a Rencontre through the trash (ADR-0036). Its Tournaments are detached,
// never deleted; their pages are rewritten without the room.
func (d *Service) TrashRencontre(ctx context.Context, id int64) (int64, error) {
	trashID, members, err := d.trashRencontre(ctx, id)
	if err != nil {
		return 0, err
	}
	d.writePages(context.WithoutCancel(ctx), members...)
	return trashID, nil
}

func (d *Service) trashRencontre(ctx context.Context, id int64) (_ int64, members []int64, err error) {
	d, release, err := d.lockRoom(ctx, 0, id)
	if err != nil {
		return 0, nil, err
	}
	defer release(&err)
	r, err := d.st.Rencontres().Get(ctx, d.scope, id)
	if err != nil {
		return 0, nil, err
	}
	trashID, err := trash.Rencontre(ctx, d.st, d.scope, id)
	return trashID, r.TournamentIDs, err
}

// SetRencontreTableOutOfService declares a table of the room out of service, or back in service.
// Declared once, recorded in every member Direction.
func (d *Service) SetRencontreTableOutOfService(ctx context.Context, id int64, table int, out bool) (*RencontreView, error) {
	err := d.lockedRoom(ctx, id, func(ctx context.Context, _ storage.Tx, store direction.Store, r *domain.Rencontre, room direction.Room) error {
		if table <= 0 || table > r.Tables {
			return direction.Refusef("rencontre: the room has no table %d", table)
		}
		room.Unavailable = slices.DeleteFunc(slices.Clone(room.Unavailable), func(t int) bool { return t == table })
		if out {
			room.Unavailable = append(room.Unavailable, table)
		}
		return applyRoom(ctx, store, r.TournamentIDs, room, false)
	})
	if err != nil {
		return nil, err
	}
	return d.afterRoomGesture(ctx, id)
}

// SetRencontreBreaks replaces the room's breaks — a meal, the prize-giving — in every member
// Direction at once. breaksJSON is the engine's own list of {start, end}.
func (d *Service) SetRencontreBreaks(ctx context.Context, id int64, breaksJSON string) (*RencontreView, error) {
	var breaks []tournoi.TimeRange
	if breaksJSON != "" {
		if err := json.Unmarshal([]byte(breaksJSON), &breaks); err != nil {
			return nil, direction.Refused(fmt.Errorf("rencontre: breaks: %w", err))
		}
	}
	err := d.lockedRoom(ctx, id, func(ctx context.Context, _ storage.Tx, store direction.Store, r *domain.Rencontre, room direction.Room) error {
		room.Breaks = breaks
		return applyRoom(ctx, store, r.TournamentIDs, room, true)
	})
	if err != nil {
		return nil, err
	}
	return d.afterRoomGesture(ctx, id)
}

// lockedRoom is inRoom under the room's lock, released on return: what a room gesture does
// next — rewriting its wall page — runs without it.
func (d *Service) lockedRoom(ctx context.Context, id int64, fn func(context.Context, storage.Tx, direction.Store, *domain.Rencontre, direction.Room) error) (err error) {
	d, release, err := d.lockRoom(ctx, 0, id)
	if err != nil {
		return err
	}
	defer release(&err)
	return d.inRoom(ctx, id, fn)
}

// inRoom runs fn inside one transaction with the Rencontre and its room
// as the members' logs state it. Nothing fn writes survives an error.
func (d *Service) inRoom(ctx context.Context, id int64, fn func(context.Context, storage.Tx, direction.Store, *domain.Rencontre, direction.Room) error) error {
	return d.inTx(ctx, func(tx storage.Tx, store direction.Store) error {
		r, err := tx.Rencontres().Get(ctx, d.scope, id)
		if err != nil {
			return err
		}
		v, err := d.rencontreView(ctx, tx, store, r)
		if err != nil {
			return err
		}
		return fn(ctx, tx, store, r, v.Room)
	})
}

// applyRoom writes the room into every member Direction that is not already in it. withBreaks
// says whether the breaks are part of the gesture; the tables always are.
func applyRoom(ctx context.Context, store direction.Store, members []int64, room direction.Room, withBreaks bool) error {
	now := time.Now()
	for _, tid := range members {
		dir, err := direction.Open(ctx, store, tid)
		if errors.Is(err, direction.ErrNoDirection) {
			continue
		}
		if err != nil {
			return err
		}
		cfg, err := dir.Config()
		if err != nil {
			return err
		}
		if !withBreaks {
			room.Breaks = cfg.Breaks
		}
		if direction.SameRoom(cfg, room) {
			continue
		}
		if dir.State() != nil && dir.State().Finished {
			// A closed event plays at no table: the room moves on without it.
			continue
		}
		if err := dir.SetConfigAt(ctx, direction.WithRoom(cfg, room), now); err != nil {
			return fmt.Errorf("rencontre: tournament %d: %w", tid, err)
		}
	}
	return nil
}

// alignedConfig is the Tournament's configuration on the room's tables.
func (d *Service) alignedConfig(ctx context.Context, store direction.Store, tournamentID, rencontreID int64) (tournoi.Config, error) {
	r, err := d.st.Rencontres().Get(ctx, d.scope, rencontreID)
	if err != nil {
		return tournoi.Config{}, err
	}
	v, err := d.rencontreView(ctx, d.st, store, r)
	if err != nil {
		return tournoi.Config{}, err
	}
	dir, err := direction.Open(ctx, store, tournamentID)
	if err != nil {
		return tournoi.Config{}, err
	}
	cfg, err := dir.Config()
	if err != nil {
		return tournoi.Config{}, err
	}
	return direction.WithTables(cfg, v.Room), nil
}

func sameTables(a, b tournoi.Config) bool {
	return a.Tables.Count == b.Tables.Count && slices.Equal(a.Tables.Unavailable, b.Tables.Unavailable)
}

// sisterRoom is what the room looks like from one Tournament: the tables its sister events play
// on right now and, among its own Participants, those sat at one of their matches. Replayed from
// the sisters' logs at each call, never stored. A Tournament outside any Rencontre sees an empty
// room and proposes exactly as before.
type sisterRoom struct {
	// tables maps each table a sister event plays on to that event's name.
	tables map[int]string
	// players maps each Participant of this Tournament who plays next door to that seat.
	players map[tournoi.PlayerID]direction.Seat
	// plan is the Tournament's table properties and rooms; members the persons behind its
	// doubles Participants, whom a table may be kept for.
	plan    direction.TablePlan
	members map[string][]string
}

// external is the room as the engine takes it.
func (r sisterRoom) external() tournoi.External {
	busy := make([]int, 0, len(r.tables))
	for n := range r.tables {
		busy = append(busy, n)
	}
	slices.Sort(busy)
	return tournoi.External{BusyTables: busy, BusyPlayers: direction.BusyPlayers(r.players)}
}

// propose is what the engine proposes to dir in this room, under its table properties: the
// sisters' tables and the closed ones skipped, a kept table given to its holder's match.
func (r sisterRoom) propose(dir *direction.Direction, now time.Time) []tournoi.Action {
	return dir.ProposeIn(now, r.external(), r.plan, r.members)
}

// roomAround replays the sister events of a Tournament. me is its own replayed Direction, to
// find which of its Participants play next door and to read its table properties; nil when
// only the sisters' tables are wanted.
func (d *Service) roomAround(ctx context.Context, tournamentID int64, me *direction.Direction) sisterRoom {
	alone := sisterRoom{tables: map[int]string{}, players: map[tournoi.PlayerID]direction.Seat{}}
	if me != nil {
		cfg, _ := me.Config()
		alone.plan, _ = d.planFor(ctx, tournamentID, cfg)
		alone.members = d.memberNames(ctx, tournamentID)
	}
	rid, err := d.st.Rencontres().Of(ctx, d.scope, tournamentID)
	if err != nil || rid == 0 {
		return alone
	}
	r, err := d.st.Rencontres().Get(ctx, d.scope, rid)
	if err != nil {
		return alone
	}
	return d.roomFrom(ctx, r, tournamentID, me, d.openMembers(ctx, r, tournamentID))
}

// member is one event of a Rencontre, replayed once for every reader of the room. dir is nil
// and err says why when its log does not replay.
type member struct {
	tid  int64
	name string
	dir  *direction.Direction
	err  error
}

// openMembers replays the events of a Rencontre, in its order, except skip (0 skips none).
func (d *Service) openMembers(ctx context.Context, r *domain.Rencontre, skip int64) []member {
	out := make([]member, 0, len(r.TournamentIDs))
	for _, tid := range r.TournamentIDs {
		if tid == skip {
			continue
		}
		m := member{tid: tid, name: fmt.Sprintf("#%d", tid)}
		if t, err := d.st.Tournaments().Get(ctx, d.scope, tid); err == nil && t.Name != "" {
			m.name = t.Name
		}
		m.dir, m.err = direction.Open(ctx, d.dirStore(), tid)
		out = append(out, m)
	}
	return out
}

// roomFrom is the room around one Tournament, read from members already replayed: the tables
// and players of every other member that replays.
func (d *Service) roomFrom(ctx context.Context, r *domain.Rencontre, tournamentID int64, me *direction.Direction, members []member) sisterRoom {
	out := sisterRoom{tables: map[int]string{}, players: map[tournoi.PlayerID]direction.Seat{}}
	var sisters []direction.Sister
	for _, m := range members {
		if m.tid == tournamentID || m.dir == nil {
			continue
		}
		for _, n := range direction.BusyTables(m.dir) {
			out.tables[n] = m.name
		}
		if me != nil {
			sisters = append(sisters, direction.Sister{Name: m.name, Dir: m.dir, Members: d.memberNames(ctx, m.tid)})
		}
	}
	if me != nil {
		out.members = d.memberNames(ctx, tournamentID)
		out.players = me.BusyIn(direction.PlayingElsewhere(sisters...), out.members)
		cfg, _ := me.Config()
		out.plan = planIn(r, tournamentID, cfg)
	}
	return out
}

// memberNames gives the two persons behind each doubles Participant, by Participant id; empty
// for a singles event.
func (d *Service) memberNames(ctx context.Context, tournamentID int64) map[string][]string {
	pairs, err := d.Pairs(ctx, tournamentID)
	if err != nil || len(pairs) == 0 {
		return nil
	}
	out := make(map[string][]string, len(pairs))
	for id, ms := range pairs {
		for _, m := range ms {
			out[id] = append(out[id], m.Name)
		}
	}
	return out
}

// setMemberConfig saves a configuration for a Tournament of a Rencontre. When its room part
// changes — tables, tables out of service, breaks — the gesture is the room's: every member
// Direction records it, in the same transaction.
func (d *Service) setMemberConfig(ctx context.Context, rencontreID, tournamentID int64, cfg tournoi.Config) error {
	return d.inRoom(ctx, rencontreID, func(ctx context.Context, tx storage.Tx, store direction.Store, r *domain.Rencontre, _ direction.Room) error {
		me, err := direction.Open(ctx, store, tournamentID)
		if err != nil {
			return err
		}
		if err := me.SetConfigAt(ctx, cfg, time.Now()); err != nil {
			return err
		}
		room := direction.RoomOf(cfg)
		if room.Tables != r.Tables && room.Tables > 0 {
			r.Tables = room.Tables
			if err := tx.Rencontres().Update(ctx, d.scope, *r); err != nil {
				return err
			}
		}
		return applyRoom(ctx, store, r.TournamentIDs, room, true)
	})
}

// roomAlsoFor names the other events a configuration change will reach because it touches the
// room. Empty when the Tournament is in no Rencontre, or the room part is unchanged.
func (d *Service) roomAlsoFor(ctx context.Context, tournamentID int64, cur, next tournoi.Config) []string {
	if direction.SameRoom(cur, direction.RoomOf(next)) {
		return nil
	}
	rid, err := d.st.Rencontres().Of(ctx, d.scope, tournamentID)
	if err != nil || rid == 0 {
		return nil
	}
	r, err := d.st.Rencontres().Get(ctx, d.scope, rid)
	if err != nil {
		return nil
	}
	var names []string
	for _, tid := range r.TournamentIDs {
		if tid == tournamentID {
			continue
		}
		if t, err := d.st.Tournaments().Get(ctx, d.scope, tid); err == nil {
			names = append(names, t.Name)
		}
	}
	return names
}
