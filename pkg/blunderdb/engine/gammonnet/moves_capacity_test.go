// SPDX-License-Identifier: MIT

package gammonnet

import "testing"

// trajectoryBound is the largest number of distinct trajectory multisets a
// side can choose for k sub-moves of one double, over every way of spreading
// fifteen checkers. A point holding n checkers contributes the partitions of
// its share of the k steps into at most n parts, so the count for one spread
// is the x^k coefficient of the product of those generating functions; the
// spread is a partition of 15 (where the points are does not matter).
func trajectoryBound(k int) int {
	// parts[n][j]: partitions of j into at most n parts, j <= k.
	parts := make([][]int, NumCheckers+1)
	for n := range parts {
		parts[n] = make([]int, k+1)
		for j := 0; j <= k; j++ {
			parts[n][j] = partitionsAtMost(j, n, j)
		}
	}
	best := 0
	var walk func(left, maxPart int, poly []int)
	walk = func(left, maxPart int, poly []int) {
		if left == 0 {
			if poly[k] > best {
				best = poly[k]
			}
			return
		}
		for n := min(left, maxPart); n >= 1; n-- {
			next := make([]int, k+1)
			for a := 0; a <= k; a++ {
				for b := 0; a+b <= k; b++ {
					next[a+b] += poly[a] * parts[n][b]
				}
			}
			walk(left-n, n, next)
		}
	}
	unit := make([]int, k+1)
	unit[0] = 1
	walk(NumCheckers, NumCheckers, unit)
	return best
}

// partitionsAtMost counts the partitions of j into at most n parts, none
// larger than maxPart.
func partitionsAtMost(j, n, maxPart int) int {
	if j == 0 {
		return 1
	}
	if n == 0 || maxPart == 0 {
		return 0
	}
	total := 0
	for p := min(j, maxPart); p >= 1; p-- {
		total += partitionsAtMost(j-p, n-1, p)
	}
	return total
}

// TestGenerationCapacityCoversEveryDouble: the capacities are not tuned to a
// corpus, they cover the bound the generator's comment proves.
func TestGenerationCapacityCoversEveryDouble(t *testing.T) {
	want := [MaxMovesPerPlay + 1]int{1, 15, 120, 680, 3060}
	for k := 1; k <= MaxMovesPerPlay; k++ {
		if got := trajectoryBound(k); got != want[k] {
			t.Fatalf("trajectory bound for %d sub-moves: %d, want %d", k, got, want[k])
		}
	}
	if maxLevel < want[MaxMovesPerPlay] || MaxPlays < want[MaxMovesPerPlay] {
		t.Fatalf("maxLevel %d / MaxPlays %d below the proven bound %d", maxLevel, MaxPlays, want[MaxMovesPerPlay])
	}
	if dedupSlots < 4*maxLevel/3 {
		t.Fatalf("dedup table of %d slots too loaded for %d entries", dedupSlots, maxLevel)
	}
}

// TestLegalPlaysOnTheWidestDouble: fifteen White blots against Black's blots
// with eleven checkers off, rolling 1-1. Hitting tells apart plays that would
// otherwise coincide, which pushes the count of distinct plays past 2048 — the
// position a rollout reached, and that generation used to refuse.
func TestLegalPlaysOnTheWidestDouble(t *testing.T) {
	p := Position{
		Points: [NumPoints]int8{-1, -1, 1, 1, 0, 1, 0, 1, 1, 0, 1, 0, 1, 1, -1, 1, 1, 1, -1, 1, 1, 0, 1, 1},
		Off:    [2]uint8{0, 11},
		Turn:   White,
	}
	if !p.Valid() {
		t.Fatal("fixture is not a valid position")
	}
	var g Generator
	plays := make([]Play, MaxPlays)
	n := g.LegalPlays(&p, 1, 1, plays)
	want := dfsOracle(&p, 1, 1)
	if n <= 2048 {
		t.Fatalf("got %d plays, want upstream's %d (more than 2048)", n, len(want))
	}
	if !samePlays(plays, n, want) {
		t.Fatalf("generator differs from upstream's plays or order (%d vs %d plays)", n, len(want))
	}
}
