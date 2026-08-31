// SPDX-License-Identifier: MIT

package gammonnet

// THE REFERENTIAL BOUNDARY.
//
// gammonNet works, at a match score, on the MWC scale — its spec says so
// (docs/specs/t34-videau-spec.md §5, "Même mécanique, sur l'échelle MWC"),
// and the port keeps that faithfully: Decide returns match winning chances,
// Value and valueFromProbs return 2×MWC−1. Both are legitimate INTERNAL
// scales — a verdict and a move ranking are invariant under any increasing
// affine map of the MWC, which is exactly what those two are.
//
// Neither is what an engine PRINTS. XG and gnubg display normalised equity
// (gnubg's mwc2eq): the MWC mapped so that winning the current cube outright
// is +1 and losing it outright is −1,
//
//	eq = (2·mwc − cash − pass) / (cash − pass)
//
// with cash/pass the MWC after collecting/conceding the cube's own value dry.
// The two coincide ONLY at double-match-point, where cash = 1 and pass = 0.
// Everywhere else 2×MWC−1 is compressed by (cash − pass) — a factor 6.4 at
// 5-away/5-away — and, as soon as the score is lopsided, shifted as well,
// since cash + pass is then no longer 1. A raw MWC is not even centred on
// zero. Displayed as equities, both are wrong, and wrong by different
// amounts in the same panel.
//
// That distinction was lost upstream: gn_met.h presents 2×MWC−1 as "the
// equivalent-to-money scale that engines print", ADR-0016 repeated it, and
// the port inherited it. ADR-0019 fixes it here — the conversion belongs at
// the domain edge (EvaluatePosition and internal/gui's race bonus), so the
// cube model and the search keep the C's own scales and their gold files,
// while everything blunderDB stores or shows is on the one scale the rest of
// the application already speaks: normalised equity, the same units an
// imported XG or GNUbg analysis sits in, in the same column.

// minAnchorSpread guards the division below. A normal MET has cash > pass
// strictly (collecting points never lowers a match winning chance), so this
// only ever fires on a degenerate table — refused rather than divided by.
const minAnchorSpread = 1e-9

// EquityScale converts one position's internal gammonNet values into the
// normalised equity blunderDB stores and displays. Money play is the
// identity: money equities are already points, on the very scale normalised
// equity mimics.
//
// The anchors are a property of the position (its score and its cube), not
// of any one number, so a scale is built once per evaluated position and
// applied to every equity that comes out of it — the cube decision, the
// candidate moves, the cubeless fact. That is also why converting at the
// root is exact rather than an approximation: within one search the score
// and cube never move, and the map is affine, so converting each leaf and
// converting the root agree — including through the search's per-ply
// negation, since the opponent's own anchors are (1−pass, 1−cash) and the
// normalised equity therefore negates exactly as the MWC scale does.
type EquityScale struct {
	cash  float64 // MWC after collecting the cube's value dry
	pass  float64 // MWC after conceding it dry
	match bool    // false ⇒ money play, every conversion is the identity
}

// NewEquityScale builds the scale for one position. state == nil is money
// play. ok is false when the state cannot be evaluated at all (the same
// refusal Decide makes) or when the MET's anchors are degenerate — never a
// silent fall back to an unconverted number, which would look plausible and
// be wrong by a factor of five.
func NewEquityScale(state *MatchState) (EquityScale, bool) {
	if state == nil {
		return EquityScale{}, true
	}
	cash, okCash := metAfter(*state, state.Cube, true)
	pass, okPass := metAfter(*state, state.Cube, false)
	if !okCash || !okPass || cash-pass < minAnchorSpread {
		return EquityScale{}, false
	}
	return EquityScale{cash: cash, pass: pass, match: true}, true
}

// IsMatch reports whether this scale actually converts anything — true at a
// match score, false in money play, where every method is the identity.
func (s EquityScale) IsMatch() bool { return s.match }

// FromDecision converts a number Decide returned: money points per unit of
// cube in money play, a match winning chance at a score.
func (s EquityScale) FromDecision(v float64) float64 {
	if !s.match {
		return v
	}
	return (2.0*v - s.cash - s.pass) / (s.cash - s.pass)
}

// FromSearch converts a number the search returned — Value, valueFromProbs,
// CubelessValue, Candidate.Equity: money points in money play, 2×MWC−1 at a
// score.
func (s EquityScale) FromSearch(v float64) float64 {
	if !s.match {
		return v
	}
	return (v + 1.0 - s.cash - s.pass) / (s.cash - s.pass)
}
