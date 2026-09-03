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
			// A masked "internal error" response is otherwise a dead end for
			// diagnosing what actually failed: the client only ever sees the
			// generic message (backend internals must not leak), so the real
			// cause — stashed via SetErr — has to surface somewhere. Error
			// level (rather than Info) so it is not lost in request-volume
			// noise.
			if rec.err != nil {
				args = append(args, "err", rec.err.Error())
				logger.Error("http request", args...)
				return
			}
			logger.Info("http request", args...)
		})
	}
}

// maxLoggedTenantLen bounds how much of the raw X-Tenant-ID header value
// reaches the log line. This middleware logs it before Tenant validates the
// header (deliberately — a rejected request must still be traceable), so an
// oversized or adversarial header value must not blow up a log line; a valid
// tenant is a short decimal integer (A.1) and is never truncated by this.
const maxLoggedTenantLen = 64

// truncateForLog bounds v to maxLoggedTenantLen runes, marking a cut with a
// trailing ellipsis so a truncated value is never mistaken for a complete
// one (#232).
func truncateForLog(v string) string {
	r := []rune(v)
	if len(r) <= maxLoggedTenantLen {
		return v
	}
	return string(r[:maxLoggedTenantLen]) + "…"
}
