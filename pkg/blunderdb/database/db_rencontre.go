package database

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
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
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
func (d *Database) CreateRencontre(name, startsOn, endsOn string, tables int) (*RencontreView, error) {
	if name == "" {
		return nil, fmt.Errorf("rencontre: a name is required")
	}
	if tables <= 0 {
		return nil, fmt.Errorf("rencontre: the room needs at least one table")
	}
	id, err := d.store.Rencontres().Create(context.Background(), "", domain.Rencontre{
		Name: name, StartsOn: startsOn, EndsOn: endsOn, Tables: tables,
	})
	if err != nil {
		return nil, err
	}
	return d.GetRencontre(id)
}

// ListRencontres returns every Rencontre, newest first.
func (d *Database) ListRencontres() ([]RencontreView, error) {
	ctx := context.Background()
	rs, err := d.store.Rencontres().List(ctx, "")
	if err != nil {
		return nil, err
	}
	out := make([]RencontreView, 0, len(rs))
	for _, r := range rs {
		v, err := d.rencontreView(ctx, d.store, d.DirectionStore(), r)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, nil
}

// GetRencontre returns one Rencontre with its room and members.
func (d *Database) GetRencontre(id int64) (*RencontreView, error) {
	ctx := context.Background()
	r, err := d.store.Rencontres().Get(ctx, "", id)
	if err != nil {
		return nil, err
	}
	return d.rencontreView(ctx, d.store, d.DirectionStore(), r)
}

// RencontreOf names the Rencontre a Tournament plays in, 0 when none.
func (d *Database) RencontreOf(tournamentID int64) (int64, error) {
	return d.store.Rencontres().Of(context.Background(), "", tournamentID)
}

func (d *Database) rencontreView(ctx context.Context, stores storage.Stores, store direction.Store, r *domain.Rencontre) (*RencontreView, error) {
	v := &RencontreView{Rencontre: *r, Room: direction.Room{Tables: r.Tables}, Members: []RencontreMember{}}
	roomRead := false
	for _, tid := range r.TournamentIDs {
		m := RencontreMember{TournamentID: tid}
		if t, err := stores.Tournaments().Get(ctx, "", tid); err == nil {
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
func (d *Database) UpdateRencontre(id int64, name, startsOn, endsOn string, tables int) (*RencontreView, error) {
	if name == "" || tables <= 0 {
		return nil, fmt.Errorf("rencontre: a name and at least one table are required")
	}
	err := d.inRoom(id, func(ctx context.Context, tx storage.Tx, store direction.Store, r *domain.Rencontre, room direction.Room) error {
		changed := r.Tables != tables
		r.Name, r.StartsOn, r.EndsOn, r.Tables = name, startsOn, endsOn, tables
		if err := tx.Rencontres().Update(ctx, "", *r); err != nil {
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
	return d.afterRoomGesture(id)
}

// PreviewAttachToRencontre shows what attaching changes in the Tournament's configuration — its
// tables become the room's — without writing anything.
func (d *Database) PreviewAttachToRencontre(tournamentID, rencontreID int64) (*ConfigPreview, error) {
	next, err := d.alignedConfig(context.Background(), d.DirectionStore(), tournamentID, rencontreID)
	if err != nil {
		return nil, err
	}
	blob, err := json.Marshal(next)
	if err != nil {
		return nil, err
	}
	return d.PreviewDirectionConfig(tournamentID, string(blob))
}

// AttachToRencontre puts a directed Tournament in the room: its tables are aligned by a
// configuration change, and membership is recorded, in one transaction. Permitted at any time,
// the event under way included. A Tournament without a Direction has no room to join.
func (d *Database) AttachToRencontre(tournamentID, rencontreID int64) (*RencontreView, error) {
	err := d.inRoom(rencontreID, func(ctx context.Context, tx storage.Tx, store direction.Store, r *domain.Rencontre, room direction.Room) error {
		if of, err := tx.Rencontres().Of(ctx, "", tournamentID); err != nil {
			return err
		} else if of != 0 && of != rencontreID {
			return fmt.Errorf("rencontre: tournament %d already plays in another Rencontre", tournamentID)
		}
		if err := alignTables(ctx, store, tournamentID, room); err != nil {
			return err
		}
		return tx.Rencontres().Attach(ctx, "", tournamentID, rencontreID)
	})
	if err != nil {
		return nil, err
	}
	return d.afterRoomGesture(rencontreID)
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

// realignRencontre puts every member of a Rencontre back on the room's tables, as attaching
// them does. A restored Rencontre needs it: its events kept their own tables while detached.
func (d *Database) realignRencontre(id int64) error {
	return d.inRoom(id, func(ctx context.Context, _ storage.Tx, store direction.Store, r *domain.Rencontre, room direction.Room) error {
		for _, tid := range r.TournamentIDs {
			if err := alignTables(ctx, store, tid, room); err != nil {
				return fmt.Errorf("rencontre: tournament %d: %w", tid, err)
			}
		}
		return nil
	})
}

// DetachFromRencontre takes a Tournament out of its room. It keeps its log and its tables.
func (d *Database) DetachFromRencontre(tournamentID int64) error {
	ctx := context.Background()
	rid, _ := d.store.Rencontres().Of(ctx, "", tournamentID)
	if err := d.store.Rencontres().Attach(ctx, "", tournamentID, 0); err != nil {
		return err
	}
	if rid != 0 {
		_, _ = d.WriteRencontrePage(rid)
	}
	return nil
}

// TrashRencontre deletes a Rencontre through the trash (ADR-0036). Its Tournaments are detached,
// never deleted.
func (d *Database) TrashRencontre(id int64) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return trash.Rencontre(context.Background(), d.store, "", id)
}

// SetRencontreTableOutOfService declares a table of the room out of service, or back in service.
// Declared once, recorded in every member Direction.
func (d *Database) SetRencontreTableOutOfService(id int64, table int, out bool) (*RencontreView, error) {
	err := d.inRoom(id, func(ctx context.Context, _ storage.Tx, store direction.Store, r *domain.Rencontre, room direction.Room) error {
		if table <= 0 || table > r.Tables {
			return fmt.Errorf("rencontre: the room has no table %d", table)
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
	return d.afterRoomGesture(id)
}

// SetRencontreBreaks replaces the room's breaks — a meal, the prize-giving — in every member
// Direction at once. breaksJSON is the engine's own list of {start, end}.
func (d *Database) SetRencontreBreaks(id int64, breaksJSON string) (*RencontreView, error) {
	var breaks []tournoi.TimeRange
	if breaksJSON != "" {
		if err := json.Unmarshal([]byte(breaksJSON), &breaks); err != nil {
			return nil, fmt.Errorf("rencontre: breaks: %w", err)
		}
	}
	err := d.inRoom(id, func(ctx context.Context, _ storage.Tx, store direction.Store, r *domain.Rencontre, room direction.Room) error {
		room.Breaks = breaks
		return applyRoom(ctx, store, r.TournamentIDs, room, true)
	})
	if err != nil {
		return nil, err
	}
	return d.afterRoomGesture(id)
}

// inRoom runs fn inside one SQL transaction, under the write lock, with the Rencontre and its room
// as the members' logs state it. Nothing fn writes survives an error.
func (d *Database) inRoom(id int64, fn func(context.Context, storage.Tx, direction.Store, *domain.Rencontre, direction.Room) error) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}
	ctx := context.Background()
	sqlTx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = sqlTx.Rollback() }()
	tx := sqlite.WrapTx(sqlTx)
	store := directionStore{d: d, tx: sqlTx}
	r, err := tx.Rencontres().Get(ctx, "", id)
	if err != nil {
		return err
	}
	v, err := d.rencontreView(ctx, tx, store, r)
	if err != nil {
		return err
	}
	if err := fn(ctx, tx, store, r, v.Room); err != nil {
		return err
	}
	return sqlTx.Commit()
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
func (d *Database) alignedConfig(ctx context.Context, store direction.Store, tournamentID, rencontreID int64) (tournoi.Config, error) {
	r, err := d.store.Rencontres().Get(ctx, "", rencontreID)
	if err != nil {
		return tournoi.Config{}, err
	}
	v, err := d.rencontreView(ctx, d.store, store, r)
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

// outside is the room as the engine takes it, for one Tournament.
func (d *Database) outside(ctx context.Context, tournamentID int64, me *direction.Direction) tournoi.External {
	return d.roomAround(ctx, tournamentID, me).external()
}

// roomAround replays the sister events of a Tournament. me is its own replayed Direction, to
// find which of its Participants play next door; nil when only the tables are wanted.
func (d *Database) roomAround(ctx context.Context, tournamentID int64, me *direction.Direction) sisterRoom {
	out := sisterRoom{tables: map[int]string{}, players: map[tournoi.PlayerID]direction.Seat{}}
	rid, err := d.store.Rencontres().Of(ctx, "", tournamentID)
	if err != nil || rid == 0 {
		return out
	}
	r, err := d.store.Rencontres().Get(ctx, "", rid)
	if err != nil {
		return out
	}
	var sisters []direction.Sister
	for _, tid := range r.TournamentIDs {
		if tid == tournamentID {
			continue
		}
		o, err := direction.Open(ctx, d.DirectionStore(), tid)
		if err != nil {
			continue
		}
		name := fmt.Sprintf("#%d", tid)
		if t, err := d.store.Tournaments().Get(ctx, "", tid); err == nil && t.Name != "" {
			name = t.Name
		}
		for _, n := range direction.BusyTables(o) {
			out.tables[n] = name
		}
		sisters = append(sisters, direction.Sister{Name: name, Dir: o, Members: d.memberNames(tid)})
	}
	if me != nil {
		out.players = me.BusyIn(direction.PlayingElsewhere(sisters...), d.memberNames(tournamentID))
	}
	return out
}

// memberNames gives the two persons behind each doubles Participant, by Participant id; empty
// for a singles event.
func (d *Database) memberNames(tournamentID int64) map[string][]string {
	pairs, err := d.Pairs(tournamentID)
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
func (d *Database) setMemberConfig(rencontreID, tournamentID int64, cfg tournoi.Config) error {
	return d.inRoom(rencontreID, func(ctx context.Context, tx storage.Tx, store direction.Store, r *domain.Rencontre, _ direction.Room) error {
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
			if err := tx.Rencontres().Update(ctx, "", *r); err != nil {
				return err
			}
		}
		return applyRoom(ctx, store, r.TournamentIDs, room, true)
	})
}

// roomAlsoFor names the other events a configuration change will reach because it touches the
// room. Empty when the Tournament is in no Rencontre, or the room part is unchanged.
func (d *Database) roomAlsoFor(ctx context.Context, tournamentID int64, cur, next tournoi.Config) []string {
	if direction.SameRoom(cur, direction.RoomOf(next)) {
		return nil
	}
	rid, err := d.store.Rencontres().Of(ctx, "", tournamentID)
	if err != nil || rid == 0 {
		return nil
	}
	r, err := d.store.Rencontres().Get(ctx, "", rid)
	if err != nil {
		return nil
	}
	var names []string
	for _, tid := range r.TournamentIDs {
		if tid == tournamentID {
			continue
		}
		if t, err := d.store.Tournaments().Get(ctx, "", tid); err == nil {
			names = append(names, t.Name)
		}
	}
	return names
}
