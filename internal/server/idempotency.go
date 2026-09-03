package server

import (
	"container/list"
	"net/http"
	"sync"
	"time"
)

// IdempotencyKeyHeader is the request header a caller supplies to make a
// retried call safe. Most /v1 methods need no such mechanism at all — see
// the package doc comment on idempotency — but a handful of "create" calls
// genuinely insert a new row on every invocation with no natural dedup key
// (unlike positions.save, which Zobrist-hashes its content): collections.create,
// tournaments.create, anki.reviewCard (#236). Wrap exactly those routes'
// handler with withIdempotency; every other route ignores this header
// entirely.
const IdempotencyKeyHeader = "Idempotency-Key"

// idempotencyReplayedHeader marks a response served from the cache rather
// than by re-running the handler, so a caller (and a test) can tell the two
// apart without inspecting the body.
const idempotencyReplayedHeader = "Idempotency-Replayed"

// idempotencyTTL bounds how long a (tenant, route, key) result is
// remembered — long enough to cover a client's realistic retry window (a
// dropped connection, a retried batch job) without holding every key
// forever. 24h matches the convention several public idempotency-key APIs
// (e.g. Stripe) use.
const idempotencyTTL = 24 * time.Hour

// idempotencyMaxEntries hard-caps the store the same way RateLimiter caps
// its per-tenant buckets (middleware.DefaultMaxBuckets): without a ceiling a
// caller that mints a fresh key per attempt (defeating the point, but not
// this server's problem to prevent) would grow the store without bound
// between sweeps.
const idempotencyMaxEntries = 10_000

// idempotencyResult is a cached response: only ever stored for a genuinely
// successful (2xx) attempt — see idempotencyStore.set's doc comment for why
// a failed attempt is never cached.
type idempotencyResult struct {
	status      int
	contentType string
	body        []byte
	expiresAt   time.Time
}

// idempotencyStore caches at most one result per (tenant, route, key)
// triple, evicting the least-recently-used entry once idempotencyMaxEntries
// is reached — the same bounded-LRU shape as middleware.RateLimiter's bucket
// map, for the same reason.
type idempotencyStore struct {
	mu      sync.Mutex
	entries map[string]*list.Element // key -> element wrapping *idempotencyEntry
	order   *list.List               // front = most recently used
	now     func() time.Time
}

type idempotencyEntry struct {
	key    string
	result idempotencyResult
}

func newIdempotencyStore(now func() time.Time) *idempotencyStore {
	if now == nil {
		now = time.Now
	}
	return &idempotencyStore{
		entries: make(map[string]*list.Element),
		order:   list.New(),
		now:     now,
	}
}

// get returns the cached result for key, if present and not expired. An
// expired entry is evicted on the way out rather than left for the next
// sweep, so a caller never observes stale data even between sweeps.
func (s *idempotencyStore) get(key string) (idempotencyResult, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	el, ok := s.entries[key]
	if !ok {
		return idempotencyResult{}, false
	}
	entry := el.Value.(*idempotencyEntry)
	if s.now().After(entry.result.expiresAt) {
		s.order.Remove(el)
		delete(s.entries, key)
		return idempotencyResult{}, false
	}
	s.order.MoveToFront(el)
	return entry.result, true
}

// set stores result under key, evicting the least-recently-used entry first
// if the store is already at idempotencyMaxEntries. Only ever called for a
// 2xx response (see withIdempotency): a request that failed did not durably
// create anything on the "create" routes this guards, so a caller retrying
// with the same key after a failure gets a genuine new attempt rather than
// being stuck replaying that failure for 24h.
func (s *idempotencyStore) set(key string, result idempotencyResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if el, ok := s.entries[key]; ok {
		el.Value.(*idempotencyEntry).result = result
		s.order.MoveToFront(el)
		return
	}
	if len(s.entries) >= idempotencyMaxEntries {
		oldest := s.order.Back()
		if oldest != nil {
			s.order.Remove(oldest)
			delete(s.entries, oldest.Value.(*idempotencyEntry).key)
		}
	}
	el := s.order.PushFront(&idempotencyEntry{key: key, result: result})
	s.entries[key] = el
}

// idempotencyBufferingRecorder captures a handler's response instead of
// writing it straight through, so withIdempotency can decide whether to
// cache it (2xx only) after seeing the final status — a decision that can
// only be made once the handler has finished, which for these three
// same-shaped rpc/rpcVoid JSON routes is always a single WriteHeader+Write,
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

// withIdempotency wraps next so a request carrying IdempotencyKeyHeader
// replays a prior successful response instead of re-running the handler.
// Requests with no such header (the overwhelming majority — this is opt-in)
// pass straight through with no buffering at all. Scoped by tenant AND path
// AND key: a name collision between two tenants' independently-chosen keys,
// or between two different idempotent routes, must never cross-contaminate.
func (s *Server) withIdempotency(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get(IdempotencyKeyHeader)
		if key == "" {
			next(w, r)
			return
		}
		cacheKey := scopeOf(r) + "\x00" + r.URL.Path + "\x00" + key

		if cached, ok := s.idempotency.get(cacheKey); ok {
			if ct := cached.contentType; ct != "" {
				w.Header().Set("Content-Type", ct)
			}
			w.Header().Set(idempotencyReplayedHeader, "true")
			w.WriteHeader(cached.status)
			_, _ = w.Write(cached.body)
			return
		}

		rec := newIdempotencyBufferingRecorder()
		next(rec, r)

		for k, v := range rec.header {
			w.Header()[k] = v
		}
		w.WriteHeader(rec.status)
		_, _ = w.Write(rec.body)

		if rec.status >= 200 && rec.status < 300 {
			s.idempotency.set(cacheKey, idempotencyResult{
				status:      rec.status,
				contentType: rec.header.Get("Content-Type"),
				body:        rec.body,
				expiresAt:   s.idempotency.now().Add(idempotencyTTL),
			})
		}
	}
}
