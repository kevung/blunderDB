package race

import "math"

// CubeState describes the doubling-cube situation from the point of view of
// the player on roll.
type CubeState string

const (
	CubeCentered CubeState = "centered"
	CubeOwned    CubeState = "owned"   // player on roll owns the cube
	CubeAgainst  CubeState = "against" // opponent owns the cube
)

// Verdict is the money cube recommendation for the player on roll. Empty when
// the cube is against (no decision to make).
type Verdict string

const (
	VerdictNoDouble   Verdict = "no_double"
	VerdictDoubleTake Verdict = "double_take"
	VerdictDoublePass Verdict = "double_pass"
	// VerdictTooGood is "too good to double": playing on is worth more than
	// cashing. The exact table names it as the evaluated regime does, through
	// VerdictFromEquities (ADR-0020).
	VerdictTooGood Verdict = "too_good"
)

// CubeVerdict holds the cube data for the player on roll, in units of the
// current cube value: money for the exact-regime table (MoneyFromEntry never
// reads a match score), normalised equity (ADR-0019) when internal/gui's
// evaluated regime fills it at a match score.
type CubeVerdict struct {
	CubeState  CubeState `json:"cube_state"`
	Cubeless   float64   `json:"cubeless"`
	NoDouble   float64   `json:"no_double"`   // continuation equity (cube against: the only equity)
	DoubleTake float64   `json:"double_take"` // 2 × plane 3
	DoublePass float64   `json:"double_pass"` // +1 by definition
	Verdict    Verdict   `json:"verdict,omitempty"`
}

// moneyEps absorbs the uint16 quantisation of the stored equities.
const moneyEps = 1e-9

// VerdictFromEquities is the one cube-decision rule (ADR-0020): given the three
// option equities on the caller's scale — money points here, gammonNet's match
// MWC in gammonnet.Verdict — it names the verdict, TooGood included.
//
// eps absorbs quantisation: moneyEps for the uint16-stored table, 0 for
// gammonNet's exact floats. It lives in race because gammonnet imports race
// and the reverse would cycle.
//
// Outside TooGood the rule is pinned against gnubg on 160 fixture states
// (testdata/money_fixtures.json): double iff DT ≥ ND and DP ≥ ND (weak — gnubg
// doubles on ties, where doubling costs nothing and punishes a wrong take);
// take iff DT < DP. TooGood requires nd > dp (up to eps), the threshold the
// double branch reads from the other side, so no triple matches both.
func VerdictFromEquities(nd, dt, dp, eps float64) Verdict {
	double := math.Min(dt, dp)
	switch {
	case nd > dp+eps && nd >= double-eps:
		return VerdictTooGood
	case dt >= nd-eps && dp >= nd-eps:
		if dt < dp-eps {
			return VerdictDoubleTake
		}
		return VerdictDoublePass
	default:
		return VerdictNoDouble
	}
}

// MoneyFromEntry reconstructs the money cube analysis from a database entry,
// the verdict from VerdictFromEquities.

func MoneyFromEntry(e Entry, state CubeState) CubeVerdict {
	m := CubeVerdict{
		CubeState:  state,
		Cubeless:   e.Cubeless,
		DoubleTake: 2 * e.Against,
		DoublePass: 1.0,
	}
	switch state {
	case CubeOwned:
		m.NoDouble = e.OwnedND
	case CubeAgainst:
		// No decision: the only meaningful equity is the continuation with
		// the cube against; DT/DP describe the opponent's future recube and
		// are not a decision of ours — leave them for display consistency.
		m.NoDouble = e.Against
		return m
	default:
		m.CubeState = CubeCentered
		m.NoDouble = e.CenteredND
	}
	m.Verdict = VerdictFromEquities(m.NoDouble, m.DoubleTake, m.DoublePass, moneyEps)
	return m
}
