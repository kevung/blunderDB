package rollout

import "math/bits"

// stratumGames is the size of a full quasi-random block: the first two rolls
// of a game take every one of the 36×36 ordered pairs exactly once per block.
const stratumGames = 36 * 36

// splitmix64 is the generator behind every die: small, with no state but one
// word, so a game's dice are a pure function of (seed, game) and never of
// the worker or the order games complete in.
func splitmix64(state *uint64) uint64 {
	*state += 0x9E3779B97F4A7C15
	z := *state
	z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
	z = (z ^ (z >> 27)) * 0x94D049BB133111EB
	return z ^ (z >> 31)
}

// below36 maps a uniform 64-bit word onto 0..35 by its high bits, without the
// modulo bias a remainder would carry.
func below36(x uint64) int {
	hi, _ := bits.Mul64(x, 36)
	return int(hi)
}

// mix derives an independent stream from a seed and a label.
func mix(seed, label uint64) uint64 {
	s := seed ^ (label+1)*0xD1B54A32D192ED03
	splitmix64(&s)
	return splitmix64(&s)
}

// gameDice is the roll sequence of one game. Every candidate of a rollout
// replays the same gameDice for a given game number: common random numbers,
// so a lucky sequence lifts every candidate alike and the difference keeps
// only what the plays changed.
//
// The first two rolls are stratified. Within a block of 1296 games, game u
// opens with permutation one's roll u mod 36, and its second roll is
// permutation two's entry (u/36 + u mod 36) mod 36 — a cyclic Latin square,
// so any multiple of 36 games sees each first roll equally often and each
// second roll equally often, and 1296 games see every ordered pair once.
// The permutations are redrawn per block, so blocks do not repeat. From the
// third roll on, the dice are the game's own pseudo-random stream.
type gameDice struct {
	first, second int
	stream        uint64
}

func newGameDice(seed uint64, game int) gameDice {
	block := uint64(game / stratumGames)
	u := game % stratumGames
	p1 := permutation36(mix(seed, block<<1))
	p2 := permutation36(mix(seed, block<<1|1))
	return gameDice{
		first:  int(p1[u%36]),
		second: int(p2[(u/36+u%36)%36]),
		stream: mix(seed, 1<<40|uint64(game)),
	}
}

// roll returns the dice of half-move ply; plies must be asked in order, each
// once, since from the third on they come off the game's stream.
func (g *gameDice) roll(ply int) (d1, d2 int) {
	var idx int
	switch ply {
	case 0:
		idx = g.first
	case 1:
		idx = g.second
	default:
		idx = below36(splitmix64(&g.stream))
	}
	return idx/6 + 1, idx%6 + 1
}

// permutation36 is a Fisher–Yates shuffle of 0..35 driven by seed.
func permutation36(seed uint64) [36]uint8 {
	var p [36]uint8
	for i := range p {
		p[i] = uint8(i)
	}
	s := seed
	for i := 35; i > 0; i-- {
		hi, _ := bits.Mul64(splitmix64(&s), uint64(i+1))
		j := int(hi)
		p[i], p[j] = p[j], p[i]
	}
	return p
}
