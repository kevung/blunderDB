package bearoffgen

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"sync"
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

// TwoSidedState is a run's whole state: the table so far, and the diagonal it
// has reached. It is what a pause writes down and a resume picks up — the
// sweep needs nothing else, since every diagonal reads only earlier ones.
type TwoSidedState struct {
	Points, Checkers int
	// Diagonal is the next diagonal to compute, in 0 … 2n-2. A finished run
	// has Diagonal == 2n-1.
	Diagonal int
	// Body[(us*n+them)*4+k] is equity k of the pair (us, them).
	Body []int16
}

// NewTwoSidedState allocates the state of a run that has not started.
//
// Memory: the whole table is held as one []int16 of nPos²×4 entries — 6.8 MB
// for TS-06-06, 1.2 GB for TS-06-11. That is the file's own body layout, so
// writing it is a copy.
func NewTwoSidedState(points, checkers int) *TwoSidedState {
	n := NumPositions(points, checkers)
	return &TwoSidedState{
		Points:   points,
		Checkers: checkers,
		Body:     make([]int16, int64(n)*int64(n)*planeCount),
	}
}

// Done reports whether the sweep has covered every diagonal.
func (st *TwoSidedState) Done() bool {
	n := NumPositions(st.Points, st.Checkers)
	return st.Diagonal >= 2*n-1
}

// TwoSided generates the two-sided table for `points` points and `checkers`
// chequers and writes it to w, on every core the machine has. progress, when
// non-nil, is called with the number of pairs done and the total.
func TwoSided(ctx context.Context, w io.Writer, points, checkers int, progress func(done, total int64)) error {
	st := NewTwoSidedState(points, checkers)
	if err := ComputeTwoSided(ctx, st, runtime.NumCPU(), progress); err != nil {
		return err
	}
	return WriteTwoSided(w, st)
}

// ComputeTwoSided runs the sweep from wherever `st` left off, across `workers`
// goroutines (0 or 1 = serial, anything else capped at NumCPU).
//
// Parallelism here decides WHO computes an entry, never what it is worth: a
// pair (us, them) reads only pairs whose us+them is strictly smaller — its
// successors move chequers forward, so their index is always lower — which is
// to say only diagonals already finished. Entries on one diagonal cannot see
// each other, so splitting a diagonal across cores is byte-for-byte the same
// table. TestTwoSided_6x6_IdenticalToGnubg is what holds that claim.
func ComputeTwoSided(ctx context.Context, st *TwoSidedState, workers int, progress func(done, total int64)) error {
	points, checkers := st.Points, st.Checkers
	n := NumPositions(points, checkers)
	total := int64(n) * int64(n)
	body := st.Body
	if int64(len(body)) != total*planeCount {
		return fmt.Errorf("bearoffgen: state holds %d entries, want %d", len(body), total*planeCount)
	}
	if workers <= 0 {
		workers = 1
	}
	if max := runtime.NumCPU(); workers > max {
		workers = max
	}

	boards := make([][]int, n)
	for i := range boards {
		boards[i] = make([]int, points)
		PositionFromIndex(boards[i], i, points, checkers)
	}

	// reach[i][r] holds the successor indices of position i for roll r, where
	// r indexes the 21 distinct rolls. Precomputed once: the sweep asks for
	// them nPos² times, and they do not depend on the opponent at all.
	reach := make([][][]int, n)
	if err := parallelFor(ctx, workers, n, func(lo, hi int) {
		g := newGen(points, checkers) // one generator per worker: it owns buffers
		for i := lo; i < hi; i++ {
			reach[i] = make([][]int, 0, 21)
			for d1 := 1; d1 <= 6; d1++ {
				for d2 := 1; d2 <= d1; d2++ {
					// g owns its result slice, so copy what we keep.
					reach[i] = append(reach[i], append([]int(nil), g.successors(boards[i], d1, d2)...))
				}
			}
		}
	}); err != nil {
		return err
	}

	// The sweep, along diagonals of constant us+them: a pair (us, them) reads
	// row `them` at columns j < us, so every value it needs is already there.
	// makebearoff.c walks the same order in two loops, above and below the
	// diagonal.
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

	// Diagonal d holds the pairs with us+them == d, and reads only diagonals
	// below it. So a diagonal is a barrier and its entries are independent:
	// resume means starting at st.Diagonal, and parallelism means splitting
	// one diagonal, never spanning two.
	for d := st.Diagonal; d < 2*n-1; d++ {
		lo, hi := 0, d // them runs over [lo, hi]
		if d >= n {
			lo = d - n + 1
		}
		if hi > n-1 {
			hi = n - 1
		}
		if err := parallelFor(ctx, workers, hi-lo+1, func(a, b int) {
			for k := a; k < b; k++ {
				them := lo + k
				sweep(d-them, them)
			}
		}); err != nil {
			// Cancelled mid-diagonal: the diagonal is incomplete, so the
			// resume point stays where it was. Redoing one diagonal is the
			// price of not writing a half-computed one down as finished.
			return err
		}
		st.Diagonal = d + 1
		if progress != nil {
			progress(pairsThroughDiagonal(n, d+1), total)
		}
	}
	return nil
}

// pairsThroughDiagonal counts the pairs on diagonals 0 … d-1, which is what
// "done" means to a caller watching a resumable run.
func pairsThroughDiagonal(n, d int) int64 {
	var done int64
	for i := 0; i < d; i++ {
		lo, hi := 0, i
		if i >= n {
			lo = i - n + 1
		}
		if hi > n-1 {
			hi = n - 1
		}
		done += int64(hi - lo + 1)
	}
	return done
}

// parallelFor splits [0, count) into `workers` contiguous chunks and runs fn
// on each. Serial when workers is 1, which is what the small domains and the
// tests want: no goroutine, no barrier, same result.
func parallelFor(ctx context.Context, workers, count int, fn func(lo, hi int)) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if count <= 0 {
		return nil
	}
	if workers <= 1 || count < workers {
		fn(0, count)
		return ctx.Err()
	}
	var wg sync.WaitGroup
	chunk := (count + workers - 1) / workers
	for lo := 0; lo < count; lo += chunk {
		hi := lo + chunk
		if hi > count {
			hi = count
		}
		wg.Add(1)
		go func(lo, hi int) {
			defer wg.Done()
			fn(lo, hi)
		}(lo, hi)
	}
	wg.Wait()
	return ctx.Err()
}

// WriteTwoSided writes a finished state out in the order makebearoff.c writes
// it: the 40-byte header, then the body pair by pair, us-major.
func WriteTwoSided(w io.Writer, st *TwoSidedState) error {
	if _, err := w.Write(headerTwoSided(st.Points, st.Checkers)); err != nil {
		return fmt.Errorf("bearoffgen: write header: %w", err)
	}
	buf := make([]byte, 0, 1<<16)
	for idx := 0; idx < len(st.Body); idx++ {
		u := uint16(int32(st.Body[idx]) + 0x8000)
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
