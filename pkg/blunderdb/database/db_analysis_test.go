package database

import "testing"

// TestComputeIsCloseCube verifies the gnuBG isCloseCubedecision mapping.
// Reference: gnubg/eval.c:5088-5100
//
//	rDouble = min(DoubleTakeEquity, 1.0)
//	isClose = (OptimalEquity - rDouble) < 0.16
func TestComputeIsCloseCube(t *testing.T) {
	cases := []struct {
		name             string
		noDouble         float64
		doubleTake       float64
		doublePass       float64
		bestAction       string
		playedCubeAction string
		want             int64
	}{
		{
			name:       "trivial NoDouble — far from doubling point",
			noDouble:   0.20, doubleTake: 0.50, doublePass: 1.00,
			bestAction: "No Double",
			// rOptimal=0.20, rDouble=min(0.50,1.0)=0.50, diff=-0.30 < 0.16 → close!
			// Wait: 0.20 - 0.50 = -0.30 which is < 0.16, so it's close.
			// Actually: rOptimal=noDouble=0.20, rDouble=0.50 → diff = -0.30 < 0.16 → close
			// Hmm, that seems wrong. Let me re-read gnuBG:
			// rDouble = MIN(arDouble[OUTPUT_TAKE], 1.0) = min(0.50, 1.0) = 0.50
			// if (arDouble[OUTPUT_OPTIMAL] - rDouble < rThr) → 0.20 - 0.50 = -0.30 < 0.16 → returns 1 (close)
			// So even a "trivial no double" where NoDouble=0.20, DT=0.50 is flagged as close!
			// The formula flags close when optimal is BELOW or close to DT.
			// Non-close would be: optimal >> DT (e.g. noDouble=1.0, DT=0.50: 1.0-0.50=0.50 >= 0.16)
			want: 1,
		},
		{
			name:       "clear NoDouble — noDouble well above DoubleTake",
			noDouble:   0.80, doubleTake: 0.40, doublePass: 1.00,
			bestAction: "No Double",
			// rOptimal=0.80, rDouble=0.40, diff=0.40 >= 0.16 → NOT close
			want: 0,
		},
		{
			name:       "near doubling point — within threshold",
			noDouble:   0.52, doubleTake: 0.50, doublePass: 1.00,
			bestAction: "No Double",
			// rOptimal=0.52, rDouble=0.50, diff=0.02 < 0.16 → close
			want: 1,
		},
		{
			name:       "DoubleTake best — diff always 0",
			noDouble:   0.40, doubleTake: 0.60, doublePass: 1.00,
			bestAction: "Double, Take",
			// rOptimal=0.60 (DoubleTake), rDouble=min(0.60,1.0)=0.60, diff=0 < 0.16 → close
			want: 1,
		},
		{
			name:       "DoublePass best",
			noDouble:   0.40, doubleTake: 0.70, doublePass: 0.80,
			bestAction: "Double, Pass",
			// rOptimal=0.80, rDouble=min(0.70,1.0)=0.70, diff=0.10 < 0.16 → close
			want: 1,
		},
		{
			name:             "Take action — always close",
			noDouble:         0.0, doubleTake: 0.0, doublePass: 0.0,
			bestAction:       "",
			playedCubeAction: "Take",
			want:             1,
		},
		{
			name:             "Pass action — always close",
			noDouble:         0.0, doubleTake: 0.0, doublePass: 0.0,
			bestAction:       "",
			playedCubeAction: "Pass",
			want:             1,
		},
		{
			name:       "nil analysis — no cube data",
			bestAction: "",
			want:       0,
		},
		{
			name:       "DoubleTake equity capped at 1.0",
			noDouble:   1.10, doubleTake: 1.20, doublePass: 2.00,
			bestAction: "No Double",
			// rOptimal=1.10, rDouble=min(1.20,1.0)=1.0, diff=0.10 < 0.16 → close
			want: 1,
		},
		{
			name:       "NoDouble well above capped take — not close",
			noDouble:   1.50, doubleTake: 1.20, doublePass: 2.00,
			bestAction: "No Double",
			// rOptimal=1.50, rDouble=min(1.20,1.0)=1.0, diff=0.50 >= 0.16 → NOT close
			want: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var dca *DoublingCubeAnalysis
			if tc.bestAction != "" || tc.noDouble != 0 || tc.doubleTake != 0 || tc.doublePass != 0 {
				dca = &DoublingCubeAnalysis{
					BestCubeAction:          tc.bestAction,
					CubefulNoDoubleEquity:   tc.noDouble,
					CubefulDoubleTakeEquity: tc.doubleTake,
					CubefulDoublePassEquity: tc.doublePass,
				}
			}
			got := computeIsCloseCube(dca, tc.playedCubeAction)
			if got != tc.want {
				t.Errorf("computeIsCloseCube: got %d, want %d", got, tc.want)
			}
		})
	}
}
