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
// round out of a bracket, a player whose tournament is over, a player who goes through to the
// next phase. The engine's ranking notes do not say it: a bracket player waiting for an opponent
// carries the match that awaits them as their "exit", and a Swiss survivor who misses the cut is
// still "alive" in the phase they will not leave. So it is read from the phases themselves.

// StatusKind is what the wall says of one player.
type StatusKind string

const (
	// StatusQualified: the phase is over and the player enters the next one.
	StatusQualified StatusKind = "qualified"
	// StatusBye: the player is drawn into the bracket without a match in its first rounds.
	StatusBye StatusKind = "bye"
	// StatusEliminated: the player has no match left to play.
	StatusEliminated StatusKind = "eliminated"
)

// PlayerStatus is one line of the wall's answer.
type PlayerStatus struct {
	Player tournoi.PlayerID
	Name   string
	Kind   StatusKind
	// Round is, for a bye, the round of the bracket the player enters at (from 1).
	Round int
	// Phase is, for a qualification, the name of the phase the player enters.
	Phase tournoi.PhaseConfig
}

// Statuses says, at now, which players sit out, are out, or go through. Nothing while the
// tournament is a draft, finished, or waiting only for its close: the standings say it then.
func (d *Direction) Statuses(now time.Time) []PlayerStatus {
	st := d.st
	if st == nil || st.Finished || st.Current < 0 || st.Current >= len(st.Phases) {
		return nil
	}
	ph := st.Phases[st.Current]
	var next *tournoi.PhaseState
	for _, a := range st.ProposeAt(now) {
		switch a.Kind {
		case tournoi.ActFinish:
			return nil
		case tournoi.ActNextPhase:
			next = d.nextPhaseEntrants(a, now)
		}
	}

	status := map[tournoi.PlayerID]PlayerStatus{}
	set := func(p tournoi.PlayerID, s PlayerStatus) {
		if p == "" || p == tournoi.BYE || st.Withdrawn[p] {
			return
		}
		s.Player, s.Name = p, d.playerName(p)
		status[p] = s
	}
	// Out at an earlier phase: they entered one but not the phase under way.
	for _, earlier := range st.Phases[:st.Current] {
		for _, p := range earlier.Entrants {
			if !slices.Contains(ph.Entrants, p) {
				set(p, PlayerStatus{Kind: StatusEliminated})
			}
		}
	}
	if next != nil {
		// The phase is over: whoever the engine draws into the next one goes through, and
		// everyone else stops here.
		for _, p := range ph.Entrants {
			if slices.Contains(next.Entrants, p) {
				set(p, PlayerStatus{Kind: StatusQualified, Phase: next.Cfg})
			} else {
				set(p, PlayerStatus{Kind: StatusEliminated})
			}
		}
	} else {
		for _, p := range ph.Entrants {
			if phaseEliminated(st, ph, p) {
				set(p, PlayerStatus{Kind: StatusEliminated})
			}
		}
		for p, round := range bracketByes(st, ph) {
			set(p, PlayerStatus{Kind: StatusBye, Round: round})
		}
	}

	out := make([]PlayerStatus, 0, len(status))
	for _, s := range status {
		out = append(out, s)
	}
	order := map[StatusKind]int{StatusQualified: 0, StatusBye: 1, StatusEliminated: 2}
	slices.SortFunc(out, func(a, b PlayerStatus) int {
		return cmp.Or(cmp.Compare(order[a.Kind], order[b.Kind]), cmp.Compare(a.Name, b.Name),
			cmp.Compare(a.Player, b.Player))
	})
	return out
}

// nextPhaseEntrants draws the next phase on a copy of the log, which is how the engine itself
// picks who goes through; nothing is written to the Direction.
func (d *Direction) nextPhaseEntrants(a tournoi.Action, now time.Time) *tournoi.PhaseState {
	cp, err := tournoi.Replay(d.journal)
	if err != nil {
		return nil
	}
	ev, err := cp.EventFromAction(a, now)
	if err != nil || cp.Apply(ev) != nil || cp.Current >= len(cp.Phases) {
		return nil
	}
	return cp.Phases[cp.Current]
}

// phaseEliminated says whether a player of the phase under way has no match left in it. In a
// phase with lives it is the last life lost; in a bracket, a lost match that sent the player
// nowhere: a loser routed to a consolation already holds a place in a match not yet played.
func phaseEliminated(st *tournoi.State, ph *tournoi.PhaseState, p tournoi.PlayerID) bool {
	switch ph.Cfg.Kind {
	case tournoi.KindSwissLives, tournoi.KindGSL:
		return ph.Lives[p] > 0 && ph.Losses[p] >= ph.Lives[p]
	case tournoi.KindBracket, tournoi.KindLivesBracket:
		if !ph.Drawn || ph.Losses[p] == 0 {
			return false
		}
		for _, sec := range ph.Sections {
			for _, g := range sec.Matches {
				if !g.Done && !g.Skipped && (g.Players[0] == p || g.Players[1] == p) {
					return false
				}
			}
		}
		for _, m := range st.Running() {
			if m.A == p || m.B == p {
				return false
			}
		}
		return true
	}
	return false
}

// bracketByes gives, for each player drawn against an empty place, the round they enter at —
// while they have not played yet in the phase.
func bracketByes(st *tournoi.State, ph *tournoi.PhaseState) map[tournoi.PlayerID]int {
	if !ph.Drawn || (ph.Cfg.Kind != tournoi.KindBracket && ph.Cfg.Kind != tournoi.KindLivesBracket) {
		return nil
	}
	out := map[tournoi.PlayerID]int{}
	for _, sec := range ph.Sections {
		for r, idx := range sec.Rounds {
			for _, i := range idx {
				if i < 0 || i >= len(sec.Matches) {
					continue
				}
				g := sec.Matches[i]
				for side, p := range g.Players {
					if p == "" || p == tournoi.BYE || g.Players[1-side] != tournoi.BYE {
						continue
					}
					out[p] = max(out[p], r+2)
				}
			}
		}
	}
	busy := map[tournoi.PlayerID]bool{}
	for _, m := range st.Running() {
		busy[m.A], busy[m.B] = true, true
	}
	for p := range out {
		if busy[p] || ph.Wins[p]+ph.Losses[p] > 0 {
			delete(out, p)
		}
	}
	return out
}

// StatusLine renders what the wall says of one player, in the host's language.
func StatusLine(s PlayerStatus, cat *Catalog) string {
	l := NewLabeler(cat, nil)
	switch s.Kind {
	case StatusQualified:
		return wallTerm(cat, "rencontre.wall.qualified", "qualifié(e) — {phase}",
			map[string]any{"phase": l.PhaseName(s.Phase)})
	case StatusBye:
		return wallTerm(cat, "rencontre.wall.bye", "exempté(e) — entre au tour {round}",
			map[string]any{"round": s.Round})
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
