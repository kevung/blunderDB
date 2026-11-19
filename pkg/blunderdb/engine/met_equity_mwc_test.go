package engine

import (
	"math"
	"testing"
)

// TestEquityToMWC locks the anchors of the normalised scale and its agreement
// with ConvertEMGLossToMWCLoss: the MWC gap between two equities is the loss
// their difference converts to.
func TestEquityToMWC(t *testing.T) {
	// Double match point: MWC = (1+eq)/2.
	if got := EquityToMWC(0.2, 6, 6, 1, 7); math.Abs(got-0.6) > 1e-6 {
		t.Errorf("DMP eq 0.2: got %v, want 0.6", got)
	}
	for _, c := range []struct{ s0, s1, cube, length int }{{0, 0, 1, 7}, {2, 4, 2, 7}, {3, 1, 1, 5}, {0, 10, 4, 11}} {
		win := GnuBGGetME(c.s0, c.s1, c.length, 0, c.cube, 0, false)
		lose := GnuBGGetME(c.s0, c.s1, c.length, 0, c.cube, 1, false)
		if got := EquityToMWC(1, c.s0, c.s1, c.cube, c.length); math.Abs(got-win) > 1e-6 {
			t.Errorf("%+v eq +1: got %v, want %v", c, got, win)
		}
		if got := EquityToMWC(-1, c.s0, c.s1, c.cube, c.length); math.Abs(got-lose) > 1e-6 {
			t.Errorf("%+v eq -1: got %v, want %v", c, got, lose)
		}
		gap := EquityToMWC(0.31, c.s0, c.s1, c.cube, c.length) - EquityToMWC(0.187, c.s0, c.s1, c.cube, c.length)
		if loss := ConvertEMGLossToMWCLoss(123, c.s0, c.s1, 0, c.cube, c.length); math.Abs(gap-loss) > 1e-12 {
			t.Errorf("%+v gap %v, loss %v", c, gap, loss)
		}
	}
	for _, length := range []int{0, -1, 99999} {
		if got := EquityToMWC(0.5, 0, 0, 1, length); !math.IsNaN(got) {
			t.Errorf("money (length %d): got %v, want NaN", length, got)
		}
	}
}
