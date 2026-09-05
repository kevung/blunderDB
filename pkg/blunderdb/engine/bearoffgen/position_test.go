package bearoffgen

import "testing"

// The ranking must be a bijection over the domain, and must agree with the one
// the reader already uses (engine/epc.go's positionBearoff) — the file's rows
// are addressed by it.
func TestPositionIndex_IsABijectionOverTheDomain(t *testing.T) {
	const points, checkers = 6, 6
	n := NumPositions(points, checkers)
	if n != combination(points+checkers, points) {
		t.Fatalf("NumPositions = %d", n)
	}

	seen := make([]bool, n)
	board := make([]int, points)
	for id := 0; id < n; id++ {
		PositionFromIndex(board, id, points, checkers)

		total := 0
		for _, c := range board {
			if c < 0 {
				t.Fatalf("id %d gives a negative count: %v", id, board)
			}
			total += c
		}
		if total > checkers {
			t.Fatalf("id %d holds %d chequers, more than %d: %v", id, total, checkers, board)
		}
		if got := PositionIndex(board, points, checkers); got != id {
			t.Fatalf("round trip: id %d -> %v -> %d", id, board, got)
		}
		if seen[id] {
			t.Fatalf("id %d produced twice", id)
		}
		seen[id] = true
	}
}

// Position 0 is the borne-off position: it is what the sweep treats as
// terminal, and getting it wrong would shift the whole file.
func TestPositionIndex_ZeroIsEmpty(t *testing.T) {
	board := make([]int, 6)
	PositionFromIndex(board, 0, 6, 6)
	for i, c := range board {
		if c != 0 {
			t.Errorf("index 0 must be the empty board, got %d on point %d", c, i+1)
		}
	}
}

func TestNumPositions_MatchesTheKnownSizes(t *testing.T) {
	// The side of the square table for the domains ADR-0027 names.
	for _, tc := range []struct{ points, checkers, want int }{
		{6, 6, 924},
		{6, 11, 12376},
		{6, 15, 54264},
	} {
		if got := NumPositions(tc.points, tc.checkers); got != tc.want {
			t.Errorf("NumPositions(%d,%d) = %d, want %d", tc.points, tc.checkers, got, tc.want)
		}
	}
	// And the file size the ADR gives for TS-06-06: 40 + n²×8.
	n := NumPositions(6, 6)
	if size := 40 + n*n*8; size != 6830248 {
		t.Errorf("TS-06-06 size = %d, want 6830248 (the size of gnubg_ts0.bd)", size)
	}
}
