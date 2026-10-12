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
	flag atomic.Pointer[atomic.Bool]
}

// stopped is the checkpoint the recursion reads once per roll: an atomic
// load against a subtree of evaluations, invisible in the timings.
func (s *Searcher) stopped() bool {
	f := s.stop.flag.Load()
	return f != nil && f.Load()
}

// BindContext ties the searches run until release to ctx: once ctx is done,
// the search in flight unwinds at its next roll and returns ErrCanceled (Probs
// reports false). A search ctx never interrupts returns the same result, bit
// for bit: the checkpoint only reads a flag.
//
// One binding at a time; the searcher is not goroutine-safe, as ever.
func (s *Searcher) BindContext(ctx context.Context) (release func()) {
	f := new(atomic.Bool)
	f.Store(ctx.Err() != nil)
	s.stop.flag.Store(f)
	unhook := context.AfterFunc(ctx, func() { f.Store(true) })
	return func() {
		unhook()
		s.stop.flag.CompareAndSwap(f, nil)
	}
}

// Canceled reports whether the current binding's context is done.
func (s *Searcher) Canceled() bool { return s.stopped() }
