package direction

import (
	"slices"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// The properties of the tables (ADR-0058): a name, a room, a reservation, persons the table is
// kept for. The engine knows none of them. They change only what the host proposes: before the
// engine is asked, the tables an event may not be given are added to the busy tables; after,
// the match of a person a table is kept for is moved onto it when it is free. Nothing here is
// written in a log.

// TablePlan is what the table properties say to one event: the tables of its room, their
// properties as its owner — the Rencontre, or the Tournament playing alone — holds them, and
// the rooms the event may play in.
type TablePlan struct {
	// Count is the number of tables; 0 bounds nothing.
	Count int `json:"count"`
	// Unavailable are the tables out of service.
	Unavailable []int                 `json:"unavailable,omitempty"`
	Settings    []domain.TableSetting `json:"settings"`
	// Rooms are the rooms the event may play in; empty means every table.
	Rooms []string `json:"rooms"`
}

// Setting returns the properties of table n, if it has any.
func (p TablePlan) Setting(n int) (domain.TableSetting, bool) {
	for _, s := range p.Settings {
		if s.Number == n {
			return s, true
		}
	}
	return domain.TableSetting{}, false
}

// RoomNames are the rooms of the plan in the order of their first table; empty when no table
// carries a room, the single implicit room of ADR-0058 §2.
func RoomNames(settings []domain.TableSetting) []string {
	sorted := slices.Clone(settings)
	slices.SortFunc(sorted, func(a, b domain.TableSetting) int { return a.Number - b.Number })
	var out []string
	for _, s := range sorted {
		if s.Room != "" && !slices.Contains(out, s.Room) {
			out = append(out, s.Room)
		}
	}
	return out
}

// Allowed says whether the event may play at table n: every table when it is restricted to no
// room or when no table carries a room, otherwise only the tables of its rooms.
func (p TablePlan) Allowed(n int) bool {
	if len(p.Rooms) == 0 || len(RoomNames(p.Settings)) == 0 {
		return true
	}
	s, ok := p.Setting(n)
	return ok && slices.Contains(p.Rooms, s.Room)
}

// Closed are the tables the engine must never propose to this event, sorted: those outside its
// rooms, the reserved ones and the ones kept for someone — a table kept for a person is given
// to that person's match afterwards (AssignKept), never by the engine.
func (p TablePlan) Closed() []int {
	var out []int
	for _, s := range p.Settings {
		if s.Number > 0 && (p.Count == 0 || s.Number <= p.Count) && (s.Reserved || len(s.AssignedTo) > 0) {
			out = append(out, s.Number)
		}
	}
	if len(p.Rooms) > 0 && len(RoomNames(p.Settings)) > 0 {
		for n := 1; n <= p.Count; n++ {
			if !p.Allowed(n) {
				out = append(out, n)
			}
		}
	}
	return sortedUnique(out)
}

// AssignKept gives each proposed match whose players include a person a table is kept for
// that table, when it is free: not in taken (running here or next door, out of service), in
// the event's rooms, open to the match's section and phase in cfg, and given to no other
// proposal of the batch. Two such persons meeting play on the smaller of their free tables. A
// proposal waiting for a table that finds one this way stops waiting; one that finds none keeps
// what the engine gave it (ADR-0058 §8, §9), and the table a holder's match leaves goes to a
// proposal of the batch still waiting for one. persons names the people behind a Participant.
func AssignKept(acts []tournoi.Action, p TablePlan, tables tournoi.Tables, taken map[int]bool, persons func(tournoi.PlayerID) []string) []tournoi.Action {
	kept := map[string][]int{}
	for _, s := range p.Settings {
		for _, name := range s.AssignedTo {
			kept[name] = append(kept[name], s.Number)
		}
	}
	if len(kept) == 0 {
		return acts
	}
	used := map[int]bool{}
	for _, a := range acts {
		if a.Kind == tournoi.ActStartMatch && a.Table > 0 {
			used[a.Table] = true
		}
	}
	out := slices.Clone(acts)
	var freed []int
	for i, a := range out {
		if a.Kind != tournoi.ActStartMatch || (a.Reason != tournoi.ReasonNone && a.Reason != tournoi.ReasonWaitingTable) {
			continue
		}
		var mine []int
		for _, id := range []tournoi.PlayerID{a.A, a.B} {
			for _, name := range persons(id) {
				mine = append(mine, kept[name]...)
			}
		}
		slices.Sort(mine)
		for _, n := range slices.Compact(mine) {
			if n == a.Table {
				break // already there
			}
			if used[n] || taken[n] || (p.Count > 0 && n > p.Count) || !p.Allowed(n) ||
				slices.Contains(p.Unavailable, n) || !tables.AvailableFor(n, a.Section, a.Phase) {
				continue
			}
			if a.Table > 0 {
				delete(used, a.Table)
				freed = append(freed, a.Table)
			}
			used[n] = true
			out[i].Table = n
			if a.Reason == tournoi.ReasonWaitingTable {
				out[i].Reason = tournoi.ReasonNone
			}
			break
		}
	}
	// A table a holder's match left is free for the batch: the first proposal still waiting
	// that may play there takes it, rather than wait for a table nobody uses.
	closed := p.Closed()
	slices.Sort(freed)
	for _, n := range freed {
		if used[n] || slices.Contains(closed, n) || !p.Allowed(n) {
			continue
		}
		for i, a := range out {
			if a.Kind != tournoi.ActStartMatch || a.Reason != tournoi.ReasonWaitingTable || a.Table != 0 ||
				!tables.AvailableFor(n, a.Section, a.Phase) {
				continue
			}
			used[n] = true
			out[i].Table, out[i].Reason = n, tournoi.ReasonNone
			break
		}
	}
	return out
}

// ProposeIn is ProposeWith under the table properties: the engine is told the closed tables
// are busy, and the matches of the persons a table is kept for are moved onto it. ext carries
// what the sister events hold; members names the persons behind each doubles Participant.
func (d *Direction) ProposeIn(now time.Time, ext tournoi.External, p TablePlan, members map[string][]string) []tournoi.Action {
	if d.st == nil {
		return nil
	}
	taken := map[int]bool{}
	for _, n := range ext.BusyTables {
		taken[n] = true
	}
	for _, m := range d.st.Running() {
		if m.Table > 0 {
			taken[m.Table] = true
		}
	}
	ext.BusyTables = sortedUnique(append(slices.Clone(ext.BusyTables), p.Closed()...))
	acts := d.st.ProposeWith(now, ext)
	return AssignKept(acts, p, d.st.Config.Tables, taken, func(id tournoi.PlayerID) []string {
		return persons(d.st, id, members)
	})
}

// KeptFor says whether table n is kept, in p, for a person playing as a or b — whose match the
// table is then given to (ADR-0058 §8).
func (d *Direction) KeptFor(p TablePlan, n int, a, b tournoi.PlayerID, members map[string][]string) bool {
	s, ok := p.Setting(n)
	if !ok || d.st == nil {
		return false
	}
	for _, id := range []tournoi.PlayerID{a, b} {
		for _, name := range persons(d.st, id, members) {
			if slices.Contains(s.AssignedTo, name) {
				return true
			}
		}
	}
	return false
}
