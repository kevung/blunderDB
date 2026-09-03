package middleware

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestLogging_SurfacesStashedErrorAtErrorLevel guards the fix for "500
// without a server-side trace": a handler that masks an internal error
// behind a generic body (the client must never see backend internals) used
// to leave no trace of what actually failed anywhere in the server logs. A
// handler now reaches its ResponseWriter's SetErr (responseRecorder
// implements it) before masking, and Logging must both include that error in
// the log line and log at Error level (not Info) so a 500 is not lost in
// request-volume noise.
func TestLogging_SurfacesStashedErrorAtErrorLevel(t *testing.T) {
	var buf strings.Builder
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if es, ok := w.(interface{ SetErr(error) }); ok {
			es.SetErr(errors.New("boom: disk full"))
		}
		w.WriteHeader(http.StatusInternalServerError)
	})

	mw := Logging(logger, map[string]bool{}, nil)(handler)
	req := httptest.NewRequest(http.MethodGet, "/v1/positions.list", nil)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	out := buf.String()
	if !strings.Contains(out, "level=ERROR") {
		t.Errorf("log line not at Error level:\n%s", out)
	}
	if !strings.Contains(out, "boom: disk full") {
		t.Errorf("log line missing the stashed error:\n%s", out)
	}
	if !strings.Contains(out, "status=500") {
		t.Errorf("log line missing the 500 status:\n%s", out)
	}
}

// TestLogging_NoStashedErrorStaysInfo covers the ordinary (non-error) path:
// no SetErr call, no "err" field, Info level as before.
func TestLogging_NoStashedErrorStaysInfo(t *testing.T) {
	var buf strings.Builder
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := Logging(logger, map[string]bool{}, nil)(handler)
	req := httptest.NewRequest(http.MethodGet, "/v1/positions.list", nil)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	out := buf.String()
	if !strings.Contains(out, "level=INFO") {
		t.Errorf("log line not at Info level:\n%s", out)
	}
	if strings.Contains(out, "err=") {
		t.Errorf("log line has an unexpected err field:\n%s", out)
	}
}

// TestLogging_IncludesRequestIDAndTraceparent guards #238: a request that
// went through RequestID must have its correlation id and (when present) its
// traceparent surface in the same request-completion log line Logging
// already emits, rather than being a separate, harder-to-correlate log
// stream.
func TestLogging_IncludesRequestIDAndTraceparent(t *testing.T) {
	var buf strings.Builder
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := RequestID(Logging(logger, map[string]bool{}, nil)(handler))
	req := httptest.NewRequest(http.MethodGet, "/v1/positions.list", nil)
	req.Header.Set(RequestIDHeader, "abc-123")
	req.Header.Set(TraceparentHeader, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	out := buf.String()
	if !strings.Contains(out, "request_id=abc-123") {
		t.Errorf("log line missing request_id:\n%s", out)
	}
	if !strings.Contains(out, "traceparent=00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01") {
		t.Errorf("log line missing traceparent:\n%s", out)
	}
}

// TestLogging_NoRequestIDMiddlewareOmitsFields: Logging must not fabricate a
// request_id/traceparent field when it runs without RequestID ahead of it
// (e.g. a unit test that exercises Logging alone, as every other test in
// this file does).
func TestLogging_NoRequestIDMiddlewareOmitsFields(t *testing.T) {
	var buf strings.Builder
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := Logging(logger, map[string]bool{}, nil)(handler)
	req := httptest.NewRequest(http.MethodGet, "/v1/positions.list", nil)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	out := buf.String()
	if strings.Contains(out, "request_id=") {
		t.Errorf("log line has an unexpected request_id field with no RequestID middleware:\n%s", out)
	}
	if strings.Contains(out, "traceparent=") {
		t.Errorf("log line has an unexpected traceparent field with no RequestID middleware:\n%s", out)
	}
}
