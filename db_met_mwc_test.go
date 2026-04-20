package main

import (
	"math"
	"testing"
)

// TestConvertEMGLossToMWCLoss_MoneyGame verifies that a money-game position
// (matchLength ≤ 0) returns NaN.
func TestConvertEMGLossToMWCLoss_MoneyGame(t *testing.T) {
	result := ConvertEMGLossToMWCLoss(100, 0, 0, 0, 1, 0)
	if !math.IsNaN(result) {
		t.Errorf("expected NaN for money game, got %f", result)
	}
}

// TestConvertEMGLossToMWCLoss_NegativeMatchLength also returns NaN.
func TestConvertEMGLossToMWCLoss_NegativeMatchLength(t *testing.T) {
	result := ConvertEMGLossToMWCLoss(100, 0, 0, 0, 1, -1)
	if !math.IsNaN(result) {
		t.Errorf("expected NaN for negative matchLength, got %f", result)
	}
}

// TestConvertEMGLossToMWCLoss_DMP tests DMP (1-away / 1-away, cube=1).
// In DMP: mwcWin=1.0, mwcLose=0.0, denom=1.0
// ΔMWC = emgMillipoints/1000 × 1.0/2.0 = emgMillipoints/2000
func TestConvertEMGLossToMWCLoss_DMP(t *testing.T) {
	emgMP := 200 // 0.2 EMG
	result := ConvertEMGLossToMWCLoss(emgMP, 0, 0, 0, 1, 1)
	if math.IsNaN(result) {
		t.Fatal("unexpected NaN for DMP")
	}
	want := float64(emgMP) / 2000.0
	if math.Abs(result-want) > 1e-5 {
		t.Errorf("DMP: got %.6f, want %.6f", result, want)
	}
}

// TestConvertEMGLossToMWCLoss_RoundTrip verifies the round-trip property:
// converting a ΔMWC to ΔEMG with the forward formula and back with
// ConvertEMGLossToMWCLoss recovers the original ΔMWC to within 1e-5
// (float32 precision).
//
// Test cases span DMP (1pt), Crawford, 3-away/5-away, 7pt, 11pt, 21pt matches
// with cube values 1 and 2.
func TestConvertEMGLossToMWCLoss_RoundTrip(t *testing.T) {
	type testCase struct {
		name        string
		score0      int
		score1      int
		matchLength int
		fMove       int
		cubeValue   int
	}
	cases := []testCase{
		// DMP: both players need 1 more point
		{"DMP", 0, 0, 1, 0, 1},
		// Crawford: 1-away (leader) vs 3-away (trailer), cube dead (cube=1)
		{"Crawford 1v3", 0, 2, 3, 1, 1},
		// 3-away 5-away, cube=1
		{"3away5away cube1", 2, 4, 7, 0, 1},
		// 3-away 5-away, cube=2
		{"3away5away cube2", 2, 4, 7, 0, 2},
		// 7-point match, even score, cube=1
		{"7pt even", 3, 3, 7, 0, 1},
		// 11-point match, 4away 7away, cube=2
		{"11pt 4a7a cube2", 7, 4, 11, 0, 2},
		// 21-point match, cube=4
		{"21pt cube4", 10, 8, 21, 0, 4},
	}

	// An arbitrary ΔMWC loss to test with (1.5% = 0.015).
	const deltaMWCIn = 0.015

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mwcWin := float32(gnuBGGetME(tc.score0, tc.score1, tc.matchLength, tc.fMove, tc.cubeValue, tc.fMove, false))
			mwcLose := float32(gnuBGGetME(tc.score0, tc.score1, tc.matchLength, tc.fMove, tc.cubeValue, 1-tc.fMove, false))
			denom := mwcWin - mwcLose
			if denom < 1e-7 && denom > -1e-7 {
				t.Skipf("degenerate denom (%.6f) for %s — skipping", denom, tc.name)
			}

			// Forward: ΔMWC → ΔEMG millipoints
			// ΔEMG = ΔMWC × 2 / (mwcWin − mwcLose)
			deltaEMG := deltaMWCIn * 2.0 / float64(denom)
			deltaEMGMP := int(deltaEMG * 1000)

			// Inverse
			got := ConvertEMGLossToMWCLoss(deltaEMGMP, tc.score0, tc.score1, tc.fMove, tc.cubeValue, tc.matchLength)
			if math.IsNaN(got) {
				t.Fatalf("unexpected NaN for %s", tc.name)
			}

			// Reconstruct the expected value from the same integer truncation.
			wantExact := float64(deltaEMGMP) / 1000.0 * float64(denom) / 2.0
			if math.Abs(got-wantExact) > 1e-5 {
				t.Errorf("round-trip failed: got %.8f, want %.8f (diff %.2e)", got, wantExact, math.Abs(got-wantExact))
			}

			// Sign must be positive (a loss).
			if got < 0 {
				t.Errorf("MWC loss should be non-negative, got %.8f", got)
			}
		})
	}
}

// TestConvertEMGLossToMWCLoss_ZeroError verifies that zero error yields zero MWC loss.
func TestConvertEMGLossToMWCLoss_ZeroError(t *testing.T) {
	result := ConvertEMGLossToMWCLoss(0, 3, 3, 0, 1, 7)
	if math.IsNaN(result) {
		t.Fatal("unexpected NaN")
	}
	if result != 0.0 {
		t.Errorf("expected 0.0, got %f", result)
	}
}

// TestConvertEMGLossToMWCLoss_Crawford tests a Crawford score
// (score 4-away / Crawford 1-away, cube=1). The conversion must return a
// plausible non-NaN value with the correct sign.
func TestConvertEMGLossToMWCLoss_Crawford(t *testing.T) {
	// 5-point match: leader needs 1, trailer needs 4 (trailer is on roll, fMove=1)
	result := ConvertEMGLossToMWCLoss(100, 0, 3, 1, 1, 5)
	if math.IsNaN(result) {
		t.Fatal("unexpected NaN for Crawford position")
	}
	// A positive error should yield a positive MWC loss.
	if result <= 0 {
		t.Errorf("expected positive MWC loss, got %.6f", result)
	}
}
