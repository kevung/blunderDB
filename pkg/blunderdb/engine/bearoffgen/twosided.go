package bearoffgen

import (
	"context"
	"fmt"
	"io"
)

// The two-sided sweep, a port of makebearoff.c's generate_ts / BearOff2 /
// CubeEquity.
//
// The table holds four equities per (us, them) pair, in the file's own order:
//
//	0  cubeless
//	1  we own the cube
//	2  the cube is centred
//	3  the opponent owns the cube
//
// Each is an int16 in "one plus" units: +0x7FFF is a certain win, ~0x7FFF a
// certain loss, and the file stores them offset by 0x8000 into a uint16.

const (
	equityWin  = int32(0x7FFF)
	equityLoss = int32(^0x7FFF) // one's complement, as in C: -0x8000

	// planeCount is how many equities each pair carries (cubeful).
	planeCount = 4
)

// headerTwoSided is the 40-byte ASCII header gnubg writes.
func headerTwoSided(points, checkers int) []byte {
	h := fmt.Sprintf("gnubg-TS-%02d-%02d-%1dxxxxxxxxxxxxxxxxxxxxxxx\n", points, checkers, 1)
	if len(h) != 40 {
		panic(fmt.Sprintf("bearoffgen: header is %d bytes, want 40", len(h)))
	}
	return []byte(h)
}

// cubeEquity is makebearoff.c's CubeEquity: given the no-double, double-take
// and double-pass equities, what the position is worth to the player on roll.
// Integer arithmetic throughout, including the truncating halves.
func cubeEquity(nd, dt, dp int32) int32 {
	if dt >= nd/2 && dp >= nd {
		// It is a double.
		if dt >= dp/2 {
			return dp // double, pass
		}
		return 2 * dt // double, take
	}
	return nd // no double
}

// TwoSided generates the two-sided table for `points` points and `checkers`
// chequers and writes it to w. progress, when non-nil, is called with the
// number of pairs done and the total.
//
// Memory: the whole table is held as one []int16 of nPos²×4 entries — 6.8 MB
// for TS-06-06, 1.2 GB for TS-06-11. That is the file's own body layout, so
// writing it is a copy.
func TwoSided(ctx context.Context, w io.Writer, points, checkers int, progress func(done, total int64)) error {
	n := NumPositions(points, checkers)
	total := int64(n) * int64(n)

	// body[(us*n+them)*4+k] is equity k of the pair (us, them).
	body := make([]int16, total*planeCount)

	boards := make([][]int, n)
	for i := range boards {
		boards[i] = make([]int, points)
		PositionFromIndex(boards[i], i, points, checkers)
	}

	// reach[i][r] holds the successor indices of position i for roll r, where
	// r indexes the 21 distinct rolls. Precomputed once: the sweep asks for
	// them nPos² times, and they do not depend on the opponent at all.
	reach := make([][][]int, n)
	seen := make(map[int]struct{}, 64)
	for i := 0; i < n; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		reach[i] = make([][]int, 0, 21)
		for d1 := 1; d1 <= 6; d1++ {
			for d2 := 1; d2 <= d1; d2++ {
				reach[i] = append(reach[i], successors(boards[i], d1, d2, points, checkers, seen))
			}
		}
	}

	// The sweep, along diagonals of constant us+them: a pair (us, them) reads
	// row `them` at columns j < us, so every value it needs is already there.
	// makebearoff.c walks the same order in two loops, above and below the
	// diagonal.
	var done int64
	sweep := func(us, them int) {
		off := (us*n + them) * planeCount
		if us == 0 {
			// We have borne off: a win, whatever the cube says.
			for k := 0; k < planeCount; k++ {
				body[off+k] = int16(equityWin)
			}
			return
		}
		if them == 0 {
			for k := 0; k < planeCount; k++ {
				body[off+k] = int16(equityLoss)
			}
			return
		}

		var totals [planeCount]int32
		for r, succ := range reach[us] {
			var best [planeCount]int32
			for k := range best {
				best[k] = -0xFFFF
			}
			for _, j := range succ {
				// The successor is read from the OPPONENT's point of view:
				// after our move it is their turn, so the pair is (them, j).
				var sij [planeCount]int32
				switch {
				case them == 0:
					for k := range sij {
						sij[k] = equityWin
					}
				case j == 0:
					for k := range sij {
						sij[k] = equityLoss
					}
				default:
					base := (them*n + j) * planeCount
					for k := 0; k < planeCount; k++ {
						sij[k] = int32(body[base+k])
					}
				}

				// Cubeless: their equity negated.
				if sij[0] < -best[0] {
					best[0] = ^sij[0]
				}
				// We own the cube: from their side, they do not own it.
				if sij[3] < -best[1] {
					best[1] = ^sij[3]
				}
				// Centred for us is centred for them.
				if k := cubeEquity(sij[2], sij[3], equityWin); ^k > best[2] {
					best[2] = ^k
				}
				// They own the cube: from their side, they do own it.
				if k := cubeEquity(sij[1], sij[3], equityWin); ^k > best[3] {
					best[3] = ^k
				}
			}
			weight := int32(2)
			if isDouble(r) {
				weight = 1
			}
			for k := 0; k < planeCount; k++ {
				totals[k] += weight * best[k]
			}
		}
		for k := 0; k < planeCount; k++ {
			body[off+k] = int16(totals[k] / 36)
		}
	}

	for i := 0; i < n; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		for j := 0; j <= i; j++ {
			sweep(i-j, j)
			done++
		}
		if progress != nil {
			progress(done, total)
		}
	}
	for i := 0; i < n; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		for j := i + 1; j < n; j++ {
			sweep(i+n-j, j)
			done++
		}
		if progress != nil {
			progress(done, total)
		}
	}

	if _, err := w.Write(headerTwoSided(points, checkers)); err != nil {
		return fmt.Errorf("bearoffgen: write header: %w", err)
	}
	// The body, in the order makebearoff.c writes it: pair by pair, us-major.
	buf := make([]byte, 0, 1<<16)
	for idx := 0; idx < len(body); idx++ {
		u := uint16(int32(body[idx]) + 0x8000)
		buf = append(buf, byte(u&0xFF), byte(u>>8))
		if len(buf) >= 1<<16-2 {
			if _, err := w.Write(buf); err != nil {
				return fmt.Errorf("bearoffgen: write body: %w", err)
			}
			buf = buf[:0]
		}
	}
	if len(buf) > 0 {
		if _, err := w.Write(buf); err != nil {
			return fmt.Errorf("bearoffgen: write body: %w", err)
		}
	}
	return nil
}

// isDouble reports whether roll index r (0..20, the order d1=1..6, d2=1..d1)
// is a double. A double is played once in the 36-roll sum, a non-double twice.
func isDouble(r int) bool {
	i := 0
	for d1 := 1; d1 <= 6; d1++ {
		for d2 := 1; d2 <= d1; d2++ {
			if i == r {
				return d1 == d2
			}
			i++
		}
	}
	return false
}
