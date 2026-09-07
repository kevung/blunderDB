package database

import (
	"context"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
)

// The brackets of a Direction (ADR-0047 §5.6, issue #373).
//
// The engine draws them too — package render produces an SVG for the standalone display page —
// but the view INSIDE the application is drawn by the frontend, and that is a deliberate split:
// in here a bracket is translated in nine languages and clickable, and both of those live in
// the frontend. What Go hands over is the structure, with the engine's codes untranslated.

// BracketMatch is one place in a graph: who is expected there, who played, and what came of it.
type BracketMatch struct {
	Key   string        `json:"key"`
	Label tournoi.Label `json:"label"`
	// Round is the display row this match sits on, from the section's own rounds.
	Round  int    `json:"round"`
	Length int    `json:"length"`
	A      string `json:"a,omitempty"`
	B      string `json:"b,omitempty"`
	AName  string `json:"aName,omitempty"`
	BName  string `json:"bName,omitempty"`
	// MatchID is the launched match filling this place, empty while nobody has played it.
	MatchID string `json:"matchId,omitempty"`
	Winner  string `json:"winner,omitempty"`
	ScoreA  int    `json:"scoreA,omitempty"`
	ScoreB  int    `json:"scoreB,omitempty"`
	Done    bool   `json:"done"`
	Running bool   `json:"running"`
	// Walkover: a place resolved without being played, because the opponent is a bye or was
	// withdrawn. Skipped: a conditional match the tournament never needed (a reset final).
	Walkover bool `json:"walkover,omitempty"`
	Skipped  bool `json:"skipped,omitempty"`
	// Flagged marks a match the engine complains about, so the view can show it in place
	// rather than only in a list far from the bracket.
	Flagged bool `json:"flagged,omitempty"`
}

// BracketSection is one graph: the main draw, a consolation, a pool, a GSL block.
type BracketSection struct {
	Name string `json:"name"`
	// Kind is the engine's code for the family of graph — main, conso, last, gf, gsl, se,
	// poule, barrage — rendered by the frontend.
	Kind    string         `json:"kind"`
	Group   int            `json:"group,omitempty"`
	Block   int            `json:"block,omitempty"`
	Rounds  int            `json:"rounds"`
	Matches []BracketMatch `json:"matches"`
	// Players and Spots describe a playoff, which has no graph: its pairings are drawn as
	// they go, and what the view shows is who is still in and for how many places.
	Players []string `json:"players,omitempty"`
	Spots   int      `json:"spots,omitempty"`
}

// BracketPhase is one phase's worth of graphs, with what the view needs to title it.
type BracketPhase struct {
	Index    int                 `json:"index"`
	Kind     string              `json:"kind"`
	Name     string              `json:"name,omitempty"`
	Current  bool                `json:"current"`
	Drawn    bool                `json:"drawn"`
	Sections []BracketSection    `json:"sections"`
	Lives    []LivesRow          `json:"lives,omitempty"`
	Config   tournoi.PhaseConfig `json:"config"`
}

// LivesRow is one line of the lives board: a Swiss phase has no graph, and what a director
// reads there is who has how many lives left, whom they have met, and how long they have been
// waiting.
type LivesRow struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Lives     int      `json:"lives"`
	Wins      int      `json:"wins"`
	Losses    int      `json:"losses"`
	Byes      int      `json:"byes"`
	Opponents []string `json:"opponents,omitempty"`
	Playing   bool     `json:"playing"`
	Out       bool     `json:"out"`
}

// Brackets returns every phase's graphs, oldest first. Past phases are kept: a director looks
// back at the Swiss table while the bracket is running.
func (d *Database) Brackets(tournamentID int64) ([]BracketPhase, error) {
	dir, err := direction.Open(context.Background(), d.DirectionStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	st := dir.State()
	if st == nil {
		return nil, nil
	}
	flagged := map[tournoi.MatchID]bool{}
	for _, w := range st.Warnings {
		if w.Match != "" {
			flagged[w.Match] = true
		}
	}
	out := make([]BracketPhase, 0, len(st.Phases))
	for _, ph := range st.Phases {
		bp := BracketPhase{
			Index: ph.Index, Kind: ph.Cfg.Kind, Name: ph.Cfg.Name,
			Current: ph.Index == st.Current, Drawn: ph.Drawn, Config: ph.Cfg,
		}
		for _, sec := range ph.Sections {
			bs := BracketSection{
				Name: sec.Name, Kind: sec.Kind, Group: sec.Group, Block: sec.Block,
				Rounds: len(sec.Rounds), Spots: sec.Spots,
			}
			for _, p := range sec.Players {
				bs.Players = append(bs.Players, playerNameIn(st, p))
			}
			// The section's own rounds give each match its display row; a match in no round
			// (a conditional reset final) sits after the last one.
			row := map[string]int{}
			for r, idx := range sec.Rounds {
				for _, i := range idx {
					if i >= 0 && i < len(sec.Matches) {
						row[sec.Matches[i].Key] = r
					}
				}
			}
			for i := range sec.Matches {
				g := sec.Matches[i]
				bm := BracketMatch{
					Key: g.Key, Label: g.Label, Length: g.Length,
					A: string(g.Players[0]), B: string(g.Players[1]),
					AName: playerNameIn(st, g.Players[0]), BName: playerNameIn(st, g.Players[1]),
					MatchID: string(g.MatchID), Winner: string(g.Winner),
					Done: g.Done, Walkover: g.Walkover, Skipped: g.Skipped,
				}
				if r, ok := row[g.Key]; ok {
					bm.Round = r
				} else {
					bm.Round = len(sec.Rounds)
				}
				if m := st.Matches[g.MatchID]; m != nil {
					bm.ScoreA, bm.ScoreB = m.ScoreA, m.ScoreB
					bm.Running = m.Status == tournoi.Running
					bm.Flagged = flagged[m.ID]
				}
				bs.Matches = append(bs.Matches, bm)
			}
			bp.Sections = append(bp.Sections, bs)
		}
		// A Swiss or GSL phase has no graph before its switch: the lives board is its view.
		if ph.Cfg.Kind == tournoi.KindSwissLives || len(ph.Sections) == 0 {
			bp.Lives = livesRows(st, ph)
		}
		out = append(out, bp)
	}
	return out, nil
}

func livesRows(st *tournoi.State, ph *tournoi.PhaseState) []LivesRow {
	busy := map[tournoi.PlayerID]bool{}
	for _, m := range st.Running() {
		if m.Phase == ph.Index {
			busy[m.A], busy[m.B] = true, true
		}
	}
	rows := make([]LivesRow, 0, len(ph.Entrants))
	for _, id := range ph.Entrants {
		left := ph.Lives[id] - ph.Losses[id]
		if left < 0 {
			left = 0
		}
		r := LivesRow{
			ID: string(id), Name: playerNameIn(st, id),
			Lives: left, Wins: ph.Wins[id], Losses: ph.Losses[id], Byes: ph.Byes[id],
			Playing: busy[id], Out: left == 0 || st.Withdrawn[id],
		}
		for _, o := range ph.Opponents[id] {
			r.Opponents = append(r.Opponents, playerNameIn(st, o))
		}
		rows = append(rows, r)
	}
	return rows
}
