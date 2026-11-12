package service

import (
	"context"

	tournoi "github.com/kevung/nicomaque"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
)

// PlayerStatusRow answers "am I playing?" for one entry: the engine's own status
// (tournoi.State.Statuses), codes only, with the match the player is at when one is running.
// A host words it in its own language.
type PlayerStatusRow struct {
	Player string             `json:"player"`
	Kind   tournoi.StatusKind `json:"kind"`
	// Phase is the phase the status speaks of: the current one (playing, bye), the next
	// (qualified, undecided), the last entered (eliminated, withdrawn, winner).
	Phase   int    `json:"phase"`
	Section string `json:"section,omitempty"`
	Round   int    `json:"round,omitempty"`
	// Label places the player's next match, as a code (tournoi.Label), never a wording.
	Label tournoi.Label `json:"label"`
	// Lives is what the player has left in the current phase, 0 where the format counts none.
	Lives int `json:"lives"`
	// MatchID, Table and Opponent name the running match of the player, when there is one.
	MatchID  string `json:"matchId,omitempty"`
	Table    int    `json:"table,omitempty"`
	Opponent string `json:"opponent,omitempty"`
}

// Statuses lists the status of every entry, in entry order, or of the entries named by ids, in
// the order given. An id that names no entry is left out: the engine would call it "not
// entered", which says of a stranger what is only true of a latecomer.
func (d *Service) Statuses(ctx context.Context, tournamentID int64, ids []string) ([]PlayerStatusRow, error) {
	dir, err := direction.Open(ctx, d.dirStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	st := dir.State()
	if st == nil {
		return nil, nil
	}
	running := map[tournoi.PlayerID]*tournoi.Match{}
	for _, m := range st.Running() {
		running[m.A], running[m.B] = m, m
	}
	ph := st.Phases[st.Current]
	row := func(s tournoi.PlayerStatus) PlayerStatusRow {
		r := PlayerStatusRow{Player: string(s.Player), Kind: s.Kind, Phase: s.Phase, Section: s.Section,
			Round: s.Round, Label: s.Label, Lives: max(ph.Lives[s.Player]-ph.Losses[s.Player], 0)}
		if m := running[s.Player]; m != nil {
			r.MatchID, r.Table, r.Opponent = string(m.ID), m.Table, string(m.B)
			if m.B == s.Player {
				r.Opponent = string(m.A)
			}
		}
		return r
	}
	out := []PlayerStatusRow{}
	if ids == nil {
		for _, s := range st.Statuses() {
			if s.Player != tournoi.BYE {
				out = append(out, row(s))
			}
		}
		return out, nil
	}
	for _, id := range ids {
		p := tournoi.PlayerID(id)
		if _, entered := st.Players[p]; entered && p != tournoi.BYE {
			out = append(out, row(st.StatusOf(p)))
		}
	}
	return out, nil
}
