// Command blunderdb-loadtest is a self-contained HTTP load generator for a
// running `blunderdb serve` daemon: it drives /v1/* with a scenario mix
// across many tenants and writes a JSON + Markdown latency report.
//
//	blunderdb-loadtest --target http://localhost:8080 \
//	  --tenants 10000 --rps 10000 --duration 60s --scenario mixed \
//	  --output report.json
//
// --rps 0 means unbounded (closed-loop: as fast as --concurrency allows).
// Tenants are sent as integers 1..N (see scenario.go for named-tenants).
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type config struct {
	target      string
	tenants     int
	rps         int
	duration    time.Duration
	rampUp      time.Duration
	scenario    string
	concurrency int
	output      string
	seed        int64
	maxMatches  int
}

func main() {
	cfg := config{}
	flag.StringVar(&cfg.target, "target", "http://localhost:8080", "base URL of the blunderdb serve daemon")
	flag.IntVar(&cfg.tenants, "tenants", 100, "number of distinct tenants (X-Tenant-ID values 1..N)")
	flag.IntVar(&cfg.rps, "rps", 0, "target requests/sec (0 = unbounded, closed-loop)")
	flag.DurationVar(&cfg.duration, "duration", 10*time.Second, "measurement duration")
	flag.DurationVar(&cfg.rampUp, "ramp-up", 2*time.Second, "stagger worker start over this window")
	flag.StringVar(&cfg.scenario, "scenario", "mixed", "scenario: "+scenarioNames)
	flag.IntVar(&cfg.concurrency, "concurrency", 50, "number of concurrent worker goroutines")
	flag.StringVar(&cfg.output, "output", "report.json", "path for the JSON report (a .md companion is written too)")
	flag.Int64Var(&cfg.seed, "seed", 1, "PRNG seed for reproducible runs")
	flag.IntVar(&cfg.maxMatches, "max-matches", 30000, "library scenario: matches.list pages are drawn at offsets below this")
	flag.Parse()

	sc, ok := scenarios[cfg.scenario]
	if cfg.scenario == "library" {
		var err error
		if sc, err = librarySetup(cfg.target, numericTenant(1), cfg.maxMatches); err != nil {
			fmt.Fprintf(os.Stderr, "loadtest: %v\n", err)
			os.Exit(1)
		}
		ok = true
	}
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown scenario %q (want %s)\n", cfg.scenario, scenarioNames)
		os.Exit(2)
	}
	if cfg.tenants < 1 {
		cfg.tenants = 1
	}

	rep, err := run(cfg, sc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "loadtest: %v\n", err)
		os.Exit(1)
	}
	if err := writeReports(rep, cfg.output); err != nil {
		fmt.Fprintf(os.Stderr, "write report: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("done: %d requests in %.1fs → %.0f rps, %d errors, p50=%.1fms p95=%.1fms p99=%.1fms\n",
		rep.TotalRequests, rep.DurationS, rep.RPSAchieved, rep.Errors,
		rep.Latency.P50, rep.Latency.P95, rep.Latency.P99)
	fmt.Printf("reports: %s + %s\n", cfg.output, mdName(cfg.output))
}

func run(cfg config, sc scenario) (report, error) {
	pk := newPicker(sc.ops)

	// Keep-alive HTTP client: reuse connections so TCP setup does not dominate.
	transport := &http.Transport{
		MaxIdleConns:        cfg.concurrency * 2,
		MaxIdleConnsPerHost: cfg.concurrency * 2,
		MaxConnsPerHost:     cfg.concurrency * 2,
		IdleConnTimeout:     90 * time.Second,
	}
	client := &http.Client{Transport: transport, Timeout: 30 * time.Second}
	defer transport.CloseIdleConnections()

	// stop ends the issuing of new requests; in-flight ones finish on their
	// own context, so the deadline adds no spurious errors to the tail.
	stop := make(chan struct{})
	timer := time.AfterFunc(cfg.duration, func() { close(stop) })
	defer timer.Stop()

	// Optional fixed-rate ticker shared by all workers (rps>0 only).
	var ticks <-chan time.Time
	if cfg.rps > 0 {
		t := time.NewTicker(time.Second / time.Duration(cfg.rps))
		defer t.Stop()
		ticks = t.C
	}

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		samples []sample
	)

	start := time.Now()
	for w := 0; w < cfg.concurrency; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// Ramp-up: stagger worker entry over the ramp window.
			if cfg.rampUp > 0 {
				delay := time.Duration(int64(cfg.rampUp) * int64(id) / int64(cfg.concurrency))
				select {
				case <-time.After(delay):
				case <-stop:
					return
				}
			}
			rng := rand.New(rand.NewSource(cfg.seed + int64(id)))
			local := make([]sample, 0, 1024)
			for {
				if ticks != nil {
					select {
					case <-stop:
						flush(&mu, &samples, local)
						return
					case <-ticks:
					}
				} else {
					select {
					case <-stop:
						flush(&mu, &samples, local)
						return
					default:
					}
				}
				o := pk.pick(rng)
				tenant := rng.Intn(cfg.tenants) + 1
				local = append(local, doRequest(client, cfg.target, sc.tenant(tenant), sc.ok, o, rng))
				if len(local) >= 4096 {
					flush(&mu, &samples, local)
					local = local[:0]
				}
			}
		}(w)
	}
	wg.Wait()
	elapsed := time.Since(start)

	return buildReport(cfg, elapsed, samples), nil
}

func flush(mu *sync.Mutex, dst *[]sample, src []sample) {
	if len(src) == 0 {
		return
	}
	mu.Lock()
	*dst = append(*dst, src...)
	mu.Unlock()
}

// doRequest issues one request for op o under the given X-Tenant-ID and
// returns its sample. A status ok rejects, a transport error or a context
// expiry counts as a failure, and so does a stream that ends on an error
// line: its status was sent before the error happened.
func doRequest(client *http.Client, target, tenant string, ok func(int) bool, o op, rng *rand.Rand) sample {
	path, body := o.build(rng)
	t0 := time.Now()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, target+path, jsonBody(body))
	if err != nil {
		return sample{op: o.name, dur: time.Since(t0), fail: true, code: "request"}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenant)

	resp, err := client.Do(req)
	if err != nil {
		return sample{op: o.name, dur: time.Since(t0), fail: true, code: transportCode(err)}
	}
	data, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	s := sample{op: o.name, dur: time.Since(t0), status: resp.StatusCode}
	switch {
	case err != nil:
		s.fail, s.code = true, transportCode(err)
	case !ok(resp.StatusCode):
		s.fail, s.code = true, fmt.Sprintf("%d %s", resp.StatusCode, errorCode(data))
	case resp.StatusCode < 300 && streamError(data):
		s.fail, s.code = true, "stream "+errorCode(lastLine(data))
	}
	return s
}

func transportCode(err error) string {
	if errors.Is(err, context.DeadlineExceeded) || os.IsTimeout(err) {
		return "timeout"
	}
	return "transport"
}

// errorCode reads the code of an error envelope, with a hint of its message
// when it names the lock SQLite gave up on.
func errorCode(body []byte) string {
	var env struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &env) != nil || env.Error.Code == "" {
		return "?"
	}
	m := strings.ToLower(env.Error.Message)
	if strings.Contains(m, "busy") || strings.Contains(m, "locked") {
		return env.Error.Code + "/busy"
	}
	return env.Error.Code
}

func lastLine(data []byte) []byte {
	data = bytes.TrimRight(data, "\n")
	if i := bytes.LastIndexByte(data, '\n'); i >= 0 {
		return data[i+1:]
	}
	return data
}

// streamError reports whether an NDJSON answer ends on its error line.
func streamError(data []byte) bool {
	return bytes.HasPrefix(lastLine(data), []byte(`{"error":`))
}

func mdName(jsonPath string) string {
	if len(jsonPath) > 5 && jsonPath[len(jsonPath)-5:] == ".json" {
		return jsonPath[:len(jsonPath)-5] + ".md"
	}
	return jsonPath + ".md"
}
