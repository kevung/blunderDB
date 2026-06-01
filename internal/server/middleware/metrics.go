package middleware

import (
	"net/http"
	"time"

	"github.com/kevung/blunderdb/internal/server/metrics"
)

// Metrics records request count and latency into the registry, labelling by a
// bounded route label (known path or "unmatched") so probing arbitrary 404
// paths cannot inflate label cardinality.
func Metrics(reg *metrics.Registry, known map[string]bool, now func() time.Time) func(http.Handler) http.Handler {
	if now == nil {
		now = time.Now
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := now()
			rec := newResponseRecorder(w)
			next.ServeHTTP(rec, r)
			reg.ObserveRequest(r.Method, routeLabel(r, known), rec.status, now().Sub(start))
		})
	}
}
