package storage

import (
	"sort"
	"strings"
)

// This file holds the backend-independent half of PlayerTable: deciding who won
// a match, folding the per-player figures together, and ordering the rows. The
// backends contribute only the SQL that fills the inputs below, so the rules a
// reader would argue about are stated once and tested once — the same split
// PickReferencePlayer already uses for tournament badges.

// PlayerDecisionStat is one backend row of the counted-decisions query: the
// figures for one player and one decision type (0 = checker, 1 = cube).
type PlayerDecisionStat struct {
	Name         string
	DecisionType int
	SumErrMP     int64
	Count        int
	Errors       int
	Blunders     int
}

// MatchOutcomeRow is one match retained by the filter, with the points each
// seat took across its games and the checker moves played in it.
type MatchOutcomeRow struct {
	Player1     string
	Player2     string
	MatchLength int32
	// Points1/Points2 are the points won by seat 1 / seat 2, summed over the
	// match's games. A match is not stored with a winner of its own, so the
	// outcome is derived from these (see MatchOutcome).
	Points1 int32
	Points2 int32
	// CheckerMoves is the number of checker decisions played in the match, both
	// players together — the Snowie denominator for each of them.
	CheckerMoves int
}

// PlayerLuckAcc is the luck measured for one player: the signed total and the
// number of rolls it covers, which is never the number of rolls played.
type PlayerLuckAcc struct {
	SumMP int64
	Rolls int
}

// MatchOutcome reports which seat won a match: +1 for player 1, -1 for player
// 2, 0 when the match has no decided outcome.
//
// blunderDB stores no winner on a match, only points per game, so the rule has
// to be stated here:
//
//   - A match to N points is won by the seat that reaches N. If neither does,
//     the match is unfinished — a truncated log, an abandoned match — and
//     counts as neither a win nor a loss for anybody. That is why a player's
//     wins and losses can add up to less than their match count.
//   - A money session (no target score) is won by whoever took more points,
//     and is a draw at equal points. This mirrors gnuBG's own session result
//     (+1 / 0 / -1, relational.c MatchResult).
func MatchOutcome(matchLength, points1, points2 int32) int {
	if matchLength > 0 {
		switch {
		case points1 >= matchLength && points2 >= matchLength:
			// Both sides at the target is corrupt data, not a result.
			return 0
		case points1 >= matchLength:
			return 1
		case points2 >= matchLength:
			return -1
		default:
			return 0
		}
	}
	switch {
	case points1 > points2:
		return 1
	case points2 > points1:
		return -1
	default:
		return 0
	}
}

// BuildPlayerRows folds the backends' per-player queries into the table's rows.
//
// snowieErrMP holds each player's total error over ALL their decisions (the
// Snowie numerator, which unlike PR does not go through the counted-decision
// filter); its denominator is accumulated here from the matches the player
// appears in, both seats together.
func BuildPlayerRows(
	decisions []PlayerDecisionStat,
	matches []MatchOutcomeRow,
	snowieErrMP map[string]int64,
	luck map[string]PlayerLuckAcc,
) []PlayerRow {
	rows := map[string]*PlayerRow{}
	get := func(name string) *PlayerRow {
		if r, ok := rows[name]; ok {
			return r
		}
		r := &PlayerRow{Name: name}
		rows[name] = r
		return r
	}

	sumErr := map[string]int64{}
	snowieDenom := map[string]int{}

	for _, d := range decisions {
		if d.Name == "" {
			continue
		}
		r := get(d.Name)
		r.Decisions += d.Count
		r.Errors += d.Errors
		r.Blunders += d.Blunders
		sumErr[d.Name] += d.SumErrMP
		switch d.DecisionType {
		case 0:
			r.CheckerDecisions += d.Count
			r.PRChecker = pr(d.SumErrMP, d.Count)
		case 1:
			r.CubeDecisions += d.Count
			r.PRCube = pr(d.SumErrMP, d.Count)
		}
	}

	for _, m := range matches {
		outcome := MatchOutcome(m.MatchLength, m.Points1, m.Points2)
		for seat, name := range map[int]string{1: m.Player1, -1: m.Player2} {
			if name == "" {
				continue
			}
			r := get(name)
			r.Matches++
			// Both seats divide by the same denominator: the Snowie rate
			// measures one player's errors against the length of the game they
			// were both playing.
			snowieDenom[name] += m.CheckerMoves
			switch {
			case outcome == seat:
				r.Wins++
			case outcome == -seat:
				r.Losses++
			}
		}
	}

	for name, r := range rows {
		r.PR = pr(sumErr[name], r.Decisions)
		r.SnowieER = pr(snowieErrMP[name], snowieDenom[name])
		if l, ok := luck[name]; ok {
			r.LuckMPSum = l.SumMP
			r.LuckRolls = l.Rolls
		}
	}

	out := make([]PlayerRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, *r)
	}
	sortPlayerRows(out)
	return out
}

// pr is the Performance Rating: 500 × mean error in EMG. Zero decisions gives
// 0, which callers must read together with the decision count rather than as a
// perfect score.
func pr(sumErrMP int64, n int) float64 {
	if n == 0 {
		return 0
	}
	return 500 * float64(sumErrMP) / 1000 / float64(n)
}

// sortPlayerRows orders the table as it is first shown: best PR first, since
// the table exists to compare players. Players with no counted decision have no
// PR to rank — a zero there means "nothing measured", not "played perfectly" —
// so they go last instead of leading the table.
func sortPlayerRows(rows []PlayerRow) {
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if (a.Decisions == 0) != (b.Decisions == 0) {
			return b.Decisions == 0
		}
		if a.Decisions != 0 && a.PR != b.PR {
			return a.PR < b.PR
		}
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})
}
