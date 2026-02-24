package main

import (
	"math"
	"testing"
)

func TestComputeEPC(t *testing.T) {
	tests := []struct {
		name        string
		board       [6]int
		expectedEPC float64
		tolerance   float64
	}{
		{
			// Full jan closed with extra checkers on 4, 5, 6 points
			// points 1-6: 2, 2, 2, 3+2=5, 3+2=5, 3+2=5 => total 21... wait
			// "6 points and 3 extra checkers on point 4, 5, 6"
			// Actually: full jan = 2 on each of 6 points = 12 checkers
			// + 3 extra on points 4,5,6 = 1 on each = 15 total
			// So: [2, 2, 2, 3, 3, 3]
			name:        "full jan closed with extra checkers on 4,5,6",
			board:       [6]int{2, 2, 2, 3, 3, 3},
			expectedEPC: 66.47,
			tolerance:   0.1,
		},
		{
			// Full jan closed and 3 checkers off = 12 checkers on points 1-6
			// 2 on each point
			name:        "full jan closed 3 checkers off",
			board:       [6]int{2, 2, 2, 2, 2, 2},
			expectedEPC: 51.94,
			tolerance:   0.1,
		},
		{
			// 6 checkers on point 1
			name:        "6 checkers on point 1",
			board:       [6]int{6, 0, 0, 0, 0, 0},
			expectedEPC: 22.00,
			tolerance:   0.1,
		},
		{
			// Checkers on 4, 5, 6 (12 checkers off)
			// 1 checker on each of points 4, 5, 6 = 3 checkers
			name:        "checkers on 4, 5, 6",
			board:       [6]int{0, 0, 0, 1, 1, 1},
			expectedEPC: 20.31,
			tolerance:   0.1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := ComputeEPC(test.board)
			if err != nil {
				t.Fatalf("ComputeEPC failed: %v", err)
			}
			if math.Abs(result.EPC-test.expectedEPC) > test.tolerance {
				t.Errorf("EPC = %.2f, expected %.2f (tolerance %.2f)", result.EPC, test.expectedEPC, test.tolerance)
			}
			t.Logf("Board %v: EPC=%.2f, MeanRolls=%.3f, StdDev=%.3f, PipCount=%d, Wastage=%.2f",
				test.board, result.EPC, result.MeanRolls, result.StdDev, result.PipCount, result.Wastage)
		})
	}
}

func TestPositionBearoff(t *testing.T) {
	// Position with 0 checkers should be index 0
	idx := positionBearoff([6]int{0, 0, 0, 0, 0, 0}, 6, 15)
	if idx != 0 {
		t.Errorf("Expected index 0 for empty board, got %d", idx)
	}

	// Position with 1 checker on point 1 should be index 1
	idx = positionBearoff([6]int{1, 0, 0, 0, 0, 0}, 6, 15)
	if idx != 1 {
		t.Errorf("Expected index 1 for 1 checker on point 1, got %d", idx)
	}
}

func TestCombination(t *testing.T) {
	// C(21,6) = 54264
	c := combination(21, 6)
	if c != 54264 {
		t.Errorf("C(21,6) = %d, expected 54264", c)
	}
}

func TestZeroCheckers(t *testing.T) {
	result, err := ComputeEPC([6]int{0, 0, 0, 0, 0, 0})
	if err != nil {
		t.Fatalf("ComputeEPC failed: %v", err)
	}
	if result.EPC != 0 {
		t.Errorf("EPC should be 0 for no checkers, got %.2f", result.EPC)
	}
}
