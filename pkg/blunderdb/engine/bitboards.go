package engine

import "github.com/kevung/blunderdb/pkg/blunderdb/domain"

// OccupancyMasks computes four 26-bit occupancy and point masks from a Board.
// Bit i is set if the condition holds for point index i (0=WhiteBar, 1-24=board, 25=BlackBar).
//
//	occ1 (Black occupancy): bit i set if Black has ≥1 checker at point i.
//	occ2 (White occupancy): bit i set if White has ≥1 checker at point i.
//	pt1  (Black point mask): bit i set if Black has ≥2 checkers at point i (a "made point").
//	pt2  (White point mask): bit i set if White has ≥2 checkers at point i.
//
// All four masks are returned as uint32 (26 bits fit, MSB unused).
func OccupancyMasks(b *domain.Board) (occ1, occ2, pt1, pt2 uint32) {
	for i, p := range b.Points {
		if p.Checkers <= 0 || p.Color < 0 {
			continue
		}
		bit := uint32(1) << i
		if p.Color == domain.Black {
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
func CheckerStructureMasks(filter domain.Position) (occ1Req, pt1Req, occ2Req, pt2Req uint32, tight bool) {
	for i, p := range filter.Board.Points {
		if p.Checkers <= 0 || p.Color < 0 {
			continue
		}
		bit := uint32(1) << i
		if p.Color == domain.Black {
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

// ExclusionMasks compiles an exclusion template Position ("Sauf") into bitmasks
// for the OR-semantics pre-filter: a position is rejected if it contains ANY of
// the excluded elements. Each excluded point contributes to exactly one mask:
//
//	single1/single2: points where the color must have ≥1 checker (template count 1)
//	                 → tested against occupancy_1/2.
//	made1/made2:     points where the color must have ≥2 checkers (template count 2)
//	                 → tested against point_mask_1/2.
//
// Points with a template count >2 are NOT represented (a 26-bit mask cannot
// express an exact ≥3 threshold); those are left to the authoritative Go-side
// check (Position.ContainsAnyCheckerOf). The SQL pre-filter keeps a position when
// (occupancy_1 & single1)=0 AND (point_mask_1 & made1)=0 AND (…color 2…), which is
// a valid over-approximation of "kept".
func ExclusionMasks(filter domain.Position) (single1, made1, single2, made2 uint32) {
	for i, p := range filter.Board.Points {
		if p.Checkers <= 0 || p.Color < 0 {
			continue
		}
		bit := uint32(1) << i
		switch p.Color {
		case domain.ExcludeEmpty:
			// "Must be empty": reject if either colour occupies the point.
			single1 |= bit
			single2 |= bit
		case domain.Black:
			switch p.Checkers {
			case 1:
				single1 |= bit
			case 2:
				made1 |= bit
			}
		default: // White
			switch p.Checkers {
			case 1:
				single2 |= bit
			case 2:
				made2 |= bit
			}
		}
	}
	return
}

// MatchesCheckerStructure reports whether board b satisfies the four mask
// requirements produced by CheckerStructureMasks. Call this for the fast
// bitmask screen; the caller is responsible for exact-count checks when tight=true.
func MatchesCheckerStructure(b *domain.Board, occ1Req, pt1Req, occ2Req, pt2Req uint32) bool {
	occ1, occ2, pt1, pt2 := OccupancyMasks(b)
	return occ1&occ1Req == occ1Req &&
		pt1&pt1Req == pt1Req &&
		occ2&occ2Req == occ2Req &&
		pt2&pt2Req == pt2Req
}
