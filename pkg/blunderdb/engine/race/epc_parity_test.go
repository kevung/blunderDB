package race

import (
	"math/rand"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// legacyComputeEPC is the pre-unification implementation copied verbatim
// (modulo variable names) from database/db_epc.go, kept here as an oracle so
// the unified ComputeEPC provably preserves its behaviour. The legacy server
// copy (internal/server computeEPCSide) computed the same values; the only
// intentional divergence of the unified code from *this* oracle is
// CheckerCount, which now counts the player's whole army (server semantics)
// instead of home-board checkers only — the GUI never displayed it.
func legacyComputeEPC(position domain.Position) map[string]interface{} {
	var bottomBoard [6]int
	bottomTotal := 0
	bottomAllInHome := true
	for i := 0; i < 6; i++ {
		pt := position.Board.Points[i+1]
		if pt.Color == domain.Black {
			bottomBoard[i] = pt.Checkers
			bottomTotal += pt.Checkers
		}
	}
	for i := 7; i <= 24; i++ {
		pt := position.Board.Points[i]
		if pt.Color == domain.Black && pt.Checkers > 0 {
			bottomAllInHome = false
			break
		}
	}
	if position.Board.Points[domain.BlackBar].Color == domain.Black && position.Board.Points[domain.BlackBar].Checkers > 0 {
		bottomAllInHome = false
	}

	var topBoard [6]int
	topTotal := 0
	topAllInHome := true
	for i := 0; i < 6; i++ {
		pt := position.Board.Points[24-i]
		if pt.Color == domain.White {
			topBoard[i] = pt.Checkers
			topTotal += pt.Checkers
		}
	}
	for i := 1; i <= 18; i++ {
		pt := position.Board.Points[i]
		if pt.Color == domain.White && pt.Checkers > 0 {
			topAllInHome = false
			break
		}
	}
	if position.Board.Points[domain.WhiteBar].Color == domain.White && position.Board.Points[domain.WhiteBar].Checkers > 0 {
		topAllInHome = false
	}

	result := map[string]interface{}{
		"bottomEPC":       nil,
		"topEPC":          nil,
		"bottomAllInHome": bottomAllInHome,
		"topAllInHome":    topAllInHome,
	}
	if bottomAllInHome && bottomTotal > 0 {
		if epc, err := engine.ComputeEPC(bottomBoard); err == nil {
			result["bottomEPC"] = epc
		}
	}
	if topAllInHome && topTotal > 0 {
		if epc, err := engine.ComputeEPC(topBoard); err == nil {
			result["topEPC"] = epc
		}
	}
	return result
}

// place drops n checkers of color on point idx if the point is empty or
// already that color; it returns how many were actually placed.
func place(b *domain.Board, idx, color, n int) int {
	if n <= 0 {
		return 0
	}
	if b.Points[idx].Checkers > 0 && b.Points[idx].Color != color {
		return 0
	}
	b.Points[idx].Color = color
	b.Points[idx].Checkers += n
	return n
}

func randomBoard(rng *rand.Rand, blackHomeOnly, whiteHomeOnly, withBars bool) domain.Board {
	var b domain.Board
	for i := range b.Points {
		b.Points[i] = domain.Point{Color: -1, Checkers: 0}
	}
	remB, remW := rng.Intn(16), rng.Intn(16)
	for remB > 0 {
		var idx int
		if blackHomeOnly {
			idx = 1 + rng.Intn(6)
		} else {
			idx = 1 + rng.Intn(24)
		}
		remB -= place(&b, idx, domain.Black, 1+rng.Intn(min(3, remB)))
	}
	for remW > 0 {
		var idx int
		if whiteHomeOnly {
			idx = 19 + rng.Intn(6)
		} else {
			idx = 1 + rng.Intn(24)
		}
		remW -= place(&b, idx, domain.White, 1+rng.Intn(min(3, remW)))
	}
	if withBars {
		if rng.Intn(2) == 0 {
			place(&b, domain.BlackBar, domain.Black, 1)
		}
		if rng.Intn(2) == 0 {
			place(&b, domain.WhiteBar, domain.White, 1)
		}
	}
	return b
}

func assertParity(t *testing.T, b domain.Board) {
	t.Helper()
	got := ComputeEPC(&b)
	want := legacyComputeEPC(domain.Position{Board: b})

	if got.Bottom.AllInHome != want["bottomAllInHome"].(bool) {
		t.Fatalf("bottom AllInHome: got %v want %v (board %+v)", got.Bottom.AllInHome, want["bottomAllInHome"], b)
	}
	if got.Top.AllInHome != want["topAllInHome"].(bool) {
		t.Fatalf("top AllInHome: got %v want %v (board %+v)", got.Top.AllInHome, want["topAllInHome"], b)
	}
	checkEPC := func(side string, gotEPC *engine.EPCResult, wantAny interface{}) {
		wantEPC, _ := wantAny.(*engine.EPCResult)
		if (gotEPC == nil) != (wantEPC == nil) {
			t.Fatalf("%s EPC nil-ness: got %v want %v (board %+v)", side, gotEPC, wantEPC, b)
		}
		if gotEPC != nil && *gotEPC != *wantEPC {
			t.Fatalf("%s EPC: got %+v want %+v (board %+v)", side, *gotEPC, *wantEPC, b)
		}
	}
	checkEPC("bottom", got.Bottom.EPC, want["bottomEPC"])
	checkEPC("top", got.Top.EPC, want["topEPC"])
}

func TestComputeEPC_ParityWithLegacy(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 2000; i++ {
		assertParity(t, randomBoard(rng, false, false, false))
	}
	for i := 0; i < 2000; i++ {
		assertParity(t, randomBoard(rng, true, true, false))
	}
	for i := 0; i < 2000; i++ {
		assertParity(t, randomBoard(rng, rng.Intn(2) == 0, rng.Intn(2) == 0, true))
	}
	// Empty board: no checkers at all.
	assertParity(t, randomBoard(rng, true, true, false))
	var empty domain.Board
	for i := range empty.Points {
		empty.Points[i] = domain.Point{Color: -1}
	}
	assertParity(t, empty)
}

// TestComputeEPC_BarBlocksAllInHome pins the bar-aware semantics the server
// copy used to get wrong: a checker on the bar must clear AllInHome.
func TestComputeEPC_BarBlocksAllInHome(t *testing.T) {
	var b domain.Board
	for i := range b.Points {
		b.Points[i] = domain.Point{Color: -1}
	}
	place(&b, 1, domain.Black, 3)
	place(&b, domain.BlackBar, domain.Black, 1)
	place(&b, 24, domain.White, 2)
	place(&b, domain.WhiteBar, domain.White, 2)

	got := ComputeEPC(&b)
	if got.Bottom.AllInHome || got.Bottom.EPC != nil {
		t.Fatalf("black checker on the bar must block AllInHome: %+v", got.Bottom)
	}
	if got.Top.AllInHome || got.Top.EPC != nil {
		t.Fatalf("white checker on the bar must block AllInHome: %+v", got.Top)
	}
	if got.Bottom.CheckerCount != 4 || got.Top.CheckerCount != 4 {
		t.Fatalf("CheckerCount must include bar checkers: %+v / %+v", got.Bottom, got.Top)
	}
}
