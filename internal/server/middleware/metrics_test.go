package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/kevung/blunderdb/internal/server/metrics"
)

// pathLabels returns the distinct path="…" label values in an exposition.
func pathLabels(text string) map[string]bool {
	re := regexp.MustCompile(`path="([^"]*)"`)
	out := map[string]bool{}
	for _, m := range re.FindAllStringSubmatch(text, -1) {
		out[m[1]] = true
	}
	return out
}

// TestMetrics_UnknownPathsShareOneLabel is the cardinality guard: a scanner
// probing thousands of URLs must not create a series per URL. Every path
// outside the known set folds into the single "unmatched" label.
func TestMetrics_UnknownPathsShareOneLabel(t *testing.T) {
	reg := metrics.New()
	known := map[string]bool{"/healthz": true, "/v1/positions.list": true}
	handler := Metrics(reg, known, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if known[r.URL.Path] {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))

	for i := 0; i < 1000; i++ {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/probe/%d/../../etc/passwd%d", i, i), nil)
		handler.ServeHTTP(httptest.NewRecorder(), req)
	}
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/healthz", nil))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/v1/positions.list", nil))
	// A near miss on a known route (trailing slash) is still unknown — the
	// label is the exact route pattern — while a query string is not part of
	// the path and must neither create a label nor demote the route.
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/healthz/", nil))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/v1/positions.list?x=1", nil))

	var out strings.Builder
	reg.WritePrometheus(&out)
	text := out.String()

	labels := pathLabels(text)
	want := map[string]bool{"/healthz": true, "/v1/positions.list": true, "unmatched": true}
	if fmt.Sprint(labels) != fmt.Sprint(want) {
		t.Fatalf("path labels = %v, want exactly %v\n%s", labels, want, text)
	}
	if !strings.Contains(text, `blunderdb_http_requests_total{method="GET",path="unmatched",status="404"} 1001`) {
		t.Errorf("unmatched counter should hold the 1000 probes plus /healthz/:\n%s", text)
	}
	if !strings.Contains(text, `blunderdb_http_requests_total{method="GET",path="/v1/positions.list",status="200"} 1`) {
		t.Errorf("a query string must not change the route label:\n%s", text)
	}
}

// TestMetrics_RecordsStatusAndDuration: the label carries the status the
// handler actually wrote, and the duration is measured with the injected
// clock (so the histogram is testable without sleeping).
func TestMetrics_RecordsStatusAndDuration(t *testing.T) {
	reg := metrics.New()
	clock := time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)
	now := func() time.Time {
		clock = clock.Add(40 * time.Millisecond) // each call advances: start, then end
		return clock
	}
	handler := Metrics(reg, map[string]bool{"/v1/x": true}, now)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/v1/x", nil))

	var out strings.Builder
	reg.WritePrometheus(&out)
	text := out.String()
	for _, line := range []string{
		`blunderdb_http_requests_total{method="POST",path="/v1/x",status="418"} 1`,
		`blunderdb_http_request_duration_seconds_bucket{method="POST",path="/v1/x",le="0.025"} 0`,
		`blunderdb_http_request_duration_seconds_bucket{method="POST",path="/v1/x",le="0.05"} 1`,
		`blunderdb_http_request_duration_seconds_sum{method="POST",path="/v1/x"} 0.04`,
	} {
		if !strings.Contains(text, line) {
			t.Errorf("missing %q in:\n%s", line, text)
		}
	}
}

// TestMetrics_DefaultStatusIsOK: a handler that writes a body without calling
// WriteHeader is a 200, and the recorder must report it as such.
func TestMetrics_DefaultStatusIsOK(t *testing.T) {
	reg := metrics.New()
	handler := Metrics(reg, map[string]bool{"/healthz": true}, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/healthz", nil))

	var out strings.Builder
	reg.WritePrometheus(&out)
	if !strings.Contains(out.String(), `blunderdb_http_requests_total{method="GET",path="/healthz",status="200"} 1`) {
		t.Errorf("implicit 200 not recorded:\n%s", out.String())
	}
}
