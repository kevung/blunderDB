// SPDX-License-Identifier: MIT

package gammonnet

// One search on several cores, bit for bit (gammonNet's
// docs/specs/recherche-parallele-spec.md, the Go transposition of its §3).
//
// # What is split, and what is not
//
// Four loops hand their independent subtrees out, then combine them serially
// in a fixed order once every subtree has returned:
//
//   - a chance node at depth >= 2: the 21 rolls (positionEquity), summed in
//     ascending roll index;
//   - a deep pass below the root at depth >= 2 with more than one candidate
//     (deepen): each task writes its own candidate's equity, the sort follows
//     on the calling goroutine;
//   - the root's deep passes (deepenGroups), one flattened queue of
//     candidate × roll in LPT order;
//   - the distribution backups at depth >= 2 (probsAt): each roll's
//     distribution in its own slot, accumulated in ascending roll index.
//
// The shallow pass of a node (generation, pruning, one batched fill of the
// siblings) stays on the node's goroutine: splitting it would empty the
// batches. Depth-1 nodes are not split: twenty-one passes of a dozen
// evaluations cost less than the hand-off.
//
// # Why the answer cannot move
//
// The network is bit for bit per position whatever batch carries it
// (ADR-0024); every sum is taken after the barrier in the fixed order, never
// as tasks finish (floating-point addition does not associate); each task
// writes only its own slot; and the shared cache returns only what the network
// would have computed. Parallelism decides who computes a term, never the
// order terms are added in.
//
// # The budget
//
// Cores is a pool of tokens. A fork lends a helper goroutine a token only when
// one is free, and runs the task on its own goroutine otherwise; it never
// waits for a token, so a fork nested inside a task can never deadlock however
// deep the tree. The process-wide budget (ProcessCores) is shared by every
// search that runs at once: a batch job holds one token per position it
// searches (Cores.Hold), and the tail of the batch, the live panel or a lone
// server request picks up whatever tokens are idle. Nothing ever asks for more
// goroutines than the budget, whatever stacks.

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Cores is a budget of goroutines that searches may run on at once.
type Cores struct {
	tokens chan struct{}
}

// NewCores is a budget of n tokens, for a component that must stay within a
// core count of its own (the server's AnalysisWorkers); everything else shares
// ProcessCores.
func NewCores(n int) *Cores {
	return &Cores{tokens: make(chan struct{}, max(n, 0))}
}

var processCores = sync.OnceValue(func() *Cores { return NewCores(runtime.GOMAXPROCS(0)) })

// ProcessCores is the budget the whole process shares: GOMAXPROCS tokens.
func ProcessCores() *Cores { return processCores() }

// Hold blocks until a token is free and takes it, for a job that keeps a core
// busy on its own (a batch goroutine searching one position). release gives
// it back; call it once.
func (c *Cores) Hold() (release func()) {
	c.tokens <- struct{}{}
	var once sync.Once
	return func() { once.Do(func() { <-c.tokens }) }
}

func (c *Cores) tryTake() bool {
	select {
	case c.tokens <- struct{}{}:
		return true
	default:
		return false
	}
}

func (c *Cores) give() { <-c.tokens }

// team is what a searcher and the helpers it lends tasks to share: the
// budget, the idle helpers, the cache, the cancellation switch. A helper is a
// Searcher over the same read-only networks with its own scratch.
type team struct {
	cores *Cores

	mu   sync.Mutex
	idle []*Searcher
}

// borrow lends an idle helper, or builds one, set up to run a subtree of s:
// same configuration, same match-state pair, same cache and switch.
func (t *team) borrow(s *Searcher) *Searcher {
	t.mu.Lock()
	var h *Searcher
	if n := len(t.idle); n > 0 {
		h = t.idle[n-1]
		t.idle = t.idle[:n-1]
	}
	t.mu.Unlock()
	if h == nil {
		h = &Searcher{
			net:   s.net,
			prune: s.prune,
			ev:    NewEvaluator(s.net),
			rolls: s.rolls,
			team:  t,
		}
		if s.prune != nil {
			h.pruneEv = NewEvaluator(s.prune)
		}
	}
	h.cfg = s.cfg
	h.cache = s.cache
	h.stop = s.stop
	h.matchStates, h.hasMatchState = s.matchStates, s.hasMatchState
	return h
}

func (t *team) giveBack(h *Searcher) {
	t.mu.Lock()
	t.idle = append(t.idle, h)
	t.mu.Unlock()
}

// searchCounts are a searcher's counters, carried back from a helper to the
// goroutine that opened the fork (gn_search.c's GnCounts).
type searchCounts struct {
	lent                         time.Duration
	evals, pruneEvals, cacheHits uint64
	batchFilled, batchSlotted    uint64
	cubeValuations, nestedForks  uint64
}

// takeCounts returns s's counters and zeroes them.
func (s *Searcher) takeCounts() searchCounts {
	c := searchCounts{s.lent, s.evals, s.pruneEvals, s.cacheHits, s.batchFilled, s.batchSlotted, s.cubeValuations, s.nestedForks}
	s.lent = 0
	s.evals, s.pruneEvals, s.cacheHits = 0, 0, 0
	s.batchFilled, s.batchSlotted, s.cubeValuations, s.nestedForks = 0, 0, 0, 0
	return c
}

func (s *Searcher) addCounts(c searchCounts) {
	s.lent += c.lent
	s.evals += c.evals
	s.pruneEvals += c.pruneEvals
	s.cacheHits += c.cacheHits
	s.batchFilled += c.batchFilled
	s.batchSlotted += c.batchSlotted
	s.cubeValuations += c.cubeValuations
	s.nestedForks += c.nestedForks
}

// LentTime is the time helpers have spent on this searcher's searches since
// it was built — what its forks cost beyond the caller's own wall time. Engine
// time accounting adds it: a search spread over eight cores costs eight cores.
// ResetCounters leaves it alone; read it between searches.
func (s *Searcher) LentTime() time.Duration { return s.lent }

// parallel reports whether s may fork at all.
func (s *Searcher) parallel() bool { return s.team != nil }

// fork runs task(w, k) for every k in [0, n) and returns once all have
// returned; false if any task did, or once the search is stopped. w is the
// searcher the task must use: s itself, or a helper lent for the task. Tasks
// are handed out in increasing k (callers order them longest first), pulled
// from one counter by the calling goroutine and by every helper it recruits;
// before each task it runs, the caller recruits a helper for every free token
// while more tasks remain unclaimed than goroutines pull them. A task writes only its own slot: fork
// returns before any slot is read.
func (s *Searcher) fork(n int, task func(w *Searcher, k int) bool) bool {
	t := s.team
	if t == nil || n < 2 {
		for k := 0; k < n; k++ {
			if !task(s, k) {
				return false
			}
		}
		return true
	}

	if s.inTask {
		s.nestedForks++
	}
	var (
		next    atomic.Int64
		active  atomic.Int64
		failed  atomic.Bool
		wg      sync.WaitGroup
		countMu sync.Mutex
		counts  searchCounts
	)
	pull := func(w *Searcher) {
		for !failed.Load() {
			k := int(next.Add(1)) - 1
			if k >= n {
				return
			}
			if !task(w, k) {
				failed.Store(true)
				return
			}
		}
	}
	for !failed.Load() {
		for int64(n)-next.Load() > active.Load()+1 && t.cores.tryTake() {
			active.Add(1)
			h := t.borrow(s)
			wg.Add(1)
			go func() {
				defer wg.Done()
				start := time.Now()
				h.inTask = true
				pull(h)
				c := h.takeCounts()
				c.lent += time.Since(start)
				t.giveBack(h)
				t.cores.give()
				countMu.Lock()
				counts.add(c)
				countMu.Unlock()
			}()
		}
		k := int(next.Add(1)) - 1
		if k >= n {
			break
		}
		outer := s.inTask
		s.inTask = true
		if !task(s, k) {
			failed.Store(true)
		}
		s.inTask = outer
	}
	wg.Wait()
	s.addCounts(counts)
	return !failed.Load()
}

func (c *searchCounts) add(o searchCounts) {
	c.lent += o.lent
	c.evals += o.evals
	c.pruneEvals += o.pruneEvals
	c.cacheHits += o.cacheHits
	c.batchFilled += o.batchFilled
	c.batchSlotted += o.batchSlotted
	c.cubeValuations += o.cubeValuations
	c.nestedForks += o.nestedForks
}

// splitDepth is the shallowest node whose subtrees are handed out (spec §2):
// below it a subtree costs less than the hand-off.
const splitDepth = 2
