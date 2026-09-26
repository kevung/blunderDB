package middleware

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestLogging_SurfacesStashedErrorAtErrorLevel: an error stashed via SetErr
// before masking appears in the log line, at Error level.
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

// TestLogging_IncludesRequestIDAndTraceparent: the request id and traceparent
// appear in the same completion log line.
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

// TestLogging_NoRequestIDMiddlewareOmitsFields: without RequestID ahead,
// Logging fabricates no request_id/traceparent field.
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
