package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

// latencyStats holds the percentile summary of a set of request latencies, in
// milliseconds.
type latencyStats struct {
	P50 float64 `json:"p50"`
	P95 float64 `json:"p95"`
	P99 float64 `json:"p99"`
	Max float64 `json:"max"`
}

// endpointStat is the per-endpoint breakdown.
type endpointStat struct {
	Count   int          `json:"count"`
	Errors  int          `json:"errors"`
	Latency latencyStats `json:"latency_ms"`
}

// report is the JSON document written at the end of a run.
type report struct {
	Target        string                  `json:"target"`
	Scenario      string                  `json:"scenario"`
	Tenants       int                     `json:"tenants"`
	RPSTarget     int                     `json:"rps_target"` // 0 = unbounded
	Concurrency   int                     `json:"concurrency"`
	DurationS     float64                 `json:"duration_s"`
	TotalRequests int                     `json:"total_requests"`
	RPSAchieved   float64                 `json:"rps_achieved"`
	Errors        int                     `json:"errors"`
	Latency       latencyStats            `json:"latency_ms"`
	ByEndpoint    map[string]endpointStat `json:"by_endpoint"`
}

// sample is one recorded request outcome.
type sample struct {
	op   string
	dur  time.Duration
	fail bool
}

// percentile returns the p-th percentile (0..100) of a sorted millisecond
// slice using nearest-rank.
func percentile(sortedMs []float64, p float64) float64 {
	if len(sortedMs) == 0 {
		return 0
	}
	rank := int(p / 100 * float64(len(sortedMs)))
	if rank >= len(sortedMs) {
		rank = len(sortedMs) - 1
	}
	return sortedMs[rank]
}

func statsOf(durs []time.Duration) latencyStats {
	ms := make([]float64, len(durs))
	for i, d := range durs {
		ms[i] = float64(d.Microseconds()) / 1000
	}
	sort.Float64s(ms)
	st := latencyStats{P50: percentile(ms, 50), P95: percentile(ms, 95), P99: percentile(ms, 99)}
	if len(ms) > 0 {
		st.Max = ms[len(ms)-1]
	}
	return st
}

// buildReport aggregates the samples into a report.
func buildReport(cfg config, elapsed time.Duration, samples []sample) report {
	all := make([]time.Duration, 0, len(samples))
	byOpDur := map[string][]time.Duration{}
	byOpErr := map[string]int{}
	errors := 0
	for _, s := range samples {
		all = append(all, s.dur)
		byOpDur[s.op] = append(byOpDur[s.op], s.dur)
		if s.fail {
			errors++
			byOpErr[s.op]++
		}
	}

	byEndpoint := map[string]endpointStat{}
	for op, durs := range byOpDur {
		byEndpoint[op] = endpointStat{
			Count:   len(durs),
			Errors:  byOpErr[op],
			Latency: statsOf(durs),
		}
	}

	secs := elapsed.Seconds()
	rps := 0.0
	if secs > 0 {
		rps = float64(len(samples)) / secs
	}
	return report{
		Target:        cfg.target,
		Scenario:      cfg.scenario,
		Tenants:       cfg.tenants,
		RPSTarget:     cfg.rps,
		Concurrency:   cfg.concurrency,
		DurationS:     secs,
		TotalRequests: len(samples),
		RPSAchieved:   rps,
		Errors:        errors,
		Latency:       statsOf(all),
		ByEndpoint:    byEndpoint,
	}
}

// writeReports writes the JSON report to jsonPath and a human-readable Markdown
// companion alongside it (same name with a .md extension).
func writeReports(rep report, jsonPath string) error {
	data, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(jsonPath, append(data, '\n'), 0o644); err != nil {
		return err
	}
	mdPath := strings.TrimSuffix(jsonPath, ".json") + ".md"
	return os.WriteFile(mdPath, []byte(rep.markdown()), 0o644)
}

func (r report) markdown() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# blunderdb-loadtest report\n\n")
	fmt.Fprintf(&b, "- Target: `%s`\n", r.Target)
	fmt.Fprintf(&b, "- Scenario: **%s**\n", r.Scenario)
	fmt.Fprintf(&b, "- Tenants: %d, Concurrency: %d, RPS target: %s\n",
		r.Tenants, r.Concurrency, rpsLabel(r.RPSTarget))
	fmt.Fprintf(&b, "- Duration: %.1f s\n\n", r.DurationS)
	fmt.Fprintf(&b, "| Metric | Value |\n|---|---|\n")
	fmt.Fprintf(&b, "| Total requests | %d |\n", r.TotalRequests)
	fmt.Fprintf(&b, "| RPS achieved | %.1f |\n", r.RPSAchieved)
	fmt.Fprintf(&b, "| Errors | %d |\n", r.Errors)
	fmt.Fprintf(&b, "| p50 | %.1f ms |\n", r.Latency.P50)
	fmt.Fprintf(&b, "| p95 | %.1f ms |\n", r.Latency.P95)
	fmt.Fprintf(&b, "| p99 | %.1f ms |\n", r.Latency.P99)
	fmt.Fprintf(&b, "| max | %.1f ms |\n\n", r.Latency.Max)

	fmt.Fprintf(&b, "## By endpoint\n\n")
	fmt.Fprintf(&b, "| Endpoint | Count | Errors | p50 | p95 | p99 |\n|---|---|---|---|---|---|\n")
	for _, op := range sortedKeys(r.ByEndpoint) {
		e := r.ByEndpoint[op]
		fmt.Fprintf(&b, "| %s | %d | %d | %.1f | %.1f | %.1f |\n",
			op, e.Count, e.Errors, e.Latency.P50, e.Latency.P95, e.Latency.P99)
	}
	return b.String()
}

func rpsLabel(rps int) string {
	if rps <= 0 {
		return "unbounded"
	}
	return fmt.Sprintf("%d", rps)
}

func sortedKeys(m map[string]endpointStat) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
