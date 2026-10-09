package rollout

import (
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/race"
)

// maxHalfMoves ends an untruncated game that has not finished — a
// precaution, never reached by play: the leaf is then valued as at a
// truncation.
const maxHalfMoves = 2000

// maxMoneyCube stops a money cube from doubling past it inside a game.
const maxMoneyCube = 1 << 12

// noOwner is a centred cube in cubeState.owner.
const noOwner = -1

// cubeState is the cube in absolute terms: its value and the colour that
// owns it, never a point of view.
type cubeState struct {
	value int
	owner int
}

// branch is one thing being rolled out: a pre-roll position with a cube.
// skipFirstCube says its side on roll has already made the cube decision of
// this turn — the "no double" branch of a cube rollout.
type branch struct {
	start         gammonnet.Position
	cube          cubeState
	skipFirstCube bool
}

// table is the fixed context of one rollout, shared read-only by every
// worker: whose point of view results take, the root cube they are counted
// in, the match score in absolute terms.
//
// Every value inside a game is in the root's « native » scale: money points
// per root cube, or the root's match winning chance. Both are affine in the
// output equity, so means, differences and standard deviations convert once
// at the end (EquityScale, ADR-0019).
type table struct {
	root uint8
	// answer says the rollout is of a take/pass position, rolled out as the
	// doubler's decision before the double: its chances are reported turned
	// to the answerer on roll at the stored position.
	answer   bool
	rootCube int
	match    bool
	away     [2]int // indexed by colour
	crawford bool
	jacoby   bool // money only
	// withoutLuck keeps the luck out of the result (Options.withoutLuck).
	withoutLuck bool
	// exec runs a batch's games; nil plays them on the rollout's own goroutines.
	exec     Exec
	settings Settings
	bearoff  *race.TwoSided
}

// outcome is one game of one branch.
type outcome struct {
	value   float64 // native, root's view, luck removed
	chances [gammonnet.NumOutputs]float64
}

// worker plays games. It owns its searchers; nothing in it is shared.
type worker struct {
	t      *table
	policy *gammonnet.Searcher // Settings.Ply: plays, cube actions, leaves
	luck   *gammonnet.Searcher // 0-ply: the luck of each roll
	cands  []gammonnet.Candidate
}

func newWorker(t *table) (*worker, error) {
	policy, err := gammonnet.NewSearcher(gammonnet.DefaultConfig(t.settings.Ply))
	if err != nil {
		return nil, err
	}
	luck, err := gammonnet.NewSearcher(gammonnet.DefaultConfig(0))
	if err != nil {
		return nil, err
	}
	return &worker{t: t, policy: policy, luck: luck, cands: make([]gammonnet.Candidate, gammonnet.MaxPlays)}, nil
}

func (t *table) ownerView(c cubeState, mover uint8) gammonnet.CubeOwner {
	switch c.owner {
	case noOwner:
		return gammonnet.CubeCentred
	case int(mover):
		return gammonnet.CubeOwned
	}
	return gammonnet.CubeOpponent
}

// state is the match as mover sees it at cube c; nil in money play.
func (t *table) state(c cubeState, mover uint8) *gammonnet.MatchState {
	if !t.match {
		return nil
	}
	return &gammonnet.MatchState{AwayOnRoll: t.away[mover], AwayOpponent: t.away[1-mover], Cube: c.value, Crawford: t.crawford}
}

// toRoot turns a native value from mover's view into the root's.
func (t *table) toRoot(mover uint8, v float64, c cubeState) float64 {
	if t.match {
		if mover == t.root {
			return v
		}
		return 1 - v
	}
	v *= float64(c.value) / float64(t.rootCube)
	if mover != t.root {
		return -v
	}
	return v
}

// fromSearch turns a search value (money points per cube, or 2×MWC−1) into
// the native scale of the same side.
func (t *table) fromSearch(v float64) float64 {
	if t.match {
		return (v + 1) / 2
	}
	return v
}

func (t *table) canDouble(c cubeState, mover uint8) bool {
	if t.crawford || (c.owner != noOwner && c.owner != int(mover)) {
		return false
	}
	return t.match || c.value < maxMoneyCube
}

// configure aims s at mover's view of the cube and score. SearchConfig has
// no Jacoby rule: the search values its leaves without it, as gammonNet's
// own search does. Jacoby enters where the rollout settles a value itself —
// the cube decision (decide), the truncation leaf and the final stake.
func (t *table) configure(s *gammonnet.Searcher, ply int, c cubeState, mover uint8) error {
	cfg := gammonnet.DefaultConfig(ply)
	if st := t.state(c, mover); st != nil {
		cfg.UseMatch = true
		cfg.Match = *st
	}
	owner := t.ownerView(c, mover)
	cfg.UseCube = true
	cfg.CubeOwner = owner
	cfg.CubeX = gammonnet.DefaultEfficiency(owner)
	return s.Reconfigure(cfg)
}

// decide is mover's cube decision on probs, and the native value of the
// position to mover once it is taken. noDouble forces the no-double value.
func (t *table) decide(probs *[gammonnet.NumOutputs]float32, c cubeState, mover uint8, noDouble bool) (gammonnet.CubeAction, float64) {
	owner := t.ownerView(c, mover)
	st := t.state(c, mover)
	eff := gammonnet.DefaultEfficiency(owner)
	// Plain Decide, never the beaver rule (ADR-0060 rule 4): a beavered game
	// would have to carry a 4c then 8c cube through a two-answer exchange the
	// game loop does not model, so a rollout's double/take is without beaver
	// even under has_beaver.
	d, ok := gammonnet.Decide(probs, owner, st, eff, t.jacoby && owner == gammonnet.CubeCentred)
	if !ok {
		// A cube the model cannot price (beyond its horizon) is held dead.
		if v, ok := gammonnet.Value(probs, owner, st, eff); ok {
			return gammonnet.NoDouble, t.fromSearch(v)
		}
		return gammonnet.NoDouble, t.fromSearch(gammonnet.CubelessValue(probs, st))
	}
	if noDouble || !t.canDouble(c, mover) {
		return gammonnet.NoDouble, d.EquityNoDouble
	}
	switch d.Action {
	case gammonnet.DoubleTake:
		return d.Action, d.EquityDoubleTake
	case gammonnet.DoublePass:
		return d.Action, d.EquityDoublePass
	}
	return d.Action, d.EquityNoDouble
}

// play runs one game of b with dice and returns its result in the root's
// view, luck removed.
//
// Variance reduction: before each roll the mover's value is taken one ply
// deep — the mean, over the 21 rolls, of the best 0-ply play after each —
// and the luck of the actual roll is its own value less that mean. The sum
// of those lucks is subtracted from the game's result. Its expectation is
// zero exactly, whatever the network's errors, because the mean is taken
// over the very function evaluated at the actual roll: the correction
// lowers the variance and never moves the mean.
func (w *worker) play(b *branch, dice gameDice) (outcome, error) {
	t := w.t
	pos := b.start
	c := b.cube
	luck := 0.0
	for ply := 0; ; ply++ {
		mover := pos.Turn
		if over(&pos) {
			return t.terminal(&pos, c, luck), nil
		}
		skipCube := ply == 0 && b.skipFirstCube
		if v, chances, ok := t.exactBearoff(&pos, c, skipCube); ok {
			return outcome{value: v - luck, chances: chances}, nil
		}
		truncated := t.settings.Truncation > 0 && ply >= t.settings.Truncation
		if truncated || ply >= maxHalfMoves {
			if err := t.configure(w.policy, t.settings.Ply, c, mover); err != nil {
				return outcome{}, err
			}
			probs, ok := w.policy.Probs(&pos)
			if !ok {
				return outcome{}, fmt.Errorf("rollout: position not evaluable at the leaf")
			}
			_, v := t.decide(&probs, c, mover, skipCube)
			return outcome{value: t.toRoot(mover, v, c) - luck, chances: t.rootChances(&probs, mover)}, nil
		}

		if !skipCube && t.canDouble(c, mover) {
			if err := t.configure(w.policy, t.settings.Ply, c, mover); err != nil {
				return outcome{}, err
			}
			probs, ok := w.policy.Probs(&pos)
			if !ok {
				return outcome{}, fmt.Errorf("rollout: position not evaluable for the cube")
			}
			switch action, _ := t.decide(&probs, c, mover, false); action {
			case gammonnet.DoubleTake:
				c = cubeState{value: c.value * 2, owner: int(1 - mover)}
			case gammonnet.DoublePass:
				return t.passed(mover, c, luck), nil
			}
		}

		d1, d2 := dice.roll(ply)
		if err := t.configure(w.luck, 0, c, mover); err != nil {
			return outcome{}, err
		}
		mean, actual, best, moved, err := w.rollLuck(&pos, c, d1, d2)
		if err != nil {
			return outcome{}, err
		}
		if !t.withoutLuck {
			luck += t.toRoot(mover, actual, c) - t.toRoot(mover, mean, c)
		}

		// At 0 ply the luck pass has already found the policy's play.
		if t.settings.Ply > 0 {
			if err := t.configure(w.policy, t.settings.Ply, c, mover); err != nil {
				return outcome{}, err
			}
			cand, ok, err := w.policy.BestPlay(&pos, d1, d2)
			if err != nil {
				return outcome{}, err
			}
			moved = ok
			best = cand.Play.Result
		}
		if moved {
			pos = best
		} else {
			pos.Turn = 1 - pos.Turn
		}
	}
}

// rollLuck values the 21 rolls from pos at 0 ply (the luck searcher already
// aimed at mover and c): mean is their weighted average, actual the value of
// (d1, d2), best the 0-ply play for (d1, d2) and moved false on a dance. The
// rolls are summed in a fixed order, so the mean is the same bit for bit
// wherever it is computed.
func (w *worker) rollLuck(pos *gammonnet.Position, c cubeState, d1, d2 int) (mean, actual float64, best gammonnet.Position, moved bool, err error) {
	t := w.t
	hi, lo := d1, d2
	if lo > hi {
		hi, lo = lo, hi
	}
	sum := 0.0
	danceValue, danceKnown := 0.0, false
	for a := 1; a <= 6; a++ {
		for b := 1; b <= a; b++ {
			n, err := w.luck.Plays(pos, a, b, w.cands)
			if err != nil {
				return 0, 0, gammonnet.Position{}, false, err
			}
			var v float64
			if n > 0 {
				v = t.fromSearch(w.cands[0].Equity)
			} else {
				if !danceKnown {
					if danceValue, err = w.danceValue(pos, c); err != nil {
						return 0, 0, gammonnet.Position{}, false, err
					}
					danceKnown = true
				}
				v = danceValue
			}
			weight := 2.0
			if a == b {
				weight = 1
			}
			sum += weight * v
			if a == hi && b == lo {
				actual = v
				if n > 0 {
					best, moved = w.cands[0].Play.Result, true
				}
			}
		}
	}
	return sum / 36, actual, best, moved, nil
}

// danceValue is the native value, to the mover, of passing: the opponent's
// 0-ply cubeful value of the same board, negated.
func (w *worker) danceValue(pos *gammonnet.Position, c cubeState) (float64, error) {
	t := w.t
	passed := *pos
	passed.Turn = 1 - pos.Turn
	probs, ok := w.luck.Probs(&passed)
	if !ok {
		return 0, fmt.Errorf("rollout: position not evaluable after a dance")
	}
	opp := passed.Turn
	owner := t.ownerView(c, opp)
	v, ok := gammonnet.Value(&probs, owner, t.state(c, opp), gammonnet.DefaultEfficiency(owner))
	if !ok {
		v = gammonnet.CubelessValue(&probs, t.state(c, opp))
	}
	if t.match {
		return 1 - t.fromSearch(v), nil
	}
	return -v, nil
}

// rootChances turns a mover's distribution into the root's.
func (t *table) rootChances(p *[gammonnet.NumOutputs]float32, mover uint8) [gammonnet.NumOutputs]float64 {
	var out [gammonnet.NumOutputs]float64
	if mover == t.root {
		for i := range out {
			out[i] = float64(p[i])
		}
		return out
	}
	out[gammonnet.PWin] = 1 - float64(p[gammonnet.PWin])
	out[gammonnet.PWinGammon] = float64(p[gammonnet.PLoseGammon])
	out[gammonnet.PWinBackgammon] = float64(p[gammonnet.PLoseBackgammon])
	out[gammonnet.PLoseGammon] = float64(p[gammonnet.PWinGammon])
	out[gammonnet.PLoseBackgammon] = float64(p[gammonnet.PWinBackgammon])
	return out
}

// passed ends a game on a refused double: mover wins the cube as it stood.
func (t *table) passed(mover uint8, c cubeState, luck float64) outcome {
	return t.settle(int(mover), 1, c, luck)
}

// terminal ends a finished game: its winner takes the stake times the cube,
// gammons not counting under Jacoby while the cube is centred.
func (t *table) terminal(p *gammonnet.Position, c cubeState, luck float64) outcome {
	winner := gammonnet.White
	if p.Off[gammonnet.Black] == gammonnet.NumCheckers {
		winner = gammonnet.Black
	}
	stake := stakeOf(p, winner)
	if t.jacoby && c.owner == noOwner {
		stake = 1
	}
	return t.settle(winner, stake, c, luck)
}

func (t *table) settle(winner, stake int, c cubeState, luck float64) outcome {
	rootWins := winner == int(t.root)
	var o outcome
	if rootWins {
		o.chances[gammonnet.PWin] = 1
		if stake >= 2 {
			o.chances[gammonnet.PWinGammon] = 1
		}
		if stake >= 3 {
			o.chances[gammonnet.PWinBackgammon] = 1
		}
	} else {
		if stake >= 2 {
			o.chances[gammonnet.PLoseGammon] = 1
		}
		if stake >= 3 {
			o.chances[gammonnet.PLoseBackgammon] = 1
		}
	}
	points := stake * c.value
	if t.match {
		o.value = t.mwcAfter(points, rootWins) - luck
		return o
	}
	v := float64(points) / float64(t.rootCube)
	if !rootWins {
		v = -v
	}
	o.value = v - luck
	return o
}

// mwcAfter is the root's match winning chance once points go to one side.
func (t *table) mwcAfter(points int, rootWins bool) float64 {
	us, them := t.away[t.root], t.away[1-t.root]
	matchTo := max(us, them)
	who := 0
	if !rootWins {
		who = 1
	}
	return engine.GnuBGGetME(matchTo-us, matchTo-them, matchTo, 0, points, who, t.crawford)
}

// over reports a finished game.
func over(p *gammonnet.Position) bool {
	return p.Off[gammonnet.White] == gammonnet.NumCheckers || p.Off[gammonnet.Black] == gammonnet.NumCheckers
}

// homeLow is the first index of colour's home board: White bears off
// towards index 0, Black towards 23 (gammonnet.Position).
func homeLow(colour int) int {
	if colour == gammonnet.White {
		return 0
	}
	return 18
}

// owns reports colour's checkers on board index i.
func owns(p *gammonnet.Position, colour, i int) int {
	n := int(p.Points[i])
	if colour == gammonnet.White && n > 0 {
		return n
	}
	if colour == gammonnet.Black && n < 0 {
		return -n
	}
	return 0
}

// stakeOf is the stake of a finished game won by winner: a loser who has
// borne off any checker loses a single game; otherwise a checker on the bar
// or in the winner's home board makes it a backgammon, and anything else a
// gammon.
func stakeOf(p *gammonnet.Position, winner int) int {
	loser := 1 - winner
	if p.Off[loser] > 0 {
		return 1
	}
	if p.Bar[loser] > 0 {
		return 3
	}
	low := homeLow(winner)
	for i := low; i < low+6; i++ {
		if owns(p, loser, i) > 0 {
			return 3
		}
	}
	return 2
}

// homeBoard is colour's one-sided bearoff board (home[i] on its (i+1)-point)
// and whether every checker it has left is there.
func homeBoard(p *gammonnet.Position, colour int) (home [6]int, allHome bool) {
	if p.Bar[colour] > 0 {
		return home, false
	}
	low := homeLow(colour)
	for i := 0; i < gammonnet.NumPoints; i++ {
		n := owns(p, colour, i)
		if n == 0 {
			continue
		}
		if i < low || i >= low+6 {
			return home, false
		}
		if colour == gammonnet.White {
			home[i] = n
		} else {
			home[23-i] = n
		}
	}
	return home, true
}

// exactBearoff values a pure bearoff the two-sided table covers, when both
// sides have borne a checker off (the table is gammonless, so only then is
// it exact). Money play reads the table's cubeful equities; at a match score
// the table's exact winning chance goes through the cube model.
func (t *table) exactBearoff(p *gammonnet.Position, c cubeState, noDouble bool) (float64, [gammonnet.NumOutputs]float64, bool) {
	var chances [gammonnet.NumOutputs]float64
	if t.bearoff == nil || p.Off[gammonnet.White] == 0 || p.Off[gammonnet.Black] == 0 {
		return 0, chances, false
	}
	mover := p.Turn
	us, ok1 := homeBoard(p, int(mover))
	them, ok2 := homeBoard(p, int(1-mover))
	if !ok1 || !ok2 || !t.bearoff.Covers(us, them) {
		return 0, chances, false
	}
	e, err := t.bearoff.Lookup(us, them)
	if err != nil {
		return 0, chances, false
	}
	probs := [gammonnet.NumOutputs]float32{gammonnet.PWin: float32(e.WinProb)}
	chances = t.rootChances(&probs, mover)

	var v float64
	if t.match {
		_, v = t.decide(&probs, c, mover, noDouble)
	} else {
		v = moneyBearoffValue(e, t.ownerView(c, mover), noDouble || !t.canDouble(c, mover))
	}
	return t.toRoot(mover, v, c), chances, true
}

// moneyBearoffValue is the exact money value, per unit of cube, of a
// bearoff to the side on roll, its cube decision taken optimally unless
// noDouble.
func moneyBearoffValue(e race.Entry, owner gammonnet.CubeOwner, noDouble bool) float64 {
	state := race.CubeCentered
	switch owner {
	case gammonnet.CubeOwned:
		state = race.CubeOwned
	case gammonnet.CubeOpponent:
		return e.Against
	}
	m := race.MoneyFromEntry(e, state)
	if noDouble {
		return m.NoDouble
	}
	switch m.Verdict {
	case race.VerdictDoubleTake:
		return m.DoubleTake
	case race.VerdictDoublePass:
		return m.DoublePass
	}
	return m.NoDouble
}
