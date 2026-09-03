package middleware

import (
	"bufio"
	"net"
	"net/http"
)

// responseRecorder wraps an http.ResponseWriter to capture the status code and
// the number of bytes written, while transparently delegating optional
// interfaces (Flusher for NDJSON streaming, Hijacker) to the wrapped writer.
type responseRecorder struct {
	http.ResponseWriter
	status      int
	bytes       int64
	wroteHeader bool
	err         error
}

// SetErr attaches the original error behind a response — a handler masking a
// storage error as the generic "internal error" body a client sees (so
// backend internals never leak) calls this first, so Logging can still put
// the real cause in the server-side log line. A handler type-asserts its
// http.ResponseWriter to `interface{ SetErr(error) }` to reach it without a
// cross-package dependency on this concrete type.
func (r *responseRecorder) SetErr(err error) { r.err = err }

func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{ResponseWriter: w, status: http.StatusOK}
}

func (r *responseRecorder) WriteHeader(code int) {
	if r.wroteHeader {
		return
	}
	r.status = code
	r.wroteHeader = true
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += int64(n)
	return n, err
}

// Flush implements http.Flusher so NDJSON streaming handlers can flush through
// the recorder. It is a no-op if the wrapped writer is not a Flusher.
func (r *responseRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack implements http.Hijacker by delegating to the wrapped writer.
func (r *responseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := r.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

// Unwrap exposes the wrapped writer to http.ResponseController (the
// documented mechanism a wrapping ResponseWriter uses to let
// SetReadDeadline/SetWriteDeadline reach the real connection through it).
// Both Metrics and Logging wrap with a responseRecorder, so a handler's `w`
// is two of these deep; without Unwrap the controller would stop at the
// first one — which does not itself implement SetWriteDeadline — and every
// call would fail with http.ErrNotSupported (#234).
func (r *responseRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }
