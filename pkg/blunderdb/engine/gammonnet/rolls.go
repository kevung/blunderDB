package gammonnet

// NumRolls is the number of distinct rolls: (1,1) and (2,1) are rolls, (1,2) is
// the same roll as (2,1) counted twice.
const NumRolls = 21

type diceRoll struct {
	d1, d2 int8
	weight float64
}

// buildRolls returns the 21 distinct rolls in the reference's order —
// (1,1)(1,2)…(1,6)(2,2)…(2,6)(3,3)…(6,6) — with the weights it uses.
//
// The order is part of the contract, not a detail: the search accumulates
// `sum += weight * best` over this table in ascending index order, in float64,
// and a different order gives a different last bit.
func buildRolls() [NumRolls]diceRoll {
	var out [NumRolls]diceRoll
	n := 0
	for a := int8(1); a <= 6; a++ {
		for b := a; b <= 6; b++ {
			w := 2.0 / 36.0
			if a == b {
				w = 1.0 / 36.0
			}
			out[n] = diceRoll{d1: a, d2: b, weight: w}
			n++
		}
	}
	return out
}
