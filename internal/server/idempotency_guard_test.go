package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// bareIdempotencyServer is the least Server withIdempotency needs: its store.
// Driving a fake handler through it lets a test count the handler's calls and
// hold it mid-flight, which no real route allows.
func bareIdempotencyServer() *Server {
	return &Server{idempotency: newIdempotencyStore(nil)}
}

func idemRequest(key, body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/v1/fake.create", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set(IdempotencyKeyHeader, key)
	return r
}

// TestIdempotency_KeyReusedWithDifferentBodyIs422: a key names one request,
// not a slot; reusing it for another payload is the client's bug, answered
// 422 rather than by replaying a response that describes something else.
func TestIdempotency_KeyReusedWithDifferentBodyIs422(t *testing.T) {
	ts, _ := idempotencyTestServer(t)

	resp1 := createCollection(t, ts, "1", "reused", "Openings")
	resp1.Body.Close()
	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("first attempt: status %d", resp1.StatusCode)
	}

	resp2 := createCollection(t, ts, "1", "reused", "Endgames")
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("reused key, other body: status %d, want 422", resp2.StatusCode)
	}
	var env errorEnvelope
	if err := json.NewDecoder(resp2.Body).Decode(&env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if env.Error.Code != CodeIdempotencyMismatch {
		t.Errorf("code = %q, want %q", env.Error.Code, CodeIdempotencyMismatch)
	}
}

// TestIdempotency_HandlerReadsTheBody: fingerprinting consumes the body; the
// handler must still see every byte of it.
func TestIdempotency_HandlerReadsTheBody(t *testing.T) {
	s := bareIdempotencyServer()
	var seen string
	h := s.withIdempotency(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		seen = string(b)
	})
	h(httptest.NewRecorder(), idemRequest("k", `{"name":"x"}`))
	if seen != `{"name":"x"}` {
		t.Errorf("handler read %q, want the full body", seen)
	}
}

// TestIdempotency_ReplayRestoresHeaders: a replay is the original response,
// headers included — a gesture's Direction-Version or a create's Location is
// part of what the client is retrying for.
func TestIdempotency_ReplayRestoresHeaders(t *testing.T) {
	s := bareIdempotencyServer()
	var calls atomic.Int32
	h := s.withIdempotency(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Direction-Version", "3")
		w.Header().Set("ETag", `"abc"`)
		w.Header().Set("Location", "/v1/fake/7")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":7}`))
	})

	h(httptest.NewRecorder(), idemRequest("k", `{}`))
	rec := httptest.NewRecorder()
	h(rec, idemRequest("k", `{}`))

	if n := calls.Load(); n != 1 {
		t.Fatalf("handler ran %d times, want 1", n)
	}
	if rec.Code != http.StatusCreated || rec.Body.String() != `{"id":7}` {
		t.Errorf("replay = %d %q, want 201 {\"id\":7}", rec.Code, rec.Body.String())
	}
	for k, want := range map[string]string{
		"Content-Type":            "application/json",
		"Direction-Version":       "3",
		"ETag":                    `"abc"`,
		"Location":                "/v1/fake/7",
		idempotencyReplayedHeader: "true",
	} {
		if got := rec.Header().Get(k); got != want {
			t.Errorf("replayed %s = %q, want %q", k, got, want)
		}
	}
}

// TestIdempotency_ServerErrorNotCached: a 5xx committed nothing a retry
// should be stuck with; the next attempt with the same key runs for real.
func TestIdempotency_ServerErrorNotCached(t *testing.T) {
	s := bareIdempotencyServer()
	var calls atomic.Int32
	h := s.withIdempotency(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte("ok"))
	})
	h(httptest.NewRecorder(), idemRequest("k", `{}`))
	rec := httptest.NewRecorder()
	h(rec, idemRequest("k", `{}`))
	if rec.Code != http.StatusOK || calls.Load() != 2 {
		t.Errorf("retry after 5xx: status %d, calls %d; want 200 and 2", rec.Code, calls.Load())
	}
	if rec.Header().Get(idempotencyReplayedHeader) != "" {
		t.Error("retry after 5xx was marked as a replay")
	}
}

// TestIdempotency_ConcurrentSameKeyRunsOnce: a client that retries before
// its first attempt answered must not create twice; the duplicate waits for
// the first and is handed its response.
func TestIdempotency_ConcurrentSameKeyRunsOnce(t *testing.T) {
	s := bareIdempotencyServer()
	var calls atomic.Int32
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	h := s.withIdempotency(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		started <- struct{}{}
		<-release
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1}`))
	})

	rec1, rec2 := httptest.NewRecorder(), httptest.NewRecorder()
	var wg sync.WaitGroup
	wg.Go(func() { h(rec1, idemRequest("k", `{}`)) })
	<-started
	wg.Go(func() { h(rec2, idemRequest("k", `{}`)) })
	waitForIdempotencyWaiters(t, s.idempotency, 1)

	// A different body under the in-flight key is refused at once, without
	// waiting for the first attempt.
	recBad := httptest.NewRecorder()
	h(recBad, idemRequest("k", `{"other":true}`))
	if recBad.Code != http.StatusUnprocessableEntity {
		t.Errorf("other body under an in-flight key: status %d, want 422", recBad.Code)
	}

	close(release)
	wg.Wait()

	if n := calls.Load(); n != 1 {
		t.Fatalf("handler ran %d times, want 1", n)
	}
	if rec2.Code != http.StatusOK || rec2.Body.String() != `{"id":1}` {
		t.Errorf("duplicate got %d %q, want the first attempt's 200 {\"id\":1}", rec2.Code, rec2.Body.String())
	}
	if rec2.Header().Get(idempotencyReplayedHeader) != "true" {
		t.Error("duplicate not marked as a replay")
	}
}

// waitForIdempotencyWaiters spins until n requests are parked on an in-flight
// key: the duplicate must be waiting before the first one is released, or
// the test would pass by sequencing rather than by deduplication.
func waitForIdempotencyWaiters(t *testing.T, s *idempotencyStore, n int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for s.waiting() < n {
		if time.Now().After(deadline) {
			t.Fatalf("no request parked on the in-flight key")
		}
		runtime.Gosched()
	}
}

// TestIdempotencyStore_CapIsPerTenant: a tenant minting keys by the thousand
// evicts its own oldest keys, never another tenant's.
func TestIdempotencyStore_CapIsPerTenant(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	store := newIdempotencyStore(func() time.Time { return now })
	res := idempotencyResult{status: 200, expiresAt: now.Add(idempotencyTTL)}

	store.set("quiet", "mine", res)
	for i := 0; i <= idempotencyMaxEntriesPerTenant; i++ {
		store.set("chatty", keyFor(i), res)
	}
	if _, ok := store.get("quiet", "mine"); !ok {
		t.Error("a chatty tenant evicted another tenant's key")
	}
	if _, ok := store.get("chatty", keyFor(0)); ok {
		t.Error("the chatty tenant exceeded its own cap without evicting")
	}
}
