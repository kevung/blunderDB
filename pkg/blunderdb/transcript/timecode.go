package transcript

import (
	"fmt"
	"math"
	"slices"
)

// TimeActions is what a Replay does to the ActionInfos of actions, for a caller
// that walked them one at a time with a [Machine]: their Repères, and the
// durations they give where no Arbiter measured one. infos[i] describes
// actions[i].
func TimeActions(actions []Action, infos []ActionInfo) { timeActions(actions, infos) }

// timeActions writes on each ActionInfo the Repères of its Action and the
// durations they give (ADR-0082 rule 2), and marks the Repères out of order.
//
// A cube decision runs from the previous Action's instant, whichever side it
// was, to the roll; a checker play from the roll to the action; a double, an
// answer, a resignation from the previous Action's instant to its own. A
// game's first play and a roll the side could not double before have no cube
// decision; a dance and an unrecorded play have no checker decision. A
// duration an Arbiter measured stands; a missing Repère, or one marked
// TimecodeBackwards, leaves the durations that use it unknown.
//
// A checker play with no action Repère is ESTIMATED from its roll to the next
// Action's first instant — its roll, or a cube action's own — and marked so: an
// upper bound, which takes in the dice being picked up and thrown. Nothing
// estimates a cube decision: the end of the previous play and the start of the
// thinking about the cube cannot be told apart.
//
// infos are copies the caller owns, but their Inconsistencies may still share
// an array with the Replayer's cache, which an append must not write into.
func timeActions(actions []Action, infos []ActionInfo) {
	n := min(len(actions), len(infos))
	// The Repères in order, nil where missing or out of order: the only instants
	// a duration, measured or estimated, may use.
	rollAt := make([]*int64, n)
	tickAt := make([]*int64, n)
	var marks []repere
	for i := 0; i < n; i++ {
		a, info := actions[i], &infos[i]
		info.DecisionMS, info.CubeDecisionMS = a.DecisionMS, a.CubeDecisionMS
		info.RollTickMS, info.TickMS = a.RollTickMS, a.TickMS
		if a.RollTickMS != nil && rolls(a.Kind) {
			marks = append(marks, repere{action: i, ms: *a.RollTickMS})
		}
		if a.TickMS != nil {
			marks = append(marks, repere{action: i, ms: *a.TickMS, action2: true})
		}
	}
	kept := inOrder(marks)
	for j, m := range marks {
		ms := m.ms
		if kept[j] {
			if m.action2 {
				tickAt[m.action] = &ms
			} else {
				rollAt[m.action] = &ms
			}
			continue
		}
		info := &infos[m.action]
		rule, what := RuleRollOutOfOrder, "roll"
		if m.action2 {
			rule, what = RuleActionOutOfOrder, "action"
		}
		info.Inconsistencies = append(info.Inconsistencies[:len(info.Inconsistencies):len(info.Inconsistencies)],
			Inconsistency{Kind: TimecodeBackwards, Rule: rule, Args: map[string]int{"ms": int(ms)},
				Detail: fmt.Sprintf("the %s is timed at %s, out of the video's order", what, clock(ms))})
	}

	var prevTick *int64
	for i := 0; i < n; i++ {
		a, info := actions[i], &infos[i]
		roll, tick := rollAt[i], tickAt[i]
		switch {
		case rolls(a.Kind):
			if info.CubeDecisionMS == nil && info.cubeChoice {
				info.CubeDecisionMS = span(prevTick, roll)
			}
			if info.DecisionMS == nil && a.Kind == KindChecker {
				info.DecisionMS = span(roll, tick)
				// A Repère that ran backwards is not missing: it is wrong, and
				// nothing stands in for it.
				if info.DecisionMS == nil && a.TickMS == nil && i+1 < n {
					if info.DecisionMS = span(roll, nextInstant(actions[i+1].Kind, rollAt[i+1], tickAt[i+1])); info.DecisionMS != nil {
						info.DecisionEstimated = true
					}
				}
			}
		case isCubeAction(a.Kind):
			if info.DecisionMS == nil {
				info.DecisionMS = span(prevTick, tick)
			}
		}
		prevTick = tick
	}
}

// isCubeAction reports the Actions whose decision runs from the previous
// Action's instant to their own.
func isCubeAction(k Kind) bool {
	return k == KindDouble || k == KindTake || k == KindPass || k == KindResign
}

// nextInstant is the first instant an Action gives that ends the play before
// it: its roll, or a cube action's own Repère. A checker play timed by its
// action alone gives none — that instant ends its own thinking too.
func nextInstant(k Kind, roll, tick *int64) *int64 {
	if rolls(k) {
		return roll
	}
	if isCubeAction(k) {
		return tick
	}
	return nil
}

// repere is one instant of the document: an Action's roll, or its action
// (action2), in the order the document gives them.
type repere struct {
	action  int
	ms      int64
	action2 bool
}

// inOrder keeps the largest set of Repères that runs forward through the
// document, and reports the others as out of order. Blaming the fewest Repères
// blames the wrong one: an instant typed far ahead of the video (an Action
// inserted while the video had moved on) is the one out of place, not every
// correct instant after it. Between two equal choices the earlier Repère is
// kept.
func inOrder(marks []repere) []bool {
	n := len(marks)
	kept := make([]bool, n)
	if n == 0 {
		return kept
	}
	// from[i]: the length of the longest non-decreasing run of Repères that
	// starts at i, read with a Fenwick tree of suffix maxima over the instants'
	// ranks.
	ranks := make([]int64, n)
	for i, m := range marks {
		ranks[i] = m.ms
	}
	slices.Sort(ranks)
	ranks = slices.Compact(ranks)
	rank := func(ms int64) int {
		r, _ := slices.BinarySearch(ranks, ms)
		return len(ranks) - r // 1-based, reversed: a prefix is the instants >= ms
	}
	tree := make([]int, len(ranks)+1)
	from := make([]int, n)
	best := 0
	for i := n - 1; i >= 0; i-- {
		r := rank(marks[i].ms)
		longest := 0
		for k := r; k > 0; k -= k & -k {
			longest = max(longest, tree[k])
		}
		from[i] = longest + 1
		for k := r; k < len(tree); k += k & -k {
			tree[k] = max(tree[k], from[i])
		}
		best = max(best, from[i])
	}
	want, floor := best, int64(math.MinInt64)
	for i := 0; i < n && want > 0; i++ {
		if from[i] == want && marks[i].ms >= floor {
			kept[i], want, floor = true, want-1, marks[i].ms
		}
	}
	return kept
}

// span is the duration from one Repère to the next, nil when either is unknown.
func span(from, to *int64) *int64 {
	if from == nil || to == nil {
		return nil
	}
	d := *to - *from
	return &d
}

// clock writes an instant of the media as h:mm:ss.mmm, the way a player reads it.
func clock(ms int64) string {
	s := ms / 1000
	return fmt.Sprintf("%d:%02d:%02d.%03d", s/3600, s/60%60, s%60, ms%1000)
}
