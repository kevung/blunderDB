// SPDX-License-Identifier: MIT

package gammonnet

// The exact certain-loss reading of spec §5.5: a reading, never an equity
// judgement — the network takes no part. Valid only in a race, where the two
// sides are independent.

type resignGoal int

const (
	goalAllOff resignGoal = iota
	goalOneOff
	goalOutOfZone
)

// The 21 rolls, largest first: an existential search finds its witness soonest
// among the big rolls, and the order changes no answer.
var resignRolls = [NumRolls][2]int{
	{6, 6}, {5, 5}, {4, 4}, {6, 5}, {3, 3}, {6, 4}, {5, 4}, {6, 3}, {5, 3},
	{2, 2}, {6, 2}, {4, 3}, {5, 2}, {6, 1}, {4, 2}, {5, 1}, {1, 1}, {3, 2},
	{4, 1}, {3, 1}, {2, 1},
}

// resignDistance is the distance, in pips, from point index i to bearing off.
func resignDistance(side uint8, i int) int {
	if side == White {
		return i + 1
	}
	return NumPoints - i
}

func resignOwns(pos *Position, side uint8, i int) bool {
	if side == White {
		return pos.Points[i] > 0
	}
	return pos.Points[i] < 0
}

func resignCount(pos *Position, i int) int {
	n := int(pos.Points[i])
	if n < 0 {
		return -n
	}
	return n
}

// isRace: nobody on the bar, and every checker of one side past every checker
// of the other. White runs towards index 0, Black towards 23.
func isRace(pos *Position) bool {
	if pos.Bar[White] != 0 || pos.Bar[Black] != 0 {
		return false
	}
	whiteBack, blackBack := -1, NumPoints
	for i := 0; i < NumPoints; i++ {
		if pos.Points[i] > 0 {
			whiteBack = i
		}
		if pos.Points[i] < 0 && blackBack == NumPoints {
			blackBack = i
		}
	}
	return whiteBack < blackBack
}

// inZone counts side's checkers in the opponent's home board, bar included.
func inZone(pos *Position, side uint8) int {
	n := int(pos.Bar[side])
	for i := 0; i < NumPoints; i++ {
		if resignOwns(pos, side, i) && resignDistance(side, i) > 18 {
			n += resignCount(pos, i)
		}
	}
	return n
}

func goalReached(pos *Position, side uint8, goal resignGoal) bool {
	switch goal {
	case goalAllOff:
		return pos.Off[side] == NumCheckers
	case goalOneOff:
		return pos.Off[side] >= 1
	default:
		return inZone(pos, side) == 0
	}
}

// pipCount is gn_position_pip_count: a checker on the bar travels 25 pips.
func pipCount(pos *Position, side uint8) int {
	pips := 0
	for i := 0; i < NumPoints; i++ {
		if resignOwns(pos, side, i) {
			pips += resignCount(pos, i) * resignDistance(side, i)
		}
	}
	return pips + int(pos.Bar[side])*25
}

// pipsNeeded is a lower bound on the pips side must travel before goal can
// hold: pure arithmetic, so it only ever discards a branch truly out of reach.
func pipsNeeded(pos *Position, side uint8, goal resignGoal) int {
	need := 0
	switch goal {
	case goalAllOff:
		return pipCount(pos, side)
	case goalOneOff:
		// Every checker must be home (distance <= 6), then one more pip.
		for i := 0; i < NumPoints; i++ {
			if d := resignDistance(side, i); resignOwns(pos, side, i) && d > 6 {
				need += resignCount(pos, i) * (d - 6)
			}
		}
		return need + 1
	default:
		for i := 0; i < NumPoints; i++ {
			if d := resignDistance(side, i); resignOwns(pos, side, i) && d > 18 {
				need += resignCount(pos, i) * (d - 18)
			}
		}
		return need
	}
}

// diceNeeded is a lower bound on the dice side must spend before goal can
// hold: a die moves one checker at most six pips.
func diceNeeded(pos *Position, side uint8, goal resignGoal) int {
	edge := 18
	switch goal {
	case goalAllOff:
		edge = 0
	case goalOneOff:
		edge = 6
	}
	need := 0
	for i := 0; i < NumPoints; i++ {
		if d := resignDistance(side, i); resignOwns(pos, side, i) && d > edge {
			need += resignCount(pos, i) * ((d - edge + 5) / 6)
		}
	}
	if goal == goalOneOff {
		need++ // the die that bears the checker off
	}
	return need
}

// reach: exists, can side reach goal within k rolls for SOME dice; otherwise,
// is it SURE to, whatever the dice. The player picks the play either way,
// closest to the goal first — an order that decides how soon the answer is
// found, never which. level indexes the play buffer of this recursion depth.
func (p *Policy) reach(pos *Position, side uint8, goal resignGoal, k int, exists bool, level int) bool {
	if goalReached(pos, side, goal) {
		return true
	}
	if k == 0 {
		return false
	}
	if pipsNeeded(pos, side, goal) > 24*k || diceNeeded(pos, side, goal) > 4*k {
		return false
	}
	if goal == goalAllOff {
		left := NumCheckers - int(pos.Off[side])
		// A roll bears off at most four checkers, and a non-double — which
		// the dice can always deal — at most two.
		perRoll := 2
		if exists {
			perRoll = 4
		}
		if left > perRoll*k {
			return false
		}
	}

	mover := *pos
	mover.Turn = side
	plays := p.plays[level]

	for _, roll := range resignRolls {
		n := p.gen.LegalPlays(&mover, roll[0], roll[1], plays)
		if n < 0 {
			return false // unreadable: never claim a certainty
		}
		ok := false
		if n == 0 {
			ok = p.reach(&mover, side, goal, k-1, exists, level+1)
		}
		for j := 0; j < n && !ok; j++ {
			if goalReached(&plays[j].Result, side, goal) {
				ok = true
			}
		}
		if !ok && k > 1 && n > 0 {
			// Closest first, by a selection walk, so an early witness stops
			// the whole enumeration.
			order := make([]int, n)
			for j := 0; j < n; j++ {
				order[j] = pipsNeeded(&plays[j].Result, side, goal)
			}
			for tried := 0; tried < n && !ok; tried++ {
				best := -1
				for j := 0; j < n; j++ {
					if order[j] >= 0 && (best < 0 || order[j] < order[best]) {
						best = j
					}
				}
				order[best] = -1
				ok = p.reach(&plays[best].Result, side, goal, k-1, exists, level+1)
			}
		}
		if exists && ok {
			return true
		}
		if !exists && !ok {
			return false
		}
	}
	return !exists
}

// certainLoss is gn_policy_certain_loss: 1, 2 or 3 when the player on roll
// has lost for certain and the value is certain, 0 otherwise. The raw value:
// Jacoby is the caller's to apply.
func (p *Policy) certainLoss(pos *Position) int {
	if !pos.Valid() || pos.isOver() || !isRace(pos) {
		return 0
	}
	me := pos.Turn
	opp := 1 - me

	for k := 1; k <= PolicyResignHorizon; k++ {
		if !p.reach(pos, opp, goalAllOff, k, false, 0) {
			continue
		}
		if p.reach(pos, me, goalAllOff, k, true, 0) {
			continue
		}
		// The opponent is done within k rolls and the player, who rolls
		// first, cannot be; the opponent cannot finish before soonest rolls,
		// so the player gets at least that many.
		soonest := 1
		for soonest < k && !p.reach(pos, opp, goalAllOff, soonest, true, 0) {
			soonest++
		}
		single := pos.Off[me] > 0 || p.reach(pos, me, goalOneOff, soonest, false, 0)
		gammon := pos.Off[me] == 0 && !p.reach(pos, me, goalOneOff, k, true, 0)
		switch {
		case single:
			return 1
		case gammon:
			if !p.reach(pos, me, goalOutOfZone, k, true, 0) {
				return 3
			}
			if inZone(pos, me) == 0 || p.reach(pos, me, goalOutOfZone, soonest, false, 0) {
				return 2
			}
		}
		return 0
	}
	return 0
}
