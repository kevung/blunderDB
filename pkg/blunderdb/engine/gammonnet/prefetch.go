package gammonnet

import (
	"context"
	"sync"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// Descending a roll — the map of the opponent's replies to a roll's best
// play — should answer at once. A Prefetcher computes, behind the map on
// screen, the maps one roll below it into a MapCache, the cache the display
// reads: every child at 0-ply, then every child at 1-ply, then the child of
// the row looked at alone at the display depth (a 2-ply map costs a hundred
// times a 1-ply one, and only the branch looked at is worth it). The row
// looked at goes first at every depth.
//
// The phases computed are those the display will look up (rollMap's upTo):
// a descent finds them under its own keys and replays them bit for bit,
// then searches only what is missing. The search is the display's, run by
// another executor: the cache holds no value a fresh computation would not
// give.
//
// A Prefetcher runs one request at a time. A different request — another
// branch, another row looked at — cancels the one running; the same request
// again while it runs is that run, not a second one.

// PrefetchRequest is the map on screen and what to compute below it.
type PrefetchRequest struct {
	// Pos is the position of the map on screen, its roller not yet rolled.
	Pos domain.Position
	// Rows are its 21 rows: each row's play (Move, empty for a dance) leads
	// to one child.
	Rows []RollMapRow
	// MET values the maps (nil: the built-in table).
	MET *engine.MET
	// Ply and PruneK are the display's: the children are computed as a
	// descent will ask them.
	Ply, PruneK int
	// Focus is the index of the row looked at, -1 for none: its child alone
	// goes past 1-ply.
	Focus int
}

// PrefetchFunc is told, once per child and depth, that the child of row
// index is held in the cache up to ply. Calls are serialised.
type PrefetchFunc func(index, ply int)

// Prefetcher computes PrefetchRequests one at a time into a MapCache.
type Prefetcher struct {
	cache   *MapCache
	scope   string
	exec    PrefetchExec
	onPanic func(any)

	mu      sync.Mutex
	key     prefetchKey
	running bool
	cancel  context.CancelFunc
	done    chan struct{}
}

// PrefetchExec builds the executor of one request's phases; ctx is the
// request's, which a newer request cancels.
type PrefetchExec func(ctx context.Context, ply, pruneK int) CellExec

// NewPrefetcher computes into cache under scope, each child's phases run by
// exec — BackgroundExec on the desktop, so that the computation on screen
// keeps its cores. onPanic receives a panic of a request's goroutine — its
// search, its executor's workers, its PrefetchFunc — which then ends that
// request alone; nil lets the panic through.
func NewPrefetcher(cache *MapCache, scope string, exec PrefetchExec, onPanic func(any)) *Prefetcher {
	return &Prefetcher{cache: cache, scope: scope, exec: exec, onPanic: onPanic}
}

type prefetchKey struct {
	pos    domain.Position
	moves  [NumRolls]string
	met    string
	ply    int
	pruneK int
	focus  int
}

// Start computes req under ctx and returns a channel closed once it has
// ended, finished or cancelled. onChild (may be nil) follows its progress.
// A request equal to the one still running returns that run's channel, and
// its own onChild is not called; any other cancels the running one first.
// An ill-formed request (not 21 rows) starts nothing.
func (p *Prefetcher) Start(ctx context.Context, req PrefetchRequest, onChild PrefetchFunc) <-chan struct{} {
	key, ok := p.keyOf(req)
	p.mu.Lock()
	defer p.mu.Unlock()
	if ok && p.running && p.key == key {
		return p.done
	}
	p.stopLocked()
	done := make(chan struct{})
	if !ok {
		close(done)
		return done
	}
	ctx, cancel := context.WithCancel(ctx)
	p.key, p.running, p.cancel, p.done = key, true, cancel, done
	go func() {
		defer close(done)
		defer func() {
			cancel()
			p.mu.Lock()
			if p.done == done {
				p.running = false
			}
			p.mu.Unlock()
		}()
		defer func() {
			if r := recover(); r != nil {
				if p.onPanic == nil {
					panic(r)
				}
				p.onPanic(r)
			}
		}()
		p.run(ctx, req, onChild)
	}()
	return done
}

// Stop cancels the request running, if any, without waiting for it.
func (p *Prefetcher) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stopLocked()
}

func (p *Prefetcher) stopLocked() {
	if p.cancel != nil {
		p.cancel()
	}
	p.running, p.cancel = false, nil
}

func (p *Prefetcher) keyOf(req PrefetchRequest) (prefetchKey, bool) {
	if len(req.Rows) != NumRolls || req.Ply < 0 || req.Ply > MaxPly {
		return prefetchKey{}, false
	}
	k := prefetchKey{pos: keyPosition(req.Pos), met: p.cache.metDigest(req.MET), ply: req.Ply, pruneK: req.PruneK, focus: req.Focus}
	k.pos.Dice = [2]int{0, 0}
	for i, r := range req.Rows {
		k.moves[i] = r.Move
	}
	return k, true
}

// run computes the children depth by depth, the row looked at first.
func (p *Prefetcher) run(ctx context.Context, req PrefetchRequest, onChild PrefetchFunc) {
	order := make([]int, 0, NumRolls)
	if req.Focus >= 0 && req.Focus < NumRolls {
		order = append(order, req.Focus)
	}
	for i := range NumRolls {
		if i != req.Focus {
			order = append(order, i)
		}
	}
	var children [NumRolls]*domain.Position
	for _, i := range order {
		r := req.Rows[i]
		if child, err := PositionAfterPlay(req.Pos, r.Die1, r.Die2, r.Move); err == nil {
			children[i] = &child
		}
	}

	exec := p.exec(ctx, req.Ply, req.PruneK)
	step := func(i, upTo int) bool {
		if children[i] == nil {
			return true
		}
		_, err := p.cache.rollMap(ctx, p.scope, *children[i], req.MET, req.Ply, req.PruneK, upTo, exec, nil)
		if ctx.Err() != nil {
			return false
		}
		if err != nil {
			children[i] = nil // not a map: a finished game, a broken search
			return true
		}
		if onChild != nil {
			onChild(i, upTo)
		}
		return true
	}
	for _, depth := range mapDepths(min(req.Ply, 1)) {
		for _, i := range order {
			if !step(i, depth) {
				return
			}
		}
	}
	if req.Ply >= 2 && req.Focus >= 0 && req.Focus < NumRolls {
		step(req.Focus, req.Ply)
	}
}

// BackgroundExec is the CellExec of a computation that yields to the one on
// screen (ADR-0120): workers goroutines (0: one per token of cores), each
// taking its token with HoldBackground for one cell at a time, its searcher
// parking at each roll while a visible computation claims cores. A claim
// therefore stops it from starting cells and parks the ones it runs; ctx
// ending while it waits for a token ends the phase with nothing more run. A
// worker whose searcher cannot be built runs nothing and the phase returns
// why; a panic in a worker is raised again on the caller's goroutine, once
// the others have stopped.
func BackgroundExec(cores *Cores, workers int) PrefetchExec {
	if workers <= 0 {
		workers = max(cores.Cap(), 1)
	}
	return func(ctx context.Context, ply, pruneK int) CellExec {
		return func(n int, cell func(int, *Searcher)) error {
			var mu sync.Mutex
			next := 0
			var failed error
			var panicked any
			var wg sync.WaitGroup
			for range min(workers, n) {
				wg.Go(func() {
					defer func() {
						if r := recover(); r != nil {
							mu.Lock()
							if panicked == nil {
								panicked = r
							}
							next = n // the others take no further cell
							mu.Unlock()
						}
					}()
					searcher, err := NewBatchSearcher(ply, pruneK)
					if err != nil {
						mu.Lock()
						if failed == nil {
							failed = err
						}
						mu.Unlock()
						return
					}
					searcher.YieldsTo(cores)
					if CoresFor(ply) != nil {
						searcher.WithCores(cores)
					}
					for {
						mu.Lock()
						i := next
						next++
						mu.Unlock()
						if i >= n {
							return
						}
						release, err := cores.HoldBackground(ctx)
						if err != nil {
							return
						}
						func() {
							defer release()
							cell(i, searcher)
						}()
					}
				})
			}
			wg.Wait()
			if panicked != nil {
				panic(panicked)
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			return failed
		}
	}
}
