package direction

import (
	"slices"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
)

// The room several Directions share when their Tournaments play in one Rencontre (ADR-0056).
//
// Each Direction's log stays complete on its own: a gesture on the room — a table out of
// service, a break — is written as one configuration change in EVERY member log, and the tables
// the sister events occupy are never stored anywhere. They are replayed from the sisters' logs at
// each proposal and handed to the engine as tournoi.External, which is not an event: it changes
// with every match next door and is not part of this tournament's history.

// Room is the part of a configuration that belongs to the room rather than to one event: how many
// tables, which are out of service, and the breaks. Reserved tables (a streamed table, a table of
// honour) stay the event's own.
type Room struct {
	Tables      int                 `json:"tables"`
	Unavailable []int               `json:"unavailable"`
	Breaks      []tournoi.TimeRange `json:"breaks"`
}

// RoomOf reads the room out of a configuration.
func RoomOf(cfg tournoi.Config) Room {
	return Room{
		Tables:      cfg.Tables.Count,
		Unavailable: slices.Clone(cfg.Tables.Unavailable),
		Breaks:      slices.Clone(cfg.Breaks),
	}
}

// WithTables returns cfg on the room's tables — count and out of service — keeping its own
// reservations and breaks. It is what attaching a Tournament to a Rencontre aligns.
func WithTables(cfg tournoi.Config, r Room) tournoi.Config {
	cfg.Tables.Count = r.Tables
	cfg.Tables.Unavailable = sortedUnique(r.Unavailable)
	return cfg
}

// WithRoom returns cfg on the whole room: tables and breaks.
func WithRoom(cfg tournoi.Config, r Room) tournoi.Config {
	cfg = WithTables(cfg, r)
	cfg.Breaks = slices.Clone(r.Breaks)
	return cfg
}

// SameRoom says whether a configuration already sits in the room, so a gesture that changes
// nothing for a member writes nothing in its log.
func SameRoom(cfg tournoi.Config, r Room) bool {
	cur := RoomOf(cfg)
	return cur.Tables == r.Tables &&
		slices.Equal(sortedUnique(cur.Unavailable), sortedUnique(r.Unavailable)) &&
		slices.EqualFunc(cur.Breaks, r.Breaks, func(a, b tournoi.TimeRange) bool {
			return a.Start.Equal(b.Start) && a.End.Equal(b.End)
		})
}

// BusyTables are the tables where the other Directions have a match running now, sorted. They
// are what the engine must skip when it gives a table to a proposal of this one.
func BusyTables(others ...*Direction) []int {
	var busy []int
	for _, o := range others {
		if o == nil || o.st == nil {
			continue
		}
		for _, m := range o.st.Running() {
			if m.Table > 0 {
				busy = append(busy, m.Table)
			}
		}
	}
	return sortedUnique(busy)
}

// Seat is where a person plays right now in a sister event of the room.
type Seat struct {
	Event string `json:"event"`
	Table int    `json:"table"`
}

// Sister is a sister event as the room sees it: its name, its replayed Direction and, for a
// doubles event, the persons behind each Participant (by Participant id).
type Sister struct {
	Name    string
	Dir     *Direction
	Members map[string][]string
}

// persons are the people a Participant stands for: its two members for a pair, its own name
// otherwise. The name is the link between events (ADR-0056 §3), compared exactly.
func persons(st *tournoi.State, id tournoi.PlayerID, members map[string][]string) []string {
	if m := members[string(id)]; len(m) > 0 {
		return m
	}
	if p := st.Players[id]; p != nil && p.Name != "" {
		return []string{p.Name}
	}
	return nil
}

// PlayingElsewhere maps every person at a running match of the sister events to where they
// play. Replayed from the sisters' logs at each call, never stored.
func PlayingElsewhere(sisters ...Sister) map[string]Seat {
	out := map[string]Seat{}
	for _, s := range sisters {
		if s.Dir == nil || s.Dir.st == nil {
			continue
		}
		for _, m := range s.Dir.st.Running() {
			for _, id := range []tournoi.PlayerID{m.A, m.B} {
				for _, name := range persons(s.Dir.st, id, s.Members) {
					out[name] = Seat{Event: s.Name, Table: m.Table}
				}
			}
		}
	}
	return out
}

// BusyIn names the Participants of d that someone in elsewhere stands for — a pair as soon as
// one of its members plays next door — with the seat that holds them. These are the
// BusyPlayers the engine must not pair, and what the waiting list says about each.
func (d *Direction) BusyIn(elsewhere map[string]Seat, members map[string][]string) map[tournoi.PlayerID]Seat {
	out := map[tournoi.PlayerID]Seat{}
	if d.st == nil || len(elsewhere) == 0 {
		return out
	}
	for _, id := range d.st.Order {
		for _, name := range persons(d.st, id, members) {
			if seat, ok := elsewhere[name]; ok {
				out[id] = seat
				break
			}
		}
	}
	return out
}

// BusyPlayers is the sorted list of the ids in busy, as tournoi.External carries them.
func BusyPlayers(busy map[tournoi.PlayerID]Seat) []tournoi.PlayerID {
	out := make([]tournoi.PlayerID, 0, len(busy))
	for id := range busy {
		out = append(out, id)
	}
	slices.Sort(out)
	return out
}

// ProposeWith asks the engine what to do at a given instant, knowing what the room outside this
// tournament looks like. A proposal that finds no table left waits (ReasonWaitingTable) instead
// of landing on a table a sister event is playing on.
func (d *Direction) ProposeWith(now time.Time, ext tournoi.External) []tournoi.Action {
	if d.st == nil {
		return nil
	}
	return d.st.ProposeWith(now, ext)
}

func sortedUnique(v []int) []int {
	out := slices.Clone(v)
	slices.Sort(out)
	return slices.Compact(out)
}
