package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kevung/blunderdb/internal/server/middleware"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// idempotencyTestServer builds an httptest server with a frozen clock, so a
// test can advance it deterministically past idempotencyTTL without a real
// sleep.
func idempotencyTestServer(t *testing.T) (*httptest.Server, *time.Time) {
	t.Helper()
	st, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	frozen := time.Unix(1_700_000_000, 0)
	srv, err := New(Options{
		Storage: st,
		now:     func() time.Time { return frozen },
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts, &frozen
}

func createCollection(t *testing.T, ts *httptest.Server, tenant, idempotencyKey, name string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/collections.create",
		strings.NewReader(`{"name":"`+name+`","description":""}`))
	req.Header.Set(middleware.TenantHeader, tenant)
	req.Header.Set("Content-Type", "application/json")
	if idempotencyKey != "" {
		req.Header.Set(IdempotencyKeyHeader, idempotencyKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /v1/collections.create: %v", err)
	}
	return resp
}

func decodeIDResp(t *testing.T, resp *http.Response) idResp {
	t.Helper()
	defer resp.Body.Close()
	var out idResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode idResp: %v", err)
	}
	return out
}

// TestIdempotency_ReplaysCachedResponse guards #236: a retried
// collections.create carrying the same Idempotency-Key must get the FIRST
// attempt's id back, not create a second, identically-named collection —
// collections.create has no natural dedup key the way positions.save's
// Zobrist hash gives it one.
func TestIdempotency_ReplaysCachedResponse(t *testing.T) {
	ts, _ := idempotencyTestServer(t)

	resp1 := createCollection(t, ts, "1", "retry-key-1", "Openings")
	defer resp1.Body.Close()
	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("first attempt: status %d", resp1.StatusCode)
	}
	first := decodeIDResp(t, resp1)

	resp2 := createCollection(t, ts, "1", "retry-key-1", "Openings")
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("replayed attempt: status %d", resp2.StatusCode)
	}
	if got := resp2.Header.Get(idempotencyReplayedHeader); got != "true" {
		t.Errorf("%s = %q, want %q", idempotencyReplayedHeader, got, "true")
	}
	second := decodeIDResp(t, resp2)

	if second.ID != first.ID {
		t.Errorf("replayed id = %d, want the first attempt's id %d (a second row was created)", second.ID, first.ID)
	}
}

// TestIdempotency_NoKeyMeansNoDedup: the overwhelming majority of calls send
// no Idempotency-Key at all and must behave exactly as before — two calls
// with no key create two independent collections.
func TestIdempotency_NoKeyMeansNoDedup(t *testing.T) {
	ts, _ := idempotencyTestServer(t)

	resp1 := createCollection(t, ts, "1", "", "Openings")
	defer resp1.Body.Close()
	first := decodeIDResp(t, resp1)
	if got := resp1.Header.Get(idempotencyReplayedHeader); got != "" {
		t.Errorf("unexpected %s header with no Idempotency-Key: %q", idempotencyReplayedHeader, got)
	}

	resp2 := createCollection(t, ts, "1", "", "Openings")
	defer resp2.Body.Close()
	second := decodeIDResp(t, resp2)

	if second.ID == first.ID {
		t.Error("two calls with no Idempotency-Key produced the same id — they must be independent")
	}
}

// TestIdempotency_ScopedByTenant: the same key used by two different tenants
// must never let one tenant's create be silently replayed as another's —
// each tenant gets its own row.
func TestIdempotency_ScopedByTenant(t *testing.T) {
	ts, _ := idempotencyTestServer(t)

	resp1 := createCollection(t, ts, "1", "shared-key", "Openings")
	defer resp1.Body.Close()
	first := decodeIDResp(t, resp1)

	resp2 := createCollection(t, ts, "2", "shared-key", "Openings")
	defer resp2.Body.Close()
	second := decodeIDResp(t, resp2)

	if resp2.Header.Get(idempotencyReplayedHeader) == "true" {
		t.Error("tenant 2's create was replayed from tenant 1's cache entry")
	}
	if second.ID == first.ID {
		t.Error("two different tenants sharing an Idempotency-Key got the same id")
	}
}

// TestIdempotency_FailedAttemptNotCached: a request that never reaches a 2xx
// (here, malformed JSON — a deterministic 400 from decodeJSON, reached
// through withIdempotency exactly like any other failure the wrapped
// handler can produce) must not be cached: a client that fixes its request
// and retries with the same key gets a genuine attempt, not a replayed
// failure for the next 24h.
func TestIdempotency_FailedAttemptNotCached(t *testing.T) {
	ts, _ := idempotencyTestServer(t)

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/collections.create", strings.NewReader(`{not valid json`))
	req.Header.Set(middleware.TenantHeader, "1")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(IdempotencyKeyHeader, "retry-after-fix")
	resp1, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("first attempt: %v", err)
	}
	resp1.Body.Close()
	if resp1.StatusCode != http.StatusBadRequest {
		t.Fatalf("first (malformed) attempt: status %d, want %d", resp1.StatusCode, http.StatusBadRequest)
	}

	resp2 := createCollection(t, ts, "1", "retry-after-fix", "Now With A Name")
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("retry with a fixed request: status %d", resp2.StatusCode)
	}
	if resp2.Header.Get(idempotencyReplayedHeader) == "true" {
		t.Error("the failed first attempt was replayed instead of genuinely retried")
	}
}

// TestIdempotencyStore_ExpiresAfterTTL guards the store's own TTL logic in
// isolation, without going through HTTP.
func TestIdempotencyStore_ExpiresAfterTTL(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	store := newIdempotencyStore(func() time.Time { return now })

	store.set("k", idempotencyResult{status: 200, body: []byte("x"), expiresAt: now.Add(idempotencyTTL)})
	if _, ok := store.get("k"); !ok {
		t.Fatal("expected a hit immediately after set")
	}

	now = now.Add(idempotencyTTL + time.Second)
	if _, ok := store.get("k"); ok {
		t.Error("expected a miss once past idempotencyTTL")
	}
}

// TestIdempotencyStore_EvictsLeastRecentlyUsed guards the bounded-size
// eviction (#236, mirroring middleware.RateLimiter's bucket cap): once full,
// the least-recently-used entry — not an arbitrary one — is evicted.
func TestIdempotencyStore_EvictsLeastRecentlyUsed(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	store := newIdempotencyStore(func() time.Time { return now })

	// Shrink the effective cap for this test by filling past a small number
	// and relying on idempotencyMaxEntries directly would take 10k
	// iterations; instead this test only checks the LRU *ordering* logic on
	// a handful of entries plus one eviction-triggering fill, using the
	// real constant so it stays honest about the actual cap.
	for i := 0; i < idempotencyMaxEntries; i++ {
		store.set(keyFor(i), idempotencyResult{status: 200, expiresAt: now.Add(idempotencyTTL)})
	}
	// Touch the first key so it is no longer the least-recently-used one.
	if _, ok := store.get(keyFor(0)); !ok {
		t.Fatal("expected key 0 to still be present before the evicting insert")
	}
	// One more insert must evict the least-recently-used entry, which is
	// now key 1 (0 was just touched, 1 was not).
	store.set("one-more", idempotencyResult{status: 200, expiresAt: now.Add(idempotencyTTL)})

	if _, ok := store.get(keyFor(0)); !ok {
		t.Error("recently-touched key 0 was evicted, want it kept")
	}
	if _, ok := store.get(keyFor(1)); ok {
		t.Error("least-recently-used key 1 was kept, want it evicted")
	}
}

func keyFor(i int) string {
	return "k" + string(rune('a'+i%26)) + string(rune('0'+i/26))
}
