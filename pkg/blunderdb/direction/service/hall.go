package service

import (
	"context"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// The Hall is the room of a Rencontre seen as one grid (ADR-0056 §5): one cell per physical
// table, whatever the event, each marked with its event, and the proposals of every event under
// it. It is replayed from the members' logs at every call and never stored (ADR-0047 §3).

// HallCell is one cell of the Hall: a TableCell plus the event it belongs to. A free or
// out-of-service table belongs to no event: its TournamentID is 0.
type HallCell struct {
	TableCell
	TournamentID int64  `json:"tournamentId,omitempty"`
	Event        string `json:"event,omitempty"`
	// EventIndex is the event's position in the Rencontre, what gives it its colour; -1 for a
	// table no event holds.
	EventIndex int `json:"eventIndex"`
}

// HallEvent is one event of the Rencontre as the Hall shows it: its name, its proposals, and
// the names those proposals need.
type HallEvent struct {
	TournamentID int64            `json:"tournamentId"`
	Name         string           `json:"name"`
	Index        int              `json:"index"`
	Proposals    []tournoi.Action `json:"proposals"`
	// Names gives each Participant's name by id: a proposal names players by id.
	Names map[string]string `json:"names"`
	// Error says why the event's log did not replay: its tables are missing from the Hall, the
	// other events' are still there.
	Error string `json:"error,omitempty"`
}

// HallView is the Hall of a Rencontre.
type HallView struct {
	RencontreID int64       `json:"rencontreId"`
	Name        string      `json:"name"`
	Events      []HallEvent `json:"events"`
	Cells       []HallCell  `json:"cells"`
}

// RencontreTableGrid merges the table grids of the Rencontre's events into the Hall's: one cell
// per table of the room, the match on it with its event, then the matches the room has no single
// table for — two events on one table (accepted and shown, never refused) and matches with no
// table.
func (d *Service) RencontreTableGrid(ctx context.Context, rencontreID int64) (*HallView, error) {
	r, err := d.st.Rencontres().Get(ctx, d.scope, rencontreID)
	if err != nil {
		return nil, err
	}
	return d.hallOf(ctx, r, d.openMembers(ctx, r, 0), true), nil
}

// hallOf builds the Hall from members replayed once; without proposals when only the tables are
// wanted (the wall page), which spares asking the engine for them.
func (d *Service) hallOf(ctx context.Context, r *domain.Rencontre, members []member, proposals bool) *HallView {
	v := &HallView{RencontreID: r.ID, Name: r.Name, Events: []HallEvent{}, Cells: []HallCell{}}
	type eventGrid struct {
		ev    HallEvent
		cells []TableCell
	}
	grids := make([]eventGrid, 0, len(members))
	highest := r.Tables
	now := time.Now()
	for i, m := range members {
		ev := HallEvent{TournamentID: m.tid, Name: m.name, Index: i, Proposals: []tournoi.Action{}, Names: map[string]string{}}
		if m.dir == nil {
			ev.Error = m.err.Error()
			v.Events = append(v.Events, ev)
			continue
		}
		room := d.roomFrom(ctx, m.tid, m.dir, members)
		cells := gridOf(m.dir, room, now)
		if st := m.dir.State(); proposals && st != nil {
			if p := m.dir.ProposeWith(now, room.external()); p != nil {
				ev.Proposals = p
			}
			for id, p := range st.Players {
				ev.Names[string(id)] = p.Name
			}
		}
		for _, c := range cells {
			if !c.NoTable && c.Table > highest {
				highest = c.Table
			}
		}
		v.Events = append(v.Events, ev)
		grids = append(grids, eventGrid{ev: ev, cells: cells})
	}

	hall := func(c TableCell, ev HallEvent) HallCell {
		// Seen from the Hall no table is "elsewhere": the sister event is on the same grid.
		c.Elsewhere = ""
		return HallCell{TableCell: c, TournamentID: ev.TournamentID, Event: ev.Name, EventIndex: ev.Index}
	}
	var extra, tableless []HallCell
	for n := 1; n <= highest; n++ {
		var on []HallCell
		free, unavailable, reserved := false, false, false
		for _, g := range grids {
			for _, c := range g.cells {
				switch {
				case c.NoTable || c.Table != n:
				case c.MatchID != "":
					on = append(on, hall(c, g.ev))
				case c.Unavailable:
					unavailable = true
				case c.Free:
					free = true
				case c.Reserved:
					reserved = true
				}
			}
		}
		switch {
		case len(on) > 0:
			if len(on) > 1 {
				for i := range on {
					on[i].Shared = true
				}
				extra = append(extra, on[1:]...)
			}
			v.Cells = append(v.Cells, on[0])
		case unavailable:
			v.Cells = append(v.Cells, HallCell{TableCell: TableCell{Table: n, Unavailable: true}, EventIndex: -1})
		case reserved && !free:
			// Reserved by every event that knows it: no one is placed there without a word.
			v.Cells = append(v.Cells, HallCell{TableCell: TableCell{Table: n, Reserved: true}, EventIndex: -1})
		default:
			// Free for one event is free in the Hall: the director places whom he wants.
			v.Cells = append(v.Cells, HallCell{TableCell: TableCell{Table: n, Free: true}, EventIndex: -1})
		}
	}
	for _, g := range grids {
		for _, c := range g.cells {
			if c.NoTable {
				tableless = append(tableless, hall(c, g.ev))
			}
		}
	}
	v.Cells = append(v.Cells, extra...)
	v.Cells = append(v.Cells, tableless...)
	return v
}
