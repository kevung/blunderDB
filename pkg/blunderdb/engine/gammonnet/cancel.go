// SPDX-License-Identifier: MIT

package gammonnet

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
)

// ErrCanceled is what a search interrupted through BindContext returns. It
// wraps context.Canceled so a caller that only knows contexts recognises it.
var ErrCanceled = fmt.Errorf("gammonnet: search canceled: %w", context.Canceled)

// IsCanceled reports whether err comes from an interrupted search.
func IsCanceled(err error) bool { return errors.Is(err, ErrCanceled) }

// stopSwitch is the cancellation point a searcher and its workers share. Each
// binding gets a flag of its own, so a context that fires after its binding
// was released can never stop the next search.
type stopSwitch struct {
	flag atomic.Pointer[binding]
}

// binding is one BindContext: the flag the recursion reads, and the context's
// done channel, which wakes a search parked behind a claim (Cores.yield).
type binding struct {
	stop atomic.Bool
	done <-chan struct{}
}

// stopped is the checkpoint the recursion reads once per roll: an atomic
// load against a subtree of evaluations, invisible in the timings.
func (s *Searcher) stopped() bool {
	f := s.stop.flag.Load()
	return f != nil && f.stop.Load()
}

// stopDone is the done channel of the current binding; nil (never ready)
// without one.
func (s *Searcher) stopDone() <-chan struct{} {
	if f := s.stop.flag.Load(); f != nil {
		return f.done
	}
	return nil
}

// BindContext ties the searches run until release to ctx: once ctx is done,
// the search in flight unwinds at its next roll and returns ErrCanceled (Probs
// reports false), parked behind a claim or not. A search ctx never interrupts
// returns the same result, bit for bit: the checkpoint only reads a flag.
//
// One binding at a time; the searcher is not goroutine-safe, as ever.
func (s *Searcher) BindContext(ctx context.Context) (release func()) {
	f := &binding{done: ctx.Done()}
	f.stop.Store(ctx.Err() != nil)
	s.stop.flag.Store(f)
	unhook := context.AfterFunc(ctx, func() { f.stop.Store(true) })
	return func() {
		unhook()
		s.stop.flag.CompareAndSwap(f, nil)
	}
}

// Canceled reports whether the current binding's context is done.
func (s *Searcher) Canceled() bool { return s.stopped() }
