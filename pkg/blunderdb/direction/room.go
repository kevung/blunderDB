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
