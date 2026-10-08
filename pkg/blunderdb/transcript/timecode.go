package transcript

import "fmt"

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
// infos are copies the caller owns, but their Inconsistencies may still share
// an array with the Replayer's cache, which an append must not write into.
func timeActions(actions []Action, infos []ActionInfo) {
	var last *int64
	var prevTick *int64
	for i := range infos {
		if i >= len(actions) {
			return
		}
		a, info := actions[i], &infos[i]
		info.DecisionMS, info.CubeDecisionMS = a.DecisionMS, a.CubeDecisionMS
		info.RollTickMS, info.TickMS = a.RollTickMS, a.TickMS

		roll := a.RollTickMS
		if !rolls(a.Kind) {
			roll = nil
		}
		roll = inOrder(info, &last, roll, "roll")
		tick := inOrder(info, &last, a.TickMS, "action")

		switch {
		case rolls(a.Kind):
			if info.CubeDecisionMS == nil && info.cubeChoice {
				info.CubeDecisionMS = span(prevTick, roll)
			}
			if info.DecisionMS == nil && a.Kind == KindChecker {
				info.DecisionMS = span(roll, tick)
			}
		case a.Kind == KindDouble || a.Kind == KindTake || a.Kind == KindPass || a.Kind == KindResign:
			if info.DecisionMS == nil {
				info.DecisionMS = span(prevTick, tick)
			}
		}
		prevTick = tick
	}
}

// inOrder checks one Repère against the last one the document gave and returns
// it, or nil when it runs backwards — marked then, and no duration may use it.
func inOrder(info *ActionInfo, last **int64, tick *int64, what string) *int64 {
	if tick == nil {
		return nil
	}
	if *last != nil && *tick < **last {
		info.Inconsistencies = append(info.Inconsistencies[:len(info.Inconsistencies):len(info.Inconsistencies)],
			Inconsistency{Kind: TimecodeBackwards, Detail: fmt.Sprintf("the %s is timed at %s, before %s", what, clock(*tick), clock(**last))})
		*last = tick
		return nil
	}
	*last = tick
	return tick
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
