// SPDX-License-Identifier: MIT

package gammonnet

// THE REFERENTIAL BOUNDARY.
//
// At a match score gammonNet works on the MWC scale (t34-videau-spec §5), and
// the port keeps it: Decide returns MWC, Value and valueFromProbs return
// 2×MWC−1. Both are legitimate INTERNAL scales, since verdicts and rankings
// are invariant under an increasing affine map.
//
// Neither is what an engine PRINTS. XG and gnubg display normalised equity
// (gnubg's mwc2eq), ±1 for winning/losing the current cube outright:
//
//	eq = (2·mwc − cash − pass) / (cash − pass)
//
// with cash/pass the MWC after collecting/conceding the cube's own value dry.
// The two coincide ONLY at double-match-point. Elsewhere 2×MWC−1 is
// compressed by (cash − pass) — a factor 6.4 at 5-away/5-away — and shifted
// at a lopsided score; a raw MWC is not even centred on zero.
//
// ADR-0019: the conversion happens once, at the domain edge
// (EvaluatePosition, internal/gui's race bonus). The cube model and the
// search keep the C's scales and their gold files; everything blunderDB
// stores or shows is normalised equity, the units of imported analyses.

// minAnchorSpread guards the division below. A normal MET has cash > pass
// strictly (collecting points never lowers a match winning chance), so this
// only ever fires on a degenerate table — refused rather than divided by.
const minAnchorSpread = 1e-9

// EquityScale converts one position's internal gammonNet values into the
// normalised equity blunderDB stores and displays. Money play is the
// identity: money equities are already points, on the very scale normalised
// equity mimics.
//
// The anchors belong to the position (score and cube), so a scale is built
// once per position and applied to every equity it yields. Converting at the
// root is exact: the map is affine and the anchors never move within a
// search, and the opponent's anchors (1−pass, 1−cash) make it negate as the
// MWC scale does.
type EquityScale struct {
	cash  float64 // MWC after collecting the cube's value dry
	pass  float64 // MWC after conceding it dry
	match bool    // false ⇒ money play, every conversion is the identity
}

// NewEquityScale builds the scale for one position; state == nil is money.
// ok is false for an unevaluable state or degenerate anchors — never a fall
// back to an unconverted number.
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
