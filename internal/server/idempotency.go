package server

// Idempotency-Key, as this server keeps it.
//
// Most /v1 methods need no such mechanism: a read is safe to repeat.
// withIdempotency wraps the calls that write something new on every
// invocation with no natural dedup key — the "create" calls (collections,
// tournaments, directions, rencontres, anki.reviewCard) — and
// positions.save, whose effect dedups on the Zobrist hash but whose response
// (created) does not: a retry after a lost response must replay created=true.
// It also wraps the gestures that change a direction, a rencontre or
// a transcription. Every other route ignores the header.
//
// What a key guarantees:
//
//   - A key names one request: (tenant, path, key), plus a fingerprint of the
//     method, path and body. Reusing the key for another body is refused
//     with 422 CodeIdempotencyMismatch, never answered with the response of
//     a request the client did not send.
//   - A replay is the original response — status, body and the handler's
//     headers (Content-Type, Direction-Version, ETag, Location…) — marked
//     with Idempotency-Replayed: true.
//   - The replay is served before anything the wrapped handler checks. On a
//     gesture that means before If-Match: a retried gesture whose first
//     attempt succeeded gets that success back, not the 428 a missing
//     If-Match or the 409 a now-stale revision would earn a fresh attempt.
//   - Concurrent duplicates run the handler once: the second request waits
//     for the first and is handed its response, whatever its status.
//   - Only a 2xx is remembered: a failed attempt wrote nothing durable, so a
//     later retry with the same key is a genuine new attempt.
//   - Keys live in this process's memory for idempotencyTTL, at most
//     idempotencyMaxEntriesPerTenant per tenant. They survive neither a
//     restart nor a hop to another instance behind a load balancer.

import (
	"bytes"
	"container/list"
	"crypto/sha256"
	"io"
	"net/http"
	"sync"
	"time"
)

// IdempotencyKeyHeader is the request header a caller supplies to make a
// retried call safe on the routes withIdempotency wraps.
const IdempotencyKeyHeader = "Idempotency-Key"

// idempotencyReplayedHeader marks a response served from the store rather
// than by running the handler for this request, so a caller (and a test)
// can tell the two apart without inspecting the body.
const idempotencyReplayedHeader = "Idempotency-Replayed"

// idempotencyTTL bounds how long a result is remembered — long enough to
// cover a client's realistic retry window (a dropped connection, a retried
// batch job) without holding every key forever. 24h matches the convention
// several public idempotency-key APIs (e.g. Stripe) use.
const idempotencyTTL = 24 * time.Hour

// idempotencyMaxEntriesPerTenant caps each tenant's keys, evicting that
// tenant's least-recently-used one. Per tenant, not global: a client that
// mints a fresh key per attempt must only cost itself its own replays, never
// another tenant's.
const idempotencyMaxEntriesPerTenant = 1_000

// idempotencyNotReplayed are the recorded headers a replay does not restore:
// the transport sets them for the response actually being written, and the
// replay marker is this middleware's own.
var idempotencyNotReplayed = map[string]bool{
	"Content-Length":          true,
	"Date":                    true,
	idempotencyReplayedHeader: true,
}

// idempotencyResult is a recorded response and the fingerprint of the
// request that produced it.
type idempotencyResult struct {
	fingerprint [sha256.Size]byte
	status      int
	header      http.Header
	body        []byte
	expiresAt   time.Time
}

// idempotencyFlight is a keyed request whose handler is still running.
// result is valid once done is closed.
type idempotencyFlight struct {
	fingerprint [sha256.Size]byte
	done        chan struct{}
	result      idempotencyResult
	waiters     int
}

// idempotencyLRU is one tenant's remembered results, front = most recently
// used.
type idempotencyLRU struct {
	entries map[string]*list.Element // key -> element wrapping *idempotencyEntry
	order   *list.List
}

type idempotencyEntry struct {
	key    string
	result idempotencyResult
}

type idempotencyStore struct {
	mu       sync.Mutex
	tenants  map[string]*idempotencyLRU
	inflight map[string]*idempotencyFlight // tenant + "\x00" + key
	now      func() time.Time
}

func newIdempotencyStore(now func() time.Time) *idempotencyStore {
	if now == nil {
		now = time.Now
	}
	return &idempotencyStore{
		tenants:  make(map[string]*idempotencyLRU),
		inflight: make(map[string]*idempotencyFlight),
		now:      now,
	}
}

// get returns tenant's remembered result for key, if present and not
// expired. An expired entry is evicted on the way out, so a caller never
// observes it even though nothing sweeps the store.
func (s *idempotencyStore) get(tenant, key string) (idempotencyResult, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getLocked(tenant, key)
}

func (s *idempotencyStore) getLocked(tenant, key string) (idempotencyResult, bool) {
	lru, ok := s.tenants[tenant]
	if !ok {
		return idempotencyResult{}, false
	}
	el, ok := lru.entries[key]
	if !ok {
		return idempotencyResult{}, false
	}
	entry := el.Value.(*idempotencyEntry)
	if s.now().After(entry.result.expiresAt) {
		lru.order.Remove(el)
		delete(lru.entries, key)
		if len(lru.entries) == 0 {
			delete(s.tenants, tenant)
		}
		return idempotencyResult{}, false
	}
	lru.order.MoveToFront(el)
	return entry.result, true
}

// set remembers result under (tenant, key), evicting that tenant's
// least-recently-used entry first if it is already at the cap.
func (s *idempotencyStore) set(tenant, key string, result idempotencyResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.setLocked(tenant, key, result)
}

func (s *idempotencyStore) setLocked(tenant, key string, result idempotencyResult) {
	lru, ok := s.tenants[tenant]
	if !ok {
		lru = &idempotencyLRU{entries: make(map[string]*list.Element), order: list.New()}
		s.tenants[tenant] = lru
	}
	if el, ok := lru.entries[key]; ok {
		el.Value.(*idempotencyEntry).result = result
		lru.order.MoveToFront(el)
		return
	}
	if len(lru.entries) >= idempotencyMaxEntriesPerTenant {
		if oldest := lru.order.Back(); oldest != nil {
			lru.order.Remove(oldest)
			delete(lru.entries, oldest.Value.(*idempotencyEntry).key)
		}
	}
	lru.entries[key] = lru.order.PushFront(&idempotencyEntry{key: key, result: result})
}

// idempotencyClaim is what begin found for a keyed request.
type idempotencyClaim int

const (
	claimLead     idempotencyClaim = iota // nothing known: run the handler
	claimReplay                           // a result is remembered or was shared
	claimMismatch                         // the key names another request
)

// begin decides a keyed request's fate in one critical section, so two
// duplicates can never both become the one that runs the handler. A
// duplicate of an in-flight request blocks until it finishes or ctxDone
// fires (ok=false: the caller is gone, write nothing).
func (s *idempotencyStore) begin(tenant, key string, fp [sha256.Size]byte, ctxDone <-chan struct{}) (claim idempotencyClaim, res idempotencyResult, ok bool) {
	s.mu.Lock()
	if cached, hit := s.getLocked(tenant, key); hit {
		s.mu.Unlock()
		if cached.fingerprint != fp {
			return claimMismatch, idempotencyResult{}, true
		}
		return claimReplay, cached, true
	}
	fk := tenant + "\x00" + key
	flight, running := s.inflight[fk]
	if !running {
		s.inflight[fk] = &idempotencyFlight{fingerprint: fp, done: make(chan struct{})}
		s.mu.Unlock()
		return claimLead, idempotencyResult{}, true
	}
	if flight.fingerprint != fp {
		s.mu.Unlock()
		return claimMismatch, idempotencyResult{}, true
	}
	flight.waiters++
	s.mu.Unlock()

	select {
	case <-flight.done:
		ok = true
	case <-ctxDone:
	}
	s.mu.Lock()
	flight.waiters--
	res = flight.result
	s.mu.Unlock()
	return claimReplay, res, ok
}

// finish publishes the leader's result to its waiters and, for a 2xx only,
// remembers it: a failed attempt wrote nothing durable, so a later retry
// with the same key deserves a genuine new attempt rather than a day of
// replayed failure.
func (s *idempotencyStore) finish(tenant, key string, result idempotencyResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fk := tenant + "\x00" + key
	flight := s.inflight[fk]
	delete(s.inflight, fk)
	if result.status >= 200 && result.status < 300 {
		result.expiresAt = s.now().Add(idempotencyTTL)
		s.setLocked(tenant, key, result)
	}
	if flight != nil {
		flight.result = result
		close(flight.done)
	}
}

// waiting counts the requests parked behind an in-flight duplicate.
func (s *idempotencyStore) waiting() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, f := range s.inflight {
		n += f.waiters
	}
	return n
}

// idempotencyBufferingRecorder captures a handler's response instead of
// writing it straight through, so it can be remembered and shared once the
// final status is known. The wrapped routes are single-shot JSON calls,
// never a stream.
type idempotencyBufferingRecorder struct {
	header http.Header
	status int
	body   []byte
}

func newIdempotencyBufferingRecorder() *idempotencyBufferingRecorder {
	return &idempotencyBufferingRecorder{header: make(http.Header), status: http.StatusOK}
}

func (r *idempotencyBufferingRecorder) Header() http.Header { return r.header }

func (r *idempotencyBufferingRecorder) WriteHeader(code int) { r.status = code }

func (r *idempotencyBufferingRecorder) Write(b []byte) (int, error) {
	r.body = append(r.body, b...)
	return len(b), nil
}

// idempotencyFingerprint identifies the request a key was first used for.
func idempotencyFingerprint(r *http.Request, body []byte) [sha256.Size]byte {
	h := sha256.New()
	_, _ = io.WriteString(h, r.Method+"\x00"+r.URL.Path+"\x00")
	_, _ = h.Write(body)
	var fp [sha256.Size]byte
	copy(fp[:], h.Sum(nil))
	return fp
}

func writeIdempotencyResult(w http.ResponseWriter, res idempotencyResult, replayed bool) {
	for k, v := range res.header {
		w.Header()[k] = append([]string(nil), v...)
	}
	if replayed {
		w.Header().Set(idempotencyReplayedHeader, "true")
	}
	w.WriteHeader(res.status)
	_, _ = w.Write(res.body)
}

// withIdempotency wraps next with the Idempotency-Key guarantees described
// at the top of this file. Requests with no key (the overwhelming majority —
// this is opt-in) pass straight through with no buffering at all. Scoped by
// tenant AND path AND key: two tenants' independently-chosen keys, or one
// key on two routes, never cross-contaminate.
func (s *Server) withIdempotency(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get(IdempotencyKeyHeader)
		if key == "" {
			next(w, r)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeDecodeError(w, "request body", err)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))

		tenant := scopeOf(r)
		storeKey := r.URL.Path + "\x00" + key
		fp := idempotencyFingerprint(r, body)

		claim, res, ok := s.idempotency.begin(tenant, storeKey, fp, r.Context().Done())
		switch claim {
		case claimMismatch:
			writeErrorCode(w, CodeIdempotencyMismatch,
				"Idempotency-Key already used for a different request; use a new key")
			return
		case claimReplay:
			if ok {
				writeIdempotencyResult(w, res, true)
			}
			return
		case claimLead:
		}

		// Waiters must be released even if next panics, or they would block
		// until their own clients give up.
		result := idempotencyResult{fingerprint: fp, status: http.StatusInternalServerError}
		defer func() { s.idempotency.finish(tenant, storeKey, result) }()

		rec := newIdempotencyBufferingRecorder()
		next(rec, r)

		header := make(http.Header, len(rec.header))
		for k, v := range rec.header {
			if !idempotencyNotReplayed[k] {
				header[k] = append([]string(nil), v...)
			}
		}
		result = idempotencyResult{fingerprint: fp, status: rec.status, header: header, body: rec.body}
		writeIdempotencyResult(w, result, false)
	}
}
