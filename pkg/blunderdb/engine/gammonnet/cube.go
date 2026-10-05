// SPDX-License-Identifier: MIT

package gammonnet

// The cube model, ported from gammonNet's gn_cube.c/gn_cube.h. Every formula
// has a paragraph in upstream docs/specs/t34-videau-spec.md; a discrepancy
// from it is a bug, never an improvisation.
//
// Two different quantities are both called "the take point":
//   - the fixed breakpoints of the fully-live (x = 1) curve (livePoints, the
//     level anchors): they never move with efficiency, they are where the
//     piecewise shape bends;
//   - TakePoint / the level tp/cp fields: the actual take/cash points at the
//     chosen efficiency, a separate closed form.
//
// janowskiEquity and levelBlend blend the fixed curves by efficiency;
// TakePoint and tp/cp never touch them. They are computed independently, as
// gammonNet does, rather than one derived from the other.

import (
	"math"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/race"
)

// CubeOwner is who may turn the cube — gn_cube.h's GnCubeOwner. A
// disposition relative to the player on roll, never an owner id.
type CubeOwner int

const (
	CubeCentred  CubeOwner = iota // neither side owns it; either may double
	CubeOwned                     // the player on roll owns it
	CubeOpponent                  // the opponent owns it; the player on roll cannot double
)

// Mirror returns the same cube seen from the other side of the table. The
// search and any rollout must mirror ownership at every turn swap.
func (o CubeOwner) Mirror() CubeOwner {
	switch o {
	case CubeOwned:
		return CubeOpponent
	case CubeOpponent:
		return CubeOwned
	default:
		return CubeCentred
	}
}

// The six mutually exclusive outcomes denested from the network's five
// nested probabilities.
const (
	outWinSingle = iota
	outWinGammon
	outWinBackgammon
	outLoseSingle
	outLoseGammon
	outLoseBackgammon
	numOutcomes
)

// probsExclusive denests the five nested probabilities into six mutually
// exclusive outcome masses — gn_infer_reference.c's gn_probs_exclusive.
// Naive subtraction yields negative masses on real positions; they are
// floored at zero here, once, so no use site reimplements it.
func probsExclusive(probs *[NumOutputs]float32) [numOutcomes]float64 {
	win := float64(probs[PWin])
	winG := float64(probs[PWinGammon])
	winBG := float64(probs[PWinBackgammon])
	loseG := float64(probs[PLoseGammon])
	loseBG := float64(probs[PLoseBackgammon])

	values := [numOutcomes]float64{
		win - winG,
		winG - winBG,
		winBG,
		(1.0 - win) - loseG,
		loseG - loseBG,
		loseBG,
	}
	for i, v := range values {
		if v < 0 {
			values[i] = 0
		}
	}
	return values
}

// CubeInputs holds what Janowski's model needs beyond the winning chance:
// the average points won and lost, given the position. A scalar equity
// destroys exactly this.
type CubeInputs struct {
	Win        float64 // P(win), any margin
	WinPoints  float64 // E[points | win]  -- at least 1
	LosePoints float64 // E[points | lose] -- at least 1
}

// CubeInputsFromProbs fills a CubeInputs from a distribution — gn_cube_inputs.
// A conditional expectation over zero mass (win of exactly 0 or 1) is set to
// 1, not NaN, so a degenerate distribution does not poison callers.
func CubeInputsFromProbs(probs *[NumOutputs]float32) CubeInputs {
	outcomes := probsExclusive(probs)

	win := outcomes[outWinSingle] + outcomes[outWinGammon] + outcomes[outWinBackgammon]
	lose := outcomes[outLoseSingle] + outcomes[outLoseGammon] + outcomes[outLoseBackgammon]

	winPoints := 1.0
	if win > 0.0 {
		winPoints = (outcomes[outWinSingle] + 2.0*outcomes[outWinGammon] + 3.0*outcomes[outWinBackgammon]) / win
	}
	losePoints := 1.0
	if lose > 0.0 {
		losePoints = (outcomes[outLoseSingle] + 2.0*outcomes[outLoseGammon] + 3.0*outcomes[outLoseBackgammon]) / lose
	}

	return CubeInputs{Win: win, WinPoints: winPoints, LosePoints: losePoints}
}

// DefaultEfficiency is gammonNet's measured cube efficiency by owner state,
// each value fitted against a different column of gammonNet's exact two-sided
// bearoff table (upstream docs/mesures/t34-efficacite.json).
//
// These are branch coefficients, not a property of the position: a
// deliberate divergence from gnubg and XG, which index efficiency by position
// class (ADR-0029; gammonNet spec §3 forbids borrowing their constants).
// SearchConfig.CubeX at a mirrored leaf and Decide's eDT read the root's
// coefficient, exactly as the C does: a branch-local coefficient was
// measured negligible, and the model keeps the root's (ADR-0029).
func DefaultEfficiency(owner CubeOwner) float64 {
	switch owner {
	case CubeOwned:
		return 0.566
	case CubeOpponent:
		return 0.687
	default:
		return 0.688
	}
}

// ── Money: the Janowski model, per unit of cube ─────────────────────────────

// janowskiE is the cubeless equity per unit of cube, pW - (1-p)L. Linear in p
// by construction — it is also E_dead, verbatim: a dead cube is never turned
// again, so cubeful and cubeless equity coincide.
func janowskiE(p, w, l float64) float64 {
	return p*w - (1.0-p)*l
}

// livePoints are the fixed breakpoints of the x=1 (fully live) equity curve.
func livePoints(w, l float64) (tpLive, cpLive float64) {
	denom := w + l + 0.5
	return (l - 0.5) / denom, (l + 1.0) / denom
}

// segment is the value at p of the straight line through (x0, y0) and
// (x1, y1). Every piece of every live curve in this file is one of these, so
// the pieces are named by their endpoints — the spec's own notation — rather
// than by an expanded slope a sign slip could hide in. A degenerate segment
// (x1 <= x0, which a solved breakpoint can produce at the extremes)
// returns its own endpoint rather than dividing by zero.
func segment(p, x0, y0, x1, y1 float64) float64 {
	if x1-x0 <= 0.0 {
		return y1
	}
	return y0 + (y1-y0)*((p-x0)/(x1-x0))
}

// janowskiEquity is the Janowski equity of one cube state, per unit of cube,
// at the given efficiency: dead is e(p); live is piecewise linear over all of
// [0, 1], from (0, -L) to (1, +W), bending at the cube state's breakpoints.
//
// The tails are not plateaux (ADR-0022): above CP_live the holder plays on
// for the gammon and the curve rises to W. Capping it at max(1, e(p)) prices
// the retained cube at zero and makes TooGood unreachable.
func janowskiEquity(p, w, l float64, owner CubeOwner, efficiency float64) float64 {
	tpLive, cpLive := livePoints(w, l)
	dead := janowskiE(p, w, l)

	var live float64
	switch owner {
	case CubeOwned:
		// (0, -L) to (CP_live, +1) to (1, +W).
		if p <= cpLive {
			live = segment(p, 0.0, -l, cpLive, 1.0)
		} else {
			live = segment(p, cpLive, 1.0, 1.0, w)
		}
	case CubeOpponent:
		// (0, -L) to (TP_live, -1) to (1, +W).
		if p <= tpLive {
			live = segment(p, 0.0, -l, tpLive, -1.0)
		} else {
			live = segment(p, tpLive, -1.0, 1.0, w)
		}
	default: // CubeCentred
		// The other two glued together: (0, -L) to (TP_live, -1) to
		// (CP_live, +1) to (1, +W).
		switch {
		case p <= tpLive:
			live = segment(p, 0.0, -l, tpLive, -1.0)
		case p <= cpLive:
			live = segment(p, tpLive, -1.0, cpLive, 1.0)
		default:
			live = segment(p, cpLive, 1.0, 1.0, w)
		}
	}

	return (1.0-efficiency)*dead + efficiency*live
}

// TakePoint is TP(x) or CP(x), picked by owner. CubeOwned returns CP(x), my
// winning chance past which the opponent would no longer take; otherwise the
// cube could land on me and it returns my own take point TP(x).
//
// ok is false for an unusable input (never for one from CubeInputsFromProbs,
// whose W and L floors keep the denominator positive).
func TakePoint(in CubeInputs, owner CubeOwner, efficiency float64) (tp float64, ok bool) {
	w, l := in.WinPoints, in.LosePoints
	denom := w + l + efficiency/2.0
	if denom <= 0.0 {
		return 0, false
	}
	if owner == CubeOwned {
		return (l + 0.5 + efficiency/2.0) / denom, true
	}
	return (l - 0.5) / denom, true
}

// ── Match: the redouble recursion at the score ──────────────────────────────
//
// Money's live curve is closed-form because money is scale-invariant. A
// match score breaks that (at 2-away/4-away the leader's cube dies at 2
// while the trailer's redouble to 4 is free), so the live curves are rebuilt
// per stake level, as gn_cube.c does.

// matchMaxAway is the away-score horizon this recursion trusts: the extent of
// the match equity table (state.MET.GetME). Beyond it the table silently reuses
// its last row, so such a state is refused rather than approximated.
// gammonNet's own ceiling is 25, its table's extent.
const matchMaxAway = engine.MaxScore

// MatchState is the match context a cube decision needs.
type MatchState struct {
	AwayOnRoll   int  // points the player on roll still needs; >= 1
	AwayOpponent int  // points the opponent still needs; >= 1
	Cube         int  // cube value: 1, 2, 4, 8, ...
	Crawford     bool // true iff the game being evaluated IS the Crawford game
	// MET is the match equity table the state is valued with; nil is the
	// built-in Kazaross-XG2 (ADR-0068). The evaluation cache holds network
	// outputs only, so a searcher reused across tables never mixes them.
	MET *engine.MET
}

// IsValid reports whether the state can be evaluated at all: positive away
// scores within this recursion's horizon (matchMaxAway), and a cube that is
// a power of two.
func (s MatchState) IsValid() bool {
	if s.AwayOnRoll < 1 || s.AwayOpponent < 1 {
		return false
	}
	if s.AwayOnRoll > matchMaxAway || s.AwayOpponent > matchMaxAway {
		return false
	}
	// The Crawford flag is only coherent when one away score is already 1;
	// state.MET.GetME assumes it (unlike gn_met_after, which silently
	// falls back to the pre-Crawford table), so anything else is refused.
	if s.Crawford && s.AwayOnRoll != 1 && s.AwayOpponent != 1 {
		return false
	}
	return s.Cube >= 1 && s.Cube&(s.Cube-1) == 0
}

// Swap returns the same state seen from the other side of the table. Cube
// and Crawford are shared facts; only the away scores trade places.
func (s MatchState) Swap() MatchState {
	s.AwayOnRoll, s.AwayOpponent = s.AwayOpponent, s.AwayOnRoll
	return s
}

// metAfter is the on-roll player's MWC if the game ends with points going to
// one side, through the match equity table (state.MET.GetME) rather than a
// re-ported gn_met.c. Only away scores matter, so matchTo is the larger one.
func metAfter(state MatchState, points int, onRollWins bool) (float64, bool) {
	if !state.IsValid() || points < 1 {
		return 0, false
	}
	matchTo := state.AwayOnRoll
	if state.AwayOpponent > matchTo {
		matchTo = state.AwayOpponent
	}
	score0 := matchTo - state.AwayOnRoll
	score1 := matchTo - state.AwayOpponent
	fWhoWins := 0
	if !onRollWins {
		fWhoWins = 1
	}
	return state.MET.GetME(score0, score1, matchTo, 0, points, fWhoWins, state.Crawford), true
}

// matchWinningChance is the on-roll player's MWC if the cube never moves
// again this game — gn_match_winning_chance. The C's six-outcome sum
// `sum_i outcomes[i] * metAfter(STAKE[i]*cube, WE_WIN[i])` is exactly
// `winMass*branchMwc(...,true) + loseMass*branchMwc(...,false)`.
func matchWinningChance(state MatchState, probs *[NumOutputs]float32) (float64, bool) {
	outcomes := probsExclusive(probs)
	winMass := outcomes[outWinSingle] + outcomes[outWinGammon] + outcomes[outWinBackgammon]
	loseMass := outcomes[outLoseSingle] + outcomes[outLoseGammon] + outcomes[outLoseBackgammon]

	winMWC, ok1 := branchMwc(state, outcomes, state.Cube, true)
	loseMWC, ok2 := branchMwc(state, outcomes, state.Cube, false)
	if !ok1 || !ok2 {
		return 0, false
	}
	return winMass*winMWC + loseMass*loseMWC, true
}

// matchEquity is 2×MWC−1 — gn_match_equity (ADR-0016). ok is false only when
// state is not IsValid().
func matchEquity(state MatchState, probs *[NumOutputs]float32) (float64, bool) {
	if !state.IsValid() {
		return 0, false
	}
	mwc, ok := matchWinningChance(state, probs)
	if !ok {
		return 0, false
	}
	return 2*mwc - 1, true
}

// valueFromProbs is the cubeless value of a distribution from its own side:
// money equity with no match state, 2×MWC−1 otherwise — gn_search.c's
// value_from_probs without its cube branch (the cubeful leaf is Value,
// ADR-0023). An invalid state values as 0, as value_from_probs does.
func valueFromProbs(probs *[NumOutputs]float32, state *MatchState) float64 {
	if state == nil {
		return float64(moneyEquity(probs))
	}
	eq, ok := matchEquity(*state, probs)
	if !ok {
		return 0
	}
	return eq
}

// CubelessValue is valueFromProbs, exported for internal/gui's race-regime
// bonus (evaluateRaceRegime).
func CubelessValue(probs *[NumOutputs]float32, state *MatchState) float64 {
	return valueFromProbs(probs, state)
}

// matchLevel is one stake level of the §9 chain.
type matchLevel struct {
	dead    bool    // this stake covers both away scores: the base case
	loseAvg float64 // MWC of losing, at this position's gammon mix
	winAvg  float64 // MWC of winning, same mix
	pass    float64 // MWC after conceding this stake dry (never gammon-weighted)
	cash    float64 // MWC after collecting this stake dry
	tp      float64 // my take point, resolved against the 2x level
	cp      float64 // the opponent's take point, resolved against the 2x level
}

// maxCubeLevels bounds the chain up to the first dead level: from cube 1,
// seven doublings exceed matchMaxAway (64); eight always leaves room for the
// 2c level even when c is already dead.
const maxCubeLevels = 8

// branchMwc is the MWC of one branch (win or lose) at stake, weighted by the
// position's single/gammon/backgammon mix within it; a massless branch is a
// plain single game.
func branchMwc(state MatchState, outcomes [numOutcomes]float64, stake int, onRollWins bool) (float64, bool) {
	var single, gammon, bg float64
	if onRollWins {
		single, gammon, bg = outcomes[outWinSingle], outcomes[outWinGammon], outcomes[outWinBackgammon]
	} else {
		single, gammon, bg = outcomes[outLoseSingle], outcomes[outLoseGammon], outcomes[outLoseBackgammon]
	}
	mass := single + gammon + bg

	m1, ok1 := metAfter(state, 1*stake, onRollWins)
	m2, ok2 := metAfter(state, 2*stake, onRollWins)
	m3, ok3 := metAfter(state, 3*stake, onRollWins)
	if !ok1 || !ok2 || !ok3 {
		return 0, false
	}
	if mass <= 0.0 {
		return m1, true
	}
	return (single/mass)*m1 + (gammon/mass)*m2 + (bg/mass)*m3, true
}

// levelDead is M_dead(p; k), linear between the two gammon-mix anchors, at
// the queried p. Never cache it at the position's own p: levelSolve reads
// it at the pieces' endpoints.
func levelDead(lv *matchLevel, p float64) float64 {
	return (1.0-p)*lv.loseAvg + p*lv.winAvg
}

// levelLive is the fully-live curve of one stake level: janowskiEquity's
// shape with (0, -L), (1, +W) and ±1 replaced by loseAvg, winAvg, pass and
// cash. The tails run to the anchors (ADR-0022). On a dead level it is the
// dead line.
//
// Monotone non-decreasing in p, since loseAvg <= pass <= cash <= winAvg by
// construction: levelSolve's inversion stands on it.
func levelLive(lv *matchLevel, p float64, owner CubeOwner) float64 {
	if lv.dead {
		return levelDead(lv, p)
	}

	switch owner {
	case CubeOwned:
		if p <= lv.cp {
			return segment(p, 0.0, lv.loseAvg, lv.cp, lv.cash)
		}
		return segment(p, lv.cp, lv.cash, 1.0, lv.winAvg)
	case CubeOpponent:
		if p <= lv.tp {
			return segment(p, 0.0, lv.loseAvg, lv.tp, lv.pass)
		}
		return segment(p, lv.tp, lv.pass, 1.0, lv.winAvg)
	default: // CubeCentred
		switch {
		case p <= lv.tp:
			return segment(p, 0.0, lv.loseAvg, lv.tp, lv.pass)
		case p <= lv.cp:
			return segment(p, lv.tp, lv.pass, lv.cp, lv.cash)
		default:
			return segment(p, lv.cp, lv.cash, 1.0, lv.winAvg)
		}
	}
}

// levelBlend is M(x) = (1-x)*M_dead + x*M_live.
func levelBlend(lv *matchLevel, p float64, owner CubeOwner, efficiency float64) float64 {
	return (1.0-efficiency)*levelDead(lv, p) + efficiency*levelLive(lv, p, owner)
}

// levelSegment is one piece of a level curve, named by its endpoints like
// segment's.
type levelSegment struct {
	x0, y0, x1, y1 float64
}

// levelSegments writes the pieces of the fully-live curve in ascending p —
// gn_cube.c's level_segments, the list levelSolve inverts. levelLive picks
// among the same pieces with the same breakpoint rule (a breakpoint belongs
// to the piece on its left). Returns how many were written: 1 on a dead
// level, 2 or 3 otherwise.
//
// The centred order assumes tp <= cp, which holds because owning the cube is
// worth at least the opponent owning it at the same stake; were it violated,
// the middle piece is degenerate and levelSolve skips it.
func levelSegments(lv *matchLevel, owner CubeOwner, segs *[3]levelSegment) int {
	if lv.dead {
		segs[0] = levelSegment{0.0, lv.loseAvg, 1.0, lv.winAvg}
		return 1
	}
	switch owner {
	case CubeOwned:
		segs[0] = levelSegment{0.0, lv.loseAvg, lv.cp, lv.cash}
		segs[1] = levelSegment{lv.cp, lv.cash, 1.0, lv.winAvg}
		return 2
	case CubeOpponent:
		segs[0] = levelSegment{0.0, lv.loseAvg, lv.tp, lv.pass}
		segs[1] = levelSegment{lv.tp, lv.pass, 1.0, lv.winAvg}
		return 2
	default: // CubeCentred
		segs[0] = levelSegment{0.0, lv.loseAvg, lv.tp, lv.pass}
		segs[1] = levelSegment{lv.tp, lv.pass, lv.cp, lv.cash}
		segs[2] = levelSegment{lv.cp, lv.cash, 1.0, lv.winAvg}
		return 3
	}
}

// levelSolve is the p where a monotone level curve reaches target, in closed
// form (gn_cube.c's level_solve, spec §9): find the piece whose endpoint
// values bracket the target and solve that straight line. blend < 0 inverts
// the fully-live curve (breakpoint resolution); otherwise the curve blended
// at that efficiency (the reported take point) — exact, since levelDead is
// affine on [0, 1], so the blended curve is affine on the same pieces.
//
// The answer is inf{p : f(p) >= target} clamped to [0, 1]: a target at or
// under f(0) answers 0, one above f(1) answers 1, a flat piece answers its
// left bound, and a degenerate piece (x1 <= x0) is skipped. A NaN in the
// target or the curve answers NaN rather than falling through to a plausible
// 1.0.
func levelSolve(lv *matchLevel, owner CubeOwner, blend, target float64) float64 {
	var segs [3]levelSegment
	n := levelSegments(lv, owner, &segs)
	if math.IsNaN(target) {
		return target
	}
	for i := 0; i < n; i++ {
		s := segs[i]
		v0, v1 := s.y0, s.y1
		if s.x1-s.x0 <= 0.0 {
			continue
		}
		if blend >= 0.0 {
			v0 = (1.0-blend)*levelDead(lv, s.x0) + blend*s.y0
			v1 = (1.0-blend)*levelDead(lv, s.x1) + blend*s.y1
		}
		if math.IsNaN(v0) || math.IsNaN(v1) {
			return math.NaN()
		}
		if target <= v0 {
			return s.x0
		}
		if target <= v1 {
			if v1-v0 <= 0.0 {
				return s.x0
			}
			return s.x0 + (s.x1-s.x0)*((target-v0)/(v1-v0))
		}
	}
	return 1.0
}

// buildLevels builds the chain: levels[0] at the current cube, each next
// level at double the stake, ending on the first dead level but never before
// levels[1] (the double/take branch always needs 2c). Returns the number of
// levels, or 0 to refuse (unevaluable state, or a chain not dead within
// maxCubeLevels).
//
// Coupé en deux (spec §7.1) : les ancres ne dépendent que du candidat, les
// points de rupture du niveau au-dessus ; seule la seconde moitié se met en
// lot.
func buildLevels(state MatchState, outcomes [numOutcomes]float64) ([maxCubeLevels]matchLevel, int) {
	var levels [maxCubeLevels]matchLevel
	count := buildLevelAnchors(state, outcomes, &levels)
	if count == 0 {
		return levels, 0
	}
	resolveLevels(&levels, count)
	return levels, count
}

// buildLevelAnchors remplit loseAvg, winAvg, pass et cash de chaque niveau et
// laisse tp = 0, cp = 1. Rend le nombre de niveaux, ou 0 pour refuser.
//
// La forme de la chaîne ne dépend que de state, jamais de outcomes : c'est
// ce qui permet aux voies d'un lot de résoudre le même niveau au même moment
// (cubeValueBatch le vérifie).
func buildLevelAnchors(state MatchState, outcomes [numOutcomes]float64, levels *[maxCubeLevels]matchLevel) int {
	count := 0
	stake := state.Cube
	if stake > matchMaxAway {
		stake = matchMaxAway
	}

	for {
		lv := &levels[count]
		lv.dead = stake >= state.AwayOnRoll && stake >= state.AwayOpponent

		var okLose, okWin, okPass, okCash bool
		lv.loseAvg, okLose = branchMwc(state, outcomes, stake, false)
		lv.winAvg, okWin = branchMwc(state, outcomes, stake, true)
		lv.pass, okPass = metAfter(state, stake, false)
		lv.cash, okCash = metAfter(state, stake, true)
		lv.tp = 0.0
		lv.cp = 1.0
		if !okLose || !okWin || !okPass || !okCash {
			return 0
		}

		count++
		if count >= 2 && lv.dead {
			break
		}
		if count == maxCubeLevels {
			return 0
		}
		// Same cap on the way up: past the horizon a doubled stake buys
		// nothing new (every payout already saturated at matchMaxAway).
		if stake <= matchMaxAway {
			stake *= 2
		}
	}
	return count
}

// resolveLevels résout les points de rupture du plus profond au moins
// profond, si bien que chaque inversion vise un niveau 2k déjà complet.
func resolveLevels(levels *[maxCubeLevels]matchLevel, count int) {
	for i := count - 2; i >= 0; i-- {
		levels[i].tp = levelSolve(&levels[i+1], CubeOwned, -1.0, levels[i].pass)
		levels[i].cp = levelSolve(&levels[i+1], CubeOpponent, -1.0, levels[i].cash)
	}
}

// ── The leaf valuation for the search ───────────────────────────────────────

// Value is the cubeful value of one distribution, on the search's negating
// scale: money points per unit of cube when state == nil, otherwise 2×MWC−1
// through the redouble recursion at state's cube. Both negate between sides
// provided the caller mirrors owner and swaps state at each ply. ok is false
// for what cannot be valued.
func Value(probs *[NumOutputs]float32, owner CubeOwner, state *MatchState, efficiency float64) (float64, bool) {
	in := CubeInputsFromProbs(probs)

	if state == nil {
		return janowskiEquity(in.Win, in.WinPoints, in.LosePoints, owner, efficiency), true
	}
	if !state.IsValid() {
		return 0, false
	}

	outcomes := probsExclusive(probs)
	levels, count := buildLevels(*state, outcomes)
	if count < 2 {
		return 0, false
	}
	// No cube in play in the Crawford game: dead value at the current stake,
	// since the chain would price doublings the rules forbid. Mirrors
	// gn_cube.c.
	if state.Crawford {
		return 2.0*levelDead(&levels[0], in.Win) - 1.0, true
	}
	return 2.0*levelBlend(&levels[0], in.Win, owner, efficiency) - 1.0, true
}

// ── The decision, money and match sharing one verdict table ────────────────

// CubeAction is what the on-roll player should do, and what the opponent
// should answer.
type CubeAction int

const (
	NoDouble   CubeAction = iota
	DoubleTake            // double, taken
	DoublePass            // doubling wins outright: the opponent must pass
	TooGood               // playing on is worth more than cashing
)

// Verdict applies the verdict table to (eND, eDT, eDP) on any one scale.
// The comparison lives in race.VerdictFromEquities (ADR-0020: one shape for a
// cube decision) because race must never import gammonnet. eps is 0: these
// inputs are exact floats, with no uint16 quantisation to absorb.
func Verdict(eND, eDT, eDP float64) CubeAction {
	return cubeActionFromVerdict(race.VerdictFromEquities(eND, eDT, eDP, 0))
}

// cubeActionFromVerdict maps race.Verdict onto CubeAction, the same four
// outcomes.
func cubeActionFromVerdict(v race.Verdict) CubeAction {
	switch v {
	case race.VerdictDoubleTake:
		return DoubleTake
	case race.VerdictDoublePass:
		return DoublePass
	case race.VerdictTooGood:
		return TooGood
	default:
		return NoDouble
	}
}

// Decision is what the player on roll should do, and the numbers that
// explain it.
type Decision struct {
	Action CubeAction
	// Equity of doubling and of not doubling, on the same scale, so the
	// caller sees the margin, not only the verdict.
	EquityNoDouble float64
	EquityDouble   float64
	// EquityDoubleTake and EquityDoublePass are the two branches EquityDouble
	// is the minimum of, for a ND/DT/DP table.
	EquityDoubleTake float64
	EquityDoublePass float64
	// TakePoint is the opponent's take point at this state, for reporting.
	TakePoint float64
	// Beaver is the beaver/raccoon answer (DecideEx); Enabled is false unless
	// asked for at money. The fields above stay the no-beaver answer, bit for
	// bit, whatever was asked.
	Beaver BeaverDecision
}

// BeaverDecision is the cube decision when the opponent may beaver and the
// doubler may then raccoon — a money-only session rule (gammonNet spec §4bis,
// gn_cube.h's GnCubeBeaver). Beaver: the taker redoubles at once and keeps
// the cube, the game is played for 4c. Raccoon: the doubler answers with 8c
// and keeps the cube in turn. Neither can be declined; one raccoon, no
// further round. Every equity is per unit of the current cube, from the
// doubler's side, on Decision's scale.
type BeaverDecision struct {
	// Enabled is true when computed: money with the flag set. Otherwise every
	// other field is zero.
	Enabled bool
	// Action is the verdict with beavers on the table. DoubleTake covers both
	// "taken" and "beavered"; Beaver tells them apart.
	Action CubeAction
	// EquityDouble is min(pass, take, beaver): what doubling is worth once
	// the opponent picks his best answer.
	EquityDouble float64
	// EquityTake is 2·E(opponent owns), the plain take branch, repeated so
	// the three answers sit side by side.
	EquityTake float64
	// EquityBeaver is 4·E(opponent owns), or 8·E(I own) when the raccoon
	// pays.
	EquityBeaver float64
	// Beaver: the opponent's best answer is a beaver, strictly better for
	// him than taking and passing. A tie is a plain take.
	Beaver bool
	// Raccoon: once beavered, the doubler should raccoon (8·E(I own)
	// strictly above 4·E(opponent owns)). Answered whether or not beavering
	// was right, so an actual beaver can be judged too.
	Raccoon bool
}

// UnderBeaver is d with the opponent's answer to a double taken from the
// beaver decision when one was computed: the take branch becomes his best
// non-pass answer (take or beaver), EquityDouble and Action follow. Without
// a beaver decision, d unchanged. For the domain edge, where a session played
// with beavers is judged by them; the engine's own fields never move.
func (d Decision) UnderBeaver() Decision {
	if !d.Beaver.Enabled {
		return d
	}
	d.Action = d.Beaver.Action
	d.EquityDoubleTake = math.Min(d.Beaver.EquityTake, d.Beaver.EquityBeaver)
	d.EquityDouble = d.Beaver.EquityDouble
	return d
}

// decideBeaver is gn_cube.c's decide_beaver: the opponent's three answers to
// a double and the doubler's answer to a beaver, each a whole-cube multiple
// of the same two Janowski curves the plain decision uses — a beaver changes
// the stake and who holds the cube, never the model.
func decideBeaver(in CubeInputs, owner CubeOwner, efficiency, eND float64) BeaverDecision {
	eTheirs := janowskiEquity(in.Win, in.WinPoints, in.LosePoints, CubeOpponent, efficiency)
	eMine := janowskiEquity(in.Win, in.WinPoints, in.LosePoints, CubeOwned, efficiency)
	eDP := 1.0
	eDT := 2.0 * eTheirs
	eBeavered := 4.0 * eTheirs
	eRaccooned := 8.0 * eMine

	out := BeaverDecision{Enabled: true}
	out.Raccoon = eRaccooned > eBeavered
	eBV := eBeavered
	if out.Raccoon {
		eBV = eRaccooned
	}
	out.Beaver = eBV < eDT && eBV < eDP

	// The opponent takes or beavers, whichever hurts the doubler more; the
	// verdict table weighs that against passing as it weighs a plain take.
	eAnswer := eDT
	if eBV < eDT {
		eAnswer = eBV
	}
	out.EquityTake = eDT
	out.EquityBeaver = eBV
	out.EquityDouble = eDP
	if eAnswer < eDP {
		out.EquityDouble = eAnswer
	}
	if owner == CubeOpponent {
		// No double to answer: the replies would describe a move the player
		// on roll cannot make.
		out.Action = NoDouble
		out.Beaver, out.Raccoon = false, false
	} else {
		out.Action = Verdict(eND, eAnswer, eDP)
	}
	return out
}

// Decide is the money or match-score cube decision. state == nil is money;
// otherwise the decision is taken in MWC through the equity table, which
// replaces the money verdict rather than correcting it.
//
// jacoby applies to the "don't double" branch only, and only with a centred
// cube in a money game: in a match the equity table already prices gammons.
//
// ok is false when the state is not evaluable.
func Decide(probs *[NumOutputs]float32, owner CubeOwner, state *MatchState, efficiency float64, jacoby bool) (Decision, bool) {
	return DecideEx(probs, owner, state, efficiency, jacoby, false)
}

// DecideEx is Decide with the beaver rule (gn_cube_decide_ex): beaver fills
// Decision.Beaver in a money game; the other fields are Decide's, computed by
// the same code. In a match the flag has no effect: beavers are a
// money-session rule.
func DecideEx(probs *[NumOutputs]float32, owner CubeOwner, state *MatchState, efficiency float64, jacoby, beaver bool) (Decision, bool) {
	in := CubeInputsFromProbs(probs)

	if state == nil {
		wND, lND := in.WinPoints, in.LosePoints
		if jacoby && owner == CubeCentred {
			wND, lND = 1.0, 1.0
		}

		// eDT is the opponent's branch priced at the caller's (current
		// owner's) efficiency, as gn_cube.c does (ADR-0029). The
		// match branch below has the same shape.
		eND := janowskiEquity(in.Win, wND, lND, owner, efficiency)
		eDT := 2.0 * janowskiEquity(in.Win, in.WinPoints, in.LosePoints, CubeOpponent, efficiency)
		eDP := 1.0

		// denom = WinPoints + LosePoints + efficiency/2 >= 2 given
		// CubeInputsFromProbs' >= 1 floors: never refuses in practice.
		tp, _ := TakePoint(in, CubeOwned, efficiency)

		action := NoDouble
		if owner != CubeOpponent {
			// The verdict table presupposes doubling is an option.
			action = Verdict(eND, eDT, eDP)
		}

		dec := Decision{
			Action:           action,
			EquityNoDouble:   eND,
			EquityDouble:     math.Min(eDT, eDP),
			EquityDoubleTake: eDT,
			EquityDoublePass: eDP,
			TakePoint:        tp,
		}
		if beaver {
			dec.Beaver = decideBeaver(in, owner, efficiency, eND)
		}
		return dec, true
	}

	if !state.IsValid() {
		return Decision{}, false
	}

	outcomes := probsExclusive(probs)
	// The chain: levels[0] is the current cube (the "no double" curve),
	// levels[1] the doubled stake the opponent would own after a take.
	// Everything deeper exists only to resolve these two.
	levels, count := buildLevels(*state, outcomes)
	if count < 2 {
		return Decision{}, false
	}

	eND := levelBlend(&levels[0], in.Win, owner, efficiency)
	eDT := levelBlend(&levels[1], in.Win, CubeOpponent, efficiency)
	eDP := levels[0].cash
	eDouble := math.Min(eDT, eDP)
	if state.Crawford {
		// No cube in play: dead value, and the double branch reported equal
		// to no-double so the panel never reads a "missed double" where
		// doubling is illegal. Mirrors gn_cube.c.
		eND = levelDead(&levels[0], in.Win)
		eDT = eND
		eDouble = eND
	}

	// The opponent's take point at the doubled stake, on the blended curve
	// the decision used.
	tp := levelSolve(&levels[1], CubeOpponent, efficiency, eDP)

	// Post-Crawford needs no special case: metAfter encodes the post-Crawford
	// table, so the trailer's double and the leader's free drop fall out of
	// Verdict.
	action := NoDouble
	if !state.Crawford && owner != CubeOpponent {
		action = Verdict(eND, eDT, eDP)
	}

	return Decision{
		Action:           action,
		EquityNoDouble:   eND,
		EquityDouble:     eDouble,
		EquityDoubleTake: eDT,
		EquityDoublePass: eDP,
		TakePoint:        tp,
	}, true
}
