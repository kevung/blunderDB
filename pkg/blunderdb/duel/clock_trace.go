package duel

import "github.com/kevung/blunderdb/pkg/blunderdb/storage"

// MatchClock is the clock of a Match played under a Cadence, replayed from
// the durations its Moves kept, with the arithmetic the Arbiter keeps the
// clock with (charge, reserve, overrun).
type MatchClock struct {
	// Start is each player's reserve at the start, in milliseconds (index 0
	// is player 1).
	Start [2]int64 `json:"start"`
	// Remaining is the reserve left to the player of each Move after it, in
	// milliseconds, indexed like the Match's Moves; an overrun leaves 0.
	Remaining []int64 `json:"remaining"`
}

// Replay plays a Match's recorded turns through the clock. A cube decision and
// the play after its roll are one turn of the clock and share one delay; an
// unknown duration charges nothing.
func (c Cadence) Replay(matchLength int, turns storage.MatchTurns) MatchClock {
	start := c.reserveMS(matchLength, [2]int{int(turns.Score[0]), int(turns.Score[1])})
	out := MatchClock{Start: [2]int64{start, start}, Remaining: make([]int64, len(turns.Turns))}
	reserve := out.Start
	delay := int64(c.Delay) * 1000
	for i, t := range turns.Turns {
		var turn int64
		pay := func(ms *int64) {
			if ms == nil {
				return
			}
			reserve[t.Player] = max(reserve[t.Player]-chargeMS(delay, turn, *ms), 0)
			turn += *ms
		}
		pay(t.CubeMS)
		pay(t.PlayMS)
		out.Remaining[i] = reserve[t.Player]
	}
	return out
}
