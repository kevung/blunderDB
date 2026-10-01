package middleware

import (
	"net/http"
	"time"

	"github.com/kevung/blunderdb/internal/server/metrics"
)

// Metrics records request count and latency into the registry, labelling by a
// bounded route label (known path or "unmatched") so probing arbitrary 404
// paths cannot inflate label cardinality.
// A route untimed names is counted but kept out of the latency histogram: an event stream lasts
// as long as its client listens, and its duration says nothing of the daemon's speed.
func Metrics(reg *metrics.Registry, known, untimed map[string]bool, now func() time.Time) func(http.Handler) http.Handler {
	if now == nil {
		now = time.Now
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := now()
			rec := newResponseRecorder(w)
			next.ServeHTTP(rec, r)
			if untimed[r.URL.Path] {
				reg.CountRequest(r.Method, routeLabel(r, known), rec.status)
				return
			}
			reg.ObserveRequest(r.Method, routeLabel(r, known), rec.status, now().Sub(start))
		})
	}
}
