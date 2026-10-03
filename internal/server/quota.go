package server

import (
	"context"
	"net/http"
	"sync"
	"time"
)

// TenantQuotas bound what one tenant may take from an instance it shares with
// others. A zero field is unlimited, the default: an instance serving one
// person has nothing to share.
//
// The quotas are the daemon's own accounting, not a security boundary: the
// tenant they apply to is the X-Tenant-ID the proxy set (ADR-0005).
type TenantQuotas struct {
	// MaxPositions: an import is refused (413) once the tenant stores this
	// many positions. One already running is not cut short, so a tenant may
	// end above the bound by one import's worth.
	MaxPositions int64 `json:"maxPositions"`

	// AnalysisSecondsPerDay is the engine time a tenant may spend per UTC day,
	// summed over every evaluation it asks for (sweeps, comparisons, a cube
	// matrix, a bare evaluation). Past it, a request is refused (429) and a
	// running sweep stops at the next position.
	AnalysisSecondsPerDay int64 `json:"analysisSecondsPerDay"`

	// MaxConcurrentImports: imports of one tenant running at once; one more is
	// refused (429).
	MaxConcurrentImports int `json:"maxConcurrentImports"`
}

// quotaLedger holds each tenant's use against TenantQuotas. Analysis time is
// kept for the current UTC day only: the first charge of a new day drops the
// previous one.
type quotaLedger struct {
	limits TenantQuotas
	now    func() time.Time

	mu      sync.Mutex
	day     string
	spent   map[string]time.Duration
	imports map[string]int
}

func newQuotaLedger(limits TenantQuotas, now func() time.Time) *quotaLedger {
	if now == nil {
		now = time.Now
	}
	return &quotaLedger{limits: limits, now: now, spent: map[string]time.Duration{}, imports: map[string]int{}}
}

// rollLocked starts a new day's account when the UTC date changed.
func (q *quotaLedger) rollLocked() {
	if d := q.now().UTC().Format(time.DateOnly); d != q.day {
		q.day, q.spent = d, map[string]time.Duration{}
	}
}

// analysisExhausted reports whether scope has spent its engine time today.
func (q *quotaLedger) analysisExhausted(scope string) bool {
	if q.limits.AnalysisSecondsPerDay <= 0 {
		return false
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	q.rollLocked()
	return q.spent[scope] >= time.Duration(q.limits.AnalysisSecondsPerDay)*time.Second
}

// charge adds engine time to scope's account for today.
func (q *quotaLedger) charge(scope string, d time.Duration) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.rollLocked()
	q.spent[scope] += d
}

// spender charges engine time to scope and reports whether some is left: a
// worker stops picking positions on false.
func (q *quotaLedger) spender(scope string) func(time.Duration) bool {
	return func(d time.Duration) bool {
		q.charge(scope, d)
		return !q.analysisExhausted(scope)
	}
}

// beginImport claims one of scope's concurrent import slots; endImport gives
// it back. A refused claim takes nothing.
func (q *quotaLedger) beginImport(scope string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	if n := q.limits.MaxConcurrentImports; n > 0 && q.imports[scope] >= n {
		return false
	}
	q.imports[scope]++
	return true
}

func (q *quotaLedger) endImport(scope string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.imports[scope] <= 1 {
		delete(q.imports, scope)
		return
	}
	q.imports[scope]--
}

// usage is scope's engine time today and its imports in flight.
func (q *quotaLedger) usage(scope string) (time.Duration, int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.rollLocked()
	return q.spent[scope], q.imports[scope]
}

// refuseAnalysis writes the 429 of a tenant whose engine time is spent and
// reports whether it did.
func (s *Server) refuseAnalysis(w http.ResponseWriter, scope string) bool {
	if !s.quota.analysisExhausted(scope) {
		return false
	}
	spent, _ := s.quota.usage(scope)
	writeErrorDetails(w, CodeQuotaExceeded, "analysis time for today is spent", map[string]any{
		"quota": "analysisSecondsPerDay", "limit": s.quota.limits.AnalysisSecondsPerDay, "used": int64(spent / time.Second),
	})
	return true
}

// metered runs one engine computation and charges its time to scope.
func metered[T any](s *Server, scope string, fn func() (T, error)) (T, error) {
	start := time.Now()
	defer func() { s.quota.charge(scope, time.Since(start)) }()
	return fn()
}

// refuseImport claims an import slot for scope, or writes the refusal of a
// quota and reports false. On true the caller releases the slot with
// s.quota.endImport.
func (s *Server) refuseImport(ctx context.Context, w http.ResponseWriter, scope string) bool {
	if max := s.quota.limits.MaxPositions; max > 0 {
		counts, err := s.opts.Storage.Metadata().Counts(ctx, scope)
		if err != nil {
			writeStorageError(w, err)
			return true
		}
		if int64(counts.Positions) >= max {
			writeErrorDetails(w, CodeStorageQuotaExceeded, "this tenant stores as many positions as it may", map[string]any{
				"quota": "maxPositions", "limit": max, "used": counts.Positions,
			})
			return true
		}
	}
	if !s.quota.beginImport(scope) {
		_, n := s.quota.usage(scope)
		writeErrorDetails(w, CodeQuotaExceeded, "too many imports of this tenant at once", map[string]any{
			"quota": "maxConcurrentImports", "limit": s.quota.limits.MaxConcurrentImports, "used": n,
		})
		return true
	}
	return false
}
