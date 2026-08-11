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
