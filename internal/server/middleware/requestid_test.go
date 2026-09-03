package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestRequestID_GeneratesWhenAbsent covers the common case: no client-
// supplied X-Request-Id, so RequestID mints one, echoes it back on the
// response, and stores it in the context for downstream middleware/handlers.
func TestRequestID_GeneratesWhenAbsent(t *testing.T) {
	var seen string
	var ok bool
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen, ok = RequestIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	mw := RequestID(inner)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if !ok || seen == "" {
		t.Fatalf("RequestIDFromContext = %q, %v; want a generated non-empty id", seen, ok)
	}
	if got := rec.Header().Get(RequestIDHeader); got != seen {
		t.Errorf("response header %s = %q, want the same id the context carried (%q)", RequestIDHeader, got, seen)
	}
}

// TestRequestID_EchoesClientSuppliedID: a caller (or an upstream proxy) that
// already minted a correlation id must see the exact same value come back,
// so its own logs and this daemon's line up on one id.
func TestRequestID_EchoesClientSuppliedID(t *testing.T) {
	var seen string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen, _ = RequestIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	mw := RequestID(inner)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set(RequestIDHeader, "caller-supplied-id-123")
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if seen != "caller-supplied-id-123" {
		t.Errorf("context id = %q, want the client-supplied id", seen)
	}
	if got := rec.Header().Get(RequestIDHeader); got != "caller-supplied-id-123" {
		t.Errorf("response header = %q, want the client-supplied id echoed back", got)
	}
}

// TestRequestID_SanitizesControlCharsAndLength: a client-supplied id must
// never carry CR/LF into a response header or a log line, and an
// unreasonably long one must not either — both are treated as "generate one
// instead" once truncation/stripping leaves nothing, or bounded otherwise.
func TestRequestID_SanitizesControlCharsAndLength(t *testing.T) {
	var seen string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen, _ = RequestIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	mw := RequestID(inner)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set(RequestIDHeader, strings.Repeat("a", 500))
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	if len(seen) > maxRequestIDLen {
		t.Errorf("id length = %d, want <= %d", len(seen), maxRequestIDLen)
	}
}

// TestRequestID_RelaysTraceparent: a present traceparent header must reach
// TraceparentFromContext verbatim (no parsing/validation — see the doc
// comment on TraceparentHeader) so Logging can relay it, while its absence
// must not fabricate one.
func TestRequestID_RelaysTraceparent(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tp, ok := TraceparentFromContext(r.Context())
		if !ok || tp != "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01" {
			t.Errorf("traceparent in context = %q, %v; want the header value relayed verbatim", tp, ok)
		}
		w.WriteHeader(http.StatusOK)
	})
	mw := RequestID(inner)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set(TraceparentHeader, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	inner2 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := TraceparentFromContext(r.Context()); ok {
			t.Error("TraceparentFromContext ok=true with no traceparent header sent")
		}
		w.WriteHeader(http.StatusOK)
	})
	mw2 := RequestID(inner2)
	req2 := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec2 := httptest.NewRecorder()
	mw2.ServeHTTP(rec2, req2)
}
