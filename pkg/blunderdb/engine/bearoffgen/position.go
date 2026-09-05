package bearoffgen

// The combinatorial indexing of a bearoff position, ported from gnubg's
// positionid.c. A position of `checkers` chequers over `points` points is a
// multiset, ranked here in the same order gnubg ranks it — the order the file
// itself is laid out in, so the index IS the row number.

// combination is C(n, k), memoised over the small range the generator uses.
// gnubg computes it the same way; the numbers stay far below int64 overflow
// for every domain a person can generate (C(26,11) ≈ 7.7e6).
func combination(n, k int) int {
	if k < 0 || k > n {
		return 0
	}
	if k > n-k {
		k = n - k
	}
	result := 1
	for i := 0; i < k; i++ {
		result = result * (n - i) / (i + 1)
	}
	return result
}

// NumPositions is how many distinct arrangements of `checkers` chequers over
// `points` points exist — the side of the square table.
func NumPositions(points, checkers int) int {
	return combination(points+checkers, points)
}

// positionF ranks a bit pattern: the combinatorial number system, as gnubg's
// PositionF does.
func positionF(bits uint32, n, k int) int {
	index := 0
	for n > 0 {
		n--
		if bits&(1<<uint(n)) != 0 {
			if k > 0 {
				index += combination(n, k)
			}
			k--
			if k == 0 {
				break
			}
		}
	}
	return index
}

// positionInv is positionF's inverse (gnubg's PositionInv), written as a loop
// rather than the C recursion: the depth is checkers+points, small, but a loop
// keeps the hot path allocation-free.
func positionInv(id, n, r int) uint32 {
	var bits uint32
	for {
		if r == 0 {
			return bits
		}
		if n == r {
			return bits | (1<<uint(n) - 1)
		}
		nc := combination(n-1, r)
		if id >= nc {
			bits |= 1 << uint(n-1)
			id -= nc
			n--
			r--
			continue
		}
		n--
	}
}

// PositionIndex ranks one arrangement. board[i] is the number of chequers on
// point i+1, point 1 being the one that bears off next.
func PositionIndex(board []int, points, checkers int) int {
	j := points - 1
	for i := 0; i < points; i++ {
		j += board[i]
	}
	bits := uint32(1) << uint(j)
	for i := 0; i < points-1; i++ {
		j -= board[i] + 1
		bits |= 1 << uint(j)
	}
	return positionF(bits, checkers+points, points)
}

// PositionFromIndex fills board with the arrangement ranked id. board must
// have at least `points` entries; they are all overwritten.
func PositionFromIndex(board []int, id, points, checkers int) {
	bits := positionInv(id, checkers+points, points)
	for i := 0; i < points; i++ {
		board[i] = 0
	}
	j := points - 1
	for i := 0; i < checkers+points; i++ {
		if bits&(1<<uint(i)) != 0 {
			if j == 0 {
				break
			}
			j--
		} else {
			board[j]++
		}
	}
}
