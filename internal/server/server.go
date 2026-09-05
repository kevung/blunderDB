// Package server implements the `blunderdb serve` HTTP + JSON daemon. It
// exposes the engine over RPC-style routes (POST /v1/<family>.<method>) backed
// by a storage.Storage value.
//
// Security: this daemon performs NO authentication. It trusts the
// X-Tenant-ID header injected by an upstream reverse-proxy and MUST NOT be
// exposed directly to the public internet. See tasks/headless/06-serve-http.md.
package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/netutil"

	"github.com/kevung/blunderdb/internal/server/handlers"
	"github.com/kevung/blunderdb/internal/server/middleware"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// Server is the HTTP daemon. Construct it with New and run it with Run.
type Server struct {
	opts   Options
	health *handlers.Health
	http   *http.Server
	// opsHTTP is the second listener the /ops/ family gets when OpsAddr is
	// set; nil otherwise, and then /ops/ lives on http like any other route.
	opsHTTP    *http.Server
	knownPaths map[string]bool
	// uploadPaths is the exact set of routes limitBody exempts from
	// MaxBodyBytes — the multipart import uploads, which cap themselves at
	// ImportMaxBodyBytes (see uploadRoutes in handlers_imports.go).
	uploadPaths map[string]bool
	// allowedMethod maps every registered pattern to the one HTTP method it
	// accepts, so a request to a known path with the wrong method gets the
	// API's own JSON error envelope (405 + Allow) instead of net/http's
	// automatic text/plain response — see methodNotAllowed (#232).
	allowedMethod map[string]string

	imports *importRegistry
	// gammonnetJobs tracks in-flight gammonNet catch-up sweeps (#130), kept
	// separate from imports so cancelling one can never be confused with the
	// other — reuses importRegistry's scope-keyed cancel bookkeeping under
	// its own instance rather than a new type for the same three methods.
	gammonnetJobs *importRegistry
	rl            *middleware.RateLimiter // nil when rate limiting is disabled
	// spool bounds the total bytes concurrently in-flight imports may hold
	// spooled to $TMPDIR — see handleImport and Options.MaxSpoolBytes (#234).
	spool *spoolQuota
	// idempotency backs withIdempotency: at most one cached response per
	// (tenant, route, Idempotency-Key) triple, for the handful of routes
	// with no natural dedup key (#236) — see idempotency.go.
	idempotency *idempotencyStore
}

// New builds a Server from opts. It returns an error if no Storage is set.
func New(opts Options) (*Server, error) {
	if opts.Storage == nil {
		return nil, errors.New("server: Options.Storage is required")
	}
	opts.applyDefaults()

	s := &Server{
		opts: opts,
		health: &handlers.Health{
			Storage:         opts.Storage,
			Metrics:         opts.Metrics,
			ExpectedVersion: domain.DatabaseVersion,
		},
		imports:       newImportRegistry(),
		gammonnetJobs: newImportRegistry(),
		spool:         newSpoolQuota(opts.MaxSpoolBytes),
		idempotency:   newIdempotencyStore(opts.now),
	}
	if opts.RateLimitRPS > 0 {
		s.rl = middleware.NewRateLimiter(opts.RateLimitRPS, opts.RateLimitBurst, opts.now)
	}

	mux := http.NewServeMux()
	s.knownPaths = make(map[string]bool)
	s.uploadPaths = uploadPaths()
	s.allowedMethod = make(map[string]string)
	for _, rt := range s.routes() {
		mux.HandleFunc(rt.method+" "+rt.pattern, rt.handler)
		s.knownPaths[rt.pattern] = true
		s.allowedMethod[rt.pattern] = rt.method
	}
	// Catch-all: any unmatched path returns the JSON error envelope.
	mux.HandleFunc("/", s.notFound)

	s.http = &http.Server{
		Addr:              opts.Addr,
		Handler:           s.chain(s.withDeadlines(s.methodNotAllowed(mux))),
		ReadHeaderTimeout: opts.ReadHeaderTimeout,
		// Read/WriteTimeout stay unset — see Options.IdleTimeout's doc
		// comment for why; withDeadlines is the per-request replacement.
		IdleTimeout: opts.IdleTimeout,
	}

	// The ops listener, when asked for. It runs the same middleware chain as
	// the main one — a purge still needs its tenant scope, its rate limit and
	// its request id — over a mux that serves nothing but /ops/.
	if opts.OpsAddr != "" {
		opsMux := http.NewServeMux()
		for _, rt := range s.opsRoutes() {
			opsMux.HandleFunc(rt.method+" "+rt.pattern, rt.handler)
			s.knownPaths[rt.pattern] = true
			s.allowedMethod[rt.pattern] = rt.method
		}
		opsMux.HandleFunc("/", s.notFound)
		s.opsHTTP = &http.Server{
			Addr:              opts.OpsAddr,
			Handler:           s.chain(s.withDeadlines(s.methodNotAllowed(opsMux))),
			ReadHeaderTimeout: opts.ReadHeaderTimeout,
			IdleTimeout:       opts.IdleTimeout,
		}
	}
	return s, nil
}

// withDeadlines sets a per-request read/write deadline via
// http.ResponseController before dispatching to next: RequestTimeout for an
// ordinary call, the far more generous StreamTimeout for a route
// streamingPaths names (#234). Errors from SetReadDeadline/SetWriteDeadline
// are ignored — they fail only when the underlying connection genuinely
// cannot support a deadline (e.g. it has been hijacked already), in which
// case there is nothing more useful to do than proceed without one.
func (s *Server) withDeadlines(next http.Handler) http.Handler {
	streaming := s.streamingPaths()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timeout := s.opts.RequestTimeout
		if streaming[r.URL.Path] {
			timeout = s.opts.StreamTimeout
		}
		if timeout > 0 {
			deadline := time.Now().Add(timeout)
			rc := http.NewResponseController(w)
			_ = rc.SetReadDeadline(deadline)
			_ = rc.SetWriteDeadline(deadline)
		}
		next.ServeHTTP(w, r)
	})
}

// chain wraps the mux with the middleware stack. Order (outermost first):
// requestID → recover → metrics → logging → cors → tenant → mux. requestID
// is outermost so every log line below it — including a panic's, and a
// rejected request's — can carry the correlation id; recover is next so it
// catches panics from every remaining layer; tenant is innermost so
// r.Pattern is set by the mux for the metrics/logging labels read after next
// returns.
func (s *Server) chain(mux http.Handler) http.Handler {
	h := mux
	// Rate limiting sits just inside Tenant so it can read the tenant from the
	// context; it is only mounted when enabled (zero overhead otherwise).
	if s.rl != nil {
		h = middleware.RateLimit(s.rl, func(w http.ResponseWriter, _ *http.Request) {
			s.opts.Metrics.IncRateLimitRejected()
			writeErrorCode(w, CodeRateLimited, "too many requests")
		})(h)
	}
	h = middleware.Tenant(s.publicPaths(), s.opts.SingleTenant, func(w http.ResponseWriter, _ *http.Request, msg string) {
		writeErrorCode(w, CodeInvalid, msg)
	})(h)
	// Compression sits outside the tenant gate and inside logging/metrics: it
	// must see the handler's Content-Type (hence inside the mux's own layers)
	// while the byte counts the log line reports stay the uncompressed ones.
	h = middleware.Compress(h)
	h = middleware.CORS(s.opts.CORSAllowOrigin)(h)
	h = middleware.Logging(s.opts.Logger, s.knownPaths, s.opts.now)(h)
	if s.opts.EnableMetrics {
		h = middleware.Metrics(s.opts.Metrics, s.knownPaths, s.opts.now)(h)
	}
	h = s.limitBody(h)
	h = middleware.Recover(s.opts.Logger, func(w http.ResponseWriter, _ *http.Request) {
		writeErrorCode(w, CodeInternal, "internal error")
	})(h)
	h = middleware.RequestID(h)
	return h
}

// publicPaths is the set of routes reachable without a tenant: the probes and
// /metrics. Derived rather than listed so a new probe is public the day it
// lands, and a new domain route can never be.
//
// /ops/ is NOT public, and the exclusion is explicit. This used to read
// "everything outside /v1", which was the same set as long as the only
// non-/v1 routes were the probes — moving vacuum and purge under /ops/ (G.5,
// #233) silently made the two most dangerous calls in the daemon the only
// ones needing no tenant at all. A purge names the tenant it destroys in the
// header it is given; it needs that header more than any other route, not
// less. routes_smoke_test.go's TestRoutesSmoke_TenantRequired is what caught
// it, and still covers both prefixes.
func (s *Server) publicPaths() map[string]bool {
	public := make(map[string]bool)
	for _, rt := range s.routes() {
		if strings.HasPrefix(rt.pattern, "/v1/") || strings.HasPrefix(rt.pattern, "/ops/") {
			continue
		}
		public[rt.pattern] = true
	}
	return public
}

// limitBody caps request bodies to guard against OOM from a malicious client.
// The upload endpoints — exactly uploadPaths, not a "/v1/imports." prefix
// that would also cover imports.cancel (#160) — are exempt from the small
// default cap: they carry uploaded match files and apply their own (larger)
// limit while spooling. A declared Content-Length over the cap is refused
// 413 before a byte of it is read; an undeclared one is cut off by
// MaxBytesReader and the handler's decoder reports it (writeDecodeError).
func (s *Server) limitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil && !s.uploadPaths[r.URL.Path] {
			if r.ContentLength > s.opts.MaxBodyBytes {
				writeBodyTooLarge(w, s.opts.MaxBodyBytes)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, s.opts.MaxBodyBytes)
		}
		next.ServeHTTP(w, r)
	})
}

// Handler exposes the fully-wired http.Handler for tests (httptest).
func (s *Server) Handler() http.Handler { return s.http.Handler }

// rateLimitSweepInterval and rateLimitMaxIdle govern eviction of idle per-tenant
// buckets so the map doesn't grow without bound.
const (
	rateLimitSweepInterval = 5 * time.Minute
	rateLimitMaxIdle       = 30 * time.Minute
)

// sweepRateLimiter periodically evicts idle tenant buckets and publishes the
// live bucket count to the metrics registry, until ctx is cancelled.
func (s *Server) sweepRateLimiter(ctx context.Context) {
	t := time.NewTicker(rateLimitSweepInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.opts.Metrics.SetRateLimitBuckets(s.rl.Sweep(rateLimitMaxIdle))
		}
	}
}

// poolStatsSweepInterval bounds how stale blunderdb_pg_pool_* can be.
const poolStatsSweepInterval = 15 * time.Second

// poolStatsProvider is implemented by postgres.Storage — duck-typed so this
// package need not import the postgres package just for this (it already
// does, in serve.go, but the interface keeps this specific dependency
// explicit and minimal). The SQLite backend does not implement it, so the
// sweep below is simply never started against it.
type poolStatsProvider interface {
	PoolStats() (acquired, idle, max int32, waitCount int64)
}

// sweepPoolStats periodically publishes the PostgreSQL connection pool's
// state to the metrics registry (#235), until ctx is cancelled.
func (s *Server) sweepPoolStats(ctx context.Context, pool poolStatsProvider) {
	t := time.NewTicker(poolStatsSweepInterval)
	defer t.Stop()
	publish := func() {
		acquired, idle, max, waitCount := pool.PoolStats()
		s.opts.Metrics.SetPoolStats(acquired, idle, max, waitCount)
	}
	publish() // first data point without waiting a full interval
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			publish()
		}
	}
}

// businessMetricsSweepInterval bounds how stale the imports/gammonNet/spool
// gauges and blunderdb_database_size_bytes can be (#238). Cheap enough (an
// in-memory map length, an atomic load, and one query against the already-
// open backend) to run this often even though database size does not
// actually change every 15 seconds in practice.
const businessMetricsSweepInterval = 15 * time.Second

// sizeProvider is implemented by a storage backend that can report its own
// on-disk (or server-side) footprint — sqlite.Storage stats its main file,
// postgres.Storage asks pg_database_size. Duck-typed like poolStatsProvider
// above: a backend that doesn't implement it simply never has
// blunderdb_database_size_bytes published (#238).
type sizeProvider interface {
	DatabaseSizeBytes(ctx context.Context) (int64, error)
}

// sweepBusinessMetrics periodically publishes in-flight-work and database-
// size gauges to the metrics registry (#238), until ctx is cancelled.
func (s *Server) sweepBusinessMetrics(ctx context.Context) {
	t := time.NewTicker(businessMetricsSweepInterval)
	defer t.Stop()
	publish := func() {
		s.opts.Metrics.SetImportsInFlight(s.imports.count())
		s.opts.Metrics.SetImportSpoolBytes(s.spool.usage())
		s.opts.Metrics.SetGammonNetSweepsInFlight(s.gammonnetJobs.count())
		if sp, ok := s.opts.Storage.(sizeProvider); ok {
			if n, err := sp.DatabaseSizeBytes(ctx); err == nil {
				s.opts.Metrics.SetDatabaseSizeBytes(n)
			}
		}
	}
	publish() // first data point without waiting a full interval
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			publish()
		}
	}
}

// Run starts the server and blocks until ctx is cancelled, then shuts down
// gracefully within ShutdownTimeout. It returns the listener/serve error, or
// nil on a clean shutdown.
func (s *Server) Run(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.opts.Addr)
	if err != nil {
		return fmt.Errorf("server: listen %s: %w", s.opts.Addr, err)
	}
	if s.opts.MaxConnections > 0 {
		// The (n+1)-th concurrent connection blocks in Accept until one of
		// the first n closes, instead of every connection a client opens
		// getting its own goroutine and file descriptor unconditionally
		// (#234).
		ln = netutil.LimitListener(ln, s.opts.MaxConnections)
	}

	if s.opsHTTP != nil {
		opsLn, err := net.Listen("tcp", s.opsHTTP.Addr)
		if err != nil {
			return fmt.Errorf("server: listen ops %s: %w", s.opsHTTP.Addr, err)
		}
		go func() {
			s.opts.Logger.Warn("ops endpoints listening on a separate address — never expose this one through the public proxy", "addr", opsLn.Addr().String())
			if err := s.opsHTTP.Serve(opsLn); err != nil && !errors.Is(err, http.ErrServerClosed) {
				s.opts.Logger.Error("ops server error", "err", err)
			}
		}()
		defer func() {
			shutCtx, cancel := context.WithTimeout(context.Background(), s.opts.ShutdownTimeout)
			defer cancel()
			_ = s.opsHTTP.Shutdown(shutCtx)
		}()
	}

	if s.rl != nil {
		go s.sweepRateLimiter(ctx)
	}
	if s.opts.EnableMetrics {
		if pool, ok := s.opts.Storage.(poolStatsProvider); ok {
			go s.sweepPoolStats(ctx, pool)
		}
		go s.sweepBusinessMetrics(ctx)
	}

	errCh := make(chan error, 1)
	go func() {
		s.opts.Logger.Info("serving", "addr", ln.Addr().String())
		errCh <- s.http.Serve(ln)
	}()

	select {
	case <-ctx.Done():
		s.opts.Logger.Info("shutting down")
		// Cancel every in-flight import/gammonNet job BEFORE Shutdown: each
		// one's handler is watching its own context and, once cancelled,
		// emits a trailing {"event":"cancelled"} and returns on its own —
		// see runGammonNetSweep and handleImport. Left to Shutdown alone,
		// a streaming handler is either waited on past ShutdownTimeout (an
		// import mid a 512 MiB upload) or has its connection cut the moment
		// the deadline passes, with no chance to tell the client why
		// (#234).
		s.imports.cancelAll()
		s.gammonnetJobs.cancelAll()
		shutCtx, cancel := context.WithTimeout(context.Background(), s.opts.ShutdownTimeout)
		defer cancel()
		if err := s.http.Shutdown(shutCtx); err != nil {
			return fmt.Errorf("server: shutdown: %w", err)
		}
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("server: serve: %w", err)
	}
}
