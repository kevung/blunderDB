package database

import (
	"context"
	"fmt"
	"strconv"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
)

// Changing the configuration of a tournament ALREADY UNDER WAY (issue #385, ADR-0047 §3).
//
// A director changes their mind: at 22 h they lower the switch to finish earlier, on the
// Saturday evening they add a consolation they had not planned. The engine accepts all of it
// and refuses exactly two things — removing a phase that is open, and changing the format of a
// phase that has begun.
//
// What this file adds is the SENTENCE BEFORE THE ACT: applying a configuration mid-tournament
// is not a form being saved, it is a decision, and a decision is shown before it is taken.
// PreviewDirectionConfig answers "what exactly am I about to change?" — as codes and facts, so
// the nine languages read it in their own.

// ConfigChange is one difference between the configuration in force and the one about to be
// saved. Never a sentence: a code, the phase it concerns, and the two values.
type ConfigChange struct {
	// Code names the setting: "length", "target", "kind", "tableCount"… The frontend renders
	// it; this file never writes a word a user reads.
	Code string `json:"code"`
	// Phase is 1-based, and 0 when the change is not about a phase.
	Phase int    `json:"phase"`
	From  string `json:"from,omitempty"`
	To    string `json:"to,omitempty"`
	// Reason, on a refusal, says why this change is impossible: "finished", "drawn",
	// "started", "opened".
	Reason string `json:"reason,omitempty"`
}

// PhaseLock says whether a phase's FORMAT still changes, and why not.
//
// The field is greyed with its reason rather than hidden: a director who cannot find a control
// assumes they looked badly, and looks again.
type PhaseLock struct {
	Phase  int    `json:"phase"`
	Kind   string `json:"kind"`
	Locked bool   `json:"locked"`
	Reason string `json:"reason,omitempty"`
}

// ConfigPreview is what the confirmation shows: what changes, what cannot, and what the
// tournament has already opened.
type ConfigPreview struct {
	Changes  []ConfigChange `json:"changes"`
	Refusals []ConfigChange `json:"refusals"`
	Locks    []PhaseLock    `json:"locks"`
	// Opened counts the phases the tournament has already entered. None of them is removable,
	// so the view knows where its "remove" gesture stops.
	Opened  int  `json:"opened"`
	Current int  `json:"current"`
	Started bool `json:"started"`
}

// PreviewDirectionConfig compares a candidate configuration with the one in force, without
// writing anything.
//
// Called with the unmodified configuration it reports no change and still fills Locks — which
// is how the settings view knows, on opening, which formats are frozen.
func (d *Database) PreviewDirectionConfig(tournamentID int64, configJSON string) (*ConfigPreview, error) {
	next, err := parseDirectionConfig(configJSON)
	if err != nil {
		return nil, err
	}
	dir, err := direction.Open(context.Background(), d.DirectionStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	cur, err := dir.Config()
	if err != nil {
		return nil, err
	}
	st := dir.State()
	if st == nil {
		return nil, direction.ErrNoDirection
	}
	p := &ConfigPreview{Opened: len(st.Phases), Current: st.Current, Started: dir.Started()}
	p.Changes = diffConfig(cur, next)
	p.Locks = phaseLocks(st, next)
	p.Refusals = configRefusals(st, next)
	return p, nil
}

// phaseLocks reports, phase by phase, whether its format is still open to change.
func phaseLocks(st *tournoi.State, next tournoi.Config) []PhaseLock {
	locks := make([]PhaseLock, 0, len(next.Phases))
	for i := range next.Phases {
		l := PhaseLock{Phase: i + 1, Kind: next.Phases[i].Kind}
		if i < len(st.Phases) {
			l.Reason = lockReason(st, i)
			l.Locked = l.Reason != ""
		}
		locks = append(locks, l)
	}
	return locks
}

// lockReason names what froze a phase's format, in the order the director would say it:
// it is over, its draw is made, matches are running in it.
func lockReason(st *tournoi.State, i int) string {
	ph := st.Phases[i]
	switch {
	case i < st.Current:
		return "finished"
	case ph.Drawn:
		return "drawn"
	case ph.Started:
		return "started"
	}
	return ""
}

// configRefusals restates, as codes, what the engine's acceptConfig would refuse.
//
// It is stated twice on purpose: the engine refuses, and the panel must be able to SAY SO
// BEFORE the director clicks — a refusal discovered after the fact reads as a bug.
func configRefusals(st *tournoi.State, next tournoi.Config) []ConfigChange {
	var out []ConfigChange
	if len(next.Phases) < len(st.Phases) {
		out = append(out, ConfigChange{
			Code: "phaseRemoved", Phase: len(next.Phases) + 1,
			From: strconv.Itoa(len(st.Phases)), To: strconv.Itoa(len(next.Phases)),
			Reason: "opened",
		})
		return out
	}
	for i := range st.Phases {
		if next.Phases[i].Kind == st.Phases[i].Cfg.Kind {
			continue
		}
		if reason := lockReason(st, i); reason != "" {
			out = append(out, ConfigChange{
				Code: "kind", Phase: i + 1,
				From: st.Phases[i].Cfg.Kind, To: next.Phases[i].Kind, Reason: reason,
			})
		}
	}
	return out
}

// diffConfig lists what separates two configurations, in reading order: the tournament as a
// whole, then each phase.
func diffConfig(cur, next tournoi.Config) []ConfigChange {
	var out []ConfigChange
	add := func(code string, phase int, from, to string) {
		if from != to {
			out = append(out, ConfigChange{Code: code, Phase: phase, From: from, To: to})
		}
	}
	num := func(code string, phase, from, to int) { add(code, phase, itoa(from), itoa(to)) }
	yesno := func(code string, phase int, from, to bool) {
		add(code, phase, strconv.FormatBool(from), strconv.FormatBool(to))
	}

	add("name", 0, cur.Name, next.Name)
	num("tableCount", 0, cur.Tables.Count, next.Tables.Count)
	add("minPerPoint", 0, fmt.Sprintf("%g", cur.MinPerPoint), fmt.Sprintf("%g", next.MinPerPoint))
	num("breaks", 0, len(cur.Breaks), len(next.Breaks))
	num("prizes", 0, len(cur.Prizes.Sections), len(next.Prizes.Sections))
	add("entryFee", 0, fmt.Sprintf("%g", cur.Prizes.EntryFee), fmt.Sprintf("%g", next.Prizes.EntryFee))

	for i := len(next.Phases); i < len(cur.Phases); i++ {
		out = append(out, ConfigChange{Code: "phaseRemoved", Phase: i + 1, From: cur.Phases[i].Kind})
	}
	for i := range next.Phases {
		n := next.Phases[i]
		if i >= len(cur.Phases) {
			out = append(out, ConfigChange{Code: "phaseAdded", Phase: i + 1, To: n.Kind})
			continue
		}
		c := cur.Phases[i]
		p := i + 1
		add("kind", p, c.Kind, n.Kind)
		num("length", p, c.Length, n.Length)
		num("lives", p, c.Lives, n.Lives)
		num("target", p, c.Target, n.Target)
		num("finalLength", p, c.FinalLength, n.FinalLength)
		add("lengths", p, joinInts(c.Lengths), joinInts(n.Lengths))
		num("lengthLate", p, c.LengthLate, n.LengthLate)
		num("lateThreshold", p, c.LateThreshold, n.LateThreshold)
		num("batchMinutes", p, c.BatchMinutes, n.BatchMinutes)
		add("mode", p, c.Mode, n.Mode)
		add("pairing", p, c.Pairing, n.Pairing)
		add("entry", p, c.Entry, n.Entry)
		add("seeding", p, c.Seeding, n.Seeding)
		num("groupSize", p, c.GroupSize, n.GroupSize)
		num("qualifiers", p, c.Qualifiers, n.Qualifiers)
		yesno("avoidClubs", p, c.AvoidClubs, n.AvoidClubs)
		yesno("allowRematch", p, c.AllowRematch, n.AllowRematch)
		yesno("consolation", p, c.Consolation, n.Consolation)
		yesno("lastChance", p, c.LastChance, n.LastChance)
		yesno("reconciliation", p, c.Reconciliation, n.Reconciliation)
		yesno("recharge", p, c.Recharge, n.Recharge)
	}
	return out
}

func itoa(n int) string { return strconv.Itoa(n) }

// joinInts renders a length-per-round list as the organiser announces it: "15, 13, 11".
func joinInts(v []int) string {
	s := ""
	for i, n := range v {
		if i > 0 {
			s += ", "
		}
		s += strconv.Itoa(n)
	}
	return s
}
