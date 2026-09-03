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
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/race"
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
//
// THESE ARE THREE BRANCH COEFFICIENTS, NOT A PROPERTY OF THE POSITION, and
// that is a deliberate divergence from gnubg and XG, which index the cube
// efficiency by POSITION CLASS (contact 0.68, one-sided bearoff 0.6, race
// interpolated 0.6→0.7 on pip count) and never by the owner —
// docs/recherche/P6-videau-janowski.md documents theirs, ADR-0028 decides
// ours. gammonNet's spec §3 forbids borrowing another engine's constants, and
// each of the three above was fitted against a DIFFERENT column of the exact
// two-sided table. Two callers read one where the model asks for another —
// SearchConfig.CubeX at a mirrored leaf, and Decide's eDT below — and both
// match the C exactly; ADR-0028 measures the gap and says why the correction
// belongs upstream.
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
// (x1 <= x0, which a bisected breakpoint can produce at the extremes)
// returns its own endpoint rather than dividing by zero.
func segment(p, x0, y0, x1, y1 float64) float64 {
	if x1-x0 <= 0.0 {
		return y1
	}
	return y0 + (y1-y0)*((p-x0)/(x1-x0))
}

// janowskiEquity is the Janowski equity of one cube state, per unit of cube,
// at the given efficiency.
//
// dead is e(p), the dead branch verbatim. The live branch is piecewise
// linear across the WHOLE of [0, 1], tails included: it runs from (0, -L) to
// (1, +W), bending at the breakpoints the cube state puts in its way.
//
// THE TAILS ARE NOT PLATEAUX, and this is the whole of the TooGood verdict.
// Above CP_live a cube holder does not stop at the cash equivalent of +1: he
// plays the game on for the gammon, still holding the cube, and the curve
// rises to the average win W at p = 1. Below TP_live its mirror falls to -L.
// Flattening either tail — this file did, capping the top at max(1, e(p)),
// until ADR-0022 — prices the retained cube at zero and makes eND > +1
// impossible unless the CUBELESS equity already exceeds a point, which is to
// say it makes TooGood unreachable on every real position.
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
//
// Money's endpoints (0, -L) and (1, +W) become this level's own MWC anchors,
// loseAvg and winAvg, and its cash equivalents ±1 become cash and pass. The
// tails run to the anchors here for the same reason they do in money
// (ADR-0022): past the cash point the game is played on, not conceded, and
// the level is worth its winning anchor at p = 1.
//
// Monotone non-decreasing in p for each state — loseAvg <= pass <= cash <=
// winAvg holds by construction (conceding k dry points beats losing an
// average of k, 2k, 3k; collecting k is worse than winning that average), so
// every piece rises. That is the property levelSolve's bisection stands on.
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

// laneCurve est la courbe vive du niveau au-dessus, avec tout ce qui NE
// DÉPEND PAS de p sorti des soixante itérations : les deux dénominateurs de
// segment (x1−x0), les deux numérateurs (y1−y0), le drapeau « mort » et le
// choix de possession.
//
// C'EST UNE OPTIMISATION D'IMPLÉMENTATION, PAS UNE RÉVISION DU MODÈLE, et
// elle est propre à ce portage. gcc inline level_live dans la bissection de
// gammonNet ; le compilateur Go la juge trop complexe (`go build
// -gcflags=-m` : cannot inline levelLive), si bien qu'ici chaque pas payait
// un appel, un switch sur la possession et deux soustractions de segment —
// soixante fois par point de rupture, six points de rupture par valuation.
// Mesurée ici sur le poste isolé et entrelacé : ×1,24 à ×1,51 selon la taille
// de fratrie. L'ADR-0003 range ce genre de gain du côté implémentation, et il
// reste chez celui qui le porte : il n'a rien à faire remonter en amont, où
// il n'existe pas.
//
// L'ARITHMÉTIQUE EST INCHANGÉE, au bit près. y0 + (y1−y0)·((p−x0)/(x1−x0))
// s'écrit ici avec (y1−y0) et (x1−x0) calculés une fois : ce sont les mêmes
// soustractions, sur les mêmes valeurs, et elles ne dépendent pas de p. Ce
// qui serait faux, et n'est PAS fait, c'est précalculer la pente
// (y1−y0)/(x1−x0) — une division de moins, mais un résultat différent.
type laneCurve struct {
	dead    bool
	loseAvg float64
	winAvg  float64
	brk     float64 // cp sous CubeOwned, tp sous CubeOpponent
	mid     float64 // cash sous CubeOwned, pass sous CubeOpponent
	dLo     float64 // brk − 0
	nLo     float64 // mid − loseAvg
	dHi     float64 // 1 − brk
	nHi     float64 // winAvg − mid
}

// set prépare la courbe vive du niveau lv telle que la voit owner. Rend
// false pour CubeCentred, dont la courbe a trois segments : resolveLevels ne
// la demande jamais (tp vient de la possédée, cp de l'adverse), et un
// troisième segment sortirait `at` du budget d'inlining sans rien servir.
// L'appelant retombe alors sur levelLive.
func (c *laneCurve) set(lv *matchLevel, owner CubeOwner) bool {
	switch owner {
	case CubeOwned:
		c.brk, c.mid = lv.cp, lv.cash
	case CubeOpponent:
		c.brk, c.mid = lv.tp, lv.pass
	default:
		return false
	}
	c.dead = lv.dead
	c.loseAvg, c.winAvg = lv.loseAvg, lv.winAvg
	c.dLo, c.nLo = c.brk-0.0, c.mid-c.loseAvg
	c.dHi, c.nHi = 1.0-c.brk, c.winAvg-c.mid
	return true
}

// at est levelLive sur cette courbe, terme pour terme : le segment choisi,
// et lui seul, est calculé. C'est la forme que veut une bissection SÉRIELLE
// (levelSolve), où la chaîne est bornée par la latence — une division de plus
// y coûte, et la comparaison p <= brk, elle, devient prévisible dès que
// l'intervalle de bissection est tombé d'un côté de la rupture.
func (c *laneCurve) at(p float64) float64 {
	if c.dead {
		return (1.0-p)*c.loseAvg + p*c.winAvg
	}
	if p <= c.brk {
		if c.dLo <= 0.0 {
			return c.mid
		}
		return c.loseAvg + c.nLo*((p-0.0)/c.dLo)
	}
	if c.dHi <= 0.0 {
		return c.winAvg
	}
	return c.mid + c.nHi*((p-c.brk)/c.dHi)
}

// cubeSolveLifted éteint la levée. Il n'existe QUE pour la mesure — sa valeur
// par défaut est la levée allumée, et rien dans l'application ne le pose. Il
// est ici plutôt que sur le Searcher parce que levelSolve est trois appels
// sous Value et n'a aucun chemin par lequel un chercheur pourrait le lui
// transmettre. Il est lu une fois par point de rupture, pas une fois par pas,
// donc il ne coûte rien ; et il n'est écrit qu'entre deux recherches, jamais
// pendant, ce qui est la même règle que Counters.
var cubeSolveLifted = true

// levelSolve finds the p where a monotone level curve crosses target: the
// functions are piecewise-linear and monotone, so bisection suffices.
// blend < 0 bisects the fully-live curve (breakpoint resolution inside the
// recursion); otherwise the curve blended at that efficiency (the reported
// take point).
//
// La courbe est préparée une fois avant les soixante pas (laneCurve) : les
// dénominateurs et numérateurs des segments ne dépendent pas de p, et le
// compilateur Go refuse d'inliner levelLive, si bien que chaque pas payait un
// appel et un switch. Même arithmétique, mêmes bits — la levée est le pendant
// scalaire de celle du lot, et elle est mesurée séparément d'elle : sans
// cette levée, le lot rendrait ×1,2 au lieu de ce que le §4 des mesures
// rapporte, et une bonne moitié de ce gain n'aurait rien eu à voir avec
// l'entrelacement des voies.
func levelSolve(lv *matchLevel, owner CubeOwner, blend, target float64) float64 {
	var c laneCurve
	lifted := cubeSolveLifted && c.set(lv, owner)

	low, high := 0.0, 1.0
	for i := 0; i < 60; i++ {
		mid := 0.5 * (low + high)
		var live float64
		if lifted {
			live = c.at(mid)
		} else {
			live = levelLive(lv, mid, owner)
		}
		value := live
		if blend >= 0.0 {
			value = (1.0-blend)*levelDead(lv, mid) + blend*live
		}
		below := value < target
		if below {
			low = mid
		}
		if !below {
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
// Coupé en deux (ADR-0003 / spec §7.1) parce que les deux moitiés n'ont pas
// la même nature : les ANCRES d'un niveau ne dépendent que de ce candidat,
// les POINTS DE RUPTURE dépendent du niveau au-dessus et se résolvent par
// bissection. C'est la seconde moitié qui se met en lot, et la coupe est
// tout ce que le lot demande à ce fichier.
func buildLevels(state MatchState, outcomes [numOutcomes]float64) ([maxCubeLevels]matchLevel, int) {
	var levels [maxCubeLevels]matchLevel
	count := buildLevelAnchors(state, outcomes, &levels)
	if count == 0 {
		return levels, 0
	}
	resolveLevels(&levels, count)
	return levels, count
}

// buildLevelAnchors remplit les ancres de chaque niveau de la chaîne —
// loseAvg, winAvg, pass, cash — et laisse les points de rupture à leurs
// bornes triviales (tp = 0, cp = 1). Rend le nombre de niveaux, ou 0 pour
// refuser.
//
// LA FORME DE LA CHAÎNE NE DÉPEND QUE DE state : le nombre de niveaux, les
// enjeux et lequel est mort se lisent dans state.Cube et les deux away
// scores, jamais dans outcomes. C'est ce qui permet à toutes les voies d'un
// lot de résoudre le même niveau au même moment — et cubeValueBatch le
// VÉRIFIE au lieu de le supposer.
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

// resolveLevels résout les points de rupture, les plus profonds d'abord, si
// bien que chaque bissection vise un niveau 2k déjà complet : la forme
// itérative de la récursion, chaque (état, k) calculé une fois depuis le cas
// de base.
func resolveLevels(levels *[maxCubeLevels]matchLevel, count int) {
	for i := count - 2; i >= 0; i-- {
		levels[i].tp = levelSolve(&levels[i+1], CubeOwned, -1.0, levels[i].pass)
		levels[i].cp = levelSolve(&levels[i+1], CubeOpponent, -1.0, levels[i].cash)
	}
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
	// The Crawford game has no cube in play at all — the same flat fact
	// Decide applies to its verdict, and it applies to the VALUE too: the
	// chain above prices doublings the rules forbid, and walking it here
	// valued the opening at 4-away/1-away Crawford at +0.68 against +0.16
	// for the dead cube (gnubg cubeful == gnubg cubeless in that game,
	// probe of 2026-09-02). Dead value at the current stake, whoever "owns"
	// a cube nobody can turn. Mirrors gn_cube.c.
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

// Verdict applies the verdict table to whatever (eND, eDT, eDP) the caller
// hands in — money points, match MWC or exact table equities all read the
// same way. Exported so a cubeful rollout or an exact-reference benchmark
// can share the same four comparisons Decide uses, rather than keeping a
// second copy of one rule.
//
// The comparison itself lives in race.VerdictFromEquities (#193/C.6: ADR-0020
// said a cube decision has one shape; before this it had two — this table
// here, and MoneyFromEntry's own 3-way copy that could never produce
// TooGood). It moved to race rather than the other way round because
// gammonNet may import race — nothing there depends back on gammonNet — while
// race must never import gammonNet: its own internal test files already
// import race for the exact-table comparison, and Go refuses the resulting
// cycle. eps is 0 here: this package's inputs are exact floats, with no
// uint16 quantisation to absorb (contrast MoneyFromEntry's moneyEps).
func Verdict(eND, eDT, eDP float64) CubeAction {
	return cubeActionFromVerdict(race.VerdictFromEquities(eND, eDT, eDP, 0))
}

// cubeActionFromVerdict is a straight rename between the two enums that name
// the same four outcomes — race.Verdict (shared with the exact-table
// verdict and the wire payload internal/gui builds) and gammonnet.CubeAction
// (this package's own internal currency, threaded through Decision.Action).
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
// There is no beaver parameter, deliberately, not by omission (#193/C.6):
// domain.Position.HasBeaver is stored and hashed like HasJacoby, but nothing
// in this decision reads it. Beaver ("take → beaver": the taker immediately
// redoubles, keeping the cube) changes who owns the cube and at what value
// the SAME instant a take is decided, which this function cannot express —
// Decide answers "double, take, or pass" for a single cube state, not a
// sequence of two decisions at two cube values. Modelling it (the plan's own
// candidate rule: money, centred cube, taker redoubles when eDT > +1) is a
// gammonNet spec question first (its own spec §2), same as movefilters and
// distillation in C.13 — until that lands, HasBeaver is read nowhere past
// storage, and is exactly as decorative as this comment says, not a bug this
// fiche is silently leaving behind.
//
// ok is false when the state is not evaluable.
func Decide(probs *[NumOutputs]float32, owner CubeOwner, state *MatchState, efficiency float64, jacoby bool) (Decision, bool) {
	in := CubeInputsFromProbs(probs)

	if state == nil {
		wND, lND := in.WinPoints, in.LosePoints
		if jacoby && owner == CubeCentred {
			wND, lND = 1.0, 1.0
		}

		// eDT is the OPPONENT branch — after a take he holds the doubled
		// cube — but it is priced at the CURRENT owner's efficiency, the one
		// the caller passed. Under DefaultEfficiency's per-branch fit the
		// coherent value would be DefaultEfficiency(CubeOpponent); gn_cube.c
		// :754 passes the caller's just the same, so this stays until
		// gammonNet moves. Measured on 604 real cube decisions at a score:
		// the take point shifts 0.0011 on average (0.0069 at worst) and NO
		// verdict flips. #192/C.5, ADR-0028 — the match branch below and its
		// levelSolve take point have the same shape.
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
	eDouble := math.Min(eDT, eDP)
	if state.Crawford {
		// No cube in play: the position is worth its dead value (see
		// Value), and there is no double branch to price — it is reported
		// worth exactly what not doubling is, so the panel's "best of the
		// two" subtraction reads a zero-cost non-option, never a "missed
		// double" in a game where doubling is illegal. Mirrors gn_cube.c.
		eND = levelDead(&levels[0], in.Win)
		eDT = eND
		eDouble = eND
	}

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
		EquityDouble:     eDouble,
		EquityDoubleTake: eDT,
		EquityDoublePass: eDP,
		TakePoint:        tp,
	}, true
}
