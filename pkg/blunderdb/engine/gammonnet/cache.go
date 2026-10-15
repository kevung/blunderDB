// SPDX-License-Identifier: MIT

package gammonnet

import "sync"

// The evaluation cache: a direct-mapped table from a position to the raw leaf
// distribution the network produced for it.
//
// # Why a hit can never change a result
//
// The key is the whole position and nothing else — no depth, score or cube —
// and the value is the network's output for it, so a hit returns exactly what
// a miss would compute. That is what allows a searcher to own its table without
// a lock, or to share it with its team behind one lock per slice of slots (as
// the reference does): only the hit rate differs.
//
// The pruning pass must never read or write it, or the small network's
// ordering would depend on evaluation history.

const (
	fnvOffset64 = 0xcbf29ce484222325
	fnvPrime64  = 0x100000001b3

	// defaultCacheLog2 sizes the ROOT's table at 65 536 entries (~3.7 MB),
	// smaller than the reference's shared 2^19; the root is where
	// transpositions across the whole tree recur.
	defaultCacheLog2 = 16

	// cacheLockLog2 is how many locks guard a shared table: one per slice
	// of slots, gn_evalcache.c's 4096. A reader racing a writer on one slot
	// could see the whole key of one position and half the probabilities of
	// another.
	cacheLockLog2 = 12
)

type cacheEntry struct {
	key      Position
	probs    [NumOutputs]float32
	occupied bool
}

type evalCache struct {
	entries []cacheEntry
	mask    uint64

	// locks is nil while one goroutine owns the table; share allocates it
	// before a team reads and writes it at once.
	locks    []sync.Mutex
	lockMask uint64
}

func newEvalCache(log2 uint) *evalCache {
	if log2 == 0 || log2 > 24 {
		log2 = defaultCacheLog2
	}
	n := uint64(1) << log2
	return &evalCache{entries: make([]cacheEntry, n), mask: n - 1}
}

// share makes the table safe for concurrent use. Called once, before the
// first helper is lent; idempotent.
func (c *evalCache) share() {
	if c.locks != nil {
		return
	}
	n := c.mask + 1
	if n > 1<<cacheLockLog2 {
		n = 1 << cacheLockLog2
	}
	c.locks = make([]sync.Mutex, n)
	c.lockMask = n - 1
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
	i := hashPosition(p) & c.mask
	e := &c.entries[i]
	if c.locks == nil {
		if !e.occupied || e.key != *p {
			return false
		}
		*out = e.probs
		return true
	}
	l := &c.locks[i&c.lockMask]
	l.Lock()
	hit := e.occupied && e.key == *p
	if hit {
		*out = e.probs
	}
	l.Unlock()
	return hit
}

// store writes p's distribution, replacing whatever shared its slot. Newest
// wins: no probing, no chaining.
func (c *evalCache) store(p *Position, probs *[NumOutputs]float32) {
	i := hashPosition(p) & c.mask
	e := &c.entries[i]
	if c.locks == nil {
		e.key, e.probs, e.occupied = *p, *probs, true
		return
	}
	l := &c.locks[i&c.lockMask]
	l.Lock()
	e.key, e.probs, e.occupied = *p, *probs, true
	l.Unlock()
}
