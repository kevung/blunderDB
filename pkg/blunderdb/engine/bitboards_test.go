package engine

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// initialBoard returns the standard starting backgammon position.
// Black (0) moves from 24→1. White (1) moves from 1→24.
// Board indices 0=WhiteBar, 1..24=board points, 25=BlackBar.
func initialBoard() domain.Board {
	var b domain.Board
	// Black checkers
	b.Points[24] = domain.Point{Checkers: 2, Color: domain.Black}
	b.Points[13] = domain.Point{Checkers: 5, Color: domain.Black}
	b.Points[8] = domain.Point{Checkers: 3, Color: domain.Black}
	b.Points[6] = domain.Point{Checkers: 5, Color: domain.Black}
	// White checkers
	b.Points[1] = domain.Point{Checkers: 2, Color: domain.White}
	b.Points[12] = domain.Point{Checkers: 5, Color: domain.White}
	b.Points[17] = domain.Point{Checkers: 3, Color: domain.White}
	b.Points[19] = domain.Point{Checkers: 5, Color: domain.White}
	return b
}

func TestOccupancyMasks_InitialPosition(t *testing.T) {
	b := initialBoard()
	occ1, occ2, pt1, pt2 := OccupancyMasks(&b)

	// Black (occ1) should have bits at 6, 8, 13, 24.
	wantOcc1 := uint32(1<<6 | 1<<8 | 1<<13 | 1<<24)
	if occ1 != wantOcc1 {
		t.Errorf("occ1 got 0x%08x, want 0x%08x", occ1, wantOcc1)
	}

	// White (occ2) should have bits at 1, 12, 17, 19.
	wantOcc2 := uint32(1<<1 | 1<<12 | 1<<17 | 1<<19)
	if occ2 != wantOcc2 {
		t.Errorf("occ2 got 0x%08x, want 0x%08x", occ2, wantOcc2)
	}

	// Black point mask: all Black points have ≥2 checkers (min is 3 on point 8).
	wantPt1 := wantOcc1
	if pt1 != wantPt1 {
		t.Errorf("pt1 got 0x%08x, want 0x%08x", pt1, wantPt1)
	}

	// White point mask: all White points have ≥2 checkers (min is 3 on point 17).
	wantPt2 := wantOcc2
	if pt2 != wantPt2 {
		t.Errorf("pt2 got 0x%08x, want 0x%08x", pt2, wantPt2)
	}
}

func TestOccupancyMasks_SingleBlot(t *testing.T) {
	// A blot (1 checker) should appear in occ but not in pt.
	var b domain.Board
	b.Points[10] = domain.Point{Checkers: 1, Color: domain.Black}

	occ1, occ2, pt1, pt2 := OccupancyMasks(&b)

	if occ1 != 1<<10 {
		t.Errorf("occ1 got 0x%08x, want 0x%08x", occ1, uint32(1<<10))
	}
	if pt1 != 0 {
		t.Errorf("pt1 should be 0 for a blot, got 0x%08x", pt1)
	}
	if occ2 != 0 || pt2 != 0 {
		t.Errorf("occ2/pt2 should be zero")
	}
}

func TestOccupancyMasks_FivePrime(t *testing.T) {
	// Black 5-prime on points 11-15 — each point with 2 checkers.
	var b domain.Board
	for i := 11; i <= 15; i++ {
		b.Points[i] = domain.Point{Checkers: 2, Color: domain.Black}
	}

	_, _, pt1, _ := OccupancyMasks(&b)

	// Bits 11..15 must all be set in pt1.
	const primeMask = uint32(1<<11 | 1<<12 | 1<<13 | 1<<14 | 1<<15) // 0xF800
	if pt1&primeMask != primeMask {
		t.Errorf("5-prime not detected: pt1=0x%08x, expected bits 11-15 set (mask 0x%08x)", pt1, primeMask)
	}
}

func TestCheckerStructureMasks_RoundTrip(t *testing.T) {
	// Build a filter with Black anchor on point 20 (2 checkers).
	filter := domain.Position{}
	filter.Board.Points[20] = domain.Point{Checkers: 2, Color: domain.Black}

	occ1Req, pt1Req, occ2Req, pt2Req, tight := CheckerStructureMasks(filter)

	if tight {
		t.Error("tight should be false for exactly-2 checkers")
	}

	// A board that satisfies the filter.
	var b domain.Board
	b.Points[20] = domain.Point{Checkers: 2, Color: domain.Black}
	b.Points[5] = domain.Point{Checkers: 2, Color: domain.White}

	if !MatchesCheckerStructure(&b, occ1Req, pt1Req, occ2Req, pt2Req) {
		t.Error("board satisfying the filter should match")
	}

	// A board that doesn't (Black blot on 20, not a point).
	var b2 domain.Board
	b2.Points[20] = domain.Point{Checkers: 1, Color: domain.Black}

	if MatchesCheckerStructure(&b2, occ1Req, pt1Req, occ2Req, pt2Req) {
		t.Error("board with blot should not match pt1Req")
	}
}

func TestCheckerStructureMasks_TightFlag(t *testing.T) {
	// Template with 3 Black checkers → tight should be true.
	filter := domain.Position{}
	filter.Board.Points[7] = domain.Point{Checkers: 3, Color: domain.Black}

	_, _, _, _, tight := CheckerStructureMasks(filter)
	if !tight {
		t.Error("tight should be true when template has >2 checkers on a point")
	}
}
