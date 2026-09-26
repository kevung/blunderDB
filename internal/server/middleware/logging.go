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
			args := []any{
				"method", r.Method,
				"route", routeLabel(r, known),
				"path", r.URL.Path,
				"status", rec.status,
				"bytes", rec.bytes,
				"tenant", truncateForLog(r.Header.Get(TenantHeader)),
				"duration_ms", float64(now().Sub(start).Microseconds()) / 1000.0,
			}
			if id, ok := RequestIDFromContext(r.Context()); ok {
				args = append(args, "request_id", id)
			}
			// Relayed verbatim, unparsed, so the request can be grepped.
			if tp, ok := TraceparentFromContext(r.Context()); ok {
				args = append(args, "traceparent", tp)
			}
			// A masked "internal error" hides its cause from the client; the
			// SetErr cause surfaces here, at Error level to stand out.
			if rec.err != nil {
				args = append(args, "err", rec.err.Error())
				logger.Error("http request", args...)
				return
			}
			logger.Info("http request", args...)
		})
	}
}

// maxLoggedTenantLen bounds the raw X-Tenant-ID logged before Tenant
// validates it; a valid tenant is a short integer and is never cut.
const maxLoggedTenantLen = 64

// truncateForLog bounds v to maxLoggedTenantLen runes, marking a cut with a
// trailing ellipsis so it is never mistaken for a complete value.
func truncateForLog(v string) string {
	r := []rune(v)
	if len(r) <= maxLoggedTenantLen {
		return v
	}
	return string(r[:maxLoggedTenantLen]) + "…"
}
