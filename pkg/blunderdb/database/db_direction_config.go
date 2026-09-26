package database

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
)

// Changing the configuration of a tournament ALREADY UNDER WAY (tasks/nicomaque/fonctionnel.md §3). The engine refuses
// exactly two things: removing an open phase, and changing the format of a begun one. A
// mid-tournament change is a decision, so PreviewDirectionConfig shows it first — as codes and
// facts, for every language to phrase.

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
// The field is greyed with its reason rather than hidden, so the director knows why.
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
	p.Changes = append(diffConfig(cur, next), displacedMatches(st, cur, next)...)
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
// Stated twice on purpose: the panel must say so BEFORE the click; a refusal after the fact
// reads as a bug.
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
	add("tablesUnavailable", 0, joinInts(sortedInts(cur.Tables.Unavailable)), joinInts(sortedInts(next.Tables.Unavailable)))
	if reservedKey(cur.Tables.Reserved) != reservedKey(next.Tables.Reserved) {
		// The numbers alone may be equal while the purpose changed (the final's table becomes
		// the consolation's): the change is still one, and still shown.
		out = append(out, ConfigChange{Code: "tablesReserved", From: reservedTables(cur.Tables.Reserved), To: reservedTables(next.Tables.Reserved)})
	}
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

// displacedMatches names each running match whose table the new configuration takes out of
// service, with the free table it could move to — "unavailableBusy", From the table,
// To the proposal — or "unavailableBusyFull" when the hall has none left. The engine does not
// move a running match on its own, and the director should not have to count free tables.
func displacedMatches(st *tournoi.State, cur, next tournoi.Config) []ConfigChange {
	var out []ConfigChange
	running := st.Running()
	used := map[int]bool{}
	for _, m := range running {
		if m.Table > 0 {
			used[m.Table] = true
		}
	}
	for _, m := range running {
		if m.Table <= 0 || !slices.Contains(next.Tables.Unavailable, m.Table) || slices.Contains(cur.Tables.Unavailable, m.Table) {
			continue
		}
		free := freeTable(next.Tables, used, m)
		if free == 0 {
			out = append(out, ConfigChange{Code: "unavailableBusyFull", From: itoa(m.Table)})
			continue
		}
		used[free] = true
		out = append(out, ConfigChange{Code: "unavailableBusy", From: itoa(m.Table), To: itoa(free)})
	}
	return out
}

// freeTable is the smallest table a running match could move to: in service, not reserved for
// something else, not taken. The same rule as the engine's assignTables; 0 when there is none.
func freeTable(t tournoi.Tables, used map[int]bool, m *tournoi.Match) int {
	limit := t.Count
	if limit == 0 {
		limit = len(used) + len(t.Unavailable) + len(t.Reserved) + 1
	}
	for n := 1; n <= limit; n++ {
		if !used[n] && t.AvailableFor(n, m.Section, m.Phase) {
			return n
		}
	}
	return 0
}

func sortedInts(v []int) []int {
	out := slices.Clone(v)
	slices.Sort(out)
	return out
}

// reservedTables renders the reserved tables' numbers, in order.
func reservedTables(rules []tournoi.TableRule) string {
	var n []int
	for _, r := range rules {
		n = append(n, r.Table)
	}
	return joinInts(sortedInts(n))
}

// reservedKey is a canonical form of the reservations, so their order in the file is no change.
func reservedKey(rules []tournoi.TableRule) string {
	c := slices.Clone(rules)
	slices.SortFunc(c, func(a, b tournoi.TableRule) int {
		return cmp.Or(cmp.Compare(a.Table, b.Table), strings.Compare(a.Section, b.Section), cmp.Compare(a.Phase, b.Phase))
	})
	b, _ := json.Marshal(c)
	return string(b)
}

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
