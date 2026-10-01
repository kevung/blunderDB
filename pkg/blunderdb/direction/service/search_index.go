package service

import (
	"context"
	"fmt"
	"slices"

	tournoi "github.com/PileOfCells/backgammon-tournoi"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
)

// Kinds of a SearchEntry.
const (
	SearchEpreuve = "epreuve"
	SearchPlayer  = "player"
	SearchTable   = "table"
	SearchMatch   = "match"
)

// SearchEntry is one thing a director may look for in the room: an event, a player, a table or a
// running match. The index is rebuilt from the replayed logs at each call, never stored, so it
// cannot disagree with the tables the director sees.
type SearchEntry struct {
	Kind string `json:"kind"`
	// TournamentID is the event the entry belongs to; 0 for a table nobody sits at.
	TournamentID int64  `json:"tournamentId"`
	Epreuve      string `json:"epreuve,omitempty"`
	// Name is the player, the event, or "A – B" for a match.
	Name string `json:"name"`
	// PlayerID and Club describe a player; Opponent, who they are playing right now.
	PlayerID string `json:"playerId,omitempty"`
	Club     string `json:"club,omitempty"`
	Opponent string `json:"opponent,omitempty"`
	// State is a CODE: for a player the Participants state (playing, free, withdrawn, leaving,
	// absent), for a table running, free or unavailable.
	State string `json:"state,omitempty"`
	Table int    `json:"table,omitempty"`
	// MatchID names the running match of a "match" entry.
	MatchID string `json:"matchId,omitempty"`
}

// RencontreSearchIndex lists what a director may look for in a Rencontre: its events, the
// players of every event (those at a table first), its tables whichever event occupies them, and
// the matches running. One call, so the front does not replay each event's participants itself.
func (d *Service) RencontreSearchIndex(ctx context.Context, id int64) ([]SearchEntry, error) {
	r, err := d.st.Rencontres().Get(ctx, d.scope, id)
	if err != nil {
		return nil, err
	}
	return d.searchIndex(ctx, r.TournamentIDs, r.Tables, nil)
}

// DirectionSearchIndex is the same index for the Tournament a director has open: its room when it
// plays in a Rencontre, itself alone otherwise.
func (d *Service) DirectionSearchIndex(ctx context.Context, tournamentID int64) ([]SearchEntry, error) {
	if rid, err := d.RencontreOf(ctx, tournamentID); err == nil && rid != 0 {
		return d.RencontreSearchIndex(ctx, rid)
	}
	dir, err := direction.Open(ctx, d.dirStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	cfg, err := dir.Config()
	if err != nil {
		return nil, err
	}
	room := direction.RoomOf(cfg)
	return d.searchIndex(ctx, []int64{tournamentID}, room.Tables, room.Unavailable)
}

func (d *Service) searchIndex(ctx context.Context, members []int64, tables int, unavailable []int) ([]SearchEntry, error) {
	var epreuves, playing, others, matches []SearchEntry
	occupied := map[int]SearchEntry{}
	for _, tid := range members {
		name := fmt.Sprintf("#%d", tid)
		if t, err := d.st.Tournaments().Get(ctx, d.scope, tid); err == nil && t.Name != "" {
			name = t.Name
		}
		epreuves = append(epreuves, SearchEntry{Kind: SearchEpreuve, TournamentID: tid, Epreuve: name, Name: name})
		dir, err := direction.Open(ctx, d.dirStore(), tid)
		if err != nil {
			continue
		}
		if cfg, err := dir.Config(); err == nil && unavailable == nil {
			unavailable = direction.RoomOf(cfg).Unavailable
		}
		st := dir.State()
		if st == nil {
			continue
		}
		running := map[tournoi.PlayerID]*tournoi.Match{}
		for _, m := range st.Running() {
			running[m.A], running[m.B] = m, m
			if m.Table <= 0 {
				continue
			}
			a, b := wallPlayerName(st, m.A), wallPlayerName(st, m.B)
			e := SearchEntry{Kind: SearchMatch, TournamentID: tid, Epreuve: name, Name: a + " – " + b, Table: m.Table, MatchID: string(m.ID), State: "running"}
			matches = append(matches, e)
			occupied[m.Table] = e
		}
		rows, err := d.Participants(ctx, tid)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			e := SearchEntry{Kind: SearchPlayer, TournamentID: tid, Epreuve: name, Name: row.Name, PlayerID: row.ID, Club: row.Club, State: row.State, Table: row.Table}
			if m := running[tournoi.PlayerID(row.ID)]; m != nil {
				other := m.A
				if other == tournoi.PlayerID(row.ID) {
					other = m.B
				}
				e.Opponent = wallPlayerName(st, other)
			}
			if row.State == "playing" {
				playing = append(playing, e)
			} else {
				others = append(others, e)
			}
		}
	}
	slots := make([]SearchEntry, 0, tables)
	for t := 1; t <= tables; t++ {
		if o, ok := occupied[t]; ok {
			o.Kind = SearchTable
			o.MatchID = ""
			slots = append(slots, o)
			continue
		}
		state := "free"
		if slices.Contains(unavailable, t) {
			state = "unavailable"
		}
		slots = append(slots, SearchEntry{Kind: SearchTable, Table: t, State: state})
	}
	out := make([]SearchEntry, 0, len(epreuves)+len(playing)+len(others)+len(slots)+len(matches))
	out = append(out, epreuves...)
	out = append(out, playing...)
	out = append(out, others...)
	out = append(out, slots...)
	out = append(out, matches...)
	return out, nil
}
