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
	// cashing. Before #193 (ADR-0020: "a cube decision has one shape"),
	// MoneyFromEntry's own 3-way ND/DT/DP comparison could never produce
	// this — it folded every such position into VerdictNoDouble, silently
	// losing the distinction — while gammonnet.CubeAction.TooGood could.
	// VerdictFromEquities below is the one rule both now share, so the
	// exact table names it exactly as the evaluated regime does. The app
	// already has this concept elsewhere (domain.TooGood,
	// engine/analysiscodec.go's BestCubeVerdict), so it is named here
	// rather than invented.
	VerdictTooGood Verdict = "too_good"
)

// CubeVerdict holds the cube data for the player on roll, in units of the
// current cube value: the exact-regime table's own referential is always
// money (MoneyFromEntry never reads a match score), but the SAME struct is
// also how internal/gui's evaluated-regime bonus reports gammonNet's
// verdict at a match score, on the normalised-equity scale ADR-0019 defines
// there — which is why this type is no longer called Money (#190/C.3: the
// name lied on three of its four equity fields as soon as a caller filled
// it in at a score, which predates this rename by a full ADR).
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

// VerdictFromEquities is the one cube-decision rule, ADR-0020's "one shape"
// carried one level further into the comparison itself (#193/C.6): given the
// three option equities on whatever scale the caller works in — money points
// here, gammonNet's own match MWC in gammonnet.Verdict's caller — it names
// the verdict, TooGood included, with a single four-way comparison.
//
// eps absorbs quantisation: MoneyFromEntry below passes moneyEps (the stored
// equities are uint16), gammonnet.Verdict passes 0 (exact floats, no
// quantisation to absorb). At eps == 0 this is exactly gammonNet's own
// former inline four-way switch (cube.go), moved here because gammonNet may
// import race (it already values leaves with race's own cube inputs
// elsewhere) but race must never import gammonNet — its own internal test
// files already import race for the exact-table comparison, and Go refuses
// the resulting cycle (see internal/gui/gammonnet_eval.go's longer note on
// the same constraint).
//
// The two branches other than TooGood are MoneyFromEntry's original 3-way
// rule, verbatim, pinned against gnubg on 160 fixture states
// (testdata/money_fixtures.json): double iff DT ≥ ND and DP ≥ ND (weak
// inequalities — gnubg doubles on equity ties, where doubling costs nothing
// and punishes a wrong take); the opponent takes iff DT < DP. TooGood only
// carves out the region that rule used to fold into NoDouble: nd is
// evaluable only when it is NOT at least dp (up to eps) — the same threshold
// the double/no-double branch already reads on the other side of the
// switch — so the two conditions never contest the same equity triple.
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

// MoneyFromEntry reconstructs the money cube analysis from a database entry.
// The verdict comes from VerdictFromEquities (#193/C.6) — before that, this
// function ran its own 3-way ND/DT/DP comparison inline and could never
// report VerdictTooGood; see the type doc comment above and the package's
// tests for what actually changes on real bearoff positions.
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
