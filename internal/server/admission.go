package server

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

// heavyGate admits the requests that scan the library a few at a time. Left
// unbounded, a burst of them holds every pooled connection for seconds and a
// read of one row, a millisecond of work, queues behind them for the pool.
// Past the slots a request waits its turn; past the queue, or after
// heavyWait, it is answered unavailable with Retry-After rather than kept
// until the client gives up.
type heavyGate struct {
	slots   chan struct{}
	waiting atomic.Int64
	queue   int64
	wait    time.Duration
}

// heavyQueuePerSlot bounds the queue: beyond it a request would wait longer
// than heavyWait on any library where such a read takes seconds.
const (
	heavyQueuePerSlot = 8
	heavyWait         = 20 * time.Second
	heavyRetryAfter   = "5"
)

func newHeavyGate(slots int) *heavyGate {
	if slots <= 0 {
		return nil
	}
	return &heavyGate{slots: make(chan struct{}, slots), queue: int64(slots) * heavyQueuePerSlot, wait: heavyWait}
}

// heavyRoute reports whether a route reads across the library: every
// statistic, a search other than parsing it, the match list.
func heavyRoute(pattern string) bool {
	switch pattern {
	case "/v1/search.count", "/v1/search.find", "/v1/search.ids", "/v1/search.query", "/v1/search.indexOf", "/v1/matches.list":
		return true
	}
	return strings.HasPrefix(pattern, "/v1/stats.")
}

func (g *heavyGate) wrap(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		select {
		case g.slots <- struct{}{}:
		default:
			if g.waiting.Add(1) > g.queue {
				g.waiting.Add(-1)
				g.refuse(w)
				return
			}
			ctx, cancel := context.WithTimeout(r.Context(), g.wait)
			select {
			case g.slots <- struct{}{}:
				cancel()
				g.waiting.Add(-1)
			case <-ctx.Done():
				cancel()
				g.waiting.Add(-1)
				if r.Context().Err() == nil || errors.Is(r.Context().Err(), context.DeadlineExceeded) {
					g.refuse(w)
				}
				return // cancelled: nobody is listening
			}
		}
		defer func() { <-g.slots }()
		next(w, r)
	}
}

func (g *heavyGate) refuse(w http.ResponseWriter) {
	w.Header().Set("Retry-After", heavyRetryAfter)
	writeErrorCode(w, CodeUnavailable, "too many library-wide reads in flight; retry shortly")
}
