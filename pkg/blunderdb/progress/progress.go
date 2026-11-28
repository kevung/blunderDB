// Package progress carries a long operation's progress from the engine to
// whoever launched it: the desktop's panel of tasks, the CLI's stderr.
//
// The reporter travels in the context rather than in every signature: an
// export, an archive or a printed sheet goes through several layers
// (database, ingest, printsheet) that all take a context already, and the
// report is observation only — an operation runs the same with or without
// one. A layer that knows how far it is calls Report; a context without a
// reporter makes Report a no-op.
package progress

import (
	"context"
	"sync"
	"time"
)

// Func receives how many units of an operation are done of total; total 0
// means the operation does not know its size yet.
type Func func(done, total int)

type key struct{}

// With returns a context whose operations report to f.
func With(ctx context.Context, f Func) context.Context {
	if f == nil {
		return ctx
	}
	return context.WithValue(ctx, key{}, f)
}

// Report tells the context's reporter, if any, how far the operation is.
func Report(ctx context.Context, done, total int) {
	if f, ok := ctx.Value(key{}).(Func); ok {
		f(done, total)
	}
}

// Gap is the least time between two reports a Throttle forwards: about four
// a second, as often as a progress bar is worth redrawing.
const Gap = 250 * time.Millisecond

// Throttle forwards to f at most one report per gap, except the first, the
// last (done == total) and any change of total, which always pass. Safe for
// concurrent use.
func Throttle(gap time.Duration, f Func) Func {
	var mu sync.Mutex
	var last time.Time
	lastTotal := -1
	return func(done, total int) {
		mu.Lock()
		now := time.Now()
		pass := total != lastTotal || (total > 0 && done >= total) || now.Sub(last) >= gap
		if pass {
			last, lastTotal = now, total
		}
		mu.Unlock()
		if pass {
			f(done, total)
		}
	}
}
