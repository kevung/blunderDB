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

// TestLogging_ReadTenantsOnAcrossRoutes: a read across tenants logs whom it
// read, bounded; another route does not log the header.
func TestLogging_ReadTenantsOnAcrossRoutes(t *testing.T) {
	logLine := func(path, readTenants string) string {
		var buf strings.Builder
		mw := Logging(slog.New(slog.NewTextHandler(&buf, nil)), map[string]bool{}, nil)(
			http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
		req := httptest.NewRequest(http.MethodPost, path, nil)
		req.Header.Set(ReadTenantsHeader, readTenants)
		mw.ServeHTTP(httptest.NewRecorder(), req)
		return buf.String()
	}
	if out := logLine("/v1/across.matchesList", "2,3"); !strings.Contains(out, "read_tenants=2,3") {
		t.Errorf("across route does not log its read set:\n%s", out)
	}
	long := strings.Repeat("12345,", 40)
	if out := logLine("/v1/across.matchesList", long); strings.Contains(out, long) || !strings.Contains(out, "…") {
		t.Errorf("a long read set is not truncated:\n%s", out)
	}
	if out := logLine("/v1/matches.list", "2,3"); strings.Contains(out, "read_tenants") {
		t.Errorf("a route outside across.* logs the header:\n%s", out)
	}
}

// TestCORS_DoesNotAllowReadTenants: a browser never sends X-Read-Tenants —
// only the proxy writes it — so a preflight does not allow it.
func TestCORS_DoesNotAllowReadTenants(t *testing.T) {
	mw := CORS("https://app.example")(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	req := httptest.NewRequest(http.MethodOptions, "/v1/across.matchesList", nil)
	req.Header.Set("Origin", "https://app.example")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	allowed := rec.Header().Get("Access-Control-Allow-Headers")
	if allowed == "" {
		t.Fatal("no Access-Control-Allow-Headers on the preflight")
	}
	if strings.Contains(allowed, ReadTenantsHeader) {
		t.Errorf("CORS allows %s: %s", ReadTenantsHeader, allowed)
	}
}
