package database

import (
	"context"
	"strconv"
	"strings"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
)

// The brackets of a Direction (tasks/nicomaque/fonctionnel.md §5.6).
//
// The engine renders an SVG for the display page, but the in-app view is the frontend's
// (translated, clickable); Go hands over the structure with untranslated codes.

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
	// Feeds lists the places of other matches whose winner (or loser) takes a seat here, so
	// the view can draw the lines of the graph without knowing the engine's indices.
	Feeds []BracketFeed `json:"feeds,omitempty"`
}

// BracketFeed links a seat of a match to the match it comes from.
type BracketFeed struct {
	// Side is the seat fed: 0 for A, 1 for B.
	Side int `json:"side"`
	// Section is the source match's section name; empty when it is the same graph.
	Section string `json:"section,omitempty"`
	Key     string `json:"key"`
	// Loser is true when the seat goes to the loser of the source match.
	Loser bool `json:"loser,omitempty"`
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
		sections := ph.Sections
		skeleton := false
		if len(sections) == 0 {
			sections, skeleton = bracketSkeleton(st, ph)
		}
		for _, sec := range sections {
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
				bm.Feeds = bracketFeeds(sections, sec, g)
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
				if skeleton {
					// An undrawn draw has no players, no byes, no results: only its shape.
					bm = BracketMatch{Key: bm.Key, Label: bm.Label, Round: bm.Round, Length: bm.Length, Feeds: bm.Feeds}
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

// bracketFeeds resolves the engine's index-based sources into match keys. A source that is a
// fixed player, or that points outside the graphs, draws no line.
func bracketFeeds(all []*tournoi.Section, sec *tournoi.Section, g tournoi.GMatch) []BracketFeed {
	var feeds []BracketFeed
	for side, src := range g.Src {
		if src.Player != "" {
			continue
		}
		from := sec
		if src.Section != "" && src.Section != sec.Name {
			found := false
			for _, o := range all {
				if o.Name == src.Section {
					from, found = o, true
					break
				}
			}
			if !found {
				continue
			}
		}
		if src.From < 0 || src.From >= len(from.Matches) {
			continue
		}
		f := BracketFeed{Side: side, Key: from.Matches[src.From].Key, Loser: src.Loser}
		if from.Name != sec.Name {
			f.Section = from.Name
		}
		feeds = append(feeds, f)
	}
	return feeds
}

// bracketSkeleton is the shape a not-yet-drawn elimination phase will take, asked of the
// engine itself: a throwaway tournament with as many anonymous players as will enter, drawn
// once. Nothing is returned when the number of entrants is not known before the draw (players
// coming from a Swiss phase that has not finished, lives that differ from one to the next),
// because a guessed size would be wrong rather than merely rough.
func bracketSkeleton(st *tournoi.State, ph *tournoi.PhaseState) ([]*tournoi.Section, bool) {
	cfg := ph.Cfg
	if ph.Drawn || (cfg.Kind != tournoi.KindBracket && cfg.Kind != tournoi.KindLivesBracket) {
		return nil, false
	}
	n := len(ph.Entrants)
	if n == 0 {
		active := 0
		for id := range st.Players {
			if !st.Withdrawn[id] {
				active++
			}
		}
		switch {
		case cfg.Entry == "all" || (ph.Index == 0 && (cfg.Entry == "" || cfg.Entry == "survivors")):
			n = active
		case strings.HasPrefix(cfg.Entry, "top:"):
			if k, err := strconv.Atoi(cfg.Entry[4:]); err == nil && k > 0 {
				n = min(k, active)
			}
		}
		// Players who arrive with the lives they have left cannot be sized in advance.
		if cfg.Kind == tournoi.KindLivesBracket && ph.Index != 0 {
			n = 0
		}
	}
	if n < 2 {
		return nil, false
	}
	cfg.Entry, cfg.Seeding = "", ""
	cfg.Name = ""
	sim, _, err := tournoi.New(tournoi.Config{Name: "skeleton", Phases: []tournoi.PhaseConfig{cfg}}, 1, time.Unix(0, 0))
	if err != nil {
		return nil, false
	}
	for i := 0; i < n; i++ {
		id := tournoi.PlayerID("sk" + strconv.Itoa(i))
		if sim.Apply(tournoi.Event{Version: tournoi.JournalVersion, Kind: tournoi.EvPlayerAdded, Player: &tournoi.Player{ID: id, Name: string(id)}}) != nil {
			return nil, false
		}
	}
	for _, a := range sim.Propose() {
		if a.Kind != tournoi.ActDraw || a.Draw == nil {
			continue
		}
		if sim.Apply(tournoi.Event{Version: tournoi.JournalVersion, Kind: tournoi.EvDraw, Phase: a.Phase, Section: a.Section, Draw: a.Draw}) != nil {
			return nil, false
		}
		if sim.Current >= 0 && sim.Current < len(sim.Phases) && len(sim.Phases[sim.Current].Sections) > 0 {
			return sim.Phases[sim.Current].Sections, true
		}
	}
	return nil, false
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

// wallBracket is the bracket a wall page shows for a Direction: its current phase when that is an
// elimination phase already drawn. A Swiss phase, a pool, a playoff or a phase not yet drawn has
// no tree to show, and nothing is returned.
func (d *Database) wallBracket(tournamentID int64, event string) *direction.WallBracket {
	phases, err := d.Brackets(tournamentID)
	if err != nil {
		return nil
	}
	for _, ph := range phases {
		if !ph.Current || !ph.Drawn {
			continue
		}
		br := direction.WallBracket{Event: event}
		for _, sec := range ph.Sections {
			if sec.Kind == "poule" || sec.Kind == "barrage" || sec.Rounds == 0 || len(sec.Matches) == 0 {
				continue
			}
			ws := direction.WallBracketSection{Name: sec.Name}
			for _, m := range sec.Matches {
				wm := direction.WallBracketMatch{
					Key: m.Key, Label: m.Label, Length: m.Length, Round: m.Round,
					AName: m.AName, BName: m.BName, ScoreA: m.ScoreA, ScoreB: m.ScoreB,
					Done: m.Done, Running: m.Running, Skipped: m.Skipped,
				}
				switch {
				case m.Winner == "":
				case m.Winner == m.A:
					wm.Winner = 1
				case m.Winner == m.B:
					wm.Winner = 2
				}
				for _, f := range m.Feeds {
					wm.Feeds = append(wm.Feeds, direction.WallBracketFeed{Side: f.Side, Section: f.Section, Key: f.Key, Loser: f.Loser})
				}
				ws.Matches = append(ws.Matches, wm)
			}
			br.Sections = append(br.Sections, ws)
		}
		if len(br.Sections) > 0 {
			return &br
		}
	}
	return nil
}
