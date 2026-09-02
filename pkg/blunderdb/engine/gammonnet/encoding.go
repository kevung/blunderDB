// SPDX-License-Identifier: MIT

package gammonnet

// The 196-feature perspective encoding, reproduced from gammonNet's
// gn_encoding.c, which itself reproduces encoding.py's `perspective196`.
//
// The network was trained on this layout, so "reproduced" means bit for bit,
// not "equivalent".
//
//	MY block (98)    24 points × 4 thermometer units      = 96
//	                 my bar   × 0.5                       =  1
//	                 my off   / 15.0                      =  1
//	OPP block (98)   the same, for the opponent           = 98
//	                                                total = 196
//
// Always from the ON-ROLL player's point of view. When Black is on roll the
// point indices are MIRRORED (index i becomes 23-i), so "my home board" always
// occupies the same features. The network learns one function: P(the on-roll
// player wins | board).
//
// The four cube inputs of the cubeful variants are deliberately absent: the
// model this port retains is cubeless.
const (
	// NumFeatures is the network's input width.
	NumFeatures = 196

	featuresPerPoint = 4
	myBlockOffset    = 0
	myBarIndex       = 96
	myOffIndex       = 97
	oppBlockOffset   = 98
	oppBarIndex      = 194
	oppOffIndex      = 195

	barScale = 0.5
	offScale = 1.0 / 15.0
)

// Encode writes the 196 features of p, seen by the player on turn, into out.
//
// It reports false, leaving out untouched, when the position is not
// structurally valid — refused, never approximated.
func Encode(p *Position, out *[NumFeatures]float32) bool {
	if !p.Valid() {
		return false
	}
	encodeLegal(p, out)
	return true
}

// encodeLegal is Encode on a position the caller already knows is legal.
//
// Valid() is not cheap — two passes over the twenty-four points plus two
// checker counts — and it was HALF of Encode: 91 ns for the whole encoding,
// 45 ns for the validation alone. A 2-ply decision encodes some fifty
// thousand positions, so the search paid that half fifty thousand times to
// re-establish something it had established by construction.
//
// By construction, precisely: a node's own position is validated by
// Generator.LegalPlays before a single play is generated, and every play's
// Result is that position with checkers moved — apply cannot invent or lose
// one. The two entry points that take a position from outside (Searcher.Plays
// and Searcher.Probs) validate it themselves, once, before the recursion
// starts. Everything in between is internal.
//
// Encode keeps its validation. A caller outside this package hands in a
// position this package did not build, and refusing it is the contract
// (network.go's EvaluatePosition, and every test).
func encodeLegal(p *Position, out *[NumFeatures]float32) {
	*out = [NumFeatures]float32{}

	me := p.Turn
	opponent := uint8(Black)
	if me == Black {
		opponent = White
	}

	for i := 0; i < NumPoints; i++ {
		n := p.Points[i]
		if n == 0 {
			continue
		}
		// Where this physical point lands in the feature vector. White reads
		// the board in index order; Black reads it mirrored, so that its home
		// board occupies the same features as White's does. Get this backwards
		// and nothing crashes — the evaluations simply stop meaning what they
		// say.
		slot := i
		if me != White {
			slot = NumPoints - 1 - i
		}

		count := int(n)
		if count < 0 {
			count = -count
		}
		if (n > 0) == (me == White) {
			encodeCheckers(out, myBlockOffset+slot*featuresPerPoint, count)
		} else {
			encodeCheckers(out, oppBlockOffset+slot*featuresPerPoint, count)
		}
	}

	// Computed in float64 and rounded once, as the C does: the scales are
	// double literals there, so `off * (1.0/15.0)` is a double multiply
	// followed by a single narrowing. Doing the division in float32 instead
	// would round twice and diverge in the last bit.
	out[myBarIndex] = float32(float64(p.Bar[me]) * barScale)
	out[myOffIndex] = float32(float64(p.Off[me]) * offScale)
	out[oppBarIndex] = float32(float64(p.Bar[opponent]) * barScale)
	out[oppOffIndex] = float32(float64(p.Off[opponent]) * offScale)
}

// encodeCheckers writes the 4-unit thermometer: 1, 2, 3, then (n-3)/2.
func encodeCheckers(x *[NumFeatures]float32, offset, count int) {
	if count >= 1 {
		x[offset] = 1
	}
	if count >= 2 {
		x[offset+1] = 1
	}
	if count >= 3 {
		x[offset+2] = 1
		if count >= 4 {
			x[offset+3] = float32(float64(count-3) * 0.5)
		}
	}
}
