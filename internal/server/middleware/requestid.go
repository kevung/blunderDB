package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

// RequestIDHeader is the request/response header carrying the correlation id
// for one HTTP call: a client (or an upstream proxy) may set it, and the
// value that ends up in play — theirs, echoed back, or one this daemon
// minted — is always present on the response and in the request-completion
// log line, so a support request ("my import at 14:32 failed") can be tied
// to one exact log line without guessing on timestamps (#238).
const RequestIDHeader = "X-Request-Id"

// TraceparentHeader is the W3C Trace Context header. This daemon does not
// parse or validate it — no tracing library is wired in, deliberately, to
// keep the dependency surface the metrics package already avoids
// (prometheus/client_golang) from growing a tracing SDK too — it only relays
// the raw value into the request-completion log line so a caller that DOES
// run a tracing pipeline upstream (gammonGo, a proxy) can still correlate
// this daemon's logs with a trace by grep, without blunderDB understanding
// the format at all.
const TraceparentHeader = "traceparent"

// maxRequestIDLen bounds how much of a client-supplied X-Request-Id reaches
// the response header and the logs: an adversarial or buggy client must not
// be able to blow up a log line or smuggle control characters into a header
// (mirrors middleware.Logging's maxLoggedTenantLen for the same reason).
const maxRequestIDLen = 128

type requestIDKey struct{}
type traceparentKey struct{}

// RequestID assigns a correlation id to every request: the client-supplied
// X-Request-Id when present (sanitized and bounded), otherwise a fresh
// 128-bit random id (hex). Either way the id is echoed back as the response
// header of the same name and stored in the request context so downstream
// middleware (Logging, Recover) and handlers can read it via
// RequestIDFromContext. traceparent, when present, is relayed the same way
// via TraceparentFromContext — see TraceparentHeader's doc comment.
//
// This is the outermost middleware: it must wrap even Recover, so a panic's
// log line and the 500 response it produces both still carry the id a
// client-visible failure needs to be correlated by.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := sanitizeHeaderValue(r.Header.Get(RequestIDHeader))
		if id == "" {
			id = newRequestID()
		}
		w.Header().Set(RequestIDHeader, id)
		ctx := context.WithValue(r.Context(), requestIDKey{}, id)

		if tp := sanitizeHeaderValue(r.Header.Get(TraceparentHeader)); tp != "" {
			ctx = context.WithValue(ctx, traceparentKey{}, tp)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// newRequestID returns a fresh opaque id: 128 bits from crypto/rand, hex —
// the same shape as internal/server's import ids (newImportID), chosen so a
// request id is visually distinguishable from neither a tenant nor a
// database row id.
func newRequestID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// sanitizeHeaderValue trims a client-supplied header value to
// maxRequestIDLen runes and strips ASCII control characters (in particular
// CR/LF, which would otherwise corrupt the response header or a log line);
// an empty result (missing header, or control characters only) means
// "generate one instead".
func sanitizeHeaderValue(v string) string {
	r := []rune(v)
	out := make([]rune, 0, len(r))
	for _, c := range r {
		if c < 0x20 || c == 0x7f {
			continue
		}
		out = append(out, c)
		if len(out) >= maxRequestIDLen {
			break
		}
	}
	return string(out)
}

// RequestIDFromContext returns the request id RequestID stored in ctx, and
// whether one was present (false only when a handler runs outside the
// RequestID middleware, e.g. a unit test that calls it directly).
func RequestIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(requestIDKey{}).(string)
	return v, ok
}

// TraceparentFromContext returns the raw traceparent header value RequestID
// stashed in ctx, if the request carried one.
func TraceparentFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(traceparentKey{}).(string)
	return v, ok
}
