package direction

import (
	"cmp"
	"fmt"
	"html"
	"slices"
	"strings"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
)

// What the wall answers to "am I playing?" (ADR-0047, the players' screen): a player who sits a
// round out, a player whose tournament is over, a player who goes through to the next phase, a
// player whose fate waits on the rest of the phase. The answer is the engine's own
// (State.Statuses): it follows from the phase rules — passage, byes, repechage — which the host
// never codes a second time.

// StatusKind is the engine's code for a player's status.
type StatusKind = tournoi.StatusKind

// The kinds the wall shows. A player still playing is not listed (they know), nor a player
// never entered (the late arrivals' notice says where they come in), nor a withdrawn player,
// who has left the room.
const (
	StatusWinner     = tournoi.StatusWinner
	StatusQualified  = tournoi.StatusQualified
	StatusBye        = tournoi.StatusBye
	StatusUndecided  = tournoi.StatusUndecided
	StatusEliminated = tournoi.StatusEliminated
	StatusWithdrawn  = tournoi.StatusWithdrawn
)

// wallOrder is the order of the wall's lines, and the set of kinds it shows.
var wallOrder = map[StatusKind]int{
	StatusWinner: 0, StatusQualified: 1, StatusBye: 2, StatusUndecided: 3, StatusEliminated: 4,
}

// PlayerStatus is one line of the wall's answer: the engine's status, with the name the wall
// shows and the configuration of the phase it speaks of.
type PlayerStatus struct {
	tournoi.PlayerStatus
	Name string
	// PhaseCfg is the phase named by Phase, for its name on the wall.
	PhaseCfg tournoi.PhaseConfig
}

// Statuses says, at now, which players sit out, are out, go through or wait to know. Nothing
// while the tournament is a draft, finished, or waiting only for its close: the standings say it
// then.
func (d *Direction) Statuses(now time.Time) []PlayerStatus {
	st := d.st
	if st == nil || st.Finished {
		return nil
	}
	for _, a := range st.ProposeAt(now) {
		if a.Kind == tournoi.ActFinish {
			return nil
		}
	}
	var out []PlayerStatus
	for _, s := range st.Statuses() {
		if _, shown := wallOrder[s.Kind]; !shown || s.Player == tournoi.BYE {
			continue
		}
		ps := PlayerStatus{PlayerStatus: s, Name: d.playerName(s.Player)}
		if s.Phase >= 0 && s.Phase < len(st.Config.Phases) {
			ps.PhaseCfg = st.Config.Phases[s.Phase]
		}
		out = append(out, ps)
	}
	slices.SortFunc(out, func(a, b PlayerStatus) int {
		return cmp.Or(cmp.Compare(wallOrder[a.Kind], wallOrder[b.Kind]), cmp.Compare(a.Name, b.Name),
			cmp.Compare(a.Player, b.Player))
	})
	return out
}

// StatusLine renders what the wall says of one player, in the host's language.
func StatusLine(s PlayerStatus, cat *Catalog) string {
	l := NewLabeler(cat, nil)
	switch s.Kind {
	case StatusWinner:
		return wallTerm(cat, "rencontre.wall.winner", "vainqueur", nil)
	case StatusQualified:
		return wallTerm(cat, "rencontre.wall.qualified", "qualifié(e) — {phase}",
			map[string]any{"phase": l.PhaseName(s.PhaseCfg)})
	case StatusBye:
		if s.Label.Kind == tournoi.LabelRound {
			// A Swiss bye: the player sits this round out and plays the next.
			return wallTerm(cat, "rencontre.wall.byeRound", "exempté(e) — rejoue à la ronde {round}",
				map[string]any{"round": s.Round})
		}
		return wallTerm(cat, "rencontre.wall.bye", "exempté(e) — entre au tour {round}",
			map[string]any{"round": s.Round})
	case StatusUndecided:
		return wallTerm(cat, "rencontre.wall.undecided", "pas encore fixé — {phase}",
			map[string]any{"phase": l.PhaseName(s.PhaseCfg)})
	case StatusWithdrawn:
		return wallTerm(cat, "rencontre.wall.withdrawn", "retiré(e)", nil)
	}
	return wallTerm(cat, "rencontre.wall.eliminated", "éliminé(e)", nil)
}

// StatusBlock renders the wall's answer to "am I playing?" under a heading, or nothing when
// there is nothing to say. The heading names the event when the wall shows several.
func StatusBlock(statuses []PlayerStatus, event string, cat *Catalog) string {
	if len(statuses) == 0 {
		return ""
	}
	var b strings.Builder
	head := wallTerm(cat, "rencontre.wall.status", "Est-ce que je joue ?", nil)
	if event != "" {
		head += " — " + event
	}
	fmt.Fprintf(&b, `<section class="statuts"><h2>%s</h2><ul>`, html.EscapeString(head))
	for _, s := range statuses {
		fmt.Fprintf(&b, `<li class="%s">%s — %s</li>`, s.Kind,
			html.EscapeString(s.Name), html.EscapeString(StatusLine(s, cat)))
	}
	b.WriteString(`</ul></section>`)
	return b.String()
}

// WithStatuses puts the block just above the page's credit line.
func WithStatuses(page, block string) string {
	if block == "" {
		return page
	}
	const credit = `<p class="credit">`
	i := strings.LastIndex(page, credit)
	if i < 0 {
		return page
	}
	return page[:i] + block + page[i:]
}
