package server

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"testing"
)

// TestMetrics_PathLabelsAreBounded checks the cardinality guard end to end,
// through the real route table: probing the daemon with arbitrary URLs must
// leave /metrics with the declared routes and a single "unmatched" label —
// never a series per probed URL (which is how a scanner would exhaust a
// Prometheus server's memory).
func TestMetrics_PathLabelsAreBounded(t *testing.T) {
	ts := newTestServer(t)
	get := func(path string) {
		t.Helper()
		resp, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
	}
	for i := 0; i < 200; i++ {
		get(fmt.Sprintf("/wp-admin/%d", i))
		get(fmt.Sprintf("/v1/positions.list/%d", i)) // a known prefix, an unknown path
	}
	get("/healthz")

	resp, err := http.Get(ts.URL + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)

	labels := map[string]bool{}
	for _, m := range regexp.MustCompile(`path="([^"]*)"`).FindAllStringSubmatch(string(out), -1) {
		labels[m[1]] = true
	}
	want := map[string]bool{"/healthz": true, "unmatched": true}
	if fmt.Sprint(labels) != fmt.Sprint(want) {
		t.Fatalf("path labels = %v, want exactly %v\n%s", labels, want, out)
	}
	// The 400 probes land in one series per status they got (the tenant
	// middleware refuses them before routing, so it is 400 today; a 404 after
	// a routing change would be as fine) — never one per URL.
	var probes, series int
	for _, m := range regexp.MustCompile(`blunderdb_http_requests_total\{method="GET",path="unmatched",status="\d+"\} (\d+)\n`).FindAllStringSubmatch(string(out), -1) {
		n, _ := strconv.Atoi(m[1])
		probes += n
		series++
	}
	if probes != 400 || series > 2 {
		t.Errorf("unmatched probes = %d in %d series, want 400 in at most 2:\n%s", probes, series, out)
	}
}
