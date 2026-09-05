package bearoffgen

import (
	"context"
	"fmt"
	"io"
)

// The one-sided sweep, a port of makebearoff.c's generate_os / BearOff /
// WriteOS / WriteIndex / CalcIndex / RollsOS.
//
// A one-sided table answers "how many rolls to bear these chequers off?" — a
// distribution over 0..31 rolls, plus a second distribution for saving the
// gammon. It is what the EPC reads, and unlike the two-sided table it says
// nothing about the opponent: one race, in isolation.
//
// The file is compressed in gnubg's own way, which is not a general-purpose
// compressor: each distribution is stored as the run of its non-zero entries,
// and an index says where that run starts and how long it is. A position where
// everything happens between roll 7 and roll 12 costs 12 bytes instead of 64.
//
// Two arithmetic details decide identity, and neither is obvious:
//
//   - the 36 outcomes are summed as integers and divided with `(sum+18)/36`,
//     which rounds to nearest, half up;
//   - that rounding does not sum to 0xFFFF, so the *mode* — the most likely
//     number of rolls — absorbs the residual. Picking a different entry to
//     absorb it gives a table that is right on average and wrong everywhere.

const (
	// osRolls is how many roll counts a distribution covers: 0..31.
	osRolls = 32
	// osCheckers is fixed at 15 for a one-sided table: gnubg's own
	// generate_os hard-codes it, and the file name carries only the points.
	osCheckers = 15
)

// headerOneSided is the 40-byte header, with gammon distributions and
// compression both on — the shape blunderDB reads.
func headerOneSided(points int) []byte {
	h := fmt.Sprintf("gnubg-OS-%02d-15-%1d-%1d-0xxxxxxxxxxxxxxxxxxx\n", points, 1, 1)
	if len(h) != 40 {
		panic(fmt.Sprintf("bearoffgen: one-sided header is %d bytes, want 40", len(h)))
	}
	return []byte(h)
}

// rollsOS is the expected number of rolls of a distribution, scaled: the sum of
// i×p(i). It is what "best move" means here — fewest expected rolls — and ties
// go to the first move examined, exactly as the C does with a strict <.
func rollsOS(dist []uint16) uint32 {
	var j uint32
	for i := 1; i < osRolls; i++ {
		j += uint32(i) * uint32(dist[i])
	}
	return j
}

// calcIndex finds the run of non-zero entries in dist[off:off+32]: where it
// starts, and how long it is.
//
// It takes the whole 64-entry array and an offset rather than a 32-entry slice,
// to reproduce one edge of the C exactly. When every entry is zero, gnubg's
// CalcIndex leaves its j at 32 and then scans i = 0..32 inclusive — reading one
// past the distribution, into the first gammon entry of the same array. That
// case does not arise in a real sweep (position 0 is handled separately, and a
// gammon distribution always has its 0xFFFF×36 seed), but a port that bounds
// the scan at 31 would answer differently if it ever did, and silently.
func calcIndex(dist []uint16, off int) (idx, nonZero int) {
	j := osRolls
	for i := osRolls - 1; i >= 0; i-- {
		if dist[off+i] != 0 {
			j = i
			break
		}
	}
	idx = 0
	for i := 0; i <= j && off+i < len(dist); i++ {
		if dist[off+i] != 0 {
			idx = i
			break
		}
	}
	return idx, j - idx + 1
}

// OneSided generates the one-sided table for `points` points (15 chequers) and
// writes it to w.
//
// Unlike the two-sided sweep, this one is strictly sequential: position i reads
// only positions j < i, so there is nothing to parallelise across — and nothing
// to get wrong about the order either.
func OneSided(ctx context.Context, w io.Writer, points int, progress func(done, total int64)) error {
	n := NumPositions(points, osCheckers)

	// dists[i] holds the 64 uint16 of position i: 32 for bearing off, 32 for
	// saving the gammon.
	dists := make([][]uint16, n)
	board := make([]int, points)
	g := newGen(points, osCheckers)

	for i := 0; i < n; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		dist := make([]uint16, 2*osRolls)
		if i == 0 {
			// Everything already off: zero rolls, with certainty.
			dist[0] = 0xFFFF
			dist[osRolls] = 0xFFFF
			dists[i] = dist
			if progress != nil {
				progress(int64(i+1), int64(n))
			}
			continue
		}

		PositionFromIndex(board, i, points, osCheckers)
		onBoard := 0
		for _, c := range board {
			onBoard += c
		}

		// aProb accumulates the 36 outcomes before the division.
		aProb := make([]int64, 2*osRolls)
		if onBoard < osCheckers {
			// Some chequers are already off, so the gammon is saved outright.
			aProb[osRolls] = 0xFFFF * 36
		}

		for d1 := 1; d1 <= 6; d1++ {
			for d2 := 1; d2 <= d1; d2++ {
				succ := g.successors(board, d1, d2)

				var bestWin, bestGammon []uint16
				usBest, usGammonBest := ^uint32(0), ^uint32(0)
				for _, j := range succ {
					sj := dists[j]
					if us := rollsOS(sj[:osRolls]); us < usBest {
						usBest = us
						bestWin = sj[:osRolls]
					}
					if us := rollsOS(sj[osRolls:]); us < usGammonBest {
						usGammonBest = us
						bestGammon = sj[osRolls:]
					}
				}
				if bestWin == nil {
					return fmt.Errorf("bearoffgen: position %d has no move for %d-%d", i, d1, d2)
				}

				weight := int64(2)
				if d1 == d2 {
					weight = 1
				}
				// The distribution shifts by one roll: reaching the successor
				// took this roll.
				for k := 0; k < osRolls-1; k++ {
					aProb[k+1] += weight * int64(bestWin[k])
					if onBoard == osCheckers {
						aProb[osRolls+k+1] += weight * int64(bestGammon[k])
					}
				}
			}
		}

		normalise(aProb[:osRolls], dist[:osRolls])
		normalise(aProb[osRolls:], dist[osRolls:])
		dists[i] = dist

		if progress != nil {
			progress(int64(i+1), int64(n))
		}
	}

	// The file: header, then the whole index, then the runs. gnubg writes the
	// index straight out and the body to a temporary file, then concatenates;
	// holding both in memory is the same bytes without the temporary file.
	if _, err := w.Write(headerOneSided(points)); err != nil {
		return fmt.Errorf("bearoffgen: write header: %w", err)
	}

	index := make([]byte, 0, n*8)
	body := make([]byte, 0, n*24)
	var npos uint32
	for i := 0; i < n; i++ {
		dist := dists[i]
		idx, nz := calcIndex(dist, 0)
		gidx, gnz := calcIndex(dist, osRolls)

		index = append(index,
			byte(npos), byte(npos>>8), byte(npos>>16), byte(npos>>24),
			byte(nz), byte(idx), byte(gnz), byte(gidx))
		npos += uint32(nz) + uint32(gnz)

		for k := idx; k < idx+nz; k++ {
			body = append(body, byte(dist[k]&0xFF), byte(dist[k]>>8))
		}
		for k := gidx; k < gidx+gnz; k++ {
			v := dist[osRolls+k]
			body = append(body, byte(v&0xFF), byte(v>>8))
		}
	}
	if _, err := w.Write(index); err != nil {
		return fmt.Errorf("bearoffgen: write index: %w", err)
	}
	if _, err := w.Write(body); err != nil {
		return fmt.Errorf("bearoffgen: write body: %w", err)
	}
	return nil
}

// normalise divides the 36-outcome sums into probabilities that sum to exactly
// 0xFFFF, the mode absorbing the rounding residual.
func normalise(sums []int64, out []uint16) {
	var total int64
	mode := 0
	for i := range sums {
		out[i] = uint16((sums[i] + 18) / 36)
		total += int64(out[i])
		if out[i] > out[mode] {
			mode = i
		}
	}
	out[mode] -= uint16(total - 0xFFFF)
}
