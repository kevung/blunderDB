package server

import (
	"context"
	"sync"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
)

// analysisPool is the daemon's one set of engine workers, shared by every
// tenant. A request does not start goroutines of its own: it queues a job,
// and the workers take the job's positions one at a time, going round the
// tenants that have work. Two tenants sweeping at once each get half the
// cores, not all of them each; a tenant that asks for one evaluation while
// another sweeps a million positions waits for one position, not for the
// sweep.
//
// A tenant's weight (Options.AnalysisWeights, 1 when unlisted) is how many
// positions it is served per turn: a weight of 3 takes three positions
// while a tenant of weight 1 takes one. Within a tenant, jobs run in the
// order they came.
type analysisPool struct {
	workers int
	weights map[string]int

	start sync.Once
	mu    sync.Mutex
	cond  *sync.Cond
	// queues holds each tenant's jobs in arrival order; ring the tenants
	// with work, in turn order. turn indexes ring and credit is what ring[turn]
	// may still take this turn.
	queues map[string][]*poolJob
	ring   []string
	turn   int
	credit int
	closed bool
}

// searcherFor hands a unit the worker's own Searcher for a ply and pruning
// width, made on first use and kept; nil when one cannot be made, which the
// evaluation functions take as "make one per position".
type searcherFor func(ply, pruneK int) *gammonnet.Searcher

// poolJob is one request's work: next hands the pool the job's next unit,
// or false once there is none (the positions are exhausted, the request was
// cancelled, the tenant's engine time is spent). It is called under the
// pool's lock and must not block.
type poolJob struct {
	next func() (func(searcherFor), bool)

	inflight  int
	exhausted bool
	done      chan struct{}
}

// wait blocks until the job has handed out its last unit and every unit
// handed out has run.
func (j *poolJob) wait() { <-j.done }

func newAnalysisPool(workers int, weights map[string]int) *analysisPool {
	p := &analysisPool{workers: max(workers, 1), weights: weights, queues: map[string][]*poolJob{}}
	p.cond = sync.NewCond(&p.mu)
	return p
}

// submit queues a job for scope and returns it; the caller waits on it.
func (p *analysisPool) submit(scope string, next func() (func(searcherFor), bool)) *poolJob {
	p.start.Do(func() {
		for range p.workers {
			go p.work()
		}
	})
	j := &poolJob{next: next, done: make(chan struct{})}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		j.exhausted = true
		close(j.done)
		return j
	}
	if len(p.queues[scope]) == 0 {
		p.ring = append(p.ring, scope)
	}
	p.queues[scope] = append(p.queues[scope], j)
	p.cond.Broadcast()
	return j
}

// run queues one unit for scope, waits for it and reports whether it ran:
// false when ctx ended before its turn came or the pool was closed.
func (p *analysisPool) run(ctx context.Context, scope string, unit func(searcherFor)) bool {
	taken, ran := false, false
	p.submit(scope, func() (func(searcherFor), bool) {
		if taken || ctx.Err() != nil {
			return nil, false
		}
		taken = true
		return func(get searcherFor) {
			if ctx.Err() != nil {
				return
			}
			ran = true
			unit(get)
		}, true
	}).wait()
	return ran
}

// close stops the workers once their current unit is done; a job still
// queued is ended unrun.
func (p *analysisPool) close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return
	}
	p.closed = true
	for _, jobs := range p.queues {
		for _, j := range jobs {
			p.exhaustLocked(j)
		}
	}
	p.queues, p.ring = map[string][]*poolJob{}, nil
	p.cond.Broadcast()
}

func (p *analysisPool) weight(scope string) int {
	if w := p.weights[scope]; w > 0 {
		return w
	}
	return 1
}

// exhaustLocked marks j as handing out no more units, and ends it when none
// is running.
func (p *analysisPool) exhaustLocked(j *poolJob) {
	if j.exhausted {
		return
	}
	j.exhausted = true
	if j.inflight == 0 {
		close(j.done)
	}
}

// takeLocked returns the next unit to run, going round the tenants, or
// false when no job has one.
func (p *analysisPool) takeLocked() (*poolJob, func(searcherFor), bool) {
	for len(p.ring) > 0 {
		if p.turn >= len(p.ring) {
			p.turn = 0
		}
		scope := p.ring[p.turn]
		if p.credit <= 0 {
			p.credit = p.weight(scope)
		}
		jobs := p.queues[scope]
		for len(jobs) > 0 {
			j := jobs[0]
			if unit, ok := j.next(); ok {
				j.inflight++
				p.queues[scope] = jobs
				if p.credit--; p.credit <= 0 {
					p.turn++
				}
				return j, unit, true
			}
			p.exhaustLocked(j)
			jobs = jobs[1:]
		}
		// This tenant has nothing left: it leaves the ring, and the next
		// tenant, now at the same index, starts a fresh turn.
		delete(p.queues, scope)
		p.ring = append(p.ring[:p.turn], p.ring[p.turn+1:]...)
		p.credit = 0
	}
	return nil, nil, false
}

func (p *analysisPool) work() {
	searchers := map[[2]int]*gammonnet.Searcher{}
	get := func(ply, pruneK int) *gammonnet.Searcher {
		k := [2]int{ply, pruneK}
		if s, ok := searchers[k]; ok {
			return s
		}
		s, err := gammonnet.NewBatchSearcher(ply, pruneK)
		if err != nil {
			s = nil
		}
		searchers[k] = s
		return s
	}
	for {
		p.mu.Lock()
		var (
			j    *poolJob
			unit func(searcherFor)
			ok   bool
		)
		for !p.closed {
			if j, unit, ok = p.takeLocked(); ok {
				break
			}
			p.cond.Wait()
		}
		if p.closed && !ok {
			p.mu.Unlock()
			return
		}
		p.mu.Unlock()

		unit(get)

		p.mu.Lock()
		j.inflight--
		if j.exhausted && j.inflight == 0 {
			close(j.done)
		}
		p.mu.Unlock()
	}
}
