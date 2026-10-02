package service

import (
	"context"
	"fmt"
	"slices"
	"strings"

	tournoi "github.com/PileOfCells/backgammon-tournoi"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The properties of the tables (ADR-0058): name, room, reservation, persons a table is kept
// for. They belong to whoever owns the tables — the Rencontre for its events, the Tournament
// playing alone — and change only what is proposed and where a gesture may seat a match. None
// of them is written in a log.

// TablePlan is the effective table properties of a Tournament and the rooms it may play in:
// its Rencontre's when it plays in one, its own otherwise. The single place that decides it.
func (d *Service) TablePlan(ctx context.Context, tournamentID int64) (direction.TablePlan, error) {
	cfg := tournoi.Config{}
	if dir, err := direction.Open(ctx, d.dirStore(), tournamentID); err == nil {
		if c, err := dir.Config(); err == nil {
			cfg = c
		}
	}
	return d.planFor(ctx, tournamentID, cfg)
}

// planFor is TablePlan for a configuration already read.
func (d *Service) planFor(ctx context.Context, tournamentID int64, cfg tournoi.Config) (direction.TablePlan, error) {
	rid, err := d.st.Rencontres().Of(ctx, d.scope, tournamentID)
	if err != nil {
		return planAlone(cfg, nil), err
	}
	if rid != 0 {
		r, err := d.st.Rencontres().Get(ctx, d.scope, rid)
		if err != nil {
			return planAlone(cfg, nil), err
		}
		return planIn(r, tournamentID, cfg), nil
	}
	own, err := d.st.Rencontres().TournamentTableSettings(ctx, d.scope, tournamentID)
	return planAlone(cfg, own), err
}

// planAlone is the plan of a Tournament playing on its own tables: no room restricts it.
func planAlone(cfg tournoi.Config, settings []domain.TableSetting) direction.TablePlan {
	return direction.TablePlan{
		Count: cfg.Tables.Count, Unavailable: slices.Clone(cfg.Tables.Unavailable),
		Settings: nonNilSettings(settings), Rooms: []string{},
	}
}

// planIn is the plan of a member of r: the Rencontre's tables and the member's rooms.
func planIn(r *domain.Rencontre, tournamentID int64, cfg tournoi.Config) direction.TablePlan {
	p := planAlone(cfg, r.TableSettings)
	if r.Tables > 0 {
		p.Count = r.Tables
	}
	if rooms := r.EventRooms[tournamentID]; len(rooms) > 0 {
		p.Rooms = slices.Clone(rooms)
	}
	return p
}

func nonNilSettings(s []domain.TableSetting) []domain.TableSetting {
	if s == nil {
		return []domain.TableSetting{}
	}
	return s
}

// checkSettings refuses a list the store would refuse, in the director's words, and a table
// past the last one.
func checkSettings(settings []domain.TableSetting, count int) error {
	seen := map[int]bool{}
	for _, s := range settings {
		switch {
		case s.Number <= 0:
			return direction.Refusef("tables: %d is not a table", s.Number)
		case count > 0 && s.Number > count:
			return direction.Refusef("tables: there is no table %d, the last is %d", s.Number, count)
		case seen[s.Number]:
			return direction.Refusef("tables: table %d is given twice", s.Number)
		}
		seen[s.Number] = true
	}
	return nil
}

// checkRunningStays refuses a change of rooms that would leave a running match on a table its
// event may no longer play at (ADR-0058 §11), naming that table. before and after give a
// member's plan on either side of the change; members whose plan does not move are skipped.
func checkRunningStays(ctx context.Context, store direction.Store, members []member, before, after func(tid int64) direction.TablePlan) error {
	for _, m := range members {
		dir, err := direction.Open(ctx, store, m.tid)
		if err != nil || dir.State() == nil {
			continue
		}
		was, will := before(m.tid), after(m.tid)
		for _, match := range dir.State().Running() {
			if match.Table > 0 && was.Allowed(match.Table) && !will.Allowed(match.Table) {
				return direction.Refusef("tables: table %d holds a running match of %s, which would no longer be in its rooms", match.Table, m.name)
			}
		}
	}
	return nil
}

// checkEveryEventHasATable refuses table properties under which an event restricted to rooms
// would find no table in them — its rooms renamed or emptied — naming the event: its matches
// would wait for a table forever.
func checkEveryEventHasATable(r *domain.Rencontre, members []member) error {
	for _, m := range members {
		p := planIn(r, m.tid, tournoi.Config{})
		if len(p.Rooms) == 0 || len(direction.RoomNames(p.Settings)) == 0 {
			continue
		}
		found := false
		for _, s := range p.Settings {
			if (p.Count == 0 || s.Number <= p.Count) && p.Allowed(s.Number) {
				found = true
				break
			}
		}
		if !found {
			return direction.Refusef("tables: %s plays only in %s, where no table would be left", m.name, strings.Join(p.Rooms, ", "))
		}
	}
	return nil
}

// SetRencontreTables replaces the table properties of a Rencontre. Renaming, reserving or
// keeping a table for someone is permitted at any time; moving a table to another room is
// refused while it holds a running match of an event that may no longer play there.
func (d *Service) SetRencontreTables(ctx context.Context, id int64, settings []domain.TableSetting) (*RencontreView, error) {
	settings = nonNilSettings(settings)
	err := d.lockedRoom(ctx, id, func(ctx context.Context, tx storage.Tx, store direction.Store, r *domain.Rencontre, _ direction.Room) error {
		if err := checkSettings(settings, r.Tables); err != nil {
			return err
		}
		next := *r
		next.TableSettings = settings
		members := d.membersOf(ctx, tx, r)
		if err := checkRunningStays(ctx, store, members,
			func(tid int64) direction.TablePlan { return planIn(r, tid, tournoi.Config{}) },
			func(tid int64) direction.TablePlan { return planIn(&next, tid, tournoi.Config{}) }); err != nil {
			return err
		}
		if err := checkEveryEventHasATable(&next, members); err != nil {
			return err
		}
		return tx.Rencontres().SetTableSettings(ctx, d.scope, id, settings)
	})
	if err != nil {
		return nil, err
	}
	return d.afterRoomGesture(ctx, id)
}

// SetEventRooms sets the rooms an event of the Rencontre may play in; none means every table.
// Each room must be carried by a table. Taking away a room that holds one of the event's
// running matches is refused.
func (d *Service) SetEventRooms(ctx context.Context, rencontreID, tournamentID int64, rooms []string) (*RencontreView, error) {
	var clean []string
	for _, room := range rooms {
		if room != "" && !slices.Contains(clean, room) {
			clean = append(clean, room)
		}
	}
	err := d.lockedRoom(ctx, rencontreID, func(ctx context.Context, tx storage.Tx, store direction.Store, r *domain.Rencontre, _ direction.Room) error {
		if !slices.Contains(r.TournamentIDs, tournamentID) {
			return direction.Refusef("rencontre: tournament %d does not play in this Rencontre", tournamentID)
		}
		known := direction.RoomNames(r.TableSettings)
		for _, room := range clean {
			if !slices.Contains(known, room) {
				return direction.Refusef("rencontre: no table is in room %q", room)
			}
		}
		next := *r
		next.EventRooms = map[int64][]string{tournamentID: clean}
		var only []member
		for _, m := range d.membersOf(ctx, tx, r) {
			if m.tid == tournamentID {
				only = append(only, m)
			}
		}
		if err := checkRunningStays(ctx, store, only,
			func(tid int64) direction.TablePlan { return planIn(r, tid, tournoi.Config{}) },
			func(tid int64) direction.TablePlan { return planIn(&next, tid, tournoi.Config{}) }); err != nil {
			return err
		}
		return tx.Rencontres().SetEventRooms(ctx, d.scope, tournamentID, clean)
	})
	if err != nil {
		return nil, err
	}
	return d.afterRoomGesture(ctx, rencontreID)
}

// SetDirectionTables replaces the table properties of a Tournament that plays alone. One that
// plays in a Rencontre is refused: its tables are the Rencontre's, set there, and properties
// written here would stay invisible until it is detached (ADR-0058 §3).
func (d *Service) SetDirectionTables(ctx context.Context, tournamentID int64, settings []domain.TableSetting) (*DirectionView, error) {
	if err := d.setDirectionTables(ctx, tournamentID, nonNilSettings(settings)); err != nil {
		return nil, err
	}
	return d.viewAfter(ctx, tournamentID)
}

func (d *Service) setDirectionTables(ctx context.Context, tournamentID int64, settings []domain.TableSetting) (err error) {
	d, release, shared, err := d.lockTables(ctx, tournamentID)
	if err != nil {
		return err
	}
	defer release(&err)
	if rid, err := d.st.Rencontres().Of(ctx, d.scope, tournamentID); err != nil {
		return err
	} else if shared || rid != 0 {
		return direction.Refusef("direction: tournament %d plays in a Rencontre: its tables are set on the Rencontre", tournamentID)
	}
	dir, err := direction.Open(ctx, d.dirStore(), tournamentID)
	if err != nil {
		return err
	}
	cfg, err := dir.Config()
	if err != nil {
		return err
	}
	if err := checkSettings(settings, cfg.Tables.Count); err != nil {
		return err
	}
	return d.st.Rencontres().SetTournamentTableSettings(ctx, d.scope, tournamentID, settings)
}

// membersOf names the members of r, read through the transaction's stores.
func (d *Service) membersOf(ctx context.Context, tx storage.Stores, r *domain.Rencontre) []member {
	out := make([]member, 0, len(r.TournamentIDs))
	for _, tid := range r.TournamentIDs {
		m := member{tid: tid, name: fmt.Sprintf("#%d", tid)}
		if t, err := tx.Tournaments().Get(ctx, d.scope, tid); err == nil && t.Name != "" {
			m.name = t.Name
		}
		out = append(out, m)
	}
	return out
}

// tableRefused refuses seating a match of the event at a table outside its rooms (ADR-0058
// §10). A reserved table, or one kept for someone, accepts an explicit gesture.
func tableRefused(p direction.TablePlan, table int, event string) error {
	if table > 0 && !p.Allowed(table) {
		s, _ := p.Setting(table)
		if s.Room == "" {
			return direction.Refusef("direction: table %d is in no room of %s", table, event)
		}
		return direction.Refusef("direction: table %d is in room %q, where %s does not play", table, s.Room, event)
	}
	return nil
}

// refuseOutside refuses seating a match of the Direction dir at a table outside its rooms.
func (d *Service) refuseOutside(ctx context.Context, dir *direction.Direction, tournamentID int64, table int) error {
	cfg, err := dir.Config()
	if err != nil {
		return err
	}
	p, err := d.planFor(ctx, tournamentID, cfg)
	if err != nil {
		return err
	}
	name := fmt.Sprintf("tournament %d", tournamentID)
	if t, err := d.st.Tournaments().Get(ctx, d.scope, tournamentID); err == nil && t.Name != "" {
		name = t.Name
	}
	return tableRefused(p, table, name)
}

// refuseClosed refuses confirming a proposal on a table the engine is no longer given — one
// reserved, or kept for someone, since the proposal was shown — unless the table is kept for a
// player of that match. A confirmation is not a gesture of the director's: it seats what was
// proposed, and a proposal never lands on a closed table (ADR-0058 §6, §7).
func (d *Service) refuseClosed(ctx context.Context, dir *direction.Direction, tournamentID int64, a tournoi.Action) error {
	cfg, err := dir.Config()
	if err != nil {
		return err
	}
	p, err := d.planFor(ctx, tournamentID, cfg)
	if err != nil {
		return err
	}
	if !slices.Contains(p.Closed(), a.Table) || dir.KeptFor(p, a.Table, a.A, a.B, d.memberNames(ctx, tournamentID)) {
		return nil
	}
	if s, _ := p.Setting(a.Table); len(s.AssignedTo) > 0 && !s.Reserved {
		return direction.Refusef("direction: table %d is kept for %s", a.Table, strings.Join(s.AssignedTo, ", "))
	}
	return direction.Refusef("direction: table %d is reserved", a.Table)
}

// closedTables are the tables a match seated without a number must not land on: those the
// engine is never given for this Direction (ADR-0058 §6).
func (d *Service) closedTables(ctx context.Context, dir *direction.Direction, tournamentID int64) (map[int]string, error) {
	cfg, err := dir.Config()
	if err != nil {
		return nil, err
	}
	p, err := d.planFor(ctx, tournamentID, cfg)
	if err != nil {
		return nil, err
	}
	out := map[int]string{}
	for _, n := range p.Closed() {
		out[n] = ""
	}
	return out, nil
}
