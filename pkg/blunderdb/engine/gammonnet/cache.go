// SPDX-License-Identifier: MIT

package gammonnet

// The evaluation cache: a direct-mapped table from a position to the raw leaf
// distribution the network produced for it.
//
// # Why a hit can never change a result
//
// The key is the whole position and nothing else — no depth, no score, no cube.
// The value is the network's own output for that position, which does not
// depend on how the search arrived there. So a hit returns exactly what a miss
// would have computed. That property is what allows the cache to be per
// goroutine rather than shared: only the hit rate differs, never the answer.
//
// The reference keeps ONE shared cache and is explicit that its parallelism is
// by process. Giving each goroutine its own is what lets this port scale
// linearly without a lock, and it costs nothing semantically.
//
// The pruning pass must never read or write it. The small network's ordering
// would otherwise depend on evaluation history, which is the one way a cache
// could start changing results.

const (
	fnvOffset64 = 0xcbf29ce484222325
	fnvPrime64  = 0x100000001b3

	// defaultCacheLog2 sizes the table at 65 536 entries, about 3.7 MB. The
	// reference defaults to 2^19 because it keeps a single shared table; one
	// per goroutine has to be smaller.
	defaultCacheLog2 = 16
)

type cacheEntry struct {
	key      Position
	probs    [NumOutputs]float32
	occupied bool
}

type evalCache struct {
	entries []cacheEntry
	mask    uint64
	hits    uint64
	misses  uint64
}

func newEvalCache(log2 uint) *evalCache {
	if log2 == 0 || log2 > 24 {
		log2 = defaultCacheLog2
	}
	n := uint64(1) << log2
	return &evalCache{entries: make([]cacheEntry, n), mask: n - 1}
}

// hashPosition is FNV-1a over the 29 bytes the reference hashes, in the same
// order: points, then bar, then off, then turn.
func hashPosition(p *Position) uint64 {
	h := uint64(fnvOffset64)
	for _, b := range p.Points {
		h ^= uint64(uint8(b))
		h *= fnvPrime64
	}
	for _, b := range p.Bar {
		h ^= uint64(b)
		h *= fnvPrime64
	}
	for _, b := range p.Off {
		h ^= uint64(b)
		h *= fnvPrime64
	}
	h ^= uint64(p.Turn)
	h *= fnvPrime64
	return h
}

// lookup returns the stored distribution for p, if any. The full key is
// compared, so a hash collision never produces a wrong answer — it produces a
// miss.
func (c *evalCache) lookup(p *Position, out *[NumOutputs]float32) bool {
	e := &c.entries[hashPosition(p)&c.mask]
	if !e.occupied || e.key != *p {
		c.misses++
		return false
	}
	*out = e.probs
	c.hits++
	return true
}

// store writes p's distribution, replacing whatever shared its slot. Newest
// wins: no probing, no chaining.
func (c *evalCache) store(p *Position, probs *[NumOutputs]float32) {
	e := &c.entries[hashPosition(p)&c.mask]
	e.key = *p
	e.probs = *probs
	e.occupied = true
}
