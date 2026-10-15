package server

import (
	"context"
	"net/http"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
)

// poolOrder runs, on a one-worker pool held until every job is queued, n
// units for each tenant listed and returns the tenants in the order their
// units ran.
func poolOrder(t *testing.T, weights map[string]int, jobs map[string]int, order []string) []string {
	t.Helper()
	p := newAnalysisPool(1, weights)
	t.Cleanup(p.close)
	gate := make(chan struct{})
	held := p.submit("gate", func() func() (func(searcherFor), bool) {
		taken := false
		return func() (func(searcherFor), bool) {
			if taken {
				return nil, false
			}
			taken = true
			return func(searcherFor) { <-gate }, true
		}
	}())
	var mu sync.Mutex
	var ran []string
	var queued []*poolJob
	for _, scope := range order {
		left := jobs[scope]
		queued = append(queued, p.submit(scope, func() (func(searcherFor), bool) {
			if left == 0 {
				return nil, false
			}
			left--
			return func(searcherFor) {
				mu.Lock()
				ran = append(ran, scope)
				mu.Unlock()
			}, true
		}))
	}
	close(gate)
	held.wait()
	for _, j := range queued {
		j.wait()
	}
	return ran
}

// TestAnalysisPoolTakesTenantsInTurn: a tenant that queues one position
// behind another's long sweep waits for one of the sweep's positions, not
// for the whole sweep.
func TestAnalysisPoolTakesTenantsInTurn(t *testing.T) {
	got := poolOrder(t, nil, map[string]int{"club": 5, "alice": 1}, []string{"club", "alice"})
	want := []string{"club", "alice", "club", "club", "club", "club"}
	if !slices.Equal(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
}

// TestAnalysisPoolWeights: a tenant of weight 3 is served three positions
// per turn while one of weight 1 is served one.
func TestAnalysisPoolWeights(t *testing.T) {
	got := poolOrder(t, map[string]int{"club": 3}, map[string]int{"club": 6, "alice": 3}, []string{"club", "alice"})
	want := []string{"club", "club", "club", "alice", "club", "club", "club", "alice", "alice"}
	if !slices.Equal(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
}

// TestAnalysisPoolRunsJobsOfOneTenantInOrder: within a tenant, a job queued
// second waits for the first.
func TestAnalysisPoolRunsJobsOfOneTenantInOrder(t *testing.T) {
	p := newAnalysisPool(2, nil)
	t.Cleanup(p.close)
	var mu sync.Mutex
	var ran []int
	mk := func(id, n int) func() (func(searcherFor), bool) {
		return func() (func(searcherFor), bool) {
			if n == 0 {
				return nil, false
			}
			n--
			return func(searcherFor) { mu.Lock(); ran = append(ran, id); mu.Unlock() }, true
		}
	}
	a, b := p.submit("t", mk(1, 50)), p.submit("t", mk(2, 1))
	a.wait()
	b.wait()
	if i := slices.Index(ran, 2); i < 49 {
		t.Fatalf("the second job ran at %d, before the first had handed out its 50 units", i)
	}
}

// TestAnalysisPoolCloseEndsQueuedJobs: closing the pool ends what is still
// queued, so no request waits forever on a daemon shutting down.
func TestAnalysisPoolCloseEndsQueuedJobs(t *testing.T) {
	p := newAnalysisPool(1, nil)
	gate := make(chan struct{})
	held := p.submit("a", func() func() (func(searcherFor), bool) {
		taken := false
		return func() (func(searcherFor), bool) {
			if taken {
				return nil, false
			}
			taken = true
			return func(searcherFor) { <-gate }, true
		}
	}())
	queued := p.submit("b", func() (func(searcherFor), bool) { return func(searcherFor) {}, true })
	p.close()
	queued.wait()
	close(gate)
	held.wait()
}

func TestParseAnalysisWeights(t *testing.T) {
	got, err := parseAnalysisWeights(" club=3, alice=1 ")
	if err != nil || got["club"] != 3 || got["alice"] != 1 || len(got) != 2 {
		t.Fatalf("parse = %v, %v", got, err)
	}
	for _, bad := range []string{"club", "club=0", "=2", "club=x"} {
		if _, err := parseAnalysisWeights(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

// TestAnalysisPoolRunSkipsAnEndedContext: a unit whose caller left before
// its turn does not run.
func TestAnalysisPoolRunSkipsAnEndedContext(t *testing.T) {
	p := newAnalysisPool(1, nil)
	t.Cleanup(p.close)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if p.run(ctx, "a", func(searcherFor) { t.Error("ran for a caller already gone") }) {
		t.Error("run reports a unit that did not run")
	}
}

const evalXGID = "XGID=-b----E-C---eE---c-e----B-:0:0:1:52:0:0:0:0:10"

// TestEvaluateOnStoppedPoolIs503: an evaluation the stopping daemon no
// longer takes is refused, never answered 200 with an empty verdict.
func TestEvaluateOnStoppedPoolIs503(t *testing.T) {
	ts, srv := newQuotaTestServer(t, TenantQuotas{})
	srv.analysis.close()
	resp := post(t, ts, "/v1/gammonnet.evaluate", gammonnetEvaluateReq{XGID: evalXGID})
	defer resp.Body.Close()
	status, e := errorOf(t, resp)
	if status != http.StatusServiceUnavailable || e.Code != CodeUnavailable {
		t.Fatalf("evaluate on a stopped pool: %d %+v; want 503 unavailable", status, e)
	}
}

// TestAnalysisPoolRunSkipsAContextEndedInQueue: a caller that leaves while
// its unit waits behind another tenant's costs nothing once the turn comes —
// the evaluate route's client gone before its turn.
func TestAnalysisPoolRunSkipsAContextEndedInQueue(t *testing.T) {
	p := newAnalysisPool(1, nil)
	t.Cleanup(p.close)
	gate := make(chan struct{})
	held := p.submit("other", func() func() (func(searcherFor), bool) {
		taken := false
		return func() (func(searcherFor), bool) {
			if taken {
				return nil, false
			}
			taken = true
			return func(searcherFor) { <-gate }, true
		}
	}())
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan bool)
	go func() {
		result <- p.run(ctx, "a", func(searcherFor) { t.Error("ran for a caller gone while queued") })
	}()
	cancel()
	close(gate)
	held.wait()
	if <-result {
		t.Error("run reports a unit that did not run")
	}
}

// TestBoundSearcherStopsOnCancel: a worker's searcher bound to a cancelled
// request stops instead of finishing the search, and is free again after
// release.
func TestBoundSearcherStopsOnCancel(t *testing.T) {
	s, err := gammonnet.NewBatchSearcher(2, 0)
	if err != nil {
		t.Fatal(err)
	}
	get := &workerEngine{searchers: map[[2]int]*gammonnet.Searcher{{2, 0}: s}}
	pos, err := domain.DecodeXGID("XGID=-b----E-C---eE---c-e----B-:0:0:1:00:0:0:0:0:10")
	if err != nil {
		t.Fatal(err)
	}
	pos.Dice = [2]int{0, 0}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	searcher, release := boundSearcher(ctx, get, 2, 0)
	_, err = gammonnet.EvaluatePositionWithMET(searcher, pos, nil, 2, 0, 0)
	release()
	if !gammonnet.IsCanceled(err) {
		t.Fatalf("err = %v, want a cancelled search", err)
	}

	searcher, release = boundSearcher(context.Background(), get, 2, 0)
	_, err = gammonnet.EvaluatePositionWithMET(searcher, pos, nil, 2, 0, 0)
	release()
	if err != nil {
		t.Fatalf("after release: %v", err)
	}

	if s, release := boundSearcher(ctx, &workerEngine{searchers: map[[2]int]*gammonnet.Searcher{{2, 0}: nil}}, 2, 0); s != nil {
		t.Fatal("no searcher built, yet one returned")
	} else {
		release()
	}
}

// TestUnitCostCountsLentTime holds a unit's price to the engine time it used:
// a 2-ply search that forks into idle workers costs its wall time plus what
// the helpers spent, or a tenant's quota would buy as many cores as were idle.
func TestUnitCostCountsLentTime(t *testing.T) {
	if testing.Short() {
		t.Skip("a 2-ply search")
	}
	pos, err := domain.DecodeXGID("XGID=-b----E-C---eE---c-e----B-:0:0:1:00:0:0:0:0:10")
	if err != nil {
		t.Fatal(err)
	}
	pos.Dice = [2]int{0, 0}

	e := &workerEngine{cores: gammonnet.NewCores(4), searchers: map[[2]int]*gammonnet.Searcher{}}
	cost := e.clock()
	start := time.Now()
	searcher, release := boundSearcher(context.Background(), e, 2, 0)
	if _, err := gammonnet.EvaluatePositionWithMET(searcher, pos, nil, 2, 0, 0); err != nil {
		t.Fatal(err)
	}
	release()
	wall, c := time.Since(start), cost()
	if searcher.LentTime() == 0 {
		t.Fatal("the 2-ply search lent nothing to helpers with three tokens free")
	}
	if c < wall+searcher.LentTime() {
		t.Fatalf("unit priced %v, wall %v + lent %v", c, wall, searcher.LentTime())
	}
}
