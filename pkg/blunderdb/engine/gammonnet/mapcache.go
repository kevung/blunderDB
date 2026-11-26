package gammonnet

import (
	"container/list"
	"sync"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// MapCache keeps the phases of roll and score maps already computed, so that
// the same question is answered without a search: a map asked again, a map at
// a depth an earlier, deeper map passed through, or a map computed ahead of
// the user (the next roll's) before it is asked.
//
// A phase is kept under everything its values depend on, and nothing else is
// trusted to coincide:
//
//   - the scope, so that two tenants of the daemon never share an entry, nor
//     learn from a fast answer what another one studied;
//   - the position, every field but those that only record where it came from
//     (ID, IndividuallyImported, Flagged): a field added to domain.Position
//     joins the key by itself;
//   - the match equity table by its digest (nil is the built-in table);
//   - the search configuration the phase ran with, whole (depth, filters,
//     PruneK, root stage, cube and match state): a roll map's provisional
//     phase runs with its target's filters at a shallower depth, and is shared
//     with a shallower map only when that configuration is the same;
//   - the kind of map and, for a score map, the match length.
//
// The network, the kernel and the published search levels are fixed for the
// life of the process, which is the life of the cache. The workers and the
// executor are not in the key: the map does not depend on them
// (TestRollMap_FinalRowsAreTheRowsSearchedAlone).
//
// Memory is bounded by an estimate of the entries' size; the least recently
// used are dropped first. A MapCache is safe for concurrent use; two calls
// computing the same phase at once both search, and the later one is kept.
type MapCache struct {
	mu      sync.Mutex
	budget  int64
	used    int64
	order   *list.List // front: most recently used
	entries map[mapKey]*list.Element
}

// DefaultMapCacheBytes is the budget of the process's MapCache: a few
// thousand roll maps, or a few hundred full 15-point score maps.
const DefaultMapCacheBytes = 32 << 20

// NewMapCache is an empty cache holding about budget bytes of maps.
func NewMapCache(budget int64) *MapCache {
	return &MapCache{budget: budget, order: list.New(), entries: map[mapKey]*list.Element{}}
}

var processMaps = sync.OnceValue(func() *MapCache { return NewMapCache(DefaultMapCacheBytes) })

// Maps is the process's cache, the one the desktop app, the CLI and the
// daemon compute their maps through.
func Maps() *MapCache { return processMaps() }

const (
	rollMapKind = iota + 1
	scoreMapKind
)

type mapKey struct {
	scope  string
	kind   int
	pos    domain.Position
	met    string
	cfg    SearchConfig
	length int
}

type mapEntry struct {
	key   mapKey
	rows  []RollMapRow
	cells []ScoreMapCell
	cost  int64
}

// keyPosition drops the fields that record a position's provenance, not its
// game.
func keyPosition(pos domain.Position) domain.Position {
	pos.ID = 0
	pos.IndividuallyImported = false
	pos.Flagged = false
	return pos
}

// keyConfig is cfg without the table pointer, which the key holds by digest.
func keyConfig(cfg SearchConfig) SearchConfig {
	cfg.Match.MET = nil
	return cfg
}

func rollMapKey(scope string, pos domain.Position, met string, cfg SearchConfig) mapKey {
	return mapKey{scope: scope, kind: rollMapKind, pos: keyPosition(pos), met: met, cfg: keyConfig(cfg)}
}

// scoreMapKey: a score map's cells each run ConfigForPositionMET at their own
// score, so the shared part of their configuration is the level at depth.
func scoreMapKey(scope string, pos domain.Position, met string, length, depth, pruneK int) mapKey {
	cfg := DefaultConfig(depth)
	if pruneK > 0 {
		cfg.PruneK = pruneK
	}
	return mapKey{scope: scope, kind: scoreMapKind, pos: keyPosition(pos), met: met, cfg: cfg, length: length}
}

// metDigest is met's digest, or nothing when c keeps nothing.
func (c *MapCache) metDigest(met *engine.MET) string {
	if c == nil {
		return ""
	}
	return met.Digest()
}

func (c *MapCache) rows(k mapKey) ([]RollMapRow, bool) {
	e, ok := c.get(k)
	if !ok {
		return nil, false
	}
	return cloneRows(e.rows), true
}

func (c *MapCache) cells(k mapKey) ([]ScoreMapCell, bool) {
	e, ok := c.get(k)
	if !ok {
		return nil, false
	}
	return cloneCells(e.cells), true
}

func (c *MapCache) putRows(k mapKey, rows []RollMapRow) {
	if c == nil {
		return
	}
	cost := int64(256)
	for _, r := range rows {
		cost += 96 + int64(len(r.Move))
	}
	c.put(&mapEntry{key: k, rows: cloneRows(rows), cost: cost})
}

func (c *MapCache) putCells(k mapKey, cells []ScoreMapCell) {
	if c == nil {
		return
	}
	cost := int64(256)
	for _, x := range cells {
		cost += 192 + int64(len(x.Move)+len(x.SecondMove)+len(x.Reason)+len(x.Verdict))
	}
	c.put(&mapEntry{key: k, cells: cloneCells(cells), cost: cost})
}

func (c *MapCache) get(k mapKey) (*mapEntry, bool) {
	if c == nil {
		return nil, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.entries[k]
	if !ok {
		return nil, false
	}
	c.order.MoveToFront(el)
	return el.Value.(*mapEntry), true
}

func (c *MapCache) put(e *mapEntry) {
	if e.cost > c.budget {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.entries[e.key]; ok {
		c.used -= el.Value.(*mapEntry).cost
		c.order.Remove(el)
	}
	c.entries[e.key] = c.order.PushFront(e)
	c.used += e.cost
	for c.used > c.budget {
		last := c.order.Back()
		old := last.Value.(*mapEntry)
		c.order.Remove(last)
		delete(c.entries, old.key)
		c.used -= old.cost
	}
}

// Len is the number of phases held.
func (c *MapCache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}

// cloneRows copies rows down to their pointers, so that neither a caller nor
// the cache can change what the other holds.
func cloneRows(rows []RollMapRow) []RollMapRow {
	out := make([]RollMapRow, len(rows))
	for i, r := range rows {
		if r.MarketLost != nil {
			v := *r.MarketLost
			r.MarketLost = &v
		}
		out[i] = r
	}
	return out
}

func cloneCells(cells []ScoreMapCell) []ScoreMapCell {
	out := make([]ScoreMapCell, len(cells))
	for i, x := range cells {
		x.SecondEquity = clonePtr(x.SecondEquity)
		x.Gap = clonePtr(x.Gap)
		out[i] = x
	}
	return out
}

func clonePtr(p *float64) *float64 {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}
