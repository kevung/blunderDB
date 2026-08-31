// SPDX-License-Identifier: MIT

package gammonnet

// The cube model, ported from gammonNet's gn_cube.c/gn_cube.h. See those
// files (and docs/specs/t34-videau-spec.md upstream) before editing: every
// formula below has a paragraph number there, and a discrepancy from it is a
// bug, never an improvisation.
//
// ONE SEPARATION THAT MATTERS THROUGHOUT THIS FILE. Two DIFFERENT quantities
// both depend on cube efficiency and both get called "the take point" in
// conversation:
//
//   - the fixed breakpoints of the fully-live (x = 1) equity curve, returned
//     by livePoints/level_dead's level anchors. They never move with
//     efficiency; they are where the piecewise shape bends.
//   - TakePoint / the level tp/cp fields: the ACTUAL take/cash points at the
//     chosen efficiency, a separate closed form. These are what a caller
//     wants to know ("should I take"), not where a curve bends.
//
// janowskiEquity and level_blend blend the FIXED dead/live curves by
// efficiency; TakePoint and the level tp/cp never touch them, and vice versa.
// The two are algebraically related but computed independently here, exactly
// as gammonNet presents them, rather than deriving one from the other.

import (
	"math"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// CubeOwner is who may turn the cube — gn_cube.h's GnCubeOwner. The three
// cases are not symmetric: a centred cube is an option for both players, an
// owned cube for exactly one — which is why this is a disposition, never an
// owner id.
type CubeOwner int

const (
	CubeCentred  CubeOwner = iota // neither side owns it; either may double
	CubeOwned                     // the player on roll owns it
	CubeOpponent                  // the opponent owns it; the player on roll cannot double
)

// Mirror returns the same cube seen from the other side of the table: mine
// becomes theirs, centred stays centred. The search and any rollout must
// mirror ownership at every turn swap — forgetting it values every other ply
// with the wrong player holding the cube.
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
// exclusive outcome masses. Ported from gn_infer_reference.c's
// gn_probs_exclusive, called rather than reimplemented at every use site:
// subtracting nested probabilities naively yields a NEGATIVE probability on
// real positions (upstream's T10), floored at zero here, once.
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
// A win probability of exactly 0 or 1 leaves one conditional expectation
// averaged over zero mass; that is set to 1 (a plain single game), not NaN,
// so a degenerate distribution does not poison every caller that multiplies
// by it downstream.
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

// DefaultEfficiency is gammonNet's own measured cube efficiency, by owner
// state (T34, fitted 2026-08-07 against gammonNet's exact two-sided bearoff
// oracle — docs/mesures/t34-efficacite.json in the gammonNet repository:
// owned 0.566, centred 0.688, opponent 0.687, each with its own residual).
//
// blunderDB reuses these published, measured values as a documented starting
// point rather than an untuned guess. They were fitted against gammonNet's
// own oracle, not blunderDB's (ADR-0009, engine/race/twosided.go); re-fitting
// locally is a deliberate follow-up, not done here — see issue #122's
// discussion for why a first port keeps the upstream measurement rather than
// blocking on a second one.
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

// janowskiEquity is the Janowski equity of one cube state, per unit of cube,
// at the given efficiency.
//
// dead is e(p), used unclamped for the dead branch and as the "too good"
// continuation for the live one: beyond the point where a live cube's value
// saturates at the cash equivalent, playing on is scored by the plain
// cubeless equity, because that IS what happens once nobody has anything
// left to gain by turning the cube.
func janowskiEquity(p, w, l float64, owner CubeOwner, efficiency float64) float64 {
	tpLive, cpLive := livePoints(w, l)
	dead := janowskiE(p, w, l)

	var live float64
	switch owner {
	case CubeOwned:
		// (0, -L) to (CP_live, +1); beyond, max(1, e(p)).
		if p <= cpLive {
			live = -l + (1.0+l)*(p/cpLive)
		} else {
			live = math.Max(dead, 1.0)
		}
	case CubeOpponent:
		// Below TP_live, min(-1, e(p)); above, (TP_live, -1) to (1, W).
		if p <= tpLive {
			live = math.Min(dead, -1.0)
		} else {
			live = -1.0 + (w+1.0)*((p-tpLive)/(1.0-tpLive))
		}
	default: // CubeCentred
		// min(-1, e(p)) below TP_live; (TP_live,-1) to (CP_live,1); above,
		// max(1, e(p)). The centred curve is the other two glued together.
		if p <= tpLive {
			live = math.Min(dead, -1.0)
		} else if p <= cpLive {
			live = -1.0 + 2.0*((p-tpLive)/(cpLive-tpLive))
		} else {
			live = math.Max(dead, 1.0)
		}
	}

	return (1.0-efficiency)*dead + efficiency*live
}

// TakePoint is TP(x) or CP(x), picked by owner.
//
// owner == CubeOwned means I hold the cube and am weighing whether to turn
// it: the number that matters is not my own take point (nobody can double
// me) but CP(x), the boundary on MY winning chance past which my opponent
// would no longer take. CubeCentred or CubeOpponent means the cube could
// still land on me: the number that matters is my own take point, TP(x).
//
// ok is false for an unusable input (never for a CubeInputs produced by
// CubeInputsFromProbs, whose W and L floors keep the denominator positive),
// mirroring the refusal to clamp a wrong number into looking right.
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

// Equity is the cubeful money equity, in points, from the point of view of
// the player on roll, for a cube currently at cube under owner.
func Equity(in CubeInputs, owner CubeOwner, cube int, efficiency float64) float64 {
	return float64(cube) * janowskiEquity(in.Win, in.WinPoints, in.LosePoints, owner, efficiency)
}

// ── Match: the redouble recursion at the score ──────────────────────────────
//
// Money's live curve exists in closed form because money is scale-invariant:
// doubling the stake doubles every equity, so one recursion step looks like
// every other. A match score breaks that symmetry — at 2-away/4-away the
// leader's cube dies at 2 while the trailer's redouble to 4 is free, and a
// naive transposition of money's breakpoints onto the MWC scale cannot see
// that asymmetry. So the live curves are rebuilt here, per stake level,
// exactly as gammonNet's gn_cube.c does it.

// matchMaxAway is the away-score horizon this recursion trusts: blunderDB's
// own MET (engine.GnuBGGetME) extends Kazaross-XG2 with a Zadeh fallback up
// to this many points (engine.MaxScore) — beyond it, additional away points
// silently reuse the table's last row, so a state past this horizon is
// refused rather than approximated. This is gammonNet's own ceiling moved
// from 25 (its explicit table's extent, where it refuses) to blunderDB's 64
// — see #122's note on branching the MET.
const matchMaxAway = engine.MaxScore

// MatchState is the match context a cube decision needs.
type MatchState struct {
	AwayOnRoll   int  // points the player on roll still needs; >= 1
	AwayOpponent int  // points the opponent still needs; >= 1
	Cube         int  // cube value: 1, 2, 4, 8, ...
	Crawford     bool // true iff the game being evaluated IS the Crawford game
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
	// The Crawford game is, by definition, the one played right after a
	// player reaches match point: the flag is only coherent when one of the
	// two away scores is already 1 entering the game. engine.GnuBGGetME
	// assumes exactly that when it takes its "crawford is behind us" branch
	// (it only special-cases the side actually AT match point, unlike
	// gn_met_after's fallback that silently re-derives the pre-Crawford
	// table when neither side is) — so a crawford flag without either away
	// score at 1 is refused here, rather than fed to a lookup that does not
	// expect it.
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

// metAfter is the on-roll player's match winning chance if the game ends
// with points going to one side, routed through blunderDB's own MET
// (engine.GnuBGGetME) rather than a re-ported gn_met.c — see matchMaxAway.
//
// GnuBGGetME wants absolute scores and a match length; only the away scores
// matter here, so an arbitrary matchTo that reproduces them exactly is
// picked (matchTo = the larger away score, so at least one of the two scores
// is 0).
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
	return engine.GnuBGGetME(score0, score1, matchTo, 0, points, fWhoWins, state.Crawford), true
}

// matchWinningChance is the on-roll player's match winning chance if the
// cube never moves again this game, at its current value — gn_match_winning_
// chance, ported by reusing branchMwc's win/lose branch averages (each
// already the outcome-weighted MWC of its own branch at a stake) rather than
// re-deriving the six-outcome sum gn_met.c computes term by term: the C's
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

// matchEquity is 2×MWC−1 — gn_match_equity, the match-referential counterpart
// of MoneyEquity (ADR-0016). ok is false only when state is not IsValid();
// with a valid state, GnuBGGetME's own clamping means matchWinningChance
// never itself fails.
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

// valueFromProbs is the value of a distribution from its own side's point of
// view: cubeless money equity with no match state, 2×MWC−1 otherwise —
// gn_search.c's value_from_probs, without the cube branch (use_cube is a
// follow-up tranche, ADR-0016). A failure (only reachable with an invalid
// state, which NewSearcher already refuses at construction) values the
// distribution as 0 rather than propagating a search-wide failure over one
// node — the same choice value_from_probs makes.
func valueFromProbs(probs *[NumOutputs]float32, state *MatchState) float64 {
	if state == nil {
		return float64(MoneyEquity(probs))
	}
	eq, ok := matchEquity(*state, probs)
	if !ok {
		return 0
	}
	return eq
}

// CubelessValue is valueFromProbs, exported for a cold-path caller outside
// this package that needs a plain (non-cube) value in a position's own
// referential — internal/gui's race-regime bonus (evaluateRaceRegime) —
// same precedent as InvertProbs.
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

// maxCubeLevels bounds the chain from the current cube up to the first dead
// level. matchMaxAway is 64, so a chain from cube 1 is
// 1,2,4,8,16,32,64,128 — seven doublings to exceed the horizon; eight leaves
// room to always materialise the 2c level even when c is already dead.
const maxCubeLevels = 8

// branchMwc is the weighted MWC average of one branch (win, or lose) at
// stake, folding in the position's own single/gammon/backgammon mix within
// that branch. Falls back to a plain single game when the branch carries no
// mass — the least committal answer, and one that is never actually weighted
// into anything by a zero mass anyway.
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

// levelDead is M_dead(p; k): linear in p between the two gammon-mix anchors,
// evaluated at the QUERIED p. Never cached at the position's own p — the
// bisections below probe many p that are not the position's, and a cached
// MWC would be silently wrong at every one of them.
func levelDead(lv *matchLevel, p float64) float64 {
	return (1.0-p)*lv.loseAvg + p*lv.winAvg
}

// levelLive is the fully-live curve of one stake level — janowskiEquity's
// piecewise shape with this level's anchors and breakpoints. On a dead level
// the shape collapses to the dead line for every cube state.
func levelLive(lv *matchLevel, p float64, owner CubeOwner) float64 {
	dead := levelDead(lv, p)
	if lv.dead {
		return dead
	}

	switch owner {
	case CubeOwned:
		if p <= lv.cp {
			return lv.loseAvg + (lv.cash-lv.loseAvg)*(p/lv.cp)
		}
		return math.Max(dead, lv.cash)
	case CubeOpponent:
		if p <= lv.tp {
			return math.Min(dead, lv.pass)
		}
		return lv.pass + (lv.winAvg-lv.pass)*((p-lv.tp)/(1.0-lv.tp))
	default: // CubeCentred
		if p <= lv.tp {
			return math.Min(dead, lv.pass)
		}
		if p <= lv.cp {
			return lv.pass + (lv.cash-lv.pass)*((p-lv.tp)/(lv.cp-lv.tp))
		}
		return math.Max(dead, lv.cash)
	}
}

// levelBlend is M(x) = (1-x)*M_dead + x*M_live.
func levelBlend(lv *matchLevel, p float64, owner CubeOwner, efficiency float64) float64 {
	return (1.0-efficiency)*levelDead(lv, p) + efficiency*levelLive(lv, p, owner)
}

// levelSolve finds the p where a monotone level curve crosses target: the
// functions are piecewise-linear and monotone, so bisection suffices.
// blend < 0 bisects the fully-live curve (breakpoint resolution inside the
// recursion); otherwise the curve blended at that efficiency (the reported
// take point).
func levelSolve(lv *matchLevel, owner CubeOwner, blend, target float64) float64 {
	low, high := 0.0, 1.0
	for i := 0; i < 60; i++ {
		mid := 0.5 * (low + high)
		var value float64
		if blend < 0.0 {
			value = levelLive(lv, mid, owner)
		} else {
			value = levelBlend(lv, mid, owner, blend)
		}
		if value < target {
			low = mid
		} else {
			high = mid
		}
	}
	return 0.5 * (low + high)
}

// buildLevels builds the chain: levels[0] at the current cube, each next
// level at double the stake, ending on the first dead level — and never
// before levels[1], because callers always need the 2c level for the
// double/take branch, even when the current cube is already dead.
//
// Returns the number of levels, or 0 to refuse (an unevaluable state, or a
// chain that failed to die within maxCubeLevels — unreachable while
// matchMaxAway bounds away scores, and refused rather than approximated if
// that bound ever moves).
//
// Breakpoints are resolved backwards, deepest first, so each level's
// bisection targets a fully-built 2k level: an iterative form of the
// recursion, each (state, k) computed once from the base case down.
func buildLevels(state MatchState, outcomes [numOutcomes]float64) ([maxCubeLevels]matchLevel, int) {
	var levels [maxCubeLevels]matchLevel
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
			return levels, 0
		}

		count++
		if count >= 2 && lv.dead {
			break
		}
		if count == maxCubeLevels {
			return levels, 0
		}
		// Same cap on the way up: past the horizon a doubled stake buys
		// nothing new (every payout already saturated at matchMaxAway).
		if stake <= matchMaxAway {
			stake *= 2
		}
	}

	for i := count - 2; i >= 0; i-- {
		levels[i].tp = levelSolve(&levels[i+1], CubeOwned, -1.0, levels[i].pass)
		levels[i].cp = levelSolve(&levels[i+1], CubeOpponent, -1.0, levels[i].cash)
	}
	return levels, count
}

// ── The leaf valuation for the search ───────────────────────────────────────

// Value is the cubeful value of one distribution, on the search's negating
// scale.
//
// state == nil values in money points per unit of cube; otherwise in match
// equity 2*MWC-1 through the redouble recursion, at state's own cube value.
// Both scales NEGATE between sides provided the caller mirrors owner (Owned
// <-> Opponent) and swaps state along with the perspective — exactly what an
// expectiminimax's recursion does at each ply. That antisymmetry is what
// lets a search carry cubeful values with the same negations it uses for
// cubeless ones.
//
// ok is false for a distribution or state that cannot be valued — refused,
// never approximated.
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

// Verdict applies the verdict table to whatever (eND, eDT, eDP) the caller
// hands in — money points, match MWC or exact table equities all read the
// same way. Exported so a cubeful rollout or an exact-reference benchmark
// can share the same four comparisons Decide uses, rather than keeping a
// second copy of one rule.
func Verdict(eND, eDT, eDP float64) CubeAction {
	eDouble := math.Min(eDT, eDP)
	switch {
	case eND > eDP && eND >= eDouble:
		return TooGood
	case eDT >= eDP:
		return DoublePass
	case eDouble > eND:
		return DoubleTake
	default:
		return NoDouble
	}
}

// Decision is what the player on roll should do, and the numbers that
// explain it.
type Decision struct {
	Action CubeAction
	// Equity of doubling and of not doubling, on the same scale, so the
	// caller can see the margin rather than only the verdict. A decision
	// that is right by 0.001 and one that is right by 0.5 are not the same
	// decision.
	EquityNoDouble float64
	EquityDouble   float64
	// EquityDoubleTake and EquityDoublePass are the two branches EquityDouble
	// is the minimum of — kept apart because a caller reporting a ND/DT/DP
	// table (the same shape XG and gnubg import) needs all three, not just
	// the one Verdict acted on.
	EquityDoubleTake float64
	EquityDoublePass float64
	// TakePoint is the opponent's take point at this state, for reporting.
	TakePoint float64
}

// Decide is the money or match-score cube decision.
//
// state == nil is a pure money game. Otherwise the decision is taken in
// match winning chance through the equity table — a different question with
// a different answer, since at 2-away/2-away a gammon wins the match and the
// whole doubling window moves. The score is NOT a correction applied to a
// money verdict; it replaces it.
//
// jacoby applies the Jacoby rule — gammons and backgammons do not count
// before the cube has been turned — only to the "don't double" branch, and
// only when it can actually matter: a centred cube in a money game. Once the
// cube has been turned (owner != CubeCentred) the flag is silently without
// effect, because Jacoby governs the game before the first double, not after
// it — and in a match the question does not arise at all: the equity table
// already prices gammons at the score, which is what Jacoby exists to
// approximate in money play.
//
// ok is false when the state is not evaluable.
func Decide(probs *[NumOutputs]float32, owner CubeOwner, state *MatchState, efficiency float64, jacoby bool) (Decision, bool) {
	in := CubeInputsFromProbs(probs)

	if state == nil {
		wND, lND := in.WinPoints, in.LosePoints
		if jacoby && owner == CubeCentred {
			wND, lND = 1.0, 1.0
		}

		eND := janowskiEquity(in.Win, wND, lND, owner, efficiency)
		eDT := 2.0 * janowskiEquity(in.Win, in.WinPoints, in.LosePoints, CubeOpponent, efficiency)
		eDP := 1.0

		// denom = WinPoints + LosePoints + efficiency/2 >= 2 given
		// CubeInputsFromProbs' >= 1 floors: never refuses in practice.
		tp, _ := TakePoint(in, CubeOwned, efficiency)

		action := NoDouble
		if owner != CubeOpponent {
			// A cube the opponent owns cannot be turned by the player on
			// roll: the verdict table presupposes doubling is an option, so
			// outside that precondition there is nothing to weigh.
			action = Verdict(eND, eDT, eDP)
		}

		return Decision{
			Action:           action,
			EquityNoDouble:   eND,
			EquityDouble:     math.Min(eDT, eDP),
			EquityDoubleTake: eDT,
			EquityDoublePass: eDP,
			TakePoint:        tp,
		}, true
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

	// The opponent's take point at the doubled stake, on the curve the
	// decision actually used — blended at this efficiency, bisected because
	// no closed form survives the score.
	tp := levelSolve(&levels[1], CubeOpponent, efficiency, eDP)

	// Two forced branches, and only one of them is a rule. The Crawford game
	// has no cube in play at all, flat fact. Post-Crawford gets NO special
	// case: the trailer's mandatory double and the leader's free drop fall
	// out of Verdict on their own, because metAfter already encodes the
	// post-Crawford table.
	action := NoDouble
	if !state.Crawford && owner != CubeOpponent {
		action = Verdict(eND, eDT, eDP)
	}

	return Decision{
		Action:           action,
		EquityNoDouble:   eND,
		EquityDouble:     math.Min(eDT, eDP),
		EquityDoubleTake: eDT,
		EquityDoublePass: eDP,
		TakePoint:        tp,
	}, true
}
