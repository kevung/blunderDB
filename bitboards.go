package main

// OccupancyMasks computes four 26-bit occupancy and point masks from a Board.
// Bit i is set if the condition holds for point index i (0=WhiteBar, 1-24=board, 25=BlackBar).
//
//	occ1 (Black occupancy): bit i set if Black has ≥1 checker at point i.
//	occ2 (White occupancy): bit i set if White has ≥1 checker at point i.
//	pt1  (Black point mask): bit i set if Black has ≥2 checkers at point i (a "made point").
//	pt2  (White point mask): bit i set if White has ≥2 checkers at point i.
//
// All four masks are returned as uint32 (26 bits fit, MSB unused).
func OccupancyMasks(b *Board) (occ1, occ2, pt1, pt2 uint32) {
	for i, p := range b.Points {
		if p.Checkers <= 0 || p.Color < 0 {
			continue
		}
		bit := uint32(1) << i
		if p.Color == Black {
			occ1 |= bit
			if p.Checkers >= 2 {
				pt1 |= bit
			}
		} else { // White
			occ2 |= bit
			if p.Checkers >= 2 {
				pt2 |= bit
			}
		}
	}
	return
}

// CheckerStructureMasks compiles a template Position (used as a board-pattern
// filter in the search UI) into four occupancy/point-mask requirements.
//
//	occ1Req: Black must have ≥1 checker on each set bit.
//	pt1Req:  Black must have ≥2 checkers on each set bit.
//	occ2Req: White must have ≥1 checker on each set bit.
//	pt2Req:  White must have ≥2 checkers on each set bit.
//
// tight is true when the template contains exact checker counts > 2, which
// require an additional Go-side check beyond the bitmask test.
func CheckerStructureMasks(filter Position) (occ1Req, pt1Req, occ2Req, pt2Req uint32, tight bool) {
	for i, p := range filter.Board.Points {
		if p.Checkers <= 0 || p.Color < 0 {
			continue
		}
		bit := uint32(1) << i
		if p.Color == Black {
			occ1Req |= bit
			if p.Checkers >= 2 {
				pt1Req |= bit
			}
			if p.Checkers > 2 {
				tight = true
			}
		} else { // White
			occ2Req |= bit
			if p.Checkers >= 2 {
				pt2Req |= bit
			}
			if p.Checkers > 2 {
				tight = true
			}
		}
	}
	return
}

// MatchesCheckerStructure reports whether board b satisfies the four mask
// requirements produced by CheckerStructureMasks. Call this for the fast
// bitmask screen; the caller is responsible for exact-count checks when tight=true.
func MatchesCheckerStructure(b *Board, occ1Req, pt1Req, occ2Req, pt2Req uint32) bool {
	occ1, occ2, pt1, pt2 := OccupancyMasks(b)
	return occ1&occ1Req == occ1Req &&
		pt1&pt1Req == pt1Req &&
		occ2&occ2Req == occ2Req &&
		pt2&pt2Req == pt2Req
}
