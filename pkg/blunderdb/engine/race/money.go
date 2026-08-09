package race

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
)

// Money holds the exact money-game cube data for the player on roll, in
// units of the current cube value. Gammonless domain (both sides have borne
// off ≥ 4 checkers whenever both armies fit an 11-checker database), no
// Jacoby.
type Money struct {
	CubeState  CubeState `json:"cube_state"`
	Cubeless   float64   `json:"cubeless"`
	NoDouble   float64   `json:"no_double"`   // continuation equity (cube against: the only equity)
	DoubleTake float64   `json:"double_take"` // 2 × plane 3
	DoublePass float64   `json:"double_pass"` // +1 by definition
	Verdict    Verdict   `json:"verdict,omitempty"`
}

// moneyEps absorbs the uint16 quantisation of the stored equities.
const moneyEps = 1e-9

// MoneyFromEntry reconstructs the money cube analysis from a database entry.
// Decision rule pinned against gnubg (160/160 fixture states): double iff
// DT ≥ ND and DP ≥ ND (weak inequalities — gnubg doubles on equity ties,
// where doubling costs nothing and punishes a wrong take); the opponent
// takes iff DT < DP.
func MoneyFromEntry(e Entry, state CubeState) Money {
	m := Money{
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
	double := m.DoubleTake >= m.NoDouble-moneyEps && m.DoublePass >= m.NoDouble-moneyEps
	if !double {
		m.Verdict = VerdictNoDouble
	} else if m.DoubleTake < m.DoublePass-moneyEps {
		m.Verdict = VerdictDoubleTake
	} else {
		m.Verdict = VerdictDoublePass
	}
	return m
}
