package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// Logging emits one structured log line per request once it completes. The
// route is a bounded label (known path or "unmatched"); the tenant is read
// from the request header (this middleware sits outside Tenant so it also logs
// tenant-rejected requests).
func Logging(logger *slog.Logger, known map[string]bool, now func() time.Time) func(http.Handler) http.Handler {
	if now == nil {
		now = time.Now
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := now()
			rec := newResponseRecorder(w)
			next.ServeHTTP(rec, r)
			if logger == nil {
				return
			}
			logger.Info("http request",
				"method", r.Method,
				"route", routeLabel(r, known),
				"path", r.URL.Path,
				"status", rec.status,
				"bytes", rec.bytes,
				"tenant", r.Header.Get(TenantHeader),
				"duration_ms", float64(now().Sub(start).Microseconds())/1000.0,
			)
		})
	}
}
