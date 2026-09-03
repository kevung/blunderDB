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
	// per goroutine has to be smaller. This is the ROOT's own size — the
	// root is where a transposition across the whole tree can recur, so it
	// keeps the reference's proportions. A WORKER's cache is workerCacheLog2,
	// far smaller (#196/C.9): see that constant's own comment for why.
	defaultCacheLog2 = 16

	// workerCacheLog2 sizes a WORKER's own cache (search.go's WithWorkers) —
	// deliberately far smaller than the root's: a worker only ever drains
	// its share of ONE flattened deepenGroups queue (search.go) for the
	// life of one decision, never the whole tree, so its working set of
	// distinct positions is small regardless of how many cores share the
	// job.
	//
	// Measured (#196/C.9), Counters() across a canonical 2-ply Probs call at
	// 16 workers, both the opening position and a busy contact position
	// (search_test.go's busyXGID): big-network evals stay flat — within
	// noise (<0.2%) — from 2^10 through 2^16 entries. The "+4.5% d'évals en
	// double" the ticket opened with is NOT a cache-size effect (shrinking
	// the cache 64× does not move it) — it is inherent to workers not
	// sharing state at all, by design (cache.go's own doc comment: a
	// per-goroutine cache is what lets this scale without a lock). So 2^12
	// (4 096 entries, ~230 KB/worker) keeps a margin over the measured
	// floor while cutting the pool's cache footprint 16× — 58.7 MB down to
	// 3.7 MB total at 16 workers, the same total the OLD single-worker
	// default cost by itself.
	workerCacheLog2 = 12
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
