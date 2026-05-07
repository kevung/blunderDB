package main

import "math/rand"

// Zobrist hash tables for Position identity.
// Populated from a fixed seed in init(). Never change the seed or the
// population order — doing so invalidates any stored hashes.
var (
	zobristPoint        [26][16][2]uint64 // [point_index 0..25][checker_count 1..15][color 0=Black 1=White]
	zobristPlayerOnRoll [2]uint64
	zobristDice         [7][7]uint64 // [die1][die2] indices 1..6; [0][0] = no dice
	zobristCubeValue    [11]uint64   // cube values 1,2,4,8,16,32,64,128,256,512,1024 → indices 0..10
	zobristCubeOwner    [3]uint64    // 0=Black, 1=White, 2=None (maps -1 → 2)
	zobristScore1       [64]uint64
	zobristScore2       [64]uint64
	zobristMatchLength  [64]uint64 // reserved for future match-length field
	zobristHasJacoby    uint64
	zobristHasBeaver    uint64
	zobristDecisionType [2]uint64     // [0]=CheckerAction, [1]=CubeAction
	zobristBearoff      [2][16]uint64 // [color 0=Black 1=White][checker_count 0..15]
)

func init() {
	//nolint:gosec // fixed seed is intentional — hash stability matters
	r := rand.New(rand.NewSource(0xB10DE4DB))
	next := r.Uint64

	for i := range zobristPoint {
		for j := range zobristPoint[i] {
			for k := range zobristPoint[i][j] {
				zobristPoint[i][j][k] = next()
			}
		}
	}
	for i := range zobristPlayerOnRoll {
		zobristPlayerOnRoll[i] = next()
	}
	for i := range zobristDice {
		for j := range zobristDice[i] {
			zobristDice[i][j] = next()
		}
	}
	for i := range zobristCubeValue {
		zobristCubeValue[i] = next()
	}
	for i := range zobristCubeOwner {
		zobristCubeOwner[i] = next()
	}
	for i := range zobristScore1 {
		zobristScore1[i] = next()
	}
	for i := range zobristScore2 {
		zobristScore2[i] = next()
	}
	for i := range zobristMatchLength {
		zobristMatchLength[i] = next()
	}
	zobristHasJacoby = next()
	zobristHasBeaver = next()
	for i := range zobristDecisionType {
		zobristDecisionType[i] = next()
	}
	for i := range zobristBearoff {
		for j := range zobristBearoff[i] {
			zobristBearoff[i][j] = next()
		}
	}
}

// cubeValueIndex maps a cube value (1,2,4,…,1024) to an index 0..10.
func cubeValueIndex(v int) int {
	idx := 0
	n := v
	for n > 1 {
		n >>= 1
		idx++
	}
	if idx > 10 {
		idx = 10
	}
	return idx
}

// cubeOwnerIndex maps cube owner (None=-1, Black=0, White=1) to index 0..2.
func cubeOwnerIndex(owner int) int {
	if owner < 0 {
		return 2
	}
	if owner > 1 {
		return 2
	}
	return owner
}

// ZobristHash computes a Zobrist hash for position identity.
// The position is normalized to player_on_roll=0 before hashing, so a
// position and its PlayerOnRoll=1 mirror always produce the same hash.
// Position.ID is excluded from the hash.
func ZobristHash(p *Position) uint64 {
	norm := p.NormalizeForStorage()
	norm.ID = 0

	var h uint64

	// Board points (indices 0=WhiteBar .. 25=BlackBar)
	for i, pt := range norm.Board.Points {
		if pt.Checkers <= 0 || pt.Color < 0 {
			continue
		}
		cnt := pt.Checkers
		if cnt > 15 {
			cnt = 15
		}
		h ^= zobristPoint[i][cnt][pt.Color]
	}

	// Bearoff counts
	for color := 0; color < 2; color++ {
		cnt := norm.Board.Bearoff[color]
		if cnt < 0 {
			cnt = 0
		}
		if cnt > 15 {
			cnt = 15
		}
		h ^= zobristBearoff[color][cnt]
	}

	// PlayerOnRoll is always 0 after normalization.
	h ^= zobristPlayerOnRoll[0]

	// Dice — treat as unordered pair: sort so (6,5) == (5,6).
	d1, d2 := norm.Dice[0], norm.Dice[1]
	if d1 > d2 {
		d1, d2 = d2, d1
	}
	if d1 >= 1 && d1 <= 6 && d2 >= 1 && d2 <= 6 {
		h ^= zobristDice[d1][d2]
	} else {
		h ^= zobristDice[0][0]
	}

	// Cube — Cube.Value is the exponent (0 = cube at 1, 1 = cube at 2, …).
	// Use it directly as the array index; do NOT apply cubeValueIndex() again
	// (cubeValueIndex applies log2 and is reserved for callers that have the
	// actual cube value such as 1, 2, 4, 8, …).
	h ^= zobristCubeValue[cubeExponentIndex(norm.Cube.Value)]
	h ^= zobristCubeOwner[cubeOwnerIndex(norm.Cube.Owner)]

	// Score (clamped to [0, 63])
	s1 := norm.Score[0]
	if s1 < 0 {
		s1 = 0
	} else if s1 > 63 {
		s1 = 63
	}
	s2 := norm.Score[1]
	if s2 < 0 {
		s2 = 0
	} else if s2 > 63 {
		s2 = 63
	}
	h ^= zobristScore1[s1]
	h ^= zobristScore2[s2]

	// Boolean flags
	if norm.HasJacoby != 0 {
		h ^= zobristHasJacoby
	}
	if norm.HasBeaver != 0 {
		h ^= zobristHasBeaver
	}

	// Decision type (0=CheckerAction, 1=CubeAction)
	dt := norm.DecisionType
	if dt < 0 || dt > 1 {
		dt = 0
	}
	h ^= zobristDecisionType[dt]

	return h
}

// cubeExponentIndex returns the Zobrist array index for a Position.Cube.Value.
// Position.Cube.Value is the cube exponent (0 = cube at 1, 1 = cube at 2,
// 2 = cube at 4, …, 10 = cube at 1024). This is a direct bounds clamp.
func cubeExponentIndex(exp int) int {
	if exp < 0 {
		return 0
	}
	if exp > 10 {
		return 10
	}
	return exp
}
