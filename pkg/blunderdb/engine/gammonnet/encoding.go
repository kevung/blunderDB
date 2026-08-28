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

	return true
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
