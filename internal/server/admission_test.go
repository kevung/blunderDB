package server

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// TestHeavyGate_BoundsAndRefuses: past its slots a library-wide read waits,
// past the queue or the wait it is answered 503 with Retry-After, and a read
// of one row never goes through the gate.
func TestHeavyGate_BoundsAndRefuses(t *testing.T) {
	g := newHeavyGate(1)
	g.queue = 1
	g.wait = 50 * time.Millisecond
	release := make(chan struct{})
	entered := make(chan struct{}, 4)
	h := g.wrap(func(w http.ResponseWriter, _ *http.Request) {
		entered <- struct{}{}
		<-release
	}, nil)
	serve := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h(w, httptest.NewRequest(http.MethodPost, "/v1/stats.compute", nil))
		return w
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); serve() }()
	<-entered // the one slot is taken

	waited := make(chan *httptest.ResponseRecorder, 1)
	go func() { waited <- serve() }() // queued, then times out
	for g.waiting.Load() == 0 {
		time.Sleep(time.Millisecond)
	}
	if w := serve(); w.Code != http.StatusServiceUnavailable || w.Header().Get("Retry-After") == "" {
		t.Errorf("beyond the queue: status %d, Retry-After %q; want 503 with Retry-After", w.Code, w.Header().Get("Retry-After"))
	}
	if w := <-waited; w.Code != http.StatusServiceUnavailable {
		t.Errorf("past the wait: status %d, want 503", w.Code)
	}
	close(release)
	wg.Wait()
	if w := serve(); w.Code != http.StatusOK {
		t.Errorf("slot free again: status %d, want 200", w.Code)
	}

	if heavyRoute("/v1/positions.load") || heavyRoute("/v1/search.parse") || !heavyRoute("/v1/stats.compute") || !heavyRoute("/v1/matches.list") {
		t.Error("heavyRoute classifies a route wrongly")
	}
}

// TestHeavyGate_WaitsForStatsBeforeTakingASlot: a statistics read arriving
// during the start fill waits for it outside the gate, so the slot stays free
// for the other library-wide reads.
func TestHeavyGate_WaitsForStatsBeforeTakingASlot(t *testing.T) {
	g := newHeavyGate(1)
	ready := make(chan struct{})
	served := make(chan struct{})
	h := g.wrap(func(http.ResponseWriter, *http.Request) {}, ready)
	go func() {
		h(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/v1/stats.compute", nil))
		close(served)
	}()
	other := g.wrap(func(http.ResponseWriter, *http.Request) {}, nil)
	w := httptest.NewRecorder()
	other(w, httptest.NewRequest(http.MethodPost, "/v1/search.find", nil))
	if w.Code != http.StatusOK || len(g.slots) != 0 {
		t.Fatalf("while the fill runs: other read status %d, slots held %d; want 200 and 0", w.Code, len(g.slots))
	}
	close(ready)
	<-served
}
